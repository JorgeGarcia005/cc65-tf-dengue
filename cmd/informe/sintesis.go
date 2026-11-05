package main

// Resumen y conclusiones del informe: llevan cifras, así que también se
// generan desde los reportes JSON en vez de escribirse a mano.

import (
	"fmt"
	"slices"
	"strings"
)

type medidaGrad struct {
	Workers    int
	Speedup    float64
	Eficiencia float64
}

type sintesis struct {
	P perfil
	L limpieza
	M modelo

	AnoPico   string // año con más casos notificados
	CasosPico int
	AnoMin    string
	AnoMax    string

	// Gradiente sobre el gold real en la máquina con más CPU.
	GradMaq      string
	GradMin      float64 // menor de los mejores speedups entre sistemas operativos
	GradMax      float64
	GradWorkers  int // workers del mejor speedup
	GradEf8      medidaGrad
	GradEf16     medidaGrad
	FilasEq      int
	CIGrad       float64 // mejor speedup en el CI (2 vCPU)
	LoaderMax    float64
	LoaderMin    float64
	TodoBitABit  bool
	TopDeptos    []grupoRiesgo
	TopEtapa     grupoRiesgo
	SegEpocaReal float64
}

func calcularSintesis(p perfil, l limpieza, m modelo) sintesis {
	s := sintesis{P: p, L: l, M: m, TodoBitABit: m.E.Equivale}
	anos := slices.Clone(p.Anos)
	slices.Sort(anos)
	s.AnoMin, s.AnoMax = anos[0], anos[len(anos)-1]
	for _, a := range anos {
		c := p.Perfil.PorAno[a]
		if t := c[0] + c[1] + c[2]; t > s.CasosPico {
			s.AnoPico, s.CasosPico = a, t
		}
	}

	maxCPU := 0
	for _, b := range m.Bench {
		maxCPU = max(maxCPU, b.Maquina.NumCPU)
	}
	s.GradMin = -1
	for _, b := range m.Bench {
		for _, x := range b.Mediciones {
			s.TodoBitABit = s.TodoBitABit && x.Bits
		}
		mejor := mejorGrad(b)
		if b.Maquina.NumCPU < maxCPU {
			s.CIGrad = max(s.CIGrad, mejor.Speedup)
			continue
		}
		if !strings.HasPrefix(b.Fuente, "gold") {
			continue
		}
		s.GradMaq = cpuCorta(b.Maquina.CPU) + " (" + fmt.Sprint(b.Maquina.NumCPU) + " hilos)"
		if mejor.Speedup > s.GradMax {
			s.GradMax, s.GradWorkers = mejor.Speedup, mejor.Workers
			s.GradEf8, s.GradEf16 = medidaEn(b, 8), medidaEn(b, 16)
			if b.FilasEq != nil {
				s.FilasEq = *b.FilasEq
			}
		}
		if s.GradMin < 0 || mejor.Speedup < s.GradMin {
			s.GradMin = mejor.Speedup
		}
	}

	s.LoaderMin = -1
	for _, b := range l.Bench {
		if b.Maquina.NumCPU < maxCPU {
			continue
		}
		mejor := 0.0
		for _, x := range b.Mediciones {
			mejor = max(mejor, x.Speedup)
		}
		s.LoaderMax = max(s.LoaderMax, mejor)
		if s.LoaderMin < 0 || mejor < s.LoaderMin {
			s.LoaderMin = mejor
		}
	}

	s.TopDeptos = m.E.PorDepto[:min(3, len(m.E.PorDepto))]
	for _, g := range m.E.PorEtapa {
		if g.Calib > s.TopEtapa.Calib {
			s.TopEtapa = g
		}
	}
	return s
}

func mejorGrad(b benchGrad) medidaGrad {
	var m medidaGrad
	for _, x := range b.Mediciones {
		if x.Speedup > m.Speedup {
			m = medidaGrad{x.Workers, x.Speedup, x.Eficiencia}
		}
	}
	return m
}

func medidaEn(b benchGrad, workers int) medidaGrad {
	for _, x := range b.Mediciones {
		if x.Workers == workers && x.Config != "secuencial" {
			return medidaGrad{x.Workers, x.Speedup, x.Eficiencia}
		}
	}
	return medidaGrad{}
}

const plantillaResumen = `<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 1. Resumen

El presente trabajo aborda la anticipación de casos de **dengue con signos de alarma o grave** en el
Perú, los que requieren hospitalización, para orientar el refuerzo de camas, vigilancia y cerco
epidemiológico (ODS 3: Salud y bienestar). Se usan los **{{miles .P.Perfil.Filas}} casos** notificados
entre {{.AnoMin}} y {{.AnoMax}} del conjunto de datos abierto de vigilancia epidemiológica del MINSA.

En esta entrega se construyeron en Go las dos primeras partes del sistema. La **Parte A** carga, limpia
y valida el archivo completo con un *worker pool* que reparte bloques de bytes; el resultado es idéntico
byte a byte para cualquier número de workers y conserva {{miles .L.L.Salidas}} filas válidas. La
**Parte B** entrena una **regresión logística** con descenso de gradiente cuyo cálculo se reparte en
bloques de tamaño fijo entre goroutines, con acumuladores privados y una reducción en orden de bloque,
de modo que el gradiente concurrente es **idéntico bit a bit** al secuencial.

En un {{.GradMaq}}, el cálculo del gradiente sobre el conjunto de entrenamiento real alcanza un speedup
de **{{dos .GradMin}}× a {{dos .GradMax}}×** según el sistema operativo, y la limpieza de
{{dos .LoaderMin}}× a {{dos .LoaderMax}}×. El modelo logra un AUC de {{cuatro .M.E.Test.AUC}} en prueba
y señala el mayor riesgo en {{deptos .TopDeptos}}. Por último, se diseñó el clúster de nodos que se
implementará en PC4 (parameter server síncrono por TCP) y se verificó su protocolo con Spin.
`

