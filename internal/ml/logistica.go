// Package ml implementa una regresión logística binaria con descenso de
// gradiente full-batch, pensada para calcularse por bloques de tamaño fijo.
//
// Pérdida (con pesos por muestra sᵢ y regularización L2 sobre los pesos, no
// sobre el sesgo):
//
//	L(w) = (1/S) Σᵢ sᵢ · ℓ(zᵢ, yᵢ) + (λ/2)·‖w‖²,  zᵢ = xᵢᵀw + b,  S = Σᵢ sᵢ
//	ℓ(z, y) = log(1 + eᶻ) − y·z
//
// Gradiente:
//
//	∂L/∂wⱼ = (1/S) Σᵢ sᵢ (σ(zᵢ) − yᵢ) xᵢⱼ + λ·wⱼ
//	∂L/∂b  = (1/S) Σᵢ sᵢ (σ(zᵢ) − yᵢ)
//
// Todas las sumas son sobre filas, así que se pueden partir en bloques. Para
// que cualquier forma de repartir los bloques (un hilo, N workers o N nodos)
// dé un resultado idéntico bit a bit, las sumas parciales de cada bloque se
// acumulan siempre en orden de índice de bloque (ver ReducirEnOrden).
package ml

import (
	"errors"
	"fmt"
	"math"
)

// TamBloquePorDefecto es el número de filas por bloque si Config no indica otro.
const TamBloquePorDefecto = 4096

// Dataset es una matriz de features densa en orden por filas y sus etiquetas.
type Dataset struct {
	X []float64 // N·D valores, fila i en X[i·D : (i+1)·D]
	Y []float64 // N etiquetas en {0, 1}
	N int       // filas
	D int       // features (sin contar el sesgo)
}

// NuevoDataset valida las dimensiones y las etiquetas.
func NuevoDataset(x, y []float64, d int) (*Dataset, error) {
	if d <= 0 || len(y) == 0 {
		return nil, errors.New("ml: dataset vacío o sin features")
	}
	if len(x) != len(y)*d {
		return nil, fmt.Errorf("ml: len(x)=%d no es %d filas × %d features", len(x), len(y), d)
	}
	for i, v := range y {
		if v != 0 && v != 1 {
			return nil, fmt.Errorf("ml: etiqueta %v en la fila %d (debe ser 0 o 1)", v, i)
		}
	}
	return &Dataset{X: x, Y: y, N: len(y), D: d}, nil
}

// Fila devuelve la fila i sin copiarla.
func (ds *Dataset) Fila(i int) []float64 { return ds.X[i*ds.D : (i+1)*ds.D] }

// Config define cómo se calcula la pérdida.
type Config struct {
	TamBloque int     // filas por bloque; 0 = TamBloquePorDefecto
	Lambda    float64 // regularización L2 (no se aplica al sesgo); ≥ 0
	PesoPos   float64 // peso de cada muestra con y = 1; 0 = 1; nunca negativo
	PesoNeg   float64 // peso de cada muestra con y = 0; 0 = 1; nunca negativo
}

// normalizada completa los valores por defecto. Un peso negativo, un λ
// negativo o un valor no finito es un error de programación: entra en pánico
// con un mensaje claro en vez de corregirlo en silencio, porque reemplazarlo
// por 1 escondería el error y entrenaría otro modelo sin avisar.
func (c Config) normalizada() Config {
	for _, v := range []struct {
		nombre string
		valor  float64
	}{{"PesoPos", c.PesoPos}, {"PesoNeg", c.PesoNeg}, {"Lambda", c.Lambda}} {
		if v.valor < 0 || math.IsNaN(v.valor) || math.IsInf(v.valor, 0) {
			panic(fmt.Sprintf("ml: Config.%s = %v (debe ser finito y ≥ 0)", v.nombre, v.valor))
		}
	}
	if c.TamBloque <= 0 {
		c.TamBloque = TamBloquePorDefecto
	}
	if c.PesoPos == 0 {
		c.PesoPos = 1
	}
	if c.PesoNeg == 0 {
		c.PesoNeg = 1
	}
	return c
}

// NumBloques es la cantidad de bloques de tam filas que cubren n filas.
func NumBloques(n, tam int) int { return (n + tam - 1) / tam }

// LimitesBloque devuelve el rango [ini, fin) de filas del bloque k.
func LimitesBloque(k, n, tam int) (ini, fin int) {
	ini = k * tam
	fin = min(ini+tam, n)
	return ini, fin
}

// Parcial son las sumas sin normalizar de un bloque (o de varios ya reducidos).
type Parcial struct {
	Grad    []float64 // D+1 sumas: Σ sᵢ(σ(zᵢ)−yᵢ)xᵢⱼ y, al final, la del sesgo
	Perdida float64   // Σ sᵢ·ℓ(zᵢ, yᵢ)
	Peso    float64   // Σ sᵢ
}

// NuevoParcial reserva un Parcial para d features.
func NuevoParcial(d int) Parcial { return Parcial{Grad: make([]float64, d+1)} }

// Acumular suma otro parcial a p (p += q).
func (p *Parcial) Acumular(q *Parcial) {
	for j, v := range q.Grad {
		p.Grad[j] += v
	}
	p.Perdida += q.Perdida
	p.Peso += q.Peso
}

// Gradiente es el resultado normalizado y regularizado de una pasada completa.
type Gradiente struct {
	Grad    []float64 // ∂L/∂w (D valores) y ∂L/∂b al final
	Perdida float64   // L(w)
}

// Calculador calcula el gradiente completo para unos pesos dados.
// Lo implementan la versión secuencial, el worker pool y el coordinador del clúster.
type Calculador interface {
	Gradiente(w []float64) Gradiente
}

