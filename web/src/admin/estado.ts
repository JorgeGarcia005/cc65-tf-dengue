// Estado del panel admin (#69) a partir de los eventos del clúster.
// reducir(estado, evento) es puro: lo usan igual el WebSocket en vivo y la
// réplica de un .jsonl grabado.

/** Versión de cluster.Evento que sabe leer la SPA (cluster.VersionEventos). */
export const VERSION_EVENTOS = 1

// «modelo» lo emite la API al publicar el modelo entrenado (versión en detalle,
// pesos finales); los de «epoca» son los usados en esa época, antes de actualizarlos.
export const TIPOS = ['registro', 'tarea', 'gradiente', 'epoca', 'reasignacion', 'descarte', 'abortado', 'fin', 'modelo'] as const
export type TipoEvento = (typeof TIPOS)[number]

/** cluster.MetricasNodo: la última época que calculó el nodo (#91). */
export interface MetricasNodo {
  epoca: number
  shards: number
  calculo_ms: number
  latencia_ms: number
  red_ms: number
  /** 0..1; ausente si el nodo no informa su CPU. */
  uso_cpu?: number
}

/** cluster.EstadoNodo. */
export interface EstadoNodo {
  nombre: string
  remoto: string
  workers: number
  sano: boolean
  carga: number
  motivo?: string
  metricas?: MetricasNodo
}

/** cluster.Evento (internal/cluster/eventos.go), uno por mensaje del WebSocket o línea del .jsonl. */
export interface EventoCluster {
  v: number
  tipo: TipoEvento
  t: string
  epoca: number
  nodo?: string
  shard?: number
  perdida?: number
  pesos?: number[]
  micros_calculo?: number
  /** En «gradiente» (#91): CPU del nodo durante el cálculo y µs desde su primera tarea de la época. */
  micros_cpu?: number
  micros_ida_vuelta?: number
  detalle?: string
  nodos?: EstadoNodo[]
}

/** Métricas que muestra el panel por nodo (#91), de /cluster o de los eventos «gradiente». */
export interface MetricasVista {
  epoca: number
  calculoMs: number
  latenciaMs: number
  redMs: number
  /** 0..1, o null si el nodo no informa su CPU. */
  usoCpu: number | null
}

export interface NodoVista extends EstadoNodo {
  ultimoShard?: number
  microsCalculo?: number
  vista?: MetricasVista
  /** CPU acumulada de la época de `vista` (un nodo con varios shards los calcula en serie). */
  microsCpuEpoca?: number | null
  /** Veces que salió del clúster en esta sesión del panel (un nodo cortado se vuelve a registrar enseguida). */
  caidas?: number
}

export type Nivel = 'info' | 'aviso' | 'alerta'

/** De dónde vino un evento: el WebSocket en vivo o un .jsonl reproducido. */
export type Origen = 'vivo' | 'replica'

export interface LineaBitacora {
  id: number
  t: string
  tipo: TipoEvento | 'local'
  nivel: Nivel
  texto: string
}

export interface EstadoPanel {
  curva: { epoca: number; perdida: number }[]
  /** Pesos de la última época (con el sesgo al final), o null antes de la primera. */
  pesos: number[] | null
  epoca: number
  /**
   * El contador de épocas del coordinador es acumulado entre entrenamientos de
   * una misma corrida: se resta este desfase (la primera época vista − 1) para
   * que cada entrenamiento cuente desde 1. null hasta la primera época.
   */
  baseEpoca: number | null
  nodos: Record<string, NodoVista>
  bitacora: LineaBitacora[]
  reasignaciones: number
  terminado: boolean
  abortado: string | null
  /**
   * El modelo que la API anunció al publicarlo (evento «modelo»). `origen` dice si
   * llegó por el WebSocket en vivo o de una réplica: solo el vivo se compara con
   * GET /api/v1/modelo (PR-87 H1).
   */
  modelo: { version: string; pesos: number[]; origen: Origen } | null
  /** Contador para el id de cada línea de la bitácora (clave estable en React). */
  siguienteLinea: number
}

export const ESTADO_INICIAL: EstadoPanel = {
  curva: [],
  pesos: null,
  epoca: 0,
  baseEpoca: null,
  nodos: {},
  bitacora: [],
  reasignaciones: 0,
  terminado: false,
  abortado: null,
  modelo: null,
  siguienteLinea: 1,
}

/** Líneas que guarda la bitácora: las más viejas se descartan. */
export const MAX_BITACORA = 200
/** Cada cuántas épocas se anota una en la bitácora (la curva las tiene todas). */
export const ANOTAR_CADA = 25

