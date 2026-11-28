package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"
)

// Enunciado TB2 §6: «Panel administrador: métricas del cluster (uso CPU,
// latencia, número de nodos, logs)». La latencia de la API sale de aquí.
func TestLatenciasPercentilesYVentana(t *testing.T) {
	var l latencias
	if r := l.resumen(); r.Ventana != 0 || !r.CumpleLimiteP95 || r.LimiteP95Ms != 100 {
		t.Fatalf("vacío: %+v", r)
	}
	for i := 1; i <= 100; i++ { // 1..100 ms; la mitad desde la caché
		l.anotar(time.Duration(i)*time.Millisecond, i%2 == 0)
	}
	r := l.resumen()
	if r.Predicciones != 100 || r.Ventana != 100 || r.P50Ms != 50 || r.P95Ms != 95 || r.MaxMs != 100 || r.AciertosCache != 0.5 || !r.CumpleLimiteP95 {
		t.Fatalf("resumen = %+v", r)
	}
	// La ventana se queda con las últimas VentanaLatencias: 2 000 lentas desplazan a las rápidas.
	for i := 0; i < 2000; i++ {
		l.anotar(150*time.Millisecond, false)
	}
	r = l.resumen()
	if r.Predicciones != 2100 || r.Ventana != VentanaLatencias || r.P50Ms != 150 || r.CumpleLimiteP95 || r.AciertosCache != 0 {
		t.Fatalf("tras llenar la ventana: %+v", r)
	}
}

func TestLatenciasConcurrentes(t *testing.T) {
	var l latencias
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				l.anotar(time.Millisecond, false)
				if i%100 == 0 {
					_ = l.resumen()
				}
			}
		}()
	}
	wg.Wait()
	if r := l.resumen(); r.Predicciones != 16*500 || r.Ventana != VentanaLatencias {
		t.Fatalf("resumen = %+v", r)
	}
}

// GET /api/v1/cluster trae, además del estado del clúster, la latencia de las
// predicciones que la API ya atendió.
func TestClusterIncluyeLatenciaDeLaAPI(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{Cluster: clusterFalso{}})
	for i := 0; i < 3; i++ {
		if cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones", consultaLoreto); cod != http.StatusOK {
			t.Fatalf("predicción: %d %v", cod, v)
		}
	}
	cod, v := pedir(t, "GET", ts.URL+"/api/v1/cluster", "")
	if cod != http.StatusOK {
		t.Fatalf("cluster: %d %v", cod, v)
	}
	if _, ok := v["nodos"].([]any); !ok {
		t.Fatalf("se perdió el estado del clúster: %v", v)
	}
	b, _ := json.Marshal(v["api"])
	var m MetricasAPI
	if err := json.Unmarshal(b, &m); err != nil || m.Predicciones != 3 || m.Ventana != 3 || m.P95Ms <= 0 || m.LimiteP95Ms != 100 {
		t.Fatalf("api = %s (%v)", b, err)
	}
}
