package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
)

// #69: -eventos graba cada evento como una línea JSON que el panel admin
// (web/src/admin/replica.ts) puede reproducir sin clúster.
func TestAbrirEventosGrabaUnaLineaPorEvento(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "eventos_demo.jsonl")
	var op opciones
	cerrar, err := abrirEventos(ruta, &op, silencio)
	if err != nil {
		t.Fatal(err)
	}
	if op.grabar == nil || op.observador == nil {
		t.Fatal("abrirEventos no colgó el grabador")
	}
	for _, e := range []cluster.Evento{
		{V: cluster.VersionEventos, Tipo: cluster.EvRegistro, Nodo: "nodo-0"},
		{V: cluster.VersionEventos, Tipo: cluster.EvEpoca, Epoca: 1, Perdida: 0.69, Pesos: []float64{0.5, -0.25}},
		{V: cluster.VersionEventos, Tipo: cluster.EvFin, Epoca: 1},
	} {
		op.observador(e)
	}
	cerrar()
	cerrar() // idempotente

	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var tipos []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e cluster.Evento
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("línea %d no es un evento: %v", len(tipos)+1, err)
		}
		tipos = append(tipos, e.Tipo)
	}
	if len(tipos) != 3 || tipos[0] != cluster.EvRegistro || tipos[1] != cluster.EvEpoca || tipos[2] != cluster.EvFin {
		t.Fatalf("eventos grabados %v; quería registro, epoca, fin", tipos)
	}
}

func TestAbrirEventosSinRutaNoGraba(t *testing.T) {
	var op opciones
	cerrar, err := abrirEventos("", &op, silencio)
	if err != nil || op.grabar != nil || op.observador != nil {
		t.Fatalf("sin ruta no debe colgar nada: err=%v", err)
	}
	cerrar()
}

func TestAbrirEventosRutaInvalida(t *testing.T) {
	var op opciones
	if _, err := abrirEventos(filepath.Join(t.TempDir(), "no-existe", "x.jsonl"), &op, silencio); err == nil {
		t.Fatal("una carpeta inexistente debe dar error")
	}
}
