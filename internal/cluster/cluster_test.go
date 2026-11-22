package cluster

import (
	"errors"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// escenario es un dataset sintético partido en shards en disco, con el
// resultado del entrenamiento secuencial como referencia.
type escenario struct {
	ds     *ml.Dataset
	cfg    ml.Config
	ent    ml.Entrenamiento
	shards []Shard
	ref    ml.Resultado
}

// 10 007 filas con bloques de 512 → 20 bloques, el último incompleto.
func nuevoEscenario(t *testing.T, numShards, epocas int) *escenario {
	t.Helper()
	ds, _ := ml.Sintetico(10_007, 6, 42)
	pos, neg := ml.PesosBalanceados(ds.Y)
	e := &escenario{
		ds:  ds,
		cfg: ml.Config{TamBloque: 512, Lambda: 1e-3, PesoPos: pos, PesoNeg: neg},
		ent: ml.Entrenamiento{Epocas: epocas, Tasa: 0.5},
	}
	partes, err := gold.Particionar(gold.Archivo{Dataset: ds, TamBloque: e.cfg.TamBloque}, numShards)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for s, a := range partes {
		ruta := filepath.Join(dir, fmt.Sprintf("shard-%d.gold", s))
		if err := gold.Escribir(ruta, a); err != nil {
			t.Fatal(err)
		}
		desc, _, err := LeerShard(ruta)
		if err != nil {
			t.Fatal(err)
		}
		e.shards = append(e.shards, desc)
	}
	e.ref = ml.Entrenar(ml.NuevoSecuencial(ds, e.cfg), ds.D, e.ent)
	return e
}

// clusterDePrueba es un coordinador real escuchando en 127.0.0.1.
type clusterDePrueba struct {
	t    *testing.T
	e    *escenario
	c    *Coordinador
	dir  string
	logs *registro
	wg   sync.WaitGroup
}

type registro struct {
	mu     sync.Mutex
	lineas []string
}

func (r *registro) anotar(formato string, args ...any) {
	r.mu.Lock()
	r.lineas = append(r.lineas, fmt.Sprintf(formato, args...))
	r.mu.Unlock()
}

func (r *registro) contiene(texto string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.lineas {
		if strings.Contains(l, texto) {
			return true
		}
	}
	return false
}

func nuevoCluster(t *testing.T, e *escenario, timeoutEpoca time.Duration) *clusterDePrueba {
	t.Helper()
	return nuevoClusterObservado(t, e, timeoutEpoca, nil)
}

// nuevoClusterObservado es nuevoCluster con un observador de eventos (#64).
func nuevoClusterObservado(t *testing.T, e *escenario, timeoutEpoca time.Duration, obs func(Evento)) *clusterDePrueba {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cp := &clusterDePrueba{t: t, e: e, dir: ln.Addr().String(), logs: &registro{}}
	cp.c, err = Nuevo(ln, Opciones{
		Shards: e.shards, D: e.ds.D, Config: e.cfg,
		TimeoutRegistro: 5 * time.Second, TimeoutEpoca: timeoutEpoca, TimeoutEnvio: 2 * time.Second,
		Log: cp.logs.anotar, Observador: obs,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cp.c.Cerrar()
		cp.wg.Wait()
	})
	return cp
}

func (cp *clusterDePrueba) conectar() net.Conn {
	cp.t.Helper()
	conn, err := net.Dial("tcp", cp.dir)
	if err != nil {
		cp.t.Fatal(err)
	}
	return conn
}

// esperarRegistrados bloquea hasta que el coordinador publique n nodos. Los
// tests conectan los nodos de uno en uno para que el índice de cada uno (y por
// tanto el reparto inicial de shards) sea conocido.
func (cp *clusterDePrueba) esperarRegistrados(n int) {
	cp.t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for len(cp.c.Estado().Nodos) < n {
		if time.Now().After(limite) {
			cp.t.Fatalf("no se registraron %d nodos", n)
		}
		time.Sleep(time.Millisecond)
	}
}

// registrar corre EsperarNodos en segundo plano mientras f conecta los nodos.
func (cp *clusterDePrueba) registrar(n int, f func()) {
	cp.t.Helper()
	listo := make(chan error, 1)
	go func() { listo <- cp.c.EsperarNodos(n) }()
	f()
	if err := <-listo; err != nil {
		cp.t.Fatal(err)
	}
}

// nodoReal conecta un Nodo de producción.
func (cp *clusterDePrueba) nodoReal(nombre string, workers int) {
	conn := cp.conectar()
	cp.wg.Add(1)
	go func() {
		defer cp.wg.Done()
		(&Nodo{Nombre: nombre, Workers: workers}).Servir(conn)
	}()
}

// guion describe cómo se porta un nodo de prueba. Calcula bien (con
// ml.ParcialBloque) salvo en lo que el guion le pide hacer mal.
type guion struct {
	demora       time.Duration // tiempo extra por shard calculado
	colgado      bool          // nunca responde a `epoca`
	cerrarTras   time.Duration // cierra la conexión este tiempo después de su primera `epoca`
	cerrarEnEp   int           // cierra la conexión al recibir la `epoca` de esta época (-1 = nunca)
	duplicar     bool          // envía cada gradiente dos veces
	epocaVieja   bool          // responde con el número de la época anterior
	informaError bool          // responde `error` a cada `epoca`
	sinListo     bool          // carga el shard pero no envía `listo`

	asignaciones atomic.Int64 // cuántas `asignacion` recibió
	calculos     atomic.Int64 // cuántos shards calculó
}

func (cp *clusterDePrueba) nodoGuion(nombre string, g *guion) {
	conn := cp.conectar()
	cp.wg.Add(1)
	go func() {
		defer cp.wg.Done()
		defer conn.Close()
		pc := protocolo.Nueva(conn)
		pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Registro, Nodo: nombre, Version: protocolo.Version, Workers: 1}, 0)
		shards := map[int]gold.Archivo{}
		var cfg ml.Config
		var unaVez sync.Once
		for {
			m, err := pc.Recibir(0)
			if err != nil || m.Tipo == protocolo.Fin {
				return
			}
			switch m.Tipo {
			case protocolo.Asignacion:
				g.asignaciones.Add(1)
				desc, a, err := LeerShard(m.Ruta)
				if err != nil {
					cp.t.Errorf("%s: %v", nombre, err)
					return
				}
				shards[m.Shard] = a
				cfg = ml.Config{TamBloque: m.TamBloque, Lambda: m.Config.Lambda, PesoPos: m.Config.PesoPos, PesoNeg: m.Config.PesoNeg}
				if !g.sinListo {
					pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Listo, Shard: m.Shard, BloqueInicial: desc.BloqueInicial,
						NumBloques: desc.NumBloques, N: a.N, D: a.D}, 0)
				}
			case protocolo.Epoca:
				if g.cerrarTras > 0 {
					unaVez.Do(func() { time.AfterFunc(g.cerrarTras, func() { conn.Close() }) })
				}
				switch {
				case g.cerrarEnEp == m.Epoca:
					return
				case g.colgado:
					continue
				case g.informaError:
					pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Error, Epoca: m.Epoca, Shard: m.Shard, Detalle: "fallo simulado"}, 0)
					continue
				}
				a := shards[m.Shard]
				parciales := make([]ml.Parcial, ml.NumBloques(a.N, a.TamBloque))
				for k := range parciales {
					parciales[k] = ml.NuevoParcial(a.D)
					ml.ParcialBloque(a.Dataset, cfg, m.W, k, &parciales[k])
				}
				g.calculos.Add(1)
				time.Sleep(g.demora)
				resp := protocolo.Mensaje{Tipo: protocolo.Gradiente, Epoca: m.Epoca, Shard: m.Shard,
					BloqueInicial: a.BloqueInicial, Parciales: parciales}
				if g.epocaVieja {
					resp.Epoca = max(m.Epoca-1, 0)
					if m.Epoca == 0 {
						resp.Epoca = 99
					}
				}
				pc.Enviar(resp, 0)
				if g.duplicar {
					pc.Enviar(resp, 0)
				}
			}
		}
	}()
}

