package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
)

const consultaLoreto = `{"departamento":"loreto","ano":2024,"semana":10,"edad":34,"sexo":"m"}`

func predecirConCabecera(t *testing.T, url string) (string, map[string]any) {
	t.Helper()
	res, err := http.Post(url+"/api/v1/predicciones", "application/json", strings.NewReader(consultaLoreto))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var v map[string]any
	if res.StatusCode != 200 {
		t.Fatalf("código %d", res.StatusCode)
	}
	if err := jsonDecode(res, &v); err != nil {
		t.Fatal(err)
	}
	return res.Header.Get("X-Cache"), v
}

// La caché se invalida al publicar un modelo nuevo: la clave lleva la versión.
func TestCacheSeInvalidaAlPublicarModelo(t *testing.T) {
	cache := almacen.NuevaCacheMemoria(time.Now)
	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 {
		w := make([]float64, d+1)
		w[d] = 3 // otro sesgo: otro riesgo
		return w
	}}
	close(ent.soltar)
	s, ts := servidorDePrueba(t, Opciones{Cache: cache, Entrenador: ent})

	x1, v1 := predecirConCabecera(t, ts.URL)
	x2, v2 := predecirConCabecera(t, ts.URL)
	if x1 != "MISS" || x2 != "HIT" || v1["riesgo"] != v2["riesgo"] {
		t.Fatalf("antes de entrenar: %s %s %v %v", x1, x2, v1, v2)
	}
	antes := s.ModeloActual().Version

	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":2}`)
	id := v["id"].(string)
	esperar(t, func() bool {
		_, v := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
		return v["estado"] == Terminado
	})

	x3, v3 := predecirConCabecera(t, ts.URL)
	if x3 != "MISS" || v3["version_modelo"] == antes || v3["riesgo"] == v1["riesgo"] {
		t.Fatalf("tras publicar: %s %v (antes %s, riesgo %v)", x3, v3, antes, v1["riesgo"])
	}
	if x4, _ := predecirConCabecera(t, ts.URL); x4 != "HIT" {
		t.Fatalf("la consulta repetida con el modelo nuevo debía ser HIT, fue %s", x4)
	}
}

type cacheRota struct{}

func (cacheRota) Obtener(context.Context, string) ([]byte, bool, error) {
	return nil, false, errors.New("redis: conexión rechazada")
}
func (cacheRota) Guardar(context.Context, string, []byte, time.Duration) error {
	return errors.New("redis: conexión rechazada")
}

