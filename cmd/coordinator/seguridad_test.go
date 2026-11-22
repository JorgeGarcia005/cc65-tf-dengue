package main

import (
	"context"
	"strings"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
)

func TestPrepararSeguridad(t *testing.T) {
	ctx := context.Background()
	entorno := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	mem := almacen.NuevaMemoria()
	f, err := prepararSeguridad(ctx, mem, entorno(map[string]string{
		"JWT_SECRETO":        "un-secreto-de-al-menos-32-bytes!!",
		"VIGIA_ADMIN_CORREO": "admin@vigia.pe", "VIGIA_ADMIN_CLAVE": "clave-del-admin-larga",
	}), silencio)
	if err != nil || f == nil {
		t.Fatalf("firmador %v, err %v", f, err)
	}
	if _, ok, _ := mem.Usuario(ctx, "admin@vigia.pe"); !ok {
		t.Fatal("no sembró al admin")
	}
	if f, err := prepararSeguridad(ctx, mem, entorno(nil), silencio); err == nil || f != nil || !strings.Contains(err.Error(), "JWT_SECRETO") {
		t.Fatalf("sin JWT_SECRETO debía fallar: firmador %v, err %v", f, err)
	}
	if _, err := prepararSeguridad(ctx, mem, entorno(map[string]string{"JWT_SECRETO": "corto"}), silencio); err == nil {
		t.Fatal("un secreto corto debía fallar")
	}
	if _, err := prepararSeguridad(ctx, mem, entorno(map[string]string{
		"JWT_SECRETO":           "un-secreto-de-al-menos-32-bytes!!",
		"VIGIA_ANALISTA_CORREO": "a@b.pe", "VIGIA_ANALISTA_CLAVE": "corta"}), silencio); err == nil {
		t.Fatal("una clave sembrada corta debía fallar")
	}
}
