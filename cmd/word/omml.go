package main

// Conversión de un subconjunto de LaTeX a OMML (ecuaciones nativas de Word).
// Cubre lo que usa el informe: fracciones, sumatorias, subíndices y
// superíndices, letras griegas, normas y espacios. Lo desconocido se escribe
// tal cual, para que nunca se pierda texto.

import (
	"strings"
	"unicode"
)

var simbolos = map[string]string{
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε", "eta": "η",
	"theta": "θ", "lambda": "λ", "mu": "μ", "pi": "π", "sigma": "σ", "tau": "τ", "phi": "φ",
	"nabla": "∇", "top": "⊤", "cdot": "·", "times": "×", "dots": "…", "ldots": "…", "cdots": "⋯",
	"lVert": "‖", "rVert": "‖", "Vert": "‖", "leq": "≤", "geq": "≥", "neq": "≠", "approx": "≈",
	"in": "∈", "to": "→", "leftarrow": "←", "infty": "∞", "partial": "∂", "{": "{", "}": "}",
}

// funcionesTexto se escriben en redonda (sin cursiva), como en LaTeX.
var funcionesTexto = map[string]bool{"log": true, "exp": true, "ln": true, "max": true, "min": true}

// espacios en LaTeX → carácter de espacio Unicode.
var espacios = map[string]string{",": " ", ";": " ", "quad": " ", "qquad": "  ", " ": " "}

// tamaños que solo agrandan el delimitador siguiente: se ignoran.
var tamanos = map[string]bool{"big": true, "Big": true, "bigg": true, "Bigg": true, "left": true, "right": true}

type latex struct {
	s []rune
	i int
}

// LatexAOMML devuelve el contenido de un <m:oMath> para la expresión dada.
func LatexAOMML(expr string) string {
	p := &latex{s: []rune(strings.TrimSpace(expr))}
	return p.secuencia(0)
}

// secuencia lee átomos hasta el fin o hasta '}' (si dentro de un grupo).
// Una sumatoria toma como cuerpo el término que la sigue, hasta el siguiente
// '+', '-', '=', ',' o espacio grande al mismo nivel.
func (p *latex) secuencia(nivel int) string {
	var b strings.Builder
	for p.i < len(p.s) {
		if p.s[p.i] == '}' {
			if nivel > 0 {
				return b.String()
			}
			p.i++
			continue
		}
		b.WriteString(p.atomoConScripts())
	}
	return b.String()
}

// termino lee átomos mientras no aparezca un separador de términos fuera de
// paréntesis o corchetes.
func (p *latex) termino() string {
	var b strings.Builder
	prof := 0
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == '}' {
			break
		}
		if prof == 0 && (c == '+' || c == '-' || c == '=' || c == ',' || c == '−' ||
			p.empiezaCon(`\qquad`) || p.empiezaCon(`\quad`)) {
			break
		}
		switch c {
		case '(', '[':
			prof++
		case ')', ']':
			if prof == 0 {
				return b.String() // cierra un paréntesis de afuera
			}
			prof--
		}
		b.WriteString(p.atomoConScripts())
	}
	return b.String()
}

func (p *latex) empiezaCon(t string) bool {
	return strings.HasPrefix(string(p.s[p.i:]), t)
}

