# PR-81 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/64-observador`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO (resuelto por Jorge) | `a316576` — `reports/bench_observador_jorge-pc_gold.json`, generado en el gold real (765 297 filas × 31) con `go run ./cmd/benchmark -observador -gold data/gold/train.bin -nodos 1,3,5`: las 6 mediciones (sin/con × 1, 3, 5 nodos) son iguales bit a bit al secuencial; 3 792 eventos publicados, 0 descartados. Sobrecosto (media recortada; IC95 bootstrap sin → con): 1 nodo −2,6 % [95,8–102,2 ms → 95,4–99,6 ms; se solapan: ruido]; 3 nodos +1,1 % [33,8–34,4 → 34,3–35,5; se tocan]; 5 nodos **+6,5 %** [22,3–22,9 → 23,7–24,5; no se solapan: el costo es real y crece con los eventos por época]. Máquina: Ryzen 5 3600 de Jorge (`jorge-pc`), no la Ryzen 7 de la PC4. El sobrecosto es una comparación dentro de la misma corrida, así que no mezcla procesadores. El JSON se generó con `9ea4363`: el procedimiento de medición no cambió en esta ronda; H4 solo agrega campos descriptivos a `protocolo` y guardas que no se dispararon. |
| H2 | MAYOR | ACEPTO | `b4238dc` — `Grabador` con `sync.RWMutex` + `cerrado`: `Observar` envía bajo `RLock` y, si ya se cerró, cuenta el evento como descartado; `Cerrar` cierra el canal una sola vez bajo `Lock` (idempotente), drena y devuelve el primer error. Tests: `TestGrabadorObservarYCerrarConcurrentes` (×50, 4 goroutines observando mientras se cierra, más un `Observar` posterior: escritos + descartados = observados, sin pánico, `-race`), `TestGrabadorDevuelveElErrorDeEscritura` y `TestGrabadorNoBloquea` (cierra `pw` y espera a la goroutine lectora). |
| H3 | MAYOR | ACEPTO | `b4238dc` — `Coordinador.terminado` (bajo `muBucle`): `Terminar` no hace nada si hubo un fallo (ya se avisó `fin` abortado y se emitió `EvAbortado`) o si ya terminó. Tests: `TestEventosAbortadoSinFin` (el único nodo cae → `descarte` → `abortado` con detalle, y `Terminar` después no emite `fin`) y `TestTerminarEmiteUnSoloFin`. El recolector ahora también altera `e.Nodos` (y guarda una copia de lo recibido); el estado del coordinador no cambia. |
| H4 | MENOR | ACEPTO | `4a5ee7e` — `medirObservador` fuerza `op.observador = nil` en la corrida de control (test: un observador del llamador recibe 0 eventos), falla si algún evento no se codifica o si el tiempo sin observador no es positivo, y `protocolo` lleva los campos de `ejecutarCluster`. `b4238dc` — `proyeccion` comprueba `sc.Err()`, compara los bits de los pesos y agrega `Detalle` y los campos deterministas de `Nodos`. **Nombre `tarea`:** el issue #64 decía `asignacion`, pero el evento se emite cada vez que se envía `epoca` de un shard (incluya o no el mensaje `asignacion`), así que `tarea` describe mejor lo que pasa. Queda anotado aquí y en la descripción del PR para #65 y #69. |

## Verificación

```text
$ make check                                                     en verde (incluye TestFragmentosSalenDelFuente)
$ go test -race -count=2 -run 'Observador|Eventos|Grabador|Terminar' ./internal/cluster   ok
$ go test -race -run 'Observador' ./cmd/benchmark                                        ok
```
