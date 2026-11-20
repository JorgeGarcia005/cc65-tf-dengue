// Package protocolo define los mensajes que intercambian el coordinador y los
// nodos del clúster y su forma en la red (ADR-0005).
//
// Transporte: una conexión TCP persistente por nodo, iniciada por el nodo.
// Framing: un mensaje = un objeto JSON en una línea (JSON + '\n').
//
// Se lee con json.Decoder y no con bufio.Scanner, que corta las líneas en
// 64 KiB: un `gradiente` con un parcial por bloque puede superar ese tamaño.
//
// encoding/json escribe cada float64 con la representación decimal más corta
// que vuelve exactamente al mismo valor, así que pesos y parciales conservan
// sus 64 bits en el viaje. NaN y ±Inf no existen en JSON: Enviar los rechaza
// con ErrNoFinito antes de escribir un solo byte, y quien llama responde con
// un mensaje `error` (un gradiente no finito es un fallo, no un dato).
package protocolo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// Version del protocolo. Un nodo con otra versión se rechaza en el registro.
const Version = 1

// MaxMensaje es el tamaño máximo de un mensaje recibido. Protege al proceso de
// un par defectuoso que envía una línea sin fin. Con d = 31 un parcial ocupa
// menos de 1 KiB en JSON, así que el límite admite decenas de miles de bloques.
const MaxMensaje = 64 << 20

// Tipo identifica cada mensaje del protocolo.
type Tipo string

const (
	Registro   Tipo = "registro"   // nodo → coordinador, al conectarse
	Asignacion Tipo = "asignacion" // coordinador → nodo: carga este shard
	Listo      Tipo = "listo"      // nodo → coordinador: shard cargado y validado
	Epoca      Tipo = "epoca"      // coordinador → nodo: calcula el shard con estos pesos
	Gradiente  Tipo = "gradiente"  // nodo → coordinador: un parcial por bloque del shard
	Error      Tipo = "error"      // ambos sentidos
	Fin        Tipo = "fin"        // coordinador → nodo: terminó el entrenamiento
)

// Motivos de un mensaje `fin`.
const (
	MotivoCompleto = "completo"
	MotivoAbortado = "abortado"
)

// Errores con nombre, para distinguirlos con errors.Is.
var (
	ErrNoFinito  = errors.New("protocolo: el mensaje contiene NaN o ±Inf")
	ErrVersion   = errors.New("protocolo: versión incompatible")
	ErrInvalido  = errors.New("protocolo: mensaje inválido")
	ErrMuyGrande = errors.New("protocolo: mensaje más grande que MaxMensaje")
)

// ConfigML son los hiperparámetros de la pérdida que el nodo necesita para que
// sus parciales sean los mismos que calcularía el coordinador.
type ConfigML struct {
	Lambda  float64 `json:"lambda"`
	PesoPos float64 `json:"peso_pos"`
	PesoNeg float64 `json:"peso_neg"`
}

// Mensaje es el sobre único del protocolo: el campo Tipo dice qué campos
// aplican. Los campos vacíos no viajan; un entero ausente vale 0 (la época 0 y
// el shard 0 son válidos y se leen igual).
//
//	registro    Nodo, Version, Workers
//	asignacion  Shard, Ruta, SHA256, BloqueInicial, NumBloques, TamBloque, D, Config
//	listo       Shard, BloqueInicial, NumBloques, N, D
//	epoca       Epoca, Shard, W
//	gradiente   Epoca, Shard, BloqueInicial, Parciales (MicrosCalculo, opcional)
//	error       Epoca, Shard, Detalle
//	fin         Motivo
type Mensaje struct {
	Tipo          Tipo         `json:"tipo"`
	Nodo          string       `json:"nodo,omitempty"`
	Version       int          `json:"version,omitempty"`
	Workers       int          `json:"workers,omitempty"`
	Epoca         int          `json:"epoca,omitempty"`
	Shard         int          `json:"shard,omitempty"`
	Ruta          string       `json:"ruta,omitempty"`
	SHA256        string       `json:"sha256,omitempty"`
	BloqueInicial int          `json:"bloque_inicial,omitempty"`
	NumBloques    int          `json:"num_bloques,omitempty"`
	TamBloque     int          `json:"tam_bloque,omitempty"`
	N             int          `json:"n,omitempty"`
	D             int          `json:"d,omitempty"`
	Config        *ConfigML    `json:"config,omitempty"`
	W             []float64    `json:"w,omitempty"`
	Parciales     []ml.Parcial `json:"parciales,omitempty"`
	Detalle       string       `json:"mensaje,omitempty"`
	// MicrosCalculo es lo que tardó el nodo en calcular los parciales de un
	// gradiente (solo cómputo, sin red ni JSON). Lo usa el benchmark del
	// clúster (#32) para separar cómputo de red y espera.
	MicrosCalculo int64  `json:"us_calculo,omitempty"`
	Motivo        string `json:"motivo,omitempty"`
}

