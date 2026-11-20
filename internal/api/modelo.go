package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// AnoMaximo acota el año de una consulta: el modelo extrapola la tendencia
// anual (ano_z) y más allá de este año la extrapolación no tiene sentido.
const AnoMaximo = 2100

// Modelo es lo que escribe cmd/entrenar en data/gold/modelo.json: pesos (con
// el sesgo al final), el esquema de variables, el umbral de clasificación y el
// ajuste ln(PesoPos/PesoNeg) que calibra la probabilidad. Es inmutable: un
// entrenamiento nuevo publica otro *Modelo.
type Modelo struct {
	Pesos       []float64         `json:"pesos"`
	Esquema     *features.Esquema `json:"esquema"`
	Umbral      float64           `json:"umbral"`
	AjustePrior float64           `json:"ajuste_prior"`
	Generado    string            `json:"generado"`
	// Version identifica los pesos: los primeros 12 hex del sha256 del JSON.
	// La caché de predicciones la usa en la clave (#29).
	Version string `json:"-"`
}

// CargarModelo lee un modelo.json y valida que pesos y esquema sean coherentes.
func CargarModelo(b []byte) (*Modelo, error) {
	var m Modelo
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("modelo: %w", err)
	}
	if m.Esquema == nil {
		return nil, errors.New("modelo: falta el esquema")
	}
	if err := m.Esquema.Preparar(); err != nil {
		return nil, fmt.Errorf("modelo: %w", err)
	}
	if err := validarNumeros(&m); err != nil {
		return nil, err
	}
	if len(m.Pesos) != m.Esquema.D()+1 {
		return nil, fmt.Errorf("modelo: %d pesos para %d variables (se esperaban %d con el sesgo)",
			len(m.Pesos), m.Esquema.D(), m.Esquema.D()+1)
	}
	if m.Umbral <= 0 || m.Umbral >= 1 {
		return nil, fmt.Errorf("modelo: umbral %v fuera de (0, 1)", m.Umbral)
	}
	m.Version = version(b)
	return &m, nil
}

// conPesos devuelve un modelo nuevo con el mismo esquema y otros pesos.
func (m *Modelo) conPesos(w []float64, generado string) (*Modelo, error) {
	n := &Modelo{Pesos: w, Esquema: m.Esquema, Umbral: m.Umbral, AjustePrior: m.AjustePrior, Generado: generado}
	if len(w) != m.Esquema.D()+1 {
		return nil, fmt.Errorf("el entrenamiento devolvió %d pesos, se esperaban %d", len(w), m.Esquema.D()+1)
	}
	b, err := json.Marshal(n)
	if err != nil {
		return nil, err
	}
	n.Version = version(b)
	return n, nil
}

// validarNumeros exige que las medias sean finitas, las desviaciones positivas
// (Llenar divide por ellas) y los pesos y el ajuste finitos: un artefacto
// incompleto se rechaza al cargar, no con una predicción NaN.
func validarNumeros(m *Modelo) error {
	e := m.Esquema
	for _, c := range []struct {
		nombre   string
		v        float64
		positivo bool
	}{
		{"media_edad", e.MediaEdad, false}, {"desv_edad", e.DesvEdad, true},
		{"media_ano", e.MediaAno, false}, {"desv_ano", e.DesvAno, true},
		{"ajuste_prior", m.AjustePrior, false},
	} {
		if math.IsNaN(c.v) || math.IsInf(c.v, 0) || (c.positivo && c.v <= 0) {
			return fmt.Errorf("modelo: %s = %v inválido (debe ser finito%s)", c.nombre, c.v, map[bool]string{true: " y positivo"}[c.positivo])
		}
	}
	for j, w := range m.Pesos {
		if math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("modelo: peso %d = %v no es finito", j, w)
		}
	}
	return nil
}

func version(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:6])
}

// Consulta es el cuerpo de POST /api/v1/predicciones. Usa los mismos campos y
// reglas que el CSV del MINSA (ingest.LimpiarCampos).
type Consulta struct {
	Departamento string   `json:"departamento"`
	Ano          int      `json:"ano"`
	Semana       int      `json:"semana"`
	Edad         *float64 `json:"edad"`      // puntero: distingue 0 (recién nacido) de ausente o null
	TipoEdad     string   `json:"tipo_edad"` // A (años, por defecto), M (meses) o D (días)
	Sexo         string   `json:"sexo"`
}

