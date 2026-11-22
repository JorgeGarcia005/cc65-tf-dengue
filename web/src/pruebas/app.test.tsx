import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import type { Sesion } from '../api/cliente'
import { ProveedorSesion } from '../sesion/Sesion'

const AHORA = 1_800_000_000

function montar(ruta: string, sesion?: Partial<Sesion>) {
  if (sesion) {
    sessionStorage.setItem(
      'vigia.sesion',
      JSON.stringify({ token: 't', correo: 'x@vigia.pe', rol: 'analista', expira: AHORA + 3600, ...sesion }),
    )
  }
  return render(
    <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
      <MemoryRouter initialEntries={[ruta]}>
        <App />
      </MemoryRouter>
    </ProveedorSesion>,
  )
}

const menu = () => within(screen.getByRole('navigation', { name: 'Principal' }))

describe('guardas por rol', () => {
  it('sin sesión manda al login', () => {
    montar('/mapa')
    expect(screen.getByRole('heading', { name: 'Vigía' })).toBeInTheDocument()
    expect(screen.getByLabelText('Correo')).toBeInTheDocument()
  })

  it('una sesión vencida cuenta como sin sesión', () => {
    montar('/', { expira: AHORA - 1 })
    expect(screen.getByLabelText('Clave')).toBeInTheDocument()
  })

  it('el analista ve riesgo, predicción y evaluación, pero no el clúster', () => {
    montar('/', { rol: 'analista' })
    for (const v of ['Inicio', 'Mapa', 'Predicción', 'Evaluación']) {
      expect(menu().getByRole('link', { name: v })).toBeInTheDocument()
    }
    expect(menu().queryByRole('link', { name: 'Clúster' })).toBeNull()
  })

  it('el analista que entra a /admin recibe el 403 propio', () => {
    montar('/admin', { rol: 'analista' })
    expect(screen.getByRole('heading', { name: 'Esta vista no es para tu rol' })).toBeInTheDocument()
  })

  it('el admin ve el clúster', () => {
    montar('/admin', { rol: 'admin' })
    expect(menu().getByRole('link', { name: 'Clúster' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Clúster y entrenamiento en vivo' })).toBeInTheDocument()
  })

  it('salir borra la sesión y vuelve al login', async () => {
    montar('/', { rol: 'admin' })
    await userEvent.click(screen.getByRole('button', { name: 'Salir' }))
    expect(sessionStorage.getItem('vigia.sesion')).toBeNull()
    expect(screen.getByLabelText('Correo')).toBeInTheDocument()
  })

  it('ruta desconocida con sesión → 404', () => {
    montar('/no-existe', { rol: 'analista' })
    expect(screen.getByRole('heading', { name: 'No existe esta página' })).toBeInTheDocument()
  })
})

describe('login', () => {
  async function escribir() {
    await userEvent.type(screen.getByLabelText('Correo'), 'ana@diresa.pe')
    await userEvent.type(screen.getByLabelText('Clave'), 'secreta')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
  }

  it('con credenciales correctas guarda la sesión y vuelve a la vista pedida', async () => {
    const f = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ token: 'jwt', correo: 'ana@diresa.pe', rol: 'analista', expira: AHORA + 3600 })),
    )
    vi.stubGlobal('fetch', f)
    montar('/mapa')
    await escribir()
    expect(await screen.findByRole('heading', { name: 'Mapa de riesgo' })).toBeInTheDocument()
    const [url, init] = f.mock.calls[0]!
    expect(url).toBe('http://api:8080/api/v1/auth/login')
    expect(JSON.parse(init.body)).toEqual({ correo: 'ana@diresa.pe', clave: 'secreta' })
    expect(JSON.parse(sessionStorage.getItem('vigia.sesion')!).token).toBe('jwt')
  })

  it('con clave incorrecta muestra un error genérico y limpia la clave', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"error":"x"}', { status: 401 })))
    montar('/')
    await escribir()
    expect(await screen.findByRole('alert')).toHaveTextContent('Correo o clave incorrectos.')
    expect(screen.getByLabelText('Clave')).toHaveValue('')
  })

  it('con rate limit dice cuánto esperar', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('{"error":"x"}', { status: 429, headers: { 'Retry-After': '30' } })),
    )
    montar('/')
    await escribir()
    expect(await screen.findByRole('alert')).toHaveTextContent('Vuelve a probar en 30 s.')
  })
})
