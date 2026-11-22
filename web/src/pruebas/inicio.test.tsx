import { readFileSync } from 'node:fs'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import { TODAS, TODOS, type DocumentoEvaluacion } from '../api/tipos'
import { enlaceMapa, gruposDeEdad, semanasPico, topPrioridades } from '../inicio/datos'
import { etiquetaDepto, etiquetaEtapa, formato } from '../mapa/datos'
import { ProveedorSesion } from '../sesion/Sesion'

const AHORA = 1_800_000_000
// El fixture es el reporte real: el inicio debe mostrar exactamente lo que dice.
const REAL = JSON.parse(readFileSync('../reports/evaluacion.json', 'utf8')) as DocumentoEvaluacion

function montar(opciones: { versionPublicada?: string; evaluacion?: Response } = {}) {
  sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'a@b.pe', rol: 'analista', expira: AHORA + 3600 }))
  const f = vi.fn((url: string) => {
    if (url.endsWith('/api/v1/evaluacion')) return Promise.resolve(opciones.evaluacion ?? new Response(JSON.stringify(REAL)))
    if (url.endsWith('/api/v1/modelo')) return Promise.resolve(new Response(JSON.stringify({ version: opciones.versionPublicada ?? REAL.version_modelo })))
    return Promise.resolve(new Response('{}', { status: 404 }))
  })
  vi.stubGlobal('fetch', f)
  render(
    <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    </ProveedorSesion>,
  )
  return f
}

describe('inicio: respuesta a la pregunta de impacto', () => {
  it('muestra el top 10 en el mismo orden que reports/evaluacion.json', async () => {
    const f = montar()
    await screen.findByRole('heading', { name: /prioridades/ })
    const filas = document.querySelectorAll('tr[data-prioridad]')
    expect(filas).toHaveLength(10)
    REAL.prioridades.slice(0, 10).forEach((p, i) => {
      const celdas = within(filas[i] as HTMLElement).getAllByRole('cell')
      expect(celdas[0]).toHaveTextContent(String(i + 1))
      expect(celdas[1]).toHaveTextContent(etiquetaDepto(p.departamento))
      expect(celdas[2]).toHaveTextContent(String(p.semana))
      expect(celdas[3]).toHaveTextContent(etiquetaEtapa(p.etapa))
      expect(celdas[7]).toHaveTextContent(formato.uno(p.casos_esperados))
    })
    expect(f.mock.calls.map((c) => c[0])).toContain('http://api:8080/api/v1/evaluacion')
  })

  it('cada prioridad lleva al mapa con esa combinación filtrada', async () => {
    montar()
    await screen.findByRole('heading', { name: /prioridades/ })
    const p = REAL.prioridades[0]!
    const enlace = within(document.querySelector('tr[data-prioridad="1"]') as HTMLElement).getByRole('link')
    expect(enlace).toHaveAttribute('href', enlaceMapa(p))
    expect(enlaceMapa(p)).toBe(`/mapa?depto=${encodeURIComponent(p.departamento).replace(/%20/g, '+')}&semana=${p.semana}&etapa=${p.etapa}&metrica=esperados`)
  })

  it('KPIs del modelo con su versión, tal como vienen del reporte', async () => {
    montar()
    await screen.findByRole('heading', { name: /El modelo, en cifras/ })
    const kpi = (k: string) => document.querySelector(`[data-kpi="${k}"]`)!
    expect(kpi('version')).toHaveTextContent(REAL.version_modelo)
    expect(kpi('auc')).toHaveTextContent(formato.decimal(REAL.auc_trapecio))
    expect(kpi('recall')).toHaveTextContent(formato.porcentaje(REAL.metricas_test.Recall))
    expect(kpi('precision')).toHaveTextContent(formato.porcentaje(REAL.metricas_test.Precision))
    expect(kpi('brier')).toHaveTextContent(formato.decimal(REAL.brier_calibrado))
    expect(kpi('prevalencia')).toHaveTextContent(formato.porcentaje(REAL.positivos_test / REAL.filas_test))
    expect(screen.queryByText(/Regenera reports\/evaluacion.json/)).not.toBeInTheDocument()
  })

  it('avisa si la evaluación es de otro modelo que el publicado', async () => {
    montar({ versionPublicada: 'otra000version' })
    expect(await screen.findByText(/Regenera reports\/evaluacion.json/)).toHaveTextContent('otra000version')
  })

  it('semanas pico y grupos de edad salen del cubo nacional', async () => {
    const picos = semanasPico(REAL.riesgo)
    expect(picos).toHaveLength(5)
    for (const c of picos) expect([c.departamento, c.etapa]).toEqual([TODOS, TODAS])
    for (let i = 1; i < picos.length; i++) expect(picos[i - 1]!.casos_esperados).toBeGreaterThanOrEqual(picos[i]!.casos_esperados)
    const edades = gruposDeEdad(REAL.riesgo)
    expect(edades.map((c) => c.etapa).sort()).toEqual(REAL.riesgo.etapas.filter((e) => e !== TODAS).sort())
    for (let i = 1; i < edades.length; i++) expect(edades[i - 1]!.riesgo_calibrado_medio).toBeGreaterThanOrEqual(edades[i]!.riesgo_calibrado_medio)

    montar()
    await screen.findByRole('heading', { name: 'Semanas pico' })
    expect([...document.querySelectorAll('li[data-semana]')].map((l) => Number(l.getAttribute('data-semana')))).toEqual(picos.map((c) => c.semana))
    expect([...document.querySelectorAll('tr[data-etapa]')].map((l) => l.getAttribute('data-etapa'))).toEqual(edades.map((c) => c.etapa))
  })

  it('topPrioridades no reordena', () => {
    const al = [...REAL.prioridades].reverse()
    expect(topPrioridades(al, 3)).toEqual(al.slice(0, 3))
  })

  it('si la API falla, lo dice', async () => {
    montar({ evaluacion: new Response(JSON.stringify({ error: 'la evaluación no está disponible' }), { status: 503 }) })
    expect(await screen.findByRole('alert')).toHaveTextContent('la evaluación no está disponible')
  })
})