const finito = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v)
const opcional = (v: unknown, ok: (x: unknown) => boolean) => v === undefined || ok(v)
const texto = (v: unknown) => typeof v === 'string'
const entero = (v: unknown) => Number.isInteger(v)

function esMetricasNodo(v: unknown): v is MetricasNodo {
  const m = v as Record<string, unknown> | null
  return (
    !!m &&
    typeof m === 'object' &&
    entero(m.epoca) &&
    entero(m.shards) &&
    finito(m.calculo_ms) &&
    finito(m.latencia_ms) &&
    finito(m.red_ms) &&
    opcional(m.uso_cpu, (u) => finito(u) && (u as number) >= 0 && (u as number) <= 1)
  )
}

function esEstadoNodo(v: unknown): v is EstadoNodo {
  const n = v as Record<string, unknown> | null
  return (
    !!n &&
    typeof n === 'object' &&
    texto(n.nombre) &&
    typeof n.sano === 'boolean' &&
    entero(n.carga) &&
    entero(n.workers) &&
    opcional(n.motivo, texto) &&
    opcional(n.metricas, esMetricasNodo)
  )
}

/** De la forma de /api/v1/cluster a la que pinta el panel. */
function vistaDe(m: MetricasNodo): MetricasVista {
  return { epoca: m.epoca, calculoMs: m.calculo_ms, latenciaMs: m.latencia_ms, redMs: m.red_ms, usoCpu: m.uso_cpu ?? null }
}

/**
 * Métricas en vivo de un evento «gradiente» (#91). Dentro de una misma época
 * los shards de un nodo se suman (los calcula en serie); la latencia es la del
 * último gradiente, medida desde su primera tarea.
 */
function conGradiente(n: NodoVista, ev: EventoCluster): Pick<NodoVista, 'vista' | 'microsCpuEpoca'> {
  if (ev.micros_ida_vuelta === undefined || ev.micros_calculo === undefined) return {}
  const misma = n.vista?.epoca === ev.epoca
  const calcUs = (misma ? n.vista!.calculoMs * 1000 : 0) + ev.micros_calculo
  const previa = misma ? n.microsCpuEpoca : 0
  const cpuUs = previa === null || previa === undefined || !ev.micros_cpu ? null : previa + ev.micros_cpu
  const latUs = ev.micros_ida_vuelta
  const usoCpu = cpuUs !== null && calcUs > 0 && n.workers > 0 ? Math.min(cpuUs / (calcUs * n.workers), 1) : null
  return {
    vista: { epoca: ev.epoca, calculoMs: calcUs / 1000, latenciaMs: latUs / 1000, redMs: Math.max(latUs - calcUs, 0) / 1000, usoCpu },
    microsCpuEpoca: cpuUs,
  }
}

/** Valida un evento antes de reducirlo: un mensaje raro no rompe el panel. */
export function esEvento(v: unknown): v is EventoCluster {
  const e = v as Record<string, unknown> | null
  return (
    !!e &&
    typeof e === 'object' &&
    e.v === VERSION_EVENTOS &&
    (TIPOS as readonly unknown[]).includes(e.tipo) &&
    texto(e.t) &&
    entero(e.epoca) &&
    opcional(e.nodo, texto) &&
    opcional(e.shard, entero) &&
    opcional(e.perdida, finito) &&
    opcional(e.micros_calculo, finito) &&
    opcional(e.micros_cpu, finito) &&
    opcional(e.micros_ida_vuelta, finito) &&
    opcional(e.detalle, texto) &&
    opcional(e.pesos, (p) => Array.isArray(p) && p.every(finito)) &&
    opcional(e.nodos, (ns) => Array.isArray(ns) && ns.every(esEstadoNodo))
  )
}

/** Los nodos en orden de nombre (nodo-0, nodo-1, … o los hostnames del Compose). */
export function nodosOrdenados(e: EstadoPanel): NodoVista[] {
  return Object.values(e.nodos).sort((a, b) => a.nombre.localeCompare(b.nombre, 'es', { numeric: true }))
}

function conLinea(e: EstadoPanel, t: string, tipo: LineaBitacora['tipo'], nivel: Nivel, txt: string): EstadoPanel {
  const linea = { id: e.siguienteLinea, t, tipo, nivel, texto: txt }
  return { ...e, siguienteLinea: e.siguienteLinea + 1, bitacora: [...e.bitacora, linea].slice(-MAX_BITACORA) }
}

/** Una línea de la propia SPA (conexión, reintentos, lanzamiento…), no del coordinador. */
export function anotar(e: EstadoPanel, txt: string, nivel: Nivel, t = new Date().toISOString()): EstadoPanel {
  return conLinea(e, t, 'local', nivel, txt)
}

