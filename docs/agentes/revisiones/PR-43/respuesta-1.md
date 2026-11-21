# Respuesta — PR #43 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-20 · **Base:** `feature/31-spin-adr5` @ `7fd1f3e`

Acepto los cinco hallazgos. Los reproduje antes de corregir: la rama `full` sin alcanzar con
capacidad N, el envío a un nodo descartado al abrir la época 1 y el `[ "" != 0 ]` de `verificar.sh`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `040522d` + `026b75e`: variable fantasma `saturado[N]`, puesta en las dos ramas `full(aNodo[j])`; las aserciones pasan a `!vivo \|\| sospechado \|\| saturado`. La corrida con capacidad 1 ya **no usa `-A`**: recorre la rama `full` con todas las aserciones activas (26 025 432 estados, 0 errores; `pan-seg` solo deja sin alcanzar el `else` de `DESCARTAR` que la guarda previa hace imposible) |
| H2 | BLOQUEANTE | ACEPTO | `040522d`: al difundir la época, un shard cuyo dueño está descartado no se envía (`:: descartado[j] -> skip`) y lo reasigna `cubrir()`, como `coordinador.go`. Cada envío de `EPOCA` va precedido de `assert(!descartado[j])`, la aserción con la que el juez encontró el contraejemplo; ahora el modelo pasa con 0 errores |
| H3 | MAYOR | ACEPTO | `040522d`: `N`, `EPOCAS`, `CAIDAS` y `SOSPECHAS` se pueden fijar con `-D` y las guardas del shard 3 entran con `#if N > 3`. Nueva aserción de balanceo en la barrera: entre nodos sanos, `carga[k] <= carga[j] + 1`. `verificar.sh` corre N = 4, 1 época, sin sospechas (4 923 889 estados, 0 errores) y un mutante nuevo, `MUT_PRIMER_SANO`, que la rompe (1 error en 5 269 estados). El README explica que con N = 3 la menor carga solo decide por desempate |
| H4 | MAYOR | ACEPTO | `026b75e`: `errores` debe ser entero (si no, falla); con esperado 0, `Search not completed`, `max search depth too small` u `out of memory` hacen fallar; los mutantes se evalúan con `[ "$errores" -gt 0 ]` |
| H5 | MENOR | ACEPTO | `0f73e04`: el README describe el contraejemplo mínimo real del plazo global (70 estados, sin caídas: una sospecha falsa sobre el shard 0 recién difundido y el plazo global descarta al dueño vivo del shard 1) y lo separa de la motivación en el sistema real (test 3 del ADR-0005). `026b75e`: `verificar.sh` escribe `reports/spin.json`, que usará `cmd/informe` en #33 |

## Verificación

```text
$ bash spin/verificar.sh          (Spin 6.5.2, gcc 13.3, 1 min 57 s)
modelo N=3 (canal de capacidad N)                seguridad      17949492       0 0
modelo N=3 (canal de capacidad N)                termina        17814453       0 0
modelo N=3, canal de capacidad 1                 seguridad      26025432       0 0
modelo N=3, canal de capacidad 1                 termina        25707167       0 0
modelo N=4, 1 época, sin sospechas               seguridad       4923889       0 0
modelo N=4, 1 época, sin sospechas               termina         4920817       0 0
mutante: envío bloqueante (capacidad 1)          seguridad        239247       1 >0
mutante: reasigna solo el shard vencido          seguridad         36613       1 >0
mutante: plazo global                            seguridad            70       1 >0
mutante: primer nodo sano (N=4)                  seguridad          5269       1 >0
escrito reports/spin.json
RESULTADO: todo como se esperaba
```
