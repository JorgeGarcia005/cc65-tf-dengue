package ingest

import (
	"bytes"
	"hash/maphash"
	"strconv"
)

// Regla identifica por qué se descartó una fila. Una fila se descarta por la
// primera regla que incumple, en este orden, así que cada fila cuenta una sola vez.
type Regla int

// Reglas de limpieza, en orden de aplicación.
const (
	OK Regla = iota
	MalFormada
	DiagnosticoDesconocido
	EnfermedadInconsistente
	AnoFueraDeRango
	SemanaFueraDeRango
	EdadInvalida
	EdadFueraDeRango
	SexoInvalido
	TipoDxInvalido
	UbigeoInvalido
	DepartamentoVacio
	numReglas
)

// NombresReglas describe cada regla para el reporte y el informe.
var NombresReglas = [numReglas]string{
	OK:                      "ok",
	MalFormada:              "mal_formada (comillas o columnas ≠ 15)",
	DiagnosticoDesconocido:  "diagnostico fuera de A97.0/A97.1/A97.2",
	EnfermedadInconsistente: "enfermedad no coincide con el código CIE-10",
	AnoFueraDeRango:         "año fuera de 2000–2025",
	SemanaFueraDeRango:      "semana epidemiológica fuera de 1–53",
	EdadInvalida:            "edad no numérica, negativa o tipo_edad ∉ {A,M,D}",
	EdadFueraDeRango:        "edad mayor a 110 años",
	SexoInvalido:            "sexo ∉ {F,M}",
	TipoDxInvalido:          "tipo_dx ∉ {C,P}",
	UbigeoInvalido:          "ubigeo no numérico o de largo ≠ 5/6",
	DepartamentoVacio:       "departamento vacío",
}

// Límites de las reglas.
const (
	AnoMin  = 2000
	AnoMax  = 2025
	EdadMax = 110
)

// Clases según CIE-10: 0 = A97.0 sin signos de alarma, 1 = A97.1 con signos de
// alarma, 2 = A97.2 grave. El objetivo binario del modelo es Clase ≥ 1.
var (
	codigos      = [][]byte{[]byte("A97.0"), []byte("A97.1"), []byte("A97.2")}
	enfermedades = [][]byte{
		[]byte("DENGUE SIN SIGNOS DE ALARMA"),
		[]byte("DENGUE CON SIGNOS DE ALARMA"),
		[]byte("DENGUE GRAVE"),
	}
)

// EncabezadoSilver son las columnas del CSV limpio.
const EncabezadoSilver = "departamento,provincia,distrito,ubigeo,ano,semana,edad_anios,sexo,tipo_dx,clase\n"

// Registro es una fila limpia y normalizada (capa silver).
type Registro struct {
	Departamento, Provincia, Distrito string
	Ubigeo                            string // siempre 6 dígitos
	Ano, Semana                       int
	EdadAnios                         float64
	Sexo                              byte // 'F' o 'M'
	TipoDx                            byte // 'C' confirmado o 'P' probable
	Clase                             int  // 0, 1 o 2
}

// Objetivo es 1 si el caso es con signos de alarma o grave.
func (r Registro) Objetivo() float64 {
	if r.Clase >= 1 {
		return 1
	}
	return 0
}

// LimpiarCampos valida y normaliza una fila ya separada en campos. corregido
// indica si se rellenó un ubigeo de 5 dígitos (la fila se conserva).
func LimpiarCampos(c [][]byte) (r Registro, regla Regla, corregido bool) {
	clase := -1
	for i, cod := range codigos {
		if bytes.Equal(c[ColDiagnostico], cod) {
			clase = i
		}
	}
	if clase < 0 {
		return r, DiagnosticoDesconocido, false
	}
	if !bytes.Equal(bytes.TrimSpace(c[ColEnfermedad]), enfermedades[clase]) {
		return r, EnfermedadInconsistente, false
	}
	ano, err := strconv.Atoi(string(c[ColAno]))
	if err != nil || ano < AnoMin || ano > AnoMax {
		return r, AnoFueraDeRango, false
	}
	semana, err := strconv.Atoi(string(c[ColSemana]))
	if err != nil || semana < 1 || semana > 53 {
		return r, SemanaFueraDeRango, false
	}
	edad, ok := EdadEnAnios(c[ColEdad], c[ColTipoEdad])
	if !ok {
		return r, EdadInvalida, false
	}
	if edad > EdadMax {
		return r, EdadFueraDeRango, false
	}
	if len(c[ColSexo]) != 1 || (c[ColSexo][0] != 'F' && c[ColSexo][0] != 'M') {
		return r, SexoInvalido, false
	}
	if len(c[ColTipoDx]) != 1 || (c[ColTipoDx][0] != 'C' && c[ColTipoDx][0] != 'P') {
		return r, TipoDxInvalido, false
	}
	ubigeo := c[ColUbigeo]
	if (len(ubigeo) != 5 && len(ubigeo) != 6) || !soloDigitos(ubigeo) {
		return r, UbigeoInvalido, false
	}
	if len(ubigeo) == 5 { // se perdió el cero inicial (p. ej. Excel)
		ubigeo = append([]byte{'0'}, ubigeo...)
		corregido = true
	}
	depto := bytes.TrimSpace(c[ColDepartamento])
	if len(depto) == 0 {
		return r, DepartamentoVacio, false
	}
	return Registro{
		Departamento: string(depto),
		Provincia:    string(bytes.TrimSpace(c[ColProvincia])),
		Distrito:     string(bytes.TrimSpace(c[ColDistrito])),
		Ubigeo:       string(ubigeo),
		Ano:          ano,
		Semana:       semana,
		EdadAnios:    edad,
		Sexo:         c[ColSexo][0],
		TipoDx:       c[ColTipoDx][0],
		Clase:        clase,
	}, OK, corregido
}

