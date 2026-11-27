import { describe, expect, it } from 'vitest'
import { ESTADO_INICIAL, anotar, esEvento, nodosOrdenados, reducir, reiniciarEntrenamiento, type EstadoPanel, type EventoCluster } from '../admin/estado'

// Eventos con la forma de cluster.Evento (internal/cluster/eventos.go).
const T = '2026-11-26T20:00:00.000-05:00'
const nodo = (nombre: string, sano = true, carga = 1, motivo?: string) => ({ nombre, remoto: '172.20.0.5:1', workers: 16, sano, carga, ...(motivo ? { motivo } : {}) })
const ev = (e: Partial<EventoCluster> & Pick<EventoCluster, 'tipo'>): EventoCluster => ({ v: 1, t: T, epoca: 0, ...e })

const registros = [
  ev({ tipo: 'registro', nodo: 'nodo-0', nodos: [nodo('nodo-0', true, 0)] }),
  ev({ tipo: 'registro', nodo: 'nodo-1', nodos: [nodo('nodo-0', true, 0), nodo('nodo-1', true, 0)] }),
  ev({ tipo: 'registro', nodo: 'nodo-2', nodos: [nodo('nodo-0', true, 0), nodo('nodo-1', true, 0), nodo('nodo-2', true, 0)] }),
]
const aplicar = (evs: EventoCluster[], inicial: EstadoPanel = ESTADO_INICIAL) => evs.reduce(reducir, inicial)

describe('esEvento', () => {
  it('acepta un evento del coordinador y rechaza lo que no lo es', () => {
    expect(esEvento(ev({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0, 0] }))).toBe(true)
    expect(esEvento(ev({ tipo: 'tarea', nodo: 'nodo-0', shard: 0 }))).toBe(true)
    expect(esEvento({ ...ev({ tipo: 'epoca' }), v: 2 })).toBe(false)
    expect(esEvento({ ...ev({ tipo: 'epoca' }), tipo: 'otro' })).toBe(false)
    expect(esEvento({ ...ev({ tipo: 'epoca' }), epoca: '1' })).toBe(false)
    expect(esEvento({ ...ev({ tipo: 'epoca' }), perdida: 'x' })).toBe(false)
    expect(esEvento({ ...ev({ tipo: 'epoca' }), pesos: [1, 'a'] })).toBe(false)
    expect(esEvento({ ...ev({ tipo: 'registro' }), nodos: [{ nombre: 1 }] })).toBe(false)
    expect(esEvento(null)).toBe(false)
    expect(esEvento('{"v":1}')).toBe(false)
  })
})

