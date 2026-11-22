// Package ws implementa el lado servidor de WebSocket (RFC 6455) con la
// biblioteca estándar (ADR-0007): handshake sobre net/http + Hijack, frames de
// texto del servidor sin máscara, lectura de frames enmascarados del cliente,
// ping/pong y cierre. No hay extensiones ni subprotocolos; el servidor no
// fragmenta y rechaza los mensajes fragmentados del cliente.
package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// guid de RFC 6455 §1.3 para calcular Sec-WebSocket-Accept.
const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// MaxMensaje es el tamaño máximo de un frame del cliente (ADR-0007).
const MaxMensaje = 64 << 10

// Opcodes (RFC 6455 §5.2).
const (
	opContinuacion = 0x0
	opTexto        = 0x1
	opBinario      = 0x2
	opCierre       = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// Códigos de cierre (RFC 6455 §7.4.1).
const (
	CierreNormal      = 1000
	CierreSaliendo    = 1001 // el servidor se apaga (SIGTERM)
	CierreProtocolo   = 1002
	CierreNoSoportado = 1003
	CierreMuyGrande   = 1009
)

// ErrCerrado indica que la conexión terminó (cierre del cliente o error de protocolo).
var ErrCerrado = errors.New("ws: conexión cerrada")

// ClaveAceptacion calcula Sec-WebSocket-Accept a partir de Sec-WebSocket-Key.
func ClaveAceptacion(clave string) string {
	h := sha1.Sum([]byte(clave + guid))
	return base64.StdEncoding.EncodeToString(h[:])
}

// Conn es una conexión WebSocket aceptada. Escribir es seguro desde varias
// goroutines; leer, solo desde una.
type Conn struct {
	nc        net.Conn
	r         *bufio.Reader
	plazo     time.Duration // plazo de cada escritura
	cerrarUna sync.Once

	// escritura serializa los frames y protege cerrado: una vez enviado el
	// frame de cierre no sale ningún otro (RFC 6455 §5.5.1).
	escritura sync.Mutex
	cerrado   bool

	// plazoLectura, si es > 0, acota la espera de cada frame del cliente: con
	// pings cada intervalo, un cliente mudo (TCP medio abierto) cae en ese plazo.
	// Lo fija Hub.Servir antes de leer; solo lo usa la goroutine lectora.
	plazoLectura time.Duration
}

// Aceptar valida el handshake y, si es correcto, toma la conexión TCP y responde
// 101. Si origenPermitido no es nil y devuelve false para el Origin, responde 403.
// Ante un error ya respondió con el código HTTP adecuado.
func Aceptar(w http.ResponseWriter, r *http.Request, origenPermitido func(string) bool) (*Conn, error) {
	falla := func(codigo int, msg string) (*Conn, error) {
		if codigo == http.StatusUpgradeRequired {
			w.Header().Set("Sec-WebSocket-Version", "13")
		}
		http.Error(w, msg, codigo)
		return nil, errors.New("ws: " + msg)
	}
	switch {
	case r.Method != http.MethodGet:
		return falla(http.StatusMethodNotAllowed, "el handshake debe ser GET")
	case !r.ProtoAtLeast(1, 1):
		return falla(http.StatusBadRequest, "el handshake requiere HTTP/1.1")
	case !contieneToken(r.Header.Values("Connection"), "upgrade") || !contieneToken(r.Header.Values("Upgrade"), "websocket"):
		return falla(http.StatusBadRequest, "faltan las cabeceras Upgrade: websocket y Connection: Upgrade")
	case r.Header.Get("Sec-WebSocket-Version") != "13":
		return falla(http.StatusUpgradeRequired, "solo se admite Sec-WebSocket-Version 13")
	}
	clave := r.Header.Get("Sec-WebSocket-Key")
	if b, err := base64.StdEncoding.DecodeString(clave); err != nil || len(b) != 16 {
		return falla(http.StatusBadRequest, "Sec-WebSocket-Key inválida")
	}
	if origenPermitido != nil && !origenPermitido(r.Header.Get("Origin")) {
		return falla(http.StatusForbidden, "origen no permitido")
	}
	// NewResponseController llega al Hijacker aunque un middleware envuelva w.
	nc, rw, err := http.NewResponseController(w).Hijack()
	if errors.Is(err, http.ErrNotSupported) {
		return falla(http.StatusInternalServerError, "el servidor no permite tomar la conexión")
	}
	if err != nil {
		return nil, fmt.Errorf("ws: hijack: %w", err)
	}
	// Hijack no quita los plazos del http.Server (ReadTimeout/WriteTimeout de
	// api.NuevoHTTP): sin esto la conexión moriría a los 15–30 s. Desde aquí cada
	// escritura pone su propio plazo y la lectura espera lo que haga falta.
	if err := nc.SetDeadline(time.Time{}); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ws: quitar plazos: %w", err)
	}
	// rw.Reader puede traer bytes del cliente ya leídos por el servidor HTTP.
	c := &Conn{nc: nc, r: rw.Reader, plazo: 10 * time.Second}
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + ClaveAceptacion(clave) + "\r\n\r\n"
	if err := c.escribirCrudo([]byte(resp)); err != nil {
		nc.Close()
		return nil, err
	}
	return c, nil
}

