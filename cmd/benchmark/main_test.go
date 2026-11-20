package main

import (
	"math"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// TestPasadasPorMedicionSuperaLaResolucionDelReloj: en Windows el reloj avanza
// a saltos de ~0,5–1 ms, así que una pasada de 30 µs se mediría como 0. Cada
// medición debe agrupar suficientes pasadas para durar al menos el mínimo.
func TestPasadasPorMedicionSuperaLaResolucionDelReloj(t *testing.T) {
	const pasada = 2 * time.Millisecond
	const minimo = 20 * time.Millisecond
	k := pasadasPorMedicion(func() { time.Sleep(pasada) }, minimo)
	if k < 2 {
		t.Fatalf("k = %d: una pasada de %v no alcanza %v", k, pasada, minimo)
	}
	if k > 10 {
		t.Fatalf("k = %d: sobran pasadas (cada una dura al menos %v)", k, pasada)
	}
	if k := pasadasPorMedicion(func() { time.Sleep(30 * time.Millisecond) }, minimo); k != 1 {
		t.Fatalf("k = %d: una pasada larga debe medirse sola", k)
	}
}

// Issue #32: el benchmark del clúster mide 1–3 nodos con el protocolo real,
// verifica cada repetición bit a bit contra el secuencial y separa cómputo,
// reducción, red y espera, JSON y bytes por época.
func TestBenchmarkClusterSeparaTiemposYVerificaPesos(t *testing.T) {
	ds, _ := ml.Sintetico(30_000, 12, 3)
	train := gold.Archivo{Dataset: ds, TamBloque: 1024}
	op := opcionesCluster{nodos: []int{1, 2, 3}, reps: 2, epocas: 4, workersNodo: 1, tasa: 1, lambda: 1e-3,
		recorte: 0, logf: func(string, ...any) {}}
	med, err := medirCluster(train, op)
	if err != nil {
		t.Fatal(err)
	}
	if len(med) != 3 {
		t.Fatalf("%d mediciones, se esperaban 3", len(med))
	}
	for _, m := range med {
		d := m.Desglose
		if !m.Equivalente || m.Repeticiones != 2 || m.SegPorEpoca.N != 2 {
			t.Fatalf("%d nodos: %+v", m.Nodos, m)
		}
		if !(d.Epoca > 0 && d.CalculoNodoMax > 0 && d.CalculoNodoMax <= d.CalculoNodos && d.Reduccion > 0 &&
			d.RedYEspera >= 0 && d.Serializacion > 0 && d.FraccionRed >= 0 && d.FraccionRed < 1) {
			t.Fatalf("%d nodos: desglose inconsistente %+v", m.Nodos, d)
		}
		if m.BytesPorEpoca <= float64(8*(ds.D+1)) {
			t.Fatalf("%d nodos: %.0f bytes por época", m.Nodos, m.BytesPorEpoca)
		}
	}
	if med[0].Speedup != 1 || med[0].KarpFlatt != nil || med[1].KarpFlatt == nil || med[2].KarpFlatt == nil {
		t.Fatalf("speedup o Karp-Flatt mal definidos: %+v", med)
	}
	// Karp-Flatt coherente con el speedup: S = 1 / (e + (1−e)/p).
	for _, m := range med[1:] {
		p, e := float64(m.Nodos), *m.KarpFlatt
		if s := 1 / (e + (1-e)/p); math.Abs(s-m.Speedup) > 1e-9 {
			t.Fatalf("%d nodos: speedup %v, Karp-Flatt implica %v", m.Nodos, m.Speedup, s)
		}
	}
}

func TestBenchmarkClusterRechazaOpcionesInvalidas(t *testing.T) {
	ds, _ := ml.Sintetico(1000, 3, 1)
	if _, err := medirCluster(gold.Archivo{Dataset: ds, TamBloque: 256}, opcionesCluster{nodos: []int{1}, reps: 0, epocas: 1, workersNodo: 1, logf: func(string, ...any) {}}); err == nil {
		t.Fatal("reps = 0 debía fallar")
	}
}