// Sigmoide es σ(z) = 1/(1+e⁻ᶻ), sin desbordes para |z| grande.
func Sigmoide(z float64) float64 {
	if z >= 0 {
		return 1 / (1 + math.Exp(-z))
	}
	e := math.Exp(z)
	return e / (1 + e)
}

// perdidaLogistica es ℓ(z, y) = log(1+eᶻ) − y·z, estable para |z| grande.
func perdidaLogistica(z, y float64) float64 {
	return math.Max(z, 0) + math.Log1p(math.Exp(-math.Abs(z))) - y*z
}

// Puntaje es zᵢ = xᵢᵀw + b para una fila.
func Puntaje(w, x []float64) float64 {
	d := len(x)
	z := w[d]
	for j, v := range x {
		z += w[j] * v
	}
	return z
}

// ParcialBloque calcula las sumas del bloque k en out (que se sobrescribe).
// Es la única función que toca las filas: secuencial, concurrente y
// distribuido la comparten, por eso sus resultados por bloque son idénticos.
//
// La pérdida y el peso se acumulan en variables locales (registros) y se
// escriben en out una sola vez al final: así dos goroutines que procesan
// bloques vecinos no escriben fila a fila en la misma línea de caché.
func ParcialBloque(ds *Dataset, cfg Config, w []float64, k int, out *Parcial) {
	cfg = cfg.normalizada()
	clear(out.Grad)
	ini, fin := LimitesBloque(k, ds.N, cfg.TamBloque)
	d := ds.D
	grad := out.Grad
	var perdida, peso float64
	for i := ini; i < fin; i++ {
		x := ds.Fila(i)
		y := ds.Y[i]
		s := cfg.PesoNeg
		if y == 1 {
			s = cfg.PesoPos
		}
		z := Puntaje(w, x)
		r := s * (Sigmoide(z) - y)
		for j, v := range x {
			grad[j] += r * v
		}
		grad[d] += r
		perdida += s * perdidaLogistica(z, y)
		peso += s
	}
	out.Perdida, out.Peso = perdida, peso
}

// copiarEn copia p en dst (mismas dimensiones) sin reservar memoria.
func (p *Parcial) copiarEn(dst *Parcial) {
	copy(dst.Grad, p.Grad)
	dst.Perdida, dst.Peso = p.Perdida, p.Peso
}

// ReducirEnOrden acumula los parciales en el orden del slice (índice de bloque).
// Este orden fijo es lo que garantiza el determinismo bit a bit.
func ReducirEnOrden(parciales []Parcial, d int) Parcial {
	total := NuevoParcial(d)
	for k := range parciales {
		total.Acumular(&parciales[k])
	}
	return total
}

// Finalizar normaliza por el peso total y agrega la regularización L2.
func Finalizar(total Parcial, cfg Config, w []float64) Gradiente {
	d := len(w) - 1
	g := Gradiente{Grad: make([]float64, d+1)}
	var norma2 float64
	for j := 0; j < d; j++ {
		g.Grad[j] = total.Grad[j]/total.Peso + cfg.Lambda*w[j]
		norma2 += w[j] * w[j]
	}
	g.Grad[d] = total.Grad[d] / total.Peso
	g.Perdida = total.Perdida/total.Peso + 0.5*cfg.Lambda*norma2
	return g
}

// Secuencial recorre los bloques uno tras otro en un solo hilo.
type Secuencial struct {
	ds      *Dataset
	cfg     Config
	parcial Parcial
}

// NuevoSecuencial crea el calculador de referencia.
func NuevoSecuencial(ds *Dataset, cfg Config) *Secuencial {
	return &Secuencial{ds: ds, cfg: cfg.normalizada(), parcial: NuevoParcial(ds.D)}
}

// Gradiente implementa Calculador.
func (s *Secuencial) Gradiente(w []float64) Gradiente {
	total := NuevoParcial(s.ds.D)
	for k := 0; k < NumBloques(s.ds.N, s.cfg.TamBloque); k++ {
		ParcialBloque(s.ds, s.cfg, w, k, &s.parcial)
		total.Acumular(&s.parcial)
	}
	return Finalizar(total, s.cfg, w)
}

// PesosBalanceados devuelve pesos por clase que igualan el aporte total de
// positivos y negativos: n/(2·n_pos) y n/(2·n_neg).
func PesosBalanceados(y []float64) (pos, neg float64) {
	var npos float64
	for _, v := range y {
		npos += v
	}
	n := float64(len(y))
	nneg := n - npos
	if npos == 0 || nneg == 0 {
		return 1, 1
	}
	return n / (2 * npos), n / (2 * nneg)
}

// Entrenamiento son los hiperparámetros del descenso de gradiente.
type Entrenamiento struct {
	Epocas int
	Tasa   float64
}

// Resultado son los pesos finales y la pérdida al inicio de cada época.
type Resultado struct {
	W        []float64
	Historia []float64
}

// Entrenar ejecuta descenso de gradiente full-batch desde w = 0.
// No sabe si el gradiente se calcula en uno, varios hilos o varios nodos.
func Entrenar(calc Calculador, d int, e Entrenamiento) Resultado {
	w := make([]float64, d+1)
	historia := make([]float64, 0, e.Epocas)
	for range e.Epocas {
		g := calc.Gradiente(w)
		historia = append(historia, g.Perdida)
		for j := range w {
			w[j] -= e.Tasa * g.Grad[j]
		}
	}
	return Resultado{W: w, Historia: historia}
}

// Probabilidad es σ(xᵀw + b) para una fila.
func Probabilidad(w, x []float64) float64 { return Sigmoide(Puntaje(w, x)) }
