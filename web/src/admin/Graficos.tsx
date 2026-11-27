import { scaleLinear } from 'd3-scale'
import { formato } from '../mapa/datos'
import type { NodoVista } from './estado'

// Gráficos SVG del panel admin (#69): solo atributos de SVG (la CSP de cmd/web
// no admite estilos inline), viewBox fijo y width 100 %.

const r1 = (v: number) => Math.round(v * 10) / 10

export function CurvaPerdida({ curva, epocasPedidas }: { curva: { epoca: number; perdida: number }[]; epocasPedidas: number | null }) {
  const A = 520
  const H = 240
  const M = { izq: 56, der: 12, arr: 12, aba: 36 }
  const ultima = curva.at(-1)
  const xMax = Math.max(epocasPedidas ?? 0, ultima?.epoca ?? 0, 1)
  const valores = curva.map((p) => p.perdida)
  const yMin = valores.length ? Math.min(...valores) : 0
  const yMax = valores.length ? Math.max(...valores) : 1
  const x = scaleLinear().domain([1, Math.max(2, xMax)]).range([M.izq, A - M.der])
  const y = scaleLinear()
    .domain(yMin === yMax ? [yMin - 0.01, yMax + 0.01] : [yMin, yMax])
    .nice()
    .range([H - M.aba, M.arr])
  const puntos = curva.map((p) => `${r1(x(p.epoca))},${r1(y(p.perdida))}`).join(' ')
  const etiqueta = ultima
    ? `Pérdida por época: ${curva.length} épocas, de ${formato.decimal(curva[0]!.perdida)} a ${formato.decimal(ultima.perdida)}`
    : 'Pérdida por época: todavía no hay épocas'
  return (
    <svg viewBox={`0 0 ${A} ${H}`} width="100%" role="img" aria-label={etiqueta} data-grafico="perdida">
      <g className="eje" aria-hidden="true">
        <line x1={M.izq} x2={A - M.der} y1={H - M.aba} y2={H - M.aba} />
        <line x1={M.izq} x2={M.izq} y1={H - M.aba} y2={M.arr} />
        {y.ticks(4).map((t) => (
          <g key={t}>
            <line className="eje-guia" x1={M.izq} x2={A - M.der} y1={r1(y(t))} y2={r1(y(t))} />
            <text x={M.izq - 6} y={r1(y(t)) + 4} textAnchor="end">
              {formato.decimal(t)}
            </text>
          </g>
        ))}
        {x.ticks(5).map((t) => (
          <text key={t} x={r1(x(t))} y={H - M.aba + 16} textAnchor="middle">
            {t}
          </text>
        ))}
        <text className="eje-titulo" x={(M.izq + A - M.der) / 2} y={H - 4} textAnchor="middle">
          Época
        </text>
      </g>
      {curva.length > 0 && <polyline className="serie serie-rosa" data-serie="perdida" data-puntos={curva.length} points={puntos} />}
      {ultima && <circle className="marca-umbral" cx={r1(x(ultima.epoca))} cy={r1(y(ultima.perdida))} r={4.5} />}
    </svg>
  )
}

/** Barras horizontales centradas en 0: un peso por variable y el sesgo al final. */
export function BarrasPesos({ pesos, nombres }: { pesos: number[] | null; nombres: string[] }) {
  const n = pesos?.length ?? 0
  const fila = 14
  const A = 520
  const M = { izq: 210, der: 64 }
  const H = Math.max(1, n) * fila + 8
  const max = Math.max(1e-9, ...(pesos ?? []).map(Math.abs))
  const x = scaleLinear().domain([-max, max]).range([M.izq, A - M.der])
  const etiqueta = pesos
    ? `Pesos de la última época: ${n} valores entre ${formato.decimal(Math.min(...pesos))} y ${formato.decimal(Math.max(...pesos))}`
    : 'Pesos: todavía no hay épocas'
  return (
    <svg viewBox={`0 0 ${A} ${H}`} width="100%" role="img" aria-label={etiqueta} data-grafico="pesos">
      <line className="referencia" x1={x(0)} x2={x(0)} y1={0} y2={H} />
      {(pesos ?? []).map((p, j) => {
        const y0 = 4 + j * fila
        const a = x(Math.min(0, p))
        const b = x(Math.max(0, p))
        return (
          <g key={j} data-peso={j}>
            <text className="barra-nombre" x={M.izq - 6} y={y0 + 10} textAnchor="end">
              {nombres[j] ?? (j === n - 1 ? 'sesgo' : `w${j}`)}
            </text>
            <rect className={p >= 0 ? 'barra-sube' : 'barra-baja'} x={r1(a)} y={y0 + 1} width={Math.max(0.5, r1(b - a))} height={fila - 4} />
            <text className="barra-valor" x={A - 4} y={y0 + 10} textAnchor="end">
              {formato.decimal(p)}
            </text>
          </g>
        )
      })}
    </svg>
  )
}

/** Estrella: el coordinador al centro y los nodos alrededor; en rojo si se cayeron. */
export function Topologia({ nodos }: { nodos: NodoVista[] }) {
  const A = 360
  const H = 300
  const cx = A / 2
  const cy = H / 2
  const radio = 105
  const pos = nodos.map((_, i) => {
    const ang = -Math.PI / 2 + (2 * Math.PI * i) / Math.max(1, nodos.length)
    return { x: r1(cx + radio * Math.cos(ang)), y: r1(cy + radio * Math.sin(ang)) }
  })
  const sanos = nodos.filter((n) => n.sano).length
  return (
    <svg viewBox={`0 0 ${A} ${H}`} width="100%" role="img" aria-label={`Topología en estrella: coordinador y ${nodos.length} nodos, ${sanos} sanos${nodos.length > sanos ? `, ${nodos.length - sanos} caídos` : ''}`} data-grafico="topologia">
      {nodos.map((n, i) => (
        <line key={`l-${n.nombre}`} className={n.sano ? 'enlace-sano' : 'enlace-caido'} x1={cx} y1={cy} x2={pos[i]!.x} y2={pos[i]!.y} />
      ))}
      <rect className="coordinador" x={cx - 58} y={cy - 18} width={116} height={36} rx={2} />
      <text className="topo-texto" x={cx} y={cy + 5} textAnchor="middle">
        coordinador
      </text>
      {nodos.map((n, i) => (
        <g key={n.nombre} data-nodo={n.nombre} data-sano={n.sano}>
          <circle className={n.sano ? 'nodo-sano' : 'nodo-caido'} cx={pos[i]!.x} cy={pos[i]!.y} r={26} />
          <text className="topo-texto" x={pos[i]!.x} y={pos[i]!.y - 2} textAnchor="middle">
            {n.nombre.length > 10 ? `${n.nombre.slice(0, 9)}…` : n.nombre}
          </text>
          <text className="topo-sub" x={pos[i]!.x} y={pos[i]!.y + 12} textAnchor="middle">
            {n.sano ? `${n.carga} shard${n.carga === 1 ? '' : 's'}` : 'caído'}
          </text>
        </g>
      ))}
    </svg>
  )
}
