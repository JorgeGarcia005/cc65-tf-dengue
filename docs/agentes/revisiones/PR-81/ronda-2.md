# Veredicto — PR-81 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/64-observador` @ `7eda05f` (base: `integracion/tb2` @ `0f9c0f5`)
- **Fecha:** 2026-11-22
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H4**) quedaron resueltos y verificados: `reports/bench_observador_jorge-pc_gold.json` (`a316576`) aporta la medición real del sobrecosto sobre `data/gold/train.bin` (765 297 filas × 31; 1, 3 y 5 nodos; las 6 series equivalentes bit a bit al secuencial; 3 792 eventos publicados, 0 descartados; sobrecosto −2,6 % / +1,1 % / +6,5 %); `Grabador` (`b4238dc`) elimina el pánico *send on closed channel* protegiendo envío y cierre con `sync.RWMutex` + `cerrado` y suma pruebas de concurrencia, idempotencia y error de escritura; `Coordinador.Terminar()` (`b4238dc`) garantiza la exclusión mutua con `EvAbortado` e idempotencia mediante `terminado` bajo `muBucle`; y `medirObservador` / `ejecutarObservador` / `proyeccion` (`4a5ee7e`, `b4238dc`) incorporan las guardas y la documentación solicitadas. Todas las verificaciones locales (`gofmt`, `go vet`, equivalencia bit a bit en `internal/cluster` e `internal/ml`, `go test -race -count=1 ./...` con 24/24 paquetes PASS y 0 carreras) y los 7/7 jobs del CI sobre `7eda05f` están en verde.

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

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/cluster ./cmd/benchmark
github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster -> [bufio context crypto/sha256 encoding/hex encoding/json errors fmt .../cluster/protocolo .../gold .../ml io math net os sync sync/atomic time]
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark -> [bytes encoding/json errors flag fmt .../cluster .../cluster/protocolo .../estadistica .../gold .../maquina .../ml .../ws log math math/rand/v2 net os path/filepath runtime slices strconv strings sync sync/atomic time]

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.88s)   # nodos=1, 2, 3, 5
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.19s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	2.097s
--- PASS: TestEquivalencia (0.39s)                           # workers=1, 2, 4, 8, 16, 64
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	1.419s

$ go test -race -count=2 -v -run 'Observador|Eventos|Grabador|Terminar' ./internal/cluster
PASS (2/2 repeticiones: TestSinFugasDeGoroutinesAlTerminarOAbortar, TestObservadorBitABit/{nodos=1,nodos=3},
TestEventosSecuenciaYContenido, TestEventosCaidaDeUnNodo, TestGrabadorReproducible, TestGrabadorNoBloquea,
TestGrabadorObservarYCerrarConcurrentes, TestGrabadorDevuelveElErrorDeEscritura, TestEventosAbortadoSinFin,
TestTerminarEmiteUnSoloFin)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	4.370s

