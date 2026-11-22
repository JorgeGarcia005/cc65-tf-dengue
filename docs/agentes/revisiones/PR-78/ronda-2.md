# Veredicto — PR-78 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/63-websocket` @ `07c70ed` (base: `integracion/tb2` @ `b405959`)
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H6**) quedaron resueltos y verificados con evidencia directa. La base `integracion/tb2` @ `b405959` quedó integrada (`07c70ed`) y los 6 jobs del CI en GitHub pasan en verde (run `44974689657`). `internal/ws` se mantiene 100 % en biblioteca estándar (`AGENTS.md` §3.2) y resuelve limpiamente los aspectos críticos de concurrencia y RFC 6455: `Hub.Cerrar()` idempotente con sincronización `h.mu` + `h.activos.Wait()` y cierre `1001` para `SIGTERM`, manejo seguro de `intervaloPing <= 0`, expiración de clientes mudos con `SetReadDeadline` (`2·intervaloPing`), serialización atómica del cierre bajo `c.escritura` (imposibilitando frames tras `opCierre`) con recorte UTF-8 seguro, handshake con `Header.Values`, `http.NewResponseController(w).Hijack()`, `SetDeadline(time.Time{})` y HTTP/1.1, además de pruebas concurrentes sin `t.Fatal` en goroutines hijas que pasan 10/10 corridas con `-race`, 0 carreras, 0 fugas de goroutines y 91,6 % de cobertura.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ go list -f '{{.Imports}}' ./internal/ws
[bufio crypto/sha1 encoding/base64 encoding/binary errors fmt io net net/http strings sync sync/atomic time unicode/utf8]

$ go test -race -count=10 ./internal/ws
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	8.185s

$ go test -race -count=1 -coverprofile=/tmp/ws.cov ./internal/ws && go tool cover -func=/tmp/ws.cov
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.748s	coverage: 91.6% of statements
hub.go: NuevoHub/Suscribir/Cerrar/Desuscribir/Publicar/Clientes/Descartados/Mensajes/Hecho = 100.0%, Servir = 85.7%
ws.go:  ClaveAceptacion/contieneToken/EscribirTexto/Ping/Cerrar/cerrarSinFrame/cerrar/recortarUTF8/armarFrame/escribirFrame/escribirCrudo/enviar = 100.0%
ws.go:  Aceptar = 79.3%, LeerMensaje = 90.9%, leerFrame = 88.2%
total:  (statements) 91.6%

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.503s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.290s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.611s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.255s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.216s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.408s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.203s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.023s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.094s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.930s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	7.898s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.073s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	8.248s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.560s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.764s

