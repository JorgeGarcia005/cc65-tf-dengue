package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// modeloDePrueba arma un modelo.json con el mismo formato que escribe
// cmd/entrenar: dos departamentos con columna propia y el resto en OTROS.
func modeloDePrueba(t *testing.T) []byte {
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
	pesos := make([]float64, esq.D()+1)
	for j := range pesos {
		pesos[j] = 0.1 * float64(j%7-3)
	}
	b, err := json.Marshal(map[string]any{
		"pesos": pesos, "esquema": esq, "umbral": 0.5, "ajuste_prior": 1.9, "generado": "2026-11-07T01:02:22-05:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// servidorDePrueba levanta la API. Si op no trae Firmador, crea uno y el
// servidor de prueba agrega un token de admin a toda petición que no traiga
// Authorization: así las pruebas funcionales anteriores a la autorización
// (#61) siguen probando lo suyo. La autorización se prueba sin esa ayuda en
// seguridad_test.go (servidorSeguro).
func servidorDePrueba(t *testing.T, op Opciones) (*Servidor, *httptest.Server) {
	t.Helper()
	inyectar := ""
	if op.Firmador == nil {
		op.Firmador = firmadorDePrueba(t)
		inyectar = tokenDePrueba(t, op.Firmador, "admin@vigia.pe", auth.RolAdmin)
	}
	s, h := servidorSeguro(t, op)
	if inyectar != "" {
		interno := h
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				r.Header.Set("Authorization", "Bearer "+inyectar)
			}
			interno.ServeHTTP(w, r)
		})
	}
	ts := httptest.NewServer(h)
	t.Cleanup(func() { ts.Close() })
	return s, ts
}

// servidorSeguro crea el servidor con el modelo de prueba y devuelve su
// Handler tal cual, sin tokens agregados.
func servidorSeguro(t *testing.T, op Opciones) (*Servidor, http.Handler) {
	t.Helper()
	m, err := CargarModelo(modeloDePrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	op.Modelo = m
	s, err := Nuevo(op)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Cerrar)
	return s, s.Handler()
}

func firmadorDePrueba(t *testing.T) *auth.Firmador {
	t.Helper()
	f, err := auth.NuevoFirmador([]byte(secretoLogin), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func tokenDePrueba(t *testing.T, f *auth.Firmador, correo string, rol auth.Rol) string {
	t.Helper()
	tok, _, err := f.Firmar(correo, rol)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func pedir(t *testing.T, metodo, url, cuerpo string) (int, map[string]any) {
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
	if len(b) > 0 && strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s %s: JSON inválido %q", metodo, url, b)
		}
	}
	return res.StatusCode, v
}

// --- Predicción -------------------------------------------------------------

func TestPrediccionCoincideConElModeloEntrenado(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	m, _ := CargarModelo(modeloDePrueba(t))
	cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones",
		`{"departamento":"loreto","ano":2024,"semana":10,"edad":34,"sexo":"m"}`)
	if cod != http.StatusOK {
		t.Fatalf("código %d: %v", cod, v)
	}
	// La misma cuenta que cmd/entrenar: Llenar → σ(w·x + b) → calibrar.
	x := make([]float64, m.Esquema.D())
	m.Esquema.Llenar(ingest.Registro{Departamento: "LORETO", Ano: 2024, Semana: 10, EdadAnios: 34, Sexo: 'M'}, x)
	z := m.Pesos[len(x)]
	for j := range x {
		z += m.Pesos[j] * x[j]
	}
	p := ml.Sigmoide(z)
	if got := v["probabilidad_ponderada"].(float64); math.Float64bits(got) != math.Float64bits(p) {
		t.Fatalf("probabilidad_ponderada %v, se esperaba %v", got, p)
	}
	if got := v["riesgo"].(float64); math.Float64bits(got) != math.Float64bits(ml.Calibrar(p, 1.9)) {
		t.Fatalf("riesgo %v, se esperaba %v", got, ml.Calibrar(p, 1.9))
	}
	clase := 0.0
	if p >= 0.5 {
		clase = 1
	}
	if v["clase"].(float64) != clase || v["version_modelo"] != m.Version || v["departamento_modelo"] != "LORETO" {
		t.Fatalf("respuesta inesperada: %v", v)
	}
}

