package almacen

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Usuario de la SPA (colección usuarios, ADR-0006 y ADR-0007). Correo es el _id
// ya normalizado. Hash nunca se serializa a JSON: no puede salir por la API.
type Usuario struct {
	Correo string    `json:"correo" bson:"_id"`
	Hash   string    `json:"-" bson:"hash"`
	Rol    string    `json:"rol" bson:"rol"`
	Creado time.Time `json:"creado" bson:"creado"`
}

// Usuarios guarda las cuentas de la SPA. Memoria y mongo.Almacen la implementan.
type Usuarios interface {
	// GuardarUsuario crea la cuenta o la reemplaza (alta y cambio de clave).
	// El correo se normaliza antes de guardar.
	GuardarUsuario(ctx context.Context, u Usuario) error
	// Usuario busca por correo (normalizado); false si no existe.
	Usuario(ctx context.Context, correo string) (Usuario, bool, error)
}

// ErrUsuarioInvalido: correo vacío, sin hash o rol desconocido.
var ErrUsuarioInvalido = errors.New("almacen: usuario sin correo, hash o rol válido")

// NormalizarCorreo quita espacios y pasa a minúsculas: en Mongo el _id
// distingue mayúsculas (ADR-0007).
func NormalizarCorreo(c string) string { return strings.ToLower(strings.TrimSpace(c)) }

// PrepararUsuario normaliza el correo y valida los campos antes de guardar.
func PrepararUsuario(u Usuario) (Usuario, error) {
	u.Correo = NormalizarCorreo(u.Correo)
	if u.Correo == "" || u.Hash == "" || (u.Rol != "admin" && u.Rol != "analista") {
		return Usuario{}, ErrUsuarioInvalido
	}
	return u, nil
}

func (m *Memoria) GuardarUsuario(_ context.Context, u Usuario) error {
	u, err := PrepararUsuario(u)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.usuarios == nil {
		m.usuarios = map[string]Usuario{}
	}
	m.usuarios[u.Correo] = u
	return nil
}

func (m *Memoria) Usuario(_ context.Context, correo string) (Usuario, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.usuarios[NormalizarCorreo(correo)]
	return u, ok, nil
}

var _ Usuarios = (*Memoria)(nil)
