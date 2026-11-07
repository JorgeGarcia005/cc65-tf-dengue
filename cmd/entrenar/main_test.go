package main

import (
	"math"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

func TestCalibrar(t *testing.T) {
	// Sin ajuste, la probabilidad no cambia.
	for _, p := range []float64{0.01, 0.3, 0.5, 0.9} {
		if got := Calibrar(p, 0); math.Abs(got-p) > 1e-12 {
			t.Errorf("Calibrar(%v, 0) = %v", p, got)
		}
	}
	// Con pesos 4:1 a favor de positivos, una predicción de 0.8 (odds 4) vuelve a odds 1.
	if got := Calibrar(0.8, math.Log(4)); math.Abs(got-0.5) > 1e-12 {
		t.Errorf("Calibrar(0.8, ln 4) = %v, quiere 0.5", got)
	}
	// Extremos sin NaN.
	for _, p := range []float64{0, 1} {
		if got := Calibrar(p, 1); math.IsNaN(got) || got < 0 || got > 1 {
			t.Errorf("Calibrar(%v) = %v", p, got)
		}
	}
}

// TestAgruparPorSemana: cada caso cae en la semana que codifican sus columnas
// seno y coseno; las semanas salen en orden 1..52 y solo las que tienen casos.
func TestAgruparPorSemana(t *testing.T) {
	const d = 3 // columnas: otra, semana_sin, semana_cos
	semanas := []int{5, 5, 5, 30, 52}
	y := []float64{1, 0, 0, 1, 0}
	p := []float64{0.5, 0.5, 0.2, 0.9, 0.1}
	x := make([]float64, 0, d*len(semanas))
	for _, s := range semanas {
		a := 2 * math.Pi * float64(s-1) / 52
		x = append(x, 7, math.Sin(a), math.Cos(a))
	}
	ds, err := ml.NuevoDataset(x, y, d)
	if err != nil {
		t.Fatal(err)
	}
	gs := agruparPorSemana(ds, y, p, 1, 2, 0)
	if len(gs) != 3 {
		t.Fatalf("%d grupos, quiere 3: %+v", len(gs), gs)
	}
	quiere := []struct {
		semana, casos int
		obs, pred     float64
	}{{5, 3, 1.0 / 3, 0.4}, {30, 1, 1, 0.9}, {52, 1, 0, 0.1}}
	for k, q := range quiere {
		g := gs[k]
		if g.Semana != q.semana || g.Casos != q.casos ||
			math.Abs(g.TasaObservada-q.obs) > 1e-12 || math.Abs(g.RiesgoMedio-q.pred) > 1e-12 {
			t.Errorf("grupo %d = %+v, quiere semana %d, %d casos, obs %.3f, pred %.3f", k, g, q.semana, q.casos, q.obs, q.pred)
		}
	}
}
