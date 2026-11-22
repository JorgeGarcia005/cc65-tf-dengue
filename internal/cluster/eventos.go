package cluster

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// VersionEventos del esquema de Evento. La SPA y la réplica la verifican.
const VersionEventos = 1

// Tipos de evento del protocolo (ADR-0005), en el orden en que ocurren.
const (
	EvRegistro     = "registro"     // un nodo se registró
	EvTarea        = "tarea"        // se envió `epoca` (y `asignacion` si hacía falta) de un shard a un nodo
	EvGradiente    = "gradiente"    // llegó el gradiente de un shard
	EvEpoca        = "epoca"        // barrera cumplida: gradiente reducido y pérdida de la época
	EvReasignacion = "reasignacion" // un shard pasó a otro nodo
	EvDescarte     = "descarte"     // un nodo salió del clúster
	EvAbortado     = "abortado"     // el entrenamiento falló
	EvFin          = "fin"          // el entrenamiento terminó bien
)

// Evento es lo que ve el observador del coordinador. Pesos y Nodos son copias:
// el observador no puede alterar el entrenamiento.
type Evento struct {
	V       int          `json:"v"`
	Tipo    string       `json:"tipo"`
	T       time.Time    `json:"t"`
	Epoca   int          `json:"epoca"` // época en curso (para EvEpoca, la que terminó: 1, 2, …)
	Nodo    string       `json:"nodo,omitempty"`
	Shard   *int         `json:"shard,omitempty"`
	Perdida float64      `json:"perdida,omitempty"`
	Pesos   []float64    `json:"pesos,omitempty"` // w usado en la época, con el sesgo al final
	Micros  int64        `json:"micros_calculo,omitempty"`
	Detalle string       `json:"detalle,omitempty"`
	Nodos   []EstadoNodo `json:"nodos,omitempty"`
}

// emitir entrega un evento al observador, si hay. Lo llama solo el bucle de
// épocas (o Terminar, con muBucle), así que los eventos salen en orden.
func (c *Coordinador) emitir(e Evento) {
	if c.op.Observador == nil {
		return
	}
	e.V, e.T = VersionEventos, time.Now()
	c.op.Observador(e)
}

func shard(s int) *int { return &s }

// Grabador escribe los eventos como JSON por línea (reports/eventos_*.jsonl)
// sin frenar al coordinador: Observar los encola en un canal con búfer y una
// goroutine los escribe; si el búfer está lleno, el evento se descarta y se
// cuenta. Cerrar vacía la cola y vuelca el archivo. Observar después de Cerrar
// (o a la vez) no entra en pánico: el evento se cuenta como descartado.
type Grabador struct {
	mu          sync.RWMutex // el envío (RLock) y el cierre del canal (Lock) no se cruzan
	cerrado     bool
	cola        chan Evento
	hecho       chan struct{}
	descartados atomic.Int64
	err         error // lo escribe la goroutine escritora; se lee después de <-hecho
}

// NuevoGrabador empieza a escribir en w.
func NuevoGrabador(w io.Writer, bufer int) *Grabador {
	g := &Grabador{cola: make(chan Evento, max(bufer, 1)), hecho: make(chan struct{})}
	go func() {
		defer close(g.hecho)
		bw := bufio.NewWriter(w)
		enc := json.NewEncoder(bw) // una línea por evento
		for e := range g.cola {
			if g.err == nil {
				g.err = enc.Encode(e)
			}
		}
		if g.err == nil {
			g.err = bw.Flush()
		}
	}()
	return g
}

// Observar se pasa como Opciones.Observador. No bloquea.
func (g *Grabador) Observar(e Evento) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.cerrado {
		g.descartados.Add(1)
		return
	}
	select {
	case g.cola <- e:
	default:
		g.descartados.Add(1)
	}
}

// Descartados devuelve cuántos eventos no se grabaron (cola llena o grabador cerrado).
func (g *Grabador) Descartados() int64 { return g.descartados.Load() }

// Cerrar deja de aceptar eventos, espera a que se escriban los ya encolados y
// devuelve el primer error de escritura. Es idempotente.
func (g *Grabador) Cerrar() error {
	g.mu.Lock()
	if !g.cerrado {
		g.cerrado = true
		close(g.cola)
	}
	g.mu.Unlock()
	<-g.hecho
	return g.err
}
