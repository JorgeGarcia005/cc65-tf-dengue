package api

import (
	"math"
	"slices"
	"sync"
	"time"
)

// LimiteP95 es el requisito del enunciado para la API de predicción.
const LimiteP95 = 100 * time.Millisecond

// VentanaLatencias es cuántas predicciones recientes entran en los percentiles
// que ve el panel admin. Con ~10 000 peticiones por segundo en el benchmark,
// 1024 cubren la última décima de segundo; en la demo, varios minutos.
const VentanaLatencias = 1024

// MetricasAPI es la latencia de POST /api/v1/predicciones medida dentro del
// servidor (del inicio del handler a la respuesta escrita), en las últimas
// VentanaLatencias peticiones. No incluye la red ni la cola del cliente: el
// p95 del benchmark (cmd/benchapi) es el que se mide como el cliente.
type MetricasAPI struct {
	Predicciones    int64   `json:"predicciones"`   // desde que arrancó el servidor
	Ventana         int     `json:"ventana"`        // cuántas entran en los percentiles
	P50Ms           float64 `json:"p50_ms"`         // mediana en la ventana
	P95Ms           float64 `json:"p95_ms"`         // percentil 95 en la ventana
	MaxMs           float64 `json:"max_ms"`         // la más lenta en la ventana
	AciertosCache   float64 `json:"aciertos_cache"` // fracción servida desde Redis en la ventana
	NoRegistradas   int64   `json:"no_registradas"` // descartadas por el búfer hacia MongoDB (ADR-0006)
	LimiteP95Ms     float64 `json:"limite_p95_ms"`  // el requisito del enunciado
	CumpleLimiteP95 bool    `json:"cumple_limite_p95"`
}

// latencias es un búfer circular protegido por mutex: cada predicción anota
// su duración (O(1)); el percentil se calcula al consultar, sobre una copia.
type latencias struct {
	mu     sync.Mutex
	dur    [VentanaLatencias]time.Duration
	hit    [VentanaLatencias]bool
	sig    int   // próxima posición a escribir
	llenas int   // cuántas posiciones tienen dato (≤ VentanaLatencias)
	total  int64 // predicciones desde el arranque
}

func (l *latencias) anotar(d time.Duration, desdeCache bool) {
	l.mu.Lock()
	l.dur[l.sig], l.hit[l.sig] = d, desdeCache
	l.sig = (l.sig + 1) % VentanaLatencias
	l.llenas = min(l.llenas+1, VentanaLatencias)
	l.total++
	l.mu.Unlock()
}

func (l *latencias) resumen() MetricasAPI {
	l.mu.Lock()
	n, total := l.llenas, l.total
	ms := make([]float64, n)
	aciertos := 0
	for i := 0; i < n; i++ {
		ms[i] = float64(l.dur[i]) / float64(time.Millisecond)
		if l.hit[i] {
			aciertos++
		}
	}
	l.mu.Unlock()
	m := MetricasAPI{Predicciones: total, Ventana: n, LimiteP95Ms: float64(LimiteP95 / time.Millisecond)}
	if n == 0 {
		m.CumpleLimiteP95 = true // sin predicciones no hay incumplimiento que mostrar
		return m
	}
	slices.Sort(ms)
	m.P50Ms, m.P95Ms, m.MaxMs = rango(ms, 0.50), rango(ms, 0.95), ms[n-1]
	m.AciertosCache = float64(aciertos) / float64(n)
	m.CumpleLimiteP95 = m.P95Ms < m.LimiteP95Ms
	return m
}

// rango es el percentil por el método del rango más cercano (el mismo que
// usa cmd/benchapi): el menor valor con al menos q·n observaciones ≤ él.
func rango(ordenados []float64, q float64) float64 {
	k := int(math.Ceil(q*float64(len(ordenados)))) - 1
	return ordenados[max(0, min(k, len(ordenados)-1))]
}
