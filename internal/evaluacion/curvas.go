// Package evaluacion arma la evaluación experimental de la TB2 (#59): curva ROC,
// calibración, riesgo cruzado departamento × semana × grupo de edad y la lectura
// de los benchmarks de escalabilidad y latencia. Todo es determinista: las
// sumas recorren las filas en orden y los resultados se ordenan por clave.
package evaluacion

import (
	"cmp"
	"errors"
	"math"
	"slices"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// PuntoROC es un punto de la curva ROC. Umbral es la probabilidad calibrada a
// partir de la cual se clasifica como positivo.
type PuntoROC struct {
	FPR    float64 `json:"fpr"`
	TPR    float64 `json:"tpr"`
	Umbral float64 `json:"umbral_calibrado"`
}

// ROC recorre los puntajes de mayor a menor (los empates avanzan juntos) y
// devuelve la curva reducida a lo sumo a maxPuntos puntos y el AUC por la regla
// del trapecio sobre la curva completa. Con empates contados como medio par,
// ese AUC es igual al de Mann-Whitney (ml.AUC).
func ROC(y, p []float64, ajuste float64, maxPuntos int) ([]PuntoROC, float64, error) {
	if len(y) != len(p) {
		return nil, 0, errors.New("evaluacion: y y p tienen largos distintos")
	}
	if maxPuntos < 2 {
		return nil, 0, errors.New("evaluacion: la ROC necesita al menos 2 puntos")
	}
	idx := indicesOrdenados(p, func(a, b float64) int { return cmp.Compare(b, a) })
	var npos, nneg float64
	for _, v := range y {
		if v == 1 {
			npos++
		} else {
			nneg++
		}
	}
	if npos == 0 || nneg == 0 {
		return nil, 0, errors.New("evaluacion: la ROC requiere positivos y negativos")
	}
	curva := []PuntoROC{{0, 0, 1}}
	var vp, fp, auc float64
	for i := 0; i < len(idx); {
		j := i
		for j < len(idx) && p[idx[j]] == p[idx[i]] {
			if y[idx[j]] == 1 {
				vp++
			} else {
				fp++
			}
			j++
		}
		ant := curva[len(curva)-1]
		pt := PuntoROC{fp / nneg, vp / npos, ml.Calibrar(p[idx[i]], ajuste)}
		auc += (pt.FPR - ant.FPR) * (pt.TPR + ant.TPR) / 2
		curva = append(curva, pt)
		i = j
	}
	return reducir(curva, maxPuntos), auc, nil
}

// reducir conserva el primer y el último punto y toma el resto a intervalos
// regulares de índice.
func reducir(c []PuntoROC, max int) []PuntoROC {
	if len(c) <= max {
		return c
	}
	out := make([]PuntoROC, max)
	for k := range max {
		out[k] = c[k*(len(c)-1)/(max-1)]
	}
	return out
}

// Bin es un tramo de la curva de calibración, con el intervalo de Wilson al 95 %
// de la tasa observada.
type Bin struct {
	Casos     int     `json:"casos"`
	Predicho  float64 `json:"riesgo_calibrado_medio"`
	Observado float64 `json:"tasa_observada"`
	IC95Inf   float64 `json:"ic95_inf"`
	IC95Sup   float64 `json:"ic95_sup"`
}

// Calibracion ordena por riesgo calibrado y lo parte en k tramos de igual tamaño
// (el último absorbe el resto). Compara el riesgo medio predicho con la tasa
// observada de cada tramo.
func Calibracion(y, pc []float64, k int) ([]Bin, error) {
	if len(y) != len(pc) || k < 1 || len(y) < k {
		return nil, errors.New("evaluacion: datos insuficientes para la calibración")
	}
	idx := indicesOrdenados(pc, cmp.Compare[float64])
	out := make([]Bin, k)
	for b := range k {
		ini, fin := b*len(idx)/k, (b+1)*len(idx)/k
		var pos, suma float64
		for _, i := range idx[ini:fin] {
			pos += y[i]
			suma += pc[i]
		}
		n := float64(fin - ini)
		inf, sup := Wilson(pos, n)
		out[b] = Bin{fin - ini, suma / n, pos / n, inf, sup}
	}
	return out, nil
}

// Brier es el error cuadrático medio de las probabilidades calibradas.
func Brier(y, pc []float64) float64 {
	var s float64
	for i := range y {
		d := pc[i] - y[i]
		s += d * d
	}
	return s / float64(len(y))
}

// Wilson devuelve el intervalo de confianza al 95 % de una proporción pos/n.
func Wilson(pos, n float64) (inf, sup float64) {
	if n == 0 {
		return 0, 0
	}
	const z = 1.959963984540054
	f := pos / n
	den := 1 + z*z/n
	centro := (f + z*z/(2*n)) / den
	radio := z * math.Sqrt(f*(1-f)/n+z*z/(4*n*n)) / den
	return math.Max(0, centro-radio), math.Min(1, centro+radio)
}

// indicesOrdenados devuelve los índices de v ordenados por comparar, estable
// (a igual valor, conserva el orden de las filas).
func indicesOrdenados(v []float64, comparar func(a, b float64) int) []int {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return comparar(v[a], v[b]) })
	return idx
}
