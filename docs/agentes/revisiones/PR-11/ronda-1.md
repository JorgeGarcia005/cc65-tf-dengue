# Veredicto — PR-11 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/7-rl-secuencial` @ `133662c`
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
--- PASS: TestGradienteNoDependeDelTamanoDeBloqueSalvoRedondeo (0.01s)
=== RUN   TestEntrenarRecuperaLosPesos
--- PASS: TestEntrenarRecuperaLosPesos (9.83s)
=== RUN   TestPesosBalanceados
--- PASS: TestPesosBalanceados (0.00s)
=== RUN   TestAUCCasosConocidos
--- PASS: TestAUCCasosConocidos (0.00s)
=== RUN   TestAUCRequiereAmbasClases
--- PASS: TestAUCRequiereAmbasClases (0.00s)
=== RUN   TestConfusionYMetricas
--- PASS: TestConfusionYMetricas (0.00s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	11.645s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (este PR define la línea base secuencial y las primitivas por bloque; `TestEquivalencia` corresponde al issue #8) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 (en curso) | Formaliza la pérdida ponderada por clase (`PesoPos`, `PesoNeg`, `PesosBalanceados`) y las métricas (`AUC`, `Recall`, `F1`) acordes al desbalance declarado en `ADR-0001`. |
| Implementación | 8 | Base sólida (en curso) | Núcleo 100 % stdlib (`math`, `slices`, `errors`, `fmt`), partición en bloques de límites fijos (`LimitesBloque`, `ParcialBloque`) y reducción determinista (`ReducirEnOrden`, `Finalizar`) verificada con diferencias finitas. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | Las ecuaciones de pérdida, gradiente ponderado y regularización L2 documentadas en `internal/ml/logistica.go` sirven de base directa para la sección de diseño del modelo distribuido (PC3). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H1 — `ReducirEnOrden` y `Probabilidad` sin test directo en esta rama · MENOR

- **Dónde:** `internal/ml/logistica.go:176-182` y `internal/ml/logistica.go:263`
- **Evidencia:** Al ejecutar `go test -coverprofile=cover.out ./...` y `go tool cover -func=cover.out`, el paquete reporta `93.3%` de cobertura total, pero `ReducirEnOrden` (`0.0%`) y `Probabilidad` (`0.0%`) no se invocan en `internal/ml/logistica_test.go` porque `Secuencial.Gradiente` (`internal/ml/logistica.go:211-218`) acumula directamente con `total.Acumular(&s.parcial)` dentro del bucle.
- **Corrección sugerida:** Añadir un test unitario que verifique que calcular `ParcialBloque` por separado y reducir con `ReducirEnOrden` + `Finalizar` entrega exactamente los mismos bits (`==`) que `Secuencial.Gradiente`, y usar `Probabilidad` en un test de evaluación tras `Entrenar`. No bloquea este PR dado que `ReducirEnOrden` se ejercita en `TestEquivalencia` del issue #8.

### H2 — Normalización ante pesos negativos y validación de longitudes en `Evaluar` · MENOR

- **Dónde:** `internal/ml/logistica.go:65-76` y `internal/ml/metricas.go:62-66`
- **Evidencia:** En `Config.normalizada()` (`internal/ml/logistica.go:69-74`), `TamBloque` valida `<= 0`, pero `PesoPos` y `PesoNeg` solo comprueban `== 0`; un peso negativo ingresado por error produciría `total.Peso <= 0` en `Finalizar` (`internal/ml/logistica.go:190-194`). Asimismo, `Evaluar(y, p, umbral)` indexa `p[i]` asumiendo `len(p) >= len(y)` sin validar `len(y) == len(p)` como sí hace `AUC(y, p)` en `internal/ml/metricas.go:21-23`.
- **Corrección sugerida:** Cambiar la guarda de `PesoPos` y `PesoNeg` en `Config.normalizada()` a `<= 0` y validar `len(y) == len(p)` al inicio de `Evaluar`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Ronda 1 (sin rondas previas). |

## Cómo explicarlo en la sustentación

Dividimos el dataset en bloques de tamaño fijo (`TamBloque = 4096`) cuyos límites `[ini, fin)` dependen únicamente del índice de bloque `k` y no del número de hilos ni de nodos. Cada bloque produce tres acumuladores sin normalizar en `ParcialBloque` —la suma de gradientes `Σ sᵢ(σ(zᵢ)−yᵢ)xᵢ`, la pérdida acumulada `Σ sᵢℓ(zᵢ,yᵢ)` y el peso efectivo `S = Σ sᵢ`—, que luego se suman siempre en orden de índice `0..B−1` (`ReducirEnOrden`) antes de dividir por `S` y aplicar la regularización L2 (`Finalizar`), garantizando que la versión secuencial, el worker pool y el clúster TCP den exactamente el mismo resultado IEEE-754 bit a bit.