func TestPrediccionDepartamentoSinColumnaVaAOtros(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones",
		`{"departamento":"Tumbes","ano":2024,"semana":10,"edad":8,"tipo_edad":"M","sexo":"F"}`)
	if cod != http.StatusOK || v["departamento_modelo"] != features.OtrosDepartamentos {
		t.Fatalf("código %d: %v", cod, v)
	}
}

func TestPrediccionEntradaInvalidaDa400EnEspanol(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	casos := map[string]string{
		"semana 0":           `{"departamento":"LORETO","ano":2024,"semana":0,"edad":30,"sexo":"F"}`,
		"semana 54":          `{"departamento":"LORETO","ano":2024,"semana":54,"edad":30,"sexo":"F"}`,
		"edad negativa":      `{"departamento":"LORETO","ano":2024,"semana":5,"edad":-1,"sexo":"F"}`,
		"edad > 110":         `{"departamento":"LORETO","ano":2024,"semana":5,"edad":111,"sexo":"F"}`,
		"tipo_edad X":        `{"departamento":"LORETO","ano":2024,"semana":5,"edad":3,"tipo_edad":"X","sexo":"F"}`,
		"sexo inválido":      `{"departamento":"LORETO","ano":2024,"semana":5,"edad":30,"sexo":"Z"}`,
		"sin departamento":   `{"ano":2024,"semana":5,"edad":30,"sexo":"F"}`,
		"año antes del gold": `{"departamento":"LORETO","ano":2005,"semana":5,"edad":30,"sexo":"F"}`,
		"campo desconocido":  `{"departamento":"LORETO","ano":2024,"semana":5,"edad":30,"sexo":"F","clase":1}`,
		"JSON roto":          `{"departamento":`,
	}
	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones", cuerpo)
			msg, _ := v["error"].(string)
			if cod != http.StatusBadRequest || msg == "" {
				t.Fatalf("código %d, cuerpo %v", cod, v)
			}
			if strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(msg, "unknown field") {
				t.Fatalf("mensaje no está en español: %q", msg)
			}
		})
	}
}

func TestPrediccionCuerpoDemasiadoGrande(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	cuerpo := `{"departamento":"` + strings.Repeat("A", 1<<20) + `"}`
	cod, _ := pedir(t, "POST", ts.URL+"/api/v1/predicciones", cuerpo)
	if cod != http.StatusBadRequest && cod != http.StatusRequestEntityTooLarge {
		t.Fatalf("código %d", cod)
	}
}

// --- Rutas --------------------------------------------------------------------

type entrenadorFalso struct {
	mu        sync.Mutex
	soltar    chan struct{}
	pedidos   []ParametrosEntrenamiento
	falla     error
	resultado func(d int) []float64
}

