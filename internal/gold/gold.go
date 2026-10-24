// Package gold guarda y lee la matriz de entrenamiento en un formato binario
// simple, y la parte en shards por bloques contiguos para los nodos del clúster.
//
// Formato (little endian):
//
//	"CC65GOLD"        8 bytes
//	version uint32    = 1
//	d       uint32    variables por fila (sin sesgo)
//	n       uint64    filas
//	tam     uint32    filas por bloque (TamBloque de ml.Config)
//	bloque0 uint64    índice global del primer bloque de este archivo
//	x       n·d float64
//	y       n   float64
//
// bloque0 permite que cada nodo sepa qué bloques globales tiene, para que el
// coordinador reduzca los gradientes en el orden global de bloque.
package gold

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

const (
	magia   = "CC65GOLD"
	version = 1
	tamCab  = 8 + 4 + 4 + 8 + 4 + 8
)

// Archivo es un dataset gold con su ubicación en la numeración global de bloques.
type Archivo struct {
	*ml.Dataset
	TamBloque     int
	BloqueInicial int
}

// Escribir guarda el archivo en ruta.
func Escribir(ruta string, a Archivo) error {
	f, err := os.Create(ruta)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(f, 1<<20)
	cab := make([]byte, 0, tamCab)
	cab = append(cab, magia...)
	cab = binary.LittleEndian.AppendUint32(cab, version)
	cab = binary.LittleEndian.AppendUint32(cab, uint32(a.D))
	cab = binary.LittleEndian.AppendUint64(cab, uint64(a.N))
	cab = binary.LittleEndian.AppendUint32(cab, uint32(a.TamBloque))
	cab = binary.LittleEndian.AppendUint64(cab, uint64(a.BloqueInicial))
	if _, err := w.Write(cab); err != nil {
		f.Close()
		return err
	}
	var b [8]byte
	for _, s := range [][]float64{a.X, a.Y} {
		for _, v := range s {
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
			if _, err := w.Write(b[:]); err != nil {
				f.Close()
				return err
			}
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Leer carga un archivo gold completo y valida su cabecera y tamaño.
func Leer(ruta string) (Archivo, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return Archivo{}, err
	}
	return Decodificar(b)
}

// Decodificar interpreta los bytes de un archivo gold.
func Decodificar(b []byte) (Archivo, error) {
	if len(b) < tamCab || string(b[:8]) != magia {
		return Archivo{}, errors.New("gold: cabecera inválida")
	}
	le := binary.LittleEndian
	if v := le.Uint32(b[8:]); v != version {
		return Archivo{}, fmt.Errorf("gold: versión %d no soportada", v)
	}
	d := int(le.Uint32(b[12:]))
	n := int(le.Uint64(b[16:]))
	tam := int(le.Uint32(b[24:]))
	b0 := int(le.Uint64(b[28:]))
	if want := tamCab + 8*(n*d+n); len(b) != want {
		return Archivo{}, fmt.Errorf("gold: %d bytes, se esperaban %d", len(b), want)
	}
	cuerpo := b[tamCab:]
	x := make([]float64, n*d)
	for i := range x {
		x[i] = math.Float64frombits(le.Uint64(cuerpo[8*i:]))
	}
	cuerpo = cuerpo[8*n*d:]
	y := make([]float64, n)
	for i := range y {
		y[i] = math.Float64frombits(le.Uint64(cuerpo[8*i:]))
	}
	ds, err := ml.NuevoDataset(x, y, d)
	if err != nil {
		return Archivo{}, err
	}
	return Archivo{Dataset: ds, TamBloque: tam, BloqueInicial: b0}, nil
}

// RangoShards reparte numBloques bloques en n shards contiguos lo más parejos
// posible y devuelve [ini, fin) de bloques del shard s.
func RangoShards(numBloques, n, s int) (ini, fin int) {
	return s * numBloques / n, (s + 1) * numBloques / n
}

// Particionar corta el archivo en n shards por bloques contiguos completos.
// Concatenar los shards en orden reproduce el archivo original.
func Particionar(a Archivo, n int) ([]Archivo, error) {
	if a.BloqueInicial != 0 {
		return nil, errors.New("gold: solo se particiona un archivo completo (bloque inicial 0)")
	}
	b := ml.NumBloques(a.N, a.TamBloque)
	if n < 1 || n > b {
		return nil, fmt.Errorf("gold: %d shards para %d bloques", n, b)
	}
	shards := make([]Archivo, n)
	for s := range n {
		bi, bf := RangoShards(b, n, s)
		fi, _ := ml.LimitesBloque(bi, a.N, a.TamBloque)
		_, ff := ml.LimitesBloque(bf-1, a.N, a.TamBloque)
		shards[s] = Archivo{
			Dataset:       &ml.Dataset{X: a.X[fi*a.D : ff*a.D], Y: a.Y[fi:ff], N: ff - fi, D: a.D},
			TamBloque:     a.TamBloque,
			BloqueInicial: bi,
		}
	}
	return shards, nil
}
