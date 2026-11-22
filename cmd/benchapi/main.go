// Command benchapi mide la latencia de POST /api/v1/predicciones con carga
// concurrente (issue #30, AGENTS.md §3.5) y escribe
// reports/bench_api_<maquina>.json con p50/p95/p99, RPS y errores de dos
// corridas:
//
//   - fria: cada consulta es distinta, así que ninguna está en la caché
//     (Redis): mide el cálculo del modelo más la escritura en la caché;
//   - caliente: un conjunto pequeño de consultas ya calentadas, que salen de
//     la caché (X-Cache: HIT).
//
// La latencia se mide en el cliente: desde que se envía la petición hasta leer
// la respuesta completa, igual que la ve Postman.
//
// Las rutas exigen JWT desde #61 (ADR-0007): benchapi inicia sesión con
// -correo y la clave de $BENCH_CLAVE (nunca como flag, para que no quede en el
// historial), o usa un token ya emitido (-token / $BENCH_TOKEN). El login no
// se mide; con token el reporte por defecto es bench_api_<maquina>_jwt.json.
//
// Uso, con la API levantada (scripts/evidencia-api.sh o el Compose en modo
// servicio):
//
//	BENCH_CLAVE=… go run ./cmd/benchapi -url http://localhost:8080 -correo analista@vigia.pe -exigir-p95 100ms
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina"
)

// opciones de una medición.
type opciones struct {
	url          string
	peticiones   int // por corrida
	concurrencia int
	calientes    int // consultas distintas de la corrida caliente
	semilla      uint64
	timeout      time.Duration
	token        string // JWT de las peticiones; vacío = sin Authorization
	correo       string // si no es vacío, inicia sesión con correo y clave
	clave        string
}

// corrida es el resultado de una serie de peticiones.
type corrida struct {
	Nombre       string  `json:"nombre"`
	Descripcion  string  `json:"descripcion"`
	Peticiones   int     `json:"peticiones"`
	Errores      int     `json:"errores"`
	Hits         int     `json:"cache_hits"`
	Misses       int     `json:"cache_misses"`
	SegTotal     float64 `json:"seg_total"`
	RPS          float64 `json:"rps"`
	MediaMs      float64 `json:"media_ms"`
	P50Ms        float64 `json:"p50_ms"`
	P95Ms        float64 `json:"p95_ms"`
	P99Ms        float64 `json:"p99_ms"`
	MaxMs        float64 `json:"max_ms"`
	PrimerError  string  `json:"primer_error,omitempty"`
	ConCache     bool    `json:"con_cache"` // la API devolvió X-Cache
	CumpleLimite *bool   `json:"cumple_limite,omitempty"`
}

// reporte es reports/bench_api_<maquina>.json.
type reporte struct {
	Generado      string    `json:"generado"`
	Maquina       string    `json:"maquina"`
	SO            string    `json:"so"`
	CPUs          int       `json:"cpus_cliente"`
	URL           string    `json:"url"`
	Autenticacion string    `json:"autenticacion"` // "jwt" o "ninguna"
	VersionModelo string    `json:"version_modelo"`
	Concurrencia  int       `json:"concurrencia"`
	Semilla       uint64    `json:"semilla"`
	LimiteP95Ms   float64   `json:"limite_p95_ms,omitempty"`
	Corridas      []corrida `json:"corridas"`
}

