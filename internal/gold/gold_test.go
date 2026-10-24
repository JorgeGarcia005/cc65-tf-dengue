package gold

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

func TestEscribirLeerIdaYVuelta(t *testing.T) {
	ds, _ := ml.Sintetico(1_234, 5, 3)
	ruta := filepath.Join(t.TempDir(), "x.bin")
	if err := Escribir(ruta, Archivo{Dataset: ds, TamBloque: 100, BloqueInicial: 7}); err != nil {
		t.Fatal(err)
	}
	a, err := Leer(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if a.N != ds.N || a.D != ds.D || a.TamBloque != 100 || a.BloqueInicial != 7 {
		t.Fatalf("cabecera: %+v", a)
	}
	for i := range ds.X {
		if math.Float64bits(a.X[i]) != math.Float64bits(ds.X[i]) {
			t.Fatalf("x[%d] difiere", i)
		}
	}
	for i := range ds.Y {
		if a.Y[i] != ds.Y[i] {
			t.Fatalf("y[%d] difiere", i)
		}
	}
}

func TestDecodificarRechazaArchivosDanados(t *testing.T) {
	if _, err := Decodificar([]byte("NOESGOLD")); err == nil {
		t.Error("esperaba error con cabecera inválida")
	}
	ds, _ := ml.Sintetico(10, 2, 1)
	ruta := filepath.Join(t.TempDir(), "x.bin")
	if err := Escribir(ruta, Archivo{Dataset: ds, TamBloque: 4}); err != nil {
		t.Fatal(err)
	}
	b, _ := Leer(ruta)
	_ = b
	raw := mustLeerBytes(t, ruta)
	if _, err := Decodificar(raw[:len(raw)-8]); err == nil {
		t.Error("esperaba error con archivo truncado")
	}
}

// TestShardsReproducenElGradienteGlobal es la base del clúster (PC4): si cada
// nodo calcula los parciales de sus bloques locales y el coordinador los
// concatena en orden de shard, la reducción da los mismos bits que la versión
// secuencial sobre el archivo completo.
func TestShardsReproducenElGradienteGlobal(t *testing.T) {
	ds, _ := ml.Sintetico(10_037, 6, 9)
	cfg := ml.Config{TamBloque: 256, Lambda: 1e-3, PesoPos: 3, PesoNeg: 0.6}
	w := []float64{0.1, -0.2, 0.3, 0.05, -0.15, 0.2, 0.01}
	ref := ml.NuevoSecuencial(ds, cfg).Gradiente(w)
	completo := Archivo{Dataset: ds, TamBloque: cfg.TamBloque}

	for _, n := range []int{1, 2, 3, 5, 8} {
		shards, err := Particionar(completo, n)
		if err != nil {
			t.Fatal(err)
		}
		var parciales []ml.Parcial
		filas, siguiente := 0, 0
		for _, s := range shards {
			if s.BloqueInicial != siguiente {
				t.Fatalf("n=%d: shard empieza en el bloque %d, se esperaba %d", n, s.BloqueInicial, siguiente)
			}
			b := ml.NumBloques(s.N, s.TamBloque)
			for k := range b {
				p := ml.NuevoParcial(s.D)
				ml.ParcialBloque(s.Dataset, cfg, w, k, &p)
				parciales = append(parciales, p)
			}
			siguiente += b
			filas += s.N
		}
		if filas != ds.N {
			t.Fatalf("n=%d: los shards suman %d filas, quiere %d", n, filas, ds.N)
		}
		g := ml.Finalizar(ml.ReducirEnOrden(parciales, ds.D), cfg, w)
		for j := range ref.Grad {
			if math.Float64bits(g.Grad[j]) != math.Float64bits(ref.Grad[j]) {
				t.Fatalf("n=%d: grad[%d] %.17g vs %.17g", n, j, g.Grad[j], ref.Grad[j])
			}
		}
	}
}

func TestParticionarValida(t *testing.T) {
	ds, _ := ml.Sintetico(100, 2, 1)
	a := Archivo{Dataset: ds, TamBloque: 10}
	if _, err := Particionar(a, 11); err == nil {
		t.Error("esperaba error con más shards que bloques")
	}
	a.BloqueInicial = 3
	if _, err := Particionar(a, 2); err == nil {
		t.Error("esperaba error al particionar un shard")
	}
}

func mustLeerBytes(t *testing.T, ruta string) []byte {
	t.Helper()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
