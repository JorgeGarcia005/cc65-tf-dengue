import type { DocumentoEvaluacionCompleto, Fuente, Metricas, TramoCalibracion } from '../api/tipos'
import { esEvaluacion, proporcion } from '../inicio/datos'

const finito = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v)
const objeto = (v: unknown): v is Record<string, unknown> => !!v && typeof v === 'object' && !Array.isArray(v)
const lista = (v: unknown, cada: (x: Record<string, unknown>) => boolean) => Array.isArray(v) && v.length > 0 && v.every((x) => objeto(x) && cada(x))
const todosFinitos = (x: Record<string, unknown>, campos: string[]) => campos.every((c) => finito(x[c]))

/**
 * Forma completa de GET /api/v1/evaluacion que usa la vista de evaluación:
 * lo que valida el inicio (esEvaluacion) más la ROC, la calibración, la
 * escalabilidad, la latencia y las fuentes. Un 200 con HTML de proxy o con un
 * reporte viejo no llega a pintarse (PR-85 H1).
 */
export function esDocumentoEvaluacionCompleto(v: unknown, version: number): v is DocumentoEvaluacionCompleto {
  if (!esEvaluacion(v, version)) return false
  const d = v as unknown as Record<string, unknown>
  const m = d.metricas_test as unknown as Record<string, unknown>
  const esc = d.escalabilidad
  const lat = d.latencia_api
  return (
    ['VP', 'FP', 'VN', 'FN'].every((k) => Number.isInteger(m[k]) && (m[k] as number) >= 0) &&
    todosFinitos(m, ['Exactitud', 'F1']) &&
    lista(d.fuentes, (f) => typeof f.archivo === 'string') &&
    lista(d.roc, (p) => todosFinitos(p, ['fpr', 'tpr'])) &&
    lista(d.calibracion, (t) => Number.isInteger(t.casos) && (t.casos as number) >= 0 && todosFinitos(t, ['riesgo_calibrado_medio', 'tasa_observada', 'ic95_inf', 'ic95_sup'])) &&
    objeto(esc) &&
    objeto(esc.maquina) &&
    // Todo campo de texto que se pinta como hijo de JSX se valida como string: un
    // objeto ahí tumbaría React y dejaría la SPA en blanco (revisión previa H1).
    typeof esc.maquina.cpu === 'string' &&
    typeof esc.maquina.so === 'string' &&
    (esc.maquina.arquitectura === undefined || typeof esc.maquina.arquitectura === 'string') &&
    (esc.maquina.num_cpu === undefined || finito(esc.maquina.num_cpu)) &&
    lista(
      esc.mediciones,
      (x) =>
        Number.isInteger(x.nodos) &&
        (x.nodos as number) > 0 &&
        todosFinitos(x, ['speedup', 'eficiencia']) &&
        objeto(x.seg_por_epoca) &&
        todosFinitos(x.seg_por_epoca, ['media_recortada', 'ic95_inf', 'ic95_sup']) &&
        objeto(x.desglose_por_epoca) &&
        finito(x.desglose_por_epoca.fraccion_red_y_espera),
    ) &&
    objeto(lat) &&
    typeof lat.maquina === 'string' &&
    finito(lat.limite_p95_ms) &&
    lista(lat.corridas, (c) => typeof c.nombre === 'string' && typeof c.cumple_limite === 'boolean' && todosFinitos(c, ['peticiones', 'errores', 'p50_ms', 'p95_ms', 'p99_ms', 'rps']))
  )
}

/** Punto de la ROC del umbral del modelo, desde la matriz de confusión (sin NaN si una clase está vacía). */
export function puntoUmbral(m: Metricas): { fpr: number; tpr: number } {
  return { fpr: proporcion(m.FP, m.FP + m.VN), tpr: proporcion(m.VP, m.VP + m.FN) }
}

/** Filas = clase real (positivo, negativo); columnas = predicho (positivo, negativo). */
export function filasConfusion(m: Metricas): [[number, number], [number, number]] {
  return [
    [m.VP, m.FN],
    [m.FP, m.VN],
  ]
}

/** El archivo de doc.fuentes que coincide con el patrón: el pie de cada gráfico no se escribe a mano. */
export function fuente(doc: { fuentes: Fuente[] }, patron: RegExp): string | null {
  return doc.fuentes.find((f) => patron.test(f.archivo))?.archivo ?? null
}

/** Cuántos deciles tienen su riesgo predicho dentro del IC 95 % de la tasa observada. */
export function tramosDentroIC(tramos: TramoCalibracion[]): number {
  return tramos.filter((t) => t.riesgo_calibrado_medio >= t.ic95_inf && t.riesgo_calibrado_medio <= t.ic95_sup).length
}

/**
 * Lectura honesta del AUC, armada con su valor. Cortes de Hosmer y Lemeshow
 * (Applied Logistic Regression, 3.ª ed., 2013, cap. 5): 0,5 sin discriminación;
 * 0,5–0,7 pobre; 0,7–0,8 aceptable; 0,8–0,9 excelente; ≥ 0,9 sobresaliente.
 * El corte de 0,6 es nuestro, para separar «casi azar» dentro del tramo pobre.
 */
export function lecturaAuc(auc: number): string {
  if (auc < 0.6) return 'apenas mejor que el azar: no sirve para ordenar casos'
  if (auc < 0.7) return 'capacidad de discriminación modesta (pobre según Hosmer y Lemeshow, < 0,7): sirve para priorizar, no para diagnosticar'
  if (auc < 0.8) return 'capacidad de discriminación aceptable: sirve para priorizar; para diagnosticar hace falta la clínica'
  if (auc < 0.9) return 'capacidad de discriminación excelente en el conjunto de prueba'
  return 'capacidad de discriminación sobresaliente en el conjunto de prueba'
}
