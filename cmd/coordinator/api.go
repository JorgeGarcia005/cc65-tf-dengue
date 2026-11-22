package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// PlazoCierreHTTP es lo que espera el cierre ordenado a que terminen las
// peticiones en curso tras SIGTERM.
const PlazoCierreHTTP = 10 * time.Second

// ejecutarAPI es el modo servicio del coordinador (issue #28, H2 del PR #39):
// levanta el clúster, espera a los nodos y sirve la API REST en lnHTTP hasta
// que ctx se cancele (SIGTERM/SIGINT). Cada POST /api/v1/entrenamientos corre
// un entrenamiento distribuido sobre los mismos nodos; al terminar, la API
// publica el modelo nuevo. Al cerrar, cancela el entrenamiento en curso y
// envía `fin` a los nodos.
// baseDatos son la persistencia y la caché de la API (#29). Con valores nil
// la API usa el almacén en memoria y no tiene caché (tests y modo local).
type baseDatos struct {
	Almacen almacen.Almacen
	Cache   almacen.Cache
	// Firmador emite y verifica los JWT (prepararSeguridad, obligatorio en
	// modo servicio); nil = login y rutas protegidas en 503 (solo tests).
	Firmador *auth.Firmador
}

func ejecutarAPI(ctx context.Context, ln, lnHTTP net.Listener, op opciones, modelo []byte, bd baseDatos, logf func(string, ...any)) error {
	base, err := api.CargarModelo(modelo)
	if err != nil {
		ln.Close()
		lnHTTP.Close()
		return err
	}
	c, p, err := prepararCluster(ln, op, logf)
	if err != nil {
		lnHTTP.Close()
		return err
	}
	defer c.Cerrar()
	// Antes de api.Nuevo, que guarda el modelo del archivo como vigente en un
	// almacén vacío: uno incompatible no debe quedar persistido.
	if base.Esquema.D() != p.d {
		lnHTTP.Close()
		return fmt.Errorf("el modelo tiene %d variables y los shards %d: regenera el gold y el modelo juntos", base.Esquema.D(), p.d)
	}
	ctx, cancelarNodos := context.WithCancel(ctx)
	defer cancelarNodos()
	ent := &entrenadorCluster{calc: c, d: p.d, lambda: op.lambda, progreso: op.progreso, logf: logf}
	srv, err := api.Nuevo(api.Opciones{Modelo: base, Entrenador: ent, Cluster: c, Almacen: bd.Almacen, Cache: bd.Cache, Firmador: bd.Firmador,
		Evaluacion: op.evaluacion, OrigenesCORS: op.origenes, Log: logf})
	if err != nil {
		lnHTTP.Close()
		return err
	}
	// El modelo que publica la API puede venir del almacén (el vigente de una
	// ejecución anterior): debe tener las mismas variables que los shards.
	if dm := srv.ModeloActual().Esquema.D(); dm != p.d {
		srv.Cerrar()
		lnHTTP.Close()
		return fmt.Errorf("el modelo tiene %d variables y los shards %d: regenera el gold y el modelo juntos", dm, p.d)
	}
	// La API responde desde ya (salud, modelo, predicciones y estado del
	// clúster mientras se registran los nodos); un entrenamiento pedido antes
	// de que estén todos falla con un mensaje claro. La espera se cancela con
	// ctx (SIGTERM) y se espera a su goroutine antes de cerrar (PR-39 H8).
	var wgNodos sync.WaitGroup
	wgNodos.Add(1)
	go func() {
		defer wgNodos.Done()
		if err := c.EsperarNodosContext(ctx, op.nodos); err != nil {
			ent.sinNodos.Store(&err)
			logf("los nodos no se registraron: %v", err)
			return
		}
		ent.listo.Store(true)
		logf("%d nodos registrados: la API acepta entrenamientos", op.nodos)
	}()

	logf("API en http://%s (Swagger UI en /docs)", lnHTTP.Addr())
	errHTTP := api.Servir(ctx, api.NuevoHTTP(srv.Handler()), lnHTTP, PlazoCierreHTTP)
	// Orden del cierre: ya no entran peticiones; se cancela y espera el
	// entrenamiento en curso (Servidor.Cerrar) y la espera de nodos; recién
	// entonces se avisa `fin` a los nodos sanos y defer cierra el clúster.
	// ctx ya está cancelado salvo que Servir haya fallado: se cancela aparte.
	cancelarNodos()
	srv.Cerrar()
	wgNodos.Wait()
	c.Terminar()
	return errHTTP
}