// Prediccion es la respuesta de POST /api/v1/predicciones.
type Prediccion struct {
	// Riesgo es la probabilidad calibrada de dengue con signos de alarma o
	// grave: la escala de la tasa observada.
	Riesgo float64 `json:"riesgo"`
	// ProbabilidadPonderada es σ(w·x+b), la salida del modelo entrenado con
	// pesos por clase; es la que se compara con el umbral.
	ProbabilidadPonderada float64 `json:"probabilidad_ponderada"`
	Clase                 int     `json:"clase"`
	Etiqueta              string  `json:"etiqueta"`
	Umbral                float64 `json:"umbral"`
	DepartamentoModelo    string  `json:"departamento_modelo"`
	VersionModelo         string  `json:"version_modelo"`
}

// ErrConsulta marca los errores de validación (→ 400).
var ErrConsulta = errors.New("consulta inválida")

func errConsulta(formato string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrConsulta, fmt.Sprintf(formato, args...))
}

// registro valida la consulta con las reglas de limpieza del loader y la
// convierte en el registro que entiende features.Esquema.Llenar.
func (m *Modelo) registro(c Consulta) (ingest.Registro, error) {
	var r ingest.Registro
	r.Departamento = strings.ToUpper(strings.TrimSpace(c.Departamento))
	if r.Departamento == "" {
		return r, errConsulta("falta el departamento")
	}
	if c.Ano < m.Esquema.AnoMin || c.Ano > AnoMaximo {
		return r, errConsulta("año %d fuera de %d–%d (el modelo se entrenó desde %d)", c.Ano, m.Esquema.AnoMin, AnoMaximo, m.Esquema.AnoMin)
	}
	r.Ano = c.Ano
	if c.Semana < 1 || c.Semana > 53 {
		return r, errConsulta("semana epidemiológica %d fuera de 1–53", c.Semana)
	}
	r.Semana = c.Semana
	tipo := strings.ToUpper(strings.TrimSpace(c.TipoEdad))
	if tipo == "" {
		tipo = "A"
	}
	if c.Edad == nil {
		return r, errConsulta("falta la edad")
	}
	if *c.Edad != math.Trunc(*c.Edad) || math.IsInf(*c.Edad, 0) {
		return r, errConsulta("la edad debe ser un número entero")
	}
	edad, ok := ingest.EdadEnAnios([]byte(fmt.Sprint(int64(*c.Edad))), []byte(tipo))
	if !ok {
		return r, errConsulta("edad %v con tipo_edad %q: la edad debe ser ≥ 0 y tipo_edad A, M o D", *c.Edad, c.TipoEdad)
	}
	if edad > ingest.EdadMax {
		return r, errConsulta("edad mayor a %d años", ingest.EdadMax)
	}
	r.EdadAnios = edad
	switch s := strings.ToUpper(strings.TrimSpace(c.Sexo)); s {
	case "F", "M":
		r.Sexo = s[0]
	default:
		return r, errConsulta("sexo %q: debe ser F o M", c.Sexo)
	}
	return r, nil
}

// Predecir valida la consulta y aplica el modelo.
func (m *Modelo) Predecir(c Consulta) (Prediccion, error) {
	r, err := m.registro(c)
	if err != nil {
		return Prediccion{}, err
	}
	d := m.Esquema.D()
	x := make([]float64, d)
	m.Esquema.Llenar(r, x)
	z := m.Pesos[d]
	for j, v := range x {
		z += m.Pesos[j] * v
	}
	p := ml.Sigmoide(z)
	pr := Prediccion{
		Riesgo:                ml.Calibrar(p, m.AjustePrior),
		ProbabilidadPonderada: p,
		Umbral:                m.Umbral,
		Etiqueta:              "sin signos de alarma",
		DepartamentoModelo:    features.OtrosDepartamentos,
		VersionModelo:         m.Version,
	}
	if p >= m.Umbral {
		pr.Clase, pr.Etiqueta = 1, "con signos de alarma o grave"
	}
	for _, dep := range m.Esquema.Departamentos {
		if dep == r.Departamento {
			pr.DepartamentoModelo = dep
		}
	}
	return pr, nil
}
