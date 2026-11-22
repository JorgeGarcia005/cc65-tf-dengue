import { OTROS } from '../api/tipos'
import { etiquetaDepto, etiquetaEtapa, formato, type IndiceRiesgo } from './datos'

const A = 300
const H = 110

/** Detalle de un departamento: cifras del filtro y curva por semana (observada vs calibrada). */
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
  const maximo = Math.max(1e-9, ...curva.flatMap((c) => (c ? [c.tasa_observada, c.riesgo_calibrado_medio] : [])))
  const punto = (i: number, v: number) => `${(i / 51) * A},${H - (v / maximo) * (H - 8)}`
  const linea = (f: (i: number) => number | undefined) =>
    curva.flatMap((c, i) => (c && !c.insuficiente ? [punto(i, f(i)!)] : [])).join(' ')

  return (
    <aside className="tarjeta panel-depto" aria-labelledby="titulo-depto">
      <p className="antetitulo">{semana === 0 ? 'Todas las semanas' : `Semana ${semana}`} · {etiquetaEtapa(etapa)}</p>
      <h2 id="titulo-depto" className="titulo-panel">
        {etiquetaDepto(depto)}
      </h2>
      {grupo === OTROS && (
        <p className="nota">
          Con pocos casos en el entrenamiento, el modelo lo agrupa con otros departamentos: las cifras son las del grupo.
        </p>
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
      <figure className="curva">
        <svg viewBox={`0 0 ${A} ${H}`} role="img" aria-label={`Riesgo por semana en ${etiquetaDepto(depto)}`}>
          {semana > 0 && <line className="curva-semana" x1={(semana - 1) / 51 * A} x2={(semana - 1) / 51 * A} y1={0} y2={H} />}
          <polyline className="curva-observada" points={linea((i) => curva[i]?.tasa_observada)} />
          <polyline className="curva-calibrada" points={linea((i) => curva[i]?.riesgo_calibrado_medio)} />
        </svg>
        <figcaption>
          <span className="clave clave-calibrada">riesgo calibrado</span>{' '}
          <span className="clave clave-observada">tasa observada</span> · semanas 1–52
        </figcaption>
      </figure>
    </aside>
  )
}
