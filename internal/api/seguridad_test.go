package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

const (
	origenSPA   = "http://localhost:8081"
	origenAjeno = "https://otro-sitio.example"
	evalMinima  = `{"version_esquema":1,"version_modelo":"e69395b36d2f","riesgo":{},"prioridades":[]}`
)

// apiSegura es la API completa, sin tokens agregados, con lo necesario para
// que cada ruta pueda responder 2xx a quien tiene permiso.
type apiSegura struct {
	ts              *httptest.Server
	analista, admin string // tokens
	firmador        *auth.Firmador
	idEntrenamiento string
}

func nuevaApiSegura(t *testing.T, op Opciones) *apiSegura {
	t.Helper()
	mem := almacen.NuevaMemoria()
	if err := mem.GuardarUsuario(context.Background(), almacen.Usuario{Correo: "ana@diresa.pe", Hash: hashRapido(t, "clave-de-ana-123"), Rol: "analista"}); err != nil {
		t.Fatal(err)
	}
	f := firmadorDePrueba(t)
	op.Almacen, op.Firmador = mem, f
	op.Entrenador, op.Cluster, op.Evaluacion = entrenadorListo(), clusterFalso{}, []byte(evalMinima)
	if op.OrigenesCORS == nil {
		op.OrigenesCORS = []string{origenSPA}
	}
	_, h := servidorSeguro(t, op)
	a := &apiSegura{ts: httptest.NewServer(h), firmador: f,
		analista: tokenDePrueba(t, f, "ana@diresa.pe", auth.RolAnalista),
		admin:    tokenDePrueba(t, f, "admin@vigia.pe", auth.RolAdmin)}
	t.Cleanup(a.ts.Close)
	return a
}

// pedir hace la petición con el token y las cabeceras dadas.
func (a *apiSegura) pedir(t *testing.T, metodo, ruta, token, cuerpo string, cab map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(metodo, a.ts.URL+ruta, strings.NewReader(cuerpo))
	if err != nil {
		t.Fatal(err)
	}
	if cuerpo != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, b
}

// entrenamientoTerminado lanza un entrenamiento como admin y espera a que
// termine, para que GET /api/v1/entrenamientos/{id} tenga un id real.
func (a *apiSegura) entrenamientoTerminado(t *testing.T) string {
	t.Helper()
	res, b := a.pedir(t, "POST", "/api/v1/entrenamientos", a.admin, "", nil)
	var e map[string]any
	if res.StatusCode != http.StatusAccepted || json.Unmarshal(b, &e) != nil {
		t.Fatalf("POST entrenamientos: %d %s", res.StatusCode, b)
	}
	id := e["id"].(string)
	limite := time.Now().Add(5 * time.Second)
	for e["estado"] == EnCurso {
		if time.Now().After(limite) {
			t.Fatal("el entrenamiento no terminó")
		}
		time.Sleep(10 * time.Millisecond)
		_, b = a.pedir(t, "GET", "/api/v1/entrenamientos/"+id, a.admin, "", nil)
		_ = json.Unmarshal(b, &e)
	}
	return id
}

// cuerposMatriz da una petición válida por ruta, para que quien tiene permiso
// reciba 2xx. Una ruta nueva sin muestra hace fallar el test (no se olvida).
var cuerposMatriz = map[string]string{
	"POST /api/v1/auth/login":     `{"correo":"ana@diresa.pe","clave":"clave-de-ana-123"}`,
	"POST /api/v1/predicciones":   `{"departamento":"LORETO","ano":2024,"semana":10,"edad":34,"sexo":"M"}`,
	"POST /api/v1/entrenamientos": "",
}