// contieneToken busca token en todas las líneas de una cabecera de lista
// (RFC 9110 §5.3: «Connection: keep-alive» y «Connection: Upgrade» en dos líneas
// equivalen a una con los dos valores).
func contieneToken(lineas []string, token string) bool {
	for _, linea := range lineas {
		for _, t := range strings.Split(linea, ",") {
			if strings.EqualFold(strings.TrimSpace(t), token) {
				return true
			}
		}
	}
	return false
}

// EscribirTexto envía un mensaje de texto en un solo frame.
func (c *Conn) EscribirTexto(p []byte) error { return c.escribirFrame(opTexto, p) }

// Ping envía un ping de control.
func (c *Conn) Ping() error { return c.escribirFrame(opPing, nil) }

// Cerrar envía el frame de cierre y cierra la conexión TCP. Es idempotente y,
// como toma el candado de escritura, ningún frame sale después del cierre.
// El motivo se recorta a 123 bytes sin partir caracteres UTF-8.
func (c *Conn) Cerrar(codigo uint16, motivo string) error {
	return c.cerrar(func() {
		p := binary.BigEndian.AppendUint16(nil, codigo)
		_ = c.enviar(armarFrame(opCierre, append(p, recortarUTF8(motivo, 123)...)))
	})
}

// cerrarSinFrame cierra el socket sin mandar nada (la conexión ya falló).
func (c *Conn) cerrarSinFrame() { _ = c.cerrar(nil) }

func (c *Conn) cerrar(antes func()) error {
	var err error
	c.cerrarUna.Do(func() {
		c.escritura.Lock()
		defer c.escritura.Unlock()
		if antes != nil {
			antes()
		}
		c.cerrado = true
		err = c.nc.Close()
	})
	return err
}

func recortarUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

func armarFrame(op byte, p []byte) []byte {
	cab := []byte{0x80 | op} // FIN + opcode; el servidor nunca enmascara
	switch n := len(p); {
	case n <= 125:
		cab = append(cab, byte(n))
	case n <= 0xFFFF:
		cab = binary.BigEndian.AppendUint16(append(cab, 126), uint16(n))
	default:
		cab = binary.BigEndian.AppendUint64(append(cab, 127), uint64(n))
	}
	return append(cab, p...)
}

func (c *Conn) escribirFrame(op byte, p []byte) error { return c.escribirCrudo(armarFrame(op, p)) }

func (c *Conn) escribirCrudo(b []byte) error {
	c.escritura.Lock()
	defer c.escritura.Unlock()
	if c.cerrado {
		return ErrCerrado
	}
	return c.enviar(b)
}

