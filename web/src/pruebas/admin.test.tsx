import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import { formato } from '../mapa/datos'
import { ProveedorSesion } from '../sesion/Sesion'

const AHORA = 1_800_000_000
const T = '2026-11-26T20:00:00.000Z'
const PESOS_FINALES = [0.125, -0.5, 0.0625] // 2 variables + sesgo
const MODELO = { version: 'v-final', generado: T, umbral: 0.5, ajuste_prior: 0, ano_min: 2000, departamentos: [], sesgo: PESOS_FINALES[2], variables: [
  { nombre: 'edad', peso: PESOS_FINALES[0] },
  { nombre: 'ano', peso: PESOS_FINALES[1] },
] }
const nodo = (nombre: string, sano = true) => ({ nombre, remoto: 'x:1', workers: 16, sano, carga: 1 })
const CLUSTER = { epocas: 0, perdida: 0, reasignaciones: 0, nodos: [nodo('nodo-0'), nodo('nodo-1'), nodo('nodo-2')] }

// WebSocket falso: guarda las instancias para que el test mande eventos.
class WSFalso {
  static instancias: WSFalso[] = []
  onopen: (() => void) | null = null
  onmessage: ((m: { data: unknown }) => void) | null = null
  onclose: ((c: { code: number; reason: string }) => void) | null = null
  onerror: (() => void) | null = null
  constructor(readonly url: string) {
    WSFalso.instancias.push(this)
  }
  close() {}
  emitir(e: object) {
    act(() => this.onmessage?.({ data: JSON.stringify({ v: 1, t: T, epoca: 0, ...e }) }))
  }
}

interface Llamada {
  url: string
  metodo: string
  cuerpo?: unknown
}

function montar(opciones: { modelo?: object; rechazarRitmo?: boolean; ticket?: string; cluster?: object } = {}) {
  WSFalso.instancias = []
  sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'admin@vigia.pe', rol: 'admin', expira: AHORA + 3600 }))
  const llamadas: Llamada[] = []
  let entrenamiento = { id: 'ent-1', estado: 'en_curso', epocas: 3, epoca: 0 }
  const json = (v: unknown, status = 200) => Promise.resolve(new Response(JSON.stringify(v), { status }))
  const f = vi.fn((url: string, init?: RequestInit) => {
    const metodo = init?.method ?? 'GET'
    const cuerpo = init?.body ? JSON.parse(String(init.body)) : undefined
    llamadas.push({ url, metodo, cuerpo })
    if (url.endsWith('/api/v1/cluster')) return json(opciones.cluster ?? CLUSTER)
    if (url.endsWith('/api/v1/modelo')) return json(opciones.modelo ?? MODELO)
    if (url.endsWith('/api/v1/ws/ticket')) return json({ ticket: opciones.ticket ?? `tk-${llamadas.length}`, expira: AHORA + 30 })
    if (url.endsWith('/api/v1/entrenamientos') && metodo === 'POST') {
      if (opciones.rechazarRitmo && (cuerpo as { ritmo_ms?: number }).ritmo_ms) return json({ error: 'ritmo_ms solo existe en modo demo (coordinador con -demo)' }, 400)
      entrenamiento = { ...entrenamiento, epocas: (cuerpo as { epocas: number }).epocas }
      return json(entrenamiento, 202)
    }
    if (url.includes('/api/v1/entrenamientos/')) return json(entrenamiento)
    if (url.includes('/api/v1/demo/nodos/')) return json({ nodo: 'nodo-1', estado: 'cortado' }, 202)
    return json({ error: 'no' }, 404)
  })
  vi.stubGlobal('fetch', f)
  vi.stubGlobal('WebSocket', WSFalso)
  render(
    <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
      <MemoryRouter initialEntries={['/admin']}>
        <App />
      </MemoryRouter>
    </ProveedorSesion>,
  )
  return {
    llamadas,
    /** El próximo GET /entrenamientos/{id} lo devuelve terminado. */
    terminar: () => {
      entrenamiento = { ...entrenamiento, estado: 'terminado', epoca: entrenamiento.epocas }
    },
  }
}

async function conectado() {
  await waitFor(() => expect(WSFalso.instancias).toHaveLength(1))
  const ws = WSFalso.instancias[0]!
  act(() => ws.onopen?.())
  return ws
}

