package cluster

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// Opciones configura un Coordinador.
type Opciones struct {
	Shards []Shard   // en orden; deben cubrir los bloques 0..B-1 sin huecos
	D      int       // variables por fila (sin el sesgo)
	Config ml.Config // TamBloque (obligatorio), Lambda y pesos por clase

	// TimeoutRegistro es cuánto espera EsperarNodos. Por defecto 30 s.
	TimeoutRegistro time.Duration
	// TimeoutEpoca es el plazo de cada shard: desde que se envía su `epoca`
	// hasta que llega su `gradiente`. En la primera época incluye la carga del
	// archivo. Un nodo con k shards los calcula uno tras otro, así que el plazo
	// debe cubrir k cálculos. Por defecto 60 s.
	TimeoutEpoca time.Duration
	// TimeoutEnvio es el plazo de cada escritura a un nodo. Por defecto 5 s.
	TimeoutEnvio time.Duration
	// Log recibe los sucesos del clúster (registros, descartes, reasignaciones).
	Log func(formato string, args ...any)
	// Observador recibe cada Evento del protocolo (#64), en orden, desde el
	// bucle de épocas. No debe bloquear (ws.Hub y Grabador encolan sin
	// esperar) ni guardar el slice de un evento para modificarlo: recibe copias,
	// así que no puede cambiar el resultado numérico.
	Observador func(Evento)
}

// EstadoNodo es la vista pública de un nodo.
type EstadoNodo struct {
	Nombre  string `json:"nombre"`
	Remoto  string `json:"remoto"`
	Workers int    `json:"workers"`
	Sano    bool   `json:"sano"`
	Carga   int    `json:"carga"`            // shards asignados
	Motivo  string `json:"motivo,omitempty"` // por qué se descartó
	// Metricas son las de la última época que calculó el nodo (#91, panel
	// admin: «uso CPU, latencia, número de nodos, logs»). nil hasta su primera época.
	Metricas *MetricasNodo `json:"metricas,omitempty"`
}

// MetricasNodo resume la última época en que un nodo calculó al menos un shard.
// LatenciaMs va de que el coordinador termina de enviar la primera tarea de la
// época al nodo hasta que recibe su último gradiente; CalculoMs es lo que el
// nodo dice que tardó en calcular; RedMs, la diferencia (red, JSON y cola).
// UsoCPU es la CPU del proceso del nodo durante el cálculo sobre la capacidad
// de sus workers (0..1); nil si el nodo no la informa (p. ej. fuera de Linux).
type MetricasNodo struct {
	Epoca      int      `json:"epoca"`
	Shards     int      `json:"shards"`
	CalculoMs  float64  `json:"calculo_ms"`
	LatenciaMs float64  `json:"latencia_ms"`
	RedMs      float64  `json:"red_ms"`
	UsoCPU     *float64 `json:"uso_cpu,omitempty"`
}

// Estado es una copia del estado del entrenamiento para la API.
type Estado struct {
	Epocas         int          `json:"epocas"` // épocas completadas
	Perdida        float64      `json:"perdida"`
	Nodos          []EstadoNodo `json:"nodos"`
	Reasignaciones int          `json:"reasignaciones"`
	Error          string       `json:"error,omitempty"`
}

// nodo es el estado de un nodo; solo lo toca la goroutine del bucle de épocas.
type nodo struct {
	nombre  string
	workers int
	conn    *protocolo.Conexion
	sano    bool
	motivo  string
	cargado map[int]bool  // shards que el nodo confirmó con `listo`
	met     *MetricasNodo // última época que calculó (#91)
}

// evento es lo que una goroutine lectora le entrega al bucle de épocas.
type evento struct {
	id   int // identificador de la conexión
	conn *protocolo.Conexion
	msg  protocolo.Mensaje
	err  error
}

