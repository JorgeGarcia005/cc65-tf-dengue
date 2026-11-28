package main

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// celdaRiesgo es una celda del cubo departamento × semana × etapa de
// reports/evaluacion.json (internal/evaluacion.Celda). Semana 0 = todas las
// semanas; etapa "todas" = todas las edades; departamento "TODOS" = el país.
type celdaRiesgo struct {
	Departamento string  `json:"departamento"`
	Semana       int     `json:"semana"`
	Etapa        string  `json:"etapa"`
	Casos        int     `json:"casos"`
	ConSignos    int     `json:"con_signos_observados"`
	Tasa         float64 `json:"tasa_observada"`
	Riesgo       float64 `json:"riesgo_calibrado_medio"`
	Esperados    float64 `json:"casos_esperados"`
	Fraccion     float64 `json:"fraccion_nacional"`
	Razon        float64 `json:"razon_vs_nacional"`
	Insuficiente bool    `json:"insuficiente"`
}

// evaluacionTB2 es lo que el informe usa de reports/evaluacion.json.
type evaluacionTB2 struct {
	VersionModelo string `json:"version_modelo"`
	FilasTest     int    `json:"filas_test"`
	PositivosTest int    `json:"positivos_test"`
	Metricas      struct {
		Umbral                       float64
		VP, FP, VN, FN               int
		Exactitud, Precision, Recall float64
		F1, AUC                      float64
	} `json:"metricas_test"`
	Brier       float64 `json:"brier_calibrado"`
	Calibracion []struct {
		Casos  int     `json:"casos"`
		Riesgo float64 `json:"riesgo_calibrado_medio"`
		Tasa   float64 `json:"tasa_observada"`
		Inf    float64 `json:"ic95_inf"`
		Sup    float64 `json:"ic95_sup"`
	} `json:"calibracion"`
	Riesgo struct {
		MinCasos int           `json:"min_casos"`
		Celdas   []celdaRiesgo `json:"celdas"`
	} `json:"riesgo"`
	Prioridades []celdaRiesgo `json:"prioridades"`
}

var (
	evalUnaVez sync.Once
	evalDatos  evaluacionTB2
)

// Eval carga reports/evaluacion.json una vez por ejecución.
func (d datosTB2) Eval() evaluacionTB2 {
	evalUnaVez.Do(func() { leer("reports/evaluacion.json", &evalDatos) })
	return evalDatos
}

// filaImpacto es una fila de las tablas de la sección 18, con la fracción
// acumulada de los casos esperados del país.
type filaImpacto struct {
	celdaRiesgo
	Acumulada float64
}

func acumular(cs []celdaRiesgo) []filaImpacto {
	out := make([]filaImpacto, len(cs))
	var acum float64
	for i, c := range cs {
		acum += c.Fraccion
		out[i] = filaImpacto{c, acum}
	}
	return out
}

func porEsperados(a, b celdaRiesgo) int { return cmp.Compare(b.Esperados, a.Esperados) }

// ImpactoDeptos: los departamentos (todas las semanas y edades) ordenados
// por casos esperados con signos de alarma.
func (d datosTB2) ImpactoDeptos() []filaImpacto {
	if deptosCache != nil {
		return deptosCache
	}
	var cs []celdaRiesgo
	for _, c := range d.Eval().Riesgo.Celdas {
		if c.Semana == 0 && c.Etapa == "todas" && c.Departamento != "TODOS" {
			cs = append(cs, c)
		}
	}
	slices.SortStableFunc(cs, porEsperados)
	deptosCache = acumular(cs)
	return deptosCache
}

var deptosCache []filaImpacto

// DeptosHasta es cuántos de los primeros departamentos hacen falta para
// llegar a la fracción f de los casos esperados del país.
func (d datosTB2) DeptosHasta(f float64) int {
	for i, x := range d.ImpactoDeptos() {
		if x.Acumulada >= f {
			return i + 1
		}
	}
	return len(d.ImpactoDeptos())
}

// Nacional es la celda del país completo (todas las semanas y edades).
func (d datosTB2) Nacional() celdaRiesgo {
	for _, c := range d.Eval().Riesgo.Celdas {
		if c.Semana == 0 && c.Etapa == "todas" && c.Departamento == "TODOS" {
			return c
		}
	}
	return celdaRiesgo{}
}

// SemanasPais: el país por semana epidemiológica (todas las edades), en
// orden de semana.
func (d datosTB2) SemanasPais() []celdaRiesgo {
	var cs []celdaRiesgo
	for _, c := range d.Eval().Riesgo.Celdas {
		if c.Semana > 0 && c.Etapa == "todas" && c.Departamento == "TODOS" {
			cs = append(cs, c)
		}
	}
	slices.SortFunc(cs, func(a, b celdaRiesgo) int { return cmp.Compare(a.Semana, b.Semana) })
	return cs
}

// ventanaSemanas es el tramo más corto de semanas consecutivas que reúne al
// menos la fracción objetivo de los casos esperados del país.
type ventanaSemanas struct {
	Desde, Hasta, Semanas int
	Fraccion              float64
	Pico                  celdaRiesgo
}

