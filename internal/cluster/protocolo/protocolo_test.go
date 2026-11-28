package protocolo

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

const plazoTest = 5 * time.Second

// transportes devuelve un par de conexiones enlazadas por cada transporte en el
// que se prueba el protocolo: net.Pipe (síncrono, sin búfer: delata a quien
// escribe sin que nadie lea) y TCP real por 127.0.0.1.
func transportes(t *testing.T) map[string]func() (a, b net.Conn) {
	t.Helper()
	return map[string]func() (net.Conn, net.Conn){
		"pipe": func() (net.Conn, net.Conn) {
			a, b := net.Pipe()
			t.Cleanup(func() { a.Close(); b.Close() })
			return a, b
		},
		"tcp": func() (net.Conn, net.Conn) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			aceptada := make(chan net.Conn, 1)
			go func() {
				c, err := ln.Accept()
				if err != nil {
					close(aceptada)
					return
				}
				aceptada <- c
			}()
			a, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			b, ok := <-aceptada
			if !ok {
				t.Fatal("no se aceptó la conexión")
			}
			t.Cleanup(func() { a.Close(); b.Close() })
			return a, b
		},
	}
}

func enCadaTransporte(t *testing.T, f func(t *testing.T, a, b *Conexion)) {
	t.Helper()
	for nombre, abrir := range transportes(t) {
		t.Run(nombre, func(t *testing.T) {
			a, b := abrir()
			f(t, Nueva(a), Nueva(b))
		})
	}
}

// valoresDificiles son los float64 que más fácil pierden bits en un viaje por
// texto: ceros con signo, subnormales, extremos y decimales sin forma exacta.
var valoresDificiles = []float64{
	0, math.Copysign(0, -1), 1, -1, 0.1, 1.0 / 3, math.Pi, -math.E,
	math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, // subnormales mínimos
	2.2250738585072009e-308,           // el mayor subnormal
	2.2250738585072014e-308,           // el menor normal
	math.MaxFloat64, -math.MaxFloat64, // extremos
	math.Nextafter(1, 2), 1e-17, 123456789.123456789, 5e-324, 1e23,
}

func mismosBits(t *testing.T, contexto string, quiere, tiene []float64) {
	t.Helper()
	if len(quiere) != len(tiene) {
		t.Fatalf("%s: largos distintos %d vs %d", contexto, len(quiere), len(tiene))
	}
	for j := range quiere {
		if math.Float64bits(quiere[j]) != math.Float64bits(tiene[j]) {
			t.Fatalf("%s: posición %d: %#016x vs %#016x (%.17g vs %.17g)", contexto, j,
				math.Float64bits(quiere[j]), math.Float64bits(tiene[j]), quiere[j], tiene[j])
		}
	}
}

func mensajesDeEjemplo() []Mensaje {
	return []Mensaje{
		{Tipo: Registro, Nodo: "nodo-1", Version: Version, Workers: 8},
		{Tipo: Asignacion, Shard: 2, Ruta: "/datos/shard-2.gold", SHA256: "abc123", BloqueInicial: 124,
			NumBloques: 63, TamBloque: 4096, D: 31, Config: &ConfigML{Lambda: 1e-3, PesoPos: 3.2, PesoNeg: 0.59}},
		{Tipo: Listo, Shard: 2, BloqueInicial: 124, NumBloques: 63, N: 257_000, D: 31},
		{Tipo: Epoca, Epoca: 7, Shard: 2, W: []float64{0.25, -0.5, 1e-9}},
		{Tipo: Epoca, Epoca: 0, Shard: 0, W: []float64{0, 0}}, // época 0 y shard 0 son válidos
		{Tipo: Gradiente, Epoca: 7, Shard: 2, BloqueInicial: 124, Parciales: []ml.Parcial{
			{Grad: []float64{1.5, -2.5, 0.125}, Perdida: 10.75, Peso: 4096},
			{Grad: []float64{0.1, 0.2, 0.3}, Perdida: 3.3, Peso: 1000.5},
		}},
		{Tipo: Error, Epoca: 3, Shard: 1, Detalle: "gradiente no finito en el bloque 70"},
		{Tipo: Fin, Motivo: MotivoCompleto},
		{Tipo: Fin, Motivo: MotivoAbortado},
	}
}

