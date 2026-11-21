package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
)

// Pruebas de los hallazgos de la ronda 1 del PR-40.

// almacenConFallas envuelve la memoria y falla o demora a pedido.
type almacenConFallas struct {
	*almacen.Memoria
	fallaModelo, fallaFinal, fallaLectura atomic.Bool
	demoraFinal                           time.Duration
	finales                               atomic.Int32
}

func nuevoAlmacenConFallas() *almacenConFallas {
	return &almacenConFallas{Memoria: almacen.NuevaMemoria()}
}

func (a *almacenConFallas) GuardarModelo(ctx context.Context, m almacen.Modelo) error {
	if a.fallaModelo.Load() {
		return errors.New("mongo: escritura rechazada")
	}
	return a.Memoria.GuardarModelo(ctx, m)
}

func (a *almacenConFallas) ModeloVigente(ctx context.Context) (almacen.Modelo, bool, error) {
	if a.fallaLectura.Load() {
		return almacen.Modelo{}, false, errors.New("mongo: sin conexión")
	}
	return a.Memoria.ModeloVigente(ctx)
}

func (a *almacenConFallas) GuardarEntrenamiento(ctx context.Context, e almacen.Entrenamiento) error {
	if e.Estado != almacen.EnCurso {
		time.Sleep(a.demoraFinal)
		if a.fallaFinal.Load() {
			return errors.New("mongo: escritura rechazada")
		}
		defer a.finales.Add(1)
	}
	return a.Memoria.GuardarEntrenamiento(ctx, e)
}

func entrenadorListo() *entrenadorFalso {
	e := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 {
		w := make([]float64, d+1)
		w[d] = 2
		return w
	}}
	close(e.soltar)
	return e
}

func esperarEstado(t *testing.T, url, id, estado string) map[string]any {
	t.Helper()
	var v map[string]any
	esperar(t, func() bool {
		_, v = pedir(t, "GET", url+"/api/v1/entrenamientos/"+id, "")
		return v["estado"] == estado
	})
	return v
}

// --- H1 ---------------------------------------------------------------------

func TestModeloNoGuardadoNoSePublica(t *testing.T) {
	alm := nuevoAlmacenConFallas()
	s, ts := servidorDePrueba(t, Opciones{Almacen: alm, Entrenador: entrenadorListo()})
	antes := s.ModeloActual().Version
	alm.fallaModelo.Store(true)
	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":2}`)
	v = esperarEstado(t, ts.URL, v["id"].(string), Fallido)
	if !strings.Contains(v["error"].(string), "guardar el modelo") || v["version_modelo"] != nil {
		t.Fatalf("entrenamiento: %v", v)
	}
	if s.ModeloActual().Version != antes {
		t.Fatal("se publicó un modelo que el almacén no guardó")
	}
	vig, _, _ := alm.Memoria.ModeloVigente(context.Background())
	if vig.Version != antes {
		t.Fatalf("vigente en el almacén: %s, servido: %s", vig.Version, antes)
	}
}

func TestEstadoFinalNoGuardadoSeReconciliaAlReiniciar(t *testing.T) {
	alm := nuevoAlmacenConFallas()
	alm.fallaFinal.Store(true)
	s, ts := servidorDePrueba(t, Opciones{Almacen: alm, Entrenador: entrenadorListo(), Cluster: clusterFalso{}})
	// 7 épocas: no es múltiplo de PersistirCada, así que el progreso tampoco quedó guardado.
	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":7}`)
	id := v["id"].(string)
	antes := esperarEstado(t, ts.URL, id, Terminado)
	version := s.ModeloActual().Version
	if e, _, _ := alm.Memoria.Entrenamiento(context.Background(), id); e.Estado != EnCurso {
		t.Fatalf("la prueba supone que el estado final no se guardó: %s", e.Estado)
	}
	// Reinicio con el almacén ya sano: el modelo guardado prueba que terminó.
	alm.fallaFinal.Store(false)
	s2, ts2 := servidorDePrueba(t, Opciones{Almacen: alm})
	v2 := esperarEstado(t, ts2.URL, id, Terminado)
	if v2["version_modelo"] != version || s2.ModeloActual().Version != version {
		t.Fatalf("tras reiniciar: %v, servido %s, esperado %s", v2, s2.ModeloActual().Version, version)
	}
	// PR-40 ronda 2, H1: el reinicio conserva las métricas finales, no solo la versión.
	for _, campo := range []string{"epoca", "historia", "perdida", "nodos", "reasignaciones", "fin", "estado"} {
		if fmt.Sprint(antes[campo]) != fmt.Sprint(v2[campo]) {
			t.Errorf("%s: antes del cierre %v, tras reiniciar %v", campo, antes[campo], v2[campo])
		}
	}
	if fmt.Sprint(v2["historia"]) != "[0.69 0.5]" || v2["epoca"].(float64) != 7 || len(v2["nodos"].([]any)) != 1 {
		t.Fatalf("tras reiniciar: %v", v2)
	}
}