func TestCacheCaidaNoFallaLaPrediccion(t *testing.T) {
	var mu sync.Mutex
	var logs []string
	_, ts := servidorDePrueba(t, Opciones{Cache: cacheRota{}, Log: func(f string, a ...any) {
		mu.Lock()
		logs = append(logs, f)
		mu.Unlock()
	}})
	if x, v := predecirConCabecera(t, ts.URL); x != "MISS" || v["riesgo"] == nil {
		t.Fatalf("%s %v", x, v)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(logs) < 2 {
		t.Fatalf("los errores de la caché debían registrarse en el log: %v", logs)
	}
}

func TestPrediccionesSeRegistranEnElAlmacen(t *testing.T) {
	mem := almacen.NuevaMemoria()
	s, ts := servidorDePrueba(t, Opciones{Almacen: mem, Cache: almacen.NuevaCacheMemoria(time.Now)})
	predecirConCabecera(t, ts.URL)
	predecirConCabecera(t, ts.URL)
	esperar(t, func() bool { return len(mem.Predicciones()) == 2 })
	ps := mem.Predicciones()
	c := ps[0].Consulta
	if c.Departamento != "LORETO" || c.Sexo != "M" || c.EdadAnios != 34 || ps[0].VersionModelo != s.ModeloActual().Version {
		t.Fatalf("registro: %+v", ps[0])
	}
	if ps[0].DesdeCache || !ps[1].DesdeCache {
		t.Fatalf("desde_cache: %v %v", ps[0].DesdeCache, ps[1].DesdeCache)
	}
}

func TestEntrenamientoSePersisteYSeConsultaTrasReinicio(t *testing.T) {
	mem := almacen.NuevaMemoria()
	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 { return make([]float64, d+1) }}
	close(ent.soltar)
	s, ts := servidorDePrueba(t, Opciones{Almacen: mem, Entrenador: ent, Cluster: clusterFalso{}})
	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":4}`)
	id := v["id"].(string)
	esperar(t, func() bool {
		e, ok, _ := mem.Entrenamiento(context.Background(), id)
		return ok && e.Estado == Terminado
	})
	e, _, _ := mem.Entrenamiento(context.Background(), id)
	if len(e.Historia) != 2 || e.Historia[1] != 0.5 || len(e.Nodos) != 1 || e.VersionModelo != s.ModeloActual().Version || e.Fin == nil {
		t.Fatalf("persistido: %+v", e)
	}
	vig, ok, _ := mem.ModeloVigente(context.Background())
	if !ok || vig.Version != s.ModeloActual().Version || vig.EntrenamientoID != id {
		t.Fatalf("modelo vigente: %+v", vig)
	}
	// Un servidor nuevo (reinicio) con el mismo almacén ve el entrenamiento.
	m, err := ModeloDeDocumento(vig)
	if err != nil || m.Version != vig.Version {
		t.Fatalf("ModeloDeDocumento: %v %v", m, err)
	}
	_, ts2 := servidorDePrueba(t, Opciones{Almacen: mem})
	cod, v2 := pedir(t, "GET", ts2.URL+"/api/v1/entrenamientos/"+id, "")
	if cod != 200 || v2["estado"] != Terminado {
		t.Fatalf("tras reinicio: %d %v", cod, v2)
	}
}

// almacenLento bloquea cada registro hasta que se cierre soltar.
type almacenLento struct {
	*almacen.Memoria
	soltar chan struct{}
}

func (a almacenLento) RegistrarPrediccion(ctx context.Context, r almacen.RegistroPrediccion) error {
	select {
	case <-a.soltar:
	case <-ctx.Done():
	}
	return a.Memoria.RegistrarPrediccion(ctx, r)
}

// Con Mongo lento la predicción no espera: el búfer se llena y se descarta.
func TestRegistrosLlenosSeDescartanSinBloquear(t *testing.T) {
	lento := almacenLento{Memoria: almacen.NuevaMemoria(), soltar: make(chan struct{})}
	s, ts := servidorDePrueba(t, Opciones{Almacen: lento})
	defer close(lento.soltar)
	inicio := time.Now()
	for range BufferRegistros + 50 {
		predecirConCabecera(t, ts.URL)
	}
	if d := time.Since(inicio); d > 20*time.Second {
		t.Fatalf("las predicciones esperaron al almacén: %v", d)
	}
	if s.RegistrosDescartados() == 0 {
		t.Fatal("se esperaban registros descartados con el búfer lleno")
	}
}

func jsonDecode(res *http.Response, v any) error {
	return json.NewDecoder(res.Body).Decode(v)
}

// PR-40 H9: la respuesta 202 y el GET antes de la primera época traen listas, no null.
func TestRutasEntrenamientoNuevoTraeListasVacias(t *testing.T) {
	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 { return make([]float64, d+1) }}
	s, ts := servidorDePrueba(t, Opciones{Entrenador: ent})
	_ = s
	defer close(ent.soltar)
	res, err := http.Post(ts.URL+"/api/v1/entrenamientos", "application/json", strings.NewReader(`{"epocas":3}`))
	if err != nil {
		t.Fatal(err)
	}
	var crudo map[string]json.RawMessage
	if err := jsonDecode(res, &crudo); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if string(crudo["historia"]) == "null" || string(crudo["nodos"]) == "null" {
		t.Fatalf("respuesta 202: historia %s, nodos %s", crudo["historia"], crudo["nodos"])
	}
}
