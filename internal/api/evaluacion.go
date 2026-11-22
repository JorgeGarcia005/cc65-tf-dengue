package api

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// VersionEvaluacion es la versión de reports/evaluacion.json que sabe servir la
// API (internal/evaluacion.VersionEsquema).
const VersionEvaluacion = 1

// documento es un JSON fijo listo para servir: plano, comprimido y con ETag.
// Se arma una vez al arrancar; servirlo no asigna memoria.
type documento struct {
	plano, gz []byte
	etag      string
}

func nuevoDocumento(b []byte) (*documento, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	return &documento{plano: b, gz: buf.Bytes(), etag: `"` + hex.EncodeToString(h[:8]) + `"`}, nil
}

// cargarEvaluacion valida reports/evaluacion.json y arma los dos documentos:
// el completo (GET /api/v1/evaluacion) y el del mapa, solo con el riesgo
// cruzado y las prioridades (GET /api/v1/riesgo). Los números pasan como
// json.RawMessage: no se re-redondea ninguna cifra.
func cargarEvaluacion(b []byte) (completo, riesgo *documento, err error) {
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(b, &campos); err != nil {
		return nil, nil, fmt.Errorf("evaluación: JSON inválido: %w", err)
	}
	var version int
	if err := json.Unmarshal(campos["version_esquema"], &version); err != nil || version != VersionEvaluacion {
		return nil, nil, fmt.Errorf("evaluación: version_esquema %s, se esperaba %d", campos["version_esquema"], VersionEvaluacion)
	}
	for _, k := range []string{"version_modelo", "riesgo", "prioridades"} {
		if len(campos[k]) == 0 {
			return nil, nil, errors.New("evaluación: falta " + k)
		}
	}
	sub, err := json.Marshal(map[string]json.RawMessage{
		"version_esquema": campos["version_esquema"], "version_modelo": campos["version_modelo"],
		"riesgo": campos["riesgo"], "prioridades": campos["prioridades"],
	})
	if err != nil {
		return nil, nil, err
	}
	if completo, err = nuevoDocumento(b); err != nil {
		return nil, nil, err
	}
	riesgo, err = nuevoDocumento(sub)
	return completo, riesgo, err
}

// servirDocumento responde con ETag (304 si no cambió) y gzip si el cliente lo acepta.
func servirDocumento(w http.ResponseWriter, r *http.Request, d *documento) {
	if d == nil {
		responderError(w, http.StatusServiceUnavailable, "evaluación no disponible: genere reports/evaluacion.json con go run ./cmd/evaluacion")
		return
	}
	cab := w.Header()
	cab.Set("ETag", d.etag)
	cab.Set("Cache-Control", "no-cache") // revalida con If-None-Match en cada visita
	cab.Set("Vary", "Accept-Encoding")
	if coincideETag(r.Header.Get("If-None-Match"), d.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	cab.Set("Content-Type", "application/json; charset=utf-8")
	cuerpo := d.plano
	if aceptaGzip(r.Header.Get("Accept-Encoding")) {
		cab.Set("Content-Encoding", "gzip")
		cuerpo = d.gz
	}
	cab.Set("Content-Length", fmt.Sprint(len(cuerpo)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(cuerpo)
	}
}

func coincideETag(cabecera, etag string) bool {
	for _, e := range strings.Split(cabecera, ",") {
		if e = strings.TrimSpace(e); e == etag || e == "W/"+etag || e == "*" {
			return true
		}
	}
	return false
}

func aceptaGzip(cabecera string) bool {
	for _, c := range strings.Split(cabecera, ",") {
		nombre, q, _ := strings.Cut(strings.TrimSpace(c), ";")
		if strings.EqualFold(strings.TrimSpace(nombre), "gzip") && strings.ReplaceAll(strings.TrimSpace(q), " ", "") != "q=0" {
			return true
		}
	}
	return false
}

func (s *Servidor) verEvaluacion(w http.ResponseWriter, r *http.Request) {
	servirDocumento(w, r, s.evaluacion)
}

func (s *Servidor) verRiesgo(w http.ResponseWriter, r *http.Request) {
	servirDocumento(w, r, s.riesgo)
}
