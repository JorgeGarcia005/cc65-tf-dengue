import { useState, type FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { ErrorApi } from '../api/cliente'
import { useSesion } from '../sesion/Sesion'
import { destinoTrasLogin } from '../rutas'

function mensajeDe(e: unknown): string {
  if (e instanceof ErrorApi) {
    if (e.estado === 401) return 'Correo o clave incorrectos.'
    if (e.estado === 429)
      return e.reintentarEn
        ? `Demasiados intentos. Vuelve a probar en ${e.reintentarEn} s.`
        : 'Demasiados intentos. Espera un momento.'
    if (e.estado === 0) return e.message
    return `No se pudo entrar (${e.message}).`
  }
  return 'No se pudo entrar.'
}

export default function Login() {
  const { sesion, entrar, aviso } = useSesion()
  const navegar = useNavigate()
  const desde = (useLocation().state as { desde?: unknown } | null)?.desde
  const [correo, setCorreo] = useState('')
  const [clave, setClave] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  if (sesion) return <Navigate to={destinoTrasLogin(desde, sesion.rol)} replace />

  async function enviar(ev: FormEvent) {
    ev.preventDefault()
    setError(null)
    setEnviando(true)
    try {
      const nueva = await entrar(correo.trim(), clave)
      navegar(destinoTrasLogin(desde, nueva.rol), { replace: true })
    } catch (e) {
      setError(mensajeDe(e))
      setClave('')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <main className="login">
      <section className="tarjeta login-tarjeta" aria-labelledby="titulo-login">
        <p className="antetitulo">CC65 · Vigilancia epidemiológica</p>
        <h1 id="titulo-login" className="titulo">
          Vigía
        </h1>
        <p className="bajada">
          Dónde, cuándo y en qué grupos de edad reforzar camas, vigilancia y cerco por dengue con signos de alarma.
        </p>
        {aviso && (
          <p className="aviso" role="status">
            {aviso}
          </p>
        )}
        <form onSubmit={enviar} noValidate>
          <label htmlFor="correo">Correo</label>
          <input
            id="correo"
            type="email"
            autoComplete="username"
            required
            value={correo}
            onChange={(e) => setCorreo(e.target.value)}
          />
          <label htmlFor="clave">Clave</label>
          <input
            id="clave"
            type="password"
            autoComplete="current-password"
            required
            value={clave}
            onChange={(e) => setClave(e.target.value)}
          />
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
          <button type="submit" className="boton" disabled={enviando || !correo.trim() || !clave}>
            {enviando ? 'Entrando…' : 'Entrar'}
          </button>
        </form>
      </section>
    </main>
  )
}
