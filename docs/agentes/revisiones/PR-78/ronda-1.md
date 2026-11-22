# Veredicto — PR-78 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/63-websocket` @ `22b98df` (base: `integracion/tb2` @ `396c4ce`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR añade el paquete `internal/ws` (`ws.go`, `hub.go`, `ws_test.go`; 363 líneas de producción y 292 de pruebas) implementando el lado servidor de WebSocket RFC 6455 y un hub pub/sub no bloqueante **exclusivamente con la biblioteca estándar de Go** (`AGENTS.md` §3.2). Las verificaciones locales (`gofmt`, `go vet`, `go test -race -count=1 ./...` y `go test -race -count=10 ./internal/ws`) pasan con 0 carreras, el handshake verifica el vector RFC 6455 §1.3 y el límite de 64 KiB se valida antes de asignar memoria (evitando OOM). Sin embargo, se emiten **1 hallazgo bloqueante**, **4 mayores** y **1 menor** antes de aprobar: el PR no tiene ningún job de CI ejecutado porque `#56` (PR #74) y su dependencia `#57` (PR #75, ADR-0007) siguen abiertos (**H1**); `Hub` carece de `Cerrar()` para apagado ordenado en `SIGTERM`, `Servir` entra en pánico sin recuperación si `intervaloPing <= 0` y `Ping`/`Pong` no aplica plazo de lectura (**H2**); `Conn.Cerrar` puede emitir un frame posterior al cierre (RFC 6455 §5.5.1), `LeerMensaje` esquiva `cerrarUna` en errores de I/O y `motivo` se trunca por bytes sin respetar UTF-8 (**H3**); `Aceptar` usa `Header.Get` (rompe cabeceras `Connection` repetidas de RFC 9110) y `w.(http.Hijacker)` en lugar de `http.NewResponseController` (**H4**); y las pruebas llaman a `t.Fatal` desde goroutines auxiliares y dejan sin cubrir `Ping` (0 %) ni el largo de 64 bits pese a declararlos en el PR (**H5**).

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

$ go list -f '{{.Imports}}' ./internal/ws
[bufio crypto/sha1 encoding/base64 encoding/binary errors fmt io net net/http strings sync sync/atomic time]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.305s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.262s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.463s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.266s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.213s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.424s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.201s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.025s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.077s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.862s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.006s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.944s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.038s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.072s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.256s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.449s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.052s

$ go test -race -count=10 -v ./internal/ws
PASS (10/10 iteraciones: TestHandshakeVectorRFC, TestHandshakeRespuestas,
      TestFramesEcoPingCierre, TestFramesRechazados,
      TestLentoNoBloqueaAlPublicador, TestHubConcurrenteSinFugas)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.319s

$ go test -coverprofile=/tmp/ws.cov ./internal/ws && go tool cover -func=/tmp/ws.cov
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	0.020s	coverage: 89.7% of statements
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws/hub.go:80:	Hecho		0.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws/ws.go:123:	Ping		0.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws/ws.go:137:	escribirFrame	83.3%
total:								(statements)	89.7%

$ gh pr checks 78
no checks reported on the 'feature/63-websocket' branch (exit 1)