// Coordinador es el parameter server síncrono.
//
// Concurrencia (ADR-0005):
//
//   - Una goroutine acepta conexiones y lanza una goroutine lectora por
//     conexión. Las lectoras solo decodifican mensajes y los envían al canal
//     eventos; son las únicas que leen de su conexión.
//   - El bucle de épocas (quien llama a EsperarNodos y GradienteErr) es el
//     único que recibe de eventos, el único que escribe en las conexiones y
//     el único dueño del estado del entrenamiento (nodos, duenio, listo,
//     vence, parciales). Por eso ese estado no necesita mutex. muBucle solo
//     impide que dos goroutines hagan de bucle de épocas a la vez.
//   - La API no toca ese estado: lee con Estado() una copia publicada bajo
//     sync.RWMutex.
//   - Toda escritura lleva plazo (TimeoutEnvio); un error de escritura es una
//     caída del nodo, así que enviar nunca bloquea el bucle para siempre.
type Coordinador struct {
	op      Opciones
	ln      net.Listener
	bloques int

	eventos chan evento
	hecho   chan struct{}
	wg      sync.WaitGroup
	cerrar  sync.Once

	muBucle sync.Mutex
	// --- estado del bucle de épocas (protegido por muBucle) ---
	nodos     []*nodo
	porID     map[int]int // conexión → índice en nodos
	duenio    []int       // duenio[s] = nodo que calcula el shard s, o -1
	asignado  []bool      // asignado[s]: s ya tuvo dueño alguna vez
	listo     []bool      // listo[s]: llegó el gradiente de s en esta época
	enviado   []bool      // enviado[s]: s tiene una `epoca` en curso en su dueño
	vence     []time.Time // vence[s]: plazo propio del shard s
	parciales []ml.Parcial
	w         []float64 // pesos de la época en curso
	epoca     int
	reasign   int
	fallo     error
	terminado bool // Terminar ya avisó el fin exitoso

	calcNodo map[int]int64 // µs de cómputo de la época en curso, por nodo
	calcSuma int64         // y la suma de todos los shards
	// Métricas del panel admin (#91), por nodo y época: cuándo se le envió la
	// primera tarea, cuándo llegó su último gradiente, cuántos shards hizo y
	// cuánta CPU informó (cpuNodo < 0: algún shard sin dato).
	envioNodo  map[int]time.Time
	ultimoNodo map[int]time.Time
	shardsNodo map[int]int
	cpuNodo    map[int]int64

	muEstado sync.RWMutex
	estado   Estado
	tiempos  Tiempos

	muConns sync.Mutex
	conns   map[int]*protocolo.Conexion // todas las abiertas, para Cerrar
	nombres map[int]string              // nombre registrado de cada conexión abierta, para CortarNodo
	// bytes de las conexiones ya cerradas, para que Tiempos no los pierda
	bytesEnvCerradas, bytesRecCerradas int64
}

// Nuevo valida las opciones y empieza a aceptar nodos en ln.
func Nuevo(ln net.Listener, op Opciones) (*Coordinador, error) {
	if op.D < 1 || op.Config.TamBloque < 1 || len(op.Shards) == 0 {
		return nil, errors.New("cluster: faltan D, Config.TamBloque o los shards")
	}
	for _, v := range []float64{op.Config.Lambda, op.Config.PesoPos, op.Config.PesoNeg} {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("cluster: hiperparámetro %v (debe ser finito y ≥ 0)", v)
		}
	}
	bloques := 0
	for s, sh := range op.Shards {
		if sh.BloqueInicial != bloques || sh.NumBloques < 1 || sh.N < 1 {
			return nil, fmt.Errorf("cluster: el shard %d empieza en el bloque %d con %d bloques; se esperaba que empezara en %d",
				s, sh.BloqueInicial, sh.NumBloques, bloques)
		}
		bloques += sh.NumBloques
	}
	if op.TimeoutRegistro <= 0 {
		op.TimeoutRegistro = 30 * time.Second
	}
	if op.TimeoutEpoca <= 0 {
		op.TimeoutEpoca = 60 * time.Second
	}
	if op.TimeoutEnvio <= 0 {
		op.TimeoutEnvio = 5 * time.Second
	}
	if op.Log == nil {
		op.Log = func(string, ...any) {}
	}
	c := &Coordinador{
		op:         op,
		ln:         ln,
		bloques:    bloques,
		eventos:    make(chan evento),
		hecho:      make(chan struct{}),
		porID:      map[int]int{},
		calcNodo:   map[int]int64{},
		envioNodo:  map[int]time.Time{},
		ultimoNodo: map[int]time.Time{},
		shardsNodo: map[int]int{},
		cpuNodo:    map[int]int64{},
		duenio:     make([]int, len(op.Shards)),
		asignado:   make([]bool, len(op.Shards)),
		listo:      make([]bool, len(op.Shards)),
		enviado:    make([]bool, len(op.Shards)),
		vence:      make([]time.Time, len(op.Shards)),
		parciales:  make([]ml.Parcial, bloques),
		conns:      map[int]*protocolo.Conexion{},
		nombres:    map[int]string{},
	}
	for s := range c.duenio {
		c.duenio[s] = -1
	}
	for k := range c.parciales {
		c.parciales[k] = ml.NuevoParcial(op.D)
	}
	c.wg.Add(1)
	go c.aceptar()
	return c, nil
}

