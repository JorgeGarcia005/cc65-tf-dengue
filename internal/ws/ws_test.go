package ws

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

const claveRFC = "dGhlIHNhbXBsZSBub25jZQ==" // RFC 6455 §1.3

func TestHandshakeVectorRFC(t *testing.T) {
	if got := ClaveAceptacion(claveRFC); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("Sec-WebSocket-Accept = %q", got)
	}
}

// cliente es un cliente WebSocket mínimo para las pruebas. Los métodos *E
// devuelven error y se usan dentro de goroutines (t.Fatal solo vale en la del
// test); los demás envuelven a esos y llaman a t.Fatal.
type cliente struct {
	nc net.Conn
	r  *bufio.Reader
}

// conectarE abre TCP y manda el handshake. En cab, "_proto" cambia la versión
// HTTP y un valor con "\r\n" agrega una segunda línea de la misma cabecera.
func conectarE(url string, cab map[string]string) (*cliente, *http.Response, error) {
	nc, err := net.Dial("tcp", strings.TrimPrefix(url, "http://"))
	if err != nil {
		return nil, nil, err
	}
	base := map[string]string{"Upgrade": "websocket", "Connection": "keep-alive, Upgrade",
		"Sec-WebSocket-Version": "13", "Sec-WebSocket-Key": claveRFC, "Origin": "http://localhost:8081"}
	for k, v := range cab {
		base[k] = v
	}
	proto := "HTTP/1.1"
	if p, ok := base["_proto"]; ok {
		proto = p
		delete(base, "_proto")
	}
	req := "GET /ws " + proto + "\r\nHost: prueba\r\n"
	for k, v := range base {
		if v != "" {
			req += k + ": " + v + "\r\n"
		}
	}
	if _, err := nc.Write([]byte(req + "\r\n")); err != nil {
		nc.Close()
		return nil, nil, err
	}
	r := bufio.NewReader(nc)
	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		nc.Close()
		return nil, nil, err
	}
	_ = nc.SetDeadline(time.Now().Add(5 * time.Second))
	return &cliente{nc, r}, resp, nil
}

func conectar(t *testing.T, url string, cab map[string]string) (*cliente, *http.Response) {
	t.Helper()
	c, resp, err := conectarE(url, cab)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.nc.Close() })
	return c, resp
}

// enviarE escribe un frame; enmascarado como exige RFC 6455 salvo que mascara sea false.
func (c *cliente) enviarE(primerByte byte, p []byte, mascara bool, largoDeclarado int) error {
	b := []byte{primerByte}
	m := byte(0)
	if mascara {
		m = 0x80
	}
	switch n := largoDeclarado; {
	case n <= 125:
		b = append(b, m|byte(n))
	case n <= 0xFFFF:
		b = binary.BigEndian.AppendUint16(append(b, m|126), uint16(n))
	default:
		b = binary.BigEndian.AppendUint64(append(b, m|127), uint64(n))
	}
	if mascara {
		k := [4]byte{1, 2, 3, 4}
		b = append(b, k[:]...)
		for i, x := range p {
			b = append(b, x^k[i%4])
		}
	} else {
		b = append(b, p...)
	}
	_, err := c.nc.Write(b)
	return err
}

func (c *cliente) enviar(t *testing.T, primerByte byte, p []byte, mascara bool, largoDeclarado int) {
	t.Helper()
	if err := c.enviarE(primerByte, p, mascara, largoDeclarado); err != nil {
		t.Fatal(err)
	}
}

