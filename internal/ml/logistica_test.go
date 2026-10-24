package ml

import (
	"math"
	"math/rand/v2"
	"testing"
)

// datasetSintetico genera n filas con d features y etiquetas de una regresión
// logística conocida. Es determinista: misma semilla, mismos datos.
func datasetSintetico(t testing.TB, n, d int, semilla uint64) (*Dataset, []float64) {
	t.Helper()
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
	ds, err := NuevoDataset(x, y, d)
	if err != nil {
		t.Fatal(err)
	}
	return ds, verdaderos
}

func TestSigmoideEstable(t *testing.T) {
	casos := []struct{ z, quiere float64 }{
		{0, 0.5},
		{1000, 1},
		{-1000, 0},
		{2, 1 / (1 + math.Exp(-2))},
		{-2, 1 / (1 + math.Exp(2))},
	}
	for _, c := range casos {
		got := Sigmoide(c.z)
		if math.IsNaN(got) || math.Abs(got-c.quiere) > 1e-15 {
			t.Errorf("Sigmoide(%v) = %v, quiere %v", c.z, got, c.quiere)
		}
	}
}

func TestPerdidaLogisticaEstable(t *testing.T) {
	// Para z grande, log(1+e^z) no debe desbordar.
	if v := perdidaLogistica(800, 0); math.IsInf(v, 0) || math.Abs(v-800) > 1e-9 {
		t.Errorf("perdidaLogistica(800, 0) = %v, quiere 800", v)
	}
	if v := perdidaLogistica(-800, 0); v < 0 || v > 1e-300 {
		t.Errorf("perdidaLogistica(-800, 0) = %v, quiere ~0", v)
	}
	if v := perdidaLogistica(0, 1); math.Abs(v-math.Ln2) > 1e-15 {
		t.Errorf("perdidaLogistica(0, 1) = %v, quiere ln 2", v)
	}
}

func TestNuevoDatasetValida(t *testing.T) {
	if _, err := NuevoDataset([]float64{1, 2, 3}, []float64{0, 1}, 2); err == nil {
		t.Error("esperaba error: x no es múltiplo de d")
	}
	if _, err := NuevoDataset([]float64{1, 2}, []float64{0, 2}, 1); err == nil {
		t.Error("esperaba error: etiqueta distinta de 0 o 1")
	}
	if _, err := NuevoDataset(nil, nil, 3); err == nil {
		t.Error("esperaba error: dataset vacío")
	}
}

func TestBloquesCubrenTodoSinSolaparse(t *testing.T) {
	ds, _ := datasetSintetico(t, 10_007, 3, 1)
	cfg := Config{TamBloque: 1000}
	b := NumBloques(ds.N, cfg.TamBloque)
	if b != 11 {
		t.Fatalf("NumBloques = %d, quiere 11", b)
	}
	siguiente := 0
	for k := 0; k < b; k++ {
		ini, fin := LimitesBloque(k, ds.N, cfg.TamBloque)
		if ini != siguiente || fin <= ini {
			t.Fatalf("bloque %d = [%d,%d), quiere empezar en %d", k, ini, fin, siguiente)
		}
		siguiente = fin
	}
	if siguiente != ds.N {
		t.Fatalf("los bloques cubren %d filas, quiere %d", siguiente, ds.N)
	}
}

// TestGradienteDiferenciasFinitas compara el gradiente analítico con la
// derivada numérica central de la pérdida, incluidos pesos de clase y L2.
func TestGradienteDiferenciasFinitas(t *testing.T) {
	ds, _ := datasetSintetico(t, 2_000, 5, 7)
	cfg := Config{TamBloque: 128, Lambda: 0.01, PesoPos: 2.5, PesoNeg: 0.7}
	calc := NuevoSecuencial(ds, cfg)
	r := rand.New(rand.NewPCG(3, 4))
	w := make([]float64, ds.D+1)
	for j := range w {
		w[j] = 0.3 * r.NormFloat64()
	}
	g := calc.Gradiente(w)
	const h = 1e-6
	for j := range w {
		mas := append([]float64(nil), w...)
		menos := append([]float64(nil), w...)
		mas[j] += h
		menos[j] -= h
		numerico := (calc.Gradiente(mas).Perdida - calc.Gradiente(menos).Perdida) / (2 * h)
		errRel := math.Abs(numerico-g.Grad[j]) / math.Max(1e-8, math.Abs(numerico)+math.Abs(g.Grad[j]))
		if errRel > 1e-6 {
			t.Errorf("w[%d]: analítico %.10g, numérico %.10g (error relativo %.2e)", j, g.Grad[j], numerico, errRel)
		}
	}
}

func TestGradienteNoDependeDelTamanoDeBloqueSalvoRedondeo(t *testing.T) {
	ds, _ := datasetSintetico(t, 5_000, 4, 11)
	w := []float64{0.1, -0.2, 0.3, -0.4, 0.05}
	a := NuevoSecuencial(ds, Config{TamBloque: 64}).Gradiente(w)
	b := NuevoSecuencial(ds, Config{TamBloque: 5_000}).Gradiente(w)
	for j := range a.Grad {
		if math.Abs(a.Grad[j]-b.Grad[j]) > 1e-12 {
			t.Errorf("grad[%d]: %v vs %v", j, a.Grad[j], b.Grad[j])
		}
	}
}

func TestEntrenarRecuperaLosPesos(t *testing.T) {
	ds, verdaderos := datasetSintetico(t, 40_000, 3, 21)
	calc := NuevoSecuencial(ds, Config{TamBloque: 4096})
	res := Entrenar(calc, ds.D, Entrenamiento{Epocas: 600, Tasa: 1.0})
	for j := range verdaderos {
		if math.Abs(res.W[j]-verdaderos[j]) > 0.1 {
			t.Errorf("w[%d] = %.3f, verdadero %.3f", j, res.W[j], verdaderos[j])
		}
	}
	for e := 1; e < len(res.Historia); e++ {
		if res.Historia[e] > res.Historia[e-1]+1e-12 {
			t.Fatalf("la pérdida subió en la época %d: %v → %v", e, res.Historia[e-1], res.Historia[e])
		}
	}
}

func TestPesosBalanceados(t *testing.T) {
	pos, neg := PesosBalanceados([]float64{1, 0, 0, 0})
	if math.Abs(pos-2) > 1e-15 || math.Abs(neg-4.0/6.0) > 1e-15 {
		t.Errorf("PesosBalanceados = (%v, %v), quiere (2, 0.667)", pos, neg)
	}
	// Con pesos balanceados, cada clase aporta la mitad del peso total.
	if math.Abs(pos*1-neg*3) > 1e-12 {
		t.Errorf("las clases no pesan lo mismo: %v vs %v", pos*1, neg*3)
	}
}
