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
