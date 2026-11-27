package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
)

// El gold sintético tiene la forma del real: train.bin, test.bin y shards de
// bloques contiguos que, juntos, cubren exactamente el train.
func TestGenerarSinteticoEscribeTrainTestYShardsContiguos(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gold")
	if err := generarSintetico(dir, 5_000, 3, 512, 0.2); err != nil {
		t.Fatal(err)
	}
	train, err := gold.Leer(filepath.Join(dir, "train.bin"))
	if err != nil {
		t.Fatal(err)
	}
	test, err := gold.Leer(filepath.Join(dir, "test.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if train.N != 5_000 || train.D != variablesSintetico || train.TamBloque != 512 || test.N != 1_000 {
		t.Fatalf("train %d×%d (bloque %d), test %d filas", train.N, train.D, train.TamBloque, test.N)
	}
	filas, bloque := 0, 0
	for s := range 3 {
		a, err := gold.Leer(filepath.Join(dir, fmt.Sprintf("shard_%d.bin", s)))
		if err != nil {
			t.Fatal(err)
		}
		if a.BloqueInicial != bloque {
			t.Fatalf("shard %d empieza en el bloque %d, quiere %d", s, a.BloqueInicial, bloque)
		}
		for i := range a.N {
			if a.Y[i] != train.Y[filas+i] || a.Fila(i)[0] != train.Fila(filas + i)[0] {
				t.Fatalf("shard %d, fila %d: no coincide con el train", s, i)
			}
		}
		filas += a.N
		bloque += (a.N + 511) / 512
	}
	if filas != train.N {
		t.Fatalf("los shards suman %d filas, el train tiene %d", filas, train.N)
	}
	// Auditoría TB2 B1: también el esquema, con las mismas variables que el gold,
	// para que cmd/entrenar escriba modelo.json y la API arranque.
	b, err := os.ReadFile(filepath.Join(dir, "esquema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var esq features.Esquema
	if err := json.Unmarshal(b, &esq); err != nil || esq.Preparar() != nil || esq.D() != train.D {
		t.Fatalf("esquema.json: %v (D %d, gold %d)", err, esq.D(), train.D)
	}
	if err := generarSintetico(dir, 100, 5, 512, 0.2); err == nil {
		t.Error("5 shards para 1 bloque: esperaba error")
	}
}
