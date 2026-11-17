// Package api expone el sistema por HTTP (Parte C): predicción del riesgo de
// dengue con signos de alarma, consulta del modelo publicado, estado del
// clúster y entrenamientos distribuidos. Solo usa net/http (AGENTS.md §3.2).
//
// Rutas (documentadas en docs/api/openapi.yaml y servidas en /docs):
//
//	GET  /healthz
//	GET  /api/v1/modelo
//	POST /api/v1/predicciones
//	GET  /api/v1/cluster
//	POST /api/v1/entrenamientos
//	GET  /api/v1/entrenamientos/{id}
//
// Concurrencia: net/http atiende cada petición en su goroutine. El modelo
// publicado vive en un atomic.Pointer (lecturas sin bloqueo; un entrenamiento
// nuevo lo reemplaza entero) y el historial de entrenamientos tiene su mutex.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	apidoc "github.com/JorgeGarcia005/cc65-tf-dengue/docs/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
)

// MaxCuerpo limita el cuerpo de cualquier petición (las consultas pesan < 1 KB).
const MaxCuerpo = 64 << 10

// EstadoCluster da la foto del clúster; *cluster.Coordinador lo implementa.
type EstadoCluster interface {
	Estado() cluster.Estado
}

// Middleware envuelve un handler. JWT y CORS (TB2) entran aquí sin tocar las rutas.
type Middleware func(http.Handler) http.Handler

// Opciones del servidor. Modelo es obligatorio; sin Entrenador o sin Cluster
// las rutas correspondientes responden 503.
type Opciones struct {
	Modelo     *Modelo
	Entrenador Entrenador
	Cluster    EstadoCluster
	Middleware []Middleware // se aplican en orden: el primero es el más externo
	Log        func(formato string, args ...any)
}

// Servidor atiende la API.
type Servidor struct {
	op       Opciones
	modelo   atomic.Pointer[Modelo]
	ents     entrenamientos
	ctx      context.Context
	cancelar context.CancelFunc
}

// Nuevo crea el servidor con el modelo inicial.
func Nuevo(op Opciones) (*Servidor, error) {
	if op.Modelo == nil {
		return nil, errors.New("api: falta el modelo")
	}
	if op.Log == nil {
		op.Log = log.Printf
	}
	s := &Servidor{op: op}
	s.modelo.Store(op.Modelo)
	s.ctx, s.cancelar = context.WithCancel(context.Background())
	return s, nil
}

// ModeloActual devuelve el modelo publicado.
func (s *Servidor) ModeloActual() *Modelo { return s.modelo.Load() }

// Cerrar cancela el entrenamiento en curso, si hay uno.
func (s *Servidor) Cerrar() { s.cancelar() }

// Ruta describe una ruta del mux. Documentar indica si debe figurar en el OpenAPI.
type Ruta struct {
	Metodo, Ruta string
	Documentar   bool
	manejar      func(*Servidor, http.ResponseWriter, *http.Request)
}

// Rutas es la tabla única de la que salen el mux y el test del OpenAPI.
func Rutas() []Ruta {
	return []Ruta{
		{"GET", "/healthz", true, (*Servidor).salud},
		{"GET", "/api/v1/modelo", true, (*Servidor).verModelo},
		{"POST", "/api/v1/predicciones", true, (*Servidor).predecir},
		{"GET", "/api/v1/cluster", true, (*Servidor).verCluster},
		{"POST", "/api/v1/entrenamientos", true, (*Servidor).crearEntrenamiento},
		{"GET", "/api/v1/entrenamientos/{id}", true, (*Servidor).verEntrenamiento},
		{"GET", "/docs", false, (*Servidor).swaggerUI},
		{"GET", "/docs/openapi.yaml", false, (*Servidor).openapi},
	}
}

// Handler arma el mux con los middlewares.
func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	for _, r := range Rutas() {
		manejar := r.manejar
		mux.HandleFunc(r.Metodo+" "+r.Ruta, func(w http.ResponseWriter, req *http.Request) { manejar(s, w, req) })
	}
	var h http.Handler = mux
	for i := len(s.op.Middleware) - 1; i >= 0; i-- {
		h = s.op.Middleware[i](h)
	}
	return s.recuperar(h)
}

