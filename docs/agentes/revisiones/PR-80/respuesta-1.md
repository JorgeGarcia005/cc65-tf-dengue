# PR-80 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/60-login`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MAYOR | ACEPTO | `b08d7d0` — `limitador.Reservar(ip)` comprueba y reserva en la misma sección crítica, contando **fallos vigentes + intentos en curso**; devuelve una función `liberar(resultado)` (con `sync.Once`) que el handler llama en un `defer`, también ante un pánico: fallo → se anota, éxito → se borran los fallos, neutro (JSON inválido, error de base) → no cuenta. Si el cupo está ocupado por intentos en curso → 429 con `Retry-After: 1`. `TestLoginRafagaConcurrente`: 20 peticiones simultáneas con clave errada → exactamente 5 × 401 y 15 × 429 (con `Retry-After`), con `-race` y `-count=3`. `TestLoginRateLimit` ahora mata el mutante sin `Exito` (comprobado: quitando el borrado, el test falla con 429). `TestLimitadorCuentaIntentosEnCurso` cubre la reserva, el intento neutro y la doble liberación. |
| H2 | MAYOR | ACEPTO | `b08d7d0` — la poda global corre a lo sumo una vez cada media ventana (`ultimaPoda`) y el mapa tiene tope duro (`maxIPsLimitador` = 8 192): si sigue lleno y llega una IP nueva, se desaloja una entrada. `TestLimitadorMemoriaAcotada`: 100 IPs activas en la misma ventana con tope 16 → el mapa no pasa de 16 y la poda corre una sola vez; pasada la ventana queda solo la IP nueva. |
| H3 | MAYOR | ACEPTO | `b08d7d0` — `SembrarUsuarios` exige los mismos topes que el login (`MaxBytesCorreo` = 254 y `MaxBytesClave` = 256, **en bytes**; el mensaje del 400 ahora dice bytes) y rechaza que dos roles compartan el correo normalizado. `TestSembrarUsuarios`: clave de 257 bytes, 150 «ñ» (300 bytes), correo de más de 254 bytes y admin/analista con «Misma@b.pe»/«misma@B.pe» → error. |
| H4 | MENOR | ACEPTO | `b08d7d0` — `Nuevo` precalcula `auth.HashSenuelo()` en segundo plano si hay firmador (el primer login paga una sola derivación); `deploy/.env.example` usa `RandomNumberGenerator` (y advierte que `Get-Random` no sirve); el comentario de paquete de `api` lista el login y el de `mongo` menciona `Usuarios`; el 503 distingue «falta JWT_SECRETO» de «no hay almacén de usuarios» (`TestLoginSinUsuariosDa503`). |

## Verificación

```text
$ make check                  gofmt, vet y go test -race ./... en verde (incluidos cmd/coordinator e internal/almacen/mongo)
$ go test -race -count=3 -run 'Login|Limitador|Sembrar|IPCliente' ./internal/api    ok
```
