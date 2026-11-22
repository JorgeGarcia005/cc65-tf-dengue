import { Link } from 'react-router-dom'
import { useSesion } from '../sesion/Sesion'

export function Prohibido() {
  const { sesion } = useSesion()
  return (
    <section className="tarjeta pagina-error" aria-labelledby="titulo-403">
      <p className="antetitulo">403</p>
      <h1 id="titulo-403" className="titulo">
        Esta vista no es para tu rol
      </h1>
      <p>
        Entraste como <strong>{sesion?.rol}</strong>. El panel del clúster solo lo usa el rol admin.
      </p>
      <Link className="boton" to="/">
        Volver al inicio
      </Link>
    </section>
  )
}

export function NoEncontrado() {
  return (
    <section className="tarjeta pagina-error" aria-labelledby="titulo-404">
      <p className="antetitulo">404</p>
      <h1 id="titulo-404" className="titulo">
        No existe esta página
      </h1>
      <Link className="boton" to="/">
        Volver al inicio
      </Link>
    </section>
  )
}