func TestCadaMensajeHaceIdaYVueltaSinCambios(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		for _, m := range mensajesDeEjemplo() {
			errEnvio := make(chan error, 1)
			go func() { errEnvio <- a.Enviar(m, plazoTest) }()
			recibido, err := b.Recibir(plazoTest)
			if err != nil {
				t.Fatalf("%s: recibir: %v", m.Tipo, err)
			}
			if err := <-errEnvio; err != nil {
				t.Fatalf("%s: enviar: %v", m.Tipo, err)
			}
			if !reflect.DeepEqual(m, recibido) {
				t.Errorf("%s:\n envié   %+v\n recibí  %+v", m.Tipo, m, recibido)
			}
		}
	})
}

// ADR-0005, verificación 2: los parciales y los pesos conservan sus 64 bits.
func TestParcialYPesosHacenIdaYVueltaBitABit(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		grad := valoresDificiles
		for i, v := range valoresDificiles {
			envio := []Mensaje{
				{Tipo: Gradiente, Epoca: 1, Shard: 1, Parciales: []ml.Parcial{
					{Grad: grad, Perdida: v, Peso: math.Abs(valoresDificiles[len(valoresDificiles)-1-i]) + math.SmallestNonzeroFloat64}}},
				{Tipo: Epoca, Epoca: 1, Shard: 1, W: grad},
			}
			for _, m := range envio {
				go a.Enviar(m, plazoTest)
				r, err := b.Recibir(plazoTest)
				if err != nil {
					t.Fatal(err)
				}
				if m.Tipo == Epoca {
					mismosBits(t, "w", m.W, r.W)
					continue
				}
				p, q := m.Parciales[0], r.Parciales[0]
				mismosBits(t, "grad", p.Grad, q.Grad)
				mismosBits(t, "pérdida y peso", []float64{p.Perdida, p.Peso}, []float64{q.Perdida, q.Peso})
			}
		}
	})
}

// Los parciales de un cálculo real, tal como salen del worker pool de un nodo.
func TestParcialesDeUnShardRealHacenIdaYVueltaBitABit(t *testing.T) {
	ds, _ := ml.Sintetico(3_000, 31, 7)
	w := make([]float64, ds.D+1)
	for j := range w {
		w[j] = math.Sin(float64(j)) / 3
	}
	pool := ml.NuevoPool(nil, ml.Config{TamBloque: 256, Lambda: 1e-3, PesoPos: 3, PesoNeg: 0.6}, 4)
	defer pool.Cerrar()
	parciales := pool.Parciales(ds, w)

	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		go a.Enviar(Mensaje{Tipo: Gradiente, Epoca: 4, Shard: 1, BloqueInicial: 12, Parciales: parciales}, plazoTest)
		r, err := b.Recibir(plazoTest)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Parciales) != len(parciales) {
			t.Fatalf("llegaron %d parciales, salieron %d", len(r.Parciales), len(parciales))
		}
		for k := range parciales {
			mismosBits(t, fmt.Sprintf("bloque %d", k), parciales[k].Grad, r.Parciales[k].Grad)
			mismosBits(t, fmt.Sprintf("bloque %d, pérdida y peso", k),
				[]float64{parciales[k].Perdida, parciales[k].Peso},
				[]float64{r.Parciales[k].Perdida, r.Parciales[k].Peso})
		}
	})
}

func TestMensajeConValorNoFinitoNoSeEnviaYLaConexionSigueSana(t *testing.T) {
	for _, malo := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
			casos := []Mensaje{
				{Tipo: Gradiente, Epoca: 2, Shard: 1, Parciales: []ml.Parcial{{Grad: []float64{1, malo}, Perdida: 1, Peso: 1}}},
				{Tipo: Gradiente, Epoca: 2, Shard: 1, Parciales: []ml.Parcial{{Grad: []float64{1, 2}, Perdida: malo, Peso: 1}}},
				{Tipo: Epoca, Epoca: 2, Shard: 1, W: []float64{malo, 0}},
			}
			for _, m := range casos {
				// Sin lector al otro lado: sobre net.Pipe, si Enviar escribiera
				// aunque fuera un byte se quedaría bloqueado hasta el plazo.
				err := a.Enviar(m, 200*time.Millisecond)
				if !errors.Is(err, ErrNoFinito) {
					t.Fatalf("%s con %v: err = %v, quiere ErrNoFinito", m.Tipo, malo, err)
				}
			}
			// La conexión sigue sirviendo: el nodo avisa con un mensaje `error`.
			aviso := Mensaje{Tipo: Error, Epoca: 2, Shard: 1, Detalle: "gradiente no finito"}
			go a.Enviar(aviso, plazoTest)
			r, err := b.Recibir(plazoTest)
			if err != nil || !reflect.DeepEqual(r, aviso) {
				t.Fatalf("tras el rechazo: %+v, %v", r, err)
			}
		})
	}
}