// aceptar lanza una goroutine lectora por cada conexión entrante.
func (c *Coordinador) aceptar() {
	defer c.wg.Done()
	for id := 0; ; id++ {
		conn, err := c.ln.Accept()
		if err != nil {
			return // el listener se cerró
		}
		pc := protocolo.Nueva(conn)
		c.muConns.Lock()
		c.conns[id] = pc
		c.muConns.Unlock()
		c.wg.Add(1)
		go c.leer(id, pc)
	}
}

// leer es la goroutine lectora de una conexión: exige un `registro` y después
// reenvía cada mensaje al bucle de épocas. Termina con el primer error.
func (c *Coordinador) leer(id int, pc *protocolo.Conexion) {
	defer c.wg.Done()
	defer func() {
		pc.Cerrar()
		c.muConns.Lock()
		e, r := pc.Bytes()
		c.bytesEnvCerradas += e
		c.bytesRecCerradas += r
		delete(c.conns, id)
		delete(c.nombres, id)
		c.muConns.Unlock()
	}()
	m, err := pc.Recibir(c.op.TimeoutRegistro)
	if err != nil || m.Tipo != protocolo.Registro {
		c.op.Log("conexión de %s rechazada: registro inválido (%s): %v", pc.Remoto(), m.Tipo, err)
		return
	}
	c.muConns.Lock()
	c.nombres[id] = m.Nodo
	c.muConns.Unlock()
	if !c.entregar(evento{id: id, conn: pc, msg: m}) {
		return
	}
	for {
		m, err := pc.Recibir(0)
		if !c.entregar(evento{id: id, msg: m, err: err}) || err != nil {
			return
		}
	}
}

// ErrNodoDesconocido: CortarNodo no encontró un nodo conectado con ese nombre.
var ErrNodoDesconocido = errors.New("cluster: no hay un nodo conectado con ese nombre")

// CortarNodo cierra la conexión TCP del nodo, como si se cortara el cable (modo
// demo de la API, #65). No toca el estado del entrenamiento: la lectora de esa
// conexión recibe el error y el bucle de épocas descarta al nodo y reasigna sus
// shards por el camino normal de una caída (ADR-0005). Si no hay entrenamiento
// en curso, el descarte se procesa al empezar el siguiente. Se puede llamar
// desde cualquier goroutine, también desde el Observador.
func (c *Coordinador) CortarNodo(nombre string) error {
	for {
		// El nombre se retira bajo el mutex antes de cerrar: dos cortes
		// concurrentes (o seguidos, con el bucle en reposo) no eligen la misma
		// conexión, y un nodo que se reconecta con el mismo nombre (otro id) es
		// el siguiente en cortarse (PR-84 H1). conns conserva la entrada hasta
		// que la lectora termine, para Tiempos y Cerrar.
		c.muConns.Lock()
		var pc *protocolo.Conexion
		for id, n := range c.nombres {
			if n == nombre {
				pc = c.conns[id]
				delete(c.nombres, id)
				break
			}
		}
		c.muConns.Unlock()
		if pc == nil {
			return fmt.Errorf("%w: %q", ErrNodoDesconocido, nombre)
		}
		err := pc.Cerrar()
		if errors.Is(err, net.ErrClosed) {
			continue // ya la cerró el bucle (descarte) o Cerrar: probar con otra del mismo nombre
		}
		if err == nil {
			c.op.Log("demo: se corta la conexión del nodo %s", nombre)
		}
		return err
	}
}

