import { describe, expect, it } from 'vitest'
import { efectosDepartamento, pesosDelModelo, varDepto } from '../admin/mapaVivo'

// Modelo mínimo: edad, dos departamentos con columna propia, OTROS y el sesgo al final.
const VARS = ['edad_z', 'depto_lima', 'depto_madre_de_dios', 'depto_otros']
const MAPA = ['LIMA', 'MADRE DE DIOS', 'TACNA']

describe('mapa en vivo (#69): efecto de cada departamento en el logit', () => {
  it('el nombre del mapa se vuelve el de la variable del modelo', () => {
    expect(varDepto('MADRE DE DIOS')).toBe('depto_madre_de_dios')
    expect(varDepto('LIMA')).toBe('depto_lima')
  })

  it('centra en el promedio de los departamentos y da la razón de odds', () => {
    // depto: 0.3, -0.1, -0.2 → media 0 → Δ = 0.3, -0.1, -0.2
    const r = efectosDepartamento([9, 0.3, -0.1, -0.2, 5], VARS, MAPA)!
    expect(r).not.toBeNull()
    expect(r.efectos.get('LIMA')).toMatchObject({ variable: 'depto_lima', agrupado: false })
    expect(r.efectos.get('LIMA')!.delta).toBeCloseTo(0.3, 12)
    expect(r.efectos.get('LIMA')!.razonOdds).toBeCloseTo(Math.exp(0.3), 12)
    expect(r.efectos.get('MADRE DE DIOS')!.delta).toBeCloseTo(-0.1, 12)
    expect(r.max).toBeCloseTo(0.3, 12)
  })

  it('un departamento sin columna propia usa la de OTROS y queda marcado', () => {
    const r = efectosDepartamento([0, 0.3, -0.1, -0.2, 0], VARS, MAPA)!
    expect(r.efectos.get('TACNA')).toMatchObject({ variable: 'depto_otros', agrupado: true })
    expect(r.efectos.get('TACNA')!.delta).toBeCloseTo(-0.2, 12)
  })

  it('no depende del sesgo ni de un desplazamiento común de los departamentos', () => {
    const a = efectosDepartamento([0, 0.3, -0.1, -0.2, 0], VARS, MAPA)!
    const b = efectosDepartamento([0, 1.3, 0.9, 0.8, -7], VARS, MAPA)!
    for (const d of MAPA) expect(b.efectos.get(d)!.delta).toBeCloseTo(a.efectos.get(d)!.delta, 12)
  })

  it('con todos los pesos en cero (época 0) no hay NaN y la escala no colapsa', () => {
    const r = efectosDepartamento([0, 0, 0, 0, 0], VARS, MAPA)!
    for (const d of MAPA) {
      expect(r.efectos.get(d)!.delta).toBe(0)
      expect(r.efectos.get(d)!.razonOdds).toBe(1)
    }
    expect(r.max).toBeGreaterThan(0)
  })

  it('sin pesos, con largos que no cuadran o sin variables de departamento devuelve null', () => {
    expect(efectosDepartamento(null, VARS, MAPA)).toBeNull()
    expect(efectosDepartamento([0, 1, 2], VARS, MAPA)).toBeNull()
    expect(efectosDepartamento([1, 2], ['edad_z'], MAPA)).toBeNull()
    expect(efectosDepartamento([0, 0.3, Number.NaN, -0.2, 0], VARS, MAPA)).toBeNull()
  })

  it('pesosDelModelo arma variables + sesgo desde GET /api/v1/modelo', () => {
    expect(pesosDelModelo({ sesgo: 5, variables: [{ nombre: 'a', peso: 1 }, { nombre: 'b', peso: 2 }] })).toEqual([1, 2, 5])
    expect(pesosDelModelo(null)).toBeNull()
  })
})
