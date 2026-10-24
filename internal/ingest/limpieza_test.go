package ingest

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"testing"
)

func campos(t *testing.T, linea string) [][]byte {
	t.Helper()
	c, err := ParsearLinea([]byte(linea), nil)
	if err != nil {
		t.Fatalf("%q: %v", linea, err)
	}
	return c
}

const base = `PIURA,SULLANA,BELLAVISTA,BELLAVISTA,"DENGUE CON SIGNOS DE ALARMA",2024,44,A97.1,C,31,200602,2006020001,71,A,F`

// reemplazar cambia la columna col de la línea base.
func reemplazar(t *testing.T, col int, valor string) string {
	t.Helper()
	c := campos(t, base)
	partes := make([]string, len(c))
	for i, v := range c {
		partes[i] = string(v)
		if strings.ContainsAny(partes[i], " ,") {
			partes[i] = `"` + partes[i] + `"`
		}
	}
	partes[col] = valor
	return strings.Join(partes, ",")
}

func TestLimpiarCamposValida(t *testing.T) {
	r, regla, corr := LimpiarCampos(campos(t, base))
	if regla != OK || corr {
		t.Fatalf("regla=%v corregido=%v", regla, corr)
	}
	if r.Departamento != "PIURA" || r.Ubigeo != "200602" || r.Ano != 2024 || r.Semana != 44 ||
		r.EdadAnios != 71 || r.Sexo != 'F' || r.TipoDx != 'C' || r.Clase != 1 || r.Objetivo() != 1 {
		t.Fatalf("registro = %+v", r)
	}
}

func TestCadaReglaSeDetecta(t *testing.T) {
	casos := []struct {
		col   int
		valor string
		regla Regla
	}{
		{ColDiagnostico, "A90", DiagnosticoDesconocido},
		{ColEnfermedad, `"DENGUE GRAVE"`, EnfermedadInconsistente}, // A97.1 dice "con signos"
		{ColAno, "1999", AnoFueraDeRango},
		{ColAno, "2026", AnoFueraDeRango},
		{ColSemana, "0", SemanaFueraDeRango},
		{ColSemana, "54", SemanaFueraDeRango},
		{ColEdad, "NA", EdadInvalida},
		{ColTipoEdad, "X", EdadInvalida},
		{ColEdad, "111", EdadFueraDeRango},
		{ColSexo, "I", SexoInvalido},
		{ColTipoDx, "Z", TipoDxInvalido},
		{ColUbigeo, "2006", UbigeoInvalido},
		{ColUbigeo, "20060A", UbigeoInvalido},
		{ColDepartamento, "", DepartamentoVacio},
	}
	for _, c := range casos {
		_, regla, _ := LimpiarCampos(campos(t, reemplazar(t, c.col, c.valor)))
		if regla != c.regla {
			t.Errorf("columna %s = %q: regla %q, quiere %q", Columnas[c.col], c.valor, NombresReglas[regla], NombresReglas[c.regla])
		}
	}
}

func TestUbigeoDeCincoDigitosSeCorrige(t *testing.T) {
	r, regla, corr := LimpiarCampos(campos(t, reemplazar(t, ColUbigeo, "60801")))
	if regla != OK || !corr || r.Ubigeo != "060801" {
		t.Fatalf("regla=%v corregido=%v ubigeo=%q", regla, corr, r.Ubigeo)
	}
}

func TestEdadEnMesesYDias(t *testing.T) {
	r, regla, _ := LimpiarCampos(campos(t, reemplazar(t, ColTipoEdad, "M")))
	if regla != OK || r.EdadAnios != 71.0/12 {
		t.Fatalf("71 meses → %v años (regla %v)", r.EdadAnios, regla)
	}
}

