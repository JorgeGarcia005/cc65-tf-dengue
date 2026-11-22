package main

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// preparar escribe un gold de prueba y un modelo.json como los de cmd/entrenar.
func preparar(t *testing.T) opciones {
	t.Helper()
	dir := t.TempDir()
	r := rand.New(rand.NewPCG(11, 11))
	deptos := []string{"LORETO", "PIURA", "TUMBES"}
	regs := make([]ingest.Registro, 2048)
	for i := range regs {
		regs[i] = ingest.Registro{Departamento: deptos[r.IntN(3)], Ano: 2020, Semana: 1 + r.IntN(52),
			EdadAnios: float64(r.IntN(80)), Sexo: 'F'}
		if r.IntN(5) == 0 {
			regs[i].Clase = 1
		}
	}
	esq, err := features.Ajustar(regs, 2011, 10)
	if err != nil {
		t.Fatal(err)
	}
	x, y := make([]float64, len(regs)*esq.D()), make([]float64, len(regs))
	for i, rg := range regs {
		esq.Llenar(rg, x[i*esq.D():(i+1)*esq.D()])
		y[i] = rg.Objetivo()
	}
	ds, err := ml.NuevoDataset(x, y, esq.D())
	if err != nil {
		t.Fatal(err)
	}
	if err := gold.Escribir(filepath.Join(dir, "test.bin"), gold.Archivo{Dataset: ds, TamBloque: 256}); err != nil {
		t.Fatal(err)
	}
	pos, neg := ml.PesosBalanceados(y)
	w := ml.Entrenar(ml.NuevoSecuencial(ds, ml.Config{TamBloque: 256, PesoPos: pos, PesoNeg: neg}), ds.D,
		ml.Entrenamiento{Epocas: 10, Tasa: 1}).W
	m, _ := json.Marshal(map[string]any{"pesos": w, "esquema": esq, "umbral": 0.5, "ajuste_prior": math.Log(pos / neg)})
	modelo := filepath.Join(dir, "modelo.json")
	if err := os.WriteFile(modelo, m, 0o644); err != nil {
		t.Fatal(err)
	}
	return opciones{gold: dir, modelo: modelo, cluster: "../../reports/bench_cluster_deez_gold.json",
		benchAPI: "../../reports/bench_api_deez.json", salida: filepath.Join(dir, "evaluacion.json"),
		minCasos: 5, top: 5, puntosROC: 50, deciles: 10}
}

func TestEjecutarDeterminista(t *testing.T) {
	o := preparar(t)
	if _, err := ejecutar(o); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(o.salida)
	if _, err := ejecutar(o); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(o.salida)
	if !bytes.Equal(a, b) {
		t.Fatal("dos corridas producen archivos distintos")
	}
	var rep struct {
		Fuentes []struct{ Archivo, Sha256 string }
		Version string `json:"version_modelo"`
	}
	if err := json.Unmarshal(a, &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Fuentes) != 4 || len(rep.Version) != 12 {
		t.Fatalf("fuentes o versión inesperadas: %+v", rep)
	}
}

func TestEjecutarFaltaArchivo(t *testing.T) {
	o := preparar(t)
	o.cluster = filepath.Join(o.gold, "no-existe.json")
	if _, err := ejecutar(o); err == nil {
		t.Fatal("debía fallar sin el benchmark del clúster")
	}
}
