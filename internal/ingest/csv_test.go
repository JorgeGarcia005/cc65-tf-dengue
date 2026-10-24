package ingest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParsearLinea(t *testing.T) {
	casos := []struct {
		linea  string
		quiere []string
		err    error
	}{
		{
			`HUANUCO,"LEONCIO PRADO",LUYANDO,NARANJILLO,"DENGUE SIN SIGNOS DE ALARMA",2000,40,A97.0,P,10,100605,,18,A,F`,
			[]string{"HUANUCO", "LEONCIO PRADO", "LUYANDO", "NARANJILLO", "DENGUE SIN SIGNOS DE ALARMA", "2000", "40", "A97.0", "P", "10", "100605", "", "18", "A", "F"},
			nil,
		},
		{ // comillas escapadas y coma dentro de un campo
			`A,"B ""x"", y",C,D,E,2000,1,A97.0,C,1,100101,,1,A,`,
			[]string{"A", `B "x", y`, "C", "D", "E", "2000", "1", "A97.0", "C", "1", "100101", "", "1", "A", ""},
			nil,
		},
		{`A,"sin cerrar,C`, nil, ErrComillas},
		{`A,B"C,D`, nil, ErrComillas},
		{`A,"B"x,C`, nil, ErrComillas},
		{`A,B,C`, nil, ErrColumnas},
	}
	for _, c := range casos {
		got, err := ParsearLinea([]byte(c.linea), nil)
		if !errors.Is(err, c.err) {
			t.Errorf("%q: error %v, quiere %v", c.linea, err, c.err)
			continue
		}
		if c.err != nil {
			continue
		}
		for i := range c.quiere {
			if string(got[i]) != c.quiere[i] {
				t.Errorf("%q campo %d = %q, quiere %q", c.linea, i, got[i], c.quiere[i])
			}
		}
	}
}

func TestEncabezadoReal(t *testing.T) {
	datos, err := os.ReadFile("testdata/muestra.csv")
	if err != nil {
		t.Fatal(err)
	}
	enc, cuerpo := SepararEncabezado(datos)
	if err := ValidarEncabezado(enc); err != nil {
		t.Fatal(err)
	}
	n := 0
	ParaCadaLinea(cuerpo, func(l []byte) {
		if _, err := ParsearLinea(l, nil); err != nil {
			t.Errorf("fila real %d: %v", n, err)
		}
		n++
	})
	if n != 8 {
		t.Errorf("filas = %d, quiere 8", n)
	}
	if err := ValidarEncabezado([]byte("a,b,c")); err == nil {
		t.Error("esperaba error con encabezado distinto")
	}
}

func TestPartirEnBloquesNoCortaLineas(t *testing.T) {
	var sb strings.Builder
	for i := range 5_000 {
		fmt.Fprintf(&sb, "fila-%d,%s\r\n", i, strings.Repeat("x", i%37))
	}
	datos := []byte(sb.String())
	for _, tam := range []int{1, 7, 100, 4096, 1 << 30} {
		rangos := PartirEnBloques(datos, tam)
		var juntas bytes.Buffer
		siguiente := 0
		for _, r := range rangos {
			if r.Ini != siguiente {
				t.Fatalf("tam=%d: hueco o solape en %d", tam, r.Ini)
			}
			if r.Fin < len(datos) && datos[r.Fin-1] != '\n' {
				t.Fatalf("tam=%d: el bloque termina a mitad de línea", tam)
			}
			juntas.Write(datos[r.Ini:r.Fin])
			siguiente = r.Fin
		}
		if !bytes.Equal(juntas.Bytes(), datos) {
			t.Fatalf("tam=%d: los bloques no reconstruyen los datos", tam)
		}
	}
	if len(PartirEnBloques(nil, 10)) != 0 {
		t.Error("datos vacíos deben dar cero bloques")
	}
}

func TestMapearEnOrdenEsDeterministaYSinCarreras(t *testing.T) {
	var sb strings.Builder
	for i := range 20_000 {
		fmt.Fprintf(&sb, "%d\n", i)
	}
	datos := []byte(sb.String())
	rangos := PartirEnBloques(datos, 512)
	contar := func(_ int, b []byte) int { return bytes.Count(b, []byte{'\n'}) }
	ref := MapearEnOrden(datos, rangos, 1, contar)
	for _, w := range []int{2, 4, 8, 64} {
		got := MapearEnOrden(datos, rangos, w, contar)
		for k := range ref {
			if got[k] != ref[k] {
				t.Fatalf("workers=%d bloque %d: %d vs %d", w, k, got[k], ref[k])
			}
		}
	}
	total := 0
	for _, v := range ref {
		total += v
	}
	if total != 20_000 {
		t.Errorf("líneas contadas = %d", total)
	}
}

func mustLeer(t *testing.T, ruta string) []byte {
	t.Helper()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParsearLineaTolerante(t *testing.T) {
	reales := []string{
		`LORETO,"MARISCAL RAMON CASTILLA","RAMON CASTILLA","ISLA SANTA ROSA "AMAZONAS"","DENGUE CON SIGNOS DE ALARMA",2012,46,A97.1,C,16,160401,1604010076,28,A,F`,
		`LORETO,MAYNAS,PUNCHANA,"PSJE. VICTORIA N"51","DENGUE SIN SIGNOS DE ALARMA",2004,39,A97.0,P,16,160108,,21,A,M`,
		`ICA,ICA,PACHACUTEC,"SAN FRANCISCO "A"","DENGUE SIN SIGNOS DE ALARMA",2020,51,A97.0,P,11,110105,1101050019,62,A,F`,
	}
	localidades := []string{`ISLA SANTA ROSA "AMAZONAS"`, `PSJE. VICTORIA N"51`, `SAN FRANCISCO "A"`}
	for i, l := range reales {
		if _, err := ParsearLinea([]byte(l), nil); !errors.Is(err, ErrComillas) {
			t.Fatalf("fila %d: el parseo estricto debía fallar por comillas, dio %v", i, err)
		}
		c, reparada, err := ParsearLineaTolerante([]byte(l), nil, NumColumnas)
		if err != nil || !reparada {
			t.Fatalf("fila %d: err=%v reparada=%v", i, err, reparada)
		}
		if string(c[ColLocalidad]) != localidades[i] || string(c[ColEnfermedad])[:6] != "DENGUE" || len(c[ColSexo]) != 1 {
			t.Fatalf("fila %d: campos %q", i, c)
		}
	}
	// Una línea válida no se marca como reparada.
	if _, reparada, err := ParsearLineaTolerante([]byte(reales[0][:0]+`A,B,C,D,E,2000,1,A97.0,C,1,100101,,1,A,F`), nil, NumColumnas); err != nil || reparada {
		t.Fatalf("línea válida: err=%v reparada=%v", err, reparada)
	}
	// Si ni el modo laxo da 15 columnas, sigue siendo error.
	if _, _, err := ParsearLineaTolerante([]byte(`A,"B`), nil, NumColumnas); err == nil {
		t.Fatal("esperaba error")
	}
}
