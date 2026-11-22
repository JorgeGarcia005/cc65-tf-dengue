package main

import (
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"math"
	"sync/atomic"
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
		// La reducción dura decenas de µs: en Windows puede medirse 0 (PR-42 H5).
		if !(d.Epoca > 0 && d.CalculoNodoMax > 0 && d.CalculoNodoMax <= d.CalculoNodos && d.Reduccion >= 0 &&
			d.RedYEspera >= 0 && d.SerializacionShard > 0 && d.SerializacionShard <= d.SerializacionTotal &&
			d.FraccionRed >= 0 && d.FraccionRed < 1) {
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

// PR-42 H1: si los nodos no llegan a registrarse, medirCluster vuelve con el
// error en vez de quedarse esperando a los nodos que sí se conectaron.
func TestBenchmarkClusterSinTodosLosNodosVuelveConError(t *testing.T) {
	ds, _ := ml.Sintetico(4000, 4, 2)
	op := opcionesCluster{nodos: []int{2}, reps: 1, epocas: 1, workersNodo: 1, tasa: 1, lambda: 1e-3,
		logf: func(string, ...any) {}, timeoutRegistro: 300 * time.Millisecond, conectar: 1}
	fin := make(chan error, 1)
	go func() {
		_, err := medirCluster(gold.Archivo{Dataset: ds, TamBloque: 512}, op)
		fin <- err
	}()
	select {
	case err := <-fin:
		if err == nil {
			t.Fatal("con 1 de 2 nodos debía fallar el registro")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("medirCluster quedó colgado tras fallar el registro")
	}
}

// PR-42 H7: -nodos desordenado o con repetidos se ordena y la base es el
// menor; Karp-Flatt solo se define contra 1 nodo.
func TestBenchmarkClusterOrdenaNodosYKarpFlattSoloContraUnNodo(t *testing.T) {
	ds, _ := ml.Sintetico(6000, 4, 2)
	train := gold.Archivo{Dataset: ds, TamBloque: 512}
	base := opcionesCluster{reps: 1, epocas: 2, workersNodo: 1, tasa: 1, lambda: 1e-3, logf: func(string, ...any) {}}
	op := base
	op.nodos = []int{3, 1, 2, 2}
	med, err := medirCluster(train, op)
	if err != nil {
		t.Fatal(err)
	}
	if len(med) != 3 || med[0].Nodos != 1 || med[1].Nodos != 2 || med[2].Nodos != 3 || med[0].Speedup != 1 {
		t.Fatalf("orden o base incorrectos: %+v", med)
	}
	op = base
	op.nodos = []int{3, 2}
	med, err = medirCluster(train, op)
	if err != nil {
		t.Fatal(err)
	}
	if med[0].Nodos != 2 || med[0].KarpFlatt != nil || med[1].KarpFlatt != nil {
		t.Fatalf("con base de 2 nodos no se define Karp-Flatt: %+v", med)
	}
}

func TestBenchmarkObservadorPublicaYNoCambiaLosPesos(t *testing.T) {
	ds, _ := ml.Sintetico(20_000, 8, 5)
	train := gold.Archivo{Dataset: ds, TamBloque: 1024}
	op := opcionesCluster{nodos: []int{1, 3}, reps: 2, epocas: 3, workersNodo: 1, tasa: 1, lambda: 1e-3,
		recorte: 0, logf: func(string, ...any) {}}
	rep, err := medirObservador(train, op)
	if err != nil {
		t.Fatal(err) // medirCluster ya exige pesos bit a bit iguales al secuencial en cada repetición
	}
	if len(rep.Sin) != 2 || len(rep.Con) != 2 || len(rep.Sobrecosto) != 2 {
		t.Fatalf("mediciones: %+v", rep)
	}
	// Por cada nodos: registro + (calentamiento + 2 repeticiones) × 3 épocas con tarea,
	// gradiente por shard y la época; al menos una época por repetición.
	if rep.Eventos < int64(2*3*3) {
		t.Fatalf("eventos publicados: %d", rep.Eventos)
	}
	for _, s := range rep.Sobrecosto {
		if !(s.MsSin > 0 && s.MsCon > 0) {
			t.Fatalf("sobrecosto sin tiempos: %+v", s)
		}
	}
	if op.observador != nil {
		t.Fatal("medirObservador no debe modificar las opciones del llamador")
	}
	// Aunque el llamador traiga un observador, la corrida de control va sin él.
	var llamadas atomic.Int64
	op.observador = func(cluster.Evento) { llamadas.Add(1) }
	op.nodos, op.reps, op.epocas = []int{1}, 1, 1
	if _, err := medirObservador(train, op); err != nil {
		t.Fatal(err)
	}
	if llamadas.Load() != 0 {
		t.Fatalf("el observador del llamador recibió %d eventos", llamadas.Load())
	}
}
