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

// PR-40 ronda 2, H5: con Redis lento, una operación termina cuando vence el
// contexto de la petición y no cuando vence el ReadTimeout del cliente.
func TestRedisRespetaElPlazoDelContexto(t *testing.T) {
	dir := os.Getenv("REDIS_ADDR")
	if dir == "" {
		dir = "localhost:6379"
	}
	proxy := contrato.ProxyLento(t, dir, 120*time.Millisecond)
	c, err := redis.Conectar(context.Background(), proxy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Cerrar() })
	for i := range 5 {
		ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Millisecond)
		ini := time.Now()
		_, _, errO := c.Obtener(ctx, "pred:plazo")
		d := time.Since(ini)
		cancelar()
		if errO == nil || d > 60*time.Millisecond {
			t.Fatalf("lectura %d: %v en %v (plazo 15 ms)", i, errO, d)
		}
		ctx, cancelar = context.WithTimeout(context.Background(), 15*time.Millisecond)
		ini = time.Now()
		errG := c.Guardar(ctx, "pred:plazo", []byte("x"), time.Minute)
		d = time.Since(ini)
		cancelar()
		if errG == nil || d > 60*time.Millisecond {
			t.Fatalf("escritura %d: %v en %v (plazo 15 ms)", i, errG, d)
		}
	}
}