func TestSilverIdaYVuelta(t *testing.T) {
	r, _, _ := LimpiarCampos(campos(t, base))
	r.Distrito = `SAN "X", Y`
	linea := AgregarSilver(nil, r)
	c, err := ParsearLineaN(bytes.TrimRight(linea, "\n"), nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if string(c[2]) != `SAN "X", Y` || string(c[6]) != "71" || string(c[9]) != "1" {
		t.Fatalf("silver = %q", linea)
	}
}

func TestLimpiezaConcurrenteIgualALaSecuencial(t *testing.T) {
	_, cuerpo := SepararEncabezado(csvSintetico(40_000, 5))
	refSilver, ref := LimpiarConcurrente(cuerpo, 1<<30, 1)
	refHash := sha256.Sum256(refSilver)
	for _, tam := range []int{2048, 65536} {
		for _, w := range []int{1, 2, 4, 8} {
			silver, l := LimpiarConcurrente(cuerpo, tam, w)
			if sha256.Sum256(silver) != refHash {
				t.Fatalf("tam=%d workers=%d: el silver difiere del secuencial", tam, w)
			}
			if l.Salidas != ref.Salidas || l.FilasIdenticas != ref.FilasIdenticas || l.Positivos != ref.Positivos {
				t.Fatalf("tam=%d workers=%d: resumen %+v vs %+v", tam, w, l, ref)
			}
		}
	}
	if ref.Entradas != 40_000 || ref.Salidas+ref.Descartadas != ref.Entradas {
		t.Fatalf("conteos inconsistentes: %+v", ref)
	}
	suma := 0
	for _, n := range ref.PorRegla {
		suma += n
	}
	if suma != ref.Descartadas {
		t.Fatalf("Σ por regla = %d, descartadas = %d", suma, ref.Descartadas)
	}
	// Defectos sembrados por csvSintetico: comillas sin cerrar (i%1999==7, irreparables),
	// edad NA (i%997==13), sexo X (i%997==29).
	if ref.PorRegla[NombresReglas[MalFormada]] != 21 {
		t.Errorf("mal formadas = %d, quiere 21", ref.PorRegla[NombresReglas[MalFormada]])
	}
	if ref.PorRegla[NombresReglas[EdadInvalida]] == 0 || ref.PorRegla[NombresReglas[SexoInvalido]] == 0 {
		t.Errorf("faltan descartes sembrados: %v", ref.PorRegla)
	}
	if !bytes.HasPrefix(refSilver, []byte(EncabezadoSilver)) {
		t.Error("el silver no empieza con el encabezado")
	}
}

func TestMuestraRealPasaLaLimpieza(t *testing.T) {
	datos := mustLeer(t, "testdata/muestra.csv")
	_, cuerpo := SepararEncabezado(datos)
	_, l := LimpiarConcurrente(cuerpo, 0, 2)
	if l.Salidas != 8 || l.Positivos != 1 {
		t.Fatalf("muestra real: %+v", l)
	}
}

func TestLeerSilverConcurrenteIgualAlRegistroOriginal(t *testing.T) {
	_, cuerpo := SepararEncabezado(csvSintetico(20_000, 8))
	silver, l := LimpiarConcurrente(cuerpo, 4096, 4)
	for _, w := range []int{1, 4} {
		regs, err := LeerSilverConcurrente(silver, 2048, w)
		if err != nil {
			t.Fatal(err)
		}
		if len(regs) != l.Salidas {
			t.Fatalf("workers=%d: %d registros, quiere %d", w, len(regs), l.Salidas)
		}
		// Reescribir los registros leídos debe dar exactamente el mismo silver.
		otra := []byte(EncabezadoSilver)
		for _, r := range regs {
			otra = AgregarSilver(otra, r)
		}
		if !bytes.Equal(otra, silver) {
			t.Fatalf("workers=%d: silver → registros → silver no es idéntico", w)
		}
	}
	if _, err := LeerSilverConcurrente([]byte("a,b\n"), 0, 1); err == nil {
		t.Error("esperaba error con encabezado inválido")
	}
}
