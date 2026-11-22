// Package api expone el sistema por HTTP (Parte C): predicción del riesgo de
// dengue con signos de alarma, consulta del modelo publicado, estado del
// clúster y entrenamientos distribuidos. Solo usa net/http (AGENTS.md §3.2).
//
// Rutas (documentadas en docs/api/openapi.yaml y servidas en /docs):
//
//	GET  /healthz
//	POST /api/v1/auth/login   (JWT, ADR-0007)
//	GET  /api/v1/modelo
//	POST /api/v1/predicciones
//	GET  /api/v1/riesgo       (cubo de riesgo para el mapa, de reports/evaluacion.json)
//	GET  /api/v1/evaluacion   (reports/evaluacion.json completo)
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
	"sync"
	"sync/atomic"
	"time"

	apidoc "github.com/JorgeGarcia005/cc65-tf-dengue/docs/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
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

// Opciones del servidor. Sin Entrenador o sin Cluster las rutas
// correspondientes responden 503.
type Opciones struct {
	// Modelo es el de data/gold/modelo.json: se usa solo si el almacén no
	// tiene un modelo vigente (ADR-0006, PR-40 H2).
	Modelo     *Modelo
	Entrenador Entrenador
	Cluster    EstadoCluster
	// Almacen persiste modelos, entrenamientos y predicciones (MongoDB en
	// producción). Si es nil se usa almacen.Memoria.
	Almacen almacen.Almacen
	// Cache guarda predicciones (Redis en producción). Si es nil no hay caché.
	Cache almacen.Cache
	// Evaluacion es el contenido de reports/evaluacion.json (#59): lo sirven
	// GET /api/v1/evaluacion y, recortado, GET /api/v1/riesgo. Vacío = 503.
	Evaluacion []byte
	// Usuarios guarda las cuentas del login. Si es nil y Almacen implementa
	// almacen.Usuarios (Memoria y Mongo lo hacen), se usa Almacen.
	Usuarios almacen.Usuarios
	// Firmador emite los JWT del login (ADR-0007). Sin él, el login responde 503.
	Firmador   *auth.Firmador
	Middleware []Middleware // se aplican en orden: el primero es el más externo
	Log        func(formato string, args ...any)
	// PlazoCierre acota cuánto espera Cerrar a que se vacíe el registro de
	// predicciones (PR-40 H6). Por defecto, DefectoPlazoCierre.
	PlazoCierre time.Duration
}

// BufferRegistros es la capacidad del canal hacia el registrador de
// predicciones: si se llena, el registro se descarta y la predicción no espera.
const BufferRegistros = 1024

// Plazos (PR-40 H5, H6). La caché tiene un presupuesto fijo dentro de los
// 100 ms de la predicción: si Redis no responde a tiempo, se calcula.
const (
	PresupuestoLecturaCache   = 15 * time.Millisecond
	PresupuestoEscrituraCache = 15 * time.Millisecond
	DefectoPlazoCierre        = 5 * time.Second
	plazoRegistro             = 2 * time.Second
	plazoArranque             = 15 * time.Second
)

// Servidor atiende la API.
type Servidor struct {
	op          Opciones
	modelo      atomic.Pointer[Modelo]
	ents        entrenamientos
	registros   chan almacen.RegistroPrediccion
	descartados atomic.Int64
	ctx         context.Context // se cancela al cerrar: detiene el entrenamiento
	cancelar    context.CancelFunc
	// ctxDrenaje se cancela cuando vence PlazoCierre: corta las escrituras
	// pendientes del registrador.
	ctxDrenaje      context.Context
	cancelarDrenaje context.CancelFunc

	cierreMu      sync.Mutex // protege cerrado y el Add de wgEntrenar
	cerrado       bool
	wgEntrenar    sync.WaitGroup
	wgRegistrador sync.WaitGroup
	cerrarUnaVez  sync.Once

	evaluacion, riesgo *documento // reports/evaluacion.json precomprimido; nil si no hay
	limite             *limitador // intentos fallidos de login por IP
}

