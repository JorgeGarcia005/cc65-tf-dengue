package ml

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"testing"
)

func pesosAleatorios(d int, semilla uint64) []float64 {
	r := rand.New(rand.NewPCG(semilla, 1))
	w := make([]float64, d+1)
	for j := range w {
		w[j] = 0.5 * r.NormFloat64()
	}
	return w
}

func mismosBits(t *testing.T, contexto string, a, b []float64) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("%s: largos distintos %d vs %d", contexto, len(a), len(b))
	}
	for j := range a {
		if math.Float64bits(a[j]) != math.Float64bits(b[j]) {
			t.Fatalf("%s: posición %d difiere: %.17g vs %.17g", contexto, j, a[j], b[j])
		}
	}
}

// TestEquivalencia exige que el worker pool dé exactamente los mismos bits que
// la versión secuencial, para cualquier número de workers: tanto un gradiente
// suelto como los pesos después de 10 épocas de entrenamiento.
func TestEquivalencia(t *testing.T) {
	// 10 007 filas con bloques de 512 → 20 bloques, el último incompleto.
	ds, _ := datasetSintetico(t, 10_007, 6, 42)
	pos, neg := PesosBalanceados(ds.Y)
	cfg := Config{TamBloque: 512, Lambda: 1e-3, PesoPos: pos, PesoNeg: neg}
	w := pesosAleatorios(ds.D, 9)
	ent := Entrenamiento{Epocas: 10, Tasa: 0.5}

	sec := NuevoSecuencial(ds, cfg)
	gRef := sec.Gradiente(w)
	resRef := Entrenar(sec, ds.D, ent)

	for _, workers := range []int{1, 2, 4, 8, 16, 64} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			p := NuevoPool(ds, cfg, workers)
			defer p.Cerrar()

			g := p.Gradiente(w)
			mismosBits(t, "gradiente", gRef.Grad, g.Grad)
			mismosBits(t, "pérdida", []float64{gRef.Perdida}, []float64{g.Perdida})

			res := Entrenar(p, ds.D, ent)
			mismosBits(t, "pesos tras 10 épocas", resRef.W, res.W)
			mismosBits(t, "historia de pérdida", resRef.Historia, res.Historia)
		})
	}
}

func TestPoolCopiaLosPesos(t *testing.T) {
	ds, _ := datasetSintetico(t, 3_000, 3, 5)
	p := NuevoPool(ds, Config{TamBloque: 100}, 4)
	defer p.Cerrar()
	w := pesosAleatorios(ds.D, 2)
	antes := p.Gradiente(w)
	w[0] += 10 // el llamador modifica sus pesos después de la llamada
	original := pesosAleatorios(ds.D, 2)
	despues := p.Gradiente(original)
	mismosBits(t, "el pool no debe retener el slice del llamador", antes.Grad, despues.Grad)
}

func TestPoolCerrarEsIdempotente(t *testing.T) {
	ds, _ := datasetSintetico(t, 100, 2, 1)
	p := NuevoPool(ds, Config{}, 3)
	p.Cerrar()
	p.Cerrar()
}

func TestPoolUsarDespuesDeCerrarEntraEnPanico(t *testing.T) {
	ds, _ := datasetSintetico(t, 100, 2, 1)
	p := NuevoPool(ds, Config{}, 2)
	p.Cerrar()
	defer func() {
		if recover() == nil {
			t.Error("esperaba pánico al usar un pool cerrado")
		}
	}()
	p.Gradiente(make([]float64, ds.D+1))
}

func TestWorkersInvalidosSeAjustan(t *testing.T) {
	ds, _ := datasetSintetico(t, 100, 2, 1)
	p := NuevoPool(ds, Config{}, 0)
	defer p.Cerrar()
	if p.Workers() != 1 {
		t.Errorf("Workers() = %d con 0 pedidos, quiere 1", p.Workers())
	}
}

func BenchmarkGradiente(b *testing.B) {
	ds, _ := datasetSintetico(b, 200_000, 40, 3)
	cfg := Config{}
	w := pesosAleatorios(ds.D, 4)
	b.Run("secuencial", func(b *testing.B) {
		s := NuevoSecuencial(ds, cfg)
		for b.Loop() {
			s.Gradiente(w)
		}
	})
	for _, workers := range []int{2, 4, 8} {
		b.Run(fmt.Sprintf("pool-%d", workers), func(b *testing.B) {
			p := NuevoPool(ds, cfg, workers)
			defer p.Cerrar()
			for b.Loop() {
				p.Gradiente(w)
			}
		})
	}
}

// PR-12 H1: varias goroutines llamando Gradiente a la vez sobre el mismo Pool.
// El mutex debe serializarlas (sin carreras bajo -race) y cada una debe recibir
// exactamente el gradiente de sus propios pesos.
func TestPoolLlamadasConcurrentes(t *testing.T) {
	ds, _ := datasetSintetico(t, 4_000, 4, 17)
	cfg := Config{TamBloque: 256}
	p := NuevoPool(ds, cfg, 4)
	defer p.Cerrar()
	sec := NuevoSecuencial(ds, cfg)

	const llamadores = 8
	pesos := make([][]float64, llamadores)
	esperados := make([]Gradiente, llamadores)
	for i := range pesos {
		pesos[i] = pesosAleatorios(ds.D, uint64(100+i))
		esperados[i] = sec.Gradiente(pesos[i])
	}

	obtenidos := make([]Gradiente, llamadores)
	var wg sync.WaitGroup
	for i := range llamadores {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 5 {
				obtenidos[i] = p.Gradiente(pesos[i])
			}
		}()
	}
	wg.Wait()

	for i := range llamadores {
		mismosBits(t, fmt.Sprintf("llamador %d", i), esperados[i].Grad, obtenidos[i].Grad)
	}
}

// PR-12 H1: un w de largo incorrecto falla con mensaje claro en vez de mezclar
// pesos de la época anterior.
func TestPesosDeLargoIncorrectoEntranEnPanico(t *testing.T) {
	ds, _ := datasetSintetico(t, 500, 3, 1)
	p := NuevoPool(ds, Config{}, 2)
	defer p.Cerrar()
	calcs := map[string]Calculador{"pool": p, "secuencial": NuevoSecuencial(ds, Config{})}
	for nombre, c := range calcs {
		for _, largo := range []int{ds.D, ds.D + 2} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("%s con len(w)=%d: esperaba pánico", nombre, largo)
					}
				}()
				c.Gradiente(make([]float64, largo))
			}()
		}
	}
}
