import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import type { Celda, Cubo, DocumentoRiesgo } from '../api/tipos'
import { COLORES, IndiceRiesgo, escalaPara, etiquetaDepto, etiquetaEtapa, normalizarDepto, tramos } from '../mapa/datos'
import { MS_POR_SEMANA } from '../paginas/Mapa'
import { DEPARTAMENTOS } from '../mapa/geo'
import { ProveedorSesion } from '../sesion/Sesion'

const AHORA = 1_800_000_000

function celda(departamento: string, semana: number, etapa: string, riesgo: number, casos: number): Celda {
  return {
    departamento, semana, etapa, casos, con_signos_observados: Math.round(casos * riesgo),
    tasa_observada: riesgo * 0.9, riesgo_calibrado_medio: riesgo, casos_esperados: riesgo * casos,
    fraccion_nacional: 0, razon_vs_nacional: 1, insuficiente: casos < 30,
  }
}

/** Cubo chico: 4 departamentos del modelo, OTROS y TODOS; semanas 0–2; dos etapas. */
function cuboDePrueba(): Cubo {
  const deptos: [string, number][] = [['LORETO', 0.3], ['PIURA', 0.2], ['TUMBES', 0.1], ['LIMA', 0.05], ['OTROS', 0.15], ['TODOS', 0.12]]
  const celdas: Celda[] = []
  for (const [d, r] of deptos) {
    for (const s of [0, 1, 2]) {
      for (const e of ['nino_0_11', 'todas']) {
        // En la semana 2, TUMBES tiene pocos casos (insuficiente).
        const casos = d === 'TUMBES' && s === 2 ? 10 : 100 * (s + 1)
        celdas.push(celda(d, s, e, r * (1 + s / 10) * (e === 'todas' ? 1 : 0.8), casos))
      }
    }
  }
  return { min_casos: 30, departamentos: deptos.map(([d]) => d), etapas: ['nino_0_11', 'todas'], celdas }
}

const DOC: DocumentoRiesgo = { version_esquema: 1, version_modelo: 'abc123def456', riesgo: cuboDePrueba(), prioridades: [] }

describe('datos geográficos', () => {
  it('el GeoJSON tiene los 25 departamentos y su sha256 coincide con el LEEME', () => {
    expect(DEPARTAMENTOS.features).toHaveLength(25)
    const sha = createHash('sha256').update(readFileSync('src/datos/peru_departamental_simple.geojson')).digest('hex')
    expect(readFileSync('src/datos/LEEME.md', 'utf8')).toContain(sha)
  })

  it('cada departamento de reports/evaluacion.json existe en el mapa', () => {
    const real = JSON.parse(readFileSync('../reports/evaluacion.json', 'utf8')) as { riesgo: Cubo }
    const geo = new Set(DEPARTAMENTOS.features.map((f) => f.properties.NOMBDEP))
    const delModelo = real.riesgo.departamentos.filter((d) => d !== 'OTROS' && d !== 'TODOS')
    expect(delModelo.length).toBeGreaterThan(0)
    for (const d of delModelo) expect(geo.has(d), d).toBe(true)
    // y los del mapa que no están en el modelo caen en OTROS
    const indice = new IndiceRiesgo(real.riesgo)
    for (const g of geo) expect(indice.grupo(g)).toBe(delModelo.includes(g) ? g : 'OTROS')
  })
})

describe('escala y etiquetas', () => {
  it('el departamento con más riesgo recibe el color más oscuro y el de menos el más claro', () => {
    const indice = new IndiceRiesgo(cuboDePrueba())
    const esc = escalaPara(indice, 1, 'todas', 'riesgo')
    expect(esc.color(indice.celda('LORETO', 1, 'todas')!.riesgo_calibrado_medio)).toBe(COLORES[4])
    expect(esc.color(indice.celda('LIMA', 1, 'todas')!.riesgo_calibrado_medio)).toBe(COLORES[0])
  })

  it('etiquetas legibles de etapas y departamentos', () => {
    expect(etiquetaEtapa('nino_0_11')).toBe('Niños 0–11')
    expect(etiquetaEtapa('adulto_mayor_60_mas')).toBe('Adultos mayores 60+')
    expect(etiquetaEtapa('todas')).toBe('Todas las edades')
    expect(etiquetaDepto('MADRE DE DIOS')).toBe('Madre de Dios')
    expect(etiquetaDepto('LA LIBERTAD')).toBe('La Libertad')
    expect(etiquetaDepto('TODOS')).toBe('Perú')
  })
})

function montarMapa(ruta = '/mapa') {
  sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'a@b.pe', rol: 'analista', expira: AHORA + 3600 }))
  const f = vi.fn().mockResolvedValue(new Response(JSON.stringify(DOC)))
  vi.stubGlobal('fetch', f)
  render(
    <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
      <MemoryRouter initialEntries={[ruta]}>
        <App />
      </MemoryRouter>
    </ProveedorSesion>,
  )
  return f
}

const depto = (nombre: string) => document.querySelector<SVGPathElement>(`path[data-depto="${nombre}"]`)!

