import { esEvento, type EventoCluster } from './estado'
import type { Reloj } from './ws'

// Modo réplica (#69): reproduce un eventos_*.jsonl grabado por
// cluster.NuevoGrabador sin clúster ni API, con el mismo reducer que en vivo.

export const VELOCIDADES = [1, 4, 16] as const
/** Una pausa del archivo más larga que esto se acorta: la réplica no se queda quieta. */
export const MAX_PAUSA_MS = 2000

/** Un evento por línea; las vacías se ignoran. Cualquier línea inválida es un error con su número. */
export function parsearJsonl(texto: string): EventoCluster[] {
  const eventos: EventoCluster[] = []
  texto.split(/\r?\n/).forEach((linea, i) => {
    if (!linea.trim()) return
    let v: unknown
    try {
      v = JSON.parse(linea)
    } catch {
      throw new Error(`línea ${i + 1}: no es JSON válido`)
    }
    if (!esEvento(v)) throw new Error(`línea ${i + 1}: no es un evento del clúster (versión ${String((v as { v?: unknown })?.v)})`)
    eventos.push(v)
  })
  if (eventos.length === 0) throw new Error('el archivo está vacío: no hay eventos que reproducir')
  return eventos
}

const relojNavegador: Reloj = {
  setTimeout: (f, ms) => window.setTimeout(f, ms),
  clearTimeout: (id) => window.clearTimeout(id as number),
}

/**
 * Entrega los eventos en orden, separados por el tiempo que los separó al
 * grabarse dividido por la velocidad (con un tope de MAX_PAUSA_MS).
 */
export function reproducir(
  eventos: EventoCluster[],
  velocidad: number,
  alEvento: (e: EventoCluster) => void,
  op: { reloj?: Reloj; alTerminar?: () => void } = {},
): { detener(): void } {
  const reloj = op.reloj ?? relojNavegador
  let i = 0
  let pendiente: unknown = null
  let detenido = false

  function siguiente() {
    pendiente = null
    if (detenido) return
    const e = eventos[i++]!
    alEvento(e)
    if (i >= eventos.length) {
      op.alTerminar?.()
      return
    }
    const dt = Date.parse(eventos[i]!.t) - Date.parse(e.t)
    const ms = Math.min(MAX_PAUSA_MS, Math.max(0, Number.isFinite(dt) ? dt / velocidad : 0))
    pendiente = reloj.setTimeout(siguiente, ms)
  }

  siguiente()
  return {
    detener() {
      detenido = true
      if (pendiente !== null) reloj.clearTimeout(pendiente)
    },
  }
}
