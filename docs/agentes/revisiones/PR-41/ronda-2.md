# Veredicto — PR-41 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/30-latencia-api` @ `fff75dd`
- **Fecha:** 2026-11-20
- **Decisión:** APROBADO

## Verificación automática

> **Nota del entorno local:** el proxy del sandbox del VM devolvió `403 Forbidden` al descargar `github.com/klauspost/compress@v1.16.7` desde `proxy.golang.org`, lo que impidió compilar localmente los dos paquetes que importan el driver de MongoDB (`internal/almacen/mongo` y `cmd/coordinator`). Todo el resto de paquetes —incluido `cmd/benchapi` y sus 9 pruebas contra `internal/api`— se verificó localmente con `-race`, y la suite completa (`go test -race -count=1 ./...`), MongoDB/Redis, OpenAPI, Spin y Docker pasaron 6/6 en verde en el CI de GitHub Actions ([run 44686182476](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44686182476) sobre `fff75dd`).

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./cmd/benchapi ./cmd/benchmark ./cmd/entrenar ./cmd/gold ./cmd/informe ./cmd/loader ./cmd/node ./cmd/perfil ./cmd/revalidar ./cmd/word ./internal/almacen ./internal/api ./internal/cluster/... ./internal/estadistica ./internal/features ./internal/gold ./internal/ingest ./internal/maquina ./internal/ml
(vacío = ok)

