package mongo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// PR-40 H8 (rondas 1 y 2): un fallo de conexión no expone usuario,
// contraseña ni opciones sensibles de la cadena de consulta.
func TestErrorDeConexionNoExponeCredenciales(t *testing.T) {
	marcadores := []string{"usuario-ficticio", "secreto-ficticio", "clave-tls-ficticia", "token-ficticio"}
	for _, uri := range []string{
		"mongodb://usuario-ficticio:secreto-ficticio@127.0.0.1:1/?connectTimeoutMS=200",
		"mongodb://usuario-ficticio:secreto-ficticio@127.0.0.1:no-es-puerto",
		"mongodb+srv://usuario-ficticio:secreto-ficticio@cluster.invalid",
		"mongodb://127.0.0.1:1/?tls=true&tlsCertificateKeyFilePassword=clave-tls-ficticia&connectTimeoutMS=200",
		"mongodb://127.0.0.1:1/?tlsCertificateKeyFilePassword=clave-tls-ficticia&tlsInsecure=nada",
		"mongodb://127.0.0.1:1/?authMechanism=MONGODB-AWS&authMechanismProperties=AWS_SESSION_TOKEN:token-ficticio",
	} {
		ctx, cancelar := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		a, err := Conectar(ctx, uri, "cc65_test")
		cancelar()
		if err == nil {
			_ = a.Cerrar(context.Background())
			t.Fatalf("%s: se esperaba error", uri)
		}
		for _, m := range marcadores {
			if strings.Contains(err.Error(), m) {
				t.Fatalf("el error expone %q: %s", m, err)
			}
		}
	}
}

func TestDestino(t *testing.T) {
	cases := map[string]string{
		"mongodb://u:p@h:27017/db?x=1":                     "mongodb://h:27017",
		"mongodb://h1:27017,h2:27018/?replicaSet=rs":       "mongodb://h1:27017,h2:27018",
		"mongodb+srv://solo@cluster.example/db":            "mongodb+srv://cluster.example",
		"mongodb://h?tlsCertificateKeyFilePassword=secret": "mongodb://h",
		"no-es-uri": "(URI no válida)",
	}
	for in, want := range cases {
		if got := destino(in); got != want {
			t.Errorf("destino(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

func TestLimpiarQuitaValoresDeLaConsulta(t *testing.T) {
	uri := "mongodb://u:clave-larga@h/?tlsCertificateKeyFilePassword=p%40ss-marcador&appName=app"
	err := errors.New("fallo con " + uri + " y la opción p@ss-marcador y clave-larga")
	msg := limpiar(err, uri).Error()
	for _, s := range []string{"p%40ss-marcador", "p@ss-marcador", "clave-larga"} {
		if strings.Contains(msg, s) {
			t.Fatalf("quedó %q en %q", s, msg)
		}
	}
}
