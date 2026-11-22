package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dist(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for nombre, cuerpo := range map[string]string{
		"index.html":      "<!doctype html><title>Vigía</title>",
		"assets/app-1.js": "console.log(1)",
		"favicon.svg":     "<svg/>",
		"config.json":     `{"api_url":"http://desarrollo"}`, // el de public/ no debe ganar
	} {
		if err := os.WriteFile(filepath.Join(d, filepath.FromSlash(nombre)), []byte(cuerpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(d), "secreto.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

func pedir(t *testing.T, h http.Handler, metodo, ruta string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(metodo, ruta, nil))
	return w
}

func TestCabecerasYCSP(t *testing.T) {
	h, err := nuevoServidor(dist(t), "https://api.vigia.pe")
	if err != nil {
		t.Fatal(err)
	}
	w := pedir(t, h, "GET", "/")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Vigía") {
		t.Fatalf("/ = %d %q", w.Code, w.Body)
	}
	csp := w.Header().Get("Content-Security-Policy")
	for _, d := range []string{"default-src 'self'", "script-src 'self'", "style-src 'self'",
		"connect-src 'self' https://api.vigia.pe wss://api.vigia.pe", "frame-ancestors 'none'", "object-src 'none'"} {
		if !strings.Contains(csp, d) {
			t.Errorf("CSP sin %q: %s", d, csp)
		}
	}
	if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
		t.Errorf("CSP con unsafe-*: %s", csp)
	}
	for k, v := range map[string]string{"X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer",
		"X-Frame-Options": "DENY", "Cache-Control": "no-cache"} {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q, quería %q", k, got, v)
		}
	}
}

func TestConfigJSON(t *testing.T) {
	h, _ := nuevoServidor(dist(t), "http://localhost:8080/")
	w := pedir(t, h, "GET", "/config.json")
	if w.Body.String() != `{"api_url":"http://localhost:8080"}` || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("config = %q (%s)", w.Body, w.Header().Get("Cache-Control"))
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "ws://localhost:8080") {
		t.Fatalf("CSP sin ws://: %s", csp)
	}
}

func TestRutasDelSPA(t *testing.T) {
	h, _ := nuevoServidor(dist(t), "http://localhost:8080")
	casos := []struct {
		metodo, ruta string
		codigo       int
		contiene     string
	}{
		{"GET", "/mapa", 200, "Vigía"},
		{"GET", "/admin/nodos", 200, "Vigía"},
		{"GET", "/assets/app-1.js", 200, "console.log"},
		{"GET", "/assets/no-existe.js", 404, ""},
		{"GET", "/favicon.svg", 200, "<svg/>"},
		{"GET", "/../secreto.txt", 404, ""},
		{"GET", "/assets", 200, "Vigía"}, // carpeta: no lista archivos
		{"HEAD", "/", 200, ""},
		{"POST", "/", 405, ""},
	}
	for _, c := range casos {
		w := pedir(t, h, c.metodo, c.ruta)
		if w.Code != c.codigo || !strings.Contains(w.Body.String(), c.contiene) {
			t.Errorf("%s %s = %d %q, quería %d con %q", c.metodo, c.ruta, w.Code, w.Body, c.codigo, c.contiene)
		}
	}
	if cc := pedir(t, h, "GET", "/assets/app-1.js").Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("assets sin caché inmutable: %q", cc)
	}
	if strings.Contains(pedir(t, h, "GET", "/assets/").Body.String(), "app-1.js") {
		t.Error("lista el contenido de assets/")
	}
}

func TestConfiguracionInvalida(t *testing.T) {
	d := dist(t)
	for _, u := range []string{"", "localhost:8080", "ftp://x", "http://", "http://api/v1"} {
		if _, err := nuevoServidor(d, u); err == nil {
			t.Errorf("API_URL %q debía fallar", u)
		}
	}
	if _, err := nuevoServidor(t.TempDir(), "http://localhost:8080"); err == nil {
		t.Error("sin index.html debía fallar")
	}
}
