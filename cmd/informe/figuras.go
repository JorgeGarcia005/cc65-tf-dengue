package main

// Gráficos SVG del informe, generados desde los mismos reportes JSON que las
// tablas (AGENTS.md §3.6). Son estáticos (van a Word e impresión): fondo
// blanco, paleta categórica de referencia en orden fijo y como máximo tres
// series por gráfico, que son las que la paleta valida en todos los pares.

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Paleta categórica (modo claro), en orden fijo: azul, naranja, aqua.
var serie = [...]string{"#2a78d6", "#eb6834", "#1baf7a"}

const (
	tinta     = "#0b0b0b" // texto principal
	tintaSec  = "#52514e" // ejes y etiquetas secundarias
	rejilla   = "#e4e3df"
	superfic  = "#ffffff"
	fuenteSVG = "Arial, Helvetica, sans-serif"
)

type lienzo struct {
	b    strings.Builder
	w, h float64
}

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

func (l *lienzo) linea(x1, y1, x2, y2 float64, color string, grosor float64, extra string) {
	fmt.Fprintf(&l.b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%g"%s/>`+"\n",
		x1, y1, x2, y2, color, grosor, extra)
}

func (l *lienzo) cerrar() string { return l.b.String() + "</svg>\n" }

func escaparXML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// leyenda dibuja muestras de color con su nombre en una fila.
func (l *lienzo) leyenda(x, y float64, nombres []string, colores []string, discontinua map[int]bool) {
	for i, n := range nombres {
		if discontinua[i] {
			l.linea(x, y-4, x+18, y-4, colores[i], 2, ` stroke-dasharray="4 3"`)
		} else {
			fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="12" height="12" rx="2" fill="%s"/>`+"\n", x, y-10, colores[i])
		}
		l.texto(x+24, y, n, 12, tinta, "start", "")
		x += 24 + 6.4*float64(len([]rune(n))) + 18
	}
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

// figuraCasosPorAno: barras apiladas por año (sin signos / con signos / grave).
func figuraCasosPorAno(p perfil) string {
	anos := slices.Clone(p.Anos)
	slices.Sort(anos)
	l := nuevoLienzo(760, 380, "Casos notificados de dengue por año y clasificación")
	x0, y0, x1, y1 := 70.0, 58.0, 740.0, 320.0
	maxT := 0
	for _, a := range anos {
		c := p.Perfil.PorAno[a]
		maxT = max(maxT, c[0]+c[1]+c[2])
	}
	paso := pasoEje(float64(maxT))
	tope := math.Ceil(float64(maxT)/paso) * paso
	escalaY := func(v float64) float64 { return y1 - (y1-y0)*v/tope }
	for v := 0.0; v <= tope+1e-9; v += paso {
		y := escalaY(v)
		l.linea(x0, y, x1, y, rejilla, 1, "")
		l.texto(x0-8, y+4, miles(int(v)), 11, tintaSec, "end", "")
	}
	l.texto(x0-55, y0-8, "Casos", 12, tintaSec, "start", "")
	banda := (x1 - x0) / float64(len(anos))
	ancho := banda * 0.72
	for i, a := range anos {
		c := p.Perfil.PorAno[a]
		x := x0 + banda*float64(i) + (banda-ancho)/2
		base := 0.0
		for k := 0; k < 3; k++ {
			if c[k] == 0 {
				continue
			}
			ya, yb := escalaY(base), escalaY(base+float64(c[k]))
			alto := ya - yb - 1 // 1 px de separación entre segmentos
			if alto > 0.5 {
				fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`+"\n", x, yb, ancho, alto, serie[k])
			}
			base += float64(c[k])
		}
		if i%2 == 0 || i == len(anos)-1 {
			l.texto(x+ancho/2, y1+16, a, 11, tintaSec, "middle", "")
		}
	}
	l.linea(x0, y1, x1, y1, tintaSec, 1, "")
	l.leyenda(x0, 22, []string{"Sin signos de alarma (A97.0)", "Con signos de alarma (A97.1)", "Grave (A97.2)"}, serie[:], nil)
	l.texto(x0, 352, "Fuente: MINSA / CDC Perú, datosabiertos.gob.pe (reports/perfil_bronze.json).", 11, tintaSec, "start", "")
	return l.cerrar()
}

// curvaSpeedup es una serie de (workers, speedup) con su nombre.
type curvaSpeedup struct {
	Nombre  string
	Workers []int
	Speedup []float64
}

