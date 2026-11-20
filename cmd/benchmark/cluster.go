package main

// Benchmark del clúster (issue #32, modo -cluster): escalabilidad del
// entrenamiento distribuido por número de nodos.
//
// Protocolo (fijado en código para que sea reproducible):
//  1. Para cada N de -nodos se parte el gold en N shards (límites de bloque
//     fijos, como en el despliegue) y se levanta en el mismo proceso un
//     coordinador y N nodos que hablan por TCP en 127.0.0.1, con el protocolo
//     real (JSON por línea) y -workers-nodo goroutines por nodo. Con 1 worker
//     por nodo cada nodo equivale a una máquina de un núcleo; la red es la de
//     loopback, así que el costo de red medido es el mínimo.
//  2. Una corrida de calentamiento (carga los shards en los nodos) y R
//     repeticiones medidas de -epocas épocas cada una, sobre los mismos nodos.
//  3. Cada repetición se compara bit a bit con el entrenamiento secuencial
//     antes de registrar su tiempo: si difiere, el benchmark falla.
//  4. Por repetición se registra el tiempo por época y su descomposición:
//     cómputo del nodo más lento, reducción en el coordinador, red y espera
//     (el resto), y bytes por época. La serialización JSON se mide aparte,
//     codificando y decodificando los mensajes de una época (CPU).
//  5. Estimador: media recortada con IC 95 % por bootstrap. Speedup y
//     eficiencia contra 1 nodo; fracción serial de Karp-Flatt
//     e = (1/S − 1/p) / (1 − 1/p).

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// opcionesCluster del modo -cluster.
type opcionesCluster struct {
	nodos       []int
	reps        int
	epocas      int
	workersNodo int
	tasa        float64
	lambda      float64
	recorte     float64
	logf        func(string, ...any)
	// Para los tests: plazo de registro (0 = 30 s) y cuántos nodos se
	// conectan de verdad (0 = todos).
	timeoutRegistro time.Duration
	conectar        int
}

// medicionCluster es el resultado para un número de nodos.
type medicionCluster struct {
	Nodos               int                 `json:"nodos"`
	SegPorEpoca         estadistica.Resumen `json:"seg_por_epoca"`
	Speedup             float64             `json:"speedup"`
	Eficiencia          float64             `json:"eficiencia"`
	KarpFlatt           *float64            `json:"karp_flatt,omitempty"` // no se define con 1 nodo
	Desglose            desglose            `json:"desglose_por_epoca"`
	BytesPorEpoca       float64             `json:"bytes_por_epoca"`
	Equivalente         bool                `json:"equivalente_bit_a_bit"`
	Repeticiones        int                 `json:"repeticiones"`
	EpocasPorRepeticion int                 `json:"epocas_por_repeticion"`
}

// desglose de una época (medias sobre las repeticiones, en segundos).
type desglose struct {
	Epoca          float64 `json:"seg_epoca"`
	CalculoNodoMax float64 `json:"seg_calculo_nodo_mas_lento"`
	CalculoNodos   float64 `json:"seg_calculo_todos_los_nodos"`
	Reduccion      float64 `json:"seg_reduccion"`
	RedYEspera     float64 `json:"seg_red_y_espera"` // época − cómputo del más lento − reducción
	// JSON medido aparte (CPU de un hilo): el total de todos los shards, que
	// en el clúster se reparte entre nodos y lectoras del coordinador, y el del
	// shard más pesado, que es lo que cae en el camino crítico y está incluido
	// en red y espera.
	SerializacionTotal float64 `json:"seg_json_cpu_todos_los_shards"`
	SerializacionShard float64 `json:"seg_json_shard_mas_pesado"`
	// FraccionRed es (red y espera) / época: cuánto de la época no es cómputo.
	// Todos los campos del desglose son medias aritméticas sobre las
	// repeticiones (el recorte se aplica solo a seg_por_epoca).
	FraccionRed float64 `json:"fraccion_red_y_espera"`
}

