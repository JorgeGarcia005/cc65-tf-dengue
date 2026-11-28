package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	reDecision = regexp.MustCompile(`\*\*Decisión:\*\*\s*\**\s*(APROBADO|CAMBIOS|RECHAZADO)`)
	// Formato de los primeros veredictos (PR-39 a PR-47): la decisión en
	// negrita al comienzo de una línea.
	reDecisionAntigua = regexp.MustCompile(`(?m)^\*\*(APROBADO|CAMBIOS|RECHAZADO)\b`)
	reHallazgo        = regexp.MustCompile(`(?m)^###\s+H\d+\b.*·\s*(BLOQUEANTE|MAYOR|MENOR)\s*$`)
	reRondaArch       = regexp.MustCompile(`^ronda-(\d+)\.md$`)
)

// resumenRevisiones cuenta los veredictos del juez en docs/agentes/revisiones:
// por PR, cuántas rondas tuvo y cómo terminó; y los hallazgos por severidad.
type resumenRevisiones struct {
	PRs, Rondas, Aprobados, PrimeraRonda int
	PorRondas                            map[int]int // PRs según su número de rondas
	Hallazgos                            map[string]int
}

func leerRevisiones(dir string) (resumenRevisiones, error) {
	r := resumenRevisiones{PorRondas: map[int]int{}, Hallazgos: map[string]int{}}
	prs, err := filepath.Glob(filepath.Join(dir, "PR-*"))
	if err != nil {
		return r, err
	}
	for _, pr := range prs {
		entradas, err := os.ReadDir(pr)
		if err != nil {
			return r, err
		}
		var rondas []int
		for _, e := range entradas {
			if m := reRondaArch.FindStringSubmatch(e.Name()); m != nil {
				n, _ := strconv.Atoi(m[1])
				rondas = append(rondas, n)
			}
		}
		if len(rondas) == 0 {
			continue
		}
		slices.Sort(rondas)
		final := ""
		for _, n := range rondas {
			b, err := os.ReadFile(filepath.Join(pr, fmt.Sprintf("ronda-%d.md", n)))
			if err != nil {
				return r, err
			}
			m := reDecision.FindSubmatch(b)
			if m == nil {
				m = reDecisionAntigua.FindSubmatch(b)
			}
			if m == nil {
				return r, fmt.Errorf("%s/ronda-%d.md: sin «**Decisión:**»", pr, n)
			}
			final = string(m[1])
			for _, h := range reHallazgo.FindAllSubmatch(b, -1) {
				r.Hallazgos[string(h[1])]++
			}
		}
		r.PRs++
		r.Rondas += len(rondas)
		r.PorRondas[len(rondas)]++
		if final == "APROBADO" {
			r.Aprobados++
			if len(rondas) == 1 {
				r.PrimeraRonda++
			}
		}
	}
	return r, nil
}

// regionRevisiones: la tabla del ecosistema multiagente (sección 19.3).
func regionRevisiones() (string, error) {
	r, err := leerRevisiones("docs/agentes/revisiones")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Tabla: Revisiones del juez sobre los Pull Requests del proyecto {#tab:revisiones}\n\n")
	b.WriteString("| Indicador | Valor |\n|---|---:|\n")
	fmt.Fprintf(&b, "| Pull Requests con veredicto del juez | %d |\n", r.PRs)
	fmt.Fprintf(&b, "| Rondas de revisión en total | %d |\n", r.Rondas)
	fmt.Fprintf(&b, "| Aprobados en la primera ronda | %d |\n", r.PrimeraRonda)
	for _, n := range []int{2, 3} {
		fmt.Fprintf(&b, "| Pull Requests que necesitaron %d rondas | %d |\n", n, r.PorRondas[n])
	}
	fmt.Fprintf(&b, "| Terminados con veredicto APROBADO | %d |\n", r.Aprobados)
	for _, s := range []string{"BLOQUEANTE", "MAYOR", "MENOR"} {
		fmt.Fprintf(&b, "| Hallazgos de severidad %s | %d |\n", strings.ToLower(s), r.Hallazgos[s])
	}
	b.WriteString("\nNota: Cuenta los archivos `ronda-<k>.md` de `docs/agentes/revisiones/PR-<n>/`: la decisión de cada ronda y los hallazgos con su severidad. Generado por `cmd/informe`.\n")
	return b.String(), nil
}
