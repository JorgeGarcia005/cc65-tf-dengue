package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// goldDePrueba escribe train.bin y los shards de un dataset sintético.
func goldDePrueba(t *testing.T, filas, shards int) (dir string, train gold.Archivo) {
	t.Helper()
	dir = t.TempDir()
	ds, _ := ml.Sintetico(filas, 8, 7)
	train = gold.Archivo{Dataset: ds, TamBloque: 256}
	partes, err := gold.Particionar(train, shards)
	if err != nil {
		t.Fatal(err)
	}
	if err := gold.Escribir(filepath.Join(dir, "train.bin"), train); err != nil {
		t.Fatal(err)
	}
	for s, p := range partes {
		if err := gold.Escribir(filepath.Join(dir, fmt.Sprintf("shard_%d.bin", s)), p); err != nil {
			t.Fatal(err)
		}
	}
	return dir, train
}

func escuchar(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func lanzarNodos(t *testing.T, dir string, n int) *sync.WaitGroup {
	t.Helper()
	var wg sync.WaitGroup
	for i := range n {
		conn, err := net.Dial("tcp", dir)
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			(&cluster.Nodo{Nombre: fmt.Sprintf("nodo-%d", i), Workers: 2}).Servir(conn)
		}()
	}
	return &wg
}

func silencio(string, ...any) {}

func TestEjecutarEntrenaEnElClusterYDaElModeloSecuencial(t *testing.T) {
	dir, train := goldDePrueba(t, 6_000, 3)
	ln := escuchar(t)
	nodos := lanzarNodos(t, ln.Addr().String(), 3)
	op := opciones{gold: dir, nodos: 3, epocas: 15, tasa: 0.5, lambda: 1e-3, verificar: true,
		tRegistro: 5 * time.Second, tEpoca: 10 * time.Second}
	rep, err := ejecutar(ln, op, silencio, nil)
	if err != nil {
		t.Fatal(err)
	}
	nodos.Wait() // los nodos reciben `fin` y terminan

	if rep.Verificado == nil || !*rep.Verificado {
		t.Fatal("el reporte no marca el modelo como igual al secuencial")
	}
	pos, neg := ml.PesosBalanceados(train.Y)
	cfg := ml.Config{TamBloque: train.TamBloque, Lambda: op.lambda, PesoPos: pos, PesoNeg: neg}
	ref := ml.Entrenar(ml.NuevoSecuencial(train.Dataset, cfg), train.D, ml.Entrenamiento{Epocas: 15, Tasa: 0.5})
	if !mismosBits(ref.W[:train.D], rep.Pesos) || !mismosBits(ref.W[train.D:], []float64{rep.Sesgo}) || !mismosBits(ref.Historia, rep.Historia) {
		t.Fatal("pesos, sesgo o historia del reporte difieren del secuencial")
	}
	if rep.Shards != 3 || rep.FilasTrain != 6_000 || rep.D != 8 || rep.Epocas != 15 || len(rep.Cluster.Nodos) != 3 ||
		rep.PesoPos != pos || rep.PesoNeg != neg {
		t.Errorf("reporte: %+v", rep)
	}

	// El reporte en disco usa las claves que lee `cmd/entrenar -comparar`.
	ruta := filepath.Join(t.TempDir(), "sub", "cluster.json")
	if err := escribirJSON(ruta, rep); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(ruta)
	var crudo map[string]json.RawMessage
	if err := json.Unmarshal(b, &crudo); err != nil {
		t.Fatal(err)
	}
	for _, clave := range []string{"epocas", "tasa_aprendizaje", "lambda", "peso_positivos", "peso_negativos", "pesos", "sesgo", "perdida_por_epoca", "cluster", "igual_al_secuencial"} {
		if _, ok := crudo[clave]; !ok {
			t.Errorf("falta la clave %q en el reporte", clave)
		}
	}
}

// Con menos nodos que shards el reparto es por menor carga y el modelo no cambia.
func TestEjecutarConDosNodosYCincoShards(t *testing.T) {
	dir, _ := goldDePrueba(t, 5_000, 5)
	ln := escuchar(t)
	nodos := lanzarNodos(t, ln.Addr().String(), 2)
	rep, err := ejecutar(ln, opciones{gold: dir, nodos: 2, epocas: 6, tasa: 0.5, verificar: true,
		tRegistro: 5 * time.Second, tEpoca: 10 * time.Second}, silencio, nil)
	if err != nil {
		t.Fatal(err)
	}
	nodos.Wait()
	if rep.Shards != 5 || !*rep.Verificado {
		t.Errorf("reporte: %+v", rep)
	}
}

