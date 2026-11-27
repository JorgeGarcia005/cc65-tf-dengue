import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import type { Sesion } from '../api/cliente'
import { ProveedorSesion, useSesion } from '../sesion/Sesion'
import { destinoTrasLogin } from '../rutas'

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

describe('sesión y destino tras el login (PR-79 H3, H4)', () => {
  const okLogin = (token: string, rol: 'admin' | 'analista') =>
    new Response(JSON.stringify({ token, correo: `${rol}@vigia.pe`, rol, expira: AHORA + 3600 }))

  function Sonda() {
    const { api } = useSesion()
    return (
      <button type="button" onClick={() => api.get('/api/v1/modelo').catch(() => undefined)}>
        sonda
      </button>
    )
  }

  function montarConSonda(ruta: string | { pathname: string; state: unknown }, sesion?: Partial<Sesion>) {
    if (sesion) {
      sessionStorage.setItem(
        'vigia.sesion',
        JSON.stringify({ token: 't1', correo: 'x@vigia.pe', rol: 'analista', expira: AHORA + 3600, ...sesion }),
      )
    }
    return render(
      <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
        <MemoryRouter initialEntries={[ruta]}>
          <Sonda />
          <App />
        </MemoryRouter>
      </ProveedorSesion>,
    )
  }

  it('un 401 de la API cierra la sesión, vuelve al login y avisa', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"error":"vencido"}', { status: 401 })))
    montarConSonda('/mapa', { rol: 'analista' })
    await userEvent.click(screen.getByRole('button', { name: 'sonda' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Tu sesión venció. Vuelve a entrar.')
    expect(screen.getByLabelText('Correo')).toBeInTheDocument()
    expect(sessionStorage.getItem('vigia.sesion')).toBeNull()
  })

  it('un 401 tardío de la sesión anterior no cierra la sesión nueva', async () => {
    let responderViejo: (r: Response) => void = () => undefined
    // Desde #69 el panel admin también pide datos al montar: el mock separa la
    // petición de la sonda (la que queda pendiente con t1) del login y del resto.
    let sondaEnCurso = false
    const f = vi.fn((url: string, init?: RequestInit) => {
      if (url.endsWith('/api/v1/auth/login')) return Promise.resolve(okLogin('t2', 'analista'))
      if (sondaEnCurso) {
        sondaEnCurso = false
        expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer t1')
        return new Promise<Response>((ok) => (responderViejo = ok))
      }
      return Promise.resolve(new Response('{}'))
    })
    vi.stubGlobal('fetch', f)
    montarConSonda('/admin', { rol: 'admin' })
    sondaEnCurso = true
    await userEvent.click(screen.getByRole('button', { name: 'sonda' }))
    await userEvent.click(screen.getByRole('button', { name: 'Salir' }))
    await userEvent.type(screen.getByLabelText('Correo'), 'analista@vigia.pe')
    await userEvent.type(screen.getByLabelText('Clave'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByRole('heading', { name: 'Dónde y cuándo reforzar' })).toBeInTheDocument()
    responderViejo(new Response('{"error":"vencido"}', { status: 401 }))
    await new Promise((r) => setTimeout(r, 20))
    expect(JSON.parse(sessionStorage.getItem('vigia.sesion')!).token).toBe('t2')
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('admin sale desde /admin y el analista que entra aterriza en el inicio, no en 403', async () => {
    // Una Response nueva por llamada: desde #69 el panel admin también pide
    // datos al montar, y un cuerpo ya leído no se puede volver a leer.
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(okLogin('t2', 'analista'))))
    montarConSonda('/admin', { rol: 'admin' })
    await userEvent.click(screen.getByRole('button', { name: 'Salir' }))
    await userEvent.type(screen.getByLabelText('Correo'), 'analista@vigia.pe')
    await userEvent.type(screen.getByLabelText('Clave'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByRole('heading', { name: 'Dónde y cuándo reforzar' })).toBeInTheDocument()
  })

  it.each(['//evil.example', '/\\evil.example', 'https://evil.example', '/login', '/admin'])(
    'un destino no seguro o sin permiso (%s) lleva al inicio',
    async (desde) => {
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okLogin('t', 'analista')))
      montarConSonda({ pathname: '/login', state: { desde } })
      await userEvent.type(screen.getByLabelText('Correo'), 'a@vigia.pe')
      await userEvent.type(screen.getByLabelText('Clave'), 'x')
      await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
      expect(await screen.findByRole('heading', { name: 'Dónde y cuándo reforzar' })).toBeInTheDocument()
    },
  )

  it('conserva la query de la vista pedida', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okLogin('t', 'analista')))
    montarConSonda('/mapa?semana=20&etapa=adulto_30_59')
    await userEvent.type(screen.getByLabelText('Correo'), 'a@vigia.pe')
    await userEvent.type(screen.getByLabelText('Clave'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByRole('heading', { name: 'Mapa de riesgo' })).toBeInTheDocument()
    expect(destinoTrasLogin('/mapa?semana=20', 'analista')).toBe('/mapa?semana=20')
  })

  it('un correo de solo espacios no habilita Entrar', async () => {
    montarConSonda('/')
    await userEvent.type(screen.getByLabelText('Correo'), '   ')
    await userEvent.type(screen.getByLabelText('Clave'), 'x')
    expect(screen.getByRole('button', { name: 'Entrar' })).toBeDisabled()
  })

  it('un login 200 sin cuerpo válido muestra error, sin romper', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>proxy</html>', { status: 200 })))
    montarConSonda('/')
    await userEvent.type(screen.getByLabelText('Correo'), 'a@vigia.pe')
    await userEvent.type(screen.getByLabelText('Clave'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Entrar' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('No se pudo entrar.')
    expect(sessionStorage.getItem('vigia.sesion')).toBeNull()
  })

  it('una sesión guardada corrupta o con token vacío se borra', () => {
    sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: '', correo: 'x', rol: 'admin', expira: AHORA + 60 }))
    montarConSonda('/')
    expect(screen.getByLabelText('Correo')).toBeInTheDocument()
    expect(sessionStorage.getItem('vigia.sesion')).toBeNull()
  })
})