func (c *cliente) recibirE() (op byte, p []byte, err error) {
	var cab [2]byte
	if _, err = io.ReadFull(c.r, cab[:]); err != nil {
		return
	}
	if cab[1]&0x80 != 0 {
		return 0, nil, errors.New("el servidor envió un frame enmascarado")
	}
	n := uint64(cab[1] & 0x7F)
	switch n {
	case 126:
		var b [2]byte
		if _, err = io.ReadFull(c.r, b[:]); err != nil {
			return
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err = io.ReadFull(c.r, b[:]); err != nil {
			return
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	p = make([]byte, n)
	_, err = io.ReadFull(c.r, p)
	return cab[0] & 0x0F, p, err
}

func (c *cliente) recibir(t *testing.T) (byte, []byte) {
	t.Helper()
	op, p, err := c.recibirE()
	if err != nil {
		t.Fatal(err)
	}
	return op, p
}

// servidorEco acepta y devuelve en texto cada mensaje recibido.
func servidorEco(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, func(o string) bool { return o == "http://localhost:8081" })
		if err != nil {
			return
		}
		for {
			m, err := c.LeerMensaje()
			if err != nil {
				return
			}
			_ = c.EscribirTexto(m)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// esperar repite cond hasta que sea cierta o venza el plazo.
func esperar(t *testing.T, plazo time.Duration, que string, cond func() bool) {
	t.Helper()
	fin := time.Now().Add(plazo)
	for !cond() {
		if time.Now().After(fin) {
			t.Fatalf("no se cumplió en %v: %s", plazo, que)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// sinFugas falla si, pasado un plazo, hay más goroutines que antes.
func sinFugas(t *testing.T, antes int) {
	t.Helper()
	fin := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > antes && time.Now().Before(fin) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > antes {
		t.Fatalf("fuga de goroutines: %d antes, %d después", antes, n)
	}
}

// servidorHub sirve cada conexión con h.Servir.
func servidorHub(h *Hub, ping time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, nil)
		if err != nil {
			return
		}
		h.Servir(c, h.Suscribir(), ping)
	}))
}

func TestHandshakeRespuestas(t *testing.T) {
	srv := servidorEco(t)
	casos := []struct {
		nombre string
		cab    map[string]string
		codigo int
	}{
		{"válido", nil, 101},
		{"sin Upgrade", map[string]string{"Upgrade": ""}, 400},
		{"versión 8", map[string]string{"Sec-WebSocket-Version": "8"}, 426},
		{"clave corta", map[string]string{"Sec-WebSocket-Key": "YWJj"}, 400},
		{"origen ajeno", map[string]string{"Origin": "https://evil.example"}, 403},
		{"sin origen", map[string]string{"Origin": ""}, 403},
		{"Connection en dos líneas", map[string]string{"Connection": "keep-alive\r\nConnection: Upgrade"}, 101},
		{"HTTP/1.0", map[string]string{"_proto": "HTTP/1.0"}, 400},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, resp := conectar(t, srv.URL, c.cab)
			if resp.StatusCode != c.codigo {
				t.Fatalf("código = %d, quería %d", resp.StatusCode, c.codigo)
			}
			if c.codigo == 101 && resp.Header.Get("Sec-WebSocket-Accept") != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
				t.Fatalf("accept = %q", resp.Header.Get("Sec-WebSocket-Accept"))
			}
			if c.codigo == 426 && resp.Header.Get("Sec-WebSocket-Version") != "13" {
				t.Fatal("426 sin Sec-WebSocket-Version: 13")
			}
		})
	}
	resp, err := http.Post(srv.URL, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 405 {
		t.Fatalf("POST = %d, quería 405", resp.StatusCode)
	}
}
func TestFramesEcoPingCierre(t *testing.T) {
	srv := servidorEco(t)
	c, _ := conectar(t, srv.URL, nil)

	c.enviar(t, 0x81, []byte("hola"), true, 4)
	if op, p := c.recibir(t); op != opTexto || string(p) != "hola" {
		t.Fatalf("eco = %#x %q", op, p)
	}
	largo := []byte(strings.Repeat("x", 300)) // largo de 16 bits
	c.enviar(t, 0x81, largo, true, len(largo))
	if _, p := c.recibir(t); len(p) != 300 {
		t.Fatalf("eco largo = %d bytes", len(p))
	}
	c.enviar(t, 0x89, []byte("p1"), true, 2)
	if op, p := c.recibir(t); op != opPong || string(p) != "p1" {
		t.Fatalf("pong = %#x %q", op, p)
	}
	c.enviar(t, 0x88, binary.BigEndian.AppendUint16(nil, 1000), true, 2)
	if op, p := c.recibir(t); op != opCierre || binary.BigEndian.Uint16(p) != CierreNormal {
		t.Fatalf("cierre = %#x %v", op, p)
	}
}
func TestFramesRechazados(t *testing.T) {
	casos := []struct {
		nombre string
		enviar func(t *testing.T, c *cliente)
		codigo uint16
	}{
		{"sin máscara", func(t *testing.T, c *cliente) { c.enviar(t, 0x81, []byte("x"), false, 1) }, CierreProtocolo},
		{"RSV1", func(t *testing.T, c *cliente) { c.enviar(t, 0xC1, []byte("x"), true, 1) }, CierreProtocolo},
		{"opcode reservado", func(t *testing.T, c *cliente) { c.enviar(t, 0x83, []byte("x"), true, 1) }, CierreProtocolo},
		{"ping de 126 bytes", func(t *testing.T, c *cliente) { c.enviar(t, 0x89, make([]byte, 126), true, 126) }, CierreProtocolo},
		{"más de 64 KiB", func(t *testing.T, c *cliente) { c.enviar(t, 0x82, nil, true, MaxMensaje+1) }, CierreMuyGrande},
		{"binario", func(t *testing.T, c *cliente) { c.enviar(t, 0x82, []byte{1}, true, 1) }, CierreNoSoportado},
		{"fragmentado", func(t *testing.T, c *cliente) { c.enviar(t, 0x01, []byte("a"), true, 1) }, CierreNoSoportado},
		{"continuación suelta", func(t *testing.T, c *cliente) { c.enviar(t, 0x80, []byte("a"), true, 1) }, CierreProtocolo},
	}
	srv := servidorEco(t)
	for _, k := range casos {
		t.Run(k.nombre, func(t *testing.T) {
			c, _ := conectar(t, srv.URL, nil)
			k.enviar(t, c)
			op, p := c.recibir(t)
			if op != opCierre || len(p) < 2 || binary.BigEndian.Uint16(p) != k.codigo {
				t.Fatalf("respuesta = %#x %v, quería cierre %d", op, p, k.codigo)
			}
		})
	}
}
func TestLentoNoBloqueaAlPublicador(t *testing.T) {
	h := NuevoHub(4)
	lento := h.Suscribir() // nunca lee
	inicio := time.Now()
	for i := 0; i < 1000; i++ {
		h.Publicar([]byte("epoca"))
	}
	if d := time.Since(inicio); d > 100*time.Millisecond {
		t.Fatalf("publicar 1000 mensajes tardó %v con un cliente que no lee", d)
	}
	if lento.Descartados() != 996 || h.Descartados() != 996 || len(lento.Mensajes()) != 4 {
		t.Fatalf("descartados cliente=%d hub=%d pendientes=%d", lento.Descartados(), h.Descartados(), len(lento.Mensajes()))
	}
}

func TestHubConcurrenteSinFugas(t *testing.T) {
	antes := runtime.NumGoroutine()
	h := NuevoHub(8)
	srv := servidorHub(h, 5*time.Millisecond) // pings mientras dura la prueba
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, resp, err := conectarE(srv.URL, nil)
			if err != nil || resp.StatusCode != 101 {
				t.Errorf("conexión: %v %v", err, resp)
				return
			}
			defer c.nc.Close()
			for { // responde pings y espera "fin"
				op, p, err := c.recibirE()
				if err != nil {
					t.Errorf("recibir: %v", err)
					return
				}
				if op == opPing {
					if err := c.enviarE(0x8A, p, true, len(p)); err != nil {
						t.Errorf("pong: %v", err)
						return
					}
				}
				if op == opTexto && string(p) == "fin" {
					break
				}
			}
			if err := c.enviarE(0x88, binary.BigEndian.AppendUint16(nil, 1000), true, 2); err != nil {
				t.Errorf("cierre: %v", err)
				return
			}
			for { // pueden llegar mensajes en vuelo antes del eco del cierre
				op, _, err := c.recibirE()
				if err != nil {
					t.Errorf("eco del cierre: %v", err)
					return
				}
				if op == opCierre {
					return
				}
			}
		}()
	}
	esperar(t, 3*time.Second, "16 clientes suscritos", func() bool {
		h.Publicar([]byte("epoca"))
		return h.Clientes() == 16
	})
	for i := 0; i < 50; i++ {
		h.Publicar([]byte("epoca"))
	}
	time.Sleep(30 * time.Millisecond) // varios pings de por medio
	// "fin" se repite: a un cliente con el búfer lleno se le puede descartar uno.
	esperar(t, 3*time.Second, "todos los clientes dados de baja", func() bool {
		h.Publicar([]byte("fin"))
		return h.Clientes() == 0
	})
	wg.Wait()
	srv.Close()
	sinFugas(t, antes)
}

