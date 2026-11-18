package almacen

import (
	"context"
	"sync"
	"time"
)

// Memoria implementa Almacen en memoria, protegida por un mutex. La usan los
// tests de la API y el coordinador cuando se arranca sin MongoDB.
// El registro de predicciones es un búfer circular de capacidad acotada: sin
// Mongo no se persiste nada y la RAM no crece con el tráfico (PR-40 H10).
type Memoria struct {
	mu             sync.Mutex
	modelos        map[string]Modelo
	entrenamientos map[string]Entrenamiento
	predicciones   []RegistroPrediccion // búfer circular
	siguiente      int                  // posición donde se escribe el próximo
	limite         int
}

// MaxPrediccionesMemoria es cuántas predicciones retiene NuevaMemoria.
const MaxPrediccionesMemoria = 10000

// NuevaMemoria crea un almacén vacío que retiene las últimas
// MaxPrediccionesMemoria predicciones.
func NuevaMemoria() *Memoria { return NuevaMemoriaConLimite(MaxPrediccionesMemoria) }

// NuevaMemoriaConLimite crea un almacén que retiene las últimas n predicciones.
func NuevaMemoriaConLimite(n int) *Memoria {
	return &Memoria{modelos: map[string]Modelo{}, entrenamientos: map[string]Entrenamiento{}, limite: max(n, 1)}
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
	if len(m.predicciones) < m.limite {
		m.predicciones = append(m.predicciones, r)
	} else {
		m.predicciones[m.siguiente] = r
	}
	m.siguiente = (m.siguiente + 1) % m.limite
	return nil
}

// Predicciones devuelve las retenidas, de la más antigua a la más reciente.
func (m *Memoria) Predicciones() []RegistroPrediccion {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.predicciones) < m.limite {
		return append([]RegistroPrediccion(nil), m.predicciones...)
	}
	return append(append([]RegistroPrediccion(nil), m.predicciones[m.siguiente:]...), m.predicciones[:m.siguiente]...)
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

func (m *Memoria) ModeloDeEntrenamiento(_ context.Context, id string) (Modelo, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mod := range m.modelos {
		if mod.EntrenamientoID == id {
			return mod.Copia(), true, nil
		}
	}
	return Modelo{}, false, nil
}

func (m *Memoria) CrearEntrenamiento(_ context.Context, e Entrenamiento) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entrenamientos[e.ID]; ok {
		return ErrDuplicado
	}
	m.entrenamientos[e.ID] = e.Copia()
	return nil
}

func (m *Memoria) EntrenamientosEnCurso(context.Context) ([]Entrenamiento, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var en []Entrenamiento
	for _, e := range m.entrenamientos {
		if e.Estado == EnCurso {
			en = append(en, e.Copia())
		}
	}
	return en, nil
}
