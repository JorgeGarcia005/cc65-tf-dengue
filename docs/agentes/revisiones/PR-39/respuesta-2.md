# Respuesta — PR #39 · Ronda 2

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-19 · **Base:** `feature/28-api` @ `d6bd8fe`

Acepto los dos hallazgos. Los dos eran reales: los reproduje con los tests nuevos antes de corregir
(mutantes al final).

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H8 | BLOQUEANTE | ACEPTO | `681b4cf`: `cluster.EsperarNodosContext(ctx, n)` agrega `case <-ctx.Done()` al `select` de la espera (`EsperarNodos` delega con `context.Background()`). En `ejecutarAPI` la espera usa el `ctx` del servicio, corre en una goroutine con `sync.WaitGroup` y el cierre sigue este orden: `cancelarNodos()` → `srv.Cerrar()` → `wgNodos.Wait()` → `c.Terminar()` (los nodos sanos reciben `fin`) → `defer c.Cerrar()`. Test: `TestServicioSinNodosRegistradosNoEntrenaYCierraSinEsperarElRegistro`, con `tRegistro: 60s` y 1 de 2 nodos conectado: el cierre tarda menos de 2 s y el nodo conectado termina con `fin` completo (`Servir` devuelve `nil`) |
| H9 | MAYOR | ACEPTO | `3826576`: `Servidor` tiene `cierreMu`, `cerrado` y `wgEntrenar`. `crearEntrenamiento` hace el `Add` bajo `cierreMu` y responde 503 «el servidor se está cerrando» si ya se cerró (el `Add` nunca corre en paralelo al `Wait`); `entrenar` hace `defer wgEntrenar.Done()`; `Cerrar` marca el cierre, cancela y espera (idempotente, con `sync.Once`). Test: `TestCerrarEsperaAlEntrenamientoEnCursoYNoAdmiteOtro`: un `Entrenador` que tarda 300 ms después de la cancelación; `Cerrar()` no vuelve antes, el estado ya quedó `fallido` y un POST posterior da 503 |

La misma estructura de cierre (`cierreMu` + `cerrado` + `wgEntrenar`) es la que el PR #40 ya usa para
su H6, así el rebase del #40 sobre este PR no la duplica.

## Mutantes

| Mutante | Hallazgo | Resultado |
|---|---|---|
| `ejecutarAPI` con `c.EsperarNodos(op.nodos)` (sin contexto) | H8 | `FAIL ... (30.03s): el servicio no se detuvo tras cancelar el contexto` |
| `Servidor.Cerrar` sin `wgEntrenar.Wait()` | H9 | `FAIL: Cerrar volvió antes de que terminara el entrenamiento` |

## Verificación

```text
$ gofmt -l . ; go vet ./...        → vacío
$ go test -race -count=1 ./...     → todos ok, sin carreras
$ go test -race -count=5 ./cmd/coordinator ./internal/api
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	5.570s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.596s
$ go test -race -count=1 -run Servicio -v ./cmd/coordinator
--- PASS: TestServicioEntrenaEnElClusterYPublicaElModelo (0.21s)
--- PASS: TestServicioRechazaUnLambdaDistintoAlDelCluster (0.02s)
--- PASS: TestServicioSinNodosRegistradosNoEntrenaYCierraSinEsperarElRegistro (0.02s)
--- PASS: TestServicioRechazaModeloYShardsIncompatibles (0.02s)
```

El test de H8 pasó de 10.03 s (lo que midió el juez, esperando `tRegistro`) a 0.02 s con un `tRegistro` de 60 s.
