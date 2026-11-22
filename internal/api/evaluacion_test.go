package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"slices"
	"testing"
)

func pedirDoc(t *testing.T, url string, cab map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	// Transport propio sin descompresión automática, para ver la respuesta real.
	res, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, b
}

func TestEvaluacionYRiesgoConElReporteReal(t *testing.T) {
	real, err := os.ReadFile("../../reports/evaluacion.json")
	if err != nil {
		t.Fatal(err)
	}
	_, ts := servidorDePrueba(t, Opciones{Evaluacion: real})

	res, b := pedirDoc(t, ts.URL+"/api/v1/evaluacion", nil)
	if res.StatusCode != 200 || !bytes.Equal(b, real) || res.Header.Get("ETag") == "" {
		t.Fatalf("evaluación: %d, %d bytes (quería %d), ETag %q", res.StatusCode, len(b), len(real), res.Header.Get("ETag"))
	}

	res, b = pedirDoc(t, ts.URL+"/api/v1/riesgo", map[string]string{"Accept-Encoding": "gzip"})
	if res.StatusCode != 200 || res.Header.Get("Content-Encoding") != "gzip" || !slices.Contains(res.Header.Values("Vary"), "Accept-Encoding") {
		t.Fatalf("riesgo: %d %v", res.StatusCode, res.Header)
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	plano, _ := io.ReadAll(zr)
	var sub, completo map[string]json.RawMessage
	if err := json.Unmarshal(plano, &sub); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(real, &completo)
	if len(sub) != 4 || !bytes.Equal(sub["riesgo"], completo["riesgo"]) || !bytes.Equal(sub["prioridades"], completo["prioridades"]) {
		t.Fatalf("el riesgo no es el subconjunto exacto del reporte: claves %d", len(sub))
	}
	if len(b) >= len(plano)/3 {
		t.Errorf("gzip no comprime: %d → %d bytes", len(plano), len(b))
	}

	etagGz := res.Header.Get("ETag")
	if cc := res.Header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Fatalf("Cache-Control %q", cc)
	}
	res, _ = pedirDoc(t, ts.URL+"/api/v1/riesgo", map[string]string{"Accept-Encoding": "gzip", "If-None-Match": etagGz})
	if res.StatusCode != http.StatusNotModified {
		t.Fatalf("con el mismo ETag se esperaba 304, llegó %d", res.StatusCode)
	}
	// La representación sin gzip tiene otro ETag (RFC 9110 §8.8.3): el de gzip no la valida.
	res, _ = pedirDoc(t, ts.URL+"/api/v1/riesgo", map[string]string{"If-None-Match": etagGz})
	if res.StatusCode != 200 || res.Header.Get("ETag") == etagGz {
		t.Fatalf("sin gzip: %d con ETag %q (el de gzip es %q)", res.StatusCode, res.Header.Get("ETag"), etagGz)
	}
}

// PR-82 H3: el handler no borra el Vary: Origin que pone el middleware de CORS
// (el real desde #61, PR-83 H4).
func TestEvaluacionConservaVaryDeCORS(t *testing.T) {
	real, _ := os.ReadFile("../../reports/evaluacion.json")
	_, ts := servidorDePrueba(t, Opciones{Evaluacion: real, OrigenesCORS: []string{"http://localhost:8081"}})
	res, _ := pedirDoc(t, ts.URL+"/api/v1/riesgo", map[string]string{"Origin": "http://localhost:8081"})
	vary := res.Header.Values("Vary")
	if !slices.Contains(vary, "Origin") || !slices.Contains(vary, "Accept-Encoding") || res.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("Vary = %v, ACAO %q; quería Origin y Accept-Encoding", vary, res.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestEvaluacionSinReporteDa503(t *testing.T) {
	_, ts := servidorDePrueba(t, Opciones{})
	for _, ruta := range []string{"/api/v1/evaluacion", "/api/v1/riesgo"} {
		if res, b := pedirDoc(t, ts.URL+ruta, nil); res.StatusCode != 503 {
			t.Fatalf("%s: %d %s", ruta, res.StatusCode, b)
		}
	}
}

func TestEvaluacionInvalidaNoArranca(t *testing.T) {
	for _, b := range []string{`{`, `{"version_esquema":2,"version_modelo":"x","riesgo":{},"prioridades":[]}`,
		`{"version_esquema":1,"version_modelo":"x","riesgo":{}}`,
		`{"version_esquema":1,"version_modelo":null,"riesgo":null,"prioridades":null}`,
		`{"version_esquema":1,"version_modelo":"","riesgo":{},"prioridades":[]}`,
		`{"version_esquema":1,"version_modelo":"x","riesgo":[],"prioridades":[]}`,
		`{"version_esquema":1,"version_modelo":"x","riesgo":{},"prioridades":{}}`} {
		if _, _, err := cargarEvaluacion([]byte(b)); err == nil {
			t.Errorf("%s: debía fallar", b)
		}
	}
}

func TestAceptaGzipYETag(t *testing.T) {
	for cab, quiere := range map[string]bool{"gzip": true, "br, gzip;q=0.8": true, "GZIP": true, "gzip;q=0": false,
		"gzip;q=0.0": false, "gzip;q=0.000": false, "gzip; Q=0": false, "br": false, "": false, "*": true,
		"*;q=0": false, "gzip;q=0, *": false, "deflate, x-gzip": true} {
		if aceptaGzip(cab) != quiere {
			t.Errorf("aceptaGzip(%q) != %v", cab, quiere)
		}
	}
	if !coincideETag(`"a", W/"b"`, `"b"`) || coincideETag(`"a"`, `"b"`) || !coincideETag("*", `"x"`) {
		t.Error("coincideETag")
	}
}

func TestDocumentoNoCompartePlano(t *testing.T) {
	b := []byte(`{"a":1}`)
	d, err := nuevoDocumento(b)
	if err != nil {
		t.Fatal(err)
	}
	b[2] = 'X'
	if string(d.plano) != `{"a":1}` {
		t.Fatalf("el documento cambió con el slice del llamador: %s", d.plano)
	}
}
