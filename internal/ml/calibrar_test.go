package ml

import (
	"math"
	"testing"
)

func TestCalibrarDeshaceElAjusteDelLogit(t *testing.T) {
	// Sin ajuste no cambia nada; con ajuste a, logit(Calibrar(p)) = logit(p) − a.
	for _, p := range []float64{0.01, 0.3, 0.5, 0.9} {
		if got := Calibrar(p, 0); math.Abs(got-p) > 1e-12 {
			t.Fatalf("Calibrar(%v, 0) = %v", p, got)
		}
		q := Calibrar(p, 2)
		if d := math.Log(p/(1-p)) - math.Log(q/(1-q)); math.Abs(d-2) > 1e-9 {
			t.Fatalf("logit(p) − logit(q) = %v, se esperaba 2", d)
		}
	}
	if q := Calibrar(1, 1); math.IsNaN(q) || q <= 0 || q >= 1 {
		t.Fatalf("Calibrar(1, 1) = %v", q)
	}
}
