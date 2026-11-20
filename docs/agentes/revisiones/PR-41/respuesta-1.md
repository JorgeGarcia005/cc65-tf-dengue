# Respuesta — PR #41 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-20 · **Base:** `feature/30-latencia-api` @ `cab634f`

Acepto los cinco hallazgos. Los tres casos de H1 quedaron como tests que fallaban con el código anterior.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MAYOR | ACEPTO | `6714fc6`: si la API tiene caché, `verificar` exige 0 HIT en la corrida fría y 100 % HIT en la caliente; `medir` rechaza `-n` o `-calientes` mayores que `combinaciones()/2` (las dos corridas usan mitades disjuntas). Tests: `TestCacheCaidaFallaLaCorridaCaliente` (caché que siempre falla → la API responde MISS), `TestSemillaRepetidaFallaLaCorridaFria` y `TestOpcionesInvalidasSeRechazan`. `TestSinCacheLoReporta` sigue igual. La evidencia ya medida (0/2000 y 2000/0) cumple las reglas nuevas |
| H2 | MAYOR | ACEPTO | `81eab02`: se quitó `\|\| true` de Newman en `scripts/evidencia-api.sh`; una aserción fallida hace fallar la evidencia (el `trap` sigue bajando Compose). Sin `npx`, el script lo avisa y no corre Newman |
| H3 | MENOR | ACEPTO | `6714fc6`: el cliente clona `http.DefaultTransport` con `IdleConnTimeout` y cierra las conexiones inactivas al terminar; `-timeout ≤ 0` y `-exigir-p95 < 0` se rechazan (`TestOpcionesInvalidasSeRechazan`) |
| H4 | MENOR | ACEPTO | `81eab02`: la «predicción nueva» usa una semana y una edad al azar (script *prerequest* en variables de colección) y exige `X-Cache: MISS`; la repetida usa las mismas variables y exige `HIT`. La descripción de la colección explica el 409 entre corridas seguidas y por qué el entrenamiento se queda en 600 épocas |
| H5 | MENOR | ACEPTO | `6714fc6`: el nombre de la máquina sale de `internal/maquina.Host()`. Sobre la imagen (99,1 % de 40 MB): queda anotado como seguimiento antes de sumar otro binario en `cmd/`; este PR no lo agranda |

## Verificación

```text
$ gofmt -l . ; go vet ./cmd/benchapi   → vacío
$ go test -race -count=1 ./cmd/benchapi
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	1.776s
```
