package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plantillaMinima arma un paquete como el de la carátula oficial de UPC (#24):
// <w:sectPr> con atributos y un <w:document> que solo declara w y r.
func plantillaMinima(raiz string) *paquete {
	p := &paquete{partes: map[string][]byte{}}
	p.poner("[Content_Types].xml", []byte(`<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`))
	p.poner("word/document.xml", []byte(`<?xml version="1.0" encoding="UTF-8"?>`+raiz+
		`<w:body><w:p><w:r><w:t>TB1</w:t></w:r></w:p>`+
		`<w:p><w:r><w:t>Indice</w:t></w:r></w:p>`+
		`<w:sectPr w:rsidR="00AB12CD" w:rsidSect="00AB12CD"><w:pgSz w:w="11906" w:h="16838"/></w:sectPr>`+
		`</w:body></w:document>`))
	p.poner("word/_rels/document.xml.rels", []byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`))
	p.poner("word/numbering.xml", []byte(`<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`+
		`<w:abstractNum w:abstractNumId="0"><w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/></w:lvl></w:abstractNum>`+
		`<w:abstractNum w:abstractNumId="1"><w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/></w:lvl></w:abstractNum>`+
		`<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num><w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num></w:numbering>`))
	p.poner("word/settings.xml", []byte(`<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:compat/></w:settings>`))
	return p
}

const raizSoloWR = `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" w:conformance="transitional">`

// generar arma el .docx con una sección que tiene figura, ecuaciones y
// enlace (usa w, m, r, wp, a, pic y asvg). Corre desde la carpeta de la
// sección con ruta relativa, como main, y exige cero avisos: si el Markdown
// no se carga, el test falla (PR-47 H2).
func generar(t *testing.T, raiz string, cambios reemplazos) (string, error) {
	t.Helper()
	dir := t.TempDir()
	md := "# 1. Prueba\n\nTexto con [enlace](https://x.pe) y $x^2$.\n\n$$\\frac{1}{S}\\sum_i s_i$$\n\n![Figura](f.svg)\n"
	if err := os.WriteFile(filepath.Join(dir, "01-x.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200"></svg>`
	if err := os.WriteFile(filepath.Join(dir, "f.svg"), []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	docx, avisos, err := armar(plantillaMinima(raiz), ".", []string{"01-x.md"}, cambios)
	if err != nil {
		return "", err
	}
	if len(avisos) > 0 {
		t.Fatalf("avisos de armar: %v", avisos)
	}
	return documentoGenerado(t, docx), nil
}

func documentoGenerado(t *testing.T, docx []byte) string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			b, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
	}
	t.Fatal("el .docx no tiene word/document.xml")
	return ""
}

// exigirContenido cuenta elementos por nombre expandido (URI + nombre local):
// la figura, las ecuaciones y el enlace tienen que estar, y en el vocabulario
// de Word, no solo con el prefijo correcto (PR-47 H2 y H3).
func exigirContenido(t *testing.T, doc string) {
	t.Helper()
	ns := map[string]string{}
	for _, n := range nsDocumento {
		ns[n[0]] = n[1]
	}
	quiere := map[xml.Name]int{
		{Space: ns["pic"], Local: "pic"}:      1,
		{Space: ns["a"], Local: "graphic"}:    1,
		{Space: ns["wp"], Local: "inline"}:    1,
		{Space: ns["asvg"], Local: "svgBlip"}: 1,
		{Space: ns["m"], Local: "oMath"}:      2,
		{Space: ns["w"], Local: "hyperlink"}:  1,
	}
	hay := map[xml.Name]int{}
	dec := xml.NewDecoder(strings.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("document.xml: %v", err)
		}
		if e, ok := tok.(xml.StartElement); ok {
			hay[e.Name]++
		}
	}
	for n, c := range quiere {
		if hay[n] != c {
			t.Errorf("{%s}%s: %d elementos, quiere %d", n.Space, n.Local, hay[n], c)
		}
	}
}

// raizDe devuelve la etiqueta de <w:document> tal como está escrita.
func raizDe(t *testing.T, doc string) xml.StartElement {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(doc))
	for {
		tok, err := dec.RawToken()
		if err != nil {
			t.Fatalf("sin <w:document>: %v", err)
		}
		if e, ok := tok.(xml.StartElement); ok {
			return e
		}
	}
}

func declaraciones(e xml.StartElement, prefijo string) []string {
	var uris []string
	for _, a := range e.Attr {
		if a.Name.Space == "xmlns" && a.Name.Local == prefijo {
			uris = append(uris, a.Value)
		}
	}
	return uris
}

// TestPlantillaConSectPrConAtributos: issue #24, falla 1.
func TestPlantillaConSectPrConAtributos(t *testing.T) {
	doc, err := generar(t, raizSoloWR, reemplazos{{"TB1", "PC4"}})
	if err != nil {
		t.Fatalf("armar: %v", err)
	}
	if n := strings.Count(doc, "<w:sectPr"); n != 1 {
		t.Errorf("%d <w:sectPr> en el documento, quiere 1 (el de la plantilla, al final)", n)
	}
	if !strings.Contains(doc, `<w:sectPr w:rsidR="00AB12CD"`) {
		t.Error("se perdió el <w:sectPr> de la plantilla con sus atributos")
	}
	exigirContenido(t, doc)
}

// TestPlantillaSinNamespacesDeFiguras: issue #24, falla 2. El documento
// generado declara cada prefijo que usa, una sola vez, con su URI.
func TestPlantillaSinNamespacesDeFiguras(t *testing.T) {
	doc, err := generar(t, raizSoloWR, nil)
	if err != nil {
		t.Fatalf("armar: %v", err)
	}
	if err := xmlBienFormado(doc); err != nil {
		t.Fatalf("document.xml: %v", err)
	}
	raiz := raizDe(t, doc)
	for _, ns := range nsDocumento {
		if uris := declaraciones(raiz, ns[0]); len(uris) != 1 || uris[0] != ns[1] {
			t.Errorf("xmlns:%s en <w:document>: %v, quiere [%s]", ns[0], uris, ns[1])
		}
	}
	if len(raiz.Attr) != len(nsDocumento)+1 {
		t.Errorf("<w:document> con %d atributos, quiere %d (los de la plantilla y los agregados)", len(raiz.Attr), len(nsDocumento)+1)
	}
	exigirContenido(t, doc)
}

// TestRaizConMayorQueEnUnAtributo: un '>' entre comillas es válido en XML y
// no cierra la etiqueta (PR-47 H1). La raíz conserva sus atributos.
func TestRaizConMayorQueEnUnAtributo(t *testing.T) {
	raiz := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" xmlns:upc="urn:plantilla:upc" ` +
		`mc:Ignorable="upc" upc:nota="Nivel > 1">`
	doc, err := generar(t, raiz, nil)
	if err != nil {
		t.Fatalf("armar: %v", err)
	}
	r := raizDe(t, doc)
	var nota string
	for _, a := range r.Attr {
		if a.Name.Space == "upc" && a.Name.Local == "nota" {
			nota = a.Value
		}
	}
	if nota != "Nivel > 1" {
		t.Errorf("upc:nota = %q, quiere %q", nota, "Nivel > 1")
	}
	exigirContenido(t, doc)
}

// TestPrefijoDeLaPlantillaConOtroURI: si la plantilla usa pic (u otro prefijo
// del contenido) para otro vocabulario, la figura cambiaría de significado
// sin que nada falle. armar se niega y dice qué prefijo choca (PR-47 H3).
func TestPrefijoDeLaPlantillaConOtroURI(t *testing.T) {
	raiz := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:pic="urn:plantilla:portada">`
	_, err := generar(t, raiz, nil)
	if err == nil {
		t.Fatal("armar aceptó xmlns:pic con otro URI")
	}
	for _, quiere := range []string{"xmlns:pic", "urn:plantilla:portada", nsPic} {
		if !strings.Contains(err.Error(), quiere) {
			t.Errorf("el error no menciona %q: %v", quiere, err)
		}
	}
}

const nsPic = "http://schemas.openxmlformats.org/drawingml/2006/picture"

// TestXMLBienFormadoRechazaPrefijosSinDeclarar: encoding/xml acepta un
// prefijo sin declarar; LibreOffice no («unbound prefix»). La comprobación de
// armar tiene que detectarlo.
func TestXMLBienFormadoRechazaPrefijosSinDeclarar(t *testing.T) {
	malos := []string{
		`<w:document xmlns:w="urn:w"><a:graphic/></w:document>`,
		`<w:document xmlns:w="urn:w"><w:p pic:x="1"/></w:document>`,
		`<w:document xmlns:w="urn:w"><w:p xmlns:a="urn:a"/><a:x/></w:document>`, // fuera de su ámbito
	}
	for _, s := range malos {
		if err := xmlBienFormado(s); err == nil {
			t.Errorf("aceptó %s", s)
		}
	}
	buenos := []string{
		`<w:document xmlns:w="urn:w"><w:t xml:space="preserve">x</w:t></w:document>`,
		`<w:document xmlns:w="urn:w"><w:p><x:y xmlns:x="urn:x"/></w:p></w:document>`,
		`<Types xmlns="urn:t"><Default Extension="svg"/></Types>`,
	}
	for _, s := range buenos {
		if err := xmlBienFormado(s); err != nil {
			t.Errorf("rechazó %s: %v", s, err)
		}
	}
}
