import { OTROS } from '../api/tipos'
import { etiquetaDepto, etiquetaEtapa, formato, tramos, type IndiceRiesgo } from './datos'
import { DEPARTAMENTOS } from './geo'

const A = 300
const H = 110

/** Detalle de un departamento (u OTROS): cifras del filtro y curva por semana (observada vs calibrada). */
export default function PanelDepto({
  indice,
  depto,
  semana,
  etapa,
}: {
  indice: IndiceRiesgo
  depto: string
  semana: number
  etapa: string
}) {
  const grupo = indice.grupo(depto)
  const celda = indice.celda(grupo, semana, etapa)
  const curva = indice.curva(grupo, etapa)
  const calibrada = tramos(curva, (c) => c.riesgo_calibrado_medio)
  const observada = tramos(curva, (c) => c.tasa_observada)
  // La escala sale solo de lo que se dibuja: una semana con 1 caso (tasa 100 %) no aplasta la curva.
  const maximo = Math.max(1e-9, ...[...calibrada, ...observada].flat().map(([, v]) => v))
  const x = (i: number) => (i / 51) * A
  const y = (v: number) => H - (v / maximo) * (H - 8)
  const agrupados = DEPARTAMENTOS.features
    .map((f) => f.properties.NOMBDEP)
    .filter((n) => indice.grupo(n) === OTROS)
    .map(etiquetaDepto)

  return (
    <aside className="tarjeta panel-depto" aria-labelledby="titulo-depto">
      <p className="antetitulo">
        {semana === 0 ? 'Todas las semanas' : `Semana ${semana}`} · {etiquetaEtapa(etapa)}
      </p>
      <h2 id="titulo-depto" className="titulo-panel">
        {etiquetaDepto(depto)}
      </h2>
      {depto === OTROS ? (
        <p className="nota">
          Departamentos con pocos casos en el entrenamiento, que el modelo trata como un solo grupo: {agrupados.join(', ')}.
        </p>
      ) : (
        grupo === OTROS && (
          <p className="nota">
            Con pocos casos en el entrenamiento, el modelo lo agrupa con {agrupados.filter((n) => n !== etiquetaDepto(depto)).join(', ')}: las cifras son las del grupo.
          </p>
        )
      )}
      {celda ? (
        <dl className="cifras">
          <div>
            <dt>Riesgo calibrado</dt>
            <dd>{formato.porcentaje(celda.riesgo_calibrado_medio)}</dd>
          </div>
          <div>
            <dt>Tasa observada</dt>
            <dd>{formato.porcentaje(celda.tasa_observada)}</dd>
          </div>
          <div>
            <dt>Casos esperados con signos de alarma</dt>
            <dd>{formato.numero(celda.casos_esperados)}</dd>
          </div>
          <div>
            <dt>Casos notificados (test)</dt>
            <dd>{formato.entero(celda.casos)}</dd>
          </div>
          <div>
            <dt>Frente al promedio nacional</dt>
            <dd>×{formato.numero(celda.razon_vs_nacional)}</dd>
          </div>
        </dl>
      ) : (
        <p>Sin casos con este filtro.</p>
      )}
      {celda?.insuficiente && (
        <p className="aviso-insuficiente">Menos de {indice.cubo.min_casos} casos: tómalo con cautela.</p>
      )}
      {calibrada.length === 0 ? (
        <p className="nota" data-prueba="curva-vacia">
          Ninguna semana alcanza {indice.cubo.min_casos} casos con este filtro: no hay curva semanal que mostrar.
        </p>
      ) : (
        <figure className="curva">
          <svg viewBox={`0 0 ${A} ${H}`} role="img" aria-label={`Riesgo por semana en ${etiquetaDepto(depto)}`}>
            {semana > 0 && <line className="curva-semana" x1={x(semana - 1)} x2={x(semana - 1)} y1={0} y2={H} />}
            <Serie clase="curva-observada" segmentos={observada} x={x} y={y} />
            <Serie clase="curva-calibrada" segmentos={calibrada} x={x} y={y} />
          </svg>
          <figcaption>
            <span className="clave clave-calibrada">riesgo calibrado</span>{' '}
            <span className="clave clave-observada">tasa observada</span> · semanas 1–52; los cortes son semanas con
            menos de {indice.cubo.min_casos} casos
          </figcaption>
        </figure>
      )}
    </aside>
  )
}

/** Un subcamino por tramo de semanas consecutivas; una semana aislada se marca con un punto. */
function Serie({
  clase,
  segmentos,
  x,
  y,
}: {
  clase: string
  segmentos: [number, number][][]
  x: (i: number) => number
  y: (v: number) => number
}) {
  const d = segmentos
    .filter((s) => s.length > 1)
    .map((s) => s.map(([i, v], k) => `${k === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(''))
    .join('')
  return (
    <g className={clase}>
      {d && <path d={d} />}
      {segmentos
        .filter((s) => s.length === 1)
        .map((s) => {
          const [i, v] = s[0]!
          return <circle key={i} cx={x(i)} cy={y(v)} r={2.5} />
        })}
    </g>
  )
}
