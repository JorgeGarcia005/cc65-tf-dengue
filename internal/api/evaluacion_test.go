package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
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
	if res.StatusCode != 200 || res.Header.Get("Content-Encoding") != "gzip" || res.Header.Get("Vary") != "Accept-Encoding" {
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

	res, _ = pedirDoc(t, ts.URL+"/api/v1/riesgo", map[string]string{"If-None-Match": res.Header.Get("ETag")})
	if res.StatusCode != http.StatusNotModified {
		t.Fatalf("con el mismo ETag se esperaba 304, llegó %d", res.StatusCode)
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
		`{"version_esquema":1,"version_modelo":"x","riesgo":{}}`} {
		if _, _, err := cargarEvaluacion([]byte(b)); err == nil {
			t.Errorf("%s: debía fallar", b)
		}
	}
}

func TestAceptaGzipYETag(t *testing.T) {
	for cab, quiere := range map[string]bool{"gzip": true, "br, gzip;q=0.8": true, "GZIP": true, "gzip;q=0": false, "br": false, "": false} {
		if aceptaGzip(cab) != quiere {
			t.Errorf("aceptaGzip(%q) != %v", cab, quiere)
		}
	}
	if !coincideETag(`"a", W/"b"`, `"b"`) || coincideETag(`"a"`, `"b"`) || !coincideETag("*", `"x"`) {
		t.Error("coincideETag")
	}
}
