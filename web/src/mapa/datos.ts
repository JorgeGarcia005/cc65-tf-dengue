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

  /** Semanas que aparecen en el cubo (1–52), en orden. */
  semanas(): number[] {
    return [...new Set(this.cubo.celdas.map((c) => c.semana).filter((s) => s > 0))].sort((a, b) => a - b)
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

export interface Escala {
  color(v: number): string
  /** Cortes entre clases (4 valores para 5 clases). */
  cortes: number[]
}

/**
 * Escala por cuantiles sobre los departamentos del modelo con datos suficientes:
 * cada color agrupa una quinta parte. OTROS y TODOS no entran al cálculo.
 */
export function escalaPara(indice: IndiceRiesgo, semana: number, etapa: string, m: Metrica): Escala {
  const valores = [...indice.modelo]
    .filter((d) => d !== OTROS)
    .map((d) => indice.celda(d, semana, etapa))
    .filter((c): c is Celda => !!c && !c.insuficiente)
    .map((c) => valor(c, m)!)
  const q = scaleQuantile<string>().domain(valores.length ? valores : [0, 1]).range([...COLORES])
  return { color: (v) => q(v), cortes: q.quantiles() }
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
  return d.toLowerCase().replace(/(^|\s)\p{L}/gu, (l) => l.toUpperCase())
}
