import type { Consulta, Contribucion, Prediccion } from '../api/tipos'
import { etiquetaDepto, etiquetaEtapa } from '../mapa/datos'

/** Mismas reglas que la API (internal/api/modelo.go, limpieza del loader). */
export const EDAD_MAX = 110
export const ANO_MAX = 2100

/** Lo que escribe la persona en el formulario, todavía como texto. */
export interface Formulario {
  departamento: string
  ano: string
  semana: string
  edad: string
  tipo_edad: string
  sexo: string
}

export type Errores = Partial<Record<keyof Formulario, string>>

const entero = (s: string) => (/^\s*-?\d+\s*$/.test(s) ? Number(s) : NaN)

/** Edad en años según tipo_edad, como ingest.EdadEnAnios. */
export function edadEnAnios(edad: number, tipo: string): number {
  return tipo === 'M' ? edad / 12 : tipo === 'D' ? edad / 365 : edad
}

/**
 * Valida el formulario antes de enviarlo. Devuelve la consulta o los errores
 * por campo; la API vuelve a validar (y su mensaje se muestra si rechaza).
 */
export function validar(f: Formulario, anoMin: number): { consulta?: Consulta; errores: Errores } {
  const e: Errores = {}
  const ano = entero(f.ano)
  const semana = entero(f.semana)
  const edad = entero(f.edad)
  if (!f.departamento) e.departamento = 'Elige un departamento.'
  if (!Number.isInteger(ano) || ano < anoMin || ano > ANO_MAX) e.ano = `El año debe ser un entero entre ${anoMin} y ${ANO_MAX} (el modelo se entrenó desde ${anoMin}).`
  if (!Number.isInteger(semana) || semana < 1 || semana > 53) e.semana = 'La semana epidemiológica va de 1 a 53.'
  if (!['A', 'M', 'D'].includes(f.tipo_edad)) e.tipo_edad = 'Elige años, meses o días.'
  if (!Number.isInteger(edad) || edad < 0) e.edad = 'La edad debe ser un entero mayor o igual a 0.'
  else if (!e.tipo_edad && edadEnAnios(edad, f.tipo_edad) > EDAD_MAX) e.edad = `La edad no puede pasar de ${EDAD_MAX} años.`
  if (f.sexo !== 'F' && f.sexo !== 'M') e.sexo = 'Elige el sexo.'
  if (Object.keys(e).length > 0) return { errores: e }
  return {
    errores: e,
    consulta: {
      departamento: f.departamento, ano, semana, edad,
      tipo_edad: f.tipo_edad as Consulta['tipo_edad'], sexo: f.sexo as Consulta['sexo'],
    },
  }
}

/** Nombre legible de una variable del modelo (features.Esquema.Nombres). */
export function etiquetaVariable(v: string, departamentos: string[] = []): string {
  if (v === 'edad_z') return 'Edad'
  if (v === 'sexo_m') return 'Sexo masculino'
  if (v === 'semana_sin' || v === 'semana_cos') return `Semana del año (${v === 'semana_sin' ? 'seno' : 'coseno'})`
  if (v === 'ano_z') return 'Año'
  if (v.startsWith('etapa_')) return `Grupo: ${etiquetaEtapa(v.slice(6))}`
  if (v.startsWith('depto_')) {
    const clave = v.slice(6).replace(/_/g, ' ').toUpperCase()
    const d = departamentos.find((x) => x === clave) ?? clave
    return `Departamento: ${etiquetaDepto(d)}`
  }
  return v
}

/** Contribuciones que mueven el logit (las one-hot en 0 aportan 0), de la de más peso a la de menos. */
export function contribucionesActivas(cs: Contribucion[]): Contribucion[] {
  return cs.filter((c) => c.aporte !== 0).sort((a, b) => Math.abs(b.aporte) - Math.abs(a.aporte))
}

/**
 * sesgo + Σ aportes en el orden del modelo: la misma suma que hace la API, así
 * que debe dar el logit exacto (no «casi»).
 */
export function sumaLogit(p: Prediccion): number {
  let z = p.sesgo
  for (const c of p.contribuciones) z += c.aporte
  return z
}

export const sigmoide = (z: number) => 1 / (1 + Math.exp(-z))
