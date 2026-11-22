package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLeerEvaluacion(t *testing.T) {
	if b, err := leerEvaluacion("", silencio); b != nil || err != nil {
		t.Fatal("sin ruta debía devolver nil, nil")
	}
	if b, err := leerEvaluacion(filepath.Join(t.TempDir(), "no-existe.json"), silencio); b != nil || err != nil {
		t.Fatalf("sin archivo la API arranca igual: %v %v", b, err)
	}
	dir := t.TempDir()
	vacio := filepath.Join(dir, "vacio.json")
	if err := os.WriteFile(vacio, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerEvaluacion(vacio, silencio); err == nil {
		t.Fatal("un archivo vacío es un reporte roto: debía fallar")
	}
	ruta := filepath.Join(dir, "evaluacion.json")
	if err := os.WriteFile(ruta, []byte(`{"version_esquema":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := leerEvaluacion(ruta, silencio); err != nil || string(b) != `{"version_esquema":1}` {
		t.Fatalf("contenido %q, err %v", b, err)
	}
}
