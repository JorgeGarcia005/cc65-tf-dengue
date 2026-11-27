# Respuesta — PR-87 · Ronda 1

- **Implementador:** Claude Code (Claude Opus 5.5), con José Villanueva
- **Veredicto respondido:** `ronda-1.md` → **CAMBIOS** sobre `dc81271`
- **Commit de la respuesta:** `4b91f27`
- **Fecha:** 2026-11-26

Se aceptan los tres hallazgos. Cada uno tiene al menos un test que fallaba en `dc81271` y pasa en
`4b91f27` (11 tests nuevos; 126/126 en Vitest). No se tocó Go: la parte del evento `modelo` que la
ronda 1 dio por buena queda igual.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 — la réplica dispara `GET /api/v1/modelo` y un falso `bit a bit ✗`, y hereda «/ N» y «terminado» del entrenamiento en vivo | MAYOR | ACEPTO | `4b91f27`. El evento `modelo` guarda `origen` (`vivo` o `replica`, `reducir(e, ev, origen)` en `admin/estado.ts`), y el efecto del sello solo corre con `origen === 'vivo'`: ya no depende de `enReplica.current`, que `alTerminar` bajaba en el mismo turno. Los efectos del sello y del sondeo tienen guardia `vivo`. Al iniciar la réplica se hace `setEntrenamiento(null)`, y el KPI de estado muestra `réplica`, `abortado`, `terminado` o `sin lanzar` cuando no hay entrenamiento en vivo. Tests: «una réplica terminada en «modelo» no pide GET /modelo, no pinta sello ni arrastra «/ N»» (`admin.test.tsx`; reproduce la prueba del juez: entrenamiento de 15 épocas en vivo y luego `.jsonl` terminado en `modelo`) y «reducir marca si el modelo vino en vivo (por defecto) o de una réplica» (`estado.test.ts`). |
| H2 — la bitácora mezcla horas UTC y del servidor (±5 h) | MENOR | ACEPTO | `4b91f27`. `horaBitacora(t)` en `admin/estado.ts` formatea con `Intl.DateTimeFormat('es-PE', { hour12: false })` en la zona del navegador, y cae a `t.slice(11, 19)` si `t` no es una fecha. Test: «el mismo instante en UTC y en -05:00 se muestra igual». Verificado contra el Compose: líneas de la SPA y del coordinador quedan todas en 23:04. |
| H3.1 — `new WebSocket` que lanza deja una promesa rechazada y «Conectando…»; mensajes tras `cerrar()` | MENOR | ACEPTO | `4b91f27`. `crearWS` va dentro de `try/catch` y deja la conexión `cerrado` con el motivo, sin reintentar (URL mala o `SecurityError` no se arreglan reintentando). `onopen` y `onmessage` salen si `terminado`. Tests: «si new WebSocket lanza…» y «tras cerrar(), un mensaje o un open en vuelo del socket viejo no se entregan» (`ws.test.ts`). |
| H3.2 — con 2 o 4 nodos, «cayó N×» del nodo inferior sale del `viewBox` | MENOR | ACEPTO | `4b91f27`. `Topologia` pasa a `viewBox="0 0 360 320"`. Test: «topología con %i nodos caídos y vueltos: ningún texto sale del viewBox» para 2 y 4 nodos. |
| H3.3 — ticket `""`, tasa `Infinity` y sondeo que recrea el intervalo | MENOR | ACEPTO | `4b91f27`. Se rechaza el ticket vacío y se exige `Number.isFinite(tasa)`. El sondeo depende de `[api, id, estado]` en vez del objeto `entrenamiento`, y tiene guardia `vivo`. Tests: «un ticket vacío no abre el WebSocket» y «una tasa infinita no se envía». |
| H3.4 — `carga` y `workers` opcionales en `esEstadoNodo` pero interpolados sin respaldo | MENOR | ACEPTO | `4b91f27`. Pasan a ser enteros obligatorios: `cluster.EstadoNodo` los serializa siempre (sin `omitempty`). Un `.jsonl` manual que los omita ahora da error con su número de línea en vez de pintar `undefined`. Test: «un nodo sin carga o sin workers no es un evento válido». |

Al escribir el test de H1 apareció un detalle de los propios tests: `evs.reduce(reducir, inicial)`
pasaba el índice del arreglo como tercer argumento, que ahora es `origen`. Los dos helpers
(`estado.test.ts` y `replica.test.ts`) llaman ahora a `reducir(e, x)` de forma explícita. La vista
no tenía ese problema: usa su propio reductor.

**Verificación:** `npx tsc -b`, `npx vitest run` (126/126, 9 archivos) y `npm run build` en
verde, sin `url(data:` en el CSS. Sin cambios en Go desde la ronda 1. Probado contra el Compose
(`DEMO=true`) con el usuario admin: entrenamiento de 100 épocas, bitácora en una sola zona
horaria, topología con `viewBox` de 320 y sello `bit a bit ✓`.

Queda listo para la ronda 2.
