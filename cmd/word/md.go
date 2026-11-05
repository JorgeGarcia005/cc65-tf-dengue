package main

// Conversión del subconjunto de Markdown que usa el informe a
// WordprocessingML: títulos, párrafos, listas, tablas, bloques de código,
// citas, imágenes, ecuaciones ($…$ y $$…$$), enlaces e inclusiones.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	fuenteTexto  = "Times New Roman"
	fuenteCodigo = "Consolas"
	anchoTexto   = 9122 // dxa: ancho de la página carta menos los márgenes de la plantilla
	emuPorDxa    = 635
)

// estilo describe las propiedades de una corrida de texto. rPr las escribe
// una sola vez cada una y en el orden que exige el esquema (CT_RPr).
type estilo struct {
	Fuente     string
	Negrita    bool
	Cursiva    bool
	SinCursiva bool // anula una cursiva heredada del estilo de párrafo
	Color      string
	Tam        int // medios puntos
	Subrayado  bool
	Sombreado  string
}

func (e estilo) rPr() string {
	var b strings.Builder
	if e.Fuente != "" {
		fmt.Fprintf(&b, `<w:rFonts w:ascii="%[1]s" w:hAnsi="%[1]s" w:cs="%[1]s" w:eastAsia="%[1]s"/>`, e.Fuente)
	}
	if e.Negrita {
		b.WriteString(`<w:b/><w:bCs/>`)
	}
	if e.Cursiva {
		b.WriteString(`<w:i/><w:iCs/>`)
	} else if e.SinCursiva {
		b.WriteString(`<w:i w:val="0"/><w:iCs w:val="0"/>`)
	}
	if e.Color != "" {
		fmt.Fprintf(&b, `<w:color w:val="%s"/>`, e.Color)
	}
	if e.Tam > 0 {
		fmt.Fprintf(&b, `<w:sz w:val="%[1]d"/><w:szCs w:val="%[1]d"/>`, e.Tam)
	}
	if e.Subrayado {
		b.WriteString(`<w:u w:val="single"/>`)
	}
	if e.Sombreado != "" {
		fmt.Fprintf(&b, `<w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, e.Sombreado)
	}
	return b.String()
}

func texto(tam int) estilo { return estilo{Fuente: fuenteTexto, Tam: tam} }

// Recursos que el documento necesita además del XML del cuerpo.
type imagen struct {
	SVG    []byte
	Nombre string // base sin extensión, p. ej. "figura3"
	RelSVG string
	RelPNG string
}

type enlace struct {
	URL string
	Rel string
}

// documento acumula el cuerpo y los recursos mientras se convierte.
type documento struct {
	b        strings.Builder
	imagenes []imagen
	enlaces  map[string]string // url → rId
	orden    []enlace
	numIds   []int // listas numeradas creadas (cada una reinicia en 1)
	figuras  int
	docPr    int
	numBase  int // primer numId libre en numbering.xml
	bulletID int // numId de viñetas de la plantilla
	dir      string
	avisos   []string
	// referencias activa el formato APA 7 de la lista de referencias
	// (directiva <!-- estilo: referencias --> en el Markdown).
	referencias bool
}

func nuevoDocumento(dir string, numBase, bulletID int) *documento {
	return &documento{enlaces: map[string]string{}, docPr: 1000, numBase: numBase, bulletID: bulletID, dir: dir}
}

var (
	reImagen   = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)]+)\)\s*$`)
	reIncluir  = regexp.MustCompile(`^<!--\s*incluir(-codigo)?:\s*(\S+)\s*-->\s*$`)
	reNumerada = regexp.MustCompile(`^(\d+)\.\s+(.*)$`)
	reTitulo   = regexp.MustCompile(`^(#{1,4})\s+(.*)$`)
	reEstilo   = regexp.MustCompile(`^<!--\s*estilo:\s*(\S+)\s*-->\s*$`)
	reCaptura  = regexp.MustCompile(`^<!--\s*captura:\s*(.+?)\s*-->\s*$`)
)

// Convertir agrega al cuerpo el Markdown del archivo (ruta relativa a d.dir).
func (d *documento) Convertir(ruta string) {
	b, err := os.ReadFile(filepath.Join(d.dir, ruta))
	if err != nil {
		d.aviso("no se pudo leer %s: %v", ruta, err)
		return
	}
	d.bloques(lineasDe(b), filepath.Dir(ruta))
}

func lineasDe(b []byte) []string {
	return strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
}

func (d *documento) aviso(f string, a ...any) { d.avisos = append(d.avisos, fmt.Sprintf(f, a...)) }

func (d *documento) bloques(lineas []string, base string) {
	for i := 0; i < len(lineas); {
		t := strings.TrimSpace(lineas[i])
		switch {
		case t == "":
			i++
		case reIncluir.MatchString(t):
			m := reIncluir.FindStringSubmatch(t)
			ruta := filepath.Join(base, m[2])
			if m[1] == "" {
				d.Convertir(ruta)
			} else if c, err := os.ReadFile(filepath.Join(d.dir, ruta)); err == nil {
				d.codigo(lineasDe([]byte(strings.TrimRight(string(c), "\r\n"))))
			} else {
				d.aviso("no se pudo incluir %s: %v", ruta, err)
			}
			i++
		case reCaptura.MatchString(t):
			d.captura(reCaptura.FindStringSubmatch(t)[1])
			i++
		case reEstilo.MatchString(t):
			d.referencias = reEstilo.FindStringSubmatch(t)[1] == "referencias"
			i++
		case strings.HasPrefix(t, "<!--"):
			for i < len(lineas) && !strings.Contains(lineas[i], "-->") {
				i++
			}
			i++
		case strings.HasPrefix(t, "```"):
			lang := strings.TrimPrefix(t, "```")
			j := i + 1
			for j < len(lineas) && !strings.HasPrefix(strings.TrimSpace(lineas[j]), "```") {
				j++
			}
			if lang == "mermaid" {
				d.aviso("se omitió un bloque mermaid (usar una figura SVG)")
			} else {
				d.codigo(lineas[i+1 : min(j, len(lineas))])
			}
			i = j + 1
		case strings.HasPrefix(t, "$$"):
			i = d.ecuacion(lineas, i)
		case reTitulo.MatchString(t):
			m := reTitulo.FindStringSubmatch(t)
			d.titulo(len(m[1]), m[2])
			i++
		case reImagen.MatchString(t):
			m := reImagen.FindStringSubmatch(t)
			d.figura(m[1], filepath.Join(base, m[2]))
			i++
		case strings.HasPrefix(t, "|"):
			j := i
			for j < len(lineas) && strings.HasPrefix(strings.TrimSpace(lineas[j]), "|") {
				j++
			}
			d.tabla(lineas[i:j])
			i = j
		case strings.HasPrefix(t, ">"):
			var partes []string
			for i < len(lineas) && strings.HasPrefix(strings.TrimSpace(lineas[i]), ">") {
				partes = append(partes, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lineas[i]), ">")))
				i++
			}
			d.cita(strings.Join(partes, " "))
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || reNumerada.MatchString(t):
			i = d.lista(lineas, i)
		default:
			var partes []string
			for i < len(lineas) {
				s := strings.TrimSpace(lineas[i])
				if s == "" || strings.HasPrefix(s, "|") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "```") ||
					strings.HasPrefix(s, "$$") || strings.HasPrefix(s, "<!--") || reImagen.MatchString(s) ||
					(len(partes) > 0 && (strings.HasPrefix(s, "- ") || reNumerada.MatchString(s))) {
					break
				}
				partes = append(partes, s)
				i++
			}
			ppr := `<w:jc w:val="both"/>`
			if d.referencias { // APA 7: sangría francesa y sin justificar
				ppr = `<w:spacing w:after="160"/><w:ind w:left="720" w:hanging="720"/><w:jc w:val="left"/>`
			}
			d.parrafo(strings.Join(partes, " "), ppr, texto(22))
		}
	}
	d.referencias = false // el estilo no pasa de un archivo al siguiente
}

