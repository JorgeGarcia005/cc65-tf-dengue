// Package estadistica resume tiempos de ejecución repetidos con estimadores
// robustos a valores atípicos (media recortada) y su intervalo de confianza.
package estadistica

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
)

// Resumen describe una muestra de mediciones.
type Resumen struct {
	N              int     `json:"n"`
	MediaRecortada float64 `json:"media_recortada"`
	Recorte        float64 `json:"recorte"`
	Mediana        float64 `json:"mediana"`
	Desviacion     float64 `json:"desviacion"`
	Min            float64 `json:"min"`
	Max            float64 `json:"max"`
	IC95Inf        float64 `json:"ic95_inf"`
	IC95Sup        float64 `json:"ic95_sup"`
}

// MediaRecortada descarta ⌊n·frac⌋ valores en cada extremo y promedia el resto.
// frac debe estar en [0, 0.5); fuera de ese rango entra en pánico con un mensaje
// claro (es un error de programación, igual que una Config inválida en ml).
func MediaRecortada(xs []float64, frac float64) float64 {
	ValidarRecorte(frac)
	if len(xs) == 0 {
		return math.NaN()
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	k := int(float64(len(s)) * frac)
	s = s[k : len(s)-k]
	var suma float64
	for _, v := range s {
		suma += v
	}
	return suma / float64(len(s))
}

// ValidarRecorte entra en pánico si frac no está en [0, 0.5). Con frac < 0.5
// siempre queda al menos un valor tras recortar: 2·⌊n·frac⌋ < n.
func ValidarRecorte(frac float64) {
	if !(frac >= 0 && frac < 0.5) {
		panic(fmt.Sprintf("estadistica: recorte = %v (debe estar en [0, 0.5))", frac))
	}
}

// Mediana de la muestra.
func Mediana(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

// Desviacion es la desviación estándar muestral (n−1).
func Desviacion(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var media float64
	for _, v := range xs {
		media += v
	}
	media /= float64(len(xs))
	var ss float64
	for _, v := range xs {
		ss += (v - media) * (v - media)
	}
	return math.Sqrt(ss / float64(len(xs)-1))
}

// ICBootstrap estima el intervalo de confianza al 95 % de la media recortada
// con remuestreo bootstrap (percentiles 2,5 y 97,5). Es determinista por semilla.
func ICBootstrap(xs []float64, frac float64, remuestras int, semilla uint64) (inf, sup float64) {
	if len(xs) == 0 || remuestras <= 0 {
		return math.NaN(), math.NaN()
	}
	r := rand.New(rand.NewPCG(semilla, 0x51ab))
	estimaciones := make([]float64, remuestras)
	muestra := make([]float64, len(xs))
	for b := range estimaciones {
		for i := range muestra {
			muestra[i] = xs[r.IntN(len(xs))]
		}
		estimaciones[b] = MediaRecortada(muestra, frac)
	}
	slices.Sort(estimaciones)
	return percentil(estimaciones, 0.025), percentil(estimaciones, 0.975)
}

// percentil sobre un slice ya ordenado, con interpolación lineal.
func percentil(ordenado []float64, q float64) float64 {
	pos := q * float64(len(ordenado)-1)
	i := int(pos)
	if i >= len(ordenado)-1 {
		return ordenado[len(ordenado)-1]
	}
	f := pos - float64(i)
	return ordenado[i]*(1-f) + ordenado[i+1]*f
}

// Resumir calcula todos los estadísticos de una muestra.
func Resumir(xs []float64, frac float64) Resumen {
	r := Resumen{N: len(xs), Recorte: frac}
	if len(xs) == 0 {
		return r
	}
	r.MediaRecortada = MediaRecortada(xs, frac)
	r.Mediana = Mediana(xs)
	r.Desviacion = Desviacion(xs)
	r.Min = slices.Min(xs)
	r.Max = slices.Max(xs)
	r.IC95Inf, r.IC95Sup = ICBootstrap(xs, frac, 2000, 1)
	return r
}
