// Command entrenar entrena la regresión logística sobre el gold real con el
// worker pool (Parte B dentro de un nodo) y evalúa en el conjunto de prueba.
//
// Uso:
//
//	go run ./cmd/entrenar -epocas 600 -tasa 2
package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

type coeficiente struct {
	Variable  string  `json:"variable"`
	Peso      float64 `json:"peso"`
	RazonOdds float64 `json:"razon_de_odds"`
}

type grupo struct {
	Grupo         string  `json:"grupo"`
	Casos         int     `json:"casos_test"`
	TasaObservada float64 `json:"tasa_observada"`
	RiesgoMedio   float64 `json:"riesgo_predicho_medio"`
	RiesgoCalib   float64 `json:"riesgo_calibrado_medio"`
}

type reporte struct {
	Generado     string            `json:"generado"`
	Workers      int               `json:"workers"`
	Epocas       int               `json:"epocas"`
	Tasa         float64           `json:"tasa_aprendizaje"`
	Lambda       float64           `json:"lambda"`
	PesoPos      float64           `json:"peso_positivos"`
	PesoNeg      float64           `json:"peso_negativos"`
	FilasTrain   int               `json:"filas_train"`
	FilasTest    int               `json:"filas_test"`
	D            int               `json:"variables"`
	Equivale     bool              `json:"gradiente_pool_igual_al_secuencial"`
	SegTotal     float64           `json:"seg_entrenamiento"`
	SegPorEpoca  float64           `json:"seg_por_epoca"`
	Historia     []float64         `json:"perdida_por_epoca"`
	Test         ml.Metricas       `json:"metricas_test"`
	Train        ml.Metricas       `json:"metricas_train"`
	Pesos        []float64         `json:"pesos"`
	Sesgo        float64           `json:"sesgo"`
	Coeficientes []coeficiente     `json:"coeficientes_por_magnitud"`
	PorDepto     []grupo           `json:"riesgo_por_departamento"`
	PorEtapa     []grupo           `json:"riesgo_por_etapa_de_vida"`
	AjustePrior  float64           `json:"ajuste_prior_ln_pesopos_sobre_pesoneg"`
	Esquema      *features.Esquema `json:"esquema"`
}

func main() {
	dir := flag.String("gold", "data/gold", "carpeta con train.bin, test.bin y esquema.json")
	epocas := flag.Int("epocas", 600, "épocas de descenso de gradiente full-batch")
	tasa := flag.Float64("tasa", 2.0, "tasa de aprendizaje")
	lambda := flag.Float64("lambda", 1e-4, "regularización L2")
	workers := flag.Int("workers", runtime.NumCPU(), "workers del pool")
	umbral := flag.Float64("umbral", 0.5, "umbral de clasificación")
	salida := flag.String("salida", "reports/entrenamiento.json", "reporte")
	modelo := flag.String("modelo", "data/gold/modelo.json", "pesos entrenados (para la API)")
	flag.Parse()

	train := leerGold(filepath.Join(*dir, "train.bin"))
	test := leerGold(filepath.Join(*dir, "test.bin"))
	var esq features.Esquema
	leerJSON(filepath.Join(*dir, "esquema.json"), &esq)
	if err := esq.Preparar(); err != nil {
		log.Fatal(err)
	}
	if esq.D() != train.D || esq.D() != test.D {
		log.Fatalf("D del esquema (%d) no coincide con el gold (%d, %d)", esq.D(), train.D, test.D)
	}

	pos, neg := ml.PesosBalanceados(train.Y)
	cfg := ml.Config{TamBloque: train.TamBloque, Lambda: *lambda, PesoPos: pos, PesoNeg: neg}
	pool := ml.NuevoPool(train.Dataset, cfg, *workers)
	defer pool.Cerrar()

	w0 := make([]float64, train.D+1)
	gSec := ml.NuevoSecuencial(train.Dataset, cfg).Gradiente(w0)
	gPool := pool.Gradiente(w0)
	equivale := slices.EqualFunc(gSec.Grad, gPool.Grad, func(a, b float64) bool {
		return math.Float64bits(a) == math.Float64bits(b)
	})
	if !equivale {
		log.Fatal("el gradiente del pool difiere del secuencial")
	}

	ini := time.Now()
	res := ml.Entrenar(pool, train.D, ml.Entrenamiento{Epocas: *epocas, Tasa: *tasa})
	seg := time.Since(ini).Seconds()

	pTest := predecir(res.W, test.Dataset)
	pTrain := predecir(res.W, train.Dataset)
	mTest, err := ml.Evaluar(test.Y, pTest, *umbral)
	if err != nil {
		log.Fatal(err)
	}
	mTrain, err := ml.Evaluar(train.Y, pTrain, *umbral)
	if err != nil {
		log.Fatal(err)
	}

	rep := reporte{
		Generado: time.Now().Format(time.RFC3339), Workers: *workers, Epocas: *epocas, Tasa: *tasa,
		Lambda: *lambda, PesoPos: pos, PesoNeg: neg, FilasTrain: train.N, FilasTest: test.N, D: train.D,
		Equivale: equivale, SegTotal: seg, SegPorEpoca: seg / float64(*epocas), Historia: res.Historia,
		Test: mTest, Train: mTrain, Pesos: res.W[:train.D], Sesgo: res.W[train.D], Esquema: &esq,
	}
	for j, nombre := range esq.Nombres {
		rep.Coeficientes = append(rep.Coeficientes, coeficiente{nombre, res.W[j], math.Exp(res.W[j])})
	}
	slices.SortFunc(rep.Coeficientes, func(a, b coeficiente) int { return cmp.Compare(math.Abs(b.Peso), math.Abs(a.Peso)) })
	ajuste := math.Log(pos / neg)
	rep.AjustePrior = ajuste
	rep.PorDepto = agrupar(test.Dataset, test.Y, pTest, esq, "depto_", ajuste)
	rep.PorEtapa = agrupar(test.Dataset, test.Y, pTest, esq, "etapa_", ajuste)

	escribirJSON(*salida, rep)
	escribirJSON(*modelo, map[string]any{"pesos": res.W, "esquema": &esq, "umbral": *umbral,
		"ajuste_prior": ajuste, "generado": rep.Generado})

	fmt.Printf("train: %d filas · test: %d filas · %d variables · pesos de clase +%.3f / −%.3f\n",
		train.N, test.N, train.D, pos, neg)
	fmt.Printf("entrenamiento: %d épocas en %.1f s (%.1f ms/época, %d workers); pérdida %.4f → %.4f\n",
		*epocas, seg, 1000*rep.SegPorEpoca, *workers, res.Historia[0], res.Historia[len(res.Historia)-1])
	fmt.Printf("test: AUC %.4f · recall %.3f · precisión %.3f · F1 %.3f · exactitud %.3f\n",
		mTest.AUC, mTest.Recall, mTest.Precision, mTest.F1, mTest.Exactitud)
	fmt.Println("variables más influyentes:")
	for _, c := range rep.Coeficientes[:8] {
		fmt.Printf("  %-28s peso %+.3f · razón de odds %.2f\n", c.Variable, c.Peso, c.RazonOdds)
	}
}

