package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Observación 2 del profesor en la PC3: el código fuente del algoritmo
// distribuido va en el informe copiado del repositorio, no pegado a mano.
// Si alguien cambia el código y no regenera el informe (o edita un fragmento
// a mano), este test falla.
func TestFragmentosSalenDelFuente(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	archivos, err := filepath.Glob("docs/informe/pc4/*.md")
	if err != nil || len(archivos) == 0 {
		t.Fatalf("sin secciones en docs/informe/pc4: %v", err)
	}
	reFuente := regexp.MustCompile(`<!-- generado:fuente:`)
	fragmentos := map[string]int{}
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		fragmentos[filepath.Base(a)] = len(reFuente.FindAll(b, -1))
		nuevo, err := expandirRegiones(b)
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		if string(nuevo) != string(b) {
			t.Errorf("%s no coincide con el repositorio: corra go run ./cmd/informe", a)
		}
	}
	// Mínimo por sección: el algoritmo distribuido completo (bucle, época, dos
	// pasadas, plazo, nodo y reducción), la API y las entidades en Go.
	minimos := map[string]int{"08-distribuido.md": 9, "10-api.md": 4, "11-bd.md": 8}
	for archivo, min := range minimos {
		if fragmentos[archivo] < min {
			t.Errorf("%s tiene %d fragmentos de código fuente; se esperaban al menos %d", archivo, fragmentos[archivo], min)
		}
	}
}

func TestExtraerFuncionYMetodo(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	f, err := extraerFuncion("internal/ml/logistica.go", "ReducirEnOrden")
	if err != nil || !strings.HasPrefix(f, "// ReducirEnOrden") || !strings.HasSuffix(f, "}") {
		t.Fatalf("función: %q, %v", f, err)
	}
	m, err := extraerFuncion("internal/cluster/coordinador.go", "(*Coordinador).cubrir")
	if err != nil || !strings.Contains(m, "func (c *Coordinador) cubrir() error {") {
		t.Fatalf("método: %q, %v", m, err)
	}
	ty, err := extraerFuncion("internal/almacen/almacen.go", "RegistroPrediccion")
	if err != nil || !strings.HasPrefix(ty, "// RegistroPrediccion") || !strings.Contains(ty, "type RegistroPrediccion struct {") || !strings.HasSuffix(ty, "}") {
		t.Fatalf("tipo: %q, %v", ty, err)
	}
	if _, err := extraerFuncion("internal/ml/logistica.go", "NoExiste"); err == nil {
		t.Fatal("una función inexistente debía dar error")
	}
}

func TestRegionDesconocidaDaError(t *testing.T) {
	if _, err := expandirRegiones([]byte("<!-- generado:otra -->\nx\n<!-- /generado -->")); err == nil {
		t.Fatal("se esperaba error")
	}
}

func TestExtraerTipoDentroDeUnBloque(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "x.go")
	src := "package x\n\ntype (\n\t// A es uno.\n\tA struct{ N int }\n\tB int\n)\n"
	if err := os.WriteFile(ruta, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := extraerFuncion(ruta, "A")
	if err != nil || got != "// A es uno.\ntype A struct{ N int }" {
		t.Fatalf("tipo en bloque: %q, %v", got, err)
	}
}

// TestInformeTB2AlDia: los fragmentos de código, de archivos y las cifras en
// línea del informe final coinciden con el repositorio y los reportes. Si
// alguien edita un valor a mano o cambia el código sin regenerar, falla.
func TestInformeTB2AlDia(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	archivos, err := filepath.Glob("docs/informe/tb2/*.md")
	if err != nil || len(archivos) == 0 {
		t.Fatalf("sin secciones en docs/informe/tb2: %v", err)
	}
	p, l, m := cargar()
	datos := nuevoDatosTB2(calcularSintesis(p, l, m))
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		nuevo, err := expandirRegiones(b)
		if err == nil {
			nuevo, err = expandirValores(nuevo, datos)
		}
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		if string(nuevo) != string(b) {
			t.Errorf("%s no coincide con el repositorio: corra go run ./cmd/informe", a)
		}
	}
}

// TestEvidenciasSoloDelScript (#70): toda evidencia versionada salió de
// scripts/evidencias-tb2.sh y terminó bien. Una evidencia fallida, editada sin
// su cabecera o pegada a mano no pasa el CI.
func TestEvidenciasSoloDelScript(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	archivos, _ := filepath.Glob("docs/informe/tb2/evidencias/*.txt")
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		lineas := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if len(lineas) < 5 || !strings.HasPrefix(lineas[0], "# evidencia: ") ||
			!strings.HasPrefix(lineas[1], "# fecha: ") || !strings.HasPrefix(lineas[3], "# commit: ") {
			t.Errorf("%s: sin la cabecera de scripts/evidencias-tb2.sh", a)
			continue
		}
		if lineas[len(lineas)-1] != "# resultado: OK" {
			t.Errorf("%s: el paso no terminó bien (última línea %q)", a, lineas[len(lineas)-1])
		}
	}
}