describe('reducir', () => {
  it('es puro: no muta el estado anterior', () => {
    const antes = aplicar(registros)
    const copia = structuredClone(antes)
    reducir(antes, ev({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0.1, 0.2] }))
    expect(antes).toEqual(copia)
  })

  it('registro: los nodos salen de la instantánea del coordinador y la bitácora lo anota', () => {
    const e = aplicar(registros)
    expect(nodosOrdenados(e).map((n) => n.nombre)).toEqual(['nodo-0', 'nodo-1', 'nodo-2'])
    expect(nodosOrdenados(e).every((n) => n.sano)).toBe(true)
    expect(e.bitacora.filter((l) => l.tipo === 'registro')).toHaveLength(3)
    expect(e.bitacora.at(-1)!.texto).toContain('nodo-2')
  })

  it('epoca: agrega (época, pérdida) a la curva y guarda los pesos', () => {
    const e = aplicar([
      ...registros,
      ev({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0, 0, 0] }),
      ev({ tipo: 'epoca', epoca: 2, perdida: 0.6, pesos: [0.1, -0.2, 0.05] }),
    ])
    expect(e.curva).toEqual([
      { epoca: 1, perdida: 0.69 },
      { epoca: 2, perdida: 0.6 },
    ])
    expect(e.pesos).toEqual([0.1, -0.2, 0.05])
    expect(e.epoca).toBe(2)
  })

  it('tarea y gradiente: marcan actividad por nodo sin llenar la bitácora', () => {
    const e = aplicar([
      ...registros,
      ev({ tipo: 'tarea', epoca: 0, nodo: 'nodo-1', shard: 1 }),
      ev({ tipo: 'gradiente', epoca: 0, nodo: 'nodo-1', shard: 1, micros_calculo: 1830 }),
    ])
    const n1 = nodosOrdenados(e).find((n) => n.nombre === 'nodo-1')!
    expect(n1.ultimoShard).toBe(1)
    expect(n1.microsCalculo).toBe(1830)
    expect(e.bitacora.some((l) => l.tipo === 'tarea' || l.tipo === 'gradiente')).toBe(false)
  })

  it('descarte y reasignación: el nodo cae, la bitácora lo dice y se cuenta la reasignación', () => {
    const e = aplicar([
      ...registros,
      ev({ tipo: 'epoca', epoca: 2, perdida: 0.6, pesos: [0] }),
      ev({ tipo: 'descarte', epoca: 2, nodo: 'nodo-1', detalle: 'conexión cortada', nodos: [nodo('nodo-0'), nodo('nodo-1', false, 0, 'conexión cortada'), nodo('nodo-2')] }),
      ev({ tipo: 'reasignacion', epoca: 2, nodo: 'nodo-0', shard: 1 }),
    ])
    const caido = nodosOrdenados(e).find((n) => n.nombre === 'nodo-1')!
    expect(caido.sano).toBe(false)
    expect(caido.motivo).toBe('conexión cortada')
    expect(e.reasignaciones).toBe(1)
    const ultimas = e.bitacora.slice(-2)
    expect(ultimas[0]).toMatchObject({ tipo: 'descarte', nivel: 'alerta' })
    expect(ultimas[0]!.texto).toContain('nodo-1')
    expect(ultimas[1]).toMatchObject({ tipo: 'reasignacion' })
    expect(ultimas[1]!.texto).toMatch(/shard 1.*nodo-0/)
  })

  it('reasignación de un nodo que no estaba en la instantánea: lo agrega en vez de romperse', () => {
    const e = aplicar([ev({ tipo: 'reasignacion', epoca: 1, nodo: 'nodo-9', shard: 0 })])
    expect(nodosOrdenados(e).map((n) => n.nombre)).toEqual(['nodo-9'])
  })

  it('fin marca terminado; abortado guarda el motivo', () => {
    expect(aplicar([...registros, ev({ tipo: 'fin', epoca: 5 })]).terminado).toBe(true)
    const a = aplicar([...registros, ev({ tipo: 'abortado', epoca: 3, detalle: 'sin nodos sanos' })])
    expect(a.abortado).toBe('sin nodos sanos')
    expect(a.bitacora.at(-1)).toMatchObject({ tipo: 'abortado', nivel: 'alerta' })
  })

  it('la bitácora anota las épocas sin inundarse y nunca pasa del tope', () => {
    const epocas = Array.from({ length: 1000 }, (_, i) => ev({ tipo: 'epoca', epoca: i + 1, perdida: 0.5, pesos: [0] }))
    const e = aplicar(epocas)
    expect(e.curva).toHaveLength(1000)
    expect(e.bitacora.length).toBeLessThanOrEqual(200)
    expect(e.bitacora.some((l) => l.texto.includes('época 1 '))).toBe(true)
  })

  it('reiniciarEntrenamiento limpia curva y pesos pero conserva nodos y bitácora', () => {
    const e = aplicar([...registros, ev({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0] }), ev({ tipo: 'fin', epoca: 1 })])
    const r = reiniciarEntrenamiento(e)
    expect(r.curva).toEqual([])
    expect(r.pesos).toBeNull()
    expect(r.terminado).toBe(false)
    expect(r.reasignaciones).toBe(0)
    expect(nodosOrdenados(r)).toHaveLength(3)
    expect(r.bitacora).toEqual(e.bitacora)
  })

  it('anotar agrega una línea local (p. ej. reconexión) con su nivel', () => {
    const e = anotar(ESTADO_INICIAL, 'reintento en 1 s', 'aviso', T)
    expect(e.bitacora).toEqual([{ id: 1, t: T, tipo: 'local', nivel: 'aviso', texto: 'reintento en 1 s' }])
  })
})

describe('conCluster', () => {
  it('siembra los nodos desde GET /api/v1/cluster y valida su forma', async () => {
    const { conCluster, esEstadoCluster } = await import('../admin/estado')
    const c = { epocas: 0, perdida: 0, reasignaciones: 0, nodos: [nodo('b-host'), nodo('a-host', false, 0, 'se cayó')] }
    expect(esEstadoCluster(c)).toBe(true)
    expect(esEstadoCluster({ ...c, nodos: null })).toBe(false)
    expect(esEstadoCluster('<html>')).toBe(false)
    const e = conCluster(ESTADO_INICIAL, c)
    expect(nodosOrdenados(e).map((n) => [n.nombre, n.sano])).toEqual([
      ['a-host', false],
      ['b-host', true],
    ])
    expect(e.bitacora).toEqual([])
  })
})

