// Package features convierte los registros limpios (silver) en la matriz numérica
// que entrena la regresión logística (gold).
//
// Variables (todas conocidas al momento de la notificación, sin fuga de datos):
//
//	edad_z                      edad en años estandarizada con la media y desviación del train
//	etapa_<tramo>               etapa de vida MINSA, one-hot (5)
//	sexo_m                      1 si es hombre
//	semana_sin, semana_cos      estacionalidad: la semana 52 queda cerca de la semana 1
//	ano_z                       tendencia anual estandarizada con el train
//	depto_<nombre>              departamento one-hot; los que tienen pocos casos van a OTROS
//
// Se excluyen a propósito: el diagnóstico y la enfermedad (son el objetivo) y
// tipo_dx (confirmado/probable), porque los casos más graves se confirman con
// más frecuencia en laboratorio: usarlo filtraría información del objetivo.
package features

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

// OtrosDepartamentos agrupa los departamentos con pocos casos en el train.
const OtrosDepartamentos = "OTROS"

// Esquema fija cómo se construyen las variables. Se calcula solo con el train
// y se guarda junto al gold: la API de predicción lo necesita para transformar
// una consulta nueva exactamente igual.
type Esquema struct {
	AnoMin        int      `json:"ano_min"`
	MinCasosDepto int      `json:"min_casos_departamento"`
	Departamentos []string `json:"departamentos"`
	MediaEdad     float64  `json:"media_edad"`
	DesvEdad      float64  `json:"desv_edad"`
	MediaAno      float64  `json:"media_ano"`
	DesvAno       float64  `json:"desv_ano"`
	Nombres       []string `json:"nombres"`
	indiceDepto   map[string]int
}

// D es el número de variables (sin contar el sesgo).
func (e *Esquema) D() int { return len(e.Nombres) }

// Ajustar calcula el esquema con los registros de entrenamiento.
func Ajustar(train []ingest.Registro, anoMin, minCasosDepto int) (*Esquema, error) {
	if len(train) < 2 {
		return nil, errors.New("features: se necesitan al menos 2 registros de entrenamiento")
	}
	e := &Esquema{AnoMin: anoMin, MinCasosDepto: minCasosDepto}
	conteo := map[string]int{}
	var sumEdad, sumAno float64
	for _, r := range train {
		conteo[r.Departamento]++
		sumEdad += r.EdadAnios
		sumAno += float64(r.Ano)
	}
	n := float64(len(train))
	e.MediaEdad, e.MediaAno = sumEdad/n, sumAno/n
	var ssEdad, ssAno float64
	for _, r := range train {
		ssEdad += (r.EdadAnios - e.MediaEdad) * (r.EdadAnios - e.MediaEdad)
		ssAno += (float64(r.Ano) - e.MediaAno) * (float64(r.Ano) - e.MediaAno)
	}
	e.DesvEdad = math.Max(math.Sqrt(ssEdad/(n-1)), 1e-12)
	e.DesvAno = math.Max(math.Sqrt(ssAno/(n-1)), 1e-12)
	for d, c := range conteo {
		if c >= minCasosDepto {
			e.Departamentos = append(e.Departamentos, d)
		}
	}
	slices.Sort(e.Departamentos)
	e.Departamentos = append(e.Departamentos, OtrosDepartamentos)
	e.construirNombres()
	return e, nil
}

func (e *Esquema) construirNombres() {
	e.Nombres = []string{"edad_z"}
	for _, t := range ingest.TramosEdad {
		e.Nombres = append(e.Nombres, "etapa_"+t.Nombre)
	}
	e.Nombres = append(e.Nombres, "sexo_m", "semana_sin", "semana_cos", "ano_z")
	e.indiceDepto = map[string]int{}
	for i, d := range e.Departamentos {
		e.indiceDepto[d] = i
		e.Nombres = append(e.Nombres, "depto_"+strings.ReplaceAll(strings.ToLower(d), " ", "_"))
	}
}

// Preparar reconstruye los índices internos tras leer el esquema de un JSON.
func (e *Esquema) Preparar() error {
	if len(e.Departamentos) == 0 || e.Departamentos[len(e.Departamentos)-1] != OtrosDepartamentos {
		return errors.New("features: esquema sin la categoría OTROS al final")
	}
	nombres := e.Nombres
	e.construirNombres()
	if !slices.Equal(nombres, e.Nombres) {
		return fmt.Errorf("features: los nombres del esquema no coinciden con los departamentos")
	}
	return nil
}

// Llenar escribe en x (largo D) las variables del registro.
func (e *Esquema) Llenar(r ingest.Registro, x []float64) {
	clear(x)
	x[0] = (r.EdadAnios - e.MediaEdad) / e.DesvEdad
	x[1+ingest.TramoEdad(r.EdadAnios)] = 1
	j := 1 + len(ingest.TramosEdad)
	if r.Sexo == 'M' {
		x[j] = 1
	}
	angulo := 2 * math.Pi * float64(r.Semana-1) / 52
	x[j+1] = math.Sin(angulo)
	x[j+2] = math.Cos(angulo)
	x[j+3] = (float64(r.Ano) - e.MediaAno) / e.DesvAno
	k, ok := e.indiceDepto[r.Departamento]
	if !ok {
		k = len(e.Departamentos) - 1 // OTROS
	}
	x[j+4+k] = 1
}

// EsTest decide, de forma determinista y sin depender del orden de proceso, si
// la fila con índice global i va al conjunto de prueba. Usa un hash (splitmix64)
// del índice: así el split no cambia con el número de workers ni de nodos.
func EsTest(i int, fracTest float64) bool {
	z := uint64(i) + 0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11)/(1<<53) < fracTest
}