// bufio.Scanner corta en 64 KiB; el protocolo no.
func TestGradienteDeMasDe64KiBLlegaCompleto(t *testing.T) {
	const bloques, d = 400, 31
	parciales := make([]ml.Parcial, bloques)
	for k := range parciales {
		parciales[k] = ml.NuevoParcial(d)
		for j := range parciales[k].Grad {
			parciales[k].Grad[j] = math.Sqrt(float64(k*100+j)) / 7
		}
		parciales[k].Perdida, parciales[k].Peso = float64(k)/3, 4096
	}
	m := Mensaje{Tipo: Gradiente, Epoca: 1, Shard: 0, Parciales: parciales}
	linea, err := Codificar(&m)
	if err != nil {
		t.Fatal(err)
	}
	if len(linea) <= bufio.MaxScanTokenSize {
		t.Fatalf("el mensaje de prueba mide %d bytes; debe superar %d", len(linea), bufio.MaxScanTokenSize)
	}
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		go a.Enviar(m, plazoTest)
		r, err := b.Recibir(plazoTest)
		if err != nil {
			t.Fatal(err)
		}
		for k := range parciales {
			mismosBits(t, fmt.Sprintf("bloque %d", k), parciales[k].Grad, r.Parciales[k].Grad)
		}
	})
}

func TestCadaMensajeViajaEnUnaSolaLineaJSON(t *testing.T) {
	for _, m := range mensajesDeEjemplo() {
		linea, err := Codificar(&m)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(linea), "\n"); n != 1 || linea[len(linea)-1] != '\n' {
			t.Errorf("%s: %d saltos de línea en %q", m.Tipo, n, linea)
		}
		var crudo map[string]any
		if err := json.Unmarshal(linea, &crudo); err != nil || crudo["tipo"] != string(m.Tipo) {
			t.Errorf("%s: no es un objeto JSON con su tipo: %v", m.Tipo, err)
		}
	}
}

func TestNodoQueNoRespondeVenceElPlazoYDaUnErrorClaro(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		inicio := time.Now()
		_, err := a.Recibir(80 * time.Millisecond)
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("err = %v, quiere os.ErrDeadlineExceeded", err)
		}
		if !strings.Contains(err.Error(), "plazo 80ms") {
			t.Errorf("el error no dice qué plazo venció: %v", err)
		}
		if d := time.Since(inicio); d < 60*time.Millisecond || d > 2*time.Second {
			t.Errorf("tardó %v con un plazo de 80 ms", d)
		}
	})
}

// Spin, mutante 1: enviar a un par que no lee no puede bloquear para siempre.
func TestEnviarAUnParQueNoLeeVenceElPlazo(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		grande := Mensaje{Tipo: Epoca, Epoca: 1, W: make([]float64, 200_000)}
		var err error
		for range 200 { // TCP: hasta llenar los búferes del sistema
			if err = a.Enviar(grande, 100*time.Millisecond); err != nil {
				break
			}
		}
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("err = %v, quiere os.ErrDeadlineExceeded", err)
		}
	})
}

func TestRecibirSinPlazoEsperaHastaQueLlegueElMensaje(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		m := Mensaje{Tipo: Fin, Motivo: MotivoCompleto}
		go func() {
			time.Sleep(150 * time.Millisecond)
			a.Enviar(m, 0)
		}()
		r, err := b.Recibir(0)
		if err != nil || !reflect.DeepEqual(r, m) {
			t.Fatalf("%+v, %v", r, err)
		}
	})
}

func TestConexionCerradaPorElParDaEOF(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		a.Cerrar()
		if _, err := b.Recibir(plazoTest); err != io.EOF {
			t.Fatalf("err = %v, quiere io.EOF", err)
		}
	})
}

func TestConexionCortadaAMitadDeMensajeNoEsEOFLimpio(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	go func() {
		a.Write([]byte(`{"tipo":"gradiente","epoca":3,"parciales":[{"grad":[1,2`))
		a.Close()
	}()
	_, err := Nueva(b).Recibir(plazoTest)
	if err == nil || err == io.EOF {
		t.Fatalf("err = %v: un mensaje truncado no puede pasar por cierre limpio", err)
	}
}