$ go test -race -count=1 -v ./cmd/benchapi
=== RUN   TestMedirSeparaCacheFriaYCaliente
--- PASS: TestMedirSeparaCacheFriaYCaliente (0.20s)
=== RUN   TestSinCacheLoReporta
--- PASS: TestSinCacheLoReporta (0.03s)
=== RUN   TestExigirP95FallaSiNoSeCumple
--- PASS: TestExigirP95FallaSiNoSeCumple (0.02s)
=== RUN   TestErroresDelServidorSeCuentanYFallan
--- PASS: TestErroresDelServidorSeCuentanYFallan (0.02s)
=== RUN   TestGeneradorNoRepiteConsultas
--- PASS: TestGeneradorNoRepiteConsultas (0.62s)
=== RUN   TestColeccionPostmanCubreLasRutas
--- PASS: TestColeccionPostmanCubreLasRutas (0.00s)
=== RUN   TestCacheCaidaFallaLaCorridaCaliente
--- PASS: TestCacheCaidaFallaLaCorridaCaliente (0.03s)
=== RUN   TestSemillaRepetidaFallaLaCorridaFria
--- PASS: TestSemillaRepetidaFallaLaCorridaFria (0.08s)
=== RUN   TestOpcionesInvalidasSeRechazan
--- PASS: TestOpcionesInvalidasSeRechazan (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.033s

$ go test -race -count=1 ./cmd/benchapi ./cmd/benchmark ./cmd/entrenar ./cmd/gold ./cmd/node ./cmd/revalidar ./cmd/word ./internal/almacen ./internal/api ./internal/cluster/... ./internal/estadistica ./internal/features ./internal/gold ./internal/ingest ./internal/maquina ./internal/ml
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	3.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.015s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.396s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.710s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.736s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.260s
(17/17 paquetes del núcleo stdlib: PASS, 0 carreras)

$ bash -n scripts/evidencia-api.sh && bash scripts/check-skills-sync.sh && python3 -m json.tool reports/bench_api_jorge-pc.json >/dev/null && python3 -m json.tool docs/api/cc65-tf.postman_collection.json >/dev/null
skills: ok

$ gh pr checks 41   # run 44686182476 sobre fff75dd
Docker (imagen, clúster y caída de un nodo)	pass	2m12s
Ecosistema de agentes                       	pass	5s
Go (gofmt, vet, test -race)                 	pass	59s
MongoDB y Redis (tests de integración)      	pass	37s
OpenAPI 3 válido                            	pass	18s
Spin (modelo del protocolo y mutantes)      	pass	2m4s
```

Verificación en vivo de los escenarios corregidos (binario `cmd/benchapi`, prueba *overlay* con `-race` y ejecución de los scripts *prerequest* y *test* de `docs/api/cc65-tf.postman_collection.json` contra `internal/api.Servidor` real en copia temporal fuera del repositorio):

```text
$ benchapi -url http://127.0.0.1:38893 -n 400 -c 8 -calientes 16 -semilla 42 -exigir-p95 100ms
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria          400       0    28145     0.15     1.02     1.48     2.79      0    400  p95 < 100 ms: true
caliente      400       0    33699     0.13     0.57     1.83     2.36    400      0  p95 < 100 ms: true
EXIT=0 · "maquina": "sandbox"

$ benchapi -url http://127.0.0.1:38893 -n 400 -c 8 -calientes 16 -semilla 42 -exigir-p95 100ms
corrida fria: 400 de 400 respuestas salieron de la caché; use otra -semilla
EXIT=1

$ benchapi -url http://127.0.0.1:39673 -n 100 -c 4 -calientes 8 -semilla 99 -exigir-p95 100ms   # caché caída
corrida caliente: solo 0 de 100 respuestas salieron de la caché (¿Redis caído?)
EXIT=1

$ benchapi -url http://127.0.0.1:38893 -n 20 -c 2 -calientes 2 -exigir-p95 -10ms
-timeout debe ser positivo y -exigir-p95 no puede ser negativo   (EXIT=1)

$ benchapi -url http://127.0.0.1:38893 -n 20 -c 2 -calientes 2 -timeout 0s
-timeout debe ser positivo y -exigir-p95 no puede ser negativo   (EXIT=1)

$ go test -race -count=1 -v -overlay <scratch>/overlay.json -run TestJuez ./cmd/benchapi
=== RUN   TestJuezRonda2VerificaReporteRealYLimites
--- PASS: TestJuezRonda2VerificaReporteRealYLimites (0.01s)
(valida reports/bench_api_jorge-pc.json con la nueva verificar(), rechaza -calientes > mitad, -timeout < 0, 1 MISS en caliente y 1 HIT en fría)

Colección docs/api/cc65-tf.postman_collection.json (tras precalentar LORETO/2024/10/30/A/F):
- Corrida 1 (semana_prueba=31, edad_prueba=11): 8 peticiones, 25 aserciones, 0 fallas (ítem 3 X-Cache=MISS, ítem 4 X-Cache=HIT)
- Corrida 2 (semana_prueba=16, edad_prueba=57): 8 peticiones, 25 aserciones, 0 fallas (ítem 3 X-Cache=MISS, ítem 4 X-Cache=HIT)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (`gofmt -l .` vacío; `go vet` limpio en núcleo stdlib local y en CI job `136120977226`) |
| `go test -race ./...` | ✅ (carreras: 0; 9/9 tests en `cmd/benchapi`, 17/17 paquetes stdlib locales y 24/24 en CI job `136120977226`) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; ambos pasaron en verde) |
| Latencia API p95 < 100 ms | ✅ (`reports/bench_api_jorge-pc.json`, Compose con 3 nodos + MongoDB + Redis + gold real, 16 clientes, 2000 req/corrida): `fria` p95 = **3.51 ms** (7128 RPS, 0 HIT / 2000 MISS), `caliente` p95 = **2.88 ms** (8809 RPS, 2000 HIT / 0 MISS), 0 errores, aprobado por el nuevo `verificar(rep)` |
| CI del PR (`gh pr checks 41`) | ✅ 6/6 en verde sobre `fff75dd` (run `44686182476`) |
| Restricciones AGENTS.md §3 | ✅ solo stdlib en `cmd/benchapi` (`go.mod` sin cambios); `data/` fuera del repo; evidencia en `reports/bench_api_jorge-pc.json` lista para `cmd/informe` (#33) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Completo para la verificación de latencia en PC4 | Worker pool concurrente sin estado compartido mutable (`partes[w]` privado, cancelación limpia, `CloseIdleConnections`), verificación estricta de separación `MISS`/`HIT` y de `p95 < 100 ms`, evidencia real en Compose (3.51 ms fría / 2.88 ms caliente) y Newman como compuerta real en `evidencia-api.sh`. |
| Interfaz | 5 | Completo para pruebas de contrato de PC4 | `docs/api/cc65-tf.postman_collection.json` cubre las 6 rutas documentadas de `api.Rutas()` con 8 peticiones y 25 aserciones automáticas (incluyendo `X-Cache: MISS` → `HIT` por corrida). La SPA entra en TB2. |
| Informe | 2 | Completo para la evidencia JSON de latencia | `reports/bench_api_jorge-pc.json` listo para consumo automático por `cmd/informe` en #33 (sin cifras manuales). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — `cmd/benchapi` aprobaba `caliente` sin HITs (Redis caído), `fria` con HITs (semilla repetida) o rangos solapados · MAYOR | **RESUELTO** | Commit `6714fc6`: `medir` (`cmd/benchapi/main.go:147-150`) exige `op.peticiones <= mitad && op.calientes <= mitad` sobre mitades disjuntas de `combinaciones()`; `verificar` (`cmd/benchapi/main.go:197-216`) exige `Hits == 0` en `fria` y `Hits == Peticiones` en `caliente` cuando `ConCache` es `true`, manteniendo intacto el modo sin caché (`TestSinCacheLoReporta`). Cubierto por `TestCacheCaidaFallaLaCorridaCaliente`, `TestSemillaRepetidaFallaLaCorridaFria` y `TestOpcionesInvalidasSeRechazan`, y comprobado en vivo (código 1 ante caché caída y semilla repetida; `reports/bench_api_jorge-pc.json` pasa la nueva verificación). |
| H2 (ronda 1) — `scripts/evidencia-api.sh:62` silenciaba los fallos de Newman con `\|\| true` · MAYOR | **RESUELTO** | Commit `81eab02`: se quitó `\|\| true` de la invocación de Newman en `scripts/evidencia-api.sh:63`. Con `set -euo pipefail` (`línea 7`), cualquier aserción fallida en Newman aborta el bloque `\| tee` con salida distinta de 0 y ejecuta el `trap EXIT` que baja Compose. |
| H3 (ronda 1) — Conexiones inactivas sin cerrar en `http.Transport` y aceptación silenciosa de `-exigir-p95` o `-timeout` no positivos · MENOR | **RESUELTO** | Commit `6714fc6`: `cmd/benchapi/main.go:131-138` rechaza `op.timeout <= 0 \|\| limite < 0`, clona `http.DefaultTransport` con `IdleConnTimeout: 30 * time.Second` y llama a `defer cli.CloseIdleConnections()`. Verificado en unit tests y en vivo con `-exigir-p95 -10ms` y `-timeout 0s` (código 1). |
| H4 (ronda 1) — Consulta fija ya calentada en el ítem 3 de Postman y 409 sin documentar en corridas seguidas · MENOR | **RESUELTO** | Commit `81eab02`: `docs/api/cc65-tf.postman_collection.json:5, 82-184` usa *prerequest* con `semana_prueba` (1–52) y `edad_prueba` (0–99) sobre `PIURA`/2024/`M`, añade la aserción explícita `X-Cache == "MISS"` en el ítem 3, reutiliza las mismas variables en el ítem 4 (`X-Cache == "HIT"`) y documenta el 409 entre corridas consecutivas y las 600 épocas. Comprobado en vivo en dos corridas completas consecutivas tras calentar `LORETO` del paso 5: 25/25 aserciones en verde en ambas. |
| H5 (ronda 1) — Observación preventiva sobre imagen Docker (99,1 % de 40 MB), `internal/maquina` y casillas del PR · MENOR | **RESUELTO** | Commit `6714fc6`: `cmd/benchapi/main.go:40, 103-106` reutiliza `maquina.Host()`. En CI job `136121367204` la imagen `cc65-tf` pesa `39,655,676 bytes` (< 40 MB, en verde); mantener el seguimiento de acotar `deploy/Dockerfile` a los binarios del contenedor antes de añadir otro `cmd/` (p. ej. #32). Nota editorial no bloqueante: actualizar las casillas y el bloque «Uso de IA» en la descripción del PR #41 en GitHub antes del merge. |

## Cómo explicarlo en la sustentación

Para demostrar de forma reproducible que la API responde en menos de 100 ms, `cmd/benchapi` lanza un *worker pool* de clientes HTTP concurrentes (solo biblioteca estándar, con acumuladores privados `partes[w]` sin estado compartido mutable) que genera consultas válidas y disjuntas a partir de `GET /api/v1/modelo`, separa una corrida fría (`X-Cache: MISS`, 0 HIT exigidos) de una caliente (`X-Cache: HIT`, 100 % HIT exigidos para detectar Redis caído) y falla con código 1 si el p95 no baja del límite o si falla la separación de caché. En el clúster real en Docker Compose (3 nodos + MongoDB + Redis + dataset gold real, 16 clientes y 2000 peticiones por corrida), el p95 medido en `reports/bench_api_jorge-pc.json` fue **3,51 ms** en frío (7128 RPS) y **2,88 ms** en caliente (8809 RPS) con cero errores, complementado por la colección de Postman (`docs/api/cc65-tf.postman_collection.json`, 8 peticiones y 25 aserciones de estado, caché `MISS`→`HIT` y latencia < 100 ms) ejecutada como compuerta obligatoria por Newman en `scripts/evidencia-api.sh`.
