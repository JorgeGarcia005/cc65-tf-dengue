# Veredicto — PR-41 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/30-latencia-api` @ `de04c90`
- **Fecha:** 2026-11-20
- **Decisión:** CAMBIOS

## Verificación automática

> **Nota del entorno local:** el proxy del sandbox del VM devolvió `403 Forbidden` al descargar la dependencia transitiva `github.com/klauspost/compress@v1.16.7` desde `proxy.golang.org`, lo que impidió compilar localmente los dos paquetes que importan el driver de MongoDB (`internal/almacen/mongo` y `cmd/coordinator`). Todo el resto de paquetes —incluido `cmd/benchapi` y sus pruebas contra `internal/api`— se verificó localmente con `-race`, y la suite completa de 24 paquetes más MongoDB/Redis, OpenAPI y Docker se verificó en verde en el CI de GitHub Actions ([run 44683044456](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44683044456)).

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./cmd/benchapi ./cmd/benchmark ./cmd/entrenar ./cmd/gold ./cmd/informe ./cmd/loader ./cmd/node ./cmd/perfil ./cmd/revalidar ./cmd/word ./internal/almacen ./internal/api ./internal/cluster/... ./internal/estadistica ./internal/features ./internal/gold ./internal/ingest ./internal/maquina ./internal/ml
(vacío = ok)

