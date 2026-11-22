# PR-77 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/59-evaluacion`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO (resuelto por Jorge) | `56dc101` — Jorge generó `reports/evaluacion.json` en su PC con `data/gold` real: dos corridas con el mismo SHA-256 `DDC12D9B49B5411A9A8ABB942F6200758629FEEDF3FEDEDA9EBAD22BEFD36FBF` (1,88 MB). `auc_trapecio` = 0,6220849410863555 contra `metricas_test.AUC` = 0,6220849410863315 de `reports/entrenamiento.json` (diferencia 2,4e-14 < 1e-6). Las 5 fuentes quedan registradas con su sha256 (test.bin, modelo.json, entrenamiento.json y los dos benchmarks). |
| H2 | BLOQUEANTE | ACEPTO (dependencia) | El CI no corre porque #74 (#56) no estaba fusionado. Tras fusionarlo se actualiza esta rama («Update branch») y corren los jobs. |
| H3 | MAYOR | ACEPTO | `75d3d31` — `Generar` exige `Test.D == Esquema.D()` y `len(Pesos) == D+1`, con un error que da las tres dimensiones; `Cruzar` valida dataset, esquema y largos de `y`/`pc`; `NuevosGrupos` valida esquema nulo y que las columnas one-hot quepan en `D`. `TestGenerarDimensionesDistintas` cubre D−2 y D+2 (antes: pánico y lectura de la fila siguiente), `pc` de otro largo y dataset nulo. |
| H4 | MAYOR | ACEPTO | `75d3d31` + `e9c2f46` — `Prioridades` devuelve siempre un slice no nulo (vacío si `n ≤ 0`): se serializa `[]`, no `null`. `cmd/evaluacion` rechaza `-top < 0` (y otros parámetros fuera de rango) y ya no indexa `Prioridades[0]` sin comprobar. Tests: `TestPrioridadesVaciasSeSerializanComoLista`, `TestEjecutarParametrosInvalidos`. |
| H5 | MAYOR | ACEPTO | `e9c2f46` — si se pasa `-entrenamiento`, `metricas_test.AUC` tiene que existir y estar en (0, 1]; si no, error (para no comparar: `-entrenamiento ""`). `TestEjecutarAUCReferencia` prueba AUC coincidente (5 fuentes), divergente, sin `metricas_test` (como `entrenamiento_cluster.json`), AUC 0 y JSON inválido. Cobertura de `cmd/evaluacion`: 47,4 % → 58,5 % (`main` sigue sin test; `ejecutar` cubre la lógica). |
| H6 | MENOR | ACEPTO | `1348021` — comentario de `Calibracion` con el reparto real (⌊b·n/k⌋); `ROC` rechaza puntajes NaN o infinitos (antes: bucle infinito); `Brier` vacío = 0. `75d3d31`: `Cruzar` no produce NaN si el riesgo nacional es 0. `TestGuardasNumericas` y `TestCruzarRiesgoNacionalCero`. El marcador `PR-<n>` de la descripción se corrige a `PR-77`. |

## ¿Cambia `reports/evaluacion.json`?

No. Con entradas válidas las cuentas son las mismas (`proporcion(a, b)` = `a / b` cuando `b ≠ 0`, en el mismo orden de operaciones) y las prioridades ya eran 15. Para confirmarlo, Jorge vuelve a correr `go run ./cmd/evaluacion` y comprueba el mismo SHA-256 de arriba.

## Verificación

```text
$ gofmt -l internal/evaluacion cmd/evaluacion     (vacío)
$ go vet ./internal/evaluacion ./cmd/evaluacion   (vacío)
$ go test -race -count=1 -cover ./internal/evaluacion ./cmd/evaluacion
ok   .../internal/evaluacion   coverage: 86.6% of statements
ok   .../cmd/evaluacion        coverage: 58.5% of statements
```
