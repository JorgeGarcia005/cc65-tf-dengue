package cluster

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"
	"testing"
	"time"
)

// recolector guarda los eventos y además intenta alterar los pesos recibidos:
// si el coordinador no pasara copias, el entrenamiento cambiaría.
type recolector struct {
	mu  sync.Mutex
	evs []Evento
}

func (r *recolector) observar(e Evento) {
	for i := range e.Pesos {
		e.Pesos[i] = math.NaN()
	}
	r.mu.Lock()
	r.evs = append(r.evs, e)
	r.mu.Unlock()
}

func (r *recolector) tipos() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var t []string
	for _, e := range r.evs {
		t = append(t, e.Tipo)
	}
	return t
}

func entrenarCon(t *testing.T, e *escenario, n int, obs func(Evento)) *clusterDePrueba {
	t.Helper()
	cp := nuevoClusterObservado(t, e, 10*time.Second, obs)
	cp.registrar(n, func() {
		for i := range n {
			cp.nodoReal(fmt.Sprintf("nodo-%d", i), 2)
			cp.esperarRegistrados(i + 1)
		}
	})
	res, err := cp.entrenar(30 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	cp.c.Terminar()
	return cp
}

// #64: el observador no cambia el resultado. Con 1 y 3 nodos, los pesos y la
// pérdida con observador son idénticos bit a bit a los de sin observador (y
// ambos al secuencial), aunque el observador destroce los pesos que recibe.
func TestObservadorBitABit(t *testing.T) {
	for _, n := range []int{1, 3} {
		t.Run(fmt.Sprintf("nodos=%d", n), func(t *testing.T) {
			e := nuevoEscenario(t, n, 8)
			sin := nuevoClusterObservado(t, e, 10*time.Second, nil)
			sin.registrar(n, func() {
				for i := range n {
					sin.nodoReal(fmt.Sprintf("s-%d", i), 2)
				}
			})
			resSin, err := sin.entrenar(30 * time.Second)
			sin.exigirIgualAlSecuencial(resSin, err)

			var r recolector
			con := nuevoClusterObservado(t, e, 10*time.Second, r.observar)
			con.registrar(n, func() {
				for i := range n {
					con.nodoReal(fmt.Sprintf("c-%d", i), 2)
				}
			})
			resCon, err := con.entrenar(30 * time.Second)
			if err != nil {
				t.Fatal(err)
			}
			mismosBits(t, "pesos con y sin observador", resSin.W, resCon.W)
			mismosBits(t, "pérdida con y sin observador", resSin.Historia, resCon.Historia)
			if len(r.tipos()) == 0 {
				t.Fatal("el observador no recibió eventos")
			}
		})
	}
}

func TestEventosSecuenciaYContenido(t *testing.T) {
	const shards, epocas = 3, 5
	e := nuevoEscenario(t, shards, epocas)
	var r recolector
	entrenarCon(t, e, 2, r.observar)

	cuenta := map[string]int{}
	for _, tp := range r.tipos() {
		cuenta[tp]++
	}
	if cuenta[EvRegistro] != 2 || cuenta[EvTarea] != shards*epocas || cuenta[EvGradiente] != shards*epocas ||
		cuenta[EvEpoca] != epocas || cuenta[EvFin] != 1 || cuenta[EvDescarte] != 0 {
		t.Fatalf("conteo de eventos: %v", cuenta)
	}
	k := 0
	gradientes := 0
	for _, ev := range r.evs {
		if ev.V != VersionEventos || ev.T.IsZero() {
			t.Fatalf("evento sin versión o sin tiempo: %+v", ev)
		}
		switch ev.Tipo {
		case EvGradiente:
			gradientes++
			if ev.Shard == nil || ev.Nodo == "" || ev.Epoca != k {
				t.Fatalf("gradiente mal formado: %+v", ev)
			}
		case EvEpoca:
			k++
			// La barrera: la época k solo termina con los gradientes de todos sus shards.
			if gradientes != shards*k || ev.Epoca != k || len(ev.Pesos) != e.ds.D+1 {
				t.Fatalf("época %d tras %d gradientes: %+v", k, gradientes, ev)
			}
			if math.Float64bits(ev.Perdida) != math.Float64bits(e.ref.Historia[k-1]) {
				t.Fatalf("pérdida de la época %d: %v, quería %v", k, ev.Perdida, e.ref.Historia[k-1])
			}
		}
	}
	if ultimo := r.evs[len(r.evs)-1]; ultimo.Tipo != EvFin {
		t.Fatalf("el último evento debe ser fin: %+v", ultimo)
	}
}

func TestEventosCaidaDeUnNodo(t *testing.T) {
	e := nuevoEscenario(t, 2, 3)
	var r recolector
	cp := nuevoClusterObservado(t, e, 10*time.Second, r.observar)
	caido := nuevoGuion()
	caido.cerrarEnEp = 1
	cp.registrar(2, func() {
		cp.nodoReal("sano", 2)
		cp.esperarRegistrados(1)
		cp.nodoGuion("caido", caido)
	})
	res, err := cp.entrenar(20 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)
	tipos := r.tipos()
	iDesc, iReas := slices.Index(tipos, EvDescarte), slices.Index(tipos, EvReasignacion)
	if iDesc < 0 || iReas < iDesc {
		t.Fatalf("se esperaba descarte y luego reasignación: %v", tipos)
	}
	d := r.evs[iDesc]
	if d.Nodo != "caido" || d.Detalle == "" || len(d.Nodos) != 2 || d.Nodos[1].Sano {
		t.Fatalf("descarte: %+v", d)
	}
	if re := r.evs[iReas]; re.Nodo != "sano" || re.Shard == nil {
		t.Fatalf("reasignación: %+v", re)
	}
}

// proyeccion deja lo que no depende del reloj, de los puertos ni del tiempo de
// cálculo: lo que debe repetirse entre corridas.
func proyeccion(t *testing.T, jsonl []byte) []string {
	t.Helper()
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(jsonl))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var e Evento
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("línea no es JSON: %q", sc.Text())
		}
		sh := -1
		if e.Shard != nil {
			sh = *e.Shard
		}
		out = append(out, fmt.Sprintf("%d %s %d %s %d %x %v", e.V, e.Tipo, e.Epoca, e.Nodo, sh,
			math.Float64bits(e.Perdida), e.Pesos))
	}
	return out
}