// entregar pasa un evento al bucle de épocas; devuelve false si el coordinador
// se cerró antes de que alguien lo recibiera.
func (c *Coordinador) entregar(ev evento) bool {
	select {
	case c.eventos <- ev:
		return true
	case <-c.hecho:
		return false
	}
}

// EsperarNodos bloquea hasta que haya n nodos sanos registrados o venza
// TimeoutRegistro (ErrRegistro).
func (c *Coordinador) EsperarNodos(n int) error {
	return c.EsperarNodosContext(context.Background(), n)
}

// EsperarNodosContext es EsperarNodos, pero además vuelve con ctx.Err() si se
// cancela ctx: un servicio que recibe SIGTERM mientras espera nodos no queda
// bloqueado hasta TimeoutRegistro (PR-39 H8). Los nodos ya registrados siguen
// sanos y reciben `fin` con Terminar.
func (c *Coordinador) EsperarNodosContext(ctx context.Context, n int) error {
	c.muBucle.Lock()
	defer c.muBucle.Unlock()
	limite := time.NewTimer(c.op.TimeoutRegistro)
	defer limite.Stop()
	for c.sanos() < n {
		select {
		case ev := <-c.eventos:
			c.atender(ev)
		case <-limite.C:
			return fmt.Errorf("%w: %d de %d nodos en %v", ErrRegistro, c.sanos(), n, c.op.TimeoutRegistro)
		case <-c.hecho:
			return ErrCerrado
		case <-ctx.Done():
			return fmt.Errorf("cluster: espera de nodos cancelada con %d de %d: %w", c.sanos(), n, ctx.Err())
		}
	}
	return nil
}

// GradienteErr implementa ml.CalculadorFalible: ejecuta una época en el clúster
// y devuelve el gradiente completo, idéntico bit a bit al secuencial.
func (c *Coordinador) GradienteErr(w []float64) (ml.Gradiente, error) {
	c.muBucle.Lock()
	defer c.muBucle.Unlock()
	if c.fallo != nil {
		return ml.Gradiente{}, c.fallo
	}
	if len(w) != c.op.D+1 {
		return ml.Gradiente{}, fmt.Errorf("cluster: len(w) = %d, se esperaban %d", len(w), c.op.D+1)
	}
	ini := time.Now()
	clear(c.calcNodo)
	clear(c.envioNodo)
	clear(c.ultimoNodo)
	clear(c.shardsNodo)
	clear(c.cpuNodo)
	c.calcSuma = 0
	c.w = w
	for s := range c.listo {
		c.listo[s], c.enviado[s] = false, false
	}
	pendientes := len(c.op.Shards)
	if err := c.cubrir(); err != nil {
		return ml.Gradiente{}, c.abortar(err)
	}
	for pendientes > 0 {
		plazo := time.NewTimer(time.Until(c.proximoVencimiento()))
		select {
		case ev := <-c.eventos:
			plazo.Stop()
			if c.atender(ev) {
				pendientes--
			}
		case <-plazo.C:
			c.vencer(time.Now())
		case <-c.hecho:
			plazo.Stop()
			return ml.Gradiente{}, ErrCerrado
		}
		// Segunda pasada: recién después de procesar el suceso completo (que
		// pudo descartar a varios nodos) se reasignan los shards sin dueño.
		if err := c.cubrir(); err != nil {
			return ml.Gradiente{}, c.abortar(err)
		}
	}
	// Barrera cumplida: están los parciales de los B bloques. Reducción en
	// orden global de bloque, igual que la versión secuencial.
	iniRed := time.Now()
	g := ml.Finalizar(ml.ReducirEnOrden(c.parciales, c.op.D), c.op.Config, w)
	fin := time.Now()
	c.epoca++
	c.muEstado.Lock()
	c.tiempos.Epocas++
	c.tiempos.SegEpoca += fin.Sub(ini).Seconds()
	c.tiempos.SegReduccion += fin.Sub(iniRed).Seconds()
	var calcMax int64
	for _, us := range c.calcNodo {
		calcMax = max(calcMax, us)
	}
	c.tiempos.SegCalculoMax += float64(calcMax) / 1e6
	c.tiempos.SegCalculoSuma += float64(c.calcSuma) / 1e6
	c.muEstado.Unlock()
	c.cerrarMetricas()
	c.publicar(g.Perdida)
	if c.op.Observador != nil {
		c.emitir(Evento{Tipo: EvEpoca, Epoca: c.epoca, Perdida: g.Perdida, Pesos: append([]float64(nil), w...)})
	}
	return g, nil
}