// Nuevo crea el servidor: elige el modelo inicial (el vigente del almacén o,
// si no hay, el de op.Modelo, que se guarda como vigente), cierra los
// entrenamientos que una ejecución anterior dejó en curso y arranca el
// registrador de predicciones (una goroutine que escribe en el almacén).
func Nuevo(op Opciones) (*Servidor, error) {
	if op.Log == nil {
		op.Log = log.Printf
	}
	if op.Almacen == nil {
		op.Almacen = almacen.NuevaMemoria()
	}
	if op.PlazoCierre <= 0 {
		op.PlazoCierre = DefectoPlazoCierre
	}
	if u, ok := op.Almacen.(almacen.Usuarios); op.Usuarios == nil && ok {
		op.Usuarios = u
	}
	if op.Firmador != nil {
		// El hash señuelo cuesta un PBKDF2 completo: se calcula ya, en segundo
		// plano, para que el primer login no pague dos derivaciones.
		go auth.HashSenuelo()
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoArranque)
	defer cancelar()
	inicial, err := modeloInicial(ctx, op)
	if err != nil {
		return nil, err
	}
	s := &Servidor{op: op, registros: make(chan almacen.RegistroPrediccion, BufferRegistros), limite: nuevoLimitador()}
	if len(op.Evaluacion) > 0 {
		if s.evaluacion, s.riesgo, err = cargarEvaluacion(op.Evaluacion); err != nil {
			return nil, err
		}
	}
	s.ents = entrenamientos{almacen: op.Almacen, log: op.Log, nuevoID: nuevoIDEntrenamiento}
	if err := s.ents.reconciliar(ctx); err != nil {
		return nil, fmt.Errorf("api: entrenamientos pendientes: %w", err)
	}
	s.modelo.Store(inicial)
	s.ctx, s.cancelar = context.WithCancel(context.Background())
	s.ctxDrenaje, s.cancelarDrenaje = context.WithCancel(context.Background())
	s.wgRegistrador.Add(1)
	go s.registrador()
	return s, nil
}

// modeloInicial distingue «no hay modelo guardado» (usa el del archivo y lo
// guarda) de «no se pudo leer» (error: no se arranca con un modelo distinto
// al que el almacén considera vigente).
func modeloInicial(ctx context.Context, op Opciones) (*Modelo, error) {
	doc, ok, err := op.Almacen.ModeloVigente(ctx)
	if err != nil {
		return nil, fmt.Errorf("api: leer el modelo vigente: %w", err)
	}
	if ok {
		m, err := ModeloDeDocumento(doc)
		if err != nil {
			return nil, fmt.Errorf("api: el modelo vigente %s es inválido: %w", doc.Version, err)
		}
		op.Log("api: modelo inicial %s, del almacén", m.Version)
		return m, nil
	}
	if op.Modelo == nil {
		return nil, errors.New("api: el almacén no tiene modelo y no se indicó uno")
	}
	if err := op.Almacen.GuardarModelo(ctx, op.Modelo.Documento(time.Now().UTC())); err != nil {
		return nil, fmt.Errorf("api: guardar el modelo inicial: %w", err)
	}
	op.Log("api: modelo inicial %s, del archivo", op.Modelo.Version)
	return op.Modelo, nil
}

// ModeloActual devuelve el modelo publicado.
func (s *Servidor) ModeloActual() *Modelo { return s.modelo.Load() }

// RegistrosDescartados cuenta las predicciones que no se registraron porque
// el búfer estaba lleno o venció el plazo de cierre.
func (s *Servidor) RegistrosDescartados() int64 { return s.descartados.Load() }

// Cerrar deja de admitir entrenamientos, cancela el que esté en curso y
// espera a que escriba su estado final; luego vacía el registro de
// predicciones con un plazo global (PR-40 H6). Recién después se puede
// cerrar el almacén. Es idempotente.
func (s *Servidor) Cerrar() {
	s.cerrarUnaVez.Do(func() {
		s.cierreMu.Lock()
		s.cerrado = true
		s.cierreMu.Unlock()
		s.cancelar()
		s.wgEntrenar.Wait()
		temporizador := time.AfterFunc(s.op.PlazoCierre, s.cancelarDrenaje)
		s.wgRegistrador.Wait()
		temporizador.Stop()
		s.cancelarDrenaje()
	})
}

// registrador es el único consumidor de s.registros. El canal nunca se cierra
// (los handlers podrían enviar después): termina con el contexto y antes
// vacía lo que quede, hasta que venza el plazo de cierre.
func (s *Servidor) registrador() {
	defer s.wgRegistrador.Done()
	guardar := func(r almacen.RegistroPrediccion) {
		ctx, cancelar := context.WithTimeout(s.ctxDrenaje, plazoRegistro)
		defer cancelar()
		if err := s.op.Almacen.RegistrarPrediccion(ctx, r); err != nil && s.ctxDrenaje.Err() == nil {
			s.op.Log("api: no se pudo registrar la predicción: %v", err)
		}
	}
	for {
		select {
		case r := <-s.registros:
			guardar(r)
		case <-s.ctx.Done():
			for {
				if s.ctxDrenaje.Err() != nil {
					n := int64(len(s.registros))
					s.descartados.Add(n)
					s.op.Log("api: venció el plazo de cierre; %d registros sin escribir", n)
					return
				}
				select {
				case r := <-s.registros:
					guardar(r)
				default:
					return
				}
			}
		}
	}
}

