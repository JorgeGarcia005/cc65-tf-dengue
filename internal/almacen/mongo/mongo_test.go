package mongo

import (
	"context"
	"strings"
	"testing"
	"time"
)

// PR-40 H8: un fallo de conexión no expone usuario ni contraseña de la URI.
func TestErrorDeConexionNoExponeCredenciales(t *testing.T) {
	for _, uri := range []string{
		"mongodb://usuario-ficticio:secreto-ficticio@127.0.0.1:1/?connectTimeoutMS=200",
		"mongodb://usuario-ficticio:secreto-ficticio@127.0.0.1:no-es-puerto",
		"mongodb+srv://usuario-ficticio:secreto-ficticio@cluster.invalid",
	} {
		ctx, cancelar := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		a, err := Conectar(ctx, uri, "cc65_test")
		cancelar()
		if err == nil {
			_ = a.Cerrar(context.Background())
			t.Fatalf("%s: se esperaba error", uri)
		}
		if msg := err.Error(); strings.Contains(msg, "secreto-ficticio") || strings.Contains(msg, "usuario-ficticio") {
			t.Fatalf("el error expone credenciales: %s", msg)
		}
	}
}

func TestSinCredenciales(t *testing.T) {
	cases := map[string]string{
		"mongodb://u:p@h:27017/db?x=1": "mongodb://***@h:27017/db?x=1",
		"mongodb://h:27017":            "mongodb://h:27017",
		"mongodb+srv://solo@h":         "mongodb+srv://***@h",
	}
	for in, want := range cases {
		if got := sinCredenciales(in); got != want {
			t.Errorf("sinCredenciales(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}
