package evaluacion

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// Niveles «todos» de cada dimensión del cubo de riesgo.
const (
	TodosDepartamentos = "TODOS"
	TodasEtapas        = "todas"
	TodasSemanas       = 0
)

// Celda es una combinación departamento × semana × grupo de edad del conjunto de
// prueba. CasosEsperados es la suma del riesgo calibrado: cuántos casos con
// signos de alarma predice el modelo entre esos casos notificados. Es lo que
// llena camas, así que es la medida con la que se prioriza.
type Celda struct {
	Departamento    string  `json:"departamento"`
	Semana          int     `json:"semana"`
	Etapa           string  `json:"etapa"`
	Casos           int     `json:"casos"`
	Positivos       int     `json:"con_signos_observados"`
	TasaObservada   float64 `json:"tasa_observada"`
	RiesgoCalibrado float64 `json:"riesgo_calibrado_medio"`
	CasosEsperados  float64 `json:"casos_esperados"`
	Fraccion        float64 `json:"fraccion_nacional"`
	RazonNacional   float64 `json:"razon_vs_nacional"`
	Insuficiente    bool    `json:"insuficiente"`
}

// Cubo es el riesgo cruzado con sus niveles «todos». Las celdas sin casos se omiten.
type Cubo struct {
	MinCasos      int      `json:"min_casos"`
	Departamentos []string `json:"departamentos"`
	Etapas        []string `json:"etapas"`
	Celdas        []Celda  `json:"celdas"`
}

// Grupos recupera de una fila de variables el índice de departamento, la semana
// epidemiológica (1–52) y el índice del grupo de edad.
type Grupos struct {
	jDepto, jEtapa, jSin, jCos, nDepto, nEtapa int
}

// NuevosGrupos ubica las columnas en el esquema.
func NuevosGrupos(esq *features.Esquema) (Grupos, error) {
	if esq == nil {
		return Grupos{}, errors.New("evaluacion: esquema nulo")
	}
	g := Grupos{jDepto: -1, jEtapa: esq.Indice("etapa_" + ingest.TramosEdad[0].Nombre),
		jSin: esq.Indice("semana_sin"), jCos: esq.Indice("semana_cos"),
		nDepto: len(esq.Departamentos), nEtapa: len(ingest.TramosEdad)}
	if len(esq.Departamentos) > 0 {
		g.jDepto = esq.Indice("depto_" + strings.ReplaceAll(strings.ToLower(esq.Departamentos[0]), " ", "_"))
	}
	if g.jDepto < 0 || g.jEtapa < 0 || g.jSin < 0 || g.jCos < 0 ||
		g.jDepto+g.nDepto > esq.D() || g.jEtapa+g.nEtapa > esq.D() {
		return Grupos{}, errors.New("evaluacion: el esquema no tiene las columnas de departamento, etapa o semana")
	}
	return g, nil
}

// De devuelve (departamento, semana, etapa) de la fila x.
func (g Grupos) De(x []float64) (depto, semana, etapa int, err error) {
	depto, etapa = unoCaliente(x[g.jDepto:g.jDepto+g.nDepto]), unoCaliente(x[g.jEtapa:g.jEtapa+g.nEtapa])
	if depto < 0 || etapa < 0 {
		return 0, 0, 0, errors.New("evaluacion: fila sin departamento o etapa one-hot")
	}
	return depto, features.SemanaDeAngulo(x[g.jSin], x[g.jCos]), etapa, nil
}

func unoCaliente(v []float64) int {
	k := -1
	for i, x := range v {
		if x == 1 {
			if k >= 0 {
				return -1
			}
			k = i
		}
	}
	return k
}

type acumulado struct {
	casos, pos int
	suma       float64
}

