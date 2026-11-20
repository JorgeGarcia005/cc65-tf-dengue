package main

// Regiones generadas dentro de secciones escritas a mano (informe PC4).
//
// Una sección .md puede tener regiones de la forma
//
//	<!-- generado:NOMBRE -->
//	…contenido que reescribe cmd/informe…
//	<!-- /generado -->
//
// NOMBRE es una de:
//
//	fuente:RUTA#FUNCION   la función FUNCION del archivo Go RUTA, copiada del
//	                      fuente con su comentario (FUNCION o (*Tipo).Metodo)
//	spin                  la tabla de reports/spin.json
//	cluster               la tabla de reports/bench_cluster_*.json
//	api                   la tabla de reports/bench_api_*.json
//
// Así el código del algoritmo y las cifras del informe salen del repositorio
// al generar, nunca se pegan a mano (AGENTS.md §3.6; observación 2 del
// profesor en la PC3). El test TestFragmentosSalenDelFuente lo comprueba.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var reRegion = regexp.MustCompile(`(?s)<!-- generado:([^ ]+) -->\n.*?<!-- /generado -->`)

// regenerarRegiones reescribe las regiones de cada .md de dir y devuelve los
// archivos que cambiaron.
func regenerarRegiones(dir string) ([]string, error) {
	archivos, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	var cambiados []string
	for _, a := range archivos {
		viejo, err := os.ReadFile(a)
		if err != nil {
			return nil, err
		}
		nuevo, err := expandirRegiones(viejo)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", a, err)
		}
		if !bytes.Equal(viejo, nuevo) {
			if err := os.WriteFile(a, nuevo, 0o644); err != nil {
				return nil, err
			}
			cambiados = append(cambiados, a)
		}
	}
	return cambiados, nil
}

// expandirRegiones devuelve el texto con cada región regenerada.
func expandirRegiones(texto []byte) ([]byte, error) {
	var primerErr error
	salida := reRegion.ReplaceAllFunc(texto, func(m []byte) []byte {
		nombre := string(reRegion.FindSubmatch(m)[1])
		cuerpo, err := generarRegion(nombre)
		if err != nil && primerErr == nil {
			primerErr = fmt.Errorf("región %s: %w", nombre, err)
		}
		return []byte("<!-- generado:" + nombre + " -->\n" + cuerpo + "<!-- /generado -->")
	})
	return salida, primerErr
}

func generarRegion(nombre string) (string, error) {
	switch {
	case strings.HasPrefix(nombre, "fuente:"):
		ruta, fn, ok := strings.Cut(strings.TrimPrefix(nombre, "fuente:"), "#")
		if !ok {
			return "", fmt.Errorf("falta #FUNCION")
		}
		codigo, err := extraerFuncion(ruta, fn)
		if err != nil {
			return "", err
		}
		return "```go\n// " + ruta + "\n" + codigo + "\n```\n", nil
	case nombre == "spin":
		return tablaSpin()
	case nombre == "cluster":
		return tablaCluster()
	case nombre == "api":
		return tablaAPI()
	}
	return "", fmt.Errorf("región desconocida")
}

// extraerFuncion devuelve el texto exacto de una función (con su comentario)
// tal como está en el archivo. fn es "Nombre" o "(*Tipo).Nombre".
func extraerFuncion(ruta, fn string) (string, error) {
	src, err := os.ReadFile(ruta)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, ruta, src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	tipo, nombre := "", fn
	if strings.HasPrefix(fn, "(") {
		t, n, ok := strings.Cut(strings.TrimPrefix(fn, "("), ").")
		if !ok {
			return "", fmt.Errorf("%s: se esperaba (*Tipo).Metodo", fn)
		}
		tipo, nombre = strings.TrimPrefix(t, "*"), n
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != nombre || (fd.Recv == nil) != (tipo == "") {
			continue
		}
		if tipo != "" && receptor(fd) != tipo {
			continue
		}
		ini := fd.Pos()
		if fd.Doc != nil {
			ini = fd.Doc.Pos()
		}
		return string(src[fset.Position(ini).Offset:fset.Position(fd.End()).Offset]), nil
	}
	return "", fmt.Errorf("%s no tiene la función %s", ruta, fn)
}

