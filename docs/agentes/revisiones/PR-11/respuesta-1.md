# Respuesta — PR-11 · Ronda 1

- **Implementador:** Claude Code · modelo: Claude Opus 5.5
- **Responde a:** `ronda-1.md` (APROBADO, 2 hallazgos MENORES)
- **Commit de correcciones:** `5ca64f5`

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 — `ReducirEnOrden` y `Probabilidad` sin test directo | MENOR | ACEPTO | `5ca64f5` — `TestReducirEnOrdenIgualASecuencial` (bloques por separado + `ReducirEnOrden` + `Finalizar` = `Secuencial.Gradiente` bit a bit) y `TestProbabilidadTrasEntrenar` (rango [0,1], igual a σ(xᵀw+b), AUC ≥ 0,75 en sintético). Cobertura: `ReducirEnOrden` 100 %, `Probabilidad` 100 %, paquete 97,8 % |
| H2 — pesos negativos y largos en `Evaluar` | MENOR | ACEPTO con otra corrección | `5ca64f5` — ver nota |

## Nota sobre H2

Se aceptó el problema, pero no la corrección sugerida (cambiar `== 0` por `<= 0`). Con esa guarda, un
`PesoPos = -2` se convertiría en silencio en `1`: el entrenamiento seguiría y produciría otro modelo sin
avisar, que es peor que fallar. En su lugar:

- `Config.normalizada` mantiene `0 → 1` como valor por defecto, pero **entra en pánico** con un mensaje
  claro si `PesoPos`, `PesoNeg` o `Lambda` son negativos, NaN o infinitos (es un error de programación,
  como un índice fuera de rango). Test: `TestConfigInvalidaEntraEnPanico` con 5 casos.
- `Evaluar` ahora devuelve `(Metricas, error)` y rechaza `len(y) != len(p)`, igual que `AUC`.
  Test: `TestEvaluarRechazaLargosDistintos`.

## Verificación

```text
$ gofmt -l .
$ go vet ./...
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	10.186s
```
