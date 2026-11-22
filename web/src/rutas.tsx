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

/**
 * Adónde ir después del login: la vista que se pidió, solo si es una ruta interna
 * (empieza con una «/», no con «//» ni «\\») y el rol nuevo puede verla; si no, al inicio.
 * Así un analista que entra después de que un admin salió desde /admin no
 * aterriza en un 403.
 */
export function destinoTrasLogin(desde: unknown, rol: Rol): string {
  if (
    typeof desde !== 'string' ||
    !desde.startsWith('/') ||
    desde.startsWith('//') ||
    desde.includes('\\') ||
    desde.startsWith('/login')
  ) {
    return '/'
  }
  const ruta = desde.split(/[?#]/)[0]
  const vista = VISTAS.find((v) => v.ruta === ruta)
  return vista && !vista.roles.includes(rol) ? '/' : desde
}