// Matriz ruta × rol de ADR-0007, generada desde Rutas(): publica con y sin
// token; bearer × {sin token, analista, admin} → 401 / 403 / 2xx.
func TestMatrizRoles(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	id := a.entrenamientoTerminado(t)
	quienes := []struct {
		nombre, token string
		rol           auth.Rol
	}{{"sin token", "", ""}, {"analista", a.analista, auth.RolAnalista}, {"admin", a.admin, auth.RolAdmin}}

	for _, r := range Rutas() {
		clave := r.Metodo + " " + r.Ruta
		cuerpo, hay := cuerposMatriz[clave]
		if r.Metodo == "POST" && !hay {
			t.Fatalf("%s: agrega una petición de muestra a cuerposMatriz", clave)
		}
		ruta := strings.ReplaceAll(r.Ruta, "{id}", id)
		for _, q := range quienes {
			quiere := "2xx"
			switch {
			case r.Modo == Bearer && q.token == "":
				quiere = "401"
			case r.Modo == Bearer && !slices.Contains(r.Roles, q.rol):
				quiere = "403"
			}
			res, b := a.pedir(t, r.Metodo, ruta, q.token, cuerpo, nil)
			obtenido := res.Status[:3]
			if quiere == "2xx" && res.StatusCode/100 == 2 {
				obtenido = "2xx"
			}
			t.Logf("%-36s %-9s %s → %d", clave, q.nombre, r.Modo, res.StatusCode)
			if obtenido != quiere {
				t.Errorf("%s como %s (modo %s): %d %s, quería %s", clave, q.nombre, r.Modo, res.StatusCode, b, quiere)
			}
			if quiere == "401" && !strings.HasPrefix(res.Header.Get("WWW-Authenticate"), "Bearer") {
				t.Errorf("%s: 401 sin WWW-Authenticate: Bearer", clave)
			}
		}
	}
}

func TestMatrizRolesTokenAjenoYSinFirmador(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	otro, err := auth.NuevoFirmador([]byte("otro-secreto-distinto-de-32-bytes!"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ajeno := tokenDePrueba(t, otro, "admin@vigia.pe", auth.RolAdmin)
	for _, tok := range []string{ajeno, "no-es-un-jwt", a.admin + "x"} {
		if res, b := a.pedir(t, "GET", "/api/v1/cluster", tok, "", nil); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("token %.20q…: %d %s, quería 401", tok, res.StatusCode, b)
		}
	}

	// Sin firmador la API falla cerrada: 503 en toda ruta protegida, nunca 2xx.
	_, h := servidorSeguro(t, Opciones{Cluster: clusterFalso{}})
	ts := httptest.NewServer(h)
	defer ts.Close()
	for _, r := range Rutas() {
		if r.Modo != Bearer {
			continue
		}
		req, _ := http.NewRequest(r.Metodo, ts.URL+strings.ReplaceAll(r.Ruta, "{id}", "x"), nil)
		req.Header.Set("Authorization", "Bearer "+a.admin)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s %s sin firmador: %d, quería 503", r.Metodo, r.Ruta, res.StatusCode)
		}
	}
}

// --- CORS -----------------------------------------------------------------------

func TestCORSPreflightOrigenPermitido(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	// Ruta protegida y sin Authorization: el preflight no llega a la autenticación.
	res, _ := a.pedir(t, "OPTIONS", "/api/v1/predicciones", "", "", map[string]string{
		"Origin": origenSPA, "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "authorization, content-type"})
	h := res.Header
	if res.StatusCode != http.StatusNoContent || h.Get("Access-Control-Allow-Origin") != origenSPA ||
		h.Get("Access-Control-Allow-Methods") != "GET, POST" || h.Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" ||
		h.Get("Access-Control-Max-Age") != "600" || !slices.Contains(h.Values("Vary"), "Origin") {
		t.Fatalf("preflight permitido: %d %v", res.StatusCode, h)
	}
	if h.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("nunca debe ir Access-Control-Allow-Credentials")
	}
}

func TestCORSPreflightOrigenNoPermitido(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	res, _ := a.pedir(t, "OPTIONS", "/api/v1/predicciones", "", "", map[string]string{
		"Origin": origenAjeno, "Access-Control-Request-Method": "POST"})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight ajeno: %d", res.StatusCode)
	}
	for k := range res.Header {
		if strings.HasPrefix(k, "Access-Control-Allow-") {
			t.Errorf("preflight de un origen no permitido lleva %s: %v", k, res.Header[k])
		}
	}
	// Un OPTIONS que no es preflight sigue al mux (405), no se responde aquí.
	if res, _ := a.pedir(t, "OPTIONS", "/api/v1/predicciones", "", "", nil); res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("OPTIONS sin preflight: %d, quería 405", res.StatusCode)
	}
}