func TestPingPongYClienteMudo(t *testing.T) {
	h := NuevoHub(4)
	srv := servidorHub(h, 20*time.Millisecond)
	defer srv.Close()

	// Un cliente que responde los pings sigue conectado mucho más que 2·intervalo.
	vivo, _ := conectar(t, srv.URL, nil)
	pings := 0
	fin := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(fin) {
		op, p := vivo.recibir(t)
		if op != opPing {
			t.Fatalf("se esperaba un ping, llegó %#x", op)
		}
		pings++
		vivo.enviar(t, 0x8A, p, true, len(p))
	}
	if pings < 5 {
		t.Fatalf("solo %d pings en 200 ms con intervalo de 20 ms", pings)
	}
	h.Publicar([]byte("sigo"))
	for {
		op, p := vivo.recibir(t)
		if op == opTexto {
			if string(p) != "sigo" {
				t.Fatalf("mensaje %q", p)
			}
			break
		}
		vivo.enviar(t, 0x8A, p, true, len(p))
	}

	vivo.nc.Close()
	esperar(t, time.Second, "el vivo dado de baja", func() bool { return h.Clientes() == 0 })

	// Un cliente mudo (no responde nada, como un TCP medio abierto) cae en ~2·intervalo.
	mudo, _ := conectar(t, srv.URL, nil)
	esperar(t, time.Second, "el mudo suscrito", func() bool { return h.Clientes() == 1 })
	inicio := time.Now()
	for {
		if _, _, err := mudo.recibirE(); err != nil {
			break // el servidor cerró la conexión
		}
	}
	if d := time.Since(inicio); d > 500*time.Millisecond {
		t.Fatalf("el cliente mudo tardó %v en caer (plazo 40 ms)", d)
	}
	esperar(t, time.Second, "el mudo dado de baja", func() bool { return h.Clientes() == 0 })
}