// recuperar convierte un pánico en un 500 sin tumbar el proceso.
func (s *Servidor) recuperar(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.op.Log("api: pánico en %s %s: %v\n%s", r.Method, r.URL.Path, v, debug.Stack())
				responderError(w, http.StatusInternalServerError, "error interno")
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// --- Handlers -----------------------------------------------------------------

func (s *Servidor) salud(w http.ResponseWriter, _ *http.Request) {
	responder(w, http.StatusOK, map[string]string{"estado": "ok", "version_modelo": s.ModeloActual().Version})
}

type variable struct {
	Nombre string  `json:"nombre"`
	Peso   float64 `json:"peso"`
}

func (s *Servidor) verModelo(w http.ResponseWriter, _ *http.Request) {
	m := s.ModeloActual()
	vars := make([]variable, m.Esquema.D())
	for j, n := range m.Esquema.Nombres {
		vars[j] = variable{n, m.Pesos[j]}
	}
	responder(w, http.StatusOK, map[string]any{
		"version": m.Version, "generado": m.Generado, "umbral": m.Umbral,
		"ajuste_prior": m.AjustePrior, "sesgo": m.Pesos[len(vars)], "variables": vars,
		"departamentos": m.Esquema.Departamentos, "ano_min": m.Esquema.AnoMin,
	})
}

func (s *Servidor) predecir(w http.ResponseWriter, r *http.Request) {
	var c Consulta
	if !leerJSON(w, r, &c) {
		return
	}
	p, err := s.ModeloActual().Predecir(c)
	if err != nil {
		responderError(w, http.StatusBadRequest, err.Error())
		return
	}
	responder(w, http.StatusOK, p)
}

func (s *Servidor) verCluster(w http.ResponseWriter, _ *http.Request) {
	if s.op.Cluster == nil {
		responderError(w, http.StatusServiceUnavailable, "este servidor no tiene un clúster conectado")
		return
	}
	responder(w, http.StatusOK, s.op.Cluster.Estado())
}

func (s *Servidor) crearEntrenamiento(w http.ResponseWriter, r *http.Request) {
	if s.op.Entrenador == nil {
		responderError(w, http.StatusServiceUnavailable, "este servidor no tiene un clúster para entrenar")
		return
	}
	p := ParametrosEntrenamiento{Epocas: DefectoEpocas, Tasa: DefectoTasa, Lambda: DefectoLambda}
	if !leerJSON(w, r, &p) {
		return
	}
	switch {
	case p.Epocas < 1 || p.Epocas > MaxEpocas:
		responderError(w, http.StatusBadRequest, fmt.Sprintf("épocas %d fuera de 1–%d", p.Epocas, MaxEpocas))
		return
	case !(p.Tasa > 0) || p.Tasa > 100:
		responderError(w, http.StatusBadRequest, fmt.Sprintf("tasa %v fuera de (0, 100]", p.Tasa))
		return
	case !(p.Lambda >= 0) || p.Lambda > 10:
		responderError(w, http.StatusBadRequest, fmt.Sprintf("lambda %v fuera de [0, 10]", p.Lambda))
		return
	}
	base := s.ModeloActual()
	p.D = base.Esquema.D()
	t, ok := s.ents.iniciar(p)
	if !ok {
		responderError(w, http.StatusConflict, fmt.Sprintf("ya hay un entrenamiento en curso (%s)", t.ID))
		return
	}
	go s.entrenar(t.ID, p, base)
	w.Header().Set("Location", "/api/v1/entrenamientos/"+t.ID)
	responder(w, http.StatusAccepted, t)
}

// entrenar corre en su goroutine; al terminar bien publica el modelo nuevo.
func (s *Servidor) entrenar(id string, p ParametrosEntrenamiento, base *Modelo) {
	inicio := time.Now()
	w, err := s.op.Entrenador.Entrenar(s.ctx, p, func(e int, perdida float64) { s.ents.progreso(id, e, perdida) })
	var version string
	if err == nil {
		var m *Modelo
		if m, err = base.conPesos(w, time.Now().Format(time.RFC3339)); err == nil {
			s.modelo.Store(m)
			version = m.Version
		}
	}
	s.ents.terminar(id, version, err)
	if err != nil {
		s.op.Log("api: entrenamiento %s falló tras %v: %v", id, time.Since(inicio).Round(time.Millisecond), err)
	} else {
		s.op.Log("api: entrenamiento %s terminado en %v; modelo publicado %s", id, time.Since(inicio).Round(time.Millisecond), version)
	}
}

func (s *Servidor) verEntrenamiento(w http.ResponseWriter, r *http.Request) {
	t, ok := s.ents.consultar(r.PathValue("id"))
	if !ok {
		responderError(w, http.StatusNotFound, fmt.Sprintf("no existe el entrenamiento %q", r.PathValue("id")))
		return
	}
	responder(w, http.StatusOK, t)
}

// --- JSON ---------------------------------------------------------------------

func responder(w http.ResponseWriter, codigo int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(v)
}

func responderError(w http.ResponseWriter, codigo int, msg string) {
	responder(w, codigo, map[string]string{"error": msg})
}

// leerJSON decodifica el cuerpo de forma estricta y responde el error en
// español. Devuelve false si ya respondió.
func leerJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxCuerpo))
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil && dec.More() {
		err = errors.New("hay datos después del objeto JSON")
	}
	if err == nil {
		return true
	}
	var (
		sintaxis *json.SyntaxError
		tipo     *json.UnmarshalTypeError
		grande   *http.MaxBytesError
		msg      string
		codigo   = http.StatusBadRequest
	)
	switch {
	case errors.As(err, &grande):
		codigo, msg = http.StatusRequestEntityTooLarge, fmt.Sprintf("el cuerpo supera %d bytes", MaxCuerpo)
	case errors.As(err, &sintaxis), errors.Is(err, io.ErrUnexpectedEOF):
		msg = "JSON mal formado"
	case errors.Is(err, io.EOF):
		msg = "falta el cuerpo JSON"
	case errors.As(err, &tipo):
		msg = fmt.Sprintf("el campo %q debe ser de tipo %s", tipo.Field, nombreTipo(tipo.Type.Kind().String()))
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		msg = "campo desconocido " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	default:
		msg = "cuerpo inválido: " + err.Error()
	}
	responderError(w, codigo, msg)
	return false
}

func nombreTipo(k string) string {
	switch k {
	case "string":
		return "texto"
	case "int", "int64", "float64":
		return "número"
	}
	return k
}

// --- Documentación ------------------------------------------------------------

func especificacion() []byte { return apidoc.OpenAPI }

func (s *Servidor) openapi(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(especificacion())
}

// swaggerUI carga Swagger UI desde un CDN con versión fija; el servidor solo
// entrega esta página y el YAML (sin dependencias Go de terceros).
const paginaSwagger = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>API dengue CC65 · Swagger UI</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
<script>
window.ui = SwaggerUIBundle({ url: "/docs/openapi.yaml", dom_id: "#swagger-ui" });
</script>
</body>
</html>
`

func (s *Servidor) swaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, paginaSwagger)
}
