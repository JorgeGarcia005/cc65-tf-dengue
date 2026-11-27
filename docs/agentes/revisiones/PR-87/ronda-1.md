# Veredicto — PR-87 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/69-admin-vivo` @ `dc81271` (base: `integracion/tb2` @ `c6b75ab`)
- **Fecha:** 2026-11-26
- **Decisión:** CAMBIOS

El PR implementa el panel admin en vivo de `/admin` (`web/src/paginas/Admin.tsx`, `web/src/admin/{estado,ws,replica}.ts`, `web/src/admin/Graficos.tsx`, `web/src/sesion/Sesion.tsx` y estilos con prefijo `.vista-admin` en `web/src/estilos/vigia.css`) y agrega en Go el evento WebSocket `EvModelo` (`internal/cluster/eventos.go`, `internal/api/api.go`, `docs/api/openapi.yaml`, `internal/api/modelo_ws_test.go`, `cmd/coordinator/demo_test.go`).

**Evaluación de los cambios en Go (`EvModelo`, bloqueo y carreras):** la decisión de reemplazar el sello contra `EvEpoca` por un evento `modelo` emitido por la API está **plenamente justificada**: en `internal/ml/logistica.go:305-314` e `internal/cluster/coordinador.go:383`, `EvEpoca` lleva los pesos `w` *antes* de aplicar `w[j] -= e.Tasa * g.Grad[j]`, mientras que el modelo publicado tiene ese paso aplicado y su `Version` solo existe tras persistir en el almacén (`api.go:553-560`). `anunciarModelo` se llama después de `s.modelo.Store(m)` (evitando que el `GET /api/v1/modelo` del front lea una versión vieja), solo cuando `GuardarModelo` tuvo éxito (`TestEntrenamientoFallidoNoAnunciaModelo`, `TestModeloNoGuardadoNoSePublica`), y **no puede bloquear al entrenamiento**: corre fuera de la barrera de épocas y `ws.Hub.Publicar` (`internal/ws/hub.go:83-94`) usa envío no bloqueante (`select` con `default`) sin sostener `h.mu` sobre E/S ni esperas. Tampoco hay **carreras de datos**: `api.Nuevo` garantiza `s.op.Hub != nil` (`api.go:177-179`), `m.Pesos` se copia antes de serializar a un `[]byte` nuevo, `Hub.Publicar` serializa publicaciones concurrentes con `h.mu` y es seguro incluso junto con `Hub.Cerrar()` (que vacía `h.clientes` sin cerrar `c.ch`). El núcleo permanece 100 % stdlib (`AGENTS.md` §3.2), `TestEquivalencia*` y `TestServicioDemoCortarNodoPorLaAPI` (incluida corrida `GOMAXPROCS=1 -count=5`) pasan con `-race` y `0` carreras, `cmd/benchapi` contra el gold real marca `p95 = 1,41 ms` fría / `1,70 ms` caliente (< 100 ms), y los 7/7 jobs del CI en GitHub están en verde.

