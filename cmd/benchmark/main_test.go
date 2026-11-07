package main

import (
	"testing"
	"time"
)

// TestPasadasPorMedicionSuperaLaResolucionDelReloj: en Windows el reloj avanza
// a saltos de ~0,5–1 ms, así que una pasada de 30 µs se mediría como 0. Cada
// medición debe agrupar suficientes pasadas para durar al menos el mínimo.
func TestPasadasPorMedicionSuperaLaResolucionDelReloj(t *testing.T) {
	const pasada = 2 * time.Millisecond
	const minimo = 20 * time.Millisecond
	k := pasadasPorMedicion(func() { time.Sleep(pasada) }, minimo)
	if k < 2 {
		t.Fatalf("k = %d: una pasada de %v no alcanza %v", k, pasada, minimo)
	}
	if k > 10 {
		t.Fatalf("k = %d: sobran pasadas (cada una dura al menos %v)", k, pasada)
	}
	if k := pasadasPorMedicion(func() { time.Sleep(30 * time.Millisecond) }, minimo); k != 1 {
		t.Fatalf("k = %d: una pasada larga debe medirse sola", k)
	}
}
