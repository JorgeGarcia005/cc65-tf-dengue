import { Navigate, NavLink, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import type { Rol } from './api/cliente'
import { useSesion } from './sesion/Sesion'
import { VISTAS } from './rutas'
import Login from './paginas/Login'
import { NoEncontrado, Prohibido } from './paginas/Errores'

/** Exige sesión; sin ella, manda al login recordando a dónde iba. */
function ConSesion() {
  const { sesion } = useSesion()
  const ubicacion = useLocation()
  if (!sesion) return <Navigate to="/login" replace state={{ desde: ubicacion.pathname }} />
  return <Marco />
}

/** Exige uno de los roles; si no, muestra el 403 propio sin salir del marco. */
function ConRol({ roles, children }: { roles: Rol[]; children: React.ReactElement }) {
  const { puede } = useSesion()
  return puede(...roles) ? children : <Prohibido />
}

function Marco() {
  const { sesion, salir, puede } = useSesion()
  return (
    <div className="marco">
      <a className="saltar" href="#contenido">
        Saltar al contenido
      </a>
      <header className="cabecera">
        <div className="marca">
          <span className="marca-sello" aria-hidden="true" />
          <span className="marca-nombre">Vigía</span>
          <span className="marca-sub">dengue con signos de alarma</span>
        </div>
        <nav aria-label="Principal">
          <ul className="menu">
            {VISTAS.filter((v) => puede(...v.roles)).map((v) => (
              <li key={v.ruta}>
                <NavLink to={v.ruta} end={v.ruta === '/'}>
                  {v.menu}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className="usuario">
          <span className="usuario-correo">{sesion?.correo}</span>
          <span className={`etiqueta etiqueta-${sesion?.rol}`}>{sesion?.rol}</span>
          <button type="button" className="boton boton-fantasma" onClick={salir}>
            Salir
          </button>
        </div>
      </header>
      <main id="contenido" className="contenido">
        <Outlet />
      </main>
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<ConSesion />}>
        {VISTAS.map((v) => (
          <Route key={v.ruta} path={v.ruta} element={<ConRol roles={v.roles}>{v.elemento}</ConRol>} />
        ))}
        <Route path="*" element={<NoEncontrado />} />
      </Route>
    </Routes>
  )
}
