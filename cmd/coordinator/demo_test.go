package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// eventosWS abre el WebSocket del servicio con un ticket y devuelve un canal
// con los eventos que llegan (cluster.Evento).
func eventosWS(t *testing.T, base, origen string) <-chan cluster.Evento {
	t.Helper()
	codigo, v, _ := pedirJSON(t, "POST", base+"/api/v1/ws/ticket", "")
	if codigo != http.StatusOK {
		t.Fatalf("ticket: %d %v", codigo, v)
	}
	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	fmt.Fprintf(conn, "GET /api/v1/ws?ticket=%s HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\nOrigin: %s\r\n\r\n", v["ticket"], origen)
	r := bufio.NewReader(conn)
	res, err := http.ReadResponse(r, nil)
	if err != nil || res.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake: %v %v", res, err)
	}
	evs := make(chan cluster.Evento, 1024)
	go func() {
		defer close(evs)
		for {
			var cab [2]byte
			if _, err := io.ReadFull(r, cab[:]); err != nil {
				return
			}
			n := int(cab[1] & 0x7f)
			if n == 126 {
				var ext [2]byte
				io.ReadFull(r, ext[:])
				n = int(binary.BigEndian.Uint16(ext[:]))
			}
			p := make([]byte, n)
			if _, err := io.ReadFull(r, p); err != nil || cab[0]&0x0f != 0x1 {
				continue // ping o cierre: sin evento
			}
			var e cluster.Evento
			if json.Unmarshal(p, &e) == nil {
				evs <- e
			}
		}
	}()
	return evs
}

// esperarEvento consume eventos hasta uno que cumpla f.
func esperarEvento(t *testing.T, evs <-chan cluster.Evento, que string, f func(cluster.Evento) bool) cluster.Evento {
	t.Helper()
	limite := time.After(30 * time.Second)
	for {
		select {
		case e, ok := <-evs:
			if !ok {
				t.Fatalf("el WebSocket se cerró esperando %s", que)
			}
			if f(e) {
				return e
			}
		case <-limite:
			t.Fatalf("no llegó %s por el WebSocket", que)
		}
	}
}

// #65 de punta a punta: con -demo, el admin sigue el entrenamiento por el
// WebSocket, corta un nodo desde la API, ve el descarte y la reasignación, y el
// modelo publicado es igual bit a bit al secuencial.
func TestServicioDemoCortarNodoPorLaAPI(t *testing.T) {
	const epocas = 15
	modelo, dir, train := modeloYGold(t, 6_000, 3)
	origen := "http://localhost:8081"
	op := opciones{gold: dir, nodos: 3, lambda: 1e-3, tRegistro: 10 * time.Second, tEpoca: 10 * time.Second,
		origenes: []string{origen}, demo: true}
	base, dirCluster, _ := servicioDePrueba(t, modelo, op)
	esperar(t, "la API responde", func() bool {
		res, err := http.Get(base + "/healthz")
		if err == nil {
			res.Body.Close()
		}
		return err == nil
	})
	evs := eventosWS(t, base, origen)
	lanzarNodos(t, dirCluster, 3)
	for i := range 3 {
		esperarEvento(t, evs, "el registro de los nodos", func(e cluster.Evento) bool {
			return e.Tipo == cluster.EvRegistro && e.Nodo == fmt.Sprintf("nodo-%d", i)
		})
	}

	var ubicacion string
	esperar(t, "el clúster acepta un entrenamiento", func() bool {
		codigo, v, h := pedirJSON(t, "POST", base+"/api/v1/entrenamientos",
			fmt.Sprintf(`{"epocas": %d, "tasa": 0.5, "lambda": 0.001, "ritmo_ms": 40}`, epocas))
		if codigo != http.StatusAccepted {
			t.Fatalf("POST entrenamientos: %d %v", codigo, v)
		}
		ubicacion = h.Get("Location")
		_, e, _ := pedirJSON(t, "GET", base+ubicacion, "")
		return !strings.Contains(fmt.Sprint(e["error"]), "todavía espera")
	})
	esperarEvento(t, evs, "la época 2", func(e cluster.Evento) bool { return e.Tipo == cluster.EvEpoca && e.Epoca == 2 })
	if codigo, v, _ := pedirJSON(t, "POST", base+"/api/v1/demo/nodos/nodo-1/cortar", ""); codigo != http.StatusAccepted {
		t.Fatalf("cortar nodo-1: %d %v", codigo, v)
	}
	d := esperarEvento(t, evs, "el descarte de nodo-1", func(e cluster.Evento) bool { return e.Tipo == cluster.EvDescarte })
	if d.Nodo != "nodo-1" || d.Epoca >= epocas {
		t.Fatalf("descarte: %+v (el ritmo debía dejar épocas por delante)", d)
	}
	esperarEvento(t, evs, "la reasignación", func(e cluster.Evento) bool { return e.Tipo == cluster.EvReasignacion })
	esperarEvento(t, evs, "la última época", func(e cluster.Evento) bool { return e.Tipo == cluster.EvEpoca && e.Epoca == epocas })

	ent := entrenamientoTerminado(t, base, ubicacion)
	if ent["estado"] != "terminado" {
		t.Fatalf("entrenamiento = %v", ent)
	}
	_, m, _ := pedirJSON(t, "GET", base+"/api/v1/modelo", "")
	pos, neg := ml.PesosBalanceados(train.Y)
	cfg := ml.Config{TamBloque: train.TamBloque, Lambda: 1e-3, PesoPos: pos, PesoNeg: neg}
	ref := ml.Entrenar(ml.NuevoSecuencial(train.Dataset, cfg), train.D, ml.Entrenamiento{Epocas: epocas, Tasa: 0.5})
	for j, v := range m["variables"].([]any) {
		if p := v.(map[string]any)["peso"].(float64); math.Float64bits(p) != math.Float64bits(ref.W[j]) {
			t.Fatalf("peso %d tras el corte %v, secuencial %v", j, p, ref.W[j])
		}
	}
	if s := m["sesgo"].(float64); math.Float64bits(s) != math.Float64bits(ref.W[train.D]) {
		t.Fatalf("sesgo tras el corte %v, secuencial %v", s, ref.W[train.D])
	}
	_, c, _ := pedirJSON(t, "GET", base+"/api/v1/cluster", "")
	if c["reasignaciones"].(float64) < 1 {
		t.Fatalf("clúster sin reasignaciones: %v", c)
	}
}

// Sin -demo, el servicio no expone el corte (404) aunque el token sea de admin.
func TestServicioSinDemoNoExponeElCorte(t *testing.T) {
	modelo, dir, _ := modeloYGold(t, 1_000, 1)
	op := opciones{gold: dir, nodos: 1, tRegistro: 10 * time.Second, tEpoca: 10 * time.Second}
	base, _, _ := servicioDePrueba(t, modelo, op)
	esperar(t, "la API responde", func() bool {
		res, err := http.Get(base + "/healthz")
		if err == nil {
			res.Body.Close()
		}
		return err == nil
	})
	if codigo, _, _ := pedirJSON(t, "POST", base+"/api/v1/demo/nodos/nodo-0/cortar", ""); codigo != http.StatusNotFound {
		t.Fatalf("cortar sin -demo: %d, quería 404", codigo)
	}
}
