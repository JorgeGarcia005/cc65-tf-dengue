import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ErrorApi } from '../api/cliente'
import type { DocumentoEvaluacion, ModeloPublicado } from '../api/tipos'
import { enlaceMapa, esEvaluacion, gruposDeEdad, proporcion, semanasPico, topPrioridades } from '../inicio/datos'
import { etiquetaDepto, etiquetaEtapa, formato } from '../mapa/datos'
import { useSesion } from '../sesion/Sesion'

const VERSION_ESPERADA = 1

// Inicio (#67): la respuesta a la pregunta de impacto antes que cualquier otra
// cosa. Todas las cifras vienen de GET /api/v1/evaluacion (reports/evaluacion.json)
// y de GET /api/v1/modelo; el front no calcula ni inventa ninguna.
export default function Inicio() {
  const { api } = useSesion()
  const [doc, setDoc] = useState<DocumentoEvaluacion | null>(null)
  const [modelo, setModelo] = useState<ModeloPublicado | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let vivo = true
    const fallar = (e: unknown) => vivo && setError(e instanceof ErrorApi ? e.message : String(e))
    api
      .get<unknown>('/api/v1/evaluacion')
      .then((d) => {
        if (!vivo) return
        if (!esEvaluacion(d, VERSION_ESPERADA)) throw new Error('la evaluación no tiene el formato esperado (versión o campos)')
        setDoc(d)
      })
      .catch(fallar)
    // El modelo publicado solo sirve para avisar si la evaluación es de otro: si
    // falla, el inicio sigue.
    api
      .get<ModeloPublicado>('/api/v1/modelo')
      .then((m) => vivo && typeof m?.version === 'string' && setModelo(m))
      .catch(() => undefined)
    return () => {
      vivo = false
    }
  }, [api])

  const cabecera = (
    <>
      <p className="antetitulo">Pregunta de impacto · ODS 3</p>
      <h1 id="titulo-vista" className="titulo">
        Dónde y cuándo reforzar
      </h1>
    </>
  )
  if (error || !doc) {
    return (
      <div className="vista-inicio" aria-busy={!error}>
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

  const top = topPrioridades(doc.prioridades)
  const picos = semanasPico(doc.riesgo)
  const edades = gruposDeEdad(doc.riesgo)
  const m = doc.metricas_test
  const otroModelo = modelo && modelo.version !== doc.version_modelo

  return (
    <div className="vista-inicio">
      <header className="vista-cabecera">
        {cabecera}
        <p className="bajada">
          Combinaciones de departamento, semana epidemiológica y grupo de edad con más <strong>casos esperados con signos de alarma o graves</strong>:
          volumen de casos por riesgo calibrado, que es lo que ocupa camas y vigilancia. Calculado sobre el conjunto de prueba (
          {formato.entero(doc.filas_test)} casos notificados).
        </p>
        {otroModelo && (
          <p className="aviso" role="status">
            Esta evaluación es del modelo {doc.version_modelo}; la API publica el {modelo.version}. Regenera reports/evaluacion.json.
          </p>
        )}
      </header>

      <section className="tarjeta" aria-labelledby="titulo-prioridades">
        <h2 id="titulo-prioridades" className="titulo-panel">
          Las {top.length} prioridades
        </h2>
        <div className="tabla-envoltura">
          <table className="tabla" aria-describedby="titulo-prioridades">
            <thead>
              <tr>
                <th scope="col">#</th>
                <th scope="col">Departamento</th>
                <th scope="col">Semana</th>
                <th scope="col">Grupo de edad</th>
                <th scope="col">Casos</th>
                <th scope="col">Tasa observada</th>
                <th scope="col">Riesgo calibrado</th>
                <th scope="col">Casos esperados</th>
              </tr>
            </thead>
            <tbody>
              {top.map((c, i) => (
                <tr key={`${c.departamento}-${c.semana}-${c.etapa}`} data-prioridad={i + 1}>
                  <td>{i + 1}</td>
                  <td>
                    <Link to={enlaceMapa(c)} aria-label={`Ver ${etiquetaDepto(c.departamento)}, semana ${c.semana}, ${etiquetaEtapa(c.etapa)} en el mapa`}>
                      {etiquetaDepto(c.departamento)}
                    </Link>
                  </td>
                  <td>{c.semana}</td>
                  <td>{etiquetaEtapa(c.etapa)}</td>
                  <td>{formato.entero(c.casos)}</td>
                  <td>{formato.porcentaje(c.tasa_observada)}</td>
                  <td>{formato.porcentaje(c.riesgo_calibrado_medio)}</td>
                  <td>
                    <strong>{formato.uno(c.casos_esperados)}</strong>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="nota">Orden de reports/evaluacion.json · mínimo {doc.riesgo.min_casos} casos por combinación</p>
      </section>

      <div className="inicio-columnas">
        <section className="tarjeta" aria-labelledby="titulo-picos">
          <h2 id="titulo-picos" className="titulo-panel">
            Semanas pico
          </h2>
          <p className="bajada">Nivel nacional, todas las edades, por casos esperados.</p>
          <ol className="barras" aria-label="Semanas con más casos esperados">
            {picos.map((c) => (
              <li key={c.semana} data-semana={c.semana}>
                <span className="barras-etiqueta">Semana {c.semana}</span>
                <span className="barras-barra" style={{ width: `${100 * proporcion(c.casos_esperados, picos[0]!.casos_esperados)}%` }} aria-hidden="true" />
                <span className="barras-valor">{formato.uno(c.casos_esperados)}</span>
              </li>
            ))}
          </ol>
        </section>

        <section className="tarjeta" aria-labelledby="titulo-edades">
          <h2 id="titulo-edades" className="titulo-panel">
            Grupos de edad
          </h2>
          <p className="bajada">Riesgo calibrado nacional, de mayor a menor, frente a la tasa observada.</p>
          <div className="tabla-envoltura">
            <table className="tabla">
              <thead>
                <tr>
                  <th scope="col">Grupo</th>
                  <th scope="col">Riesgo calibrado</th>
                  <th scope="col">Tasa observada</th>
                  <th scope="col">Casos</th>
                </tr>
              </thead>
              <tbody>
                {edades.map((c) => (
                  <tr key={c.etapa} data-etapa={c.etapa}>
                    <td>{etiquetaEtapa(c.etapa)}</td>
                    <td>{formato.porcentaje(c.riesgo_calibrado_medio)}</td>
                    <td>{formato.porcentaje(c.tasa_observada)}</td>
                    <td>{formato.entero(c.casos)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <section className="tarjeta" aria-labelledby="titulo-modelo">
        <h2 id="titulo-modelo" className="titulo-panel">
          El modelo, en cifras
        </h2>
        <dl className="cifras cifras-kpi">
          <div>
            <dt>Versión</dt>
            <dd data-kpi="version">{doc.version_modelo}</dd>
          </div>
          <div>
            <dt>AUC (test)</dt>
            <dd data-kpi="auc">{formato.decimal(doc.auc_trapecio)}</dd>
          </div>
          <div>
            <dt>Recall</dt>
            <dd data-kpi="recall">{formato.porcentaje(m.Recall)}</dd>
          </div>
          <div>
            <dt>Precisión</dt>
            <dd data-kpi="precision">{formato.porcentaje(m.Precision)}</dd>
          </div>
          <div>
            <dt>Brier (calibrado)</dt>
            <dd data-kpi="brier">{formato.decimal(doc.brier_calibrado)}</dd>
          </div>
          <div>
            <dt>Prevalencia en test</dt>
            <dd data-kpi="prevalencia">{formato.porcentaje(proporcion(doc.positivos_test, doc.filas_test))}</dd>
          </div>
        </dl>
        <p className="bajada" data-kpi="lectura">
          Con el umbral {formato.decimal(m.Umbral)} el modelo reconoce {formato.porcentaje(m.Recall)} de los casos con signos de alarma, con una precisión de{' '}
          {formato.porcentaje(m.Precision)}: sirve para priorizar dónde y cuándo vigilar, no para diagnosticar un caso. Detalle en{' '}
          <Link to="/evaluacion">Evaluación</Link>.
        </p>
      </section>
    </div>
  )
}