**Evaluación del front y tests ajustados (`app.test.tsx`):** `tsc -b`, los 115/115 tests de Vitest (`admin` 9/9, `ws` 8/8, `replica` 8/8), `vite build` sin `url(data:` en el CSS y los dos tests modificados de `app.test.tsx` pasan y **siguen comprobando exactamente lo mismo** (en `app.test.tsx:153-180` incluso refuerza el chequeo verificando `Authorization: Bearer t1` sobre la petición de la sonda; en `app.test.tsx:182-192` solo devuelve una `Response` nueva por llamada porque `/admin` ahora consulta al montar). En nuestra auditoría en vivo en Google Chrome headless contra `coordinator -demo` (gold real de 765 297 filas) + 3 procesos `cmd/node` + `cmd/web` (CSP estricta `style-src 'self'`), hubo 0 errores de consola y 0 violaciones CSP, el layout respondió sin desborde (`scrollWidth === clientWidth`) a `1280`, `768` y `375 px` antes y después de entrenar, cortar `nodo-1` en vivo produjo `descarte` + `reasignacion` + `cayó 1× · volvió`, y el entrenamiento cerró 15/15 épocas con sello `bit a bit ✓` (32 pesos). (Nota de alcance: el enunciado resumido del issue #69 mencionaba «y mapa repintándose», pero el diseño acordado en §5 y Playwright #70 no incluye un mapa dentro de `/admin` porque el cubo de `/api/v1/riesgo` es precalculado por `cmd/evaluacion`; `Admin.tsx:298` aclara correctamente que las predicciones pasan a usar el modelo publicado.)

No hay hallazgos **BLOQUEANTES**; se emiten **1 hallazgo MAYOR** y **2 MENORES** antes de aprobar: el modo réplica dispara `GET /api/v1/modelo` cuando un `.jsonl` termina en evento `modelo` (mostrando un falso `bit a bit ✗`) y hereda `entrenamiento` de una corrida previa (`Época: 1 / 15` y `Estado: terminado`) (**H1**); la bitácora mezcla horas UTC y hora local saltando 5 horas entre líneas consecutivas (**H2**); y endurecimientos menores en `ws.ts`, `Graficos.tsx` y `Admin.tsx` (**H3**).

## Verificación automática

```text
$ go version
go version go1.26.10 windows/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ go list -f '{{.Imports}}' ./internal/api ./internal/cluster
internal/api     -> solo stdlib + paquetes internos (docs/api, almacen, auth, cluster, features, ingest, ml, ws)
internal/cluster -> solo stdlib + paquetes internos (protocolo, gold, ml)

$ go test -race -count=1 ./...
24/25 paquetes con tests en verde (0 carreras).
- cmd/web: falla en Windows por limpieza de t.TempDir() (limitación conocida del entorno, mismo estado en integracion/tb2; no tocado por este PR; pasa en CI Linux).
- internal/api: en corrida paralela simultánea con otras suites hubo un connectex transitorio en TestCabecerasLimiteDeCuerpo413; re-ejecutado aislado con -race -count=1: PASS completo (16.118s), 0 carreras.

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.70s) [nodos=1, 2, 3, 5]
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.13s)
--- PASS: TestEquivalencia (0.21s) [workers=1, 2, 4, 8, 16, 64]

$ go test -race -count=1 -run TestServicioDemoCortarNodoPorLaAPI -v ./cmd/coordinator
--- PASS: TestServicioDemoCortarNodoPorLaAPI (0.70s)

$ GOMAXPROCS=1 go test -race -count=5 -failfast -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator
ok  github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator 6.936s

$ GOMAXPROCS=1 go test -race -count=10 -failfast -run 'TestEntrenamientoPublicaElModeloPorElWebSocket|TestEntrenamientoFallidoNoAnunciaModelo' ./internal/api
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/api 3.340s

$ <cmd/benchapi contra cmd/coordinator con F:\Projects\cc65-tf-dengue\data\gold (modelo 462d0758b48b, 31 variables, JWT, 16 clientes)>
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms
fria         2000       0    20261     0.56     1.41     1.64     2.63  p95 < 100 ms: true
caliente     2000       0    16597     0.73     1.70     2.62     4.03  p95 < 100 ms: true

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm run build && Select-String -SimpleMatch 'url(data:' dist\assets\*.css
Test Files  9 passed (9)
     Tests  115 passed (115)
npm test -- admin   -> 9 passed (9)
npm test -- ws      -> 8 passed (8)
npm test -- replica -> 8 passed (8)
✓ built in 651ms · url(data: -> (vacío = ok)

$ <auditoría en vivo: coordinator -demo (gold real 765 297 filas) + 3 cmd/node + cmd/web (CSP style-src 'self') + Chrome headless>
- Conexión WS: «En vivo», 3 nodos sembrados desde GET /api/v1/cluster.
- Entrenamiento 15 épocas (tasa 2, ritmo 80 ms) + «Tumbar nodo-1» en época >= 2:
  época 15 / 15, pérdida 0.677538, reasignaciones 1, estado terminado, curva 15 puntos, 32 pesos,
  nodo-1 marcado «cayó 1× · volvió», bitácora con descarte + reasignacion + registro + modelo,
  sello data-sello="ok" («bit a bit ✓», 32 pesos, versión 804b3dec429b).
- Consola y CSP: 0 errores de consola, 0 excepciones, 0 violaciones de CSP.
- Responsive (antes y después de entrenar/tumbar/sello):
  @ 1280px: scrollWidth=1265 === clientWidth=1265 (ok)
  @  768px: scrollWidth=753  === clientWidth=753  (ok)
  @  375px: scrollWidth=375  === clientWidth=375  (ok)
- Réplica tras entrenamiento vivo (archivo .jsonl de 3 eventos terminando en {tipo:"modelo", detalle:"v-historica"}):
  epocaKpi="1 / 15", estadoKpi="terminado", selloAttr="mal" («bit a bit ✗ la versión anunciada (v-historica) no es la publicada (804b3dec429b)») -> ver H1.
- Bitácora en vivo (host UTC-5): líneas locales en «03:51:17» intercaladas con eventos del servidor en «22:51:17» -> ver H2.

$ gh pr view 87 --json statusCheckRollup   # sobre dc81271
7/7 jobs CI en verde (Ecosistema de agentes, Spin, OpenAPI 3 válido, MongoDB y Redis, Go -race, SPA, Docker).
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (24/25 local + 0 carreras; `cmd/web` fallo conocido de `TempDir` en Windows, verde en CI Linux) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` + `TestServicioDemoCortarNodoPorLaAPI` 5/5 bajo `GOMAXPROCS=1 -race`) |
| Latencia API p95 < 100 ms | ✅ (`1,41 ms` fría / `1,70 ms` caliente con gold real y JWT) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP y responsive en Chrome) | ✅ (115/115 tests, 0 errores consola/CSP, sin desborde en 1280/768/375 px) |
| CI GitHub (PR #87) | ✅ (7/7 jobs en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Explica claramente la arquitectura parameter server síncrono, el corte de nodo y la garantía de determinismo en la UI. |
| Implementación | 8 | 8.0 / 8 | `EvModelo` bien justificado, 100 % stdlib, `Publicar` no bloquea, 0 carreras de datos, equivalencia bit a bit verificada incluso con corte de nodo y `p95 < 2 ms`. |
| Interfaz | 5 | 4.4 / 5 | Panel en vivo limpio, accesible, responsive y compatible con CSP estricta; corregir H1 (réplica sin clúster contaminada por `enReplica.current` y `entrenamiento` previo), H2 (huso horario en bitácora) y H3 (detalles menores) antes de aprobar. |
| Informe | 2 | — | No corresponde a este PR (`docs/api/openapi.yaml` actualizado y validado en CI). |
| Video | 2 | — | No corresponde a este PR (flujo en vivo de entrenamiento + tumbar nodo + sello ✓ ya operativo para grabarlo). |

## Hallazgos

### H1 — El modo réplica dispara `GET /api/v1/modelo` (falso `bit a bit ✗`) cuando el `.jsonl` incluye `modelo` y hereda `entrenamiento` de la corrida previa · MAYOR

- **Dónde:** `web/src/paginas/Admin.tsx:158-164`, `web/src/paginas/Admin.tsx:232-245`, `web/src/paginas/Admin.tsx:307-323`, `web/src/paginas/Admin.tsx:375`
- **Evidencia:**
  1. En `web/src/admin/replica.ts:52-57`, al entregar el último evento del archivo (`i >= eventos.length`), `reproducir` llama a `alEvento(e)` y enseguida, **en el mismo turno síncrono**, llama a `op.alTerminar?.()`. En `Admin.tsx:239-243`, `alTerminar` pone `enReplica.current = false` *antes* de que React procese el re-render de `despachar({ tipo: 'evento', evento: e })` y ejecute el `useEffect` de `Admin.tsx:158-164` (`if (!estado.modelo || enReplica.current) return`). Lo mismo sucede si el usuario pulsa «Detener réplica» tras un evento `modelo`.
  2. Como resultado, cualquier `.jsonl` que contenga el nuevo evento `modelo` al final dispara `GET /api/v1/modelo` contra la API en vivo aunque la réplica es «sin clúster» (criterio 3 del issue #69). Reproducido en vivo en Chrome tras un entrenamiento de 15 épocas con un `.jsonl` de 3 eventos terminado en `{tipo: "modelo", detalle: "v-historica", pesos: [0.1, -0.2]}`: la UI pintó `data-sello="mal"` con `bit a bit ✗ la versión anunciada (v-historica) no es la publicada (804b3dec429b)` (y sin API levantada mostraría un error de red en el sello).
  3. Ese mismo `useEffect` (`Admin.tsx:158-164`) tampoco tiene guardia de cancelación (`let vivo = true; return () => { vivo = false }`), a diferencia de `Admin.tsx:101-113`: una respuesta tardía de `cargarModelo()` tras reiniciar entrenamiento o desmontar puede pisar `sello`.
  4. En `Admin.tsx:235-236`, `alternarReplica()` reinicia `estado` y `sello`, pero **no limpia `setEntrenamiento(null)`**. En la misma prueba en vivo, durante y después de reproducir el `.jsonl` de 1 época los indicadores quedaron contaminados con la corrida en vivo anterior: `data-kpi="epoca"` mostró **`1 / 15`** (y `CurvaPerdida` estiró el eje X a `15` vía `epocasPedidas={entrenamiento?.epocas}`) y `data-kpi="estado"` mostró **`terminado`** todo el tiempo. Además, `estado.terminado` y `estado.abortado` (calculados por `reducir` en `estado.ts:191,197`) no se muestran en `data-kpi="estado"` (`Admin.tsx:322`), por lo que en una réplica sin entrenamiento previo siempre aparece `sin lanzar`.
- **Corrección sugerida:**
  - Marcar en el estado (o en la acción `evento`) si el evento `modelo` vino del WebSocket en vivo o de la réplica (p. ej. `modelo: { version, pesos, enVivo: boolean } | null` o no disparar el sello para eventos de réplica) en vez de depender del ref mutable `enReplica.current` dentro del `useEffect`. Añadir guardia `vivo` en el `useEffect` de `Admin.tsx:158-164`.
  - Llamar `setEntrenamiento(null)` al iniciar la réplica en `Admin.tsx:235` y reflejar `estado.abortado` / `estado.terminado` / `reproduciendo` en `data-kpi="estado"` cuando `entrenamiento === null`.
  - Agregar un test en `web/src/pruebas/admin.test.tsx` que reproduzca un `.jsonl` terminado en `modelo` después de un entrenamiento en vivo y compruebe que no se llama a `GET /api/v1/modelo` para el sello, no aparece `[data-sello]` y `data-kpi="epoca"` no arrastra `/ N`.

### H2 — La bitácora mezcla marcas de tiempo UTC y hora local del servidor (±5 h entre líneas consecutivas) · MENOR

- **Dónde:** `web/src/paginas/Admin.tsx:398`, `web/src/admin/estado.ts:137`
- **Evidencia:**
  - `anotar()` (`estado.ts:137`) genera marcas con `new Date().toISOString()` (siempre en UTC, sufijo `Z`), mientras Go (`internal/cluster/eventos.go:53`, `internal/api/api.go:578`) serializa `time.Now()` con el huso horario del servidor (en Windows Perú `-05:00`, o UTC en Docker frente a un `.jsonl` grabado en anfitrión).
  - `Admin.tsx:398` recorta la cadena en crudo (`<time dateTime={l.t}>{l.t.slice(11, 19)}</time>`) ignorando el desplazamiento horario. En la corrida en vivo en este host (`UTC-5`) la bitácora intercaló líneas con saltos de 5 horas:
    ```text
    03:51:17 entrenamiento ent-... lanzado: 15 épocas, tasa 2
    22:51:17 época 1 · pérdida 0.693147
    03:51:17 se pidió cortar nodo-1
    22:51:17 nodo-1 salió del clúster: ...
    03:51:19 entrenamiento terminado: modelo 804b3dec429b
    ```
- **Corrección sugerida:** Formatear `l.t` en una sola zona horaria usando `Date.parse(l.t)` con `Intl.DateTimeFormat('es-PE', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })` (con respaldo a `l.t.slice(11, 19)` si `l.t` no es fecha válida).

### H3 — Endurecimiento menor en `ws.ts`, `Graficos.tsx`, `Admin.tsx` y `estado.ts` · MENOR

- **Dónde:** `web/src/admin/ws.ts:106-121`, `web/src/admin/Graficos.tsx:93-128`, `web/src/paginas/Admin.tsx:121, 137-155, 173, 357`, `web/src/admin/estado.ts:103`
- **Evidencia:**
  1. **`ws.ts:106-121`:** `crearWS(urlWS(op.apiUrl, ticket))` no está envuelto en `try/catch` dentro de `async function abrir()` (llamada con `void abrir()`). Si `new WebSocket(...)` lanza síncronamente (URL malformada o `SecurityError` por `ws://` desde `https://`), produce una *unhandled promise rejection* y deja `conexion` atascada en `Conectando…`. Además, `s.onopen` y `s.onmessage` no comprueban `if (terminado) return`, por lo que un mensaje que llegue en vuelo tras `cerrar()` aún invoca `alEvento`.
  2. **`Graficos.tsx:93-128` (`Topologia`):** Con `H = 300`, `cy = 150` y `radio = 105`, para cualquier número par de nodos (2 o 4), el nodo inferior queda en `y = 255` y el rótulo `<text className="topo-caidas" y={pos[i]!.y + 46}>` cae en `y = 301`, **fuera de `viewBox="0 0 360 300"`** (recortado).
  3. **`Admin.tsx:121, 137-155, 173`:** `typeof r?.ticket !== 'string'` (línea 121) acepta `""` como ticket; `!(tasa > 0)` (línea 173) acepta `Infinity` (`!Number.isFinite(tasa)`); y el `useEffect` de sondeo (`137-155`) depende de `[api, entrenamiento]`, destruyendo y recreando el `setInterval` cada 2 s al llegar un nuevo objeto `t`, además de carecer de guardia `vivo` en sus `.then()`.
  4. **`estado.ts:103` y `Admin.tsx:357`:** `esEstadoNodo` permite `carga` y `workers` `undefined` (`opcional(...)`), pero `Admin.tsx:357` interpola `${n.carga}` y `${n.workers}` sin respaldo (no ocurre con el coordinador Go actual, que no usa `omitempty`, pero pintaría `undefined` ante un `.jsonl` manual que los omita).
- **Corrección sugerida:** Envolver `crearWS(...)` en `try/catch` (pasando a `cerrado` con motivo) y guardar `if (terminado) return` en `onopen`/`onmessage`; subir `H` a `320` (o bajar `radio` a `96`) en `Topologia`; exigir `r.ticket !== ''`, `Number.isFinite(tasa)`, dependencia `[api, entrenamiento?.id, entrenamiento?.estado]` con `vivo` en el sondeo, y `n.carga ?? 0` / `n.workers ?? 0` (o exigir enteros en `esEstadoNodo`).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda oficial del PR #87. |

## Cómo explicarlo en la sustentación

> El panel de administración autentica el WebSocket con un ticket de un solo uso pedido con el JWT y reconecta con espera exponencial (`0,5 s · 2^k` hasta `30 s` + jitter) pidiendo siempre un ticket nuevo. Cada evento del coordinador y de la API pasa por la función pura `reducir(estado, evento)` —la misma que usa el modo réplica sobre un `.jsonl` sin clúster— para actualizar en vivo la topología en estrella, la curva de pérdida, las barras de los 32 pesos y la bitácora. Cuando un nodo cae o se corta con `DEMO=true`, el coordinador reasigna sus shards manteniendo la reducción por índice de bloque; al persistir y publicar el modelo, la API anuncia sus pesos finales por WebSocket sin bloquear al entrenamiento (`EvModelo`, ya que cada `EvEpoca` lleva los pesos previos a `w -= tasa·g`), y el panel verifica identidad exacta (`===`) contra `GET /api/v1/modelo` mientras `TestServicioDemoCortarNodoPorLaAPI` garantiza esa misma igualdad bit a bit frente al entrenamiento secuencial.
