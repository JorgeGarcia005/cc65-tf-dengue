package main

// Gráficos SVG del informe, generados desde los mismos reportes JSON que las
// tablas (AGENTS.md §3.6). Siguen el estilo de una figura científica impresa
// (APA 7, 7.22–7.36): blanco y negro, ejes enmarcados con marcas hacia
// dentro, series distinguidas por el trazo y el marcador (no por el color),
// coma decimal y sin título ni fuente dentro de la imagen: el número, el
// título y la nota van en el pie del informe.

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	tinta     = "#000000"
	gris      = "#7f7f7f"
	grisClaro = "#d9d9d9"
	superfic  = "#ffffff"
	fuenteSVG = "Times New Roman, Times, serif"
	marca     = 5.0 // largo de las marcas de los ejes (hacia dentro)
)

// trazo es cómo se distingue una serie en blanco y negro.
type trazo struct {
	Discontinuo string // stroke-dasharray ("" = continuo)
	Marcador    string // circulo, cuadrado, triangulo
	Relleno     bool   // marcador relleno o hueco
}

var trazos = [...]trazo{
	{"", "circulo", true},
	{"7 4", "cuadrado", false},
	{"2 3", "triangulo", true},
}

type lienzo struct {
	b    strings.Builder
	w, h float64
}

// nuevoLienzo abre el SVG; titulo queda como <title> (accesibilidad), no se
// dibuja.
func nuevoLienzo(w, h float64, titulo string) *lienzo {
	l := &lienzo{w: w, h: h}
	fmt.Fprintf(&l.b, `<svg xmlns="http://www.w3.org/2000/svg" width="%g" height="%g" viewBox="0 0 %g %g" font-family="%s">`+"\n", w, h, w, h, fuenteSVG)
	fmt.Fprintf(&l.b, `<title>%s</title>`+"\n", escaparXML(titulo))
	fmt.Fprintf(&l.b, `<rect width="%g" height="%g" fill="%s"/>`+"\n", w, h, superfic)
	return l
}

func (l *lienzo) texto(x, y float64, s string, tam float64, color, ancla string, extra string) {
	fmt.Fprintf(&l.b, `<text x="%.1f" y="%.1f" font-size="%g" fill="%s" text-anchor="%s"%s>%s</text>`+"\n",
		x, y, tam, color, ancla, extra, escaparXML(s))
}

// textoVertical escribe el título de un eje Y, girado 90°.
func (l *lienzo) textoVertical(x, y float64, s string) {
	l.texto(x, y, s, 14, tinta, "middle", fmt.Sprintf(` transform="rotate(-90 %.1f %.1f)"`, x, y))
}

func (l *lienzo) linea(x1, y1, x2, y2 float64, color string, grosor float64, extra string) {
	fmt.Fprintf(&l.b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%g"%s/>`+"\n",
		x1, y1, x2, y2, color, grosor, extra)
}

func (l *lienzo) rect(x, y, w, h float64, relleno string, borde float64) {
	trazo := ""
	if borde > 0 {
		trazo = fmt.Sprintf(` stroke="%s" stroke-width="%g"`, tinta, borde)
	}
	fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"%s/>`+"\n", x, y, w, h, relleno, trazo)
}

// marco dibuja el recuadro del área de trazado.
func (l *lienzo) marco(x0, y0, x1, y1 float64) {
	fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="none" stroke="%s" stroke-width="1"/>`+"\n", x0, y0, x1-x0, y1-y0, tinta)
}

// marcaY y marcaX: marca hacia dentro del marco con su etiqueta afuera.
func (l *lienzo) marcaY(x0, x1, y float64, etiqueta string) {
	l.linea(x0, y, x0+marca, y, tinta, 1, "")
	l.linea(x1, y, x1-marca, y, tinta, 1, "")
	l.texto(x0-6, y+4.5, etiqueta, 13, tinta, "end", "")
}

func (l *lienzo) marcaX(x, y0, y1 float64, etiqueta string) {
	l.linea(x, y1, x, y1-marca, tinta, 1, "")
	l.linea(x, y0, x, y0+marca, tinta, 1, "")
	if etiqueta != "" {
		l.texto(x, y1+17, etiqueta, 13, tinta, "middle", "")
	}
}

// marcador dibuja el símbolo de una serie centrado en (x, y).
func (l *lienzo) marcador(x, y float64, t trazo) {
	relleno := superfic
	if t.Relleno {
		relleno = tinta
	}
	estilo := fmt.Sprintf(`fill="%s" stroke="%s" stroke-width="1.2"`, relleno, tinta)
	switch t.Marcador {
	case "cuadrado":
		fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="8" height="8" %s/>`+"\n", x-4, y-4, estilo)
	case "triangulo":
		fmt.Fprintf(&l.b, `<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" %s/>`+"\n", x, y-5, x-5, y+4, x+5, y+4, estilo)
	default:
		fmt.Fprintf(&l.b, `<circle cx="%.1f" cy="%.1f" r="4" %s/>`+"\n", x, y, estilo)
	}
}

