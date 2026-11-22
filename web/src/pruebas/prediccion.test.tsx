import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import type { Contribucion, Prediccion } from '../api/tipos'
import { formato } from '../mapa/datos'
import { contribucionesActivas, edadEnAnios, etiquetaVariable, sumaLogit, validar, type Formulario } from '../prediccion/datos'
import { ProveedorSesion } from '../sesion/Sesion'

const AHORA = 1_800_000_000
const MODELO = { version: 'abc123def456', ano_min: 2011, departamentos: ['LORETO', 'PIURA', 'OTROS'], umbral: 0.5 }

/** Respuesta con la forma de la API: el logit es la suma en orden, como en Go. */
function prediccion(departamento_modelo = 'PIURA'): Prediccion {
  const contribuciones: Contribucion[] = [
    { variable: 'edad_z', valor: -1.25, peso: 0.4, aporte: -1.25 * 0.4 },
    { variable: 'etapa_nino_0_11', valor: 1, peso: 0.9, aporte: 0.9 },
    { variable: 'etapa_adulto_30_59', valor: 0, peso: -0.3, aporte: 0 },
    { variable: 'sexo_m', valor: 0, peso: 0.05, aporte: 0 },
    { variable: 'semana_sin', valor: 0.7, peso: 0.2, aporte: 0.7 * 0.2 },
    { variable: 'depto_piura', valor: 1, peso: 0.35, aporte: 0.35 },
  ]
  const sesgo = -2.1
  let logit = sesgo
  for (const c of contribuciones) logit += c.aporte
  const p = 1 / (1 + Math.exp(-logit))
  return {
    riesgo: 0.137, probabilidad_ponderada: p, clase: p >= 0.5 ? 1 : 0, etiqueta: 'sin signos de alarma', umbral: 0.5,
    departamento_modelo, version_modelo: MODELO.version, logit, sesgo, contribuciones,
  }
}

function montar(respuesta: () => Response) {
  sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'a@b.pe', rol: 'analista', expira: AHORA + 3600 }))
  const f = vi.fn((url: string, _init?: RequestInit) => {
    if (url.endsWith('/api/v1/modelo')) return Promise.resolve(new Response(JSON.stringify(MODELO)))
    if (url.endsWith('/api/v1/predicciones')) return Promise.resolve(respuesta())
    return Promise.resolve(new Response('{}', { status: 404 }))
  })
  vi.stubGlobal('fetch', f)
  render(
    <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
      <MemoryRouter initialEntries={['/prediccion']}>
        <App />
      </MemoryRouter>
    </ProveedorSesion>,
  )
  return f
}

const posts = (f: ReturnType<typeof montar>) => f.mock.calls.filter((c) => String(c[0]).endsWith('/api/v1/predicciones'))

async function llenar(u: ReturnType<typeof userEvent.setup>, depto = 'PIURA') {
  await u.selectOptions(screen.getByLabelText('Departamento'), depto)
  const ano = screen.getByLabelText('Año')
  await u.clear(ano)
  await u.type(ano, '2024')
  await u.type(screen.getByLabelText(/Semana epidemiológica/), '14')
  await u.type(screen.getByLabelText('Edad'), '8')
  await u.click(screen.getByRole('radio', { name: 'Femenino' }))
}

describe('predicción: validación del formulario', () => {
  const ok: Formulario = { departamento: 'PIURA', ano: '2024', semana: '14', edad: '8', tipo_edad: 'A', sexo: 'F' }

  it('acepta un caso válido y lo convierte en la consulta de la API', () => {
    expect(validar(ok, 2011)).toEqual({ errores: {}, consulta: { departamento: 'PIURA', ano: 2024, semana: 14, edad: 8, tipo_edad: 'A', sexo: 'F' } })
  })

  it('rechaza cada campo fuera de las reglas de la API', () => {
    const casos: [Partial<Formulario>, keyof Formulario][] = [
      [{ departamento: '' }, 'departamento'],
      [{ ano: '2010' }, 'ano'],
      [{ ano: '2101' }, 'ano'],
      [{ ano: '2024.5' }, 'ano'],
      [{ semana: '0' }, 'semana'],
      [{ semana: '54' }, 'semana'],
      [{ semana: 'diez' }, 'semana'],
      [{ edad: '-1' }, 'edad'],
      [{ edad: '3.5' }, 'edad'],
      [{ edad: '111' }, 'edad'],
      [{ edad: '1333', tipo_edad: 'M' }, 'edad'], // 111 años en meses
      [{ tipo_edad: 'X' }, 'tipo_edad'],
      [{ sexo: '' }, 'sexo'],
    ]
    for (const [cambio, campo] of casos) {
      const r = validar({ ...ok, ...cambio }, 2011)
      expect(r.consulta, JSON.stringify(cambio)).toBeUndefined()
      expect(Object.keys(r.errores), JSON.stringify(cambio)).toEqual([campo])
    }
    expect(edadEnAnios(1320, 'M')).toBe(110)
    expect(validar({ ...ok, edad: '1320', tipo_edad: 'M' }, 2011).consulta).toBeDefined()
  })

  it('con errores no llama a la API y marca los campos', async () => {
    const u = userEvent.setup()
    const f = montar(() => new Response(JSON.stringify(prediccion())))
    await screen.findByText(/versión abc123def456/)
    await u.click(screen.getByRole('button', { name: 'Calcular riesgo' }))
    expect(posts(f)).toHaveLength(0)
    expect(screen.getByLabelText('Departamento')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByText('Elige un departamento.')).toBeInTheDocument()
    expect(screen.getByText('La semana epidemiológica va de 1 a 53.')).toBeInTheDocument()
  })
})

