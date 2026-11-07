package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const nsPrueba = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`

func envolver(cuerpo string) string { return `<w:body ` + nsPrueba + `>` + cuerpo + `</w:body>` }

func TestEcuacionesDelInformeSonXMLValido(t *testing.T) {
	for _, expr := range []string{
		`L(w) = \frac{1}{S}\sum_i s_i\,\big[\log(1+e^{z_i}) - y_i z_i\big] + \frac{\lambda}{2}\lVert w\rVert^2,\qquad z_i = x_i^\top w + b`,
		`\nabla_w L = \frac{1}{S}\sum_i s_i\,(\sigma(z_i) - y_i)\,x_i + \lambda w,\qquad S = \sum_i s_i`,
		`B_1,\dots,B_K`, `e^{w}`, `\frac{1}{S}\sum_k g_k + \lambda w`, `((g_0 + g_1) + g_2) + \dots`,
		`a < b & c`, `\desconocido{x}`, `x_`, `\frac{1}`,
	} {
		x := `<m:oMath>` + LatexAOMML(expr) + `</m:oMath>`
		if err := xmlBienFormado(envolver(x)); err != nil {
			t.Errorf("%q: %v", expr, err)
		}
	}
}

// La sumatoria debe tomar como cuerpo el término completo, aunque tenga un
// '+' dentro de paréntesis: log(1+e^z) no puede quedar partido.
func TestSumatoriaTomaElTerminoCompleto(t *testing.T) {
	x := LatexAOMML(`\sum_i s_i\,[\log(1+e^{z_i}) - y_i z_i] + \lambda`)
	ini := strings.Index(x, "<m:e>")
	fin := strings.LastIndex(x, "</m:nary>")
	if ini < 0 || fin < 0 {
		t.Fatalf("no hay sumatoria: %s", x)
	}
	cuerpo := x[ini:fin]
	for _, quiere := range []string{"log", ">1<", ">+<", ">e<", ">y<"} {
		if !strings.Contains(cuerpo, quiere) {
			t.Errorf("el cuerpo de la sumatoria no contiene %q", quiere)
		}
	}
	if strings.Contains(cuerpo, "λ") {
		t.Error("λ quedó dentro de la sumatoria: el término termina en el '+' de afuera")
	}
}

func TestAnchosSumanElAnchoDeLaPagina(t *testing.T) {
	casos := [][][]string{
		{{"Parte", "Componente"}, {"A", "Cargador concurrente con un texto muy largo que ocupa mucho"}},
		{{"a", "b", "c", "d", "e", "f", "g", "h"}},
		{{"TestMapearEnOrdenEsDeterministaYSinCarreras", "TestShardsReproducenElGradienteGlobal", "x"}},
	}
	for _, filas := range casos {
		nc := len(filas[0])
		a := anchosColumnas(filas, nc, 20)
		suma := 0
		for j, w := range a {
			if w <= 0 {
				t.Errorf("columna %d con ancho %d", j, w)
			}
			suma += w
		}
		if suma != anchoTexto {
			t.Errorf("anchos %v suman %d, quiere %d", a, suma, anchoTexto)
		}
	}
	// La palabra más corta del encabezado no debe partirse: "Parte" necesita
	// más que un reparto puramente proporcional.
	a := anchosColumnas([][]string{{"Parte", "Componente"}, {"A", strings.Repeat("palabra ", 30)}}, 2, 20)
	if a[0] < 5*110 {
		t.Errorf("columna «Parte» de %d dxa: se partiría la palabra", a[0])
	}
}

func TestInlineEscapaYFormatea(t *testing.T) {
	d := nuevoDocumento(".", 200, 4)
	x := d.inline("**negrita** y *cursiva* con `a<b && c` y [enlace](https://x.pe/?a=1&b=2) y $s_i$ y 3 < 4", texto(22))
	if err := xmlBienFormado(envolver(`<w:p>` + x + `</w:p>`)); err != nil {
		t.Fatalf("XML mal formado: %v\n%s", err, x)
	}
	for _, quiere := range []string{"<w:i/>", "Consolas", "<w:hyperlink", "<m:oMath>", "3 &lt; 4", "a&lt;b &amp;&amp; c"} {
		if !strings.Contains(x, quiere) {
			t.Errorf("falta %q en %s", quiere, x)
		}
	}
	// Sin negrita dentro del texto y sin marcas ** visibles.
	if strings.Contains(x, "<w:b/>") || strings.Contains(x, "**") {
		t.Errorf("el texto no debe llevar negrita: %s", x)
	}
	// En un encabezado (estilo base en negrita) la negrita sí se mantiene.
	e := texto(20)
	e.Negrita = true
	if !strings.Contains(d.inline("Columna", e), "<w:b/>") {
		t.Error("el encabezado perdió la negrita")
	}
	if len(d.orden) != 1 || d.orden[0].URL != "https://x.pe/?a=1&b=2" {
		t.Errorf("enlaces: %+v", d.orden)
	}
}

// Cada corrida debe tener sus propiedades una sola vez: un rPr con dos
// w:rFonts o dos w:color hace que Word marque el documento como dañado.
func TestRPrSinElementosRepetidos(t *testing.T) {
	d := nuevoDocumento(".", 200, 4)
	e := texto(20)
	e.Color, e.Negrita = "FFFFFF", true
	x := d.inline("`codigo` y [enlace](https://a.pe)", e)
	for _, rpr := range strings.Split(x, "<w:rPr>")[1:] {
		rpr = rpr[:strings.Index(rpr, "</w:rPr>")]
		for _, tag := range []string{"<w:rFonts", "<w:color", "<w:sz ", "<w:b/>"} {
			if strings.Count(rpr, tag) > 1 {
				t.Errorf("%s repetido en rPr: %s", tag, rpr)
			}
		}
	}
}

func TestMarkdownCompletoEsXMLValido(t *testing.T) {
	dir := t.TempDir()
	md := "# 1. Título\n\nPárrafo con **negrita**\ny continuación.\n\n- uno\n  sigue\n- dos\n\n1. a\n2. b\n\n" +
		"| A | B |\n|---|---:|\n| `x|y` | 2 |\n\n> cita\n\n$$x^2$$\n\n```go\nfunc main() {}\n```\n\n" +
		"<!-- estilo: referencias -->\n\nAutor, A. (2020). *Título*.\n\n![Figura de prueba](f.svg)\n\n" +
		"<!-- captura: Ejecución del loader -->\n"
	if err := os.WriteFile(filepath.Join(dir, "01-x.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200"></svg>`
	if err := os.WriteFile(filepath.Join(dir, "f.svg"), []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}
	d := nuevoDocumento(dir, 200, 4)
	d.Convertir("01-x.md")
	if len(d.avisos) > 0 {
		t.Errorf("avisos: %v", d.avisos)
	}
	if err := xmlBienFormado(envolver(d.b.String())); err != nil {
		t.Fatalf("XML mal formado: %v", err)
	}
	if len(d.numIds) != 1 || len(d.imagenes) != 1 {
		t.Errorf("listas numeradas %d (quiere 1), imágenes %d (quiere 1)", len(d.numIds), len(d.imagenes))
	}
	if !strings.Contains(d.b.String(), `w:hanging="720"`) {
		t.Error("las referencias no tienen sangría francesa")
	}
	if !strings.Contains(d.b.String(), ">x|y<") {
		t.Error("una barra dentro de `código` no debe partir la celda")
	}
	// La captura se numera después de la figura: Figura 2.
	if !strings.Contains(d.b.String(), "Figura 2. Ejecución del loader") || !strings.Contains(d.b.String(), "[Pegar aquí la captura]") {
		t.Error("falta el espacio para la captura con su pie numerado")
	}
}

