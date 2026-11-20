package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// modeloYGold arma un modelo.json con el formato de cmd/entrenar y un gold
// sintético con el mismo número de variables, partido en shards.
func modeloYGold(t *testing.T, filas, shards int) (modelo []byte, dir string, train gold.Archivo) {
	t.Helper()
	var regs []ingest.Registro
	for i := range 40 {
		d := "LORETO"
		if i%2 == 0 {
			d = "PIURA"
		}
		regs = append(regs, ingest.Registro{Departamento: d, Ano: 2015 + i%8, Semana: 1 + i, EdadAnios: float64(i), Sexo: 'F'})
	}
	esq, err := features.Ajustar(regs, 2011, 5)
	if err != nil {
		t.Fatal(err)
	}
	modelo, err = json.Marshal(map[string]any{
		"pesos": make([]float64, esq.D()+1), "esquema": esq, "umbral": 0.5, "ajuste_prior": 0.3,
		"generado": "2026-11-19T00:00:00-05:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	dir = t.TempDir()
	ds, _ := ml.Sintetico(filas, esq.D(), 11)
	train = gold.Archivo{Dataset: ds, TamBloque: 256}
	partes, err := gold.Particionar(train, shards)
	if err != nil {
		t.Fatal(err)
	}
	for s, p := range partes {
		if err := gold.Escribir(filepath.Join(dir, fmt.Sprintf("shard_%d.bin", s)), p); err != nil {
			t.Fatal(err)
		}
	}
	return modelo, dir, train
}

// servicioDePrueba levanta el modo API con nodos que se conectan cuando el
// test los lanza. Devuelve la URL base, la dirección TCP del clúster y una
// función que detiene el servicio (como SIGTERM) y devuelve su error.
func servicioDePrueba(t *testing.T, modelo []byte, op opciones) (base, dirCluster string, detener func() error) {
	t.Helper()
	ln, lnHTTP := escuchar(t), escuchar(t)
	ctx, cancelar := context.WithCancel(context.Background())
	fin := make(chan error, 1)
	go func() { fin <- ejecutarAPI(ctx, ln, lnHTTP, op, modelo, silencio) }()
	detenido := false
	var errFin error
	detener = func() error {
		if !detenido {
			detenido = true
			cancelar()
			select {
			case errFin = <-fin:
			case <-time.After(30 * time.Second):
				t.Fatal("el servicio no se detuvo tras cancelar el contexto")
			}
		}
		return errFin
	}
	t.Cleanup(func() { detener() })
	return "http://" + lnHTTP.Addr().String(), ln.Addr().String(), detener
}

func pedirJSON(t *testing.T, metodo, url, cuerpo string) (int, map[string]any, http.Header) {
	t.Helper()
	req, err := http.NewRequest(metodo, url, strings.NewReader(cuerpo))
	if err != nil {
		t.Fatal(err)
	}
	if cuerpo != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var v map[string]any
	if strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s %s: JSON inválido %q", metodo, url, b)
		}
	}
	return res.StatusCode, v, res.Header
}

