package ml

import (
	"math"
	"testing"
)

func TestAUCCasosConocidos(t *testing.T) {
	casos := []struct {
		nombre string
		y, p   []float64
		quiere float64
	}{
		{"perfecto", []float64{0, 0, 1, 1}, []float64{0.1, 0.2, 0.8, 0.9}, 1},
		{"invertido", []float64{0, 0, 1, 1}, []float64{0.9, 0.8, 0.2, 0.1}, 0},
		{"todo empatado", []float64{0, 1, 0, 1}, []float64{0.5, 0.5, 0.5, 0.5}, 0.5},
		// 3 de 4 pares (pos, neg) bien ordenados.
		{"mixto", []float64{0, 1, 0, 1}, []float64{0.1, 0.4, 0.5, 0.8}, 0.75},
		// Un empate entre pos y neg cuenta medio par.
		{"empate parcial", []float64{0, 1}, []float64{0.5, 0.5}, 0.5},
	}
	for _, c := range casos {
		got, err := AUC(c.y, c.p)
		if err != nil {
			t.Fatalf("%s: %v", c.nombre, err)
		}
		if math.Abs(got-c.quiere) > 1e-12 {
			t.Errorf("%s: AUC = %v, quiere %v", c.nombre, got, c.quiere)
		}
	}
}

func TestAUCRequiereAmbasClases(t *testing.T) {
	if _, err := AUC([]float64{1, 1}, []float64{0.2, 0.3}); err == nil {
		t.Error("esperaba error sin negativos")
	}
}

func TestConfusionYMetricas(t *testing.T) {
	y := []float64{1, 1, 1, 0, 0, 0, 0, 1}
	p := []float64{0.9, 0.6, 0.4, 0.7, 0.2, 0.1, 0.3, 0.8}
	m := Evaluar(y, p, 0.5)
	if m.VP != 3 || m.FN != 1 || m.FP != 1 || m.VN != 3 {
		t.Fatalf("confusión = %+v", m)
	}
	cerca := func(nombre string, got, quiere float64) {
		if math.Abs(got-quiere) > 1e-12 {
			t.Errorf("%s = %v, quiere %v", nombre, got, quiere)
		}
	}
	cerca("recall", m.Recall, 0.75)
	cerca("precision", m.Precision, 0.75)
	cerca("f1", m.F1, 0.75)
	cerca("exactitud", m.Exactitud, 0.75)
	// 14 de los 16 pares (positivo, negativo) quedan bien ordenados.
	cerca("auc", m.AUC, 0.875)
}
