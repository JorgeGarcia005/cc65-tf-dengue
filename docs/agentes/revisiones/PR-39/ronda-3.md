# Veredicto — PR-39 · Ronda 3

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/28-api` @ `e60ec55`
- **Fecha:** 2026-11-19
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.064s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.051s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.170s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.395s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.037s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	1.942s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.730s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.759s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.014s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.051s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.248s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.016s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	9.790s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.49s)  (nodos=1, 2, 3, 5)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.11s)
--- PASS: TestEquivalencia (0.25s)  (workers=1, 2, 4, 8, 16, 64)

$ go test -race -count=5 -v -run 'TestServicio|TestCerrarEspera' ./cmd/coordinator ./internal/api
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	8.504s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	2.585s
(TestServicioSinNodosRegistradosNoEntrenaYCierraSinEsperarElRegistro: 0.02s–0.06s en 5/5 corridas con tRegistro=60s)
(TestCerrarEsperaAlEntrenamientoEnCursoYNoAdmiteOtro: 0.30s–0.32s en 5/5 corridas)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalenciaClusterSecuencialBitABit`, `TestEquivalenciaConMenosNodosQueShards`, `TestEquivalencia` y `TestServicioEntrenaEnElClusterYPublicaElModelo` en verde) |
| Latencia API p95 < 100 ms | no aplica (`cmd/benchapi` y Postman pertenecen al issue #30; `cmd/` no contiene `benchapi` todavía) |
| CI del PR (`gh pr checks 39`) | ✅ 4/4 en verde: `Ecosistema de agentes` (12s), `OpenAPI 3 válido` (20s), `Go (gofmt, vet, test -race)` (49s) y `Docker (imagen, clúster y caída de un nodo)` (1m44s) |
| Restricciones AGENTS.md §3 | ✅ Solo stdlib (`go.mod` sin terceros), `data/` sin archivos de datos versionados (`MANIFEST.sha256` y `README.md`), determinismo bit a bit intacto |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Completo para el alcance del issue #28 | API REST hecha solo con `net/http` integrada en `cmd/coordinator` y Compose, validación estricta en español, entrenamiento distribuido por TCP con equivalencia bit a bit, cierre ordenado completo ante `SIGTERM` sin bloqueos ni goroutines huérfanas y cero carreras bajo `-race`. Quedan para otros issues de PC4 el benchmark/Postman (#30) y MongoDB/Redis (#29). |
| Interfaz | 5 | Completo para el contrato de PC4 | Especificación OpenAPI 3.0.3 validada en CI con `@redocly/cli@1.25.0`, cubierta contra el mux por test y servida con Swagger UI en `/docs` dentro de Compose. La SPA entra en TB2. |
| Informe | 2 | Mantiene trazabilidad | Evidencia reproducible en `scripts/evidencia-api.sh` y en el historial de rondas del PR; la redacción del informe PC4 va en su issue aparte. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos. Los dos hallazgos pendientes de la ronda 2 (H8 y H9) quedaron corregidos y verificados con tests reproducibles bajo `-race`, sin regresiones en H1–H7 ni en la equivalencia bit a bit del clúster.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H8 (ronda 2 · BLOQUEANTE) — `SIGTERM` mientras se esperan nodos bloqueaba el cierre durante todo `TimeoutRegistro` | RESUELTO | Commit `681b4cf`: `Coordinador.EsperarNodosContext(ctx, n)` (`internal/cluster/coordinador.go:240-258`) atiende `<-ctx.Done()` en el `select` y suelta `muBucle` de inmediato. En `cmd/coordinator/api.go:44-80`, `ejecutarAPI` corre la espera con `wgNodos` y cierra en orden `cancelarNodos()` → `srv.Cerrar()` → `wgNodos.Wait()` → `c.Terminar()` → `defer c.Cerrar()`. No hay posibilidad de interbloqueo porque `ent.listo` solo pasa a `true` tras salir de `EsperarNodosContext`. Verificado en `TestServicioSinNodosRegistradosNoEntrenaYCierraSinEsperarElRegistro` (`cmd/coordinator/api_test.go:230-283`, `tRegistro: 60s` con 1/2 nodos): cierra en `0.02s–0.06s` (antes `10.03s` con `tRegistro: 10s`) y el nodo conectado recibe `fin` completo (`err == nil`). |
| H9 (ronda 2 · MAYOR) — `Servidor.Cerrar()` no esperaba a la goroutine de entrenamiento | RESUELTO | Commit `3826576`: `Servidor` añade `cierreMu`, `cerrado`, `wgEntrenar` y `cerrarUnaVez` (`internal/api/api.go:66-69,93-101,227-248`). `crearEntrenamiento` hace `wgEntrenar.Add(1)` bajo `cierreMu` solo si `!s.cerrado` (de lo contrario responde 503; si `ents.iniciar` devuelve 409 compensa con `Done()`), `entrenar` ejecuta `defer s.wgEntrenar.Done()`, y `Cerrar()` marca `cerrado = true` bajo `cierreMu`, cancela `s.ctx` una vez (`sync.Once`) y llama a `s.wgEntrenar.Wait()` (relación *happens-before* garantizada entre todo `Add(1)` y `Wait()`, e idempotente). Verificado en `TestCerrarEsperaAlEntrenamientoEnCursoYNoAdmiteOtro` (`internal/api/api_test.go:615-635`): `Cerrar()` espera los 300 ms del entrenador lento, deja el estado `fallido` registrado antes de retornar, rechaza un `POST` posterior con 503 y tolera una segunda llamada a `Cerrar()`. |
| H1–H7 (ronda 1) — Validación de edad, integración en Compose, desviaciones/JSON 500, cuerpo opcional, JSON sobrante, OpenAPI YAML y `"nodos":[]` | RESUELTO | Confirmados sin regresiones: `go test -race -count=1 -v -run 'Prediccion|Rutas|OpenAPICubreLasRutas|Cierre|CerrarEspera' ./internal/api` en verde (17/17 tests) y CI `OpenAPI 3 válido` + `Docker` en verde. |

## Cómo explicarlo en la sustentación

El coordinador opera en modo servicio (`-api :8080`) sirviendo la API REST y Swagger UI (`/docs`) desde el mismo binario Go con solo `net/http`, mientras mantiene conectados los nodos de ML por TCP. Cada predicción valida los campos con las mismas reglas del loader, transforma la consulta con el esquema guardado en `modelo.json` y lee el modelo publicado sin bloqueos mediante `atomic.Pointer`, mientras que `POST /api/v1/entrenamientos` ejecuta el entrenamiento distribuido en segundo plano sobre los nodos y reemplaza el modelo entero al terminar (con pesos idénticos bit a bit al secuencial). Al recibir `SIGTERM`, el servicio drena HTTP con `Shutdown`, rechaza nuevos entrenamientos con 503, cancela y espera ordenadamente con `WaitGroup` tanto a la goroutine de entrenamiento como a la espera de nodos (`EsperarNodosContext`), y recién entonces envía `fin` a los nodos conectados antes de cerrar el clúster.
