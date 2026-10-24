package ml

import "math/rand/v2"

// Sintetico genera n filas con d features normales estándar y etiquetas
// muestreadas de una regresión logística con pesos verdaderos conocidos.
// Es determinista (misma semilla, mismos datos). Se usa en tests y benchmarks
// para medir el cómputo sin depender del dataset real; los resultados del
// informe sobre el modelo usan siempre el gold de dengue.
func Sintetico(n, d int, semilla uint64) (*Dataset, []float64) {
	r := rand.New(rand.NewPCG(semilla, semilla^0x9e3779b97f4a7c15))
	verdaderos := make([]float64, d+1)
	for j := range verdaderos {
		verdaderos[j] = r.NormFloat64()
	}
	x := make([]float64, n*d)
	y := make([]float64, n)
	for i := 0; i < n; i++ {
		z := verdaderos[d]
		for j := 0; j < d; j++ {
			v := r.NormFloat64()
			x[i*d+j] = v
			z += verdaderos[j] * v
		}
		if r.Float64() < Sigmoide(z) {
			y[i] = 1
		}
	}
	return &Dataset{X: x, Y: y, N: n, D: d}, verdaderos
}

// Prefijo devuelve un Dataset con las primeras n filas, sin copiar memoria.
func (ds *Dataset) Prefijo(n int) *Dataset {
	n = min(n, ds.N)
	return &Dataset{X: ds.X[:n*ds.D], Y: ds.Y[:n], N: n, D: ds.D}
}
