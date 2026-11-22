import { geoMercator, geoPath } from 'd3-geo'
import { useMemo, useState, type KeyboardEvent } from 'react'
import { OTROS } from '../api/tipos'
import { DEPARTAMENTOS } from './geo'
import { escalaPara, etiquetaDepto, formato, valor, type Escala, type IndiceRiesgo, type Metrica } from './datos'

const ANCHO = 520
const ALTO = 680

// La proyección no depende de los datos: se calcula una sola vez por carga del módulo.
const proyeccion = geoMercator().fitSize([ANCHO, ALTO], DEPARTAMENTOS)
const CAMINOS = DEPARTAMENTOS.features.map((f) => ({ nombre: f.properties.NOMBDEP, d: geoPath(proyeccion)(f) ?? '' }))

interface Props {
  indice: IndiceRiesgo
  semana: number
  etapa: string
  metrica: Metrica
  /** Departamento elegido, o OTROS (resalta todos los agrupados). */
  seleccionado: string | null
  onSeleccionar(depto: string): void
}

/** Coroplético por departamento (SVG). Cada departamento es un botón accesible con teclado. */
export default function MapaPeru({ indice, semana, etapa, metrica, seleccionado, onSeleccionar }: Props) {
  const escala = useMemo(() => escalaPara(indice, semana, etapa, metrica), [indice, semana, etapa, metrica])
  const [sobre, setSobre] = useState<string | null>(null)

  const deptos = CAMINOS.map(({ nombre, d }) => {
    const grupo = indice.grupo(nombre)
    const celda = indice.celda(grupo, semana, etapa)
    const v = valor(celda, metrica)
    const texto =
      celda === undefined
        ? `${etiquetaDepto(nombre)}: sin casos con este filtro`
        : `${etiquetaDepto(nombre)}${grupo === OTROS ? ' (agrupado en Otros)' : ''}: ` +
          `${metrica === 'riesgo' ? 'riesgo' : 'casos esperados'} ${formato.metrica(v!, metrica)}, ` +
          `${formato.entero(celda.casos)} casos${celda.insuficiente ? ', datos insuficientes' : ''}`
    const elegido = nombre === seleccionado || (seleccionado === OTROS && grupo === OTROS)
    return { nombre, grupo, celda, v, texto, d, elegido }
  })

  const activo = deptos.find((x) => x.nombre === (sobre ?? seleccionado))

  function tecla(e: KeyboardEvent, nombre: string) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onSeleccionar(nombre)
    }
  }

  return (
    <figure className="mapa">
      <svg viewBox={`0 0 ${ANCHO} ${ALTO}`} role="group" aria-label="Mapa de riesgo por departamento">
        <defs>
          <pattern id="rayado" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
            <rect width="6" height="6" fill="#f4eedf" />
            <line x1="0" y1="0" x2="0" y2="6" stroke="#111" strokeWidth="1.4" />
          </pattern>
        </defs>
        {deptos.map((x) => (
          <path
            key={x.nombre}
            d={x.d}
            className={'depto' + (x.elegido ? ' depto-sel' : '') + (x.grupo === OTROS ? ' depto-otros' : '')}
            fill={x.celda === undefined || x.celda.insuficiente || escala.vacia ? 'url(#rayado)' : escala.color(x.v!)}
            role="button"
            tabIndex={0}
            aria-label={x.texto}
            aria-pressed={x.elegido}
            data-depto={x.nombre}
            onClick={() => onSeleccionar(x.nombre)}
            onKeyDown={(e) => tecla(e, x.nombre)}
            onMouseEnter={() => setSobre(x.nombre)}
            onMouseLeave={() => setSobre(null)}
            onFocus={() => setSobre(x.nombre)}
            onBlur={() => setSobre(null)}
          />
        ))}
      </svg>
      <figcaption className="mapa-info" aria-live="polite">
        {activo ? activo.texto : 'Pasa el cursor o usa Tab sobre un departamento.'}
      </figcaption>
      <Leyenda escala={escala} metrica={metrica} />
    </figure>
  )
}

function Leyenda({ escala, metrica }: { escala: Escala; metrica: Metrica }) {
  const rango = (min: number, max: number) => {
    const a = formato.metrica(min, metrica)
    const b = formato.metrica(max, metrica)
    return a === b ? a : `${a} – ${b}`
  }
  return (
    <ul className="leyenda" aria-label="Leyenda: rango de valores de los departamentos en cada color">
      {escala.vacia && <li>Ningún departamento tiene datos suficientes con este filtro.</li>}
      {escala.clases.map((c) => (
        <li key={c.color}>
          <span className="leyenda-color" style={{ background: c.color }} aria-hidden="true" />
          {rango(c.min, c.max)}
        </li>
      ))}
      <li>
        <span className="leyenda-color leyenda-rayado" aria-hidden="true" />
        Datos insuficientes o sin casos
      </li>
    </ul>
  )
}
