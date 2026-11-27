package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws"
)

// #69: al publicar un modelo, la API avisa por el WebSocket con sus pesos
// finales. Los de cada evento `epoca` son los de antes de la última
// actualización (w -= tasa·g), así que el sello «bit a bit» del panel admin
// solo puede comparar contra este evento.
func TestEntrenamientoPublicaElModeloPorElWebSocket(t *testing.T) {
	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 {
		w := make([]float64, d+1)
		for j := range w {
			w[j] = 0.1*float64(j) - 1.0/3 // valores sin representación exacta: compara bits, no redondeos
		}
		return w
	}}
	hub := ws.NuevoHub(BuferWS)
	cliente := hub.Suscribir()
	s, ts := servidorDePrueba(t, Opciones{Entrenador: ent, Cluster: clusterFalso{}, Hub: hub})

	cod, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":20,"tasa":2}`)
	if cod != http.StatusAccepted {
		t.Fatalf("crear entrenamiento: %d %v", cod, v)
	}
	close(ent.soltar)

	var e cluster.Evento
	limite := time.After(5 * time.Second)
	for e.Tipo != cluster.EvModelo {
		select {
		case b := <-cliente.Mensajes():
			if err := json.Unmarshal(b, &e); err != nil {
				t.Fatalf("mensaje del hub no es JSON: %v", err)
			}
		case <-limite:
			t.Fatal("no llegó el evento modelo por el WebSocket")
		}
	}

	m := s.ModeloActual()
	if e.V != cluster.VersionEventos || e.Detalle != m.Version || e.Epoca != 20 || e.T.IsZero() {
		t.Fatalf("evento modelo = %+v; quería v=%d, detalle=%s, epoca=20", e, cluster.VersionEventos, m.Version)
	}
	// Bit a bit contra lo que sirve GET /api/v1/modelo (variables + sesgo al final).
	_, pub := pedir(t, "GET", ts.URL+"/api/v1/modelo", "")
	vars := pub["variables"].([]any)
	if len(e.Pesos) != len(vars)+1 {
		t.Fatalf("el evento trae %d pesos; el modelo publicado tiene %d + sesgo", len(e.Pesos), len(vars))
	}
	for j, x := range vars {
		if p := x.(map[string]any)["peso"].(float64); math.Float64bits(p) != math.Float64bits(e.Pesos[j]) {
			t.Fatalf("peso %d: evento %v, publicado %v", j, e.Pesos[j], p)
		}
	}
	if sg := pub["sesgo"].(float64); math.Float64bits(sg) != math.Float64bits(e.Pesos[len(vars)]) {
		t.Fatalf("sesgo: evento %v, publicado %v", e.Pesos[len(vars)], sg)
	}
}

// Un entrenamiento fallido no publica modelo: tampoco lo anuncia.
func TestEntrenamientoFallidoNoAnunciaModelo(t *testing.T) {
	ent := &entrenadorFalso{soltar: make(chan struct{}), falla: errors.New("cluster: no quedan nodos sanos")}
	close(ent.soltar)
	hub := ws.NuevoHub(BuferWS)
	cliente := hub.Suscribir()
	_, ts := servidorDePrueba(t, Opciones{Entrenador: ent, Cluster: clusterFalso{}, Hub: hub})
	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":3}`)
	id := v["id"].(string)
	esperar(t, func() bool {
		_, v := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
		return v["estado"] == "fallido"
	})
	for {
		select {
		case b := <-cliente.Mensajes():
			var e cluster.Evento
			if json.Unmarshal(b, &e) == nil && e.Tipo == cluster.EvModelo {
				t.Fatalf("un entrenamiento fallido anunció un modelo: %s", b)
			}
		default:
			return
		}
	}
}
