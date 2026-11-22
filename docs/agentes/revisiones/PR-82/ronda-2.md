# Veredicto — PR-82 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/66-mapa` @ `50f5bb4` (base: `integracion/tb2` @ `0f9c0f5`)
- **Fecha:** 2026-11-22
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H5**) quedaron resueltos con evidencia reproducible en los commits `28a3c0a` (API/coordinador) y `6613351` (SPA/tests/estilos) sin introducir defectos ni cifras escritas a mano (`AGENTS.md` §3.2, §3.6): `?depto=` se normaliza a mayúsculas y se valida contra el GeoJSON + `OTROS` (`normalizarDepto`), resaltando los 5 departamentos agrupados al elegir `OTROS` (**H1**); `PanelDepto` divide la curva semanal en tramos contiguos `!c.insuficiente` (`tramos`), calcula el eje Y solo sobre los puntos dibujados, dibuja `<circle>` para semanas aisladas y muestra aviso explícito cuando no hay semanas suficientes (**H2**); `servirDocumento` preserva `Vary: Origin` con `Add`, emite `Cache-Control: private, no-cache`, diferencia ETag entre plano y gzip, parsea `Accept-Encoding` con `q` real (`ParseFloat`, `x-gzip`, `*`, cabeceras múltiples), valida tipos JSON no nulos y clona el slice (**H3**); `escalaPara` elimina el dominio artificial `[0, 1]`, muestra rangos reales por clase usada sin intervalos `X – X` (auditado sobre las 636 combinaciones reales de `reports/evaluacion.json`) y suma pruebas de leyenda, panel y reproducción semanal (**H4**); y se resolvieron los ajustes de rendimiento SVG, temporizador único, foco en `.depto-sel:focus-visible`, higiene y reporte vacío (**H5**). Toda la verificación local Go + SPA (`tsc -b`, 46/46 Vitest, `vite build` sin `url(data:)`, 24/24 paquetes Go con `-race`, 0 carreras) y los 7/7 jobs del CI sobre `50f5bb4` están en verde. *(Nota operativa de merge: tal como señala la descripción del PR, conviene fusionar primero #80 y resolver después el pequeño conflicto en `internal/api/api.go`.)*

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/api
github.com/JorgeGarcia005/cc65-tf-dengue/internal/api -> [bytes compress/gzip context crypto/rand crypto/sha256 encoding/hex encoding/json errors fmt .../docs/api .../almacen .../cluster .../features .../ingest .../ml io log math net net/http runtime/debug slices strconv strings sync sync/atomic time]

$ go test -race -count=1 -v -run 'Evaluacion|AceptaGzip|OpenAPI|Documento|LeerEvaluacion' ./internal/api ./cmd/coordinator
PASS (9/9 tests: TestOpenAPICubreLasRutas, TestOpenAPIMapasEnLineaBienFormados, TestEvaluacionYRiesgoConElReporteReal,
TestEvaluacionConservaVaryDeCORS, TestEvaluacionSinReporteDa503, TestEvaluacionInvalidaNoArranca, TestAceptaGzipYETag,
TestDocumentoNoCompartePlano, TestLeerEvaluacion)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	10.244s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	1.037s

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.465s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.202s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.704s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.777s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.282s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.237s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.426s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.060s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.144s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.032s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.074s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	13.702s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.979s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.714s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.385s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.119s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.076s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.275s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.045s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.270s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.773s

$ (en web/) npm ci --no-audit --no-fund && npm run typecheck && npm test && npm run build && ! grep -l 'url(data:' dist/assets/*.css
✓ src/pruebas/mapa.test.tsx (19 tests)
✓ src/pruebas/app.test.tsx (22 tests)
✓ src/pruebas/cliente.test.ts (5 tests)
Test Files  3 passed (3) · Tests  46 passed (46)
✓ built in 969ms · SPA OK (sin data:)

$ sha256sum web/src/datos/peru_departamental_simple.geojson
3d34d5450a1b1f059eebdf7386e5b5d071a5063bf3679fcbbbc5314493791290  web/src/datos/peru_departamental_simple.geojson

$ grep -rnE '[0-9]+[.,][0-9]{2,}' web/src --include='*.ts*' | grep -v pruebas
(vacío = ok)

$ <auditoría Node + d3-scale sobre reports/evaluacion.json real (636 filtros y 126 curvas)>
geoTotal: 25 · modeloSinTodos: 21 · faltanEnMapa: [] · enOtros: [APURIMAC, AREQUIPA, HUANCAVELICA, MOQUEGUA, TACNA]
leyenda (636): vacias=46 (sin dominio [0,1]), conXguionX=0, conRangoInvertido=0
curvas (126): sinCurva=39 (aviso explícito), unPunto=2 (<circle>), multiTramo=51 (subcaminos sin puentear huecos)

$ gh pr checks 82   # run 44980191071 sobre 50f5bb4
Docker (imagen, clúster y caída de un nodo)	pass	1m53s
Ecosistema de agentes                       	pass	8s
Go (gofmt, vet, test -race)                 	pass	59s
MongoDB y Redis (tests de integración)      	pass	46s
OpenAPI 3 válido                            	pass	21s
SPA (tipos, tests, build e imagen)          	pass	44s
Spin (modelo del protocolo y mutantes)      	pass	1m54s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 24/24 paquetes con tests PASS) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; suite completa en verde) |
| Latencia API p95 < 100 ms | no aplica (`POST /api/v1/predicciones` sin cambios; `/api/v1/riesgo` y `/api/v1/evaluacion` sirven buffers precomprimidos en memoria) |
| SPA (`tsc -b`, Vitest, `vite build`, sin `url(data:)`) | ✅ (46/46 tests PASS, build 969 ms, sin `data:` en CSS) |
| CI GitHub Actions sobre `50f5bb4` | ✅ (7/7 jobs PASS, run `44980191071`) |

## Verificación contra los criterios del issue #66

| Criterio (#66) | Estado | Evidencia |
|---|---|---|
| 1. GeoJSON departamental versionado con fuente, licencia y sha256; todos los departamentos de `evaluacion.json` casan con el mapa | ✅ | `web/src/datos/peru_departamental_simple.geojson` (MPL-2.0, `LEEME.md`, sha256 `3d34d545…1290`), 20/20 del modelo casan y 5 caen en `OTROS` (verificado en `mapa.test.tsx` y auditoría Node). |
| 2. Filtros semana (con reproducción), grupo de edad y métrica actualizan mapa, leyenda y panel | ✅ | `web/src/paginas/Mapa.tsx`, `web/src/mapa/MapaPeru.tsx`, `web/src/mapa/datos.ts`; cubierto por `mapa.test.tsx` (filtros, cambio de leyenda/cifras del panel y reproducción con `vi.useFakeTimers()`). |
| 3. Panel del departamento: curva por semana y tasa observada vs calibrada; celdas `insuficiente` señaladas | ✅ | `web/src/mapa/PanelDepto.tsx`: `tramos()` con escala Y sobre puntos dibujados, `<path>` por tramo, `<circle>` por semana aislada, aviso `data-prueba="curva-vacia"` y aviso de `insuficiente`; capturas E2E diferidas a #70. |
| 4. Navegable con teclado y tooltip accesible | ✅ | `role="button"`, `tabIndex={0}`, `Enter`/`Space`, `aria-pressed`, `aria-live="polite"`, `.depto-sel:focus-visible` y tabla alternativa; cubierto en `mapa.test.tsx`. |
| 5. Ninguna cifra en el código: todo viene de `GET /api/v1/riesgo` | ✅ | `grep -rnE '[0-9]+[.,][0-9]{2,}' web/src --include='*.ts*' \| grep -v pruebas` sin coincidencias. |
| 6. `GET /api/v1/riesgo` y `GET /api/v1/evaluacion` sirven `reports/evaluacion.json` con gzip y ETag; sin el archivo → 503 | ✅ | `internal/api/evaluacion.go` (100 % stdlib, `Vary` con `Add`, `Cache-Control: private, no-cache`, ETag por representación, parser `q` RFC 9110, 503 sin archivo) y 9/9 tests en verde (autorización por rol en #61). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Responde directamente la pregunta de impacto social (ODS 3) con cifras trazables de `reports/evaluacion.json`, sin distorsiones en curvas ni rangos inventados. |
| Implementación | 8 | Avanza API y despliegue | Endpoints `/api/v1/riesgo` y `/api/v1/evaluacion` en Go stdlib con precompresión gzip, ETag por representación, `Vary` compatible con CORS (`ADR-0007`), Compose actualizado y guardía CSP en CI (`assetsInlineLimit: 0`). |
| Interfaz | 5 | Vista principal completa | Coroplético accesible por teclado con tabla alternativa, filtros compartibles en URL, reproducción semanal con temporizador único, leyenda con rangos observados y curva semanal por tramos sin inventar certeza. |
| Informe | 2 | sin cambios | No corresponde a este PR. |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `6613351` (`web/src/mapa/datos.ts:108-113`, `Mapa.tsx:15, 26`, `MapaPeru.tsx:39, 65-70`, `PanelDepto.tsx:29-52`, `mapa.test.tsx:165-192`): `normalizarDepto` normaliza a mayúsculas y solo acepta nombres del GeoJSON o `OTROS`; al seleccionar `OTROS` se resaltan todos los departamentos agrupados y el panel los enumera explícitamente. |
| H2 (ronda 1 · BLOQUEANTE) | RESUELTO | `6613351` (`web/src/mapa/datos.ts:88-106`, `PanelDepto.tsx:23-28, 82-131`, `vigia.css:495-511`, `mapa.test.tsx:194-214`): `tramos` divide la curva por semanas contiguas `!c.insuficiente`, `maximo` usa solo puntos dibujados, `Serie` emite subcaminos `M…L…` y `<circle>` en semanas aisladas, y muestra `data-prueba="curva-vacia"` cuando ninguna semana alcanza `min_casos`. |
| H3 (ronda 1 · MAYOR) | RESUELTO | `28a3c0a` (`internal/api/evaluacion.go:29-45, 60-72, 95-154`, `evaluacion_test.go:63-140`): `cab.Add("Vary", "Accept-Encoding")` probado con middleware CORS (`TestEvaluacionConservaVaryDeCORS`), `Cache-Control: private, no-cache`, ETag diferenciado (`etag` vs `etagGzip`), `Accept-Encoding`/`If-None-Match` con `Values` y `strconv.ParseFloat` (`gzip;q=0.0`, `q=0.000`, `Q=0`, `*`, `x-gzip`), validación tipada que rechaza `null`/vacío en `cargarEvaluacion` y `slices.Clone(b)` en `nuevoDocumento` (`TestDocumentoNoCompartePlano`). |
| H4 (ronda 1 · MAYOR) | RESUELTO | `6613351` (`web/src/mapa/datos.ts:44-86`, `MapaPeru.tsx:66, 89-109`, `mapa.test.tsx:216-269`): `escalaPara` devuelve `vacia: true` sin inventar `[0, 1]` y construye `clases` con mínimo y máximo observados de cada color usado (`conXguionX = 0` en los 636 filtros reales); tests añadidos para escala vacía, 1 solo departamento, actualización de leyenda y panel con filtros y reproducción con temporizadores falsos. |
| H5 (ronda 1 · MENOR) | RESUELTO | `6613351` + `28a3c0a` (`MapaPeru.tsx:11-12`, `Mapa.tsx:71-90`, `vigia.css:416-419`, `datos.ts`, `LEEME.md:9`, `api.go:10-11`, `cmd/coordinator/main.go:315-332`, `evaluacion_test.go`): proyección SVG calculada una sola vez por módulo, temporizador único con refs durante reproducción, foco visible en `.depto-sel:focus-visible`, sin línea en blanco final en `MapaPeru.tsx`, `IndiceRiesgo.semanas()` eliminado, `LEEME.md` corregido a `mapa.test.tsx`, comentario de paquete de `api.go` actualizado, archivo `-evaluacion` vacío rechazado con error (`TestLeerEvaluacion`) y «Madre de Dios» con preposición en minúscula. |

## Cómo explicarlo en la sustentación

La vista del mapa responde la pregunta de impacto social (ODS 3): a partir del cubo `departamento × semana × grupo de edad` calculado en `reports/evaluacion.json` y servido precomprimido con `gzip`, `Cache-Control: private, no-cache`, `Vary` compatible con CORS y `ETag` diferenciado por representación desde la API en Go stdlib (`/api/v1/riesgo`), la SPA pinta los 25 departamentos del Perú con `d3-geo` usando rangos reales por quintiles de riesgo calibrado o de casos esperados con signos de alarma (la medida que anticipa demanda de camas). Los filtros y la selección quedan validados en la URL compartible, la reproducción semanal avanza semana a semana con un único temporizador, la vista es navegable por teclado con tabla alternativa, y tanto el mapa (rayado) como la curva semanal del panel (cortes por tramos y escala Y solo sobre semanas suficientes) señalan explícitamente la falta de datos para no inventar certeza.
