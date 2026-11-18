// Package redis implementa almacen.Cache sobre Redis con el cliente
// go-redis (ADR-0003, ADR-0006). Los plazos son cortos a propósito: si Redis
// tarda, la API calcula la predicción en vez de esperar.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	drv "github.com/redis/go-redis/v9"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
)

// Cache guarda predicciones con TTL. Es segura para uso concurrente (el
// cliente mantiene un pool de conexiones).
type Cache struct{ c *drv.Client }

var _ almacen.Cache = (*Cache)(nil)

// Conectar crea el cliente y comprueba la conexión con PING.
func Conectar(ctx context.Context, direccion string) (*Cache, error) {
	c := drv.NewClient(&drv.Options{
		Addr:         direccion,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
	})
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("redis: sin conexión a %s: %w", direccion, err)
	}
	return &Cache{c: c}, nil
}

// Obtener devuelve false si la clave no existe o expiró.
func (c *Cache) Obtener(ctx context.Context, clave string) ([]byte, bool, error) {
	b, err := c.c.Get(ctx, clave).Bytes()
	if errors.Is(err, drv.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis: %w", err)
	}
	return b, true, nil
}

// Guardar escribe la clave con SET … EX.
func (c *Cache) Guardar(ctx context.Context, clave string, valor []byte, ttl time.Duration) error {
	if err := c.c.Set(ctx, clave, valor, ttl).Err(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	return nil
}

// Cerrar libera el pool de conexiones.
func (c *Cache) Cerrar() error { return c.c.Close() }
