import { TODAS, TODOS, type Celda, type Cubo, type DocumentoEvaluacion } from '../api/tipos'

/** Cuántas combinaciones muestra el inicio (criterio de #67). */
export const TOP = 10

/**
 * Las combinaciones departamento × semana × grupo de edad en el mismo orden de
 * reports/evaluacion.json (cmd/evaluacion ya las ordena por casos esperados):
 * el front no reordena ni recalcula.
 */
export function topPrioridades(prioridades: Celda[], n = TOP): Celda[] {
  return prioridades.slice(0, n)
}

/** Semanas con más casos esperados con signos de alarma a nivel nacional (todas las edades). */
export function semanasPico(cubo: Cubo, n = 5): Celda[] {
  return cubo.celdas
    .filter((c) => c.departamento === TODOS && c.etapa === TODAS && c.semana > 0 && !c.insuficiente)
    .sort((a, b) => b.casos_esperados - a.casos_esperados || a.semana - b.semana)
    .slice(0, n)
}

/** Grupos de edad a nivel nacional, del de mayor riesgo calibrado al de menor. */
export function gruposDeEdad(cubo: Cubo): Celda[] {
  return cubo.celdas
    .filter((c) => c.departamento === TODOS && c.semana === 0 && c.etapa !== TODAS)
    .sort((a, b) => b.riesgo_calibrado_medio - a.riesgo_calibrado_medio)
}

const finito = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v)

/**
 * Forma mínima de GET /api/v1/evaluacion que usa el inicio; la API valida
 * riesgo y prioridades, pero no las métricas (PR-85 H1).
 */
export function esEvaluacion(v: unknown, version: number): v is DocumentoEvaluacion {
  const d = v as Record<string, unknown> | null
  if (!d || typeof d !== 'object' || d.version_esquema !== version || typeof d.version_modelo !== 'string') return false
  const r = d.riesgo as Record<string, unknown> | null
  const m = d.metricas_test as Record<string, unknown> | null
  return (
    !!r && Array.isArray(r.celdas) && Array.isArray(r.etapas) && finito(r.min_casos) && Array.isArray(d.prioridades) &&
    Number.isInteger(d.filas_test) && Number.isInteger(d.positivos_test) && finito(d.auc_trapecio) && finito(d.brier_calibrado) &&
    !!m && finito(m.Recall) && finito(m.Precision) && finito(m.Umbral)
  )
}

/** a/b, o 0 si b no es positivo: un reporte vacío no pinta NaN. */
export const proporcion = (a: number, b: number) => (b > 0 ? a / b : 0)

/** Enlace al mapa con la combinación ya filtrada (los filtros del mapa viven en la URL). */
export function enlaceMapa(c: Celda): string {
  const p = new URLSearchParams()
  if (c.departamento !== TODOS) p.set('depto', c.departamento)
  if (c.semana > 0) p.set('semana', String(c.semana))
  if (c.etapa !== TODAS) p.set('etapa', c.etapa)
  p.set('metrica', 'esperados')
  return `/mapa?${p}`
}