func (e *entrenadorFalso) Entrenar(ctx context.Context, p ParametrosEntrenamiento, progreso func(int, float64)) ([]float64, error) {
	e.mu.Lock()
	e.pedidos = append(e.pedidos, p)
	e.mu.Unlock()
	progreso(1, 0.69)
	select {
	case <-e.soltar:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if e.falla != nil {
		return nil, e.falla
	}
	progreso(p.Epocas, 0.5)
	return e.resultado(p.D), nil
}

type clusterFalso struct{}

func (clusterFalso) Estado() cluster.Estado {
	return cluster.Estado{Epocas: 7, Perdida: 0.67, Nodos: []cluster.EstadoNodo{{Nombre: "nodo-1", Sano: true, Carga: 1}}}
}

func esperar(t *testing.T, cond func() bool) {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(limite) {
			t.Fatal("tiempo agotado")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRutas(t *testing.T) {
	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 {
		w := make([]float64, d+1)
		w[0] = 1
		return w
	}}
	s, ts := servidorDePrueba(t, Opciones{Entrenador: ent, Cluster: clusterFalso{}})
	versionInicial := s.ModeloActual().Version

	if cod, v := pedir(t, "GET", ts.URL+"/healthz", ""); cod != 200 || v["estado"] != "ok" {
		t.Fatalf("healthz: %d %v", cod, v)
	}
	cod, v := pedir(t, "GET", ts.URL+"/api/v1/modelo", "")
	if cod != 200 || v["version"] != versionInicial || len(v["variables"].([]any)) != s.ModeloActual().Esquema.D() {
		t.Fatalf("modelo: %d %v", cod, v)
	}
	cod, v = pedir(t, "GET", ts.URL+"/api/v1/cluster", "")
	if cod != 200 || v["epocas"].(float64) != 7 || len(v["nodos"].([]any)) != 1 {
		t.Fatalf("cluster: %d %v", cod, v)
	}

	// Parámetros inválidos.
	if cod, _ := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":0}`); cod != 400 {
		t.Fatalf("épocas 0 debía dar 400, dio %d", cod)
	}
	cod, v = pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":20,"tasa":2}`)
	if cod != http.StatusAccepted || v["id"] == nil {
		t.Fatalf("crear entrenamiento: %d %v", cod, v)
	}
	id := v["id"].(string)
	// Solo uno a la vez.
	if cod, _ := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":5}`); cod != http.StatusConflict {
		t.Fatalf("segundo entrenamiento simultáneo debía dar 409, dio %d", cod)
	}
	esperar(t, func() bool {
		_, v := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
		return v["epoca"].(float64) == 1
	})
	close(ent.soltar)
	esperar(t, func() bool {
		_, v := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
		return v["estado"] == "terminado"
	})
	_, v = pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
	nueva := s.ModeloActual().Version
	if nueva == versionInicial || v["version_modelo"] != nueva || v["perdida"].(float64) != 0.5 {
		t.Fatalf("el entrenamiento terminado debía publicar un modelo nuevo: %v (antes %s)", v, versionInicial)
	}
	if ent.pedidos[0].Epocas != 20 || ent.pedidos[0].Tasa != 2 || ent.pedidos[0].Lambda != DefectoLambda {
		t.Fatalf("parámetros recibidos: %+v", ent.pedidos[0])
	}
	if cod, _ := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/no-existe", ""); cod != 404 {
		t.Fatalf("id inexistente debía dar 404, dio %d", cod)
	}
	if cod, _ := pedir(t, "DELETE", ts.URL+"/api/v1/modelo", ""); cod != http.StatusMethodNotAllowed {
		t.Fatalf("método no permitido debía dar 405, dio %d", cod)
	}
}

func TestRutasEntrenamientoFallidoNoPublicaModelo(t *testing.T) {
	ent := &entrenadorFalso{soltar: make(chan struct{}), falla: errors.New("cluster: no quedan nodos sanos")}
	close(ent.soltar)
	s, ts := servidorDePrueba(t, Opciones{Entrenador: ent})
	antes := s.ModeloActual().Version
	_, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", `{"epocas":3}`)
	id := v["id"].(string)
	esperar(t, func() bool {
		_, v := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
		return v["estado"] == "fallido"
	})
	_, v = pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+id, "")
	if !strings.Contains(v["error"].(string), "no quedan nodos") || s.ModeloActual().Version != antes {
		t.Fatalf("estado: %v", v)
	}
}

func TestRutasSinClusterNiEntrenadorDan503(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	for _, r := range [][2]string{{"GET", "/api/v1/cluster"}, {"POST", "/api/v1/entrenamientos"}} {
		cod, v := pedir(t, r[0], ts.URL+r[1], `{"epocas":1}`)
		if cod != http.StatusServiceUnavailable || v["error"] == nil {
			t.Fatalf("%s %s: %d %v", r[0], r[1], cod, v)
		}
	}
}

