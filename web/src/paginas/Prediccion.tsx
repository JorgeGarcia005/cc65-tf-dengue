import { useEffect, useId, useMemo, useState, type FormEvent } from 'react'
import { ErrorApi } from '../api/cliente'
import { OTROS, type ModeloPublicado, type Prediccion as Resultado } from '../api/tipos'
import { etiquetaDepto, formato } from '../mapa/datos'
import { DEPARTAMENTOS } from '../mapa/geo'
import { contribucionesActivas, etiquetaVariable, sumaLogit, validar, type Errores, type Formulario } from '../prediccion/datos'
import { useSesion } from '../sesion/Sesion'

const NOMBRES = DEPARTAMENTOS.features.map((f) => f.properties.NOMBDEP).sort((a, b) => a.localeCompare(b, 'es'))

interface Respuesta {
  pred: Resultado
  ms: number
  cache: string | null
  pedido: string
}

// Predicción de un caso (#67): mismo cálculo que la API (POST /api/v1/predicciones),
// con la contribución de cada variable al logit, la latencia medida en el
// navegador y si la respuesta salió de Redis.
export default function Prediccion() {
  const { api } = useSesion()
  const [modelo, setModelo] = useState<ModeloPublicado | null>(null)
  const [form, setForm] = useState<Formulario>({ departamento: '', ano: String(new Date().getFullYear()), semana: '', edad: '', tipo_edad: 'A', sexo: '' })
  const [errores, setErrores] = useState<Errores>({})
  const [errorApi, setErrorApi] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)
  const [resp, setResp] = useState<Respuesta | null>(null)
  const id = useId()

  useEffect(() => {
    let vivo = true
    api
      .get<ModeloPublicado>('/api/v1/modelo')
      .then((m) => vivo && setModelo(m))
      .catch((e: unknown) => vivo && setErrorApi(e instanceof ErrorApi ? e.message : String(e)))
    return () => {
      vivo = false
    }
  }, [api])

  const anoMin = modelo?.ano_min ?? 2000
  const cambiar = (campo: keyof Formulario) => (e: { target: { value: string } }) => setForm((f) => ({ ...f, [campo]: e.target.value }))

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const { consulta, errores: errs } = validar(form, anoMin)
    setErrores(errs)
    setErrorApi(null)
    if (!consulta) return
    setEnviando(true)
    try {
      const r = await api.postMedido<Resultado>('/api/v1/predicciones', consulta)
      setResp({ pred: r.datos, ms: r.ms, cache: r.cabeceras.get('X-Cache'), pedido: consulta.departamento })
    } catch (err) {
      setResp(null)
      setErrorApi(err instanceof ErrorApi ? err.message : String(err))
    } finally {
      setEnviando(false)
    }
  }

  const campo = (nombre: keyof Formulario) => ({
    id: `${id}-${nombre}`,
    'aria-invalid': errores[nombre] ? true : undefined,
    'aria-describedby': errores[nombre] ? `${id}-${nombre}-error` : undefined,
  })
  const mensaje = (nombre: keyof Formulario) =>
    errores[nombre] ? (
      <span id={`${id}-${nombre}-error`} className="error-campo">
        {errores[nombre]}
      </span>
    ) : null

  return (
    <div className="vista-prediccion">
      <header className="vista-cabecera">
        <p className="antetitulo">Un caso notificado</p>
        <h1 id="titulo-vista" className="titulo">
          Predicción de un caso
        </h1>
        <p className="bajada">
          Riesgo calibrado de que un caso de dengue notificado sea con signos de alarma o grave, con las variables demográficas que usa el modelo
          {modelo ? ` (versión ${modelo.version})` : ''}.
        </p>
      </header>

      <div className="prediccion-columnas">
        <form className="tarjeta formulario" onSubmit={enviar} noValidate aria-labelledby="titulo-vista">
          <label htmlFor={`${id}-departamento`}>Departamento</label>
          <select {...campo('departamento')} value={form.departamento} onChange={cambiar('departamento')}>
            <option value="">Elige…</option>
            {NOMBRES.map((d) => (
              <option key={d} value={d}>
                {etiquetaDepto(d)}
              </option>
            ))}
          </select>
          {mensaje('departamento')}

          <label htmlFor={`${id}-ano`}>Año</label>
          <input {...campo('ano')} inputMode="numeric" value={form.ano} onChange={cambiar('ano')} />
          {mensaje('ano')}

          <label htmlFor={`${id}-semana`}>Semana epidemiológica (1–53)</label>
          <input {...campo('semana')} inputMode="numeric" value={form.semana} onChange={cambiar('semana')} />
          {mensaje('semana')}

          <div className="formulario-fila">
            <div>
              <label htmlFor={`${id}-edad`}>Edad</label>
              <input {...campo('edad')} inputMode="numeric" value={form.edad} onChange={cambiar('edad')} />
            </div>
            <div>
              <label htmlFor={`${id}-tipo_edad`}>En</label>
              <select {...campo('tipo_edad')} value={form.tipo_edad} onChange={cambiar('tipo_edad')}>
                <option value="A">años</option>
                <option value="M">meses</option>
                <option value="D">días</option>
              </select>
            </div>
          </div>
          {mensaje('edad')}
          {mensaje('tipo_edad')}

          <fieldset aria-invalid={errores.sexo ? true : undefined} aria-describedby={errores.sexo ? `${id}-sexo-error` : undefined}>
            <legend>Sexo</legend>
            {(['F', 'M'] as const).map((s) => (
              <label key={s} className="opcion">
                <input type="radio" name={`${id}-sexo`} value={s} checked={form.sexo === s} onChange={cambiar('sexo')} />
                {s === 'F' ? 'Femenino' : 'Masculino'}
              </label>
            ))}
          </fieldset>
          {mensaje('sexo')}

          <button type="submit" className="boton" disabled={enviando}>
            {enviando ? 'Calculando…' : 'Calcular riesgo'}
          </button>
          {errorApi && (
            <p className="error" role="alert">
              {errorApi}
            </p>
          )}
        </form>

        {resp && <ResultadoCaso resp={resp} departamentos={modelo?.departamentos ?? []} />}
      </div>
    </div>
  )
}

