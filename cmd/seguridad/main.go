// Command seguridad prueba desde fuera, contra la API en ejecución, cada
// control de seguridad de la integración SPA–API (ADR-0007) y escribe
// reports/seguridad.json con lo esperado, lo obtenido y si coincide. Lo corre
// scripts/evidencias-tb2.sh seguridad (paso 13 de docs/guias/jose-70.md); el
// JSON no se edita a mano.
//
//	JWT_SECRETO=… VIGIA_ADMIN_CORREO=… VIGIA_ADMIN_CLAVE=… \
//	VIGIA_ANALISTA_CORREO=… VIGIA_ANALISTA_CLAVE=… \
//	go run ./cmd/seguridad -url http://localhost:8080 -origen http://localhost:8081
//
// Con el secreto firma un token vencido (no se puede pedir a la API) usando el
// mismo internal/auth que el coordinador. El caso del límite de intentos va al
// final y, antes de terminar, espera a que la ventana se libere y comprueba
// que el login correcto vuelve a funcionar, para no dejar bloqueado el paso
// siguiente.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// Caso es una fila de reports/seguridad.json.
type Caso struct {
	Caso     string `json:"caso"`
	Peticion string `json:"peticion"`
	Esperado string `json:"esperado"`
	Obtenido string `json:"obtenido"`
	OK       bool   `json:"ok"`
}

// Reporte es reports/seguridad.json.
type Reporte struct {
	Generado  string `json:"generado"`
	URL       string `json:"url"`
	OrigenSPA string `json:"origen_spa"`
	Casos     []Caso `json:"casos"`
	TodosOK   bool   `json:"todos_ok"`
}

// Credenciales de las dos cuentas y el secreto JWT de la corrida.
type Credenciales struct {
	Secreto                       string
	AdminCorreo, AdminClave       string
	AnalistaCorreo, AnalistaClave string
}

