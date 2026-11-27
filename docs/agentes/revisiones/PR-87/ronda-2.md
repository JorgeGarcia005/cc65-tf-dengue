# Veredicto — PR-87 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/69-admin-vivo` @ `3c5c34e` (código en `4b91f27`; base: `integracion/tb2` @ `c6b75ab`)
- **Fecha:** 2026-11-26
- **Decisión:** APROBADO

El implementador aceptó los tres hallazgos de la ronda 1 (`H1`, `H2`, `H3.1`–`H3.4`) en un único commit de código (`4b91f27`, sin tocar Go ni OpenAPI) y añadió 11 tests que reproducen cada caso (126/126 en Vitest). Se verificó cada corrección tanto por código/tests como en vivo en Google Chrome headless contra `coordinator -demo` (gold real de 765 297 filas) + 3 procesos `cmd/node` + `cmd/web` con CSP estricta (`style-src 'self'`):

- **H1 cerrado en código y en vivo:** el evento `modelo` guarda su `origen` (`'vivo' | 'replica'`, `web/src/admin/estado.ts:48, 79, 167, 200`), el efecto del sello solo corre para `origen === 'vivo'` con guardia `vivo` (`Admin.tsx:161-172`), `alternarReplica()` limpia `setEntrenamiento(null)` (`Admin.tsx:252`) y `data-kpi="estado"` refleja `réplica` / `abortado` / `terminado` / `sin lanzar` (`Admin.tsx:339-348`). En la prueba en vivo tras un entrenamiento de 15 épocas con sello `bit a bit ✓`, reproducir un `.jsonl` de 3 eventos terminado en `{"tipo":"modelo","detalle":"v-historica",...}` generó **0 peticiones** `GET /api/v1/modelo` (`3 → 3` en telemetría de red CDP), no pintó `[data-sello]` (`null`), y mostró `data-kpi="epoca" = "1"` y `data-kpi="estado" = "sin lanzar"` (sin heredar `/ 15` ni `terminado`).
- **H2 cerrado en código y en vivo:** `horaBitacora` (`web/src/admin/estado.ts:270-280`, usado en `Admin.tsx:425`) formatea todas las marcas en la zona del navegador (`Intl.DateTimeFormat('es-PE', { hour12: false })`). En la corrida en vivo en host `UTC-5`, las 9 líneas consecutivas de la bitácora (locales de la SPA + eventos Go `epoca`, `descarte`, `reasignacion`, `registro`, `modelo`) salieron uniformemente en `23:11:51`–`23:11:54` (0 saltos de ±5 h).
- **H3.1–H3.4 cerrados y sin regresiones:** `crearWS` dentro de `try/catch` + guardas `if (terminado) return` en `onopen`/`onmessage` (`ws.ts:106-123`), `Topologia` con `H = 320` (`Graficos.tsx:94`), rechazo de ticket vacío y `tasa = Infinity` + sondeo estable por `[api, idEntrenamiento, enCursoApi]` con guardia `vivo` (`Admin.tsx:123, 141-159, 189`), y `carga`/`workers` obligatorios en `esEstadoNodo` (`estado.ts:110`). El ajuste de los helpers en `estado.test.ts:14` y `replica.test.ts:97` (`evs.reduce((e, x) => reducir(e, x), ...)`) no debilita ninguna comprobación: evita que `Array.prototype.reduce` pase el índice numérico como tercer argumento (`origen`), mientras `estado.test.ts:147, 208-214` comprueba explícitamente tanto `'vivo'` como `'replica'`.
- **Sin regresiones sobre lo aprobado en ronda 1:** el sello `bit a bit ✓` (32 pesos), la reconexión con ticket nuevo (`npm test -- ws` 10/10), la réplica ×1/×4/×16 (`npm test -- replica` 8/8), el responsive (`scrollWidth === clientWidth` en `1280`, `768` y `375 px`), la CSP estricta (0 errores/violaciones), Go con `-race` y los **7/7 jobs del CI de GitHub** están en verde.

## Verificación automática

