import { useEffect, useState } from 'react'
import { ErrorApi } from '../api/cliente'
import type { DocumentoEvaluacionCompleto } from '../api/tipos'
import { esDocumentoEvaluacionCompleto, filasConfusion, fuente, lecturaAuc, puntoUmbral, tramosDentroIC } from '../evaluacion/datos'
import { GraficoCalibracion, GraficoNodos, GraficoROC } from '../evaluacion/Graficos'
import { proporcion } from '../inicio/datos'
import { formato } from '../mapa/datos'
import { useSesion } from '../sesion/Sesion'

const VERSION_ESPERADA = 1
const ORIGEN = 'reports/evaluacion.json'

const nombreCorrida = (n: string) => ({ fria: 'Fría', caliente: 'Caliente' })[n] ?? n
const ms = (seg: number) => formato.uno(seg * 1000)

/** Nombre del archivo de origen tomado de doc.fuentes; si no está declarado, lo dice. */
function Fuente({ archivo }: { archivo: string | null }) {
  return <span data-fuente>{archivo ?? `fuente no declarada en ${ORIGEN}`}</span>
}

// Evaluación del modelo (#68): ROC, calibración, matriz de confusión,
// escalabilidad por nodos y latencia de la API. Todas las cifras vienen de
// GET /api/v1/evaluacion (reports/evaluacion.json, generado por cmd/evaluacion);
// el front solo las formatea y cita su archivo de origen al pie de cada gráfico.
export default function Evaluacion() {
  const { api } = useSesion()
  const [doc, setDoc] = useState<DocumentoEvaluacionCompleto | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let vivo = true
    api
      .get<unknown>('/api/v1/evaluacion')
      .then((d) => {
        if (!vivo) return
        if (!esDocumentoEvaluacionCompleto(d, VERSION_ESPERADA)) throw new Error('la evaluación no tiene el formato esperado (versión o campos)')
        setDoc(d)
      })
      .catch((e: unknown) => vivo && setError(e instanceof ErrorApi ? e.message : String(e instanceof Error ? e.message : e)))
    return () => {
      vivo = false
    }
  }, [api])

  const cabecera = (
    <>
      <p className="antetitulo">Conjunto de prueba · métricas y escalabilidad</p>
      <h1 id="titulo-vista" className="titulo">
        Evaluación del modelo
      </h1>
    </>
  )
  if (error || !doc) {
    return (
      <div className="vista-evaluacion" aria-busy={!error}>
        <header className="vista-cabecera">{cabecera}</header>
        {error ? (
          <p className="error" role="alert">
            No se pudo cargar la evaluación: {error}
          </p>
        ) : (
          <p className="nota">Cargando la evaluación…</p>
        )}
      </div>
    )
  }

  const m = doc.metricas_test
  const u = puntoUmbral(m)
  const [[vp, fn], [fp, vn]] = filasConfusion(m)
  const dentro = tramosDentroIC(doc.calibracion)
  const alto = doc.calibracion.at(-1)!
  const med = doc.escalabilidad.mediciones
  const primera = med[0]!
  const ultima = med.at(-1)!
  const maq = doc.escalabilidad.maquina
  const lat = doc.latencia_api
  const peorP95 = Math.max(...lat.corridas.map((c) => c.p95_ms))
  const benchCluster = fuente(doc, /bench_cluster_/)
  const benchApi = fuente(doc, /bench_api_/)

  return (
    <div className="vista-evaluacion">
      <header className="vista-cabecera">
        {cabecera}
        <p className="bajada">
          Cómo le va a la regresión logística distribuida con casos que no vio al entrenar, cuánto se acelera el entrenamiento al sumar nodos y cuánto
          tarda la API en responder una predicción.
        </p>
        <dl className="cifras cifras-kpi">
          <div>
            <dt>Versión del modelo</dt>
            <dd data-kpi="version">{doc.version_modelo}</dd>
          </div>
          <div>
            <dt>Casos de prueba</dt>
            <dd data-kpi="filas">{formato.entero(doc.filas_test)}</dd>
          </div>
          <div>
            <dt>Con signos de alarma</dt>
            <dd data-kpi="positivos">{formato.entero(doc.positivos_test)}</dd>
          </div>
          <div>
            <dt>Prevalencia</dt>
            <dd data-kpi="prevalencia">{formato.porcentaje(proporcion(doc.positivos_test, doc.filas_test))}</dd>
          </div>
        </dl>
      </header>

      <div className="evaluacion-columnas">
        <section className="tarjeta" data-seccion="roc" aria-labelledby="titulo-roc">
          <h2 id="titulo-roc" className="titulo-panel">
            Curva ROC
          </h2>
          <figure className="grafico">
            <GraficoROC roc={doc.roc} auc={doc.auc_trapecio} metricas={m} />
            <figcaption>
              Fuente: {ORIGEN} · <code>roc</code> (conjunto de prueba, {formato.entero(doc.filas_test)} casos). Diagonal punteada: azar. Círculo
              amarillo: umbral {formato.decimal(m.Umbral)}.
            </figcaption>
          </figure>
          <p className="bajada" data-prueba="lectura">
            AUC = {formato.decimal(doc.auc_trapecio)}: {lecturaAuc(doc.auc_trapecio)}. Con el umbral {formato.decimal(m.Umbral)} reconoce{' '}
            {formato.porcentaje(u.tpr)} de los casos con signos de alarma, a costa de marcar {formato.porcentaje(u.fpr)} de los casos sin signos.
          </p>
        </section>

        <section className="tarjeta" data-seccion="calibracion" aria-labelledby="titulo-calibracion">
          <h2 id="titulo-calibracion" className="titulo-panel">
            Calibración por deciles
          </h2>
          <figure className="grafico">
            <GraficoCalibracion tramos={doc.calibracion} />
            <figcaption>
              Fuente: {ORIGEN} · <code>calibracion</code> · Brier calibrado = <span data-prueba="brier">{formato.decimal(doc.brier_calibrado)}</span>.
              Barras: IC 95 % de la tasa observada; diagonal: calibración perfecta.
            </figcaption>
          </figure>
          <p className="bajada">
            En {dentro} de {doc.calibracion.length} deciles el riesgo predicho cae dentro del IC 95 % de la tasa observada. En el decil más alto el
            modelo predice {formato.porcentaje(alto.riesgo_calibrado_medio)} y se observó {formato.porcentaje(alto.tasa_observada)}.
          </p>
          <div className="tabla-envoltura">
            <table className="tabla">
              <caption className="visualmente-oculto">Deciles de riesgo calibrado con la tasa observada</caption>
              <thead>
                <tr>
                  <th scope="col">Decil</th>
                  <th scope="col">Casos</th>
                  <th scope="col">Predicho</th>
                  <th scope="col">Observado</th>
                  <th scope="col">IC 95 %</th>
                </tr>
              </thead>
              <tbody>
                {doc.calibracion.map((t, i) => (
                  <tr key={i} data-tramo={i + 1}>
                    <th scope="row">{i + 1}</th>
                    <td>{formato.entero(t.casos)}</td>
                    <td>{formato.porcentaje(t.riesgo_calibrado_medio)}</td>
                    <td>{formato.porcentaje(t.tasa_observada)}</td>
                    <td>
                      {formato.porcentaje(t.ic95_inf)} – {formato.porcentaje(t.ic95_sup)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div className="evaluacion-columnas">
        <section className="tarjeta" data-seccion="confusion" aria-labelledby="titulo-confusion">
          <h2 id="titulo-confusion" className="titulo-panel">
            Matriz de confusión
          </h2>
          <div className="tabla-envoltura">
            <table className="tabla confusion">
              <caption className="visualmente-oculto">Casos de prueba por clase real y clase predicha con el umbral {formato.decimal(m.Umbral)}</caption>
              <thead>
                <tr>
                  <td />
                  <th scope="col">Predicho: con signos</th>
                  <th scope="col">Predicho: sin signos</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <th scope="row">Real: con signos</th>
                  <td className="acierto" data-celda="VP">
                    <abbr title="verdaderos positivos">VP</abbr> {formato.entero(vp)}
                  </td>
                  <td className="fallo" data-celda="FN">
                    <abbr title="falsos negativos">FN</abbr> {formato.entero(fn)}
                  </td>
                </tr>
                <tr>
                  <th scope="row">Real: sin signos</th>
                  <td className="fallo" data-celda="FP">
                    <abbr title="falsos positivos">FP</abbr> {formato.entero(fp)}
                  </td>
                  <td className="acierto" data-celda="VN">
                    <abbr title="verdaderos negativos">VN</abbr> {formato.entero(vn)}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <dl className="cifras cifras-kpi">
            <div>
              <dt>Umbral</dt>
              <dd data-kpi="umbral">{formato.decimal(m.Umbral)}</dd>
            </div>
            <div>
              <dt>Exactitud</dt>
              <dd data-kpi="exactitud">{formato.porcentaje(m.Exactitud)}</dd>
            </div>
            <div>
              <dt>Precisión</dt>
              <dd data-kpi="precision">{formato.porcentaje(m.Precision)}</dd>
            </div>
            <div>
              <dt>Recall</dt>
              <dd data-kpi="recall">{formato.porcentaje(m.Recall)}</dd>
            </div>
            <div>
              <dt>F1</dt>
              <dd data-kpi="f1">{formato.decimal(m.F1)}</dd>
            </div>
          </dl>
          <p className="nota">
            Fuente: {ORIGEN} · <code>metricas_test</code> ({formato.entero(vp + fn + fp + vn)} casos)
          </p>
        </section>

        <section className="tarjeta" data-seccion="latencia" aria-labelledby="titulo-latencia">
          <h2 id="titulo-latencia" className="titulo-panel">
            Latencia de la API
          </h2>
          <p className="bajada">
            POST /api/v1/predicciones: p95 de {formato.uno(peorP95)} ms en la peor corrida, frente al límite de {formato.entero(lat.limite_p95_ms)} ms.
          </p>
          <div className="tabla-envoltura">
            <table className="tabla">
              <caption className="visualmente-oculto">Percentiles de latencia por corrida, en milisegundos</caption>
              <thead>
                <tr>
                  <th scope="col">Corrida</th>
                  <th scope="col">p50 (ms)</th>
                  <th scope="col">p95 (ms)</th>
                  <th scope="col">p99 (ms)</th>
                  <th scope="col">Pet./s</th>
                  <th scope="col">Errores</th>
                  <th scope="col">Límite p95</th>
                </tr>
              </thead>
              <tbody>
                {lat.corridas.map((c) => (
                  <tr key={c.nombre} data-corrida={c.nombre}>
                    <td>
                      {nombreCorrida(c.nombre)} · {formato.entero(c.peticiones)} pet.
                    </td>
                    <td>{formato.uno(c.p50_ms)}</td>
                    <td>
                      <strong>{formato.uno(c.p95_ms)}</strong>
                    </td>
                    <td>{formato.uno(c.p99_ms)}</td>
                    <td>{formato.entero(c.rps)}</td>
                    <td>{formato.entero(c.errores)}</td>
                    <td>
                      <span className={c.cumple_limite ? 'cuadra' : 'no-cuadra'}>
                        {c.cumple_limite ? '✓ cumple' : '✗ no cumple'} &lt; {formato.entero(lat.limite_p95_ms)} ms
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="nota">
            Fuente: <Fuente archivo={benchApi} /> (vía {ORIGEN} · <code>latencia_api</code>) · máquina {lat.maquina}
          </p>
        </section>
      </div>

      <section className="tarjeta" data-seccion="escalabilidad" aria-labelledby="titulo-escalabilidad">
        <h2 id="titulo-escalabilidad" className="titulo-panel">
          Escalabilidad por número de nodos
        </h2>
        <p className="bajada">
          Con {ultima.nodos} nodos una época baja de {ms(primera.seg_por_epoca.media_recortada)} ms ({primera.nodos} nodo
          {primera.nodos === 1 ? '' : 's'}) a {ms(ultima.seg_por_epoca.media_recortada)} ms: speedup {formato.decimal(ultima.speedup)} y eficiencia{' '}
          {formato.porcentaje(ultima.eficiencia)}. La red y la espera del coordinador son {formato.porcentaje(ultima.desglose_por_epoca.fraccion_red_y_espera)}{' '}
          de la época.
        </p>
        <div className="evaluacion-columnas">
          <figure className="grafico">
            <GraficoNodos mediciones={med} serie="speedup" />
            <figcaption>
              Speedup frente al ideal (línea punteada). Fuente: <Fuente archivo={benchCluster} /> (vía {ORIGEN} · <code>escalabilidad</code>)
            </figcaption>
          </figure>
          <figure className="grafico">
            <GraficoNodos mediciones={med} serie="eficiencia" />
            <figcaption>
              Eficiencia frente al 100 % (línea punteada). Fuente: {benchCluster ?? ORIGEN} (vía {ORIGEN} · <code>escalabilidad</code>)
            </figcaption>
          </figure>
        </div>
        <div className="tabla-envoltura">
          <table className="tabla">
            <caption className="visualmente-oculto">Tiempo por época, speedup y eficiencia por número de nodos</caption>
            <thead>
              <tr>
                <th scope="col">Nodos</th>
                <th scope="col">ms por época (media recortada)</th>
                <th scope="col">IC 95 % (ms)</th>
                <th scope="col">Speedup</th>
                <th scope="col">Eficiencia</th>
                <th scope="col">Red y espera</th>
              </tr>
            </thead>
            <tbody>
              {med.map((x) => (
                <tr key={x.nodos} data-nodos={x.nodos}>
                  <td>{x.nodos}</td>
                  <td>{ms(x.seg_por_epoca.media_recortada)}</td>
                  <td>
                    {ms(x.seg_por_epoca.ic95_inf)} – {ms(x.seg_por_epoca.ic95_sup)}
                  </td>
                  <td>{formato.decimal(x.speedup)}</td>
                  <td>{formato.porcentaje(x.eficiencia)}</td>
                  <td>{formato.porcentaje(x.desglose_por_epoca.fraccion_red_y_espera)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="nota">
          Máquina: {maq.cpu} · {maq.so}
          {maq.arquitectura ? `/${maq.arquitectura}` : ''}
          {maq.num_cpu ? ` · ${maq.num_cpu} CPU lógicas` : ''}
        </p>
      </section>
    </div>
  )
}
