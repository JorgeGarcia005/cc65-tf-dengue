//go:build integracion

package redis_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis"
)

// Requiere Redis: docker compose -f deploy/docker-compose.bd.yml up -d
func TestRedisCumpleElContrato(t *testing.T) {
	dir := os.Getenv("REDIS_ADDR")
	if dir == "" {
		dir = "localhost:6379"
	}
	c, err := redis.Conectar(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Cerrar() })
	contrato.Cache(t, c, time.Sleep) // TTL real: espera 1,5 s
}