```text
$ git rev-parse --short HEAD
3c5c34e

$ git diff --stat dc81271..3c5c34e -- internal/ cmd/ docs/api/
(vacío = sin cambios en Go ni OpenAPI desde la ronda 1)

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ go test -race -count=1 ./internal/api/ ./internal/cluster/ ./cmd/coordinator/
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	17.018s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	6.583s   # re-ejecutado aislado tras WSAENOBUFS transitorio en Windows al correr 3 suites a la vez
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	5.210s
(0 carreras; cmd/web falla en Windows por TempDir como en integracion/tb2 y pasa en CI Linux)

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm run build && Select-String -SimpleMatch 'url(data:' dist\assets\*.css
Test Files  9 passed (9)
     Tests  126 passed (126)
npm test -- admin   -> 14 passed (14)
npm test -- ws      -> 10 passed (10)
npm test -- replica -> 8 passed (8)
✓ built in 670ms · url(data: -> (vacío = ok)

$ <auditoría en vivo sobre juez-87-r2/web/dist: coordinator -demo (gold real 765 297 filas) + 3 cmd/node + cmd/web (CSP style-src 'self') + Chrome headless>
- Conexión WS: «En vivo», 3 nodos sembrados desde GET /api/v1/cluster.
- Entrenamiento 15 épocas (tasa 2, ritmo 80 ms) + «Tumbar nodo-1» en época >= 2:
  época 15 / 15, pérdida 0.677538, reasignaciones 1, estado terminado, curva 15 puntos, 32 pesos,
  nodo-1 «cayó 1× · volvió», sello data-sello="ok" («bit a bit ✓», 32 pesos, versión 80986f048a9a).
- Bitácora en vivo (H2, host UTC-5): las 9 líneas (locales + servidor) en 23:11:51–23:11:54 (0 saltos de 5 h).
- Réplica tras entrenamiento en vivo (H1, .jsonl de 3 eventos terminado en {"tipo":"modelo","detalle":"v-historica",...}):
  modeloReqsBefore=3, modeloReqsAfter=3 (0 peticiones GET /api/v1/modelo durante/tras la réplica),
  selloAttr=null, epocaKpi="1", estadoKpi="sin lanzar".
- Consola y CSP: 0 errores de consola, 0 excepciones, 0 violaciones de CSP.
- Responsive (antes y después de entrenar/tumbar/sello):
  @ 1280px: scrollWidth=1265 === clientWidth=1265 (ok)
  @  768px: scrollWidth=753  === clientWidth=753  (ok)
  @  375px: scrollWidth=375  === clientWidth=375  (ok)

$ gh pr checks 87   # run 45937801070 sobre 3c5c34e
Docker (imagen, clúster y caída de un nodo)	pass	1m52s
Ecosistema de agentes                      	pass	10s
Go (gofmt, vet, test -race)                	pass	1m25s
MongoDB y Redis (tests de integración)     	pass	47s
OpenAPI 3 válido                           	pass	20s
SPA (tipos, tests, build e imagen)         	pass	53s
Spin (modelo del protocolo y mutantes)     	pass	2m2s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...`) | ✅ |
| `go test -race` (`internal/api`, `internal/cluster`, `cmd/coordinator`) | ✅ (0 carreras; `cmd/web` fallo conocido de `TempDir` en Windows, verde en CI Linux) |
| Equivalencia secuencial = distribuido | ✅ (sin cambios en Go desde ronda 1; `internal/cluster` y `cmd/coordinator` en verde con `-race`) |
| Latencia API p95 < 100 ms | ✅ (sin cambios en Go desde ronda 1; `1,41 ms` fría / `1,70 ms` caliente medidos en ronda 1) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP y responsive en Chrome) | ✅ (126/126 tests, 0 errores consola/CSP, sin desborde en 1280/768/375 px) |
| CI GitHub Actions (PR #87 @ `3c5c34e`) | ✅ (7/7 jobs PASS) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Explica claramente la arquitectura parameter server síncrono, la caída/reasignación de nodos y el determinismo bit a bit. |
| Implementación | 8 | 8.0 / 8 | `EvModelo` bien justificado, núcleo 100 % stdlib, `Publicar` no bloquea, 0 carreras de datos, equivalencia bit a bit con corte de nodo y `p95 < 2 ms`. |
| Interfaz | 5 | 5.0 / 5 | Panel admin en vivo completo, accesible, responsive (`1280`/`768`/`375 px`), compatible con CSP estricta, reconexión con ticket nuevo, modo réplica aislado sin llamadas a la API y bitácora en una sola zona horaria. |
| Informe | 2 | — | No corresponde a este PR (`docs/api/openapi.yaml` actualizado y validado en CI). |
| Video | 2 | — | No corresponde a este PR (flujo completo de entrenamiento en vivo + tumbar nodo + sello ✓ listo para grabar). |

## Hallazgos

Sin hallazgos abiertos (**0 bloqueantes, 0 mayores, 0 menores**).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — Réplica dispara `GET /api/v1/modelo` (falso `bit a bit ✗`) y hereda `entrenamiento` previo · MAYOR | RESUELTO (CERRADO) | `4b91f27`: `modelo.origen` (`'vivo' \| 'replica'`) en `web/src/admin/estado.ts:48, 79, 167, 200`; guardia `origen === 'vivo'` + `vivo` en `Admin.tsx:161-172`; `setEntrenamiento(null)` en `Admin.tsx:252`; KPI `Estado` en `Admin.tsx:339-348`. Verificado en Vitest (`admin.test.tsx:214-244`, `estado.test.ts:208-214`) y en vivo en Chrome (0 peticiones `GET /api/v1/modelo` en réplica, `selloAttr=null`, `epoca="1"`, `estado="sin lanzar"`). |
| H2 (ronda 1) — Bitácora mezcla horas UTC y hora local (±5 h) · MENOR | RESUELTO (CERRADO) | `4b91f27`: `horaBitacora` con `Intl.DateTimeFormat('es-PE', { hour12: false })` en `web/src/admin/estado.ts:270-280` y `Admin.tsx:425`. Verificado en Vitest (`estado.test.ts:217-227`) y en vivo en Chrome (9/9 líneas consecutivas en `23:11:51`–`23:11:54`). |
| H3.1 (ronda 1) — `crearWS` fuera de `try/catch` y `onopen`/`onmessage` tras `cerrar()` · MENOR | RESUELTO (CERRADO) | `4b91f27`: `try/catch` que cierra con motivo y sin reintentar + `if (terminado) return` en `web/src/admin/ws.ts:106-123`. Verificado con 2 tests nuevos en `ws.test.ts:196-225` (`npm test -- ws`: 10/10 PASS). |
| H3.2 (ronda 1) — Rótulo `cayó N×` recortado fuera de `viewBox` con 2 o 4 nodos · MENOR | RESUELTO (CERRADO) | `4b91f27`: `H = 320` en `web/src/admin/Graficos.tsx:94` (`y = 265 + 46 = 311 <= 316`). Verificado con `it.each([2, 4])` en `admin.test.tsx:264-270`. |
| H3.3 (ronda 1) — Ticket `""`, `tasa = Infinity` y recreación del `setInterval` cada 2 s · MENOR | RESUELTO (CERRADO) | `4b91f27`: `r.ticket === ''` (`Admin.tsx:123`), `!Number.isFinite(tasa)` (`Admin.tsx:189`), sondeo por `[api, idEntrenamiento, enCursoApi]` con guardia `vivo` (`Admin.tsx:141-159`). Verificado con 2 tests nuevos en `admin.test.tsx:247-261`. |
| H3.4 (ronda 1) — `carga` y `workers` opcionales en `esEstadoNodo` sin respaldo al interpolar · MENOR | RESUELTO (CERRADO) | `4b91f27`: `entero(n.carga) && entero(n.workers)` obligatorios en `web/src/admin/estado.ts:110`. Verificado en `estado.test.ts:231-238`. |

## Cómo explicarlo en la sustentación

> El panel de administración autentica el WebSocket con un ticket de un solo uso pedido con el JWT y reconecta con espera exponencial (`0,5 s · 2^k` hasta `30 s` + jitter) pidiendo siempre un ticket nuevo. Cada evento del coordinador y de la API pasa por la función pura `reducir(estado, evento, origen)` —la misma que usa el modo réplica sobre un `.jsonl` sin clúster, aislando su estado y sin consultar la API— para actualizar en vivo la topología en estrella, la curva de pérdida, las barras de los 32 pesos y la bitácora en la zona horaria del navegador. Cuando un nodo cae o se corta con `DEMO=true`, el coordinador reasigna sus shards manteniendo la reducción por índice de bloque; al persistir y publicar el modelo, la API anuncia sus pesos finales por WebSocket sin bloquear al entrenamiento (`EvModelo`, ya que cada `EvEpoca` lleva los pesos previos a `w -= tasa·g`), y el panel verifica identidad exacta (`===`) contra `GET /api/v1/modelo` mientras `TestServicioDemoCortarNodoPorLaAPI` garantiza esa misma igualdad bit a bit frente al entrenamiento secuencial.
