package api

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// Modo de autenticación de una ruta (ADR-0007).
type Modo int

const (
	// Publica no exige token.
	Publica Modo = iota
	// Bearer exige "Authorization: Bearer <jwt>" y que el rol esté en Ruta.Roles.
	Bearer
	// Ticket es el handshake del WebSocket: Origin en la lista de CORS y
	// ?ticket= de un solo uso (el navegador no puede mandar Authorization ahí).
	Ticket
)

func (m Modo) String() string {
	switch m {
	case Bearer:
		return "bearer"
	case Ticket:
		return "ticket"
	}
	return "publica"
}

// Roles de la matriz de ADR-0007.
var (
	rolesTodos = []auth.Rol{auth.RolAdmin, auth.RolAnalista}
	soloAdmin  = []auth.Rol{auth.RolAdmin}
)

// Cabeceras CORS (ADR-0007). Expuestas: sin esto el navegador oculta X-Cache
// de la predicción, Retry-After del login y Location del entrenamiento.
const (
	corsExpuestas = "X-Cache, Retry-After, Location"
	corsMetodos   = "GET, POST"
	corsCabeceras = "Authorization, Content-Type"
	corsMaxAge    = "600"
)

// ParsearOrigenes lee la lista blanca de CORS_ORIGENES («http://a:8081,https://b»).
// Cada origen es esquema://host[:puerto], sin ruta; se rechazan «*» y «null».
func ParsearOrigenes(s string) ([]string, error) {
	var origenes []string
	for _, o := range strings.Split(s, ",") {
		if o = strings.TrimSpace(o); o == "" {
			continue
		}
		norm, err := normalizarOrigen(o)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(origenes, norm) {
			origenes = append(origenes, norm)
		}
	}
	return origenes, nil
}

// normalizarOrigen deja el origen como lo envía el navegador en la cabecera
// Origin: esquema y host en minúsculas, sin barra final.
func normalizarOrigen(o string) (string, error) {
	u, err := url.Parse(o)
	switch {
	case err != nil:
		return "", fmt.Errorf("origen CORS %q: %w", o, err)
	case u.Scheme != "http" && u.Scheme != "https":
		return "", fmt.Errorf("origen CORS %q: el esquema debe ser http o https", o)
	case u.Hostname() == "" || strings.Contains(u.Hostname(), "*") || u.User != nil:
		return "", fmt.Errorf("origen CORS %q: falta el host, lleva comodín o credenciales", o)
	case strings.HasSuffix(u.Host, ":") || !puertoValido(u.Port()):
		return "", fmt.Errorf("origen CORS %q: el puerto debe estar entre 1 y 65535", o)
	case u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.HasSuffix(o, "?") || strings.HasSuffix(o, "#"):
		return "", fmt.Errorf("origen CORS %q: un origen no lleva ruta, consulta ni fragmento", o)
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}

// puertoValido acepta un puerto vacío (el del esquema) o un número de 1 a 65535.
func puertoValido(p string) bool {
	if p == "" {
		return true
	}
	n, err := strconv.Atoi(p)
	return err == nil && n >= 1 && n <= 65535 && p[0] != '0'
}

// cabecerasSeguridad pone las cabeceras comunes a toda respuesta y limita el
// cuerpo de los POST a MaxCuerpo (413 al leerlo de más).
func cabecerasSeguridad(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodPost && r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, MaxCuerpo)
		}
		next.ServeHTTP(w, r)
	})
}

// cors decora toda respuesta (2xx, 4xx y 5xx) de un origen permitido y
// responde el preflight aquí, antes de la autenticación y del 405 del mux: el
// navegador no manda Authorization en él. Nunca «*» ni Allow-Credentials.
func (s *Servidor) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")
		origen := r.Header.Get("Origin")
		permitido := origen != "" && slices.Contains(s.op.OrigenesCORS, origen)
		if permitido {
			h.Set("Access-Control-Allow-Origin", origen)
			h.Set("Access-Control-Expose-Headers", corsExpuestas)
		}
		if r.Method == http.MethodOptions && origen != "" && r.Header.Get("Access-Control-Request-Method") != "" {
			if permitido {
				h.Set("Access-Control-Allow-Methods", corsMetodos)
				h.Set("Access-Control-Allow-Headers", corsCabeceras)
				h.Set("Access-Control-Max-Age", corsMaxAge)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// proteger exige un JWT válido con uno de los roles. Sin firmador la ruta
// responde 503: una API sin secreto falla cerrada, nunca queda abierta.
// Cache-Control: no-store es el valor por defecto de toda respuesta
// autenticada; un handler puede cambiarlo (los documentos con ETag usan
// «private, no-cache», que tampoco deja copia en cachés compartidas).
func (s *Servidor) proteger(h http.Handler, roles []auth.Rol) http.Handler {
	if s.op.Firmador == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			responderError(w, http.StatusServiceUnavailable, "la autenticación no está configurada (falta JWT_SECRETO)")
		})
	}
	conRol := auth.Requiere(roles...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	}))
	return s.op.Firmador.Autenticar(conRol)
}