// registrar encola sin bloquear.
func (s *Servidor) registrar(r almacen.RegistroPrediccion) {
	select {
	case s.registros <- r:
	default:
		s.descartados.Add(1)
	}
}

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
		{"POST", "/api/v1/auth/login", true, (*Servidor).login},
		{"GET", "/api/v1/modelo", true, (*Servidor).verModelo},
		{"POST", "/api/v1/predicciones", true, (*Servidor).predecir},
		{"GET", "/api/v1/riesgo", true, (*Servidor).verRiesgo},
		{"GET", "/api/v1/evaluacion", true, (*Servidor).verEvaluacion},
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
	inicio := time.Now()
	var c Consulta
	if !leerJSON(w, r, &c, false) {
		return
	}
	m := s.ModeloActual() // una sola lectura: clave, cálculo y registro usan el mismo modelo
	reg, err := m.registro(c)
	if err != nil {
		responderError(w, http.StatusBadRequest, err.Error())
		return
	}
	cn := normalizada(reg)
	clave := almacen.ClavePrediccion(m.Version, cn)
	ctx := r.Context()

	// Cache-aside (ADR-0006) con presupuesto fijo (PR-40 H5): un fallo o una
	// demora de la caché no falla la predicción ni consume los 100 ms.
	cacheSana := s.op.Cache != nil
	if cacheSana {
		ctxLeer, cancelar := context.WithTimeout(ctx, PresupuestoLecturaCache)
		b, ok, err := s.op.Cache.Obtener(ctxLeer, clave)
		cancelar()
		if err != nil {
			cacheSana = false // no se intenta una segunda espera para guardar
			s.op.Log("api: caché no disponible al leer: %v", err)
		}
		var p Prediccion
		if ok && json.Unmarshal(b, &p) == nil {
			w.Header().Set("X-Cache", "HIT")
			responderBytes(w, http.StatusOK, b)
			s.registrar(registroDe(cn, p, true, inicio))
			return
		}
	}
	p := m.prediccion(reg)
	b, err := json.Marshal(p)
	if err != nil {
		responderError(w, http.StatusInternalServerError, "error interno al codificar la respuesta")
		return
	}
	if s.op.Cache != nil {
		if cacheSana {
			ctxGuardar, cancelar := context.WithTimeout(ctx, PresupuestoEscrituraCache)
			if err := s.op.Cache.Guardar(ctxGuardar, clave, b, almacen.TTLPrediccion); err != nil {
				s.op.Log("api: caché no disponible al guardar: %v", err)
			}
			cancelar()
		}
		w.Header().Set("X-Cache", "MISS")
	}
	responderBytes(w, http.StatusOK, b)
	s.registrar(registroDe(cn, p, false, inicio))
}

func registroDe(c almacen.ConsultaNormalizada, p Prediccion, desdeCache bool, inicio time.Time) almacen.RegistroPrediccion {
	return almacen.RegistroPrediccion{Fecha: time.Now().UTC(), VersionModelo: p.VersionModelo, Consulta: c,
		Riesgo: p.Riesgo, Clase: p.Clase, DesdeCache: desdeCache, Microsegundos: time.Since(inicio).Microseconds()}
}

func (s *Servidor) verCluster(w http.ResponseWriter, _ *http.Request) {
	if s.op.Cluster == nil {
		responderError(w, http.StatusServiceUnavailable, "este servidor no tiene un clúster conectado")
		return
	}
	est := s.op.Cluster.Estado()
	if est.Nodos == nil {
		est.Nodos = []cluster.EstadoNodo{} // el contrato declara un arreglo, también antes del primer registro
	}
	responder(w, http.StatusOK, est)
}