// esperar consulta f hasta que devuelva true o venza el plazo.
func esperar(t *testing.T, que string, f func() bool) {
	t.Helper()
	limite := time.Now().Add(30 * time.Second)
	for !f() {
		if time.Now().After(limite) {
			t.Fatalf("no se cumplió a tiempo: %s", que)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func entrenamientoTerminado(t *testing.T, base, ubicacion string) map[string]any {
	t.Helper()
	var ent map[string]any
	esperar(t, "el entrenamiento termina", func() bool {
		_, ent, _ = pedirJSON(t, "GET", base+ubicacion, "")
		return ent["estado"] != "en_curso"
	})
	return ent
}

// H2 del PR #39: la API vive en el coordinador, entrena en el clúster real
// (TCP) y publica un modelo idéntico bit a bit al secuencial.
func TestServicioEntrenaEnElClusterYPublicaElModelo(t *testing.T) {
	modelo, dir, train := modeloYGold(t, 6_000, 3)
	op := opciones{gold: dir, nodos: 3, lambda: 1e-3, tRegistro: 10 * time.Second, tEpoca: 10 * time.Second}
	base, dirCluster, detener := servicioDePrueba(t, modelo, op)

	// La API responde antes de que lleguen los nodos.
	esperar(t, "la API responde", func() bool {
		res, err := http.Get(base + "/healthz")
		if err == nil {
			res.Body.Close()
		}
		return err == nil && res.StatusCode == http.StatusOK
	})
	_, m0, _ := pedirJSON(t, "GET", base+"/api/v1/modelo", "")

	nodos := lanzarNodos(t, dirCluster, 3)
	esperar(t, "los 3 nodos aparecen en GET /api/v1/cluster", func() bool {
		_, c, _ := pedirJSON(t, "GET", base+"/api/v1/cluster", "")
		ns, _ := c["nodos"].([]any)
		return len(ns) == 3
	})
	// Un entrenamiento pedido justo al registrarse el último nodo puede llegar
	// antes de que EsperarNodos devuelva: se reintenta hasta que lo acepte.
	var ent map[string]any
	esperar(t, "el clúster acepta un entrenamiento", func() bool {
		codigo, _, h := pedirJSON(t, "POST", base+"/api/v1/entrenamientos", `{"epocas": 25, "tasa": 0.5, "lambda": 0.001}`)
		if codigo != http.StatusAccepted {
			t.Fatalf("POST /api/v1/entrenamientos = %d, se esperaba 202", codigo)
		}
		ent = entrenamientoTerminado(t, base, h.Get("Location"))
		return ent["estado"] == "terminado" || !strings.Contains(fmt.Sprint(ent["error"]), "todavía espera")
	})
	if ent["estado"] != "terminado" || ent["epoca"] != 25.0 {
		t.Fatalf("entrenamiento = %v, se esperaba terminado con 25 épocas", ent)
	}

	_, m1, _ := pedirJSON(t, "GET", base+"/api/v1/modelo", "")
	if m1["version"] != ent["version_modelo"] || m1["version"] == m0["version"] {
		t.Fatalf("modelo publicado %v; entrenamiento %v; inicial %v", m1["version"], ent["version_modelo"], m0["version"])
	}
	pos, neg := ml.PesosBalanceados(train.Y)
	cfg := ml.Config{TamBloque: train.TamBloque, Lambda: 1e-3, PesoPos: pos, PesoNeg: neg}
	ref := ml.Entrenar(ml.NuevoSecuencial(train.Dataset, cfg), train.D, ml.Entrenamiento{Epocas: 25, Tasa: 0.5})
	vars := m1["variables"].([]any)
	for j, v := range vars {
		if p := v.(map[string]any)["peso"].(float64); math.Float64bits(p) != math.Float64bits(ref.W[j]) {
			t.Fatalf("peso %d publicado %v, secuencial %v", j, p, ref.W[j])
		}
	}
	if s := m1["sesgo"].(float64); math.Float64bits(s) != math.Float64bits(ref.W[train.D]) {
		t.Fatalf("sesgo publicado %v, secuencial %v", s, ref.W[train.D])
	}

	// Swagger UI y contrato servidos por el coordinador.
	for _, ruta := range []string{"/docs", "/docs/openapi.yaml"} {
		res, err := http.Get(base + ruta)
		if err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: %v %v", ruta, res, err)
		}
		res.Body.Close()
	}

	// Cierre como SIGTERM: el servicio termina sin error y los nodos reciben `fin`.
	if err := detener(); err != nil {
		t.Fatalf("cierre: %v", err)
	}
	listo := make(chan struct{})
	go func() { nodos.Wait(); close(listo) }()
	select {
	case <-listo:
	case <-time.After(10 * time.Second):
		t.Fatal("los nodos no terminaron tras el cierre del coordinador")
	}
}

func TestServicioRechazaUnLambdaDistintoAlDelCluster(t *testing.T) {
	modelo, dir, _ := modeloYGold(t, 2_000, 2)
	op := opciones{gold: dir, nodos: 2, lambda: 1e-3, tRegistro: 10 * time.Second, tEpoca: 10 * time.Second}
	base, dirCluster, _ := servicioDePrueba(t, modelo, op)
	lanzarNodos(t, dirCluster, 2)
	var ent map[string]any
	esperar(t, "el entrenamiento falla por lambda y no por nodos", func() bool {
		_, _, h := pedirJSON(t, "POST", base+"/api/v1/entrenamientos", `{"epocas": 5, "lambda": 0.5}`)
		ent = entrenamientoTerminado(t, base, h.Get("Location"))
		return !strings.Contains(fmt.Sprint(ent["error"]), "todavía espera")
	})
	if ent["estado"] != "fallido" || !strings.Contains(fmt.Sprint(ent["error"]), "lambda=0.001") {
		t.Fatalf("entrenamiento = %v, se esperaba fallido por lambda", ent)
	}
}

func TestServicioSinNodosRegistradosNoEntrena(t *testing.T) {
	modelo, dir, _ := modeloYGold(t, 2_000, 2)
	op := opciones{gold: dir, nodos: 2, lambda: 1e-4, tRegistro: 10 * time.Second, tEpoca: 10 * time.Second}
	base, _, detener := servicioDePrueba(t, modelo, op)
	esperar(t, "la API responde", func() bool {
		res, err := http.Get(base + "/healthz")
		if err == nil {
			res.Body.Close()
		}
		return err == nil
	})
	codigo, _, h := pedirJSON(t, "POST", base+"/api/v1/entrenamientos", "")
	if codigo != http.StatusAccepted {
		t.Fatalf("POST = %d", codigo)
	}
	ent := entrenamientoTerminado(t, base, h.Get("Location"))
	if ent["estado"] != "fallido" || !strings.Contains(fmt.Sprint(ent["error"]), "todavía espera") {
		t.Fatalf("entrenamiento = %v, se esperaba fallido por falta de nodos", ent)
	}
	// Las predicciones siguen funcionando con el modelo inicial.
	codigo, p, _ := pedirJSON(t, "POST", base+"/api/v1/predicciones",
		`{"departamento":"LORETO","ano":2024,"semana":10,"edad":30,"tipo_edad":"A","sexo":"F"}`)
	if codigo != http.StatusOK || p["riesgo"] == nil {
		t.Fatalf("predicción = %d %v", codigo, p)
	}
	if err := detener(); err != nil {
		t.Fatalf("cierre: %v", err)
	}
}

func TestServicioRechazaModeloYShardsIncompatibles(t *testing.T) {
	modelo, _, _ := modeloYGold(t, 1_000, 2)
	dirOtro, _ := goldDePrueba(t, 1_000, 2) // 8 variables, el modelo tiene otras
	ln, lnHTTP := escuchar(t), escuchar(t)
	err := ejecutarAPI(context.Background(), ln, lnHTTP, opciones{gold: dirOtro, nodos: 2}, modelo, silencio)
	if err == nil || !strings.Contains(err.Error(), "variables") {
		t.Fatalf("err = %v, se esperaba el aviso de variables incompatibles", err)
	}
	if _, err := net.Dial("tcp", lnHTTP.Addr().String()); err == nil {
		t.Fatal("el listener HTTP quedó abierto")
	}
}
