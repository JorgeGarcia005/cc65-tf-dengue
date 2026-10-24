// Command loader es la Parte A del sistema: lee el CSV original de dengue,
// lo limpia y valida en paralelo y escribe la capa silver.
//
// Modos:
//
//	go run ./cmd/loader                    # limpia → data/silver/dengue.csv + reports/limpieza.json
//	go run ./cmd/loader -bench -reps 20    # mide 1, 2, 4, … workers → reports/bench_loader_<host>.json
//
// En ambos modos se comprueba que el silver concurrente sea idéntico byte a
// byte (sha256) al de un solo worker.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

func main() {
	entrada := flag.String("entrada", "data/bronze/vigilancia_dengue_2000_2025.csv", "CSV original del MINSA")
	silver := flag.String("silver", "data/silver/dengue.csv", "CSV limpio de salida")
	reporte := flag.String("reporte", "reports/limpieza.json", "reporte de limpieza")
	workers := flag.Int("workers", runtime.NumCPU(), "workers")
	tam := flag.Int("bloque", ingest.TamBloqueBytesPorDefecto, "tamaño objetivo de bloque en bytes")
	bench := flag.Bool("bench", false, "medir speedup en vez de escribir el silver")
	reps := flag.Int("reps", 20, "repeticiones por configuración (modo -bench)")
	recorte := flag.Float64("recorte", 0.10, "media recortada (modo -bench)")
	listaWorkers := flag.String("lista-workers", "", "workers a medir, separados por coma (modo -bench)")
	flag.Parse()
	if *workers < 1 || *reps < 1 || *tam < 1 {
		log.Fatal("-workers, -reps y -bloque deben ser ≥ 1")
	}
	estadistica.ValidarRecorte(*recorte)

	ini := time.Now()
	datos, err := os.ReadFile(*entrada)
	if err != nil {
		log.Fatal(err)
	}
	tLectura := time.Since(ini).Seconds()
	enc, cuerpo := ingest.SepararEncabezado(datos)
	if err := ingest.ValidarEncabezado(enc); err != nil {
		log.Fatal(err)
	}

	if *bench {
		medir(cuerpo, *tam, *reps, *recorte, parsearLista(*listaWorkers))
		return
	}
	limpiar(cuerpo, datos, *tam, *workers, tLectura, *entrada, *silver, *reporte)
}

type repLimpieza struct {
	Generado      string          `json:"generado"`
	Entrada       string          `json:"entrada"`
	SHA256Entrada string          `json:"sha256_entrada"`
	Silver        string          `json:"silver"`
	SHA256Silver  string          `json:"sha256_silver"`
	Workers       int             `json:"workers"`
	Bloques       int             `json:"bloques"`
	SegLectura    float64         `json:"seg_lectura_disco"`
	SegSec        float64         `json:"seg_limpieza_secuencial"`
	SegConc       float64         `json:"seg_limpieza_concurrente"`
	Speedup       float64         `json:"speedup"`
	Equivale      bool            `json:"silver_concurrente_igual_al_secuencial"`
	Limpieza      ingest.Limpieza `json:"limpieza"`
}

func limpiar(cuerpo, datos []byte, tam, workers int, tLectura float64, entrada, rutaSilver, rutaRep string) {
	ini := time.Now()
	silSec, _ := ingest.LimpiarConcurrente(cuerpo, tam, 1)
	tSec := time.Since(ini).Seconds()
	ini = time.Now()
	sil, l := ingest.LimpiarConcurrente(cuerpo, tam, workers)
	tConc := time.Since(ini).Seconds()

	hEnt := sha256.Sum256(datos)
	hSil := sha256.Sum256(sil)
	rep := repLimpieza{
		Generado: time.Now().Format(time.RFC3339), Entrada: filepath.Base(entrada),
		SHA256Entrada: hex.EncodeToString(hEnt[:]), Silver: rutaSilver,
		SHA256Silver: hex.EncodeToString(hSil[:]), Workers: workers,
		Bloques: len(ingest.PartirEnBloques(cuerpo, tam)), SegLectura: tLectura,
		SegSec: tSec, SegConc: tConc, Speedup: tSec / tConc,
		Equivale: sha256.Sum256(silSec) == hSil, Limpieza: l,
	}
	if !rep.Equivale {
		log.Fatal("el silver concurrente difiere del secuencial")
	}
	escribirArchivo(rutaSilver, sil)
	escribirJSON(rutaRep, rep)

	fmt.Printf("entrada:   %d filas\n", l.Entradas)
	fmt.Printf("silver:    %d filas (%d descartadas, %d con comillas reparadas, %d ubigeos corregidos, %d filas idénticas conservadas)\n",
		l.Salidas, l.Descartadas, l.Reparadas, l.Corregidas, l.FilasIdenticas)
	for _, nombre := range l.OrdenReglas {
		if n := l.PorRegla[nombre]; n > 0 {
			fmt.Printf("  - %-55s %d\n", nombre, n)
		}
	}
	fmt.Printf("positivos: %d (%.2f %%)\n", l.Positivos, 100*l.TasaPositivos)
	fmt.Printf("tiempo:    %.3f s secuencial · %.3f s con %d workers (%.2fx) · lectura de disco %.3f s\n",
		tSec, tConc, workers, rep.Speedup, tLectura)
}