// cubrir garantiza que cada shard pendiente tenga una `epoca` en curso en un
// nodo sano. Se usa al difundir la época y después de cualquier descarte.
// Devuelve ErrSinNodos si no queda quien calcule.
func (c *Coordinador) cubrir() error {
	for s := 0; s < len(c.op.Shards); s++ {
		if c.listo[s] || c.enviado[s] {
			continue
		}
		if err := c.despachar(s); err != nil {
			return err
		}
		// despachar pudo descartar un nodo y dejar sin dueño shards de índice
		// menor: se vuelve a recorrer desde el principio.
		s = -1
	}
	return nil
}

// despachar envía la tarea del shard s a su dueño; si no tiene dueño sano o el
// envío falla, descarta a ese nodo y prueba con el siguiente de menor carga
// (reintento al asignar, ADR-0005).
func (c *Coordinador) despachar(s int) error {
	for {
		j := c.duenio[s]
		if j < 0 {
			if j = c.elegir(); j < 0 {
				return fmt.Errorf("%w para el shard %d en la época %d", ErrSinNodos, s, c.epoca)
			}
			c.duenio[s] = j
			if c.asignado[s] {
				c.reasign++
				c.op.Log("época %d: shard %d reasignado a %s", c.epoca, s, c.nodos[j].nombre)
				c.emitir(Evento{Tipo: EvReasignacion, Epoca: c.epoca, Nodo: c.nodos[j].nombre, Shard: shard(s)})
			}
			c.asignado[s] = true
		}
		if err := c.enviarTarea(j, s); err != nil {
			c.descartar(j, fmt.Sprintf("error de envío: %v", err))
			continue
		}
		c.enviado[s] = true
		ahora := time.Now()
		c.vence[s] = ahora.Add(c.op.TimeoutEpoca) // plazo propio del shard
		if _, ok := c.envioNodo[j]; !ok {
			c.envioNodo[j] = ahora // la latencia del nodo cuenta desde su primera tarea
		}
		c.emitir(Evento{Tipo: EvTarea, Epoca: c.epoca, Nodo: c.nodos[j].nombre, Shard: shard(s)})
		return nil
	}
}

// elegir devuelve el nodo sano con menos shards asignados; si empatan, el de
// menor índice. Devuelve -1 si no hay nodos sanos.
func (c *Coordinador) elegir() int {
	mejor, cargaMejor := -1, 0
	for j, n := range c.nodos {
		if !n.sano {
			continue
		}
		if carga := c.carga(j); mejor < 0 || carga < cargaMejor {
			mejor, cargaMejor = j, carga
		}
	}
	return mejor
}

func (c *Coordinador) carga(j int) int {
	total := 0
	for _, d := range c.duenio {
		if d == j {
			total++
		}
	}
	return total
}

func (c *Coordinador) sanos() int {
	total := 0
	for _, n := range c.nodos {
		if n.sano {
			total++
		}
	}
	return total
}

