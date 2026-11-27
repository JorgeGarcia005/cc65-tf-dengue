import { describe, expect, it, vi } from 'vitest'
import { ErrorApi } from '../api/cliente'
import { conectar, esperaReintento, urlWS, type Reloj, type WSLike } from '../admin/ws'

// Un WebSocket falso: el test decide cuándo se abre, qué llega y cómo se cierra.
class WSFalso implements WSLike {
  onopen: (() => void) | null = null
  onmessage: ((m: { data: unknown }) => void) | null = null
  onclose: ((c: { code: number; reason: string }) => void) | null = null
  onerror: (() => void) | null = null
  cerradoCon: number | undefined
  constructor(readonly url: string) {}
  close(code?: number) {
    this.cerradoCon = code
  }
  abrir() {
    this.onopen?.()
  }
  recibir(data: unknown) {
    this.onmessage?.({ data })
  }
  cerrar(code: number, reason = '') {
    this.onclose?.({ code, reason })
  }
}

/** Reloj manual: los temporizadores corren solo cuando el test avanza. */
function relojManual() {
  let ahora = 0
  let id = 0
  const pendientes = new Map<number, { cuando: number; f: () => void }>()
  const reloj: Reloj = {
    setTimeout: (f, ms) => {
      pendientes.set(++id, { cuando: ahora + ms, f })
      return id
    },
    clearTimeout: (i) => void pendientes.delete(i as number),
  }
  return {
    reloj,
    pendientes: () => [...pendientes.values()].map((p) => p.cuando - ahora),
    async avanzar(ms: number) {
      ahora += ms
      for (const [i, p] of [...pendientes]) {
        if (p.cuando <= ahora) {
          pendientes.delete(i)
          p.f()
        }
      }
      await Promise.resolve()
      await Promise.resolve()
    },
  }
}

const tick = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}

function montar(opciones: { pedirTicket?: () => Promise<string> } = {}) {
  const sockets: WSFalso[] = []
  const r = relojManual()
  let n = 0
  const pedirTicket = vi.fn(opciones.pedirTicket ?? (() => Promise.resolve(`ticket-${++n}`)))
  const eventos: unknown[] = []
  const estados: string[] = []
  const invalidos: unknown[] = []
  const c = conectar({
    apiUrl: 'http://localhost:18080',
    pedirTicket,
    crearWS: (url) => {
      const s = new WSFalso(url)
      sockets.push(s)
      return s
    },
    alEvento: (e) => eventos.push(e),
    alEstado: (e) => estados.push(e.estado),
    alInvalido: (m) => invalidos.push(m),
    reloj: r.reloj,
    aleatorio: () => 0,
  })
  return { c, sockets, r, pedirTicket, eventos, estados, invalidos }
}

const EVENTO = { v: 1, tipo: 'epoca', t: '2026-11-26T20:00:00Z', epoca: 1, perdida: 0.69, pesos: [0, 0] }

describe('urlWS y esperaReintento', () => {
  it('pasa de http a ws (y de https a wss) con el ticket escapado', () => {
    expect(urlWS('http://localhost:18080/', 'a+b/c')).toBe('ws://localhost:18080/api/v1/ws?ticket=a%2Bb%2Fc')
    expect(urlWS('https://api.vigia.pe', 't')).toBe('wss://api.vigia.pe/api/v1/ws?ticket=t')
  })

  it('espera min(30 s, 0,5 s · 2^k) más jitter', () => {
    expect([0, 1, 2, 3, 4, 5, 6, 10].map((k) => esperaReintento(k, () => 0))).toEqual([500, 1000, 2000, 4000, 8000, 16000, 30000, 30000])
    // El jitter suma hasta un 20 % y nunca pasa de 30 s + 20 %.
    expect(esperaReintento(0, () => 0.999)).toBeGreaterThan(500)
    expect(esperaReintento(0, () => 0.999)).toBeLessThanOrEqual(600)
    expect(esperaReintento(20, () => 0.999)).toBeLessThanOrEqual(36000)
  })
})