func nuevoGuion() *guion { return &guion{cerrarEnEp: -1} }

func mismosBits(t *testing.T, contexto string, quiere, tiene []float64) {
	t.Helper()
	if len(quiere) != len(tiene) {
		t.Fatalf("%s: largos distintos %d vs %d", contexto, len(quiere), len(tiene))
	}
	for j := range quiere {
		if math.Float64bits(quiere[j]) != math.Float64bits(tiene[j]) {
			t.Fatalf("%s: posición %d difiere: %.17g vs %.17g", contexto, j, quiere[j], tiene[j])
		}
	}
}

// entrenar corre el entrenamiento distribuido y exige que termine dentro del
// límite: un cuelgue es un fallo del test, no una espera infinita.
func (cp *clusterDePrueba) entrenar(limite time.Duration) (ml.Resultado, error) {
	cp.t.Helper()
	type salida struct {
		res ml.Resultado
		err error
	}
	fin := make(chan salida, 1)
	go func() {
		res, err := ml.EntrenarFalible(cp.c, cp.e.ds.D, cp.e.ent)
		fin <- salida{res, err}
	}()
	select {
	case s := <-fin:
		return s.res, s.err
	case <-time.After(limite):
		cp.t.Fatalf("el entrenamiento no terminó en %v: el coordinador se colgó", limite)
		return ml.Resultado{}, nil
	}
}

