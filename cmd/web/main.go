// Command web sirve la SPA Vigía (web/dist) con la biblioteca estándar
// (ADR-0007): cabeceras de seguridad con una CSP estricta, /config.json con la
// URL de la API decidida al desplegar y vuelta a index.html para las rutas del
// SPA. Corre en su propia imagen y en otro origen que la API, así que la política
// CORS del coordinador se ejercita de verdad.
//
//	API_URL=http://localhost:8080 go run ./cmd/web -dir web/dist
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	dir := flag.String("dir", "/web", "carpeta con el build de la SPA (index.html y assets/)")
	escucha := flag.String("escucha", ":8081", "dirección HTTP")
	flag.Parse()
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	h, err := nuevoServidor(*dir, apiURL)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Addr: *escucha, Handler: h, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	go func() {
		<-ctx.Done()
		c, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = srv.Shutdown(c)
	}()
	log.Printf("web: sirviendo %s en %s (API %s)", *dir, *escucha, apiURL)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// nuevoServidor valida la carpeta y la URL de la API y arma el handler.
func nuevoServidor(dir, apiURL string) (http.Handler, error) {
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return nil, fmt.Errorf("web: falta index.html en %s: %w", dir, err)
	}
	u, err := url.Parse(apiURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" && u.Path != "/" {
		return nil, fmt.Errorf("web: API_URL %q debe ser http(s)://host[:puerto]", apiURL)
	}
	origenAPI := u.Scheme + "://" + u.Host
	origenWS := strings.Replace(origenAPI, "http", "ws", 1) // http→ws, https→wss
	config, _ := json.Marshal(map[string]string{"api_url": origenAPI})
	csp := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src 'self' " + origenAPI + " " + origenWS,
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
	archivos := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cab := w.Header()
		cab.Set("Content-Security-Policy", csp)
		cab.Set("X-Content-Type-Options", "nosniff")
		cab.Set("Referrer-Policy", "no-referrer")
		cab.Set("X-Frame-Options", "DENY")
		cab.Set("Cross-Origin-Opener-Policy", "same-origin")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			cab.Set("Allow", "GET, HEAD")
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		limpia := path.Clean("/" + r.URL.Path) // sin «..»: no sale de dir
		switch {
		case limpia == "/config.json":
			cab.Set("Content-Type", "application/json; charset=utf-8")
			cab.Set("Cache-Control", "no-store")
			_, _ = w.Write(config)
			return
		case strings.HasPrefix(limpia, "/assets/"):
			// Vite pone un hash en el nombre: se pueden guardar para siempre.
			cab.Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(limpia)))
		if err == nil && !info.IsDir() {
			if path.Base(limpia) == "index.html" {
				cab.Set("Cache-Control", "no-cache")
			}
			archivos.ServeHTTP(w, r)
			return
		}
		// Un archivo con extensión que no existe es un 404 real; una ruta sin
		// extensión (/mapa, /admin) es del router del SPA → index.html.
		if path.Ext(limpia) != "" {
			http.NotFound(w, r)
			return
		}
		cab.Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	}), nil
}
