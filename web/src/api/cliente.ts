// Cliente HTTP de la API del coordinador. El token va siempre en la cabecera
// Authorization (nunca en cookies: sin CSRF, ADR-0007). Un 401 en una ruta
// protegida significa token vencido o inválido: se avisa para cerrar la sesión.

export type Rol = 'admin' | 'analista'

export interface Sesion {
  token: string
  correo: string
  rol: Rol
  /** Vencimiento del token, en segundos Unix. */
  expira: number
}

/** Error de la API con su código HTTP y el mensaje en español del servidor. */
export class ErrorApi extends Error {
  readonly estado: number
  /** Segundos de Retry-After (429), si vino. */
  readonly reintentarEn?: number

  constructor(estado: number, mensaje: string, reintentarEn?: number) {
    super(mensaje)
    this.name = 'ErrorApi'
    this.estado = estado
    this.reintentarEn = reintentarEn
  }
}

export interface OpcionesCliente {
  baseUrl: string
  token?: () => string | undefined
  /** Se llama ante un 401 de una petición que llevaba token, con ese token. */
  alExpirar?: (tokenFallido: string) => void
}

export interface Cliente {
  get<T>(ruta: string): Promise<T>
  post<T>(ruta: string, cuerpo?: unknown): Promise<T>
  /** Como post, pero también devuelve las cabeceras y la latencia vista por el navegador. */
  postMedido<T>(ruta: string, cuerpo?: unknown): Promise<Medida<T>>
}

/** Respuesta con lo que mide el navegador: el tiempo desde el envío hasta leer el cuerpo. */
export interface Medida<T> {
  datos: T
  cabeceras: Headers
  ms: number
}

export function crearCliente({ baseUrl, token, alExpirar }: OpcionesCliente): Cliente {
  const base = baseUrl.replace(/\/+$/, '')

  async function pedir<T>(metodo: 'GET' | 'POST', ruta: string, cuerpo?: unknown): Promise<Medida<T>> {
    const cabeceras: Record<string, string> = { Accept: 'application/json' }
    const t = token?.()
    if (t) cabeceras.Authorization = `Bearer ${t}`
    if (cuerpo !== undefined) cabeceras['Content-Type'] = 'application/json'

    let resp: Response
    const inicio = performance.now()
    try {
      resp = await fetch(base + ruta, {
        method: metodo,
        headers: cabeceras,
        body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo),
        credentials: 'omit',
      })
    } catch {
      throw new ErrorApi(0, 'No se pudo conectar con la API.')
    }

    const texto = await resp.text()
    const ms = performance.now() - inicio
    let datos: unknown = undefined
    if (texto) {
      try {
        datos = JSON.parse(texto)
      } catch {
        datos = undefined
      }
    }
    if (!resp.ok) {
      if (resp.status === 401 && t) alExpirar?.(t)
      const msg =
        datos && typeof datos === 'object' && 'error' in datos && typeof datos.error === 'string'
          ? datos.error
          : `Error ${resp.status}`
      const reintento = Number(resp.headers.get('Retry-After'))
      throw new ErrorApi(resp.status, msg, Number.isFinite(reintento) && reintento > 0 ? reintento : undefined)
    }
    return { datos: datos as T, cabeceras: resp.headers, ms }
  }

  return {
    get: async (ruta) => (await pedir<never>('GET', ruta)).datos,
    post: async (ruta, cuerpo) => (await pedir<never>('POST', ruta, cuerpo)).datos,
    postMedido: (ruta, cuerpo) => pedir('POST', ruta, cuerpo),
  }
}

/** Respuesta de POST /api/v1/auth/login (#60). */
export interface RespuestaLogin {
  token: string
  correo: string
  rol: Rol
  expira: number
}

export function esSesion(v: unknown): v is Sesion {
  if (!v || typeof v !== 'object') return false
  const s = v as Record<string, unknown>
  return (
    typeof s.token === 'string' &&
    s.token.trim().length > 0 &&
    typeof s.correo === 'string' &&
    s.correo.trim().length > 0 &&
    (s.rol === 'admin' || s.rol === 'analista') &&
    typeof s.expira === 'number' &&
    Number.isFinite(s.expira)
  )
}
