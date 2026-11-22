package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws"
)

// Tiempo real y demo (ADR-0007, #65).
const (
	// VidaTicket es cuánto vale un ticket del WebSocket desde que se emite.
	VidaTicket = 30 * time.Second
	// MaxTicketsPendientes acota los tickets emitidos y sin usar: un token de
	// admin robado no puede llenar la memoria pidiendo tickets.
	MaxTicketsPendientes = 128
	// IntervaloPingWS es cada cuánto el servidor manda ping; un cliente que no
	// responde en el doble se da por caído (ws.Hub.Servir).
	IntervaloPingWS = 20 * time.Second
	// BuferWS es la cola por cliente del hub: si se llena, se descarta (el
	// entrenamiento nunca espera al navegador).
	BuferWS = 256
	// MaxRitmo acota la pausa entre épocas del modo demo.
	MaxRitmo = 2000
)

// Demo es lo que el modo -demo del coordinador agrega a la API: cortar la
// conexión de un nodo para mostrar la reasignación. *cluster.Coordinador lo
// implementa. Sin Demo (sin -demo) la ruta no existe (404) y ritmo_ms se rechaza.
type Demo interface {
	CortarNodo(nombre string) error
}

// ticket es un permiso de un solo uso para abrir el WebSocket.
type ticket struct {
	rec   auth.Reclamos
	vence time.Time
}

// tickets guarda los tickets emitidos. Consumir busca, borra y valida el
// vencimiento en la misma sección crítica: dos handshakes concurrentes con el
// mismo ticket no pasan ambos.
type tickets struct {
	mu    sync.Mutex
	m     map[string]ticket
	ahora func() time.Time
}

var errDemasiadosTickets = errors.New("demasiados tickets pendientes")

func nuevosTickets() *tickets {
	return &tickets{m: map[string]ticket{}, ahora: time.Now}
}

// Emitir crea un ticket aleatorio de 32 bytes para rec.
func (t *tickets) Emitir(rec auth.Reclamos) (string, time.Time, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", time.Time{}, err
	}
	id := base64.RawURLEncoding.EncodeToString(b[:])
	t.mu.Lock()
	defer t.mu.Unlock()
	ahora := t.ahora()
	for k, v := range t.m { // poda: los vencidos no cuentan para el tope
		if !ahora.Before(v.vence) {
			delete(t.m, k)
		}
	}
	if len(t.m) >= MaxTicketsPendientes {
		return "", time.Time{}, errDemasiadosTickets
	}
	vence := ahora.Add(VidaTicket)
	t.m[id] = ticket{rec: rec, vence: vence}
	return id, vence, nil
}

// Consumir devuelve los reclamos del ticket y lo borra. Inexistente, vencido o
// ya usado → false.
func (t *tickets) Consumir(id string) (auth.Reclamos, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	tk, ok := t.m[id]
	if !ok {
		return auth.Reclamos{}, false
	}
	delete(t.m, id)
	return tk.rec, t.ahora().Before(tk.vence)
}

// origenPermitido dice si el Origin del navegador está en CORS_ORIGENES.
func (s *Servidor) origenPermitido(origen string) bool {
	return origen != "" && slices.Contains(s.op.OrigenesCORS, origen)
}

// conTicket autentica el handshake del WebSocket (modo Ticket, ADR-0007). El
// orden importa: (1) Origin fuera de la lista → 403 antes de tocar los
// tickets, para que otro sitio no pueda quemar ni sondear uno (cross-site
// WebSocket hijacking); (2) ticket inexistente, vencido o usado → 401;
// (3) rol sin acceso → 403 (defensa en profundidad: la emisión ya lo filtra).
func (s *Servidor) conTicket(h http.Handler, roles []auth.Rol) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respuestas de error del handshake (401/403/400/426): como toda respuesta
		// autenticada, sin copia en cachés (PR-84 H3). No afecta al 101.
		w.Header().Set("Cache-Control", "no-store")
		if !s.origenPermitido(r.Header.Get("Origin")) {
			responderError(w, http.StatusForbidden, "origen no permitido para el WebSocket")
			return
		}
		rec, ok := s.tickets.Consumir(r.URL.Query().Get("ticket"))
		if !ok {
			responderError(w, http.StatusUnauthorized, "ticket inválido, vencido o ya usado: pide otro en POST /api/v1/ws/ticket")
			return
		}
		if !slices.Contains(roles, rec.Rol) {
			responderError(w, http.StatusForbidden, "tu rol no tiene acceso a este recurso")
			return
		}
		h.ServeHTTP(w, r.WithContext(auth.ConReclamos(r.Context(), rec)))
	})
}