// Validar comprueba que el mensaje tenga un tipo conocido y los campos que su
// tipo exige. No mira el estado del entrenamiento (época actual, dueño del
// shard): eso lo decide quien recibe.
func (m *Mensaje) Validar() error {
	mal := func(formato string, args ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrInvalido, m.Tipo, fmt.Sprintf(formato, args...))
	}
	if m.Epoca < 0 || m.Shard < 0 || m.BloqueInicial < 0 || m.NumBloques < 0 ||
		m.TamBloque < 0 || m.N < 0 || m.D < 0 || m.Workers < 0 {
		return mal("campo entero negativo")
	}
	switch m.Tipo {
	case Registro:
		if m.Version != Version {
			return fmt.Errorf("%w: el nodo %q habla la versión %d, el coordinador la %d",
				ErrVersion, m.Nodo, m.Version, Version)
		}
		if m.Nodo == "" {
			return mal("falta el nombre del nodo")
		}
	case Asignacion:
		if m.Ruta == "" || m.NumBloques == 0 || m.TamBloque == 0 || m.D == 0 || m.Config == nil {
			return mal("faltan ruta, num_bloques, tam_bloque, d o config")
		}
		// Un hiperparámetro negativo haría entrar en pánico a ml.NuevoPool en el
		// nodo; se rechaza aquí como mensaje inválido (PR-35 H1).
		if m.Config.Lambda < 0 || m.Config.PesoPos < 0 || m.Config.PesoNeg < 0 {
			return mal("config con un valor negativo (lambda %v, peso_pos %v, peso_neg %v)",
				m.Config.Lambda, m.Config.PesoPos, m.Config.PesoNeg)
		}
	case Listo:
		if m.NumBloques == 0 || m.N == 0 || m.D == 0 {
			return mal("faltan num_bloques, n o d")
		}
	case Epoca:
		if len(m.W) < 2 {
			return mal("w tiene %d valores (mínimo 2: un peso y el sesgo)", len(m.W))
		}
	case Gradiente:
		if m.MicrosCalculo < 0 {
			return mal("tiempo de cálculo negativo")
		}
		if len(m.Parciales) == 0 {
			return mal("sin parciales")
		}
		d := len(m.Parciales[0].Grad)
		if d < 2 {
			return mal("parcial con %d sumas (mínimo 2)", d)
		}
		for k := range m.Parciales {
			if len(m.Parciales[k].Grad) != d {
				return mal("el parcial %d tiene %d sumas y el primero %d", k, len(m.Parciales[k].Grad), d)
			}
			// Todo bloque tiene al menos una fila y los pesos por clase son
			// positivos: un peso ≤ 0 es un parcial corrupto, y con todos en
			// cero ml.Finalizar dividiría entre cero (PR-35 H1).
			if m.Parciales[k].Peso <= 0 {
				return mal("el parcial %d tiene peso %v (debe ser > 0)", k, m.Parciales[k].Peso)
			}
		}
	case Error:
		if m.Detalle == "" {
			return mal("sin texto")
		}
	case Fin:
		if m.Motivo != MotivoCompleto && m.Motivo != MotivoAbortado {
			return mal("motivo %q", m.Motivo)
		}
	default:
		return fmt.Errorf("%w: tipo desconocido %q", ErrInvalido, m.Tipo)
	}
	return nil
}

// Codificar devuelve la línea JSON del mensaje, terminada en '\n'.
func Codificar(m *Mensaje) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(m); err != nil {
		var noFinito *json.UnsupportedValueError
		if errors.As(err, &noFinito) {
			return nil, fmt.Errorf("%w (%s, época %d, shard %d)", ErrNoFinito, m.Tipo, m.Epoca, m.Shard)
		}
		return nil, fmt.Errorf("protocolo: no se pudo codificar %s: %w", m.Tipo, err)
	}
	return buf.Bytes(), nil
}

