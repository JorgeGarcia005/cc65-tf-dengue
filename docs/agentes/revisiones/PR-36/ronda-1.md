# Veredicto — PR-36 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/26-coordinador-nodo` @ `df41067`
- **Fecha:** 2026-11-14
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	2.167s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	2.385s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	11.274s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.995s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.819s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.990s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.669s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.233s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.627s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	12.979s

$ go test -race -count=1 -run Equivalencia -v ./...
=== RUN   TestEquivalenciaClusterSecuencialBitABit
=== RUN   TestEquivalenciaClusterSecuencialBitABit/nodos=1
=== RUN   TestEquivalenciaClusterSecuencialBitABit/nodos=2
=== RUN   TestEquivalenciaClusterSecuencialBitABit/nodos=3
=== RUN   TestEquivalenciaClusterSecuencialBitABit/nodos=5
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.72s)
=== RUN   TestEquivalenciaConMenosNodosQueShards
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.22s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	3.104s
=== RUN   TestEquivalencia
--- PASS: TestEquivalencia (0.23s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	1.926s

$ go test -race -count=10 ./internal/cluster/
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	53.959s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; incluye 10 corridas consecutivas con `-race -count=10` en `./internal/cluster/`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalenciaClusterSecuencialBitABit` con 1, 2, 3 y 5 nodos y `TestEquivalenciaConMenosNodosQueShards` por TCP real `127.0.0.1`) |
| Latencia API p95 < 100 ms | no aplica (la API HTTP se implementa en su propio issue de PC4) |

## Verificación contra `ADR-0005` ("Verificación en PC4", puntos 1 a 5)

| Punto ADR-0005 | Estado | Evidencia |
|---|---|---|
| **1. Equivalencia** (1, 2, 3 y 5 nodos por `127.0.0.1`, idéntico bit a bit a `ml.NuevoSecuencial`, con `-race`) | ✅ | `internal/cluster/cluster_test.go:290-324` (`TestEquivalenciaClusterSecuencialBitABit` y `TestEquivalenciaConMenosNodosQueShards`). Compara `W` e `Historia` con `math.Float64bits`. |
| **2. Ida y vuelta JSON** de `Parcial` (subnormales, `-0`, `NaN` → `error`) | ✅ | Verificado en `internal/cluster/protocolo/protocolo_test.go` (PR #35) e integrado en `internal/cluster/nodo.go:76-80` (`errors.Is(err, protocolo.ErrNoFinito)` envía mensaje `error` con `"gradiente no finito"`). |
| **3. Caída escalonada (EOF + plazo)**, criterio 1 del juez ($N=3$, `timeoutEpoca = 200 ms`, nodo 1 colgado, nodo 2 cierra a los 160 ms) | ✅ | `internal/cluster/cluster_test.go:330-360` (`TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano`). El plazo por shard (`c.vence[s]` en `coordinador.go:336, 419-442`) preserva al `nodo-0`, que termina los 3 shards con pesos idénticos al secuencial. |
| **4. Balanceo tras doble caída**, criterio 2 del juez ($N=4$, caen nodos 2 y 3 en época 0 → `carga[0]==2` y `carga[1]==2`) | ✅ | `internal/cluster/cluster_test.go:364-387` (`TestBalanceoTrasDobleCaidaReparteDosYDos`) y `coordinador.go:343-364` (`elegir()` por menor `carga(j)` con desempate por menor índice). |
| **5. Nodos falsos** (cierra conexión, no responde, duplica, época vieja, informa `error`, sin `listo`, forma incorrecta) | ✅ | `internal/cluster/cluster_test.go:391-450` (`TestNodoFalsoNoCambiaElResultadoNiCuelgaElEntrenamiento` y `TestNodoFalsoConGradienteDeFormaIncorrectaSeDescarta`). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica el planteamiento ni el EDA. |
| Implementación | 8 | Avanza PC4 (núcleo distribuido coordinador–nodo en estrella síncrona) | Implementa `cluster.Coordinador`, `cluster.Nodo` y `ml.EntrenarFalible` usando únicamente la biblioteca estándar (`go.mod` sin dependencias externas), con determinismo bit a bit, tolerancia a fallos y cero fugas de goroutines. |
| Interfaz | 5 | no aplica | La API HTTP/WebSocket y la SPA corresponden a issues posteriores. |
| Informe | 2 | Mantiene 2 / 2 | Código y pruebas documentados en español y alineados con `ADR-0005`. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos.

## Pronunciamiento sobre las decisiones consultadas en el PR

1. **`ml.CalculadorFalible` y `ml.EntrenarFalible` (`internal/ml/logistica.go:279-316`) — De acuerdo:**
   Hacer que `ml.Entrenar` delegue en `ml.EntrenarFalible` mediante el adaptador `infalible` garantiza un único bucle de actualización de pesos (`w[j] -= e.Tasa * g.Grad[j]`) para los tres modos (secuencial, worker pool y clúster) y permite cortar limpiamente el entrenamiento distribuido cuando el clúster se queda sin nodos sanos (`ErrSinNodos`), devolviendo los pesos y la historia acumulados hasta la última época completa (`TestEntrenarFalibleSeDetieneEnLaEpocaQueFalla`, `logistica_test.go:234-251`).
2. **Carga perezosa (`coordinador.go:379-393`) y `carga(j)` calculada sobre `c.duenio` (`coordinador.go:356-364`) — De acuerdo:**
   Enviar `asignacion` inmediatamente antes de la primera `epoca` cuando `!n.cargado[s]` aprovecha el orden FIFO de TCP y unifica la asignación inicial con la reasignación en la misma máquina de estados (`cubrir`/`despachar`). Calcular `carga(j)` contando `c.duenio` ($O(S)$ con $S$ pequeño) elimina por construcción cualquier desincronización de contadores tras un descarte.
3. **Decisión (4): Dos pasadas (`vencer` + `cubrir`) y por qué el mutante de una sola pasada no hace fallar los tests actuales — De acuerdo en mantener las dos pasadas:**
   En `spin/barrera.pml` el mutante de una sola pasada producía un ciclo infinito porque no existían plazos individuales ni limpieza de todos los shards del nodo descartado: al reasignar `shard 0` a un `nodo 1` que también estaba vencido y luego descartar a `nodo 1` al llegar a `s = 1`, `duenio[0]` quedaba apuntando a un nodo muerto. En `coordinador.go`, `descartar(j)` (`líneas 405-413`) limpia `c.duenio[s] = -1` y `c.enviado[s] = false` para **todos** los shards de `j`, y `cubrir()` reinicia el barrido (`s = -1`, `línea 309`); por eso una sola pasada no se cuelga. Sin embargo, fusionar `vencer` y `cubrir` en una sola pasada seguiría siendo un defecto cuando dos nodos vencen al mismo tiempo (`!ahora.Before(c.vence[0])` y `!ahora.Before(c.vence[1])`): al procesar `s = 0` se elegiría a `nodo-1` (aún no marcado como vencido), enviándole por TCP `asignacion` + `epoca` inútiles (que podrían bloquear hasta `TimeoutEnvio` si su búfer de recepción está lleno) e inflando `c.reasign` antes de descartarlo en `s = 1`. Separar `c.vencer(time.Now())` y `c.cubrir()` (`coordinador.go:276-286`) evita ese envío espurio y mantiene la correspondencia exacta con `ADR-0005` y Spin. No es necesario agregar un test extra que cuente envíos para este PR.
4. **Decisión (5): Nodo de reserva (`coordinador.go:447-456`, `cluster_test.go:581-596`) — Aceptado:**
   Aunque no se pidió en `ADR-0005`, aceptar registros tardíos en `atender(ev)` dejando al nodo nuevo sano con `carga == 0` no altera el reparto de los shards ya asignados ni el determinismo bit a bit (la reducción depende únicamente de `sh.BloqueInicial`), no introduce condiciones de carrera y aporta valor directo para demostrar recuperación en vivo con contenedores en PC4/TB2.
5. **Decisión (6): Tamaño del PR (774 líneas de producción y 702 de tests, por encima de la guía de ~400 líneas de `AGENTS.md` §4) — Aceptado con justificación:**
   La regla de `AGENTS.md` §4 dice *"Un PR = un issue = una unidad revisable (idealmente < 400 líneas de código)"*. El protocolo ya se separó previamente en el Issue #25 (PR #35) y los binarios/Docker quedan en el Issue #27. Mantener `Coordinador` y `Nodo` juntos en el PR del Issue #26 permite probar toda la máquina de estados contra `Nodo.Servir` real por TCP (`127.0.0.1`) sin dobles de prueba artificiales para el camino feliz ni para la equivalencia bit a bit.

## Seguimiento de rondas previas

No aplica (primera ronda de revisión del PR #36).

## Cómo explicarlo en la sustentación

El coordinador implementa un parameter server síncrono en estrella donde una goroutine por conexión lee mensajes JSON hacia un único canal `eventos`, y solo la goroutine del bucle de épocas escribe en las conexiones y modifica el estado del entrenamiento, eliminando la necesidad de mutex sobre las estructuras de la época. En cada época, el coordinador exige los `ml.Parcial` por bloque de todos los shards (barrera) y los reduce según su índice global de bloque mediante `ml.ReducirEnOrden`, garantizando que los pesos sean idénticos bit a bit al entrenamiento secuencial sin importar cuántos nodos participen o en qué orden respondan. Si un nodo cae por EOF o vence el plazo individual de su shard (`vence[s]`), el coordinador descarta primero a los dueños vencidos y luego reasigna los shards pendientes en la misma época al nodo sano con menor carga, reutilizando en cada nodo un único `ml.Pool` para todos sus shards.
