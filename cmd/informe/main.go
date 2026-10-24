// Command informe genera las secciones del informe que llevan cifras, a partir
// de los reportes JSON en reports/. Así ninguna cifra del informe se escribe a
// mano (AGENTS.md §3.6): si los datos cambian, se regenera.
//
// Uso:
//
//	go run ./cmd/informe            # regenera todas las secciones disponibles
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

type conteo struct {
	Valor string `json:"valor"`
	N     int    `json:"n"`
}

type perfil struct {
	Archivo  string   `json:"archivo"`
	Bytes    int      `json:"bytes"`
	SHA256   string   `json:"sha256"`
	Workers  int      `json:"workers"`
	Bloques  int      `json:"bloques"`
	SegSec   float64  `json:"seg_secuencial"`
	SegConc  float64  `json:"seg_concurrente"`
	Speedup  float64  `json:"speedup"`
	Equivale bool     `json:"concurrente_igual_a_secuencial"`
	Pos      int      `json:"positivos_con_signos_o_grave"`
	Neg      int      `json:"negativos_sin_signos"`
	Tasa     float64  `json:"tasa_positivos"`
	Anos     []string `json:"anos"`
	Perfil   struct {
		Filas      int `json:"filas"`
		FilasMalas int `json:"filas_mal_formadas"`
		ErrComill  int `json:"err_comillas"`
		ErrColum   int `json:"err_columnas"`
		Columnas   []struct {
			Nombre    string   `json:"nombre"`
			Vacios    int      `json:"vacios"`
			Distintos int      `json:"distintos"`
			Top       []conteo `json:"top"`
		} `json:"columnas"`
		Edad struct {
			Tramos  []string `json:"tramos"`
			Conteos []int    `json:"conteos"`
			NoNum   int      `json:"no_numerica"`
		} `json:"edad_anios"`
		PorAno map[string][3]int `json:"por_ano_y_clase"`
	} `json:"perfil"`
}

const plantillaDatos = `<!-- Generado por cmd/informe desde reports/perfil_bronze.json. No editar a mano. -->

# 2. Datos: perfil del dataset original (capa bronze)

## 2.1 Fuente

- **Dataset:** Vigilancia epidemiológica de dengue, MINSA / CDC Perú, Plataforma Nacional de
  Datos Abiertos (<https://www.datosabiertos.gob.pe/dataset/vigilancia-epidemiol%C3%B3gica-de-dengue>).
- **Archivo:** ` + "`{{.Archivo}}`" + ` · {{mb .Bytes}} MB · sha256 ` + "`{{.SHA256}}`" + `.
- **Unidad de análisis:** un caso de dengue notificado a la Red Nacional de Epidemiología.
- **Filas:** {{miles .Perfil.Filas}} ({{miles .Perfil.FilasMalas}} mal formadas: {{.Perfil.ErrComill}} por comillas y
  {{.Perfil.ErrColum}} por número de columnas).

## 2.2 Variables

| Columna | Vacíos | Valores distintos | Valores más frecuentes |
|---|---:|---:|---|
{{- range .Perfil.Columnas}}
| ` + "`{{.Nombre}}`" + ` | {{miles .Vacios}} | {{miles .Distintos}} | {{top .Top}} |
{{- end}}

## 2.3 Variable objetivo

El objetivo es **1** si el caso es *dengue con signos de alarma* (CIE-10 A97.1) o *dengue grave*
(A97.2) y **0** si es *dengue sin signos de alarma* (A97.0). Según la guía clínica, los casos
con signos de alarma requieren hospitalización, por lo que el objetivo funciona como indicador
de **necesidad de hospitalización** (validado con el profesor, ADR-0003).

- Sin signos (0): **{{miles .Neg}}** · Con signos o grave (1): **{{miles .Pos}}** · Proporción de positivos:
  **{{pct .Tasa}} %**. Hay desbalance de clases, que se trata con pesos por clase.

| Año | Sin signos (A97.0) | Con signos (A97.1) | Grave (A97.2) | % positivos |
|---|---:|---:|---:|---:|
{{- range $a := .Anos}}{{with index $.Perfil.PorAno $a}}
| {{$a}} | {{miles (index . 0)}} | {{miles (index . 1)}} | {{miles (index . 2)}} | {{tasaAno .}} |
{{- end}}{{end}}

**Hallazgo:** la categoría *con signos de alarma* aparece de forma marginal en 2010 y se usa
de forma sistemática desde 2011, tras la adopción de la clasificación de la OMS de 2009. Antes
los casos se registran casi todos como *sin signos*, así que esos años no sirven para aprender
el objetivo: **el modelo se entrena con los años desde 2011**, mientras que la carga y la
limpieza procesan todas las filas.

## 2.4 Edad por etapa de vida (MINSA)

| Etapa | Casos |
|---|---:|
{{- range $i, $t := .Perfil.Edad.Tramos}}
| {{$t}} | {{miles (index $.Perfil.Edad.Conteos $i)}} |
{{- end}}

Filas con edad no numérica o ` + "`tipo_edad`" + ` desconocido: **{{miles .Perfil.Edad.NoNum}}**.

## 2.5 Procesamiento concurrente del perfil

El archivo se lee en memoria y se divide en **{{.Bloques}} bloques** de ~1 MiB alineados a fin
de línea. Un *worker pool* procesa los bloques y cada uno deja su resultado en un slot propio;
los resultados se unen en orden de bloque. El perfil con {{.Workers}} workers tomó
{{seg .SegConc}} s frente a {{seg .SegSec}} s con uno (speedup {{dos .Speedup}}×), y es
**idéntico** al secuencial: {{if .Equivale}}verificado{{else}}NO verificado{{end}} por ` + "`cmd/perfil`" + `.
`

