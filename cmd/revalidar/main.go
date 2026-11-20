// Command revalidar compara, campo por campo, los reportes JSON de referencia
// (los versionados) con los que se acaban de regenerar en otra máquina.
// Ignora solo lo que depende del hardware o del momento: "generado",
// "workers", "speedup" y cualquier campo "seg_*". Todo lo demás, incluidos
// los float64, debe coincidir bit a bit (AGENTS.md §3.3 y §3.6).
//
// Uso:
//
//	go run ./cmd/revalidar -ref /tmp/ref -nuevo reports
//	go run ./cmd/revalidar -ref /tmp/ref -nuevo reports entrenamiento.json
//
// Sale con código 1 si algún archivo difiere. scripts/revalidar.sh arma el
// flujo completo (extrae la referencia de git, regenera y restaura reports/).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// porDefecto son los reportes que el issue #23 pide revalidar.
var porDefecto = []string{"perfil_bronze.json", "limpieza.json", "split.json", "entrenamiento.json"}

func main() {
	ref := flag.String("ref", "", "carpeta con los reportes de referencia")
	nuevo := flag.String("nuevo", "reports", "carpeta con los reportes regenerados")
	flag.Parse()
	if *ref == "" {
		fmt.Fprintln(os.Stderr, "falta -ref")
		os.Exit(2)
	}
	archivos := flag.Args()
	if len(archivos) == 0 {
		archivos = porDefecto
	}
	fallo := false
	for _, nombre := range archivos {
		a, errA := os.ReadFile(filepath.Join(*ref, nombre))
		b, errB := os.ReadFile(filepath.Join(*nuevo, nombre))
		if errA != nil || errB != nil {
			fmt.Printf("%-20s ERROR: %v %v\n", nombre, errA, errB)
			fallo = true
			continue
		}
		difs, err := Comparar(a, b)
		switch {
		case err != nil:
			fmt.Printf("%-20s ERROR: %v\n", nombre, err)
			fallo = true
		case len(difs) == 0:
			fmt.Printf("%-20s IGUAL (salvo generado, workers, speedup y seg_*)\n", nombre)
		default:
			fmt.Printf("%-20s DIFIERE en %d campos:\n", nombre, len(difs))
			for _, d := range difs {
				fmt.Println("    " + d)
			}
			fallo = true
		}
	}
	if fallo {
		os.Exit(1)
	}
}

// ignorado dice si un campo depende solo del hardware o del momento.
func ignorado(clave string) bool {
	switch clave {
	case "generado", "workers", "speedup":
		return true
	}
	return strings.HasPrefix(clave, "seg_")
}

// Comparar devuelve las rutas (a.b[2].c) de los campos que difieren entre dos
// JSON, ordenadas. Los números se comparan como float64 bit a bit.
func Comparar(ref, nuevo []byte) ([]string, error) {
	var a, b any
	if err := decodificar(ref, &a); err != nil {
		return nil, fmt.Errorf("referencia: %w", err)
	}
	if err := decodificar(nuevo, &b); err != nil {
		return nil, fmt.Errorf("nuevo: %w", err)
	}
	var difs []string
	comparar("", a, b, &difs)
	sort.Strings(difs)
	return difs, nil
}

func decodificar(datos []byte, v *any) error {
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.UseNumber()
	return dec.Decode(v)
}

func comparar(ruta string, a, b any, difs *[]string) {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			*difs = append(*difs, fmt.Sprintf("%s: tipo distinto", nombreRuta(ruta)))
			return
		}
		for k, va := range x {
			if ignorado(k) {
				continue
			}
			vb, ok := y[k]
			if !ok {
				*difs = append(*difs, fmt.Sprintf("%s: solo en la referencia", unir(ruta, k)))
				continue
			}
			comparar(unir(ruta, k), va, vb, difs)
		}
		for k := range y {
			if _, ok := x[k]; !ok && !ignorado(k) {
				*difs = append(*difs, fmt.Sprintf("%s: solo en el nuevo", unir(ruta, k)))
			}
		}
	case []any:
		y, ok := b.([]any)
		if !ok {
			*difs = append(*difs, fmt.Sprintf("%s: tipo distinto", nombreRuta(ruta)))
			return
		}
		if len(x) != len(y) {
			*difs = append(*difs, fmt.Sprintf("%s: %d elementos vs %d", nombreRuta(ruta), len(x), len(y)))
			return
		}
		for i := range x {
			comparar(fmt.Sprintf("%s[%d]", ruta, i), x[i], y[i], difs)
		}
	case json.Number:
		y, ok := b.(json.Number)
		if !ok {
			*difs = append(*difs, fmt.Sprintf("%s: tipo distinto", nombreRuta(ruta)))
			return
		}
		fa, errA := strconv.ParseFloat(string(x), 64)
		fb, errB := strconv.ParseFloat(string(y), 64)
		if errA != nil || errB != nil || math.Float64bits(fa) != math.Float64bits(fb) {
			*difs = append(*difs, fmt.Sprintf("%s: %s vs %s", nombreRuta(ruta), x, y))
		}
	default:
		if a != b {
			*difs = append(*difs, fmt.Sprintf("%s: %v vs %v", nombreRuta(ruta), a, b))
		}
	}
}

func unir(ruta, clave string) string {
	if ruta == "" {
		return clave
	}
	return ruta + "." + clave
}

func nombreRuta(ruta string) string {
	if ruta == "" {
		return "(raíz)"
	}
	return ruta
}
