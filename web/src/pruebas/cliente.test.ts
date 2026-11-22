import { describe, expect, it, vi } from 'vitest'
import { crearCliente, ErrorApi, esSesion } from '../api/cliente'

function respuesta(estado: number, cuerpo: unknown, cabeceras: Record<string, string> = {}) {
  return new Response(cuerpo === undefined ? '' : JSON.stringify(cuerpo), { status: estado, headers: cabeceras })
}

describe('cliente de la API', () => {
  it('manda el token en Authorization, sin cookies, y decodifica el JSON', async () => {
    const f = vi.fn().mockResolvedValue(respuesta(200, { ok: 1 }))
    vi.stubGlobal('fetch', f)
    const api = crearCliente({ baseUrl: 'http://api:8080/', token: () => 'abc' })
    await expect(api.get('/api/v1/modelo')).resolves.toEqual({ ok: 1 })
    const [url, init] = f.mock.calls[0]!
    expect(url).toBe('http://api:8080/api/v1/modelo')
    expect(init.headers.Authorization).toBe('Bearer abc')
    expect(init.credentials).toBe('omit')
  })

  it('esSesion exige token, correo y vencimiento válidos', () => {
    const base = { token: 't', correo: 'a@b.pe', rol: 'admin', expira: 1 }
    expect(esSesion(base)).toBe(true)
    for (const malo of [{ token: '' }, { token: '  ' }, { correo: '' }, { expira: Number.NaN }, { expira: Infinity }, { rol: 'root' }]) {
      expect(esSesion({ ...base, ...malo })).toBe(false)
    }
    expect(esSesion(undefined)).toBe(false)
  })

  it('un 401 con token avisa que la sesión venció', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respuesta(401, { error: 'token inválido o vencido' })))
    const alExpirar = vi.fn()
    const api = crearCliente({ baseUrl: 'http://api', token: () => 't', alExpirar })
    await expect(api.get('/x')).rejects.toMatchObject({ estado: 401, message: 'token inválido o vencido' })
    expect(alExpirar).toHaveBeenCalledExactlyOnceWith('t')
  })

  it('un 401 sin token (login fallido) no cierra ninguna sesión', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respuesta(401, { error: 'credenciales' })))
    const alExpirar = vi.fn()
    await expect(crearCliente({ baseUrl: 'http://api', alExpirar }).post('/login', {})).rejects.toBeInstanceOf(ErrorApi)
    expect(alExpirar).not.toHaveBeenCalled()
  })

  it('lee Retry-After de un 429 y traduce un fallo de red', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(respuesta(429, { error: 'espera' }, { 'Retry-After': '42' })))
    await expect(crearCliente({ baseUrl: 'http://api' }).post('/login', {})).rejects.toMatchObject({
      estado: 429,
      reintentarEn: 42,
    })
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    await expect(crearCliente({ baseUrl: 'http://api' }).get('/x')).rejects.toMatchObject({ estado: 0 })
  })
})
