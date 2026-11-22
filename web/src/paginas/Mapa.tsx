import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ErrorApi } from '../api/cliente'
import { TODAS, TODOS, type DocumentoRiesgo } from '../api/tipos'
import { IndiceRiesgo, etiquetaDepto, etiquetaEtapa, formato, valor, type Metrica } from '../mapa/datos'
import MapaPeru from '../mapa/MapaPeru'
import PanelDepto from '../mapa/PanelDepto'
import { useSesion } from '../sesion/Sesion'

const VERSION_ESPERADA = 1

/** Lee y valida los filtros de la URL (?semana=20&etapa=…&metrica=…&depto=…): se pueden compartir. */
function filtros(p: URLSearchParams, indice: IndiceRiesgo | null) {
  const semana = Number(p.get('semana') ?? 0)
  const etapa = p.get('etapa') ?? TODAS
  const metrica: Metrica = p.get('metrica') === 'esperados' ? 'esperados' : 'riesgo'
  return {
    semana: Number.isInteger(semana) && semana >= 0 && semana <= 52 ? semana : 0,
    etapa: indice && !indice.cubo.etapas.includes(etapa) ? TODAS : etapa,
    metrica,
    depto: p.get('depto'),
  }
}

export default function Mapa() {
  const { api } = useSesion()
  const [doc, setDoc] = useState<DocumentoRiesgo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [params, setParams] = useSearchParams()
  const [reproduciendo, setReproduciendo] = useState(false)
  const [tabla, setTabla] = useState(false)

  useEffect(() => {
    let vivo = true
    api
      .get<DocumentoRiesgo>('/api/v1/riesgo')
      .then((d) => {
        if (!vivo) return
        if (d?.version_esquema !== VERSION_ESPERADA) throw new Error('versión de datos no soportada')
        setDoc(d)
      })
      .catch((e: unknown) => vivo && setError(e instanceof ErrorApi ? e.message : String(e)))
    return () => {
      vivo = false
    }
  }, [api])

  const indice = useMemo(() => (doc ? new IndiceRiesgo(doc.riesgo) : null), [doc])
  const f = filtros(params, indice)

  function cambiar(cambios: Record<string, string | number | null>) {
    const p = new URLSearchParams(params)
    for (const [k, v] of Object.entries(cambios)) {
      if (v === null || v === '' || (k === 'semana' && v === 0) || (k === 'etapa' && v === TODAS) || (k === 'metrica' && v === 'riesgo')) p.delete(k)
      else p.set(k, String(v))
    }
    setParams(p, { replace: true })
  }

  // Reproducción semana a semana (1 → 52).
  const semanaRef = useRef(f.semana)
  semanaRef.current = f.semana
  useEffect(() => {
    if (!reproduciendo) return
    const id = setInterval(() => {
      const s = semanaRef.current
      if (s >= 52) {
        setReproduciendo(false)
        return
      }
      cambiar({ semana: s + 1 })
    }, 700)
    return () => clearInterval(id)
  }, [reproduciendo, params])

  if (error) {
    return (
      <section className="tarjeta">
        <h1 className="titulo">Mapa de riesgo</h1>
        <p className="error" role="alert">
          No se pudo cargar el riesgo: {error}
        </p>
      </section>
    )
  }
  if (!indice || !doc) {
    return (
      <section className="tarjeta">
        <h1 className="titulo">Mapa de riesgo</h1>
        <p role="status">Cargando el riesgo por departamento…</p>
      </section>
    )
  }

  const nacional = indice.celda(TODOS, f.semana, f.etapa)
  const filas = [...indice.modelo]
    .map((d) => ({ d, c: indice.celda(d, f.semana, f.etapa) }))
    .filter((x) => x.c)
    .sort((a, b) => (valor(b.c, f.metrica) ?? 0) - (valor(a.c, f.metrica) ?? 0))

  return (
    <div className="vista-mapa">
      <header className="vista-cabecera">
        <p className="antetitulo">Modelo {doc.version_modelo}</p>
        <h1 className="titulo">Mapa de riesgo</h1>
        <p className="bajada">
          Riesgo de dengue con signos de alarma o grave por departamento. Elige una semana epidemiológica y un grupo
          de edad; «casos esperados» suma el riesgo de cada caso notificado y es lo que ocupa camas.
        </p>
      </header>

      <form className="filtros tarjeta" onSubmit={(e) => e.preventDefault()} aria-label="Filtros del mapa">
        <div className="filtro-semana">
          <label htmlFor="semana">
            Semana epidemiológica: <strong>{f.semana === 0 ? 'todas' : f.semana}</strong>
          </label>
          <input
            id="semana"
            type="range"
            min={0}
            max={52}
            value={f.semana}
            aria-valuetext={f.semana === 0 ? 'todas las semanas' : `semana ${f.semana}`}
            onChange={(e) => cambiar({ semana: Number(e.target.value) })}
          />
          <button
            type="button"
            className="boton boton-fantasma"
            aria-pressed={reproduciendo}
            onClick={() => {
              if (!reproduciendo && f.semana >= 52) cambiar({ semana: 1 })
              setReproduciendo(!reproduciendo)
            }}
          >
            {reproduciendo ? 'Pausar' : 'Reproducir semanas'}
          </button>
        </div>
        <div>
          <label htmlFor="etapa">Grupo de edad</label>
          <select id="etapa" value={f.etapa} onChange={(e) => cambiar({ etapa: e.target.value })}>
            {indice.cubo.etapas.map((e) => (
              <option key={e} value={e}>
                {etiquetaEtapa(e)}
              </option>
            ))}
          </select>
        </div>
        <fieldset>
          <legend>Medida</legend>
          {(
            [
              ['riesgo', 'Riesgo calibrado'],
              ['esperados', 'Casos esperados'],
            ] as const
          ).map(([m, texto]) => (
            <label key={m} className="opcion">
              <input type="radio" name="metrica" value={m} checked={f.metrica === m} onChange={() => cambiar({ metrica: m })} />
              {texto}
            </label>
          ))}
        </fieldset>
        {nacional && (
          <p className="nacional">
            Perú: {formato.porcentaje(nacional.riesgo_calibrado_medio)} de riesgo ·{' '}
            {formato.numero(nacional.casos_esperados)} casos esperados en {formato.entero(nacional.casos)} notificados
          </p>
        )}
      </form>

      <div className="mapa-y-panel">
        <MapaPeru
          indice={indice}
          semana={f.semana}
          etapa={f.etapa}
          metrica={f.metrica}
          seleccionado={f.depto}
          onSeleccionar={(d) => cambiar({ depto: d })}
        />
        {f.depto ? (
          <PanelDepto indice={indice} depto={f.depto} semana={f.semana} etapa={f.etapa} />
        ) : (
          <aside className="tarjeta panel-depto">
            <p>Elige un departamento en el mapa para ver su curva por semana.</p>
          </aside>
        )}
      </div>

      <section className="tarjeta" aria-labelledby="titulo-tabla">
        <div className="tabla-cabecera">
          <h2 id="titulo-tabla" className="titulo-panel">
            Ranking de departamentos
          </h2>
          <button type="button" className="boton boton-fantasma" aria-expanded={tabla} onClick={() => setTabla(!tabla)}>
            {tabla ? 'Ocultar tabla' : 'Ver como tabla'}
          </button>
        </div>
        {tabla && (
          <table className="tabla">
            <thead>
              <tr>
                <th scope="col">Departamento</th>
                <th scope="col">Riesgo calibrado</th>
                <th scope="col">Tasa observada</th>
                <th scope="col">Casos esperados</th>
                <th scope="col">Casos</th>
              </tr>
            </thead>
            <tbody>
              {filas.map(({ d, c }) => (
                <tr key={d} className={c!.insuficiente ? 'fila-insuficiente' : undefined}>
                  <th scope="row">
                    <button type="button" className="enlace" onClick={() => cambiar({ depto: d })}>
                      {etiquetaDepto(d)}
                    </button>
                    {c!.insuficiente && <span className="etiqueta-insuf"> · insuficiente</span>}
                  </th>
                  <td>{formato.porcentaje(c!.riesgo_calibrado_medio)}</td>
                  <td>{formato.porcentaje(c!.tasa_observada)}</td>
                  <td>{formato.numero(c!.casos_esperados)}</td>
                  <td>{formato.entero(c!.casos)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}