describe('predicción: resultado', () => {
  it('muestra el riesgo calibrado, la latencia, la caché y las contribuciones que suman el logit', async () => {
    const u = userEvent.setup()
    const pred = prediccion()
    const f = montar(() => new Response(JSON.stringify(pred), { headers: { 'X-Cache': 'MISS' } }))
    await screen.findByText(/versión abc123def456/)
    await llenar(u)
    await u.click(screen.getByRole('button', { name: 'Calcular riesgo' }))

    expect(await screen.findByText(formato.porcentaje(pred.riesgo))).toHaveAttribute('data-resultado', 'riesgo')
    const [, init] = posts(f)[0]!
    expect(JSON.parse(String(init!.body))).toEqual({ departamento: 'PIURA', ano: 2024, semana: 14, edad: 8, tipo_edad: 'A', sexo: 'F' })
    expect((init!.headers as Record<string, string>).Authorization).toBe('Bearer t')
    expect(document.querySelector('[data-resultado="latencia"]')!.textContent).toMatch(/^\d+([.,]\d)? ms$/)
    expect(document.querySelector('[data-resultado="cache"]')).toHaveTextContent('calculada y guardada en la caché')

    // Solo las variables que aportan, de la de más peso a la de menos.
    const orden = [...document.querySelectorAll('li[data-variable]')].map((l) => l.getAttribute('data-variable'))
    expect(orden).toEqual(contribucionesActivas(pred.contribuciones).map((c) => c.variable))
    expect(orden).toEqual(['etapa_nino_0_11', 'edad_z', 'depto_piura', 'semana_sin'])
    expect(document.querySelector('li[data-variable="sexo_m"]')).toBeNull()
    expect(document.querySelector('[data-resultado="suma"]')).toHaveTextContent('✓ cuadra')
    expect(sumaLogit(pred)).toBe(pred.logit)
  })

  it('una respuesta desde Redis lo dice', async () => {
    const u = userEvent.setup()
    montar(() => new Response(JSON.stringify(prediccion()), { headers: { 'X-Cache': 'HIT' } }))
    await screen.findByText(/versión abc123def456/)
    await llenar(u)
    await u.click(screen.getByRole('button', { name: 'Calcular riesgo' }))
    expect(await screen.findByText('desde la caché (Redis)')).toBeInTheDocument()
  })

  it('un departamento sin columna propia avisa que usa «Otros»', async () => {
    const u = userEvent.setup()
    montar(() => new Response(JSON.stringify(prediccion('OTROS'))))
    await screen.findByText(/versión abc123def456/)
    await llenar(u, 'AMAZONAS')
    await u.click(screen.getByRole('button', { name: 'Calcular riesgo' }))
    expect(await screen.findByText(/Amazonas no tiene columna propia/)).toBeInTheDocument()
    expect(document.querySelector('[data-resultado="cache"]')).toHaveTextContent('sin caché')
  })

  it('si la suma no diera el logit, se ve', () => {
    const p = prediccion()
    expect(sumaLogit({ ...p, logit: p.logit + 1e-12 })).not.toBe(p.logit + 1e-12)
  })

  it('el rechazo de la API se muestra con su mensaje', async () => {
    const u = userEvent.setup()
    montar(() => new Response(JSON.stringify({ error: 'consulta inválida: año 2024 fuera de 2025–2100' }), { status: 400 }))
    await screen.findByText(/versión abc123def456/)
    await llenar(u)
    await u.click(screen.getByRole('button', { name: 'Calcular riesgo' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('año 2024 fuera de 2025–2100')
  })

  it('nombres legibles de las variables del modelo', () => {
    expect(etiquetaVariable('edad_z')).toBe('Edad')
    expect(etiquetaVariable('etapa_adulto_mayor_60_mas')).toBe('Grupo: Adultos mayores 60+')
    expect(etiquetaVariable('depto_madre_de_dios', ['MADRE DE DIOS'])).toBe('Departamento: Madre de Dios')
    expect(etiquetaVariable('semana_cos')).toBe('Semana del año (coseno)')
  })
})
