package estadistica

import (
	"math"
	"testing"
)

func TestMediaRecortadaDescartaAtipicos(t *testing.T) {
	xs := []float64{10, 11, 9, 10, 1000, 10, 11, 9, 10, 0.001}
	// 10 % de 10 = 1 valor por extremo: se van 0.001 y 1000.
	if got := MediaRecortada(xs, 0.1); math.Abs(got-10) > 1e-12 {
		t.Errorf("MediaRecortada = %v, quiere 10", got)
	}
	if got := MediaRecortada(xs, 0); math.Abs(got-108.0001) > 1e-9 {
		t.Errorf("sin recorte = %v, quiere la media simple 108.0001", got)
	}
}

func TestMediaRecortadaNoModificaLaEntrada(t *testing.T) {
	xs := []float64{3, 1, 2}
	MediaRecortada(xs, 0.2)
	if xs[0] != 3 || xs[1] != 1 || xs[2] != 2 {
		t.Errorf("la entrada cambió: %v", xs)
	}
}

func TestMedianaYDesviacion(t *testing.T) {
	if m := Mediana([]float64{5, 1, 3}); m != 3 {
		t.Errorf("Mediana impar = %v", m)
	}
	if m := Mediana([]float64{4, 1, 3, 2}); m != 2.5 {
		t.Errorf("Mediana par = %v", m)
	}
	if d := Desviacion([]float64{2, 4, 4, 4, 5, 5, 7, 9}); math.Abs(d-2.138089935299395) > 1e-12 {
		t.Errorf("Desviacion = %v", d)
	}
}

func TestICBootstrapContieneLaEstimacionYEsDeterminista(t *testing.T) {
	xs := []float64{1.0, 1.1, 0.9, 1.05, 0.95, 1.2, 0.8, 1.0, 1.02, 0.98}
	inf, sup := ICBootstrap(xs, 0.1, 2000, 7)
	m := MediaRecortada(xs, 0.1)
	if !(inf <= m && m <= sup) {
		t.Errorf("IC [%v, %v] no contiene la estimación %v", inf, sup, m)
	}
	inf2, sup2 := ICBootstrap(xs, 0.1, 2000, 7)
	if inf != inf2 || sup != sup2 {
		t.Error("el bootstrap no es determinista con la misma semilla")
	}
}

func TestResumirVacio(t *testing.T) {
	if r := Resumir(nil, 0.1); r.N != 0 {
		t.Errorf("Resumir(nil) = %+v", r)
	}
}
