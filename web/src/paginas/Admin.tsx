import { useCallback, useEffect, useReducer, useRef, useState, type FormEvent } from 'react'
import { ErrorApi } from '../api/cliente'
import { BarrasPesos, CurvaPerdida, Topologia } from '../admin/Graficos'
import { MapaVivo } from '../admin/MapaVivo'
import { pesosDelModelo } from '../admin/mapaVivo'
import { parsearJsonl, reproducir, VELOCIDADES } from '../admin/replica'
import {
  anotar,
  conCluster,
  ESTADO_INICIAL,
  esEstadoCluster,
  type MetricasAPI,
  esModeloParaSello,
  horaBitacora,
  nodosOrdenados,
  reducir,
  reiniciarEntrenamiento,
  selloBitABit,
  type EstadoPanel,
  type EventoCluster,
  type ModeloParaSello,
  type Nivel,
  type Origen,
  type Sello,
} from '../admin/estado'
import { conectar, type EstadoConexion } from '../admin/ws'
import { formato } from '../mapa/datos'
import { etiquetaVariable } from '../prediccion/datos'
import { useSesion } from '../sesion/Sesion'

// Panel admin en vivo (#69): lanza un entrenamiento, lo sigue por WebSocket
// (pérdida, pesos, topología y bitácora), corta un nodo en modo demo y, al
// publicarse el modelo, compara bit a bit los pesos anunciados con los que
// sirve la API. Sin clúster, reproduce un eventos_*.jsonl con el mismo reducer.

type Accion =
  | { tipo: 'evento'; evento: EventoCluster; origen?: Origen }
  | { tipo: 'anotar'; texto: string; nivel: Nivel }
  | { tipo: 'cluster'; cluster: Parameters<typeof conCluster>[1] }
  | { tipo: 'reiniciar' }
  | { tipo: 'vaciar' }

function reductor(e: EstadoPanel, a: Accion): EstadoPanel {
  switch (a.tipo) {
    case 'evento':
      return reducir(e, a.evento, a.origen)
    case 'anotar':
      return anotar(e, a.texto, a.nivel)
    case 'cluster':
      return conCluster(e, a.cluster)
    case 'reiniciar':
      return reiniciarEntrenamiento(e)
    case 'vaciar':
      return ESTADO_INICIAL
  }
}

interface EntrenamientoApi {
  id: string
  estado: 'en_curso' | 'terminado' | 'fallido'
  epocas: number
  epoca: number
  version_modelo?: string
  error?: string
}

const esEntrenamiento = (v: unknown): v is EntrenamientoApi => {
  const t = v as Record<string, unknown> | null
  return !!t && typeof t === 'object' && typeof t.id === 'string' && typeof t.estado === 'string' && Number.isInteger(t.epocas) && Number.isInteger(t.epoca)
}

const SONDEO_MS = 2000
const textoConexion: Record<EstadoConexion['estado'], string> = {
  conectando: 'Conectando…',
  abierto: 'En vivo',
  reintentando: 'Reconectando…',
  cerrado: 'Desconectado',
}