$ go test -race -count=1 -v ./cmd/benchapi
=== RUN   TestMedirSeparaCacheFriaYCaliente
--- PASS: TestMedirSeparaCacheFriaYCaliente (0.17s)
=== RUN   TestSinCacheLoReporta
--- PASS: TestSinCacheLoReporta (0.04s)
=== RUN   TestExigirP95FallaSiNoSeCumple
--- PASS: TestExigirP95FallaSiNoSeCumple (0.02s)
=== RUN   TestErroresDelServidorSeCuentanYFallan
--- PASS: TestErroresDelServidorSeCuentanYFallan (0.02s)
=== RUN   TestGeneradorNoRepiteConsultas
--- PASS: TestGeneradorNoRepiteConsultas (0.62s)
=== RUN   TestColeccionPostmanCubreLasRutas
--- PASS: TestColeccionPostmanCubreLasRutas (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	1.875s

$ go test -race -count=1 ./cmd/benchapi ./cmd/benchmark ./cmd/entrenar ./cmd/gold ./cmd/node ./cmd/revalidar ./cmd/word ./internal/almacen ./internal/api ./internal/cluster/... ./internal/estadistica ./internal/features ./internal/gold ./internal/ingest ./internal/maquina ./internal/ml
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	1.978s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.015s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.457s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.566s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.399s
(todos los 17 paquetes locales sin driver de Mongo: PASS, 0 carreras)

$ bash -n scripts/evidencia-api.sh && bash scripts/check-skills-sync.sh
skills: ok
```

Ejecución en vivo de `go run ./cmd/benchapi` y de la colección `docs/api/cc65-tf.postman_collection.json` (mediante *overlay* temporal en Go contra un `internal/api.Servidor` real con `almacen.CacheMemoria`, sin modificar el árbol de trabajo):

```text
$ go run ./cmd/benchapi -url http://127.0.0.1:37679 -n 400 -c 8 -calientes 16 -exigir-p95 100ms -salida <tmp>/bench.json
API http://127.0.0.1:37679 · modelo b7243bafb5d5 · 8 clientes · linux/amd64
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria          400       0     4973     1.28     3.58     5.34     9.31      0    400  p95 < 100 ms: true
caliente      400       0     5178     1.30     3.25     5.11     5.35    400      0  p95 < 100 ms: true

Colección docs/api/cc65-tf.postman_collection.json (corrida 1 contra internal/api.Servidor):
GET  /healthz                       -> 200 (18.4 ms)
GET  /api/v1/modelo                 -> 200 ( 3.3 ms)
POST /api/v1/predicciones           -> 200 ( 4.2 ms, X-Cache=MISS)
POST /api/v1/predicciones           -> 200 ( 1.8 ms, X-Cache=HIT)
POST /api/v1/predicciones (sin edad)-> 400 ( 1.8 ms)
GET  /api/v1/cluster                -> 200 ( 3.4 ms)
POST /api/v1/entrenamientos         -> 202 ( 1.8 ms)
GET  /api/v1/entrenamientos/{id}    -> 200 ( 1.1 ms)
Total assertions: 24, failures: 0
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (`gofmt -l .` vacío; `go vet ./...` en verde en CI y locales) |
| `go test -race ./...` | ✅ (carreras: 0; `cmd/benchapi` 6/6 PASS local + 24 paquetes en CI) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; ambos pasaron en verde) |
| Latencia API p95 < 100 ms | ✅ medición real (`reports/bench_api_jorge-pc.json`, Compose con 3 nodos + MongoDB + Redis + gold real, 16 clientes, 2000 req/corrida): `fria` p95 = **3.51 ms** (7128 RPS, 0 HIT / 2000 MISS), `caliente` p95 = **2.88 ms** (8809 RPS, 2000 HIT / 0 MISS), 0 errores, coherente con la ley de Little y con la marca de tiempo de `semilla`. Pendiente corregir los huecos de verificación en H1 y H2. |
| CI del PR (`gh pr checks 41`) | ✅ 5/5 en verde: `Ecosistema de agentes` (11s), `OpenAPI 3 válido` (20s), `MongoDB y Redis` (51s), `Go (gofmt, vet, test -race)` (54s) y `Docker` (2m0s) |
| Restricciones AGENTS.md §3 | ✅ `cmd/benchapi/main.go` usa solo biblioteca estándar (`go.mod` intacto); sin archivos en `data/`; sin cifras manuales en `docs/informe/` (la integración en `cmd/informe` corresponde a #33) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Casi completo para PC4 (pendiente H1–H2) | El worker pool de `cmd/benchapi` no comparte estado mutable (`partes[w]` privado por goroutine, cancelación limpia por contexto, 0 carreras), mide latencia real de punta a punta y acredita p95 < 4 ms con el gold real en Compose. Faltan las compuertas de verificación ante caché degradada/contaminada (H1) y fallos de Newman silenciados (H2). |
| Interfaz | 5 | Completo para el alcance de pruebas de PC4 | `docs/api/cc65-tf.postman_collection.json` cubre las 6 rutas documentadas de `api.Rutas()` (8 peticiones, 24 aserciones verificadas por `TestColeccionPostmanCubreLasRutas`, Newman y prueba en vivo). La SPA entra en TB2. |
| Informe | 2 | Completo para la evidencia JSON de API | Deja `reports/bench_api_jorge-pc.json` listo para ser consumido por `cmd/informe` en #33 (sin cifras manuales). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes; se registran **2 hallazgos mayores** (que comprometen las compuertas automáticas de verificación del issue #30) y **3 hallazgos menores** (incluyendo una observación preventiva de Docker):

### H1 — `cmd/benchapi` aprueba con código 0 una corrida `caliente` sin ningún HIT (Redis caído), una `fria` con 100 % HITs (semilla repetida) o un rango que solapa ambos conjuntos · MAYOR

- **Dónde:** `cmd/benchapi/main.go:126-138`, `cmd/benchapi/main.go:171-193`, `cmd/benchapi/main.go:375-383`
- **Evidencia:** El objetivo de `cmd/benchapi -exigir-p95 100ms` es verificar automáticamente la latencia con caché fría (MISS) y caliente (HIT). Sin embargo, `verificar(rep)` (`cmd/benchapi/main.go:182-193`) solo comprueba `c.Errores` y `c.P95Ms < limite`, sin verificar la separación HIT/MISS cuando la API anuncia caché (`c.ConCache == true`), y `medir` (`líneas 126-138`) no valida que `op.peticiones` y `op.calientes` quepan sin solaparse en `gen.combinaciones()`. Se reprodujeron tres fallos silenciosos mediante un *overlay* temporal con `-race`:
  ```text
  $ go test -race -count=1 -v -overlay <scratch>/overlay.json -run TestJuez ./cmd/benchapi
  === RUN   TestJuezCacheCaidaPasaComoCaliente
      juez_pr41_test.go:63: cacheCaida -> caliente: con_cache=true hits=0 misses=100 p95=1.82ms cumple=true verificar=<nil>
  --- PASS: TestJuezCacheCaidaPasaComoCaliente (0.07s)
  === RUN   TestJuezMismaSemillaContaminaFria
      juez_pr41_test.go:78: segunda corrida misma semilla -> fria: con_cache=true hits=100 misses=0 p95=1.62ms cumple=true verificar=<nil>
  --- PASS: TestJuezMismaSemillaContaminaFria (0.12s)
  === RUN   TestJuezColisionFriaYCalienteCuandoNSuperaMitad
      juez_pr41_test.go:91: n=5250 > combinaciones/2 (5200) -> fria: hits=32 misses=5218 verificar=<nil>
  --- PASS: TestJuezColisionFriaYCalienteCuandoNSuperaMitad (2.62s)
  ```
  En el primer caso, cuando Redis está caído pero `s.op.Cache != nil`, `internal/api/api.go:348-357` cae al cálculo local y responde `HTTP 200` con `X-Cache: MISS`; `benchapi` igualmente emite `"nombre": "caliente"`, `"descripcion": "las mismas consultas, ya en la caché"`, `"con_cache": true`, `"cache_hits": 0`, `"cumple_limite": true` y sale con código 0. En el segundo caso, repetir `-semilla` mide 100 % HITs de Redis y los reporta como corrida `fria` con código 0. En el tercero, superar `combinaciones()/2` envuelve los índices de `fria` sobre las claves ya calentadas de `caliente`.
- **Corrección sugerida:**
  1. En `medir`, comprobar que `uint64(op.peticiones) <= gen.combinaciones()/2 && uint64(op.calientes) <= gen.combinaciones()/2` (devolviendo error descriptivo si se supera).
  2. En `verificar(rep)`, cuando `c.ConCache` sea `true` (conservando intacto `TestSinCacheLoReporta` cuando `ConCache` es `false`), exigir que la corrida `caliente` efectivamente provenga de la caché (p. ej., fallar si `c.Misses > 0` o `c.Hits < c.Peticiones`, detectando Redis caído o fallo del calentamiento) y que la corrida `fria` no esté contaminada por claves previas (fallar si `c.Hits > 0`).
  3. Añadir pruebas unitarias con `-race` para caché caída, semilla repetida y límite de `combinaciones()/2`. *(Nota: `reports/bench_api_jorge-pc.json` ya tiene `0 HIT / 2000 MISS` en `fria` y `2000 HIT / 0 MISS` en `caliente`, por lo que esta corrección no invalida la evidencia ya medida.)*

### H2 — `scripts/evidencia-api.sh:62` silencia todos los fallos de Newman con `|| true` · MAYOR

- **Dónde:** `scripts/evidencia-api.sh:60-65`
- **Evidencia:** El script activa `set -euo pipefail` (`línea 7`) para abortar ante cualquier error de verificación (`evidencia-bd.sh` en línea 56, `benchapi -exigir-p95 100ms` en línea 59), y ya limpia el clúster en cualquier salida mediante `trap` (`línea 16`). Sin embargo, la línea 62 ejecuta:
  ```bash
  npx --yes newman run docs/api/cc65-tf.postman_collection.json --env-var "base_url=$url" || true
  ```
  Con `|| true`, si cualquier prueba de la colección de Postman falla (código HTTP distinto al esperado, latencia ≥ 100 ms o ausencia de `X-Cache: HIT`), el script descarta el código de salida distinto de 0 de Newman y concluye con código 0 e imprime `Listo: evidencia en evidencia/28/`.
- **Corrección sugerida:** Quitar `|| true` de `scripts/evidencia-api.sh:62` para que una aserción fallida en Newman haga fallar el script de evidencia (el `trap EXIT` seguirá bajando Compose). Si se quiere tolerar únicamente la falta de red para descargar `newman`, comprobar antes que `newman` esté disponible sin silenciar fallos de ejecución de la colección.

### H3 — Conexiones inactivas sin cerrar en `http.Transport` y aceptación silenciosa de `-exigir-p95` o `-timeout` no positivos · MENOR

- **Dónde:** `cmd/benchapi/main.go:92-93`, `cmd/benchapi/main.go:126-132`, `cmd/benchapi/main.go:172`
- **Evidencia:**
  - `medir` instancia un `&http.Transport{MaxIdleConns: op.concurrencia, MaxIdleConnsPerHost: op.concurrencia}` nuevo por llamada (`línea 131`) con `IdleConnTimeout: 0` (sin expiración) y nunca llama a `defer cli.CloseIdleConnections()`, dejando vivas las goroutines de lectura/escritura de las conexiones keep-alive al retornar `medir`.
  - Ejecutando el binario con `-exigir-p95 -10ms` contra un servidor local:
    ```text
    $ go run ./cmd/benchapi -url http://127.0.0.1:37679 -n 20 -c 2 -calientes 2 -exigir-p95 -10ms -salida <tmp>/bench_neg.json
    (código 0, sin línea "p95 < ... ms")
    ```
    La condición `if limite > 0` (`línea 172`) trata un plazo negativo igual que `0` (desactivado). Asimismo, `-timeout <= 0` desactiva el plazo del `http.Client` en vez de rechazarse.
- **Corrección sugerida:** Clonar `http.DefaultTransport.(*http.Transport)` (o fijar `IdleConnTimeout`) y agregar `defer cli.CloseIdleConnections()` en `medir`; rechazar `limite < 0` y `op.timeout <= 0` al validar las opciones.

### H4 — La colección de Postman nombra el ítem 3 como «primera vez: MISS» usando la misma consulta ya calentada por `evidencia-api.sh` y falla con 409 en corridas consecutivas rápidas · MENOR

- **Dónde:** `docs/api/cc65-tf.postman_collection.json:82-126, 252-298` y `scripts/evidencia-api.sh:47, 54, 62, 68`
- **Evidencia:**
  - El ítem 3 (`"Predicción (primera vez: MISS)"`, `línea 82`) envía `{"departamento":"LORETO","ano":2024,"semana":10,"edad":30,"tipo_edad":"A","sexo":"F"}`, idéntico al enviado en `scripts/evidencia-api.sh:47` y `:54`, por lo que al ejecutarse Newman en el paso 7b ya recibe `X-Cache: HIT` (el ítem 3 tampoco comprueba `X-Cache`).
  - En nuestra prueba en vivo contra `internal/api.Servidor`, una segunda ejecución inmediata de la colección mientras el entrenamiento del ítem 7 sigue en curso falló 4/24 aserciones: el ítem 7 (`POST /api/v1/entrenamientos` con `{"epocas": 600}`) devolvió `HTTP 409` (`ya hay un entrenamiento en curso`) y dejó `entrenamiento_id` vacío, provocando `HTTP 404` en el ítem 8. Además, en `evidencia-api.sh`, el paso 7b lanza un entrenamiento de 600 épocas justo antes de que el paso 8 envíe `SIGTERM` al coordinador, dejándolo cancelado (`fallido`) en MongoDB.
- **Corrección sugerida:** Usar una consulta distinta a la del paso 5 en el ítem 3 (o un script *prerequest* que genere valores válidos en variables de colección reutilizadas por el ítem 4 para garantizar `MISS` seguido de `HIT`, o bien ajustar el título del ítem 3 si se mantiene fija). **Importante:** conviene conservar `"epocas": 600` en el ítem 7 (no bajarlo a `1`), porque terminar un entrenamiento con pocas épocas sobrescribiría el `ModeloVigente` de 600 épocas en MongoDB (`internal/api/api.go:445-456`); basta documentar en la descripción de la colección que entre dos corridas completas debe terminar el entrenamiento en curso.

### H5 — Observación preventiva: imagen Docker al 99,1 % del límite de 40 MB, reutilización de `internal/maquina` y limpieza de casillas del PR · MENOR

- **Dónde:** `deploy/Dockerfile:22`, `.github/workflows/ci.yml:109`, `cmd/benchapi/main.go:72-83, 101-104, 385-393` y descripción del PR #41
- **Evidencia:**
  - En el job `Docker` del CI ([run 44683044456](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44683044456/job/136111094290)), agregar `cmd/benchapi` subió `cc65-tf` a **`39,647,650 bytes`** frente al límite de `40,000,000 bytes` de `.github/workflows/ci.yml:109` (solo **352 KB** de margen), porque `deploy/Dockerfile:22` compila `./cmd/...` completo incluyendo herramientas de anfitrión (`benchapi`, `informe`, `word`, `benchmark`). No rompe este PR, pero el próximo comando que se añada en `cmd/` (p. ej. #32) probablemente fallará el CI si `Dockerfile` no se acota a los binarios del contenedor.
  - `cmd/benchapi/main.go:385-393` duplica `nombreArchivo` en vez de reutilizar `maquina.Host()` (`internal/maquina/maquina.go:25-33`) y no registra `maquina.Describir()` (modelo de CPU y versión de Go) como `cmd/benchmark` y `cmd/loader`.
  - En la descripción del PR #41, tras el commit `de04c90` (`reports/bench_api_jorge-pc.json`), sigue sin marcar la segunda casilla de criterios de aceptación y figura `PR-<n>` / `pendiente` en «Uso de IA».
- **Corrección sugerida:** No es necesario tocar `deploy/Dockerfile` en este PR (dejarlo como seguimiento inmediato antes de agregar más binarios en `cmd/`); opcionalmente reutilizar `internal/maquina.Host()` y actualizar la descripción del PR al responder esta ronda.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR #41. |

## Cómo explicarlo en la sustentación

Para verificar que la API responde en menos de 100 ms sin depender de pruebas manuales, `cmd/benchapi` lanza un *worker pool* configurable de clientes HTTP concurrentes (solo biblioteca estándar, cada goroutine acumula sus tiempos en su propio slice sin estado compartido) que consulta `GET /api/v1/modelo` para generar peticiones válidas y distintas en base mixta, separando una corrida fría (`X-Cache: MISS`, que evalúa el modelo y escribe en Redis) de una corrida caliente (`X-Cache: HIT`, servida directo desde Redis) y exigiendo `p95 < 100 ms` con `-exigir-p95 100ms`. Sobre el clúster real en Docker Compose (3 nodos + MongoDB + Redis + dataset gold real, 16 clientes, 2000 peticiones por corrida), el p95 medido fue **3,51 ms** en frío (7128 RPS) y **2,88 ms** en caliente (8809 RPS), con cero errores, y la colección de Postman (`docs/api/cc65-tf.postman_collection.json`) cubre las seis rutas documentadas con 24 aserciones automáticas de estado y tiempo < 100 ms verificadas con Newman.
