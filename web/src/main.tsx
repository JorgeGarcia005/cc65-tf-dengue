import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import '@fontsource/instrument-serif/400.css'
import '@fontsource/instrument-serif/400-italic.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/700.css'
import './estilos/vigia.css'
import App from './App'
import { ProveedorSesion } from './sesion/Sesion'
import { cargarConfig } from './config'

const raiz = createRoot(document.getElementById('raiz')!)

cargarConfig()
  .then(({ apiUrl }) =>
    raiz.render(
      <StrictMode>
        <ProveedorSesion apiUrl={apiUrl}>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </ProveedorSesion>
      </StrictMode>,
    ),
  )
  .catch((e: unknown) =>
    raiz.render(
      <main className="login">
        <p className="error" role="alert">
          No se pudo cargar la configuración: {String(e)}
        </p>
      </main>,
    ),
  )
