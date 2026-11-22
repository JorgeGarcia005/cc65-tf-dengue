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

func apiConCache(t *testing.T, cache almacen.Cache) *httptest.Server {
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
	op := api.Opciones{Modelo: m, Cache: cache, Log: func(string, ...any) {}}
	s, err := api.Nuevo(op)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() { ts.Close(); s.Cerrar() })
	return ts
}

func silencio(string, ...any) {}

func TestMedirSeparaCacheFriaYCaliente(t *testing.T) {
	ts := apiDePrueba(t, true)
	op := opciones{url: ts.URL, peticiones: 300, concurrencia: 8, calientes: 16, semilla: 7, timeout: 5 * time.Second}
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
	rep, err := medir(context.Background(), opciones{url: ts.URL, peticiones: 50, concurrencia: 4, calientes: 4, semilla: 1, timeout: time.Second}, 0, silencio)
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
	rep, err := medir(context.Background(), opciones{url: ts.URL, peticiones: 20, concurrencia: 2, calientes: 2, semilla: 3, timeout: time.Second}, time.Nanosecond, silencio)
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
	rep, err := medir(context.Background(), opciones{url: falla.URL, peticiones: 10, concurrencia: 1, calientes: 1, semilla: 5, timeout: time.Second}, 0, silencio)
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
	tiene := map[string]bool{}
	for _, it := range col.Item {
		ruta := strings.TrimPrefix(it.Request.URL.Raw, "{{base_url}}")
		ruta = variable.ReplaceAllString(ruta, "{id}")
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
		if r.Documentar && !tiene[r.Metodo+" "+r.Ruta] {
			t.Errorf("la colección no tiene %s %s", r.Metodo, r.Ruta)
		}
	}
}

// PR-41 H1: con Redis caído la API responde MISS en vez de HIT; la corrida
// caliente no puede darse por buena.
func TestCacheCaidaFallaLaCorridaCaliente(t *testing.T) {
	ts := apiConCache(t, cacheCaida{})
	rep, err := medir(context.Background(), opciones{url: ts.URL, peticiones: 40, concurrencia: 4, calientes: 4, semilla: 9, timeout: time.Second}, time.Second, silencio)
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
	op := opciones{url: ts.URL, peticiones: 40, concurrencia: 4, calientes: 4, semilla: 11, timeout: time.Second}
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
	base := opciones{url: ts.URL, peticiones: 10, concurrencia: 2, calientes: 2, semilla: 1, timeout: time.Second}
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
