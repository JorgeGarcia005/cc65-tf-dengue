# PR-78 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/63-websocket`.
Nota: la ronda 1 revisó `22b98df`. El commit `bdf22ca` (subido antes del veredicto, por el H3 del PR-75) ya agregaba `NewResponseController`, `SetDeadline(time.Time{})`, la extensión declinada y `Hub.Cerrar` con 1001. Esta respuesta lo cita donde corresponde.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO (dependencia) | El CI no corre hasta fusionar #74 (#56). Orden: #74 → #75 → … Tras fusionarlo se actualiza esta rama y corren los jobs. |
| H2 | MAYOR | ACEPTO | `bdf22ca` + `d9f2909` — `Hub.Cerrar()` idempotente: marca cerrado, da de baja (cierre `1001`) y **espera** con `sync.WaitGroup` a todas las llamadas a `Servir` (el `Add` se hace bajo `h.mu`, así `Cerrar` no puede empezar a esperar antes); un `Servir` posterior cierra con 1001 sin lanzar goroutines. `intervaloPing ≤ 0` → sin pings y sin pánico. Con pings, `plazoLectura = 2·intervalo` se renueva antes de cada frame leído (ping, pong o texto): un cliente mudo cae en ese plazo. Tests: `TestHubCerrarManda1001`, `TestServirSinPing`, `TestPingPongYClienteMudo` (el mudo cae en < 500 ms con intervalo de 20 ms; `Ping` pasa de 0 % a 100 % de cobertura). |
| H3 | MAYOR | ACEPTO | `d9f2909` — `cerrado` protegido por el candado de escritura: `Cerrar` toma el candado, envía el frame de cierre, marca `cerrado` y cierra el socket en la misma sección crítica, y `escribirCrudo` devuelve `ErrCerrado` si `cerrado`. Ningún frame sale después del cierre (RFC 6455 §5.5.1). Los errores de I/O y el fallo al responder pong pasan por `cerrarSinFrame` (mismo `sync.Once`) y devuelven error envuelto en `ErrCerrado`. El motivo se recorta a ≤ 123 bytes sin partir runas (`recortarUTF8`). Test: `TestCierreSinFramesDespuesYMotivoUTF8` (motivo de 200 bytes en «ñ», escritura posterior → `ErrCerrado`, no llega ningún frame más). |
| H4 | MAYOR | ACEPTO | `bdf22ca` + `d9f2909` — `contieneToken` recorre `r.Header.Values` (cabeceras de lista en varias líneas, RFC 9110 §5.3); `http.NewResponseController(w).Hijack()` con 500 solo ante `http.ErrNotSupported`; `nc.SetDeadline(time.Time{})` tras el hijack (contrato de `Hijacker`); `r.ProtoAtLeast(1, 1)` o 400. Casos nuevos en `TestHandshakeRespuestas`: «Connection en dos líneas» → 101 y «HTTP/1.0» → 400. |
| H5 | MAYOR | ACEPTO | `f9b1ef4` — los helpers tienen versión `…E` que devuelve error; las goroutines usan solo esas y `t.Errorf`. Las esperas tienen plazo (`esperar`). Tests nuevos: ping del servidor → pong del cliente (más de 5 pings en 200 ms) y cliente mudo, `TestDesconexionAbrupta` (cierre TCP sin frame, sin fugas), `TestFrameDe64Bits` (70 000 bytes), `Hub.Cerrar` con clientes conectados sin fugas, `Connection` repetida y `permessage-deflate` declinada (`TestPlazosHeredadosNoCortanLaConexion`). |
| H6 | MENOR | ACEPTO | `f9b1ef4` — `recibirE` comprueba los errores de `io.ReadFull` en los largos extendidos. `d9f2909` — continuación suelta (`op 0x0`) → cierre `1002` (RFC 6455 §5.4); el texto fragmentado sigue en `1003`. Caso «continuación suelta» en `TestFramesRechazados`. Marcador `PR-<n>` → `PR-78` en la descripción. |

## Verificación

```text
$ gofmt -l internal/ws          (vacío)
$ go vet ./internal/ws          (vacío)
$ go test -race -count=10 ./internal/ws
ok   github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws   8.1s
$ go test -race -coverprofile=ws.cov ./internal/ws && go tool cover -func=ws.cov
Ping 100.0% · escribirFrame/armarFrame 100.0% · Hecho 100.0% · Cerrar 100.0% · total 91.6%
```
