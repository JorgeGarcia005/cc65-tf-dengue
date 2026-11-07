# Veredicto — PR-19 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/9-protocolo-tcp` @ `a615f5c`
- **Fecha:** 2026-11-07
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	2.872s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	2.899s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	3.705s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	2.692s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	8.002s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	15.200s
$ go test -race -count=1 -run "Equivalencia|ShardsReproducen" -v ./...
=== RUN   TestShardsReproducenElGradienteGlobal
--- PASS: TestShardsReproducenElGradienteGlobal (0.02s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	2.434s
=== RUN   TestEquivalencia
=== RUN   TestEquivalencia/workers=1
=== RUN   TestEquivalencia/workers=2
=== RUN   TestEquivalencia/workers=4
=== RUN   TestEquivalencia/workers=8
=== RUN   TestEquivalencia/workers=16
=== RUN   TestEquivalencia/workers=64
--- PASS: TestEquivalencia (0.17s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	2.480s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia` y `TestShardsReproducenElGradienteGlobal` en verde; `ADR-0005` exige un `ml.Parcial` por bloque sin reducción local y especifica los tests de equivalencia TCP en PC4) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Cierra el diseño formal del protocolo coordinador–nodos por TCP (`ADR-0005`), justificando la topología estrella con parameter server síncrono frente a alternativas asíncronas y *ring all-reduce*, y garantizando la equivalencia bit a bit con el secuencial (`AGENTS.md` §3.3). |
| Implementación | 8 | Diseño base de PC4 + plus formal en Spin | Define mensajes JSON por línea (`json.Encoder`/`json.Decoder`), concurrencia sin mutex en el bucle de épocas, tolerancia a fallos en dos pasadas con plazos por shard (`vence[s]`), balanceo por menor `carga[j]`, pool único reutilizable en el nodo (`Pool.Parciales`) y verificación formal del protocolo de barrera en Promela/Spin (`spin/barrera.pml`, 87 711 estados en seguridad y 171 332 en LTL `<> terminado`, 0 errores). |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 | Registra la trazabilidad completa del debate multiagente (`propuestas/protocolo-tcp.md`, `propuestas/protocolo-tcp-juez.md` y síntesis en `ADR-0005-protocolo-tcp.md`) conforme a `docs/agentes/PROTOCOLO.md`. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes ni mayores. Las 4 objeciones del debate (`docs/decisiones/propuestas/protocolo-tcp-juez.md`) fueron incorporadas íntegramente en `docs/decisiones/ADR-0005-protocolo-tcp.md` (plazo individual por shard `vence[s]`, reasignación de todos los shards pendientes en caída por EOF, bucle de reintento `asignarA(s)` ante fallo de escritura, balanceo por menor `carga[j]`, mapa `shards` con un único `ml.Pool` en el nodo y seguimiento de `cargado[j][s]` en el coordinador, además de la lista de pruebas para PC4 en `ADR-0005` §Verificación). Se deja una observación menor para tener presente al implementar `internal/cluster` en PC4:

### H1 — Validación de cabecera en mensajes `listo` recibidos a mitad de época · MENOR

- **Dónde:** `docs/decisiones/ADR-0005-protocolo-tcp.md:32` y `docs/decisiones/propuestas/protocolo-tcp.md:75-77`
- **Evidencia:** En `propuestas/protocolo-tcp.md:75-77` se especifica que el mensaje `listo` repite `bloque_inicial`, `num_bloques`, `n` y `d` leídos de la cabecera del archivo gold y que el coordinador los compara con la asignación (descartando al nodo si no coinciden). Por su parte, `ADR-0005-protocolo-tcp.md:32` indica: *"Un `listo` que llega en medio de una época solo marca `cargado[j][s] = true`"*. Si en una reasignación a mitad de época un nodo lee un archivo corrupto o con metadatos distintos, el `listo` que llega durante el bucle de épocas también debe pasar por la misma verificación de cabecera antes de marcar `cargado[j][s] = true` (y, si difiere, descartar al nodo `j` y reasignar sus shards).
- **Corrección sugerida:** No requiere modificar este PR; al implementar `internal/cluster` en PC4, reutilizar la misma función de validación de `listo` tanto en la fase de registro inicial como cuando `listo` arriba al bucle de épocas tras una reasignación.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| Objeciones 1–4 del debate previo (`protocolo-tcp-juez.md`) | RESUELTO | Incorporadas en `docs/decisiones/ADR-0005-protocolo-tcp.md:27-32, 58-71`. |

## Cómo explicarlo en la sustentación

El coordinador y los $N$ nodos entrenan en topología estrella síncrona sobre conexiones TCP persistentes con mensajes JSON delimitados por salto de línea (`json.Encoder`/`json.Decoder`). Para conservar el gradiente **idéntico bit a bit** al secuencial (`AGENTS.md` §3.3), cada nodo calcula los bloques de su shard con un único `ml.Pool` y envía un `ml.Parcial` por bloque sin reducirlos localmente; el coordinador los ubica por su índice global `bloque_inicial + k` y aplica `ReducirEnOrden` y `Finalizar`. Si un nodo cae (EOF o error de escritura con `SetWriteDeadline`) o vence el plazo individual de un shard (`vence[s]`), el coordinador descarta en dos pasadas a los dueños vencidos y reasigna los shards pendientes al nodo sano de menor `carga[j]`, propiedad cuya ausencia de *deadlock* y terminación (`<> terminado`) verificamos exhaustivamente con Spin sobre `spin/barrera.pml`.