type limpieza struct {
	SHA256Silver string  `json:"sha256_silver"`
	Workers      int     `json:"workers"`
	Bloques      int     `json:"bloques"`
	SegLectura   float64 `json:"seg_lectura_disco"`
	SegSec       float64 `json:"seg_limpieza_secuencial"`
	SegConc      float64 `json:"seg_limpieza_concurrente"`
	Speedup      float64 `json:"speedup"`
	Equivale     bool    `json:"silver_concurrente_igual_al_secuencial"`
	L            struct {
		Entradas    int            `json:"filas_entrada"`
		Salidas     int            `json:"filas_salida"`
		Descartadas int            `json:"filas_descartadas"`
		PorRegla    map[string]int `json:"descartadas_por_regla"`
		Corregidas  int            `json:"ubigeo_rellenado_con_cero"`
		Reparadas   int            `json:"comillas_sin_escapar_reparadas"`
		Positivos   int            `json:"positivos"`
		Tasa        float64        `json:"tasa_positivos"`
		Identicas   int            `json:"filas_identicas_conservadas"`
		Orden       []string       `json:"orden_reglas"`
	} `json:"limpieza"`
	Bench []benchLoader
}

type benchLoader struct {
	Archivo string
	Maquina struct {
		CPU    string `json:"cpu"`
		NumCPU int    `json:"num_cpu"`
		SO     string `json:"so"`
	} `json:"maquina"`
	Mediciones []struct {
		Workers  int `json:"workers"`
		Segundos struct {
			Media float64 `json:"media_recortada"`
			Inf   float64 `json:"ic95_inf"`
			Sup   float64 `json:"ic95_sup"`
		} `json:"segundos"`
		Speedup    float64 `json:"speedup"`
		Eficiencia float64 `json:"eficiencia"`
		FilasPorS  float64 `json:"filas_por_segundo"`
		Equivale   bool    `json:"silver_igual_al_secuencial"`
	} `json:"mediciones"`
}