// ecuacion lee $$…$$ en una o varias líneas; devuelve la línea siguiente.
func (d *documento) ecuacion(lineas []string, i int) int {
	var partes []string
	j := i
	for ; j < len(lineas); j++ {
		s := strings.TrimSpace(lineas[j])
		if j == i {
			s = strings.TrimPrefix(s, "$$")
		}
		cierra := strings.HasSuffix(s, "$$") && (j > i || len(s) >= 2)
		partes = append(partes, strings.TrimSuffix(s, "$$"))
		if cierra {
			break
		}
	}
	d.b.WriteString(`<w:p><w:pPr><w:spacing w:before="120" w:after="120"/><w:jc w:val="center"/></w:pPr><m:oMathPara><m:oMath>` +
		LatexAOMML(strings.Join(partes, " ")) + `</m:oMath></m:oMathPara></w:p>`)
	return j + 1
}

func (d *documento) titulo(nivel int, s string) {
	estilos := map[int]string{1: "Heading1", 2: "Heading2", 3: "Heading3", 4: "Heading4"}
	tam := map[int]int{1: 28, 2: 24, 3: 22, 4: 22}[nivel]
	antes := map[int]int{1: 360, 2: 240, 3: 200, 4: 160}[nivel]
	e := estilo{Fuente: fuenteTexto, Negrita: true, SinCursiva: true, Color: "000000", Tam: tam}
	fmt.Fprintf(&d.b, `<w:p><w:pPr><w:pStyle w:val="%s"/><w:keepNext/><w:keepLines/><w:spacing w:before="%d" w:after="120"/><w:jc w:val="left"/><w:rPr>%s</w:rPr></w:pPr>`, estilos[nivel], antes, e.rPr())
	d.b.WriteString(d.inline(s, e))
	d.b.WriteString(`</w:p>`)
}

