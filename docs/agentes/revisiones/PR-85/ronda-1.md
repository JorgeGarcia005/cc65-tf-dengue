# Veredicto — PR-85 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/67-inicio-prediccion` @ `d903e87` (base: `integracion/tb2` @ `00e1400`)
- **Fecha:** 2026-11-22
- **Decisión:** CAMBIOS

El PR implementa las vistas de **Inicio** (`web/src/paginas/Inicio.tsx`, `web/src/inicio/datos.ts`) y **Predicción de un caso** (`web/src/paginas/Prediccion.tsx`, `web/src/prediccion/datos.ts`), añade el desglose `logit`/`sesgo`/`contribuciones` en `POST /api/v1/predicciones` (`internal/api/modelo.go`, `internal/api/api.go`) manteniendo el núcleo 100 % stdlib (`AGENTS.md` §3.2) e invalidando entradas de caché antiguas sin contribuciones (`len(p.Contribuciones) > 0`), actualiza `docs/api/openapi.yaml` y sincroniza `docs/informe/pc4/10-api.md`. Toda la verificación local (`gofmt`, `go vet`, `-tags integracion`, 25/25 paquetes Go con `-race` y 0 carreras, 62/62 tests de Vitest incluidos `npm test -- inicio` y `npm test -- prediccion`, `tsc -b`, `vite build` sin `url(data:)`, cero cifras escritas a mano en `web/src/`) y los 7/7 jobs de GitHub Actions están en verde. En la auditoría en vivo contra binarios compilados de `cmd/coordinator` (modelo real de 31 variables reconstruido desde `reports/entrenamiento.json`, `reports/evaluacion.json`, JWT y CORS) y `cmd/web` (CSP estricta `style-src 'self'`), Google Chrome headless mostró 0 errores de consola y 0 violaciones de CSP, `sumaLogit(pred) === pred.logit` cuadró exactamente en 100/100 consultas reales, los enlaces de prioridades abren `/mapa` filtrado y `cmd/benchapi` confirmó latencias muy por debajo de 100 ms (`p95 = 3,93 ms` fría / `3,57 ms` caliente con payload de `2 865` bytes). No hay hallazgos **BLOQUEANTES**; se emiten **2 hallazgos MAYORES** y **1 MENOR** para cerrar antes de aprobar: `Prediccion.tsx` no valida el cuerpo en respuestas `200` (un HTML de proxy o JSON incompleto lanza un `TypeError` no capturado y deja la SPA en blanco) y `Inicio.tsx` tiene divisiones sin guarda ante `0` (**H1**); en anchos de tableta/pantalla partida (`768px`) y móvil (`375px`) las tablas de `Inicio` desbordan sus tarjetas y la ventana horizontalmente (**H2**); y ajustes menores de higiene (`"aporte":-0` en 15/31 campos JSON del modelo real, código muerto `sigmoide` y parámetro sin efecto `departamentos` en `prediccion/datos.ts`, estilo/ARIA de `<fieldset aria-invalid>`, asimetría FMA en `arm64` en `api_test.go:161` y nota de regeneración de `bench_api_jorge-pc_jwt.json`) (**H3**).

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

$ go list -f '{{.Imports}}' ./internal/api
[bytes compress/gzip context crypto/rand crypto/sha256 encoding/base64 encoding/hex encoding/json errors fmt github.com/JorgeGarcia005/cc65-tf-dengue/docs/api github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster github.com/JorgeGarcia005/cc65-tf-dengue/internal/features github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws io log math net net/http net/url runtime/debug slices strconv strings sync sync/atomic time unicode/utf8]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	12.153s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.883s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.714s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.822s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.261s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.391s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.445s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.753s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.073s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.190s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.026s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.051s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.128s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.992s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.432s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.102s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.043s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.029s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.999s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.235s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.752s

$ go test -race -count=1 -run 'TestPrediccionContribucionesSumanElLogit|TestCacheViejaSinContribucionesSeRecalcula|TestOpenAPI|TestPrediccionCoincideConElModeloEntrenado' -v ./internal/api
PASS (6/6: TestPrediccionCoincideConElModeloEntrenado, TestOpenAPICubreLasRutas, TestOpenAPIMapasEnLineaBienFormados, TestPrediccionContribucionesSumanElLogit, TestCacheViejaSinContribucionesSeRecalcula, TestOpenAPISeguridad)