// atomoConScripts lee un átomo y sus _ y ^ opcionales.
func (p *latex) atomoConScripts() string {
	if p.empiezaCon(`\sum`) || p.empiezaCon(`\prod`) {
		chr := "∑"
		if p.empiezaCon(`\prod`) {
			chr = "∏"
			p.i += len(`\prod`)
		} else {
			p.i += len(`\sum`)
		}
		sub, sup := p.scripts()
		cuerpo := p.termino()
		var b strings.Builder
		b.WriteString(`<m:nary><m:naryPr><m:chr m:val="` + chr + `"/><m:limLoc m:val="undOvr"/>`)
		if sub == "" {
			b.WriteString(`<m:subHide m:val="1"/>`)
		}
		if sup == "" {
			b.WriteString(`<m:supHide m:val="1"/>`)
		}
		b.WriteString(`</m:naryPr><m:sub>` + sub + `</m:sub><m:sup>` + sup + `</m:sup><m:e>` + cuerpo + `</m:e></m:nary>`)
		return b.String()
	}
	base := p.atomo()
	if base == "" {
		return ""
	}
	sub, sup := p.scripts()
	switch {
	case sub != "" && sup != "":
		return `<m:sSubSup><m:e>` + base + `</m:e><m:sub>` + sub + `</m:sub><m:sup>` + sup + `</m:sup></m:sSubSup>`
	case sub != "":
		return `<m:sSub><m:e>` + base + `</m:e><m:sub>` + sub + `</m:sub></m:sSub>`
	case sup != "":
		return `<m:sSup><m:e>` + base + `</m:e><m:sup>` + sup + `</m:sup></m:sSup>`
	}
	return base
}

// scripts lee subíndice y superíndice en cualquier orden.
func (p *latex) scripts() (sub, sup string) {
	for k := 0; k < 2 && p.i < len(p.s); k++ {
		switch p.s[p.i] {
		case '_':
			p.i++
			sub = p.argumento()
		case '^':
			p.i++
			sup = p.argumento()
		default:
			return
		}
	}
	return
}

// argumento lee un grupo {…} o un solo átomo.
func (p *latex) argumento() string {
	p.saltarEspacios()
	if p.i < len(p.s) && p.s[p.i] == '{' {
		p.i++
		r := p.secuencia(1)
		if p.i < len(p.s) && p.s[p.i] == '}' {
			p.i++
		}
		return r
	}
	return p.atomo()
}

func (p *latex) saltarEspacios() {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
}

// atomo lee un grupo, un comando o un carácter.
func (p *latex) atomo() string {
	if p.i >= len(p.s) {
		return ""
	}
	c := p.s[p.i]
	switch {
	case c == ' ':
		p.i++
		return ""
	case c == '{':
		p.i++
		r := p.secuencia(1)
		if p.i < len(p.s) && p.s[p.i] == '}' {
			p.i++
		}
		return r
	case c == '\\':
		return p.comando()
	}
	p.i++
	return corrida(string(c), false)
}

func (p *latex) comando() string {
	p.i++ // '\'
	if p.i >= len(p.s) {
		return ""
	}
	// Comandos de un solo símbolo: \, \; \{ \} \  .
	if c := p.s[p.i]; !unicode.IsLetter(c) {
		p.i++
		if e, ok := espacios[string(c)]; ok {
			return corrida(e, true)
		}
		if s, ok := simbolos[string(c)]; ok {
			return corrida(s, false)
		}
		return corrida(string(c), false)
	}
	ini := p.i
	for p.i < len(p.s) && unicode.IsLetter(p.s[p.i]) {
		p.i++
	}
	nombre := string(p.s[ini:p.i])
	switch {
	case nombre == "frac":
		num := p.argumento()
		den := p.argumento()
		return `<m:f><m:num>` + num + `</m:num><m:den>` + den + `</m:den></m:f>`
	case tamanos[nombre]:
		return ""
	case funcionesTexto[nombre]:
		return corrida(nombre, true)
	}
	if e, ok := espacios[nombre]; ok {
		return corrida(e, true)
	}
	if s, ok := simbolos[nombre]; ok {
		return corrida(s, false)
	}
	return corrida(`\`+nombre, true)
}

// corrida es un <m:r>; recta = sin cursiva.
func corrida(t string, recta bool) string {
	estilo := ""
	if recta {
		estilo = `<m:rPr><m:sty m:val="p"/></m:rPr>`
	}
	return `<m:r>` + estilo + `<w:rPr><w:rFonts w:ascii="Cambria Math" w:hAnsi="Cambria Math"/></w:rPr><m:t xml:space="preserve">` + escapar(t) + `</m:t></m:r>`
}