const plantillaConclusiones = `<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 10. Conclusiones

- **La paralelización del gradiente escala con los núcleos físicos.** En el {{.GradMaq}}, el speedup
  sobre los datos reales llega a {{dos .GradMax}}× con {{.GradWorkers}} workers. La eficiencia baja de
  {{pct .GradEf8.Eficiencia}} % con 8 workers a {{pct .GradEf16.Eficiencia}} % con 16, porque a partir de 8
  los workers comparten núcleos físicos (SMT). En el CI con 2 vCPU el mejor speedup fue
  {{dos .CIGrad}}×: el mismo código aprovecha el hardware que encuentra.
- **El determinismo no se negocia por velocidad.** Con bloques de límites fijos y una reducción en orden de
  bloque, todas las configuraciones medidas dieron un gradiente {{if .TodoBitABit}}idéntico bit a bit{{else}}DISTINTO{{end}} al
  secuencial. Esa misma propiedad permite repartir los bloques entre nodos en PC4 sin cambiar el
  resultado.
- **La concurrencia tiene un costo fijo.** Por debajo de unas {{miles .FilasEq}} filas, repartir el trabajo
  cuesta más de lo que se gana: el diseño solo conviene a partir de varios bloques por worker. La limpieza
  escala menos ({{dos .LoaderMax}}×) porque está limitada por la memoria y el análisis de texto, no por el
  cálculo.
- **Medir bien es parte del resultado.** En Windows el reloj de Go avanza a saltos de fracciones de
  milisegundo y las pasadas cortas se medían como cero; agrupar pasadas y repetir la medición en Linux
  (Debian) dio cifras coherentes en ambos sistemas.
- **El modelo responde la pregunta de impacto, con límites.** Con un AUC de {{cuatro .M.E.Test.AUC}}, solo
  con edad, sexo, lugar y semana, el modelo ordena el riesgo mejor que el azar pero no lo determina. Aun
  así identifica dónde reforzar: por departamento, {{deptos .TopDeptos}}; por edad, el grupo
  {{etapa .TopEtapa.Grupo}}, con {{pct .TopEtapa.Calib}} % de riesgo calibrado.
- **Verificar antes de implementar evitó dos errores.** El modelo Promela del protocolo del clúster encontró
  un bloqueo del coordinador al enviar a un nodo caído y un ciclo de *timeouts* al reasignar shards,
  ambos corregidos en el diseño antes de escribir el código de PC4.
`

func init() {
	funciones["deptos"] = func(gs []grupoRiesgo) string {
		var partes []string
		for _, g := range gs {
			partes = append(partes, nombreGrupo(g.Grupo)+" ("+funciones["pct"].(func(float64) string)(g.Calib)+" %)")
		}
		switch len(partes) {
		case 0:
			return ""
		case 1:
			return partes[0]
		}
		return strings.Join(partes[:len(partes)-1], ", ") + " y " + partes[len(partes)-1]
	}
	funciones["etapa"] = func(g string) string { return nombreEtapa(g) }
	// num escribe un número suelto (λ, tasa, umbral) con coma decimal.
	funciones["num"] = func(x float64) string { return strings.Replace(fmt.Sprintf("%g", x), ".", ",", 1) }
}

// conTilde corrige los nombres de departamento que el dataset trae sin tilde.
var conTilde = map[string]string{
	"junin": "Junín", "huanuco": "Huánuco", "san_martin": "San Martín",
	"apurimac": "Apurímac", "ancash": "Áncash", "otros": "otros departamentos",
}

// nombreGrupo pasa "madre_de_dios" a "Madre de Dios".
func nombreGrupo(g string) string {
	if n, ok := conTilde[g]; ok {
		return n
	}
	palabras := strings.Split(g, "_")
	for i, w := range palabras {
		if i > 0 && (w == "de" || w == "del" || w == "la") {
			continue
		}
		if w != "" {
			palabras[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(palabras, " ")
}

// nombreEtapa pasa "nino_0_11" a "niño (0–11 años)".
func nombreEtapa(g string) string {
	nombres := map[string]string{
		"nino":         "niño",
		"adolescente":  "adolescente",
		"joven":        "joven",
		"adulto":       "adulto",
		"adulto_mayor": "adulto mayor",
	}
	partes := strings.Split(g, "_")
	var rango []string
	var base []string
	for _, p := range partes {
		if p != "" && (p[0] >= '0' && p[0] <= '9') {
			rango = append(rango, p)
		} else if p != "mas" {
			base = append(base, p)
		}
	}
	n := nombres[strings.Join(base, "_")]
	if n == "" {
		n = strings.Join(base, " ")
	}
	switch {
	case len(rango) == 2:
		return n + " (" + rango[0] + "–" + rango[1] + " años)"
	case len(rango) == 1:
		return n + " (" + rango[0] + " años a más)"
	}
	return n
}
