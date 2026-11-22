package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLeerEvaluacion(t *testing.T) {
	if b := leerEvaluacion("", silencio); b != nil {
		t.Fatal("sin ruta debía devolver nil")
	}
	if b := leerEvaluacion(filepath.Join(t.TempDir(), "no-existe.json"), silencio); b != nil {
		t.Fatal("sin archivo debía devolver nil (la API arranca igual)")
	}
	ruta := filepath.Join(t.TempDir(), "evaluacion.json")
	if err := os.WriteFile(ruta, []byte(`{"version_esquema":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if b := leerEvaluacion(ruta, silencio); string(b) != `{"version_esquema":1}` {
		t.Fatalf("contenido %q", b)
	}
}
