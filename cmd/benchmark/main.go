// Command benchmark mide el tiempo de una pasada de gradiente (una época de
// descenso de gradiente full-batch) en versión secuencial y con el worker pool,
// y busca el punto de equilibrio: el tamaño de datos desde el cual la versión
// concurrente es más rápida que la secuencial.
//
// Protocolo (fijado en código para que sea reproducible):
//  1. Dataset sintético determinista con la misma forma que el gold de dengue.
//  2. Calentamiento: 2 pasadas por configuración, descartadas.
//  3. R repeticiones; en cada una, el orden de las configuraciones se baraja
//     para que la deriva térmica o de caché no favorezca a ninguna.
//  4. Estimador: media recortada al 10 % con IC 95 % por bootstrap.
//  5. Antes de medir se comprueba que cada configuración da el mismo gradiente,
//     bit a bit, que la secuencial.
//
// Uso:
//
//	go run ./cmd/benchmark -filas 1000000 -features 40 -reps 20
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

type medicion struct {
	Config       string              `json:"config"`
	Workers      int                 `json:"workers"`
	Segundos     estadistica.Resumen `json:"segundos"`
	Speedup      float64             `json:"speedup"`
	Eficiencia   float64             `json:"eficiencia"`
	Equivalencia bool                `json:"equivalente_bit_a_bit"`
}

type puntoEquilibrio struct {
	Filas       int     `json:"filas"`
	Bloques     int     `json:"bloques"`
	SegSec      float64 `json:"seg_secuencial"`
	SegConc     float64 `json:"seg_concurrente"`
	Speedup     float64 `json:"speedup"`
	ConvieneSiN bool    `json:"conviene_concurrente"`
}

type reporte struct {
	Generado   string            `json:"generado"`
	Fuente     string            `json:"fuente"`
	Maquina    map[string]any    `json:"maquina"`
	Protocolo  map[string]any    `json:"protocolo"`
	Mediciones []medicion        `json:"mediciones"`
	Equilibrio []puntoEquilibrio `json:"punto_equilibrio"`
	// FilasEquilibrio es el menor tamaño medido desde el cual la versión
	// concurrente es más rápida en ese y en todos los tamaños mayores.
	// Es nil si en ningún tamaño medido conviene la versión concurrente.
	FilasEquilibrio   *int `json:"filas_equilibrio"`
	WorkersEquilibrio int  `json:"workers_equilibrio"`
}

func main() {
	filas := flag.Int("filas", 1_000_000, "filas del dataset sintético")
	features := flag.Int("features", 40, "features por fila (sin el sesgo)")
	reps := flag.Int("reps", 20, "repeticiones medidas por configuración")
	recorte := flag.Float64("recorte", 0.10, "fracción recortada en cada extremo")
	tamBloque := flag.Int("bloque", ml.TamBloquePorDefecto, "filas por bloque")
	listaWorkers := flag.String("workers", "", "workers a medir, separados por coma (por defecto 1,2,4,… hasta 2·NumCPU)")
	listaEquilibrio := flag.String("equilibrio", "1000,4096,10000,30000,100000,300000,1000000", "tamaños para el punto de equilibrio")
	salida := flag.String("salida", "", "archivo JSON (por defecto reports/bench_gradiente_<host>.json)")
	semilla := flag.Uint64("semilla", 2026, "semilla del dataset y del barajado")
	flag.Parse()

	workers := parsearLista(*listaWorkers)
	if len(workers) == 0 {
		for w := 1; w <= 2*runtime.NumCPU(); w *= 2 {
			workers = append(workers, w)
		}
	}
	tamanos := parsearLista(*listaEquilibrio)

	log.Printf("generando dataset sintético: %d filas × %d features", *filas, *features)
	ds, _ := ml.Sintetico(*filas, *features, *semilla)
	cfg := ml.Config{TamBloque: *tamBloque, Lambda: 1e-4}
	w := make([]float64, ds.D+1)
	for j := range w {
		w[j] = 0.01 * float64(j%7-3)
	}
	r := rand.New(rand.NewPCG(*semilla, 7))

	rep := reporte{
		Generado: time.Now().Format(time.RFC3339),
		Fuente:   fmt.Sprintf("sintetico(filas=%d, features=%d, semilla=%d)", *filas, *features, *semilla),
		Maquina:  maquina(),
		Protocolo: map[string]any{
			"calentamiento": 2, "repeticiones": *reps, "orden": "barajado por repetición",
			"estimador": fmt.Sprintf("media recortada %.0f %% + IC95 bootstrap (2000)", *recorte*100),
			"medida":    "una pasada de gradiente completa (= una época)", "tam_bloque": *tamBloque,
		},
	}

	tiempos, equivalentes := medir(ds, cfg, w, workers, *reps, r)
	base := estadistica.MediaRecortada(tiempos["secuencial"], *recorte)
	nombres := []string{"secuencial"}
	for _, k := range workers {
		nombres = append(nombres, nombrePool(k))
	}
	for i, n := range nombres {
		m := medicion{Config: n, Segundos: estadistica.Resumir(tiempos[n], *recorte), Equivalencia: equivalentes[n]}
		m.Workers = 1
		if i > 0 {
			m.Workers = workers[i-1]
		}
		m.Speedup = base / m.Segundos.MediaRecortada
		m.Eficiencia = m.Speedup / float64(m.Workers)
		rep.Mediciones = append(rep.Mediciones, m)
	}

	mejor := slices.Max(workers)
	mejor = min(mejor, runtime.NumCPU())
	for _, n := range tamanos {
		if n > ds.N {
			continue
		}
		sub := ds.Prefijo(n)
		t, _ := medir(sub, cfg, w, []int{mejor}, *reps, r)
		sec := estadistica.MediaRecortada(t["secuencial"], *recorte)
		conc := estadistica.MediaRecortada(t[nombrePool(mejor)], *recorte)
		rep.Equilibrio = append(rep.Equilibrio, puntoEquilibrio{
			Filas: n, Bloques: ml.NumBloques(n, *tamBloque), SegSec: sec, SegConc: conc,
			Speedup: sec / conc, ConvieneSiN: conc < sec,
		})
	}
	rep.FilasEquilibrio = filasEquilibrio(rep.Equilibrio)
	rep.WorkersEquilibrio = mejor

	imprimir(rep)
	escribir(rep, *salida)
}

