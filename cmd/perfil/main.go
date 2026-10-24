// Command perfil describe el CSV de dengue tal como llega del portal (capa
// bronze): filas, filas mal formadas, vacíos y valores por columna, casos por
// año y por clase, y edades por etapa de vida. Lo hace en paralelo y verifica
// que el resultado sea idéntico al de un solo worker.
//
// Uso:
//
//	go run ./cmd/perfil -entrada data/bronze/vigilancia_dengue_2000_2025.csv
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

type reporte struct {
	Generado  string         `json:"generado"`
	Archivo   string         `json:"archivo"`
	Bytes     int            `json:"bytes"`
	SHA256    string         `json:"sha256"`
	Workers   int            `json:"workers"`
	TamBloque int            `json:"tam_bloque_bytes"`
	Bloques   int            `json:"bloques"`
	SegSec    float64        `json:"seg_secuencial"`
	SegConc   float64        `json:"seg_concurrente"`
	Speedup   float64        `json:"speedup"`
	Equivale  bool           `json:"concurrente_igual_a_secuencial"`
	Perfil    *ingest.Perfil `json:"perfil"`
	Positivos int            `json:"positivos_con_signos_o_grave"`
	Negativos int            `json:"negativos_sin_signos"`
	TasaPosit float64        `json:"tasa_positivos"`
	AnosOrden []string       `json:"anos"`
}

func main() {
	entrada := flag.String("entrada", "data/bronze/vigilancia_dengue_2000_2025.csv", "CSV original del MINSA")
	salida := flag.String("salida", "reports/perfil_bronze.json", "reporte JSON")
	workers := flag.Int("workers", runtime.NumCPU(), "workers del perfil concurrente")
	tam := flag.Int("bloque", ingest.TamBloqueBytesPorDefecto, "tamaño objetivo de bloque en bytes")
	flag.Parse()

	datos, err := os.ReadFile(*entrada)
	if err != nil {
		log.Fatal(err)
	}
	suma := sha256.Sum256(datos)
	enc, cuerpo := ingest.SepararEncabezado(datos)
	if err := ingest.ValidarEncabezado(enc); err != nil {
		log.Fatal(err)
	}

	ini := time.Now()
	sec := ingest.PerfilarConcurrente(cuerpo, *tam, 1)
	tSec := time.Since(ini).Seconds()
	ini = time.Now()
	conc := ingest.PerfilarConcurrente(cuerpo, *tam, *workers)
	tConc := time.Since(ini).Seconds()

	a, _ := json.Marshal(sec)
	b, _ := json.Marshal(conc)

	rep := reporte{
		Generado: time.Now().Format(time.RFC3339), Archivo: filepath.Base(*entrada),
		Bytes: len(datos), SHA256: hex.EncodeToString(suma[:]), Workers: *workers, TamBloque: *tam,
		Bloques: len(ingest.PartirEnBloques(cuerpo, *tam)), SegSec: tSec, SegConc: tConc,
		Speedup: tSec / tConc, Equivale: string(a) == string(b), Perfil: conc,
	}
	for ano, c := range conc.PorAnoClase {
		rep.AnosOrden = append(rep.AnosOrden, ano)
		rep.Negativos += c[0]
		rep.Positivos += c[1] + c[2]
	}
	slices.Sort(rep.AnosOrden)
	if rep.Positivos+rep.Negativos > 0 {
		rep.TasaPosit = float64(rep.Positivos) / float64(rep.Positivos+rep.Negativos)
	}
	if !rep.Equivale {
		log.Fatal("el perfil concurrente difiere del secuencial")
	}

	if err := os.MkdirAll(filepath.Dir(*salida), 0o755); err != nil {
		log.Fatal(err)
	}
	out, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(*salida, append(out, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("archivo:   %s (%.1f MB, sha256 %s…)\n", rep.Archivo, float64(rep.Bytes)/1e6, rep.SHA256[:12])
	fmt.Printf("filas:     %d (mal formadas: %d)\n", conc.Filas, conc.FilasMalas)
	fmt.Printf("clases:    %d sin signos · %d con signos o grave (%.2f %%)\n", rep.Negativos, rep.Positivos, 100*rep.TasaPosit)
	fmt.Printf("tiempo:    %.3f s secuencial · %.3f s con %d workers (%.2fx), %d bloques\n", tSec, tConc, *workers, rep.Speedup, rep.Bloques)
	fmt.Printf("reporte:   %s\n", *salida)
}
