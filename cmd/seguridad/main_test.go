package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

const origenSPA = "http://localhost:8081"

type clusterVacio struct{}

func (clusterVacio) Estado() cluster.Estado { return cluster.Estado{} }

// servidor levanta la API real (internal/api) con un almacén en memoria y las
// dos cuentas, como el coordinador en modo servicio.
func servidor(t *testing.T) (*httptest.Server, Credenciales) {
	t.Helper()
	cred := Credenciales{Secreto: strings.Repeat("s", auth.MinBytesSecreto),
		AdminCorreo: "admin@vigia.pe", AdminClave: "clave-admin-123",
		AnalistaCorreo: "analista@vigia.pe", AnalistaClave: "clave-analista-123"}
	var regs []ingest.Registro
	for i := range 40 {
		regs = append(regs, ingest.Registro{Departamento: []string{"LORETO", "PIURA"}[i%2], Ano: 2015 + i%8, Semana: 1 + i, EdadAnios: float64(i), Sexo: 'F'})
	}
	esq, err := features.Ajustar(regs, 2011, 5)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"pesos": make([]float64, esq.D()+1), "esquema": esq, "umbral": 0.5, "ajuste_prior": 1.0})
	m, err := api.CargarModelo(b)
	if err != nil {
		t.Fatal(err)
	}
	mem := almacen.NuevaMemoria()
	env := map[string]string{"VIGIA_ADMIN_CORREO": cred.AdminCorreo, "VIGIA_ADMIN_CLAVE": cred.AdminClave,
		"VIGIA_ANALISTA_CORREO": cred.AnalistaCorreo, "VIGIA_ANALISTA_CLAVE": cred.AnalistaClave}
	if _, err := api.SembrarUsuarios(context.Background(), mem, func(k string) string { return env[k] }); err != nil {
		t.Fatal(err)
	}
	f, err := auth.NuevoFirmador([]byte(cred.Secreto), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	s, err := api.Nuevo(api.Opciones{Modelo: m, Almacen: mem, Cluster: clusterVacio{}, Firmador: f, OrigenesCORS: []string{origenSPA}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv, cred
}

// cspSPA es la política que envía cmd/web.
const cspSPA = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self' http://localhost:8080"

// web simula la SPA: responde codigo con la CSP dada.
func web(t *testing.T, codigo int, csp string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if csp != "" {
			w.Header().Set("Content-Security-Policy", csp)
		}
		w.WriteHeader(codigo)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// casosEsperados: todos los casos menos los dos del límite de intentos.
const casosEsperados = 22

// TestCasosContraLaAPI: contra la API real, cada caso da lo esperado (salvo
// el límite de intentos, cuya ventana es de un minuto de reloj).
func TestCasosContraLaAPI(t *testing.T) {
	srv, cred := servidor(t)
	p := &Prueba{URL: srv.URL, Origen: origenSPA, Web: web(t, 200, cspSPA), Cred: cred, Cliente: srv.Client(), SinLimite: true}
	r, err := p.Correr()
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Casos) != casosEsperados {
		t.Errorf("%d casos; se esperaban %d", len(r.Casos), casosEsperados)
	}
	for _, c := range r.Casos {
		if !c.OK {
			t.Errorf("%s: esperado %s, obtenido %s", c.Caso, c.Esperado, c.Obtenido)
		}
	}
	if !r.TodosOK {
		t.Error("todos_ok = false")
	}
}

// TestTokenVencidoEsRechazable: el token que firma la herramienta es válido en
// todo menos en la fecha: la API lo rechaza solo por vencido.
func TestTokenVencidoEsRechazable(t *testing.T) {
	secreto := strings.Repeat("s", auth.MinBytesSecreto)
	tok, err := tokenVencido(secreto, "admin@vigia.pe")
	if err != nil {
		t.Fatal(err)
	}
	f, _ := auth.NuevoFirmador([]byte(secreto), time.Hour)
	if _, err := f.Verificar(tok); err == nil || !strings.Contains(err.Error(), "venc") {
		t.Fatalf("Verificar = %v; se esperaba un error de vencimiento", err)
	}
}

// TestUnCasoNoDesaparece (PR-102 H2): si la SPA no responde, no tiene CSP o
// responde 404, el caso de la CSP sigue en el reporte y todos_ok es falso.
func TestUnCasoNoDesaparece(t *testing.T) {
	caida := httptest.NewServer(http.NotFoundHandler())
	urlCaida := caida.URL
	caida.Close()
	for nombre, w := range map[string]string{
		"inalcanzable": urlCaida,
		"sin CSP":      web(t, 200, ""),
		"404 con CSP":  web(t, 404, cspSPA),
	} {
		srv, cred := servidor(t)
		p := &Prueba{URL: srv.URL, Origen: origenSPA, Web: w, Cred: cred, Cliente: srv.Client(), SinLimite: true}
		r, err := p.Correr()
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Casos) != casosEsperados || r.TodosOK {
			t.Errorf("%s: %d casos, todos_ok=%v; se esperaban %d y false", nombre, len(r.Casos), r.TodosOK, casosEsperados)
		}
	}
}
