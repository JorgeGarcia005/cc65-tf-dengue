# Veredicto — PR-87 · Ronda 3

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/69-admin-vivo` @ `1c3a1bf` (base: `integracion/tb2` @ `c6b75ab`)
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

Tras el `APROBADO` de la ronda 2 (`81acbfb` sobre `3c5c34e`), la revisión humana del equipo incorporó el commit `1c3a1bf` (`feat(web,coordinator): mapa en vivo y grabación de eventos para la réplica (#69)`), completando de punta a punta los dos detalles restantes de los criterios del issue #69 («mapa repintándose» durante el entrenamiento y grabación real de `eventos_*.jsonl` en el coordinador para el modo réplica):

- **Mapa en vivo (`web/src/admin/mapaVivo.ts`, `web/src/admin/MapaVivo.tsx`, `web/src/paginas/Admin.tsx:408-423`):** dado que el departamento entra en one-hot aditivo en el logit y solo queda identificado salvo traslación común absorbida por el sesgo, `efectosDepartamento` centra cada peso en el promedio de las columnas `depto_*` (`Δ_d = w_d − media(w_k)`, `OR = e^Δ_d`), usa `depto_otros` (`agrupado: true`) para los 5 departamentos sin columna propia en el esquema entrenado y aplica `PISO_ESCALA = 0,05` para evitar que la escala colapse cuando todos los pesos parten en cero. Los 25 `NOMBDEP` de `peru_departamental_simple.geojson` coinciden exactamente con la normalización de `internal/features/features.go:91`, se pinta solo con atributos SVG (`fill`, `d`, sin estilos inline para cumplir `style-src 'self'`), parte del modelo publicado antes de la primera época y se repinta con los pesos de cada `EvEpoca` (`mapaVivo.test.ts`: 7/7 tests; `admin.test.tsx:120-145`).
- **Grabación no bloqueante de `eventos_*.jsonl` (`cmd/coordinator/eventos.go`, `cmd/coordinator/api.go:42-50`, `cmd/coordinator/main.go:111,123-126,137-138,155`, `deploy/docker-compose.api.yml:31-33`):** `abrirEventos` conecta `cluster.NuevoGrabador(f, 4096)` al observador del coordinador (junto con `hub.Publicar` en modo `-api`), sin bloquear el bucle de épocas (`Observar` usa `RWMutex` + canal con búfer y cuenta descartes con `atomic.Int64`). Al detener el coordinador (`SIGTERM` o fin de corrida), `cerrarEventos()` vacía la cola tras `c.Cerrar()`, vuelca `bufio.Writer` y cierra el archivo de forma idempotente (`cmd/coordinator/eventos_test.go`, `cmd/coordinator/demo_test.go:100-108,167-170`).
- **Verificación en vivo de punta a punta:** además de toda la suite con `-race` y Vitest, se levantó localmente `coordinator -demo -eventos` con 3 procesos `cmd/node`, se ejecutó `cmd/benchapi -exigir-p95 100ms` (`p95 = 5,18 ms` fría y `4,17 ms` caliente, 0 errores), se lanzó un entrenamiento de 15 épocas cortando `nodo-1` en caliente (`POST /api/v1/demo/nodos/nodo-1/cortar`) y se cerró con `SIGTERM`: el archivo `eventos.jsonl` grabó los `112` eventos (`4 registro`, `45 tarea`, `45 gradiente`, `15 epoca`, `1 descarte`, `1 reasignacion`, `1 fin`) con `0` descartados.

## Verificación automática

```text
$ git rev-parse --short origin/feature/69-admin-vivo
1c3a1bf

$ git diff --check origin/integracion/tb2...origin/feature/69-admin-vivo
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	19.145s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	13.842s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	8.011s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.072s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.679s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.521s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.352s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.424s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	4.196s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.108s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.248s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.061s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	42.736s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	21.810s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	14.606s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.400s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.045s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.004s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	9.949s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	15.817s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.762s

$ go test -race -count=1 -run "Equivalencia|TestServicioDemoCortarNodoPorLaAPI" -v ./internal/ml ./internal/cluster ./cmd/coordinator
--- PASS: TestEquivalencia (0.97s)
--- PASS: TestEquivalenciaClusterSecuencialBitABit (1.85s)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.34s)
--- PASS: TestServicioDemoCortarNodoPorLaAPI (1.44s)

$ BENCH_CLAVE=... go run ./cmd/benchapi -url http://127.0.0.1:18089 -correo analista@vigia.pe -exigir-p95 100ms -salida /tmp/juez87/bench_api.json
API http://127.0.0.1:18089 · modelo ffbc0384af31 · 16 clientes · linux/amd64
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria         2000       0     7180     1.54     5.18     8.40    15.75      0      0  p95 < 100 ms: true
caliente     2000       0     8407     1.39     4.17     6.55     9.23      0      0  p95 < 100 ms: true

$ <prueba en vivo de coordinator -demo -eventos /tmp/juez87/eventos.jsonl + 3 cmd/node + corte de nodo-1 + SIGTERM>
{"estado":"cortado","nodo":"nodo-1"}
entrenamiento estado=terminado
eventos grabados en /tmp/juez87/eventos.jsonl (0 descartados)
eventos grabados: 112 {'registro': 4, 'tarea': 45, 'gradiente': 45, 'epoca': 15, 'descarte': 1, 'reasignacion': 1, 'fin': 1}

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm test -- ws && npm test -- replica && npm test -- admin && npm test -- mapaVivo && npm run build && (grep -F "url(data:" dist/assets/*.css || echo "SIN_URL_DATA")
Test Files  10 passed (10)
     Tests  134 passed (134)
ws.test.ts       -> 10 passed (10)
replica.test.ts  ->  8 passed (8)
admin.test.tsx   -> 15 passed (15)
mapaVivo.test.ts ->  7 passed (7)
✓ built in 1.35s · SIN_URL_DATA

$ gh pr checks 87 -R JorgeGarcia005/cc65-tf-dengue   # run 46068097394 sobre 1c3a1bf
Docker (imagen, clúster y caída de un nodo)	pass	1m55s
Ecosistema de agentes                      	pass	8s
Go (gofmt, vet, test -race)                	pass	1m19s
MongoDB y Redis (tests de integración)     	pass	56s
OpenAPI 3 válido                           	pass	19s
SPA (tipos, tests, build e imagen)         	pass	50s
Spin (modelo del protocolo y mutantes)     	pass	1m55s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes con tests en verde en Linux) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia`, `TestEquivalenciaClusterSecuencialBitABit`, `TestEquivalenciaConMenosNodosQueShards` y `TestServicioDemoCortarNodoPorLaAPI` en verde con `-race`) |
| Latencia API p95 < 100 ms | ✅ (`p95 = 5,18 ms` fría / `4,17 ms` caliente con JWT y 16 clientes concurrentes; 0 errores) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (134/134 tests en 10 archivos, 0 `url(data:` en CSS) |
| CI GitHub Actions (PR #87 @ `1c3a1bf`) | ✅ (7/7 jobs PASS) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Conecta la optimización distribuida en tiempo real con el impacto territorial por departamento (`OR = e^Δ` centrado frente al promedio departamental) y la tolerancia a fallos determinista. |
| Implementación | 8 | 8.0 / 8 | `EvModelo` y `-eventos` (`EVENTOS` en Compose) implementados solo con stdlib, sin bloquear al bucle de épocas (`ws.Hub` + `cluster.Grabador`), cierre idempotente antes de salir, 0 carreras y equivalencia bit a bit con corte de nodo. |
| Interfaz | 5 | 5.0 / 5 | Cubre todos los criterios de #69: lanzamiento de entrenamiento con pérdida, pesos, topología, bitácora y mapa repintándose por época, reconexión exponencial con ticket nuevo, modo réplica desde `eventos_*.jsonl` grabado por el coordinador y corte de nodo con sello `bit a bit ✓`. |
| Informe | 2 | — | No corresponde a este PR (`docs/api/openapi.yaml` actualizado y validado en CI). |
| Video | 2 | 2.0 / 2 | Deja listos tanto el flujo en vivo con caída de nodo y mapa repintándose como el respaldo determinista por `EVENTOS=/reports/eventos_demo.jsonl` para la demostración y sustentación. |

## Hallazgos

Sin hallazgos abiertos (**0 bloqueantes, 0 mayores, 0 menores**).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — Réplica disparaba `GET /api/v1/modelo` y heredaba `entrenamiento` previo · MAYOR | RESUELTO | Cerrado en `4b91f27` (verificado en ronda 2 y mantenido intacto en `1c3a1bf`). |
| H2 (ronda 1) — Bitácora mezclaba horas UTC y hora local (±5 h) · MENOR | RESUELTO | Cerrado en `4b91f27` (`horaBitacora`); mantenido intacto en `1c3a1bf`. |
| H3.1–H3.4 (ronda 1) — Endurecimiento en `ws.ts`, `Graficos.tsx`, `Admin.tsx` y `estado.ts` · MENOR | RESUELTO | Cerrados en `4b91f27`; mantenidos intactos en `1c3a1bf`. |

## Cómo explicarlo en la sustentación

> El panel de administración se conecta al coordinador mediante un WebSocket autenticado con ticket efímero de un solo uso y reconexión exponencial (`0,5 s · 2^k` hasta `30 s` + jitter) pidiendo siempre un ticket nuevo. Cada evento (`registro`, `epoca`, `descarte`, `reasignacion`, `modelo`, `fin`) pasa por la función pura `reducir(estado, evento, origen)` —la misma que alimenta el modo réplica desde un archivo `eventos_*.jsonl` grabado sin bloqueo por `cluster.Grabador` (`-eventos` / `EVENTOS`)— para actualizar en tiempo real la topología en estrella, la curva de pérdida, los 32 pesos, la bitácora y el mapa departamental de razones de odds (`e^Δ`, con `Δ = w_depto − media` independiente de edad y semana). Si se tumba un nodo en vivo (`DEMO=true`), el coordinador reasigna sus shards conservando la reducción ordenada por índice de bloque, la API anuncia los pesos finales en `EvModelo` y el panel comprueba igualdad exacta (`===`) contra `GET /api/v1/modelo`, respaldada por la equivalencia bit a bit frente al entrenamiento secuencial en `TestServicioDemoCortarNodoPorLaAPI`.
