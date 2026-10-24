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

type split struct {
	FilasSilver    int     `json:"filas_silver"`
	AnoMin         int     `json:"ano_min"`
	FilasExcluidas int     `json:"filas_excluidas_por_ano"`
	FracTest       float64 `json:"fraccion_test"`
	Train          struct {
		Filas     int     `json:"filas"`
		Positivos int     `json:"positivos"`
		Tasa      float64 `json:"tasa_positivos"`
	} `json:"train"`
	Test struct {
		Filas     int     `json:"filas"`
		Positivos int     `json:"positivos"`
		Tasa      float64 `json:"tasa_positivos"`
	} `json:"test"`
	D         int              `json:"variables"`
	TamBloque int              `json:"tam_bloque"`
	Bloques   int              `json:"bloques_train"`
	Shards    []map[string]int `json:"shards"`
}

type grupoRiesgo struct {
	Grupo string  `json:"grupo"`
	Casos int     `json:"casos_test"`
	Obs   float64 `json:"tasa_observada"`
	Calib float64 `json:"riesgo_calibrado_medio"`
}

type entrenamiento struct {
	Workers  int       `json:"workers"`
	Epocas   int       `json:"epocas"`
	Tasa     float64   `json:"tasa_aprendizaje"`
	Lambda   float64   `json:"lambda"`
	PesoPos  float64   `json:"peso_positivos"`
	PesoNeg  float64   `json:"peso_negativos"`
	Equivale bool      `json:"gradiente_pool_igual_al_secuencial"`
	SegTotal float64   `json:"seg_entrenamiento"`
	SegEpoca float64   `json:"seg_por_epoca"`
	Historia []float64 `json:"perdida_por_epoca"`
	Test     struct {
		Umbral                float64
		VP, FP, VN, FN        int
		Exactitud             float64
		Precision, Recall, F1 float64
		AUC                   float64
	} `json:"metricas_test"`
	Coef []struct {
		Variable  string  `json:"variable"`
		Peso      float64 `json:"peso"`
		RazonOdds float64 `json:"razon_de_odds"`
	} `json:"coeficientes_por_magnitud"`
	PorDepto []grupoRiesgo `json:"riesgo_por_departamento"`
	PorEtapa []grupoRiesgo `json:"riesgo_por_etapa_de_vida"`
}

type benchGrad struct {
	Archivo string
	Maquina struct {
		CPU    string `json:"cpu"`
		NumCPU int    `json:"num_cpu"`
	} `json:"maquina"`
	Fuente     string `json:"fuente"`
	Mediciones []struct {
		Config   string `json:"config"`
		Workers  int    `json:"workers"`
		Segundos struct {
			Media float64 `json:"media_recortada"`
			Inf   float64 `json:"ic95_inf"`
			Sup   float64 `json:"ic95_sup"`
		} `json:"segundos"`
		Speedup    float64 `json:"speedup"`
		Eficiencia float64 `json:"eficiencia"`
		Bits       bool    `json:"equivalente_bit_a_bit"`
	} `json:"mediciones"`
	Equilibrio []struct {
		Filas   int     `json:"filas"`
		Bloques int     `json:"bloques"`
		Sec     float64 `json:"seg_secuencial"`
		Conc    float64 `json:"seg_concurrente"`
		Speedup float64 `json:"speedup"`
	} `json:"punto_equilibrio"`
	FilasEq   *int `json:"filas_equilibrio"`
	WorkersEq int  `json:"workers_equilibrio"`
}

type modelo struct {
	S     split
	E     entrenamiento
	Bench []benchGrad
}