function ResultadoCaso({ resp, departamentos }: { resp: Respuesta; departamentos: string[] }) {
  const { pred, ms, cache, pedido } = resp
  const activas = useMemo(() => contribucionesActivas(pred.contribuciones), [pred])
  const mayor = Math.max(...activas.map((c) => Math.abs(c.aporte)), Math.abs(pred.sesgo), Number.MIN_VALUE)
  const cuadra = sumaLogit(pred) === pred.logit
  const textoCache = cache === 'HIT' ? 'desde la caché (Redis)' : cache === 'MISS' ? 'calculada y guardada en la caché' : 'sin caché'

  return (
    <section className="tarjeta resultado" aria-labelledby="titulo-resultado" aria-live="polite">
      <h2 id="titulo-resultado" className="titulo-panel">
        Riesgo calibrado
      </h2>
      <p className="riesgo-grande" data-resultado="riesgo">
        {formato.porcentaje(pred.riesgo)}
      </p>
      <p className={`etiqueta-clase clase-${pred.clase}`} data-resultado="clase">
        {pred.etiqueta}
      </p>
      <p className="nota-clase">
        La clase compara la probabilidad del modelo, entrenado con pesos por clase, con el umbral: marca a quién vigilar. El riesgo calibrado lleva esa
        probabilidad a la escala de la tasa real de casos con signos de alarma.
      </p>
      <dl className="cifras">
        <div>
          <dt>Probabilidad del modelo</dt>
          <dd>{formato.decimal(pred.probabilidad_ponderada)}</dd>
        </div>
        <div>
          <dt>Umbral</dt>
          <dd>{formato.decimal(pred.umbral)}</dd>
        </div>
        <div>
          <dt>Latencia</dt>
          <dd data-resultado="latencia">{formato.numero(ms)} ms</dd>
        </div>
        <div>
          <dt>Respuesta</dt>
          <dd data-resultado="cache">{textoCache}</dd>
        </div>
      </dl>
      {pred.departamento_modelo === OTROS && pedido !== OTROS && (
        <p className="aviso-insuficiente">
          {etiquetaDepto(pedido)} no tiene columna propia en el modelo (pocos casos en el entrenamiento): se usa la de «Otros».
        </p>
      )}

      <h3 className="subtitulo">Qué empuja el riesgo</h3>
      <p className="bajada">
        Aporte de cada variable al logit (peso × valor). Hacia la derecha sube el riesgo; hacia la izquierda, baja.
      </p>
      <ol className="contribuciones" aria-label="Contribuciones al logit">
        <li className="contribucion contribucion-sesgo">
          <span className="contribucion-nombre">Sesgo (intercepto)</span>
          <Barra aporte={pred.sesgo} mayor={mayor} />
          <span className="contribucion-valor">{signo(pred.sesgo)}</span>
        </li>
        {activas.map((c) => (
          <li key={c.variable} className="contribucion" data-variable={c.variable}>
            <span className="contribucion-nombre" title={`valor ${formato.decimal(c.valor)} × peso ${formato.decimal(c.peso)}`}>
              {etiquetaVariable(c.variable, departamentos)}
            </span>
            <Barra aporte={c.aporte} mayor={mayor} />
            <span className="contribucion-valor">{signo(c.aporte)}</span>
          </li>
        ))}
      </ol>
      <p className="suma-logit" data-resultado="suma">
        sesgo + Σ aportes = logit {signo(pred.logit)} → σ(logit) = {formato.decimal(pred.probabilidad_ponderada)}{' '}
        <span className={cuadra ? 'cuadra' : 'no-cuadra'}>{cuadra ? '✓ cuadra' : '✗ no cuadra'}</span>
      </p>
      <p className="nota">
        Modelo {pred.version_modelo} · {ceros(pred.contribuciones.length - activas.length)}
      </p>
    </section>
  )
}

function Barra({ aporte, mayor }: { aporte: number; mayor: number }) {
  const ancho = (50 * Math.abs(aporte)) / mayor
  return (
    <span className="contribucion-eje" aria-hidden="true">
      <span
        className={aporte >= 0 ? 'contribucion-barra sube' : 'contribucion-barra baja'}
        style={aporte >= 0 ? { left: '50%', width: `${ancho}%` } : { right: '50%', width: `${ancho}%` }}
      />
    </span>
  )
}

const signo = (v: number) => (v >= 0 ? '+' : '−') + formato.decimal(Math.abs(v))

const ceros = (n: number) => (n === 1 ? '1 variable en 0 (otra categoría) no aporta' : `${n} variables en 0 (otras categorías) no aportan`)
