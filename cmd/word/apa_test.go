package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// convertirPrueba escribe los archivos en un directorio temporal y los
// convierte con las dos pasadas de armar (numeración y referencias).
func convertirPrueba(t *testing.T, archivos map[string]string, secciones ...string) *documento {
	t.Helper()
	dir := t.TempDir()
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d := convertirTodo(dir, secciones, 200, 4)
	if err := xmlBienFormado(envolver(d.b.String())); err != nil {
		t.Fatalf("XML mal formado: %v", err)
	}
	return d
}

const svgPrueba = `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200"></svg>`

// parrafosDe devuelve el texto visible de cada párrafo, en orden.
func parrafosDe(x string) []string {
	var out []string
	for _, p := range reParrafo.FindAllString(x, -1) {
		out = append(out, textoParrafo(p))
	}
	return out
}

// TestTablaAPA: «Tabla N» en negrita, el título en cursiva en la línea
// siguiente (los dos encima de la tabla), solo tres líneas horizontales y
// ningún relleno; la nota va debajo con «Nota.» en cursiva (APA 7, 7.8–7.14).
func TestTablaAPA(t *testing.T) {
	md := "Tabla: Tiempo por época según el número de nodos {#tab:epoca}\n\n" +
		"| Nodos | ms |\n|---|---:|\n| 1 | 72,3 |\n| 2 | 40,8 |\n| 3 | 25,7 |\n\n" +
		"Nota: Media recortada de 10 corridas. Fuente: `reports/x.json`.\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md}, "01-a.md")
	x := d.b.String()
	ps := parrafosDe(x)
	if ps[0] != "Tabla 1" || ps[1] != "Tiempo por época según el número de nodos" {
		t.Fatalf("pie de tabla: %q", ps[:2])
	}
	pie := reParrafo.FindAllString(x, 2)
	if !strings.Contains(pie[0], "<w:b/>") || !strings.Contains(pie[1], "<w:i/>") {
		t.Error("«Tabla N» va en negrita y el título en cursiva")
	}
	if !strings.Contains(pie[0], "<w:keepNext/>") || !strings.Contains(pie[1], "<w:keepNext/>") {
		t.Error("el pie debe quedar en la misma página que la tabla")
	}
	if strings.Contains(x, "w:fill=") {
		t.Error("una tabla APA no lleva relleno de celdas")
	}
	tabla := x[strings.Index(x, "<w:tbl>"):strings.Index(x, "</w:tbl>")]
	if n := strings.Count(tabla, `w:val="single"`); n != 2*3 {
		// encabezado: arriba y abajo en cada una de las 2 columnas; última fila: abajo
		t.Errorf("líneas de la tabla: %d; se esperaban 6 (3 reglas × 2 columnas)", n)
	}
	if strings.Contains(tabla, `<w:left w:val=`) || strings.Contains(tabla, `<w:right w:val=`) || strings.Contains(tabla, "<w:insideV") {
		t.Error("una tabla APA no tiene líneas verticales")
	}
	nota := ps[len(ps)-1]
	if !strings.HasPrefix(nota, "Nota. Media recortada") {
		t.Errorf("nota: %q", nota)
	}
	if d.etiquetas["tab:epoca"] != 1 || len(d.entradas) != 1 || d.entradas[0].Tipo != "Tabla" {
		t.Errorf("etiquetas %v, entradas %v", d.etiquetas, d.entradas)
	}
}

