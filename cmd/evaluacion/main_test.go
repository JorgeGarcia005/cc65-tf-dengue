package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestEjecutarAUCReferencia(t *testing.T) {
	o := preparar(t)
	rep, err := ejecutar(o) // sin referencia: da el AUC del modelo
	if err != nil {
		t.Fatal(err)
	}
	escribir := func(contenido string) string {
		ruta := filepath.Join(o.gold, "entrenamiento.json")
		if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	o.entrenamiento = escribir(fmt.Sprintf(`{"metricas_test":{"AUC":%v}}`, rep.Metricas.AUC))
	ok, err := ejecutar(o)
	if err != nil {
		t.Fatalf("AUC coincidente: %v", err)
	}
	if len(ok.Fuentes) != 5 {
		t.Fatalf("con referencia se esperaban 5 fuentes, hay %d", len(ok.Fuentes))
	}
	for nombre, c := range map[string]string{
		"AUC divergente": `{"metricas_test":{"AUC":0.123}}`,
		"sin metricas":   `{"pesos":[1]}`, // como reports/entrenamiento_cluster.json
		"AUC cero":       `{"metricas_test":{"AUC":0}}`,
		"JSON inválido":  `{`,
	} {
		o.entrenamiento = escribir(c)
		if _, err := ejecutar(o); err == nil {
			t.Errorf("%s: debía fallar", nombre)
		}
	}
}

func TestEjecutarParametrosInvalidos(t *testing.T) {
	o := preparar(t)
	o.top = -1
	if _, err := ejecutar(o); err == nil {
		t.Fatal("-top -1 debía fallar sin pánico")
	}
	o = preparar(t)
	o.minCasos = 1_000_000 // sin prioridades: no debe entrar en pánico
	if rep, err := ejecutar(o); err != nil || len(rep.Prioridades) != 0 {
		t.Fatalf("sin prioridades: %v, %d", err, len(rep.Prioridades))
	}
}
