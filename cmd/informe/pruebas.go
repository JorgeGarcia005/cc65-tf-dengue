package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// catalogoGo cuenta, por paquete, las funciones Test*, Benchmark* y Fuzz* de
// los archivos *_test.go del repositorio (sin node_modules ni carpetas ocultas).
func catalogoGo(raiz string) (map[string][3]int, error) {
	out := map[string][3]int{}
	err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			n := d.Name()
			if ruta != raiz && (strings.HasPrefix(n, ".") || n == "node_modules" || n == "data") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
		if err != nil {
			return err
		}
		paquete := filepath.ToSlash(filepath.Dir(ruta))
		c := out[paquete]
		for nombre, o := range f.Scope.Objects {
			if o.Kind != ast.Fun {
				continue
			}
			switch {
			case strings.HasPrefix(nombre, "Test") && nombre != "TestMain":
				c[0]++
			case strings.HasPrefix(nombre, "Benchmark"):
				c[1]++
			case strings.HasPrefix(nombre, "Fuzz"):
				c[2]++
			}
		}
		out[paquete] = c
		return nil
	})
	return out, err
}

var reCasoJS = regexp.MustCompile(`(?m)^\s*(it|test)\(`)

// casosJS cuenta los casos it(…) / test(…) de los archivos que cumplen patrón.
func casosJS(patron string) (archivos, casos int, err error) {
	rutas, err := filepath.Glob(patron)
	if err != nil {
		return 0, 0, err
	}
	for _, r := range rutas {
		b, err := os.ReadFile(r)
		if err != nil {
			return 0, 0, err
		}
		if n := len(reCasoJS.FindAll(b, -1)); n > 0 {
			archivos++
			casos += n
		}
	}
	return archivos, casos, nil
}

// descripcionPaquete: para qué sirven las pruebas de cada paquete.
var descripcionPaquete = map[string]string{
	"internal/ingest":            "Reglas de limpieza, reparaciones, bloques alineados y limpieza concurrente igual a la secuencial",
	"internal/ml":                "Gradiente (diferencias finitas), worker pool igual bit a bit al secuencial, entrenamiento",
	"internal/gold":              "Formato binario del gold y shards que reproducen el gradiente global",
	"internal/features":          "Codificación de variables y split determinista",
	"internal/cluster":           "Protocolo, caídas, plazos, reasignación, balanceo y equivalencia bit a bit por TCP",
	"internal/cluster/protocolo": "Validación y límites de los mensajes",
	"internal/api":               "Rutas, contratos, matriz de roles, CORS, cabeceras, caché, registro asíncrono y cierre ordenado",
	"internal/auth":              "JWT (algoritmo fijo, firma, vencimiento) y PBKDF2",
	"internal/ws":                "Handshake RFC 6455, frames, límites y hub que no bloquea",
	"internal/almacen":           "Contrato común de las implementaciones del almacén",
	"internal/almacen/mongo":     "MongoDB real: índices, publicación atómica y reconciliación",
	"internal/evaluacion":        "AUC, calibración, cubo de riesgo y casos esperados",
	"internal/estadistica":       "Media recortada e intervalo de confianza por bootstrap",
	"cmd/benchapi":               "Generador de consultas, percentiles y verificación de la caché",
	"cmd/coordinator":            "Arranque en modo entrenamiento y servicio, opciones y verificación contra el secuencial",
	"cmd/entrenar":               "Entrenamiento sobre el gold y comparación bit a bit de reportes",
	"cmd/evaluacion":             "Evaluación determinista y AUC de referencia",
	"cmd/gold":                   "Construcción del gold y de los shards",
	"cmd/node":                   "Conexión al coordinador, reintentos y reconexión tras una caída",
	"cmd/revalidar":              "Comparación campo por campo de los reportes de referencia",
	"cmd/usuarios":               "Alta de usuarios desde variables de entorno",
	"internal/almacen/redis":     "Redis real: contrato de la caché y plazo de cada operación",
	"internal/maquina":           "Descripción del equipo de cada benchmark",
	"cmd/informe":                "Informe al día con los reportes y el código; regiones generadas",
	"cmd/word":                   "Conversión del informe a Word en formato APA",
	"cmd/seguridad":              "Casos de seguridad contra una API de prueba",
	"cmd/benchmark":              "Benchmarks que separan tiempos y verifican los pesos",
	"cmd/web":                    "Servidor de la SPA: CSP, configuración y rutas",
}

// regionPruebas: catálogo de pruebas del Anexo E.
func regionPruebas() (string, error) {
	cat, err := catalogoGo(".")
	if err != nil {
		return "", err
	}
	paquetes := make([]string, 0, len(cat))
	for p, c := range cat {
		if c[0]+c[1]+c[2] > 0 {
			paquetes = append(paquetes, p)
		}
	}
	slices.Sort(paquetes)
	var b strings.Builder
	b.WriteString("Tabla: Catálogo de pruebas automáticas del repositorio {#tab:catalogo-pruebas}\n\n")
	b.WriteString("| Paquete o carpeta | Pruebas | Benchmarks | Qué cubren |\n|---|---:|---:|---|\n")
	var tt, tb int
	for _, p := range paquetes {
		c := cat[p]
		tt, tb = tt+c[0]+c[2], tb+c[1]
		fmt.Fprintf(&b, "| `%s` | %d | %d | %s |\n", p, c[0]+c[2], c[1], descripcionPaquete[p])
	}
	fu, cu, err := casosJS("web/src/pruebas/*.test.ts*")
	if err != nil {
		return "", err
	}
	_, ce, err := casosJS("web/e2e/*.ts")
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "| `web/src/pruebas` (Vitest, %d archivos) | %d | — | Guardas de ruta, cliente de la API, vistas, reductor del panel, réplica y reconexión del WebSocket |\n", fu, cu)
	fmt.Fprintf(&b, "| `web/e2e` (Playwright) | %d | — | Flujos por rol contra el sistema desplegado, con capturas |\n", ce)
	fmt.Fprintf(&b, "| **Total** | **%d** | **%d** | |\n", tt+cu+ce, tb)
	b.WriteString("\nNota: Funciones `Test*` y `Fuzz*` de los archivos `*_test.go` y casos `it(…)`/`test(…)` de la SPA, contados por `cmd/informe`; varias pruebas de Go recorren tablas de casos, así que el número de casos ejecutados es mayor. Además: `spin/verificar.sh` (sección 10), la colección de Postman (sección 12.4), `cmd/seguridad` (sección 15.5) y las pruebas en Docker del CI (sección 9).\n")
	return b.String(), nil
}
