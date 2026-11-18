package almacen_test

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato"
)

func TestMemoriaCumpleElContrato(t *testing.T) {
	m := almacen.NuevaMemoria()
	contrato.Almacen(t, m)
	if n := len(m.Predicciones()); n != 16 {
		t.Fatalf("predicciones registradas: %d", n)
	}
}

// relojFalso permite probar el TTL sin dormir.
type relojFalso struct {
	mu sync.Mutex
	t  time.Time
}

func (r *relojFalso) ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.t
}

func (r *relojFalso) avanzar(d time.Duration) {
	r.mu.Lock()
	r.t = r.t.Add(d)
	r.mu.Unlock()
}

func TestCacheMemoriaCumpleElContrato(t *testing.T) {
	r := &relojFalso{t: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)}
	c := almacen.NuevaCacheMemoria(r.ahora)
	contrato.Cache(t, c, r.avanzar)
}

func TestCacheMemoriaConcurrente(t *testing.T) {
	c := almacen.NuevaCacheMemoria(time.Now)
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clave := []string{"a", "b"}[i%2]
			_ = c.Guardar(t.Context(), clave, []byte{byte(i)}, time.Minute)
			_, _, _ = c.Obtener(t.Context(), clave)
		}()
	}
	wg.Wait()
}

func TestClavePrediccionIncluyeVersionYNormaliza(t *testing.T) {
	c := almacen.ConsultaNormalizada{Departamento: "LORETO", Ano: 2024, Semana: 10, EdadAnios: 1, Sexo: "M"}
	a := almacen.ClavePrediccion("v1", c)
	if a != "pred:v1:LORETO:2024:10:1:M" {
		t.Fatalf("clave %q", a)
	}
	if almacen.ClavePrediccion("v2", c) == a {
		t.Fatal("la versión del modelo debe cambiar la clave")
	}
	c.EdadAnios = 8.0 / 12
	if b := almacen.ClavePrediccion("v1", c); b != "pred:v1:LORETO:2024:10:0.6666666666666666:M" {
		t.Fatalf("clave con edad en meses: %q", b)
	}
}

// PR-40 H9: la copia de un entrenamiento nuevo serializa listas vacías, no null.
func TestCopiaConservaListasVacias(t *testing.T) {
	for _, e := range []almacen.Entrenamiento{{ID: "a"}, {ID: "b", Historia: []float64{}}} {
		b, err := json.Marshal(e.Copia())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `"historia":[]`) || !strings.Contains(string(b), `"nodos":[]`) {
			t.Fatalf("JSON de la copia: %s", b)
		}
	}
}