func TestRutasSwaggerUI(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	res, err := http.Get(ts.URL + "/docs")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(strings.ToLower(string(b)), "swagger") {
		t.Fatalf("/docs: %d", res.StatusCode)
	}
	res, err = http.Get(ts.URL + "/docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.HasPrefix(b, []byte("openapi: 3")) {
		t.Fatalf("/docs/openapi.yaml: %d %q", res.StatusCode, b[:min(len(b), 40)])
	}
}

// --- OpenAPI ------------------------------------------------------------------

// rutasDelOpenAPI lee, sin dependencias, las parejas "MÉTODO /ruta" de la
// sección paths de un OpenAPI 3 indentado con dos espacios.
func rutasDelOpenAPI(t *testing.T, ruta string) []string {
	t.Helper()
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rutas []string
	enPaths, actual := false, ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		switch {
		case l == "paths:":
			enPaths = true
		case enPaths && len(l) > 0 && l[0] != ' ' && l[0] != '#':
			enPaths = false
		case enPaths && strings.HasPrefix(l, "  /") && strings.HasSuffix(l, ":"):
			actual = strings.TrimSuffix(strings.TrimSpace(l), ":")
		case enPaths && actual != "" && strings.HasPrefix(l, "    ") && !strings.HasPrefix(l, "     "):
			m := strings.TrimSuffix(strings.TrimSpace(l), ":")
			if slices.Contains([]string{"get", "post", "put", "patch", "delete"}, m) {
				rutas = append(rutas, strings.ToUpper(m)+" "+actual)
			}
		}
	}
	return rutas
}

func TestOpenAPICubreLasRutas(t *testing.T) {
	documentadas := rutasDelOpenAPI(t, "../../docs/api/openapi.yaml")
	var servidas []string
	for _, r := range Rutas() {
		if r.Documentar {
			servidas = append(servidas, r.Metodo+" "+r.Ruta)
		}
	}
	slices.Sort(documentadas)
	slices.Sort(servidas)
	if !slices.Equal(documentadas, servidas) {
		t.Fatalf("OpenAPI y mux no coinciden:\n documentadas: %v\n servidas:     %v", documentadas, servidas)
	}
	// El YAML servido en /docs es el mismo archivo de docs/api.
	disco, err := os.ReadFile("../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disco, especificacion()) {
		t.Fatal("el OpenAPI embebido difiere de docs/api/openapi.yaml")
	}
}

// --- Cierre ordenado ------------------------------------------------------------

func TestCierreOrdenadoEsperaPeticionesEnCurso(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	empezo, soltar := make(chan struct{}), make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(empezo)
		<-soltar
		fmt.Fprint(w, "listo")
	})
	ctx, cancelar := context.WithCancel(context.Background())
	termino := make(chan error, 1)
	srv := NuevoHTTP(h)
	if srv.ReadHeaderTimeout <= 0 || srv.ReadHeaderTimeout > 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v", srv.ReadHeaderTimeout)
	}
	go func() { termino <- Servir(ctx, srv, ln, 5*time.Second) }()

	respuesta := make(chan string, 1)
	go func() {
		res, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			respuesta <- "error: " + err.Error()
			return
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		respuesta <- string(b)
	}()
	<-empezo
	cancelar() // equivale a recibir SIGTERM
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-termino:
		t.Fatalf("Servir terminó con una petición en curso: %v", err)
	default:
	}
	close(soltar)
	if r := <-respuesta; r != "listo" {
		t.Fatalf("la petición en curso no terminó bien: %q", r)
	}
	if err := <-termino; err != nil {
		t.Fatalf("Servir: %v", err)
	}
	if _, err := net.DialTimeout("tcp", ln.Addr().String(), 200*time.Millisecond); err == nil {
		t.Fatal("el puerto sigue aceptando conexiones tras el cierre")
	}
}

// --- Ronda 1 del juez (PR-39) -------------------------------------------------

func ptr[T any](v T) *T { return &v }

