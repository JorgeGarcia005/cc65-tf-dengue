import { ErrorApi } from '../api/cliente'
import { esEvento, type EventoCluster } from './estado'

// Conexión del panel admin al WebSocket del coordinador (#69, ADR-0007):
// 1. POST /api/v1/ws/ticket con el JWT → ticket de un solo uso (30 s).
// 2. new WebSocket(ws://…/api/v1/ws?ticket=…). El navegador manda el Origin.
// 3. Si se cierra sin querer (todo código salvo 1000), reintenta con espera
//    min(30 s, 0,5 s · 2^k) más jitter y SIEMPRE con un ticket nuevo: el
//    anterior ya se gastó en el handshake.
// La red, el reloj y el azar se inyectan para poder testearlo sin servidor.

/** Lo mínimo de un WebSocket que usa la conexión (el del navegador lo cumple). */
export interface WSLike {
  onopen: (() => void) | null
  onmessage: ((m: { data: unknown }) => void) | null
  onclose: ((c: { code: number; reason: string }) => void) | null
  onerror: (() => void) | null
  close(code?: number, reason?: string): void
}

export interface Reloj {
  setTimeout(f: () => void, ms: number): unknown
  clearTimeout(id: unknown): void
}

export type EstadoConexion =
  | { estado: 'conectando' }
  | { estado: 'abierto' }
  | { estado: 'reintentando'; enMs: number; motivo: string }
  | { estado: 'cerrado'; motivo?: string }

export interface OpcionesConexion {
  apiUrl: string
  pedirTicket: () => Promise<string>
  alEvento: (e: EventoCluster) => void
  alEstado?: (e: EstadoConexion) => void
  /** Un mensaje que no es un cluster.Evento válido: se ignora, pero se avisa. */
  alInvalido?: (mensaje: unknown) => void
  crearWS?: (url: string) => WSLike
  reloj?: Reloj
  aleatorio?: () => number
}

export const ESPERA_BASE_MS = 500
export const ESPERA_MAX_MS = 30_000
/** Fracción máxima de jitter sobre la espera (evita que varios paneles reconecten a la vez). */
export const JITTER = 0.2
/** Cierre normal (RFC 6455 §7.4.1): lo usa el panel al salir; no se reintenta. */
export const CIERRE_NORMAL = 1000
/** El servidor se apaga o reinicia: se avisa y se reintenta. */
export const CIERRE_SALIENDO = 1001

export function esperaReintento(k: number, aleatorio: () => number = Math.random): number {
  const base = Math.min(ESPERA_MAX_MS, ESPERA_BASE_MS * 2 ** k)
  return Math.round(base * (1 + JITTER * aleatorio()))
}

export function urlWS(apiUrl: string, ticket: string): string {
  return `${apiUrl.replace(/\/+$/, '').replace(/^http/, 'ws')}/api/v1/ws?ticket=${encodeURIComponent(ticket)}`
}

const relojNavegador: Reloj = {
  setTimeout: (f, ms) => window.setTimeout(f, ms),
  clearTimeout: (id) => window.clearTimeout(id as number),
}

/** Abre la conexión y la mantiene viva hasta cerrar(). */
export function conectar(op: OpcionesConexion): { cerrar(): void } {
  const crearWS = op.crearWS ?? ((url: string) => new WebSocket(url) as unknown as WSLike)
  const reloj = op.reloj ?? relojNavegador
  const aleatorio = op.aleatorio ?? Math.random
  const avisar = (e: EstadoConexion) => op.alEstado?.(e)

  let intentos = 0
  let terminado = false
  let socket: WSLike | null = null
  let espera: unknown = null

  function reintentar(motivo: string) {
    if (terminado) return
    const enMs = esperaReintento(intentos++, aleatorio)
    avisar({ estado: 'reintentando', enMs, motivo })
    espera = reloj.setTimeout(() => {
      espera = null
      void abrir()
    }, enMs)
  }

  async function abrir() {
    if (terminado) return
    avisar({ estado: 'conectando' })
    let ticket: string
    try {
      ticket = await op.pedirTicket()
    } catch (e) {
      // Sin permiso (token vencido o rol ajeno) no tiene sentido insistir.
      if (e instanceof ErrorApi && (e.estado === 401 || e.estado === 403)) {
        terminado = true
        avisar({ estado: 'cerrado', motivo: e.message })
        return
      }
      reintentar(e instanceof Error ? e.message : 'no se pudo pedir el ticket')
      return
    }
    if (terminado) return
    const s = crearWS(urlWS(op.apiUrl, ticket))
    socket = s
    s.onopen = () => {
      intentos = 0
      avisar({ estado: 'abierto' })
    }
    s.onmessage = (m) => {
      let v: unknown
      try {
        v = typeof m.data === 'string' ? JSON.parse(m.data) : undefined
      } catch {
        v = undefined
      }
      if (esEvento(v)) op.alEvento(v)
      else op.alInvalido?.(m.data)
    }
    s.onerror = () => undefined // el navegador siempre manda close después
    s.onclose = (c) => {
      if (socket === s) socket = null
      if (terminado || c.code === CIERRE_NORMAL) {
        terminado = true
        avisar({ estado: 'cerrado', motivo: c.reason || undefined })
        return
      }
      reintentar(c.code === CIERRE_SALIENDO ? 'el servidor se está reiniciando' : `conexión perdida (código ${c.code})`)
    }
  }

  void abrir()
  return {
    cerrar() {
      if (terminado) return
      terminado = true
      if (espera !== null) reloj.clearTimeout(espera)
      espera = null
      if (socket) socket.close(CIERRE_NORMAL, 'el panel se cerró')
      else avisar({ estado: 'cerrado' })
    },
  }
}
