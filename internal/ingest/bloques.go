// Package ingest lee el CSV de vigilancia de dengue del MINSA en paralelo.
//
// El archivo se carga en memoria (≈118 MB) y se parte en bloques de bytes de
// tamaño objetivo fijo, alineados a fin de línea. Los límites de cada bloque
// dependen solo del contenido y del tamaño objetivo, nunca del número de
// workers, así que cualquier cantidad de workers procesa exactamente los mismos
// bloques. Cada bloque produce su resultado en un slot propio y los resultados
// se combinan en orden de bloque: la salida es idéntica a la secuencial.
package ingest

import (
	"bytes"
	"sync"
)

// TamBloqueBytesPorDefecto es el tamaño objetivo de cada bloque (1 MiB ≈ 9 000 filas).
const TamBloqueBytesPorDefecto = 1 << 20

// Rango es un bloque [Ini, Fin) del cuerpo del CSV; siempre termina después de
// un '\n' o al final de los datos.
type Rango struct{ Ini, Fin int }

// SepararEncabezado devuelve la primera línea (sin '\r\n') y el resto.
func SepararEncabezado(datos []byte) (encabezado, cuerpo []byte) {
	i := bytes.IndexByte(datos, '\n')
	if i < 0 {
		return bytes.TrimRight(datos, "\r"), nil
	}
	return bytes.TrimRight(datos[:i], "\r"), datos[i+1:]
}

// PartirEnBloques corta datos en bloques de ~tam bytes que terminan en '\n'.
// Ninguna línea queda partida entre dos bloques.
func PartirEnBloques(datos []byte, tam int) []Rango {
	if tam <= 0 {
		tam = TamBloqueBytesPorDefecto
	}
	var rangos []Rango
	for ini := 0; ini < len(datos); {
		fin := ini + tam
		if fin >= len(datos) {
			fin = len(datos)
		} else if j := bytes.IndexByte(datos[fin:], '\n'); j >= 0 {
			fin += j + 1
		} else {
			fin = len(datos)
		}
		rangos = append(rangos, Rango{ini, fin})
		ini = fin
	}
	return rangos
}

// MapearEnOrden aplica f a cada bloque con un worker pool y devuelve los
// resultados en orden de bloque.
//
// Sincronización: un canal reparte índices de bloque; cada worker escribe solo
// en res[k] (slot propio, sin mutex); WaitGroup marca el fin. El llamador
// recibe res cuando todos terminaron, así que lo puede leer sin carreras.
func MapearEnOrden[T any](datos []byte, rangos []Rango, workers int, f func(k int, bloque []byte) T) []T {
	res := make([]T, len(rangos))
	workers = max(1, min(workers, len(rangos)))
	if workers == 1 {
		for k, r := range rangos {
			res[k] = f(k, datos[r.Ini:r.Fin])
		}
		return res
	}
	tareas := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for k := range tareas {
				r := rangos[k]
				res[k] = f(k, datos[r.Ini:r.Fin])
			}
		}()
	}
	for k := range rangos {
		tareas <- k
	}
	close(tareas)
	wg.Wait()
	return res
}

// ParaCadaLinea llama f con cada línea del bloque, sin '\r\n'. Omite líneas vacías.
func ParaCadaLinea(bloque []byte, f func(linea []byte)) {
	for len(bloque) > 0 {
		i := bytes.IndexByte(bloque, '\n')
		var linea []byte
		if i < 0 {
			linea, bloque = bloque, nil
		} else {
			linea, bloque = bloque[:i], bloque[i+1:]
		}
		linea = bytes.TrimRight(linea, "\r")
		if len(linea) > 0 {
			f(linea)
		}
	}
}