func TestReemplazarParrafoUneCorridas(t *testing.T) {
	x := `<w:p><w:r><w:t>Smart</w:t></w:r><w:r><w:t xml:space="preserve">Transit: Demanda</w:t></w:r></w:p><w:p><w:r><w:t>TB1</w:t></w:r></w:p>`
	y, ok := reemplazarParrafo(x, "SmartTransit: Demanda", "Dengue & riesgo")
	if !ok {
		t.Fatal("no encontró el texto partido en dos corridas")
	}
	if !strings.Contains(y, "Dengue &amp; riesgo") || strings.Contains(y, "Smart") {
		t.Errorf("reemplazo incorrecto: %s", y)
	}
	if _, ok := reemplazarParrafo(y, "no existe", "x"); ok {
		t.Error("informó un reemplazo que no hizo")
	}
}

// TestLeerCodigos: los códigos salen del flag o del archivo ignorado por git,
// nunca de un valor por defecto en el código (el repositorio es público).
func TestLeerCodigos(t *testing.T) {
	archivo := filepath.Join(t.TempDir(), "codigos.txt")
	if _, err := leerCodigos("", archivo); err == nil {
		t.Error("sin flag ni archivo debe fallar")
	}
	got, err := leerCodigos(" A1 , B2 ", archivo)
	if err != nil || !slices.Equal(got, []string{"A1", "B2"}) {
		t.Fatalf("flag: %v, %v", got, err)
	}
	if err := os.WriteFile(archivo, []byte("# integrantes\nA1\n\n B2 \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = leerCodigos("", archivo)
	if err != nil || !slices.Equal(got, []string{"A1", "B2"}) {
		t.Fatalf("archivo: %v, %v", got, err)
	}
}
