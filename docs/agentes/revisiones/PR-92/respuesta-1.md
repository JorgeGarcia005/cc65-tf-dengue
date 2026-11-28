# Respuesta a la ronda 1 — PR #92

Implementador: Claude Code (Claude Opus 5.5). Revisado por Jorge.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `250c4ae` — la marca `inicioEnvio` se toma **antes** de `enviarTarea` y se guarda solo si el envío tuvo éxito; comentario de `MetricasNodo` y `docs/informe/pc4/08-distribuido.md` regenerados |
| H2 | BLOQUEANTE | ACEPTO | `c9ae564` — `conGradiente` usa `ev.epoca + 1`; la instantánea reinicia `microsCpuEpoca`; `reiniciarEntrenamiento` borra las métricas del entrenamiento anterior; tests con la numeración real del coordinador |
| H3 | MENOR | ACEPTO | `cbd9147` (Go) y `c9ae564` (SPA) — `MicrosCPU *int64` en el protocolo y en `Evento`; 0 es un dato, `nil` = sin dato; test `TestMicrosCPUCeroViajaYNegativoSeRechaza` |
| H4 | MENOR | ACEPTO | `aef97b2` — `/api/v1/ws` describe `micros_calculo`, `micros_cpu`, `micros_ida_vuelta` y las dos numeraciones de época; `required` en `MetricasAPI` y `MetricasNodo` |

## H1 — reproducción antes y después

Con un solo núcleo el defecto aparece siempre que el nodo responde antes de que vuelva `enviarTarea`:

```text
# antes (7ad8153)
$ taskset -c 0 go test -count=200 -cpu 1,2 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
47 corridas con «--- FAIL» de 400
# después (250c4ae)
$ taskset -c 0 go test -count=200 -cpu 1,2 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
ok
$ go test -race -count=20 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
ok
```

El plazo del shard (`c.vence[s]`) se sigue fijando después del envío, como antes. `max(lat − calc, 0)` en `cerrarMetricas` queda como defensa.

## H2 — numeración

- Go: `Evento.Epoca` = época en curso desde 0; `MetricasNodo.Epoca` = época cerrada desde 1 (ahora documentado en el struct y en OpenAPI).
- SPA: la vista guarda la segunda. El gradiente de la época E llega con `ev.epoca = E − 1` y se guarda como E; una instantánea anterior al cierre de E trae como mucho E − 1, así que no pisa la viva, y la de E ya cerrada no se suma con el gradiente de E + 1.
- Tests nuevos o corregidos en `metricasNodo.test.ts` (instantánea de la 7.ª + gradiente de la 8.ª con `epoca: 7` → reemplaza sin sumar y con CPU; instantánea recién cerrada no pisa la en curso; la CPU acumulada no cruza épocas; lanzar otro entrenamiento borra las métricas) y en `admin.test.tsx` (`nodo-0` con instantánea de la 7.ª y gradiente de la 8.ª). **Los 8 fallan con el `estado.ts` anterior** y pasan con el nuevo.

## Verificación

- `gofmt -l .` vacío; `go vet` y `go test -race` en verde en `internal/...` y `cmd/...` (aquí con los drivers sustituidos; el CI corre los reales).
- `cd web; npx tsc -b; npx vitest run` → 183/183; `npm run build`.
- `go test ./cmd/informe` (fragmentos del informe al día).