func receptor(fd *ast.FuncDecl) string {
	t := fd.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func coma(x float64, dec int) string {
	return strings.Replace(fmt.Sprintf("%.*f", dec, x), ".", ",", 1)
}

func tablaSpin() (string, error) {
	var r struct {
		Spin     string `json:"spin"`
		GCC      string `json:"gcc"`
		OK       bool   `json:"ok"`
		Corridas []struct {
			Corrida  string `json:"corrida"`
			Chequeo  string `json:"chequeo"`
			Estados  int    `json:"estados"`
			Errores  int    `json:"errores"`
			Esperado string `json:"esperado"`
		} `json:"corridas"`
	}
	if err := leerJSON("reports/spin.json", &r); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("| Corrida | Chequeo | Estados | Errores | Esperado |\n|---|---|---:|---:|---|\n")
	for _, c := range r.Corridas {
		esp := "0"
		if c.Esperado != "0" {
			esp = "> 0 (contraejemplo)"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %s |\n", c.Corrida, c.Chequeo, miles(c.Estados), c.Errores, esp)
	}
	fmt.Fprintf(&b, "\nFuente: `reports/spin.json` (%s, gcc %s, `bash spin/verificar.sh`).\n", r.Spin, r.GCC)
	return b.String(), nil
}

type reporteCluster struct {
	Fuente  string `json:"fuente"`
	Maquina struct {
		CPU    string `json:"cpu"`
		SO     string `json:"so"`
		NumCPU int    `json:"num_cpu"`
		Go     string `json:"go"`
	} `json:"maquina"`
	Protocolo struct {
		Reps    int `json:"repeticiones"`
		Epocas  int `json:"epocas_por_repeticion"`
		Workers int `json:"workers_por_nodo"`
	} `json:"protocolo"`
	Mediciones []struct {
		Nodos       int `json:"nodos"`
		SegPorEpoca struct {
			Media float64 `json:"media_recortada"`
			Inf   float64 `json:"ic95_inf"`
			Sup   float64 `json:"ic95_sup"`
		} `json:"seg_por_epoca"`
		Speedup    float64  `json:"speedup"`
		Eficiencia float64  `json:"eficiencia"`
		KarpFlatt  *float64 `json:"karp_flatt"`
		Desglose   struct {
			CalculoMax float64 `json:"seg_calculo_nodo_mas_lento"`
			Reduccion  float64 `json:"seg_reduccion"`
			Red        float64 `json:"seg_red_y_espera"`
			JSONShard  float64 `json:"seg_json_shard_mas_pesado"`
			Fraccion   float64 `json:"fraccion_red_y_espera"`
		} `json:"desglose_por_epoca"`
		Bytes       float64 `json:"bytes_por_epoca"`
		Equivalente bool    `json:"equivalente_bit_a_bit"`
	} `json:"mediciones"`
}

// clusterGold lee el benchmark del clúster sobre el gold real (el primero en
// orden de archivo).
func clusterGold() (reporteCluster, string, error) {
	archivos, _ := filepath.Glob("reports/bench_cluster_*_gold.json")
	slices.Sort(archivos)
	if len(archivos) == 0 {
		return reporteCluster{}, "", fmt.Errorf("no hay reports/bench_cluster_*_gold.json")
	}
	var r reporteCluster
	return r, archivos[0], leerJSON(archivos[0], &r)
}

func tablaCluster() (string, error) {
	r, archivo, err := clusterGold()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("| Nodos | ms por época (IC 95 %) | Speedup | Eficiencia | Karp-Flatt | Cómputo del nodo más lento (ms) | Red y espera (ms) | JSON del shard más pesado (ms) | KB por época |\n")
	b.WriteString("|---:|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range r.Mediciones {
		kf := "—"
		if m.KarpFlatt != nil {
			kf = coma(*m.KarpFlatt, 3)
		}
		fmt.Fprintf(&b, "| %d | %s (%s–%s) | %s | %s | %s | %s | %s | %s | %s |\n", m.Nodos,
			coma(1000*m.SegPorEpoca.Media, 2), coma(1000*m.SegPorEpoca.Inf, 2), coma(1000*m.SegPorEpoca.Sup, 2),
			coma(m.Speedup, 2), coma(m.Eficiencia, 2), kf, coma(1000*m.Desglose.CalculoMax, 2),
			coma(1000*m.Desglose.Red, 2), coma(1000*m.Desglose.JSONShard, 3), coma(m.Bytes/1024, 1))
	}
	fmt.Fprintf(&b, "\nFuente: `%s` (%s, %d hilos, %s, %s; %d repeticiones de %d épocas por número de nodos, %d worker por nodo; cada repetición verificada bit a bit contra el secuencial).\n",
		filepath.ToSlash(archivo), r.Maquina.CPU, r.Maquina.NumCPU, nombreSO(r.Maquina.SO), r.Maquina.Go,
		r.Protocolo.Reps, r.Protocolo.Epocas, r.Protocolo.Workers)
	return b.String(), nil
}

type reporteAPI struct {
	Maquina       string `json:"maquina"`
	SO            string `json:"so"`
	URL           string `json:"url"`
	VersionModelo string `json:"version_modelo"`
	Concurrencia  int    `json:"concurrencia"`
	Corridas      []struct {
		Nombre     string  `json:"nombre"`
		Peticiones int     `json:"peticiones"`
		Errores    int     `json:"errores"`
		Hits       int     `json:"cache_hits"`
		Misses     int     `json:"cache_misses"`
		RPS        float64 `json:"rps"`
		P50        float64 `json:"p50_ms"`
		P95        float64 `json:"p95_ms"`
		P99        float64 `json:"p99_ms"`
		Max        float64 `json:"max_ms"`
		Cumple     *bool   `json:"cumple_limite"`
	} `json:"corridas"`
	Limite float64 `json:"limite_p95_ms"`
}

func apiReal() (reporteAPI, string, error) {
	archivos, _ := filepath.Glob("reports/bench_api_*.json")
	slices.Sort(archivos)
	if len(archivos) == 0 {
		return reporteAPI{}, "", fmt.Errorf("no hay reports/bench_api_*.json")
	}
	var r reporteAPI
	return r, archivos[0], leerJSON(archivos[0], &r)
}

func tablaAPI() (string, error) {
	r, archivo, err := apiReal()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("| Corrida | Peticiones | Errores | HIT / MISS | Peticiones por segundo | p50 (ms) | p95 (ms) | p99 (ms) | Máx. (ms) | p95 < 100 ms |\n")
	b.WriteString("|---|---:|---:|---|---:|---:|---:|---:|---:|---|\n")
	for _, c := range r.Corridas {
		nombre := map[string]string{"fria": "Caché fría (consultas distintas)", "caliente": "Caché caliente (en Redis)"}[c.Nombre]
		cumple := "—"
		if c.Cumple != nil {
			cumple = map[bool]string{true: "sí", false: "no"}[*c.Cumple]
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %s / %s | %s | %s | %s | %s | %s | %s |\n", nombre, miles(c.Peticiones), c.Errores,
			miles(c.Hits), miles(c.Misses), miles(int(c.RPS+0.5)), coma(c.P50, 2), coma(c.P95, 2), coma(c.P99, 2), coma(c.Max, 2), cumple)
	}
	fmt.Fprintf(&b, "\nFuente: `%s` (`go run ./cmd/benchapi -exigir-p95 100ms` contra el clúster en Docker Compose con el gold real, %d clientes concurrentes, %s; modelo `%s`).\n",
		filepath.ToSlash(archivo), r.Concurrencia, nombreSO(strings.Split(r.SO, "/")[0]), r.VersionModelo)
	return b.String(), nil
}

func nombreSO(so string) string {
	switch so {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	}
	return so
}

func leerJSON(ruta string, v any) error {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", ruta, err)
	}
	return nil
}
