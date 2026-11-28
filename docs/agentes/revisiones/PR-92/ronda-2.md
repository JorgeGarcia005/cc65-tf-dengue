# Veredicto — PR-92 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/91-metricas-cluster` @ `0acb60f`
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

Los **4 hallazgos** de la ronda 1 (2 bloqueantes y 2 menores) quedaron resueltos con evidencia reproducible y sin regresiones:

- **Orden *happens-before* de la latencia del nodo (H1):** `Coordinador.despachar` (`internal/cluster/coordinador.go:465-474`) toma `inicioEnvio := time.Now()` **antes** de `c.enviarTarea(j, s)` y lo registra en `c.envioNodo[j]` únicamente cuando el envío tuvo éxito y es la primera tarea de la época para ese nodo (manteniendo `c.vence[s]` tras el envío). `TestMetricasPorNodoTrasEntrenar` pasa **100/100** corridas normales (`5,30 s`) y **200/200** corridas forzando contención extrema en un solo núcleo (`taskset -c 0 -cpu 1,2`, `9,93 s`, donde en la ronda 1 fallaba `47/400` veces), y los fragmentos de `docs/informe/pc4/08-distribuido.md` quedaron regenerados e idempotentes con `cmd/informe`.
- **Numeración unificada de épocas en la SPA (H2):** `conGradiente` (`web/src/admin/estado.ts:179-193`) convierte `const epoca = ev.epoca + 1`, guardando siempre la época en numeración 1-based como `MetricasNodo.epoca`; `conInstantanea` (`estado.ts:239-253`) conserva la vista en vivo cuando `previo.vista.epoca > (n.metricas?.epoca ?? 0)` y limpia `microsCpuEpoca: undefined` al adoptar una instantánea cerrada; y `reiniciarEntrenamiento` (`estado.ts:314-317`) reinicia las métricas por nodo antes de un nuevo entrenamiento. Cubierto con los tests actualizados y nuevos en `web/src/pruebas/metricasNodo.test.ts` (12/12) y `web/src/pruebas/admin.test.tsx` (16/16), dentro de los `183/183` tests en Vitest.
- **Distinción entre `us_cpu = 0` y ausencia de soporte de CPU (H3):** `MicrosCPU *int64` en `internal/cluster/protocolo/protocolo.go:111` y `internal/cluster/eventos.go:46` permite transmitir `0 µs` como dato válido en Unix (`0 %`) y `nil` cuando el sistema operativo no informa CPU (`—`), verificado en `TestMicrosCPUCeroViajaYNegativoSeRechaza` y en `metricasNodo.test.ts:36-39`.
- **Contrato OpenAPI completo (H4):** `docs/api/openapi.yaml:278-284, 484, 499` documenta `micros_calculo`, `micros_cpu`, `micros_ida_vuelta`, las dos convenciones de `epoca` y las listas `required` de `MetricasAPI` y `MetricasNodo`, validado por el job `OpenAPI 3 válido` y los tests de contrato en Go.

## Verificación automática

