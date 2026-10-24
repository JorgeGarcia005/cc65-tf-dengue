package ingest

import (
	"cmp"
	"slices"
	"strconv"
)

// MaxCategorias limita cuántos valores distintos se reportan por columna.
const MaxCategorias = 40

// PerfilColumna resume una columna del CSV tal como llega (bronze).
type PerfilColumna struct {
	Nombre    string         `json:"nombre"`
	Vacios    int            `json:"vacios"`
	Distintos int            `json:"distintos"`
	Valores   map[string]int `json:"-"`
	Top       []Conteo       `json:"top"`
}

// Conteo es un valor y su frecuencia.
type Conteo struct {
	Valor string `json:"valor"`
	N     int    `json:"n"`
}

// Perfil es el resultado de perfilar el archivo (o un bloque). Las sumas de
// conteos son conmutativas, así que unir perfiles de bloques en cualquier
// orden da el mismo resultado; aun así se unen en orden de bloque.
type Perfil struct {
	Filas       int               `json:"filas"`
	FilasMalas  int               `json:"filas_mal_formadas"`
	ErrComillas int               `json:"err_comillas"`
	ErrColumnas int               `json:"err_columnas"`
	Columnas    []PerfilColumna   `json:"columnas"`
	EdadAnios   Histograma        `json:"edad_anios"`
	CasosPorAno map[string]int    `json:"casos_por_ano"`
	PorAnoClase map[string][3]int `json:"por_ano_y_clase"`
}

// Histograma de edad (en años) por tramos de etapa de vida MINSA.
type Histograma struct {
	Tramos  []string `json:"tramos"`
	Conteos []int    `json:"conteos"`
	NoNum   int      `json:"no_numerica"`
}

// TramosEdad son las etapas de vida del MINSA: niño, adolescente, joven,
// adulto y adulto mayor. El límite superior de cada tramo es exclusivo.
var TramosEdad = []struct {
	Nombre string
	Hasta  float64
}{
	{"nino_0_11", 12}, {"adolescente_12_17", 18}, {"joven_18_29", 30},
	{"adulto_30_59", 60}, {"adulto_mayor_60_mas", 1e9},
}

// TramoEdad devuelve el índice del tramo MINSA para una edad en años.
func TramoEdad(anios float64) int {
	for i, t := range TramosEdad {
		if anios < t.Hasta {
			return i
		}
	}
	return len(TramosEdad) - 1
}

// NuevoPerfil crea un perfil vacío.
func NuevoPerfil() *Perfil {
	p := &Perfil{
		Columnas:    make([]PerfilColumna, NumColumnas),
		CasosPorAno: map[string]int{},
		PorAnoClase: map[string][3]int{},
		EdadAnios:   Histograma{Conteos: make([]int, len(TramosEdad))},
	}
	for i := range p.Columnas {
		p.Columnas[i] = PerfilColumna{Nombre: Columnas[i], Valores: map[string]int{}}
	}
	for _, t := range TramosEdad {
		p.EdadAnios.Tramos = append(p.EdadAnios.Tramos, t.Nombre)
	}
	return p
}

// PerfilarBloque perfila todas las líneas de un bloque.
func PerfilarBloque(bloque []byte) *Perfil {
	p := NuevoPerfil()
	buf := make([][]byte, 0, NumColumnas)
	ParaCadaLinea(bloque, func(linea []byte) {
		p.Filas++
		campos, err := ParsearLinea(linea, buf)
		switch err {
		case nil:
		case ErrComillas:
			p.FilasMalas++
			p.ErrComillas++
			return
		default:
			p.FilasMalas++
			p.ErrColumnas++
			return
		}
		for i, c := range campos {
			col := &p.Columnas[i]
			if len(c) == 0 {
				col.Vacios++
			}
			// Las columnas de alta cardinalidad (localidad, localcod) también se
			// cuentan; Top recorta lo que se reporta.
			col.Valores[string(c)]++
		}
		ano := string(campos[ColAno])
		p.CasosPorAno[ano]++
		clase := p.PorAnoClase[ano]
		switch string(campos[ColDiagnostico]) {
		case "A97.0":
			clase[0]++
		case "A97.1":
			clase[1]++
		case "A97.2":
			clase[2]++
		}
		p.PorAnoClase[ano] = clase
		if anios, ok := EdadEnAnios(campos[ColEdad], campos[ColTipoEdad]); ok {
			p.EdadAnios.Conteos[TramoEdad(anios)]++
		} else {
			p.EdadAnios.NoNum++
		}
	})
	return p
}

// EdadEnAnios convierte (edad, tipo_edad) a años: A = años, M = meses, D = días.
func EdadEnAnios(edad, tipo []byte) (float64, bool) {
	v, err := strconv.Atoi(string(edad))
	if err != nil || v < 0 {
		return 0, false
	}
	switch string(tipo) {
	case "A":
		return float64(v), true
	case "M":
		return float64(v) / 12, true
	case "D":
		return float64(v) / 365, true
	}
	return 0, false
}

// Unir suma q en p.
func (p *Perfil) Unir(q *Perfil) {
	p.Filas += q.Filas
	p.FilasMalas += q.FilasMalas
	p.ErrComillas += q.ErrComillas
	p.ErrColumnas += q.ErrColumnas
	for i := range p.Columnas {
		p.Columnas[i].Vacios += q.Columnas[i].Vacios
		for v, n := range q.Columnas[i].Valores {
			p.Columnas[i].Valores[v] += n
		}
	}
	for a, n := range q.CasosPorAno {
		p.CasosPorAno[a] += n
	}
	for a, c := range q.PorAnoClase {
		t := p.PorAnoClase[a]
		for j := range t {
			t[j] += c[j]
		}
		p.PorAnoClase[a] = t
	}
	for j := range p.EdadAnios.Conteos {
		p.EdadAnios.Conteos[j] += q.EdadAnios.Conteos[j]
	}
	p.EdadAnios.NoNum += q.EdadAnios.NoNum
}

// Cerrar calcula Distintos y Top (orden: frecuencia desc, valor asc) para el reporte.
func (p *Perfil) Cerrar() {
	for i := range p.Columnas {
		col := &p.Columnas[i]
		col.Distintos = len(col.Valores)
		col.Top = col.Top[:0]
		for v, n := range col.Valores {
			col.Top = append(col.Top, Conteo{v, n})
		}
		slices.SortFunc(col.Top, func(a, b Conteo) int {
			if c := cmp.Compare(b.N, a.N); c != 0 {
				return c
			}
			return cmp.Compare(a.Valor, b.Valor)
		})
		if len(col.Top) > MaxCategorias {
			col.Top = col.Top[:MaxCategorias]
		}
	}
}

// PerfilarConcurrente perfila el cuerpo del CSV con workers goroutines.
func PerfilarConcurrente(cuerpo []byte, tamBloque, workers int) *Perfil {
	rangos := PartirEnBloques(cuerpo, tamBloque)
	parciales := MapearEnOrden(cuerpo, rangos, workers, func(_ int, b []byte) *Perfil {
		return PerfilarBloque(b)
	})
	total := NuevoPerfil()
	for _, q := range parciales {
		total.Unir(q)
	}
	total.Cerrar()
	return total
}
