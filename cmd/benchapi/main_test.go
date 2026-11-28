package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

// apiDePrueba levanta la API real (internal/api) con un modelo de prueba, el
// almacén en memoria y, si conCache, una caché en memoria.
func apiDePrueba(t *testing.T, conCache bool) *httptest.Server {
	var c almacen.Cache
	if conCache {
		c = almacen.NuevaCacheMemoria(time.Now)
	}
	return apiConCache(t, c)
}

// cacheCaida simula Redis caído: toda operación falla.
type cacheCaida struct{}

func (cacheCaida) Obtener(context.Context, string) ([]byte, bool, error) {
	return nil, false, errors.New("redis caído")
}
func (cacheCaida) Guardar(context.Context, string, []byte, time.Duration) error {
	return errors.New("redis caído")
}

// secretoBench firma los tokens de las pruebas; la API de prueba exige JWT
// como la real (#61).
const secretoBench = "secreto-de-las-pruebas-de-benchapi!"

func firmadorBench(t *testing.T) *auth.Firmador {
	t.Helper()
	f, err := auth.NuevoFirmador([]byte(secretoBench), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// tokenBench es un token de analista válido para la API de prueba.
func tokenBench(t *testing.T) string {
	t.Helper()
	tok, _, err := firmadorBench(t).Firmar("analista@vigia.pe", auth.RolAnalista)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func apiConCache(t *testing.T, cache almacen.Cache) *httptest.Server {
	ts, _ := apiCompleta(t, cache)
	return ts
}

// apiCompleta levanta la API con JWT y devuelve también su almacén (para
// sembrar usuarios).
func apiCompleta(t *testing.T, cache almacen.Cache) (*httptest.Server, *almacen.Memoria) {
	t.Helper()
	var regs []ingest.Registro
	for i := range 60 {
		d := []string{"LORETO", "PIURA", "TUMBES"}[i%3]
		regs = append(regs, ingest.Registro{Departamento: d, Ano: 2015 + i%8, Semana: 1 + i%52, EdadAnios: float64(i), Sexo: 'F'})
	}
	esq, err := features.Ajustar(regs, 2011, 5)
	if err != nil {
		t.Fatal(err)
	}
	pesos := make([]float64, esq.D()+1)
	for j := range pesos {
		pesos[j] = 0.05 * float64(j%5-2)
	}
	b, _ := json.Marshal(map[string]any{"pesos": pesos, "esquema": esq, "umbral": 0.5, "ajuste_prior": 1.9, "generado": "2026-11-20T00:00:00-05:00"})
	m, err := api.CargarModelo(b)
	if err != nil {
		t.Fatal(err)
	}
	mem := almacen.NuevaMemoria()
	op := api.Opciones{Modelo: m, Cache: cache, Almacen: mem, Firmador: firmadorBench(t), Log: func(string, ...any) {}}
	s, err := api.Nuevo(op)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() { ts.Close(); s.Cerrar() })
	return ts, mem
}

func silencio(string, ...any) {}

func TestMedirSeparaCacheFriaYCaliente(t *testing.T) {
	ts := apiDePrueba(t, true)
	op := opciones{url: ts.URL, token: tokenBench(t), peticiones: 300, concurrencia: 8, calientes: 16, semilla: 7, timeout: 5 * time.Second}
	rep, err := medir(context.Background(), op, time.Second, silencio)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Corridas) != 2 || rep.Corridas[0].Nombre != "fria" || rep.Corridas[1].Nombre != "caliente" {
		t.Fatalf("corridas = %+v", rep.Corridas)
	}
	fria, caliente := rep.Corridas[0], rep.Corridas[1]
	if fria.Errores != 0 || caliente.Errores != 0 {
		t.Fatalf("errores: fría %d (%s), caliente %d (%s)", fria.Errores, fria.PrimerError, caliente.Errores, caliente.PrimerError)
	}
	if fria.Misses != 300 || fria.Hits != 0 {
		t.Fatalf("fría: %d MISS y %d HIT; se esperaban 300 y 0 (consultas distintas)", fria.Misses, fria.Hits)
	}
	if caliente.Hits != 300 || caliente.Misses != 0 {
		t.Fatalf("caliente: %d HIT y %d MISS; se esperaban 300 y 0", caliente.Hits, caliente.Misses)
	}
	for _, c := range rep.Corridas {
		if !c.ConCache || !(c.P50Ms > 0 && c.P50Ms <= c.P95Ms && c.P95Ms <= c.P99Ms && c.P99Ms <= c.MaxMs) || c.RPS <= 0 {
			t.Fatalf("%s: percentiles o RPS inválidos: %+v", c.Nombre, c)
		}
		if c.CumpleLimite == nil || !*c.CumpleLimite {
			t.Fatalf("%s: con un límite de 1 s debía cumplir: %+v", c.Nombre, c)
		}
	}
	if rep.VersionModelo == "" || rep.LimiteP95Ms != 1000 || rep.Concurrencia != 8 {
		t.Fatalf("reporte incompleto: %+v", rep)
	}
	if err := verificar(rep); err != nil {
		t.Fatal(err)
	}
}

func TestSinCacheLoReporta(t *testing.T) {
	ts := apiDePrueba(t, false)
	rep, err := medir(context.Background(), opciones{url: ts.URL, token: tokenBench(t), peticiones: 50, concurrencia: 4, calientes: 4, semilla: 1, timeout: time.Second}, 0, silencio)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Corridas {
		if c.ConCache || c.Hits+c.Misses != 0 || c.CumpleLimite != nil {
			t.Fatalf("%s: %+v", c.Nombre, c)
		}
	}
}

func TestExigirP95FallaSiNoSeCumple(t *testing.T) {
	ts := apiDePrueba(t, true)
	rep, err := medir(context.Background(), opciones{url: ts.URL, token: tokenBench(t), peticiones: 20, concurrencia: 2, calientes: 2, semilla: 3, timeout: time.Second}, time.Nanosecond, silencio)
	if err != nil {
		t.Fatal(err)
	}
	err = verificar(rep)
	if err == nil || !strings.Contains(err.Error(), "no es menor que") {
		t.Fatalf("err = %v, se esperaba el incumplimiento del p95", err)
	}
}

func TestErroresDelServidorSeCuentanYFallan(t *testing.T) {
	ts := apiDePrueba(t, true)
	var n atomic.Int64
	falla := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/predicciones" {
			if n.Add(1)%2 == 0 {
				http.Error(w, `{"error":"falla de prueba"}`, http.StatusInternalServerError)
				return
			}
		}
		// El resto lo atiende la API real.
		req, _ := http.NewRequest(r.Method, ts.URL+r.URL.Path, r.Body)
		req.Header = r.Header
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		for k, v := range res.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(res.StatusCode)
		var buf [4096]byte
		for {
			m, err := res.Body.Read(buf[:])
			w.Write(buf[:m])
			if err != nil {
				return
			}
		}
	}))
	t.Cleanup(falla.Close)
	rep, err := medir(context.Background(), opciones{url: falla.URL, token: tokenBench(t), peticiones: 10, concurrencia: 1, calientes: 1, semilla: 5, timeout: time.Second}, 0, silencio)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Corridas[0].Errores == 0 || !strings.Contains(rep.Corridas[0].PrimerError, "HTTP 500") {
		t.Fatalf("fría: %+v", rep.Corridas[0])
	}
	if err := verificar(rep); err == nil {
		t.Fatal("verificar debía fallar con errores")
	}
}

