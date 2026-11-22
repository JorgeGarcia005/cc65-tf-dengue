package evaluacion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// VersionEsquema del JSON. El front la verifica antes de dibujar.
const VersionEsquema = 1

// Fuente identifica un archivo de entrada por su contenido.
type Fuente struct {
	Archivo string `json:"archivo"`
	Sha256  string `json:"sha256"`
}

// NuevaFuente calcula el sha256 de b.
func NuevaFuente(archivo string, b []byte) Fuente {
	s := sha256.Sum256(b)
	return Fuente{archivo, hex.EncodeToString(s[:])}
}

// Entrada reúne lo que necesita Generar. Los benchmarks se pasan como JSON
// crudo para copiar sus cifras sin recalcularlas.
type Entrada struct {
	Test          *ml.Dataset
	Pesos         []float64 // con el sesgo al final
	Esquema       *features.Esquema
	Umbral        float64
	AjustePrior   float64
	VersionModelo string
	AUCReferencia float64 // metricas_test.AUC de reports/entrenamiento.json; 0 = no comparar
	BenchCluster  []byte
	BenchAPI      []byte
	Fuentes       []Fuente
	MinCasos      int
	TopN          int
	PuntosROC     int
	Deciles       int
}

// Reporte es reports/evaluacion.json. No lleva fecha: dos corridas con las
// mismas entradas producen el mismo archivo byte a byte.
type Reporte struct {
	VersionEsquema int           `json:"version_esquema"`
	VersionModelo  string        `json:"version_modelo"`
	Fuentes        []Fuente      `json:"fuentes"`
	FilasTest      int           `json:"filas_test"`
	PositivosTest  int           `json:"positivos_test"`
	Metricas       ml.Metricas   `json:"metricas_test"`
	AUC            float64       `json:"auc_trapecio"`
	Brier          float64       `json:"brier_calibrado"`
	ROC            []PuntoROC    `json:"roc"`
	Calibracion    []Bin         `json:"calibracion"`
	Riesgo         Cubo          `json:"riesgo"`
	Prioridades    []Celda       `json:"prioridades"`
	Escalabilidad  Escalabilidad `json:"escalabilidad"`
	LatenciaAPI    LatenciaAPI   `json:"latencia_api"`
}

// Generar calcula la evaluación completa. Falla si el AUC de la ROC se aparta
// del de Mann-Whitney o del de referencia en más de 1e-6.
func Generar(in Entrada) (Reporte, error) {
	if in.Test == nil || in.Test.N == 0 || in.Esquema == nil {
		return Reporte{}, errors.New("evaluacion: faltan el conjunto de prueba o el esquema")
	}
	if in.Test.D != in.Esquema.D() || len(in.Pesos) != in.Esquema.D()+1 {
		return Reporte{}, fmt.Errorf("evaluacion: el test tiene %d variables, el esquema %d y el modelo %d pesos (se esperaban %d)",
			in.Test.D, in.Esquema.D(), len(in.Pesos), in.Esquema.D()+1)
	}
	y := in.Test.Y
	p, pc := make([]float64, in.Test.N), make([]float64, in.Test.N)
	positivos := 0
	for i := range p {
		p[i] = ml.Probabilidad(in.Pesos, in.Test.Fila(i))
		pc[i] = ml.Calibrar(p[i], in.AjustePrior)
		positivos += int(y[i])
	}
	met, err := ml.Evaluar(y, p, in.Umbral)
	if err != nil {
		return Reporte{}, err
	}
	roc, auc, err := ROC(y, p, in.AjustePrior, in.PuntosROC)
	if err != nil {
		return Reporte{}, err
	}
	if math.Abs(auc-met.AUC) > 1e-6 || (in.AUCReferencia != 0 && math.Abs(auc-in.AUCReferencia) > 1e-6) {
		return Reporte{}, fmt.Errorf("evaluacion: AUC por trapecio %.9f ≠ Mann-Whitney %.9f o referencia %.9f",
			auc, met.AUC, in.AUCReferencia)
	}
	cal, err := Calibracion(y, pc, in.Deciles)
	if err != nil {
		return Reporte{}, err
	}
	cubo, err := Cruzar(in.Test, y, pc, in.Esquema, in.MinCasos)
	if err != nil {
		return Reporte{}, err
	}
	esc, err := LeerEscalabilidad(in.BenchCluster)
	if err != nil {
		return Reporte{}, err
	}
	lat, err := LeerLatencia(in.BenchAPI)
	if err != nil {
		return Reporte{}, err
	}
	return Reporte{
		VersionEsquema: VersionEsquema, VersionModelo: in.VersionModelo, Fuentes: in.Fuentes,
		FilasTest: in.Test.N, PositivosTest: positivos, Metricas: met, AUC: auc, Brier: Brier(y, pc),
		ROC: roc, Calibracion: cal, Riesgo: cubo, Prioridades: Prioridades(cubo, in.TopN),
		Escalabilidad: esc, LatenciaAPI: lat,
	}, nil
}

// Escalabilidad copia, sin recalcular, las mediciones de cmd/benchmark del clúster.
type Escalabilidad struct {
	Maquina    json.RawMessage `json:"maquina"`
	Mediciones []PuntoEscala   `json:"mediciones"`
}

// PuntoEscala es una fila de bench_cluster_*.json.
type PuntoEscala struct {
	Nodos    int `json:"nodos"`
	SegEpoca struct {
		Media   float64 `json:"media_recortada"`
		IC95Inf float64 `json:"ic95_inf"`
		IC95Sup float64 `json:"ic95_sup"`
	} `json:"seg_por_epoca"`
	Speedup    float64 `json:"speedup"`
	Eficiencia float64 `json:"eficiencia"`
	Desglose   struct {
		FraccionRed float64 `json:"fraccion_red_y_espera"`
	} `json:"desglose_por_epoca"`
}

// LeerEscalabilidad extrae máquina y mediciones de bench_cluster_*.json.
func LeerEscalabilidad(b []byte) (Escalabilidad, error) {
	var e Escalabilidad
	if err := json.Unmarshal(b, &e); err != nil {
		return e, fmt.Errorf("evaluacion: bench del clúster: %w", err)
	}
	if len(e.Mediciones) == 0 {
		return e, errors.New("evaluacion: bench del clúster sin mediciones")
	}
	return e, nil
}

// LatenciaAPI copia las corridas de cmd/benchapi.
type LatenciaAPI struct {
	Maquina   string  `json:"maquina"`
	LimiteP95 float64 `json:"limite_p95_ms"`
	Corridas  []struct {
		Nombre       string  `json:"nombre"`
		Peticiones   int     `json:"peticiones"`
		Errores      int     `json:"errores"`
		P50          float64 `json:"p50_ms"`
		P95          float64 `json:"p95_ms"`
		P99          float64 `json:"p99_ms"`
		RPS          float64 `json:"rps"`
		CumpleLimite bool    `json:"cumple_limite"`
	} `json:"corridas"`
}

// LeerLatencia extrae las corridas de bench_api_*.json.
func LeerLatencia(b []byte) (LatenciaAPI, error) {
	var l LatenciaAPI
	if err := json.Unmarshal(b, &l); err != nil {
		return l, fmt.Errorf("evaluacion: bench de la API: %w", err)
	}
	if len(l.Corridas) == 0 {
		return l, errors.New("evaluacion: bench de la API sin corridas")
	}
	return l, nil
}
