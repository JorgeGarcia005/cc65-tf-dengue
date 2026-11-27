import { scaleLinear, type ScaleLinear } from 'd3-scale'
import type { MedicionNodos, Metricas, PuntoROC, TramoCalibracion } from '../api/tipos'
import { formato } from '../mapa/datos'
import { puntoUmbral } from './datos'

// Gráficos SVG de la vista de evaluación (#68). Todo con atributos de SVG
// (x, y, points…): la CSP de cmd/web (style-src 'self') no admite estilos inline.
// viewBox fijo y width 100 %: el gráfico se encoge con la tarjeta sin scroll horizontal.

const A = 360
const H = 300
const M = { izq: 54, der: 14, arr: 14, aba: 44 }

// Ticks de porcentaje sin decimales: con «10,0 %» la etiqueta choca con el título del eje.
const pctEje = new Intl.NumberFormat('es-PE', { style: 'percent', maximumFractionDigits: 0 }).format

type Escala = ScaleLinear<number, number>

const r1 = (v: number) => Math.round(v * 10) / 10
const puntos = (pts: [number, number][], x: Escala, y: Escala) => pts.map(([a, b]) => `${r1(x(a))},${r1(y(b))}`).join(' ')

function Ejes({ x, y, tx, ty, fx, fy, ticksX }: { x: Escala; y: Escala; tx: string; ty: string; fx: (v: number) => string; fy: (v: number) => string; ticksX?: number[] }) {
  const [x0, x1] = x.range() as [number, number]
  const [y0, y1] = y.range() as [number, number]
  return (
    <g className="eje" aria-hidden="true">
      <line x1={x0} x2={x1} y1={y0} y2={y0} />
      <line x1={x0} x2={x0} y1={y0} y2={y1} />
      {(ticksX ?? x.ticks(5)).map((t) => (
        <g key={`x${t}`}>
          <line className="eje-guia" x1={r1(x(t))} x2={r1(x(t))} y1={y0} y2={y1} />
          <text x={r1(x(t))} y={y0 + 16} textAnchor="middle">
            {fx(t)}
          </text>
        </g>
      ))}
      {y.ticks(5).map((t) => (
        <g key={`y${t}`}>
          <line className="eje-guia" x1={x0} x2={x1} y1={r1(y(t))} y2={r1(y(t))} />
          <text x={x0 - 6} y={r1(y(t)) + 4} textAnchor="end">
            {fy(t)}
          </text>
        </g>
      ))}
      <text className="eje-titulo" x={(x0 + x1) / 2} y={H - 6} textAnchor="middle">
        {tx}
      </text>
      <text className="eje-titulo" transform={`translate(12 ${(y0 + y1) / 2}) rotate(-90)`} textAnchor="middle">
        {ty}
      </text>
    </g>
  )
}

const ejeX = (d: [number, number]) => scaleLinear().domain(d).range([M.izq, A - M.der])
const ejeY = (d: [number, number]) => scaleLinear().domain(d).range([H - M.aba, M.arr])

export function GraficoROC({ roc, auc, metricas }: { roc: PuntoROC[]; auc: number; metricas: Metricas }) {
  const x = ejeX([0, 1])
  const y = ejeY([0, 1])
  const u = puntoUmbral(metricas)
  return (
    <svg
      viewBox={`0 0 ${A} ${H}`}
      width="100%"
      role="img"
      aria-label={`Curva ROC con ${roc.length} puntos y AUC ${formato.decimal(auc)}; con el umbral ${formato.decimal(metricas.Umbral)} la sensibilidad es ${formato.porcentaje(u.tpr)} y la tasa de falsos positivos ${formato.porcentaje(u.fpr)}`}
    >
      <Ejes x={x} y={y} tx="Tasa de falsos positivos" ty="Sensibilidad" fx={formato.numero} fy={formato.numero} />
      <line className="referencia" data-serie="azar" x1={x(0)} y1={y(0)} x2={x(1)} y2={y(1)} />
      <polyline className="serie serie-rosa" data-serie="roc" data-puntos={roc.length} points={puntos(roc.map((p) => [p.fpr, p.tpr]), x, y)} />
      <circle className="marca-umbral" data-serie="umbral" data-fpr={u.fpr} data-tpr={u.tpr} cx={r1(x(u.fpr))} cy={r1(y(u.tpr))} r={6} />
      <text className="grafico-leyenda" data-prueba="auc" x={x(0.97)} y={y(0.06)} textAnchor="end">
        AUC = {formato.decimal(auc)}
      </text>
    </svg>
  )
}