/** La instantánea de nodos del coordinador reemplaza a la anterior, conservando la actividad vista. */
function conInstantanea(e: EstadoPanel, nodos: EstadoNodo[] | undefined): EstadoPanel {
  if (!nodos) return e
  const m: Record<string, NodoVista> = {}
  for (const n of nodos) {
    const previo = e.nodos[n.nombre]
    // La instantánea de /cluster trae la última época cerrada; si los eventos ya
    // mostraron una más nueva, se conserva la de los eventos.
    const vista = n.metricas && !(previo?.vista && previo.vista.epoca > n.metricas.epoca) ? vistaDe(n.metricas) : previo?.vista
    m[n.nombre] = { ...previo, ...n, vista }
  }
  return { ...e, nodos: m }
}

function tocarNodo(e: EstadoPanel, nombre: string | undefined, cambio: Partial<NodoVista>): EstadoPanel {
  if (!nombre) return e
  const previo: NodoVista = e.nodos[nombre] ?? { nombre, remoto: '', workers: 0, sano: true, carga: 0 }
  return { ...e, nodos: { ...e.nodos, [nombre]: { ...previo, ...cambio } } }
}

/** Época del evento contada desde el inicio del entrenamiento en curso. */
const relativa = (e: EstadoPanel, epoca: number) => (e.baseEpoca === null ? epoca : epoca - e.baseEpoca)

const shardDe = (ev: EventoCluster) => (ev.shard === undefined ? '' : ` el shard ${ev.shard}`)

export function reducir(e: EstadoPanel, ev: EventoCluster, origen: Origen = 'vivo'): EstadoPanel {
  switch (ev.tipo) {
    case 'registro': {
      const s = conInstantanea(tocarNodo(e, ev.nodo, { sano: true }), ev.nodos)
      return conLinea(s, ev.t, ev.tipo, 'info', `${ev.nodo ?? 'un nodo'} se registró (${Object.keys(s.nodos).length} en el clúster)`)
    }
    case 'tarea':
      return tocarNodo(e, ev.nodo, { ultimoShard: ev.shard })
    case 'gradiente': {
      const previo = ev.nodo ? e.nodos[ev.nodo] : undefined
      const metricas = previo ? conGradiente(previo, ev) : {}
      return tocarNodo(e, ev.nodo, { ultimoShard: ev.shard, microsCalculo: ev.micros_calculo, ...metricas })
    }
    case 'epoca': {
      const base = e.baseEpoca ?? ev.epoca - 1
      const epoca = ev.epoca - base
      const s: EstadoPanel = {
        ...e,
        epoca,
        baseEpoca: base,
        curva: ev.perdida === undefined ? e.curva : [...e.curva, { epoca, perdida: ev.perdida }],
        pesos: ev.pesos ?? e.pesos,
      }
      return epoca === 1 || epoca % ANOTAR_CADA === 0 ? conLinea(s, ev.t, ev.tipo, 'info', `época ${epoca} · pérdida ${ev.perdida?.toFixed(6) ?? '—'}`) : s
    }
    case 'reasignacion': {
      const s = tocarNodo({ ...e, reasignaciones: e.reasignaciones + 1 }, ev.nodo, { ultimoShard: ev.shard })
      return conLinea(s, ev.t, ev.tipo, 'aviso', `reasignacion:${shardDe(ev)} pasa a ${ev.nodo ?? '¿?'} en la época ${relativa(e, ev.epoca)}`)
    }
    case 'descarte': {
      const s = conInstantanea(tocarNodo(e, ev.nodo, { sano: false, motivo: ev.detalle, caidas: (ev.nodo ? (e.nodos[ev.nodo]?.caidas ?? 0) : 0) + 1 }), ev.nodos)
      return conLinea(s, ev.t, ev.tipo, 'alerta', `${ev.nodo ?? 'un nodo'} salió del clúster${ev.detalle ? `: ${ev.detalle}` : ''}`)
    }
    case 'abortado':
      return conLinea({ ...e, abortado: ev.detalle ?? 'el entrenamiento falló' }, ev.t, ev.tipo, 'alerta', `entrenamiento abortado: ${ev.detalle ?? 'sin detalle'}`)
    case 'modelo': {
      const s = { ...e, modelo: { version: ev.detalle ?? '', pesos: ev.pesos ?? [], origen } }
      return conLinea(s, ev.t, ev.tipo, 'info', `la API publicó el modelo ${ev.detalle ?? '¿?'} (${ev.pesos?.length ?? 0} pesos)`)
    }
    case 'fin':
      return conLinea({ ...e, terminado: true }, ev.t, ev.tipo, 'info', `fin del entrenamiento tras ${ev.epoca} épocas`)
  }
}

