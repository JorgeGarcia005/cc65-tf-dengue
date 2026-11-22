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
	"strconv"
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
	cerrado := make(chan struct{})
	go func() {
		defer close(cerrado)
		<-ctx.Done()
		c, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		if err := srv.Shutdown(c); err != nil {
			log.Printf("web: cierre: %v", err)
		}
	}()
	log.Printf("web: sirviendo %s en %s (API %s)", *dir, *escucha, apiURL)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-cerrado // ListenAndServe vuelve apenas empieza Shutdown: hay que esperar a que drene
}

// validarAPI acepta solo http(s)://host[:puerto] (con / final opcional) y
// devuelve el origen. Nada que pueda colarse en la CSP: sin credenciales, query,
// fragmento ni caracteres fuera de [A-Za-z0-9.-] en el host (un «;» inyectaría
// otra directiva).
func validarAPI(apiURL string) (string, error) {
	errFormato := fmt.Errorf("web: API_URL %q debe ser http(s)://host[:puerto]", apiURL)
	u, err := url.Parse(apiURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Opaque != "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errFormato
	}
	host, puerto := u.Hostname(), u.Port()
	if host == "" || strings.Trim(host, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-") != "" {
		return "", errFormato
	}
	if puerto != "" {
		if n, err := strconv.Atoi(puerto); err != nil || n < 1 || n > 65535 {
			return "", errFormato
		}
		host += ":" + puerto
	}
	return u.Scheme + "://" + host, nil
}

// nuevoServidor valida la carpeta y la URL de la API y arma el handler.
func nuevoServidor(dir, apiURL string) (http.Handler, error) {
	raiz, err := os.OpenRoot(dir) // ninguna ruta pedida puede salir de dir (ni con «\» en Windows)
	if err != nil {
		return nil, fmt.Errorf("web: %w", err)
	}
	if info, err := raiz.Stat("index.html"); err != nil || info.IsDir() {
		return nil, fmt.Errorf("web: falta index.html en %s", dir)
	}
	origenAPI, err := validarAPI(apiURL)
	if err != nil {
		return nil, err
	}
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
		limpia := path.Clean("/" + r.URL.Path)
		if limpia == "/config.json" {
			cab.Set("Content-Type", "application/json; charset=utf-8")
			cab.Set("Cache-Control", "no-store")
			_, _ = w.Write(config)
			return
		}
		if !strings.ContainsRune(limpia, '\\') && limpia != "/" {
			if servirArchivo(w, r, raiz, strings.TrimPrefix(limpia, "/")) {
				return
			}
		}
		// Un archivo con extensión que no existe es un 404 real (sin caché larga);
		// una ruta sin extensión (/mapa, /admin) es del router del SPA → index.html.
		if path.Ext(limpia) != "" {
			http.NotFound(w, r)
			return
		}
		servirArchivo(w, r, raiz, "index.html")
	}), nil
}

// servirArchivo entrega nombre (relativo a raiz) si existe y no es carpeta. Usa
// ServeContent: a diferencia de FileServer/ServeFile, no redirige /index.html.
// La caché inmutable va solo para assets que existen (Vite les pone un hash).
func servirArchivo(w http.ResponseWriter, r *http.Request, raiz *os.Root, nombre string) bool {
	f, err := raiz.Open(nombre)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	if strings.HasPrefix(nombre, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	return true
}
