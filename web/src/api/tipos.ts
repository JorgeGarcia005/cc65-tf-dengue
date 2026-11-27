// Tipos de GET /api/v1/riesgo (subconjunto de reports/evaluacion.json, #59).

export interface Celda {
  departamento: string // TODOS = nivel nacional; OTROS = departamentos agrupados por pocos casos
  semana: number // 0 = todas
  etapa: string // todas = todos los grupos de edad
  casos: number
  con_signos_observados: number
  tasa_observada: number
  riesgo_calibrado_medio: number
  casos_esperados: number
  fraccion_nacional: number
  razon_vs_nacional: number
  insuficiente: boolean
}

export interface Cubo {
  min_casos: number
  departamentos: string[] // los del modelo, con OTROS y TODOS al final
  etapas: string[] // con «todas» al final
  celdas: Celda[]
}

export interface DocumentoRiesgo {
  version_esquema: number
  version_modelo: string
  riesgo: Cubo
  prioridades: Celda[]
}

export const TODOS = 'TODOS'
export const OTROS = 'OTROS'
export const TODAS = 'todas'

// GET /api/v1/evaluacion (reports/evaluacion.json completo): lo que usan el
// inicio (#67) y la vista de evaluación (#68).
export interface Metricas {
  Umbral: number
  VP: number
  FP: number
  VN: number
  FN: number
  Exactitud: number
  Precision: number
  Recall: number
  F1: number
  AUC: number
}

export interface DocumentoEvaluacion extends DocumentoRiesgo {
  filas_test: number
  positivos_test: number
  metricas_test: Metricas
  auc_trapecio: number
  brier_calibrado: number
}

// El resto de reports/evaluacion.json: lo que dibuja la vista de evaluación (#68).
export interface Fuente {
  archivo: string
  sha256: string
}

export interface PuntoROC {
  fpr: number
  tpr: number
  umbral_calibrado: number
}

/** Un decil de riesgo calibrado con la tasa observada y su IC 95 %. */
export interface TramoCalibracion {
  casos: number
  riesgo_calibrado_medio: number
  tasa_observada: number
  ic95_inf: number
  ic95_sup: number
}

export interface MedicionNodos {
  nodos: number
  seg_por_epoca: { media_recortada: number; ic95_inf: number; ic95_sup: number }
  speedup: number
  eficiencia: number
  desglose_por_epoca: { fraccion_red_y_espera: number }
}

export interface Escalabilidad {
  maquina: { cpu: string; so: string; arquitectura?: string; go?: string; num_cpu?: number; gomaxprocs?: number }
  mediciones: MedicionNodos[]
}

export interface CorridaLatencia {
  nombre: string
  peticiones: number
  errores: number
  p50_ms: number
  p95_ms: number
  p99_ms: number
  rps: number
  cumple_limite: boolean
}

export interface LatenciaApi {
  maquina: string
  limite_p95_ms: number
  corridas: CorridaLatencia[]
}

export interface DocumentoEvaluacionCompleto extends DocumentoEvaluacion {
  fuentes: Fuente[]
  roc: PuntoROC[]
  calibracion: TramoCalibracion[]
  escalabilidad: Escalabilidad
  latencia_api: LatenciaApi
}

// GET /api/v1/modelo.
export interface ModeloPublicado {
  version: string
  generado: string
  umbral: number
  ajuste_prior: number
  sesgo: number
  ano_min: number
  departamentos: string[]
  variables: { nombre: string; peso: number }[]
}

// POST /api/v1/predicciones.
export interface Consulta {
  departamento: string
  ano: number
  semana: number
  edad: number
  tipo_edad: 'A' | 'M' | 'D'
  sexo: 'F' | 'M'
}

export interface Contribucion {
  variable: string
  valor: number
  peso: number
  aporte: number
}

export interface Prediccion {
  riesgo: number
  probabilidad_ponderada: number
  clase: 0 | 1
  etiqueta: string
  umbral: number
  departamento_modelo: string
  version_modelo: string
  logit: number
  sesgo: number
  contribuciones: Contribucion[]
}