const seccion = (n: string) => document.querySelector(`[data-seccion="${n}"]`) as HTMLElement

describe('panel admin en vivo', () => {
  it('pide un ticket, abre el WebSocket y siembra los nodos desde GET /cluster', async () => {
    const { llamadas } = montar()
    const ws = await conectado()
    expect(ws.url).toMatch(/^ws:\/\/api:8080\/api\/v1\/ws\?ticket=tk-\d+$/)
    expect(llamadas.some((l) => l.metodo === 'POST' && l.url.endsWith('/api/v1/ws/ticket'))).toBe(true)
    expect(document.querySelector('[data-conexion]')).toHaveTextContent('En vivo')
    await waitFor(() => expect(document.querySelectorAll('svg [data-nodo]')).toHaveLength(3))
    for (const s of document.querySelectorAll('svg')) {
      expect(s).toHaveAttribute('role', 'img')
      expect(s.getAttribute('aria-label')).toBeTruthy()
    }
  })

  it('lanza un entrenamiento con ritmo_ms y repinta pérdida, pesos y época con los eventos', async () => {
    const { llamadas } = montar()
    const ws = await conectado()
    fireEvent.change(screen.getByLabelText('Épocas'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lanzar entrenamiento' }))
    await waitFor(() => expect(llamadas.find((l) => l.metodo === 'POST' && l.url.endsWith('/api/v1/entrenamientos'))?.cuerpo).toEqual({ epocas: 3, tasa: 2, ritmo_ms: 20 }))
    await screen.findByRole('button', { name: 'Entrenando…' })

    ws.emitir({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0, 0, 0] })
    ws.emitir({ tipo: 'epoca', epoca: 2, perdida: 0.61, pesos: [0.1, -0.3, 0.05] })
    expect(document.querySelector('[data-kpi="epoca"]')).toHaveTextContent('2 / 3')
    expect(document.querySelector('[data-kpi="perdida"]')).toHaveTextContent('0.610000')
    expect(document.querySelector('polyline[data-serie="perdida"]')).toHaveAttribute('data-puntos', '2')
    expect(document.querySelectorAll('[data-grafico="pesos"] [data-peso]')).toHaveLength(3)
    // Las barras llevan los nombres del modelo publicado, con el sesgo al final.
    await waitFor(() => expect(seccion('pesos')).toHaveTextContent('Sesgo (intercepto)'))
  })

  it('el mapa en vivo parte del modelo publicado y se repinta con cada época', async () => {
    // edad, Lima, Loreto, Otros + sesgo: los departamentos sin columna (p. ej. Tacna) usan la de Otros.
    const modelo = { ...MODELO, sesgo: 0, variables: [
      { nombre: 'edad_z', peso: 0.2 },
      { nombre: 'depto_lima', peso: 0 },
      { nombre: 'depto_loreto', peso: 0 },
      { nombre: 'depto_otros', peso: 0 },
    ] }
    montar({ modelo })
    const ws = await conectado()
    const delta = (d: string) => Number(seccion('mapa-vivo').querySelector(`[data-depto="${d}"]`)?.getAttribute('data-delta'))
    await waitFor(() => expect(seccion('mapa-vivo').querySelectorAll('path[data-depto]')).toHaveLength(25))
    expect(delta('LIMA')).toBe(0)
    expect(seccion('mapa-vivo').querySelector('svg')!.getAttribute('aria-label')).toMatch(/modelo publicado v-final/)

    const fillLoreto = () => seccion('mapa-vivo').querySelector('[data-depto="LORETO"]')!.getAttribute('fill')
    const antes = fillLoreto()
    // depto: 0,3 · 0,6 · 0 → media 0,3 → Δ Lima 0, Loreto 0,3, Otros −0,3
    ws.emitir({ tipo: 'epoca', epoca: 1, perdida: 0.69, pesos: [0.2, 0.3, 0.6, 0, -1] })
    expect(delta('LIMA')).toBeCloseTo(0, 12)
    expect(delta('LORETO')).toBeCloseTo(0.3, 12)
    expect(delta('TACNA')).toBeCloseTo(-0.3, 12)
    expect(seccion('mapa-vivo').querySelector('[data-depto="TACNA"]')).toHaveAttribute('data-agrupado', 'true')
    expect(fillLoreto()).not.toBe(antes)
    expect(seccion('mapa-vivo').querySelector('svg')!.getAttribute('aria-label')).toMatch(/época 1.*el mayor es Loreto/)
  })

  // #91 · enunciado TB2 §6: el panel admin muestra uso de CPU, latencia, número de nodos y logs.
  it('muestra CPU y latencia por nodo, nodos sanos y el p95 de la API; un gradiente las actualiza', async () => {
    const metricas = { epoca: 7, shards: 1, calculo_ms: 13.9, latencia_ms: 15.2, red_ms: 1.3, uso_cpu: 0.93 }
    const api = { predicciones: 40, ventana: 40, p50_ms: 0.8, p95_ms: 1.94, max_ms: 3.2, aciertos_cache: 0.5, no_registradas: 0, limite_p95_ms: 100, cumple_limite_p95: true }
    const cluster = { ...CLUSTER, nodos: [{ ...nodo('nodo-0'), metricas }, nodo('nodo-1'), nodo('nodo-2', false)], api }
    montar({ cluster })
    const ws = await conectado()
    const m0 = await waitFor(() => {
      const el = document.querySelector('[data-metricas="nodo-0"]') as HTMLElement
      expect(el).not.toBeNull()
      return el
    })
    expect(m0.querySelector('[data-metrica="cpu"]')).toHaveTextContent(`CPU ${formato.porcentaje(0.93)}`)
    expect(m0.querySelector('[data-metrica="latencia"]')).toHaveTextContent(`latencia ${formato.uno(15.2)} ms`)
    expect(m0.querySelector('[data-metrica="desglose"]')).toHaveTextContent(`cálculo ${formato.uno(13.9)} · red ${formato.uno(1.3)}`)
    expect(document.querySelector('[data-metricas="nodo-1"]')).toBeNull() // sin épocas todavía
    expect(document.querySelector('[data-kpi="nodos"]')).toHaveTextContent('2 / 3')
    expect(document.querySelector('[data-kpi="p95-api"]')).toHaveTextContent(`${formato.uno(1.94)} ms ✓ < 100 ms`)

    // En vivo: el gradiente de la época 8 reemplaza a la instantánea (16 workers en nodo-1).
    ws.emitir({ tipo: 'gradiente', epoca: 8, nodo: 'nodo-1', shard: 1, micros_calculo: 10_000, micros_cpu: 80_000, micros_ida_vuelta: 11_000 })
    const m1 = document.querySelector('[data-metricas="nodo-1"]') as HTMLElement
    expect(m1.querySelector('[data-metrica="cpu"]')).toHaveTextContent(`CPU ${formato.porcentaje(0.5)}`)
    expect(m1.querySelector('[data-metrica="latencia"]')).toHaveTextContent(`latencia ${formato.uno(11)} ms`)
  })

  it('sin modo demo, reintenta el lanzamiento sin ritmo_ms', async () => {
    const { llamadas } = montar({ rechazarRitmo: true })
    await conectado()
    fireEvent.click(screen.getByRole('button', { name: 'Lanzar entrenamiento' }))
    await waitFor(() => expect(llamadas.filter((l) => l.metodo === 'POST' && l.url.endsWith('/api/v1/entrenamientos')).map((l) => l.cuerpo)).toEqual([
      { epocas: 600, tasa: 2, ritmo_ms: 20 },
      { epocas: 600, tasa: 2 },
    ]))
    expect(seccion('bitacora')).toHaveTextContent('no está en modo demo')
  })

  it('tumbar un nodo: lo pide a la API y la bitácora muestra el descarte y la reasignación', async () => {
    const { llamadas } = montar()
    const ws = await conectado()
    await waitFor(() => expect(document.querySelectorAll('.lista-nodos li')).toHaveLength(3))
    fireEvent.click(screen.getByRole('button', { name: 'Tumbar nodo-1' }))
    await waitFor(() => expect(llamadas.some((l) => l.metodo === 'POST' && l.url.endsWith('/api/v1/demo/nodos/nodo-1/cortar'))).toBe(true))

    ws.emitir({ tipo: 'descarte', epoca: 2, nodo: 'nodo-1', detalle: 'conexión cortada (demo)', nodos: [nodo('nodo-0'), nodo('nodo-1', false), nodo('nodo-2')] })
    ws.emitir({ tipo: 'reasignacion', epoca: 2, nodo: 'nodo-0', shard: 1 })
    const bit = seccion('bitacora')
    expect(within(bit).getByText(/nodo-1 salió del clúster/)).toHaveClass('bitacora-alerta')
    expect(bit).toHaveTextContent(/reasignacion: el shard 1 pasa a nodo-0/)
    expect(document.querySelector('svg [data-nodo="nodo-1"]')).toHaveAttribute('data-sano', 'false')
    expect(screen.getByRole('button', { name: 'Tumbar nodo-1' })).toBeDisabled()
    expect(document.querySelector('[data-kpi="reasignaciones"]')).toHaveTextContent('1')
  })

  it('sello bit a bit ✓ cuando los pesos anunciados son idénticos a GET /modelo', async () => {
    montar()
    const ws = await conectado()
    ws.emitir({ tipo: 'modelo', epoca: 3, detalle: 'v-final', pesos: PESOS_FINALES })
    const sello = await waitFor(() => {
      const s = document.querySelector('[data-sello]')
      expect(s).not.toBeNull()
      return s!
    })
    expect(sello).toHaveAttribute('data-sello', 'ok')
    expect(sello).toHaveTextContent('bit a bit ✓')
    expect(sello).toHaveTextContent('3 pesos')
    expect(seccion('entrenar')).toHaveTextContent('TestServicioDemoCortarNodoPorLaAPI')
  })

  it('sello ✗ si un solo peso difiere en el último bit', async () => {
    montar({ modelo: { ...MODELO, sesgo: 0.0625 + 2 ** -56 } })
    const ws = await conectado()
    ws.emitir({ tipo: 'modelo', epoca: 3, detalle: 'v-final', pesos: PESOS_FINALES })
    await waitFor(() => expect(document.querySelector('[data-sello]')).toHaveAttribute('data-sello', 'mal'))
    expect(document.querySelector('[data-sello]')).toHaveTextContent('el peso 2 difiere')
  })

  it('un mensaje inválido por el WebSocket no rompe el panel', async () => {
    montar()
    const ws = await conectado()
    act(() => ws.onmessage?.({ data: '<html>proxy</html>' }))
    act(() => ws.onmessage?.({ data: JSON.stringify({ v: 1, tipo: 'epoca', epoca: 'x' }) }))
    expect(screen.getByRole('heading', { level: 1, name: 'Clúster y entrenamiento en vivo' })).toBeInTheDocument()
    expect(document.querySelector('[data-kpi="epoca"]')).toHaveTextContent('0')
  })

  it('réplica: un .jsonl inválido muestra la línea con error; uno válido se carga', async () => {
    montar()
    await conectado()
    const input = screen.getByLabelText('Archivo .jsonl')
    const malo = new File(['{"v":1,"tipo":"epoca","t":"x","epoca":1}\nno-json\n'], 'malo.jsonl')
    fireEvent.change(input, { target: { files: [malo] } })
    expect(await screen.findByRole('alert')).toHaveTextContent('malo.jsonl: línea 2')

    const lineas = [{ v: 1, tipo: 'epoca', t: T, epoca: 1, perdida: 0.7, pesos: [0, 0, 0] }, { v: 1, tipo: 'fin', t: T, epoca: 1 }]
    const bueno = new File([lineas.map((l) => JSON.stringify(l)).join('\n')], 'eventos_demo.jsonl')
    fireEvent.change(input, { target: { files: [bueno] } })
    expect(await screen.findByText('eventos_demo.jsonl: 2 eventos válidos')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reproducir' }))
    await waitFor(() => expect(seccion('bitacora')).toHaveTextContent('fin de la réplica'))
    expect(document.querySelector('[data-kpi="perdida"]')).toHaveTextContent('0.700000')
  })

  it('el analista no llega al panel: ve el 403 propio y no se abre ningún WebSocket', async () => {
    WSFalso.instancias = []
    sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'a@b.pe', rol: 'analista', expira: AHORA + 3600 }))
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response('{}'))))
    vi.stubGlobal('WebSocket', WSFalso)
    render(
      <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
        <MemoryRouter initialEntries={['/admin']}>
          <App />
        </MemoryRouter>
      </ProveedorSesion>,
    )
    expect(await screen.findByRole('heading', { name: 'Esta vista no es para tu rol' })).toBeInTheDocument()
    expect(WSFalso.instancias).toHaveLength(0)
  })
})