export default function Admin() {
  const { api, apiUrl } = useSesion()
  const [estado, despachar] = useReducer(reductor, ESTADO_INICIAL)
  const [conexion, setConexion] = useState<EstadoConexion>({ estado: 'conectando' })
  const [modelo, setModelo] = useState<ModeloParaSello & { nombres: string[] } | null>(null)
  const [entrenamiento, setEntrenamiento] = useState<EntrenamientoApi | null>(null)
  const [form, setForm] = useState({ epocas: '600', tasa: '2', ritmo: '20' })
  const [enviando, setEnviando] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [sello, setSello] = useState<Sello | null>(null)
  const [cortando, setCortando] = useState<string | null>(null)
  const [metricasApi, setMetricasApi] = useState<MetricasAPI | null>(null)
  const [replica, setReplica] = useState<{ nombre: string; eventos: EventoCluster[] } | null>(null)
  const [velocidad, setVelocidad] = useState<number>(VELOCIDADES[1])
  const [reproduciendo, setReproduciendo] = useState(false)
  const enReplica = useRef(false)
  const detenerReplica = useRef<(() => void) | null>(null)
  const bitacora = useRef<HTMLOListElement>(null)

  const cargarModelo = useCallback(async () => {
    const m = await api.get<unknown>('/api/v1/modelo')
    if (!esModeloParaSello(m)) throw new Error('el modelo publicado no tiene el formato esperado')
    const nombres = [...m.variables.map((v) => etiquetaVariable(v.nombre)), 'Sesgo (intercepto)']
    setModelo({ ...m, nombres })
    return m
  }, [api])

  // Nodos actuales (los registros ya pasaron) y modelo publicado.
  useEffect(() => {
    let vivo = true
    api
      .get<unknown>('/api/v1/cluster')
      .then((c) => {
        if (!vivo || !esEstadoCluster(c)) return
        despachar({ tipo: 'cluster', cluster: c })
        setMetricasApi(c.api ?? null)
      })
      .catch((e: unknown) => vivo && despachar({ tipo: 'anotar', nivel: 'aviso', texto: `no se pudo leer el clúster: ${e instanceof Error ? e.message : e}` }))
    cargarModelo().catch(() => undefined)
    return () => {
      vivo = false
    }
  }, [api, cargarModelo])

  // WebSocket en vivo, con ticket nuevo en cada reconexión.
  useEffect(() => {
    const c = conectar({
      apiUrl,
      pedirTicket: async () => {
        const r = await api.post<{ ticket?: unknown }>('/api/v1/ws/ticket')
        if (typeof r?.ticket !== 'string' || r.ticket === '') throw new Error('la API no devolvió un ticket')
        return r.ticket
      },
      alEvento: (e) => {
        if (!enReplica.current) despachar({ tipo: 'evento', evento: e })
      },
      alEstado: (e) => {
        setConexion(e)
        if (e.estado === 'reintentando') despachar({ tipo: 'anotar', nivel: 'aviso', texto: `${e.motivo}: reintento en ${(e.enMs / 1000).toFixed(1)} s con un ticket nuevo` })
        if (e.estado === 'abierto') despachar({ tipo: 'anotar', nivel: 'info', texto: 'WebSocket conectado' })
      },
    })
    return () => c.cerrar()
  }, [api, apiUrl])

  // Mientras hay un entrenamiento en curso: su estado y el de los nodos (la carga cambia al reasignar).
  // Depende del id y del estado, no del objeto: cada sondeo trae uno nuevo y
  // recrear el intervalo cada 2 s lo desfasaba (PR-87 H3).
  const idEntrenamiento = entrenamiento?.id
  const enCursoApi = entrenamiento?.estado === 'en_curso'
  useEffect(() => {
    if (!idEntrenamiento || !enCursoApi) return
    let vivo = true
    const id = window.setInterval(() => {
      api
        .get<unknown>(`/api/v1/entrenamientos/${encodeURIComponent(idEntrenamiento)}`)
        .then((t) => {
          if (!vivo || !esEntrenamiento(t)) return
          setEntrenamiento(t)
          if (t.estado === 'fallido') despachar({ tipo: 'anotar', nivel: 'alerta', texto: `el entrenamiento falló: ${t.error ?? 'sin detalle'}` })
          if (t.estado === 'terminado') despachar({ tipo: 'anotar', nivel: 'info', texto: `entrenamiento terminado: modelo ${t.version_modelo ?? '¿?'}` })
        })
        .catch(() => undefined)
    }, SONDEO_MS)
    return () => {
      vivo = false
      window.clearInterval(id)
    }
  }, [api, idEntrenamiento, enCursoApi])

  // Métricas del clúster y de la API (#91, enunciado TB2 §6: «uso CPU, latencia,
  // número de nodos, logs»): GET /api/v1/cluster cada SONDEO_MS mientras el panel
  // está abierto, entrene o no. En réplica no se mezcla el clúster en vivo.
  useEffect(() => {
    if (reproduciendo) return
    let vivo = true
    const id = window.setInterval(() => {
      api
        .get<unknown>('/api/v1/cluster')
        .then((c) => {
          if (!vivo || !esEstadoCluster(c)) return
          despachar({ tipo: 'cluster', cluster: c })
          setMetricasApi(c.api ?? null)
        })
        .catch(() => undefined)
    }, SONDEO_MS)
    return () => {
      vivo = false
      window.clearInterval(id)
    }
  }, [api, reproduciendo])

  // Al anunciarse el modelo por el WebSocket: sello bit a bit contra GET /api/v1/modelo.
  // Solo el modelo anunciado en vivo: el de una réplica es de otra corrida y no
  // se compara con el publicado ahora (PR-87 H1).
  useEffect(() => {
    if (!estado.modelo || estado.modelo.origen !== 'vivo') return
    const anunciado = estado.modelo
    let vivo = true
    cargarModelo()
      .then((m) => vivo && setSello(selloBitABit(anunciado, m)))
      .catch((e: unknown) => vivo && setSello({ ok: false, motivo: e instanceof Error ? e.message : String(e) }))
    return () => {
      vivo = false
    }
  }, [estado.modelo, cargarModelo])

  async function lanzar(ev: FormEvent) {
    ev.preventDefault()
    setError(null)
    setSello(null)
    const epocas = Number(form.epocas)
    const tasa = Number(form.tasa)
    const ritmo = Number(form.ritmo)
    if (!Number.isInteger(epocas) || epocas < 1 || !Number.isFinite(tasa) || !(tasa > 0) || !Number.isInteger(ritmo) || ritmo < 0) {
      setError('Revisa los parámetros: épocas entero ≥ 1, tasa > 0 y ritmo entero ≥ 0.')
      return
    }
    setEnviando(true)
    despachar({ tipo: 'reiniciar' })
    const pedir = (cuerpo: object) => api.post<unknown>('/api/v1/entrenamientos', cuerpo)
    try {
      let t: unknown
      try {
        t = await pedir(ritmo > 0 ? { epocas, tasa, ritmo_ms: ritmo } : { epocas, tasa })
      } catch (e) {
        // Sin -demo, la API rechaza ritmo_ms: se lanza igual, sin pausa entre épocas.
        if (!(e instanceof ErrorApi && e.estado === 400 && /ritmo_ms/.test(e.message))) throw e
        despachar({ tipo: 'anotar', nivel: 'aviso', texto: 'la API no está en modo demo: se lanza sin ritmo_ms' })
        t = await pedir({ epocas, tasa })
      }
      if (!esEntrenamiento(t)) throw new Error('la API respondió con un formato inesperado')
      setEntrenamiento(t)
      despachar({ tipo: 'anotar', nivel: 'info', texto: `entrenamiento ${t.id} lanzado: ${epocas} épocas, tasa ${tasa}` })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setEnviando(false)
    }
  }

  async function tumbar(nombre: string) {
    setCortando(nombre)
    try {
      await api.post(`/api/v1/demo/nodos/${encodeURIComponent(nombre)}/cortar`)
      despachar({ tipo: 'anotar', nivel: 'aviso', texto: `se pidió cortar ${nombre}` })
    } catch (e) {
      const msg = e instanceof ErrorApi && e.estado === 404 ? `${e.message} (¿el coordinador corre con DEMO=true?)` : e instanceof Error ? e.message : String(e)
      despachar({ tipo: 'anotar', nivel: 'alerta', texto: `no se pudo cortar ${nombre}: ${msg}` })
    } finally {
      setCortando(null)
    }
  }

  async function elegirArchivo(archivo: File | undefined) {
    setError(null)
    if (!archivo) return
    try {
      setReplica({ nombre: archivo.name, eventos: parsearJsonl(await archivo.text()) })
    } catch (e) {
      setReplica(null)
      setError(`${archivo.name}: ${e instanceof Error ? e.message : String(e)}`)
    }
  }

  function alternarReplica() {
    if (reproduciendo) {
      detenerReplica.current?.()
      detenerReplica.current = null
      enReplica.current = false
      setReproduciendo(false)
      return
    }
    if (!replica) return
    enReplica.current = true
    setReproduciendo(true)
    setSello(null)
    setEntrenamiento(null) // sus «/ N» y su estado son de la corrida en vivo (PR-87 H1)
    despachar({ tipo: 'vaciar' })
    despachar({ tipo: 'anotar', nivel: 'info', texto: `réplica de ${replica.nombre} a ×${velocidad} (${replica.eventos.length} eventos); los eventos en vivo se ignoran` })
    detenerReplica.current = reproducir(replica.eventos, velocidad, (e) => despachar({ tipo: 'evento', evento: e, origen: 'replica' }), {
      alTerminar: () => {
        enReplica.current = false
        setReproduciendo(false)
        despachar({ tipo: 'anotar', nivel: 'info', texto: 'fin de la réplica' })
      },
    }).detener
  }

  useEffect(() => () => detenerReplica.current?.(), [])
  // La bitácora se queda en la última línea, como una terminal.
  useEffect(() => {
    const el = bitacora.current
    if (el) el.scrollTop = el.scrollHeight
  }, [estado.bitacora])

  const nodos = nodosOrdenados(estado)
  const enCurso = entrenamiento?.estado === 'en_curso'
  const ultima = estado.curva.at(-1)

  return (
    <div className="vista-admin">
      <header className="vista-cabecera">
        <p className="antetitulo">Rol admin · parameter server síncrono</p>
        <h1 id="titulo-vista" className="titulo">
          Clúster y entrenamiento en vivo
        </h1>
        <p className="bajada">
          Cada época, el coordinador manda los pesos a los nodos, espera el gradiente de cada shard y lo reduce en orden de bloque. Aquí se ve en vivo por
          WebSocket; si un nodo cae, su shard pasa a otro y el resultado no cambia.
        </p>
        <p className={`conexion conexion-${conexion.estado}`} role="status" data-conexion={conexion.estado}>
          {reproduciendo ? 'Réplica' : textoConexion[conexion.estado]}
        </p>
      </header>

      <div className="admin-columnas">
        <section className="tarjeta" aria-labelledby="titulo-entrenar" data-seccion="entrenar">
          <h2 id="titulo-entrenar" className="titulo-panel">
            Entrenamiento
          </h2>
          <form className="formulario" onSubmit={lanzar} noValidate>
            <div className="formulario-fila formulario-fila-3">
              <div>
                <label htmlFor="adm-epocas">Épocas</label>
                <input id="adm-epocas" inputMode="numeric" value={form.epocas} onChange={(e) => setForm({ ...form, epocas: e.target.value })} />
              </div>
              <div>
                <label htmlFor="adm-tasa">Tasa</label>
                <input id="adm-tasa" inputMode="decimal" value={form.tasa} onChange={(e) => setForm({ ...form, tasa: e.target.value })} />
              </div>
              <div>
                <label htmlFor="adm-ritmo">Pausa (ms, demo)</label>
                <input id="adm-ritmo" inputMode="numeric" value={form.ritmo} onChange={(e) => setForm({ ...form, ritmo: e.target.value })} />
              </div>
            </div>
            <button type="submit" className="boton" disabled={enviando || enCurso || reproduciendo}>
              {enCurso ? 'Entrenando…' : enviando ? 'Lanzando…' : 'Lanzar entrenamiento'}
            </button>
          </form>
          <p className="nota-clase">Al terminar, la API publica el modelo nuevo: la predicción y el inicio pasan a usarlo.</p>
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
          <dl className="cifras">
            <div>
              <dt>Época</dt>
              <dd data-kpi="epoca">
                {estado.epoca}
                {entrenamiento ? ` / ${entrenamiento.epocas}` : ''}
              </dd>
            </div>
            <div>
              <dt>Pérdida</dt>
              <dd data-kpi="perdida">{ultima ? ultima.perdida.toFixed(6) : '—'}</dd>
            </div>
            <div>
              <dt>Reasignaciones</dt>
              <dd data-kpi="reasignaciones">{estado.reasignaciones}</dd>
            </div>
            <div>
              <dt>Nodos sanos</dt>
              <dd data-kpi="nodos">
                {nodos.filter((n) => n.sano).length} / {nodos.length}
              </dd>
            </div>
            <div>
              <dt title="Latencia de las predicciones dentro del servidor, últimas 1024">p95 API</dt>
              <dd data-kpi="p95-api">
                {metricasApi && metricasApi.ventana > 0 ? (
                  <>
                    {formato.uno(metricasApi.p95_ms)} ms{' '}
                    <span className={metricasApi.cumple_limite_p95 ? 'cuadra' : 'no-cuadra'}>
                      {metricasApi.cumple_limite_p95 ? '✓' : '✗'} &lt; {formato.entero(metricasApi.limite_p95_ms)} ms
                    </span>
                  </>
                ) : (
                  '—'
                )}
              </dd>
            </div>
            <div>
              <dt>Estado</dt>
              <dd data-kpi="estado">
                {entrenamiento
                  ? entrenamiento.estado.replace('_', ' ')
                  : reproduciendo
                    ? 'réplica'
                    : estado.abortado
                      ? 'abortado'
                      : estado.terminado
                        ? 'terminado'
                        : 'sin lanzar'}
              </dd>
            </div>
          </dl>
          {sello && (
            <div className={sello.ok ? 'sello sello-ok' : 'sello sello-mal'} role="status" data-sello={sello.ok ? 'ok' : 'mal'}>
              <strong>{sello.ok ? 'bit a bit ✓' : 'bit a bit ✗'}</strong>
              <span>
                {sello.ok
                  ? `Los ${sello.comparados} pesos anunciados por el WebSocket son idénticos (===) a los de GET /api/v1/modelo (${estado.modelo?.version}).`
                  : sello.motivo}
              </span>
            </div>
          )}
          <p className="nota-clase">
            Que esos pesos sean además iguales bit a bit a los del entrenamiento secuencial, aunque se caiga un nodo, lo comprueba el test de Go{' '}
            <code>TestServicioDemoCortarNodoPorLaAPI</code> (cmd/coordinator).
          </p>
        </section>

        <section className="tarjeta" aria-labelledby="titulo-topologia" data-seccion="topologia">
          <h2 id="titulo-topologia" className="titulo-panel">
            Topología
          </h2>
          <figure className="grafico">
            <Topologia nodos={nodos} />
            <figcaption>Estrella: el coordinador agrega; cada nodo calcula el gradiente de sus shards. Rojo = fuera del clúster; anillo rosa = se cayó y volvió a registrarse.</figcaption>
          </figure>
          <p className="nota-clase" data-prueba="nota-metricas">
            Por nodo, de su última época: <strong>CPU</strong> del proceso mientras calcula (sobre sus workers) y <strong>latencia</strong> desde que
            el coordinador le envía la tarea hasta que recibe su gradiente, separada en cálculo y red. El p95 de la API se mide dentro del
            servidor; el de cliente, con red, lo mide <code>cmd/benchapi</code>.
          </p>
          {nodos.length === 0 ? (
            <p className="nota">Todavía no hay nodos registrados.</p>
          ) : (
            <ul className="lista-nodos">
              {nodos.map((n) => (
                <li key={n.nombre} data-nodo={n.nombre}>
                  <span className={n.sano ? 'punto-sano' : 'punto-caido'} aria-hidden="true" />
                  <span className="lista-nodos-nombre">{n.nombre}</span>
                  <span className="lista-nodos-info">{n.sano ? `${n.carga} shard${n.carga === 1 ? '' : 's'} · ${n.workers} workers` : (n.motivo ?? 'caído')}
                    {(n.caidas ?? 0) > 0 ? ` · cayó ${n.caidas}×` : ''}</span>
                  {n.vista && (
                    <span className="lista-nodos-metricas" data-metricas={n.nombre}>
                      <span data-metrica="cpu" title="CPU del proceso del nodo durante el cálculo, sobre la capacidad de sus workers">
                        CPU {n.vista.usoCpu === null ? '—' : formato.porcentaje(n.vista.usoCpu)}
                      </span>
                      <span data-metrica="latencia" title="De su primera tarea de la época a su último gradiente, medido en el coordinador">
                        latencia {formato.uno(n.vista.latenciaMs)} ms
                      </span>
                      <span data-metrica="desglose">
                        (cálculo {formato.uno(n.vista.calculoMs)} · red {formato.uno(n.vista.redMs)})
                      </span>
                    </span>
                  )}
                  <button type="button" className="boton boton-fantasma" disabled={!n.sano || cortando !== null || reproduciendo} onClick={() => void tumbar(n.nombre)} aria-label={`Tumbar ${n.nombre}`}>
                    {cortando === n.nombre ? 'Cortando…' : 'Tumbar'}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>

      <div className="admin-columnas">
        <section className="tarjeta" aria-labelledby="titulo-perdida" data-seccion="perdida">
          <h2 id="titulo-perdida" className="titulo-panel">
            Pérdida por época
          </h2>
          <figure className="grafico grafico-ancho">
            <CurvaPerdida curva={estado.curva} epocasPedidas={entrenamiento?.epocas ?? null} />
            <figcaption>Log-loss ponderada del conjunto de entrenamiento, una por época (evento «epoca»).</figcaption>
          </figure>
        </section>
        <section className="tarjeta" aria-labelledby="titulo-mapa-vivo" data-seccion="mapa-vivo">
          <h2 id="titulo-mapa-vivo" className="titulo-panel">
            Mapa en vivo
          </h2>
          <figure className="grafico">
            <MapaVivo
              pesos={estado.pesos ?? pesosDelModelo(modelo)}
              variables={modelo?.variables.map((v) => v.nombre) ?? []}
              origen={estado.pesos ? `época ${estado.epoca}` : modelo ? `modelo publicado ${modelo.version}` : 'sin pesos'}
            />
            <figcaption>
              Se repinta con los pesos de cada época: razón de odds de cada departamento frente al promedio (e<sup>Δ</sup>, Δ = w<sub>depto</sub> − media
              de los w<sub>depto</sub>). No depende de la edad ni de la semana del caso. Antes de la primera época muestra el modelo publicado.
            </figcaption>
          </figure>
        </section>
      </div>

      <div className="admin-columnas">
        <section className="tarjeta" aria-labelledby="titulo-pesos" data-seccion="pesos">
          <h2 id="titulo-pesos" className="titulo-panel">
            Pesos
          </h2>
          <figure className="grafico grafico-ancho">
            <BarrasPesos pesos={estado.pesos} nombres={modelo?.nombres ?? []} />
            <figcaption>Pesos usados en la última época, con el sesgo al final. Rosa sube el riesgo; azul lo baja.</figcaption>
          </figure>
        </section>
      </div>

      <div className="admin-columnas">
        <section className="tarjeta" aria-labelledby="titulo-bitacora" data-seccion="bitacora">
          <h2 id="titulo-bitacora" className="titulo-panel">
            Bitácora
          </h2>
          <ol ref={bitacora} className="bitacora" role="log" aria-labelledby="titulo-bitacora" tabIndex={0}>
            {estado.bitacora.map((l) => (
              <li key={l.id} className={`bitacora-${l.nivel}`} data-tipo={l.tipo}>
                <time dateTime={l.t}>{horaBitacora(l.t)}</time> {l.texto}
              </li>
            ))}
          </ol>
        </section>

        <section className="tarjeta" aria-labelledby="titulo-replica" data-seccion="replica">
          <h2 id="titulo-replica" className="titulo-panel">
            Réplica sin clúster
          </h2>
          <p className="bajada">
            Reproduce un eventos_*.jsonl grabado por el coordinador (<code>-eventos</code> o <code>EVENTOS=/reports/eventos_demo.jsonl</code> en el
            Compose), con el mismo reducer que en vivo.
          </p>
          <div className="formulario">
            <label htmlFor="adm-jsonl">Archivo .jsonl</label>
            <input id="adm-jsonl" type="file" accept=".jsonl,application/jsonl,application/x-ndjson" onChange={(e) => void elegirArchivo(e.target.files?.[0])} disabled={reproduciendo} />
            <label htmlFor="adm-velocidad">Velocidad</label>
            <select id="adm-velocidad" value={velocidad} onChange={(e) => setVelocidad(Number(e.target.value))} disabled={reproduciendo}>
              {VELOCIDADES.map((v) => (
                <option key={v} value={v}>
                  ×{v}
                </option>
              ))}
            </select>
            <button type="button" className="boton" onClick={alternarReplica} disabled={!replica || enCurso}>
              {reproduciendo ? 'Detener réplica' : 'Reproducir'}
            </button>
          </div>
          {replica && (
            <p className="nota" data-replica={replica.eventos.length}>
              {replica.nombre}: {replica.eventos.length} eventos válidos
            </p>
          )}
        </section>
      </div>
    </div>
  )
}