$ go test -race -count=1 -run Fragmentos -v ./cmd/informe
--- PASS: TestFragmentosSalenDelFuente (0.22s)

$ (en web/) npm ci --silent && npx tsc -b && npx vitest run && npm run build && ! grep -l 'url(data:' dist/assets/*.css
Test Files  5 passed (5)
     Tests  62 passed (62)
✓ built in 1.77s
NO url(data: OK

$ (en web/) npm test -- inicio && npm test -- prediccion
✓ src/pruebas/inicio.test.tsx (7 tests)
✓ src/pruebas/prediccion.test.tsx (9 tests)

$ grep -rnE '[0-9]+[.,][0-9]{2,}' web/src --include='*.ts*' | grep -v pruebas
(vacío = ok)

$ <prueba en vivo: cmd/coordinator con modelo real de 31 variables (reports/entrenamiento.json), reports/evaluacion.json, JWT y CORS + cmd/web con CSP estricta>
- POST /api/v1/predicciones (PIURA, 2024, sem 14, 8 A, F): HTTP 200, 2865 bytes, ACAO=http://127.0.0.1:18081, ACEH="X-Cache, Retry-After, Location"
- 100 consultas variadas contra el modelo real de 31 variables: 0/100 discrepancias entre sumaLogit(pred) en JS y pred.logit en Go
- go run ./cmd/benchapi -url http://127.0.0.1:18080 -token <jwt> -n 1000 -c 16 -exigir-p95 100ms:
  fria:     1000 pet., 0 err., 7875 RPS, p50=1.56 ms, p95=3.93 ms, p99=5.84 ms  (p95 < 100 ms: true)
  caliente: 1000 pet., 0 err., 8563 RPS, p50=1.41 ms, p95=3.57 ms, p99=4.79 ms  (p95 < 100 ms: true)
- Chrome headless @ 1280px contra cmd/web (style-src 'self'): 0 errores de consola, 0 violaciones CSP,
  10 prioridades exactas, enlace #1 → /mapa?depto=PIURA&semana=20&etapa=adulto_30_59&metrica=esperados (panel «Piura»),
  Predicción PIURA («✓ cuadra», barras renderizadas sin violar CSP) y AREQUIPA (aviso «Otros», «✓ cuadra»).
- Chrome headless responsive en Inicio:
  @ 1280px: docScrollW=1265 / docClientW=1265 (ok)
  @ 768px:  docScrollW=789  / docClientW=753, tabla «Grupos de edad» scrollW=366 > parentW=333 (desborde, ver H2)
  @ 375px:  docScrollW=786  / docClientW=375, tabla prioridades scrollW=624 (desborde, ver H2)

$ gh pr checks 85   # sobre d903e87
Docker (imagen, clúster y caída de un nodo)	pass	2m28s
Ecosistema de agentes                       	pass	6s
Go (gofmt, vet, test -race)                 	pass	1m3s
MongoDB y Redis (tests de integración)      	pass	55s
OpenAPI 3 válido                            	pass	22s
SPA (tipos, tests, build e imagen)          	pass	49s
Spin (modelo del protocolo y mutantes)      	pass	1m49s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (25/25 paquetes, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`) |
| Latencia API p95 < 100 ms | ✅ (`3,93 ms` fría / `3,57 ms` caliente con modelo real de 31 variables, JWT y contribuciones; `2 865` bytes/resp.) |
| SPA (`tsc -b`, Vitest, `vite build`, sin `url(data:)`, CSP en Chrome) | ✅ (62/62 tests, 0 errores consola/CSP) |
| CI GitHub (PR #85) | ✅ (7/7 jobs en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | La portada responde directamente la pregunta de impacto ODS 3 (dónde, cuándo y a qué grupo de edad reforzar) con cifras reales de `reports/evaluacion.json`, sin cifras escritas a mano. |
| Implementación | 8 | 7.8 / 8 | `POST /api/v1/predicciones` devuelve contribuciones exactas bit a bit (`float64(m.Pesos[j] * v)`), invalida caché vieja, conserva p95 < 4 ms y 100 % stdlib; detalles menores en H3 (`-0` en JSON, FMA en test `arm64`). |
| Interfaz | 5 | 4.2 / 5 | Inicio y Predicción claros, accesibles y funcionales bajo CSP estricta; corregir H1 (robustez ante 200 no-JSON/incompleto) y H2 (desborde horizontal de tablas a 768px/375px) antes de la sustentación. |
| Informe | 2 | 2.0 / 2 | `docs/informe/pc4/10-api.md` sincronizado con los fragmentos del código (`TestFragmentosSalenDelFuente` en verde). |
| Video | 2 | — | No corresponde a este PR (queda listo el flujo Inicio → Mapa → Predicción para grabarlo). |

## Hallazgos

### H1 — Respuestas HTTP 200 malformadas o incompletas sin validar en `Prediccion.tsx` (pantalla en blanco por `TypeError`) y divisiones sin guarda en `Inicio.tsx` · MAYOR

- **Dónde:** `web/src/paginas/Prediccion.tsx:33-36, 53-54, 155-157`, `web/src/api/cliente.ts:74-91`, `web/src/paginas/Inicio.tsx:27, 65-69, 138, 200`, `web/src/main.tsx:17-25`
- **Evidencia:**
  1. En `web/src/api/cliente.ts:74-91`, si un proxy o fallo intermedio responde `200 OK` con un cuerpo no-JSON (`<html>proxy</html>` o vacío), `JSON.parse` falla silenciosamente (`datos = undefined`) y `postMedido` resuelve `{ datos: undefined as T, ... }`. Este mismo patrón de fallo ya fue cubierto con guarda de tipo y test explícito para el login tras PR-79 H3 (`Sesion.tsx:73` con `esSesion`, `app.test.tsx:214` «un login 200 sin cuerpo válido muestra error, sin romper»).
  2. Sin embargo, `Prediccion.tsx:53-54` pasa `r.datos` directo a `setResp({ pred: r.datos, ... })` sin comprobar su forma. Al renderizar `<ResultadoCaso>` (`Prediccion.tsx:155`), `contribucionesActivas(pred.contribuciones)` lanza `TypeError: Cannot read properties of undefined (reading 'contribuciones')` (lo mismo ocurre si llega un JSON sin arreglo `contribuciones` o sin números `logit`/`sesgo`/`riesgo`). Como `web/src/main.tsx` y `App.tsx` no tienen `ErrorBoundary`, el error desmonta toda la SPA a pantalla en blanco en vez de mostrar el `role="alert"`.
  3. De modo análogo, `GET /api/v1/modelo` (`Prediccion.tsx:33-36`) no comprueba `typeof m?.version === 'string'` (un `200 {}` mostraría `(versión undefined)`).
  4. En `Inicio.tsx:27`, solo se verifica `d?.version_esquema !== VERSION_ESPERADA` (mientras `cargarEvaluacion` en `internal/api/evaluacion.go:51-72` valida `riesgo` y `prioridades` pero no `metricas_test`, `filas_test` ni `positivos_test`), `Inicio.tsx:138` calcula `(100 * c.casos_esperados) / picos[0]!.casos_esperados` (`width: NaN%` si el máximo fuese `0`) e `Inicio.tsx:200` calcula `doc.positivos_test / doc.filas_test` (`NaN %` si `filas_test === 0`).
- **Corrección sugerida:**
  - Agregar un validador `esPrediccion(v: unknown): v is Prediccion` en `web/src/prediccion/datos.ts` (o `api/tipos.ts`) que verifique que `v` sea objeto con `riesgo`, `probabilidad_ponderada`, `umbral`, `logit`, `sesgo` numéricos finitos y `Array.isArray(v.contribuciones)`. Si no cumple, lanzar un `Error('Respuesta de predicción inválida.')` antes de `setResp` (y añadir un test en `prediccion.test.tsx` con `200 <html>proxy</html>`, análogo a `app.test.tsx:214`). En `GET /api/v1/modelo`, solo asignar `setModelo(m)` si `m && typeof m.version === 'string'`.
  - En `Inicio.tsx`, validar que `d.riesgo`, `Array.isArray(d.prioridades)` y `d.metricas_test` estén presentes junto a `version_esquema`, y proteger los denominadores de las líneas `138` (`picos[0]!.casos_esperados > 0 ? ... : 0`) y `200` (`doc.filas_test > 0 ? doc.positivos_test / doc.filas_test : 0`).

### H2 — Desborde horizontal de las tablas en `Inicio` en anchos de tableta / pantalla partida (`768px`) y móvil (`375px`) · MAYOR

- **Dónde:** `web/src/paginas/Inicio.tsx:87-126, 145-170`, `web/src/estilos/vigia.css:574-585, 542-556`
- **Evidencia:**
  1. Medido en Google Chrome headless contra el build real sirviendo `reports/evaluacion.json`:
     - A **`1280px`** (escritorio): `docScrollW = 1265 === docClientW = 1265`, sin desborde (`tabla 1: 1141 <= 1205`, `tabla 2: 525 <= 589`).
     - A **`768px`** (tableta o ventana a media pantalla en la sustentación junto a Swagger/Postman): `docScrollW = 789 > docClientW = 753`, y la tabla de **Grupos de edad** (`scrollW = 366`) sobresale **33 px** fuera del borde derecho de su `.tarjeta` (`parentW = 333`).
     - A **`375px`** (móvil): `docScrollW = 786 > docClientW = 375` (> 2× el ancho de la pantalla) porque la tabla de 8 columnas (`scrollW = 624`) estira el grid sin scroll interno y `.inicio-columnas` / `.prediccion-columnas` (`vigia.css:582`) fija `minmax(320px, 1fr)` cuando el ancho útil dentro de `.contenido` (`padding: 28px` por lado) es `319px`.
  2. En CSS Grid los hijos tienen `min-width: auto` por defecto, y ninguna `<table className="tabla">` de `Inicio.tsx` tiene un contenedor con `overflow-x: auto`.
- **Corrección sugerida:**
  - Envolver las tablas de `Inicio.tsx` en `<div className="tabla-envoltura">` con `.tabla-envoltura { overflow-x: auto; max-width: 100%; }`.
  - En `vigia.css:574-585`, añadir `min-width: 0` a los hijos directos del grid (`.vista-inicio > *, .inicio-columnas > *, .prediccion-columnas > * { min-width: 0; }`) y usar `grid-template-columns: repeat(auto-fit, minmax(min(320px, 100%), 1fr))` (o subir el umbral de las dos columnas de `Inicio` si la tabla de 4 columnas necesita ~400 px para no requerir scroll a 768 px).

### H3 — Higiene y detalles menores en JSON de contribuciones, código muerto en `prediccion/datos.ts`, estilo/ARIA del `<fieldset>` y asimetría FMA en test `arm64` · MENOR

- **Dónde:** `internal/api/modelo.go:248`, `internal/api/api_test.go:161, 704`, `web/src/prediccion/datos.ts:54, 60-64, 83`, `web/src/paginas/Prediccion.tsx:126, 147, 212, 224`, `web/src/estilos/vigia.css:632-640`
- **Evidencia:**
  1. **`"aporte":-0` en JSON (`internal/api/modelo.go:248`):** Probado en vivo contra el modelo real de 31 variables (`reports/entrenamiento.json`), toda variable one-hot inactiva (`v == 0`) con peso negativo produce `-0.0` y `encoding/json` serializa **15 apariciones de `"aporte":-0`** en cada respuesta de `POST /api/v1/predicciones` (no rompe `c.aporte !== 0` en JS porque `-0 === 0`, pero ensucia la respuesta visible en Swagger/curl). Normalizar `if aporte == 0 { aporte = 0 }` (y comparar por valor o normalizado en `api_test.go:704`) lo elimina sin alterar `z`.
  2. **Código muerto y parámetro sin efecto (`web/src/prediccion/datos.ts:54, 62, 83`):** `export const sigmoide` (`datos.ts:83`) no se importa ni usa en ningún archivo (`rg -n sigmoide web/src/` solo lista su declaración). Además, en `etiquetaVariable(v, departamentos)` (`datos.ts:62`), `const d = departamentos.find((x) => x === clave) ?? clave` devuelve siempre `clave`, por lo que el parámetro `departamentos` (pasado en `Prediccion.tsx:147, 212`) no tiene efecto.
  3. **Estilos y ARIA de `<fieldset>` inválido (`web/src/paginas/Prediccion.tsx:126`, `web/src/estilos/vigia.css:632-640`):** `.formulario fieldset { border: 0; padding: 0; margin: 0; }` va después de `.formulario [aria-invalid='true']`, por lo que al faltar `Sexo` el fieldset queda sin borde (solo con una `box-shadow` desplazada) y aplica `aria-invalid` a un `<fieldset>` sin `role="radiogroup"`. Añadir `role="radiogroup"` y una regla `.formulario fieldset[aria-invalid='true'] { border: 2px solid var(--error); padding: 6px 10px; }` hace el aviso consistente con los inputs.
  4. **Texto de conteo en `Prediccion.tsx:224`:** Con `semana = 1` (`sin(0) = 0`), `semana_sin` tiene `aporte = 0` y entra en `«N variables en 0 (otras categorías) no aportan»`, aunque no sea una categoría; un texto como `«N variables con aporte 0 no se muestran»` es siempre exacto.
  5. **Asimetría FMA en `arm64` en test previo (`internal/api/api_test.go:161`):** `GOARCH=arm64 go test -c` + `go tool objdump` confirma que `modelo.go:248-249` compila a `FMULD + FADDD` (sin FMA, intencional y correcto para que JS y Go sumen igual), mientras `TestPrediccionCoincideConElModeloEntrenado` (`api_test.go:161`) compila a `FMADDD` y diverge en `46,6 %` (`1964/4212`) de entradas simuladas con `math.FMA` (pasa en `arm64` solo porque `semana=10, edad=34` coincide por azar; en `amd64` y `GOAMD64=v3` Go no emite FMA). Usar `float64(m.Pesos[j] * x[j])` también en `api_test.go:161` blinda el test en Apple Silicon / `arm64`.
  6. **Evidencia de latencia post-#67:** `cmd/benchapi` en vivo sobre el modelo real de 31 variables con JWT dio `p95 = 3,93 ms` (fría) y `3,57 ms` (caliente), muy lejos de 100 ms. No bloquea este PR, pero conviene regenerar `reports/bench_api_jorge-pc_jwt.json` (y `reports/evaluacion.json`) antes de cerrar TB2 (#70/#71) para que el JSON versionado corresponda al payload actual de ~2,9 KB.
- **Corrección sugerida:** Aplicar los puntos 1–5 en la misma ronda de corrección (o eliminar `sigmoide` si no se usará para verificar `σ(logit)` en `cuadra`), y dejar el punto 6 anotado para la corrida final de evidencias de TB2.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda del PR #85. |

## Cómo explicarlo en la sustentación

> La vista de **Inicio** responde la pregunta de impacto ODS 3 sin recalcular cifras en el navegador: consume `GET /api/v1/evaluacion` y muestra en el orden exacto de `reports/evaluacion.json` las 10 combinaciones departamento × semana × grupo de edad con más casos esperados con signos de alarma (enlazadas al mapa ya filtrado), las 5 semanas pico nacionales, los grupos de edad por riesgo y los KPIs del modelo con aviso si la API publica otra versión. La vista de **Predicción** valida con las mismas reglas que la API, llama a `POST /api/v1/predicciones` midiendo la latencia y leyendo `X-Cache` (expuesta por CORS), y desglosa el aporte `peso × valor` de cada variable activa verificando en el navegador que `sesgo + Σ aportes` reproduce bit a bit el `logit` devuelto por el modelo en Go (gracias al redondeo explícito `float64(m.Pesos[j] * v)` que evita fusiones FMA).