func TestRegistroConVersionDistintaSeRechaza(t *testing.T) {
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		go a.Enviar(Mensaje{Tipo: Registro, Nodo: "viejo", Version: Version + 1}, plazoTest)
		_, err := b.Recibir(plazoTest)
		if !errors.Is(err, ErrVersion) {
			t.Fatalf("err = %v, quiere ErrVersion", err)
		}
		if !strings.Contains(err.Error(), "viejo") {
			t.Errorf("el error no nombra al nodo: %v", err)
		}
	})
}

func TestMensajeDeTipoDesconocidoOIncompletoSeRechaza(t *testing.T) {
	casos := map[string]string{
		"tipo desconocido":           `{"tipo":"saludo"}`,
		"sin tipo":                   `{"epoca":1}`,
		"registro sin nombre":        `{"tipo":"registro","version":1}`,
		"asignación sin ruta":        `{"tipo":"asignacion","shard":1,"num_bloques":3,"tam_bloque":8,"d":2,"config":{}}`,
		"asignación sin config":      `{"tipo":"asignacion","ruta":"x","num_bloques":3,"tam_bloque":8,"d":2}`,
		"listo sin filas":            `{"tipo":"listo","num_bloques":3,"d":2}`,
		"época sin pesos":            `{"tipo":"epoca","epoca":1}`,
		"gradiente sin parciales":    `{"tipo":"gradiente","epoca":1}`,
		"us_calculo negativo":        `{"tipo":"gradiente","us_calculo":-1,"parciales":[{"grad":[1,2],"perdida":0,"peso":1}]}`,
		"parciales desiguales":       `{"tipo":"gradiente","parciales":[{"grad":[1,2],"perdida":0,"peso":1},{"grad":[1,2,3],"perdida":0,"peso":1}]}`,
		"época negativa":             `{"tipo":"epoca","epoca":-1,"w":[0,0]}`,
		"config con lambda negativo": `{"tipo":"asignacion","ruta":"x","num_bloques":3,"tam_bloque":8,"d":2,"config":{"lambda":-0.1,"peso_pos":1,"peso_neg":1}}`,
		"config con peso negativo":   `{"tipo":"asignacion","ruta":"x","num_bloques":3,"tam_bloque":8,"d":2,"config":{"lambda":0,"peso_pos":-1,"peso_neg":1}}`,
		"parcial con peso cero":      `{"tipo":"gradiente","parciales":[{"grad":[1,2],"perdida":0,"peso":0}]}`,
		"parcial con peso negativo":  `{"tipo":"gradiente","parciales":[{"grad":[1,2],"perdida":0,"peso":1},{"grad":[1,2],"perdida":0,"peso":-3}]}`,
		"error sin texto":            `{"tipo":"error","epoca":1}`,
		"fin sin motivo":             `{"tipo":"fin"}`,
	}
	for nombre, linea := range casos {
		t.Run(nombre, func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			defer b.Close()
			go a.Write([]byte(linea + "\n"))
			_, err := Nueva(b).Recibir(plazoTest)
			if !errors.Is(err, ErrInvalido) {
				t.Fatalf("err = %v, quiere ErrInvalido", err)
			}
		})
	}
}

// Tras un mensaje inválido pero bien formado, la conexión sigue alineada.
func TestDespuesDeUnMensajeInvalidoSeLeeElSiguiente(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	go a.Write([]byte("{\"tipo\":\"saludo\"}\n{\"tipo\":\"fin\",\"motivo\":\"completo\"}\n"))
	c := Nueva(b)
	if _, err := c.Recibir(plazoTest); !errors.Is(err, ErrInvalido) {
		t.Fatalf("primer mensaje: err = %v", err)
	}
	if m, err := c.Recibir(plazoTest); err != nil || m.Tipo != Fin {
		t.Fatalf("segundo mensaje: %+v, %v", m, err)
	}
}

func TestNumeroFueraDeRangoEnElJSONSeRechaza(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	go a.Write([]byte(`{"tipo":"epoca","w":[1e999,0]}` + "\n"))
	if _, err := Nueva(b).Recibir(plazoTest); err == nil {
		t.Fatal("1e999 no cabe en un float64: esperaba error")
	}
}

func TestLineaSinFinSeCortaEnMaxMensaje(t *testing.T) {
	l := &lectorLimitado{r: strings.NewReader(strings.Repeat("x", 100)), restante: 10}
	leido, err := io.ReadAll(l)
	if len(leido) != 10 || !errors.Is(err, ErrMuyGrande) {
		t.Fatalf("leyó %d bytes, err = %v", len(leido), err)
	}
}