describe('vista del mapa', () => {
  it('pide /api/v1/riesgo con el token y pinta los 25 departamentos', async () => {
    const f = montarMapa()
    expect(await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })).toBeInTheDocument()
    expect(document.querySelectorAll('path.depto')).toHaveLength(25)
    const [url, init] = f.mock.calls[0]!
    expect(url).toBe('http://api:8080/api/v1/riesgo')
    expect(init.headers.Authorization).toBe('Bearer t')
  })

  it('los filtros de semana, edad y medida cambian lo que dice cada departamento', async () => {
    montarMapa()
    await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })
    const antes = depto('LORETO').getAttribute('aria-label')
    fireEvent.change(screen.getByLabelText(/Semana epidemiológica/), { target: { value: '2' } })
    const semana2 = depto('LORETO').getAttribute('aria-label')
    expect(semana2).not.toBe(antes)
    await userEvent.selectOptions(screen.getByLabelText('Grupo de edad'), 'nino_0_11')
    expect(depto('LORETO').getAttribute('aria-label')).not.toBe(semana2)
    await userEvent.click(screen.getByRole('radio', { name: 'Casos esperados' }))
    expect(depto('LORETO').getAttribute('aria-label')).toMatch(/casos esperados/)
    // TUMBES en la semana 2 tiene pocos casos: rayado y avisado
    expect(depto('TUMBES').getAttribute('fill')).toBe('url(#rayado)')
    expect(depto('TUMBES').getAttribute('aria-label')).toMatch(/datos insuficientes/)
  })

  it('un departamento fuera del modelo se muestra con las cifras de OTROS', async () => {
    montarMapa()
    await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })
    expect(depto('AREQUIPA').getAttribute('aria-label')).toMatch(/agrupado en Otros/)
    await userEvent.click(depto('AREQUIPA'))
    expect(await screen.findByRole('heading', { name: 'Arequipa' })).toBeInTheDocument()
    expect(screen.getByText(/el modelo lo agrupa con/)).toHaveTextContent('Tacna')
  })

  it('se navega con el teclado: Tab y Enter seleccionan y abren el panel', async () => {
    montarMapa()
    await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })
    fireEvent.focus(depto('PIURA'))
    expect(screen.getByText(/^Piura:/)).toBeInTheDocument() // la info se anuncia al enfocar
    fireEvent.keyDown(depto('PIURA'), { key: 'Enter' })
    expect(await screen.findByRole('heading', { name: 'Piura' })).toBeInTheDocument()
    expect(depto('PIURA')).toHaveAttribute('aria-pressed', 'true')
  })

  it('los filtros vienen de la URL y la tabla ordena por la medida elegida', async () => {
    montarMapa('/mapa?semana=1&metrica=esperados&depto=LIMA')
    expect(await screen.findByRole('heading', { name: 'Lima' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Casos esperados' })).toBeChecked()
    await userEvent.click(screen.getByRole('button', { name: 'Ver como tabla' }))
    const filas = within(screen.getByRole('table')).getAllByRole('row').slice(1)
    expect(filas[0]).toHaveTextContent('Loreto')
    expect(filas.at(-1)).toHaveTextContent('Lima')
  })

  it('si la API falla, lo dice', async () => {
    sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'a@b.pe', rol: 'analista', expira: AHORA + 3600 }))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"error":"evaluación no disponible"}', { status: 503 })))
    render(
      <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
        <MemoryRouter initialEntries={['/mapa']}>
          <App />
        </MemoryRouter>
      </ProveedorSesion>,
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('evaluación no disponible')
  })
})

