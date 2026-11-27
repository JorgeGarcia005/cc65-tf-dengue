import { readFileSync } from 'node:fs'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from '../App'
import type { DocumentoEvaluacionCompleto, Metricas } from '../api/tipos'
import { esDocumentoEvaluacionCompleto, filasConfusion, fuente, lecturaAuc, puntoUmbral, tramosDentroIC } from '../evaluacion/datos'
import { formato } from '../mapa/datos'
import { ProveedorSesion } from '../sesion/Sesion'

const AHORA = 1_800_000_000
// El fixture es el reporte real: la vista debe mostrar exactamente lo que dice.
const REAL = JSON.parse(readFileSync('../reports/evaluacion.json', 'utf8')) as DocumentoEvaluacionCompleto

function montar(evaluacion?: Response) {
  sessionStorage.setItem('vigia.sesion', JSON.stringify({ token: 't', correo: 'a@b.pe', rol: 'analista', expira: AHORA + 3600 }))
  const f = vi.fn((url: string) => {
    if (url.endsWith('/api/v1/evaluacion')) return Promise.resolve(evaluacion ?? new Response(JSON.stringify(REAL)))
    return Promise.resolve(new Response('{}', { status: 404 }))
  })
  vi.stubGlobal('fetch', f)
  render(
    <ProveedorSesion apiUrl="http://api:8080" ahora={() => AHORA}>
      <MemoryRouter initialEntries={['/evaluacion']}>
        <App />
      </MemoryRouter>
    </ProveedorSesion>,
  )
  return f
}

const seccion = (nombre: string) => document.querySelector(`[data-seccion="${nombre}"]`) as HTMLElement

async function cargada() {
  await screen.findByRole('heading', { name: 'Curva ROC' })
}

describe('evaluación: el reporte real cuadra', () => {
  it('el fixture tiene la forma completa y sus totales son consistentes', () => {
    expect(esDocumentoEvaluacionCompleto(REAL, 1)).toBe(true)
    const m = REAL.metricas_test
    expect(m.VP + m.FN + m.FP + m.VN).toBe(REAL.filas_test)
    expect(m.VP + m.FN).toBe(REAL.positivos_test)
    expect(REAL.calibracion.reduce((a, t) => a + t.casos, 0)).toBe(REAL.filas_test)
  })
})

