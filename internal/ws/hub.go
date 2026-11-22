package ws

import (
	"sync"
	"sync/atomic"
	"time"
)

// Hub reparte mensajes a los clientes suscritos sin bloquear nunca al que
// publica (ADR-0007): cada cliente tiene un canal con búfer y, si está lleno, el
// mensaje se descarta y se cuenta. Así un navegador lento no frena la barrera de
// épocas del coordinador.
type Hub struct {
	mu          sync.Mutex
	clientes    map[*Cliente]struct{}
	cerrado     bool
	activos     sync.WaitGroup // llamadas a Servir en curso; Cerrar las espera
	bufer       int
	descartados atomic.Uint64
}

// Cliente es una suscripción al hub.
type Cliente struct {
	ch          chan []byte
	hecho       chan struct{}
	saliendo    atomic.Bool // la baja vino de Hub.Cerrar: se cierra con 1001
	baja        sync.Once
	descartados atomic.Uint64
}

// NuevoHub crea un hub cuyos clientes guardan hasta bufer mensajes pendientes.
func NuevoHub(bufer int) *Hub {
	return &Hub{clientes: map[*Cliente]struct{}{}, bufer: max(bufer, 1)}
}

// Suscribir agrega un cliente.
// Si el hub ya se cerró, el cliente nace dado de baja.
func (h *Hub) Suscribir() *Cliente {
	c := &Cliente{ch: make(chan []byte, h.bufer), hecho: make(chan struct{})}
	h.mu.Lock()
	cerrado := h.cerrado
	if !cerrado {
		h.clientes[c] = struct{}{}
	}
	h.mu.Unlock()
	if cerrado {
		c.saliendo.Store(true)
		c.baja.Do(func() { close(c.hecho) })
	}
	return c
}

// Cerrar da de baja a todos los clientes, no acepta nuevos y espera a que
// terminen todas las llamadas a Servir. http.Server.Shutdown no cierra las
// conexiones tomadas con Hijack: la API llama a esto al recibir SIGTERM y cada
// escritora manda el cierre 1001 (going away) y cierra el socket. Es idempotente.
func (h *Hub) Cerrar() {
	defer h.activos.Wait()
	h.mu.Lock()
	h.cerrado = true
	todos := make([]*Cliente, 0, len(h.clientes))
	for c := range h.clientes {
		todos = append(todos, c)
	}
	h.mu.Unlock()
	for _, c := range todos {
		c.saliendo.Store(true)
		h.Desuscribir(c)
	}
}

// Desuscribir quita al cliente. Es idempotente. El canal no se cierra (un
// Publicar en curso podría escribir en él); el cliente se entera por Hecho.
func (h *Hub) Desuscribir(c *Cliente) {
	h.mu.Lock()
	delete(h.clientes, c)
	h.mu.Unlock()
	c.baja.Do(func() { close(c.hecho) })
}

// Publicar envía msg a todos los clientes sin bloquear. El slice no debe
// modificarse después: todos los clientes comparten el mismo.
func (h *Hub) Publicar(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clientes {
		select {
		case c.ch <- msg:
		default:
			c.descartados.Add(1)
			h.descartados.Add(1)
		}
	}
}

// Clientes devuelve cuántos clientes hay suscritos.
func (h *Hub) Clientes() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clientes)
}

// Descartados devuelve el total de mensajes descartados por clientes lentos.
func (h *Hub) Descartados() uint64 { return h.descartados.Load() }

// Mensajes es el canal de mensajes pendientes del cliente.
func (c *Cliente) Mensajes() <-chan []byte { return c.ch }

// Hecho se cierra al desuscribir al cliente.
func (c *Cliente) Hecho() <-chan struct{} { return c.hecho }

// Descartados devuelve los mensajes que este cliente perdió por ir lento.
func (c *Cliente) Descartados() uint64 { return c.descartados.Load() }

// Servir conecta una suscripción con una conexión: una goroutine escribe los
// mensajes del hub y manda un ping cada intervaloPing (sin pings si es ≤ 0); la
// goroutine que llama lee del cliente (responde ping, atiende el cierre). Con
// pings, un cliente que no manda nada (ni el pong) en 2·intervaloPing se da por
// caído. Termina cuando el cliente cierra, falla una escritura, vence ese plazo
// o se desuscribe; siempre deja al cliente dado de baja y la conexión cerrada.
func (h *Hub) Servir(conn *Conn, c *Cliente, intervaloPing time.Duration) {
	h.mu.Lock()
	if h.cerrado {
		h.mu.Unlock()
		h.Desuscribir(c)
		_ = conn.Cerrar(CierreSaliendo, "el servidor se apaga")
		return
	}
	h.activos.Add(1) // bajo h.mu: Cerrar no puede empezar a esperar antes del Add
	h.mu.Unlock()
	defer h.activos.Done()
	defer h.Desuscribir(c)

	var tic <-chan time.Time
	if intervaloPing > 0 {
		ping := time.NewTicker(intervaloPing)
		defer ping.Stop()
		tic = ping.C
		conn.plazoLectura = 2 * intervaloPing // antes de lanzar la escritora: sin carrera
	}
	escritor := make(chan struct{})
	go func() {
		defer close(escritor)
		defer func() {
			if c.saliendo.Load() {
				_ = conn.Cerrar(CierreSaliendo, "el servidor se apaga")
			} else {
				_ = conn.Cerrar(CierreNormal, "")
			}
		}()
		for {
			select {
			case m := <-c.ch:
				if conn.EscribirTexto(m) != nil {
					return
				}
			case <-tic:
				if conn.Ping() != nil {
					return
				}
			case <-c.hecho:
				return
			}
		}
	}()
	for {
		if _, err := conn.LeerMensaje(); err != nil {
			break // los mensajes del cliente se ignoran: el canal es solo de bajada
		}
	}
	h.Desuscribir(c)
	<-escritor
}
