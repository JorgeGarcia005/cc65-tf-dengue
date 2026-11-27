package api

import (
	"encoding/json"
	"math"
	"testing"
)

// Auditoría TB2 B2: la versión identifica el modelo (pesos, esquema, umbral y
// ajuste), no el momento en que se generó. Reentrenar con los mismos
// hiperparámetros converge bit a bit a los mismos pesos, así que debe dar la
// misma versión: no vacía la caché de Redis ni desfasa reports/evaluacion.json.
func TestVersionNoDependeDeGenerado(t *testing.T) {
	base, err := CargarModelo(modeloDePrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	a, err := base.conPesos(append([]float64(nil), base.Pesos...), "2026-11-27T10:00:00-05:00")
	if err != nil {
		t.Fatal(err)
	}
	b, err := base.conPesos(append([]float64(nil), base.Pesos...), "2026-11-29T23:00:00-05:00")
	if err != nil {
		t.Fatal(err)
	}
	if a.Version != b.Version {
		t.Fatalf("mismos pesos, distinta versión: %s y %s", a.Version, b.Version)
	}
	if a.Version != base.Version {
		t.Fatalf("mismos pesos que el archivo, distinta versión: %s y %s", a.Version, base.Version)
	}

	// El archivo que escribe un entrenamiento, al cargarse, conserva la versión.
	j, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	c, err := CargarModelo(j)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != a.Version {
		t.Fatalf("al recargar el modelo cambió la versión: %s → %s", a.Version, c.Version)
	}

	// Un solo bit distinto en un peso sí cambia la versión.
	w := append([]float64(nil), base.Pesos...)
	w[0] = math.Nextafter(w[0], math.Inf(1))
	d, err := base.conPesos(w, "2026-11-27T10:00:00-05:00")
	if err != nil {
		t.Fatal(err)
	}
	if d.Version == a.Version {
		t.Fatal("pesos distintos con la misma versión")
	}
}
