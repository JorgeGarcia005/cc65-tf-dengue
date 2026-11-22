# Respuesta — PR-84 · Ronda 1

- **Implementador:** Claude Code · modelo: Claude Opus 5.5
- **Veredicto respondido:** `ronda-1.md` (sobre `11aaa88`)
- **Rama:** `feature/65-ws-ticket-demo`

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `6cda7a8` — `CortarNodo` retira el nombre bajo el mutex y salta conexiones ya cerradas · `TestCortarNodoEnReposoConcurrenteYTrasReconectar` |
| H2 | BLOQUEANTE | ACEPTO | `7b0b631` — `servirWS` suscribe antes del `101` · `TestWSSuscritoAntesDelHandshake` · el test E2E acepta los registros en cualquier orden |
| H3 | MAYOR | ACEPTO | `7b0b631` — `descartarCuerpo` (413) en `emitirTicket` y `cortarNodo` · `Cache-Control: no-store` en `conTicket` |
| H4 | MENOR | ACEPTO | `8618835` — `pm.variables.get("demo")` |

## H1 — Cortes repetidos, concurrentes y tras reconexión

`CortarNodo` busca la conexión con ese nombre y hace `delete(c.nombres, id)` bajo `muConns` antes de
soltarlo. `c.conns` conserva la entrada hasta que termina la lectora, así que `Tiempos` y `Cerrar` no la
pierden. Fuera del mutex cierra la conexión. Si recibe `net.ErrClosed` (porque el bucle la descartó o porque
se llamó a `Cerrar`), vuelve a buscar otra conexión viva con el mismo nombre. Si no la hay, devuelve
`ErrNodoDesconocido` (404 en la API) y nunca 500. El log solo se escribe cuando el cierre tuvo éxito.

El test nuevo, con el bucle en reposo, comprueba que:

- un segundo corte devuelve `ErrNodoDesconocido`;
- de 8 cortes concurrentes del mismo nodo, exactamente 1 tiene éxito;
- un nodo que se reconecta con el mismo nombre (como `cmd/node -reconexiones`) es el que se corta
  después, sin errores de socket.

Con el código anterior falla igual que la prueba en vivo del juez:

```text
--- FAIL: TestCortarNodoEnReposoConcurrenteYTrasReconectar
    eventos_test.go:402: segundo corte en reposo: close tcp 127.0.0.1:41725->127.0.0.1:43048: use of closed network connection, quería ErrNodoDesconocido
```

## H2 — Suscripción antes del 101 y test sin orden fijo

`servirWS` hace `Suscribir` antes de `ws.Aceptar` y `Desuscribir` si el handshake falla. Si `Hub.Cerrar`
corre entretanto, `Hub.Servir` cierra con 1001, como sugería el juez. `TestWSSuscritoAntesDelHandshake`
publica justo después de leer el `101`, sin esperar nada más, y exige el mensaje en 20 handshakes seguidos.
`TestServicioDemoCortarNodoPorLaAPI` acumula los `EvRegistro` en un conjunto.

Antes del arreglo reproduje la falla con el comando del juez, y después lo volví a correr:

```text
$ GOMAXPROCS=1 go test -race -count=10 -failfast -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator   (antes)
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	30.424s
$ GOMAXPROCS=1 go test -race -count=20 -failfast -timeout=300s -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator   (después)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	18.352s
```

## H3 — 413 en los POST nuevos y no-store en el handshake

`descartarCuerpo` lee el cuerpo con `http.MaxBytesReader(MaxCuerpo)` y responde 413 si se pasa. Va después
de la autenticación, así que 401 y 403 siguen antes que 413. `TestCabecerasLimiteDeCuerpo413` incluye ahora
`/api/v1/ws/ticket` y `/api/v1/demo/nodos/nodo-1/cortar`. `conTicket` pone `Cache-Control: no-store` al
entrar, y `TestTicketSoloAdminUnUsoYVence` lo comprueba en el 401 y en el 403.

Con el código anterior:

```text
seguridad_test.go:333: POST /api/v1/ws/ticket con 65544 bytes: 200 {"ticket":"…"}
seguridad_test.go:333: POST /api/v1/demo/nodos/nodo-1/cortar con 65544 bytes: 202 {"estado":"cortado",…}
tiemporeal_test.go:169: WS 401 desde http://localhost:8081: 401 Cache-Control ""
tiemporeal_test.go:169: WS 403 desde https://otro-sitio.example: 403 Cache-Control ""
```

## H4 — Variable `demo` en Postman

`pm.variables.get("demo")` resuelve el entorno o `--env-var` y, si no hay, la variable de colección
(`"false"`). `{{nodo}}` ya resolvía igual.

## Verificación

```text
$ make check
gofmt: ok · vet: ok · go test -race ./...: todos los paquetes ok (exit 0)
```
