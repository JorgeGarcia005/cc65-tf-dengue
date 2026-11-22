# Veredicto — PR-75 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/57-adr-seguridad` @ `4c3ddb5` (base: `integracion/tb2` @ `396c4ce`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR incorpora `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md` y lo enlaza desde `docs/decisiones/ADR-0006-entidades-mongo-redis.md:37`, fijando las decisiones de seguridad (JWT HS256, PBKDF2-HMAC-SHA256 con `crypto/pbkdf2` de Go 1.24, roles `admin`/`analista`, CORS con lista blanca, cabeceras y rate limit) y de tiempo real (WebSocket RFC 6455 propio con stdlib en vez de SSE, ticket de un solo uso, hub no bloqueante e imagen `web` separada en `:8081`), respetando estrictamente `AGENTS.md` §3.2 (solo biblioteca estándar), §3.3 (determinismo) y §3.6 (sin cifras a mano). La suite local completa (`gofmt`, `go vet`, `go test -race -count=1 ./...` y `scripts/check-skills-sync.sh`) pasa en verde con 0 carreras. Sin embargo, se emite **1 hallazgo bloqueante**, **3 hallazgos mayores** y **1 hallazgo menor** antes de aprobar: el PR no tiene ningún job de CI ejecutado porque la dependencia `#56` (PR #74) sigue abierta (**H1**); la política CORS omite `Access-Control-Expose-Headers` (impidiendo leer `X-Cache` y `Retry-After` desde `:8081`) y el orden del pipeline de middlewares (**H2**); el diseño de `internal/ws` con `http.Hijacker` no resuelve los plazos heredados (`ReadTimeout: 15s`, `WriteTimeout: 30s`) ni el cierre en `SIGTERM` de `internal/api/servir.go`, y formula de forma imprecisa el rechazo de `permessage-deflate` en RFC 6455 (**H3**); y existe una contradicción entre el middleware único `Authorization: Bearer` y la autenticación por ticket + `Origin` de `GET /api/v1/ws` (**H4**).

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh && wc -m AGENTS.md
skills: ok
7781 AGENTS.md

$ grep -n ADR-0007 docs/decisiones/ADR-0006*
37:| `usuarios` (TB2) | `correo` | `hash` (PBKDF2-SHA256, ver ADR-0007), `rol` (`admin`/`analista`), `creado` | Único por `_id`. Se implementa con JWT en TB2 (ADR-0007) |

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.370s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	5.974s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.460s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.032s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.263s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.200s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.409s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.141s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.022s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.061s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.866s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	7.979s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.886s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.052s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.070s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.182s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.420s

$ gh pr checks 75 -R JorgeGarcia005/cc65-tf-dengue
no checks reported on the 'feature/57-adr-seguridad' branch   (exit 1)

