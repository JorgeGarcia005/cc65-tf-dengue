package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	// RitmoMs es la pausa entre épocas, solo en modo demo (#65): deja ver el
	// entrenamiento en vivo. No cambia el resultado numérico.
	RitmoMs int `json:"ritmo_ms,omitempty"`
	D       int `json:"-"`
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
	nuevoID func() string // nuevoIDEntrenamiento; los tests fuerzan colisiones

	mu      sync.Mutex
	porID   map[string]*Entrenamiento
	enCurso string
}

// nuevoIDEntrenamiento combina la fecha (legible) con 64 bits aleatorios: no
// depende del reloj ni de un contador que se reinicia con el proceso (PR-40 H3).
func nuevoIDEntrenamiento() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("ent-%s-%s", time.Now().UTC().Format("20060102T150405"), hex.EncodeToString(b[:]))
}

// IntentosCrear: IDs distintos que se prueban si CrearEntrenamiento choca.
const IntentosCrear = 3

// errEnCurso: el clúster ya está entrenando (→ 409).
var errEnCurso = errors.New("ya hay un entrenamiento en curso")

// reservando ocupa enCurso mientras el entrenamiento se crea en el almacén.
const reservando = "(creándose)"

// iniciar reserva el clúster y crea el entrenamiento en el almacén con un ID
// nuevo. Si el almacén no lo acepta, libera la reserva y devuelve el error:
// nunca se corre un entrenamiento que no quedó registrado.
func (e *entrenamientos) iniciar(ctx context.Context, p ParametrosEntrenamiento) (Entrenamiento, error) {
	e.mu.Lock()
	if e.enCurso != "" {
		var t Entrenamiento
		if actual, ok := e.porID[e.enCurso]; ok {
			t = actual.Copia()
		}
		e.mu.Unlock()
		return t, errEnCurso
	}
	e.enCurso = reservando
	e.mu.Unlock()

	t := Entrenamiento{Estado: EnCurso, Epocas: p.Epocas, Tasa: p.Tasa, Lambda: p.Lambda,
		Historia: []float64{}, Nodos: []cluster.EstadoNodo{}, Inicio: time.Now().UTC()}
	var err error
	for range IntentosCrear {
		t.ID = e.nuevoID()
		if err = e.almacen.CrearEntrenamiento(ctx, t); !errors.Is(err, almacen.ErrDuplicado) {
			break
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if err != nil {
		e.enCurso = ""
		return Entrenamiento{}, fmt.Errorf("no se pudo registrar el entrenamiento: %w", err)
	}
	if e.porID == nil {
		e.porID = map[string]*Entrenamiento{}
	}
	guardado := t.Copia()
	e.porID[t.ID] = &guardado
	e.enCurso = t.ID
	return t.Copia(), nil
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
		e.persistir(*copia, 1)
	}
}

// terminar fija el estado final y lo persiste con reintentos. Si aun así no se
// guarda, el estado en memoria sigue siendo correcto y, tras un reinicio,
// reconciliar lo deduce del modelo guardado (PR-40 H1).
func (e *entrenamientos) terminar(id, version string, est *cluster.Estado, fin time.Time, err error) Entrenamiento {
	e.mu.Lock()
	t := e.porID[id]
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
	e.persistir(copia, 3)
	return copia
}

// resumen arma el estado final del entrenamiento id con la foto del clúster.
func (e *entrenamientos) resumen(id string, est *cluster.Estado, fin time.Time) almacen.ResumenEntrenamiento {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.porID[id]
	r := almacen.ResumenEntrenamiento{ID: id, Epoca: t.Epoca, Perdida: t.Perdida,
		Historia: append([]float64{}, t.Historia...), Nodos: []cluster.EstadoNodo{}, Fin: fin}
	if est != nil {
		r.Nodos = append(r.Nodos, est.Nodos...)
		r.Reasignaciones = est.Reasignaciones
	}
	return r
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

// persistir intenta hasta `intentos` veces, con espera creciente.
func (e *entrenamientos) persistir(t Entrenamiento, intentos int) {
	var err error
	for i := range intentos {
		if i > 0 {
			time.Sleep(time.Duration(100*i*i) * time.Millisecond)
		}
		ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		err = e.almacen.GuardarEntrenamiento(ctx, t)
		cancelar()
		if err == nil {
			return
		}
	}
	e.log("api: no se pudo guardar el entrenamiento %s (%s): %v", t.ID, t.Estado, err)
}

// reconciliar cierra, al arrancar, los entrenamientos que una ejecución
// anterior dejó en curso: si su modelo está guardado terminaron (solo faltó
// escribir el estado final); si no, se interrumpieron. Hay un solo
// coordinador, así que ninguno puede seguir corriendo (PR-40 H1).
func (e *entrenamientos) reconciliar(ctx context.Context) error {
	en, err := e.almacen.EntrenamientosEnCurso(ctx)
	if err != nil {
		return err
	}
	for _, t := range en {
		m, ok, err := e.almacen.ModeloDeEntrenamiento(ctx, t.ID)
		if err != nil {
			return err
		}
		if r, publicado := m.Resumen(t.ID); ok && publicado {
			// El modelo se guardó con el resumen final: se reconstruye completo.
			fin := r.Fin
			t.Estado, t.VersionModelo, t.Fin = Terminado, m.Version, &fin
			t.Epoca, t.Perdida, t.Reasignaciones = r.Epoca, r.Perdida, r.Reasignaciones
			t.Historia = append([]float64{}, r.Historia...)
			t.Nodos = append([]cluster.EstadoNodo{}, r.Nodos...)
		} else {
			fin := time.Now().UTC()
			t.Estado, t.Error, t.Fin = Fallido, "interrumpido: el coordinador se detuvo antes de terminar", &fin
		}
		if err := e.almacen.GuardarEntrenamiento(ctx, t); err != nil {
			return err
		}
		e.log("api: entrenamiento %s reconciliado como %s", t.ID, t.Estado)
	}
	return nil
}