func TestServirSinPing(t *testing.T) {
	h := NuevoHub(4)
	srv := servidorHub(h, 0) // intervalo ≤ 0: sin pings y sin pánico
	defer srv.Close()
	c, _ := conectar(t, srv.URL, nil)
	esperar(t, time.Second, "cliente suscrito", func() bool { return h.Clientes() == 1 })
	h.Publicar([]byte("hola"))
	if op, p := c.recibir(t); op != opTexto || string(p) != "hola" {
		t.Fatalf("%#x %q", op, p)
	}
}

func TestDesconexionAbrupta(t *testing.T) {
	antes := runtime.NumGoroutine()
	h := NuevoHub(4)
	srv := servidorHub(h, time.Hour)
	c, _ := conectar(t, srv.URL, nil)
	esperar(t, time.Second, "cliente suscrito", func() bool { return h.Clientes() == 1 })
	c.nc.Close() // sin frame de cierre
	esperar(t, time.Second, "baja tras el cierre TCP", func() bool { return h.Clientes() == 0 })
	srv.Close()
	sinFugas(t, antes)
}

func TestFrameDe64Bits(t *testing.T) {
	grande := strings.Repeat("v", 70_000) // > 65 535: largo de 64 bits
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, nil)
		if err != nil {
			return
		}
		_ = c.EscribirTexto([]byte(grande))
		_ = c.Cerrar(CierreNormal, "")
	}))
	defer srv.Close()
	c, _ := conectar(t, srv.URL, nil)
	if op, p := c.recibir(t); op != opTexto || string(p) != grande {
		t.Fatalf("frame grande: %#x, %d bytes", op, len(p))
	}
}