// medir devuelve, por configuración, los segundos de cada repetición.
func medir(ds *ml.Dataset, cfg ml.Config, w []float64, workers []int, reps int, r *rand.Rand) (map[string][]float64, map[string]bool) {
	calcs := map[string]ml.Calculador{"secuencial": ml.NuevoSecuencial(ds, cfg)}
	var pools []*ml.Pool
	for _, k := range workers {
		p := ml.NuevoPool(ds, cfg, k)
		pools = append(pools, p)
		calcs[nombrePool(k)] = p
	}
	defer func() {
		for _, p := range pools {
			p.Cerrar()
		}
	}()

	nombres := make([]string, 0, len(calcs))
	for n := range calcs {
		nombres = append(nombres, n)
	}
	slices.Sort(nombres)

	ref := calcs["secuencial"].Gradiente(w)
	equivalentes := map[string]bool{}
	for _, n := range nombres {
		g := calcs[n].Gradiente(w)
		equivalentes[n] = mismosBits(ref.Grad, g.Grad) && math.Float64bits(ref.Perdida) == math.Float64bits(g.Perdida)
		if !equivalentes[n] {
			log.Fatalf("%s no es equivalente bit a bit a la versión secuencial", n)
		}
		calcs[n].Gradiente(w) // segunda pasada de calentamiento
	}

	tiempos := map[string][]float64{}
	for i := 0; i < reps; i++ {
		r.Shuffle(len(nombres), func(a, b int) { nombres[a], nombres[b] = nombres[b], nombres[a] })
		for _, n := range nombres {
			ini := time.Now()
			calcs[n].Gradiente(w)
			tiempos[n] = append(tiempos[n], time.Since(ini).Seconds())
		}
	}
	return tiempos, equivalentes
}

func filasEquilibrio(ps []puntoEquilibrio) *int {
	var res *int
	for i := len(ps) - 1; i >= 0; i-- {
		if !ps[i].ConvieneSiN {
			break
		}
		f := ps[i].Filas
		res = &f
	}
	return res
}

func mismosBits(a, b []float64) bool {
	for j := range a {
		if math.Float64bits(a[j]) != math.Float64bits(b[j]) {
			return false
		}
	}
	return len(a) == len(b)
}

func nombrePool(k int) string { return fmt.Sprintf("pool-%02d", k) }

func parsearLista(s string) []int {
	var out []int
	for _, c := range strings.Split(s, ",") {
		if c = strings.TrimSpace(c); c == "" {
			continue
		}
		v, err := strconv.Atoi(c)
		if err != nil || v <= 0 {
			log.Fatalf("valor inválido %q en la lista", c)
		}
		out = append(out, v)
	}
	return out
}

func maquina() map[string]any {
	m := map[string]any{
		"so": runtime.GOOS, "arquitectura": runtime.GOARCH, "go": runtime.Version(),
		"num_cpu": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0),
	}
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

func imprimir(rep reporte) {
	fmt.Printf("\n%-12s %8s %12s %22s %9s %10s %6s\n", "config", "workers", "media rec.", "IC95", "speedup", "eficiencia", "bits")
	for _, m := range rep.Mediciones {
		fmt.Printf("%-12s %8d %10.2fms   [%7.2f, %7.2f]ms %8.2fx %9.0f%% %6v\n",
			m.Config, m.Workers, m.Segundos.MediaRecortada*1e3, m.Segundos.IC95Inf*1e3, m.Segundos.IC95Sup*1e3,
			m.Speedup, m.Eficiencia*100, m.Equivalencia)
	}
	fmt.Printf("\n%10s %8s %12s %12s %9s\n", "filas", "bloques", "secuencial", "concurrente", "speedup")
	for _, p := range rep.Equilibrio {
		fmt.Printf("%10d %8d %10.3fms %10.3fms %8.2fx\n", p.Filas, p.Bloques, p.SegSec*1e3, p.SegConc*1e3, p.Speedup)
	}
	if rep.FilasEquilibrio == nil {
		fmt.Printf("\npunto de equilibrio: con %d workers la concurrencia no convino en ningún tamaño medido\n", rep.WorkersEquilibrio)
	} else {
		fmt.Printf("\npunto de equilibrio: con %d workers la concurrencia conviene desde %d filas\n", rep.WorkersEquilibrio, *rep.FilasEquilibrio)
	}
}

func escribir(rep reporte, ruta string) {
	if ruta == "" {
		host, _ := os.Hostname()
		host = strings.Map(func(r rune) rune {
			if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			return '-'
		}, strings.ToLower(host))
		ruta = filepath.Join("reports", "bench_gradiente_"+host+".json")
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		log.Fatal(err)
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(ruta, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("reporte escrito en %s", ruta)
}