$ gh run list -R JorgeGarcia005/cc65-tf-dengue --branch feature/57-adr-seguridad --limit 5
[]
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 20/20 paquetes con tests en verde sobre Go 1.24.3) |
| Equivalencia secuencial = distribuido | no aplica (solo documentación; los tests de `internal/ml` e `internal/cluster` pasan en la suite) |
| Latencia API p95 < 100 ms | no aplica (solo ADR; la remedición con JWT queda en #61) |
| CI de GitHub en el PR #75 | ❌ sin checks ejecutados (dependencia #56 / PR #74 aún sin fusionar en `integracion/tb2`, **H1**) |

## Verificación contra los criterios del issue #57

| Criterio (#57) | Estado | Evidencia |
|---|---|---|
| 1. `ADR-0007` con contexto, opciones, decisión y consecuencias (JWT HS256 stdlib, PBKDF2 `crypto/pbkdf2`, roles admin/analista, CORS lista blanca, ticket WS + `Origin`, WebSocket RFC 6455 stdlib vs SSE, imagen `web` aparte) | ⚠️ incompleto (**H2, H3, H4**) | Los 7 temas figuran en `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md:23-37` y el soporte Go 1.24 está verificado (`go.mod:3`, `deploy/Dockerfile:11`), pero faltan decisiones críticas de integración en CORS (**H2**), ciclo de vida/plazos/RFC 6455 de `internal/ws` (**H3**) y contrato de autorización de `GET /api/v1/ws` (**H4**). |
| 2. Matriz ruta × rol completa (todas las rutas de `Rutas()` más las nuevas) | ✅ (con precisión en **H4**) | `ADR-0007:41-55` cubre las 8 rutas actuales de `Rutas()` (`internal/api/api.go:240-250`) y las 6 rutas nuevas de TB2 (`login`, `riesgo`, `evaluacion`, `ws/ticket`, `ws`, `demo/nodos/{nombre}/cortar`); falta reconciliar la fila de `GET /api/v1/ws` con el middleware único de `:30` (**H4**). |
| 3. Enlazado desde `docs/decisiones/` y ADR-0006 (colección `usuarios`) | ✅ | El archivo vive en `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md` (no existe índice adicional en `docs/decisiones/`) y `grep -n ADR-0007 docs/decisiones/ADR-0006*` confirma el enlace en `ADR-0006:37`. |
| Dependencia `#56 (T0)` y CI en verde | ❌ (**H1**) | `integracion/tb2` sigue en `396c4ce` (sin `.github/workflows/ci.yml` actualizado por PR #74); `gh pr checks 75` no reporta checks. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Fija el diseño de seguridad y tiempo real antes de implementar, alineado con `AGENTS.md` §3.2 (solo stdlib) y §3.3 (determinismo). |
| Implementación | 8 | Contrato base TB2 (pendiente **H1–H4**) | Este ADR guía los issues #58–#65; cerrar **H2**, **H3** y **H4** aquí evita defectos reales en CORS cross-origin, timeouts de WebSocket a los 15 s y fallos de handshake con navegadores. |
| Interfaz | 5 | Contrato de integración (pendiente **H2**) | La separación `:8081` / `:8080` ejercita CORS de verdad, pero sin `Access-Control-Expose-Headers` la SPA no podrá mostrar `X-Cache` ni `Retry-After` (**H2**). |
| Informe | 2 | Aporta trazabilidad de diseño | Registra opciones descartadas y consecuencias para la sección de integración segura del informe final (#71). |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

### H1 — El PR #75 no tiene ejecuciones de CI porque depende del PR #74 (`#56`), aún abierto · BLOQUEANTE

- **Dónde:** PR #75 (`baseRefName: integracion/tb2`, `headRefName: feature/57-adr-seguridad` @ `4c3ddb5`), `.github/workflows/ci.yml:3-7` y dependencia `#56 (T0)` del issue #57
- **Evidencia:**
  1. `gh pr checks 75 -R JorgeGarcia005/cc65-tf-dengue` sale con código 1 (`no checks reported on the 'feature/57-adr-seguridad' branch`), y `gh run list --branch feature/57-adr-seguridad` devuelve `[]`.
  2. La rama `origin/integracion/tb2` apunta todavía a `396c4ce`, donde `.github/workflows/ci.yml:3-7` solo ejecuta el workflow sobre `[main, develop]`. El PR #74 (`feature/56-integracion-tb2`, issue #56 — dependencia explícita de #57) agrega `integracion/tb2` al trigger del CI, pero permanece `OPEN`.
  3. Tanto `AGENTS.md` §5 (*«Merge solo con veredicto `APROBADO`, CI verde y aprobación de un humano»*) como la regla del juez (*«`APROBADO` solo sin BLOQUEANTES abiertos, con `gofmt`, `go vet` y `go test -race` en verde y con el CI del PR en verde»*) impiden aprobar un PR sin checks de CI ejecutados.
- **Corrección sugerida:** Fusionar primero el PR #74 (`#56`) en `integracion/tb2` y actualizar `feature/57-adr-seguridad` sobre la nueva base (`git rebase origin/integracion/tb2` o merge de `origin/integracion/tb2`) para que corran y pasen en verde todos los jobs del CI sobre el PR #75.

### H2 — La política CORS omite `Access-Control-Expose-Headers` y el orden del encadenamiento de middlewares · MAYOR

- **Dónde:** `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md:30, 32, 37` (contrastado con `internal/api/api.go:254-265, 336, 356, 425`, `internal/api/modelo.go:121-133`, `docs/api/openapi.yaml:68-71, 106-108` e issues #60, #62, #67)
- **Evidencia:**
  1. **Cabeceras de respuesta inaccesibles para la SPA:** ADR-0007:37 despliega la SPA en `:8081` y la API en `:8080` justamente para que las peticiones sean *cross-origin*. En el modelo Fetch/CORS del navegador, `fetch()` solo expone cabeceras *CORS-safelisted* a JavaScript a menos que el servidor envíe `Access-Control-Expose-Headers`. Sin embargo, `POST /api/v1/predicciones` devuelve el estado de la caché únicamente en la cabecera `X-Cache: HIT|MISS` (`internal/api/api.go:336, 356`, `docs/api/openapi.yaml:68-71`; `api.Prediccion` en `internal/api/modelo.go:121-133` no incluye `desde_cache` en el JSON), `POST /api/v1/entrenamientos` devuelve `Location` (`internal/api/api.go:425`) y el rate limit del login devuelve `Retry-After` (`ADR-0007:29`, issue #60). Sin `Access-Control-Expose-Headers`, `response.headers.get("X-Cache")` devolverá siempre `null` en `:8081`, impidiendo mostrar el acierto de caché exigido en el issue #67.
  2. **Orden de middlewares no especificado (`OPTIONS` y errores `401`/`403`/`429`):** En `internal/api/api.go:254-265`, el `http.ServeMux` registra rutas con prefijo de método (`GET`, `POST`) y aplica `s.op.Middleware` de afuera hacia adentro. Un *preflight* `OPTIONS` del navegador nunca envía `Authorization`, y el `ServeMux` respondería `405 Method Not Allowed` si llega hasta él; por ello, el middleware CORS debe ejecutarse **por fuera** del middleware de autenticación y del `ServeMux`, y escribir `Access-Control-Allow-Origin` + `Vary: Origin` antes de delegar al siguiente handler. De lo contrario, un `401` o `403` sin cabeceras CORS es bloqueado por el navegador como error opaco de red y la SPA no puede cerrar sesión al recibir `401` ni mostrar la página `403` (issue #62).
  3. **Nombres de cabecera y `OPTIONS` de origen no permitido:** `ADR-0007:32` abrevia `Allow-Methods`, `Allow-Headers` y `Max-Age` en lugar de los nombres normativos `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers` y `Access-Control-Max-Age`, y no deja claro cómo responde un preflight `OPTIONS` de un origen fuera de `CORS_ORIGENES` (sin cabeceras `Access-Control-Allow-*` antes de llegar al `ServeMux`).
- **Corrección sugerida:**
  - En `ADR-0007:32`, agregar `Access-Control-Expose-Headers: X-Cache, Retry-After, Location` (como mínimo `X-Cache` y `Retry-After`), usar los nombres completos `Access-Control-Allow-*` y precisar `Vary: Origin` y la respuesta a `OPTIONS` no permitido (204 o 403 sin cabeceras CORS).
  - Documentar explícitamente el orden de la cadena en `Handler()`: `recuperar → cabeceras de seguridad → CORS (intercepta OPTIONS antes del mux y decora respuestas 2xx/4xx/5xx de orígenes permitidos) → auth/roles → handler`.

### H3 — `internal/ws` con `http.Hijacker` choca con los plazos y el cierre de `internal/api/servir.go`, y formula mal `permessage-deflate` en RFC 6455 · MAYOR

- **Dónde:** `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md:34, 36, 68-70` (contrastado con `internal/api/servir.go:13-21, 34-40` y `cmd/coordinator/api.go:87-96`)
- **Evidencia:**
  1. **Plazos heredados del `http.Server` (`15 s` / `30 s`):** `api.NuevoHTTP` (`internal/api/servir.go:13-21`) configura `ReadTimeout: 15 * time.Second` y `WriteTimeout: 30 * time.Second`. En Go `net/http`, esos plazos quedan fijados sobre el `net.Conn` antes de ejecutar el handler y **no se limpian al invocar `Hijack()`**. Si el servidor WebSocket no llama a `conn.SetDeadline(time.Time{})` inmediatamente después del `Hijack()` (y luego aplica plazos propios por escritura y *keepalive* `ping`/`pong`), cualquier stream de entrenamiento en vivo morirá con `i/o timeout` a los 15–30 s. Además, tras `Hijack()` las lecturas deben continuar sobre el `*bufio.ReadWriter` devuelto (puede contener bytes ya leídos del cliente) y, para no romper `Hijack()` si un middleware envuelve `http.ResponseWriter`, conviene usar `http.NewResponseController(w).Hijack()` (o no envolver `ResponseWriter`).
  2. **Cierre ordenado (`SIGTERM`) y concurrencia por conexión:** `http.Server.Shutdown` (`internal/api/servir.go:36`) **no cierra ni espera conexiones secuestradas** (documentado en la stdlib de Go). En `cmd/coordinator/api.go:87-96`, `Servir` vuelve y `srv.Cerrar()` solo cierra entrenamientos y el registrador. El ADR no fija ni el modelo de goroutines del cliente WebSocket (un lector + un único escritor para serializar envíos del hub, `pong` y frame `close` sobre `net.Conn` con plazo de escritura acotado, sin carreras ni bloqueos indefinidos) ni el cierre del hub desde `Servidor.Cerrar()` (enviando frame de cierre `1001`/`1000` y cerrando sockets para garantizar 0 fugas de goroutines al recibir `SIGTERM`, criterio de CC65 e issue #63).
  3. **Extensión `permessage-deflate` en RFC 6455 §9.1:** `ADR-0007:68-70` dice: *«lo que no implementa (extensiones como `permessage-deflate`, frames fragmentados del cliente) se rechaza con el código de cierre correspondiente, nunca se ignora en silencio»*. Todos los navegadores modernos (Chrome, Firefox, Playwright) envían `Sec-WebSocket-Extensions: permessage-deflate; client_max_window_bits` en el handshake HTTP como oferta opcional. Según RFC 6455 §9.1, un servidor sin extensiones **declina la oferta omitiendo `Sec-WebSocket-Extensions` en el `101 Switching Protocols`** (antes del `101` no existe código de cierre WebSocket, y rechazar el handshake por esa cabecera impediría conectar desde cualquier navegador). El cierre con código `1002` (Protocol Error) corresponde cuando el cliente envía posteriormente un **frame con bits `RSV1`–`RSV3` ≠ 0** sin extensión negociada (o un frame fragmentado / sin máscara), y `1009` si supera 64 KiB.
- **Corrección sugerida:**
  - En `ADR-0007:34` y `:36`, precisar: (a) `Hijack()` vía `http.NewResponseController`, limpieza de plazos heredados (`SetDeadline(time.Time{})`) y lectura desde el `*bufio.ReadWriter` del hijack; (b) 1 goroutine lectora + 1 única goroutine escritora por conexión con plazo de escritura y `ping`/`pong`; y (c) cierre explícito del hub en `Servidor.Cerrar()` al recibir `SIGTERM`.
  - En `ADR-0007:68-70`, separar negociación de extensiones en el handshake (se declinan omitiendo `Sec-WebSocket-Extensions` en la respuesta `101`, conforme a RFC 6455 §9.1) del rechazo de frames no soportados (bits `RSV` ≠ 0, frames sin máscara o fragmentados → cierre `1002`; > 64 KiB → `1009`).

### H4 — Contradicción entre el middleware único `Authorization: Bearer` y la autenticación por ticket + `Origin` de `GET /api/v1/ws` · MAYOR

- **Dónde:** `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md:30, 35, 53-54` (contrastado con `internal/api/api.go:233-250` e issues #61 y #65)
- **Evidencia:**
  1. `ADR-0007:30` establece sin excepciones: *«Cada `Ruta` declara `Roles`. Un único middleware lee `Authorization: Bearer <jwt>`: sin token o token inválido → **401**; rol no incluido → **403**. La matriz de abajo sale de `Rutas()` y un test la recorre completa»*. Sin embargo, `ADR-0007:35` y `:54` establecen que `GET /api/v1/ws?ticket=…` **no recibe `Authorization: Bearer`**, sino un ticket en la URL y validación de `Origin`. Si `GET /api/v1/ws` entrara al middleware Bearer de `:30`, el navegador siempre recibiría `401` antes de evaluar `?ticket=…`; si queda exenta del middleware Bearer, el test tabla de `:30` (issue #61, `TestMatrizRoles` sobre `Rutas()`) debe saber que esta ruta se autentica por ticket y no por cabecera `Authorization`.
  2. Faltan tres reglas concretas para `GET /api/v1/ws` en `ADR-0007:35` y `:54`:
     - **Código y caso `Origin` ausente:** el issue #65 exige rechazar `Origin` fuera de `CORS_ORIGENES` con **403**, pero el ADR no especifica ni el código `403` ni el comportamiento cuando `Origin` viene vacío/ausente (rechazarlo con `403` cuando `CORS_ORIGENES` está configurado evita saltarse la comprobación).
     - **Orden de evaluación y atomicidad del consumo:** `Origin` debe comprobarse **antes** de consumir el ticket (de lo contrario un sitio malicioso podría quemar un ticket válido o sondear su validez), y el consumo de un solo uso debe ser atómico bajo mutex (`delete` + validación de expiración `30 s` y rol) para que dos handshakes concurrentes con el mismo ticket no pasen ambos bajo `-race`.
     - **Defensa en profundidad en la columna `Analista` (`:54`):** conviene aclarar que `POST /api/v1/ws/ticket` ya bloquea a `analista` con `403` (`:53`), de modo que el `403` en `GET /api/v1/ws` (porque el ticket guarda el rol) es una verificación defensiva adicional en el consumidor del ticket.
- **Corrección sugerida:**
  - En `ADR-0007:30`, aclarar que cada `Ruta` declara su modo de autenticación (`pública`, `Bearer JWT` con lista de `Roles`, o `ticket WS` con `Roles`) y cómo el test tabla recorre cada modo.
  - En `ADR-0007:35` y `:54`, fijar explícitamente: `Origin` ausente o fuera de `CORS_ORIGENES` → **403** (evaluado antes de tocar el almacén de tickets); ticket inexistente, vencido (> 30 s) o reutilizado (consumo atómico bajo mutex) → **401**; rol en ticket distinto de `admin` → **403** (defensa en profundidad frente a `:53`).

### H5 — Precisiones menores en referencias de issues y detalles de seguridad (`hmac.Equal`, IP de rate limit, `CSP connect-src` y normalización de correo) · MENOR

- **Dónde:** `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md:5, 25, 28, 29, 31, 37, 48-49, 71` y descripción del PR #75
- **Evidencia:**
  1. **Issues relacionados (`ADR-0007:5` y cuerpo del PR):** citan `issues #58–#65`, pero las rutas `GET /api/v1/riesgo` y `GET /api/v1/evaluacion` (`:48-49`) pertenecen al **issue #67**, la reconexión WS con ticket al **issue #69** y `reports/seguridad.json` (`:71`) al **issue #70** (mientras `#59` es evaluación offline en `internal/evaluacion`). También conviene reemplazar el placeholder `PR-<n>` por `PR-75` en la sección *Uso de IA* de la descripción del PR.
  2. **Detalles puntuales que previenen ambigüedades al implementar #58, #60 y #62:**
     - `Token` (`:25`): explicitar `hmac.Equal` (tiempo constante) y `base64.RawURLEncoding` (RFC 7515), al nivel de precisión con que `:27` exige `crypto/subtle.ConstantTimeCompare`.
     - `Usuarios` (`:28` y `ADR-0006:37`): normalizar `correo` (`strings.ToLower(strings.TrimSpace(...))`) al guardar y buscar, porque `_id` en MongoDB distingue mayúsculas.
     - `Login / Rate limit` (`:29`): indicar que la IP se extrae del host de `r.RemoteAddr` (`net.SplitHostPort`, sin confiar en `X-Forwarded-For` falsificable sin proxy inverso de confianza) y alinear la regla con el issue #60 (*«6.º intento **fallido** en un minuto → 429»*): un token bucket con recarga continua de 5/min repone 1 token cada 12 s, por lo que 6 fallos espaciados > 12 s dentro del mismo minuto no devolverían 429; conviene aclarar si es ventana deslizante de 1 min o token bucket de capacidad 5 y tasa 5/min sobre intentos fallidos (o todos los intentos), acotando/limpiando entradas viejas del mapa por IP.
     - `CSP` (`:31, 37`): indicar que `cmd/web` construye `connect-src 'self' <API_URL> <WS_URL>` a partir de `API_URL` (incluyendo `ws:`/`wss:`), pues API y front viven en orígenes distintos (`:8080` vs `:8081`).
- **Corrección sugerida:** Actualizar la línea de relacionados (`:5`) y el cuerpo del PR, e incorporar las cuatro precisiones de una línea en la tabla de `ADR-0007` para dejar sin ambigüedades a #58, #60 y #62.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-75. |

## Cómo explicarlo en la sustentación

El sistema separa dos roles —analista de salud pública para consultar riesgo, evaluación y predicciones, y administrador para operar el clúster— y protege la API REST solo con la biblioteca estándar de Go (`crypto/hmac`, `crypto/sha256` y `crypto/pbkdf2` de Go 1.24), declarando los permisos por ruta en `Rutas()` junto con una lista blanca estricta de CORS entre el frontend en `:8081` y la API en `:8080`. Para el entrenamiento en vivo se implementa un servidor WebSocket RFC 6455 propio sobre `http.Hijacker` autenticado con un ticket aleatorio de un solo uso de 30 s más validación de `Origin`, publicando hacia los navegadores por canales con búfer sin bloqueo para que ningún cliente lento frene la barrera del clúster ni altere los pesos bit a bit. Una vez incorporadas las precisiones de CORS (`Access-Control-Expose-Headers` y orden de middlewares), ciclo de vida de conexiones secuestradas (`SetDeadline`, cierre en `SIGTERM` y declinación de `permessage-deflate`), excepción de ticket para `/api/v1/ws` y la corrida verde del CI tras fusionar `#56`, el contrato queda cerrado y listo para implementar TB2.
