package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
)

// Valores por defecto de un entrenamiento: los mismos de cmd/entrenar y
// cmd/coordinator, para que el modelo de la API sea comparable con el reporte.
const (
	DefectoEpocas = 600
	DefectoTasa   = 2.0
	DefectoLambda = 1e-4
	MaxEpocas     = 100000
	// PersistirCada: cada cuántas épocas se escribe el progreso en el almacén.
	PersistirCada = 10
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

// Entrenamiento es la vista pública y el documento de Mongo (ADR-0006).
type Entrenamiento = almacen.Entrenamiento

// Estados de un entrenamiento.
const (
	EnCurso   = almacen.EnCurso
	Terminado = almacen.Terminado
	Fallido   = almacen.Fallido
)

// entrenamientos lleva el estado en vivo en memoria (a lo sumo uno en curso:
// el clúster es uno solo) y lo escribe en el almacén al iniciar, cada
// PersistirCada épocas y al terminar. Las escrituras se hacen fuera del mutex
// con una copia, para no bloquear las consultas mientras responde Mongo.
type entrenamientos struct {
	almacen almacen.Almacen
	log     func(string, ...any)

	mu      sync.Mutex
	porID   map[string]*Entrenamiento
	enCurso string
	sig     int
}

func (e *entrenamientos) iniciar(p ParametrosEntrenamiento) (Entrenamiento, bool) {
	e.mu.Lock()
	if e.enCurso != "" {
		t := e.porID[e.enCurso].Copia()
		e.mu.Unlock()
		return t, false
	}
	if e.porID == nil {
		e.porID = map[string]*Entrenamiento{}
	}
	e.sig++
	ahora := time.Now().UTC()
	t := &Entrenamiento{
		// La fecha evita repetir IDs de ejecuciones anteriores ya guardadas.
		ID:     fmt.Sprintf("ent-%s-%d", ahora.Format("20060102T150405"), e.sig),
		Estado: EnCurso, Epocas: p.Epocas, Tasa: p.Tasa, Lambda: p.Lambda,
		Historia: []float64{}, Nodos: []cluster.EstadoNodo{}, Inicio: ahora,
	}
	e.porID[t.ID] = t
	e.enCurso = t.ID
	copia := t.Copia()
	e.mu.Unlock()
	e.persistir(copia)
	return copia, true
}

func (e *entrenamientos) progreso(id string, epoca int, perdida float64) {
	e.mu.Lock()
	t := e.porID[id]
	t.Epoca, t.Perdida = epoca, perdida
	t.Historia = append(t.Historia, perdida)
	var copia *Entrenamiento
	if epoca%PersistirCada == 0 {
		c := t.Copia()
		copia = &c
	}
	e.mu.Unlock()
	if copia != nil {
		e.persistir(*copia)
	}
}

func (e *entrenamientos) terminar(id, version string, est *cluster.Estado, err error) Entrenamiento {
	e.mu.Lock()
	t := e.porID[id]
	fin := time.Now().UTC()
	t.Fin = &fin
	if est != nil {
		t.Nodos = append([]cluster.EstadoNodo{}, est.Nodos...)
		t.Reasignaciones = est.Reasignaciones
	}
	if err != nil {
		t.Estado, t.Error = Fallido, err.Error()
	} else {
		t.Estado, t.VersionModelo = Terminado, version
	}
	e.enCurso = ""
	copia := t.Copia()
	e.mu.Unlock()
	e.persistir(copia)
	return copia
}

// consultar busca en memoria y, si no está (otra ejecución), en el almacén.
func (e *entrenamientos) consultar(ctx context.Context, id string) (Entrenamiento, bool, error) {
	e.mu.Lock()
	t, ok := e.porID[id]
	if ok {
		c := t.Copia()
		e.mu.Unlock()
		return c, true, nil
	}
	e.mu.Unlock()
	return e.almacen.Entrenamiento(ctx, id)
}

func (e *entrenamientos) persistir(t Entrenamiento) {
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := e.almacen.GuardarEntrenamiento(ctx, t); err != nil {
		e.log("api: no se pudo guardar el entrenamiento %s: %v", t.ID, err)
	}
}
