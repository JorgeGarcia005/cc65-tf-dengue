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
	fragmentos := 0
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		fragmentos += len(reFuente.FindAll(b, -1))
		nuevo, err := expandirRegiones(b)
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		if string(nuevo) != string(b) {
			t.Errorf("%s no coincide con el repositorio: corra go run ./cmd/informe", a)
		}
	}
	// El algoritmo distribuido completo: bucle, época, dos pasadas, plazo,
	// nodo y reducción.
	if fragmentos < 8 {
		t.Fatalf("el informe tiene %d fragmentos de código fuente; se esperaban al menos 8", fragmentos)
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
	if _, err := extraerFuncion("internal/ml/logistica.go", "NoExiste"); err == nil {
		t.Fatal("una función inexistente debía dar error")
	}
}

func TestRegionDesconocidaDaError(t *testing.T) {
	if _, err := expandirRegiones([]byte("<!-- generado:otra -->\nx\n<!-- /generado -->")); err == nil {
		t.Fatal("se esperaba error")
	}
}