// enviarTarea envía `asignacion` (solo si el nodo aún no confirmó ese shard
// con `listo`) y `epoca`. TCP conserva el orden, así que el nodo carga el shard antes
// de calcularlo.
func (c *Coordinador) enviarTarea(j, s int) error {
	n, sh := c.nodos[j], c.op.Shards[s]
	if !n.cargado[s] {
		err := n.conn.Enviar(protocolo.Mensaje{
			Tipo: protocolo.Asignacion, Shard: s, Ruta: sh.Ruta, SHA256: sh.SHA256,
			BloqueInicial: sh.BloqueInicial, NumBloques: sh.NumBloques,
			TamBloque: c.op.Config.TamBloque, D: c.op.D,
			Config: &protocolo.ConfigML{Lambda: c.op.Config.Lambda, PesoPos: c.op.Config.PesoPos, PesoNeg: c.op.Config.PesoNeg},
		}, c.op.TimeoutEnvio)
		if err != nil {
			return err
		}
	}
	return n.conn.Enviar(protocolo.Mensaje{Tipo: protocolo.Epoca, Epoca: c.epoca, Shard: s, W: c.w}, c.op.TimeoutEnvio)
}

// descartar saca a un nodo del clúster: cierra su conexión y deja sin dueño
// todos sus shards. Los que no habían llegado en esta época quedan pendientes
// de reenvío; cubrir los reasigna.
func (c *Coordinador) descartar(j int, motivo string) {
	n := c.nodos[j]
	if !n.sano {
		return
	}
	n.sano, n.motivo = false, motivo
	n.conn.Cerrar()
	for s, d := range c.duenio {
		if d != j {
			continue
		}
		c.duenio[s] = -1
		if !c.listo[s] {
			c.enviado[s] = false
		}
	}
	c.op.Log("época %d: nodo %s descartado: %s", c.epoca, n.nombre, motivo)
	c.publicarNodos()
	if c.op.Observador != nil {
		c.emitir(Evento{Tipo: EvDescarte, Epoca: c.epoca, Nodo: n.nombre, Detalle: motivo, Nodos: c.vistaNodos()})
	}
}

// proximoVencimiento es el menor plazo entre los shards en curso.
func (c *Coordinador) proximoVencimiento() time.Time {
	var menor time.Time
	for s := range c.vence {
		if c.listo[s] || !c.enviado[s] {
			continue
		}
		if menor.IsZero() || c.vence[s].Before(menor) {
			menor = c.vence[s]
		}
	}
	return menor
}

// vencer es la primera pasada tras dispararse el temporizador: descarta a los
// dueños de los shards cuyo propio plazo venció, y solo a ellos. Un nodo sano
// que acaba de recibir un shard reasignado conserva su plazo completo.
func (c *Coordinador) vencer(ahora time.Time) {
	for s := range c.vence {
		if c.listo[s] || !c.enviado[s] || ahora.Before(c.vence[s]) {
			continue
		}
		c.descartar(c.duenio[s], fmt.Sprintf("venció el plazo de %v del shard %d", c.op.TimeoutEpoca, s))
	}
}