$ gh pr checks 78   # run 44974689657 sobre 07c70ed
Docker (imagen, clúster y caída de un nodo)	pass	2m33s
Ecosistema de agentes                       	pass	7s
Go (gofmt, vet, test -race)                 	pass	1m3s
MongoDB y Redis (tests de integración)      	pass	40s
OpenAPI 3 válido                            	pass	17s
Spin (modelo del protocolo y mutantes)      	pass	1m36s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 21/21 paquetes con tests en verde; 10/10 corridas de `internal/ws` con `-race`, cobertura 91,6 %) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; la suite completa pasa) |
| Latencia API p95 < 100 ms | no aplica (la conexión a la API corresponde a #64/#65) |
| CI de GitHub en el PR #78 | ✅ (6/6 jobs en verde, run `44974689657`) |

## Verificación contra los criterios del issue #63

| Criterio (#63) | Estado | Evidencia |
|---|---|---|
| 1. Handshake con el vector de RFC 6455 §1.3 (`dGhlIHNhbXBsZSBub25jZQ==` → `s3pPLMBiTxaQ9kYGzzhZRbK+xOo=`) | ✅ | `ws.go:53-143`, `TestHandshakeVectorRFC`, `TestHandshakeRespuestas` (incluye `Connection` en 2 líneas, `HTTP/1.0` → 400, origen ajeno/ausente → 403, `POST` → 405, `Version: 8` → 426) y `TestPlazosHeredadosNoCortanLaConexion` (`permessage-deflate` declinada). |
| 2. Lee frames enmascarados del cliente, responde ping con pong, cierra con 1000; sin máscara → 1002; > 64 KiB → 1009 | ✅ | `ws.go:145-311`, `TestFramesEcoPingCierre`, `TestFramesRechazados` (incluye continuación suelta → 1002, binario/fragmentado → 1003, > 64 KiB → 1009), `TestFrameDe64Bits` (70 000 bytes), `TestCierreSinFramesDespuesYMotivoUTF8`. |
| 3. Un cliente que no lee no bloquea al publicador (plazo de 100 ms) y se cuentan los descartados | ✅ | `hub.go:83-94`, `TestLentoNoBloqueaAlPublicador` (`1000` publicaciones con búfer `4` → `996` descartados, < 100 ms). |
| 4. Alta y baja de clientes concurrentes sin fugas de goroutines | ✅ | `TestHubConcurrenteSinFugas` (16 clientes por TCP con pings), `TestPingPongYClienteMudo`, `TestDesconexionAbrupta`, `TestHubCerrarManda1001` (0 fugas y 0 carreras en 10 corridas con `-race`). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Alineado con `ADR-0007` y `AGENTS.md` §3.2 (100 % stdlib). |
| Implementación | 8 | Sólido para tiempo real | WebSocket RFC 6455 propio, pub/sub no bloqueante con contrapresión acotada, cierre limpio en `SIGTERM` y 0 carreras/fugas. |
| Interfaz | 5 | Habilitador | Listo para conectarse al coordinador y a la vista de entrenamiento en vivo (#64, #65, #69). |
| Informe | 2 | Aporta evidencia | Cobertura 91,6 %, patrón pub/sub justificado y pruebas de concurrencia deterministas. |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron cerrados sin introducir defectos ni condiciones de carrera nuevas.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | Base `integracion/tb2` @ `b405959` integrada en `07c70ed`; 6/6 jobs del CI en verde (run `44974689657`) y PR #75 (`#57`) aprobado en ronda 2 con `internal/ws` alineado a ADR-0007. |
| H2 (ronda 1 · MAYOR) | RESUELTO | `bdf22ca` + `d9f2909`: `Hub.Cerrar()` (`hub.go:57-70`) idempotente con `activos.Add(1)` bajo `h.mu` (`hub.go:122-130`) y cierre `1001`; `intervaloPing <= 0` sin pánico (`hub.go:134-140`); `plazoLectura = 2·intervaloPing` renovado en `LeerMensaje` (`ws.go:224-226`). Probado en `TestHubCerrarManda1001`, `TestServirSinPing`, `TestPingPongYClienteMudo` (`Ping` = 100 %). |
| H3 (ronda 1 · MAYOR) | RESUELTO | `d9f2909`: `cerrar` (`ws.go:164-176`) ejecuta envío de `opCierre`, `c.cerrado = true` y `c.nc.Close()` bajo `c.escritura`; `escribirCrudo` (`ws.go:203-210`) rechaza tras cierre con `ErrCerrado`; errores I/O/pong pasan por `cerrarSinFrame` (`ws.go:162, 232, 239`); `recortarUTF8` (`ws.go:178-186`) respeta fronteras de runas. Probado en `TestCierreSinFramesDespuesYMotivoUTF8`. |
| H4 (ronda 1 · MAYOR) | RESUELTO | `bdf22ca` + `d9f2909`: `contieneToken` con `r.Header.Values` (`ws.go:93, 134-143`), `r.ProtoAtLeast(1, 1)` (`ws.go:91-92`), `http.NewResponseController(w).Hijack()` con 500 ante `ErrNotSupported` (`ws.go:106-112`) y `nc.SetDeadline(time.Time{})` (`ws.go:116-119`). Probado en `TestHandshakeRespuestas` y `TestPlazosHeredadosNoCortanLaConexion`. |
| H5 (ronda 1 · MAYOR) | RESUELTO | `f9b1ef4`: helpers `*E` sin `t.Fatal` en goroutines hijas (`ws_test.go:37-143, 319-356`), esperas acotadas con `esperar` (`:175-184`), pruebas nuevas para ping/pong y cliente mudo, desconexión abrupta, frame de 64 bits (70 000 bytes), `Hub.Cerrar` y `permessage-deflate`. Cobertura total 91,6 %. |
| H6 (ronda 1 · MENOR) | RESUELTO | `d9f2909` + `f9b1ef4` y descripción del PR: `recibirE` comprueba `io.ReadFull` en 16/64 bits (`ws_test.go:129-137`), continuación suelta cierra con `1002` (`ws.go:248-250`, `ws_test.go:282`) y marcador `PR-78` actualizado. |

## Cómo explicarlo en la sustentación

`internal/ws` implementa WebSocket RFC 6455 por completo con la biblioteca estándar de Go: `Aceptar` valida HTTP/1.1, las cabeceras `Upgrade`/`Connection` (incluso en varias líneas), `Sec-WebSocket-Key` y `Origin`, toma el socket vía `http.NewResponseController(w).Hijack()`, limpia los plazos heredados del `http.Server` y responde `101 Switching Protocols` declinando extensiones como `permessage-deflate`. Sobre la conexión corre un `Hub` publicador–suscriptor donde `Publicar` usa envío no bloqueante (`select`/`default`) hacia canales con búfer por cliente —descartando y contabilizando mensajes ante navegadores lentos para no frenar la barrera del clúster—, cada conexión usa una goroutine lectora con plazo `2·intervaloPing` y una escritora sincronizada por mutex que impide frames tras el cierre, y `Hub.Cerrar()` apaga limpiamente todas las conexiones con código `1001` al recibir `SIGTERM` sin fugas de goroutines.
