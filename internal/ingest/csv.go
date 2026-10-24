package ingest

import (
	"bytes"
	"errors"
	"fmt"
)

// Columnas del CSV original, en orden.
var Columnas = []string{
	"departamento", "provincia", "distrito", "localidad", "enfermedad", "ano", "semana",
	"diagnostic", "tipo_dx", "diresa", "ubigeo", "localcod", "edad", "tipo_edad", "sexo",
}

// Índices de columna.
const (
	ColDepartamento = iota
	ColProvincia
	ColDistrito
	ColLocalidad
	ColEnfermedad
	ColAno
	ColSemana
	ColDiagnostico
	ColTipoDx
	ColDiresa
	ColUbigeo
	ColLocalcod
	ColEdad
	ColTipoEdad
	ColSexo
	NumColumnas
)

// Errores de parseo de una línea.
var (
	ErrComillas = errors.New("comillas desbalanceadas")
	ErrColumnas = errors.New("número de columnas inesperado")
)

// ValidarEncabezado comprueba que el CSV tenga exactamente las columnas esperadas.
func ValidarEncabezado(encabezado []byte) error {
	campos, err := ParsearLinea(encabezado, nil)
	if err != nil {
		return fmt.Errorf("ingest: encabezado: %w", err)
	}
	for i, c := range campos {
		if string(bytes.ToLower(bytes.TrimSpace(c))) != Columnas[i] {
			return fmt.Errorf("ingest: columna %d es %q, se esperaba %q", i, c, Columnas[i])
		}
	}
	return nil
}

// ParsearLinea separa una línea del CSV original en sus NumColumnas campos.
func ParsearLinea(linea []byte, buf [][]byte) ([][]byte, error) {
	return ParsearLineaN(linea, buf, NumColumnas)
}

// ParsearLineaTolerante intenta primero el parseo estricto (RFC 4180) y, si
// falla por comillas, reintenta tratando como literal toda comilla dentro de un
// campo entrecomillado que no esté seguida de coma ni de fin de línea. Así se
// recuperan filas como `"ISLA SANTA ROSA "AMAZONAS""`, que el portal publica
// sin escapar. reparada indica que hizo falta el segundo intento.
func ParsearLineaTolerante(linea []byte, buf [][]byte, n int) (campos [][]byte, reparada bool, err error) {
	campos, err = ParsearLineaN(linea, buf, n)
	if err != ErrComillas {
		return campos, false, err
	}
	campos, err = parsearLaxo(linea, buf, n)
	return campos, err == nil, err
}

// parsearLaxo: en un campo que empieza con comillas, solo la comilla seguida de
// ',' o de fin de línea cierra el campo; las demás se toman como texto.
func parsearLaxo(linea []byte, buf [][]byte, n int) ([][]byte, error) {
	campos := buf[:0]
	i := 0
	for {
		if i < len(linea) && linea[i] == '"' {
			j := i + 1
			for {
				k := bytes.IndexByte(linea[j:], '"')
				if k < 0 {
					return campos, ErrComillas
				}
				j += k
				if j+1 == len(linea) || linea[j+1] == ',' {
					break
				}
				j++
			}
			campos = append(campos, linea[i+1:j])
			i = j + 1
		} else {
			k := bytes.IndexByte(linea[i:], ',')
			fin := len(linea)
			if k >= 0 {
				fin = i + k
			}
			campos = append(campos, linea[i:fin])
			i = fin
		}
		if i >= len(linea) {
			break
		}
		i++
		if i == len(linea) {
			campos = append(campos, linea[i:])
			break
		}
	}
	if len(campos) != n {
		return campos, ErrColumnas
	}
	return campos, nil
}

// ParsearLineaN separa una línea CSV (RFC 4180, sin saltos de línea dentro de
// campos) en exactamente n campos. Reutiliza buf si tiene capacidad.
// Los campos sin comillas son subslices de linea (sin copiar); los que llevan
// "" escapado se copian una vez.
func ParsearLineaN(linea []byte, buf [][]byte, n int) ([][]byte, error) {
	campos := buf[:0]
	i := 0
	for {
		if i < len(linea) && linea[i] == '"' {
			// Campo entre comillas.
			j := i + 1
			escapado := false
			for {
				k := bytes.IndexByte(linea[j:], '"')
				if k < 0 {
					return campos, ErrComillas
				}
				j += k
				if j+1 < len(linea) && linea[j+1] == '"' {
					escapado = true
					j += 2
					continue
				}
				break
			}
			campo := linea[i+1 : j]
			if escapado {
				campo = bytes.ReplaceAll(campo, []byte(`""`), []byte(`"`))
			}
			campos = append(campos, campo)
			i = j + 1
			if i < len(linea) && linea[i] != ',' {
				return campos, ErrComillas
			}
		} else {
			k := bytes.IndexByte(linea[i:], ',')
			fin := len(linea)
			if k >= 0 {
				fin = i + k
			}
			campo := linea[i:fin]
			if bytes.IndexByte(campo, '"') >= 0 {
				return campos, ErrComillas
			}
			campos = append(campos, campo)
			i = fin
		}
		if i >= len(linea) {
			break
		}
		i++ // salta la coma
		if i == len(linea) {
			campos = append(campos, linea[i:]) // campo final vacío
			break
		}
	}
	if len(campos) != n {
		return campos, ErrColumnas
	}
	return campos, nil
}
