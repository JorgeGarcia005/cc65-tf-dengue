package main

import (
	"math"
	"testing"
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