// TestFiguraAPA: número y título encima de la figura; las capturas comparten
// la numeración de las figuras.
func TestFiguraAPA(t *testing.T) {
	md := "![Speedup del clúster según el número de nodos](f.svg){#fig:speedup}\n\n" +
		"Nota: La línea discontinua es el speedup ideal.\n\n" +
		"<!-- captura: Panel de administración en vivo {#fig:admin} -->\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md, "f.svg": svgPrueba}, "01-a.md")
	x := d.b.String()
	ps := parrafosDe(x)
	if ps[0] != "Figura 1" || ps[1] != "Speedup del clúster según el número de nodos" {
		t.Fatalf("pie de figura: %q", ps[:2])
	}
	if i := strings.Index(x, "<w:drawing>"); i < strings.Index(x, "Speedup del clúster") {
		t.Error("el título va encima de la imagen")
	}
	if !strings.Contains(textoDe(x), "Nota. La línea discontinua") {
		t.Error("falta la nota de la figura")
	}
	if d.etiquetas["fig:speedup"] != 1 || d.etiquetas["fig:admin"] != 2 {
		t.Errorf("etiquetas: %v", d.etiquetas)
	}
	if !strings.Contains(x, "Panel de administración en vivo") || strings.Contains(x, "{#fig:admin}") {
		t.Error("la captura lleva su título sin la etiqueta")
	}
	if len(d.avisos) > 0 {
		t.Errorf("avisos: %v", d.avisos)
	}
}

// TestReferenciaCruzada: @fig:id y @tab:id dan el número definitivo aunque la
// figura esté en una sección posterior; una etiqueta desconocida da aviso.
func TestReferenciaCruzada(t *testing.T) {
	a := "Como muestra @fig:dos y la @tab:t, ver también @fig:nada.\n\n![Primera](f.svg)\n"
	b := "Tabla: Una tabla {#tab:t}\n\n| A |\n|---|\n| 1 |\n\n![Segunda](f.svg){#fig:dos}\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": a, "02-b.md": b, "f.svg": svgPrueba}, "01-a.md", "02-b.md")
	ps := parrafosDe(d.b.String())
	if ps[0] != "Como muestra Figura 2 y la Tabla 1, ver también @fig:nada." {
		t.Errorf("referencias: %q", ps[0])
	}
	if len(d.avisos) != 1 || !strings.Contains(d.avisos[0], "@fig:nada") {
		t.Errorf("avisos: %v", d.avisos)
	}
}

func TestEtiquetaRepetidaDaAviso(t *testing.T) {
	md := "![A](f.svg){#fig:x}\n\n![B](f.svg){#fig:x}\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md, "f.svg": svgPrueba}, "01-a.md")
	if len(d.avisos) != 1 || !strings.Contains(d.avisos[0], "fig:x") {
		t.Errorf("avisos: %v", d.avisos)
	}
}

// TestIndiceDeFigurasYTablas: cada entrada apunta con PAGEREF al marcador del
// pie, y el marcador existe en el cuerpo.
func TestIndiceDeFigurasYTablas(t *testing.T) {
	md := "Tabla: Métricas {#tab:m}\n\n| A |\n|---|\n| 1 |\n\n![Arquitectura *UML*](f.svg)\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md, "f.svg": svgPrueba}, "01-a.md")
	ind := indice(d.entradas)
	if err := xmlBienFormado(envolver(ind)); err != nil {
		t.Fatalf("índice mal formado: %v", err)
	}
	for _, s := range []string{"Índice de figuras", "Índice de tablas", "Figura 1. Arquitectura UML", "Tabla 1. Métricas"} {
		if !strings.Contains(textoDe(ind), s) {
			t.Errorf("el índice no tiene %q", s)
		}
	}
	reRef := regexp.MustCompile(`PAGEREF (\S+) \\h`)
	refs := reRef.FindAllStringSubmatch(ind, -1)
	if len(refs) != 2 {
		t.Fatalf("PAGEREF: %d; se esperaban 2", len(refs))
	}
	for _, r := range refs {
		if !strings.Contains(d.b.String(), `w:name="`+r[1]+`"`) {
			t.Errorf("el marcador %s no está en el cuerpo", r[1])
		}
	}
	// Sin figuras ni tablas no hay índices vacíos.
	if strings.Contains(indice(nil), "Índice de figuras") {
		t.Error("índice de figuras sin figuras")
	}
}

func textoDe(x string) string { return strings.Join(parrafosDe(x), "\n") }

// TestComentariosEnLinea: los valores generados <!--v:…-->…<!--/v--> se ven
// como texto normal, también al comienzo de una línea.
func TestComentariosEnLinea(t *testing.T) {
	md := "El AUC fue de <!--v:cuatro .AUC-->0,6789<!--/v--> y\n<!--v:miles .N-->1 070 000<!--/v--> casos.\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md}, "01-a.md")
	if ps := parrafosDe(d.b.String()); len(ps) != 1 || ps[0] != "El AUC fue de 0,6789 y 1 070 000 casos." {
		t.Errorf("párrafos: %q", ps)
	}
}

