package almacen

import (
	"context"
	"sync"
	"time"
)

// Memoria implementa Almacen en memoria, protegida por un mutex. La usan los
// tests de la API y el coordinador cuando se arranca sin MongoDB.
type Memoria struct {
	mu             sync.Mutex
	modelos        map[string]Modelo
	entrenamientos map[string]Entrenamiento
	predicciones   []RegistroPrediccion
}

// NuevaMemoria crea un almacén vacío.
func NuevaMemoria() *Memoria {
	return &Memoria{modelos: map[string]Modelo{}, entrenamientos: map[string]Entrenamiento{}}
}

func (m *Memoria) GuardarModelo(_ context.Context, mod Modelo) error {
	mod = mod.Copia()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelos[mod.Version] = mod
	return nil
}

func (m *Memoria) ModeloVigente(context.Context) (Modelo, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var vig Modelo
	ok := false
	for _, mod := range m.modelos {
		if !ok || mod.Publicado.After(vig.Publicado) {
			vig, ok = mod, true
		}
	}
	return vig.Copia(), ok, nil
}

func (m *Memoria) GuardarEntrenamiento(_ context.Context, e Entrenamiento) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entrenamientos[e.ID] = e.Copia()
	return nil
}

func (m *Memoria) Entrenamiento(_ context.Context, id string) (Entrenamiento, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entrenamientos[id]
	return e.Copia(), ok, nil
}

func (m *Memoria) RegistrarPrediccion(_ context.Context, r RegistroPrediccion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.predicciones = append(m.predicciones, r)
	return nil
}

// Predicciones devuelve una copia del registro (para tests y evidencias).
func (m *Memoria) Predicciones() []RegistroPrediccion {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]RegistroPrediccion(nil), m.predicciones...)
}

func (m *Memoria) Cerrar(context.Context) error { return nil }

// CacheMemoria implementa Cache en memoria con TTL. ahora se inyecta para
// probar la expiración sin dormir.
type CacheMemoria struct {
	mu      sync.Mutex
	ahora   func() time.Time
	valores map[string]entradaCache
}

type entradaCache struct {
	valor  []byte
	expira time.Time
}

// NuevaCacheMemoria crea una caché vacía; ahora suele ser time.Now.
func NuevaCacheMemoria(ahora func() time.Time) *CacheMemoria {
	return &CacheMemoria{ahora: ahora, valores: map[string]entradaCache{}}
}

func (c *CacheMemoria) Obtener(_ context.Context, clave string) ([]byte, bool, error) {
	ahora := c.ahora()
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.valores[clave]
	if !ok {
		return nil, false, nil
	}
	if !ahora.Before(e.expira) {
		delete(c.valores, clave)
		return nil, false, nil
	}
	return append([]byte(nil), e.valor...), true, nil
}

func (c *CacheMemoria) Guardar(_ context.Context, clave string, valor []byte, ttl time.Duration) error {
	ahora := c.ahora()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.valores[clave] = entradaCache{valor: append([]byte(nil), valor...), expira: ahora.Add(ttl)}
	return nil
}
