package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var secretoPrueba = []byte("0123456789abcdef0123456789abcdef") // 32 bytes

func firmadorFijo(t *testing.T, ahora time.Time) *Firmador {
	t.Helper()
	f, err := NuevoFirmador(secretoPrueba, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	f.ahora = func() time.Time { return ahora }
	return f
}

// armar construye un token con cabecera y reclamos arbitrarios, firmado con secreto.
func armar(t *testing.T, cabecera map[string]any, rec map[string]any, secreto []byte) string {
	t.Helper()
	b64 := base64.RawURLEncoding
	h, _ := json.Marshal(cabecera)
	p, _ := json.Marshal(rec)
	datos := b64.EncodeToString(h) + "." + b64.EncodeToString(p)
	m := hmac.New(sha256.New, secreto)
	m.Write([]byte(datos))
	return datos + "." + b64.EncodeToString(m.Sum(nil))
}

func TestJWTIdaYVuelta(t *testing.T) {
	ahora := time.Unix(1_800_000_000, 0)
	f := firmadorFijo(t, ahora)
	tok, rec, err := f.Firmar("ana@diresa.pe", RolAnalista)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(tok, ".") != 2 {
		t.Fatalf("token sin tres partes: %q", tok)
	}
	if rec.Exp != ahora.Add(time.Hour).Unix() || rec.Iss != Emisor || rec.Aud != Audiencia {
		t.Fatalf("reclamos inesperados: %+v", rec)
	}
	got, err := f.Verificar(tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != rec {
		t.Fatalf("verificar = %+v, quería %+v", got, rec)
	}
}

func TestJWTRechazos(t *testing.T) {
	ahora := time.Unix(1_800_000_000, 0)
	f := firmadorFijo(t, ahora)
	valido, _, _ := f.Firmar("admin@vigia.pe", RolAdmin)
	cab := map[string]any{"alg": "HS256", "typ": "JWT"}
	rec := func(cambios map[string]any) map[string]any {
		r := map[string]any{"sub": "x@y.pe", "rol": "admin", "iss": Emisor, "aud": Audiencia,
			"iat": ahora.Unix(), "nbf": ahora.Unix(), "exp": ahora.Add(time.Hour).Unix()}
		for k, v := range cambios {
			r[k] = v
		}
		return r
	}
	partes := strings.Split(valido, ".")
	alterado := partes[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"x","rol":"admin"}`)) + "." + partes[2]

	casos := []struct {
		nombre string
		token  string
		quiere error
	}{
		{"vacío", "", ErrMalformado},
		{"dos partes", "a.b", ErrMalformado},
		{"base64 inválido", "@@.@@.@@", ErrMalformado},
		{"carga alterada", alterado, ErrFirma},
		{"otro secreto", armar(t, cab, rec(nil), []byte("otro-secreto-de-32-bytes-xxxxxxx")), ErrFirma},
		{"alg none", armar(t, map[string]any{"alg": "none", "typ": "JWT"}, rec(nil), secretoPrueba), ErrAlgoritmo},
		{"alg HS512", armar(t, map[string]any{"alg": "HS512", "typ": "JWT"}, rec(nil), secretoPrueba), ErrAlgoritmo},
		{"vencido", armar(t, cab, rec(map[string]any{"exp": ahora.Unix()}), secretoPrueba), ErrVencido},
		{"nbf futuro", armar(t, cab, rec(map[string]any{"nbf": ahora.Add(time.Minute).Unix()}), secretoPrueba), ErrNoVigente},
		{"aud ajena", armar(t, cab, rec(map[string]any{"aud": "otra"}), secretoPrueba), ErrReclamos},
		{"iss ajeno", armar(t, cab, rec(map[string]any{"iss": "otro"}), secretoPrueba), ErrReclamos},
		{"rol desconocido", armar(t, cab, rec(map[string]any{"rol": "root"}), secretoPrueba), ErrReclamos},
		{"sin sub", armar(t, cab, rec(map[string]any{"sub": ""}), secretoPrueba), ErrReclamos},
		{"campo extra desconocido no rompe", armar(t, cab, rec(map[string]any{"x": 1}), secretoPrueba), nil},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := f.Verificar(c.token)
			if c.quiere == nil {
				if err != nil {
					t.Fatalf("error inesperado: %v", err)
				}
				return
			}
			if !errors.Is(err, c.quiere) {
				t.Fatalf("error = %v, quería %v", err, c.quiere)
			}
		})
	}
}

func TestJWTSecretoCorto(t *testing.T) {
	if _, err := NuevoFirmador([]byte("corto"), time.Hour); !errors.Is(err, ErrSecretoCorto) {
		t.Fatalf("error = %v, quería ErrSecretoCorto", err)
	}
	if _, err := NuevoFirmador(secretoPrueba, 0); err == nil {
		t.Fatal("duración 0 debería fallar")
	}
}

func TestJWTFirmarRolInvalido(t *testing.T) {
	f := firmadorFijo(t, time.Unix(1_800_000_000, 0))
	if _, _, err := f.Firmar("x@y.pe", Rol("root")); !errors.Is(err, ErrReclamos) {
		t.Fatalf("error = %v, quería ErrReclamos", err)
	}
}