func TestGeneradorNoRepiteConsultas(t *testing.T) {
	g := generador{deptos: []string{"A", "B", "C"}, anoMin: 2011, base: 123456789}
	vistas := map[string]bool{}
	for i := range 20_000 {
		k := string(g.fria(i))
		if vistas[k] {
			t.Fatalf("consulta fría %d repetida: %s", i, k)
		}
		vistas[k] = true
	}
	for i := range 64 {
		if vistas[string(g.caliente(i))] {
			t.Fatalf("la consulta caliente %d coincide con una fría", i)
		}
	}
	g2 := generador{deptos: []string{"A"}, anoMin: 2020}
	var c struct{ Ano int }
	for i := range 1000 {
		json.Unmarshal(g2.fria(i), &c)
		if c.Ano < 2020 || c.Ano > anoMax {
			t.Fatalf("año %d fuera de 2020–%d", c.Ano, anoMax)
		}
	}
}

// La colección de Postman cubre cada ruta documentada de la API y comprueba
// el estado y que responda en menos de 100 ms (issue #30).
func TestColeccionPostmanCubreLasRutas(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "cc65-tf.postman_collection.json"))
	if err != nil {
		t.Fatal(err)
	}
	var col struct {
		Item []struct {
			Name    string `json:"name"`
			Request struct {
				Method string `json:"method"`
				URL    struct {
					Raw string `json:"raw"`
				} `json:"url"`
			} `json:"request"`
			Event []struct {
				Listen string `json:"listen"`
				Script struct {
					Exec []string `json:"exec"`
				} `json:"script"`
			} `json:"event"`
		} `json:"item"`
	}
	if err := json.Unmarshal(b, &col); err != nil {
		t.Fatalf("la colección no es JSON válido: %v", err)
	}
	variable := regexp.MustCompile(`\{\{[a-z_]+\}\}`)
	parametro := regexp.MustCompile(`\{[a-z_]+\}`) // {id}, {nombre}: la variable de Postman ocupa su lugar
	tiene := map[string]bool{}
	for _, it := range col.Item {
		ruta := strings.TrimPrefix(it.Request.URL.Raw, "{{base_url}}")
		ruta = variable.ReplaceAllString(ruta, "{}")
		tiene[it.Request.Method+" "+ruta] = true
		var tests string
		for _, e := range it.Event {
			if e.Listen == "test" {
				tests += strings.Join(e.Script.Exec, "\n")
			}
		}
		// El login deriva la clave con PBKDF2 (600 000 iteraciones, ADR-0007): es
		// lento a propósito y tiene su propio tope; el de 100 ms es de la API de datos.
		limite := "responseTime).to.be.below(100)"
		if strings.HasSuffix(ruta, "/auth/login") {
			limite = "responseTime).to.be.below(1000)"
		}
		if !strings.Contains(tests, "pm.response.to.have.status") || !strings.Contains(tests, limite) {
			t.Errorf("%s: falta el test de estado o de tiempo (%s)", it.Name, limite)
		}
	}
	for _, r := range api.Rutas() {
		if r.Documentar && !tiene[r.Metodo+" "+parametro.ReplaceAllString(r.Ruta, "{}")] {
			t.Errorf("la colección no tiene %s %s", r.Metodo, r.Ruta)
		}
	}
}