func (cp *clusterDePrueba) exigirIgualAlSecuencial(res ml.Resultado, err error) {
	cp.t.Helper()
	if err != nil {
		cp.t.Fatalf("el entrenamiento abortó: %v", err)
	}
	mismosBits(cp.t, "pesos", cp.e.ref.W, res.W)
	mismosBits(cp.t, "historia de pérdida", cp.e.ref.Historia, res.Historia)
}

// ADR-0005, verificación 1: con 1, 2, 3 y 5 nodos por TCP real, los pesos y la
// pérdida de cada época son idénticos bit a bit a los del secuencial.
func TestEquivalenciaClusterSecuencialBitABit(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5} {
		t.Run(fmt.Sprintf("nodos=%d", n), func(t *testing.T) {
			e := nuevoEscenario(t, n, 10)
			cp := nuevoCluster(t, e, 10*time.Second)
			cp.registrar(n, func() {
				for i := range n {
					cp.nodoReal(fmt.Sprintf("nodo-%d", i), 1+i%3)
				}
			})
			res, err := cp.entrenar(30 * time.Second)
			cp.exigirIgualAlSecuencial(res, err)
			cp.c.Terminar()
			if est := cp.c.Estado(); est.Epocas != 10 || est.Reasignaciones != 0 {
				t.Errorf("estado final: %+v", est)
			}
		})
	}
}

