// Command word arma el informe en Word a partir de las secciones Markdown de
// docs/informe/<entregable>/ y de una plantilla .docx (el informe anterior del
// equipo). De la plantilla conserva la carátula, los estilos, la numeración y
// el pie de página; el resto del cuerpo se reemplaza por el contenido nuevo.
//
// Así el Word también se genera desde el repositorio: si cambian los datos se
// corre go run ./cmd/informe y luego este comando (AGENTS.md §3.6).
//
// Uso:
//
//	go run ./cmd/word -plantilla entregables/plantilla.docx
//
// Los códigos de alumno (un .docx por integrante) se leen de
// entregables/codigos.txt (uno por línea; la carpeta está en .gitignore) o de
// -codigos U20XXXXXXX,U20YYYYYYY. No se versionan: el repositorio es público.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func main() {
	plantilla := flag.String("plantilla", "entregables/plantilla.docx", "informe anterior (.docx) del que se toman carátula y estilos")
	fuentes := flag.String("fuentes", "docs/informe/pc3", "carpeta con las secciones Markdown")
	salida := flag.String("salida", "entregables", "carpeta de salida")
	entregable := flag.String("entregable", "PC3", "código del entregable (PC3, PC4, TB2)")
	// El enunciado (CC65-TB2-1ACC0065_202620, "Detalle de Entrega") fija el nombre
	// 1ACC0065_YYY_202610_Informe_UXXXXXXXX aunque el ciclo sea 2026-20: se respeta tal cual.
	ciclo := flag.String("ciclo", "202610", "código del nombre de archivo que exige el enunciado")
	titulo := flag.String("titulo", "Riesgo de dengue con signos de alarma: regresión logística distribuida en Go", "título de la carátula")
	tituloViejo := flag.String("titulo-plantilla", "SmartTransit: Predicción Concurrente de Demanda de Transporte", "título de la carátula de la plantilla")
	entregableViejo := flag.String("entregable-plantilla", "TB1", "código de entregable que figura en la plantilla")
	codigos := flag.String("codigos", "", "códigos de alumno separados por coma (por defecto, <salida>/codigos.txt)")
	flag.Parse()

	secciones, err := filepath.Glob(filepath.Join(*fuentes, "[0-9][0-9]-*.md"))
	if err != nil || len(secciones) == 0 {
		log.Fatalf("no hay secciones NN-*.md en %s", *fuentes)
	}
	slices.Sort(secciones)

	p, err := leerPaquete(*plantilla)
	if err != nil {
		log.Fatalf("plantilla: %v", err)
	}
	docx, avisos, err := armar(p, *fuentes, secciones, reemplazos{
		{*tituloViejo, *titulo},
		{*entregableViejo, *entregable},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range avisos {
		log.Printf("aviso: %s", a)
	}
	if err := os.MkdirAll(*salida, 0o755); err != nil {
		log.Fatal(err)
	}
	lista, err := leerCodigos(*codigos, filepath.Join(*salida, "codigos.txt"))
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range lista {
		nombre := fmt.Sprintf("1ACC0065_%s_%s_Informe_%s.docx", *entregable, *ciclo, c)
		ruta := filepath.Join(*salida, nombre)
		if err := os.WriteFile(ruta, docx, 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("escrito %s (%d KB, %d secciones)", ruta, len(docx)/1024, len(secciones))
	}
}

type reemplazos [][2]string

// paquete es el .docx como lista ordenada de partes.
type paquete struct {
	nombres []string
	partes  map[string][]byte
}

func leerPaquete(ruta string) (*paquete, error) {
	z, err := zip.OpenReader(ruta)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	p := &paquete{partes: map[string][]byte{}}
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		p.nombres = append(p.nombres, f.Name)
		p.partes[f.Name] = b
	}
	return p, nil
}

func (p *paquete) poner(nombre string, b []byte) {
	if _, ok := p.partes[nombre]; !ok {
		p.nombres = append(p.nombres, nombre)
	}
	p.partes[nombre] = b
}

func (p *paquete) quitar(nombre string) {
	delete(p.partes, nombre)
	p.nombres = slices.DeleteFunc(p.nombres, func(n string) bool { return n == nombre })
}

func (p *paquete) bytes() ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	// [Content_Types].xml va primero por convención.
	orden := slices.Clone(p.nombres)
	slices.SortStableFunc(orden, func(a, b string) int {
		switch {
		case a == "[Content_Types].xml":
			return -1
		case b == "[Content_Types].xml":
			return 1
		}
		return 0
	})
	for _, n := range orden {
		f, err := w.Create(n)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(p.partes[n]); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var (
	reParrafo   = regexp.MustCompile(`(?s)<w:p[ >].*?</w:p>`)
	reTextoRun  = regexp.MustCompile(`(?s)<w:t(?: [^>]*)?>([^<]*)</w:t>`)
	reSectPr    = regexp.MustCompile(`(?s)<w:sectPr>.*?</w:sectPr>`)
	reEmbed     = regexp.MustCompile(`r:(?:embed|id)="([^"]+)"`)
	reRelacion  = regexp.MustCompile(`<Relationship [^>]*/>`)
	reAtributo  = func(nombre string) *regexp.Regexp { return regexp.MustCompile(nombre + `="([^"]*)"`) }
	reNumID     = regexp.MustCompile(`<w:num w:numId="(\d+)"[^>]*>\s*<w:abstractNumId w:val="(\d+)"`)
	reAbstracto = regexp.MustCompile(`(?s)<w:abstractNum [^>]*w:abstractNumId="(\d+)".*?</w:abstractNum>`)
)

// armar construye el .docx completo.
func armar(p *paquete, dir string, secciones []string, cambios reemplazos) ([]byte, []string, error) {
	doc := string(p.partes["word/document.xml"])
	ini := strings.Index(doc, "<w:body>")
	fin := strings.LastIndex(doc, "</w:body>")
	if ini < 0 || fin < 0 {
		return nil, nil, fmt.Errorf("plantilla sin <w:body>")
	}
	cuerpo := doc[ini+len("<w:body>") : fin]

	// Carátula: todo lo que está antes del párrafo del índice.
	var caratula string
	for _, m := range reParrafo.FindAllStringIndex(cuerpo, -1) {
		if t := textoParrafo(cuerpo[m[0]:m[1]]); strings.HasPrefix(strings.TrimSpace(t), "Indice") || strings.HasPrefix(strings.TrimSpace(t), "Índice") {
			caratula = cuerpo[:m[0]]
			break
		}
	}
	if caratula == "" {
		return nil, nil, fmt.Errorf("no se encontró el párrafo «Indice» que cierra la carátula")
	}
	for _, c := range cambios {
		var ok bool
		caratula, ok = reemplazarParrafo(caratula, c[0], c[1])
		if !ok {
			return nil, nil, fmt.Errorf("la carátula no tiene el texto %q", c[0])
		}
	}

	// Sección de página: la primera de la plantilla es la que lleva el pie con
	// el número de página.
	sect := reSectPr.FindString(cuerpo)
	if sect == "" {
		return nil, nil, fmt.Errorf("plantilla sin <w:sectPr>")
	}
	caratula = reSectPr.ReplaceAllString(caratula, "")

	numBase, bulletID, decimal, err := numeracion(string(p.partes["word/numbering.xml"]))
	if err != nil {
		return nil, nil, err
	}
	d := nuevoDocumento(".", numBase, bulletID)
	for _, s := range secciones {
		d.Convertir(s)
	}

	var b strings.Builder
	b.WriteString(doc[:ini+len("<w:body>")])
	b.WriteString(caratula)
	b.WriteString(indice())
	b.WriteString(d.b.String())
	b.WriteString(sect)
	b.WriteString(doc[fin:])
	nuevo := b.String()
	if err := xmlBienFormado(nuevo); err != nil {
		return nil, nil, fmt.Errorf("document.xml mal formado: %w", err)
	}
	p.poner("word/document.xml", []byte(nuevo))

	// Relaciones: se conservan las que no son imágenes ni enlaces, y las
	// imágenes de la carátula; se agregan las figuras y los enlaces nuevos.
	usados := map[string]bool{}
	for _, m := range reEmbed.FindAllStringSubmatch(caratula+sect, -1) {
		usados[m[1]] = true
	}
	rels := string(p.partes["word/_rels/document.xml.rels"])
	var conservar []string
	for _, r := range reRelacion.FindAllString(rels, -1) {
		id := reAtributo("Id").FindStringSubmatch(r)[1]
		tipo := reAtributo("Type").FindStringSubmatch(r)[1]
		destino := reAtributo("Target").FindStringSubmatch(r)[1]
		esRecurso := strings.HasSuffix(tipo, "/image") || strings.HasSuffix(tipo, "/hyperlink")
		if esRecurso && !usados[id] {
			if strings.HasSuffix(tipo, "/image") {
				p.quitar("word/" + destino)
			}
			continue
		}
		conservar = append(conservar, r)
	}
	png1 := pngTransparente()
	for _, img := range d.imagenes {
		p.poner("word/media/"+img.Nombre+".svg", img.SVG)
		p.poner("word/media/"+img.Nombre+".png", png1)
		conservar = append(conservar,
			fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s.svg"/>`, img.RelSVG, img.Nombre),
			fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s.png"/>`, img.RelPNG, img.Nombre))
	}
	for _, e := range d.orden {
		conservar = append(conservar, fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`, e.Rel, escapar(e.URL)))
	}
	cab := rels[:strings.Index(rels, "<Relationship ")]
	p.poner("word/_rels/document.xml.rels", []byte(cab+strings.Join(conservar, "")+"</Relationships>"))

	// Tipos de contenido: SVG.
	ct := string(p.partes["[Content_Types].xml"])
	if !strings.Contains(ct, `Extension="svg"`) {
		ct = strings.Replace(ct, "<Default ", `<Default ContentType="image/svg+xml" Extension="svg"/><Default `, 1)
	}
	p.poner("[Content_Types].xml", []byte(ct))

	// Numeración: una lista numerada nueva por cada lista, para que reinicie en 1.
	num := string(p.partes["word/numbering.xml"])
	var nums strings.Builder
	for _, id := range d.numIds {
		fmt.Fprintf(&nums, `<w:num w:numId="%d"><w:abstractNumId w:val="%d"/><w:lvlOverride w:ilvl="0"><w:startOverride w:val="1"/></w:lvlOverride></w:num>`, id, decimal)
	}
	num = strings.Replace(num, "</w:numbering>", nums.String()+"</w:numbering>", 1)
	p.poner("word/numbering.xml", []byte(num))

	// Ajustes: que Word actualice el índice al abrir el documento, y sin
	// incrustar fuentes estándar (Times New Roman) al guardar: eso multiplica
	// el tamaño del archivo.
	set := string(p.partes["word/settings.xml"])
	set = regexp.MustCompile(`<w:embedTrueTypeFonts[^>]*/>`).ReplaceAllString(set, "")
	if !strings.Contains(set, "<w:updateFields") {
		if i := strings.Index(set, "<w:compat>"); i >= 0 {
			set = set[:i] + `<w:updateFields w:val="true"/>` + set[i:]
		}
	}
	p.poner("word/settings.xml", []byte(set))

	out, err := p.bytes()
	return out, d.avisos, err
}

// numeracion devuelve el primer numId libre, el numId de viñetas y un
// abstractNumId decimal de la plantilla.
func numeracion(num string) (libre, viñetas, decimal int, err error) {
	formato := map[string]string{}
	for _, m := range reAbstracto.FindAllStringSubmatch(num, -1) {
		lvl0 := regexp.MustCompile(`(?s)<w:lvl w:ilvl="0".*?</w:lvl>`).FindString(m[0])
		if f := regexp.MustCompile(`<w:numFmt w:val="(\w+)"`).FindStringSubmatch(lvl0); f != nil {
			formato[m[1]] = f[1]
		}
	}
	viñetas, decimal = -1, -1
	for _, m := range reNumID.FindAllStringSubmatch(num, -1) {
		var id, abs int
		fmt.Sscan(m[1], &id)
		fmt.Sscan(m[2], &abs)
		libre = max(libre, id+1)
		switch formato[m[2]] {
		case "bullet":
			if viñetas < 0 {
				viñetas = id
			}
		case "decimal":
			if decimal < 0 {
				decimal = abs
			}
		}
	}
	if viñetas < 0 || decimal < 0 {
		return 0, 0, 0, fmt.Errorf("la plantilla no tiene listas con viñetas y numeradas")
	}
	return libre + 100, viñetas, decimal, nil
}

func textoParrafo(p string) string {
	var b strings.Builder
	for _, m := range reTextoRun.FindAllStringSubmatch(p, -1) {
		b.WriteString(m[1])
	}
	return b.String()
}

// reemplazarParrafo cambia el texto del primer párrafo cuyo texto (unido entre
// corridas) contiene viejo. El texto nuevo queda en la primera corrida, con su
// formato, y las demás se vacían.
func reemplazarParrafo(xmlStr, viejo, nuevo string) (string, bool) {
	for _, m := range reParrafo.FindAllStringIndex(xmlStr, -1) {
		par := xmlStr[m[0]:m[1]]
		t := textoParrafo(par)
		if !strings.Contains(t, viejo) {
			continue
		}
		reemplazo := escapar(strings.Replace(t, viejo, nuevo, 1))
		primero := true
		par2 := reTextoRun.ReplaceAllStringFunc(par, func(run string) string {
			apertura := run[:strings.Index(run, ">")+1]
			if primero {
				primero = false
				return apertura + reemplazo + "</w:t>"
			}
			return apertura + "</w:t>"
		})
		return xmlStr[:m[0]] + par2 + xmlStr[m[1]:], true
	}
	return xmlStr, false
}

// indice inserta el título y un campo TOC que Word completa al abrir. El
// título lleva "salto de página antes" y no un salto explícito: si la carátula
// ya llena la primera página, un salto explícito dejaría una página en blanco.
func indice() string {
	e := estilo{Fuente: fuenteTexto, Negrita: true, Tam: 32}
	return `<w:p><w:pPr><w:pageBreakBefore/><w:spacing w:after="240"/></w:pPr><w:r><w:rPr>` + e.rPr() + `</w:rPr><w:t>Índice</w:t></w:r></w:p>` +
		`<w:p><w:r><w:fldChar w:fldCharType="begin" w:dirty="true"/></w:r>` +
		`<w:r><w:instrText xml:space="preserve"> TOC \o "1-2" \h \z \u </w:instrText></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r>` +
		`<w:r><w:rPr>` + texto(22).rPr() + `</w:rPr><w:t>Abra el documento en Word y acepte actualizar los campos para generar el índice.</w:t></w:r>` +
		`<w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>` +
		`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`
}

// pngTransparente es la imagen de respaldo obligatoria junto a cada SVG; Word
// 2016 o posterior muestra el SVG.
func pngTransparente() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{255, 255, 255, 0})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func xmlBienFormado(s string) error {
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		if _, err := dec.Token(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// leerCodigos devuelve los códigos de alumno del flag (separados por coma) o,
// si viene vacío, del archivo (uno por línea; ignora líneas vacías y las que
// empiezan con #). Sin códigos no hay a quién nombrar el .docx: es un error.
func leerCodigos(flagCodigos, archivo string) ([]string, error) {
	texto := strings.ReplaceAll(flagCodigos, ",", "\n")
	if strings.TrimSpace(texto) == "" {
		b, err := os.ReadFile(archivo)
		if err != nil {
			return nil, fmt.Errorf("faltan los códigos de alumno: usa -codigos o crea %s (uno por línea): %w", archivo, err)
		}
		texto = string(b)
	}
	var out []string
	for _, l := range strings.Split(texto, "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("faltan los códigos de alumno en -codigos o en %s", archivo)
	}
	return out, nil
}
