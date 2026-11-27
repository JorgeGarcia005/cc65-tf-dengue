package cluster

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// Nodo es un nodo de cómputo: carga los shards que le asignan y calcula sus
// parciales por bloque.
//
// Concurrencia: una sola goroutine (Servir) lee y escribe la conexión y es
// dueña del mapa de shards. El paralelismo está dentro de ml.Pool: un único
// pool de Workers goroutines por proceso, que atiende cualquier shard
// (ADR-0005, objeción 3 del juez: un pool por shard multiplicaría las
// goroutines que compiten por los mismos núcleos).
type Nodo struct {
	Nombre  string
	Workers int
	Log     func(formato string, args ...any)

	shards map[int]gold.Archivo
	pool   *ml.Pool
	cfg    protocolo.ConfigML
	tam    int
}

// Servir registra el nodo por conn y atiende al coordinador hasta `fin`.
// Devuelve nil si el entrenamiento terminó completo, ErrAbortado si el
// coordinador abortó, o el error de conexión.
func (n *Nodo) Servir(conn net.Conn) error {
	if n.Log == nil {
		n.Log = func(string, ...any) {}
	}
	pc := protocolo.Nueva(conn)
	defer pc.Cerrar()
	defer func() {
		if n.pool != nil {
			n.pool.Cerrar()
			n.pool = nil
		}
	}()
	n.shards = map[int]gold.Archivo{}
	workers := max(n.Workers, 1)
	if err := pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Registro, Nodo: n.Nombre, Version: protocolo.Version, Workers: workers}, 0); err != nil {
		return err
	}
	for {
		m, err := pc.Recibir(0)
		if err == io.EOF {
			return errors.New("cluster: el coordinador cerró la conexión")
		}
		if err != nil {
			return err
		}
		var resp protocolo.Mensaje
		switch m.Tipo {
		case protocolo.Asignacion:
			resp = n.cargar(m, workers)
		case protocolo.Epoca:
			resp = n.calcular(m)
		case protocolo.Fin:
			n.Log("fin del entrenamiento: %s", m.Motivo)
			if m.Motivo == protocolo.MotivoAbortado {
				return ErrAbortado
			}
			return nil
		default:
			resp = fallo(m, "mensaje inesperado %q", m.Tipo)
		}
		err = pc.Enviar(resp, 0)
		if errors.Is(err, protocolo.ErrNoFinito) {
			// Un gradiente con NaN o ±Inf no es un dato: se informa como error.
			err = pc.Enviar(fallo(m, "gradiente no finito"), 0)
		}
		if err != nil {
			return err
		}
	}
}

func fallo(m protocolo.Mensaje, formato string, args ...any) protocolo.Mensaje {
	return protocolo.Mensaje{Tipo: protocolo.Error, Epoca: m.Epoca, Shard: m.Shard, Detalle: fmt.Sprintf(formato, args...)}
}

// cargar lee el shard asignado, comprueba que sea el que el coordinador espera
// y responde `listo` con lo que leyó de la cabecera del archivo.
func (n *Nodo) cargar(m protocolo.Mensaje, workers int) protocolo.Mensaje {
	if n.pool != nil && (*m.Config != n.cfg || m.TamBloque != n.tam) {
		return fallo(m, "la asignación cambia la configuración del modelo")
	}
	desc, a, err := LeerShard(m.Ruta)
	if err != nil {
		return fallo(m, "no se pudo cargar el shard: %v", err)
	}
	if m.SHA256 != "" && m.SHA256 != desc.SHA256 {
		return fallo(m, "sha256 de %s distinto del asignado", m.Ruta)
	}
	if a.TamBloque != m.TamBloque || a.D != m.D || desc.BloqueInicial != m.BloqueInicial || desc.NumBloques != m.NumBloques {
		return fallo(m, "la cabecera de %s no coincide con la asignación", m.Ruta)
	}
	if n.pool == nil {
		n.cfg, n.tam = *m.Config, m.TamBloque
		n.pool = ml.NuevoPool(nil, ml.Config{TamBloque: m.TamBloque, Lambda: m.Config.Lambda,
			PesoPos: m.Config.PesoPos, PesoNeg: m.Config.PesoNeg}, workers)
	}
	n.shards[m.Shard] = a
	n.Log("shard %d cargado: %d filas, bloques %d..%d", m.Shard, a.N, desc.BloqueInicial, desc.BloqueInicial+desc.NumBloques-1)
	return protocolo.Mensaje{Tipo: protocolo.Listo, Shard: m.Shard, BloqueInicial: desc.BloqueInicial,
		NumBloques: desc.NumBloques, N: a.N, D: a.D}
}

// calcular devuelve un parcial por bloque del shard, sin reducir.
func (n *Nodo) calcular(m protocolo.Mensaje) protocolo.Mensaje {
	a, ok := n.shards[m.Shard]
	if !ok {
		return fallo(m, "shard %d no cargado", m.Shard)
	}
	if len(m.W) != a.D+1 {
		return fallo(m, "llegaron %d pesos, el shard necesita %d", len(m.W), a.D+1)
	}
	cpu0, okCPU := cpuProceso()
	ini := time.Now()
	parciales := n.pool.Parciales(a.Dataset, m.W)
	r := protocolo.Mensaje{Tipo: protocolo.Gradiente, Epoca: m.Epoca, Shard: m.Shard,
		BloqueInicial: a.BloqueInicial, Parciales: parciales, MicrosCalculo: time.Since(ini).Microseconds()}
	if cpu1, ok := cpuProceso(); okCPU && ok {
		r.MicrosCPU = max(cpu1-cpu0, 0) // solo métrica: no entra en el gradiente
	}
	return r
}