func TestEntrenamientoInterrumpidoQuedaFallidoAlReiniciar(t *testing.T) {
	mem := almacen.NuevaMemoria()
	_ = mem.GuardarEntrenamiento(context.Background(), almacen.Entrenamiento{ID: "ent-viejo", Estado: EnCurso, Inicio: time.Now().UTC()})
	_, ts := servidorDePrueba(t, Opciones{Almacen: mem})
	v := esperarEstado(t, ts.URL, "ent-viejo", Fallido)
	if !strings.Contains(v["error"].(string), "interrumpido") {
		t.Fatalf("%v", v)
	}
}

// --- H2 ---------------------------------------------------------------------

func TestArranqueUsaElModeloVigenteDelAlmacen(t *testing.T) {
	archivo, _ := CargarModelo(modeloDePrueba(t))
	enMongo, err := archivo.conPesos(slices.Repeat([]float64{0.3}, len(archivo.Pesos)), "2026-11-17T10:00:00-05:00")
	if err != nil {
		t.Fatal(err)
	}
	mem := almacen.NuevaMemoria()
	_ = mem.GuardarModelo(context.Background(), enMongo.Documento(time.Now().UTC()))
	s, ts := servidorDePrueba(t, Opciones{Almacen: mem})
	if s.ModeloActual().Version != enMongo.Version || s.ModeloActual().Version == archivo.Version {
		t.Fatalf("servido %s; en Mongo %s; archivo %s", s.ModeloActual().Version, enMongo.Version, archivo.Version)
	}
	_, v := predecirConCabecera(t, ts.URL)
	edad := 34.0
	p, _ := enMongo.Predecir(Consulta{Departamento: "LORETO", Ano: 2024, Semana: 10, Edad: &edad, Sexo: "M"})
	if v["riesgo"] != p.Riesgo || v["version_modelo"] != enMongo.Version {
		t.Fatalf("predicción %v, esperada %+v", v, p)
	}
}

func TestArranqueSinModeloEnAlmacenGuardaElDelArchivo(t *testing.T) {
	mem := almacen.NuevaMemoria()
	s, _ := servidorDePrueba(t, Opciones{Almacen: mem})
	vig, ok, _ := mem.ModeloVigente(context.Background())
	if !ok || vig.Version != s.ModeloActual().Version {
		t.Fatalf("el modelo del archivo no quedó como vigente: %+v ok=%v", vig, ok)
	}
}

func TestArranqueConErrorDeLecturaFalla(t *testing.T) {
	alm := nuevoAlmacenConFallas()
	alm.fallaLectura.Store(true)
	m, _ := CargarModelo(modeloDePrueba(t))
	if _, err := Nuevo(Opciones{Modelo: m, Almacen: alm}); err == nil || !strings.Contains(err.Error(), "sin conexión") {
		t.Fatalf("se esperaba error de arranque, obtuve %v", err)
	}
}

// --- H3 ---------------------------------------------------------------------

func TestEntrenamientosDeDosInstanciasNoColisionan(t *testing.T) {
	mem := almacen.NuevaMemoria()
	_, ts1 := servidorDePrueba(t, Opciones{Almacen: mem, Entrenador: entrenadorListo()})
	_, ts2 := servidorDePrueba(t, Opciones{Almacen: mem, Entrenador: entrenadorListo()})
	_, v1 := pedir(t, "POST", ts1.URL+"/api/v1/entrenamientos", `{"epocas":3}`)
	_, v2 := pedir(t, "POST", ts2.URL+"/api/v1/entrenamientos", `{"epocas":9}`)
	if v1["id"] == v2["id"] {
		t.Fatalf("mismo id en dos instancias: %v", v1["id"])
	}
	e1, ok, _ := mem.Entrenamiento(context.Background(), v1["id"].(string))
	if !ok || e1.Epocas != 3 {
		t.Fatalf("primer entrenamiento: %+v", e1)
	}
}

