//go:build integracion

package api

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis"
)

// TestBDDePuntaAPunta usa MongoDB y Redis reales (deploy/docker-compose.bd.yml):
// predice dos veces la misma consulta (MISS y luego HIT desde Redis), entrena
// y comprueba que el modelo, el entrenamiento y las predicciones quedan en Mongo.
// Siempre usa una base temporal que se borra al final: el entrenador es falso y
// su modelo no debe mezclarse con la base real cc65_dengue (PR-40 H4).
func TestBDDePuntaAPunta(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), time.Minute)
	defer cancelar()
	uri, dir := os.Getenv("MONGO_URI"), os.Getenv("REDIS_ADDR")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	if dir == "" {
		dir = "localhost:6379"
	}
	base := fmt.Sprintf("cc65_test_%d", time.Now().UnixNano())
	alm, err := mongo.Conectar(ctx, uri, base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = alm.Borrar(context.Background())
		_ = alm.Cerrar(context.Background())
	})
	cache, err := redis.Conectar(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cache.Cerrar() })

	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 {
		w := make([]float64, d+1)
		w[d] = float64(time.Now().UnixNano()%1000) / 1000 // otra versión en cada corrida
		return w
	}}
	close(ent.soltar)
	s, ts := servidorDePrueba(t, Opciones{Almacen: alm, Cache: cache, Entrenador: ent, Cluster: clusterFalso{}})

	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":2}`)
	id := v["id"].(string)
	esperar(t, func() bool {
		e, ok, _ := alm.Entrenamiento(ctx, id)
		return ok && e.Estado == Terminado
	})
	if x, _ := predecirConCabecera(t, ts.URL); x != "MISS" {
		t.Fatalf("primera consulta con el modelo nuevo: %s", x)
	}
	if x, _ := predecirConCabecera(t, ts.URL); x != "HIT" {
		t.Fatalf("segunda consulta: %s", x)
	}
	s.Cerrar() // vacía el registrador
	vig, ok, err := alm.ModeloVigente(ctx)
	if err != nil || !ok || vig.Version != s.ModeloActual().Version || len(vig.Entrenamientos) != 1 || vig.Entrenamientos[0].ID != id {
		t.Fatalf("modelo vigente en Mongo: %+v %v %v", vig, ok, err)
	}
	if _, err := ModeloDeDocumento(vig); err != nil {
		t.Fatalf("el modelo de Mongo no se puede volver a cargar: %v", err)
	}
	t.Logf("base %s: entrenamiento %s, modelo %s", base, id, vig.Version)
}