// PR-41 H1: con Redis caído la API responde MISS en vez de HIT; la corrida
// caliente no puede darse por buena.
func TestCacheCaidaFallaLaCorridaCaliente(t *testing.T) {
	ts := apiConCache(t, cacheCaida{})
	rep, err := medir(context.Background(), opciones{url: ts.URL, token: tokenBench(t), peticiones: 40, concurrencia: 4, calientes: 4, semilla: 9, timeout: time.Second}, time.Second, silencio)
	if err != nil {
		t.Fatal(err)
	}
	if err := verificar(rep); err == nil || !strings.Contains(err.Error(), "corrida caliente") {
		t.Fatalf("err = %v, se esperaba el fallo de la corrida caliente", err)
	}
}

// PR-41 H1: repetir la semilla contra la misma caché hace que la corrida fría
// salga de la caché; debe fallar.
func TestSemillaRepetidaFallaLaCorridaFria(t *testing.T) {
	ts := apiDePrueba(t, true)
	op := opciones{url: ts.URL, token: tokenBench(t), peticiones: 40, concurrencia: 4, calientes: 4, semilla: 11, timeout: time.Second}
	if _, err := medir(context.Background(), op, 0, silencio); err != nil {
		t.Fatal(err)
	}
	rep, err := medir(context.Background(), op, 0, silencio)
	if err != nil {
		t.Fatal(err)
	}
	if err := verificar(rep); err == nil || !strings.Contains(err.Error(), "corrida fria") {
		t.Fatalf("err = %v, se esperaba el fallo de la corrida fría", err)
	}
}