func main() {
	var op opciones
	flag.StringVar(&op.url, "url", "http://localhost:8080", "URL base de la API")
	flag.IntVar(&op.peticiones, "n", 2000, "peticiones por corrida")
	flag.IntVar(&op.concurrencia, "c", 16, "clientes concurrentes")
	flag.IntVar(&op.calientes, "calientes", 32, "consultas distintas de la corrida caliente")
	flag.Uint64Var(&op.semilla, "semilla", 0, "semilla de las consultas frías (0 = según la hora, para no repetir claves de corridas anteriores)")
	flag.DurationVar(&op.timeout, "timeout", 5*time.Second, "plazo de cada petición")
	exigir := flag.Duration("exigir-p95", 0, "falla (código 1) si el p95 de alguna corrida no es menor que esto, p. ej. 100ms")
	nombreMaquina := flag.String("maquina", "", "nombre de la máquina para el archivo (por defecto, el hostname)")
	salida := flag.String("salida", "", "reporte (por defecto reports/bench_api_<maquina>[_jwt].json)")
	flag.StringVar(&op.token, "token", os.Getenv("BENCH_TOKEN"), "JWT para Authorization: Bearer (por defecto $BENCH_TOKEN)")
	flag.StringVar(&op.correo, "correo", os.Getenv("BENCH_CORREO"), "inicia sesión con este correo y la clave de $BENCH_CLAVE (por defecto $BENCH_CORREO)")
	flag.Parse()
	op.clave = os.Getenv("BENCH_CLAVE")

	if op.semilla == 0 {
		op.semilla = uint64(time.Now().UnixNano())
	}
	host := maquina.Host()
	if *nombreMaquina != "" {
		host = nombreArchivo(*nombreMaquina)
	}
	if *salida == "" {
		sufijo := ""
		if op.token != "" || op.correo != "" {
			sufijo = "_jwt"
		}
		*salida = filepath.Join("reports", "bench_api_"+host+sufijo+".json")
	}

	rep, err := medir(context.Background(), op, *exigir, log.Printf)
	if err != nil {
		log.Fatal(err)
	}
	rep.Maquina = host
	imprimir(rep)
	if err := escribirJSON(*salida, rep); err != nil {
		log.Fatal(err)
	}
	log.Printf("escrito %s", *salida)
	if err := verificar(rep); err != nil {
		log.Fatal(err)
	}
}

