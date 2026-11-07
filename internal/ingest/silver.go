package ingest

import (
	"bytes"
	"fmt"
	"strconv"
)

// NumColumnasSilver es el número de columnas del CSV limpio.
const NumColumnasSilver = 10

// ParsearSilver convierte una línea del CSV silver en un Registro.
func ParsearSilver(linea []byte, buf [][]byte) (Registro, error) {
	c, err := ParsearLineaN(linea, buf, NumColumnasSilver)
	if err != nil {
		return Registro{}, err
	}
	ano, err1 := strconv.Atoi(string(c[4]))
	semana, err2 := strconv.Atoi(string(c[5]))
	edad, err3 := strconv.ParseFloat(string(c[6]), 64)
	clase, err4 := strconv.Atoi(string(c[9]))
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || len(c[7]) != 1 || len(c[8]) != 1 {
		return Registro{}, fmt.Errorf("ingest: línea silver inválida: %q", linea)
	}
	return Registro{
		Departamento: string(c[0]), Provincia: string(c[1]), Distrito: string(c[2]),
		Ubigeo: string(c[3]), Ano: ano, Semana: semana, EdadAnios: edad,
		Sexo: c[7][0], TipoDx: c[8][0], Clase: clase,
	}, nil
}

// LeerSilverConcurrente parsea el CSV silver completo (con encabezado) en
// paralelo y devuelve los registros en el orden del archivo.
func LeerSilverConcurrente(datos []byte, tamBloque, workers int) ([]Registro, error) {
	enc, cuerpo := SepararEncabezado(datos)
	if string(enc)+"\n" != EncabezadoSilver {
		return nil, fmt.Errorf("ingest: encabezado silver inesperado: %q", enc)
	}
	type parte struct {
		regs []Registro
		err  error
	}
	partes := MapearEnOrden(cuerpo, PartirEnBloques(cuerpo, tamBloque), workers, func(_ int, b []byte) parte {
		var p parte
		p.regs = make([]Registro, 0, bytes.Count(b, []byte{'\n'}))
		buf := make([][]byte, 0, NumColumnasSilver)
		ParaCadaLinea(b, func(l []byte) {
			if p.err != nil {
				return
			}
			r, err := ParsearSilver(l, buf)
			if err != nil {
				p.err = err
				return
			}
			p.regs = append(p.regs, r)
		})
		return p
	})
	total := 0
	for _, p := range partes {
		if p.err != nil {
			return nil, p.err
		}
		total += len(p.regs)
	}
	regs := make([]Registro, 0, total)
	for _, p := range partes {
		regs = append(regs, p.regs...)
	}
	return regs, nil
}
