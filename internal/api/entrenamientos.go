package api

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Valores por defecto de un entrenamiento: los mismos de cmd/entrenar y
// cmd/coordinator, para que el modelo de la API sea comparable con el reporte.
const (
	DefectoEpocas = 600
	DefectoTasa   = 2.0
	DefectoLambda = 1e-4
	MaxEpocas     = 100000
)

// ParametrosEntrenamiento es el cuerpo de POST /api/v1/entrenamientos. D lo
// completa la API con el número de variables del modelo publicado.
type ParametrosEntrenamiento struct {
	Epocas int     `json:"epocas"`
	Tasa   float64 `json:"tasa"`
	Lambda float64 `json:"lambda"`
	D      int     `json:"-"`
}

// Entrenador corre un entrenamiento completo (en el clúster) y devuelve los
// pesos con el sesgo al final. Debe llamar a progreso tras cada época y
// abandonar cuando ctx se cancele. cmd/coordinator lo implementa sobre
// internal/cluster; los tests usan uno falso.
type Entrenador interface {
	Entrenar(ctx context.Context, p ParametrosEntrenamiento, progreso func(epoca int, perdida float64)) ([]float64, error)
}

// Estados de un entrenamiento.
const (
	EnCurso   = "en_curso"
	Terminado = "terminado"
	Fallido   = "fallido"
)

// Entrenamiento es la vista pública (GET /api/v1/entrenamientos/{id}).
type Entrenamiento struct {
	ID            string  `json:"id"`
	Estado        string  `json:"estado"`
	Epocas        int     `json:"epocas"`
	Tasa          float64 `json:"tasa"`
	Lambda        float64 `json:"lambda"`
	Epoca         int     `json:"epoca"` // épocas completadas
	Perdida       float64 `json:"perdida"`
	Inicio        string  `json:"inicio"`
	Fin           string  `json:"fin,omitempty"`
	VersionModelo string  `json:"version_modelo,omitempty"`
	Error         string  `json:"error,omitempty"`
}

// entrenamientos guarda el historial en memoria (Mongo lo persistirá en #29)
// y garantiza que haya a lo sumo uno en curso: el clúster es uno solo.
type entrenamientos struct {
	mu      sync.Mutex
	porID   map[string]*Entrenamiento
	enCurso string
	sig     int
}

func (e *entrenamientos) iniciar(p ParametrosEntrenamiento) (Entrenamiento, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.enCurso != "" {
		return *e.porID[e.enCurso], false
	}
	if e.porID == nil {
		e.porID = map[string]*Entrenamiento{}
	}
	e.sig++
	t := &Entrenamiento{
		ID: fmt.Sprintf("ent-%d", e.sig), Estado: EnCurso,
		Epocas: p.Epocas, Tasa: p.Tasa, Lambda: p.Lambda,
		Inicio: time.Now().Format(time.RFC3339),
	}
	e.porID[t.ID] = t
	e.enCurso = t.ID
	return *t, true
}

func (e *entrenamientos) progreso(id string, epoca int, perdida float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.porID[id]
	t.Epoca, t.Perdida = epoca, perdida
}

func (e *entrenamientos) terminar(id, version string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.porID[id]
	t.Fin = time.Now().Format(time.RFC3339)
	if err != nil {
		t.Estado, t.Error = Fallido, err.Error()
	} else {
		t.Estado, t.VersionModelo = Terminado, version
	}
	e.enCurso = ""
}

func (e *entrenamientos) consultar(id string) (Entrenamiento, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, ok := e.porID[id]
	if !ok {
		return Entrenamiento{}, false
	}
	return *t, true
}
