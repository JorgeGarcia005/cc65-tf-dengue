// Command coordinator es el parameter server del clúster (Parte C): espera a
// los nodos por TCP, entrena la regresión logística de forma distribuida y
// escribe el reporte del entrenamiento.
//
// Uso (los nodos se conectan con cmd/node):
//
//	go run ./cmd/coordinator -gold data/gold -nodos 3 -epocas 600 -tasa 2
//
// Con -api :8080 pasa a modo servicio: no entrena al arrancar, sirve la API
// REST (Swagger UI en /docs) y entrena en el clúster con cada
// POST /api/v1/entrenamientos, hasta recibir SIGTERM.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

type opciones struct {
	gold      string // carpeta con shard_<s>.bin (y train.bin si se verifica)
	rutaNodos string // la misma carpeta tal como la ven los nodos
	nodos     int
	epocas    int
	tasa      float64
	lambda    float64
	progreso  int // registra la pérdida cada tantas épocas; 0 = nunca
	verificar bool
	tRegistro time.Duration
	tEpoca    time.Duration
	// evaluacion es reports/evaluacion.json para GET /api/v1/riesgo y
	// /api/v1/evaluacion (modo servicio); vacío = esas rutas dan 503.
	evaluacion []byte
}

// reporte usa las mismas claves que reports/entrenamiento.json para épocas,
// hiperparámetros, pesos e historia, así `cmd/entrenar -comparar` los compara.
type reporte struct {
	Generado      string         `json:"generado"`
	NodosPedidos  int            `json:"nodos_pedidos"`
	Shards        int            `json:"shards"`
	Epocas        int            `json:"epocas"`
	Tasa          float64        `json:"tasa_aprendizaje"`
	Lambda        float64        `json:"lambda"`
	PesoPos       float64        `json:"peso_positivos"`
	PesoNeg       float64        `json:"peso_negativos"`
	FilasTrain    int            `json:"filas_train"`
	D             int            `json:"variables"`
	TamBloque     int            `json:"tam_bloque"`
	Bloques       int            `json:"bloques"`
	SegTotal      float64        `json:"seg_entrenamiento"`
	SegPorEpoca   float64        `json:"seg_por_epoca"`
	Cluster       cluster.Estado `json:"cluster"`
	Verificado    *bool          `json:"igual_al_secuencial,omitempty"`
	SegSecuencial float64        `json:"seg_secuencial,omitempty"`
	Historia      []float64      `json:"perdida_por_epoca"`
	Pesos         []float64      `json:"pesos"`
	Sesgo         float64        `json:"sesgo"`
}

