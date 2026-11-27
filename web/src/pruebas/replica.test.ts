import { describe, expect, it } from 'vitest'
import { ESTADO_INICIAL, reducir, type EventoCluster } from '../admin/estado'
import { MAX_PAUSA_MS, parsearJsonl, reproducir, VELOCIDADES } from '../admin/replica'
import type { Reloj } from '../admin/ws'

// Un .jsonl sintético con la forma de cluster.NuevoGrabador: un evento por línea.
const t = (ms: number) => new Date(Date.UTC(2026, 10, 26, 20, 0, 0, ms)).toISOString()
const EVENTOS: EventoCluster[] = [
  { v: 1, tipo: 'registro', t: t(0), epoca: 0, nodo: 'nodo-0', nodos: [{ nombre: 'nodo-0', remoto: 'x', workers: 4, sano: true, carga: 0 }] },
  { v: 1, tipo: 'epoca', t: t(400), epoca: 1, perdida: 0.69, pesos: [0, 0] },
  { v: 1, tipo: 'epoca', t: t(800), epoca: 2, perdida: 0.6, pesos: [0.1, 0.2] },
  { v: 1, tipo: 'fin', t: t(60_800), epoca: 2 },
]
const JSONL = EVENTOS.map((e) => JSON.stringify(e)).join('\n') + '\n'

function relojManual() {
  let ahora = 0
  const cola: { cuando: number; f: () => void; id: number }[] = []
  let id = 0
  const reloj: Reloj = {
    setTimeout: (f, ms) => {
      cola.push({ cuando: ahora + ms, f, id: ++id })
      return id
    },
    clearTimeout: (i) => {
      const k = cola.findIndex((c) => c.id === i)
      if (k >= 0) cola.splice(k, 1)
    },
  }
  return {
    reloj,
    pendientes: () => cola.map((c) => c.cuando - ahora),
    // Avanza hasta `ahora + ms` disparando cada temporizador en su instante, así
    // los que se programan dentro de un callback cuentan desde ese instante.
    avanzar(ms: number) {
      const hasta = ahora + ms
      for (;;) {
        cola.sort((a, b) => a.cuando - b.cuando)
        if (!cola.length || cola[0]!.cuando > hasta) break
        const c = cola.shift()!
        ahora = c.cuando
        c.f()
      }
      ahora = hasta
    },
  }
}

describe('parsearJsonl', () => {
  it('lee un evento por línea e ignora las vacías (también \\r\\n)', () => {
    expect(parsearJsonl(JSONL)).toEqual(EVENTOS)
    expect(parsearJsonl(JSONL.replace(/\n/g, '\r\n') + '\n\n')).toEqual(EVENTOS)
  })

  it('una línea que no es un Evento válido da error con su número de línea', () => {
    const malo = [JSON.stringify(EVENTOS[0]), '', '{"v":1,"tipo":"epoca"', JSON.stringify(EVENTOS[1])].join('\n')
    expect(() => parsearJsonl(malo)).toThrow(/línea 3/)
    expect(() => parsearJsonl(JSON.stringify({ ...EVENTOS[1], v: 2 }))).toThrow(/línea 1.*no es un evento/)
  })

  it('un archivo vacío también es un error', () => {
    expect(() => parsearJsonl('\n  \n')).toThrow(/vacío/)
  })
})

describe('reproducir', () => {
  it('respeta los tiempos del archivo divididos por la velocidad', () => {
    const r = relojManual()
    const vistos: string[] = []
    reproducir(EVENTOS, 4, (e) => vistos.push(e.tipo), { reloj: r.reloj })
    expect(vistos).toEqual(['registro'])
    r.avanzar(99)
    expect(vistos).toHaveLength(1)
    r.avanzar(1) // 400 ms / 4
    expect(vistos).toEqual(['registro', 'epoca'])
    r.avanzar(100)
    expect(vistos).toEqual(['registro', 'epoca', 'epoca'])
  })

  it('una pausa larga del archivo se acorta a MAX_PAUSA_MS', () => {
    const r = relojManual()
    const vistos: string[] = []
    reproducir(EVENTOS, 1, (e) => vistos.push(e.tipo), { reloj: r.reloj })
    r.avanzar(800)
    expect(r.pendientes()).toEqual([MAX_PAUSA_MS])
    r.avanzar(MAX_PAUSA_MS)
    expect(vistos.at(-1)).toBe('fin')
  })

  it('reproducida con el mismo reducer, deja el mismo estado que en vivo', () => {
    const r = relojManual()
    let estado = ESTADO_INICIAL
    let terminado = false
    reproducir(EVENTOS, 16, (e) => (estado = reducir(estado, e)), { reloj: r.reloj, alTerminar: () => (terminado = true) })
    r.avanzar(10 * MAX_PAUSA_MS)
    expect(terminado).toBe(true)
    expect(estado).toEqual(EVENTOS.reduce((e, x) => reducir(e, x), ESTADO_INICIAL))
  })

  it('detener() corta la reproducción', () => {
    const r = relojManual()
    const vistos: string[] = []
    const rep = reproducir(EVENTOS, 1, (e) => vistos.push(e.tipo), { reloj: r.reloj })
    rep.detener()
    r.avanzar(10 * MAX_PAUSA_MS)
    expect(vistos).toEqual(['registro'])
  })

  it('las velocidades ofrecidas son ×1, ×4 y ×16', () => {
    expect(VELOCIDADES).toEqual([1, 4, 16])
  })
})