/** Antes de lanzar otro entrenamiento: curva y pesos de cero; nodos y bitácora siguen. */
export function reiniciarEntrenamiento(e: EstadoPanel): EstadoPanel {
  return { ...e, curva: [], pesos: null, epoca: 0, baseEpoca: null, reasignaciones: 0, terminado: false, abortado: null, modelo: null }
}

/** api.MetricasAPI: latencia de las predicciones dentro del servidor (#91). */
export interface MetricasAPI {
  predicciones: number
  ventana: number
  p50_ms: number
  p95_ms: number
  max_ms: number
  aciertos_cache: number
  no_registradas: number
  limite_p95_ms: number
  cumple_limite_p95: boolean
}

/** GET /api/v1/cluster (cluster.Estado + métricas de la API). */
export interface EstadoCluster {
  epocas: number
  perdida: number
  nodos: EstadoNodo[]
  reasignaciones: number
  error?: string
  api?: MetricasAPI
}

export function esMetricasAPI(v: unknown): v is MetricasAPI {
  const m = v as Record<string, unknown> | null
  return (
    !!m &&
    typeof m === 'object' &&
    ['predicciones', 'ventana', 'no_registradas'].every((k) => entero(m[k])) &&
    ['p50_ms', 'p95_ms', 'max_ms', 'aciertos_cache', 'limite_p95_ms'].every((k) => finito(m[k])) &&
    typeof m.cumple_limite_p95 === 'boolean'
  )
}

export function esEstadoCluster(v: unknown): v is EstadoCluster {
  const c = v as Record<string, unknown> | null
  return (
    !!c &&
    typeof c === 'object' &&
    entero(c.epocas) &&
    finito(c.perdida) &&
    entero(c.reasignaciones) &&
    Array.isArray(c.nodos) &&
    c.nodos.every(esEstadoNodo) &&
    opcional(c.error, texto) &&
    opcional(c.api, esMetricasAPI)
  )
}

/**
 * Siembra los nodos con el estado actual del clúster: cuando el admin abre el
 * panel, los `registro` ya pasaron hace rato y no vuelven a llegar.
 */
export function conCluster(e: EstadoPanel, c: EstadoCluster): EstadoPanel {
  return conInstantanea(e, c.nodos)
}

/** Lo que usa el sello de GET /api/v1/modelo. */
export interface ModeloParaSello {
  version: string
  sesgo: number
  variables: { nombre: string; peso: number }[]
}

export type Sello = { ok: true; comparados: number } | { ok: false; motivo: string; primeraDiferencia?: number }

/**
 * Sello «bit a bit»: los pesos que anunció la API por el WebSocket al publicar
 * el modelo son idénticos (===, sin tolerancia) a los que sirve GET
 * /api/v1/modelo. Que además sean iguales al entrenamiento secuencial lo
 * garantiza el test de Go TestServicioDemoCortarNodoPorLaAPI.
 */
export function selloBitABit(anunciado: { version: string; pesos: number[] }, publicado: ModeloParaSello): Sello {
  if (anunciado.version !== publicado.version) return { ok: false, motivo: `la versión anunciada (${anunciado.version}) no es la publicada (${publicado.version})` }
  const esperados = [...publicado.variables.map((v) => v.peso), publicado.sesgo]
  if (anunciado.pesos.length !== esperados.length) return { ok: false, motivo: `${anunciado.pesos.length} pesos anunciados frente a ${esperados.length} publicados` }
  const i = esperados.findIndex((p, j) => p !== anunciado.pesos[j])
  return i < 0 ? { ok: true, comparados: esperados.length } : { ok: false, motivo: `el peso ${i} difiere`, primeraDiferencia: i }
}

/** Forma mínima de GET /api/v1/modelo que necesita el sello (y los nombres de las barras). */
export function esModeloParaSello(v: unknown): v is ModeloParaSello {
  const m = v as Record<string, unknown> | null
  return (
    !!m &&
    typeof m === 'object' &&
    texto(m.version) &&
    finito(m.sesgo) &&
    Array.isArray(m.variables) &&
    m.variables.every((x) => !!x && typeof x === 'object' && texto((x as Record<string, unknown>).nombre) && finito((x as Record<string, unknown>).peso))
  )
}

const hora = new Intl.DateTimeFormat('es-PE', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })

/**
 * Hora de una línea de la bitácora en la zona del navegador (PR-87 H2). Go
 * serializa con el huso del servidor (-05:00 en Windows, Z en Docker) y la SPA
 * anota en UTC: recortar la cadena mezclaba horas con 5 h de diferencia.
 */
export function horaBitacora(t: string): string {
  const ms = Date.parse(t)
  return Number.isFinite(ms) ? hora.format(ms) : t.slice(11, 19)
}
