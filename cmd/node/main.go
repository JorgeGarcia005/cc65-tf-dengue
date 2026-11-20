// Command node es un nodo de cómputo del clúster (Parte B): se conecta al
// coordinador por TCP, carga los shards que le asignan desde un volumen de
// solo lectura y calcula sus parciales por bloque con un worker pool.
//
// Uso:
//
//	go run ./cmd/node -coordinador localhost:9000
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
)

type opciones struct {
	coordinador  string
	nombre       string
	workers      int
	espera       time.Duration // tiempo máximo intentando conectarse
	reintento    time.Duration // pausa entre intentos
	reconexiones int           // veces que vuelve a registrarse si pierde la conexión
}

func main() {
	var op opciones
	anfitrion, _ := os.Hostname()
	flag.StringVar(&op.coordinador, "coordinador", "localhost:9000", "dirección TCP del coordinador")
	flag.StringVar(&op.nombre, "nombre", anfitrion, "nombre del nodo en los logs y en la API")
	flag.IntVar(&op.workers, "workers", 0, "goroutines del worker pool (0 = núcleos disponibles)")
	flag.DurationVar(&op.espera, "espera", 60*time.Second, "tiempo máximo intentando conectarse al coordinador")
	flag.DurationVar(&op.reintento, "reintento", time.Second, "pausa entre intentos de conexión")
	flag.IntVar(&op.reconexiones, "reconexiones", 3, "veces que vuelve a registrarse si pierde la conexión")
	flag.Parse()
	if err := servir(op, log.Printf); err != nil {
		log.Fatal(err)
	}
}

// servir conecta el nodo y atiende al coordinador hasta `fin`. El coordinador
// puede no estar listo todavía (en Compose los contenedores arrancan a la
// vez), así que la conexión se reintenta durante op.espera. Si la conexión se
// pierde a mitad del entrenamiento, el nodo vuelve a registrarse: entra como
// nodo de reserva y hereda shards si otro cae.
func servir(op opciones, logf func(string, ...any)) error {
	if op.workers <= 0 {
		op.workers = runtime.NumCPU()
	}
	for intento := 0; ; intento++ {
		conn, err := conectar(op, logf)
		if err != nil {
			return err
		}
		logf("nodo %s conectado a %s (%d workers)", op.nombre, op.coordinador, op.workers)
		nodo := &cluster.Nodo{Nombre: op.nombre, Workers: op.workers, Log: logf}
		err = nodo.Servir(conn)
		if err == nil || errors.Is(err, cluster.ErrAbortado) {
			return err
		}
		if intento >= op.reconexiones {
			return fmt.Errorf("conexión perdida %d veces, la última: %w", intento+1, err)
		}
		logf("conexión perdida (%v); el nodo vuelve a registrarse", err)
	}
}

func conectar(op opciones, logf func(string, ...any)) (net.Conn, error) {
	limite := time.Now().Add(op.espera)
	for {
		conn, err := net.DialTimeout("tcp", op.coordinador, 5*time.Second)
		if err == nil {
			return conn, nil
		}
		if time.Now().Add(op.reintento).After(limite) {
			return nil, fmt.Errorf("no se pudo conectar a %s en %v: %w", op.coordinador, op.espera, err)
		}
		logf("coordinador no disponible (%v); reintento en %v", err, op.reintento)
		time.Sleep(op.reintento)
	}
}
