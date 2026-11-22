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
	"slices"
	"strconv"
	"strings"
)

// VersionEvaluacion es la versión de reports/evaluacion.json que sabe servir la
// API (internal/evaluacion.VersionEsquema).
const VersionEvaluacion = 1

// documento es un JSON fijo listo para servir: plano, comprimido y con un ETag
// por representación (RFC 9110 §8.8.3: la versión gzip no es byte a byte la
// misma, así que no comparte ETag fuerte). Se arma una vez al arrancar.
type documento struct {
	plano, gz      []byte
	etag, etagGzip string
}

func nuevoDocumento(b []byte) (*documento, error) {
	b = slices.Clone(b) // no depende de que el llamador no modifique su slice
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
	huella := hex.EncodeToString(h[:8])
	return &documento{plano: b, gz: buf.Bytes(), etag: `"` + huella + `"`, etagGzip: `"` + huella + `-gzip"`}, nil
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
	var (
		modelo      string
		riesgo_     map[string]json.RawMessage
		prioridades []json.RawMessage
	)
	switch {
	case json.Unmarshal(campos["version_modelo"], &modelo) != nil || modelo == "":
		return nil, nil, errors.New("evaluación: version_modelo debe ser un texto no vacío")
	case json.Unmarshal(campos["riesgo"], &riesgo_) != nil || riesgo_ == nil:
		return nil, nil, errors.New("evaluación: riesgo debe ser un objeto")
	case json.Unmarshal(campos["prioridades"], &prioridades) != nil || prioridades == nil:
		return nil, nil, errors.New("evaluación: prioridades debe ser un arreglo")
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

// servirDocumento negocia gzip, responde con el ETag de esa representación
// (304 si el cliente ya la tiene) y deja que el navegador revalide siempre.
func servirDocumento(w http.ResponseWriter, r *http.Request, d *documento) {
	if d == nil {
		responderError(w, http.StatusServiceUnavailable, "evaluación no disponible: genere reports/evaluacion.json con go run ./cmd/evaluacion")
		return
	}
	cab := w.Header()
	// Add, no Set: el middleware de CORS (ADR-0007) ya pudo poner Vary: Origin.
	cab.Add("Vary", "Accept-Encoding")
	// private: es una ruta autenticada, no la deben guardar cachés compartidas;
	// no-cache: el navegador sí la guarda, pero revalida con If-None-Match.
	cab.Set("Cache-Control", "private, no-cache")
	cuerpo, etag := d.plano, d.etag
	gz := aceptaGzip(strings.Join(r.Header.Values("Accept-Encoding"), ","))
	if gz {
		cuerpo, etag = d.gz, d.etagGzip
	}
	cab.Set("ETag", etag)
	if coincideETag(strings.Join(r.Header.Values("If-None-Match"), ","), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	cab.Set("Content-Type", "application/json; charset=utf-8")
	if gz {
		cab.Set("Content-Encoding", "gzip")
	}
	cab.Set("Content-Length", strconv.Itoa(len(cuerpo)))
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

// aceptaGzip lee Accept-Encoding (RFC 9110 §12.5.3): gzip con q > 0, o * con
// q > 0 si gzip no aparece explícito.
func aceptaGzip(cabecera string) bool {
	comodin := false
	for _, c := range strings.Split(cabecera, ",") {
		partes := strings.Split(c, ";")
		nombre := strings.ToLower(strings.TrimSpace(partes[0]))
		q := 1.0
		for _, p := range partes[1:] {
			k, v, _ := strings.Cut(strings.TrimSpace(p), "=")
			if strings.EqualFold(strings.TrimSpace(k), "q") {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					q = f
				}
			}
		}
		switch nombre {
		case "gzip", "x-gzip":
			return q > 0
		case "*":
			comodin = q > 0
		}
	}
	return comodin
}

func (s *Servidor) verEvaluacion(w http.ResponseWriter, r *http.Request) {
	servirDocumento(w, r, s.evaluacion)
}

func (s *Servidor) verRiesgo(w http.ResponseWriter, r *http.Request) {
	servirDocumento(w, r, s.riesgo)
}