// PR-41 H1 y H3: opciones que solaparían la fría con la caliente o que no
// tienen sentido se rechazan antes de medir.
func TestOpcionesInvalidasSeRechazan(t *testing.T) {
	ts := apiDePrueba(t, true)
	base := opciones{url: ts.URL, token: tokenBench(t), peticiones: 10, concurrencia: 2, calientes: 2, semilla: 1, timeout: time.Second}
	casos := map[string]func(*opciones) time.Duration{
		"n mayor que la mitad": func(o *opciones) time.Duration { o.peticiones = 10_000_000; return 0 },
		"timeout cero":         func(o *opciones) time.Duration { o.timeout = 0; return 0 },
		"límite negativo":      func(o *opciones) time.Duration { return -time.Millisecond },
	}
	for nombre, f := range casos {
		op := base
		limite := f(&op)
		if _, err := medir(context.Background(), op, limite, silencio); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}

// #61: con -correo y $BENCH_CLAVE inicia sesión y mide con el token; sin
// token la API responde 401 y benchapi lo explica en vez de medir errores.
func TestMedirConLoginUsaJWT(t *testing.T) {
	ts, mem := apiCompleta(t, almacen.NuevaCacheMemoria(time.Now))
	hash, err := auth.Hash("clave-del-benchmark")
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.GuardarUsuario(context.Background(), almacen.Usuario{Correo: "bench@vigia.pe", Hash: hash, Rol: "analista"}); err != nil {
		t.Fatal(err)
	}
	// 20 s: el login deriva PBKDF2 de 600 000 iteraciones, que con -race y 2 vCPU
	// ocupados por otros paquetes pasó de 5 s (auditoría TB2 B4). Es el plazo de
	// red del test, no el límite de 100 ms de la API (lo mide verificar).
	op := opciones{url: ts.URL, peticiones: 40, concurrencia: 4, calientes: 4, semilla: 13, timeout: 20 * time.Second,
		correo: "bench@vigia.pe", clave: "clave-del-benchmark"}
	rep, err := medir(context.Background(), op, time.Second, silencio)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Autenticacion != "jwt" || verificar(rep) != nil {
		t.Fatalf("autenticación %q, verificar: %v", rep.Autenticacion, verificar(rep))
	}
	for _, c := range rep.Corridas {
		if c.Errores != 0 {
			t.Fatalf("corrida %s con %d errores: %s", c.Nombre, c.Errores, c.PrimerError)
		}
	}

	op.clave = "otra-clave-cualquiera"
	if _, err := medir(context.Background(), op, 0, silencio); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("clave errada: err %v, quería el 401 del login", err)
	}
	op.correo, op.clave = "", ""
	if _, err := medir(context.Background(), op, 0, silencio); err == nil || !strings.Contains(err.Error(), "exige token") {
		t.Fatalf("sin token: err %v, quería el aviso de que la API exige token", err)
	}
	op.correo = "bench@vigia.pe"
	if _, err := medir(context.Background(), op, 0, silencio); err == nil || !strings.Contains(err.Error(), "BENCH_CLAVE") {
		t.Fatalf("correo sin clave: err %v", err)
	}
	op.token = tokenBench(t)
	if _, err := medir(context.Background(), op, 0, silencio); err == nil || !strings.Contains(err.Error(), "no ambos") {
		t.Fatalf("-token y -correo juntos: err %v", err)
	}
}