type medicion struct {
	Workers    int                 `json:"workers"`
	Segundos   estadistica.Resumen `json:"segundos"`
	Speedup    float64             `json:"speedup"`
	Eficiencia float64             `json:"eficiencia"`
	FilasPorS  float64             `json:"filas_por_segundo"`
	Equivale   bool                `json:"silver_igual_al_secuencial"`
}

type repBench struct {
	Generado   string         `json:"generado"`
	Maquina    map[string]any `json:"maquina"`
	Protocolo  map[string]any `json:"protocolo"`
	Filas      int            `json:"filas"`
	Bloques    int            `json:"bloques"`
	Mediciones []medicion     `json:"mediciones"`
}

func medir(cuerpo []byte, tam, reps int, recorte float64, workers []int) {
	if len(workers) == 0 {
		for w := 1; w <= 2*runtime.NumCPU(); w *= 2 {
			workers = append(workers, w)
		}
	}
	if !slices.Contains(workers, 1) {
		workers = append([]int{1}, workers...)
	}
	ref, l := ingest.LimpiarConcurrente(cuerpo, tam, 1)
	hRef := sha256.Sum256(ref)

	equivale := map[int]bool{}
	for _, w := range workers { // verificación + calentamiento
		s, _ := ingest.LimpiarConcurrente(cuerpo, tam, w)
		equivale[w] = sha256.Sum256(s) == hRef
		if !equivale[w] {
			log.Fatalf("workers=%d: silver distinto del secuencial", w)
		}
	}
	tiempos := map[int][]float64{}
	orden := slices.Clone(workers)
	r := rand.New(rand.NewPCG(2026, 11))
	for range reps {
		r.Shuffle(len(orden), func(a, b int) { orden[a], orden[b] = orden[b], orden[a] })
		for _, w := range orden {
			runtime.GC()
			ini := time.Now()
			ingest.LimpiarConcurrente(cuerpo, tam, w)
			tiempos[w] = append(tiempos[w], time.Since(ini).Seconds())
		}
	}
	rep := repBench{
		Generado: time.Now().Format(time.RFC3339), Maquina: maquina(), Filas: l.Entradas,
		Bloques: len(ingest.PartirEnBloques(cuerpo, tam)),
		Protocolo: map[string]any{
			"medida":        "limpieza completa en memoria (parseo + reglas + silver), sin lectura de disco",
			"calentamiento": 1, "repeticiones": reps, "orden": "barajado por repetición",
			"estimador": fmt.Sprintf("media recortada %.0f %% + IC95 bootstrap", recorte*100),
			"gc":        "runtime.GC() antes de cada medición",
		},
	}
	base := estadistica.MediaRecortada(tiempos[1], recorte)
	fmt.Printf("%8s %12s %22s %9s %10s %14s\n", "workers", "media rec.", "IC95", "speedup", "eficiencia", "filas/s")
	for _, w := range workers {
		res := estadistica.Resumir(tiempos[w], recorte)
		m := medicion{Workers: w, Segundos: res, Speedup: base / res.MediaRecortada,
			FilasPorS: float64(l.Entradas) / res.MediaRecortada, Equivale: equivale[w]}
		m.Eficiencia = m.Speedup / float64(w)
		rep.Mediciones = append(rep.Mediciones, m)
		fmt.Printf("%8d %10.1fms   [%7.1f, %7.1f]ms %8.2fx %9.0f%% %14.0f\n", w, res.MediaRecortada*1e3,
			res.IC95Inf*1e3, res.IC95Sup*1e3, m.Speedup, 100*m.Eficiencia, m.FilasPorS)
	}
	escribirJSON(filepath.Join("reports", "bench_loader_"+host()+".json"), rep)
}

func parsearLista(s string) []int {
	var out []int
	for _, c := range strings.Split(s, ",") {
		if c = strings.TrimSpace(c); c == "" {
			continue
		}
		v, err := strconv.Atoi(c)
		if err != nil || v < 1 {
			log.Fatalf("valor inválido %q en -lista-workers", c)
		}
		out = append(out, v)
	}
	return out
}

func maquina() map[string]any {
	m := map[string]any{"so": runtime.GOOS, "arquitectura": runtime.GOARCH, "go": runtime.Version(),
		"num_cpu": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0)}
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "model name") {
				m["cpu"] = strings.TrimSpace(strings.SplitN(l, ":", 2)[1])
				break
			}
		}
	}
	return m
}

func host() string {
	h, _ := os.Hostname()
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, strings.ToLower(h))
}

func escribirArchivo(ruta string, b []byte) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o644); err != nil {
		log.Fatal(err)
	}
}

func escribirJSON(ruta string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	escribirArchivo(ruta, append(b, '\n'))
	log.Printf("escrito %s", ruta)
}