// TestCodigoSinRelleno: los listados van entre dos reglas finas, sin fondo.
func TestCodigoSinRelleno(t *testing.T) {
	d := convertirPrueba(t, map[string]string{"01-a.md": "```yaml\na: 1\nb: 2\n```\n"}, "01-a.md")
	x := d.b.String()
	if strings.Contains(x, "w:fill=") || !strings.Contains(x, "<w:pBdr><w:top") {
		t.Errorf("listado: %s", x)
	}
}

func TestQuitarParrafos(t *testing.T) {
	x := `<w:p><w:r><w:t>Jorge</w:t></w:r></w:p><w:p><w:r><w:t>Mar</w:t></w:r><w:r><w:t>co X</w:t></w:r></w:p><w:p><w:r><w:t>José</w:t></w:r></w:p>`
	y, faltan := quitarParrafos(x, []string{"Marco X", "Nadie"})
	if strings.Contains(y, "Mar") || !strings.Contains(y, "Jorge") || !strings.Contains(y, "José") {
		t.Errorf("resultado: %s", y)
	}
	if len(faltan) != 1 || faltan[0] != "Nadie" {
		t.Errorf("faltan: %v", faltan)
	}
}

// TestTextoPlanoEnCeldasEIndices (PR-96 H1): las celdas y los títulos se miden
// e indexan por lo que se ve en Word, no por el Markdown con valores en línea,
// enlaces o referencias; una barra dentro de <!--v:…--> no parte la celda.
func TestTextoPlanoEnCeldasEIndices(t *testing.T) {
	md := "Tabla: Resultados con <!--v:(index .C .U).Nodos-->5<!--/v--> nodos (ver @fig:f) {#tab:r}\n\n" +
		"| Métrica | Valor | Detalle |\n|---|---:|---|\n" +
		"| AUC | <!--v:.S.M.E.Test.AUC | cuatro-->0,6221<!--/v--> | [repositorio](https://github.com/JorgeGarcia005/cc65-tf-dengue/blob/main/reports/evaluacion.json) |\n\n" +
		"Nota: Fuente: `reports/evaluacion.json`.\n\n" +
		"![Speedup con <!--v:.N-->5<!--/v--> nodos](f.svg){#fig:f}\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md, "f.svg": svgPrueba}, "01-a.md")
	if len(d.avisos) > 0 {
		t.Fatalf("avisos: %v", d.avisos)
	}
	if got := d.entradas[0].Titulo; got != "Resultados con 5 nodos (ver Figura 1)" {
		t.Errorf("título en el índice: %q", got)
	}
	if got := d.entradas[1].Titulo; got != "Speedup con 5 nodos" {
		t.Errorf("título en el índice: %q", got)
	}
	x := d.b.String()
	if strings.Contains(x, "<!--") || strings.Contains(x, "&lt;!--") {
		t.Error("el marcado de los valores en línea llegó al documento")
	}
	if got := partirFila("| AUC | <!--v:.X | cuatro-->0,6<!--/v--> | b |"); len(got) != 3 {
		t.Errorf("celdas: %q", got)
	}
	// El valor (4 caracteres) y el enlace (su texto) no se llevan el ancho.
	tabla := x[strings.Index(x, "<w:tblGrid>"):strings.Index(x, "</w:tblGrid>")]
	var anchos []int
	for _, m := range regexp.MustCompile(`w:w="(\d+)"`).FindAllStringSubmatch(tabla, -1) {
		var n int
		fmt.Sscan(m[1], &n)
		anchos = append(anchos, n)
	}
	want := anchosColumnas([][]string{{"Métrica", "Valor", "Detalle"}, {"AUC", "0,6221", "repositorio"}}, 3, 20)
	if fmt.Sprint(anchos) != fmt.Sprint(want) {
		t.Errorf("anchos %v; con el texto visible serían %v: se midió el Markdown", anchos, want)
	}
	// La nota va pegada a la regla inferior de la tabla.
	if strings.Contains(x, `</w:tbl><w:p><w:pPr><w:spacing w:before="0" w:after="120"/>`) {
		t.Error("párrafo vacío entre la tabla y su nota")
	}
}

