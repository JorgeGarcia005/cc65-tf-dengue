import { TODAS, TODOS, type Celda, type Cubo } from '../api/tipos'

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

/** Enlace al mapa con la combinación ya filtrada (los filtros del mapa viven en la URL). */
export function enlaceMapa(c: Celda): string {
  const p = new URLSearchParams()
  if (c.departamento !== TODOS) p.set('depto', c.departamento)
  if (c.semana > 0) p.set('semana', String(c.semana))
  if (c.etapa !== TODAS) p.set('etapa', c.etapa)
  p.set('metrica', 'esperados')
  return `/mapa?${p}`
}