// Con un nodo el orden de llegada es fijo: dos grabaciones del mismo
// entrenamiento coinciden salvo tiempos, puertos y microsegundos de cálculo.
func TestGrabadorReproducible(t *testing.T) {
	e := nuevoEscenario(t, 2, 4)
	var grab [2]bytes.Buffer
	for i := range grab {
		g := NuevoGrabador(&grab[i], 1024)
		entrenarCon(t, e, 1, g.Observar)
		if err := g.Cerrar(); err != nil || g.Descartados() != 0 {
			t.Fatalf("grabador: err %v, descartados %d", err, g.Descartados())
		}
	}
	a, b := proyeccion(t, grab[0].Bytes()), proyeccion(t, grab[1].Bytes())
	if len(a) == 0 || !slices.Equal(a, b) {
		t.Fatalf("dos grabaciones difieren:\n%v\n%v", a, b)
	}
}

// Un escritor lento no frena a quien observa: los eventos que no caben se descartan.
func TestGrabadorNoBloquea(t *testing.T) {
	pr, pw := io.Pipe()
	g := NuevoGrabador(pw, 4)
	inicio := time.Now()
	for range 1000 {
		g.Observar(Evento{Tipo: EvEpoca})
	}
	if d := time.Since(inicio); d > 100*time.Millisecond {
		t.Fatalf("Observar tardó %v con el escritor bloqueado", d)
	}
	if g.Descartados() == 0 {
		t.Fatal("con la cola llena debían descartarse eventos")
	}
	go io.Copy(io.Discard, pr) // destraba al escritor
	if err := g.Cerrar(); err != nil {
		t.Fatal(err)
	}
	pr.Close()
}