type reporteCluster struct {
	Generado   string            `json:"generado"`
	Fuente     string            `json:"fuente"`
	Maquina    map[string]any    `json:"maquina"`
	Protocolo  map[string]any    `json:"protocolo"`
	Mediciones []medicionCluster `json:"mediciones"`
}

// medirCluster corre el benchmark sobre train (el gold completo).
func medirCluster(train gold.Archivo, op opcionesCluster) ([]medicionCluster, error) {
	if op.reps < 1 || op.epocas < 1 || op.workersNodo < 1 || len(op.nodos) == 0 {
		return nil, errors.New("cluster: -reps, -epocas-cluster, -workers-nodo y -nodos deben ser ≥ 1")
	}
	pos, neg := ml.PesosBalanceados(train.Y)
	cfg := ml.Config{TamBloque: train.TamBloque, Lambda: op.lambda, PesoPos: pos, PesoNeg: neg}
	ent := ml.Entrenamiento{Epocas: op.epocas, Tasa: op.tasa}
	op.logf("referencia secuencial: %d épocas sobre %d filas × %d variables", op.epocas, train.N, train.D)
	ref := ml.Entrenar(ml.NuevoSecuencial(train.Dataset, cfg), train.D, ent)

	// Base = el menor número de nodos: se ordena y se quitan repetidos.
	nodos := slices.Clone(op.nodos)
	slices.Sort(nodos)
	nodos = slices.Compact(nodos)
	if nodos[0] < 1 {
		return nil, errors.New("cluster: -nodos debe tener valores ≥ 1")
	}
	var out []medicionCluster
	for _, n := range nodos {
		m, err := medirNodos(train, n, cfg, ent, ref, op)
		if err != nil {
			return nil, fmt.Errorf("%d nodos: %w", n, err)
		}
		out = append(out, m)
	}
	// Speedup y eficiencia contra la menor configuración medida. Karp-Flatt
	// solo se define contra 1 nodo (e = (1/S − 1/p)/(1 − 1/p) supone que la
	// base es la ejecución en un procesador).
	base := out[0]
	for i := range out {
		p := float64(out[i].Nodos) / float64(base.Nodos)
		s := base.SegPorEpoca.MediaRecortada / out[i].SegPorEpoca.MediaRecortada
		out[i].Speedup, out[i].Eficiencia = s, s/p
		if base.Nodos == 1 && p > 1 {
			e := (1/s - 1/p) / (1 - 1/p)
			out[i].KarpFlatt = &e
		}
	}
	return out, nil
}

