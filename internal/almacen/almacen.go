// Package almacen define la persistencia de la API (ADR-0006): MongoDB para
// modelos, entrenamientos y el registro de predicciones, y Redis como caché
// de predicciones. La API solo conoce las interfaces Almacen y Cache; Memoria
// y CacheMemoria las implementan sin contenedores para los tests, y los
// subpaquetes mongo y redis con los drivers oficiales.
package almacen

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
)

// Modelo publicado (colección modelos). Version es el _id.
type Modelo struct {
	Version     string            `json:"version" bson:"_id"`
	Pesos       []float64         `json:"pesos" bson:"pesos"`
	Esquema     *features.Esquema `json:"esquema" bson:"esquema"`
	Umbral      float64           `json:"umbral" bson:"umbral"`
	AjustePrior float64           `json:"ajuste_prior" bson:"ajuste_prior"`
	Generado    string            `json:"generado" bson:"generado"`
	// Entrenamientos que publicaron esta versión, con su resumen final. Dos
	// entrenamientos con los mismos pesos comparten versión y ambos quedan
	// aquí (PR-40 ronda 2, H11). GuardarModelo agrega, nunca reemplaza.
	Entrenamientos []ResumenEntrenamiento `json:"entrenamientos" bson:"entrenamientos"`
	// Publicado es la publicación más reciente de esta versión.
	Publicado time.Time `json:"publicado" bson:"publicado"`
}

// ResumenEntrenamiento es el estado final de un entrenamiento. Se guarda en
// la misma escritura que publica el modelo: si después se pierde la escritura
// terminal del entrenamiento, al reiniciar se reconstruye completo desde aquí
// (PR-40 ronda 2, H1).
type ResumenEntrenamiento struct {
	ID             string               `json:"id" bson:"id"`
	Epoca          int                  `json:"epoca" bson:"epoca"`
	Perdida        float64              `json:"perdida" bson:"perdida"`
	Historia       []float64            `json:"historia" bson:"historia"`
	Nodos          []cluster.EstadoNodo `json:"nodos" bson:"nodos"`
	Reasignaciones int                  `json:"reasignaciones" bson:"reasignaciones"`
	Fin            time.Time            `json:"fin" bson:"fin"`
}

// Resumen devuelve el resumen del entrenamiento id, si publicó este modelo.
func (m Modelo) Resumen(id string) (ResumenEntrenamiento, bool) {
	for _, r := range m.Entrenamientos {
		if r.ID == id {
			return r, true
		}
	}
	return ResumenEntrenamiento{}, false
}

// Copia devuelve un Modelo que no comparte pesos, esquema ni resúmenes con m
// (PR-40 H7).
func (m Modelo) Copia() Modelo {
	m.Pesos = append([]float64(nil), m.Pesos...)
	m.Esquema = m.Esquema.Copia()
	rs := make([]ResumenEntrenamiento, len(m.Entrenamientos))
	for i, r := range m.Entrenamientos {
		r.Historia = append(make([]float64, 0, len(r.Historia)), r.Historia...)
		r.Nodos = append(make([]cluster.EstadoNodo, 0, len(r.Nodos)), r.Nodos...)
		rs[i] = r
	}
	m.Entrenamientos = rs
	return m
}

// Estados de un entrenamiento.
const (
	EnCurso   = "en_curso"
	Terminado = "terminado"
	Fallido   = "fallido"
)

// Entrenamiento (colección entrenamientos). Es también la vista pública de
// GET /api/v1/entrenamientos/{id}.
type Entrenamiento struct {
	ID             string               `json:"id" bson:"_id"`
	Estado         string               `json:"estado" bson:"estado"`
	Epocas         int                  `json:"epocas" bson:"epocas"`
	Tasa           float64              `json:"tasa" bson:"tasa"`
	Lambda         float64              `json:"lambda" bson:"lambda"`
	Epoca          int                  `json:"epoca" bson:"epoca"` // épocas completadas
	Perdida        float64              `json:"perdida" bson:"perdida"`
	Historia       []float64            `json:"historia" bson:"historia"` // pérdida por época
	Nodos          []cluster.EstadoNodo `json:"nodos" bson:"nodos"`
	Reasignaciones int                  `json:"reasignaciones" bson:"reasignaciones"`
	Inicio         time.Time            `json:"inicio" bson:"inicio"`
	Fin            *time.Time           `json:"fin,omitempty" bson:"fin,omitempty"`
	VersionModelo  string               `json:"version_modelo,omitempty" bson:"version_modelo,omitempty"`
	Error          string               `json:"error,omitempty" bson:"error,omitempty"`
}