// medir corre la corrida fría y la caliente contra op.url.
func medir(ctx context.Context, op opciones, limite time.Duration, logf func(string, ...any)) (reporte, error) {
	if op.peticiones < 1 || op.concurrencia < 1 || op.calientes < 1 {
		return reporte{}, errors.New("-n, -c y -calientes deben ser positivos")
	}
	if op.timeout <= 0 || limite < 0 {
		return reporte{}, errors.New("-timeout debe ser positivo y -exigir-p95 no puede ser negativo")
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns, tr.MaxIdleConnsPerHost = op.concurrencia, op.concurrencia
	tr.IdleConnTimeout = 30 * time.Second
	cli := &http.Client{Timeout: op.timeout, Transport: tr}
	defer cli.CloseIdleConnections()
	op.url = strings.TrimRight(op.url, "/")
	if op.correo != "" {
		if op.clave == "" {
			return reporte{}, errors.New("-correo necesita la clave en $BENCH_CLAVE")
		}
		tok, err := iniciarSesion(ctx, cli, op.url, op.correo, op.clave)
		if err != nil {
			return reporte{}, err
		}
		op.token = tok
		logf("sesión iniciada como %s", op.correo)
	}
	mod, err := leerModelo(ctx, cli, op.url, op.token)
	if err != nil {
		return reporte{}, err
	}
	gen := generador{deptos: mod.deptos, anoMin: mod.anoMin, base: rand.New(rand.NewPCG(op.semilla, 0)).Uint64N(1 << 40)}
	// La fría y la caliente usan mitades disjuntas de las combinaciones: si una
	// corrida no cabe en su mitad, se repetirían consultas (PR-41 H1).
	if mitad := gen.combinaciones() / 2; uint64(op.peticiones) > mitad || uint64(op.calientes) > mitad {
		return reporte{}, fmt.Errorf("-n (%d) y -calientes (%d) deben ser ≤ %d: con este modelo hay %d consultas distintas",
			op.peticiones, op.calientes, mitad, gen.combinaciones())
	}
	rep := reporte{
		Generado: time.Now().Format(time.RFC3339), SO: runtime.GOOS + "/" + runtime.GOARCH, CPUs: runtime.NumCPU(),
		URL: op.url, Autenticacion: "ninguna", VersionModelo: mod.version, Concurrencia: op.concurrencia, Semilla: op.semilla,
		LimiteP95Ms: ms(limite),
	}
	if op.token != "" {
		rep.Autenticacion = "jwt"
	}

	// Calentamiento de conexiones y del servidor (no se mide).
	logf("calentando %d conexiones", op.concurrencia)
	if _, err := serie(ctx, cli, op.url, op.token, op.concurrencia, op.concurrencia, func(i int) []byte { return gen.caliente(i % op.calientes) }); err != nil {
		return reporte{}, err
	}
	// Las consultas calientes quedan en la caché antes de medir.
	if _, err := serie(ctx, cli, op.url, op.token, op.calientes, 1, gen.caliente); err != nil {
		return reporte{}, err
	}

	logf("corrida fría: %d consultas distintas, %d clientes", op.peticiones, op.concurrencia)
	fria, err := serie(ctx, cli, op.url, op.token, op.peticiones, op.concurrencia, gen.fria)
	if err != nil {
		return reporte{}, err
	}
	fria.Nombre = "fria"
	fria.Descripcion = "consultas distintas: calcula el modelo y guarda en la caché"

	logf("corrida caliente: %d peticiones sobre %d consultas en caché", op.peticiones, op.calientes)
	caliente, err := serie(ctx, cli, op.url, op.token, op.peticiones, op.concurrencia, func(i int) []byte { return gen.caliente(i % op.calientes) })
	if err != nil {
		return reporte{}, err
	}
	caliente.Nombre = "caliente"
	caliente.Descripcion = "las mismas consultas, ya en la caché"

	for _, c := range []*corrida{&fria, &caliente} {
		if limite > 0 {
			ok := c.P95Ms < ms(limite) && c.Errores == 0
			c.CumpleLimite = &ok
		}
		rep.Corridas = append(rep.Corridas, *c)
	}
	return rep, nil
}

// verificar falla si alguna corrida no cumple el límite o tuvo errores y, si
// la API tiene caché, si la corrida fría no salió entera del cálculo (claves
// de una corrida anterior con la misma semilla) o la caliente no salió entera
// de la caché (Redis caído o calentamiento fallido) (PR-41 H1).
func verificar(rep reporte) error {
	var errs []error
	for _, c := range rep.Corridas {
		if c.ConCache && c.Errores == 0 {
			switch {
			case c.Nombre == "fria" && c.Hits > 0:
				errs = append(errs, fmt.Errorf("corrida fria: %d de %d respuestas salieron de la caché; use otra -semilla", c.Hits, c.Peticiones))
			case c.Nombre == "caliente" && c.Hits != c.Peticiones:
				errs = append(errs, fmt.Errorf("corrida caliente: solo %d de %d respuestas salieron de la caché (¿Redis caído?)", c.Hits, c.Peticiones))
			}
		}
		if c.Errores > 0 {
			errs = append(errs, fmt.Errorf("corrida %s: %d de %d peticiones con error (%s)", c.Nombre, c.Errores, c.Peticiones, c.PrimerError))
		}
		if c.CumpleLimite != nil && !*c.CumpleLimite && c.Errores == 0 {
			errs = append(errs, fmt.Errorf("corrida %s: p95 %.2f ms no es menor que %.0f ms", c.Nombre, c.P95Ms, rep.LimiteP95Ms))
		}
	}
	return errors.Join(errs...)
}

// serie envía n peticiones con c clientes; cuerpo(i) es la consulta i. Cada
// cliente acumula sus propias medidas (sin estado compartido) y al final se
// juntan.
func serie(ctx context.Context, cli *http.Client, base, token string, n, c int, cuerpo func(int) []byte) (corrida, error) {
	type medida struct {
		dur   time.Duration
		err   string
		cache string
	}
	indices := make(chan int)
	partes := make([][]medida, c)
	var wg sync.WaitGroup
	ini := time.Now()
	for w := range c {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indices {
				d, cache, err := pedir(ctx, cli, base, token, cuerpo(i))
				m := medida{dur: d, cache: cache}
				if err != nil {
					m.err = err.Error()
				}
				partes[w] = append(partes[w], m)
			}
		}()
	}
	for i := range n {
		select {
		case indices <- i:
		case <-ctx.Done():
			close(indices)
			wg.Wait()
			return corrida{}, ctx.Err()
		}
	}
	close(indices)
	wg.Wait()
	total := time.Since(ini)

	r := corrida{Peticiones: n, SegTotal: total.Seconds(), RPS: float64(n) / total.Seconds()}
	var durs []float64
	for _, p := range partes {
		for _, m := range p {
			switch {
			case m.err != "":
				r.Errores++
				if r.PrimerError == "" {
					r.PrimerError = m.err
				}
				continue
			case m.cache == "HIT":
				r.Hits++
			case m.cache == "MISS":
				r.Misses++
			}
			durs = append(durs, ms(m.dur))
		}
	}
	r.ConCache = r.Hits+r.Misses > 0
	if len(durs) > 0 {
		slices.Sort(durs)
		var suma float64
		for _, d := range durs {
			suma += d
		}
		r.MediaMs = suma / float64(len(durs))
		r.P50Ms, r.P95Ms, r.P99Ms = percentil(durs, 50), percentil(durs, 95), percentil(durs, 99)
		r.MaxMs = durs[len(durs)-1]
	}
	return r, nil
}