// Menos nodos que shards: un nodo calcula varios shards con un único pool.
func TestEquivalenciaConMenosNodosQueShards(t *testing.T) {
	e := nuevoEscenario(t, 5, 6)
	cp := nuevoCluster(t, e, 10*time.Second)
	cp.registrar(2, func() {
		cp.nodoReal("a", 4)
		cp.esperarRegistrados(1)
		cp.nodoReal("b", 2)
	})
	res, err := cp.entrenar(30 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	if est := cp.c.Estado(); est.Nodos[0].Carga != 3 || est.Nodos[1].Carga != 2 {
		t.Errorf("reparto por menor carga: quiere 3 y 2, tiene %d y %d", est.Nodos[0].Carga, est.Nodos[1].Carga)
	}
}

// ADR-0005, verificación 3 (criterio 1 del juez). N = 3, plazo de 200 ms por
// shard. El nodo 1 se cuelga; el nodo 2 cierra la conexión a los 160 ms. El
// shard 2 pasa al nodo 0 con un plazo propio (vence a los 360 ms), así que
// cuando a los 200 ms vence el shard 1 no se descarta al nodo 0, el único sano.
func TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano(t *testing.T) {
	e := nuevoEscenario(t, 3, 3)
	cp := nuevoCluster(t, e, 200*time.Millisecond)
	sano, colgado, caido := nuevoGuion(), nuevoGuion(), nuevoGuion()
	sano.demora = 60 * time.Millisecond
	colgado.colgado = true
	caido.colgado, caido.cerrarTras = true, 160*time.Millisecond
	cp.registrar(3, func() {
		cp.nodoGuion("nodo-0", sano)
		cp.esperarRegistrados(1)
		cp.nodoGuion("nodo-1", colgado)
		cp.esperarRegistrados(2)
		cp.nodoGuion("nodo-2", caido)
	})
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)

	est := cp.c.Estado()
	if !est.Nodos[0].Sano || est.Nodos[1].Sano || est.Nodos[2].Sano {
		t.Fatalf("quiere sano solo al nodo 0: %+v", est.Nodos)
	}
	if est.Nodos[0].Carga != 3 {
		t.Errorf("el nodo 0 debe terminar con los 3 shards, tiene %d", est.Nodos[0].Carga)
	}
	if !strings.Contains(est.Nodos[1].Motivo, "venció el plazo") || !strings.Contains(est.Nodos[2].Motivo, "conexión perdida") {
		t.Errorf("motivos: nodo 1 %q, nodo 2 %q", est.Nodos[1].Motivo, est.Nodos[2].Motivo)
	}
	if !cp.logs.contiene("shard 2 reasignado a nodo-0") || !cp.logs.contiene("shard 1 reasignado a nodo-0") {
		t.Errorf("faltan las reasignaciones en el log: %v", cp.logs.lineas)
	}
}

// ADR-0005, verificación 4 (criterio 2 del juez). N = 4; los nodos 2 y 3 caen
// en la época 0. Sus shards se reparten por menor carga: 2 y 2, no 3 y 1.
func TestBalanceoTrasDobleCaidaReparteDosYDos(t *testing.T) {
	e := nuevoEscenario(t, 4, 4)
	cp := nuevoCluster(t, e, 5*time.Second)
	cae2, cae3 := nuevoGuion(), nuevoGuion()
	cae2.cerrarEnEp, cae3.cerrarEnEp = 0, 0
	cp.registrar(4, func() {
		cp.nodoReal("nodo-0", 2)
		cp.esperarRegistrados(1)
		cp.nodoReal("nodo-1", 2)
		cp.esperarRegistrados(2)
		cp.nodoGuion("nodo-2", cae2)
		cp.esperarRegistrados(3)
		cp.nodoGuion("nodo-3", cae3)
	})
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	est := cp.c.Estado()
	if est.Nodos[0].Carga != 2 || est.Nodos[1].Carga != 2 {
		t.Errorf("quiere carga 2 y 2, tiene %d y %d", est.Nodos[0].Carga, est.Nodos[1].Carga)
	}
	if est.Reasignaciones != 2 {
		t.Errorf("reasignaciones = %d, quiere 2", est.Reasignaciones)
	}
}

// ADR-0005, verificación 5: con un nodo sano al lado, un nodo que se porta mal
// no cambia el resultado ni cuelga el entrenamiento.
func TestNodoFalsoNoCambiaElResultadoNiCuelgaElEntrenamiento(t *testing.T) {
	casos := map[string]func(g *guion){
		"cierra la conexión":         func(g *guion) { g.cerrarEnEp = 1 },
		"no responde":                func(g *guion) { g.colgado = true },
		"responde dos veces":         func(g *guion) { g.duplicar = true },
		"responde con otra época":    func(g *guion) { g.epocaVieja = true },
		"informa un error":           func(g *guion) { g.informaError = true },
		"calcula sin avisar `listo`": func(g *guion) { g.sinListo = true },
	}
	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenario(t, 2, 4)
			cp := nuevoCluster(t, e, 300*time.Millisecond)
			falso := nuevoGuion()
			preparar(falso)
			cp.registrar(2, func() {
				cp.nodoReal("sano", 2)
				cp.esperarRegistrados(1)
				cp.nodoGuion("falso", falso)
			})
			res, err := cp.entrenar(20 * time.Second)
			cp.exigirIgualAlSecuencial(res, err)
		})
	}
}