// Conexion envía y recibe mensajes sobre un net.Conn.
//
// Enviar es seguro desde varias goroutines (un mutex impide que dos líneas se
// mezclen). Recibir debe llamarlo una sola goroutine: la lectora de esa
// conexión (ADR-0005).
type Conexion struct {
	conn     net.Conn
	lector   *lectorLimitado
	dec      *json.Decoder
	muEnvio  sync.Mutex
	escritos atomic.Int64 // bytes escritos (benchmark #32)
}

// Bytes devuelve los bytes enviados y recibidos por esta conexión. Es seguro
// llamarlo desde cualquier goroutine.
func (c *Conexion) Bytes() (enviados, recibidos int64) {
	return c.escritos.Load(), c.lector.leidos.Load()
}

// Nueva envuelve una conexión ya establecida.
func Nueva(conn net.Conn) *Conexion {
	l := &lectorLimitado{r: conn}
	return &Conexion{conn: conn, lector: l, dec: json.NewDecoder(l)}
}

// Enviar escribe un mensaje. Con plazo > 0, la escritura completa debe terminar
// antes de ese tiempo; si no, el error envuelve os.ErrDeadlineExceeded y la
// conexión debe darse por perdida (pudo quedar una línea a medias).
//
// La línea se codifica entera antes de escribir: si el mensaje no se puede
// codificar (ErrNoFinito) no sale ningún byte y la conexión sigue sana.
func (c *Conexion) Enviar(m Mensaje, plazo time.Duration) error {
	linea, err := Codificar(&m)
	if err != nil {
		return err
	}
	c.muEnvio.Lock()
	defer c.muEnvio.Unlock()
	var limite time.Time
	if plazo > 0 {
		limite = time.Now().Add(plazo)
	}
	// Fijar el plazo solo falla si la conexión ya está cerrada; en ese caso la
	// escritura de abajo falla de inmediato con el error que importa.
	_ = c.conn.SetWriteDeadline(limite)
	n, err := c.conn.Write(linea)
	c.escritos.Add(int64(n))
	if err != nil {
		return fmt.Errorf("protocolo: al enviar %s (plazo %v): %w", m.Tipo, plazo, err)
	}
	return nil
}

// Recibir lee y valida el siguiente mensaje. Con plazo > 0 espera como máximo
// ese tiempo y devuelve un error que envuelve os.ErrDeadlineExceeded. Si el par
// cerró la conexión entre dos mensajes devuelve io.EOF.
//
// Cualquier error que no sea de validación (ErrInvalido, ErrVersion) deja la
// conexión inservible: json.Decoder no se recupera de una lectura fallida.
func (c *Conexion) Recibir(plazo time.Duration) (Mensaje, error) {
	var limite time.Time
	if plazo > 0 {
		limite = time.Now().Add(plazo)
	}
	// Igual que en Enviar: si la conexión ya está cerrada, lo dirá la lectura
	// (io.EOF si la cerró el par), que es el error que el llamador distingue.
	_ = c.conn.SetReadDeadline(limite)
	c.lector.restante = MaxMensaje
	var m Mensaje
	if err := c.dec.Decode(&m); err != nil {
		if err == io.EOF {
			return Mensaje{}, io.EOF
		}
		return Mensaje{}, fmt.Errorf("protocolo: al recibir (plazo %v): %w", plazo, err)
	}
	if err := m.Validar(); err != nil {
		return m, err
	}
	return m, nil
}

// Cerrar cierra la conexión; desbloquea un Recibir o Enviar en curso.
func (c *Conexion) Cerrar() error { return c.conn.Close() }

// Remoto describe al par, para los logs.
func (c *Conexion) Remoto() string {
	if a := c.conn.RemoteAddr(); a != nil {
		return a.String()
	}
	return "?"
}

// lectorLimitado corta la lectura cuando un solo mensaje supera MaxMensaje.
// json.Decoder lee por adelantado, así que el límite es aproximado (puede
// contar el inicio del mensaje siguiente), lo que basta para su propósito.
type lectorLimitado struct {
	r        io.Reader
	restante int
	leidos   atomic.Int64 // bytes leídos de la conexión (benchmark #32)
}

func (l *lectorLimitado) Read(p []byte) (int, error) {
	if l.restante <= 0 {
		return 0, ErrMuyGrande
	}
	if len(p) > l.restante {
		p = p[:l.restante]
	}
	n, err := l.r.Read(p)
	l.restante -= n
	l.leidos.Add(int64(n))
	return n, err
}