describe('evaluación: vista', () => {
  it('encabezado con versión, filas de prueba y prevalencia', async () => {
    const f = montar()
    await cargada()
    expect(screen.getByRole('heading', { level: 1, name: 'Evaluación del modelo' })).toBeInTheDocument()
    const kpi = (k: string) => document.querySelector(`[data-kpi="${k}"]`)!
    expect(kpi('version')).toHaveTextContent(REAL.version_modelo)
    expect(kpi('filas')).toHaveTextContent(formato.entero(REAL.filas_test))
    expect(kpi('positivos')).toHaveTextContent(formato.entero(REAL.positivos_test))
    expect(kpi('prevalencia')).toHaveTextContent(formato.porcentaje(REAL.positivos_test / REAL.filas_test))
    expect(f.mock.calls.map((c) => c[0])).toContain('http://api:8080/api/v1/evaluacion')
  })

  it('ROC: 201 puntos de (0,0) a (1,1), AUC del reporte y punto del umbral', async () => {
    montar()
    await cargada()
    const s = seccion('roc')
    const curva = s.querySelector('polyline[data-serie="roc"]')!
    expect(curva.getAttribute('data-puntos')).toBe(String(REAL.roc.length))
    expect(curva.getAttribute('points')!.trim().split(/\s+/)).toHaveLength(201)
    expect([REAL.roc[0]!.fpr, REAL.roc[0]!.tpr]).toEqual([0, 0])
    expect([REAL.roc.at(-1)!.fpr, REAL.roc.at(-1)!.tpr]).toEqual([1, 1])
    expect(s.querySelector('[data-prueba="auc"]')).toHaveTextContent(`AUC = ${formato.decimal(REAL.auc_trapecio)}`)
    expect(s.querySelector('line[data-serie="azar"]')).toBeInTheDocument()
    const p = puntoUmbral(REAL.metricas_test)
    const marca = s.querySelector('[data-serie="umbral"]')!
    expect(marca.getAttribute('data-fpr')).toBe(String(p.fpr))
    expect(marca.getAttribute('data-tpr')).toBe(String(p.tpr))
    const svg = within(s).getByRole('img')
    expect(svg).toHaveAttribute('aria-label', expect.stringContaining(formato.decimal(REAL.auc_trapecio)))
    expect(svg).toHaveAttribute('viewBox')
  })

  it('calibración: 10 tramos en orden, casos que suman filas_test y Brier', async () => {
    montar()
    await cargada()
    const s = seccion('calibracion')
    const filas = s.querySelectorAll('tr[data-tramo]')
    expect(filas).toHaveLength(10)
    let suma = 0
    REAL.calibracion.forEach((t, i) => {
      const celdas = within(filas[i] as HTMLElement).getAllByRole('cell')
      expect(filas[i]).toHaveAttribute('data-tramo', String(i + 1))
      expect(celdas[0]).toHaveTextContent(formato.entero(t.casos))
      expect(celdas[1]).toHaveTextContent(formato.porcentaje(t.riesgo_calibrado_medio))
      expect(celdas[2]).toHaveTextContent(formato.porcentaje(t.tasa_observada))
      expect(celdas[3]).toHaveTextContent(`${formato.porcentaje(t.ic95_inf)} – ${formato.porcentaje(t.ic95_sup)}`)
      suma += t.casos
    })
    expect(suma).toBe(REAL.filas_test)
    expect(s.querySelectorAll('circle[data-serie="tramo"]')).toHaveLength(10)
    expect(s.querySelectorAll('line[data-serie="ic"]')).toHaveLength(10)
    expect(s.querySelector('[data-prueba="brier"]')).toHaveTextContent(formato.decimal(REAL.brier_calibrado))
  })

  it('matriz de confusión con VP, FN, FP y VN exactos y sus métricas', async () => {
    montar()
    await cargada()
    const s = seccion('confusion')
    const m = REAL.metricas_test
    const celda = (k: string) => s.querySelector(`[data-celda="${k}"]`)!
    expect(celda('VP')).toHaveTextContent(formato.entero(m.VP))
    expect(celda('FN')).toHaveTextContent(formato.entero(m.FN))
    expect(celda('FP')).toHaveTextContent(formato.entero(m.FP))
    expect(celda('VN')).toHaveTextContent(formato.entero(m.VN))
    const kpi = (k: string) => s.querySelector(`[data-kpi="${k}"]`)!
    expect(kpi('exactitud')).toHaveTextContent(formato.porcentaje(m.Exactitud))
    expect(kpi('precision')).toHaveTextContent(formato.porcentaje(m.Precision))
    expect(kpi('recall')).toHaveTextContent(formato.porcentaje(m.Recall))
    expect(kpi('f1')).toHaveTextContent(formato.decimal(m.F1))
    expect(kpi('umbral')).toHaveTextContent(formato.decimal(m.Umbral))
    expect(s.querySelectorAll('th[scope="row"]')).toHaveLength(2)
  })

  it('escalabilidad: un punto y una fila por medición, formateados desde el JSON', async () => {
    montar()
    await cargada()
    const s = seccion('escalabilidad')
    const med = REAL.escalabilidad.mediciones
    const filas = s.querySelectorAll('tr[data-nodos]')
    expect(filas).toHaveLength(med.length)
    med.forEach((x, i) => {
      const celdas = within(filas[i] as HTMLElement).getAllByRole('cell')
      expect(filas[i]).toHaveAttribute('data-nodos', String(x.nodos))
      // Segundos por época mostrados en milisegundos: con 3 decimales en segundos el IC se vería como «0,041 – 0,041».
      expect(celdas[0]).toHaveTextContent(String(x.nodos))
      expect(celdas[1]).toHaveTextContent(formato.uno(x.seg_por_epoca.media_recortada * 1000))
      expect(celdas[2]).toHaveTextContent(`${formato.uno(x.seg_por_epoca.ic95_inf * 1000)} – ${formato.uno(x.seg_por_epoca.ic95_sup * 1000)}`)
      expect(celdas[3]).toHaveTextContent(formato.decimal(x.speedup))
      expect(celdas[4]).toHaveTextContent(formato.porcentaje(x.eficiencia))
      expect(celdas[5]).toHaveTextContent(formato.porcentaje(x.desglose_por_epoca.fraccion_red_y_espera))
    })
    expect(s.querySelectorAll('circle[data-serie="speedup"]')).toHaveLength(med.length)
    expect(s.querySelectorAll('circle[data-serie="eficiencia"]')).toHaveLength(med.length)
    expect(s.querySelector('line[data-serie="ideal"]')).toBeInTheDocument()
    expect(within(s).getAllByRole('img')).toHaveLength(2)
    expect(s).toHaveTextContent(REAL.escalabilidad.maquina.cpu)
  })

  it('latencia: p95 de cada corrida y «cumple» cuando cumple_limite', async () => {
    montar()
    await cargada()
    const s = seccion('latencia')
    const corridas = REAL.latencia_api.corridas
    expect(corridas.length).toBeGreaterThan(0)
    for (const c of corridas) {
      const fila = s.querySelector(`tr[data-corrida="${c.nombre}"]`) as HTMLElement
      const celdas = within(fila).getAllByRole('cell')
      expect(celdas[1]).toHaveTextContent(formato.uno(c.p50_ms))
      expect(celdas[2]).toHaveTextContent(formato.uno(c.p95_ms))
      expect(celdas[3]).toHaveTextContent(formato.uno(c.p99_ms))
      expect(celdas[6]).toHaveTextContent(c.cumple_limite ? 'cumple' : 'no cumple')
      expect(celdas[6]).toHaveTextContent(String(REAL.latencia_api.limite_p95_ms))
    }
  })

  it('latencia: si una corrida no cumple, lo dice', async () => {
    const malo = structuredClone(REAL)
    malo.latencia_api.corridas[0]!.cumple_limite = false
    montar(new Response(JSON.stringify(malo)))
    await cargada()
    const fila = seccion('latencia').querySelector(`tr[data-corrida="${malo.latencia_api.corridas[0]!.nombre}"]`) as HTMLElement
    expect(within(fila).getAllByRole('cell')[6]).toHaveTextContent('no cumple')
  })

  it('cada sección cita su reports/*.json; escalabilidad y latencia lo toman de fuentes', async () => {
    montar()
    await cargada()
    for (const n of ['roc', 'calibracion', 'confusion', 'escalabilidad', 'latencia']) {
      const pie = seccion(n).querySelector('figcaption, .nota')
      expect(pie, n).not.toBeNull()
      expect(pie!.textContent, n).toMatch(/reports\/\S+\.json/)
    }
    expect(seccion('escalabilidad').querySelector('[data-fuente]')).toHaveTextContent(fuente(REAL, /bench_cluster_/)!)
    expect(seccion('latencia').querySelector('[data-fuente]')).toHaveTextContent(fuente(REAL, /bench_api_/)!)

    // Si el reporte se regenera con otro benchmark, el pie lo sigue sin tocar el front.
    const otro = structuredClone(REAL)
    otro.fuentes = otro.fuentes.map((x) => (/bench_api_/.test(x.archivo) ? { ...x, archivo: 'reports/bench_api_otra-pc_jwt.json' } : x))
    expect(fuente(otro, /bench_api_/)).toBe('reports/bench_api_otra-pc_jwt.json')
  })

  it('la lectura del AUC es honesta y sale del valor del reporte', async () => {
    montar()
    await cargada()
    expect(seccion('roc').querySelector('[data-prueba="lectura"]')).toHaveTextContent(lecturaAuc(REAL.auc_trapecio))
    expect(lecturaAuc(REAL.auc_trapecio)).toMatch(/modesta/)
    expect(lecturaAuc(0.62)).toMatch(/priorizar, no para diagnosticar/)
    expect(lecturaAuc(0.85)).not.toMatch(/modesta/)
    expect(lecturaAuc(0.5)).toMatch(/azar/)
  })

  // PR-85 H1: un 200 con otra forma no llega a la vista.
  it.each([
    ['HTML de un proxy', '<html>proxy</html>'],
    ['sin roc', JSON.stringify({ ...REAL, roc: undefined })],
    ['roc vacía', JSON.stringify({ ...REAL, roc: [] })],
    ['sin calibracion', JSON.stringify({ ...REAL, calibracion: null })],
    ['sin escalabilidad', JSON.stringify({ ...REAL, escalabilidad: {} })],
    ['sin latencia', JSON.stringify({ ...REAL, latencia_api: { corridas: 'x' } })],
    ['sin fuentes', JSON.stringify({ ...REAL, fuentes: undefined })],
    ['otra versión de esquema', JSON.stringify({ ...REAL, version_esquema: 2 })],
  ])('%s → aviso de formato, sin pantalla en blanco', async (_, cuerpo) => {
    montar(new Response(cuerpo))
    expect(await screen.findByRole('alert')).toHaveTextContent('formato esperado')
    expect(screen.getByRole('heading', { level: 1, name: 'Evaluación del modelo' })).toBeInTheDocument()
  })

  it('si la API falla, lo dice', async () => {
    montar(new Response(JSON.stringify({ error: 'la evaluación no está disponible' }), { status: 503 }))
    expect(await screen.findByRole('alert')).toHaveTextContent('la evaluación no está disponible')
  })

  it('ninguna cifra sale como NaN', async () => {
    montar()
    await cargada()
    expect(document.body.textContent).not.toMatch(/NaN|Infinity|undefined/)
    for (const el of document.querySelectorAll('svg *')) {
      for (const a of el.getAttributeNames()) expect(el.getAttribute(a), `${el.tagName}[${a}]`).not.toMatch(/NaN|Infinity/)
    }
  })
})

