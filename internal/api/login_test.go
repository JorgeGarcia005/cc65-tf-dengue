package api

import (
	"bytes"
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// hashRapido arma un hash con el formato de auth.Hash pero con pocas
// iteraciones, para que los tests con -race no tarden segundos por login.
func hashRapido(t *testing.T, clave string) string {
	t.Helper()
	sal := bytes.Repeat([]byte{7}, 16)
	k, err := pbkdf2.Key(sha256.New, clave, sal, 1000, 32)
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha256$1000$%s$%s", b64.EncodeToString(sal), b64.EncodeToString(k))
}

const secretoLogin = "secreto-de-prueba-de-32-bytes-ok!"

// servidorLogin arma la API con dos cuentas y devuelve el servidor, la URL de
// login y el firmador para verificar los tokens.
func servidorLogin(t *testing.T, logs *bytes.Buffer) (*Servidor, string, *auth.Firmador) {
	t.Helper()
	f, err := auth.NuevoFirmador([]byte(secretoLogin), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	mem := almacen.NuevaMemoria()
	for _, u := range []almacen.Usuario{
		{Correo: "admin@vigia.pe", Hash: hashRapido(t, "clave-del-admin"), Rol: "admin"},
		{Correo: "ana@diresa.pe", Hash: hashRapido(t, "clave-de-ana-123"), Rol: "analista"},
	} {
		if err := mem.GuardarUsuario(context.Background(), u); err != nil {
			t.Fatal(err)
		}
	}
	op := Opciones{Almacen: mem, Firmador: f}
	if logs != nil {
		var mu sync.Mutex
		op.Log = func(formato string, args ...any) {
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintf(logs, formato+"\n", args...)
		}
	}
	s, ts := servidorDePrueba(t, op)
	return s, ts.URL + "/api/v1/auth/login", f
}

func postLogin(t *testing.T, url, cuerpo string) (int, http.Header, []byte) {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(cuerpo))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header, b
}

func TestLoginCorrectoDevuelveTokenVerificable(t *testing.T) {
	_, url, f := servidorLogin(t, nil)
	codigo, cab, cuerpo := postLogin(t, url, `{"correo":"  ANA@diresa.PE ","clave":"clave-de-ana-123"}`)
	if codigo != 200 || cab.Get("Cache-Control") != "no-store" {
		t.Fatalf("código %d, Cache-Control %q: %s", codigo, cab.Get("Cache-Control"), cuerpo)
	}
	var r RespuestaLogin
	if err := json.Unmarshal(cuerpo, &r); err != nil {
		t.Fatal(err)
	}
	rec, err := f.Verificar(r.Token)
	if err != nil {
		t.Fatalf("el token no verifica: %v", err)
	}
	if r.Correo != "ana@diresa.pe" || r.Rol != "analista" || rec.Sub != r.Correo || rec.Exp != r.Expira {
		t.Fatalf("respuesta %+v, reclamos %+v", r, rec)
	}
}

func TestLoginCredencialesIncorrectasMismoMensaje(t *testing.T) {
	_, url, _ := servidorLogin(t, nil)
	c1, _, b1 := postLogin(t, url, `{"correo":"ana@diresa.pe","clave":"otra-clave"}`)
	c2, _, b2 := postLogin(t, url, `{"correo":"nadie@diresa.pe","clave":"otra-clave"}`) // usa el hash señuelo
	if c1 != 401 || c2 != 401 || !bytes.Equal(b1, b2) {
		t.Fatalf("clave errada %d %s · correo inexistente %d %s", c1, b1, c2, b2)
	}
	if !strings.Contains(string(b1), MensajeCredenciales) {
		t.Fatalf("mensaje inesperado: %s", b1)
	}
}

func TestLoginRateLimit(t *testing.T) {
	s, url, _ := servidorLogin(t, nil)
	var mu sync.Mutex
	reloj := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	s.limite.ahora = func() time.Time { mu.Lock(); defer mu.Unlock(); return reloj }
	avanzar := func(d time.Duration) { mu.Lock(); reloj = reloj.Add(d); mu.Unlock() }
	malo := `{"correo":"ana@diresa.pe","clave":"mala"}`
	for i := 1; i <= MaxFallosLogin; i++ {
		if c, _, b := postLogin(t, url, malo); c != 401 {
			t.Fatalf("intento %d: %d %s", i, c, b)
		}
		avanzar(time.Second)
	}
	c, cab, _ := postLogin(t, url, `{"correo":"ana@diresa.pe","clave":"clave-de-ana-123"}`)
	// Fallos en t = 0…4 s; el 6.º intento llega en t = 5 s y el primer fallo
	// sale de la ventana en t = 60 s: faltan 55 s.
	if c != 429 || cab.Get("Retry-After") != "55" {
		t.Fatalf("6.º intento: %d, Retry-After %q", c, cab.Get("Retry-After"))
	}
	avanzar(55 * time.Second)
	if c, _, b := postLogin(t, url, `{"correo":"ana@diresa.pe","clave":"clave-de-ana-123"}`); c != 200 {
		t.Fatalf("tras la ventana el login correcto debía pasar: %d %s", c, b)
	}
	if bloqueado, _ := s.limite.Bloqueado("127.0.0.1"); bloqueado {
		t.Fatal("un login correcto debe borrar los fallos de la IP")
	}
}

func TestLoginSinHashEnRespuestasNiLogs(t *testing.T) {
	var logs bytes.Buffer
	_, url, _ := servidorLogin(t, &logs)
	var todo []byte
	for _, c := range []string{`{"correo":"admin@vigia.pe","clave":"clave-del-admin"}`,
		`{"correo":"admin@vigia.pe","clave":"mala"}`, `{"correo":"x"}`} {
		_, _, b := postLogin(t, url, c)
		todo = append(todo, b...)
	}
	todo = append(todo, logs.Bytes()...)
	if bytes.Contains(todo, []byte("pbkdf2")) || bytes.Contains(todo, []byte(`"hash"`)) {
		t.Fatalf("el hash se filtró: %s", todo)
	}
}

func TestLoginValidacion(t *testing.T) {
	_, url, _ := servidorLogin(t, nil)
	casos := map[string]int{
		``:                               400,
		`{"correo":"","clave":"x"}`:      400,
		`{"correo":"a@b.pe","clave":""}`: 400,
		`{"correo":"a@b.pe","clave":"x","rol":"admin"}`:                              400, // campo desconocido
		`{"correo":"a@b.pe","clave":"` + strings.Repeat("x", maxLargoClave+1) + `"}`: 400,
		`{"correo":"` + strings.Repeat("a", maxLargoCorreo+1) + `","clave":"x"}`:     400,
		`{"correo":"a@b.pe","clave":"x"}{}`:                                          400,
	}
	for cuerpo, quiere := range casos {
		if c, _, b := postLogin(t, url, cuerpo); c != quiere {
			t.Errorf("%.40q → %d %s, quería %d", cuerpo, c, b, quiere)
		}
	}
}

func TestLoginSinFirmadorDa503(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	if c, _, b := postLogin(t, ts.URL+"/api/v1/auth/login", `{"correo":"a@b.pe","clave":"x"}`); c != 503 {
		t.Fatalf("%d %s", c, b)
	}
}

func TestLimitadorPorIPYPoda(t *testing.T) {
	l := nuevoLimitador()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ahora := t0
	l.ahora = func() time.Time { return ahora }
	for range MaxFallosLogin {
		l.Fallo("10.0.0.1")
	}
	if b, _ := l.Bloqueado("10.0.0.1"); !b {
		t.Fatal("10.0.0.1 debía quedar bloqueada")
	}
	if b, _ := l.Bloqueado("10.0.0.2"); b {
		t.Fatal("otra IP no debe quedar bloqueada")
	}
	for i := range maxIPsAntesPoda {
		l.Fallo(fmt.Sprintf("192.168.%d.%d", i/256, i%256))
	}
	ahora = t0.Add(2 * VentanaFallos) // todo vencido
	l.Fallo("10.0.0.9")               // dispara la poda
	if n := len(l.fallos); n != 1 {
		t.Fatalf("quedaron %d IPs tras la poda, quería 1", n)
	}
}

func TestIPCliente(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "203.0.113.7:51234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if ip := ipCliente(r); ip != "203.0.113.7" {
		t.Fatalf("ip = %q (no debe confiar en X-Forwarded-For)", ip)
	}
	r.RemoteAddr = "[2001:db8::1]:443"
	if ip := ipCliente(r); ip != "2001:db8::1" {
		t.Fatalf("ip = %q", ip)
	}
}

