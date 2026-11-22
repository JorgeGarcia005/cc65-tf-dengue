import type { Feature, FeatureCollection, Geometry } from 'geojson'
import texto from '../datos/peru_departamental_simple.geojson?raw'

export interface PropiedadesDepto {
  NOMBDEP: string
}

/** Los 25 departamentos (fuente y licencia en src/datos/LEEME.md). */
export const DEPARTAMENTOS = JSON.parse(texto) as FeatureCollection<Geometry, PropiedadesDepto>

export type FeatureDepto = Feature<Geometry, PropiedadesDepto>
