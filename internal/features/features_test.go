package features

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

func registros() []ingest.Registro {
	var rs []ingest.Registro
	deptos := []string{"PIURA", "PIURA", "LORETO", "LORETO", "ICA", "MOQUEGUA"}
	for i, d := range deptos {
		rs = append(rs, ingest.Registro{Departamento: d, Ano: 2011 + i, Semana: 1 + 10*i, EdadAnios: float64(5 + 12*i), Sexo: "FM"[i%2], Clase: i % 3})
	}
	return rs
}

func TestAjustarYLlenar(t *testing.T) {
	rs := registros()
	e, err := Ajustar(rs, 2011, 2)
	if err != nil {
		t.Fatal(err)
	}
	// PIURA y LORETO tienen ≥ 2 casos; ICA y MOQUEGUA van a OTROS.
	if got := e.Departamentos; len(got) != 3 || got[0] != "LORETO" || got[1] != "PIURA" || got[2] != OtrosDepartamentos {
		t.Fatalf("departamentos = %v", got)
	}
	if e.D() != 1+5+4+3 {
		t.Fatalf("D = %d (%v)", e.D(), e.Nombres)
	}
	x := make([]float64, e.D())
	e.Llenar(rs[0], x) // PIURA, 5 años, F, semana 1
	if math.Abs(x[0]-(5-e.MediaEdad)/e.DesvEdad) > 1e-12 {
		t.Errorf("edad_z = %v", x[0])
	}
	if x[1] != 1 || x[2]+x[3]+x[4]+x[5] != 0 {
		t.Errorf("etapa one-hot = %v", x[1:6])
	}
	if x[6] != 0 || x[7] != 0 || x[8] != 1 { // F, semana 1 → ángulo 0
		t.Errorf("sexo/semana = %v", x[6:9])
	}
	if x[10] != 0 || x[11] != 1 || x[12] != 0 {
		t.Errorf("depto one-hot = %v", x[10:])
	}
	e.Llenar(ingest.Registro{Departamento: "NO EXISTE", Semana: 27, EdadAnios: 70, Sexo: 'M'}, x)
	if x[12] != 1 || x[5] != 1 || x[6] != 1 {
		t.Errorf("departamento desconocido debe ir a OTROS: %v", x)
	}
	// Semana 27 está al otro lado del ciclo: coseno ≈ −1.
	if x[8] > -0.99 {
		t.Errorf("semana_cos(27) = %v", x[8])
	}
}

func TestEsquemaJSONIdaYVuelta(t *testing.T) {
	e, _ := Ajustar(registros(), 2011, 2)
	b, _ := json.Marshal(e)
	var f Esquema
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if err := f.Preparar(); err != nil {
		t.Fatal(err)
	}
	x1, x2 := make([]float64, e.D()), make([]float64, f.D())
	for _, r := range registros() {
		e.Llenar(r, x1)
		f.Llenar(r, x2)
		for j := range x1 {
			if x1[j] != x2[j] {
				t.Fatalf("variable %s difiere tras JSON", e.Nombres[j])
			}
		}
	}
	f.Nombres = f.Nombres[:3]
	if err := f.Preparar(); err == nil {
		t.Error("esperaba error con nombres inconsistentes")
	}
}

func TestEsTestDeterministaYProporcion(t *testing.T) {
	n, test := 200_000, 0
	for i := range n {
		if EsTest(i, 0.2) {
			test++
		}
		if EsTest(i, 0.2) != EsTest(i, 0.2) {
			t.Fatal("no determinista")
		}
	}
	if p := float64(test) / float64(n); math.Abs(p-0.2) > 0.005 {
		t.Errorf("proporción de test = %.4f, quiere ≈ 0,20", p)
	}
}

func TestAjustarRequiereDatos(t *testing.T) {
	if _, err := Ajustar(nil, 2011, 1); err == nil {
		t.Error("esperaba error sin datos")
	}
}

// TestSemanaDeAnguloInvierteLaCodificacion: la semana se guarda como seno y
// coseno; para reportar el riesgo por semana hay que recuperarla exactamente
// desde las variables del gold, tal como las escribe Llenar.
func TestSemanaDeAnguloInvierteLaCodificacion(t *testing.T) {
	e, err := Ajustar(registros(), 2011, 1)
	if err != nil {
		t.Fatal(err)
	}
	x := make([]float64, e.D())
	jSin, jCos := e.Indice("semana_sin"), e.Indice("semana_cos")
	if jSin < 0 || jCos < 0 {
		t.Fatal("el esquema no tiene semana_sin / semana_cos")
	}
	for s := 1; s <= 53; s++ {
		e.Llenar(ingest.Registro{Departamento: "LORETO", Ano: 2020, Semana: s, EdadAnios: 30, Sexo: 'F'}, x)
		quiere := s
		if s == 53 { // la codificación es periódica en 52: la semana 53 coincide con la 1
			quiere = 1
		}
		if got := SemanaDeAngulo(x[jSin], x[jCos]); got != quiere {
			t.Errorf("semana %d: SemanaDeAngulo = %d, quiere %d", s, got, quiere)
		}
	}
	if e.Indice("no_existe") != -1 {
		t.Error("Indice de una variable inexistente debe ser -1")
	}
}
