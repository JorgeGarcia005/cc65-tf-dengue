import { scaleQuantile } from 'd3-scale'
import { OTROS, TODAS, TODOS, type Celda, type Cubo } from '../api/tipos'

export type Metrica = 'riesgo' | 'esperados'

/** Índice del cubo por departamento, semana y etapa. */
export class IndiceRiesgo {
  private readonly celdas = new Map<string, Celda>()
  readonly modelo: Set<string>

  constructor(readonly cubo: Cubo) {
    for (const c of cubo.celdas) this.celdas.set(clave(c.departamento, c.semana, c.etapa), c)
    this.modelo = new Set(cubo.departamentos.filter((d) => d !== TODOS))
  }

  celda(depto: string, semana: number, etapa: string): Celda | undefined {
    return this.celdas.get(clave(depto, semana, etapa))
  }

  /** El grupo del modelo al que pertenece un departamento del mapa. */
  grupo(nombreGeo: string): string {
    return this.modelo.has(nombreGeo) ? nombreGeo : OTROS
  }

  /** Celdas de las semanas 1–52 de un departamento y etapa (undefined si no hubo casos). */
  curva(depto: string, etapa: string): (Celda | undefined)[] {
    return Array.from({ length: 52 }, (_, i) => this.celda(depto, i + 1, etapa))
  }

}

function clave(d: string, s: number, e: string) {
  return `${d}|${s}|${e}`
}

export function valor(c: Celda | undefined, m: Metrica): number | undefined {
  if (!c) return undefined
  return m === 'riesgo' ? c.riesgo_calibrado_medio : c.casos_esperados
}

/** Rampa de 5 clases: del papel al rosa Riso y a la tinta (contraste creciente). */
export const COLORES = ['#f3dfd9', '#f2a9bd', '#e8336d', '#a3164a', '#4a0a22'] as const

/** Una clase de la leyenda: su color y el rango de valores que de verdad cae en ella. */
export interface Clase {
  color: string
  min: number
  max: number
}

export interface Escala {
  color(v: number): string
  /** Solo las clases que tienen algún departamento, con su rango observado (sin cortes inventados). */
  clases: Clase[]
  /** No hay ningún departamento del modelo con datos suficientes para este filtro. */
  vacia: boolean
}

/**
 * Escala por cuantiles sobre los departamentos del modelo con datos suficientes:
 * cada color agrupa una quinta parte. OTROS y TODOS no entran al cálculo. La
 * leyenda muestra el mínimo y el máximo reales de cada color, así que nunca
 * imprime un rango que no esté en los datos (ni clases vacías o repetidas).
 */
export function escalaPara(indice: IndiceRiesgo, semana: number, etapa: string, m: Metrica): Escala {
  const valores = [...indice.modelo]
    .filter((d) => d !== OTROS)
    .map((d) => indice.celda(d, semana, etapa))
    .filter((c): c is Celda => !!c && !c.insuficiente)
    .map((c) => valor(c, m)!)
  if (valores.length === 0) return { color: () => COLORES[0], clases: [], vacia: true }
  const q = scaleQuantile<string>().domain(valores).range([...COLORES])
  const porColor = new Map<string, Clase>()
  for (const v of valores) {
    const color = q(v)
    const c = porColor.get(color)
    if (c) {
      c.min = Math.min(c.min, v)
      c.max = Math.max(c.max, v)
    } else {
      porColor.set(color, { color, min: v, max: v })
    }
  }
  const clases = COLORES.flatMap((col) => (porColor.has(col) ? [porColor.get(col)!] : []))
  return { color: (v) => q(v), clases, vacia: false }
}

/**
 * Tramos de la curva semanal con datos suficientes: cada tramo es una serie de
 * semanas consecutivas [índice 0–51, valor]. Un hueco (semana sin casos o con
 * pocos) corta la línea en vez de unir los extremos con una recta.
 */
export function tramos(curva: (Celda | undefined)[], f: (c: Celda) => number): [number, number][][] {
  const out: [number, number][][] = []
  let actual: [number, number][] = []
  curva.forEach((c, i) => {
    if (c && !c.insuficiente) {
      actual.push([i, f(c)])
    } else if (actual.length) {
      out.push(actual)
      actual = []
    }
  })
  if (actual.length) out.push(actual)
  return out
}

/** Departamento válido de la URL: en mayúsculas, existente en el mapa o OTROS; si no, null. */
export function normalizarDepto(v: string | null, nombresMapa: Set<string>): string | null {
  if (!v) return null
  const d = v.trim().toUpperCase()
  return d === OTROS || nombresMapa.has(d) ? d : null
}

const pct = new Intl.NumberFormat('es-PE', { style: 'percent', minimumFractionDigits: 1, maximumFractionDigits: 1 })
const num = new Intl.NumberFormat('es-PE', { maximumFractionDigits: 1 })
const ent = new Intl.NumberFormat('es-PE', { maximumFractionDigits: 0 })

export const formato = {
  porcentaje: (v: number) => pct.format(v),
  numero: (v: number) => num.format(v),
  entero: (v: number) => ent.format(v),
  metrica: (v: number, m: Metrica) => (m === 'riesgo' ? pct.format(v) : num.format(v)),
}

/** Etiqueta legible de una etapa a partir de su identificador (nino_0_11 → «Niños 0–11»). */
export function etiquetaEtapa(e: string): string {
  if (e === TODAS) return 'Todas las edades'
  const nombres: Record<string, string> = {
    nino: 'Niños',
    adolescente: 'Adolescentes',
    joven: 'Jóvenes',
    adulto: 'Adultos',
    adulto_mayor: 'Adultos mayores',
  }
  const m = /^([a-z_]+?)_(\d+)_(\d+|mas)$/.exec(e)
  if (!m) return e
  const [, base, desde, hasta] = m
  return `${nombres[base!] ?? base} ${desde}${hasta === 'mas' ? '+' : '–' + hasta}`
}

export function etiquetaDepto(d: string): string {
  if (d === TODOS) return 'Perú'
  if (d === OTROS) return 'Otros (agrupados)'
  // Mayúscula inicial por palabra, salvo la preposición «de» (Madre de Dios).
  return d
    .toLowerCase()
    .replace(/(^|\s)\p{L}/gu, (l) => l.toUpperCase())
    .replace(/ De /g, ' de ')
}