func (d *documento) parrafo(s, ppr string, e estilo) {
	d.b.WriteString(`<w:p><w:pPr>` + ppr + `</w:pPr>`)
	d.b.WriteString(d.inline(s, e))
	d.b.WriteString(`</w:p>`)
}

// cita: el informe es sobrio, así que una cita (> …) es un párrafo normal,
// sin recuadro ni color.
func (d *documento) cita(s string) {
	d.parrafo(s, `<w:jc w:val="both"/>`, texto(22))
}

// captura deja un espacio enmarcado para pegar una captura de pantalla, con
// su pie numerado como el de cualquier figura.
func (d *documento) captura(descripcion string) {
	d.figuras++
	d.b.WriteString(`<w:p><w:pPr><w:keepNext/><w:pBdr>` +
		`<w:top w:val="single" w:sz="4" w:space="40" w:color="A6A6A6"/><w:left w:val="single" w:sz="4" w:space="4" w:color="A6A6A6"/>` +
		`<w:bottom w:val="single" w:sz="4" w:space="40" w:color="A6A6A6"/><w:right w:val="single" w:sz="4" w:space="4" w:color="A6A6A6"/>` +
		`</w:pBdr><w:spacing w:before="240" w:after="120"/><w:jc w:val="center"/></w:pPr>`)
	e := estilo{Fuente: fuenteTexto, Color: "808080", Tam: 20}
	d.b.WriteString(`<w:r><w:rPr>` + e.rPr() + `</w:rPr><w:t>[Pegar aquí la captura]</w:t></w:r></w:p>`)
	pie := estilo{Fuente: fuenteTexto, Cursiva: true, Color: "404040", Tam: 20}
	d.parrafo(fmt.Sprintf("Figura %d. %s", d.figuras, descripcion), `<w:spacing w:before="0" w:after="240"/><w:jc w:val="center"/>`, pie)
}