```text
$ git rev-parse --short HEAD
0acb60f

$ git diff --check origin/integracion/tb2...origin/feature/91-metricas-cluster
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ go run ./cmd/informe && git diff --exit-code docs/informe
(sin diferencias)

$ bash scripts/check-skills-sync.sh
skills: ok

$ GOOS=windows go build ./internal/cluster ./cmd/node && GOOS=darwin go build ./internal/cluster ./cmd/node
(compila limpio en Linux, Windows y macOS)

$ go test -count=100 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.304s

$ taskset -c 0 go test -count=100 -cpu 1,2 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.930s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (1.78s) [nodos=1, 2, 3, 5]
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.40s)
--- PASS: TestEquivalencia (0.88s) [workers=1, 2, 4, 8, 16, 64]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.841s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.870s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.628s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.034s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.612s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.270s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.274s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.416s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.396s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.057s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.162s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.062s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.343s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.733s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	10.299s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.136s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.055s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.078s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.095s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.401s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.764s

$ BENCH_CLAVE=... go run ./cmd/benchapi -url http://127.0.0.1:18089 -correo analista@vigia.pe -exigir-p95 100ms
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria         2000       0     6703     1.71     5.39     9.18    12.25      0      0  p95 < 100 ms: true
caliente     2000       0     7411     1.65     5.02     6.97    17.78      0      0  p95 < 100 ms: true
GET /api/v1/cluster → .api: {"predicciones":4048,"ventana":1024,"p50_ms":0.033689,"p95_ms":0.201179,"max_ms":5.960002,"limite_p95_ms":100,"cumple_limite_p95":true}

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm run build && ! grep -q "url(data:" dist/assets/*.css
Test Files  12 passed (12)
     Tests  183 passed (183)
✓ built in 1.23s · sin url(data: en CSS

$ gh pr checks 92 -R JorgeGarcia005/cc65-tf-dengue   # run 46098532452 sobre 0acb60f
Docker (imagen, clúster y caída de un nodo)	pass	1m56s
Ecosistema de agentes                      	pass	4s
Go (gofmt, vet, test -race)                	pass	1m25s
MongoDB y Redis (tests de integración)     	pass	59s
OpenAPI 3 válido                           	pass	21s
SPA (tipos, tests, build e imagen)         	pass	54s
Spin (modelo del protocolo y mutantes)     	pass	1m55s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`26/26` suites con tests en verde, carreras: 0; `300/300` corridas de estrés en `TestMetricasPorNodoTrasEntrenar`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` idéntico bit a bit) |
| Latencia API p95 < 100 ms | ✅ (`5,39 ms` fría / `5,02 ms` caliente en cliente; `0,20 ms` p95 interno en `/api/v1/cluster`) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (`183/183` tests en verde, sin `url(data:` en CSS) |
| CI GitHub Actions (PR #92 @ `0acb60f`) | ✅ (`7/7` jobs en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | No altera el planteamiento ni la partición de datos. |
| Implementación | 8 | 8.0 / 8 | Medición de CPU con solo stdlib (`syscall.Getrusage`), latencia por nodo con orden *happens-before* garantizado, percentiles `O(1)` al escribir en la API y equivalencia bit a bit intacta. |
| Interfaz | 5 | 5.0 / 5 | Completa el panel administrador exigido en el enunciado TB2 §6 (uso de CPU, latencia con desglose en cálculo y red, nodos sanos, bitácora y `p95` de la API) tanto por sondeo como en vivo por WebSocket. |
| Informe | 2 | 2.0 / 2 | Fragmentos extraídos en `docs/informe/pc4/08-distribuido.md` y `10-api.md` sincronizados e idempotentes con `cmd/informe`. |
| Video | 2 | listo para demo | Con las épocas alineadas en 1-based y el reinicio entre entrenamientos, las métricas por nodo se actualizan fluidamente durante la demostración en vivo. |

## Hallazgos

Sin hallazgos abiertos (**0 bloqueantes, 0 mayores, 0 menores**).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — `despachar` registraba `envioNodo[j]` después de `enviarTarea` · BLOQUEANTE | RESUELTO | Resuelto en `250c4ae` (`internal/cluster/coordinador.go:465-474`); verificado con `100` corridas normales y `200` en un solo núcleo (`taskset -c 0 -cpu 1,2`) sin fallos, e informe regenerado. |
| H2 (ronda 1) — Desfase *off-by-one* entre `EvGradiente.epoca` (`0-based`) y `MetricasNodo.epoca` (`1-based`) · BLOQUEANTE | RESUELTO | Resuelto en `c9ae564` (`web/src/admin/estado.ts:179-193, 239-253, 314-317`); cubierto en `metricasNodo.test.ts` y `admin.test.tsx` con la numeración real del coordinador. |
| H3 (ronda 1) — `MicrosCPU == 0` indistinguible de «sin soporte de CPU» · MENOR | RESUELTO | Resuelto en `cbd9147` y `c9ae564` mediante `MicrosCPU *int64` (`0` viaja como `0 %`, `nil` = sin dato), probado en `TestMicrosCPUCeroViajaYNegativoSeRechaza` y en Vitest. |
| H4 (ronda 1) — Campos nuevos de `/api/v1/ws` y `required` en `MetricasAPI` / `MetricasNodo` en OpenAPI · MENOR | RESUELTO | Resuelto en `aef97b2` (`docs/api/openapi.yaml:278-284, 484, 499`), validado en CI. |

## Cómo explicarlo en la sustentación

Cada nodo lee el tiempo de CPU acumulado de su proceso con `syscall.Getrusage(RUSAGE_SELF)` justo antes y después de calcular los parciales de su shard en el worker pool, y envía esa diferencia (`*int64`, donde `0` es dato y ausencia significa plataforma sin soporte) junto con el tiempo de cómputo puro en el mensaje `gradiente`, sin alterar los pesos ni el orden determinista de reducción. En el coordinador, el bucle de épocas toma la marca temporal antes de escribir la primera tarea de la época en el socket TCP y al recibir el último gradiente del nodo calcula la latencia de extremo a extremo, separándola en cómputo y red/JSON/cola, además del uso de CPU sobre la capacidad de sus `workers`. En paralelo, la API registra en `O(1)` la duración interna de las últimas `1024` predicciones en un búfer circular protegido por mutex y calcula `p50` y `p95` fuera de la sección crítica cuando el panel consulta `GET /api/v1/cluster`.
