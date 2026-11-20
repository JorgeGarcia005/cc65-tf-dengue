package main

// Informe PC4: secciones compartidas con la PC3 (datos, limpieza, modelo y sus
// figuras), regiones generadas dentro de las secciones escritas a mano
// (regiones.go) y el resumen y las conclusiones, que llevan cifras.

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// PreguntaQueResuelve es la pregunta que resuelve el problema planteado
// (observación 1 del profesor en la PC3). Se repite con las mismas palabras
// en el problema, los objetivos, el resumen y las conclusiones.
const PreguntaQueResuelve = "¿Puede un sistema distribuido en Go, que entrena una regresión logística " +
	"sobre el millón de casos de dengue notificados por el MINSA, estimar en menos de 100 ms el riesgo de que " +
	"un caso sea con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y " +
	"grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico?"

type sintesisPC4 struct {
	S        sintesis
	Pregunta string
	Cluster  reporteCluster
	API      reporteAPI
	// Spin
	SpinEstados  int // estados del modelo N = 3 (seguridad)
	SpinMutantes int
	SpinOK       bool
}

func (s sintesisPC4) Ultimo() int { return len(s.Cluster.Mediciones) - 1 }

func generarPC4(s sintesis) {
	dir := "docs/informe/pc4"
	p4 := sintesisPC4{S: s, Pregunta: PreguntaQueResuelve}
	var err error
	if p4.Cluster, _, err = clusterGold(); err != nil {
		log.Fatal(err)
	}
	if p4.API, _, err = apiReal(); err != nil {
		log.Fatal(err)
	}
	var spin struct {
		OK       bool `json:"ok"`
		Corridas []struct {
			Corrida  string `json:"corrida"`
			Chequeo  string `json:"chequeo"`
			Estados  int    `json:"estados"`
			Esperado string `json:"esperado"`
		} `json:"corridas"`
	}
	if err := leerJSON("reports/spin.json", &spin); err != nil {
		log.Fatal(err)
	}
	p4.SpinOK = spin.OK
	for _, c := range spin.Corridas {
		if strings.HasPrefix(c.Corrida, "modelo N=3 (") && c.Chequeo == "seguridad" {
			p4.SpinEstados = c.Estados
		}
		if c.Esperado != "0" {
			p4.SpinMutantes++
		}
	}
	escribir(filepath.Join(dir, "01-resumen.md"), plantillaResumenPC4, p4)
	escribir(filepath.Join(dir, "13-conclusiones.md"), plantillaConclusionesPC4, p4)
	cambiados, err := regenerarRegiones(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range cambiados {
		log.Printf("regiones regeneradas en %s", c)
	}
}

// copiarSeccion copia una sección generada de la PC3 a la PC4 con otro número
// de sección (las secciones 5–7 conservan su número).
func copiarSeccion(desde, hasta string) {
	b, err := os.ReadFile(desde)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(hasta, b, 0o644); err != nil {
		log.Fatal(err)
	}
}

const plantillaResumenPC4 = `<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 1. Resumen

El presente trabajo aborda la anticipación de casos de **dengue con signos de alarma o grave** en el
Perú, los que requieren hospitalización, para orientar el refuerzo de camas, vigilancia y cerco
epidemiológico (ODS 3: Salud y bienestar), con los **{{miles .S.P.Perfil.Filas}} casos** notificados entre
{{.S.AnoMin}} y {{.S.AnoMax}} del conjunto de datos abierto de vigilancia epidemiológica del MINSA. La
pregunta que resuelve el problema es: *{{.Pregunta}}*

La PC3 construyó en Go la carga y limpieza concurrente (Parte A) y el entrenamiento paralelo de una
**regresión logística** con un gradiente **idéntico bit a bit** al secuencial (Parte B). Esta entrega
distribuye el entrenamiento en un **clúster de nodos** que se comunican por TCP con un coordinador
(*parameter server* síncrono en estrella), desplegado con Docker Compose; expone el modelo en una **API REST**
documentada con OpenAPI/Swagger, y guarda modelos, entrenamientos y predicciones en **MongoDB**, con
**Redis** como caché de predicciones.

Con los datos reales, el clúster reproduce bit a bit el entrenamiento secuencial también cuando cae un
nodo, y escala de {{ms (index .Cluster.Mediciones 0).SegPorEpoca.Media}} ms a
{{ms (index .Cluster.Mediciones .Ultimo).SegPorEpoca.Media}} ms por época con
{{(index .Cluster.Mediciones .Ultimo).Nodos}} nodos (speedup de {{dos (index .Cluster.Mediciones .Ultimo).Speedup}}×). La API
responde con un p95 de {{dos (index .API.Corridas 0).P95}} ms con la caché fría y de {{dos (index .API.Corridas 1).P95}} ms con
la caché caliente, muy por debajo de los 100 ms exigidos. El protocolo del clúster se verificó con Spin
({{miles .SpinEstados}} estados sin errores y {{.SpinMutantes}} mutantes que sí fallan). El modelo (AUC de
{{cuatro .S.M.E.Test.AUC}}) señala el mayor riesgo en {{deptos .S.TopDeptos}}, en el grupo
{{etapa .S.TopEtapa.Grupo}} y en las semanas epidemiológicas {{semanas (picosCasos .S.M.E.PorSemana)}}.
`

const plantillaConclusionesPC4 = `<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 13. Conclusiones

La pregunta que resuelve el problema era: *{{.Pregunta}}* La respuesta es **sí, con los límites del
dato disponible**:

- **El sistema distribuido entrena el mismo modelo que el secuencial.** Con un parcial por bloque y la
  reducción en orden global de bloque, el clúster da pesos idénticos bit a bit al entrenamiento secuencial
  con cualquier número de nodos y también cuando un nodo cae a mitad del entrenamiento y su shard se
  reasigna. La distribución no cambia la respuesta, solo el tiempo.
- **La distribución escala porque la época está dominada por el cómputo.** Con {{(index .Cluster.Mediciones .Ultimo).Nodos}}
  nodos de un núcleo, la época baja de {{ms (index .Cluster.Mediciones 0).SegPorEpoca.Media}} ms a
  {{ms (index .Cluster.Mediciones .Ultimo).SegPorEpoca.Media}} ms (speedup de {{dos (index .Cluster.Mediciones .Ultimo).Speedup}}×,
  eficiencia de {{pct (index .Cluster.Mediciones .Ultimo).Eficiencia}} %). La red y la espera en la barrera ocupan
  {{pct (index .Cluster.Mediciones .Ultimo).Desglose.Fraccion}} % de la época con {{(index .Cluster.Mediciones .Ultimo).Nodos}} nodos y los bytes por época
  casi no cambian con los nodos: la fracción serial de Karp-Flatt se mantiene baja y no crece.
- **La API cumple el requisito de latencia con margen.** Con {{.API.Concurrencia}} clientes concurrentes contra el
  clúster en Docker, el p95 fue de {{dos (index .API.Corridas 0).P95}} ms sin caché y de {{dos (index .API.Corridas 1).P95}} ms
  desde Redis, sin errores: el cálculo de una predicción es un producto punto de pocas decenas de
  variables, y la caché recorta el resto.
- **El modelo responde dónde, cuándo y en quién reforzar, con límites.** Con un AUC de {{cuatro .S.M.E.Test.AUC}},
  solo con edad, sexo, lugar y semana, el modelo ordena el riesgo mejor que el azar pero no lo determina.
  Aun así identifica dónde reforzar: por departamento, {{deptos .S.TopDeptos}}; por edad, el grupo
  {{etapa .S.TopEtapa.Grupo}}, con {{pct .S.TopEtapa.Calib}} % de riesgo calibrado; y por semana, las camas hacen más
  falta en las semanas {{semanas (picosCasos .S.M.E.PorSemana)}}, cuando se concentran los casos graves.
- **Verificar el protocolo antes y después de implementarlo evitó errores reales.** Spin recorrió
  {{miles .SpinEstados}} estados del modelo sin errores, y los {{.SpinMutantes}} mutantes (envío bloqueante, reasignación en
  una pasada, plazo global y primer nodo sano) dan contraejemplo: el modelo distingue cada decisión del
  diseño del error que corrige. La revisión entre agentes encontró además fallas de cierre y de medición
  que los tests no cubrían, y quedaron corregidas con un test cada una.
`