func main() {
	var op opciones
	escucha := flag.String("escucha", ":9000", "dirección TCP donde se registran los nodos")
	flag.StringVar(&op.gold, "gold", "data/gold", "carpeta con shard_<s>.bin")
	flag.StringVar(&op.rutaNodos, "ruta-nodos", "", "esa carpeta tal como la ven los nodos (por defecto, igual que -gold)")
	flag.IntVar(&op.nodos, "nodos", 3, "nodos que deben registrarse antes de empezar")
	flag.IntVar(&op.epocas, "epocas", 600, "épocas de descenso de gradiente full-batch")
	flag.Float64Var(&op.tasa, "tasa", 2.0, "tasa de aprendizaje")
	flag.Float64Var(&op.lambda, "lambda", 1e-4, "regularización L2")
	flag.IntVar(&op.progreso, "progreso", 50, "registra la pérdida cada tantas épocas (0 = nunca)")
	flag.BoolVar(&op.verificar, "verificar", false, "repite el entrenamiento en secuencial con train.bin y exige pesos idénticos bit a bit")
	flag.DurationVar(&op.tRegistro, "timeout-registro", 60*time.Second, "espera máxima a que se registren los nodos")
	flag.DurationVar(&op.tEpoca, "timeout-epoca", 60*time.Second, "plazo de cada shard por época")
	salida := flag.String("salida", "reports/entrenamiento_cluster.json", "reporte")
	dirAPI := flag.String("api", "", "modo servicio: sirve la API REST en esta dirección (p. ej. :8080) en vez de entrenar una vez y salir")
	rutaModelo := flag.String("modelo", "data/gold/modelo.json", "modo servicio: modelo inicial que publica la API")
	uriMongo := flag.String("mongo", os.Getenv("MONGO_URI"), "modo servicio: URI de MongoDB (vacío = en memoria; por defecto $MONGO_URI)")
	baseMongo := flag.String("base", "cc65_dengue", "modo servicio: base de datos de MongoDB")
	dirRedis := flag.String("redis", os.Getenv("REDIS_ADDR"), "modo servicio: host:puerto de Redis (vacío = sin caché; por defecto $REDIS_ADDR)")
	rutaEval := flag.String("evaluacion", os.Getenv("EVALUACION"), "modo servicio: reports/evaluacion.json para /api/v1/riesgo y /api/v1/evaluacion (por defecto $EVALUACION)")
	flag.Parse()

	ln, err := net.Listen("tcp", *escucha)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("coordinador escuchando en %s; espera %d nodos", ln.Addr(), op.nodos)
	if *dirAPI != "" {
		if op.evaluacion, err = leerEvaluacion(*rutaEval, log.Printf); err != nil {
			log.Fatal(err)
		}
		if err := servicio(ln, *dirAPI, *rutaModelo, *uriMongo, *baseMongo, *dirRedis, op); err != nil {
			log.Fatal(err)
		}
		return
	}
	_, err = ejecutar(ln, op, log.Printf, func(rep reporte) error {
		if err := escribirJSON(*salida, rep); err != nil {
			return err
		}
		log.Printf("escrito %s", *salida)
		// A partir de aquí el trabajo está hecho y guardado. Al recibir `fin`
		// los nodos terminan, y Compose con --exit-code-from detiene los demás
		// contenedores con SIGTERM: se ignora para salir con el código propio.
		signal.Ignore(syscall.SIGTERM)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

// servicio corre el modo API hasta SIGTERM o SIGINT (docker compose stop).
func servicio(ln net.Listener, dir, rutaModelo, uriMongo, baseMongo, dirRedis string, op opciones) error {
	modelo, err := os.ReadFile(rutaModelo)
	if err != nil {
		ln.Close()
		return fmt.Errorf("modelo inicial: %w", err)
	}
	bd, cerrarBD, err := conectarBD(uriMongo, baseMongo, dirRedis, log.Printf)
	if err != nil {
		ln.Close()
		return err
	}
	defer cerrarBD() // después de ejecutarAPI, que ya cerró el servidor
	if bd.Almacen == nil {
		bd.Almacen = almacen.NuevaMemoria() // la siembra y la API comparten el mismo almacén
	}
	ctxSeg, cancelarSeg := context.WithTimeout(context.Background(), plazoConexionBD)
	bd.Firmador, err = prepararSeguridad(ctxSeg, bd.Almacen, os.Getenv, log.Printf)
	cancelarSeg()
	if err != nil {
		ln.Close()
		return err
	}
	lnHTTP, err := net.Listen("tcp", dir)
	if err != nil {
		ln.Close()
		return err
	}
	ctx, parar := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer parar()
	err = ejecutarAPI(ctx, ln, lnHTTP, op, modelo, bd, log.Printf)
	log.Printf("coordinador detenido")
	return err
}

// cargarShards lee shard_0.bin, shard_1.bin… de dir y devuelve su descripción
// para el coordinador y las etiquetas de todas las filas (para los pesos por
// clase). El coordinador no conserva las matrices: los datos viven en los nodos.
func cargarShards(dir, rutaNodos string) (shards []cluster.Shard, d, tam int, y []float64, err error) {
	for s := 0; ; s++ {
		nombre := fmt.Sprintf("shard_%d.bin", s)
		desc, a, err := cluster.LeerShard(filepath.Join(dir, nombre))
		if errors.Is(err, os.ErrNotExist) && s > 0 {
			return shards, d, tam, y, nil
		}
		if err != nil {
			return nil, 0, 0, nil, fmt.Errorf("shard %d: %w", s, err)
		}
		if s == 0 {
			d, tam = a.D, a.TamBloque
		} else if a.D != d || a.TamBloque != tam {
			return nil, 0, 0, nil, fmt.Errorf("%s: d=%d y bloque=%d, el shard 0 tiene d=%d y bloque=%d", nombre, a.D, a.TamBloque, d, tam)
		}
		desc.Ruta = filepath.ToSlash(filepath.Join(rutaNodos, nombre))
		shards = append(shards, desc)
		y = append(y, a.Y...)
	}
}

// conProgreso registra la pérdida cada cierto número de épocas.
type conProgreso struct {
	ml.CalculadorFalible
	cada, epoca int
	logf        func(string, ...any)
}

func (c *conProgreso) GradienteErr(w []float64) (ml.Gradiente, error) {
	g, err := c.CalculadorFalible.GradienteErr(w)
	if err == nil && c.cada > 0 && c.epoca%c.cada == 0 {
		c.logf("época %d: pérdida %.6f", c.epoca, g.Perdida)
	}
	c.epoca++
	return g, err
}

// preparado describe el gold que atiende el clúster.
type preparado struct {
	shards  []cluster.Shard
	d, tam  int
	y       []float64
	cfg     ml.Config
	bloques int
}

// prepararCluster lee los shards, calcula los pesos por clase y crea el
// coordinador sobre ln (sin esperar a los nodos). Si falla, cierra ln.
func prepararCluster(ln net.Listener, op opciones, logf func(string, ...any)) (*cluster.Coordinador, preparado, error) {
	if op.rutaNodos == "" {
		op.rutaNodos = op.gold
	}
	shards, d, tam, y, err := cargarShards(op.gold, op.rutaNodos)
	if err != nil {
		ln.Close()
		return nil, preparado{}, err
	}
	pos, neg := ml.PesosBalanceados(y)
	cfg := ml.Config{TamBloque: tam, Lambda: op.lambda, PesoPos: pos, PesoNeg: neg}
	bloques := shards[len(shards)-1].BloqueInicial + shards[len(shards)-1].NumBloques
	logf("%d shards, %d filas, %d variables, %d bloques de %d filas", len(shards), len(y), d, bloques, tam)
	c, err := cluster.Nuevo(ln, cluster.Opciones{
		Shards: shards, D: d, Config: cfg,
		TimeoutRegistro: op.tRegistro, TimeoutEpoca: op.tEpoca, Log: logf,
	})
	if err != nil {
		ln.Close()
		return nil, preparado{}, err
	}
	return c, preparado{shards: shards, d: d, tam: tam, y: y, cfg: cfg, bloques: bloques}, nil
}

// ejecutar corre un entrenamiento completo sobre ln. guardar recibe el reporte
// (también el de un entrenamiento abortado, con lo avanzado) antes de que los
// nodos reciban `fin`: mientras el coordinador verifica y escribe, los nodos
// siguen vivos y nadie da el despliegue por terminado.
func ejecutar(ln net.Listener, op opciones, logf func(string, ...any), guardar func(reporte) error) (reporte, error) {
	if guardar == nil {
		guardar = func(reporte) error { return nil }
	}
	c, p, err := prepararCluster(ln, op, logf)
	if err != nil {
		return reporte{}, err
	}
	defer c.Cerrar()
	if err := c.EsperarNodos(op.nodos); err != nil {
		return reporte{}, err
	}
	d, cfg := p.d, p.cfg
	pos, neg := cfg.PesoPos, cfg.PesoNeg

	ent := ml.Entrenamiento{Epocas: op.epocas, Tasa: op.tasa}
	ini := time.Now()
	res, errEnt := ml.EntrenarFalible(&conProgreso{CalculadorFalible: c, cada: op.progreso, logf: logf}, d, ent)
	seg := time.Since(ini).Seconds()

	rep := reporte{
		Generado: time.Now().Format(time.RFC3339), NodosPedidos: op.nodos, Shards: len(p.shards),
		Epocas: len(res.Historia), Tasa: op.tasa, Lambda: op.lambda, PesoPos: pos, PesoNeg: neg,
		FilasTrain: len(p.y), D: d, TamBloque: p.tam, Bloques: p.bloques,
		SegTotal: seg, SegPorEpoca: seg / float64(max(len(res.Historia), 1)), Cluster: c.Estado(),
		Historia: res.Historia, Pesos: res.W[:d], Sesgo: res.W[d],
	}
	if errEnt != nil {
		return rep, errors.Join(errEnt, guardar(rep))
	}
	logf("entrenamiento distribuido: %d épocas en %.1f s (%.1f ms/época), pérdida %.6f → %.6f, %d reasignaciones",
		op.epocas, seg, 1000*rep.SegPorEpoca, res.Historia[0], res.Historia[len(res.Historia)-1], rep.Cluster.Reasignaciones)

	if op.verificar {
		train, err := gold.Leer(filepath.Join(op.gold, "train.bin"))
		if err != nil {
			c.Terminar()
			return rep, fmt.Errorf("-verificar necesita train.bin: %w", err)
		}
		ini := time.Now()
		ref := ml.Entrenar(ml.NuevoSecuencial(train.Dataset, cfg), d, ent)
		rep.SegSecuencial = time.Since(ini).Seconds()
		igual := mismosBits(ref.W, res.W) && mismosBits(ref.Historia, res.Historia)
		rep.Verificado = &igual
		if !igual {
			err := errors.New("los pesos del clúster difieren de los secuenciales")
			defer c.Terminar()
			return rep, errors.Join(err, guardar(rep))
		}
		logf("verificado: pesos e historia idénticos bit a bit al secuencial (%.1f s)", rep.SegSecuencial)
	}
	err = guardar(rep)
	c.Terminar()
	return rep, err
}

func mismosBits(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for j := range a {
		if math.Float64bits(a[j]) != math.Float64bits(b[j]) {
			return false
		}
	}
	return true
}

func escribirJSON(ruta string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	return os.WriteFile(ruta, append(b, '\n'), 0o644)
}

// leerEvaluacion carga reports/evaluacion.json si existe. Si no existe, la API
// arranca igual y sus dos rutas responden 503 (la evaluación se genera aparte
// con go run ./cmd/evaluacion y no debe impedir predecir ni entrenar). Un
// archivo que existe pero está vacío es un error: es un reporte roto.
func leerEvaluacion(ruta string, logf func(string, ...any)) ([]byte, error) {
	if ruta == "" {
		logf("sin -evaluacion: /api/v1/riesgo y /api/v1/evaluacion responden 503")
		return nil, nil
	}
	b, err := os.ReadFile(ruta)
	if errors.Is(err, fs.ErrNotExist) {
		logf("evaluación no disponible (%s no existe): /api/v1/riesgo y /api/v1/evaluacion responden 503", ruta)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("evaluación: %w", err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("evaluación: %s está vacío; vuelva a generarlo con go run ./cmd/evaluacion", ruta)
	}
	logf("evaluación cargada de %s (%d bytes)", ruta, len(b))
	return b, nil
}