// figuraSpeedup: speedup frente a workers (eje x en potencias de 2) con la
// recta ideal y = x como referencia discontinua.
func figuraSpeedup(titulo string, curvas []curvaSpeedup) string {
	if len(curvas) > len(serie) {
		log.Fatalf("figura %q: %d series; la paleta valida como máximo %d", titulo, len(curvas), len(serie))
	}
	l := nuevoLienzo(760, 400, titulo)
	x0, y0, x1, y1 := 60.0, 60.0, 600.0, 340.0
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
	lg := func(w int) float64 { return math.Log2(float64(w)) }
	escalaX := func(w int) float64 {
		return x0 + (x1-x0)*(lg(w)-lg(ws[0]))/(lg(ws[len(ws)-1])-lg(ws[0]))
	}
	for v := 0.0; v <= tope+1e-9; v += paso {
		y := escalaY(v)
		l.linea(x0, y, x1, y, rejilla, 1, "")
		l.texto(x0-8, y+4, strings.Replace(fmt.Sprintf("%g×", v), ".", ",", 1), 11, tintaSec, "end", "")
	}
	for _, w := range ws {
		x := escalaX(w)
		l.linea(x, y1, x, y1+4, tintaSec, 1, "")
		l.texto(x, y1+18, fmt.Sprint(w), 11, tintaSec, "middle", "")
	}
	l.linea(x0, y1, x1, y1, tintaSec, 1, "")
	l.texto((x0+x1)/2, y1+38, "Workers (goroutines)", 12, tintaSec, "middle", "")
	l.texto(x0-50, y0-14, "Speedup", 12, tintaSec, "start", "")

	// Recta ideal y = x, recortada al tope del eje.
	var ideal []string
	for _, w := range ws {
		if float64(w) > tope {
			// Termina donde la recta cruza el tope del eje.
			xt := x0 + (x1-x0)*(math.Log2(tope)-lg(ws[0]))/(lg(ws[len(ws)-1])-lg(ws[0]))
			ideal = append(ideal, fmt.Sprintf("%.1f,%.1f", xt, escalaY(tope)))
			break
		}
		ideal = append(ideal, fmt.Sprintf("%.1f,%.1f", escalaX(w), escalaY(float64(w))))
	}
	fmt.Fprintf(&l.b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="1.5" stroke-dasharray="4 3"/>`+"\n", strings.Join(ideal, " "), tintaSec)

	nombres := []string{"Ideal (speedup = workers)"}
	colores := []string{tintaSec}
	for i, c := range curvas {
		var pts []string
		for k, w := range c.Workers {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", escalaX(w), escalaY(c.Speedup[k])))
		}
		fmt.Fprintf(&l.b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="2" stroke-linejoin="round"/>`+"\n", strings.Join(pts, " "), serie[i])
		for k, w := range c.Workers {
			fmt.Fprintf(&l.b, `<circle cx="%.1f" cy="%.1f" r="4" fill="%s" stroke="%s" stroke-width="2"/>`+"\n", escalaX(w), escalaY(c.Speedup[k]), serie[i], superfic)
		}
		// Etiqueta directa: el mejor speedup de la serie, junto a su último punto.
		mejor := slices.Max(c.Speedup)
		ultimo := len(c.Workers) - 1
		l.texto(escalaX(c.Workers[ultimo])+10, escalaY(c.Speedup[ultimo])+4, strings.Replace(fmt.Sprintf("máx. %.2f×", mejor), ".", ",", 1), 11, tinta, "start", "")
		nombres = append(nombres, c.Nombre)
		colores = append(colores, serie[i])
	}
	// Leyenda en columna arriba a la izquierda, donde las curvas aún son
	// bajas (pocos workers), sobre un fondo que tapa la rejilla.
	lx, ly := x0+10, y0+6
	fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="290" height="%.1f" fill="%s"/>`+"\n", lx-6, ly-4, 16*float64(len(nombres))+6, superfic)
	for i, n := range nombres {
		y := ly + 12 + 16*float64(i)
		if i == 0 {
			l.linea(lx, y-4, lx+18, y-4, colores[i], 1.5, ` stroke-dasharray="4 3"`)
		} else {
			l.linea(lx, y-4, lx+18, y-4, colores[i], 2, "")
		}
		l.texto(lx+24, y, n, 11, tinta, "start", "")
	}
	return l.cerrar()
}

// figuraRiesgoDepto: barras horizontales del riesgo calibrado por departamento,
// con una marca para la tasa observada en prueba.
func figuraRiesgoDepto(e entrenamiento) string {
	g := e.PorDepto
	alto := 60 + 20*float64(len(g)) + 50
	l := nuevoLienzo(760, alto, "Riesgo de dengue con signos de alarma o grave por departamento")
	x0, x1, y0 := 190.0, 700.0, 60.0
	maxV := 0.0
	for _, d := range g {
		maxV = max(maxV, d.Calib, d.Obs)
	}
	paso := pasoEje(maxV * 100)
	tope := math.Ceil(maxV*100/paso) * paso
	escalaX := func(v float64) float64 { return x0 + (x1-x0)*v*100/tope }
	yFin := y0 + 20*float64(len(g))
	for v := 0.0; v <= tope+1e-9; v += paso {
		x := x0 + (x1-x0)*v/tope
		l.linea(x, y0-6, x, yFin, rejilla, 1, "")
		l.texto(x, yFin+16, strings.Replace(fmt.Sprintf("%g %%", v), ".", ",", 1), 11, tintaSec, "middle", "")
	}
	for i, d := range g {
		y := y0 + 20*float64(i)
		nombre := nombreGrupo(d.Grupo)
		if d.Casos < 100 {
			// Con tan pocos casos la tasa observada no es representativa.
			nombre += fmt.Sprintf(" (n = %d)", d.Casos)
		}
		l.texto(x0-8, y+12, nombre, 11, tinta, "end", "")
		fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="%.1f" height="14" rx="2" fill="%s"/>`+"\n", x0, y+2, escalaX(d.Calib)-x0, serie[0])
		xo := escalaX(d.Obs)
		l.linea(xo, y, xo, y+18, tinta, 2.5, "")
	}
	l.linea(x0, y0-6, x0, yFin, tintaSec, 1, "")
	fmt.Fprintf(&l.b, `<rect x="%.1f" y="%.1f" width="12" height="12" rx="2" fill="%s"/>`+"\n", x0, 22.0, serie[0])
	l.texto(x0+18, 32, "Riesgo calibrado medio (modelo)", 12, tinta, "start", "")
	l.linea(x0+260, 20, x0+260, 36, tinta, 2.5, "")
	l.texto(x0+268, 32, "Tasa observada en prueba", 12, tinta, "start", "")
	l.texto(20, alto-12, "Fuente: reports/entrenamiento.json (conjunto de prueba).", 11, tintaSec, "start", "")
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
