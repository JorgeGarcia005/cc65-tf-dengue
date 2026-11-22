import { geoMercator, geoPath } from 'd3-geo'
import { useMemo, useState, type KeyboardEvent } from 'react'
import { OTROS } from '../api/tipos'
import { DEPARTAMENTOS } from './geo'
import { COLORES as escalaColores, escalaPara, etiquetaDepto, formato, valor, type IndiceRiesgo, type Metrica } from './datos'

const ANCHO = 520
const ALTO = 680

interface Props {
  indice: IndiceRiesgo
  semana: number
  etapa: string
  metrica: Metrica
  seleccionado: string | null
  onSeleccionar(depto: string): void
}

/** Coroplético por departamento (SVG). Cada departamento es un botón accesible con teclado. */
export default function MapaPeru({ indice, semana, etapa, metrica, seleccionado, onSeleccionar }: Props) {
  const camino = useMemo(() => geoPath(geoMercator().fitSize([ANCHO, ALTO], DEPARTAMENTOS)), [])
  const escala = useMemo(() => escalaPara(indice, semana, etapa, metrica), [indice, semana, etapa, metrica])
  const [sobre, setSobre] = useState<string | null>(null)

  const deptos = DEPARTAMENTOS.features.map((f) => {
    const nombre = f.properties.NOMBDEP
    const grupo = indice.grupo(nombre)
    const celda = indice.celda(grupo, semana, etapa)
    const v = valor(celda, metrica)
    const texto =
      celda === undefined
        ? `${etiquetaDepto(nombre)}: sin casos con este filtro`
        : `${etiquetaDepto(nombre)}${grupo === OTROS ? ' (agrupado en Otros)' : ''}: ` +
          `${metrica === 'riesgo' ? 'riesgo' : 'casos esperados'} ${formato.metrica(v!, metrica)}, ` +
          `${formato.entero(celda.casos)} casos${celda.insuficiente ? ', datos insuficientes' : ''}`
    return { f, nombre, grupo, celda, v, texto, d: camino(f) ?? '' }
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
            className={'depto' + (x.nombre === seleccionado ? ' depto-sel' : '') + (x.grupo === OTROS ? ' depto-otros' : '')}
            fill={x.celda === undefined || x.celda.insuficiente ? 'url(#rayado)' : escala.color(x.v!)}
            role="button"
            tabIndex={0}
            aria-label={x.texto}
            aria-pressed={x.nombre === seleccionado}
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
      <Leyenda cortes={escala.cortes} metrica={metrica} />
    </figure>
  )
}

function Leyenda({ cortes, metrica }: { cortes: number[]; metrica: Metrica }) {
  const limites = ['', ...cortes.map((c) => formato.metrica(c, metrica))]
  return (
    <ul className="leyenda" aria-label="Leyenda: quintiles entre los departamentos">
      {escalaColores.map((color, i) => (
        <li key={color}>
          <span className="leyenda-color" style={{ background: color }} aria-hidden="true" />
          {i === 0 ? `< ${limites[1]}` : i === escalaColores.length - 1 ? `≥ ${limites[i]}` : `${limites[i]} – ${limites[i + 1]}`}
        </li>
      ))}
      <li>
        <span className="leyenda-color leyenda-rayado" aria-hidden="true" />
        Datos insuficientes o sin casos
      </li>
    </ul>
  )
}