// Cruzar arma el cubo departamento × semana × etapa (más sus niveles «todos»)
// con las etiquetas y y el riesgo calibrado pc de cada fila de ds.
func Cruzar(ds *ml.Dataset, y, pc []float64, esq *features.Esquema, minCasos int) (Cubo, error) {
	if ds == nil || esq == nil || ds.D != esq.D() || len(y) != ds.N || len(pc) != ds.N {
		return Cubo{}, errors.New("evaluacion: dataset, etiquetas, riesgos y esquema no coinciden en tamaño")
	}
	g, err := NuevosGrupos(esq)
	if err != nil {
		return Cubo{}, err
	}
	nD, nS, nE := g.nDepto+1, 53, g.nEtapa+1 // +1: nivel «todos»
	acc := make([]acumulado, nD*nS*nE)
	at := func(d, s, e int) *acumulado { return &acc[(d*nS+s)*nE+e] }
	for i := 0; i < ds.N; i++ {
		d, s, e, err := g.De(ds.Fila(i))
		if err != nil {
			return Cubo{}, fmt.Errorf("fila %d: %w", i, err)
		}
		for _, dd := range [2]int{d, g.nDepto} {
			for _, ss := range [2]int{s, TodasSemanas} {
				for _, ee := range [2]int{e, g.nEtapa} {
					a := at(dd, ss, ee)
					a.casos++
					a.pos += int(y[i])
					a.suma += pc[i]
				}
			}
		}
	}
	cubo := Cubo{MinCasos: minCasos, Departamentos: append(slices.Clone(esq.Departamentos), TodosDepartamentos)}
	for _, t := range ingest.TramosEdad {
		cubo.Etapas = append(cubo.Etapas, t.Nombre)
	}
	cubo.Etapas = append(cubo.Etapas, TodasEtapas)
	nac := at(g.nDepto, TodasSemanas, g.nEtapa)
	if nac.casos == 0 {
		return Cubo{}, errors.New("evaluacion: conjunto de prueba vacío")
	}
	riesgoNac := nac.suma / float64(nac.casos)
	proporcion := func(a, b float64) float64 { // sin NaN si el riesgo nacional es 0
		if b == 0 {
			return 0
		}
		return a / b
	}
	for d := range nD {
		for s := range nS {
			for e := range nE {
				a := at(d, s, e)
				if a.casos == 0 {
					continue
				}
				n := float64(a.casos)
				cubo.Celdas = append(cubo.Celdas, Celda{
					Departamento: cubo.Departamentos[d], Semana: s, Etapa: cubo.Etapas[e],
					Casos: a.casos, Positivos: a.pos, TasaObservada: float64(a.pos) / n,
					RiesgoCalibrado: a.suma / n, CasosEsperados: a.suma, Fraccion: proporcion(a.suma, nac.suma),
					RazonNacional: proporcion(a.suma/n, riesgoNac), Insuficiente: a.casos < minCasos,
				})
			}
		}
	}
	return cubo, nil
}

// Prioridades devuelve las n combinaciones concretas (un departamento, una
// semana y un grupo de edad, sin OTROS ni niveles «todos») con más casos
// esperados y datos suficientes. Empates: por departamento, semana y etapa.
func Prioridades(c Cubo, n int) []Celda {
	out := make([]Celda, 0) // nunca nil: se serializa como [] y no como null
	if n <= 0 {
		return out
	}
	for _, x := range c.Celdas {
		if !x.Insuficiente && x.Semana != TodasSemanas && x.Etapa != TodasEtapas &&
			x.Departamento != TodosDepartamentos && x.Departamento != features.OtrosDepartamentos {
			out = append(out, x)
		}
	}
	slices.SortStableFunc(out, func(a, b Celda) int {
		return cmp.Or(cmp.Compare(b.CasosEsperados, a.CasosEsperados),
			cmp.Compare(a.Departamento, b.Departamento), cmp.Compare(a.Semana, b.Semana),
			cmp.Compare(a.Etapa, b.Etapa))
	})
	return out[:min(n, len(out))]
}