func TestSembrarUsuarios(t *testing.T) {
	ctx := context.Background()
	entorno := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	mem := almacen.NuevaMemoria()
	sembrados, err := SembrarUsuarios(ctx, mem, entorno(map[string]string{
		"VIGIA_ADMIN_CORREO": "Admin@Vigia.pe", "VIGIA_ADMIN_CLAVE": "una-clave-larga",
		"VIGIA_ANALISTA_CORREO": "ana@diresa.pe", "VIGIA_ANALISTA_CLAVE": "otra-clave-larga",
	}))
	if err != nil || len(sembrados) != 2 {
		t.Fatalf("sembrados %v, err %v", sembrados, err)
	}
	u, ok, _ := mem.Usuario(ctx, "admin@vigia.pe")
	if !ok || u.Rol != "admin" || !auth.Comparar(u.Hash, "una-clave-larga") {
		t.Fatalf("admin sembrado mal: %+v", u)
	}
	if s, err := SembrarUsuarios(ctx, mem, entorno(nil)); err != nil || len(s) != 0 {
		t.Fatalf("sin variables: %v %v", s, err)
	}
	for _, m := range []map[string]string{
		{"VIGIA_ADMIN_CORREO": "a@b.pe", "VIGIA_ADMIN_CLAVE": "corta"},
		{"VIGIA_ADMIN_CORREO": "a@b.pe"},
		{"VIGIA_ANALISTA_CLAVE": "una-clave-larga"},
	} {
		if _, err := SembrarUsuarios(ctx, mem, entorno(m)); err == nil {
			t.Errorf("%v debía fallar", m)
		}
	}
}