func TestEjecutarFallaConErrorClaroSiNoSeRegistranLosNodos(t *testing.T) {
	dir, _ := goldDePrueba(t, 2_000, 2)
	ln := escuchar(t)
	nodos := lanzarNodos(t, ln.Addr().String(), 1)
	_, err := ejecutar(ln, opciones{gold: dir, nodos: 2, epocas: 3, tasa: 0.5, tRegistro: 200 * time.Millisecond}, silencio, nil)
	if err == nil || !strings.Contains(err.Error(), "1 de 2 nodos") {
		t.Fatalf("err = %v", err)
	}
	nodos.Wait() // Cerrar cortó la conexión del único nodo
}

// El reporte se guarda antes de avisar `fin`: con Compose y --exit-code-from,
// el primer nodo que termina detiene a los demás contenedores.
func TestElReporteSeGuardaAntesDeQueLosNodosRecibanFin(t *testing.T) {
	dir, _ := goldDePrueba(t, 2_000, 2)
	ln := escuchar(t)
	nodos := lanzarNodos(t, ln.Addr().String(), 2)
	terminaron := make(chan struct{})
	go func() { nodos.Wait(); close(terminaron) }()
	guardado := false
	_, err := ejecutar(ln, opciones{gold: dir, nodos: 2, epocas: 4, tasa: 0.5, verificar: true,
		tRegistro: 5 * time.Second, tEpoca: 10 * time.Second}, silencio, func(rep reporte) error {
		guardado = true
		if rep.Verificado == nil || !*rep.Verificado || len(rep.Historia) != 4 {
			t.Errorf("reporte incompleto al guardar: %+v", rep)
		}
		select {
		case <-terminaron:
			t.Error("los nodos terminaron antes de que el reporte estuviera guardado")
		case <-time.After(150 * time.Millisecond):
		}
		return nil
	})
	if err != nil || !guardado {
		t.Fatalf("err = %v, guardado = %v", err, guardado)
	}
	<-terminaron
}

func TestProgresoRegistraLaPerdidaCadaTantasEpocas(t *testing.T) {
	dir, _ := goldDePrueba(t, 2_000, 1)
	ln := escuchar(t)
	nodos := lanzarNodos(t, ln.Addr().String(), 1)
	var mu sync.Mutex
	var lineas []string
	logf := func(f string, a ...any) {
		mu.Lock()
		lineas = append(lineas, fmt.Sprintf(f, a...))
		mu.Unlock()
	}
	if _, err := ejecutar(ln, opciones{gold: dir, nodos: 1, epocas: 7, tasa: 0.5, progreso: 3,
		tRegistro: 5 * time.Second, tEpoca: 10 * time.Second}, logf, nil); err != nil {
		t.Fatal(err)
	}
	nodos.Wait()
	mu.Lock()
	defer mu.Unlock()
	var epocas []string
	for _, l := range lineas {
		if strings.HasPrefix(l, "época ") {
			epocas = append(epocas, strings.SplitN(l, ":", 2)[0])
		}
	}
	if got := strings.Join(epocas, ","); got != "época 0,época 3,época 6" {
		t.Errorf("progreso registrado: %q", got)
	}
}

func TestCargarShardsRechazaCarpetasIncompletas(t *testing.T) {
	if _, _, _, _, err := cargarShards(t.TempDir(), "/datos"); err == nil {
		t.Error("carpeta sin shard_0.bin: esperaba error")
	}
	dir, _ := goldDePrueba(t, 2_000, 3)
	shards, d, tam, y, err := cargarShards(dir, "/datos")
	if err != nil {
		t.Fatal(err)
	}
	if len(shards) != 3 || d != 8 || tam != 256 || len(y) != 2_000 || shards[2].Ruta != "/datos/shard_2.bin" || len(shards[0].SHA256) != 64 {
		t.Errorf("shards: %+v d=%d tam=%d filas=%d", shards, d, tam, len(y))
	}
	// Un shard de otra matriz (otro número de variables) en la misma carpeta.
	otro, _ := ml.Sintetico(500, 5, 1)
	if err := gold.Escribir(filepath.Join(dir, "shard_3.bin"), gold.Archivo{Dataset: otro, TamBloque: 256, BloqueInicial: 8}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := cargarShards(dir, "/datos"); err == nil || !strings.Contains(err.Error(), "shard_3.bin") {
		t.Errorf("err = %v, quiere un error que nombre shard_3.bin", err)
	}
}
