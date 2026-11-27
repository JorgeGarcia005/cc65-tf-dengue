import { describe, expect, it } from 'vitest'
import { ESTADO_INICIAL, conCluster, esEstadoCluster, esEvento, reducir, type EstadoPanel, type EventoCluster } from '../admin/estado'

// #91 · enunciado TB2 §6: «Panel administrador: métricas del cluster (uso CPU,
// latencia, número de nodos, logs)».
const T = '2026-11-27T20:00:00.000-05:00'
const ev = (e: Partial<EventoCluster> & Pick<EventoCluster, 'tipo'>): EventoCluster => ({ v: 1, t: T, epoca: 0, ...e })
const nodo = (nombre: string, extra: object = {}) => ({ nombre, remoto: 'x:1', workers: 4, sano: true, carga: 1, ...extra })
const conNodos = (...nombres: string[]): EstadoPanel => reducir(ESTADO_INICIAL, ev({ tipo: 'registro', nodo: nombres[0], nodos: nombres.map((n) => nodo(n)) }))
const API = { predicciones: 40, ventana: 40, p50_ms: 0.8, p95_ms: 1.9, max_ms: 3.2, aciertos_cache: 0.5, no_registradas: 0, limite_p95_ms: 100, cumple_limite_p95: true }

describe('métricas por nodo desde los eventos «gradiente»', () => {
  it('CPU sobre la capacidad de los workers, latencia y su desglose en cálculo y red', () => {
    // 4 workers, 10 ms de cálculo y 36 ms de CPU → 36 / (10 · 4) = 90 %; 12 ms de ida y vuelta → 2 ms de red.
    const e = reducir(conNodos('a'), ev({ tipo: 'gradiente', epoca: 3, nodo: 'a', shard: 0, micros_calculo: 10_000, micros_cpu: 36_000, micros_ida_vuelta: 12_000 }))
    expect(e.nodos.a!.vista).toEqual({ epoca: 3, calculoMs: 10, latenciaMs: 12, redMs: 2, usoCpu: 0.9 })
  })

  it('dos shards del mismo nodo en una época se suman; la época siguiente empieza de cero', () => {
    let e = conNodos('a')
    e = reducir(e, ev({ tipo: 'gradiente', epoca: 1, nodo: 'a', shard: 0, micros_calculo: 5_000, micros_cpu: 20_000, micros_ida_vuelta: 6_000 }))
    e = reducir(e, ev({ tipo: 'gradiente', epoca: 1, nodo: 'a', shard: 1, micros_calculo: 5_000, micros_cpu: 20_000, micros_ida_vuelta: 11_500 }))
    expect(e.nodos.a!.vista).toEqual({ epoca: 1, calculoMs: 10, latenciaMs: 11.5, redMs: 1.5, usoCpu: 1 })
    e = reducir(e, ev({ tipo: 'gradiente', epoca: 2, nodo: 'a', shard: 0, micros_calculo: 4_000, micros_cpu: 8_000, micros_ida_vuelta: 4_500 }))
    expect(e.nodos.a!.vista).toEqual({ epoca: 2, calculoMs: 4, latenciaMs: 4.5, redMs: 0.5, usoCpu: 0.5 })
  })

  it('un nodo que no informa CPU muestra la latencia y la CPU como null (no 0 %)', () => {
    const e = reducir(conNodos('a'), ev({ tipo: 'gradiente', epoca: 1, nodo: 'a', shard: 0, micros_calculo: 2_000, micros_ida_vuelta: 3_000 }))
    expect(e.nodos.a!.vista).toMatchObject({ latenciaMs: 3, usoCpu: null })
  })

  it('un .jsonl anterior a #91 (sin micros_ida_vuelta) no inventa métricas', () => {
    const e = reducir(conNodos('a'), ev({ tipo: 'gradiente', epoca: 1, nodo: 'a', shard: 0, micros_calculo: 2_000 }))
    expect(e.nodos.a!.vista).toBeUndefined()
    expect(e.nodos.a!.microsCalculo).toBe(2_000)
  })

  it('esEvento valida los campos nuevos', () => {
    expect(esEvento(ev({ tipo: 'gradiente', micros_cpu: 1, micros_ida_vuelta: 2 }))).toBe(true)
    expect(esEvento({ ...ev({ tipo: 'gradiente' }), micros_cpu: 'x' })).toBe(false)
    expect(esEvento({ ...ev({ tipo: 'gradiente' }), micros_ida_vuelta: null })).toBe(false)
  })
})

describe('métricas desde GET /api/v1/cluster', () => {
  const metricas = { epoca: 7, shards: 1, calculo_ms: 13.9, latencia_ms: 15.2, red_ms: 1.3, uso_cpu: 0.93 }

  it('la instantánea siembra las métricas de cada nodo y las de la API', () => {
    const c = { epocas: 7, perdida: 0.6, reasignaciones: 0, nodos: [nodo('a', { metricas })], api: API }
    expect(esEstadoCluster(c)).toBe(true)
    const e = conCluster(ESTADO_INICIAL, c)
    expect(e.nodos.a!.vista).toEqual({ epoca: 7, calculoMs: 13.9, latenciaMs: 15.2, redMs: 1.3, usoCpu: 0.93 })
  })

  it('una instantánea más vieja que los eventos no pisa la métrica en vivo', () => {
    let e = conNodos('a')
    e = reducir(e, ev({ tipo: 'gradiente', epoca: 9, nodo: 'a', shard: 0, micros_calculo: 1_000, micros_cpu: 4_000, micros_ida_vuelta: 1_500 }))
    e = conCluster(e, { epocas: 7, perdida: 0.6, reasignaciones: 0, nodos: [nodo('a', { metricas })] })
    expect(e.nodos.a!.vista?.epoca).toBe(9)
  })

  it('rechaza métricas mal formadas', () => {
    const base = { epocas: 1, perdida: 0.6, reasignaciones: 0 }
    expect(esEstadoCluster({ ...base, nodos: [nodo('a', { metricas: { ...metricas, uso_cpu: 1.5 } })] })).toBe(false)
    expect(esEstadoCluster({ ...base, nodos: [nodo('a', { metricas: { ...metricas, latencia_ms: 'x' } })] })).toBe(false)
    expect(esEstadoCluster({ ...base, nodos: [], api: { ...API, p95_ms: null } })).toBe(false)
    expect(esEstadoCluster({ ...base, nodos: [], api: { ...API, cumple_limite_p95: 'sí' } })).toBe(false)
    expect(esEstadoCluster({ ...base, nodos: [nodo('a')] })).toBe(true) // sin métricas todavía: válido
  })
})
