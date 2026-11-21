// Package cluster implementa el entrenamiento distribuido: un coordinador
// (parameter server síncrono) y N nodos de cómputo en estrella, unidos por TCP
// con el protocolo de internal/cluster/protocolo (ADR-0005).
//
// Una época:
//
//  1. El coordinador envía `epoca{e, shard, w}` al dueño de cada shard.
//  2. Cada nodo calcula, con su worker pool, un ml.Parcial por bloque del shard
//     y responde `gradiente` sin reducirlos.
//  3. El coordinador coloca cada parcial en parciales[bloque global]. Cuando
//     tiene los de todos los shards (barrera), los reduce en orden de índice
//     de bloque y actualiza los pesos.
//
// Como la reducción solo depende del índice global de bloque, no importa qué
// nodo calculó cada shard ni en qué orden llegaron las respuestas: los pesos
// son idénticos bit a bit a los del entrenamiento secuencial, con cualquier
// número de nodos e incluso si un nodo cae y otro recalcula su shard.
package cluster

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// Errores con nombre, para distinguirlos con errors.Is.
var (
	// ErrSinNodos: no queda ningún nodo sano que pueda calcular un shard.
	ErrSinNodos = errors.New("cluster: no quedan nodos sanos")
	// ErrRegistro: no se registraron a tiempo los nodos pedidos.
	ErrRegistro = errors.New("cluster: registro incompleto")
	// ErrAbortado: el coordinador avisó con `fin{abortado}`.
	ErrAbortado = errors.New("cluster: el coordinador abortó el entrenamiento")
	// ErrCerrado: se usó un coordinador ya cerrado.
	ErrCerrado = errors.New("cluster: coordinador cerrado")
)

// Shard describe un archivo gold de bloques contiguos, tal como lo ve el
// coordinador: no carga los datos, solo sabe qué bloques globales contiene.
type Shard struct {
	Ruta          string // ruta del archivo en el volumen compartido de los nodos
	SHA256        string // huella del archivo; vacía = el nodo no la verifica
	BloqueInicial int    // índice global de su primer bloque
	NumBloques    int
	N             int // filas
}

// LeerShard carga un archivo gold y devuelve su descripción con la huella
// sha256 de sus bytes. Lo usan el nodo (para cargar y verificar) y quien arma
// la lista de shards del coordinador.
func LeerShard(ruta string) (Shard, gold.Archivo, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return Shard{}, gold.Archivo{}, err
	}
	a, err := gold.Decodificar(b)
	if err != nil {
		return Shard{}, gold.Archivo{}, fmt.Errorf("%s: %w", ruta, err)
	}
	huella := sha256.Sum256(b)
	return Shard{
		Ruta:          ruta,
		SHA256:        hex.EncodeToString(huella[:]),
		BloqueInicial: a.BloqueInicial,
		NumBloques:    ml.NumBloques(a.N, a.TamBloque),
		N:             a.N,
	}, a, nil
}
