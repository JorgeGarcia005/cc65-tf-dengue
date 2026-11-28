package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

// regionProtocolo genera la región «protocolo» del informe: un mensaje de
// ejemplo de cada tipo, construido con los tipos reales de
// internal/cluster/protocolo, validado con Mensaje.Validar y serializado con
// encoding/json, igual que en la red. Para que quepan en la página usa un
// modelo de juguete de 3 variables y un shard de 2 bloques de 4 filas; los
// parciales del mensaje «gradiente» los calcula ml.ParcialBloque sobre esos
// datos con los pesos de la época 0.
func regionProtocolo() (string, error) {
	const d, tamBloque = 3, 4
	x := []float64{
		0.5, 1, 0, -1.2, 0, 1, 0.3, 1, 1, 2.1, 0, 0,
		-0.4, 1, 0, 0.9, 0, 1, 1.5, 1, 0, -0.8, 0, 1,
	}
	y := []float64{1, 0, 0, 1, 0, 0, 1, 0}
	ds, err := ml.NuevoDataset(x, y, d)
	if err != nil {
		return "", err
	}
	pos, neg := ml.PesosBalanceados(y)
	cfg := ml.Config{TamBloque: tamBloque, Lambda: 1e-4, PesoPos: pos, PesoNeg: neg}
	w := make([]float64, d+1) // época 0: pesos en cero
	parciales := make([]ml.Parcial, ml.NumBloques(ds.N, tamBloque))
	for k := range parciales {
		parciales[k] = ml.NuevoParcial(d)
		ml.ParcialBloque(ds, cfg, w, k, &parciales[k])
	}
	us := int64(41)
	// Huella de los datos de juguete (en el clúster es la del archivo del shard).
	crudo := make([]byte, 8*len(x))
	for i, v := range x {
		binary.LittleEndian.PutUint64(crudo[8*i:], math.Float64bits(v))
	}
	suma := sha256.Sum256(crudo)
	huella := hex.EncodeToString(suma[:])
	mensajes := []struct {
		sentido string
		m       protocolo.Mensaje
	}{
		{"nodo → coordinador", protocolo.Mensaje{Tipo: protocolo.Registro, Nodo: "node-2", Version: protocolo.Version, Workers: 16}},
		{"coordinador → nodo", protocolo.Mensaje{Tipo: protocolo.Asignacion, Shard: 1, Ruta: "/datos/shard_1.bin", SHA256: huella,
			BloqueInicial: 2, NumBloques: len(parciales), TamBloque: tamBloque, D: d,
			Config: &protocolo.ConfigML{Lambda: cfg.Lambda, PesoPos: pos, PesoNeg: neg}}},
		{"nodo → coordinador", protocolo.Mensaje{Tipo: protocolo.Listo, Shard: 1, BloqueInicial: 2, NumBloques: len(parciales), N: ds.N, D: d}},
		{"coordinador → nodo", protocolo.Mensaje{Tipo: protocolo.Epoca, Epoca: 0, Shard: 1, W: w}},
		{"nodo → coordinador", protocolo.Mensaje{Tipo: protocolo.Gradiente, Epoca: 0, Shard: 1, BloqueInicial: 2, Parciales: parciales,
			MicrosCalculo: 52, MicrosCPU: &us}},
		{"ambos", protocolo.Mensaje{Tipo: protocolo.Error, Epoca: 7, Shard: 1, Detalle: fmt.Sprintf("llegaron %d pesos, el shard necesita %d", d, d+1)}},
		{"coordinador → nodo", protocolo.Mensaje{Tipo: protocolo.Fin, Motivo: protocolo.MotivoCompleto}},
	}
	var b strings.Builder
	for _, e := range mensajes {
		if err := e.m.Validar(); err != nil {
			return "", fmt.Errorf("mensaje de ejemplo inválido: %w", err)
		}
		j, err := json.Marshal(e.m)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "// %s (%s)\n%s\n", e.m.Tipo, e.sentido, j)
	}
	return "```json\n" + b.String() + "```\n", nil
}

// equilibrioCluster es una fila de la estimación del punto de equilibrio del
// clúster (sección 11.4).
type equilibrioCluster struct {
	Nodos        int
	AhorroMs     float64 // cómputo que se ahorra por época con el conjunto completo
	ComunicaMs   float64 // red y espera medidas por época con el conjunto completo
	FilasMinimas int     // filas desde las que el ahorro supera la comunicación
}

// EquilibrioCluster estima, para cada número de nodos p > 1 del benchmark del
// clúster, desde cuántas filas conviene distribuir. Modelo lineal: el cómputo
// por época es c·n (c = cómputo de 1 nodo / filas de entrenamiento) y repartir
// en p nodos ahorra c·n·(1 − 1/p); conviene cuando ese ahorro supera la red y
// espera medida con p nodos, r_p, es decir n > r_p / (c·(1 − 1/p)). Como r_p se
// midió con el conjunto completo (con menos filas viajan menos bloques), la
// cota es conservadora: el equilibrio real está en esas filas o en menos.
func (d datosTB2) EquilibrioCluster() []equilibrioCluster {
	ms := d.Cluster.Mediciones
	if len(ms) == 0 || ms[0].Nodos != 1 || d.S.M.S.Train.Filas == 0 {
		return nil
	}
	c := ms[0].Desglose.CalculoMax / float64(d.S.M.S.Train.Filas) // s por fila
	var out []equilibrioCluster
	for _, m := range ms[1:] {
		p := float64(m.Nodos)
		ahorro := c * float64(d.S.M.S.Train.Filas) * (1 - 1/p)
		out = append(out, equilibrioCluster{
			Nodos:        m.Nodos,
			AhorroMs:     1000 * ahorro,
			ComunicaMs:   1000 * m.Desglose.Red,
			FilasMinimas: int(math.Ceil(m.Desglose.Red / (c * (1 - 1/p)))),
		})
	}
	return out
}