// Toda respuesta a un origen permitido lleva ACAO, incluidos 401, 403 y 404;
// a uno ajeno, nunca.
func TestCORSRespuestasDecoradas(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	casos := []struct {
		ruta, token string
		codigo      int
	}{
		{"/api/v1/modelo", a.analista, 200},
		{"/api/v1/modelo", "", 401},
		{"/api/v1/cluster", a.analista, 403},
		{"/no-existe", "", 404},
	}
	for _, c := range casos {
		res, _ := a.pedir(t, "GET", c.ruta, c.token, "", map[string]string{"Origin": origenSPA})
		if res.StatusCode != c.codigo || res.Header.Get("Access-Control-Allow-Origin") != origenSPA ||
			res.Header.Get("Access-Control-Expose-Headers") != "X-Cache, Retry-After, Location" ||
			!slices.Contains(res.Header.Values("Vary"), "Origin") {
			t.Errorf("GET %s (%d): %d %v", c.ruta, c.codigo, res.StatusCode, res.Header)
		}
		res, _ = a.pedir(t, "GET", c.ruta, c.token, "", map[string]string{"Origin": origenAjeno})
		if v := res.Header.Get("Access-Control-Allow-Origin"); v != "" {
			t.Errorf("GET %s desde un origen ajeno lleva ACAO %q", c.ruta, v)
		}
	}
}

func TestCORSOrigenesValidos(t *testing.T) {
	got, err := ParsearOrigenes(" http://localhost:8081, HTTPS://Vigia.PE ,,http://localhost:8081, http://[::1]:5173, http://192.168.1.20:8081")
	if err != nil || !slices.Equal(got, []string{"http://localhost:8081", "https://vigia.pe", "http://[::1]:5173", "http://192.168.1.20:8081"}) {
		t.Fatalf("ParsearOrigenes = %v, %v", got, err)
	}
	for _, malo := range []string{"*", "null", "localhost:8081", "ftp://a.pe", "http://a.pe/", "http://a.pe/ruta",
		"http://a.pe?x=1", "http://u:p@a.pe", "http://*", "http://",
		// PR-83 H3: host vacío, comodín parcial, puerto vacío o fuera de rango.
		"http://:8081", "http://*.example.com", "http://localhost:", "http://localhost:99999", "http://localhost:0", "http://localhost:08081"} {
		if _, err := ParsearOrigenes(malo); err == nil {
			t.Errorf("ParsearOrigenes(%q) debía fallar", malo)
		}
		m, _ := CargarModelo(modeloDePrueba(t))
		if _, err := Nuevo(Opciones{Modelo: m, OrigenesCORS: []string{malo}}); err == nil {
			t.Errorf("Nuevo con origen %q debía fallar", malo)
		}
	}
}

// --- Cabeceras y límite de cuerpo ------------------------------------------------

func TestCabecerasSeguridad(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	casos := []struct {
		metodo, ruta, token string
		codigo              int
		noStore             bool
	}{
		{"GET", "/healthz", "", 200, false},
		{"GET", "/api/v1/modelo", a.analista, 200, true},
		{"GET", "/api/v1/cluster", a.admin, 200, true},
		{"GET", "/api/v1/modelo", "", 401, true},
		{"GET", "/api/v1/cluster", a.analista, 403, true},
		{"GET", "/no-existe", "", 404, false},
	}
	for _, c := range casos {
		res, _ := a.pedir(t, c.metodo, c.ruta, c.token, "", nil)
		h := res.Header
		if res.StatusCode != c.codigo || h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s %s: %d %v", c.metodo, c.ruta, res.StatusCode, h)
		}
		if c.noStore && h.Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s: Cache-Control %q, quería no-store", c.metodo, c.ruta, h.Get("Cache-Control"))
		}
	}
}

