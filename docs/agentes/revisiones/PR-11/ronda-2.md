# Veredicto — PR-11 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/7-rl-secuencial` @ `502e7d7`
- **Fecha:** 2026-10-24
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 -v ./...
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
--- PASS: TestEntrenarRecuperaLosPesos (5.51s)
=== RUN   TestPesosBalanceados
--- PASS: TestPesosBalanceados (0.00s)
=== RUN   TestReducirEnOrdenIgualASecuencial
--- PASS: TestReducirEnOrdenIgualASecuencial (0.01s)
=== RUN   TestProbabilidadTrasEntrenar
--- PASS: TestProbabilidadTrasEntrenar (1.08s)
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
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.280s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (15/15 PASS, carreras: 0, cobertura: 97,8 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestReducirEnOrdenIgualASecuencial` verifica igualdad bit a bit con `math.Float64bits` entre bloques reducidos en orden y `Secuencial.Gradiente`) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 (en curso) | Formaliza la pérdida ponderada por clase (`PesoPos`, `PesoNeg`, `PesosBalanceados`) y las métricas (`AUC`, `Recall`, `F1`) acordes al desbalance declarado en `ADR-0001`. |
| Implementación | 8 | Base sólida (en curso) | Núcleo 100 % stdlib, partición por bloques de límites fijos y reducción determinista verificada bit a bit (`TestReducirEnOrdenIgualASecuencial`) y por diferencias finitas (`TestGradienteDiferenciasFinitas`). |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | Las ecuaciones de pérdida, gradiente ponderado y regularización L2 documentadas en `internal/ml/logistica.go` sustentan la sección de diseño del modelo distribuido (PC3). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos en esta ronda.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — `ReducirEnOrden` y `Probabilidad` sin test directo | RESUELTO | Commit `5ca64f5` agrega `TestReducirEnOrdenIgualASecuencial` (igualdad bit a bit mediante `math.Float64bits`) y `TestProbabilidadTrasEntrenar`. Cobertura de ambas funciones al 100 % y del paquete al 97,8 %. |
| H2 (ronda 1) — Normalización ante pesos negativos y validación de longitudes en `Evaluar` | RESUELTO | Commit `5ca64f5`. Convence y se acepta la solución del implementador: entrar en pánico ante `PesoPos`, `PesoNeg` o `Lambda` negativos/NaN/Inf (`TestConfigInvalidaEntraEnPanico`) es mejor que silenciar el error reemplazando por `1`, y `Evaluar` ahora devuelve `(Metricas, error)` validando `len(y) == len(p)` (`TestEvaluarRechazaLargosDistintos`). |

## Cómo explicarlo en la sustentación

Dividimos el dataset en bloques de tamaño fijo (`TamBloque = 4096`) cuyos límites `[ini, fin)` dependen únicamente del índice de bloque `k` y no del número de hilos ni de nodos. Cada bloque produce tres acumuladores sin normalizar en `ParcialBloque` —la suma de gradientes `Σ sᵢ(σ(zᵢ)−yᵢ)xᵢ`, la pérdida acumulada `Σ sᵢℓ(zᵢ,yᵢ)` y el peso efectivo `S = Σ sᵢ`—, que luego se suman siempre en orden de índice `0..B−1` (`ReducirEnOrden`) antes de dividir por `S` y aplicar la regularización L2 (`Finalizar`), garantizando que la versión secuencial, el worker pool y el clúster TCP den exactamente el mismo resultado IEEE-754 bit a bit.