// atender procesa un evento de una lectora. Devuelve true si con él se
// completó un shard de la época en curso.
func (c *Coordinador) atender(ev evento) bool {
	if ev.conn != nil { // registro de un nodo nuevo
		c.porID[ev.id] = len(c.nodos)
		c.nodos = append(c.nodos, &nodo{
			nombre: ev.msg.Nodo, workers: ev.msg.Workers, conn: ev.conn, sano: true,
			cargado: map[int]bool{},
		})
		c.op.Log("nodo %s registrado desde %s (%d workers)", ev.msg.Nodo, ev.conn.Remoto(), ev.msg.Workers)
		c.publicarNodos()
		if c.op.Observador != nil {
			c.emitir(Evento{Tipo: EvRegistro, Epoca: c.epoca, Nodo: ev.msg.Nodo, Nodos: c.vistaNodos()})
		}
		return false
	}
	j := c.porID[ev.id]
	n := c.nodos[j]
	if !n.sano {
		return false // respuesta tardía de un nodo ya descartado
	}
	if ev.err != nil {
		c.descartar(j, fmt.Sprintf("conexión perdida: %v", ev.err))
		return false
	}
	m := ev.msg
	if m.Shard >= len(c.op.Shards) {
		c.descartar(j, fmt.Sprintf("%s de un shard inexistente (%d)", m.Tipo, m.Shard))
		return false
	}
	sh := c.op.Shards[m.Shard]
	switch m.Tipo {
	case protocolo.Listo:
		if m.BloqueInicial != sh.BloqueInicial || m.NumBloques != sh.NumBloques || m.N != sh.N || m.D != c.op.D {
			c.descartar(j, fmt.Sprintf("cargó un shard %d distinto del asignado", m.Shard))
			return false
		}
		n.cargado[m.Shard] = true // puede llegar en medio de una época: solo se anota
		return false
	case protocolo.Gradiente:
		if m.Epoca != c.epoca || c.listo[m.Shard] || c.duenio[m.Shard] != j {
			return false // época vieja, duplicado o shard que ya no es suyo
		}
		if m.BloqueInicial != sh.BloqueInicial || len(m.Parciales) != sh.NumBloques || len(m.Parciales[0].Grad) != c.op.D+1 {
			c.descartar(j, fmt.Sprintf("gradiente del shard %d con forma incorrecta", m.Shard))
			return false
		}
		copy(c.parciales[sh.BloqueInicial:], m.Parciales)
		c.listo[m.Shard] = true
		c.calcNodo[j] += m.MicrosCalculo // un nodo con varios shards los calcula en serie
		c.calcSuma += m.MicrosCalculo
		ahora := time.Now()
		c.ultimoNodo[j] = ahora
		c.shardsNodo[j]++
		if m.MicrosCPU > 0 && c.cpuNodo[j] >= 0 {
			c.cpuNodo[j] += m.MicrosCPU
		} else {
			c.cpuNodo[j] = -1 // un shard sin dato: el uso de CPU de la época no se informa
		}
		var idaVuelta int64
		if t, ok := c.envioNodo[j]; ok {
			idaVuelta = ahora.Sub(t).Microseconds()
		}
		c.emitir(Evento{Tipo: EvGradiente, Epoca: c.epoca, Nodo: n.nombre, Shard: shard(m.Shard), Micros: m.MicrosCalculo,
			MicrosCPU: m.MicrosCPU, MicrosIdaVuelta: idaVuelta})
		return true
	case protocolo.Error:
		c.descartar(j, "el nodo informó: "+m.Detalle)
	default:
		c.descartar(j, fmt.Sprintf("mensaje inesperado %q", m.Tipo))
	}
	return false
}

// abortar deja al coordinador en estado de fallo y avisa a los nodos sanos.
func (c *Coordinador) abortar(err error) error {
	c.fallo = err
	c.avisarFin(protocolo.MotivoAbortado)
	c.muEstado.Lock()
	c.estado.Error = err.Error()
	c.muEstado.Unlock()
	c.op.Log("entrenamiento abortado: %v", err)
	c.emitir(Evento{Tipo: EvAbortado, Epoca: c.epoca, Detalle: err.Error()})
	return err
}

func (c *Coordinador) avisarFin(motivo string) {
	for _, n := range c.nodos {
		if n.sano {
			n.conn.Enviar(protocolo.Mensaje{Tipo: protocolo.Fin, Motivo: motivo}, c.op.TimeoutEnvio)
		}
	}
}

// Terminar avisa a los nodos sanos que el entrenamiento terminó bien.
func (c *Coordinador) Terminar() {
	c.muBucle.Lock()
	defer c.muBucle.Unlock()
	// Tras un fallo ya se avisó `fin` (abortado) y se emitió EvAbortado; y un
	// segundo Terminar no repite el aviso ni el evento.
	if c.fallo != nil || c.terminado {
		return
	}
	c.terminado = true
	c.avisarFin(protocolo.MotivoCompleto)
	c.emitir(Evento{Tipo: EvFin, Epoca: c.epoca})
}

// Cerrar deja de aceptar nodos, cierra todas las conexiones y espera a que
// terminen todas las goroutines del coordinador. Es idempotente.
func (c *Coordinador) Cerrar() {
	c.cerrar.Do(func() {
		close(c.hecho)
		c.ln.Close()
		c.muConns.Lock()
		for _, pc := range c.conns {
			pc.Cerrar()
		}
		c.muConns.Unlock()
	})
	c.wg.Wait()
}