func TestCabecerasLimiteDeCuerpo413(t *testing.T) {
	// El límite lo pone el middleware: también alcanza a un handler que lea el
	// cuerpo por su cuenta (aquí, un Middleware de Opciones que lee todo).
	var errLeido error
	leerTodo := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/leer" {
				_, errLeido = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	a := nuevaApiSegura(t, Opciones{Middleware: []Middleware{leerTodo}})
	grande := `{"x":"` + strings.Repeat("a", MaxCuerpo) + `"}`
	for _, c := range []struct{ ruta, token string }{
		{"/api/v1/predicciones", a.analista},
		{"/api/v1/entrenamientos", a.admin},
		{"/api/v1/auth/login", ""},
	} {
		if res, b := a.pedir(t, "POST", c.ruta, c.token, grande, nil); res.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("POST %s con %d bytes: %d %s, quería 413", c.ruta, len(grande), res.StatusCode, b)
		}
	}
	// PR-83 H2: un cuerpo grande que no es JSON también da 413 (no 400), y la
	// respuesta conserva CORS para la SPA.
	noJSON := strings.Repeat("a", 70_000)
	for _, c := range []struct{ ruta, token string }{
		{"/api/v1/predicciones", a.analista},
		{"/api/v1/entrenamientos", a.admin},
		{"/api/v1/auth/login", ""},
	} {
		res, b := a.pedir(t, "POST", c.ruta, c.token, noJSON, map[string]string{"Origin": origenSPA})
		if res.StatusCode != http.StatusRequestEntityTooLarge || res.Header.Get("Access-Control-Allow-Origin") != origenSPA {
			t.Errorf("POST %s con 70 000 bytes no JSON: %d %s (ACAO %q), quería 413 con CORS",
				c.ruta, res.StatusCode, b, res.Header.Get("Access-Control-Allow-Origin"))
		}
	}
	// Y la autorización va antes que el límite: sin token, 401 aunque el cuerpo sea grande.
	if res, _ := a.pedir(t, "POST", "/api/v1/predicciones", "", noJSON, nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST grande sin token: %d, quería 401", res.StatusCode)
	}
	a.pedir(t, "POST", "/leer", "", grande, nil)
	var mb *http.MaxBytesError
	if !errors.As(errLeido, &mb) || mb.Limit != MaxCuerpo {
		t.Fatalf("lectura directa del cuerpo: err %v, quería MaxBytesError de %d", errLeido, MaxCuerpo)
	}
}

// --- OpenAPI ---------------------------------------------------------------------

// operacionOpenAPI resume lo que el contrato declara de una operación.
type operacionOpenAPI struct {
	bearer  bool
	codigos []string
}

