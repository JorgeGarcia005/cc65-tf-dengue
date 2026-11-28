package main

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
)

// Cada línea de la región «protocolo» debe ser un mensaje que el protocolo
// real acepta, y debe haber uno de cada tipo.
func TestRegionProtocoloEsDelProtocoloReal(t *testing.T) {
	r, err := regionProtocolo()
	if err != nil {
		t.Fatal(err)
	}
	tipos := map[protocolo.Tipo]bool{}
	for _, l := range strings.Split(r, "\n") {
		if !strings.HasPrefix(l, "{") {
			continue
		}
		var m protocolo.Mensaje
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("%s: %v", l, err)
		}
		if err := m.Validar(); err != nil {
			t.Fatalf("%s: %v", l, err)
		}
		tipos[m.Tipo] = true
	}
	for _, tp := range []protocolo.Tipo{protocolo.Registro, protocolo.Asignacion, protocolo.Listo,
		protocolo.Epoca, protocolo.Gradiente, protocolo.Error, protocolo.Fin} {
		if !tipos[tp] {
			t.Errorf("falta un ejemplo de %q", tp)
		}
	}
}

func TestEquilibrioClusterConElModeloLineal(t *testing.T) {
	var d datosTB2
	d.S.M.S.Train.Filas = 1000
	js := `{"mediciones":[
		{"nodos":1,"desglose_por_epoca":{"seg_calculo_nodo_mas_lento":1}},
		{"nodos":2,"desglose_por_epoca":{"seg_red_y_espera":0.05}}]}`
	if err := json.Unmarshal([]byte(js), &d.Cluster); err != nil {
		t.Fatal(err)
	}
	eq := d.EquilibrioCluster()
	// Ahorro con 2 nodos: 1000 filas · 1 ms · 1/2 = 500 ms; equilibrio: 50 ms / 0,5 ms = 100 filas.
	if len(eq) != 1 || math.Abs(eq[0].AhorroMs-500) > 1e-9 || eq[0].ComunicaMs != 50 || eq[0].FilasMinimas != 100 {
		t.Fatalf("EquilibrioCluster = %+v", eq)
	}
}
