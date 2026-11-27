import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react'
import { crearCliente, esSesion, type Cliente, type RespuestaLogin, type Rol, type Sesion } from '../api/cliente'

// La sesión vive en sessionStorage: se borra al cerrar la pestaña. La CSP
// estricta de cmd/web (sin scripts en línea) reduce el riesgo de XSS (ADR-0007).
const CLAVE = 'vigia.sesion'

function leerGuardada(ahora: number): Sesion | null {
  let v: unknown = null
  try {
    v = JSON.parse(sessionStorage.getItem(CLAVE) ?? 'null')
  } catch {
    v = null
  }
  if (esSesion(v) && v.expira > ahora) return v
  sessionStorage.removeItem(CLAVE) // vencida o corrupta: no se deja guardada
  return null
}

interface ValorSesion {
  sesion: Sesion | null
  api: Cliente
  /** URL base de la API (para abrir el WebSocket del panel admin). */
  apiUrl: string
  /** Motivo del último cierre forzado (token vencido), para mostrarlo en el login. */
  aviso: string | null
  entrar(correo: string, clave: string): Promise<Sesion>
  salir(): void
  puede(...roles: Rol[]): boolean
}

const Contexto = createContext<ValorSesion | null>(null)

export function ProveedorSesion({
  apiUrl,
  ahora = () => Date.now() / 1000,
  children,
}: {
  apiUrl: string
  ahora?: () => number
  children: ReactNode
}) {
  const [sesion, setSesion] = useState<Sesion | null>(() => leerGuardada(ahora()))
  const [aviso, setAviso] = useState<string | null>(null)
  const actual = useRef(sesion)
  actual.current = sesion

  const salir = useCallback(() => {
    sessionStorage.removeItem(CLAVE)
    actual.current = null
    setSesion(null)
  }, [])

  const api = useMemo(
    () =>
      crearCliente({
        baseUrl: apiUrl,
        token: () => actual.current?.token,
        alExpirar: (tokenFallido) => {
          // Un 401 tardío de una sesión anterior no cierra la sesión actual.
          if (actual.current?.token !== tokenFallido) return
          setAviso('Tu sesión venció. Vuelve a entrar.')
          salir()
        },
      }),
    [apiUrl, salir],
  )

  const entrar = useCallback(
    async (correo: string, clave: string) => {
      const r = await crearCliente({ baseUrl: apiUrl }).post<RespuestaLogin | undefined>('/api/v1/auth/login', {
        correo,
        clave,
      })
      if (!esSesion(r)) throw new Error('Respuesta de login inválida.')
      const nueva: Sesion = { token: r.token, correo: r.correo, rol: r.rol, expira: r.expira }
      sessionStorage.setItem(CLAVE, JSON.stringify(nueva))
      setAviso(null)
      setSesion(nueva)
      return nueva
    },
    [apiUrl],
  )

  const valor = useMemo<ValorSesion>(
    () => ({
      sesion,
      api,
      apiUrl,
      aviso,
      entrar,
      salir,
      puede: (...roles) => sesion !== null && roles.includes(sesion.rol),
    }),
    [sesion, api, apiUrl, aviso, entrar, salir],
  )
  return <Contexto.Provider value={valor}>{children}</Contexto.Provider>
}

export function useSesion(): ValorSesion {
  const v = useContext(Contexto)
  if (!v) throw new Error('useSesion fuera de ProveedorSesion')
  return v
}