// enviar escribe con plazo; quien llama tiene el candado de escritura.
func (c *Conn) enviar(b []byte) error {
	_ = c.nc.SetWriteDeadline(time.Now().Add(c.plazo))
	_, err := c.nc.Write(b)
	return err
}

// LeerMensaje devuelve el siguiente mensaje de texto del cliente. Responde los
// ping con pong, ignora los pong y, ante un cierre o un error de protocolo,
// cierra la conexión con el código que corresponde y devuelve ErrCerrado.
func (c *Conn) LeerMensaje() ([]byte, error) {
	for {
		if c.plazoLectura > 0 {
			_ = c.nc.SetReadDeadline(time.Now().Add(c.plazoLectura))
		}
		fin, op, p, codigo, err := c.leerFrame()
		if err != nil {
			if codigo != 0 {
				_ = c.Cerrar(codigo, err.Error())
			} else {
				c.cerrarSinFrame() // EOF, plazo vencido o socket cerrado: no hay a quién avisar
			}
			return nil, fmt.Errorf("%w: %v", ErrCerrado, err)
		}
		switch {
		case op == opPing:
			if err := c.escribirFrame(opPong, p); err != nil {
				c.cerrarSinFrame()
				return nil, fmt.Errorf("%w: pong: %v", ErrCerrado, err)
			}
		case op == opPong: // renueva el plazo de lectura en la próxima vuelta
		case op == opCierre:
			_ = c.Cerrar(CierreNormal, "")
			return nil, ErrCerrado
		case op == opTexto && fin:
			return p, nil
		case op == opContinuacion: // continuación sin mensaje fragmentado abierto (RFC 6455 §5.4)
			_ = c.Cerrar(CierreProtocolo, "continuación sin mensaje abierto")
			return nil, fmt.Errorf("%w: continuación suelta", ErrCerrado)
		case op == opBinario:
			_ = c.Cerrar(CierreNoSoportado, "solo se aceptan mensajes de texto")
			return nil, fmt.Errorf("%w: mensaje binario", ErrCerrado)
		default: // texto fragmentado (FIN = 0)
			_ = c.Cerrar(CierreNoSoportado, "no se aceptan mensajes fragmentados")
			return nil, fmt.Errorf("%w: mensaje fragmentado", ErrCerrado)
		}
	}
}

// leerFrame lee un frame. Si el error es de protocolo, codigo indica el cierre.
func (c *Conn) leerFrame() (fin bool, op byte, p []byte, codigo uint16, err error) {
	var cab [2]byte
	if _, err = io.ReadFull(c.r, cab[:]); err != nil {
		return
	}
	fin, op = cab[0]&0x80 != 0, cab[0]&0x0F
	if cab[0]&0x70 != 0 {
		return fin, op, nil, CierreProtocolo, errors.New("bits RSV sin extensión negociada")
	}
	if cab[1]&0x80 == 0 {
		return fin, op, nil, CierreProtocolo, errors.New("frame del cliente sin máscara")
	}
	n := uint64(cab[1] & 0x7F)
	esControl := op >= opCierre
	if esControl && (n > 125 || !fin) {
		return fin, op, nil, CierreProtocolo, errors.New("frame de control inválido")
	}
	if op > opBinario && !esControl || op > opPong {
		return fin, op, nil, CierreProtocolo, fmt.Errorf("opcode %#x desconocido", op)
	}
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
	if n > MaxMensaje {
		return fin, op, nil, CierreMuyGrande, fmt.Errorf("mensaje de %d bytes (máximo %d)", n, MaxMensaje)
	}
	var mascara [4]byte
	if _, err = io.ReadFull(c.r, mascara[:]); err != nil {
		return
	}
	p = make([]byte, n)
	if _, err = io.ReadFull(c.r, p); err != nil {
		return
	}
	for i := range p {
		p[i] ^= mascara[i%4]
	}
	return fin, op, p, 0, nil
}