describe('conectar', () => {
  it('pide un ticket, abre el WebSocket y entrega los eventos válidos', async () => {
    const m = montar()
    await tick()
    expect(m.pedirTicket).toHaveBeenCalledTimes(1)
    expect(m.sockets[0]!.url).toBe('ws://localhost:18080/api/v1/ws?ticket=ticket-1')
    m.sockets[0]!.abrir()
    m.sockets[0]!.recibir(JSON.stringify(EVENTO))
    m.sockets[0]!.recibir('no es json')
    m.sockets[0]!.recibir(JSON.stringify({ v: 9, tipo: 'epoca' }))
    expect(m.eventos).toEqual([EVENTO])
    expect(m.invalidos).toHaveLength(2)
    expect(m.estados).toEqual(['conectando', 'abierto'])
  })

  it('si se cae, reintenta con espera exponencial y SIEMPRE con un ticket nuevo', async () => {
    const m = montar()
    await tick()
    m.sockets[0]!.abrir()
    m.sockets[0]!.cerrar(1006)
    expect(m.estados.at(-1)).toBe('reintentando')
    expect(m.r.pendientes()).toEqual([500])
    await m.r.avanzar(500)
    await tick()
    expect(m.pedirTicket).toHaveBeenCalledTimes(2)
    expect(m.sockets[1]!.url).toContain('ticket=ticket-2')

    // Sin abrir, vuelve a fallar: la espera se duplica.
    m.sockets[1]!.cerrar(1006)
    expect(m.r.pendientes()).toEqual([1000])
    await m.r.avanzar(1000)
    await tick()
    m.sockets[2]!.abrir() // al abrir, la espera vuelve a 0,5 s
    m.sockets[2]!.cerrar(1006)
    expect(m.r.pendientes()).toEqual([500])
    expect(new Set(m.sockets.map((s) => s.url)).size).toBe(3)
  })

  it('1001 (el servidor se apaga): avisa y reintenta', async () => {
    const m = montar()
    await tick()
    m.sockets[0]!.abrir()
    m.sockets[0]!.cerrar(1001, 'going away')
    expect(m.estados.at(-1)).toBe('reintentando')
    expect(m.r.pendientes()).toHaveLength(1)
  })

  it('1000 o cerrar() a propósito: no reintenta', async () => {
    const m = montar()
    await tick()
    m.sockets[0]!.abrir()
    m.sockets[0]!.cerrar(1000)
    expect(m.estados.at(-1)).toBe('cerrado')
    expect(m.r.pendientes()).toEqual([])

    const n = montar()
    await tick()
    n.sockets[0]!.abrir()
    n.c.cerrar()
    expect(n.sockets[0]!.cerradoCon).toBe(1000)
    n.sockets[0]!.cerrar(1000)
    expect(n.r.pendientes()).toEqual([])
    expect(n.estados.at(-1)).toBe('cerrado')
  })

  it('cerrar() mientras espera el reintento cancela el temporizador', async () => {
    const m = montar()
    await tick()
    m.sockets[0]!.cerrar(1006)
    expect(m.r.pendientes()).toHaveLength(1)
    m.c.cerrar()
    expect(m.r.pendientes()).toEqual([])
    await m.r.avanzar(60_000)
    expect(m.pedirTicket).toHaveBeenCalledTimes(1)
  })

  it('si falla el pedido de ticket por red, reintenta; con 401/403 se detiene', async () => {
    let fallos = 0
    const m = montar({ pedirTicket: () => (fallos++ === 0 ? Promise.reject(new ErrorApi(0, 'sin red')) : Promise.resolve('ok')) })
    await tick()
    expect(m.estados.at(-1)).toBe('reintentando')
    await m.r.avanzar(500)
    await tick()
    expect(m.sockets).toHaveLength(1)

    const p = montar({ pedirTicket: () => Promise.reject(new ErrorApi(403, 'tu rol no tiene acceso')) })
    await tick()
    expect(p.estados.at(-1)).toBe('cerrado')
    expect(p.r.pendientes()).toEqual([])
    expect(p.sockets).toHaveLength(0)
  })
})