export function GraficoCalibracion({ tramos }: { tramos: TramoCalibracion[] }) {
  // Misma escala en los dos ejes: la diagonal es la calibración perfecta.
  const max = Math.max(...tramos.flatMap((t) => [t.riesgo_calibrado_medio, t.ic95_sup]))
  const dom = scaleLinear().domain([0, max]).nice().domain() as [number, number]
  const x = ejeX(dom)
  const y = ejeY(dom)
  return (
    <svg
      viewBox={`0 0 ${A} ${H}`}
      width="100%"
      role="img"
      aria-label={`Calibración en ${tramos.length} deciles: el riesgo predicho va de ${formato.porcentaje(tramos[0]!.riesgo_calibrado_medio)} a ${formato.porcentaje(tramos.at(-1)!.riesgo_calibrado_medio)} y la tasa observada de ${formato.porcentaje(tramos[0]!.tasa_observada)} a ${formato.porcentaje(tramos.at(-1)!.tasa_observada)}`}
    >
      <Ejes x={x} y={y} tx="Riesgo calibrado medio" ty="Tasa observada" fx={pctEje} fy={pctEje} />
      <line className="referencia" data-serie="perfecta" x1={x(dom[0])} y1={y(dom[0])} x2={x(dom[1])} y2={y(dom[1])} />
      {tramos.map((t, i) => (
        <g key={i}>
          <line className="ic" data-serie="ic" x1={r1(x(t.riesgo_calibrado_medio))} x2={r1(x(t.riesgo_calibrado_medio))} y1={r1(y(t.ic95_inf))} y2={r1(y(t.ic95_sup))} />
          <circle className="punto-azul" data-serie="tramo" cx={r1(x(t.riesgo_calibrado_medio))} cy={r1(y(t.tasa_observada))} r={4} />
        </g>
      ))}
    </svg>
  )
}

/** Speedup o eficiencia por número de nodos, con su referencia ideal. */
export function GraficoNodos({ mediciones, serie }: { mediciones: MedicionNodos[]; serie: 'speedup' | 'eficiencia' }) {
  const nodos = mediciones.map((m) => m.nodos)
  const nMin = Math.min(...nodos)
  const nMax = Math.max(...nodos)
  const valores = mediciones.map((m) => m[serie])
  const x = ejeX(nMin === nMax ? [nMin - 1, nMax + 1] : [nMin, nMax])
  const tope = serie === 'speedup' ? Math.max(nMax, ...valores) : Math.max(1.1, ...valores)
  const y = ejeY(scaleLinear().domain([0, tope]).nice().domain() as [number, number])
  const fmt = serie === 'speedup' ? formato.numero : pctEje
  const ultima = mediciones.at(-1)!
  const etiqueta =
    serie === 'speedup'
      ? `Speedup por número de nodos, de ${nMin} a ${nMax}: con ${ultima.nodos} nodos es ${formato.decimal(ultima.speedup)} frente al ideal ${ultima.nodos}`
      : `Eficiencia por número de nodos, de ${nMin} a ${nMax}: con ${ultima.nodos} nodos es ${formato.porcentaje(ultima.eficiencia)} frente al ideal de 100 %`
  return (
    <svg viewBox={`0 0 ${A} ${H}`} width="100%" role="img" aria-label={etiqueta}>
      <Ejes x={x} y={y} tx="Nodos" ty={serie === 'speedup' ? 'Speedup' : 'Eficiencia'} fx={String} fy={fmt} ticksX={nodos} />
      {serie === 'speedup' ? (
        <line className="referencia" data-serie="ideal" x1={x(nMin)} y1={y(nMin)} x2={x(nMax)} y2={y(nMax)} />
      ) : (
        <line className="referencia" data-serie="ideal-eficiencia" x1={x.range()[0]} y1={y(1)} x2={x.range()[1]} y2={y(1)} />
      )}
      <polyline className="serie serie-azul" points={puntos(mediciones.map((m) => [m.nodos, m[serie]]), x, y)} />
      {mediciones.map((m) => (
        <circle key={m.nodos} className="punto-azul" data-serie={serie} data-nodos={m.nodos} cx={r1(x(m.nodos))} cy={r1(y(m[serie]))} r={4.5} />
      ))}
    </svg>
  )
}