// RespuestaTicket es el cuerpo de POST /api/v1/ws/ticket.
type RespuestaTicket struct {
	Ticket string `json:"ticket"`
	Expira int64  `json:"expira"` // segundos Unix
}

func (s *Servidor) emitirTicket(w http.ResponseWriter, r *http.Request) {
	if !descartarCuerpo(w, r) {
		return
	}
	rec, _ := auth.ReclamosDe(r.Context()) // proteger ya exigió el token
	id, vence, err := s.tickets.Emitir(rec)
	switch {
	case errors.Is(err, errDemasiadosTickets):
		w.Header().Set("Retry-After", fmt.Sprint(int(VidaTicket.Seconds())))
		responderError(w, http.StatusTooManyRequests, "demasiados tickets sin usar; espera a que venzan")
		return
	case err != nil:
		responderError(w, http.StatusInternalServerError, "no se pudo emitir el ticket")
		return
	}
	responder(w, http.StatusOK, RespuestaTicket{Ticket: id, Expira: vence.Unix()})
}

// servirWS hace el handshake y conecta al cliente con el hub: recibe los
// eventos del clúster (cluster.Evento en JSON) hasta que se cierre.
func (s *Servidor) servirWS(w http.ResponseWriter, r *http.Request) {
	// Suscribir antes del 101: cuando el navegador recibe el handshake ya está
	// en el hub y no pierde los eventos siguientes (PR-84 H2).
	cliente := s.op.Hub.Suscribir()
	conn, err := ws.Aceptar(w, r, nil) // el Origin ya lo validó conTicket
	if err != nil {
		s.op.Hub.Desuscribir(cliente)
		return // Aceptar ya respondió con el código adecuado
	}
	s.op.Hub.Servir(conn, cliente, IntervaloPingWS)
}

// cortarNodo es POST /api/v1/demo/nodos/{nombre}/cortar (solo con -demo).
func (s *Servidor) cortarNodo(w http.ResponseWriter, r *http.Request) {
	if !descartarCuerpo(w, r) {
		return
	}
	nombre := r.PathValue("nombre")
	err := s.op.Demo.CortarNodo(nombre)
	switch {
	case errors.Is(err, cluster.ErrNodoDesconocido):
		responderError(w, http.StatusNotFound, fmt.Sprintf("no hay un nodo conectado llamado %q", nombre))
		return
	case err != nil:
		responderError(w, http.StatusInternalServerError, "no se pudo cortar el nodo: "+err.Error())
		return
	}
	rec, _ := auth.ReclamosDe(r.Context())
	s.op.Log("api: demo: %s cortó el nodo %s", rec.Sub, nombre)
	responder(w, http.StatusAccepted, map[string]string{"nodo": nombre, "estado": "cortado"})
}

// descartarCuerpo lee y descarta el cuerpo de un POST que no lo usa, para que
// el límite de MaxCuerpo también valga ahí (413, como declara el OpenAPI;
// PR-84 H3). Va después de la autenticación: 401/403 antes que 413.
func descartarCuerpo(w http.ResponseWriter, r *http.Request) bool {
	_, err := io.Copy(io.Discard, http.MaxBytesReader(w, r.Body, MaxCuerpo))
	var grande *http.MaxBytesError
	switch {
	case errors.As(err, &grande):
		responderError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("el cuerpo supera %d bytes", MaxCuerpo))
		return false
	case err != nil:
		responderError(w, http.StatusBadRequest, "no se pudo leer el cuerpo")
		return false
	}
	return true
}