// Copia devuelve un Entrenamiento que no comparte slices con e.
func (e Entrenamiento) Copia() Entrenamiento {
	// Siempre listas no nulas: el contrato JSON declara arreglos (PR-40 H9).
	e.Historia = append(make([]float64, 0, len(e.Historia)), e.Historia...)
	e.Nodos = append(make([]cluster.EstadoNodo, 0, len(e.Nodos)), e.Nodos...)
	if e.Fin != nil {
		f := *e.Fin
		e.Fin = &f
	}
	return e
}

// ConsultaNormalizada es una consulta de predicción ya validada: departamento
// en mayúsculas, edad en años y sexo F o M.
type ConsultaNormalizada struct {
	Departamento string  `json:"departamento" bson:"departamento"`
	Ano          int     `json:"ano" bson:"ano"`
	Semana       int     `json:"semana" bson:"semana"`
	EdadAnios    float64 `json:"edad_anios" bson:"edad_anios"`
	Sexo         string  `json:"sexo" bson:"sexo"`
}

// RegistroPrediccion (colección predicciones).
type RegistroPrediccion struct {
	Fecha         time.Time           `json:"fecha" bson:"fecha"`
	VersionModelo string              `json:"version_modelo" bson:"version_modelo"`
	Consulta      ConsultaNormalizada `json:"consulta" bson:"consulta"`
	Riesgo        float64             `json:"riesgo" bson:"riesgo"`
	Clase         int                 `json:"clase" bson:"clase"`
	DesdeCache    bool                `json:"desde_cache" bson:"desde_cache"`
	Microsegundos int64               `json:"microsegundos" bson:"microsegundos"`
}

// Almacen es la persistencia de la API.
type Almacen interface {
	// GuardarModelo inserta la versión si no existe; si existe, le agrega los
	// resúmenes de m.Entrenamientos y adelanta Publicado (nunca lo atrasa). Es
	// una sola escritura: el modelo y el resumen quedan juntos o no quedan.
	GuardarModelo(ctx context.Context, m Modelo) error
	// ModeloVigente devuelve el de Publicado más reciente; false si no hay.
	ModeloVigente(ctx context.Context) (Modelo, bool, error)
	// ModeloDeEntrenamiento devuelve el modelo que publicó ese entrenamiento.
	ModeloDeEntrenamiento(ctx context.Context, id string) (Modelo, bool, error)
	// CrearEntrenamiento solo inserta: si el ID ya existe devuelve
	// ErrDuplicado y deja intacto el existente (PR-40 H3).
	CrearEntrenamiento(ctx context.Context, e Entrenamiento) error
	// GuardarEntrenamiento reemplaza por ID (progreso y estado final).
	GuardarEntrenamiento(ctx context.Context, e Entrenamiento) error
	Entrenamiento(ctx context.Context, id string) (Entrenamiento, bool, error)
	// EntrenamientosEnCurso lista los que figuran en curso: al arrancar, son
	// los que una ejecución anterior dejó sin cerrar (PR-40 H1).
	EntrenamientosEnCurso(ctx context.Context) ([]Entrenamiento, error)
	RegistrarPrediccion(ctx context.Context, r RegistroPrediccion) error
	Cerrar(ctx context.Context) error
}

// ErrDuplicado: CrearEntrenamiento recibió un ID que ya existe.
var ErrDuplicado = errors.New("almacen: ya existe un entrenamiento con ese id")

// Cache guarda respuestas de predicción ya codificadas.
type Cache interface {
	// Obtener devuelve false si la clave no existe o expiró.
	Obtener(ctx context.Context, clave string) ([]byte, bool, error)
	Guardar(ctx context.Context, clave string, valor []byte, ttl time.Duration) error
}

// TTLPrediccion es la vida de una predicción en la caché.
const TTLPrediccion = 24 * time.Hour

// ClavePrediccion arma la clave de Redis. Incluye la versión del modelo: un
// modelo nuevo usa claves nuevas, así que publicar invalida la caché.
func ClavePrediccion(version string, c ConsultaNormalizada) string {
	return fmt.Sprintf("pred:%s:%s:%d:%d:%s:%s", version, c.Departamento, c.Ano, c.Semana,
		strconv.FormatFloat(c.EdadAnios, 'g', -1, 64), c.Sexo)
}