const plantillaLimpieza = `<!-- Generado por cmd/informe desde reports/limpieza.json y reports/bench_loader_*.json. No editar a mano. -->

# 3. Limpieza y carga concurrente (Parte A)

## 3.1 Procedimiento

El loader (` + "`cmd/loader`" + `) lee el CSV original completo en memoria, lo divide en
**{{.Bloques}} bloques** de ~1 MiB que siempre terminan en un salto de línea y los reparte a un
*worker pool*. Cada worker parsea, valida y normaliza las filas de su bloque y escribe el
resultado en un slot propio; al final los bloques se concatenan **en orden**. Por eso el CSV
limpio (capa silver) es **idéntico byte a byte** para cualquier número de workers
({{if .Equivale}}verificado con sha256{{else}}NO verificado{{end}}: ` + "`{{.SHA256Silver}}`" + `).

Cada fila se evalúa contra las reglas en este orden y se descarta por la **primera** que
incumple, así que cada fila descartada se cuenta una sola vez:

| # | Regla | Filas descartadas |
|---:|---|---:|
{{- range $i, $r := .L.Orden}}
| {{inc $i}} | {{$r}} | {{miles (index $.L.PorRegla $r)}} |
{{- end}}

Además de descartar, el loader **repara** sin perder filas:

- **Comillas sin escapar:** {{miles .L.Reparadas}} filas traen comillas dentro del campo
  ` + "`localidad`" + ` (p. ej. ` + "`\"ISLA SANTA ROSA \"AMAZONAS\"\"`" + `), que el estándar CSV no admite. Se
  recuperan con un segundo intento de parseo que trata esas comillas como texto.
- **Ubigeo de 5 dígitos:** {{miles .L.Corregidas}} filas (se rellena el cero inicial).
- **Edad:** se normaliza a años según ` + "`tipo_edad`" + ` (A = años, M = meses / 12, D = días / 365).

**Filas idénticas:** {{miles .L.Identicas}} filas repiten exactamente a otra. **No se eliminan**: el
dataset no tiene identificador de paciente, y dos casos reales pueden coincidir en todos los
campos (misma localidad, semana, edad y sexo). Eliminarlas borraría casos verdaderos.

## 3.2 Resultado

| | Filas |
|---|---:|
| Entrada (bronze) | {{miles .L.Entradas}} |
| Descartadas | {{miles .L.Descartadas}} |
| **Salida (silver)** | **{{miles .L.Salidas}}** |
| Positivos (con signos o grave) | {{miles .L.Positivos}} ({{pct .L.Tasa}} %) |

## 3.3 Rendimiento de la carga concurrente
{{range .Bench}}
**{{.Maquina.CPU}}** · {{.Maquina.NumCPU}} CPU lógicas · {{.Maquina.SO}} (` + "`{{.Archivo}}`" + `)

| Workers | Media recortada (ms) | IC 95 % (ms) | Speedup | Eficiencia | Filas/s | Silver = secuencial |
|---:|---:|---|---:|---:|---:|:---:|
{{- range .Mediciones}}
| {{.Workers}} | {{ms .Segundos.Media}} | [{{ms .Segundos.Inf}}, {{ms .Segundos.Sup}}] | {{dos .Speedup}}× | {{pct .Eficiencia}} % | {{miles (entero .FilasPorS)}} | {{if .Equivale}}✔{{else}}✘{{end}} |
{{- end}}
{{end}}
Protocolo: limpieza completa en memoria (sin lectura de disco), un calentamiento, repeticiones
en orden barajado, ` + "`runtime.GC()`" + ` antes de cada medición, media recortada al 10 % e intervalo
de confianza por bootstrap.
`

func main() {
	var p perfil
	leer("reports/perfil_bronze.json", &p)
	slices.Sort(p.Anos)
	escribir("docs/informe/pc3/02-datos.md", plantillaDatos, p)

	var l limpieza
	leer("reports/limpieza.json", &l)
	archivos, _ := filepath.Glob("reports/bench_loader_*.json")
	slices.Sort(archivos)
	for _, a := range archivos {
		var b benchLoader
		leer(a, &b)
		b.Archivo = filepath.Base(a)
		l.Bench = append(l.Bench, b)
	}
	escribir("docs/informe/pc3/03-limpieza.md", plantillaLimpieza, l)
}

var funciones = template.FuncMap{
	"miles": func(n int) string {
		s := fmt.Sprint(n)
		var b strings.Builder
		for i, c := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				b.WriteRune(' ') // espacio fino, separador de miles en español
			}
			b.WriteRune(c)
		}
		return b.String()
	},
	"inc":    func(i int) int { return i + 1 },
	"entero": func(x float64) int { return int(x + 0.5) },
	"ms":     func(x float64) string { return strings.Replace(fmt.Sprintf("%.1f", 1000*x), ".", ",", 1) },
	"mb":     func(b int) string { return strings.Replace(fmt.Sprintf("%.1f", float64(b)/1e6), ".", ",", 1) },
	"pct":    func(x float64) string { return strings.Replace(fmt.Sprintf("%.2f", 100*x), ".", ",", 1) },
	"seg":    func(x float64) string { return strings.Replace(fmt.Sprintf("%.3f", x), ".", ",", 1) },
	"dos":    func(x float64) string { return strings.Replace(fmt.Sprintf("%.2f", x), ".", ",", 1) },
	"tasaAno": func(c [3]int) string {
		t := c[0] + c[1] + c[2]
		if t == 0 {
			return "—"
		}
		return strings.Replace(fmt.Sprintf("%.1f", 100*float64(c[1]+c[2])/float64(t)), ".", ",", 1)
	},
	"top": func(cs []conteo) string {
		var partes []string
		for i, c := range cs {
			if i == 3 {
				break
			}
			v := c.Valor
			if v == "" {
				v = "(vacío)"
			}
			partes = append(partes, fmt.Sprintf("%s (%d)", v, c.N))
		}
		return strings.Join(partes, "; ")
	},
}

func leer(ruta string, v any) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		log.Fatalf("%s: %v", ruta, err)
	}
}

func escribir(ruta, plantilla string, datos any) {
	t := template.Must(template.New(ruta).Funcs(funciones).Parse(plantilla))
	var buf bytes.Buffer
	if err := t.Execute(&buf, datos); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(ruta, buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("escrito %s", ruta)
}