func soloDigitos(b []byte) bool {
	for _, x := range b {
		if x < '0' || x > '9' {
			return false
		}
	}
	return true
}

// AgregarSilver escribe el registro como línea del CSV silver.
func AgregarSilver(dst []byte, r Registro) []byte {
	dst = agregarCampo(dst, r.Departamento)
	dst = append(dst, ',')
	dst = agregarCampo(dst, r.Provincia)
	dst = append(dst, ',')
	dst = agregarCampo(dst, r.Distrito)
	dst = append(dst, ',')
	dst = append(dst, r.Ubigeo...)
	dst = append(dst, ',')
	dst = strconv.AppendInt(dst, int64(r.Ano), 10)
	dst = append(dst, ',')
	dst = strconv.AppendInt(dst, int64(r.Semana), 10)
	dst = append(dst, ',')
	dst = strconv.AppendFloat(dst, r.EdadAnios, 'g', -1, 64)
	dst = append(dst, ',', r.Sexo, ',', r.TipoDx, ',')
	dst = strconv.AppendInt(dst, int64(r.Clase), 10)
	return append(dst, '\n')
}

// agregarCampo pone comillas solo si el texto tiene coma, comillas o espacios.
func agregarCampo(dst []byte, s string) []byte {
	if !bytesNecesitaComillas(s) {
		return append(dst, s...)
	}
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			dst = append(dst, '"')
		}
		dst = append(dst, s[i])
	}
	return append(dst, '"')
}

func bytesNecesitaComillas(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ',', '"', ' ', '\n', '\r':
			return true
		}
	}
	return false
}

// ResultadoBloque es lo que produce un worker por bloque.
type ResultadoBloque struct {
	Silver     []byte
	Entradas   int
	Salidas    int
	PorRegla   [numReglas]int
	Corregidas int
	Positivos  int
	Huellas    []uint64 // huella de cada línea bronze válida, para contar repetidas
}

// semilla fija para que las huellas sean las mismas en todos los workers.
var semillaHuellas = maphash.MakeSeed()

// LimpiarBloque limpia todas las líneas de un bloque.
func LimpiarBloque(bloque []byte) ResultadoBloque {
	var res ResultadoBloque
	res.Silver = make([]byte, 0, len(bloque))
	buf := make([][]byte, 0, NumColumnas)
	ParaCadaLinea(bloque, func(linea []byte) {
		res.Entradas++
		campos, err := ParsearLinea(linea, buf)
		if err != nil {
			res.PorRegla[MalFormada]++
			return
		}
		r, regla, corr := LimpiarCampos(campos)
		if regla != OK {
			res.PorRegla[regla]++
			return
		}
		if corr {
			res.Corregidas++
		}
		res.Salidas++
		if r.Clase >= 1 {
			res.Positivos++
		}
		res.Huellas = append(res.Huellas, maphash.Bytes(semillaHuellas, linea))
		res.Silver = AgregarSilver(res.Silver, r)
	})
	return res
}

// Limpieza es el resumen de toda la limpieza.
type Limpieza struct {
	Entradas       int            `json:"filas_entrada"`
	Salidas        int            `json:"filas_salida"`
	Descartadas    int            `json:"filas_descartadas"`
	PorRegla       map[string]int `json:"descartadas_por_regla"`
	Corregidas     int            `json:"ubigeo_rellenado_con_cero"`
	Positivos      int            `json:"positivos"`
	TasaPositivos  float64        `json:"tasa_positivos"`
	FilasIdenticas int            `json:"filas_identicas_conservadas"`
	OrdenReglas    []string       `json:"orden_reglas"`
}

// LimpiarConcurrente limpia el cuerpo del CSV y devuelve el silver (con
// encabezado) y el resumen. El silver es idéntico para cualquier workers.
func LimpiarConcurrente(cuerpo []byte, tamBloque, workers int) ([]byte, Limpieza) {
	rangos := PartirEnBloques(cuerpo, tamBloque)
	partes := MapearEnOrden(cuerpo, rangos, workers, func(_ int, b []byte) ResultadoBloque {
		return LimpiarBloque(b)
	})
	l := Limpieza{PorRegla: map[string]int{}}
	total := len(EncabezadoSilver)
	for i := range partes {
		total += len(partes[i].Silver)
	}
	silver := make([]byte, 0, total)
	silver = append(silver, EncabezadoSilver...)
	vistas := make(map[uint64]struct{}, len(cuerpo)/100)
	var porRegla [numReglas]int
	for i := range partes {
		p := &partes[i]
		silver = append(silver, p.Silver...)
		l.Entradas += p.Entradas
		l.Salidas += p.Salidas
		l.Corregidas += p.Corregidas
		l.Positivos += p.Positivos
		for j, n := range p.PorRegla {
			porRegla[j] += n
		}
		for _, h := range p.Huellas {
			if _, ok := vistas[h]; ok {
				l.FilasIdenticas++
			} else {
				vistas[h] = struct{}{}
			}
		}
	}
	for j := Regla(1); j < numReglas; j++ {
		l.OrdenReglas = append(l.OrdenReglas, NombresReglas[j])
		l.PorRegla[NombresReglas[j]] = porRegla[j]
	}
	l.Descartadas = l.Entradas - l.Salidas
	if l.Salidas > 0 {
		l.TasaPositivos = float64(l.Positivos) / float64(l.Salidas)
	}
	return silver, l
}