$ go test -race -count=1 -v -run 'Observador' ./cmd/benchmark
--- PASS: TestBenchmarkObservadorPublicaYNoCambiaLosPesos (3.35s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	4.381s

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.485s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.843s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.781s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.773s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.339s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.252s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.518s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.056s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.216s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.030s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.117s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.094s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.728s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.664s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.050s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.245s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.033s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.089s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.636s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.642s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.807s

$ jq '{fuente, cpu: .maquina.cpu, nodos: .protocolo.nodos, bit_a_bit_todos: ([.sin_observador[].equivalente_bit_a_bit, .con_observador[].equivalente_bit_a_bit] | all), eventos: .eventos_publicados, descartados: .eventos_descartados, sobrecosto: [.sobrecosto[] | {nodos, sin_ms: (.ms_por_epoca_sin * 10 | round / 10), con_ms: (.ms_por_epoca_con * 10 | round / 10), pct: (.relativo * 1000 | round / 10)}]}' reports/bench_observador_jorge-pc_gold.json
{
  "fuente": "gold(data/gold/train.bin, filas=765297, features=31)",
  "cpu": "AMD Ryzen 5 3600 6-Core Processor",
  "nodos": [1, 3, 5],
  "bit_a_bit_todos": true,
  "eventos": 3792,
  "descartados": 0,
  "sobrecosto": [
    {"nodos": 1, "sin_ms": 99.3, "con_ms": 96.8, "pct": -2.6},
    {"nodos": 3, "sin_ms": 34.1, "con_ms": 34.5, "pct": 1.1},
    {"nodos": 5, "sin_ms": 22.5, "con_ms": 24, "pct": 6.5}
  ]
}

$ gh pr checks 81   # run 44979758073 sobre 7eda05f
Docker (imagen, clúster y caída de un nodo)	pass	1m59s
Ecosistema de agentes                       	pass	11s
Go (gofmt, vet, test -race)                 	pass	1m10s
MongoDB y Redis (tests de integración)      	pass	41s
OpenAPI 3 válido                            	pass	22s
SPA (tipos, tests, build e imagen)          	pass	46s
Spin (modelo del protocolo y mutantes)      	pass	1m55s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 24/24 paquetes con tests PASS, incluido `cmd/informe`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalenciaClusterSecuencialBitABit`, `TestObservadorBitABit`, `TestEquivalencia` y 6/6 series del JSON real) |
| Latencia API p95 < 100 ms | no aplica (no toca la API HTTP) |
| CI GitHub Actions sobre `7eda05f` | ✅ (7/7 jobs PASS, incl. Docker y Spin) |

## Verificación contra los criterios del issue #64

| Criterio (#64) | Estado | Evidencia |
|---|---|---|
| 1. **Bit a bit:** pesos con observador == pesos sin observador con 1 y 3 nodos | ✅ | `TestObservadorBitABit/{nodos=1,nodos=3}` PASS comparando `math.Float64bits` de pesos e historia, incluso mutando `e.Pesos` con `NaN` y `e.Nodos` con `"alterado"`. |
| 2. Eventos con esquema versionado: registro, tarea (asignación/envío de época), gradiente, epoca, reasignacion, descarte, abortado, fin | ✅ | `internal/cluster/eventos.go:12-40`, `TestEventosSecuenciaYContenido`, `TestEventosCaidaDeUnNodo`, `TestEventosAbortadoSinFin` y `TestTerminarEmiteUnSoloFin` PASS (nota explícita sobre `tarea` vs `asignacion` en PR y `respuesta-1.md`). |
| 3. Grabación a `.jsonl` reproducible (salvo marcas de tiempo/puertos/`micros_calculo`) | ✅ | `Grabador` en `internal/cluster/eventos.go:55-117`, `TestGrabadorReproducible`, `TestGrabadorNoBloquea`, `TestGrabadorObservarYCerrarConcurrentes` y `TestGrabadorDevuelveElErrorDeEscritura` PASS. |
| 4. Sobrecosto medido → `reports/bench_observador_<maquina>.json` | ✅ | `reports/bench_observador_jorge-pc_gold.json` versionado (gold real, 765 297 × 31, 1/3/5 nodos, 6/6 bit a bit, 3 792 eventos, 0 descartados). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Alineado con `ADR-0005`, `ADR-0007` y `AGENTS.md` §3.3 (determinismo bit a bit). |
| Implementación | 8 | Base de entrenamiento en vivo lista | Emisión serializada bajo `muBucle` con copias inmutables, `Grabador` no bloqueante y libre de pánicos bajo concurrencia, exclusión mutua `EvAbortado`/`EvFin` y sobrecosto cuantificado en el gold real. |
| Interfaz | 5 | Habilita #65 y #69 | Esquema versionado `v: 1` y nombres de eventos documentados listos para WebSocket y la vista en vivo. |
| Informe | 2 | Aporta fuente de cifras | `reports/bench_observador_jorge-pc_gold.json` versionado y `docs/informe/pc4/08-distribuido.md` sincronizado con el fuente (tag `pc4` @ `66465e1` intacto). |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `a316576` versiona `reports/bench_observador_jorge-pc_gold.json` (gold real 765 297 × 31, Ryzen 5 3600 de Jorge, 1/3/5 nodos × 5 reps × 30 épocas, 6/6 `equivalente_bit_a_bit: true`, 3 792 eventos, 0 descartados, sobrecosto −2,6 % / +1,1 % / +6,5 %) y marca la casilla en el PR. Nota: se generó con `9ea4363` (antes de los metadatos descriptivos adicionales de `protocolo` añadidos en `4a5ee7e`); el procedimiento de medición y los resultados son idénticos. |
| H2 (ronda 1 · MAYOR) | RESUELTO | `b4238dc` (`internal/cluster/eventos.go:60-117`, `internal/cluster/eventos_test.go:224-290`): `Grabador` protege `cerrado` y el envío/cierre de `cola` con `sync.RWMutex` (`Observar` bajo `RLock` contabiliza descarte si ya cerró; `Cerrar` cierra una sola vez bajo `Lock`, suelta antes de `<-g.hecho` y devuelve `g.err`). Cubierto por `TestGrabadorObservarYCerrarConcurrentes` (×50 con 4 goroutines + `Observar` tardío + `Cerrar` idempotente), `TestGrabadorDevuelveElErrorDeEscritura` y cierre limpio de `pw`/`<-lector` en `TestGrabadorNoBloquea`. |
| H3 (ronda 1 · MAYOR) | RESUELTO | `b4238dc` (`internal/cluster/coordinador.go:117, 577-588`, `internal/cluster/eventos_test.go:31-33, 294-335`): `Coordinador.terminado` bajo `muBucle` hace que `Terminar()` retorne sin efecto si `c.fallo != nil` o ya terminó. Cubierto por `TestEventosAbortadoSinFin`, `TestTerminarEmiteUnSoloFin` y mutación de `e.Nodos` en `recolector.observar`. |
| H4 (ronda 1 · MENOR) | RESUELTO | `4a5ee7e` + `b4238dc` (`cmd/benchmark/cluster.go:399-462`, `cmd/benchmark/main_test.go:153-162`, `internal/cluster/eventos_test.go:188-202`): `medirObservador` fuerza `op.observador = nil` en el control (probado en test), falla si algún `json.Marshal` falla o `a <= 0`, `ejecutarObservador` completa `protocolo`, `proyeccion` verifica `sc.Err()` y compara bits de `Pesos` + `Detalle` + campos deterministas de `Nodos`, y el uso de `tarea` (en lugar de `asignacion`) queda documentado en el PR y en `respuesta-1.md`. |

## Cómo explicarlo en la sustentación

Como el coordinador ya concentra todo el estado del entrenamiento en una sola goroutine protegida por `muBucle`, emite cada evento del protocolo (`registro`, `tarea`, `gradiente`, `epoca`, `reasignacion`, `descarte`, `abortado`, `fin`) en orden estricto y con esquema versionado `v: 1`. Para garantizar determinismo bit a bit (`AGENTS.md` §3.3), el observador solo recibe copias de `Pesos` y `Nodos` (probado con 1 y 3 nodos destruyendo lo recibido con `NaN`) y nunca bloquea al clúster: `Grabador` encola de forma no bloqueante en un canal con búfer protegido con `sync.RWMutex` para evitar pánicos al cerrar, y `Terminar()` respeta la exclusión mutua con `EvAbortado`. En el gold real (765 297 casos), publicar los eventos por `ws.Hub` añade un sobrecosto medido de +6,5 % con 5 nodos (y dentro del ruido con 1 y 3 nodos), con cero eventos descartados y pesos idénticos bit a bit al entrenamiento secuencial.