// entrenadorCluster implementa api.Entrenador sobre el coordinador del
// clúster. El coordinador ya serializa las épocas (muBucle) y la API admite un
// solo entrenamiento a la vez.
type entrenadorCluster struct {
	calc     ml.CalculadorFalible
	d        int
	lambda   float64
	progreso int
	logf     func(string, ...any)
	listo    atomic.Bool           // todos los nodos se registraron
	sinNodos atomic.Pointer[error] // por qué no se registraron, si falló
}

// Entrenar corre p.Epocas épocas en el clúster y devuelve los pesos con el
// sesgo al final. La regularización viaja a los nodos al registrarse
// (protocolo.ConfigML), así que un λ distinto al del clúster se rechaza en vez
// de entrenar en silencio con otro.
func (e *entrenadorCluster) Entrenar(ctx context.Context, p api.ParametrosEntrenamiento, progreso func(int, float64)) ([]float64, error) {
	if !e.listo.Load() {
		if err := e.sinNodos.Load(); err != nil {
			return nil, fmt.Errorf("el clúster no tiene sus nodos: %w", *err)
		}
		return nil, errors.New("el clúster todavía espera a que se registren sus nodos")
	}
	if p.D != e.d {
		return nil, fmt.Errorf("el modelo tiene %d variables y el clúster %d", p.D, e.d)
	}
	if p.Lambda != e.lambda {
		return nil, fmt.Errorf("el clúster está configurado con lambda=%g (flag -lambda del coordinador); se pidió %g", e.lambda, p.Lambda)
	}
	calc := &conContexto{ctx: ctx, calc: e.calc, progreso: progreso, cada: e.progreso, logf: e.logf}
	ini := time.Now()
	res, err := ml.EntrenarFalible(calc, e.d, ml.Entrenamiento{Epocas: p.Epocas, Tasa: p.Tasa})
	if err != nil {
		return nil, err
	}
	e.logf("entrenamiento vía API: %d épocas en %.1f s, pérdida %.6f → %.6f",
		len(res.Historia), time.Since(ini).Seconds(), res.Historia[0], res.Historia[len(res.Historia)-1])
	return res.W, nil
}

// errCancelado indica que el entrenamiento se detuvo porque se canceló ctx.
var errCancelado = errors.New("entrenamiento cancelado")

// conContexto revisa ctx antes de cada época (una época en curso termina: su
// duración la acota -timeout-epoca) e informa el progreso tras cada una.
type conContexto struct {
	ctx      context.Context
	calc     ml.CalculadorFalible
	progreso func(int, float64)
	cada     int
	logf     func(string, ...any)
	epoca    int
}

func (c *conContexto) GradienteErr(w []float64) (ml.Gradiente, error) {
	if err := c.ctx.Err(); err != nil {
		return ml.Gradiente{}, fmt.Errorf("%w en la época %d: %w", errCancelado, c.epoca, err)
	}
	g, err := c.calc.GradienteErr(w)
	if err != nil {
		return g, err
	}
	if c.cada > 0 && c.epoca%c.cada == 0 {
		c.logf("época %d: pérdida %.6f", c.epoca, g.Perdida)
	}
	c.epoca++
	if c.progreso != nil {
		c.progreso(c.epoca, g.Perdida)
	}
	return g, nil
}

// Comprobación en compilación: el coordinador sirve como calculador y como
// estado del clúster de la API.
var (
	_ ml.CalculadorFalible = (*cluster.Coordinador)(nil)
	_ api.EstadoCluster    = (*cluster.Coordinador)(nil)
)