describe('evento modelo y sello bit a bit', () => {
  it('guarda la versión y los pesos finales que anuncia la API', () => {
    const e = aplicar([...registros, ev({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0, 0] }), ev({ tipo: 'fin', epoca: 1 }), ev({ tipo: 'modelo', epoca: 1, detalle: 'abc123', pesos: [0.1, -0.2] })])
    expect(e.modelo).toEqual({ version: 'abc123', pesos: [0.1, -0.2] })
    expect(e.bitacora.at(-1)!.texto).toContain('abc123')
    // Los de la época siguen siendo los de la época: el modelo final no los pisa.
    expect(e.pesos).toEqual([0, 0])
  })

  it('selloBitABit: ✓ solo si cada peso es idéntico (===), incluido el sesgo', async () => {
    const { selloBitABit } = await import('../admin/estado')
    const publicado = { version: 'abc', sesgo: -0.2, variables: [{ nombre: 'a', peso: 0.1 }] }
    expect(selloBitABit({ version: 'abc', pesos: [0.1, -0.2] }, publicado)).toEqual({ ok: true, comparados: 2 })
    expect(selloBitABit({ version: 'abc', pesos: [0.1000000000000001, -0.2] }, publicado)).toMatchObject({ ok: false, primeraDiferencia: 0 })
    expect(selloBitABit({ version: 'abc', pesos: [0.1, -0.2, 0] }, publicado).ok).toBe(false)
    expect(selloBitABit({ version: 'otra', pesos: [0.1, -0.2] }, publicado)).toMatchObject({ ok: false, motivo: expect.stringContaining('versión') })
  })

  it('reiniciarEntrenamiento olvida el modelo anunciado', () => {
    const e = aplicar([ev({ tipo: 'modelo', epoca: 1, detalle: 'abc', pesos: [1] })])
    expect(reiniciarEntrenamiento(e).modelo).toBeNull()
  })
})

// Visto en vivo: el contador de épocas del coordinador es acumulado entre
// entrenamientos de la misma corrida (el segundo arranca en 301).
describe('épocas relativas al entrenamiento', () => {
  it('la curva y la época cuentan desde 1 aunque el coordinador venga de otro entrenamiento', () => {
    const e = aplicar([
      ev({ tipo: 'epoca', epoca: 301, perdida: 0.69, pesos: [0] }),
      ev({ tipo: 'epoca', epoca: 302, perdida: 0.68, pesos: [0] }),
      ev({ tipo: 'reasignacion', epoca: 302, nodo: 'nodo-0', shard: 1 }),
    ])
    expect(e.curva.map((p) => p.epoca)).toEqual([1, 2])
    expect(e.epoca).toBe(2)
    expect(e.bitacora.at(-1)!.texto).toMatch(/en la época 2$/)
    expect(e.bitacora[0]!.texto).toMatch(/^época 1 /)
  })

  it('tras reiniciarEntrenamiento, el siguiente vuelve a contar desde 1', () => {
    const uno = aplicar([ev({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0] }), ev({ tipo: 'epoca', epoca: 2, perdida: 0.6, pesos: [0] })])
    const dos = aplicar([ev({ tipo: 'epoca', epoca: 3, perdida: 0.69, pesos: [0] })], reiniciarEntrenamiento(uno))
    expect(dos.curva).toEqual([{ epoca: 1, perdida: 0.69 }])
    expect(dos.epoca).toBe(1)
  })
})

// Visto en vivo: el nodo cortado se vuelve a registrar en el mismo milisegundo,
// así que «sano» vuelve a true enseguida. La caída queda contada.
describe('caídas por nodo', () => {
  it('descarte suma una caída que el registro siguiente no borra', () => {
    const e = aplicar([
      ...registros,
      ev({ tipo: 'descarte', epoca: 5, nodo: 'nodo-2', detalle: 'cortado', nodos: [nodo('nodo-0'), nodo('nodo-1'), nodo('nodo-2', false, 0)] }),
      ev({ tipo: 'registro', epoca: 5, nodo: 'nodo-2', nodos: [nodo('nodo-0'), nodo('nodo-1'), nodo('nodo-2', true, 0)] }),
    ])
    const n2 = nodosOrdenados(e).find((n) => n.nombre === 'nodo-2')!
    expect(n2.sano).toBe(true)
    expect(n2.caidas).toBe(1)
    expect(nodosOrdenados(e).find((n) => n.nombre === 'nodo-0')!.caidas ?? 0).toBe(0)
  })
})