// medirNodos levanta el clúster con n nodos y mide op.reps repeticiones.
func medirNodos(train gold.Archivo, n int, cfg ml.Config, ent ml.Entrenamiento, ref ml.Resultado, op opcionesCluster) (medicionCluster, error) {
	dir, err := os.MkdirTemp("", "cc65-bench-cluster-")
	if err != nil {
		return medicionCluster{}, err
	}
	defer os.RemoveAll(dir)
	partes, err := gold.Particionar(train, n)
	if err != nil {
		return medicionCluster{}, err
	}
	var shards []cluster.Shard
	for s, p := range partes {
		ruta := filepath.Join(dir, fmt.Sprintf("shard_%d.bin", s))
		if err := gold.Escribir(ruta, p); err != nil {
			return medicionCluster{}, err
		}
		desc, _, err := cluster.LeerShard(ruta)
		if err != nil {
			return medicionCluster{}, err
		}
		desc.Ruta = ruta
		shards = append(shards, desc)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return medicionCluster{}, err
	}
	tReg := op.timeoutRegistro
	if tReg <= 0 {
		tReg = 30 * time.Second
	}
	c, err := cluster.Nuevo(ln, cluster.Opciones{
		Shards: shards, D: train.D, Config: cfg,
		TimeoutRegistro: tReg, TimeoutEpoca: 5 * time.Minute,
	})
	if err != nil {
		ln.Close()
		return medicionCluster{}, err
	}
	// Orden de salida (LIFO): Terminar (si se llegó a registrar) → Cerrar,
	// que corta las conexiones → esperar a los nodos. Al revés, un error
	// antes de Terminar dejaba a los nodos bloqueados leyendo (PR-42 H1).
	var wg sync.WaitGroup
	defer wg.Wait()
	defer c.Cerrar()
	conectar := n
	if op.conectar > 0 {
		conectar = min(op.conectar, n)
	}
	for i := range conectar {
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			return medicionCluster{}, err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			(&cluster.Nodo{Nombre: fmt.Sprintf("nodo-%d", i), Workers: op.workersNodo}).Servir(conn)
		}()
	}
	if err := c.EsperarNodos(n); err != nil {
		return medicionCluster{}, err
	}
	defer c.Terminar()

	op.logf("%d nodos: calentamiento (carga de shards) y %d repeticiones de %d épocas", n, op.reps, ent.Epocas)
	if _, err := ml.EntrenarFalible(c, train.D, ent); err != nil {
		return medicionCluster{}, err
	}
	var segEpoca []float64
	var suma desglose
	var bytes float64
	for r := range op.reps {
		antes := c.Tiempos()
		ini := time.Now()
		res, err := ml.EntrenarFalible(c, train.D, ent)
		total := time.Since(ini).Seconds()
		if err != nil {
			return medicionCluster{}, err
		}
		if !mismosBits(ref.W, res.W) || !mismosBits(ref.Historia, res.Historia) {
			return medicionCluster{}, fmt.Errorf("repetición %d: los pesos del clúster difieren de los secuenciales", r)
		}
		d := c.Tiempos()
		e := float64(d.Epocas - antes.Epocas)
		segEpoca = append(segEpoca, total/e)
		suma.Epoca += (d.SegEpoca - antes.SegEpoca) / e
		suma.CalculoNodoMax += (d.SegCalculoMax - antes.SegCalculoMax) / e
		suma.CalculoNodos += (d.SegCalculoSuma - antes.SegCalculoSuma) / e
		suma.Reduccion += (d.SegReduccion - antes.SegReduccion) / e
		bytes += float64(d.BytesEnviados-antes.BytesEnviados+d.BytesRecibidos-antes.BytesRecibidos) / e
	}
	k := float64(op.reps)
	des := desglose{
		Epoca: suma.Epoca / k, CalculoNodoMax: suma.CalculoNodoMax / k,
		CalculoNodos: suma.CalculoNodos / k, Reduccion: suma.Reduccion / k,
	}
	des.RedYEspera = math.Max(0, des.Epoca-des.CalculoNodoMax-des.Reduccion)
	des.FraccionRed = des.RedYEspera / des.Epoca
	des.SerializacionTotal, des.SerializacionShard, err = serializacionPorEpoca(partes, ref.W, cfg)
	if err != nil {
		return medicionCluster{}, err
	}
	return medicionCluster{
		Nodos: n, SegPorEpoca: estadistica.Resumir(segEpoca, op.recorte), Desglose: des,
		BytesPorEpoca: bytes / k, Equivalente: true, Repeticiones: op.reps, EpocasPorRepeticion: ent.Epocas,
	}, nil
}