// TestNotaEnLaMismaPagina: la imagen o la captura seguida de una nota lleva
// keepNext, para que la nota no quede sola en la página siguiente.
func TestNotaEnLaMismaPagina(t *testing.T) {
	md := "![A](f.svg)\n\nNota: Uno.\n\n![B](f.svg)\n\n<!-- captura: C -->\n\nNota: Dos.\n"
	d := convertirPrueba(t, map[string]string{"01-a.md": md, "f.svg": svgPrueba}, "01-a.md")
	x := d.b.String()
	dibujos := regexp.MustCompile(`<w:p><w:pPr>(<w:keepNext/>)?[^>]*>[^<]*(?:<[^>]*>){0,3}<w:r><w:drawing>`).FindAllStringSubmatch(x, -1)
	if len(dibujos) != 2 || dibujos[0][1] == "" || dibujos[1][1] != "" {
		t.Errorf("keepNext de las imágenes: %q", dibujos)
	}
	if !strings.Contains(x, `<w:p><w:pPr><w:keepNext/><w:pBdr>`) {
		t.Error("la captura con nota no lleva keepNext")
	}
}

// pngPrueba: un PNG de w × h píxeles.
func pngPrueba(t *testing.T, w, h int) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestCapturaConArchivo: si la captura existe se inserta como figura PNG con
// su pie; si no, queda el espacio marcado y cuenta como pendiente. Las
// evidencias de texto, igual.
func TestCapturaConArchivo(t *testing.T) {
	md := "<!-- captura: Inicio de sesión {#fig:login} [capturas/01-login.png] -->\n\n" +
		"<!-- captura: Panel {#fig:panel} [capturas/02-panel.png] -->\n\n" +
		"<!-- evidencia: evidencias/ps.txt -->\n\n<!-- evidencia: evidencias/falta.txt -->\n\nVer @fig:panel.\n"
	dir := map[string]string{"01-a.md": md, "capturas/01-login.png": pngPrueba(t, 1280, 720), "evidencias/ps.txt": "NAME   STATUS\nmongo  healthy\n"}
	tmp := t.TempDir()
	for n, c := range dir {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(tmp, n)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, n), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d := convertirTodo(tmp, []string{"01-a.md"}, 200, 4)
	x := d.b.String()
	if err := xmlBienFormado(envolver(x)); err != nil {
		t.Fatal(err)
	}
	if len(d.imagenes) != 1 || d.imagenes[0].PNG == nil || d.imagenes[0].SVG != nil {
		t.Fatalf("imágenes: %+v", d.imagenes)
	}
	if !strings.Contains(x, `name="figura1.png"`) || strings.Contains(x, "svgBlip") {
		t.Error("la captura PNG no debe llevar SVG")
	}
	if !strings.Contains(textoDe(x), "mongo  healthy") || !strings.Contains(textoDe(x), "Ver Figura 2.") {
		t.Errorf("texto: %s", textoDe(x))
	}
	if len(d.pendientes) != 2 || !strings.Contains(d.pendientes[0], "capturas/02-panel.png") || d.pendientes[1] != "evidencias/falta.txt" {
		t.Errorf("pendientes: %q", d.pendientes)
	}
	if len(d.avisos) > 0 {
		t.Errorf("avisos: %v", d.avisos)
	}
}

// TestAltoMaximo (PR-98 H1): una imagen muy alta se reduce entera para que no
// pase de 20 cm, sin deformarse.
func TestAltoMaximo(t *testing.T) {
	cx, cy := extension(400, 2000)
	if cy != altoMaximo || cx*2000 != cy*400 {
		t.Errorf("cx=%d cy=%d", cx, cy)
	}
	if cx, _ := extension(800, 400); cx != int64(anchoTexto)*emuPorDxa {
		t.Errorf("una imagen apaisada va al ancho del texto: %d", cx)
	}
}