describe('ronda 1 de PR-82', () => {
  it('H1: ?depto= se normaliza y uno inexistente no abre panel', async () => {
    montarMapa('/mapa?depto=piura')
    expect(await screen.findByRole('heading', { name: 'Piura' })).toBeInTheDocument()
    expect(screen.queryByText(/agrupa/)).toBeNull() // PIURA es del modelo: sus propias cifras
    expect(depto('PIURA')).toHaveAttribute('aria-pressed', 'true')
    expect(normalizarDepto('  Madre de Dios ', new Set(['MADRE DE DIOS']))).toBe('MADRE DE DIOS')
    expect(normalizarDepto('Marte', new Set(['PIURA']))).toBeNull()
    expect(normalizarDepto('otros', new Set())).toBe('OTROS')
  })

  it('H1: un departamento inexistente en la URL no muestra datos de OTROS', async () => {
    montarMapa('/mapa?depto=NOEXISTE')
    await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })
    expect(screen.queryByRole('heading', { name: /Noexiste/i })).toBeNull()
    expect(screen.getByText(/Elige un departamento en el mapa/)).toBeInTheDocument()
  })

  it('H1: elegir OTROS en la tabla resalta los 5 agrupados y los nombra', async () => {
    montarMapa()
    await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })
    await userEvent.click(screen.getByRole('button', { name: 'Ver como tabla' }))
    await userEvent.click(screen.getByRole('button', { name: 'Otros (agrupados)' }))
    expect(await screen.findByRole('heading', { name: 'Otros (agrupados)' })).toBeInTheDocument()
    const resaltados = [...document.querySelectorAll('path.depto-sel')].map((p) => p.getAttribute('data-depto'))
    expect(resaltados.length).toBeGreaterThan(5) // en el cubo de prueba hay 4 del modelo: los 21 restantes son OTROS
    expect(resaltados).not.toContain('PIURA')
    expect(screen.getByText(/que el modelo trata como un solo grupo/)).toHaveTextContent('Arequipa')
  })

  it('H2: la escala de la curva ignora celdas insuficientes y los huecos cortan la línea', () => {
    const c = (riesgo: number, insuficiente = false) =>
      ({ riesgo_calibrado_medio: riesgo, tasa_observada: riesgo, insuficiente }) as unknown as import('../api/tipos').Celda
    const curva = [c(0.1), c(0.12), c(1, true), undefined, c(0.11), c(0.09), undefined, c(0.2)]
    const t = tramos(curva, (x) => x.riesgo_calibrado_medio)
    expect(t).toEqual([[[0, 0.1], [1, 0.12]], [[4, 0.11], [5, 0.09]], [[7, 0.2]]])
    expect(Math.max(...t.flat().map(([, v]) => v))).toBe(0.2) // la celda con 1 caso (100 %) no entra
    expect(tramos([undefined, c(1, true)], (x) => x.tasa_observada)).toEqual([])
  })

  it('H2: el panel dibuja un subcamino por tramo, un punto por semana aislada y avisa si no hay curva', async () => {
    // En el cubo de prueba TUMBES tiene la semana 2 insuficiente: tramo 1 (semana 1) aislado → punto.
    montarMapa('/mapa?depto=TUMBES')
    await screen.findByRole('heading', { name: 'Tumbes' })
    expect(document.querySelectorAll('.curva-calibrada circle')).toHaveLength(1)
    // LORETO tiene semanas 1 y 2: un tramo de dos puntos → un subcamino «M…L…»
    await userEvent.click(depto('LORETO'))
    const d = document.querySelector('.curva-calibrada path')!.getAttribute('d')!
    expect(d.match(/M/g)).toHaveLength(1)
    expect(d.match(/L/g)).toHaveLength(1)
  })

  it('H4: sin departamentos suficientes no se inventa una leyenda', () => {
    const cubo = cuboDePrueba()
    for (const x of cubo.celdas) x.insuficiente = true
    const esc = escalaPara(new IndiceRiesgo(cubo), 1, 'todas', 'riesgo')
    expect(esc.vacia).toBe(true)
    expect(esc.clases).toEqual([])
  })

  it('H4: con un solo departamento suficiente la leyenda tiene una sola clase con su valor real', () => {
    const cubo = cuboDePrueba()
    for (const x of cubo.celdas) x.insuficiente = x.departamento !== 'PIURA'
    const indice = new IndiceRiesgo(cubo)
    const esc = escalaPara(indice, 1, 'todas', 'riesgo')
    const v = indice.celda('PIURA', 1, 'todas')!.riesgo_calibrado_medio
    expect(esc.clases).toHaveLength(1)
    expect(esc.clases[0]).toMatchObject({ min: v, max: v })
  })

  it('H4: la leyenda y el panel cambian con los filtros', async () => {
    montarMapa('/mapa?depto=LORETO')
    await screen.findByRole('heading', { name: 'Loreto' })
    const leyenda = () => screen.getByRole('list', { name: /Leyenda/ }).textContent
    const cifras = () => document.querySelector('.cifras')!.textContent
    const [l0, c0] = [leyenda(), cifras()]
    fireEvent.change(screen.getByLabelText(/Semana epidemiológica/), { target: { value: '2' } })
    expect(leyenda()).not.toBe(l0)
    expect(cifras()).not.toBe(c0)
    await userEvent.click(screen.getByRole('radio', { name: 'Casos esperados' }))
    expect(leyenda()).not.toMatch(/%/)
  })

  it('H4: reproducir avanza las semanas, se pausa, se detiene en 52 y desde 52 reinicia en 1', async () => {
    montarMapa('/mapa?semana=50')
    await screen.findByRole('group', { name: 'Mapa de riesgo por departamento' })
    const semana = () => screen.getByLabelText(/Semana epidemiológica/).getAttribute('value')
    vi.useFakeTimers()
    try {
      fireEvent.click(screen.getByRole('button', { name: 'Reproducir semanas' }))
      act(() => vi.advanceTimersByTime(MS_POR_SEMANA))
      expect(semana()).toBe('51')
      fireEvent.click(screen.getByRole('button', { name: 'Pausar' }))
      act(() => vi.advanceTimersByTime(3 * MS_POR_SEMANA))
      expect(semana()).toBe('51')
      fireEvent.click(screen.getByRole('button', { name: 'Reproducir semanas' }))
      act(() => vi.advanceTimersByTime(3 * MS_POR_SEMANA))
      expect(semana()).toBe('52')
      expect(screen.getByRole('button', { name: 'Reproducir semanas' })).toBeInTheDocument() // se detuvo
      fireEvent.click(screen.getByRole('button', { name: 'Reproducir semanas' }))
      expect(semana()).toBe('1')
    } finally {
      vi.useRealTimers()
    }
  })
})