// TemporadaAlta busca la ventana más corta con al menos la mitad de los casos
// esperados del año (sin dar la vuelta al año: la temporada del dengue en el
// Perú cae dentro del año calendario, y si no, la ventana sale más larga,
// no equivocada).
func (d datosTB2) TemporadaAlta() ventanaSemanas {
	s := d.SemanasPais()
	var total float64
	var v ventanaSemanas
	for _, c := range s {
		total += c.Esperados
		if c.Esperados > v.Pico.Esperados {
			v.Pico = c
		}
	}
	if total == 0 {
		return v
	}
	v.Semanas = len(s) + 1
	for i := range s {
		var suma float64
		for j := i; j < len(s); j++ {
			suma += s[j].Esperados
			if suma/total >= 0.5 {
				if j-i+1 < v.Semanas {
					v.Desde, v.Hasta, v.Semanas, v.Fraccion = s[i].Semana, s[j].Semana, j-i+1, suma/total
				}
				break
			}
		}
	}
	return v
}

// SemanasTop: las semanas del país con más casos esperados, de mayor a menor.
func (d datosTB2) SemanasTop(n int) []celdaRiesgo {
	s := slices.Clone(d.SemanasPais())
	slices.SortStableFunc(s, porEsperados)
	return s[:min(n, len(s))]
}

// EtapasPais: el país por etapa de vida (todas las semanas), en el orden de
// edad de las etapas del MINSA.
func (d datosTB2) EtapasPais() []celdaRiesgo {
	orden := []string{"nino_0_11", "adolescente_12_17", "joven_18_29", "adulto_30_59", "adulto_mayor_60_mas"}
	var cs []celdaRiesgo
	for _, e := range orden {
		for _, c := range d.Eval().Riesgo.Celdas {
			if c.Semana == 0 && c.Etapa == e && c.Departamento == "TODOS" {
				cs = append(cs, c)
			}
		}
	}
	return cs
}

// Prioridades: las primeras n combinaciones departamento × semana × etapa
// por casos esperados (ya vienen ordenadas en el reporte), con la acumulada.
func (d datosTB2) Prioridades(n int) []filaImpacto {
	p := d.Eval().Prioridades
	return acumular(p[:min(n, len(p))])
}

// nombreDepto pasa "SAN MARTIN" a "San Martín".
func nombreDepto(s string) string {
	return nombreGrupo(strings.ToLower(strings.ReplaceAll(s, " ", "_")))
}

// coberturaPlan resume el plan de la sección 18.5: los k departamentos con
// más casos esperados durante la temporada alta.
type coberturaPlan struct {
	Deptos         []string
	Desde, Hasta   int
	Casos          int     // casos de prueba dentro del plan
	Esperados      float64 // casos esperados con signos de alarma dentro del plan
	Fraccion       float64 // del total del país
	Anos           int     // años del periodo de entrenamiento y prueba
	EsperadosAnual float64 // casos con signos de alarma por año en el plan, en todo el país (no solo prueba)
	// Celdas: fracción de las combinaciones departamento × semana del país
	// que abarca el plan (k departamentos × semanas de la temporada alta).
	Celdas float64
}

// Cobertura calcula, con las celdas departamento × semana (todas las edades),
// qué parte de los casos esperados del país cae en los k primeros
// departamentos durante la temporada alta. EsperadosAnual lleva la cifra de
// prueba a todos los casos (÷ fracción de prueba) y a un año promedio
// (÷ años con datos desde el año mínimo del modelo).
func (d datosTB2) Cobertura(k int) coberturaPlan {
	t := d.TemporadaAlta()
	p := coberturaPlan{Desde: t.Desde, Hasta: t.Hasta}
	elegidos := map[string]bool{}
	for i, x := range d.ImpactoDeptos() {
		if i < k {
			elegidos[x.Departamento] = true
			p.Deptos = append(p.Deptos, nombreDepto(x.Departamento))
		}
	}
	for _, c := range d.Eval().Riesgo.Celdas {
		if c.Etapa == "todas" && elegidos[c.Departamento] && c.Semana >= t.Desde && c.Semana <= t.Hasta {
			p.Casos += c.Casos
			p.Esperados += c.Esperados
			p.Fraccion += c.Fraccion
		}
	}
	for a := range d.S.P.Perfil.PorAno {
		if n, err := strconv.Atoi(a); err == nil && n >= d.S.M.S.AnoMin {
			p.Anos++
		}
	}
	if n := len(d.ImpactoDeptos()) * len(d.SemanasPais()); n > 0 {
		p.Celdas = float64(len(p.Deptos)*(t.Hasta-t.Desde+1)) / float64(n)
	}
	if p.Anos > 0 && d.S.M.S.FracTest > 0 {
		p.EsperadosAnual = p.Esperados / d.S.M.S.FracTest / float64(p.Anos)
	}
	return p
}

// DeptosPorVolumen y DeptosPorRiesgo separan los diez primeros departamentos
// por casos esperados según por qué están arriba: muchos casos con riesgo por
// caso no mayor que el del país (razón ≤ 1), o riesgo por caso al menos un
// 25 % mayor que el del país (razón ≥ 1,25). Devuelven los nombres en orden.
func (d datosTB2) DeptosPorVolumen() string {
	return d.deptosSi(func(r float64) bool { return r <= 1 })
}
func (d datosTB2) DeptosPorRiesgo() string {
	return d.deptosSi(func(r float64) bool { return r >= 1.25 })
}

func (d datosTB2) deptosSi(cumple func(float64) bool) string {
	var ns []string
	for i, x := range d.ImpactoDeptos() {
		if i < 10 && cumple(x.Razon) {
			ns = append(ns, nombreDepto(x.Departamento))
		}
	}
	return enumerar(ns)
}

// enumerar une nombres en castellano: «A», «A y B», «A, B y C».
func enumerar(ns []string) string {
	switch len(ns) {
	case 0:
		return "ninguno"
	case 1:
		return ns[0]
	}
	return strings.Join(ns[:len(ns)-1], ", ") + " y " + ns[len(ns)-1]
}