func TestCierreSinFramesDespuesYMotivoUTF8(t *testing.T) {
	motivo := strings.Repeat("ñ", 100) // 200 bytes: hay que recortar sin partir runas
	resultado := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, nil)
		if err != nil {
			return
		}
		_ = c.Cerrar(CierreSaliendo, motivo)
		resultado <- c.EscribirTexto([]byte("tarde")) // después del cierre: no debe salir
	}))
	defer srv.Close()
	c, _ := conectar(t, srv.URL, nil)
	op, p := c.recibir(t)
	if op != opCierre || binary.BigEndian.Uint16(p) != CierreSaliendo {
		t.Fatalf("%#x %v", op, p)
	}
	if len(p) > 125 || !utf8.Valid(p[2:]) {
		t.Fatalf("motivo de %d bytes, UTF-8 válido = %v", len(p)-2, utf8.Valid(p[2:]))
	}
	if err := <-resultado; !errors.Is(err, ErrCerrado) {
		t.Fatalf("escribir después del cierre = %v, quería ErrCerrado", err)
	}
	if _, _, err := c.recibirE(); err == nil {
		t.Fatal("llegó un frame después del cierre")
	}
	if got := recortarUTF8("aé", 2); got != "a" {
		t.Fatalf("recortarUTF8 = %q", got)
	}
}

func TestPlazosHeredadosNoCortanLaConexion(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, nil)
		if err != nil {
			return
		}
		for {
			m, err := c.LeerMensaje()
			if err != nil {
				return
			}
			_ = c.EscribirTexto(m)
		}
	}))
	// Como api.NuevoHTTP, pero cortos. Regresión: la conexión tomada debe vivir
	// más que ReadTimeout y WriteTimeout. (En Go 1.24 Hijack ya suelta el plazo de
	// lectura y Conn pone uno propio en cada escritura; Aceptar además llama a
	// SetDeadline(time.Time{}) para no depender de ese detalle interno.)
	srv.Config.ReadTimeout, srv.Config.WriteTimeout = 150*time.Millisecond, 150*time.Millisecond
	srv.Start()
	defer srv.Close()
	c, resp := conectar(t, srv.URL, map[string]string{"Sec-WebSocket-Extensions": "permessage-deflate; client_max_window_bits"})
	if resp.StatusCode != 101 || resp.Header.Get("Sec-WebSocket-Extensions") != "" {
		t.Fatalf("la extensión ofrecida debe declinarse omitiendo la cabecera: %d %q",
			resp.StatusCode, resp.Header.Get("Sec-WebSocket-Extensions"))
	}
	time.Sleep(400 * time.Millisecond)
	c.enviar(t, 0x81, []byte("sigo"), true, 4)
	if op, p := c.recibir(t); op != opTexto || string(p) != "sigo" {
		t.Fatalf("tras 400 ms la conexión debía seguir viva: %#x %q", op, p)
	}
}

func TestHubCerrarManda1001(t *testing.T) {
	antes := runtime.NumGoroutine()
	h := NuevoHub(4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, nil)
		if err != nil {
			return
		}
		h.Servir(c, h.Suscribir(), time.Hour)
	}))
	clientes := make([]*cliente, 3)
	for i := range clientes {
		clientes[i], _ = conectar(t, srv.URL, nil)
	}
	for h.Clientes() < 3 {
		time.Sleep(time.Millisecond)
	}
	h.Cerrar()
	for _, c := range clientes {
		op, p := c.recibir(t)
		if op != opCierre || binary.BigEndian.Uint16(p) != CierreSaliendo {
			t.Fatalf("al cerrar el hub se esperaba el cierre 1001: %#x %v", op, p)
		}
		c.nc.Close()
	}
	if h.Clientes() != 0 {
		t.Fatalf("quedan %d clientes", h.Clientes())
	}
	select {
	case <-h.Suscribir().Hecho():
	default:
		t.Fatal("un cliente que se suscribe después de Cerrar debe nacer dado de baja")
	}
	srv.Close()
	plazo := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > antes && time.Now().Before(plazo) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > antes {
		t.Fatalf("fuga de goroutines tras Cerrar: %d antes, %d después", antes, n)
	}
}
