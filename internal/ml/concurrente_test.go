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

// ADR-0005: Parciales devuelve un parcial por bloque, sin reducir, idéntico bit
// a bit al de ParcialBloque. Es lo que cada nodo del clúster envía al coordinador.
func TestParcialesDevuelveUnParcialPorBloqueIgualAlSecuencial(t *testing.T) {
	ds, _ := datasetSintetico(t, 5_003, 5, 11)
	pos, neg := PesosBalanceados(ds.Y)
	cfg := Config{TamBloque: 256, Lambda: 1e-3, PesoPos: pos, PesoNeg: neg}
	w := pesosAleatorios(ds.D, 3)
	for _, workers := range []int{1, 3, 8} {
		p := NuevoPool(nil, cfg, workers)
		parciales := p.Parciales(ds, w)
		p.Cerrar()
		if want := NumBloques(ds.N, cfg.TamBloque); len(parciales) != want {
			t.Fatalf("workers=%d: %d parciales, quiere %d", workers, len(parciales), want)
		}
		ref := NuevoParcial(ds.D)
		for k := range parciales {
			ParcialBloque(ds, cfg, w, k, &ref)
			ctx := fmt.Sprintf("workers=%d bloque %d", workers, k)
			mismosBits(t, ctx+" grad", ref.Grad, parciales[k].Grad)
			mismosBits(t, ctx+" pérdida y peso", []float64{ref.Perdida, ref.Peso},
				[]float64{parciales[k].Perdida, parciales[k].Peso})
		}
	}
}

// ADR-0005 (objeción 3 del juez): un nodo con varios shards usa un único Pool.
// Los mismos workers atienden datasets distintos, incluso con otro D, y cada
// llamada devuelve memoria propia que la siguiente no pisa.
func TestParcialesReutilizaLosWorkersConDatasetsDistintos(t *testing.T) {
	a, _ := datasetSintetico(t, 2_000, 4, 1)
	b, _ := datasetSintetico(t, 1_300, 7, 2)
	cfg := Config{TamBloque: 128}
	p := NuevoPool(nil, cfg, 4)
	defer p.Cerrar()
	wa, wb := pesosAleatorios(a.D, 5), pesosAleatorios(b.D, 6)

	pa := p.Parciales(a, wa)
	pb := p.Parciales(b, wb)
	pa2 := p.Parciales(a, wa)

	ga := Finalizar(ReducirEnOrden(pa, a.D), cfg.normalizada(), wa)
	gb := Finalizar(ReducirEnOrden(pb, b.D), cfg.normalizada(), wb)
	mismosBits(t, "dataset a", NuevoSecuencial(a, cfg).Gradiente(wa).Grad, ga.Grad)
	mismosBits(t, "dataset b", NuevoSecuencial(b, cfg).Gradiente(wb).Grad, gb.Grad)
	for k := range pa {
		mismosBits(t, fmt.Sprintf("a repetido, bloque %d", k), pa[k].Grad, pa2[k].Grad)
		if &pa[k].Grad[0] == &pa2[k].Grad[0] {
			t.Fatalf("bloque %d: dos llamadas comparten memoria", k)
		}
	}
}

// Gradiente y Parciales se pueden intercalar sobre el mismo Pool.
func TestGradienteYParcialesConvivenEnElMismoPool(t *testing.T) {
	ds, _ := datasetSintetico(t, 3_000, 4, 8)
	otro, _ := datasetSintetico(t, 900, 4, 9)
	cfg := Config{TamBloque: 200, Lambda: 1e-2}
	p := NuevoPool(ds, cfg, 4)
	defer p.Cerrar()
	w := pesosAleatorios(ds.D, 1)
	ref := NuevoSecuencial(ds, cfg).Gradiente(w)

	var wg sync.WaitGroup
	for i := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				mismosBits(t, "gradiente", ref.Grad, p.Gradiente(w).Grad)
			} else {
				p.Parciales(otro, w)
			}
		}()
	}
	wg.Wait()
}

func TestGradienteSinDatasetBaseEntraEnPanico(t *testing.T) {
	p := NuevoPool(nil, Config{}, 2)
	defer p.Cerrar()
	defer func() {
		if recover() == nil {
			t.Error("esperaba pánico: el pool no tiene dataset base")
		}
	}()
	p.Gradiente(make([]float64, 3))
}
