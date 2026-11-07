// Command gold construye la matriz de entrenamiento desde el CSV silver:
// filtra los años útiles, separa train/test de forma determinista, ajusta el
// esquema de variables solo con el train y escribe los archivos binarios y los
// shards de entrenamiento para los nodos.
//
// Uso:
//
//	go run ./cmd/gold -shards 3
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

type conjunto struct {
	Filas     int     `json:"filas"`
	Positivos int     `json:"positivos"`
	Tasa      float64 `json:"tasa_positivos"`
}

type reporte struct {
	Generado       string            `json:"generado"`
	FilasSilver    int               `json:"filas_silver"`
	AnoMin         int               `json:"ano_min"`
	FilasExcluidas int               `json:"filas_excluidas_por_ano"`
	FracTest       float64           `json:"fraccion_test"`
	Train          conjunto          `json:"train"`
	Test           conjunto          `json:"test"`
	D              int               `json:"variables"`
	TamBloque      int               `json:"tam_bloque"`
	Bloques        int               `json:"bloques_train"`
	Shards         []map[string]int  `json:"shards"`
	Esquema        *features.Esquema `json:"esquema"`
	SegLectura     float64           `json:"seg_lectura_silver"`
}

func main() {
	entrada := flag.String("silver", "data/silver/dengue.csv", "CSV silver")
	salida := flag.String("salida", "data/gold", "carpeta de salida")
	anoMin := flag.Int("ano-min", 2011, "primer año usado para entrenar")
	fracTest := flag.Float64("test", 0.20, "fracción de prueba")
	minDepto := flag.Int("min-depto", 100, "casos mínimos en train para tener columna propia de departamento")
	shards := flag.Int("shards", 3, "shards de entrenamiento para los nodos")
	tam := flag.Int("bloque", ml.TamBloquePorDefecto, "filas por bloque")
	workers := flag.Int("workers", runtime.NumCPU(), "workers para leer el silver")
	flag.Parse()
	if *fracTest <= 0 || *fracTest >= 1 || *shards < 1 || *tam < 1 {
		log.Fatal("-test debe estar en (0,1); -shards y -bloque ≥ 1")
	}

	ini := time.Now()
	datos, err := os.ReadFile(*entrada)
	if err != nil {
		log.Fatal(err)
	}
	regs, err := ingest.LeerSilverConcurrente(datos, ingest.TamBloqueBytesPorDefecto, *workers)
	if err != nil {
		log.Fatal(err)
	}
	tLectura := time.Since(ini).Seconds()

	// El índice global de cada fila es su posición en el silver: no depende de
	// workers ni nodos, así que el split tampoco.
	var train, test []ingest.Registro
	excluidas := 0
	for i, r := range regs {
		switch {
		case r.Ano < *anoMin:
			excluidas++
		case features.EsTest(i, *fracTest):
			test = append(test, r)
		default:
			train = append(train, r)
		}
	}
	esq, err := features.Ajustar(train, *anoMin, *minDepto)
	if err != nil {
		log.Fatal(err)
	}

	aTrain := armar(esq, train, *tam)
	aTest := armar(esq, test, *tam)
	if err := os.MkdirAll(*salida, 0o755); err != nil {
		log.Fatal(err)
	}
	guardar(filepath.Join(*salida, "train.bin"), aTrain)
	guardar(filepath.Join(*salida, "test.bin"), aTest)
	partes, err := gold.Particionar(aTrain, *shards)
	if err != nil {
		log.Fatal(err)
	}
	rep := reporte{
		Generado: time.Now().Format(time.RFC3339), FilasSilver: len(regs), AnoMin: *anoMin,
		FilasExcluidas: excluidas, FracTest: *fracTest, Train: resumen(train), Test: resumen(test),
		D: esq.D(), TamBloque: *tam, Bloques: ml.NumBloques(aTrain.N, *tam), Esquema: esq,
		SegLectura: tLectura,
	}
	for s, p := range partes {
		guardar(filepath.Join(*salida, fmt.Sprintf("shard_%d.bin", s)), p)
		rep.Shards = append(rep.Shards, map[string]int{
			"shard": s, "filas": p.N, "bloque_inicial": p.BloqueInicial, "bloques": ml.NumBloques(p.N, p.TamBloque),
		})
	}
	escribirJSON(filepath.Join(*salida, "esquema.json"), esq)
	escribirJSON("reports/split.json", rep)

	fmt.Printf("silver:  %d filas (%.2f s, %d workers); excluidas por año < %d: %d\n", len(regs), tLectura, *workers, *anoMin, excluidas)
	fmt.Printf("train:   %d filas, %.2f %% positivos · test: %d filas, %.2f %% positivos\n",
		rep.Train.Filas, 100*rep.Train.Tasa, rep.Test.Filas, 100*rep.Test.Tasa)
	fmt.Printf("gold:    %d variables, %d bloques de %d filas, %d shards en %s\n", esq.D(), rep.Bloques, *tam, *shards, *salida)
}

func armar(esq *features.Esquema, regs []ingest.Registro, tam int) gold.Archivo {
	d := esq.D()
	x := make([]float64, len(regs)*d)
	y := make([]float64, len(regs))
	for i, r := range regs {
		esq.Llenar(r, x[i*d:(i+1)*d])
		y[i] = r.Objetivo()
	}
	ds, err := ml.NuevoDataset(x, y, d)
	if err != nil {
		log.Fatal(err)
	}
	return gold.Archivo{Dataset: ds, TamBloque: tam}
}

func resumen(regs []ingest.Registro) conjunto {
	c := conjunto{Filas: len(regs)}
	for _, r := range regs {
		if r.Clase >= 1 {
			c.Positivos++
		}
	}
	if c.Filas > 0 {
		c.Tasa = float64(c.Positivos) / float64(c.Filas)
	}
	return c
}

func guardar(ruta string, a gold.Archivo) {
	if err := gold.Escribir(ruta, a); err != nil {
		log.Fatal(err)
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
}
