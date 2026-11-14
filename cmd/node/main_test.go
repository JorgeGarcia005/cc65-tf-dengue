package main

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

func silencio(string, ...any) {}

// direccionLibre reserva un puerto y lo suelta: nadie escucha todavía.
func direccionLibre(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

func opcionesDePrueba(dir string) opciones {
	return opciones{coordinador: dir, nombre: "nodo-test", workers: 2,
		espera: 5 * time.Second, reintento: 20 * time.Millisecond, reconexiones: 3}
}

// En Compose los contenedores arrancan a la vez: el nodo debe insistir hasta
// que el coordinador escuche, y después entrenar hasta `fin`.
func TestServirReintentaHastaQueElCoordinadorEscucha(t *testing.T) {
	dir := direccionLibre(t)
	fin := make(chan error, 1)
	go func() { fin <- servir(opcionesDePrueba(dir), silencio) }()

	time.Sleep(200 * time.Millisecond) // el nodo ya falló varios intentos
	ln, err := net.Listen("tcp", dir)
	if err != nil {
		t.Skipf("el puerto reservado ya no está libre: %v", err)
	}
	ds, _ := ml.Sintetico(3_000, 5, 3)
	a := gold.Archivo{Dataset: ds, TamBloque: 256}
	ruta := filepath.Join(t.TempDir(), "shard_0.bin")
	if err := gold.Escribir(ruta, a); err != nil {
		t.Fatal(err)
	}
	desc, _, err := cluster.LeerShard(ruta)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ml.Config{TamBloque: 256, Lambda: 1e-3}
	c, err := cluster.Nuevo(ln, cluster.Opciones{Shards: []cluster.Shard{desc}, D: ds.D, Config: cfg,
		TimeoutRegistro: 5 * time.Second, TimeoutEpoca: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Cerrar()
	if err := c.EsperarNodos(1); err != nil {
		t.Fatal(err)
	}
	ent := ml.Entrenamiento{Epocas: 5, Tasa: 0.5}
	res, err := ml.EntrenarFalible(c, ds.D, ent)
	if err != nil {
		t.Fatal(err)
	}
	ref := ml.Entrenar(ml.NuevoSecuencial(ds, cfg), ds.D, ent)
	for j := range ref.W {
		if ref.W[j] != res.W[j] {
			t.Fatalf("w[%d] difiere del secuencial", j)
		}
	}
	if est := c.Estado(); est.Nodos[0].Nombre != "nodo-test" || est.Nodos[0].Workers != 2 {
		t.Errorf("registro del nodo: %+v", est.Nodos[0])
	}
	c.Terminar()
	select {
	case err := <-fin:
		if err != nil {
			t.Errorf("servir tras `fin{completo}` = %v, quiere nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("el nodo no terminó tras `fin`")
	}
}

// -workers=0 (el valor por defecto en Compose) significa «todos los núcleos».
func TestWorkersCeroUsaLosNucleosDisponibles(t *testing.T) {
	dir, registros := coordinadorFalso(t, func(_ int, pc *protocolo.Conexion) {
		pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Fin, Motivo: protocolo.MotivoCompleto}, time.Second)
		time.Sleep(50 * time.Millisecond)
	})
	op := opcionesDePrueba(dir)
	op.workers = 0
	if err := servir(op, silencio); err != nil {
		t.Fatal(err)
	}
	if r, quiere := <-registros, fmt.Sprintf("registro nodo-test v1 w%d", runtime.NumCPU()); r != quiere {
		t.Errorf("registro = %q, quiere %q", r, quiere)
	}
}

func TestServirSeRindeConErrorClaroSiElCoordinadorNoAparece(t *testing.T) {
	op := opcionesDePrueba(direccionLibre(t))
	op.espera = 150 * time.Millisecond
	inicio := time.Now()
	err := servir(op, silencio)
	if err == nil || !strings.Contains(err.Error(), "no se pudo conectar") {
		t.Fatalf("err = %v", err)
	}
	if d := time.Since(inicio); d > 3*time.Second {
		t.Errorf("tardó %v en rendirse con una espera de 150 ms", d)
	}
}

// coordinadorFalso acepta conexiones y aplica un guion a cada una.
func coordinadorFalso(t *testing.T, guion func(n int, pc *protocolo.Conexion)) (dir string, registros chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	registros = make(chan string, 16)
	go func() {
		for n := 0; ; n++ {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			pc := protocolo.Nueva(conn)
			m, err := pc.Recibir(5 * time.Second)
			if err != nil {
				registros <- "error: " + err.Error()
			} else {
				registros <- fmt.Sprintf("%s %s v%d w%d", m.Tipo, m.Nodo, m.Version, m.Workers)
			}
			guion(n, pc)
			pc.Cerrar()
		}
	}()
	return ln.Addr().String(), registros
}

// Si el coordinador corta la conexión, el nodo vuelve a registrarse.
func TestServirVuelveARegistrarseSiPierdeLaConexion(t *testing.T) {
	dir, registros := coordinadorFalso(t, func(n int, pc *protocolo.Conexion) {
		if n == 2 { // a la tercera, el entrenamiento termina bien
			pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Fin, Motivo: protocolo.MotivoCompleto}, time.Second)
			time.Sleep(50 * time.Millisecond)
		}
	})
	if err := servir(opcionesDePrueba(dir), silencio); err != nil {
		t.Fatalf("err = %v, quiere nil tras `fin{completo}`", err)
	}
	for i := range 3 {
		if r := <-registros; r != "registro nodo-test v1 w2" {
			t.Errorf("conexión %d: %q", i, r)
		}
	}
}

func TestServirDejaDeReconectarTrasElLimite(t *testing.T) {
	dir, registros := coordinadorFalso(t, func(int, *protocolo.Conexion) {})
	op := opcionesDePrueba(dir)
	op.reconexiones = 2
	err := servir(op, silencio)
	if err == nil || !strings.Contains(err.Error(), "conexión perdida 3 veces") {
		t.Fatalf("err = %v", err)
	}
	if len(registros) != 3 {
		t.Errorf("se registró %d veces, quiere 3 (1 + 2 reconexiones)", len(registros))
	}
}

// `fin{abortado}` no se reintenta: el proceso debe salir con error.
func TestServirNoReconectaSiElCoordinadorAborta(t *testing.T) {
	dir, registros := coordinadorFalso(t, func(_ int, pc *protocolo.Conexion) {
		pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Fin, Motivo: protocolo.MotivoAbortado}, time.Second)
		time.Sleep(50 * time.Millisecond)
	})
	if err := servir(opcionesDePrueba(dir), silencio); !errors.Is(err, cluster.ErrAbortado) {
		t.Fatalf("err = %v, quiere ErrAbortado", err)
	}
	if len(registros) != 1 {
		t.Errorf("se registró %d veces, quiere 1", len(registros))
	}
}
