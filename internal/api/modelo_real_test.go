package api

import (
	"os"
	"testing"
)

// TestCargarModeloReal usa data/gold/modelo.json si existe (no se versiona;
// lo genera cmd/entrenar). En el CI se omite.
func TestCargarModeloReal(t *testing.T) {
	b, err := os.ReadFile("../../data/gold/modelo.json")
	if err != nil {
		t.Skip("sin data/gold/modelo.json: correr go run ./cmd/entrenar")
	}
	m, err := CargarModelo(b)
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.Predecir(Consulta{Departamento: "Loreto", Ano: 2024, Semana: 10, Edad: ptr(34.0), Sexo: "M"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Riesgo <= 0 || p.Riesgo >= 1 || p.DepartamentoModelo != "LORETO" {
		t.Fatalf("predicción inesperada: %+v", p)
	}
	t.Logf("modelo %s: %+v", m.Version, p)
}
