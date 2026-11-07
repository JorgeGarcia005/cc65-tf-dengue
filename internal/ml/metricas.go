package ml

import (
	"errors"
	"fmt"
	"slices"
)

// Metricas resume el desempeño de un clasificador binario.
type Metricas struct {
	Umbral                float64
	VP, FP, VN, FN        int
	Exactitud             float64
	Precision, Recall, F1 float64
	AUC                   float64
}

// AUC es el área bajo la curva ROC, calculada como la probabilidad de que un
// positivo al azar tenga mayor puntaje que un negativo al azar (Mann-Whitney).
// Los empates cuentan medio par.
func AUC(y, p []float64) (float64, error) {
	if len(y) != len(p) {
		return 0, errors.New("ml: y y p tienen largos distintos")
	}
	idx := make([]int, len(p))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int {
		switch {
		case p[a] < p[b]:
			return -1
		case p[a] > p[b]:
			return 1
		}
		return 0
	})
	// Rangos promedio (1-indexados) para grupos empatados.
	var sumaRangosPos, npos float64
	for i := 0; i < len(idx); {
		j := i
		for j < len(idx) && p[idx[j]] == p[idx[i]] {
			j++
		}
		rango := float64(i+j+1) / 2 // promedio de i+1 … j
		for k := i; k < j; k++ {
			if y[idx[k]] == 1 {
				sumaRangosPos += rango
				npos++
			}
		}
		i = j
	}
	nneg := float64(len(y)) - npos
	if npos == 0 || nneg == 0 {
		return 0, errors.New("ml: el AUC requiere positivos y negativos")
	}
	return (sumaRangosPos - npos*(npos+1)/2) / (npos * nneg), nil
}

// Evaluar calcula la matriz de confusión y las métricas para un umbral.
// Si falta alguna clase, AUC queda en 0.
func Evaluar(y, p []float64, umbral float64) (Metricas, error) {
	m := Metricas{Umbral: umbral}
	if len(y) != len(p) {
		return m, fmt.Errorf("ml: Evaluar con %d etiquetas y %d probabilidades", len(y), len(p))
	}
	for i := range y {
		pred := p[i] >= umbral
		switch {
		case pred && y[i] == 1:
			m.VP++
		case pred:
			m.FP++
		case y[i] == 1:
			m.FN++
		default:
			m.VN++
		}
	}
	div := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(a) / float64(b)
	}
	m.Exactitud = div(m.VP+m.VN, len(y))
	m.Precision = div(m.VP, m.VP+m.FP)
	m.Recall = div(m.VP, m.VP+m.FN)
	if m.Precision+m.Recall > 0 {
		m.F1 = 2 * m.Precision * m.Recall / (m.Precision + m.Recall)
	}
	m.AUC, _ = AUC(y, p)
	return m, nil
}
