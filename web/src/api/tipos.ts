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