// H1: sin edad o con edad null es 400; edad 0 explícita (recién nacido) es 200.
func TestPrediccionEdadAusenteONulaDa400(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	base := `"departamento":"LORETO","ano":2024,"semana":10,"sexo":"F"`
	for nombre, cuerpo := range map[string]string{
		"sin edad":  `{` + base + `}`,
		"edad null": `{` + base + `,"edad":null}`,
	} {
		cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones", cuerpo)
		if cod != http.StatusBadRequest || !strings.Contains(v["error"].(string), "edad") {
			t.Fatalf("%s: código %d, %v", nombre, cod, v)
		}
	}
	if cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones", `{`+base+`,"edad":0}`); cod != http.StatusOK {
		t.Fatalf("edad 0 explícita: código %d, %v", cod, v)
	}
}

// H3: un esquema con desviación nula o ausente no se carga.
func TestPrediccionModeloConDesviacionNulaNoSeCarga(t *testing.T) {
	for _, campo := range []string{"desv_edad", "desv_ano"} {
		var m map[string]any
		if err := json.Unmarshal(modeloDePrueba(t), &m); err != nil {
			t.Fatal(err)
		}
		m["esquema"].(map[string]any)[campo] = 0
		b, _ := json.Marshal(m)
		if _, err := CargarModelo(b); err == nil || !strings.Contains(err.Error(), campo) {
			t.Fatalf("%s = 0: se esperaba error que lo nombre, obtuve %v", campo, err)
		}
		delete(m["esquema"].(map[string]any), campo)
		b, _ = json.Marshal(m)
		if _, err := CargarModelo(b); err == nil {
			t.Fatalf("sin %s: se esperaba error", campo)
		}
	}
}

// H3: si la respuesta no se puede codificar, el cliente recibe 500 con cuerpo,
// no un 200 vacío.
func TestPrediccionRespuestaNoCodificableDa500(t *testing.T) {
	rec := httptest.NewRecorder()
	responder(rec, http.StatusOK, map[string]float64{"riesgo": math.NaN()})
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "error") {
		t.Fatalf("código %d, cuerpo %q", rec.Code, rec.Body.String())
	}
}

// H4: el cuerpo de POST /entrenamientos es opcional y usa los valores por defecto.
func TestRutasEntrenamientoSinCuerpoUsaDefectos(t *testing.T) {
	ent := &entrenadorFalso{soltar: make(chan struct{}), resultado: func(d int) []float64 { return make([]float64, d+1) }}
	close(ent.soltar)
	_, ts := servidorDePrueba(t, Opciones{Entrenador: ent})
	cod, v := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", "")
	if cod != http.StatusAccepted {
		t.Fatalf("sin cuerpo: código %d, %v", cod, v)
	}
	esperar(t, func() bool {
		ent.mu.Lock()
		defer ent.mu.Unlock()
		return len(ent.pedidos) == 1
	})
	ent.mu.Lock()
	p := ent.pedidos[0]
	ent.mu.Unlock()
	if p.Epocas != DefectoEpocas || p.Tasa != DefectoTasa || p.Lambda != DefectoLambda {
		t.Fatalf("parámetros: %+v", p)
	}
}

// H5: contenido después del objeto JSON es 400; el espacio en blanco final no.
func TestPrediccionContenidoSobranteDa400(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	ok := `{"departamento":"LORETO","ano":2024,"semana":10,"edad":3,"sexo":"F"}`
	for nombre, cuerpo := range map[string]string{
		"llave sobrante":   ok + `}`,
		"segundo objeto":   ok + `{}`,
		"texto al final":   ok + ` hola`,
		"arreglo sobrante": ok + `[]`,
	} {
		cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones", cuerpo)
		if cod != http.StatusBadRequest {
			t.Fatalf("%s: código %d, %v", nombre, cod, v)
		}
	}
	if cod, v := pedir(t, "POST", ts.URL+"/api/v1/predicciones", ok+" \n\t "); cod != http.StatusOK {
		t.Fatalf("espacio final: código %d, %v", cod, v)
	}
}

// H7: antes del primer registro de nodos, "nodos" es [] y no null.
type clusterVacio struct{}

func (clusterVacio) Estado() cluster.Estado { return cluster.Estado{} }

