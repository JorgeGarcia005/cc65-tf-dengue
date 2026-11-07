package ml

import "testing"

// PR-12 H2: Prefijo comparte memoria con el original y se recorta al tamaño.
func TestPrefijo(t *testing.T) {
	ds, _ := Sintetico(100, 3, 1)
	p := ds.Prefijo(10)
	if p.N != 10 || p.D != 3 || len(p.X) != 30 || len(p.Y) != 10 {
		t.Fatalf("Prefijo(10) = N=%d D=%d len(X)=%d len(Y)=%d", p.N, p.D, len(p.X), len(p.Y))
	}
	if &p.X[0] != &ds.X[0] || p.Fila(9)[2] != ds.Fila(9)[2] {
		t.Error("Prefijo debe compartir memoria con el dataset original")
	}
	if ds.Prefijo(1_000).N != 100 {
		t.Error("Prefijo mayor que N debe devolver N filas")
	}
}

func TestSinteticoEsDeterminista(t *testing.T) {
	a, wa := Sintetico(50, 2, 9)
	b, wb := Sintetico(50, 2, 9)
	for i := range a.X {
		if a.X[i] != b.X[i] {
			t.Fatal("X difiere con la misma semilla")
		}
	}
	for j := range wa {
		if wa[j] != wb[j] {
			t.Fatal("pesos verdaderos difieren con la misma semilla")
		}
	}
}