// Un nodo que envía un gradiente de otra forma (menos bloques) se descarta.
func TestNodoFalsoConGradienteDeFormaIncorrectaSeDescarta(t *testing.T) {
	e := nuevoEscenario(t, 2, 2)
	cp := nuevoCluster(t, e, 5*time.Second)
	cp.registrar(2, func() {
		cp.nodoReal("sano", 2)
		cp.esperarRegistrados(1)
		conn := cp.conectar()
		cp.wg.Add(1)
		go func() {
			defer cp.wg.Done()
			defer conn.Close()
			pc := protocolo.Nueva(conn)
			pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Registro, Nodo: "corto", Version: protocolo.Version}, 0)
			for {
				m, err := pc.Recibir(0)
				if err != nil {
					return
				}
				if m.Tipo == protocolo.Epoca {
					corto := ml.NuevoParcial(e.ds.D) // un solo bloque: válido como mensaje, no para el shard
					corto.Peso = 1
					pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Gradiente, Epoca: m.Epoca, Shard: m.Shard,
						BloqueInicial: e.shards[m.Shard].BloqueInicial, Parciales: []ml.Parcial{corto}}, 0)
				}
			}
		}()
	})
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	if est := cp.c.Estado(); est.Nodos[1].Sano || !strings.Contains(est.Nodos[1].Motivo, "forma incorrecta") {
		t.Errorf("el nodo corto debió descartarse: %+v", est.Nodos[1])
	}
}

func TestSinNodosSanosElEntrenamientoAbortaConErrSinNodos(t *testing.T) {
	e := nuevoEscenario(t, 2, 5)
	cp := nuevoCluster(t, e, 150*time.Millisecond)
	a, b := nuevoGuion(), nuevoGuion()
	a.cerrarEnEp, b.colgado = 2, true
	cp.registrar(2, func() {
		cp.nodoGuion("a", a)
		cp.esperarRegistrados(1)
		cp.nodoGuion("b", b)
	})
	res, err := cp.entrenar(20 * time.Second)
	if !errors.Is(err, ErrSinNodos) {
		t.Fatalf("err = %v, quiere ErrSinNodos", err)
	}
	// Las épocas 0 y 1 sí se completaron (el nodo a calculó los dos shards).
	if len(res.Historia) != 2 {
		t.Errorf("épocas completadas = %d, quiere 2", len(res.Historia))
	}
	mismosBits(t, "historia hasta el aborto", e.ref.Historia[:2], res.Historia)
	if _, err := cp.c.GradienteErr(make([]float64, e.ds.D+1)); !errors.Is(err, ErrSinNodos) {
		t.Errorf("tras abortar, GradienteErr = %v; debe repetir ErrSinNodos", err)
	}
	if est := cp.c.Estado(); !strings.Contains(est.Error, "no quedan nodos sanos") {
		t.Errorf("el estado no publica el aborto: %+v", est)
	}
}

// Un nodo sano recibe `fin{abortado}` y Servir devuelve ErrAbortado.
func TestSinNodosParaUnShardIlegibleElNodoRecibeFinAbortado(t *testing.T) {
	e := nuevoEscenario(t, 1, 2)
	e.shards[0].Ruta += ".no-existe" // ningún nodo podrá cargarlo
	cp := nuevoCluster(t, e, 2*time.Second)
	fin := make(chan error, 2)
	cp.registrar(2, func() {
		for range 2 {
			conn := cp.conectar()
			go func() { fin <- (&Nodo{Nombre: "n"}).Servir(conn) }()
		}
	})
	if _, err := cp.entrenar(20 * time.Second); !errors.Is(err, ErrSinNodos) {
		t.Fatalf("err = %v, quiere ErrSinNodos", err)
	}
	cp.c.Cerrar()
	for range 2 {
		if err := <-fin; err == nil {
			t.Error("Servir devolvió nil en un entrenamiento abortado")
		}
	}
}

