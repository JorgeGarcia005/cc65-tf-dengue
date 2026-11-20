package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
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

// seccionConTodo escribe una sección con figura, ecuación y enlace: usa los
// prefijos w, m, r, wp, a, pic y asvg.
func seccionConTodo(t *testing.T) string {
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
	return filepath.Join(dir, "01-x.md")
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

// TestPlantillaConSectPrConAtributos: issue #24, falla 1.
func TestPlantillaConSectPrConAtributos(t *testing.T) {
	docx, _, err := armar(plantillaMinima(raizSoloWR), ".", []string{seccionConTodo(t)}, reemplazos{{"TB1", "PC4"}})
	if err != nil {
		t.Fatalf("armar: %v", err)
	}
	doc := documentoGenerado(t, docx)
	if n := strings.Count(doc, "<w:sectPr"); n != 1 {
		t.Errorf("%d <w:sectPr> en el documento, quiere 1 (el de la plantilla, al final)", n)
	}
	if !strings.Contains(doc, `<w:sectPr w:rsidR="00AB12CD"`) {
		t.Error("se perdió el <w:sectPr> de la plantilla con sus atributos")
	}
}

// TestPlantillaSinNamespacesDeFiguras: issue #24, falla 2. El documento
// generado declara cada prefijo que usa, y no duplica los que ya traía.
func TestPlantillaSinNamespacesDeFiguras(t *testing.T) {
	docx, _, err := armar(plantillaMinima(raizSoloWR), ".", []string{seccionConTodo(t)}, nil)
	if err != nil {
		t.Fatalf("armar: %v", err)
	}
	doc := documentoGenerado(t, docx)
	if err := xmlBienFormado(doc); err != nil {
		t.Fatalf("document.xml: %v", err)
	}
	raiz := doc[strings.Index(doc, "<w:document"):]
	raiz = raiz[:strings.Index(raiz, ">")]
	for _, ns := range nsDocumento {
		if n := len(regexp.MustCompile(`\sxmlns:`+ns[0]+`=`).FindAllString(raiz, -1)); n != 1 {
			t.Errorf("xmlns:%s declarado %d veces en <w:document>, quiere 1", ns[0], n)
		}
	}
	if !strings.Contains(raiz, `w:conformance="transitional"`) {
		t.Error("se perdieron los atributos originales de <w:document>")
	}
}

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
