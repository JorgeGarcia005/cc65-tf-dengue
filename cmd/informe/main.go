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

func main() {
	var p perfil
	leer("reports/perfil_bronze.json", &p)
	slices.Sort(p.Anos)
	escribir("docs/informe/pc3/02-datos.md", plantillaDatos, p)
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
	"mb":  func(b int) string { return strings.Replace(fmt.Sprintf("%.1f", float64(b)/1e6), ".", ",", 1) },
	"pct": func(x float64) string { return strings.Replace(fmt.Sprintf("%.2f", 100*x), ".", ",", 1) },
	"seg": func(x float64) string { return strings.Replace(fmt.Sprintf("%.3f", x), ".", ",", 1) },
	"dos": func(x float64) string { return strings.Replace(fmt.Sprintf("%.2f", x), ".", ",", 1) },
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
