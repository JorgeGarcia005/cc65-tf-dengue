package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestEtiquetaBench(t *testing.T) {
	casos := map[string]string{
		"bench_loader_deez-wsl-debian.json":         "deez-wsl-debian",
		"bench_gradiente_deez-wsl-debian.json":      "deez-wsl-debian",
		"bench_gradiente_deez-wsl-debian_gold.json": "deez-wsl-debian",
		"bench_gradiente_deez_gold.json":            "deez",
		"bench_loader_ci-2vcpu.json":                "ci-2vcpu",
	}
	for archivo, quiere := range casos {
		if got := etiquetaBench(archivo); got != quiere {
			t.Errorf("etiquetaBench(%q) = %q, quiere %q", archivo, got, quiere)
		}
	}
}

// En la PC4, las secciones 6.3 y 7.7 muestran solo los benchmarks de una
// máquina (maquinaPC4); la PC3 conserva todas.
func TestPC4BenchmarksDeUnaMaquina(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	reArchivo := regexp.MustCompile("`(bench_[a-z]+_[^`]+\\.json)`")
	for _, sec := range []string{"06-limpieza.md", "07-modelo.md"} {
		b, err := os.ReadFile(filepath.Join("docs/informe/pc4", sec))
		if err != nil {
			t.Fatal(err)
		}
		citados := reArchivo.FindAllSubmatch(b, -1)
		if len(citados) == 0 {
			t.Errorf("%s no cita ningún benchmark", sec)
		}
		for _, c := range citados {
			if e := etiquetaBench(string(c[1])); e != maquinaPC4 {
				t.Errorf("%s cita %s (máquina %q), quiere solo %q", sec, c[1], e, maquinaPC4)
			}
		}
	}
}