describe('panel admin: revisión PR-87', () => {
  // H1: la réplica no toca la API ni arrastra el entrenamiento en vivo.
  it('una réplica terminada en «modelo» no pide GET /modelo, no pinta sello ni arrastra «/ N»', async () => {
    const { llamadas, terminar } = montar()
    await conectado()
    fireEvent.change(screen.getByLabelText('Épocas'), { target: { value: '15' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lanzar entrenamiento' }))
    await screen.findByRole('button', { name: 'Entrenando…' })
    expect(document.querySelector('[data-kpi="epoca"]')).toHaveTextContent('/ 15')

    const lineas = [
      { v: 1, tipo: 'epoca', t: T, epoca: 1, perdida: 0.7, pesos: [0, 0, 0] },
      { v: 1, tipo: 'modelo', t: T, epoca: 1, detalle: 'v-historica', pesos: [0.1, -0.2, 0] },
    ]
    fireEvent.change(screen.getByLabelText('Archivo .jsonl'), { target: { files: [new File([lineas.map((l) => JSON.stringify(l)).join('\n')], 'ev.jsonl')] } })
    await screen.findByText('ev.jsonl: 2 eventos válidos')
    const modelosAntes = llamadas.filter((l) => l.url.endsWith('/api/v1/modelo')).length
    // El botón de réplica está deshabilitado mientras hay un entrenamiento en curso: se libera al terminar.
    WSFalso.instancias[0]!.emitir({ tipo: 'modelo', epoca: 15, detalle: 'v-final', pesos: PESOS_FINALES })
    await waitFor(() => expect(document.querySelector('[data-sello]')).toHaveAttribute('data-sello', 'ok'))
    terminar()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Reproducir' })).toBeEnabled(), { timeout: 4000 })
    const modelosTrasVivo = llamadas.filter((l) => l.url.endsWith('/api/v1/modelo')).length
    expect(modelosTrasVivo).toBeGreaterThan(modelosAntes)

    fireEvent.click(screen.getByRole('button', { name: 'Reproducir' }))
    await waitFor(() => expect(seccion('bitacora')).toHaveTextContent('fin de la réplica'))
    await new Promise((r) => setTimeout(r, 50))
    expect(llamadas.filter((l) => l.url.endsWith('/api/v1/modelo')).length).toBe(modelosTrasVivo)
    expect(document.querySelector('[data-sello]')).toBeNull()
    expect(document.querySelector('[data-kpi="epoca"]')).toHaveTextContent(/^1$/)
    expect(document.querySelector('[data-kpi="estado"]')).not.toHaveTextContent('terminado')
  }, 10_000)

  // H3.3.
  it('un ticket vacío no abre el WebSocket', async () => {
    montar({ ticket: '' })
    await new Promise((r) => setTimeout(r, 50))
    expect(WSFalso.instancias).toHaveLength(0)
    await waitFor(() => expect(seccion('bitacora')).toHaveTextContent('no devolvió un ticket'))
  })

  it('una tasa infinita no se envía', async () => {
    const { llamadas } = montar()
    await conectado()
    fireEvent.change(screen.getByLabelText('Tasa'), { target: { value: 'Infinity' } })
    fireEvent.click(screen.getByRole('button', { name: 'Lanzar entrenamiento' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Revisa los parámetros')
    expect(llamadas.some((l) => l.metodo === 'POST' && l.url.endsWith('/api/v1/entrenamientos'))).toBe(false)
  })

  // H3.2: con 2 o 4 nodos, el nodo de abajo quedaba con «cayó N×» fuera del viewBox.
  it.each([2, 4])('topología con %i nodos caídos y vueltos: ningún texto sale del viewBox', async (n) => {
    const { Topologia } = await import('../admin/Graficos')
    const nodos = Array.from({ length: n }, (_, i) => ({ ...nodo(`nodo-${i}`), caidas: 1 }))
    const { container } = render(<Topologia nodos={nodos} />)
    const alto = Number(container.querySelector('svg')!.getAttribute('viewBox')!.split(' ')[3])
    for (const t of container.querySelectorAll('text')) expect(Number(t.getAttribute('y'))).toBeLessThanOrEqual(alto - 4)
  })
})
