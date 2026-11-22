// Command evaluacion genera reports/evaluacion.json (#59): métricas en test,
// ROC, calibración, riesgo cruzado departamento × semana × grupo de edad,
// prioridades por casos esperados y las cifras de escalabilidad y latencia
// copiadas de los benchmarks. Mismas entradas → mismo archivo byte a byte.
//
//	go run ./cmd/evaluacion
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
)

type opciones struct {
	gold, modelo, entrenamiento, cluster, benchAPI, salida string
	minCasos, top, puntosROC, deciles                      int
}

func main() {
	var o opciones
	flag.StringVar(&o.gold, "gold", "data/gold", "carpeta con test.bin")
	flag.StringVar(&o.modelo, "modelo", "data/gold/modelo.json", "modelo que evalúa (el que sirve la API)")
	flag.StringVar(&o.entrenamiento, "entrenamiento", "reports/entrenamiento.json", "AUC de referencia (vacío = no comparar)")
	flag.StringVar(&o.cluster, "cluster", "reports/bench_cluster_deez_gold.json", "benchmark de escalabilidad por nodos")
	flag.StringVar(&o.benchAPI, "api", "reports/bench_api_deez.json", "benchmark de latencia de la API")
	flag.StringVar(&o.salida, "salida", "reports/evaluacion.json", "reporte de salida")
	flag.IntVar(&o.minCasos, "min-casos", 30, "celdas con menos casos se marcan insuficientes")
	flag.IntVar(&o.top, "top", 15, "combinaciones prioritarias")
	flag.IntVar(&o.puntosROC, "puntos-roc", 201, "puntos de la curva ROC")
	flag.IntVar(&o.deciles, "tramos", 10, "tramos de la curva de calibración")
	flag.Parse()
	rep, err := ejecutar(o)
	if err != nil {
		log.Fatal(err)
	}
	top := fmt.Sprintf("sin combinaciones con ≥ %d casos", o.minCasos)
	if len(rep.Prioridades) > 0 {
		p := rep.Prioridades[0]
		top = fmt.Sprintf("%s semana %d %s (%.1f casos esperados)", p.Departamento, p.Semana, p.Etapa, p.CasosEsperados)
	}
	fmt.Printf("%s: %d filas de test · AUC %.4f · Brier %.4f · %d celdas · top: %s\n",
		o.salida, rep.FilasTest, rep.AUC, rep.Brier, len(rep.Riesgo.Celdas), top)
}

func ejecutar(o opciones) (evaluacion.Reporte, error) {
	if o.top < 0 || o.minCasos < 1 || o.puntosROC < 2 || o.deciles < 1 {
		return evaluacion.Reporte{}, fmt.Errorf("parámetros inválidos: -top ≥ 0, -min-casos ≥ 1, -puntos-roc ≥ 2, -tramos ≥ 1")
	}
	var fuentes []evaluacion.Fuente
	leer := func(ruta string) ([]byte, error) {
		b, err := os.ReadFile(ruta)
		if err == nil {
			fuentes = append(fuentes, evaluacion.NuevaFuente(filepath.ToSlash(ruta), b))
		}
		return b, err
	}
	rutaTest := filepath.Join(o.gold, "test.bin")
	bTest, err := leer(rutaTest)
	if err != nil {
		return evaluacion.Reporte{}, err
	}
	test, err := gold.Decodificar(bTest)
	if err != nil {
		return evaluacion.Reporte{}, fmt.Errorf("%s: %w", rutaTest, err)
	}
	bModelo, err := leer(o.modelo)
	if err != nil {
		return evaluacion.Reporte{}, err
	}
	m, err := api.CargarModelo(bModelo)
	if err != nil {
		return evaluacion.Reporte{}, err
	}
	var aucRef float64
	if o.entrenamiento != "" {
		b, err := leer(o.entrenamiento)
		if err != nil {
			return evaluacion.Reporte{}, err
		}
		var e struct {
			Test *struct{ AUC *float64 } `json:"metricas_test"`
		}
		if err := json.Unmarshal(b, &e); err != nil {
			return evaluacion.Reporte{}, fmt.Errorf("%s: %w", o.entrenamiento, err)
		}
		// Si se pidió comparar, el AUC tiene que estar: nunca se omite en silencio.
		if e.Test == nil || e.Test.AUC == nil || !(*e.Test.AUC > 0 && *e.Test.AUC <= 1) {
			return evaluacion.Reporte{}, fmt.Errorf("%s: falta metricas_test.AUC en (0, 1]; use -entrenamiento \"\" para no comparar", o.entrenamiento)
		}
		aucRef = *e.Test.AUC
	}
	bCluster, err := leer(o.cluster)
	if err != nil {
		return evaluacion.Reporte{}, err
	}
	bAPI, err := leer(o.benchAPI)
	if err != nil {
		return evaluacion.Reporte{}, err
	}
	rep, err := evaluacion.Generar(evaluacion.Entrada{
		Test: test.Dataset, Pesos: m.Pesos, Esquema: m.Esquema, Umbral: m.Umbral,
		AjustePrior: m.AjustePrior, VersionModelo: m.Version, AUCReferencia: aucRef,
		BenchCluster: bCluster, BenchAPI: bAPI, Fuentes: fuentes, MinCasos: o.minCasos,
		TopN: o.top, PuntosROC: o.puntosROC, Deciles: o.deciles,
	})
	if err != nil {
		return rep, err
	}
	b, err := json.Marshal(rep)
	if err != nil {
		return rep, err
	}
	return rep, os.WriteFile(o.salida, append(b, '\n'), 0o644)
}