func (s *Servidor) crearEntrenamiento(w http.ResponseWriter, r *http.Request) {
	if s.op.Entrenador == nil {
		responderError(w, http.StatusServiceUnavailable, "este servidor no tiene un clúster para entrenar")
		return
	}
	p := ParametrosEntrenamiento{Epocas: DefectoEpocas, Tasa: DefectoTasa, Lambda: DefectoLambda}
	if !leerJSON(w, r, &p, true) {
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

	// Admisión (PR-40 H6): tras Cerrar no se aceptan entrenamientos, y el Add
	// ocurre bajo el mismo mutex que lee Cerrar, nunca en paralelo a su Wait.
	s.cierreMu.Lock()
	if s.cerrado {
		s.cierreMu.Unlock()
		responderError(w, http.StatusServiceUnavailable, "el servidor se está cerrando")
		return
	}
	s.wgEntrenar.Add(1)
	s.cierreMu.Unlock()

	t, err := s.ents.iniciar(r.Context(), p)
	switch {
	case errors.Is(err, errEnCurso):
		s.wgEntrenar.Done()
		responderError(w, http.StatusConflict, fmt.Sprintf("ya hay un entrenamiento en curso (%s)", t.ID))
		return
	case err != nil:
		s.wgEntrenar.Done()
		responderError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	go s.entrenar(t.ID, p, base)
	w.Header().Set("Location", "/api/v1/entrenamientos/"+t.ID)
	responder(w, http.StatusAccepted, t)
}

// entrenar corre en su goroutine. Publica el modelo nuevo solo si el almacén
// lo guardó: la versión que anuncia la API siempre se puede recuperar
// después de un reinicio (PR-40 H1).
func (s *Servidor) entrenar(id string, p ParametrosEntrenamiento, base *Modelo) {
	defer s.wgEntrenar.Done()
	inicio := time.Now()
	w, err := s.op.Entrenador.Entrenar(s.ctx, p, func(e int, perdida float64) { s.ents.progreso(id, e, perdida) })
	var est *cluster.Estado
	if s.op.Cluster != nil {
		e := s.op.Cluster.Estado()
		est = &e
	}
	fin := time.Now().UTC()
	var version string
	if err == nil {
		var m *Modelo
		if m, err = base.conPesos(w, time.Now().Format(time.RFC3339)); err == nil {
			// El resumen final viaja en la misma escritura que el modelo: si luego
			// se pierde el estado terminal, reconciliar lo reconstruye completo.
			ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
			errG := s.op.Almacen.GuardarModelo(ctx, m.Documento(fin, s.ents.resumen(id, est, fin)))
			cancelar()
			if errG != nil {
				err = fmt.Errorf("no se pudo guardar el modelo %s; sigue publicado el anterior: %w", m.Version, errG)
			} else {
				s.modelo.Store(m) // las claves de caché cambian con la versión
				version = m.Version
			}
		}
	}
	s.ents.terminar(id, version, est, fin, err)
	if err != nil {
		s.op.Log("api: entrenamiento %s falló tras %v: %v", id, time.Since(inicio).Round(time.Millisecond), err)
	} else {
		s.op.Log("api: entrenamiento %s terminado en %v; modelo publicado %s", id, time.Since(inicio).Round(time.Millisecond), version)
	}
}

func (s *Servidor) verEntrenamiento(w http.ResponseWriter, r *http.Request) {
	t, ok, err := s.ents.consultar(r.Context(), r.PathValue("id"))
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "no se pudo consultar el almacén: "+err.Error())
		return
	}
	if !ok {
		responderError(w, http.StatusNotFound, fmt.Sprintf("no existe el entrenamiento %q", r.PathValue("id")))
		return
	}
	responder(w, http.StatusOK, t)
}

// --- JSON ---------------------------------------------------------------------

// responder codifica antes de escribir el código: si v no se puede codificar
// (p. ej. un NaN), el cliente recibe 500 con mensaje y no un 200 vacío.
func responder(w http.ResponseWriter, codigo int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		codigo, b = http.StatusInternalServerError, []byte(`{"error":"error interno al codificar la respuesta"}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write(append(b, '\n'))
}

// responderBytes escribe un JSON ya codificado (p. ej. desde la caché).
func responderBytes(w http.ResponseWriter, codigo int, b []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write(b)
}

func responderError(w http.ResponseWriter, codigo int, msg string) {
	responder(w, codigo, map[string]string{"error": msg})
}

// leerJSON decodifica el cuerpo de forma estricta y responde el error en
// español. Devuelve false si ya respondió. Si opcional, un cuerpo vacío es
// válido y deja v como estaba (con sus valores por defecto).
func leerJSON(w http.ResponseWriter, r *http.Request, v any, opcional bool) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxCuerpo))
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil {
		// Decoder.More solo mira dentro de un arreglo u objeto: para exigir que
		// el documento terminó hay que intentar leer otro valor y obtener EOF.
		var resto json.RawMessage
		if errResto := dec.Decode(&resto); errResto != io.EOF {
			var grande *http.MaxBytesError
			if errors.As(errResto, &grande) {
				err = errResto
			} else {
				responderError(w, http.StatusBadRequest, "hay contenido después del objeto JSON")
				return false
			}
		}
	}
	if err == nil {
		return true
	}
	if errors.Is(err, io.EOF) && opcional {
		return true // sin cuerpo: v conserva los valores por defecto
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
