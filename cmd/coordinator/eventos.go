package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
)

// BuferEventos es la cola del grabador de -eventos. Una época con 5 nodos emite
// una docena de eventos y el escritor solo hace E/S secuencial: 4096 cubre
// ráfagas largas sin frenar al coordinador (si se llena, se descarta y se cuenta).
const BuferEventos = 4096

// abrirEventos crea el .jsonl de -eventos y cuelga un cluster.Grabador como
// observador (#69: lo reproduce el modo réplica del panel admin sin clúster).
// Con ruta vacía no hace nada. La función que devuelve vacía la cola, cierra el
// archivo e informa cuántos eventos se descartaron; es idempotente.
func abrirEventos(ruta string, op *opciones, logf func(string, ...any)) (func(), error) {
	if ruta == "" {
		return func() {}, nil
	}
	f, err := os.Create(ruta)
	if err != nil {
		return nil, fmt.Errorf("-eventos: %w", err)
	}
	g := cluster.NuevoGrabador(f, BuferEventos)
	// En modo servicio, ejecutarAPI reemplaza el observador por uno que publica
	// en el hub del WebSocket y además llama a grabar; sin -api, graba solo.
	op.grabar = g.Observar
	op.observador = g.Observar
	logf("grabando los eventos del clúster en %s", ruta)
	cerrado := false
	return func() {
		if cerrado {
			return
		}
		cerrado = true
		errG := g.Cerrar()
		errF := f.Close()
		if err := errors.Join(errG, errF); err != nil {
			logf("-eventos %s: %v", ruta, err)
			return
		}
		logf("eventos grabados en %s (%d descartados)", ruta, g.Descartados())
	}, nil
}
