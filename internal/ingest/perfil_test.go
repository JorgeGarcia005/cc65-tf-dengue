package ingest

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
)

// csvSintetico genera n filas con la forma del CSV real, con algunas filas
// defectuosas a propósito para ejercitar el perfil y la limpieza.
func csvSintetico(n int, semilla uint64) []byte {
	r := rand.New(rand.NewPCG(semilla, 3))
	deptos := []string{"PIURA", "LORETO", `"LA LIBERTAD"`, "ICA", "TUMBES", "LAMBAYEQUE"}
	enf := []string{`"DENGUE SIN SIGNOS DE ALARMA"`, `"DENGUE CON SIGNOS DE ALARMA"`, `"DENGUE GRAVE"`}
	dx := []string{"A97.0", "A97.1", "A97.2"}
	var sb strings.Builder
	sb.WriteString(strings.Join(Columnas, ",") + "\n")
	for i := range n {
		c := 0
		switch x := r.Float64(); {
		case x < 0.03:
			c = 2
		case x < 0.13:
			c = 1
		}
		tipo := "A"
		if r.Float64() < 0.01 {
			tipo = "M"
		}
		sexo := "F"
		if r.IntN(2) == 1 {
			sexo = "M"
		}
		edad := fmt.Sprint(r.IntN(90))
		switch i % 997 { // defectos deterministas
		case 13:
			edad = "NA"
		case 29:
			sexo = "X"
		}
		linea := fmt.Sprintf("%s,PROV,DIST,LOC,%s,%d,%d,%s,C,31,200101,,%s,%s,%s",
			deptos[r.IntN(len(deptos))], enf[c], 2000+r.IntN(26), 1+r.IntN(52), dx[c], edad, tipo, sexo)
		if i%1999 == 7 {
			linea = `PIURA,"SIN CERRAR,DIST`
		}
		sb.WriteString(linea + "\n")
	}
	return []byte(sb.String())
}

func TestPerfilConcurrenteIgualAlSecuencial(t *testing.T) {
	_, cuerpo := SepararEncabezado(csvSintetico(30_000, 1))
	ref := PerfilarConcurrente(cuerpo, 1<<30, 1) // un solo bloque, un worker
	refJSON, _ := json.Marshal(ref)
	for _, tam := range []int{4096, 65536} {
		for _, w := range []int{1, 2, 4, 8} {
			got := PerfilarConcurrente(cuerpo, tam, w)
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != string(refJSON) {
				t.Fatalf("tam=%d workers=%d: el perfil difiere del secuencial", tam, w)
			}
		}
	}
	if ref.Filas != 30_000 {
		t.Errorf("filas = %d", ref.Filas)
	}
	if ref.ErrComillas != 16 { // i%1999==7 para i<30000 → 16 filas
		t.Errorf("err_comillas = %d, quiere 16", ref.ErrComillas)
	}
	if ref.EdadAnios.NoNum == 0 {
		t.Error("esperaba edades no numéricas en el sintético")
	}
}

func TestPerfilMuestraReal(t *testing.T) {
	datos, err := os.ReadFile("testdata/muestra.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, cuerpo := SepararEncabezado(datos)
	p := PerfilarConcurrente(cuerpo, 0, 2)
	if p.Filas != 8 || p.FilasMalas != 0 {
		t.Fatalf("filas=%d malas=%d", p.Filas, p.FilasMalas)
	}
	if p.Columnas[ColLocalcod].Vacios != 2 {
		t.Errorf("localcod vacíos = %d, quiere 2", p.Columnas[ColLocalcod].Vacios)
	}
	if c := p.PorAnoClase["2024"]; c != [3]int{1, 1, 0} {
		t.Errorf("2024 por clase = %v", c)
	}
	// edades 9, 18, 40, 2, 31, 21, 49, 71 → niño 2, joven 2, adulto 3, mayor 1
	if got := p.EdadAnios.Conteos; fmt.Sprint(got) != "[2 0 2 3 1]" {
		t.Errorf("tramos de edad = %v", got)
	}
}

func TestEdadEnAnios(t *testing.T) {
	casos := []struct {
		edad, tipo string
		quiere     float64
		ok         bool
	}{
		{"30", "A", 30, true}, {"6", "M", 0.5, true}, {"365", "D", 1, true},
		{"x", "A", 0, false}, {"-1", "A", 0, false}, {"3", "Z", 0, false},
	}
	for _, c := range casos {
		v, ok := EdadEnAnios([]byte(c.edad), []byte(c.tipo))
		if ok != c.ok || v != c.quiere {
			t.Errorf("EdadEnAnios(%s,%s) = %v,%v", c.edad, c.tipo, v, ok)
		}
	}
}

// PR-14 H1: filas con número de columnas distinto de 15 (sin problemas de comillas).
func TestPerfilCuentaErroresDeColumnas(t *testing.T) {
	bloque := []byte(strings.Join([]string{
		`PIURA,SULLANA,BELLAVISTA,BELLAVISTA,"DENGUE CON SIGNOS DE ALARMA",2024,44,A97.1,C,31,200602,2006020001,71,A,F`,
		`PIURA,SULLANA,BELLAVISTA`,                       // 3 columnas
		`A,B,C,D,E,2024,1,A97.0,C,1,100101,,1,A,F,EXTRA`, // 16 columnas
		`PIURA,"SIN CERRAR,X`,                            // comillas
	}, "\n") + "\n")
	p := PerfilarBloque(bloque)
	if p.Filas != 4 || p.FilasMalas != 3 || p.ErrColumnas != 2 || p.ErrComillas != 1 {
		t.Fatalf("filas=%d malas=%d columnas=%d comillas=%d", p.Filas, p.FilasMalas, p.ErrColumnas, p.ErrComillas)
	}
	if p.Columnas[ColDepartamento].Valores["PIURA"] != 1 {
		t.Error("solo la fila válida debe contar en las columnas")
	}
}