const plantillaModelo = `<!-- Generado por cmd/informe desde reports/split.json, reports/entrenamiento.json y reports/bench_gradiente_*.json. No editar a mano. -->

# 4. Modelo de ML distribuido y paralelización del cálculo (Parte B)

## 4.1 Datos de entrenamiento

- Se usan los casos desde **{{.S.AnoMin}}** (la categoría *con signos de alarma* se aplica de forma
  sistemática desde ese año, ver §2.3). Quedan fuera {{miles .S.FilasExcluidas}} de las
  {{miles .S.FilasSilver}} filas del silver.
- **Split determinista** {{pct .S.FracTest}} % prueba: cada fila va a prueba según un hash (splitmix64)
  de su posición en el silver, así que el split no depende del número de workers ni de nodos.

| Conjunto | Filas | Positivos | % positivos |
|---|---:|---:|---:|
| Entrenamiento | {{miles .S.Train.Filas}} | {{miles .S.Train.Positivos}} | {{pct .S.Train.Tasa}} |
| Prueba | {{miles .S.Test.Filas}} | {{miles .S.Test.Positivos}} | {{pct .S.Test.Tasa}} |

**{{.S.D}} variables**, todas conocidas al notificar el caso: edad estandarizada y etapa de vida MINSA
(5), sexo, estacionalidad de la semana epidemiológica (seno y coseno), tendencia anual y departamento
(one-hot; los de menos de 100 casos en entrenamiento se agrupan en OTROS). Se excluye ` + "`tipo_dx`" + `
(confirmado/probable) porque los casos graves se confirman en laboratorio con más frecuencia: sería
una fuga de información del objetivo (ADR-0004). La media y desviación de edad y año se calculan
**solo con el entrenamiento**.

## 4.2 Modelo: regresión logística con pesos por clase

$$L(w) = \frac{1}{S}\sum_i s_i\,\big[\log(1+e^{z_i}) - y_i z_i\big] + \frac{\lambda}{2}\lVert w\rVert^2,\qquad z_i = x_i^\top w + b$$

$$\nabla_w L = \frac{1}{S}\sum_i s_i\,(\sigma(z_i) - y_i)\,x_i + \lambda w,\qquad S = \sum_i s_i$$

Los pesos por clase ($s_i$ = {{tres .E.PesoPos}} para positivos y {{tres .E.PesoNeg}} para negativos)
igualan el aporte de ambas clases ante el desbalance. λ = {{.E.Lambda}}.

**Por qué se puede distribuir:** el gradiente es una **suma sobre filas**. Si las filas se parten en
bloques $B_1,\dots,B_K$, cada bloque aporta su suma parcial $g_k$ y el gradiente total es
$\frac{1}{S}\sum_k g_k + \lambda w$. Un worker, un hilo o un nodo puede calcular cualquier $g_k$;
solo hace falta **sumar los parciales siempre en el mismo orden de bloque** para que el resultado sea
idéntico bit a bit (la suma en punto flotante no es asociativa).

## 4.3 Paralelización en Go dentro de un nodo

| Patrón | Dónde | Para qué |
|---|---|---|
| *Worker pool* persistente | ` + "`ml.Pool`" + ` | Los workers viven todo el entrenamiento; no se crean goroutines por época |
| *Fan-out* por canal | ` + "`tareas chan int`" + ` | Reparte índices de bloque; cada bloque lo procesa un solo worker |
| Acumuladores privados | búfer local + slot ` + "`parciales[k]`" + ` | Nadie escribe donde otro escribe: sin mutex y sin *false sharing* |
| Barrera | ` + "`sync.WaitGroup`" + ` | Espera los {{.S.Bloques}} bloques de la época antes de reducir |
| *Fan-in* ordenado | ` + "`ml.ReducirEnOrden`" + ` | Un solo goroutine suma en orden de bloque → determinismo |

El gradiente del pool es **{{if .E.Equivale}}idéntico{{else}}DISTINTO{{end}} bit a bit** al secuencial
sobre los datos reales (verificado por ` + "`cmd/entrenar`" + `); ` + "`TestEquivalencia`" + ` lo exige con 1 a 64 workers.

## 4.4 Distribución en el clúster (diseño para PC4)

El entrenamiento de {{miles .S.Train.Filas}} filas se reparte en **{{len .S.Shards}} shards** de bloques
contiguos ({{.S.TamBloque}} filas por bloque, {{.S.Bloques}} bloques):

| Shard | Bloque inicial | Bloques | Filas |
|---:|---:|---:|---:|
{{- range .S.Shards}}
| {{index . "shard"}} | {{index . "bloque_inicial"}} | {{index . "bloques"}} | {{miles (index . "filas")}} |
{{- end}}

En cada época el coordinador (parameter server) envía los pesos a los nodos por TCP; cada nodo calcula
con su worker pool los parciales de sus bloques y los devuelve; el coordinador los concatena **en orden
de shard** (que es el orden global de bloque), aplica ` + "`Finalizar`" + ` y actualiza los pesos. El test
` + "`TestShardsReproducenElGradienteGlobal`" + ` comprueba que con 1, 2, 3, 5 u 8 shards el gradiente es
idéntico bit a bit al secuencial. El protocolo TCP se define en #9 (ADR pendiente).

## 4.5 Resultados del entrenamiento

Descenso de gradiente *full-batch*: {{.E.Epocas}} épocas, tasa {{.E.Tasa}}, {{.E.Workers}} workers,
{{dos .E.SegTotal}} s ({{ms .E.SegEpoca}} ms por época). Pérdida: {{cuatro (index .E.Historia 0)}} →
{{cuatro (ultimo .E.Historia)}}.

| Métrica (prueba, umbral {{.E.Test.Umbral}}) | Valor |
|---|---:|
| AUC-ROC | {{cuatro .E.Test.AUC}} |
| Recall (sensibilidad) | {{tres .E.Test.Recall}} |
| Precisión | {{tres .E.Test.Precision}} |
| F1 | {{tres .E.Test.F1}} |
| Exactitud | {{tres .E.Test.Exactitud}} |
| VP / FN / FP / VN | {{miles .E.Test.VP}} / {{miles .E.Test.FN}} / {{miles .E.Test.FP}} / {{miles .E.Test.VN}} |

El poder predictivo es **moderado**: sin comorbilidades ni datos clínicos, solo con edad, sexo, lugar y
momento, el modelo ordena el riesgo mejor que el azar pero no lo determina. Es una limitación de los
datos abiertos disponibles, declarada y aceptada (ADR-0003).

**Variables más influyentes** (razón de odds = $e^{w}$; > 1 aumenta el riesgo):

| Variable | Peso | Razón de odds |
|---|---:|---:|
{{- range $i, $c := .E.Coef}}{{if lt $i 10}}
| ` + "`{{$c.Variable}}`" + ` | {{signo $c.Peso}} | {{dos $c.RazonOdds}} |
{{- end}}{{end}}

## 4.6 Primera respuesta a la pregunta de impacto social

Riesgo de dengue con signos de alarma o grave en el conjunto de prueba. El riesgo calibrado corrige el
efecto de los pesos por clase (corrección de prior, King y Zeng, 2001) para que esté en la escala real.

| Departamento | Casos (prueba) | Tasa observada | Riesgo calibrado |
|---|---:|---:|---:|
{{- range $i, $g := .E.PorDepto}}{{if lt $i 8}}
| {{$g.Grupo}} | {{miles $g.Casos}} | {{pct $g.Obs}} % | {{pct $g.Calib}} % |
{{- end}}{{end}}

| Etapa de vida | Casos (prueba) | Tasa observada | Riesgo calibrado |
|---|---:|---:|---:|
{{- range .E.PorEtapa}}
| {{.Grupo}} | {{miles .Casos}} | {{pct .Obs}} % | {{pct .Calib}} % |
{{- end}}

## 4.7 Speedup del cálculo del gradiente
{{range .Bench}}
**{{.Maquina.CPU}}** · {{.Maquina.NumCPU}} CPU lógicas (` + "`{{.Archivo}}`" + `; {{.Fuente}})

| Configuración | Workers | Media recortada (ms) | IC 95 % (ms) | Speedup | Eficiencia | Bit a bit |
|---|---:|---:|---|---:|---:|:---:|
{{- range .Mediciones}}
| {{.Config}} | {{.Workers}} | {{ms .Segundos.Media}} | [{{ms .Segundos.Inf}}, {{ms .Segundos.Sup}}] | {{dos .Speedup}}× | {{pct .Eficiencia}} % | {{if .Bits}}✔{{else}}✘{{end}} |
{{- end}}

Punto de equilibrio con {{.WorkersEq}} workers{{if .FilasEq}}: la concurrencia conviene desde **{{miles (deref .FilasEq)}} filas**{{else}}: no se alcanzó en los tamaños medidos{{end}}.

| Filas | Bloques | Secuencial (ms) | Concurrente (ms) | Speedup |
|---:|---:|---:|---:|---:|
{{- range .Equilibrio}}
| {{miles .Filas}} | {{.Bloques}} | {{ms .Sec}} | {{ms .Conc}} | {{dos .Speedup}}× |
{{- end}}
{{end}}
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

	var m modelo
	leer("reports/split.json", &m.S)
	leer("reports/entrenamiento.json", &m.E)
	archivos, _ = filepath.Glob("reports/bench_gradiente_*.json")
	slices.Sort(archivos)
	for _, a := range archivos {
		var b benchGrad
		leer(a, &b)
		b.Archivo = filepath.Base(a)
		m.Bench = append(m.Bench, b)
	}
	escribir("docs/informe/pc3/04-modelo.md", plantillaModelo, m)
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
	"tres":   func(x float64) string { return strings.Replace(fmt.Sprintf("%.3f", x), ".", ",", 1) },
	"cuatro": func(x float64) string { return strings.Replace(fmt.Sprintf("%.4f", x), ".", ",", 1) },
	"signo":  func(x float64) string { return strings.Replace(fmt.Sprintf("%+.3f", x), ".", ",", 1) },
	"ultimo": func(xs []float64) float64 { return xs[len(xs)-1] },
	"deref":  func(p *int) int { return *p },
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
