package api

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
)

// demoFalsa conoce un solo nodo, «nodo-1».
type demoFalsa struct{ cortes *atomic.Int64 }

func (d demoFalsa) CortarNodo(nombre string) error {
	if nombre != "nodo-1" {
		return fmt.Errorf("%w: %q", cluster.ErrNodoDesconocido, nombre)
	}
	if d.cortes != nil {
		d.cortes.Add(1)
	}
	return nil
}

// wsCliente es un cliente WebSocket mínimo para las pruebas: handshake HTTP/1.1
// a mano y lectura de frames del servidor (sin máscara, RFC 6455 §5.1).
type wsCliente struct {
	conn net.Conn
	r    *bufio.Reader
}

// handshake abre GET /api/v1/ws?ticket=… con el Origin dado. Devuelve el código
// HTTP y, si es 101, el cliente conectado.
func (a *apiSegura) handshake(t *testing.T, tk, origen string) (int, *wsCliente) {
	t.Helper()
	conn, err := net.Dial("tcp", strings.TrimPrefix(a.ts.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	req := "GET /api/v1/ws?ticket=" + tk + " HTTP/1.1\r\nHost: vigia\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n"
	if origen != "" {
		req += "Origin: " + origen + "\r\n"
	}
	if _, err := io.WriteString(conn, req+"\r\n"); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(conn)
	res, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusSwitchingProtocols {
		res.Body.Close()
		conn.Close()
		return res.StatusCode, nil
	}
	if res.Header.Get("Sec-WebSocket-Accept") != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("Sec-WebSocket-Accept = %q", res.Header.Get("Sec-WebSocket-Accept"))
	}
	c := &wsCliente{conn: conn, r: r}
	t.Cleanup(func() { conn.Close() })
	return res.StatusCode, c
}

// frame lee un frame del servidor: opcode y carga.
func (c *wsCliente) frame(t *testing.T) (byte, []byte) {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var cab [2]byte
	if _, err := io.ReadFull(c.r, cab[:]); err != nil {
		t.Fatalf("leer frame: %v", err)
	}
	n := uint64(cab[1] & 0x7f)
	switch n {
	case 126:
		var ext [2]byte
		io.ReadFull(c.r, ext[:])
		n = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		io.ReadFull(c.r, ext[:])
		n = binary.BigEndian.Uint64(ext[:])
	}
	p := make([]byte, n)
	if _, err := io.ReadFull(c.r, p); err != nil {
		t.Fatalf("leer carga: %v", err)
	}
	return cab[0] & 0x0f, p
}

// pedirTicket pide un ticket con el token dado.
func (a *apiSegura) pedirTicket(t *testing.T, token string) (int, RespuestaTicket) {
	t.Helper()
	res, b := a.pedir(t, "POST", "/api/v1/ws/ticket", token, "", nil)
	var rt RespuestaTicket
	if res.StatusCode == 200 {
		if err := json.Unmarshal(b, &rt); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode, rt
}

// ticketDe emite un ticket directo en el almacén (para probar el rol que la
// API no deja emitir).
func (a *apiSegura) ticketDe(t *testing.T, rol auth.Rol) string {
	t.Helper()
	tk, _, err := a.s.tickets.Emitir(auth.Reclamos{Sub: "x@vigia.pe", Rol: rol})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

// matrizTicket: modo Ticket × {sin ticket, ticket de analista, ticket de admin}.
func (a *apiSegura) matrizTicket(t *testing.T, clave string, r Ruta) {
	t.Helper()
	_, adm := a.pedirTicket(t, a.admin)
	for _, c := range []struct {
		quien, tk string
		quiere    int
	}{{"sin ticket", "", 401}, {"analista", a.ticketDe(t, auth.RolAnalista), 403}, {"admin", adm.Ticket, 101}} {
		cod, _ := a.handshake(t, c.tk, origenSPA)
		t.Logf("%-36s %-9s %s → %d", clave, c.quien, r.Modo, cod)
		if cod != c.quiere {
			t.Errorf("%s con %s: %d, quería %d", clave, c.quien, cod, c.quiere)
		}
	}
}

func TestTicketSoloAdminUnUsoYVence(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	if cod, _ := a.pedirTicket(t, ""); cod != 401 {
		t.Fatalf("ticket sin token: %d", cod)
	}
	if cod, _ := a.pedirTicket(t, a.analista); cod != 403 {
		t.Fatalf("ticket como analista: %d, quería 403", cod)
	}
	antes := time.Now()
	cod, rt := a.pedirTicket(t, a.admin)
	if cod != 200 || len(rt.Ticket) != 43 { // 32 bytes en base64url sin relleno
		t.Fatalf("ticket como admin: %d %+v", cod, rt)
	}
	if exp := time.Unix(rt.Expira, 0); exp.Before(antes.Add(VidaTicket-time.Second)) || exp.After(antes.Add(VidaTicket+time.Second)) {
		t.Fatalf("expira %v, se esperaba ≈ ahora + %v", exp, VidaTicket)
	}
	if cod, _ := a.handshake(t, rt.Ticket, origenSPA); cod != 101 {
		t.Fatalf("primer uso: %d", cod)
	}
	if cod, _ := a.handshake(t, rt.Ticket, origenSPA); cod != 401 {
		t.Fatalf("reuso: %d, quería 401", cod)
	}

	// Vencido: el reloj del almacén avanza 30 s.
	_, rt = a.pedirTicket(t, a.admin)
	a.s.tickets.mu.Lock()
	a.s.tickets.ahora = func() time.Time { return time.Now().Add(VidaTicket) }
	a.s.tickets.mu.Unlock()
	if cod, _ := a.handshake(t, rt.Ticket, origenSPA); cod != 401 {
		t.Fatalf("ticket vencido: %d, quería 401", cod)
	}
	// Bearer en el handshake no sirve: el WebSocket solo acepta ticket.
	if res, _ := a.pedir(t, "GET", "/api/v1/ws", a.admin, "", map[string]string{"Origin": origenSPA}); res.StatusCode != 401 {
		t.Fatalf("WS con Bearer y sin ticket: %d, quería 401", res.StatusCode)
	}
}

func TestTicketTopeDePendientesY429(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	for i := range MaxTicketsPendientes {
		if cod, _ := a.pedirTicket(t, a.admin); cod != 200 {
			t.Fatalf("ticket %d: %d", i, cod)
		}
	}
	res, _ := a.pedir(t, "POST", "/api/v1/ws/ticket", a.admin, "", nil)
	if res.StatusCode != 429 || res.Header.Get("Retry-After") == "" {
		t.Fatalf("ticket %d: %d %v, quería 429 con Retry-After", MaxTicketsPendientes+1, res.StatusCode, res.Header)
	}
	// Los vencidos no cuentan: al pasar 30 s se puede volver a pedir.
	a.s.tickets.mu.Lock()
	a.s.tickets.ahora = func() time.Time { return time.Now().Add(VidaTicket) }
	a.s.tickets.mu.Unlock()
	if cod, _ := a.pedirTicket(t, a.admin); cod != 200 {
		t.Fatalf("tras vencer los pendientes: %d", cod)
	}
}

// Dos handshakes concurrentes con el mismo ticket: solo uno pasa.
func TestTicketConsumoAtomico(t *testing.T) {
	tks := nuevosTickets()
	id, _, err := tks.Emitir(auth.Reclamos{Sub: "admin@vigia.pe", Rol: auth.RolAdmin})
	if err != nil {
		t.Fatal(err)
	}
	var ok atomic.Int64
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, valido := tks.Consumir(id); valido {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("%d consumos válidos del mismo ticket, quería 1", ok.Load())
	}
}

// El Origin se valida antes que el ticket: un sitio ajeno no puede quemarlo.
func TestWSOrigenFueraDeLaListaDa403SinGastarElTicket(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	_, rt := a.pedirTicket(t, a.admin)
	for _, origen := range []string{"", origenAjeno, "null", origenSPA + "/"} {
		if cod, _ := a.handshake(t, rt.Ticket, origen); cod != 403 {
			t.Errorf("Origin %q: %d, quería 403", origen, cod)
		}
	}
	cod, c := a.handshake(t, rt.Ticket, origenSPA)
	if cod != 101 {
		t.Fatalf("el ticket se gastó con los orígenes rechazados: %d", cod)
	}
	// Conectado: recibe lo que el coordinador publica en el hub.
	esperarClientes(t, a, 1)
	a.s.op.Hub.Publicar([]byte(`{"v":1,"tipo":"epoca","epoca":3}`))
	op, p := c.frame(t)
	if op != 0x1 || string(p) != `{"v":1,"tipo":"epoca","epoca":3}` {
		t.Fatalf("frame %#x %q", op, p)
	}
}

// Cerrar el servidor (SIGTERM) despide a los clientes con 1001.
func TestWSCerrarEnvia1001(t *testing.T) {
	a := nuevaApiSegura(t, Opciones{})
	_, rt := a.pedirTicket(t, a.admin)
	_, c := a.handshake(t, rt.Ticket, origenSPA)
	esperarClientes(t, a, 1)
	a.s.Cerrar()
	op, p := c.frame(t)
	if op != 0x8 || len(p) < 2 || binary.BigEndian.Uint16(p) != 1001 {
		t.Fatalf("cierre: opcode %#x carga %v, quería close 1001", op, p)
	}
	if n := a.s.op.Hub.Clientes(); n != 0 {
		t.Fatalf("quedaron %d clientes en el hub", n)
	}
}

func esperarClientes(t *testing.T, a *apiSegura, n int) {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for a.s.op.Hub.Clientes() != n {
		if time.Now().After(limite) {
			t.Fatalf("el hub tiene %d clientes, se esperaban %d", a.s.op.Hub.Clientes(), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Sin -demo, la ruta de corte no existe y ritmo_ms se rechaza.
func TestDemoSinFlagNoExiste(t *testing.T) {
	m, err := CargarModelo(modeloDePrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	f := firmadorDePrueba(t)
	ent := entrenadorListo()
	s, err := Nuevo(Opciones{Modelo: m, Firmador: f, Entrenador: ent})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cerrar()
	a := &apiSegura{s: s, admin: tokenDePrueba(t, f, "admin@vigia.pe", auth.RolAdmin)}
	a.ts = httptestServer(t, s.Handler())
	for _, tok := range []string{"", a.admin} {
		if res, b := a.pedir(t, "POST", "/api/v1/demo/nodos/nodo-1/cortar", tok, "", nil); res.StatusCode != 404 {
			t.Errorf("cortar sin -demo (token %v): %d %s, quería 404", tok != "", res.StatusCode, b)
		}
	}
	if res, b := a.pedir(t, "POST", "/api/v1/entrenamientos", a.admin, `{"ritmo_ms":50}`, nil); res.StatusCode != 400 || !strings.Contains(string(b), "modo demo") {
		t.Fatalf("ritmo_ms sin -demo: %d %s, quería 400", res.StatusCode, b)
	}
	if len(ent.pedidos) != 0 {
		t.Fatalf("se lanzó un entrenamiento: %+v", ent.pedidos)
	}
}

func TestDemoCortarNodoYRitmo(t *testing.T) {
	var cortes atomic.Int64
	ent := entrenadorListo()
	a := nuevaApiSegura(t, Opciones{Demo: demoFalsa{cortes: &cortes}, Entrenador: ent})
	if res, b := a.pedir(t, "POST", "/api/v1/demo/nodos/nodo-1/cortar", a.admin, "", nil); res.StatusCode != 202 || cortes.Load() != 1 {
		t.Fatalf("cortar nodo-1: %d %s (cortes %d)", res.StatusCode, b, cortes.Load())
	}
	if res, _ := a.pedir(t, "POST", "/api/v1/demo/nodos/nodo-9/cortar", a.admin, "", nil); res.StatusCode != 404 {
		t.Fatalf("cortar un nodo inexistente: %d, quería 404", res.StatusCode)
	}
	if res, _ := a.pedir(t, "POST", "/api/v1/demo/nodos/nodo-1/cortar", a.analista, "", nil); res.StatusCode != 403 || cortes.Load() != 1 {
		t.Fatalf("cortar como analista: %d (cortes %d), quería 403", res.StatusCode, cortes.Load())
	}
	for cuerpo, quiere := range map[string]int{`{"ritmo_ms":-1}`: 400, fmt.Sprintf(`{"ritmo_ms":%d}`, MaxRitmo+1): 400} {
		if res, b := a.pedir(t, "POST", "/api/v1/entrenamientos", a.admin, cuerpo, nil); res.StatusCode != quiere {
			t.Errorf("%s: %d %s, quería %d", cuerpo, res.StatusCode, b, quiere)
		}
	}
	if res, b := a.pedir(t, "POST", "/api/v1/entrenamientos", a.admin, `{"epocas":3,"ritmo_ms":50}`, nil); res.StatusCode != 202 {
		t.Fatalf("ritmo_ms 50 en demo: %d %s", res.StatusCode, b)
	}
	esperar(t, func() bool {
		ent.mu.Lock()
		defer ent.mu.Unlock()
		return len(ent.pedidos) == 1
	})
	ent.mu.Lock()
	defer ent.mu.Unlock()
	if p := ent.pedidos[0]; p.RitmoMs != 50 || p.Epocas != 3 {
		t.Fatalf("parámetros que llegaron al entrenador: %+v", p)
	}
}

func httptestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts
}