func main() {
	url := flag.String("url", "http://localhost:8080", "API")
	origen := flag.String("origen", "http://localhost:8081", "origen de la SPA (uno de CORS_ORIGENES)")
	web := flag.String("web", "", "URL de la SPA para revisar su CSP (por defecto, -origen)")
	salida := flag.String("salida", "reports/seguridad.json", "reporte")
	flag.Parse()
	cred := Credenciales{
		Secreto:     os.Getenv("JWT_SECRETO"),
		AdminCorreo: os.Getenv("VIGIA_ADMIN_CORREO"), AdminClave: os.Getenv("VIGIA_ADMIN_CLAVE"),
		AnalistaCorreo: os.Getenv("VIGIA_ANALISTA_CORREO"), AnalistaClave: os.Getenv("VIGIA_ANALISTA_CLAVE"),
	}
	if cred.Secreto == "" || cred.AdminClave == "" || cred.AnalistaClave == "" {
		log.Fatal("faltan JWT_SECRETO, VIGIA_ADMIN_* o VIGIA_ANALISTA_* (los de deploy/.env de esta corrida)")
	}
	p := &Prueba{URL: strings.TrimRight(*url, "/"), Origen: *origen, Web: *web, Cred: cred,
		Cliente: &http.Client{Timeout: 15 * time.Second}, Log: log.Printf}
	if p.Web == "" {
		p.Web = p.Origen
	}
	r, err := p.Correr()
	if err != nil {
		log.Fatal(err)
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(*salida, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("escrito %s: %d casos, todos_ok=%v\n", *salida, len(r.Casos), r.TodosOK)
	if !r.TodosOK {
		os.Exit(1)
	}
}

// Prueba corre los casos contra una API.
type Prueba struct {
	URL, Origen, Web string
	Cred             Credenciales
	Cliente          *http.Client
	Log              func(string, ...any)
	// EsperaMax acota la espera a que se libere el límite de intentos.
	EsperaMax time.Duration
	// SinLimite omite los dos casos del límite de intentos (tests: la
	// ventana es de un minuto de reloj real).
	SinLimite bool
	casos     []Caso
}

func (p *Prueba) caso(nombre, peticion, esperado, obtenido string, ok bool) {
	p.casos = append(p.casos, Caso{nombre, peticion, esperado, obtenido, ok})
	marca := "✓"
	if !ok {
		marca = "✗"
	}
	p.Log("%s %-38s %-48s esperado %-28s obtenido %s", marca, nombre, peticion, esperado, obtenido)
}

// pedir hace una petición y devuelve la respuesta con el cuerpo leído.
func (p *Prueba) pedir(metodo, ruta string, cab map[string]string, cuerpo []byte) (*http.Response, []byte, error) {
	var rd io.Reader
	if cuerpo != nil {
		rd = bytes.NewReader(cuerpo)
	}
	destino := p.URL + ruta
	if strings.HasPrefix(ruta, "http") { // la SPA, en otro origen
		destino = ruta
	}
	req, err := http.NewRequest(metodo, destino, rd)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	res, err := p.Cliente.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return res, b, nil
}

func (p *Prueba) codigo(metodo, ruta string, cab map[string]string, cuerpo []byte) int {
	res, _, err := p.pedir(metodo, ruta, cab, cuerpo)
	if err != nil {
		return 0
	}
	return res.StatusCode
}

func (p *Prueba) esperarCodigo(nombre, peticion string, quiere int, metodo, ruta string, cab map[string]string, cuerpo []byte) {
	c := p.codigo(metodo, ruta, cab, cuerpo)
	p.caso(nombre, peticion, strconv.Itoa(quiere), strconv.Itoa(c), c == quiere)
}

func (p *Prueba) login(correo, clave string) (int, string, []byte) {
	cuerpo, _ := json.Marshal(map[string]string{"correo": correo, "clave": clave})
	res, b, err := p.pedir("POST", "/api/v1/auth/login", map[string]string{"Content-Type": "application/json"}, cuerpo)
	if err != nil {
		return 0, "", nil
	}
	var r struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(b, &r)
	return res.StatusCode, r.Token, b
}

func bearer(t string) map[string]string { return map[string]string{"Authorization": "Bearer " + t} }

// Correr ejecuta todos los casos, en un orden que no se estorba: el límite de
// intentos va al final.
func (p *Prueba) Correr() (Reporte, error) {
	if p.Log == nil {
		p.Log = func(string, ...any) {}
	}
	cAdmin, admin, _ := p.login(p.Cred.AdminCorreo, p.Cred.AdminClave)
	cAnal, analista, _ := p.login(p.Cred.AnalistaCorreo, p.Cred.AnalistaClave)
	if admin == "" || analista == "" {
		return Reporte{}, fmt.Errorf("no se pudo iniciar sesión (admin %d, analista %d): ¿API arriba y deploy/.env de esta corrida?", cAdmin, cAnal)
	}
	p.caso("login correcto", "POST /api/v1/auth/login (admin y analista)", "200 con token", "200 con token", true)

	// 1. Autenticación del token.
	p.esperarCodigo("sin token", "GET /api/v1/modelo", 401, "GET", "/api/v1/modelo", nil, nil)
	p.esperarCodigo("token mal formado", "GET /api/v1/modelo (Bearer abc)", 401, "GET", "/api/v1/modelo", bearer("abc"), nil)
	alterado := admin[:len(admin)-2] + map[bool]string{true: "AA", false: "BB"}[!strings.HasSuffix(admin, "AA")]
	p.esperarCodigo("firma alterada", "GET /api/v1/modelo (firma cambiada)", 401, "GET", "/api/v1/modelo", bearer(alterado), nil)
	vencido, err := tokenVencido(p.Cred.Secreto, p.Cred.AdminCorreo)
	if err != nil {
		return Reporte{}, err
	}
	p.esperarCodigo("token vencido", "GET /api/v1/modelo (exp en el pasado)", 401, "GET", "/api/v1/modelo", bearer(vencido), nil)
	otro, _ := auth.NuevoFirmador([]byte(strings.Repeat("x", auth.MinBytesSecreto)), time.Hour)
	ajeno, _, _ := otro.Firmar(p.Cred.AdminCorreo, auth.RolAdmin)
	p.esperarCodigo("token de otro secreto", "GET /api/v1/modelo (firmado con otra clave)", 401, "GET", "/api/v1/modelo", bearer(ajeno), nil)

	// 2. Autorización por rol.
	p.esperarCodigo("analista con token válido", "GET /api/v1/modelo (analista)", 200, "GET", "/api/v1/modelo", bearer(analista), nil)
	p.esperarCodigo("rol ajeno: clúster", "GET /api/v1/cluster (analista)", 403, "GET", "/api/v1/cluster", bearer(analista), nil)
	p.esperarCodigo("rol ajeno: entrenamientos", "POST /api/v1/entrenamientos (analista)", 403, "POST", "/api/v1/entrenamientos",
		map[string]string{"Authorization": "Bearer " + analista, "Content-Type": "application/json"}, []byte(`{"epocas":1}`))
	p.esperarCodigo("rol ajeno: ticket WS", "POST /api/v1/ws/ticket (analista)", 403, "POST", "/api/v1/ws/ticket", bearer(analista), nil)
	p.esperarCodigo("admin en el clúster", "GET /api/v1/cluster (admin)", 200, "GET", "/api/v1/cluster", bearer(admin), nil)

	// 3. CORS.
	pre := map[string]string{"Origin": p.Origen, "Access-Control-Request-Method": "POST",
		"Access-Control-Request-Headers": "authorization, content-type"}
	if res, _, err := p.pedir("OPTIONS", "/api/v1/predicciones", pre, nil); err == nil {
		acao := res.Header.Get("Access-Control-Allow-Origin")
		p.caso("CORS: origen de la SPA", "OPTIONS /api/v1/predicciones (Origin SPA)", "Allow-Origin = "+p.Origen,
			"Allow-Origin = "+cmpVacio(acao), acao == p.Origen)
	}
	pre["Origin"] = "https://otro-sitio.example"
	if res, _, err := p.pedir("OPTIONS", "/api/v1/predicciones", pre, nil); err == nil {
		acao := res.Header.Get("Access-Control-Allow-Origin")
		p.caso("CORS: origen ajeno", "OPTIONS /api/v1/predicciones (Origin ajeno)", "sin Allow-Origin",
			"Allow-Origin = "+cmpVacio(acao), acao == "")
	}

	// 4. Cabeceras de seguridad de la API y CSP de la SPA.
	if res, _, err := p.pedir("GET", "/api/v1/modelo", bearer(analista), nil); err == nil {
		nos := res.Header.Get("X-Content-Type-Options")
		p.caso("cabecera nosniff", "GET /api/v1/modelo", "X-Content-Type-Options: nosniff", "X-Content-Type-Options: "+cmpVacio(nos), nos == "nosniff")
	}
	if res, _, err := p.pedir("POST", "/api/v1/auth/login", map[string]string{"Content-Type": "application/json"}, []byte(`{}`)); err == nil {
		cc := res.Header.Get("Cache-Control")
		p.caso("login sin caché", "POST /api/v1/auth/login", "Cache-Control: no-store", "Cache-Control: "+cmpVacio(cc), strings.Contains(cc, "no-store"))
	}
	if p.Web == "" {
		// Sin SPA que revisar (tests de la API sola).
	} else if res, _, err := p.pedir("GET", strings.TrimRight(p.Web, "/")+"/", nil, nil); err == nil {
		csp := res.Header.Get("Content-Security-Policy")
		ok := strings.Contains(csp, "default-src 'self'") && !strings.Contains(csp, "unsafe-inline") && !strings.Contains(csp, "unsafe-eval")
		p.caso("CSP de la SPA", "GET "+p.Web+"/", "default-src 'self', sin unsafe-*", resumir(csp), ok)
	}

	// 5. WebSocket: ticket de un uso y origen.
	p.esperarCodigo("WS sin ticket", "GET /api/v1/ws (Origin SPA)", 401, "GET", "/api/v1/ws", cabWS(p.Origen), nil)
	if t, err := p.ticket(admin); err == nil {
		c := p.codigo("GET", "/api/v1/ws?ticket="+t, cabWS("https://otro-sitio.example"), nil)
		p.caso("WS de un origen ajeno", "GET /api/v1/ws?ticket=… (Origin ajeno)", "403", strconv.Itoa(c), c == 403)
	}
	if t, err := p.ticket(admin); err == nil {
		c1 := p.abrirWS(t)
		c2 := p.abrirWS(t)
		p.caso("WS con ticket válido", "GET /api/v1/ws?ticket=… (Origin SPA)", "101", strconv.Itoa(c1), c1 == 101)
		p.caso("WS: ticket reutilizado", "el mismo ticket por segunda vez", "401", strconv.Itoa(c2), c2 == 401)
	}

	// 6. Tamaño del cuerpo.
	grande := append(append([]byte(`{"x":"`), bytes.Repeat([]byte("a"), 70_000)...), []byte(`"}`)...)
	p.esperarCodigo("cuerpo demasiado grande", "POST /api/v1/predicciones de 70 KB", 413, "POST", "/api/v1/predicciones",
		map[string]string{"Authorization": "Bearer " + analista, "Content-Type": "application/json"}, grande)

	// 7. Login fallido: mismo mensaje exista o no el correo; luego, el límite.
	c1, _, b1 := p.login(p.Cred.AdminCorreo, "clave-incorrecta-1")
	c2, _, b2 := p.login("nadie@ejemplo.pe", "clave-incorrecta-2")
	p.caso("login fallido genérico", "clave errada y correo inexistente", "401 y el mismo cuerpo",
		fmt.Sprintf("%d y %d, cuerpos %s", c1, c2, map[bool]string{true: "iguales", false: "distintos"}[bytes.Equal(b1, b2)]),
		c1 == 401 && c2 == 401 && bytes.Equal(b1, b2))
	if p.SinLimite {
		return p.reporte(), nil
	}
	ultimo, intentos := 0, 2
	var reintento string
	for ; intentos < 12; intentos++ {
		res, _, err := p.pedir("POST", "/api/v1/auth/login", map[string]string{"Content-Type": "application/json"},
			[]byte(`{"correo":"nadie@ejemplo.pe","clave":"clave-incorrecta"}`))
		if err != nil {
			break
		}
		ultimo, reintento = res.StatusCode, res.Header.Get("Retry-After")
		if ultimo == 429 {
			intentos++
			break
		}
	}
	p.caso("límite de intentos", "logins fallidos seguidos desde la misma IP", "429 con Retry-After",
		fmt.Sprintf("%d tras %d intentos, Retry-After %s", ultimo, intentos, cmpVacio(reintento)), ultimo == 429 && reintento != "")
	c, err := p.esperarLiberacion(reintento)
	p.caso("login tras la ventana", "POST /api/v1/auth/login (admin) al vencer Retry-After", "200", strconv.Itoa(c), err == nil && c == 200)

	return p.reporte(), nil
}

func (p *Prueba) reporte() Reporte {
	r := Reporte{Generado: time.Now().Format(time.RFC3339), URL: p.URL, OrigenSPA: p.Origen, Casos: p.casos, TodosOK: true}
	for _, c := range p.casos {
		r.TodosOK = r.TodosOK && c.OK
	}
	return r
}

// esperarLiberacion espera Retry-After y reintenta el login correcto hasta
// que la ventana del límite se libere (como mucho EsperaMax).
func (p *Prueba) esperarLiberacion(reintento string) (int, error) {
	maxEspera := p.EsperaMax
	if maxEspera == 0 {
		maxEspera = 3 * time.Minute
	}
	s, _ := strconv.Atoi(reintento)
	limite := time.Now().Add(maxEspera)
	espera := time.Duration(max(1, s)) * time.Second
	for {
		if time.Now().Add(espera).After(limite) {
			return 0, errors.New("el límite de intentos no se liberó a tiempo")
		}
		p.Log("  esperando %v a que se libere el límite de intentos…", espera)
		time.Sleep(espera)
		c, _, _ := p.login(p.Cred.AdminCorreo, p.Cred.AdminClave)
		if c != 429 {
			return c, nil
		}
		espera = time.Second
	}
}

func (p *Prueba) ticket(admin string) (string, error) {
	_, b, err := p.pedir("POST", "/api/v1/ws/ticket", bearer(admin), nil)
	if err != nil {
		return "", err
	}
	var r struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(b, &r); err != nil || r.Ticket == "" {
		return "", fmt.Errorf("sin ticket: %s", b)
	}
	return r.Ticket, nil
}

// abrirWS hace el handshake con el origen de la SPA y cierra enseguida.
func (p *Prueba) abrirWS(ticket string) int {
	req, _ := http.NewRequest("GET", p.URL+"/api/v1/ws?ticket="+ticket, nil)
	for k, v := range cabWS(p.Origen) {
		req.Header.Set(k, v)
	}
	res, err := p.Cliente.Do(req)
	if err != nil {
		return 0
	}
	res.Body.Close() // con 101, cierra la conexión
	return res.StatusCode
}

func cabWS(origen string) map[string]string {
	return map[string]string{"Origin": origen, "Connection": "Upgrade", "Upgrade": "websocket",
		"Sec-WebSocket-Version": "13", "Sec-WebSocket-Key": "dGhlIHNhbXBsZSBub25jZQ=="}
}

// tokenVencido firma con el secreto de la corrida un token que venció hace
// una hora: exp = iat + 1 s, con el reloj una hora y un segundo atrás.
func tokenVencido(secreto, sub string) (string, error) {
	f, err := auth.NuevoFirmadorConReloj([]byte(secreto), time.Second, func() time.Time { return time.Now().Add(-time.Hour - time.Second) })
	if err != nil {
		return "", err
	}
	t, _, err := f.Firmar(sub, auth.RolAdmin)
	return t, err
}

func cmpVacio(s string) string {
	if s == "" {
		return "(ninguna)"
	}
	return s
}

func resumir(s string) string {
	if s == "" {
		return "(sin CSP)"
	}
	if len(s) > 60 {
		return s[:57] + "…"
	}
	return s
}