// serializacionPorEpoca mide (CPU, un hilo) lo que cuesta codificar y
// decodificar los mensajes de una época con los parciales reales de cada
// shard (calculados con los pesos de referencia) y el mismo json.Decoder que
// usa protocolo.Conexion. Devuelve el total de todos los shards (CPU sumada:
// en el clúster se reparte entre nodos y lectoras del coordinador) y el del
// shard más pesado (lo que cae en el camino crítico de una época). Cada medida
// agrupa k pasadas para no caer bajo la resolución del reloj (en Windows,
// ~0,5 ms) y se toma la mediana de 5.
func serializacionPorEpoca(partes []gold.Archivo, w []float64, cfg ml.Config) (total, shardMax float64, err error) {
	type par struct{ epoca, grad protocolo.Mensaje }
	var msgs []par
	for s, p := range partes {
		pool := ml.NuevoPool(nil, cfg, 1)
		parciales := pool.Parciales(p.Dataset, w)
		pool.Cerrar()
		msgs = append(msgs, par{
			epoca: protocolo.Mensaje{Tipo: protocolo.Epoca, Epoca: 1, Shard: s, W: w},
			grad: protocolo.Mensaje{Tipo: protocolo.Gradiente, Epoca: 1, Shard: s, BloqueInicial: 0,
				Parciales: parciales, MicrosCalculo: 1},
		})
	}
	idaYVuelta := func(m *protocolo.Mensaje) error {
		b, err := protocolo.Codificar(m)
		if err != nil {
			return err
		}
		var otro protocolo.Mensaje
		return json.NewDecoder(bytes.NewReader(b)).Decode(&otro)
	}
	medir := func(f func() error) (float64, error) {
		var errF error
		g := func() {
			if err := f(); err != nil {
				errF = err
			}
		}
		k := pasadasPorMedicion(g, medicionMinima)
		var ts []float64
		for range 5 {
			ini := time.Now()
			for range k {
				g()
			}
			ts = append(ts, time.Since(ini).Seconds()/float64(k))
		}
		return estadistica.Mediana(ts), errF
	}
	for i := range msgs {
		t, err := medir(func() error {
			if err := idaYVuelta(&msgs[i].epoca); err != nil {
				return err
			}
			return idaYVuelta(&msgs[i].grad)
		})
		if err != nil {
			return 0, 0, err
		}
		total += t
		shardMax = max(shardMax, t)
	}
	return total, shardMax, nil
}

func imprimirCluster(rep reporteCluster) {
	fmt.Println(rep.Fuente)
	fmt.Printf("%6s %12s %16s %8s %8s %8s %10s %10s %13s %12s\n", "nodos", "ms/época", "IC95 ms", "speedup", "efic.", "K-F", "cómp. ms", "red+esp ms", "JSON/shard ms", "KB/época")
	for _, m := range rep.Mediciones {
		kf := "—"
		if m.KarpFlatt != nil {
			kf = fmt.Sprintf("%.3f", *m.KarpFlatt)
		}
		fmt.Printf("%6d %12.2f %7.2f–%-8.2f %8.2f %8.2f %8s %10.2f %10.2f %13.2f %12.1f\n",
			m.Nodos, 1000*m.SegPorEpoca.MediaRecortada, 1000*m.SegPorEpoca.IC95Inf, 1000*m.SegPorEpoca.IC95Sup,
			m.Speedup, m.Eficiencia, kf, 1000*m.Desglose.CalculoNodoMax, 1000*m.Desglose.RedYEspera,
			1000*m.Desglose.SerializacionShard, m.BytesPorEpoca/1024)
	}
}

// ejecutarCluster es el modo -cluster de main.
func ejecutarCluster(train gold.Archivo, fuente string, op opcionesCluster, salida string) {
	med, err := medirCluster(train, op)
	if err != nil {
		log.Fatal(err)
	}
	rep := reporteCluster{
		Generado: time.Now().Format(time.RFC3339), Fuente: fuente, Maquina: maquina.Describir(),
		Protocolo: map[string]any{
			"nodos": op.nodos, "repeticiones": op.reps, "epocas_por_repeticion": op.epocas,
			"workers_por_nodo": op.workersNodo, "tasa": op.tasa, "lambda": op.lambda,
			"red":           "TCP en 127.0.0.1 (loopback), protocolo JSON por línea real",
			"calentamiento": "1 entrenamiento descartado (carga de shards)",
			"estimador":     fmt.Sprintf("media recortada %.0f %% + IC95 bootstrap (2000)", op.recorte*100),
			"verificacion":  "cada repetición se compara bit a bit con el secuencial antes de registrarse",
			"desglose":      "medias aritméticas: red y espera = época − cómputo del nodo más lento (sus shards en serie) − reducción; JSON medido aparte (CPU de un hilo, k pasadas agrupadas): total de los shards y shard más pesado (incluido en red y espera)",
			"karp_flatt":    "e = (1/S − 1/p) / (1 − 1/p), p = nodos / nodos de la base",
		},
		Mediciones: med,
	}
	imprimirCluster(rep)
	escribir(rep, salida)
}