$ gh run list --branch feature/63-websocket --limit 5 --json databaseId,workflowName,status,conclusion
[]
```

| Chequeo | Resultado |
|---|---|
| `gofmt` / `go vet` | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Solo biblioteca estándar (`AGENTS.md` §3.2) | ✅ |
| CI del PR en GitHub | ❌ (sin checks reportados; base `integracion/tb2` aún sin `#56`) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`) |
| Latencia API p95 < 100 ms | no aplica (aún no se conecta a `internal/api`; corresponde a #65) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---:|---|---|
| Planteamiento | 3 | fuera de alcance | Cubierto en PC3/PC4 y ADR-0007 (#57). |
| Implementación | 8 | parcial (bloqueado por H1; mejoras H2–H5) | Implementa el bloque «WebSocket» en Go stdlib y el patrón pub/sub sin bloqueo. Faltan `Hub.Cerrar()` para `SIGTERM`, robustez en cierre/handshake/ping y CI en verde. |
| Interfaz | 5 | fuera de alcance | La integración con el endpoint y la SPA viene en #65 y #69. |
| Informe | 2 | fuera de alcance | El PR no modifica `docs/informe/`. |
| Video | 2 | fuera de alcance | Corresponde al cierre de TB2. |

## Hallazgos

### H1 — CI del PR no ejecutado (base `integracion/tb2` sin `#56` ni `#57` fusionados) · BLOQUEANTE

- **Dónde:** PR #78 (`baseRefName: integracion/tb2`, `headRefName: feature/63-websocket` @ `22b98df`), `.github/workflows/ci.yml:3-7` y dependencia `#57 (T1)` del issue #63
- **Evidencia:**
  1. `gh pr checks 78` termina con código 1 (`no checks reported on the 'feature/63-websocket' branch`), y `gh run list --branch feature/63-websocket` devuelve `[]`.
  2. La rama `origin/integracion/tb2` apunta todavía a `396c4ce`, donde `.github/workflows/ci.yml:3-7` solo dispara el workflow en `[main, develop]`. El PR #74 (`#56`, que añade `integracion/tb2` al CI) y el PR #75 (`#57`, ADR-0007 — dependencia explícita `Depende de #57 (T1)` del issue #63, actualmente con observaciones de diseño sobre WebSocket en `PR-75/ronda-1.md` H3) siguen en estado `OPEN`.
  3. Tanto `AGENTS.md` §5 (*«Merge solo con veredicto `APROBADO`, CI verde y aprobación de un humano»*) como la regla del juez impiden aprobar un PR sin la corrida verde del CI.
- **Corrección sugerida:** Fusionar primero los PR #74 (`#56`) y #75 (`#57`) en `integracion/tb2`, actualizar `feature/63-websocket` sobre la nueva base (`git rebase origin/integracion/tb2` o merge de `origin/integracion/tb2`) y confirmar que todos los jobs del CI pasan en verde sobre el PR #78.

### H2 — Ciclo de vida en `Hub`: sin `Cerrar()` para `SIGTERM`, pánico no recuperable con `intervaloPing <= 0` y `Ping`/`Pong` sin plazo de lectura · MAYOR

- **Dónde:** `internal/ws/hub.go:13-18, 90-120` y `internal/ws/ws.go:123, 177` (contrastado con `internal/api/servir.go:36` y `cmd/coordinator/api.go:93`)
- **Evidencia:**
  1. **Fuga de goroutines y conexiones en `SIGTERM`:** `go doc net/http.Server.Shutdown` documenta expresamente: *«Shutdown does not attempt to close nor wait for hijacked connections such as WebSockets. The caller of Shutdown should separately notify such long-lived connections of shutdown and wait for them to close, if desired.»* En `internal/ws/hub.go:15`, `clientes` no está exportado y `Hub` no expone ningún método `Cerrar()` ni lleva un `sync.WaitGroup` de las llamadas activas a `Servir`. Al conectar el hub en #65, `api.Servidor.Cerrar()` (`cmd/coordinator/api.go:93`) no podrá desuscribir clientes, cerrar sockets ni esperar la terminación de las goroutines lectora y escritora (criterio central de la lista de revisión: *«¿Todos los goroutines terminan?»*, y observación H3 de `PR-75/ronda-1.md`).
  2. **Pánico fatal en goroutine hija (`hub.go:96`):** `ping := time.NewTicker(intervaloPing)` entra en pánico (`non-positive interval for NewTicker`) si un llamador pasa `intervaloPing <= 0`. Como corre dentro de `go func()` (`hub.go:93`), el pánico **no es atrapado** por `net/http` ni por `api.Servidor.recuperar` (`internal/api/api.go:268`) y tumba todo el proceso coordinador.
  3. **`Ping`/`Pong` no detecta clientes caídos silenciosamente (`ws.go:177`, `hub.go:104-107`):** `case op == opPong:` no hace nada y `leerFrame` nunca establece ni renueva `SetReadDeadline`. Ante una caída silenciosa de red (TCP *half-open*, sin FIN ni RST), `conn.Ping()` sigue escribiendo 2 bytes al búfer TCP del kernel sin error y `LeerMensaje()` permanece bloqueado hasta el timeout TCP del sistema operativo (~15 min en Linux). Además, `go tool cover -func` muestra `ws.go:123: Ping 0.0%`: la lógica de ping del servidor nunca se ejecutó durante la prueba.
- **Corrección sugerida:**
  - Añadir `Hub.Cerrar()` (idempotente) que bajo `h.mu` marque el hub como cerrado, dé de baja a todos los clientes activos (y haga que futuras llamadas a `Suscribir`/`Servir` cierren la conexión de inmediato con `1001` Going Away o `1000` sin lanzar goroutines) y espere con `sync.WaitGroup` a que terminen todos los `Servir` activos (haciendo `wg.Add(1)` bajo `h.mu` antes de soltarlo, igual que `internal/api/api.go:402-403`).
  - En `Servir`, manejar `intervaloPing <= 0` dejando el canal del ticker en `nil` (sin pings) o aplicando un valor por defecto positivo en vez de entrar en pánico.
  - Cuando `intervaloPing > 0`, establecer y renovar un `SetReadDeadline` (p. ej. `2*intervaloPing` o `intervaloPing + c.plazo`) al recibir cualquier frame/pong en `LeerMensaje` para cerrar conexiones mudas en tiempo acotado.

### H3 — Posible frame posterior al frame de cierre (RFC 6455 §5.5.1), desvío de `cerrarUna` en `LeerMensaje` y truncado de `motivo` por bytes UTF-8 · MAYOR

- **Dónde:** `internal/ws/ws.go:126-135, 150-156, 164-176`
- **Evidencia:**
  1. **Frame enviado después de `opCierre` (`ws.go:128-133, 150-156`):** `Cerrar` llama a `c.escribirFrame(opCierre, p)` —que toma y libera `c.escritura.Lock()` dentro de `escribirCrudo` (`:151-152`)— y solo después llama a `c.nc.Close()` (`:132`). Entre el `Unlock()` de `escribirCrudo` y `c.nc.Close()`, la goroutine escritora de `Hub.Servir` (`hub.go:101, 105`) puede adquirir `c.escritura.Lock()` y escribir un frame `opTexto` o `opPing` **después** del frame de cierre, infringiendo RFC 6455 §5.5.1 (*«An endpoint MUST NOT send any more data frames after sending a Close frame»*).
  2. **Desvío de `cerrarUna` y error sin cierre (`ws.go:167-168, 174-175`):** Cuando `leerFrame` falla por I/O (`codigo == 0`, p. ej. `io.EOF` por cierre abrupto del cliente), `:168` llama a `c.nc.Close()` directamente sin pasar por `c.cerrarUna`; luego el `defer conn.Cerrar(CierreNormal, "")` de `hub.go:95` entra igualmente en `cerrarUna.Do`, intenta escribir un frame de cierre en un socket ya cerrado y vuelve a llamar a `c.nc.Close()`. Por otro lado, si falla `c.escribirFrame(opPong, p)` (`:174-175`), `LeerMensaje` devuelve el error sin cerrar `c` ni envolver `ErrCerrado`, contradiciendo su documentación (`:158-160`).
  3. **Truncado por bytes en UTF-8 (`ws.go:130`):** `motivo[:min(len(motivo), 123)]` puede partir una runa UTF-8 multibyte (tildes o `ñ` en mensajes en español), generando UTF-8 inválido en el payload del frame de cierre (prohibido por RFC 6455 §5.5.1).
- **Corrección sugerida:**
  - Proteger un campo `cerrado bool` con `c.escritura`: `escribirCrudo` comprueba `if c.cerrado { return ErrCerrado }`; `Cerrar` adquiere `c.escritura`, comprueba/marca `c.cerrado = true`, escribe `opCierre` (si se pidió código) mientras mantiene el lock y luego cierra `c.nc` (garantizando que ningún frame salga después del cierre).
  - Para cierres por error de I/O (`codigo == 0`) y fallos al responder `opPong`, cerrar mediante un auxiliar interno que pase por `cerrarUna` (sin enviar frame de cierre) y devolver siempre un error envuelto con `ErrCerrado`.
  - Recortar `motivo` respetando fronteras de runas UTF-8 (≤ 123 bytes válidos).

### H4 — Handshake RFC 6455/9110 en `Aceptar`: `Header.Get` ignora cabeceras repetidas, `w.(http.Hijacker)` falla con wrappers y no limpia plazos explícitamente · MAYOR

- **Dónde:** `internal/ws/ws.go:80, 92-100`
- **Evidencia:**
  1. **Cabeceras repetidas (`ws.go:80`):** `r.Header.Get("Connection")` (y `Upgrade`) retorna únicamente la primera línea de cabecera. Si un navegador o proxy inverso envía `Connection: keep-alive` y `Connection: Upgrade` en líneas separadas (equivalentes según RFC 9110 §5.3), `Aceptar` rechaza un handshake válido con `400 Bad Request`.
  2. **Incompatibilidad con wrappers de `http.ResponseWriter` (`ws.go:92`):** la aserción directa `w.(http.Hijacker)` responde `500` si cualquier middleware (logging, métricas o cabeceras en #60–#65) envuelve `ResponseWriter` exponiendo `Unwrap() http.ResponseWriter`. `http.NewResponseController(w).Hijack()` (stdlib desde Go 1.20) desenrolla automáticamente y mantiene alineado el paquete con `PR-75/ronda-1.md` H3.
  3. **Contrato de plazos tras `Hijack()` (`ws.go:96-100`):** aunque en Go 1.24 `net/http.(*conn).hijackLocked` limpia los plazos (`net/http/server.go:325`), la documentación oficial de `net/http.Hijacker` advierte explícitamente: *«The returned net.Conn may have read or write deadlines already set, depending on the configuration of the Server. It is the caller's responsibility to set or clear those deadlines as needed.»* No llamar a `nc.SetDeadline(time.Time{})` depende de un detalle interno de `net/http` y queda expuesto ante otros `Hijacker`.
- **Corrección sugerida:**
  - Cambiar `contieneToken` para aceptar `[]string` y pasarle `r.Header.Values("Connection")` y `r.Header.Values("Upgrade")`.
  - Sustituir `w.(http.Hijacker)` por `http.NewResponseController(w).Hijack()`, respondiendo `500` cuando `errors.Is(err, http.ErrNotSupported)`.
  - Invocar `_ = nc.SetDeadline(time.Time{})` inmediatamente después de `Hijack()` (y rechazar peticiones `< HTTP/1.1` con `r.ProtoAtLeast(1, 1)` por RFC 6455 §4.2.1).

### H5 — Pruebas concurrentes: `t.Fatal` desde goroutines auxiliares, brechas de cobertura (`Ping` 0 %, largo 64 bits 0 %) y escenarios faltantes · MAYOR

- **Dónde:** `internal/ws/ws_test.go:31-59, 62-116, 240-292` (contrastado con `ws.go:123, 144-145` y `hub.go:80`)
- **Evidencia:**
  1. **Uso indebido de `t.Fatal` en goroutines hijas (`ws_test.go:253-269`):** las 16 goroutines cliente llaman a `conectar(t, ...)`, `c.recibir(t)` y `c.enviar(t, ...)`, que invocan `t.Fatal` (`:35, 50, 55, 87, 95, 98, 113`). `go doc testing.T.FailNow` prohíbe expresamente llamarlo desde goroutines distintas a la del test (*«FailNow must be called from the goroutine running the test or benchmark function, not from other goroutines created during the test»*). Además, si una goroutine cliente aborta antes de suscribirse (`:258` o `t.Fatal`), `for h.Clientes() < 16` (`:271-274`) carece de plazo límite y cuelga el test hasta el timeout global de `go test`.
  2. **Cobertura desalineada con lo declarado en el PR:** `go test -coverprofile=/tmp/ws.cov ./internal/ws && go tool cover -func=/tmp/ws.cov` reporta `ws.go:123: Ping 0.0%`, `hub.go:80: Hecho 0.0%` y `ws.go:137: escribirFrame 83.3%` (la rama de largo de 64 bits `ws.go:144-145`, > 65 535 bytes, nunca se ejecuta). Sin embargo, la descripción del PR afirma cubrir *«largos de 7, 16 y 64 bits»* y *«alta/baja y pings»* (`TestHubConcurrenteSinFugas` termina en ~10 ms antes de que venza el `intervaloPing` de 20 ms en `:248`).
  3. **Escenarios concurrentes sin probar:** no hay prueba de cierre iniciado por el servidor (`Hub.Cerrar()` con clientes activos verificando 0 fugas de goroutines), desconexión TCP abrupta sin frame `0x88`, envío de `Ping` del servidor con respuesta `Pong` (y expiración por falta de `Pong`), cabeceras `Connection` en múltiples líneas ni oferta `Sec-WebSocket-Extensions: permessage-deflate` de navegadores reales.
- **Corrección sugerida:**
  - Hacer que los helpers de cliente devuelvan `error` (o usar `t.Errorf` + `return` en las goroutines hijas) y acotar con plazo el bucle de espera `for h.Clientes() < 16` (`ws_test.go:271-274`).
  - Agregar pruebas deterministas para: frame del servidor > 65 535 bytes (largo de 64 bits), `Ping` del servidor → `Pong` del cliente y expiración de cliente mudo, `Hub.Cerrar()` con clientes conectados sin fugas de goroutines, caída TCP abrupta del cliente, `Connection` repetida en dos líneas y `Sec-WebSocket-Extensions: permessage-deflate` declinada en el `101`.

### H6 — Detalles menores en helper de prueba, continuación suelta y descripción del PR · MENOR

- **Dónde:** `internal/ws/ws_test.go:104, 108`, `internal/ws/ws.go:186-188` y descripción del PR #78
- **Evidencia:**
  1. `ws_test.go:104, 108`: `io.ReadFull(c.r, b[:])` ignora el error devuelto al leer los largos extendidos de 16 y 64 bits en `recibir`.
  2. `ws.go:186-188`: un frame de continuación suelto (`op == 0x0` sin mensaje fragmentado previo) cierra con `1003` (`CierreNoSoportado`), cuando RFC 6455 §5.4 lo clasifica como error de protocolo (`1002`).
  3. En la descripción del PR #78 (sección *Uso de IA*), permanece el marcador `docs/agentes/revisiones/PR-<n>/ronda-1.md` en lugar de `PR-78`.
- **Corrección sugerida:** Comprobar los errores de `io.ReadFull` en `ws_test.go:104, 108`, devolver `CierreProtocolo` (1002) para `op == opContinuacion` suelto si se busca apego estricto a RFC 6455 §5.4, y actualizar el marcador de la descripción del PR a `PR-78`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-78. |

## Cómo explicarlo en la sustentación

El servidor WebSocket arranca con una petición HTTP `GET` que el coordinador promociona con `Hijack` respondiendo `101 Switching Protocols` y el hash SHA-1 de la clave del navegador (RFC 6455), implementado por completo con la biblioteca estándar de Go sin librerías de terceros. Sobre esa conexión TCP corre un patrón publicador–suscriptor (`Hub`): cada cliente conectado recibe su propio canal con búfer acotado y el coordinador publica cada época mediante envío no bloqueante (`select` con `default`), descartando y contabilizando actualizaciones si un navegador va lento para nunca frenar la barrera de entrenamiento. Una vez incorporados el cierre ordenado del hub (`Hub.Cerrar()` para `SIGTERM`), la sincronización estricta del frame de cierre, el soporte de `ResponseController`/`Header.Values` y la corrida verde del CI tras fusionar `#56` y `#57`, el paquete quedará listo para conectarse a la API en `#64` y `#65`.
