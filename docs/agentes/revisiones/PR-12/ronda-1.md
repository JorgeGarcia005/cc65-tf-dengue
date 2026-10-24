# Veredicto — PR-12 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/8-gradiente-concurrente` @ `6642a95`
- **Fecha:** 2026-10-24
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 -v ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
=== RUN   TestMediaRecortadaDescartaAtipicos
--- PASS: TestMediaRecortadaDescartaAtipicos (0.00s)
=== RUN   TestMediaRecortadaNoModificaLaEntrada
--- PASS: TestMediaRecortadaNoModificaLaEntrada (0.00s)
=== RUN   TestMedianaYDesviacion
--- PASS: TestMedianaYDesviacion (0.00s)
=== RUN   TestICBootstrapContieneLaEstimacionYEsDeterminista
--- PASS: TestICBootstrapContieneLaEstimacionYEsDeterminista (0.01s)
=== RUN   TestResumirVacio
--- PASS: TestResumirVacio (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.811s
=== RUN   TestEquivalencia
=== RUN   TestEquivalencia/workers=1
=== RUN   TestEquivalencia/workers=2
=== RUN   TestEquivalencia/workers=4
=== RUN   TestEquivalencia/workers=8
=== RUN   TestEquivalencia/workers=16
=== RUN   TestEquivalencia/workers=64
--- PASS: TestEquivalencia (0.19s)
    --- PASS: TestEquivalencia/workers=1 (0.05s)
    --- PASS: TestEquivalencia/workers=2 (0.03s)
    --- PASS: TestEquivalencia/workers=4 (0.02s)
    --- PASS: TestEquivalencia/workers=8 (0.02s)
    --- PASS: TestEquivalencia/workers=16 (0.02s)
    --- PASS: TestEquivalencia/workers=64 (0.02s)
=== RUN   TestPoolCopiaLosPesos
--- PASS: TestPoolCopiaLosPesos (0.00s)
=== RUN   TestPoolCerrarEsIdempotente
--- PASS: TestPoolCerrarEsIdempotente (0.00s)
=== RUN   TestPoolUsarDespuesDeCerrarEntraEnPanico
--- PASS: TestPoolUsarDespuesDeCerrarEntraEnPanico (0.00s)
=== RUN   TestWorkersInvalidosSeAjustan
--- PASS: TestWorkersInvalidosSeAjustan (0.00s)
=== RUN   TestSigmoideEstable
--- PASS: TestSigmoideEstable (0.00s)
=== RUN   TestPerdidaLogisticaEstable
--- PASS: TestPerdidaLogisticaEstable (0.00s)
=== RUN   TestNuevoDatasetValida
--- PASS: TestNuevoDatasetValida (0.00s)
=== RUN   TestBloquesCubrenTodoSinSolaparse
--- PASS: TestBloquesCubrenTodoSinSolaparse (0.00s)
=== RUN   TestGradienteDiferenciasFinitas
--- PASS: TestGradienteDiferenciasFinitas (0.01s)
=== RUN   TestGradienteNoDependeDelTamanoDeBloqueSalvoRedondeo
--- PASS: TestGradienteNoDependeDelTamanoDeBloqueSalvoRedondeo (0.00s)
=== RUN   TestEntrenarRecuperaLosPesos
--- PASS: TestEntrenarRecuperaLosPesos (5.71s)
=== RUN   TestPesosBalanceados
--- PASS: TestPesosBalanceados (0.00s)
=== RUN   TestReducirEnOrdenIgualASecuencial
--- PASS: TestReducirEnOrdenIgualASecuencial (0.01s)
=== RUN   TestProbabilidadTrasEntrenar
--- PASS: TestProbabilidadTrasEntrenar (0.86s)
=== RUN   TestConfigInvalidaEntraEnPanico
--- PASS: TestConfigInvalidaEntraEnPanico (0.00s)
=== RUN   TestAUCCasosConocidos
--- PASS: TestAUCCasosConocidos (0.00s)
=== RUN   TestAUCRequiereAmbasClases
--- PASS: TestAUCRequiereAmbasClases (0.00s)
=== RUN   TestConfusionYMetricas
--- PASS: TestConfusionYMetricas (0.00s)
=== RUN   TestEvaluarRechazaLargosDistintos
--- PASS: TestEvaluarRechazaLargosDistintos (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.601s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (25/25 PASS, carreras: 0, cobertura total: 93,2 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia` verifica igualdad bit a bit con `math.Float64bits` en gradiente, pérdida, pesos tras 10 épocas e historia de pérdida para 1, 2, 4, 8, 16 y 64 workers) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 (en curso) | Mantiene intacta la formulación matemática ponderada y el particionamiento determinista por bloques fijos (`TamBloque = 4096`). |
| Implementación | 8 | Avance clave de PC3 (en curso) | Worker pool persistente (`Pool`) 100 % stdlib (`sync`, `math`, `slices`), libre de carreras (`-race`), sin *false sharing* (acumulador privado `local` por worker + variables locales en `ParcialBloque`), equivalencia bit a bit demostrada y benchmark reproducible (`cmd/benchmark`) con speedup `1.45x` (2 workers) y `1.64x` (4 workers) en `reports/bench_gradiente_ci-2vcpu.json`. |
| Interfaz | 5 | no aplica | No toca la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | Registra `ADR-0002` con alternativas descartadas (incluyendo evidencia empírica de *false sharing*) y genera `reports/bench_gradiente_ci-2vcpu.json` con media recortada al 10 %, IC 95 % bootstrap y punto de equilibrio (`10 000` filas), cumpliendo la regla de cero cifras a mano (`AGENTS.md` §3.6). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H1 — `Pool.Gradiente` no valida `len(w) == ds.D+1` y falta test de llamadas concurrentes a `Gradiente` · MENOR

- **Dónde:** `internal/ml/concurrente.go:83-95` y `internal/ml/concurrente_test.go`
- **Evidencia:** En `Pool.Gradiente(w)`, `copy(p.w, w)` (`internal/ml/concurrente.go:89`) copia `min(len(p.w), len(w))` elementos sin verificar que `len(w) == len(p.w)`. A diferencia de `Secuencial.Gradiente(w)` —que entra en pánico en `Puntaje` o `Finalizar` si `len(w) != ds.D+1`—, `Pool.Gradiente(w)` pasa `p.w` a `ParcialBloque` y a `Finalizar` (`internal/ml/concurrente.go:73, 95`), por lo que si un llamador pasa un `w` de longitud errónea reutiliza en silencio los restos de `p.w` de la época anterior. Además, `Pool` declara `mu sync.Mutex` (`internal/ml/concurrente.go:32, 42, 84`) para serializar llamadas concurrentes a `Gradiente`, pero `internal/ml/concurrente_test.go` no incluye un test con dos o más goroutines invocando `p.Gradiente` simultáneamente bajo `-race`.
- **Corrección sugerida:** Añadir `if len(w) != len(p.w) { panic(...) }` al inicio de `Pool.Gradiente` (y opcionalmente en `Secuencial.Gradiente` para simetría) y agregar un test corto en `internal/ml/concurrente_test.go` que invoque `p.Gradiente` desde varias goroutines en paralelo verificando que `p.mu` serializa sin carreras y entrega resultados bit a bit idénticos.

### H2 — Cobertura de `estadistica.Resumir` (caso no vacío), validación de `frac` en `MediaRecortada` y test de `Dataset.Prefijo` · MENOR

- **Dónde:** `internal/estadistica/estadistica.go:26-39, 103-115`, `internal/estadistica/estadistica_test.go:52-56` y `internal/ml/sintetico.go:33-36`
- **Evidencia:** Al ejecutar `go test -coverprofile=cover.out ./internal/... && go tool cover -func=cover.out`, `estadistica.Resumir` reporta `30.0%` de cobertura porque `TestResumirVacio` solo prueba `Resumir(nil, 0.1)` y no una muestra no vacía (líneas `108-114`), y `Dataset.Prefijo` reporta `0.0%` en `internal/ml`. Asimismo, `MediaRecortada(xs, frac)` documenta `frac debe estar en [0, 0.5)`, pero si se invoca con `frac < 0` o `frac >= 0.5` (por ejemplo, `-recorte 0.5` en `cmd/benchmark/main.go:75`) devuelve `NaN` o entra en pánico por índice de slice sin un mensaje descriptivo.
- **Corrección sugerida:** Validar `0 <= frac && frac < 0.5` en `MediaRecortada` (entrando en pánico con mensaje claro, igual que `Config.normalizada()`), ampliar `TestResumirVacio` (o añadir `TestResumir`) con una muestra no vacía y agregar un test unitario breve para `Dataset.Prefijo`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Ronda 1 (sin rondas previas). |

## Cómo explicarlo en la sustentación

En lugar de crear goroutines en cada época o usar un mutex sobre un acumulador global (lo que serializaría el cómputo y rompería el determinismo del punto flotante), creamos un *worker pool* persistente donde un canal sin búfer reparte los índices de bloque `0..B−1` (*fan-out*). Cada worker acumula su bloque en un búfer privado `local` —evitando así el *false sharing* entre líneas de caché de bloques vecinos— y solo al terminar el bloque copia el resultado a su slot exclusivo `parciales[k]` antes de señalar `wg.Done()`. Tras la barrera `wg.Wait()`, el hilo principal reduce `parciales[0..B−1]` en orden estricto de índice (`ReducirEnOrden`), logrando un *speedup* de `1.45x`–`1.64x` en 2 vCPU con un resultado IEEE-754 idéntico bit a bit al secuencial para cualquier cantidad de workers.