func predecir(w []float64, ds *ml.Dataset) []float64 {
	p := make([]float64, ds.N)
	for i := range p {
		p[i] = ml.Probabilidad(w, ds.Fila(i))
	}
	return p
}

// agrupar resume riesgo observado y predicho por las variables one-hot con prefijo.
//
// Con pesos por clase, σ(xᵀw) ordena bien los casos pero sobreestima la
// probabilidad. riesgo_calibrado aplica la corrección de prior de King y Zeng
// (2001): logit calibrado = logit − ln(PesoPos/PesoNeg), que devuelve la escala
// de la tasa real de positivos.
func agrupar(ds *ml.Dataset, y, p []float64, esq features.Esquema, prefijo string, ajuste float64) []grupo {
	var out []grupo
	for j, nombre := range esq.Nombres {
		if len(nombre) <= len(prefijo) || nombre[:len(prefijo)] != prefijo {
			continue
		}
		g := grupo{Grupo: nombre[len(prefijo):]}
		var obs, pred, cal float64
		for i := 0; i < ds.N; i++ {
			if ds.Fila(i)[j] == 1 {
				g.Casos++
				obs += y[i]
				pred += p[i]
				cal += Calibrar(p[i], ajuste)
			}
		}
		if g.Casos > 0 {
			g.TasaObservada = obs / float64(g.Casos)
			g.RiesgoMedio = pred / float64(g.Casos)
			g.RiesgoCalib = cal / float64(g.Casos)
		}
		out = append(out, g)
	}
	slices.SortFunc(out, func(a, b grupo) int { return cmp.Compare(b.RiesgoMedio, a.RiesgoMedio) })
	return out
}

func leerGold(ruta string) gold.Archivo {
	a, err := gold.Leer(ruta)
	if err != nil {
		log.Fatalf("%s: %v", ruta, err)
	}
	return a
}

func leerJSON(ruta string, v any) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		log.Fatalf("%s: %v", ruta, err)
	}
}

func escribirJSON(ruta string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(ruta, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("escrito %s", ruta)
}

// Calibrar corrige una probabilidad entrenada con pesos por clase.
func Calibrar(p, ajuste float64) float64 {
	p = math.Min(math.Max(p, 1e-15), 1-1e-15)
	return ml.Sigmoide(math.Log(p/(1-p)) - ajuste)
}
