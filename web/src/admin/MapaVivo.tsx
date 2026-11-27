import { geoMercator, geoPath } from 'd3-geo'
import { scaleLinear } from 'd3-scale'
import { DEPARTAMENTOS } from '../mapa/geo'
import { etiquetaDepto, formato } from '../mapa/datos'
import { efectosDepartamento } from './mapaVivo'

// Mapa del panel admin (#69): se repinta con los pesos de cada evento `epoca`.
// Solo atributos de SVG (fill, d): la CSP de cmd/web no admite estilos inline.

const A = 300
const H = 392
const proyeccion = geoMercator().fitSize([A, H], DEPARTAMENTOS)
const CAMINOS = DEPARTAMENTOS.features.map((f) => ({ nombre: f.properties.NOMBDEP, d: geoPath(proyeccion)(f) ?? '' }))
const NOMBRES = CAMINOS.map((c) => c.nombre)

// Paleta de vigia.css: --azul baja el riesgo, --papel neutro, --rosa lo sube (igual que las barras de pesos).
const AZUL = '#2b4edb'
const PAPEL = '#f4eedf'
const ROSA = '#e8336d'

export function MapaVivo({ pesos, variables, origen }: { pesos: number[] | null; variables: string[]; origen: string }) {
  const r = efectosDepartamento(pesos, variables, NOMBRES)
  if (!r) {
    return <p className="nota">El mapa aparece cuando hay pesos del modelo (publicado o de la primera época).</p>
  }
  const color = scaleLinear<string>().domain([-r.max, 0, r.max]).range([AZUL, PAPEL, ROSA]).clamp(true)
  const orden = [...r.efectos.entries()].sort((a, b) => b[1].delta - a[1].delta)
  const [mayor, menor] = [orden[0]!, orden.at(-1)!]
  const resumen =
    `Efecto del departamento en el riesgo (${origen}): el mayor es ${etiquetaDepto(mayor[0])} con razón de odds ${formato.decimal(mayor[1].razonOdds)} ` +
    `y el menor ${etiquetaDepto(menor[0])} con ${formato.decimal(menor[1].razonOdds)} frente al departamento promedio`
  return (
    <>
      <svg viewBox={`0 0 ${A} ${H}`} width="100%" role="img" aria-label={resumen} data-grafico="mapa-vivo">
        {CAMINOS.map(({ nombre, d }) => {
          const e = r.efectos.get(nombre)
          return (
            <path
              key={nombre}
              d={d}
              className="depto-vivo"
              fill={e ? color(e.delta) : PAPEL}
              data-depto={nombre}
              data-delta={e?.delta}
              data-agrupado={e?.agrupado || undefined}
            >
              <title>
                {e
                  ? `${etiquetaDepto(nombre)}${e.agrupado ? ' (columna de Otros)' : ''}: razón de odds ${formato.decimal(e.razonOdds)}`
                  : `${etiquetaDepto(nombre)}: sin variable en el modelo`}
              </title>
            </path>
          )
        })}
      </svg>
      <ul className="leyenda leyenda-vivo" aria-label="Leyenda del mapa en vivo">
        <li>
          <span className="leyenda-color leyenda-baja" aria-hidden="true" /> menos riesgo (OR {formato.decimal(Math.exp(-r.max))})
        </li>
        <li>
          <span className="leyenda-color leyenda-neutra" aria-hidden="true" /> promedio (OR 1)
        </li>
        <li>
          <span className="leyenda-color leyenda-sube" aria-hidden="true" /> más riesgo (OR {formato.decimal(Math.exp(r.max))})
        </li>
      </ul>
    </>
  )
}