// seguridadDelOpenAPI lee, por operación, si declara bearerAuth y qué
// códigos de respuesta tiene; y si components define el esquema bearerAuth.
func seguridadDelOpenAPI(t *testing.T) (map[string]*operacionOpenAPI, bool) {
	t.Helper()
	f, err := os.Open("../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ops := map[string]*operacionOpenAPI{}
	var (
		seccion, subseccion, ruta, esquemaSeg string
		op                                    *operacionOpenAPI
		enRespuestas, tipoHTTP, esBearer      bool
	)
	sangria := func(l string) int { return len(l) - len(strings.TrimLeft(l, " ")) }
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		s := strings.TrimSpace(l)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		n := sangria(l)
		switch {
		case n == 0:
			seccion = strings.TrimSuffix(s, ":")
		case seccion == "paths" && n == 2:
			ruta, op = strings.TrimSuffix(s, ":"), nil
		case seccion == "paths" && n == 4:
			metodo := strings.TrimSuffix(s, ":")
			op = &operacionOpenAPI{}
			ops[strings.ToUpper(metodo)+" "+ruta] = op
			enRespuestas = false
		case seccion == "paths" && op != nil && n == 6:
			enRespuestas = s == "responses:"
			if strings.HasPrefix(s, "security:") && strings.Contains(s, "bearerAuth") {
				op.bearer = true
			}
		case seccion == "paths" && op != nil && n == 8 && enRespuestas && strings.HasPrefix(s, `"`):
			op.codigos = append(op.codigos, s[1:4])
		case seccion == "components" && n == 2:
			subseccion, esquemaSeg = strings.TrimSuffix(s, ":"), ""
		case seccion == "components" && subseccion == "securitySchemes" && n == 4:
			esquemaSeg = strings.TrimSuffix(s, ":")
		case seccion == "components" && subseccion == "securitySchemes" && n == 6 && esquemaSeg == "bearerAuth":
			tipoHTTP = tipoHTTP || s == "type: http"
			esBearer = esBearer || s == "scheme: bearer"
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return ops, tipoHTTP && esBearer
}

func TestOpenAPISeguridad(t *testing.T) {
	ops, esquemaOK := seguridadDelOpenAPI(t)
	if !esquemaOK {
		t.Fatal("components.securitySchemes no define bearerAuth (type: http, scheme: bearer)")
	}
	for _, r := range Rutas() {
		if !r.Documentar {
			continue
		}
		clave := r.Metodo + " " + r.Ruta
		op := ops[clave]
		if op == nil {
			t.Errorf("%s no está en el OpenAPI", clave)
			continue
		}
		switch {
		case r.Modo == Publica && op.bearer:
			t.Errorf("%s es pública y el OpenAPI le exige bearerAuth", clave)
		case r.Modo == Bearer && !op.bearer:
			t.Errorf("%s exige token y el OpenAPI no declara security: bearerAuth", clave)
		}
		if r.Metodo == "POST" && !slices.Contains(op.codigos, "413") {
			t.Errorf("%s es POST y no declara 413 (límite de %d bytes)", clave, MaxCuerpo)
		}
		if r.Modo != Bearer {
			continue
		}
		for _, c := range []string{"401", "403"} {
			if !slices.Contains(op.codigos, c) {
				t.Errorf("%s no declara la respuesta %s (códigos: %v)", clave, c, op.codigos)
			}
		}
	}
}

// pendientesADR son filas de la matriz de ADR-0007 que aún no tienen ruta
// (WebSocket y demo, #65). Una fila que no está aquí ni en Rutas() falla.
var pendientesADR = []string{"POST /api/v1/ws/ticket", "GET /api/v1/ws", "POST /api/v1/demo/nodos/{nombre}/cortar"}

// La política de Rutas() es la que fija la tabla de ADR-0007: cambiar un rol
// en el código sin cambiar el ADR (o al revés) hace fallar este test.
func TestMatrizRolesCoincideConADR(t *testing.T) {
	b, err := os.ReadFile("../../docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md")
	if err != nil {
		t.Fatal(err)
	}
	type politica struct{ publica, analista, admin bool }
	adr := map[string]politica{}
	enTabla := false
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "### Matriz ruta × rol") {
			enTabla = true
			continue
		}
		if !enTabla || !strings.HasPrefix(l, "| `") {
			if enTabla && strings.HasPrefix(l, "#") {
				break
			}
			continue
		}
		celdas := strings.Split(l, "|")
		if len(celdas) < 5 {
			t.Fatalf("fila de la matriz mal formada: %q", l)
		}
		p := politica{
			publica:  strings.HasPrefix(strings.TrimSpace(celdas[2]), "✓"),
			analista: strings.HasPrefix(strings.TrimSpace(celdas[3]), "✓"),
			admin:    strings.HasPrefix(strings.TrimSpace(celdas[4]), "✓"),
		}
		partes := strings.Split(celdas[1], "`")
		for i := 1; i < len(partes); i += 2 {
			metodo, ruta, _ := strings.Cut(partes[i], " ")
			if metodo != "GET" && metodo != "POST" {
				continue // `-demo`, `ticket`: texto, no rutas
			}
			ruta, _, _ = strings.Cut(ruta, "?")
			adr[metodo+" "+ruta] = p
		}
	}
	if len(adr) == 0 {
		t.Fatal("no se encontró la matriz en ADR-0007")
	}
	for _, r := range Rutas() {
		clave := r.Metodo + " " + r.Ruta
		p, ok := adr[clave]
		if !ok {
			t.Errorf("%s no figura en la matriz de ADR-0007", clave)
			continue
		}
		codigo := politica{
			publica:  r.Modo == Publica,
			analista: r.Modo == Publica || slices.Contains(r.Roles, auth.RolAnalista),
			admin:    r.Modo == Publica || slices.Contains(r.Roles, auth.RolAdmin),
		}
		if codigo != p {
			t.Errorf("%s: código %+v, ADR-0007 %+v", clave, codigo, p)
		}
		delete(adr, clave)
	}
	for clave := range adr {
		if !slices.Contains(pendientesADR, clave) {
			t.Errorf("ADR-0007 tiene %s y Rutas() no", clave)
		}
	}
}