// pedir hace una predicción y devuelve su latencia (hasta leer el cuerpo) y
// el valor de X-Cache.
func pedir(ctx context.Context, cli *http.Client, base, token string, cuerpo []byte) (time.Duration, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/predicciones", bytes.NewReader(cuerpo))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	conToken(req, token)
	ini := time.Now()
	res, err := cli.Do(req)
	if err != nil {
		return 0, "", err
	}
	b, err := io.ReadAll(res.Body)
	res.Body.Close()
	d := time.Since(ini)
	if err != nil {
		return d, "", err
	}
	if res.StatusCode != http.StatusOK {
		return d, "", fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return d, res.Header.Get("X-Cache"), nil
}

// percentil por rango más cercano sobre datos ordenados.
func percentil(ordenados []float64, p float64) float64 {
	k := int(math.Ceil(p/100*float64(len(ordenados)))) - 1
	return ordenados[max(0, min(k, len(ordenados)-1))]
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// modeloPublicado es lo que benchapi necesita de GET /api/v1/modelo.
type modeloPublicado struct {
	version string
	deptos  []string
	anoMin  int
}

func leerModelo(ctx context.Context, cli *http.Client, base, token string) (modeloPublicado, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/modelo", nil)
	if err != nil {
		return modeloPublicado{}, err
	}
	conToken(req, token)
	res, err := cli.Do(req)
	if err != nil {
		return modeloPublicado{}, fmt.Errorf("la API no responde en %s: %w", base, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return modeloPublicado{}, errors.New("GET /api/v1/modelo: HTTP 401; la API exige token: usa -correo (clave en $BENCH_CLAVE) o -token")
	case res.StatusCode != http.StatusOK:
		return modeloPublicado{}, fmt.Errorf("GET /api/v1/modelo: HTTP %d", res.StatusCode)
	}
	var m struct {
		Version       string   `json:"version"`
		Departamentos []string `json:"departamentos"`
		AnoMin        int      `json:"ano_min"`
	}
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return modeloPublicado{}, fmt.Errorf("GET /api/v1/modelo: %w", err)
	}
	deptos := slices.DeleteFunc(slices.Clone(m.Departamentos), func(d string) bool { return d == "OTROS" })
	if len(deptos) == 0 {
		return modeloPublicado{}, errors.New("GET /api/v1/modelo: el modelo no tiene departamentos")
	}
	return modeloPublicado{version: m.Version, deptos: deptos, anoMin: m.AnoMin}, nil
}

// generador arma consultas válidas y deterministas. fria(i) recorre en base
// mixta departamento × año × semana × edad × sexo a partir de base, así que
// dos índices distintos nunca dan la misma consulta (ni la misma clave de
// caché) hasta agotar las combinaciones.
type generador struct {
	deptos []string
	anoMin int
	base   uint64
}

const (
	anoMax  = 2025 // último año que acepta la validación del loader (2000–2025)
	semanas = 52
	edades  = 100
)

// anos es cuántos años distintos usan las consultas: anoMin … 2025.
func (g generador) anos() uint64 { return uint64(max(1, anoMax-g.anoMin+1)) }

func (g generador) consulta(k uint64) []byte {
	sexo := []string{"F", "M"}[k%2]
	k /= 2
	edad := k % edades
	k /= edades
	semana := 1 + k%semanas
	k /= semanas
	ano := g.anoMin + int(k%g.anos())
	k /= g.anos()
	depto := g.deptos[k%uint64(len(g.deptos))]
	b, _ := json.Marshal(map[string]any{
		"departamento": depto, "ano": ano, "semana": semana, "edad": edad, "tipo_edad": "A", "sexo": sexo,
	})
	return b
}

func (g generador) combinaciones() uint64 {
	return uint64(2*edades*semanas) * g.anos() * uint64(len(g.deptos))
}

// fria(i) es la consulta i de la corrida fría.
func (g generador) fria(i int) []byte {
	return g.consulta((g.base + uint64(i)) % g.combinaciones())
}

// caliente(i) es la consulta i del conjunto caliente: lejos de las frías.
func (g generador) caliente(i int) []byte {
	return g.consulta((g.base + g.combinaciones()/2 + uint64(i)) % g.combinaciones())
}

func nombreArchivo(s string) string {
	s = strings.ToLower(s)
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, s)
}

func imprimir(rep reporte) {
	fmt.Printf("API %s · modelo %s · %d clientes · %s\n", rep.URL, rep.VersionModelo, rep.Concurrencia, rep.SO)
	fmt.Printf("%-9s %7s %7s %8s %8s %8s %8s %8s %6s %6s\n", "corrida", "pet.", "err.", "RPS", "p50 ms", "p95 ms", "p99 ms", "máx ms", "HIT", "MISS")
	for _, c := range rep.Corridas {
		fmt.Printf("%-9s %7d %7d %8.0f %8.2f %8.2f %8.2f %8.2f %6d %6d", c.Nombre, c.Peticiones, c.Errores, c.RPS, c.P50Ms, c.P95Ms, c.P99Ms, c.MaxMs, c.Hits, c.Misses)
		if c.CumpleLimite != nil {
			fmt.Printf("  p95 < %.0f ms: %v", rep.LimiteP95Ms, *c.CumpleLimite)
		}
		fmt.Println()
	}
}

func escribirJSON(ruta string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	return os.WriteFile(ruta, append(b, '\n'), 0o644)
}

// conToken agrega Authorization: Bearer si hay token.
func conToken(req *http.Request, token string) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// iniciarSesion pide un JWT a POST /api/v1/auth/login (no se mide).
func iniciarSesion(ctx context.Context, cli *http.Client, base, correo, clave string) (string, error) {
	cuerpo, err := json.Marshal(map[string]string{"correo": correo, "clave": clave})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/auth/login", bytes.NewReader(cuerpo))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("login: la API no responde en %s: %w", base, err)
	}
	defer res.Body.Close()
	var s struct {
		Token string `json:"token"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(&s)
	if res.StatusCode != http.StatusOK || s.Token == "" {
		return "", fmt.Errorf("login como %s: HTTP %d %s", correo, res.StatusCode, s.Error)
	}
	return s.Token, nil
}
