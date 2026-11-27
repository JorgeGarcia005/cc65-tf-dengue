// Mapa que se repinta en vivo durante el entrenamiento (#69, plan §2 vista 5).
//
// El modelo es una regresión logística con el departamento en one-hot, así que
// su efecto es aditivo en el logit: el peso w_d de `depto_<d>`. Con el sesgo en
// el modelo esos pesos solo están definidos salvo un desplazamiento común (lo
// absorbe el sesgo), por eso se centran en el promedio de los departamentos:
//   Δ_d = w_d − media(w_k)      razón de odds frente al departamento promedio = e^Δ_d
// Δ_d no depende de la edad, el sexo ni la semana del caso: es lo que el modelo
// aprendió del lugar, y cambia en cada época mientras los pesos convergen.
// Un departamento sin columna propia (pocos casos en train) usa la de OTROS.

export interface EfectoDepto {
  /** Variable del modelo que se usó: depto_<d> o depto_otros. */
  variable: string
  agrupado: boolean
  delta: number
  razonOdds: number
}

/** Nombre del mapa (MADRE DE DIOS) → variable del modelo (depto_madre_de_dios), como features.Esquema. */
export const varDepto = (nombreMapa: string) => 'depto_' + nombreMapa.trim().toLowerCase().replace(/ /g, '_')

const VAR_OTROS = 'depto_otros'
/** Piso de la escala de color: con todos los pesos en 0 (antes de la época 1) no colapsa a [0, 0]. */
export const PISO_ESCALA = 0.05

/**
 * Efecto de cada departamento del mapa a partir de los pesos (variables + sesgo
 * al final, como los eventos `epoca` y GET /api/v1/modelo). Devuelve null si
 * los pesos no cuadran con las variables o no hay variables de departamento.
 */
export function efectosDepartamento(
  pesos: number[] | null,
  variables: string[],
  deptosMapa: string[],
): { efectos: Map<string, EfectoDepto>; max: number } | null {
  if (!pesos || pesos.length !== variables.length + 1 || !pesos.every(Number.isFinite)) return null
  const indices = variables.flatMap((v, i) => (v.startsWith('depto_') ? [i] : []))
  if (indices.length === 0) return null
  const media = indices.reduce((s, i) => s + pesos[i]!, 0) / indices.length
  const iOtros = variables.indexOf(VAR_OTROS)
  const efectos = new Map<string, EfectoDepto>()
  let max = PISO_ESCALA
  for (const d of deptosMapa) {
    const propia = variables.indexOf(varDepto(d))
    const i = propia >= 0 ? propia : iOtros
    if (i < 0) continue
    const delta = pesos[i]! - media
    max = Math.max(max, Math.abs(delta))
    efectos.set(d, { variable: variables[i]!, agrupado: propia < 0, delta, razonOdds: Math.exp(delta) })
  }
  return { efectos, max }
}

/** Pesos (variables + sesgo) del modelo publicado, para pintar el mapa antes de la primera época. */
export function pesosDelModelo(m: { sesgo: number; variables: { nombre?: string; peso: number }[] } | null): number[] | null {
  return m ? [...m.variables.map((v) => v.peso), m.sesgo] : null
}