// PR-83 H3: mutar los roles que devuelve Rutas() no cambia la política.
func TestMatrizRolesRutasDevuelveCopias(t *testing.T) {
	for _, r := range Rutas() {
		for i := range r.Roles {
			r.Roles[i] = "intruso"
		}
	}
	for _, r := range Rutas() {
		for _, rol := range r.Roles {
			if !rol.Valido() {
				t.Fatalf("%s %s: el rol %q se filtró de una llamada anterior", r.Metodo, r.Ruta, rol)
			}
		}
	}
	if !slices.Equal(rolesTodos, []auth.Rol{auth.RolAdmin, auth.RolAnalista}) || !slices.Equal(soloAdmin, []auth.Rol{auth.RolAdmin}) {
		t.Fatalf("roles globales alterados: %v %v", rolesTodos, soloAdmin)
	}
}

// PR-83 H4: un 429 del login y un 500 por pánico también llevan CORS, para que
// la SPA lea Retry-After y el mensaje de error.
func TestCORSEnRespuestas429Y500(t *testing.T) {
	panico := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/panico" {
				panic("prueba")
			}
			next.ServeHTTP(w, r)
		})
	}
	a := nuevaApiSegura(t, Opciones{Middleware: []Middleware{panico}, Log: func(string, ...any) {}})
	var res *http.Response
	for range 6 {
		res, _ = a.pedir(t, "POST", "/api/v1/auth/login", "", `{"correo":"ana@diresa.pe","clave":"clave-equivocada"}`, map[string]string{"Origin": origenSPA})
	}
	if res.StatusCode != http.StatusTooManyRequests || res.Header.Get("Retry-After") == "" ||
		res.Header.Get("Access-Control-Allow-Origin") != origenSPA ||
		!strings.Contains(res.Header.Get("Access-Control-Expose-Headers"), "Retry-After") {
		t.Fatalf("429: %d %v", res.StatusCode, res.Header)
	}
	res, b := a.pedir(t, "GET", "/panico", "", "", map[string]string{"Origin": origenSPA})
	if res.StatusCode != http.StatusInternalServerError || res.Header.Get("Access-Control-Allow-Origin") != origenSPA ||
		res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("500 por pánico: %d %s %v", res.StatusCode, b, res.Header)
	}
}

// PR-83 H4: Cache-Control de cada tipo de respuesta (ADR-0007, fila Cabeceras).
func TestCabecerasCacheSegunRuta(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	for _, ruta := range []string{"/api/v1/riesgo", "/api/v1/evaluacion"} {
		res, _ := a.pedir(t, "GET", ruta, a.analista, "", map[string]string{"Origin": origenSPA, "Accept-Encoding": "gzip"})
		vary := res.Header.Values("Vary")
		if res.StatusCode != 200 || res.Header.Get("Cache-Control") != "private, no-cache" ||
			!slices.Contains(vary, "Origin") || !slices.Contains(vary, "Accept-Encoding") {
			t.Errorf("GET %s: %d Cache-Control %q Vary %v", ruta, res.StatusCode, res.Header.Get("Cache-Control"), vary)
		}
	}
	res, _ := a.pedir(t, "POST", "/api/v1/auth/login", "", cuerposMatriz["POST /api/v1/auth/login"], nil)
	if res.StatusCode != 200 || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("login: %d Cache-Control %q, quería no-store", res.StatusCode, res.Header.Get("Cache-Control"))
	}
}