func TestColisionAlCrearPreservaElExistente(t *testing.T) {
	mem := almacen.NuevaMemoria()
	_, ts1 := servidorDePrueba(t, Opciones{Almacen: mem, Entrenador: entrenadorListo()})
	_, v1 := pedir(t, "POST", ts1.URL+"/api/v1/entrenamientos", `{"epocas":3}`)
	id := v1["id"].(string)
	s2, ts2 := servidorDePrueba(t, Opciones{Almacen: mem, Entrenador: entrenadorListo()})
	s2.ents.nuevoID = func() string { return id } // fuerza la colisión
	if cod, v := pedir(t, "POST", ts2.URL+"/api/v1/entrenamientos", `{"epocas":9}`); cod != http.StatusServiceUnavailable {
		t.Fatalf("colisión: %d %v", cod, v)
	}
	if e, _, _ := mem.Entrenamiento(context.Background(), id); e.Epocas != 3 {
		t.Fatalf("la colisión sobrescribió el existente: %+v", e)
	}
	// La instancia 2 queda libre para otro entrenamiento.
	s2.ents.nuevoID = nuevoIDEntrenamiento
	if cod, _ := pedir(t, "POST", ts2.URL+"/api/v1/entrenamientos", `{"epocas":1}`); cod != http.StatusAccepted {
		t.Fatalf("tras la colisión: %d", cod)
	}
}

// --- H5 ---------------------------------------------------------------------

// cacheLenta demora cada operación (respetando el contexto, como go-redis).
type cacheLenta struct {
	demoraLeer, demoraGuardar time.Duration
	errGuardar                bool
	guardados                 atomic.Int32
}

func dormir(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *cacheLenta) Obtener(ctx context.Context, _ string) ([]byte, bool, error) {
	if err := dormir(ctx, c.demoraLeer); err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

func (c *cacheLenta) Guardar(ctx context.Context, _ string, _ []byte, _ time.Duration) error {
	c.guardados.Add(1)
	if c.errGuardar {
		return errors.New("redis: error")
	}
	return dormir(ctx, c.demoraGuardar)
}

func p95(t *testing.T, url string, n int) time.Duration {
	t.Helper()
	var ds []time.Duration
	for range n {
		ini := time.Now()
		predecirConCabecera(t, url)
		ds = append(ds, time.Since(ini))
	}
	slices.Sort(ds)
	return ds[(n*95+99)/100-1]
}

func TestCacheLentaNoConsumeElPresupuesto(t *testing.T) {
	lectura := &cacheLenta{demoraLeer: 120 * time.Millisecond, errGuardar: true}
	_, ts := servidorDePrueba(t, Opciones{Cache: lectura})
	if d := p95(t, ts.URL, 20); d >= 100*time.Millisecond {
		t.Fatalf("lectura lenta: p95 %v", d)
	}
	if n := lectura.guardados.Load(); n != 0 {
		t.Fatalf("tras un error de lectura no debía intentar guardar (%d intentos)", n)
	}
	escritura := &cacheLenta{demoraGuardar: 120 * time.Millisecond}
	_, ts2 := servidorDePrueba(t, Opciones{Cache: escritura})
	if d := p95(t, ts2.URL, 20); d >= 100*time.Millisecond {
		t.Fatalf("escritura lenta: p95 %v", d)
	}
}

// --- H6 ---------------------------------------------------------------------

func TestCerrarEsperaLaEscrituraFinalDelEntrenamiento(t *testing.T) {
	alm := nuevoAlmacenConFallas()
	alm.demoraFinal = 300 * time.Millisecond
	ent := &entrenadorFalso{soltar: make(chan struct{})} // no se suelta: termina al cancelar
	s, ts := servidorDePrueba(t, Opciones{Almacen: alm, Entrenador: ent})
	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":5}`)
	id := v["id"].(string)
	esperar(t, func() bool {
		_, v := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
		return v["epoca"].(float64) == 1
	})
	s.Cerrar()
	e, _, _ := alm.Memoria.Entrenamiento(context.Background(), id)
	if e.Estado != Fallido || alm.finales.Load() != 1 {
		t.Fatalf("Cerrar volvió antes de guardar el estado final: %s (%d escrituras finales)", e.Estado, alm.finales.Load())
	}
	if cod, _ := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":1}`); cod != http.StatusServiceUnavailable {
		t.Fatalf("tras Cerrar no debe admitir entrenamientos: %d", cod)
	}
}

func TestCerrarConColaLlenaRespetaElPlazo(t *testing.T) {
	lento := almacenLento{Memoria: almacen.NuevaMemoria(), soltar: make(chan struct{})} // nunca se suelta
	s, ts := servidorDePrueba(t, Opciones{Almacen: lento, PlazoCierre: 300 * time.Millisecond})
	for range BufferRegistros + 10 {
		predecirConCabecera(t, ts.URL)
	}
	listo := make(chan struct{})
	go func() { s.Cerrar(); close(listo) }()
	select {
	case <-listo:
	case <-time.After(3 * time.Second):
		close(lento.soltar) // desbloquea el almacén para que el test no quede colgado
		<-listo
		t.Fatal("Cerrar no respetó el plazo de 300 ms con la cola llena")
	}
}
