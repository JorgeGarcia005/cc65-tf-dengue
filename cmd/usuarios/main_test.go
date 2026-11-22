package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
)

func TestEjecutar(t *testing.T) {
	mem := almacen.NuevaMemoria()
	var salida bytes.Buffer
	env := map[string]string{"VIGIA_ANALISTA_CORREO": "ana@diresa.pe", "VIGIA_ANALISTA_CLAVE": "una-clave-larga"}
	if err := ejecutar(context.Background(), mem, func(k string) string { return env[k] }, &salida); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida.String(), "ana@diresa.pe (analista)") || strings.Contains(salida.String(), "una-clave-larga") {
		t.Fatalf("salida: %q", salida.String())
	}
	if err := ejecutar(context.Background(), mem, func(string) string { return "" }, &salida); err == nil {
		t.Fatal("sin variables debía fallar")
	}
}
