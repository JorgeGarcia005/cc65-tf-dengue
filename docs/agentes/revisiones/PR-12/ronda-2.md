# Veredicto — PR-12 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/8-gradiente-concurrente` @ `4211458`
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
=== RUN   TestResumir
--- PASS: TestResumir (0.01s)
=== RUN   TestRecorteInvalidoEntraEnPanico
--- PASS: TestRecorteInvalidoEntraEnPanico (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.818s
=== RUN   TestEquivalencia
=== RUN   TestEquivalencia/workers=1
=== RUN   TestEquivalencia/workers=2
=== RUN   TestEquivalencia/workers=4
=== RUN   TestEquivalencia/workers=8
=== RUN   TestEquivalencia/workers=16
=== RUN   TestEquivalencia/workers=64
--- PASS: TestEquivalencia (0.17s)
    --- PASS: TestEquivalencia/workers=1 (0.04s)
    --- PASS: TestEquivalencia/workers=2 (0.02s)
    --- PASS: TestEquivalencia/workers=4 (0.01s)
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
=== RUN   TestPoolLlamadasConcurrentes
--- PASS: TestPoolLlamadasConcurrentes (0.03s)
=== RUN   TestPesosDeLargoIncorrectoEntranEnPanico
--- PASS: TestPesosDeLargoIncorrectoEntranEnPanico (0.00s)
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
--- PASS: TestGradienteNoDependeDelTamanoDeBloqueSalvoRedondeo (0.01s)
=== RUN   TestEntrenarRecuperaLosPesos
--- PASS: TestEntrenarRecuperaLosPesos (6.59s)
=== RUN   TestPesosBalanceados
--- PASS: TestPesosBalanceados (0.00s)
=== RUN   TestReducirEnOrdenIgualASecuencial
--- PASS: TestReducirEnOrdenIgualASecuencial (0.01s)
=== RUN   TestProbabilidadTrasEntrenar
--- PASS: TestProbabilidadTrasEntrenar (1.33s)
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
=== RUN   TestPrefijo
--- PASS: TestPrefijo (0.00s)
=== RUN   TestSinteticoEsDeterminista
--- PASS: TestSinteticoEsDeterminista (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	9.944s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (31/31 PASS, carreras: 0, cobertura total: 96,9 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia` con 1, 2, 4, 8, 16 y 64 workers + `TestPoolLlamadasConcurrentes` con 8 goroutines concurrentes verifican igualdad bit a bit mediante `math.Float64bits`) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 (en curso) | Mantiene intacta la formulación matemática ponderada y el particionamiento determinista por bloques fijos (`TamBloque = 4096`). |
| Implementación | 8 | Avance clave de PC3 (en curso) | Worker pool persistente (`Pool`) 100 % stdlib (`sync`, `math`, `slices`), libre de carreras (`-race`), sin *false sharing*, validación simétrica de dimensiones (`validarPesos`), equivalencia bit a bit demostrada y benchmark reproducible (`cmd/benchmark`) con speedup `1.45x` (2 workers) y `1.64x` (4 workers). |
| Interfaz | 5 | no aplica | No toca la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | Registra `ADR-0002` con alternativas descartadas y genera `reports/bench_gradiente_ci-2vcpu.json` con media recortada al 10 %, IC 95 % bootstrap y punto de equilibrio (`10 000` filas), cumpliendo `AGENTS.md` §3.6. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos en esta ronda.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — `Pool.Gradiente` no valida `len(w) == ds.D+1` y falta test de llamadas concurrentes a `Gradiente` | RESUELTO | Commit `48d3d69` incorpora `validarPesos(w, d)` tanto en `Pool.Gradiente` como en `Secuencial.Gradiente` (`TestPesosDeLargoIncorrectoEntranEnPanico`), y añade `TestPoolLlamadasConcurrentes` (8 goroutines × 5 llamadas simultáneas con pesos distintos bajo `-race`, comprobando igualdad bit a bit frente a `Secuencial`). |
| H2 (ronda 1) — Cobertura de `estadistica.Resumir`, validación de `frac` en `MediaRecortada` y test de `Dataset.Prefijo` | RESUELTO | Commit `48d3d69` añade `ValidarRecorte(frac)` en `[0, 0.5)` (incluido `NaN`), valida los flags de entrada en `cmd/benchmark/main.go:82-87` y suma `TestResumir`, `TestRecorteInvalidoEntraEnPanico`, `TestPrefijo` y `TestSinteticoEsDeterminista`. Cobertura de `Resumir` y `Prefijo` al 100 % y de `internal/...` al 96,9 %. |

## Cómo explicarlo en la sustentación

En lugar de crear goroutines en cada época o usar un mutex sobre un acumulador global (lo que serializaría el cómputo y rompería el determinismo del punto flotante), creamos un *worker pool* persistente donde un canal sin búfer reparte los índices de bloque `0..B−1` (*fan-out*). Cada worker acumula su bloque en un búfer privado `local` —evitando así el *false sharing* entre líneas de caché de bloques vecinos— y solo al terminar el bloque copia el resultado a su slot exclusivo `parciales[k]` antes de señalar `wg.Done()`. Tras la barrera `wg.Wait()`, el hilo principal reduce `parciales[0..B−1]` en orden estricto de índice (`ReducirEnOrden`), logrando un *speedup* de `1.45x`–`1.64x` en 2 vCPU con un resultado IEEE-754 idéntico bit a bit al secuencial para cualquier cantidad de workers.