// lista convierte una lista con viñetas o numerada; devuelve la línea siguiente.
func (d *documento) lista(lineas []string, i int) int {
	numerada := reNumerada.MatchString(strings.TrimSpace(lineas[i]))
	numID := d.bulletID
	if numerada {
		numID = d.numBase + len(d.numIds)
		d.numIds = append(d.numIds, numID)
	}
	for i < len(lineas) {
		t := strings.TrimSpace(lineas[i])
		var s string
		switch {
		case numerada && reNumerada.MatchString(t):
			s = reNumerada.FindStringSubmatch(t)[2]
		case !numerada && (strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ")):
			s = t[2:]
		default:
			return i
		}
		i++
		// Líneas de continuación: sangradas y que no empiezan un ítem nuevo.
		for i < len(lineas) && strings.HasPrefix(lineas[i], "  ") && strings.TrimSpace(lineas[i]) != "" {
			c := strings.TrimSpace(lineas[i])
			if strings.HasPrefix(c, "- ") || reNumerada.MatchString(c) {
				break
			}
			s += " " + c
			i++
		}
		ppr := fmt.Sprintf(`<w:numPr><w:ilvl w:val="0"/><w:numId w:val="%d"/></w:numPr><w:spacing w:after="80"/><w:ind w:left="720" w:hanging="360"/><w:jc w:val="both"/>`, numID)
		d.parrafo(s, ppr, texto(22))
	}
	return i
}

func (d *documento) codigo(lineas []string) {
	e := estilo{Fuente: fuenteCodigo, Tam: 16}
	for k, l := range lineas {
		despues := 0
		if k == len(lineas)-1 {
			despues = 200
		}
		fmt.Fprintf(&d.b, `<w:p><w:pPr><w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/><w:spacing w:before="0" w:after="%d" w:line="240" w:lineRule="auto"/><w:ind w:left="113" w:right="113"/></w:pPr>`, despues)
		d.b.WriteString(`<w:r><w:rPr>` + e.rPr() + `</w:rPr><w:t xml:space="preserve">` + escapar(strings.ReplaceAll(l, "\t", "    ")) + `</w:t></w:r></w:p>`)
	}
}

// tabla: primera fila de encabezado y segunda de alineación.
func (d *documento) tabla(lineas []string) {
	var filas [][]string
	var alin []string
	for k, l := range lineas {
		celdas := partirFila(l)
		if k == 1 && esSeparador(celdas) {
			for _, c := range celdas {
				switch {
				case strings.HasPrefix(c, ":") && strings.HasSuffix(c, ":"):
					alin = append(alin, "center")
				case strings.HasSuffix(c, ":"):
					alin = append(alin, "right")
				default:
					alin = append(alin, "left")
				}
			}
			continue
		}
		filas = append(filas, celdas)
	}
	if len(filas) == 0 {
		return
	}
	nc := len(filas[0])
	for len(alin) < nc {
		alin = append(alin, "left")
	}
	tam := 20
	switch {
	case nc >= 7:
		tam = 16
	case nc >= 5:
		tam = 18
	}
	anchos := anchosColumnas(filas, nc, tam)

	fmt.Fprintf(&d.b, `<w:tbl><w:tblPr><w:tblW w:w="%d" w:type="dxa"/><w:jc w:val="center"/><w:tblLayout w:type="fixed"/>`, anchoTexto)
	d.b.WriteString(`<w:tblCellMar><w:top w:w="40" w:type="dxa"/><w:left w:w="100" w:type="dxa"/><w:bottom w:w="40" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar><w:tblLook w:val="0600"/></w:tblPr><w:tblGrid>`)
	for _, a := range anchos {
		fmt.Fprintf(&d.b, `<w:gridCol w:w="%d"/>`, a)
	}
	d.b.WriteString(`</w:tblGrid>`)
	for k, f := range filas {
		encabezado := k == 0
		d.b.WriteString(`<w:tr><w:trPr><w:cantSplit/>`)
		if encabezado {
			d.b.WriteString(`<w:tblHeader/>`)
		}
		d.b.WriteString(`</w:trPr>`)
		for j := 0; j < nc; j++ {
			celda := ""
			if j < len(f) {
				celda = f[j]
			}
			e := texto(tam)
			fondo := "FFFFFF"
			switch {
			case encabezado:
				fondo, e.Color, e.Negrita = "000000", "FFFFFF", true
			case k%2 == 0:
				fondo = "F2F2F2"
			}
			fmt.Fprintf(&d.b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:tcBorders><w:bottom w:val="single" w:sz="4" w:space="0" w:color="D9D9D9"/></w:tcBorders><w:shd w:val="clear" w:color="auto" w:fill="%s"/><w:vAlign w:val="center"/></w:tcPr>`, anchos[j], fondo)
			fmt.Fprintf(&d.b, `<w:p><w:pPr><w:spacing w:before="0" w:after="0" w:line="240" w:lineRule="auto"/><w:jc w:val="%s"/></w:pPr>`, alin[j])
			d.b.WriteString(d.inline(celda, e))
			d.b.WriteString(`</w:p></w:tc>`)
		}
		d.b.WriteString(`</w:tr>`)
	}
	d.b.WriteString(`</w:tbl><w:p><w:pPr><w:spacing w:before="0" w:after="120"/></w:pPr></w:p>`)
}

// anchosColumnas reparte el ancho de la página: cada columna recibe primero lo
// que necesita su palabra más larga (para que no se parta a la mitad) y el
// resto se reparte según el largo del texto visible.
func anchosColumnas(filas [][]string, nc, tam int) []int {
	porCaracter := float64(tam) * 5.5 // dxa aproximados por carácter (tam en medios puntos)
	minimos := make([]float64, nc)
	pesos := make([]float64, nc)
	for _, f := range filas {
		for j := 0; j < nc && j < len(f); j++ {
			v := textoVisible(f[j])
			pesos[j] = max(pesos[j], min(float64(utf8.RuneCountInString(v)), 60))
			for _, palabra := range strings.Fields(v) {
				minimos[j] = max(minimos[j], float64(utf8.RuneCountInString(palabra))*porCaracter+260)
			}
		}
	}
	sumaMin, sumaPesos := 0.0, 0.0
	for j := range pesos {
		minimos[j] = min(minimos[j], anchoTexto/2)
		sumaMin += minimos[j]
		pesos[j] += 4
		sumaPesos += pesos[j]
	}
	libre := max(0, anchoTexto-sumaMin)
	anchos := make([]int, nc)
	suma := 0
	for j := range pesos {
		a := minimos[j] + libre*pesos[j]/sumaPesos
		if sumaMin > anchoTexto { // no alcanza: proporcional a los mínimos
			a = anchoTexto * minimos[j] / sumaMin
		}
		anchos[j] = int(a)
		suma += anchos[j]
	}
	anchos[nc-1] += anchoTexto - suma
	return anchos
}

func partirFila(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimPrefix(l, "|")
	l = strings.TrimSuffix(l, "|")
	var celdas []string
	var b strings.Builder
	enCodigo := false
	for _, r := range l {
		switch {
		case r == '`':
			enCodigo = !enCodigo
			b.WriteRune(r)
		case r == '|' && !enCodigo:
			celdas = append(celdas, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	return append(celdas, strings.TrimSpace(b.String()))
}

func esSeparador(celdas []string) bool {
	for _, c := range celdas {
		if strings.Trim(c, ":- ") != "" || !strings.Contains(c, "-") {
			return false
		}
	}
	return true
}

func textoVisible(s string) string {
	return strings.NewReplacer("**", "", "`", "", "*", "").Replace(s)
}

// figura inserta una imagen SVG centrada y su pie numerado.
func (d *documento) figura(alt, ruta string) {
	svg, err := os.ReadFile(filepath.Join(d.dir, ruta))
	if err != nil {
		d.aviso("no se pudo leer la figura %s: %v", ruta, err)
		return
	}
	w, h := tamanoSVG(svg)
	d.figuras++
	img := imagen{
		SVG:    svg,
		Nombre: fmt.Sprintf("figura%d", d.figuras),
		RelSVG: fmt.Sprintf("rIdFigS%d", d.figuras),
		RelPNG: fmt.Sprintf("rIdFigP%d", d.figuras),
	}
	d.imagenes = append(d.imagenes, img)
	cx := int64(anchoTexto) * emuPorDxa
	cy := int64(float64(cx) * h / w)
	d.docPr++
	d.b.WriteString(`<w:p><w:pPr><w:keepNext/><w:spacing w:before="120" w:after="60"/><w:jc w:val="center"/></w:pPr><w:r><w:drawing>`)
	fmt.Fprintf(&d.b, `<wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="%d" name="%s" descr="%s"/>`, cx, cy, d.docPr, img.Nombre, escapar(alt))
	d.b.WriteString(`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic>`)
	fmt.Fprintf(&d.b, `<pic:nvPicPr><pic:cNvPr id="0" name="%s.svg"/><pic:cNvPicPr/></pic:nvPicPr>`, img.Nombre)
	fmt.Fprintf(&d.b, `<pic:blipFill><a:blip r:embed="%s"><a:extLst><a:ext uri="{96DAC541-7B7A-43D3-8B79-37D633B846F1}"><asvg:svgBlip xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main" r:embed="%s"/></a:ext></a:extLst></a:blip><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`, img.RelPNG, img.RelSVG)
	fmt.Fprintf(&d.b, `<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`, cx, cy)
	pie := estilo{Fuente: fuenteTexto, Cursiva: true, Color: "404040", Tam: 20}
	d.parrafo(fmt.Sprintf("Figura %d. %s", d.figuras, alt), `<w:spacing w:before="0" w:after="240"/><w:jc w:val="center"/>`, pie)
}

var (
	reAncho = regexp.MustCompile(`<svg[^>]*\swidth="([\d.]+)`)
	reAlto  = regexp.MustCompile(`<svg[^>]*\sheight="([\d.]+)`)
)

func tamanoSVG(b []byte) (w, h float64) {
	w, h = 800, 400
	if m := reAncho.FindSubmatch(b); m != nil {
		fmt.Sscan(string(m[1]), &w)
	}
	if m := reAlto.FindSubmatch(b); m != nil {
		fmt.Sscan(string(m[1]), &h)
	}
	return
}

var (
	reEnlace = regexp.MustCompile(`^\[([^\]]+)\]\(([^)\s]+)\)`)
	reURL    = regexp.MustCompile(`^<(https?://[^>\s]+)>`)
)

// inline convierte el formato en línea: **negrita**, *cursiva*, `código`,
// [texto](url), <url> y $matemática$.
func (d *documento) inline(s string, base estilo) string {
	var b, pendiente strings.Builder
	actual := base
	volcar := func() {
		if pendiente.Len() == 0 {
			return
		}
		b.WriteString(`<w:r><w:rPr>` + actual.rPr() + `</w:rPr><w:t xml:space="preserve">` + escapar(pendiente.String()) + `</w:t></w:r>`)
		pendiente.Reset()
	}
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		c := r[i]
		resto := string(r[i:])
		switch {
		case c == '`':
			if fin := indiceRuna(r, i+1, '`'); fin >= 0 {
				volcar()
				e := actual
				e.Fuente = fuenteCodigo
				b.WriteString(`<w:r><w:rPr>` + e.rPr() + `</w:rPr><w:t xml:space="preserve">` + escapar(string(r[i+1:fin])) + `</w:t></w:r>`)
				i = fin
				continue
			}
		case strings.HasPrefix(resto, "**"):
			// El informe no usa negrita dentro del texto: solo los títulos y
			// los encabezados de tabla van en negrita (por su estilo base).
			i++
			continue
		case c == '*' && (actual.Cursiva != base.Cursiva || (i+1 < len(r) && r[i+1] != ' ')):
			volcar()
			actual.Cursiva = !actual.Cursiva
			continue
		case c == '$' && !strings.HasPrefix(resto, "$$"):
			if fin := indiceRuna(r, i+1, '$'); fin > i+1 {
				volcar()
				b.WriteString(`<m:oMath>` + LatexAOMML(string(r[i+1:fin])) + `</m:oMath>`)
				i = fin
				continue
			}
		case c == '[':
			if m := reEnlace.FindStringSubmatch(resto); m != nil {
				volcar()
				b.WriteString(d.hipervinculo(m[1], m[2], actual))
				i += utf8.RuneCountInString(m[0]) - 1
				continue
			}
		case c == '<':
			if m := reURL.FindStringSubmatch(resto); m != nil {
				volcar()
				b.WriteString(d.hipervinculo(m[1], m[1], actual))
				i += utf8.RuneCountInString(m[0]) - 1
				continue
			}
		}
		pendiente.WriteRune(c)
	}
	volcar()
	return b.String()
}

func indiceRuna(r []rune, desde int, c rune) int {
	for k := desde; k < len(r); k++ {
		if r[k] == c {
			return k
		}
	}
	return -1
}

func (d *documento) hipervinculo(s, url string, e estilo) string {
	id, ok := d.enlaces[url]
	if !ok {
		id = fmt.Sprintf("rIdEnl%d", len(d.orden)+1)
		d.enlaces[url] = id
		d.orden = append(d.orden, enlace{URL: url, Rel: id})
	}
	e.Color, e.Subrayado = "1155CC", true
	return `<w:hyperlink r:id="` + id + `"><w:r><w:rPr>` + e.rPr() + `</w:rPr><w:t xml:space="preserve">` + escapar(s) + `</w:t></w:r></w:hyperlink>`
}

func escapar(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