// Tiempos acumula, sobre las épocas completadas, cuánto duró cada época en el
// coordinador y qué parte fue cómputo en los nodos y reducción; los bytes son
// los de todas las conexiones con nodos, también las ya cerradas (los totales
// solo crecen). Lo usa el benchmark del clúster (#32):
// red y espera = época − cómputo del nodo más lento − reducción.
type Tiempos struct {
	Epocas         int     `json:"epocas"`
	SegEpoca       float64 `json:"seg_epoca"`
	SegCalculoMax  float64 `json:"seg_calculo_max"`  // suma, por época, del nodo más lento (sus shards en serie)
	SegCalculoSuma float64 `json:"seg_calculo_suma"` // suma de todos los shards
	SegReduccion   float64 `json:"seg_reduccion"`    // ReducirEnOrden + Finalizar
	BytesEnviados  int64   `json:"bytes_enviados"`
	BytesRecibidos int64   `json:"bytes_recibidos"`
}

// Tiempos devuelve los acumulados hasta ahora; es seguro llamarlo desde
// cualquier goroutine.
func (c *Coordinador) Tiempos() Tiempos {
	c.muEstado.RLock()
	t := c.tiempos
	c.muEstado.RUnlock()
	c.muConns.Lock()
	t.BytesEnviados, t.BytesRecibidos = c.bytesEnvCerradas, c.bytesRecCerradas
	for _, pc := range c.conns {
		e, r := pc.Bytes()
		t.BytesEnviados += e
		t.BytesRecibidos += r
	}
	c.muConns.Unlock()
	return t
}

// Estado devuelve una copia del estado publicado; es seguro llamarlo desde
// cualquier goroutine mientras el entrenamiento corre.
func (c *Coordinador) Estado() Estado {
	c.muEstado.RLock()
	defer c.muEstado.RUnlock()
	e := c.estado
	e.Nodos = append([]EstadoNodo(nil), c.estado.Nodos...)
	return e
}

func (c *Coordinador) publicar(perdida float64) {
	nodos := c.vistaNodos()
	c.muEstado.Lock()
	c.estado.Epocas, c.estado.Perdida = c.epoca, perdida
	c.estado.Nodos, c.estado.Reasignaciones = nodos, c.reasign
	c.muEstado.Unlock()
}

func (c *Coordinador) publicarNodos() {
	nodos := c.vistaNodos()
	c.muEstado.Lock()
	c.estado.Nodos, c.estado.Reasignaciones = nodos, c.reasign
	c.muEstado.Unlock()
}

// cerrarMetricas guarda en cada nodo que calculó en esta época sus métricas
// (#91). Solo mide: no toca pesos, parciales ni el orden de la reducción. Cada
// época crea un *MetricasNodo nuevo, así las copias publicadas no cambian.
func (c *Coordinador) cerrarMetricas() {
	for j, calc := range c.calcNodo {
		n := c.nodos[j]
		ini, ok1 := c.envioNodo[j]
		fin, ok2 := c.ultimoNodo[j]
		if !ok1 || !ok2 {
			continue
		}
		lat := fin.Sub(ini).Microseconds()
		m := &MetricasNodo{Epoca: c.epoca, Shards: c.shardsNodo[j], CalculoMs: float64(calc) / 1e3,
			LatenciaMs: float64(lat) / 1e3, RedMs: float64(max(lat-calc, 0)) / 1e3}
		if cpu := c.cpuNodo[j]; cpu > 0 && calc > 0 && n.workers > 0 {
			u := min(float64(cpu)/(float64(calc)*float64(n.workers)), 1)
			m.UsoCPU = &u
		}
		n.met = m
	}
}

func (c *Coordinador) vistaNodos() []EstadoNodo {
	vista := make([]EstadoNodo, len(c.nodos))
	for j, n := range c.nodos {
		vista[j] = EstadoNodo{Nombre: n.nombre, Remoto: n.conn.Remoto(), Workers: n.workers,
			Sano: n.sano, Carga: c.carga(j), Motivo: n.motivo, Metricas: n.met}
	}
	return vista
}
