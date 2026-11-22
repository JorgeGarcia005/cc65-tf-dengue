package ws

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const claveRFC = "dGhlIHNhbXBsZSBub25jZQ==" // RFC 6455 §1.3

func TestHandshakeVectorRFC(t *testing.T) {
	if got := ClaveAceptacion(claveRFC); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("Sec-WebSocket-Accept = %q", got)
	}
}

// cliente es un cliente WebSocket mínimo para las pruebas.
type cliente struct {
	nc net.Conn
	r  *bufio.Reader
}

func conectar(t *testing.T, url string, cab map[string]string) (*cliente, *http.Response) {
	t.Helper()
	nc, err := net.Dial("tcp", strings.TrimPrefix(url, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { nc.Close() })
	base := map[string]string{"Upgrade": "websocket", "Connection": "keep-alive, Upgrade",
		"Sec-WebSocket-Version": "13", "Sec-WebSocket-Key": claveRFC, "Origin": "http://localhost:8081"}
	for k, v := range cab {
		base[k] = v
	}
	req := "GET /ws HTTP/1.1\r\nHost: prueba\r\n"
	for k, v := range base {
		if v != "" {
			req += k + ": " + v + "\r\n"
		}
	}
	if _, err := nc.Write([]byte(req + "\r\n")); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(nc)
	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = nc.SetDeadline(time.Now().Add(5 * time.Second))
	return &cliente{nc, r}, resp
}

// enviar escribe un frame; enmascarado como exige RFC 6455 salvo que mascara sea false.
func (c *cliente) enviar(t *testing.T, primerByte byte, p []byte, mascara bool, largoDeclarado int) {
	t.Helper()
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
	if _, err := c.nc.Write(b); err != nil {
		t.Fatal(err)
	}
}

func (c *cliente) recibir(t *testing.T) (op byte, p []byte) {
	t.Helper()
	var cab [2]byte
	if _, err := io.ReadFull(c.r, cab[:]); err != nil {
		t.Fatal(err)
	}
	if cab[1]&0x80 != 0 {
		t.Fatal("el servidor envió un frame enmascarado")
	}
	n := uint64(cab[1] & 0x7F)
	switch n {
	case 126:
		var b [2]byte
		io.ReadFull(c.r, b[:])
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		io.ReadFull(c.r, b[:])
		n = binary.BigEndian.Uint64(b[:])
	}
	p = make([]byte, n)
	if _, err := io.ReadFull(c.r, p); err != nil {
		t.Fatal(err)
	}
	return cab[0] & 0x0F, p
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Aceptar(w, r, nil)
		if err != nil {
			return
		}
		h.Servir(c, h.Suscribir(), 20*time.Millisecond)
	}))
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, resp := conectar(t, srv.URL, nil)
			if resp.StatusCode != 101 {
				t.Errorf("código %d", resp.StatusCode)
				return
			}
			for {
				op, p := c.recibir(t)
				if op == opTexto && string(p) == "fin" {
					break
				}
			}
			c.enviar(t, 0x88, binary.BigEndian.AppendUint16(nil, 1000), true, 2)
			for op, _ := c.recibir(t); op != opCierre; op, _ = c.recibir(t) {
			} // pueden llegar mensajes en vuelo antes del eco del cierre
		}()
	}
	for h.Clientes() < 16 {
		h.Publicar([]byte("epoca"))
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < 50; i++ {
		h.Publicar([]byte("epoca"))
	}
	// "fin" se repite: a un cliente con el búfer lleno se le puede descartar uno.
	for h.Clientes() > 0 {
		h.Publicar([]byte("fin"))
		time.Sleep(time.Millisecond)
	}
	wg.Wait()
	srv.Close()
	plazo := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > antes && time.Now().Before(plazo) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > antes {
		t.Fatalf("fuga de goroutines: %d antes, %d después", antes, n)
	}
}
