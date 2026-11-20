# Veredicto — PR-39 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/28-api` @ `007bc25`
- **Fecha:** 2026-11-19
- **Decisión:** CAMBIOS

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.064s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	12.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.152s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.392s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.033s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	1.643s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.773s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.629s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.015s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.050s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.017s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	7.840s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.53s)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.08s)
--- PASS: TestEquivalencia (0.20s)

$ go test -race -v -run TestServicioSinNodosRegistradosNoEntrena ./cmd/coordinator
=== RUN   TestServicioSinNodosRegistradosNoEntrena
--- PASS: TestServicioSinNodosRegistradosNoEntrena (10.03s)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalenciaClusterSecuencialBitABit`, `TestEquivalenciaConMenosNodosQueShards`, `TestEquivalencia` y `TestServicioEntrenaEnElClusterYPublicaElModelo` en verde) |
| Latencia API p95 < 100 ms | no aplica (`cmd/benchapi` y Postman corresponden al issue #30; la prueba manual en `respuesta-1.md` dio 2.6 ms en 1 petición) |
| CI del PR (`gh pr checks 39`) | ✅ 4/4 en verde: `Ecosistema de agentes`, `OpenAPI 3 válido` (`@redocly/cli@1.25.0`), `Go (gofmt, vet, test -race)` y `Docker (imagen, clúster y caída de un nodo)` |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Parcial (pendiente de H8 y H9) | Todos los hallazgos de la ronda 1 (H1–H7) quedaron corregidos y la integración con el clúster real por TCP + Docker funciona con equivalencia bit a bit; falta corregir el bloqueo al cerrar con `SIGTERM` antes del registro de nodos (H8) y la espera de la goroutine de entrenamiento en `Servidor.Cerrar` (H9). |
| Interfaz | 5 | Avanza contrato HTTP + Swagger UI | Contrato OpenAPI 3.0.3 validado en CI con Redocly y servido en `/docs` y `/docs/openapi.yaml` dentro de Compose. La SPA entra en TB2. |
| Informe | 2 | Mantiene trazabilidad | Evidencia en `respuesta-1.md` y `scripts/evidencia-api.sh`; informe PC4 se redacta en su issue aparte. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H8 — `SIGTERM` mientras se esperan nodos bloquea el cierre durante todo `TimeoutRegistro` (120 s en Compose) · BLOQUEANTE

- **Dónde:** `cmd/coordinator/api.go:38,52-66` (en interacción con `internal/cluster/coordinador.go:231-247,519-538`).
- **Evidencia:** La goroutine de `cmd/coordinator/api.go:52` llama a `c.EsperarNodos(op.nodos)`, que adquiere `c.muBucle.Lock()` (`internal/cluster/coordinador.go:232`) y solo retorna cuando se completan los nodos, vence `TimeoutRegistro` o se cierra `c.hecho` (en `c.Cerrar()`). Al recibirse `SIGTERM` antes de que se registren todos los nodos, `api.Servir` termina y la línea 65 invoca `c.Terminar()`, que intenta adquirir `c.muBucle.Lock()` (`internal/cluster/coordinador.go:520`). Como `defer c.Cerrar()` (línea 38) no se ejecuta hasta después de `c.Terminar()`, el coordinador queda colgado hasta agotar `TimeoutRegistro`. Se verificó empíricamente con el propio test del PR: `TestServicioSinNodosRegistradosNoEntrena` (`cmd/coordinator/api_test.go:228`, con `tRegistro: 10 * time.Second`) tarda `10.03s` exactos esperando el vencimiento de `tRegistro` dentro de `detener()`. Con el valor por defecto de Compose (`TIMEOUT_REGISTRO=120s`, `deploy/docker-compose.yml:45`), `docker compose stop coordinator` agota su plazo de 10 s y mata el proceso con `SIGKILL` (código 137) en vez de cerrar ordenadamente y avisar `fin` a los nodos ya conectados. Además, la goroutine de línea 52 no está sincronizada con un `sync.WaitGroup` antes de retornar de `ejecutarAPI`.
- **Corrección sugerida:** Permitir cancelar la espera de nodos cuando se cancela `ctx` (por ejemplo, añadiendo `EsperarNodosContext(ctx context.Context, n int) error` en `internal/cluster/coordinador.go` con un caso `<-ctx.Done()` en el `select` de `EsperarNodos`), esperar a la goroutine de registro con un `sync.WaitGroup` antes de `c.Terminar()` y del retorno de `ejecutarAPI`, y actualizar `TestServicioSinNodosRegistradosNoEntrena` con `tRegistro: 60 * time.Second` exigiendo que `detener()` termine en menos de 2 s (y verificando también que, si ya había nodos conectados parcialmente, reciban `fin`).

### H9 — `Servidor.Cerrar()` no espera a que termine la goroutine de entrenamiento (`go s.entrenar`) · MAYOR

- **Dónde:** `internal/api/api.go:83-84,213` y `cmd/coordinator/api.go:64-65`.
- **Evidencia:** El comentario en `cmd/coordinator/api.go:64` afirma `srv.Cerrar() // cancela el entrenamiento en curso y espera su cierre`, pero `internal/api/api.go:84` solo ejecuta `func (s *Servidor) Cerrar() { s.cancelar() }` y `crearEntrenamiento` (`internal/api/api.go:213`) lanza `go s.entrenar(t.ID, p, base)` sin `sync.WaitGroup`. Si llega `SIGTERM` durante un entrenamiento, `srv.Cerrar()` regresa mientras la goroutine `s.entrenar` sigue corriendo; `c.Terminar()` y `defer c.Cerrar()` compiten con ella por el cierre del clúster y `ejecutarAPI` puede retornar (e imprimir `coordinador detenido`) antes de que `s.entrenar` finalice `s.ents.terminar(...)` y `s.op.Log(...)`. Esto incumple la regla de concurrencia de garantizar que todas las goroutines terminen antes de completar el cierre.
- **Corrección sugerida:** Añadir un `sync.WaitGroup` en `Servidor` (y un estado de cierre bajo `s.ents.mu` para que `iniciar` rechace nuevos entrenamientos tras `Cerrar` sin carrera entre `wg.Add` y `wg.Wait`): hacer `s.wg.Add(1)` antes de lanzar `go s.entrenar`, `defer s.wg.Done()` al inicio de `entrenar`, y `s.cancelar(); s.wg.Wait()` en `Servidor.Cerrar()`. Agregar un test que cierre el servidor con un entrenamiento bloqueado en curso y compruebe que `Cerrar()` no retorna hasta que `Entrenar` haya terminado.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — Predicción sin edad usaba 0 | RESUELTO | Commit `35acda3`: `Consulta.Edad` es `*float64` (`internal/api/modelo.go:113,160-162`). `TestPrediccionEdadAusenteONulaDa400` verifica 400 para ausente/`null` y 200 para `0` explícito. |
| H2 (ronda 1) — Coordinador de Compose no servía la API | RESUELTO | Commits `4f3bbe1`, `6de0b40`, `8bad2d1`, `007bc25`: modo servicio `-api` en `cmd/coordinator`, `deploy/Dockerfile`, `deploy/docker-compose.yml`, `deploy/docker-compose.api.yml`, pruebas con clúster TCP real en `cmd/coordinator/api_test.go` y evidencia desde el host con gold real en `scripts/evidencia-api.sh` / `respuesta-1.md`. Los defectos de cierre detectados en el código nuevo de esta integración se registran aparte como H8 y H9. |
| H3 (ronda 1) — Esquema con desviación 0 y `responder` devolviendo 200 vacío | RESUELTO | Commit `e38a347`: `validarNumeros` exige `desv_edad > 0`, `desv_ano > 0` y valores finitos (`internal/api/modelo.go:79-100`); `responder` codifica con `json.Marshal` antes de escribir el código y devuelve 500 con JSON válido si falla (`internal/api/api.go:251-259`). Verificado en `TestPrediccionModeloConDesviacionNulaNoSeCarga` y `TestPrediccionRespuestaNoCodificableDa500`. |
| H4 (ronda 1) — Cuerpo opcional de `POST /api/v1/entrenamientos` resultaba obligatorio | RESUELTO | Commit `f3a2a12`: `leerJSON(..., true)` acepta `io.EOF` conservando los valores por defecto (`internal/api/api.go:192,289-291`). Verificado en `TestRutasEntrenamientoSinCuerpoUsaDefectos`. |
| H5 (ronda 1) — Lector JSON aceptaba contenido sobrante con `Decoder.More()` | RESUELTO | Commit `f3a2a12`: intenta una segunda decodificación en `json.RawMessage` exigiendo `io.EOF` y preservando `http.MaxBytesError` (`internal/api/api.go:275-284`). Verificado en `TestPrediccionContenidoSobranteDa400`. |
| H6 (ronda 1) — Propiedad inválida en OpenAPI (`tipo_edad`) y falta de validación OpenAPI 3 | RESUELTO | Commit `9309822`: descripciones en línea entrecomilladas (`docs/api/openapi.yaml:149`), `TestOpenAPIMapasEnLineaBienFormados` y job de CI `OpenAPI 3 válido` (`@redocly/cli@1.25.0`) en verde; verificado también localmente con `PyYAML`. |
| H7 (ronda 1) — Clúster vacío devolvía `"nodos":null` | RESUELTO | Commit `fbd3736`: `verCluster` normaliza `nil` a `[]cluster.EstadoNodo{}` (`internal/api/api.go:180-182`). Verificado en `TestRutasClusterVacioDevuelveArregloDeNodos`. |

## Cómo explicarlo en la sustentación

El coordinador puede operar en modo servicio (`-api :8080`) sirviendo la API REST y Swagger UI (`/docs`) desde el mismo binario Go sin librerías externas, mientras mantiene conectados los nodos de ML por TCP. Cada predicción valida los campos con las mismas reglas del loader, aplica el esquema de variables guardado en `modelo.json` y lee el modelo publicado sin bloqueos mediante `atomic.Pointer`, mientras que `POST /api/v1/entrenamientos` lanza el entrenamiento distribuido en segundo plano sobre los nodos y reemplaza el modelo al terminar (con pesos idénticos bit a bit al secuencial). Al recibir `SIGTERM`, el servicio debe drenar HTTP, cancelar tanto la espera de nodos como cualquier entrenamiento activo esperando a sus goroutines, y recién entonces avisar `fin` a los nodos antes de cerrar el clúster.