describe('evaluación: lógica pura', () => {
  const ceros: Metricas = { Umbral: 0.5, VP: 0, FP: 0, VN: 0, FN: 0, Exactitud: 0, Precision: 0, Recall: 0, F1: 0, AUC: 0 }

  it('puntoUmbral usa la matriz y con ceros no devuelve NaN', () => {
    const m = REAL.metricas_test
    expect(puntoUmbral(m)).toEqual({ fpr: m.FP / (m.FP + m.VN), tpr: m.VP / (m.VP + m.FN) })
    expect(puntoUmbral(m).tpr).toBeCloseTo(m.Recall, 12)
    expect(puntoUmbral(ceros)).toEqual({ fpr: 0, tpr: 0 })
  })

  it('filasConfusion: filas = real (positivo, negativo), columnas = predicho', () => {
    const m = REAL.metricas_test
    expect(filasConfusion(m)).toEqual([
      [m.VP, m.FN],
      [m.FP, m.VN],
    ])
  })

  it('fuente busca en fuentes y devuelve null si no está', () => {
    expect(fuente(REAL, /bench_cluster_/)).toMatch(/^reports\/bench_cluster_.*\.json$/)
    expect(fuente({ fuentes: [] }, /bench_cluster_/)).toBeNull()
  })

  it('tramosDentroIC cuenta los tramos cuyo predicho cae en el IC 95 % observado', () => {
    const esperado = REAL.calibracion.filter((t) => t.riesgo_calibrado_medio >= t.ic95_inf && t.riesgo_calibrado_medio <= t.ic95_sup).length
    expect(tramosDentroIC(REAL.calibracion)).toBe(esperado)
    expect(tramosDentroIC([])).toBe(0)
  })

  it('rechaza métricas de la matriz que no son enteros', () => {
    expect(esDocumentoEvaluacionCompleto({ ...REAL, metricas_test: { ...REAL.metricas_test, VP: 1.5 } }, 1)).toBe(false)
    expect(esDocumentoEvaluacionCompleto({ ...REAL, roc: [{ fpr: 'a', tpr: 0 }] }, 1)).toBe(false)
  })
})