// polilinea dibuja una serie con su trazo (sin marcadores).
func (l *lienzo) polilinea(pts []string, t trazo, grosor float64) {
	extra := ""
	if t.Discontinuo != "" {
		extra = fmt.Sprintf(` stroke-dasharray="%s"`, t.Discontinuo)
	}
	fmt.Fprintf(&l.b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="%g" stroke-linejoin="round"%s/>`+"\n",
		strings.Join(pts, " "), tinta, grosor, extra)
}

// muestra es una entrada de leyenda: una línea con marcador o un rectángulo.
type muestra struct {
	Nombre  string
	Trazo   *trazo  // línea
	Relleno string  // rectángulo (barras)
	Grosor  float64 // de la línea
	Raya    bool    // marca vertical (valor observado)
}

// leyenda dibuja un recuadro con una entrada por fila desde (x, y).
func (l *lienzo) leyenda(x, y float64, ms []muestra) {
	ancho := 0.0
	for _, m := range ms {
		ancho = max(ancho, 6.6*float64(len([]rune(m.Nombre))))
	}
	alto := 20*float64(len(ms)) + 8
	fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="0.8"/>`+"\n",
		x, y, ancho+52, alto, superfic, tinta)
	for i, m := range ms {
		cy := y + 14 + 20*float64(i)
		switch {
		case m.Trazo != nil:
			extra := ""
			if m.Trazo.Discontinuo != "" {
				extra = fmt.Sprintf(` stroke-dasharray="%s"`, m.Trazo.Discontinuo)
			}
			l.linea(x+8, cy, x+36, cy, tinta, max(m.Grosor, 1), extra)
			if m.Trazo.Marcador != "" {
				l.marcador(x+22, cy, *m.Trazo)
			}
		case m.Raya:
			l.linea(x+22, cy-7, x+22, cy+7, tinta, 2.5, "")
		default:
			l.rect(x+14, cy-6, 16, 12, m.Relleno, 0.8)
		}
		l.texto(x+44, cy+4.5, m.Nombre, 13, tinta, "start", "")
	}
}

func (l *lienzo) cerrar() string { return l.b.String() + "</svg>\n" }

