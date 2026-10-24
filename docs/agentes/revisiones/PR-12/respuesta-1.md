# Respuesta — PR-12 · Ronda 1

- **Implementador:** Claude Code · modelo: Claude Opus 5.5
- **Responde a:** `ronda-1.md` (APROBADO, 2 hallazgos MENORES)
- **Commit de correcciones:** `48d3d69`

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 — `Pool.Gradiente` sin validar `len(w)` y sin test de llamadas concurrentes | MENOR | ACEPTO | `48d3d69` — `validarPesos` en `Secuencial.Gradiente` y `Pool.Gradiente` (simetría, como sugirió el juez). `TestPesosDeLargoIncorrectoEntranEnPanico` (D y D+2, ambos calculadores). `TestPoolLlamadasConcurrentes`: 8 goroutines × 5 llamadas al mismo `Pool` bajo `-race`, cada una bit a bit igual a su gradiente secuencial |
| H2 — cobertura de `Resumir` y `Prefijo`, validación de `frac` | MENOR | ACEPTO | `48d3d69` — `ValidarRecorte` (pánico fuera de [0, 0.5), incluido NaN) usada por `MediaRecortada`; `cmd/benchmark` valida `-recorte`, `-reps`, `-filas`, `-features` y `-bloque` al arrancar con `log.Fatal`. Tests: `TestResumir`, `TestRecorteInvalidoEntraEnPanico`, `TestPrefijo`, `TestSinteticoEsDeterminista` |

## Nota sobre H1

`TestPoolLlamadasConcurrentes` no solo comprueba que no haya carreras: cada llamador usa pesos distintos y
verifica su propio resultado. Si el mutex no serializara las épocas, un llamador podría leer `p.w` mientras
otro lo sobrescribe y obtendría el gradiente de otros pesos; el test lo detectaría incluso sin `-race`.

## Nota sobre H2

Con `frac < 0.5` se cumple `2·⌊n·frac⌋ < n` para todo `n ≥ 1`, así que siempre queda al menos un valor tras
recortar y la media nunca es 0/0. El test lo cubre con `frac = 0.49` sobre 3 valores.

## Verificación

```text
$ gofmt -l .
$ go vet ./...
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	9.080s
```

Cobertura: `Resumir` 100 %, `Prefijo` 100 %, ambos `Gradiente` 100 %; `internal/...` 96,8 %.