func TestRegistroIncompletoVenceElPlazoConErrorClaro(t *testing.T) {
	e := nuevoEscenario(t, 2, 1)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	c, err := Nuevo(ln, Opciones{Shards: e.shards, D: e.ds.D, Config: e.cfg, TimeoutRegistro: 150 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Cerrar()
	conn, _ := net.Dial("tcp", ln.Addr().String())
	defer conn.Close()
	go (&Nodo{Nombre: "unico"}).Servir(conn)
	err = c.EsperarNodos(2)
	if !errors.Is(err, ErrRegistro) || !strings.Contains(err.Error(), "1 de 2 nodos") {
		t.Fatalf("err = %v, quiere ErrRegistro con «1 de 2 nodos»", err)
	}
}

func TestRegistroConVersionDistintaNoCuentaComoNodo(t *testing.T) {
	e := nuevoEscenario(t, 1, 1)
	cp := nuevoCluster(t, e, time.Second)
	conn := cp.conectar()
	defer conn.Close()
	protocolo.Nueva(conn).Enviar(protocolo.Mensaje{Tipo: protocolo.Registro, Nodo: "viejo", Version: protocolo.Version + 1}, 0)
	cp.registrar(1, func() { cp.nodoReal("actual", 1) })
	if est := cp.c.Estado(); len(est.Nodos) != 1 || est.Nodos[0].Nombre != "actual" {
		t.Fatalf("nodos registrados: %+v", est.Nodos)
	}
	limite := time.Now().Add(5 * time.Second)
	for !cp.logs.contiene("rechazada") { // lo anota otra goroutine lectora
		if time.Now().After(limite) {
			t.Fatal("el rechazo no quedó en el log")
		}
		time.Sleep(time.Millisecond)
	}
}

// ADR-0005: `asignacion` solo viaja si el nodo no confirmó ese shard.
func TestShardYaCargadoNoSeVuelveAAsignar(t *testing.T) {
	e := nuevoEscenario(t, 2, 6)
	cp := nuevoCluster(t, e, 5*time.Second)
	a, b := nuevoGuion(), nuevoGuion()
	cp.registrar(2, func() {
		cp.nodoGuion("a", a)
		cp.esperarRegistrados(1)
		cp.nodoGuion("b", b)
	})
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	for nombre, g := range map[string]*guion{"a": a, "b": b} {
		if g.asignaciones.Load() != 1 || g.calculos.Load() != 6 {
			t.Errorf("nodo %s: %d asignaciones y %d cálculos en 6 épocas; quiere 1 y 6",
				nombre, g.asignaciones.Load(), g.calculos.Load())
		}
	}
}

// Un `listo` que llega en medio de la época (tras una reasignación) solo se
// anota: la barrera sigue esperando el gradiente, y en las épocas siguientes
// el shard heredado ya no se reasigna.
func TestListoEnMedioDeLaEpocaNoRompeLaBarrera(t *testing.T) {
	e := nuevoEscenario(t, 2, 5)
	cp := nuevoCluster(t, e, 5*time.Second)
	hereda, cae := nuevoGuion(), nuevoGuion()
	cae.cerrarEnEp = 1
	cp.registrar(2, func() {
		cp.nodoGuion("hereda", hereda)
		cp.esperarRegistrados(1)
		cp.nodoGuion("cae", cae)
	})
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	// 5 épocas del shard 0 + 4 épocas (1 a 4) del shard 1 heredado.
	if hereda.asignaciones.Load() != 2 || hereda.calculos.Load() != 9 {
		t.Errorf("hereda: %d asignaciones y %d cálculos; quiere 2 y 9", hereda.asignaciones.Load(), hereda.calculos.Load())
	}
}

// Un nodo que se registra con el entrenamiento en marcha queda de reserva y
// recibe los shards de un nodo que cae después.
func TestNodoQueSeRegistraTardeQuedaDeReserva(t *testing.T) {
	e := nuevoEscenario(t, 1, 6)
	cp := nuevoCluster(t, e, 5*time.Second)
	primero := nuevoGuion()
	primero.demora, primero.cerrarEnEp = 40*time.Millisecond, 3
	cp.registrar(1, func() { cp.nodoGuion("primero", primero) })
	go func() {
		time.Sleep(50 * time.Millisecond)
		cp.nodoReal("reserva", 2)
	}()
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	if est := cp.c.Estado(); len(est.Nodos) != 2 || est.Nodos[1].Carga != 1 {
		t.Errorf("la reserva debió heredar el shard: %+v", est.Nodos)
	}
}

// La API lee el estado mientras el bucle de épocas lo actualiza: sin carreras
// (-race) y con valores coherentes.
func TestEstadoSePuedeLeerDuranteElEntrenamiento(t *testing.T) {
	e := nuevoEscenario(t, 3, 12)
	cp := nuevoCluster(t, e, 5*time.Second)
	cp.registrar(3, func() {
		for i := range 3 {
			cp.nodoReal(fmt.Sprintf("nodo-%d", i), 2)
		}
	})
	parar := make(chan struct{})
	var lectores sync.WaitGroup
	for range 4 {
		lectores.Add(1)
		go func() {
			defer lectores.Done()
			anterior := 0
			for {
				select {
				case <-parar:
					return
				default:
				}
				est := cp.c.Estado()
				if est.Epocas < anterior || est.Epocas > 12 || len(est.Nodos) != 3 {
					t.Errorf("estado incoherente: %+v", est)
					return
				}
				anterior = est.Epocas
				est.Nodos[0].Nombre = "pisado" // es una copia: no afecta al coordinador
			}
		}()
	}
	res, err := cp.entrenar(30 * time.Second)
	close(parar)
	lectores.Wait()
	cp.exigirIgualAlSecuencial(res, err)
	est := cp.c.Estado()
	if est.Epocas != 12 || est.Perdida != e.ref.Historia[11] {
		t.Errorf("estado final: %+v", est)
	}
	for _, n := range est.Nodos {
		if n.Nombre == "pisado" {
			t.Error("un lector modificó el estado del coordinador a través de su copia")
		}
	}
}

func TestNuevoRechazaShardsConHuecos(t *testing.T) {
	e := nuevoEscenario(t, 3, 1)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	if _, err := Nuevo(ln, Opciones{Shards: []Shard{e.shards[0], e.shards[2]}, D: e.ds.D, Config: e.cfg}); err == nil {
		t.Error("esperaba error: falta el shard 1")
	}
	if _, err := Nuevo(ln, Opciones{Shards: e.shards, D: e.ds.D, Config: ml.Config{TamBloque: 512, Lambda: -1}}); err == nil {
		t.Error("esperaba error: lambda negativo")
	}
}

// El nodo real rechaza un archivo que no es el asignado.
func TestNodoRechazaUnShardConOtraHuella(t *testing.T) {
	e := nuevoEscenario(t, 1, 2)
	e.shards[0].SHA256 = strings.Repeat("0", 64)
	cp := nuevoCluster(t, e, 2*time.Second)
	cp.registrar(1, func() { cp.nodoReal("n", 1) })
	if _, err := cp.entrenar(20 * time.Second); !errors.Is(err, ErrSinNodos) {
		t.Fatalf("err = %v, quiere ErrSinNodos", err)
	}
	if est := cp.c.Estado(); !strings.Contains(est.Nodos[0].Motivo, "sha256") {
		t.Errorf("motivo: %q", est.Nodos[0].Motivo)
	}
}

// Cerrar espera a todas las goroutines del coordinador, termine bien o aborte.
func TestSinFugasDeGoroutinesAlTerminarOAbortar(t *testing.T) {
	antes := runtime.NumGoroutine()
	for _, abortar := range []bool{false, true} {
		e := nuevoEscenario(t, 3, 3)
		cp := nuevoCluster(t, e, 150*time.Millisecond)
		cp.registrar(3, func() {
			for i := range 3 {
				g := nuevoGuion()
				g.colgado = abortar
				cp.nodoGuion(fmt.Sprintf("n%d", i), g)
			}
		})
		_, err := cp.entrenar(20 * time.Second)
		if abortar != (err != nil) {
			t.Fatalf("abortar=%v, err=%v", abortar, err)
		}
		cp.c.Terminar()
		cp.c.Cerrar()
		cp.c.Cerrar() // idempotente
		cp.wg.Wait()
	}
	limite := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > antes {
		if time.Now().After(limite) {
			buf := make([]byte, 1<<16)
			t.Fatalf("quedan %d goroutines (había %d):\n%s", runtime.NumGoroutine(), antes, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Benchmark del clúster (#32): el coordinador separa cómputo, reducción y el
// resto de cada época, y cuenta los bytes de las conexiones.
func TestTiemposSeparanComputoReduccionYBytes(t *testing.T) {
	e := nuevoEscenario(t, 3, 8)
	cp := nuevoCluster(t, e, 10*time.Second)
	cp.registrar(3, func() {
		for i := range 3 {
			cp.nodoReal(fmt.Sprintf("nodo-%d", i), 1)
		}
	})
	res, err := cp.entrenar(30 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	tm := cp.c.Tiempos()
	cp.c.Terminar()
	if tm.Epocas != 8 {
		t.Fatalf("épocas medidas = %d, se esperaban 8", tm.Epocas)
	}
	if !(tm.SegCalculoMax > 0 && tm.SegCalculoMax <= tm.SegCalculoSuma && tm.SegCalculoMax <= tm.SegEpoca) {
		t.Fatalf("cómputo inconsistente: %+v", tm)
	}
	// La reducción dura decenas de µs: con el reloj de Windows puede medirse 0.
	if !(tm.SegReduccion >= 0 && tm.SegReduccion < tm.SegEpoca) {
		t.Fatalf("reducción inconsistente: %+v", tm)
	}
	// Cada época envía los pesos a 3 shards y recibe 3 gradientes.
	if tm.BytesEnviados < int64(8*3*(e.ds.D+1)) || tm.BytesRecibidos <= tm.BytesEnviados/2 {
		t.Fatalf("bytes inconsistentes: %+v", tm)
	}
}

// PR-42 H2: los bytes de una conexión cerrada no se pierden, así que los
// totales solo crecen aunque un nodo caiga o el clúster se cierre.
func TestTiemposConservaLosBytesDeConexionesCerradas(t *testing.T) {
	e := nuevoEscenario(t, 2, 4)
	cp := nuevoCluster(t, e, 10*time.Second)
	cp.registrar(2, func() {
		cp.nodoReal("nodo-0", 1)
		cp.nodoReal("nodo-1", 1)
	})
	res, err := cp.entrenar(30 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	antes := cp.c.Tiempos()
	cp.c.Terminar()
	cp.c.Cerrar()
	despues := cp.c.Tiempos()
	if despues.BytesEnviados < antes.BytesEnviados || despues.BytesRecibidos < antes.BytesRecibidos || antes.BytesRecibidos == 0 {
		t.Fatalf("los bytes bajaron al cerrar: antes %+v, después %+v", antes, despues)
	}
}

// PR-42 H6: con un nodo que calcula varios shards, el cómputo del más lento
// es la suma de sus shards (los calcula en serie), no el shard más largo.
func TestTiemposSumanLosShardsDeUnMismoNodo(t *testing.T) {
	e := nuevoEscenario(t, 3, 4)
	cp := nuevoCluster(t, e, 10*time.Second)
	cp.registrar(1, func() { cp.nodoReal("solo", 1) })
	res, err := cp.entrenar(30 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	tm := cp.c.Tiempos()
	cp.c.Terminar()
	if tm.SegCalculoMax != tm.SegCalculoSuma {
		t.Fatalf("un solo nodo: cómputo del más lento %v, suma de shards %v", tm.SegCalculoMax, tm.SegCalculoSuma)
	}
}
