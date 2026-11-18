//go:build integracion

package mongo_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo"
)

// Requiere MongoDB: docker compose -f deploy/docker-compose.bd.yml up -d
func TestMongoCumpleElContrato(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	base := fmt.Sprintf("cc65_test_%d", time.Now().UnixNano())
	a, err := mongo.Conectar(ctx, uri, base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = a.Borrar(context.Background())
		_ = a.Cerrar(context.Background())
	})
	contrato.Almacen(t, a)
}
