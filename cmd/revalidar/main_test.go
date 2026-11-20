package main

import (
	"strings"
	"testing"
)

func TestCompararIgnoraCamposDeTiempoYWorkers(t *testing.T) {
	ref := `{"generado":"2026-10-24","workers":2,"seg_total":1.5,"speedup":1.8,"filas":10,"pesos":[0.1,0.2]}`
	nuevo := `{"generado":"2026-11-17","workers":16,"seg_total":0.3,"speedup":7.1,"filas":10,"pesos":[0.1,0.2]}`
	difs, err := Comparar([]byte(ref), []byte(nuevo))
	if err != nil {
		t.Fatal(err)
	}
	if len(difs) != 0 {
		t.Fatalf("esperaba sin diferencias, obtuve %v", difs)
	}
}

func TestCompararDetectaDiferenciaDeUnBitEnFloat(t *testing.T) {
	// 0.30000000000000004 y 0.3 difieren en el último bit de la mantisa.
	ref := `{"pesos":[0.1,0.30000000000000004]}`
	nuevo := `{"pesos":[0.1,0.3]}`
	difs, err := Comparar([]byte(ref), []byte(nuevo))
	if err != nil {
		t.Fatal(err)
	}
	if len(difs) != 1 || !strings.Contains(difs[0], "pesos[1]") {
		t.Fatalf("esperaba una diferencia en pesos[1], obtuve %v", difs)
	}
}

func TestCompararMismoNumeroEscritoDistintoEsIgual(t *testing.T) {
	difs, err := Comparar([]byte(`{"x":1}`), []byte(`{"x":1.0}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(difs) != 0 {
		t.Fatalf("1 y 1.0 son el mismo float64: %v", difs)
	}
}

func TestCompararDetectaCamposFaltantesSobrantesYAnidados(t *testing.T) {
	ref := `{"a":{"b":"x","c":[1,2]},"solo_ref":true,"seg_x":{"no":"importa"}}`
	nuevo := `{"a":{"b":"y","c":[1,2,3]},"solo_nuevo":false}`
	difs, err := Comparar([]byte(ref), []byte(nuevo))
	if err != nil {
		t.Fatal(err)
	}
	todo := strings.Join(difs, "\n")
	for _, esperado := range []string{"a.b", "a.c", "solo_ref", "solo_nuevo"} {
		if !strings.Contains(todo, esperado) {
			t.Errorf("falta diferencia en %q; obtuve:\n%s", esperado, todo)
		}
	}
	if strings.Contains(todo, "seg_x") {
		t.Errorf("seg_x debía ignorarse aunque falte en uno: %s", todo)
	}
}

func TestCompararJSONInvalido(t *testing.T) {
	if _, err := Comparar([]byte(`{`), []byte(`{}`)); err == nil {
		t.Fatal("esperaba error con JSON inválido")
	}
}
