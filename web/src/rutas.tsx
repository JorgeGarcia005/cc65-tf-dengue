import type { ReactElement } from 'react'
import type { Rol } from './api/cliente'
import Inicio from './paginas/Inicio'
import Mapa from './paginas/Mapa'
import Prediccion from './paginas/Prediccion'
import Evaluacion from './paginas/Evaluacion'
import Admin from './paginas/Admin'

export interface Vista {
  ruta: string
  titulo: string
  /** Texto corto del menú. */
  menu: string
  roles: Rol[]
  elemento: ReactElement
}

// Matriz de vistas × rol (ADR-0007 y plan TB2): el analista de salud pública ve
// riesgo, predicción y evaluación; el clúster es solo del admin.
export const VISTAS: Vista[] = [
  { ruta: '/', titulo: 'Dónde y cuándo reforzar', menu: 'Inicio', roles: ['analista', 'admin'], elemento: <Inicio /> },
  { ruta: '/mapa', titulo: 'Mapa de riesgo', menu: 'Mapa', roles: ['analista', 'admin'], elemento: <Mapa /> },
  { ruta: '/prediccion', titulo: 'Predicción de un caso', menu: 'Predicción', roles: ['analista', 'admin'], elemento: <Prediccion /> },
  { ruta: '/evaluacion', titulo: 'Evaluación del modelo', menu: 'Evaluación', roles: ['analista', 'admin'], elemento: <Evaluacion /> },
  { ruta: '/admin', titulo: 'Clúster y entrenamiento en vivo', menu: 'Clúster', roles: ['admin'], elemento: <Admin /> },
]