func TestRutasClusterVacioDevuelveArregloDeNodos(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{Cluster: clusterVacio{}})
	res, err := http.Get(ts.URL + "/api/v1/cluster")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !bytes.Contains(b, []byte(`"nodos":[]`)) {
		t.Fatalf("se esperaba \"nodos\":[], cuerpo %s", b)
	}
}

// H6: en los mapas YAML en línea ({...}) cada elemento debe ser "clave: valor";
// un texto sin comillas con coma parte el mapa y deja una clave basura.
func TestOpenAPIMapasEnLineaBienFormados(t *testing.T) {
	b, err := os.ReadFile("../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for i, l := range strings.Split(string(b), "\n") {
		ini := strings.Index(l, ": {")
		if ini < 0 || !strings.HasSuffix(strings.TrimSpace(l), "}") {
			continue
		}
		cuerpo := strings.TrimSpace(l[ini+2:])
		for _, el := range partirFlujo(cuerpo[1 : len(cuerpo)-1]) {
			clave, valor, ok := strings.Cut(el, ": ")
			if !ok || strings.TrimSpace(clave) == "" {
				t.Errorf("openapi.yaml:%d: elemento %q no es \"clave: valor\"", i+1, el)
				continue
			}
			v := strings.TrimSpace(valor)
			if !strings.HasPrefix(v, `"`) && !strings.HasPrefix(v, "[") && !strings.HasPrefix(v, "{") && strings.Contains(v, ": ") {
				t.Errorf("openapi.yaml:%d: valor %q debe ir entre comillas", i+1, v)
			}
		}
	}
}

// partirFlujo separa por comas de primer nivel, respetando comillas y corchetes.
func partirFlujo(s string) []string {
	var partes []string
	prof, comillas, ini := 0, false, 0
	for i, c := range s {
		switch {
		case c == '"':
			comillas = !comillas
		case comillas:
		case c == '[' || c == '{':
			prof++
		case c == ']' || c == '}':
			prof--
		case c == ',' && prof == 0:
			partes = append(partes, strings.TrimSpace(s[ini:i]))
			ini = i + 1
		}
	}
	return append(partes, strings.TrimSpace(s[ini:]))
}

// entrenadorLento ignora la cancelación durante un rato antes de devolver,
// como una época del clúster que tiene que terminar.
type entrenadorLento struct {
	empezo    chan struct{}
	retraso   time.Duration
	terminado atomic.Bool
}

func (e *entrenadorLento) Entrenar(ctx context.Context, p ParametrosEntrenamiento, progreso func(int, float64)) ([]float64, error) {
	close(e.empezo)
	<-ctx.Done()
	time.Sleep(e.retraso)
	e.terminado.Store(true)
	return nil, ctx.Err()
}

// PR-39 H9: Cerrar espera a la goroutine del entrenamiento y después no
// admite otro.
func TestCerrarEsperaAlEntrenamientoEnCursoYNoAdmiteOtro(t *testing.T) {
	ent := &entrenadorLento{empezo: make(chan struct{}), retraso: 300 * time.Millisecond}
	s, ts := servidorDePrueba(t, Opciones{Entrenador: ent})
	cod, cuerpo := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", "")
	if cod != http.StatusAccepted {
		t.Fatalf("POST = %d %v", cod, cuerpo)
	}
	<-ent.empezo
	s.Cerrar()
	if !ent.terminado.Load() {
		t.Fatal("Cerrar volvió antes de que terminara el entrenamiento")
	}
	_, e := pedir(t, "GET", ts.URL+"/api/v1/entrenamientos/"+cuerpo["id"].(string), "")
	if e["estado"] != Fallido {
		t.Fatalf("estado tras el cierre = %v, se esperaba %q registrado antes de volver", e["estado"], Fallido)
	}
	if cod, _ := pedir(t, "POST", ts.URL+"/api/v1/entrenamientos", ""); cod != http.StatusServiceUnavailable {
		t.Fatalf("POST tras Cerrar = %d, se esperaba 503", cod)
	}
	s.Cerrar() // idempotente
}
