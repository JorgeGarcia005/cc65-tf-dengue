package cluster

import (
	"runtime"
	"testing"
	"time"
)

// #91 (enunciado TB2 §6, panel admin: «uso CPU, latencia, número de nodos,
// logs»): tras entrenar, cada nodo publica las métricas de su última época, y
// medirlas no cambia el resultado (sigue igual bit a bit al secuencial).
func TestMetricasPorNodoTrasEntrenar(t *testing.T) {
	e := nuevoEscenario(t, 3, 4)
	var gradientes []Evento
	cp := nuevoClusterObservado(t, e, 10*time.Second, func(ev Evento) {
		if ev.Tipo == EvGradiente {
			gradientes = append(gradientes, ev) // el observador lo llama el bucle de épocas: sin carrera
		}
	})
	cp.registrar(2, func() {
		cp.nodoReal("a", 2)
		cp.esperarRegistrados(1)
		cp.nodoReal("b", 1)
	})
	res, err := cp.entrenar(30 * time.Second)
	cp.exigirIgualAlSecuencial(res, err)

	est := cp.c.Estado()
	shards := 0
	for _, n := range est.Nodos {
		m := n.Metricas
		if m == nil {
			t.Fatalf("nodo %s sin métricas", n.Nombre)
		}
		if m.Epoca != est.Epocas || m.Shards < 1 || m.CalculoMs <= 0 || m.LatenciaMs < m.CalculoMs || m.RedMs < 0 {
			t.Fatalf("métricas de %s: %+v (épocas %d)", n.Nombre, *m, est.Epocas)
		}
		if runtime.GOOS == "linux" {
			if m.UsoCPU == nil || *m.UsoCPU < 0 || *m.UsoCPU > 1 {
				t.Fatalf("uso de CPU de %s: %v", n.Nombre, m.UsoCPU)
			}
		}
		shards += m.Shards
	}
	if shards != 3 {
		t.Fatalf("las métricas cuentan %d shards en la última época; hay 3", shards)
	}
	if len(gradientes) != 3*4 {
		t.Fatalf("%d eventos gradiente; quería 12", len(gradientes))
	}
	for _, g := range gradientes {
		if g.MicrosIdaVuelta < g.Micros || (runtime.GOOS == "linux" && (g.MicrosCPU == nil || *g.MicrosCPU < 0)) {
			t.Fatalf("gradiente sin métricas: %+v", g)
		}
	}
}
