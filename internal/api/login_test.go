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
	// El login correcto borró los fallos de t = 1…4 s. Sin ese borrado, uno más
	// sumaría 5 y el siguiente intento ya recibiría 429.
	if c, _, b := postLogin(t, url, malo); c != 401 {
		t.Fatalf("fallo tras el login correcto: %d %s", c, b)
	}
	if c, _, b := postLogin(t, url, malo); c != 401 {
		t.Fatalf("un login correcto debe borrar los fallos de la IP: %d %s", c, b)
	}
}

// PR-80 H1: una ráfaga concurrente no puede saltarse el límite mientras corre
// PBKDF2. De 20 intentos simultáneos con clave errada desde la misma IP, solo
// MaxFallosLogin llegan a comparar la clave (401); el resto recibe 429.
func TestLoginRafagaConcurrente(t *testing.T) {
	_, url, _ := servidorLogin(t, nil)
	const n = 20
	var wg sync.WaitGroup
	codigos := make(chan int, n)
	listos := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-listos
			res, err := http.Post(url, "application/json", strings.NewReader(`{"correo":"ana@diresa.pe","clave":"mala"}`))
			if err != nil {
				t.Error(err)
				return
			}
			res.Body.Close()
			if res.StatusCode == 429 && res.Header.Get("Retry-After") == "" {
				t.Error("429 sin Retry-After")
			}
			codigos <- res.StatusCode
		}()
	}
	close(listos)
	wg.Wait()
	close(codigos)
	cuenta := map[int]int{}
	for c := range codigos {
		cuenta[c]++
	}
	if cuenta[401] != MaxFallosLogin || cuenta[429] != n-MaxFallosLogin {
		t.Fatalf("respuestas %v: quería %d × 401 y %d × 429", cuenta, MaxFallosLogin, n-MaxFallosLogin)
	}
}

type sinUsuarios struct{ almacen.Almacen } // no implementa almacen.Usuarios

func TestLoginSinUsuariosDa503(t *testing.T) {
	f, _ := auth.NuevoFirmador([]byte(secretoLogin), time.Hour)
	_, ts := servidorDePrueba(t, Opciones{Firmador: f, Almacen: sinUsuarios{almacen.NuevaMemoria()}})
	c, _, b := postLogin(t, ts.URL+"/api/v1/auth/login", `{"correo":"a@b.pe","clave":"x"}`)
	if c != 503 || !strings.Contains(string(b), "almacén de usuarios") {
		t.Fatalf("%d %s", c, b)
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
		`{"correo":"a@b.pe","clave":"` + strings.Repeat("x", MaxBytesClave+1) + `"}`: 400,
		`{"correo":"` + strings.Repeat("a", MaxBytesCorreo+1) + `","clave":"x"}`:     400,
		`{"correo":"a@b.pe","clave":"x"}{}`:                                          400,
	}
	for cuerpo, quiere := range casos {
		if c, _, b := postLogin(t, url, cuerpo); c != quiere {
			t.Errorf("%.40q → %d %s, quería %d", cuerpo, c, b, quiere)
		}
	}
}

func TestLoginSinFirmadorDa503(t *testing.T) {
	_, h := servidorSeguro(t, Opciones{})
	ts := httptest.NewServer(h)
	defer ts.Close()
	if c, _, b := postLogin(t, ts.URL+"/api/v1/auth/login", `{"correo":"a@b.pe","clave":"x"}`); c != 503 {
		t.Fatalf("%d %s", c, b)
	}
}

func fallar(t *testing.T, l *limitador, ip string) {
	t.Helper()
	liberar, _ := l.Reservar(ip)
	if liberar == nil {
		t.Fatalf("%s bloqueada antes de tiempo", ip)
	}
	liberar(intentoFallido)
}

func TestLimitadorPorIP(t *testing.T) {
	l := nuevoLimitador()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.ahora = func() time.Time { return t0 }
	for range MaxFallosLogin {
		fallar(t, l, "10.0.0.1")
	}
	if liberar, espera := l.Reservar("10.0.0.1"); liberar != nil || espera != VentanaFallos {
		t.Fatalf("10.0.0.1 debía quedar bloqueada un minuto: espera %v", espera)
	}
	if liberar, _ := l.Reservar("10.0.0.2"); liberar == nil {
		t.Fatal("otra IP no debe quedar bloqueada")
	}
}

func TestLimitadorCuentaIntentosEnCurso(t *testing.T) {
	l := nuevoLimitador()
	var reservas []func(resultadoIntento)
	for range MaxFallosLogin {
		liberar, _ := l.Reservar("10.0.0.1")
		if liberar == nil {
			t.Fatal("reserva rechazada antes del máximo")
		}
		reservas = append(reservas, liberar)
	}
	if liberar, espera := l.Reservar("10.0.0.1"); liberar != nil || espera != time.Second {
		t.Fatalf("con %d intentos en curso, el siguiente debía esperar 1 s: %v", MaxFallosLogin, espera)
	}
	reservas[0](intentoNeutro)
	reservas[0](intentoFallido) // la segunda liberación no hace nada
	if liberar, _ := l.Reservar("10.0.0.1"); liberar == nil {
		t.Fatal("al liberar uno como neutro debía quedar un cupo")
	}
	if n := len(l.fallos["10.0.0.1"]); n != 0 {
		t.Fatalf("un intento neutro (o liberado dos veces) no debe contar como fallo: %d", n)
	}
}

// PR-80 H2: con muchas IPs activas en la misma ventana, el mapa no pasa del
// tope y la poda no corre en cada fallo.
func TestLimitadorMemoriaAcotada(t *testing.T) {
	l := nuevoLimitador()
	l.podarDesde, l.maxIPs = 8, 16
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ahora := t0
	l.ahora = func() time.Time { return ahora }
	for i := range 100 {
		fallar(t, l, fmt.Sprintf("192.168.0.%d", i))
	}
	if n := len(l.fallos); n > 16 {
		t.Fatalf("el mapa creció a %d IPs con tope 16", n)
	}
	if !l.ultimaPoda.Equal(t0) {
		t.Fatalf("la poda debía correr una sola vez en la media ventana: última %v", l.ultimaPoda)
	}
	ahora = t0.Add(2 * VentanaFallos) // todo vencido
	fallar(t, l, "10.0.0.9")
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
		{"VIGIA_ADMIN_CORREO": "a@b.pe", "VIGIA_ADMIN_CLAVE": strings.Repeat("x", MaxBytesClave+1)},
		{"VIGIA_ADMIN_CORREO": "a@b.pe", "VIGIA_ADMIN_CLAVE": strings.Repeat("ñ", 150)}, // 150 runas, 300 bytes
		{"VIGIA_ADMIN_CORREO": strings.Repeat("a", MaxBytesCorreo) + "@b.pe", "VIGIA_ADMIN_CLAVE": "una-clave-larga"},
		{"VIGIA_ADMIN_CORREO": "Misma@b.pe", "VIGIA_ADMIN_CLAVE": "una-clave-larga",
			"VIGIA_ANALISTA_CORREO": "misma@B.pe", "VIGIA_ANALISTA_CLAVE": "otra-clave-larga"},
	} {
		if _, err := SembrarUsuarios(ctx, mem, entorno(m)); err == nil {
			t.Errorf("%v debía fallar", m)
		}
	}
}