func escaparXML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// pasoEje elige un paso "redondo" para unas 5 marcas entre 0 y max.
func pasoEje(max float64) float64 {
	bruto := max / 5
	p := math.Pow(10, math.Floor(math.Log10(bruto)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if bruto <= m*p {
			return m * p
		}
	}
	return 10 * p
}

func miles(n int) string { return funciones["miles"].(func(int) string)(n) }

// decimal escribe v con coma decimal y el sufijo dado ("%g" sin ceros de más).
func decimal(v float64, sufijo string) string {
	return strings.Replace(fmt.Sprintf("%g", v), ".", ",", 1) + sufijo
}

// figuraCasosPorAno: barras apiladas por año (sin signos / con signos / grave),
// en tres tonos de gris.
func figuraCasosPorAno(p perfil) string {
	anos := slices.Clone(p.Anos)
	slices.Sort(anos)
	l := nuevoLienzo(760, 380, "Casos notificados de dengue por año y clasificación")
	x0, y0, x1, y1 := 92.0, 14.0, 744.0, 330.0
	maxT := 0
	for _, a := range anos {
		c := p.Perfil.PorAno[a]
		maxT = max(maxT, c[0]+c[1]+c[2])
	}
	paso := pasoEje(float64(maxT))
	tope := math.Ceil(float64(maxT)*1.05/paso) * paso
	escalaY := func(v float64) float64 { return y1 - (y1-y0)*v/tope }
	rellenos := [3]string{superfic, gris, tinta}
	banda := (x1 - x0) / float64(len(anos))
	ancho := banda * 0.7
	for i, a := range anos {
		c := p.Perfil.PorAno[a]
		x := x0 + banda*float64(i) + (banda-ancho)/2
		base := 0.0
		for k := 0; k < 3; k++ {
			if c[k] == 0 {
				continue
			}
			ya, yb := escalaY(base), escalaY(base+float64(c[k]))
			if ya-yb > 0.3 {
				l.rect(x, yb, ancho, ya-yb, rellenos[k], 0.6)
			}
			base += float64(c[k])
		}
		etiqueta := ""
		if strings.HasSuffix(a, "0") || strings.HasSuffix(a, "5") {
			etiqueta = a
		}
		l.marcaX(x+ancho/2, y0, y1, etiqueta)
	}
	for v := 0.0; v <= tope+1e-9; v += paso {
		l.marcaY(x0, x1, escalaY(v), miles(int(v)))
	}
	l.marco(x0, y0, x1, y1)
	l.texto((x0+x1)/2, y1+40, "Año de notificación", 14, tinta, "middle", "")
	l.textoVertical(22, (y0+y1)/2, "Casos notificados")
	l.leyenda(x0+14, y0+12, []muestra{
		{Nombre: "Sin signos de alarma (A97.0)", Relleno: rellenos[0]},
		{Nombre: "Con signos de alarma (A97.1)", Relleno: rellenos[1]},
		{Nombre: "Grave (A97.2)", Relleno: rellenos[2]},
	})
	return l.cerrar()
}

// curvaSpeedup es una serie de (workers, speedup) con su nombre.
type curvaSpeedup struct {
	Nombre  string
	Workers []int
	Speedup []float64
}

// figuraSpeedup: speedup frente a workers (eje x en potencias de 2) con la
// recta ideal y = x como referencia.
func figuraSpeedup(titulo string, curvas []curvaSpeedup) string {
	return figuraSpeedupEjes(titulo, curvas, "Workers (goroutines)", "Ideal (speedup = workers)", false)
}

// figuraSpeedupEjes dibuja speedup frente a workers o nodos, con la recta
// ideal y = x punteada. Con lineal el eje X es lineal (nodos 1, 2, 3…); si
// no, log2 (workers 1, 2, 4…).
func figuraSpeedupEjes(titulo string, curvas []curvaSpeedup, ejeX, nombreIdeal string, lineal bool) string {
	if len(curvas) > len(trazos) {
		log.Fatalf("figura %q: %d series; en blanco y negro se distinguen como máximo %d", titulo, len(curvas), len(trazos))
	}
	l := nuevoLienzo(760, 420, titulo)
	x0, y0, x1, y1 := 78.0, 14.0, 740.0, 360.0
	var ws []int
	maxS := 1.0
	for _, c := range curvas {
		for i, w := range c.Workers {
			if !slices.Contains(ws, w) {
				ws = append(ws, w)
			}
			maxS = max(maxS, c.Speedup[i])
		}
	}
	slices.Sort(ws)
	paso := pasoEje(maxS)
	tope := math.Ceil((maxS+0.5)/paso) * paso
	escalaY := func(v float64) float64 { return y1 - (y1-y0)*v/tope }
	lg := func(w float64) float64 {
		if lineal {
			return w
		}
		return math.Log2(w)
	}
	// Margen de media marca a cada lado para que los puntos no toquen el marco.
	ini, fin := lg(float64(ws[0])), lg(float64(ws[len(ws)-1]))
	margen := (fin - ini) * 0.04
	escalaX := func(w float64) float64 { return x0 + (x1-x0)*(lg(w)-ini+margen)/(fin-ini+2*margen) }
	for v := 0.0; v <= tope+1e-9; v += paso {
		l.marcaY(x0, x1, escalaY(v), decimal(v, ""))
	}
	for _, w := range ws {
		l.marcaX(escalaX(float64(w)), y0, y1, fmt.Sprint(w))
	}
	l.texto((x0+x1)/2, y1+40, ejeX, 14, tinta, "middle", "")
	l.textoVertical(26, (y0+y1)/2, "Speedup (×)")

	// Recta ideal y = x, recortada al tope del eje.
	ideal := trazo{Discontinuo: "1.5 3"}
	var pts []string
	for _, w := range ws {
		v := float64(w)
		if v > tope {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", escalaX(tope), escalaY(tope)))
			break
		}
		pts = append(pts, fmt.Sprintf("%.1f,%.1f", escalaX(v), escalaY(v)))
	}
	l.polilinea(pts, ideal, 1)

	ms := []muestra{{Nombre: nombreIdeal, Trazo: &trazo{Discontinuo: ideal.Discontinuo}, Grosor: 1}}
	for i, c := range curvas {
		t := trazos[i]
		var pts []string
		for k, w := range c.Workers {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", escalaX(float64(w)), escalaY(c.Speedup[k])))
		}
		l.polilinea(pts, t, 1.4)
		for k, w := range c.Workers {
			l.marcador(escalaX(float64(w)), escalaY(c.Speedup[k]), t)
		}
		ms = append(ms, muestra{Nombre: c.Nombre, Trazo: &trazos[i], Grosor: 1.4})
	}
	l.marco(x0, y0, x1, y1)
	// Leyenda arriba a la izquierda, donde las curvas aún son bajas.
	l.leyenda(x0+14, y0+12, ms)
	return l.cerrar()
}

// figuraRiesgoDepto: barras horizontales del riesgo calibrado por
// departamento, con una raya para la tasa observada en prueba.
func figuraRiesgoDepto(e entrenamiento) string {
	g := e.PorDepto
	fila := 21.0
	x0, x1, y0 := 200.0, 740.0, 66.0
	yFin := y0 + fila*float64(len(g)) + 6
	l := nuevoLienzo(760, yFin+52, "Riesgo de dengue con signos de alarma o grave por departamento")
	maxV := 0.0
	for _, d := range g {
		maxV = max(maxV, d.Calib, d.Obs)
	}
	paso := pasoEje(maxV * 100)
	tope := math.Ceil(maxV*100*1.05/paso) * paso
	escalaX := func(v float64) float64 { return x0 + (x1-x0)*v*100/tope }
	for i, d := range g {
		y := y0 + 6 + fila*float64(i)
		nombre := nombreGrupo(d.Grupo)
		if d.Casos < 100 {
			// Con tan pocos casos la tasa observada no es representativa.
			nombre += fmt.Sprintf(" (n = %d)", d.Casos)
		}
		l.texto(x0-8, y+fila/2+4, nombre, 13, tinta, "end", "")
		l.rect(x0, y+4, escalaX(d.Calib)-x0, fila-8, grisClaro, 0.6)
		xo := escalaX(d.Obs)
		l.linea(xo, y+1, xo, y+fila-1, tinta, 2.5, "")
	}
	for v := 0.0; v <= tope+1e-9; v += paso {
		l.marcaX(x0+(x1-x0)*v/tope, y0, yFin, decimal(v, ""))
	}
	l.marco(x0, y0, x1, yFin)
	l.texto((x0+x1)/2, yFin+40, "Riesgo (%)", 14, tinta, "middle", "")
	l.leyenda(x0, 8, []muestra{
		{Nombre: "Riesgo calibrado medio (modelo)", Relleno: grisClaro},
		{Nombre: "Tasa observada en el conjunto de prueba", Raya: true},
	})
	return l.cerrar()
}

// figuraRiesgoSemana: dos paneles con el mismo eje de semanas, porque son dos
// medidas en escalas distintas (nunca un gráfico con dos ejes y): (a) los
// casos con signos de alarma o grave y (b) el riesgo calibrado y la tasa
// observada.
func figuraRiesgoSemana(e entrenamiento) string {
	gs := e.PorSemana
	l := nuevoLienzo(760, 470, "Casos graves y riesgo por semana epidemiológica")
	x0, x1 := 92.0, 740.0
	banda := (x1 - x0) / 52
	xSem := func(s int) float64 { return x0 + banda*(float64(s)-0.5) }

	// (a) Casos con signos de alarma o grave.
	a0, a1 := 14.0, 200.0
	maxC := 0.0
	for _, g := range gs {
		maxC = max(maxC, float64(g.Casos)*g.Obs)
	}
	paso := pasoEje(maxC)
	tope := math.Ceil(maxC*1.05/paso) * paso
	yA := func(v float64) float64 { return a1 - (a1-a0)*v/tope }
	for _, g := range gs {
		v := float64(g.Casos) * g.Obs
		l.rect(xSem(g.Semana)-banda*0.36, yA(v), banda*0.72, a1-yA(v), grisClaro, 0.6)
	}
	for v := 0.0; v <= tope+1e-9; v += paso {
		l.marcaY(x0, x1, yA(v), miles(int(v)))
	}
	for s := 1; s <= 52; s++ {
		if s == 1 || s%4 == 0 {
			l.marcaX(xSem(s), a0, a1, "")
		}
	}
	l.marco(x0, a0, x1, a1)
	l.texto(x0+10, a0+20, "(a)", 14, tinta, "start", "")
	l.textoVertical(22, (a0+a1)/2, "Casos (prueba)")

	// (b) Riesgo calibrado y tasa observada (%).
	b0, b1 := 222.0, 420.0
	maxR := 0.0
	for _, g := range gs {
		maxR = max(maxR, g.Calib, g.Obs)
	}
	pasoR := pasoEje(maxR * 100)
	// Holgura arriba para la letra del panel.
	topeR := math.Ceil(maxR*100*1.25/pasoR) * pasoR
	yB := func(v float64) float64 { return b1 - (b1-b0)*v*100/topeR }
	for k, campo := range []func(grupoRiesgo) float64{
		func(g grupoRiesgo) float64 { return g.Calib },
		func(g grupoRiesgo) float64 { return g.Obs },
	} {
		var pts []string
		for _, g := range gs {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", xSem(g.Semana), yB(campo(g))))
		}
		l.polilinea(pts, trazo{Discontinuo: trazos[k].Discontinuo}, 1.4)
	}
	for v := 0.0; v <= topeR+1e-9; v += pasoR {
		l.marcaY(x0, x1, b1-(b1-b0)*v/topeR, decimal(v, ""))
	}
	for s := 1; s <= 52; s++ {
		if s == 1 || s%4 == 0 {
			l.marcaX(xSem(s), b0, b1, fmt.Sprint(s))
		}
	}
	l.marco(x0, b0, x1, b1)
	l.texto(x0+10, b0+20, "(b)", 14, tinta, "start", "")
	l.textoVertical(22, (b0+b1)/2, "Riesgo (%)")
	l.texto((x0+x1)/2, b1+40, "Semana epidemiológica", 14, tinta, "middle", "")
	// Leyenda abajo a la derecha del panel (b): el riesgo nunca baja a cero.
	l.leyenda(x1-280, b1-62, []muestra{
		{Nombre: "Riesgo calibrado medio (modelo)", Trazo: &trazo{}, Grosor: 1.4},
		{Nombre: "Tasa observada", Trazo: &trazo{Discontinuo: trazos[1].Discontinuo}, Grosor: 1.4},
	})
	return l.cerrar()
}

func escribirFigura(ruta, svg string) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(svg), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("escrito %s", ruta)
}

// cpuCorta quita del modelo de CPU lo que no aporta a una leyenda.
func cpuCorta(cpu string) string {
	switch {
	case strings.Contains(cpu, "Ryzen 7 5800X"):
		return "AMD Ryzen 7 5800X"
	case strings.Contains(cpu, "Xeon"):
		return "Intel Xeon (CI)"
	}
	return cpu
}

// nombreMaquina resume la máquina de un benchmark para la leyenda.
func nombreMaquina(cpu string, numCPU int, so string) string {
	sistemas := map[string]string{"linux": "Linux", "windows": "Windows"}
	if s, ok := sistemas[so]; ok {
		so = s
	}
	return fmt.Sprintf("%s, %d hilos, %s", cpuCorta(cpu), numCPU, so)
}