// Aunque el diseño tiene un solo escritor por conexión, Enviar no mezcla líneas
// si dos goroutines escriben a la vez.
func TestEscritoresConcurrentesNoMezclanLineas(t *testing.T) {
	const escritores, porEscritor = 8, 25
	enCadaTransporte(t, func(t *testing.T, a, b *Conexion) {
		var wg sync.WaitGroup
		for e := range escritores {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := make([]float64, 500+e) // líneas largas: varias escrituras del SO
				for j := range w {
					w[j] = float64(e)
				}
				for i := range porEscritor {
					if err := a.Enviar(Mensaje{Tipo: Epoca, Epoca: i, Shard: e, W: w}, plazoTest); err != nil {
						t.Errorf("escritor %d: %v", e, err)
						return
					}
				}
			}()
		}
		siguiente := make([]int, escritores)
		for range escritores * porEscritor {
			m, err := b.Recibir(plazoTest)
			if err != nil {
				t.Fatalf("línea mezclada o corrupta: %v", err)
			}
			if len(m.W) != 500+m.Shard || m.W[0] != float64(m.Shard) || m.W[len(m.W)-1] != float64(m.Shard) {
				t.Fatalf("mensaje del escritor %d con contenido de otro", m.Shard)
			}
			if m.Epoca != siguiente[m.Shard] {
				t.Fatalf("escritor %d: llegó %d, se esperaba %d (TCP conserva el orden)", m.Shard, m.Epoca, siguiente[m.Shard])
			}
			siguiente[m.Shard]++
		}
		wg.Wait()
	})
}

// PR-42 H9: Bytes cuenta lo escrito y lo leído por la conexión, y se puede
// leer desde otra goroutine mientras se envía y recibe (go test -race).
func TestConexionCuentaBytesEnviadosYRecibidos(t *testing.T) {
	a, b := net.Pipe()
	ca, cb := Nueva(a), Nueva(b)
	defer ca.Cerrar()
	defer cb.Cerrar()
	m := Mensaje{Tipo: Epoca, Epoca: 1, Shard: 0, W: []float64{0.5, -1, 2}}
	linea, err := Codificar(&m)
	if err != nil {
		t.Fatal(err)
	}
	listo := make(chan struct{})
	go func() {
		defer close(listo)
		for range 3 {
			ca.Bytes() // lectura concurrente
		}
	}()
	enviado := make(chan struct{})
	go func() {
		defer close(enviado)
		for range 2 {
			if err := ca.Enviar(m, time.Second); err != nil {
				t.Error(err)
			}
		}
	}()
	for range 2 {
		if _, err := cb.Recibir(time.Second); err != nil {
			t.Fatal(err)
		}
	}
	<-enviado
	<-listo
	env, _ := ca.Bytes()
	_, rec := cb.Bytes()
	if env != int64(2*len(linea)) || rec != env {
		t.Fatalf("enviados %d, recibidos %d; se esperaban %d", env, rec, 2*len(linea))
	}
}

// PR-92 H3: us_cpu = 0 es un dato (cálculo más corto que la resolución de
// getrusage) y viaja; la ausencia de us_cpu significa «el nodo no lo informa».
func TestMicrosCPUCeroViajaYNegativoSeRechaza(t *testing.T) {
	cero := int64(0)
	b, err := json.Marshal(Mensaje{Tipo: Gradiente, Shard: 1, MicrosCPU: &cero})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"us_cpu":0`) {
		t.Fatalf("us_cpu = 0 no viaja: %s", b)
	}
	var ida Mensaje
	if err := json.Unmarshal(b, &ida); err != nil || ida.MicrosCPU == nil || *ida.MicrosCPU != 0 {
		t.Fatalf("ida y vuelta: %v %v", err, ida.MicrosCPU)
	}
	sin, _ := json.Marshal(Mensaje{Tipo: Gradiente, Shard: 1})
	if strings.Contains(string(sin), "us_cpu") {
		t.Fatalf("sin dato no debe enviarse: %s", sin)
	}
	ok := Mensaje{Tipo: Gradiente, Shard: 1, MicrosCPU: &cero, Parciales: []ml.Parcial{{Grad: []float64{1, 2}, Perdida: 1, Peso: 1}}}
	if err := ok.Validar(); err != nil {
		t.Fatalf("us_cpu = 0 es válido: %v", err)
	}
	neg := int64(-1)
	m := Mensaje{Tipo: Gradiente, Shard: 1, MicrosCPU: &neg}
	if err := m.Validar(); err == nil || !strings.Contains(err.Error(), "negativo") {
		t.Fatalf("us_cpu negativo debe rechazarse por negativo: %v", err)
	}
}
