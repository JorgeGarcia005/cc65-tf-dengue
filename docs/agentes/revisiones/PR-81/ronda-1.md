# Veredicto — PR-81 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/64-observador` @ `9ea4363` (base: `integracion/tb2` @ `0f9c0f5`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR añade un observador de eventos versionado (`v: 1`) al coordinador (`internal/cluster/eventos.go`, `coordinador.go`) que emite en orden desde el bucle de épocas pasando copias de pesos y nodos para preservar el determinismo bit a bit (`TestObservadorBitABit` con 1 y 3 nodos, incluso sobrescribiendo los pesos recibidos con `NaN`), un `Grabador` JSONL con cola con búfer y descarte no bloqueante (`TestGrabadorReproducible`, `TestGrabadorNoBloquea`), el modo `-observador` en `cmd/benchmark` que mide el sobrecosto de `json.Marshal` + `ws.Hub.Publicar` (`TestBenchmarkObservadorPublicaYNoCambiaLosPesos`) y la regeneración de `docs/informe/pc4/08-distribuido.md` verificada por `TestFragmentosSalenDelFuente`. Todo usa exclusivamente la biblioteca estándar (`AGENTS.md` §3.2), todas las pruebas locales con `-race` (incluida la equivalencia secuencial = distribuido bit a bit) y los 7 jobs del CI en GitHub pasan en verde. Sin embargo, se emiten **1 hallazgo bloqueante**, **2 mayores** y **1 menor** antes de aprobar: falta generar y versionar el JSON real del sobrecosto `reports/bench_observador_<maquina>[_gold].json` exigido por el criterio 4 del issue #64 (`Cierra #64`) y `AGENTS.md` §3.6 (**H1**); `Grabador.Observar()` entra en pánico (`send on closed channel`) si se llama en paralelo o después de `Grabador.Cerrar()` y faltan pruebas de concurrencia/error/cierre limpio del grabador (**H2**); `Coordinador.Terminar()` emite `EvFin` («el entrenamiento terminó bien») incluso después de `abortar()` (`EvAbortado`) o en llamadas repetidas, y `EvAbortado` carece de tests (**H3**); y hay detalles menores de robustez en `medirObservador`/`ejecutarObservador`, la divergencia de nombre `tarea` vs `asignacion` respecto de #64 y la proyección de `TestGrabadorReproducible` (**H4**).

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/cluster ./cmd/benchmark
github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster -> [bufio context crypto/sha256 encoding/hex encoding/json errors fmt .../protocolo .../gold .../ml io math net os sync sync/atomic time]
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark -> [bytes encoding/json errors flag fmt .../cluster .../protocolo .../estadistica .../gold .../maquina .../ml .../ws log math math/rand/v2 net os path/filepath runtime slices strconv strings sync sync/atomic time]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.357s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	8.747s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.700s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.036s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.666s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.274s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.253s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.423s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.055s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.188s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.041s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.067s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.036s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.388s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.260s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.230s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.220s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.986s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.632s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.761s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.99s) [nodos=1,2,3,5]
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.17s)
--- PASS: TestEquivalencia (0.54s) [workers=1,2,4,8,16,64]

$ go test -race -count=1 -run 'Observador|Eventos|Grabador' -v ./internal/cluster ./cmd/benchmark
--- PASS: TestObservadorBitABit (0.66s) [nodos=1,3]
--- PASS: TestEventosSecuenciaYContenido (0.17s)
--- PASS: TestEventosCaidaDeUnNodo (0.14s)
--- PASS: TestGrabadorReproducible (0.21s)
--- PASS: TestGrabadorNoBloquea (0.00s)
--- PASS: TestBenchmarkObservadorPublicaYNoCambiaLosPesos (2.43s)

$ go test -race -count=1 -run 'Fragmentos' -v ./cmd/informe
--- PASS: TestFragmentosSalenDelFuente (0.16s)

$ go run ./cmd/benchmark -observador -filas 20000 -features 8 -nodos 1,3 -reps-cluster 2 -epocas-cluster 3 -salida /tmp/bench_obs_test.json
1 nodos: 2.91 ms/época sin observador, 3.05 con observador (+4.9 %)
3 nodos: 2.47 ms/época sin observador, 1.84 con observador (-25.2 %)
(eventos_publicados: 96, eventos_descartados: 0)

$ gh pr checks 81
Docker (imagen, clúster y caída de un nodo)	pass	2m48s
Ecosistema de agentes	pass	8s
Go (gofmt, vet, test -race)	pass	57s
MongoDB y Redis (tests de integración)	pass	42s
OpenAPI 3 válido	pass	23s
SPA (tipos, tests, build e imagen)	pass	51s
Spin (modelo del protocolo y mutantes)	pass	2m3s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| CI GitHub Actions (7 jobs, incl. Docker caída de nodo y Spin) | ✅ |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalenciaClusterSecuencialBitABit`, `TestObservadorBitABit`) |
| Latencia API p95 < 100 ms | no aplica (no toca `internal/api`) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | Alineado con ADR-0005 y ADR-0007. |
| Implementación | 8 | Parcial (pendiente **H1–H3**) | El observador preserva el determinismo bit a bit (`AGENTS.md` §3.3) y solo usa stdlib (`§3.2`), pero falta el reporte real de sobrecosto (**H1**), `Grabador` entra en pánico al observar tras `Cerrar` (**H2**) y `Terminar` emite `EvFin` incluso tras `EvAbortado` (**H3**). |
| Interfaz | 5 | Base lista para #65/#69 | Define el contrato JSON versionado `v: 1` para el stream WebSocket. |
| Informe | 2 | Parcial (bloqueado por **H1**) | `docs/informe/pc4/08-distribuido.md` queda sincronizado con el fuente (`TestFragmentosSalenDelFuente` ✅), pero falta `reports/bench_observador_*.json` para respaldar las cifras del sobrecosto sin números a mano (`AGENTS.md` §3.6). |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

### H1 — Falta generar y versionar `reports/bench_observador_<maquina>[_gold].json` (criterio #4 de #64) · BLOQUEANTE

- **Dónde:** `reports/bench_observador_*.json` (ausente en `feature/64-observador` @ `9ea4363`), issue #64 (criterio 4) y descripción del PR #81 (`Cierra #64`)
- **Evidencia:**
  1. El criterio 4 del issue #64 establece: *«Sobrecosto medido → `reports/bench_observador_<maquina>.json` — verificar con: el json»*.
  2. `git diff --stat origin/integracion/tb2...origin/feature/64-observador` lista únicamente 8 archivos de código, tests y `docs/informe/pc4/08-distribuido.md`; no incluye ningún `reports/bench_observador_*.json`.
  3. La propia descripción del PR deja abierta la casilla `- [ ] Cifras: el JSON del sobrecosto lo genera Jorge con el script` (*«Pendiente (Jorge): generar el JSON real en el Ryzen 7…»*) pese a declarar `Cierra #64`. Tal como en PR #77 H1 (`reports/evaluacion.json`), cerrar #64 sin el JSON medido deja el criterio 4 incumplido y al informe final sin la fuente versionada del sobrecosto (`AGENTS.md` §3.6).
- **Corrección sugerida:** Tras aplicar las correcciones de H2–H4, ejecutar en la máquina con `data/gold/train.bin` real `go run ./cmd/benchmark -observador -gold data/gold/train.bin -nodos 1,3,5`, versionar el `reports/bench_observador_<maquina>_gold.json` resultante en esta rama y marcar la casilla en la descripción del PR.

### H2 — `Grabador.Observar()` entra en pánico (`send on closed channel`) si concurre con o sigue a `Grabador.Cerrar()`, y faltan pruebas de concurrencia/errores · MAYOR

- **Dónde:** `internal/cluster/eventos.go:61-104` y `internal/cluster/eventos_test.go:185-220`
- **Evidencia:**
  1. `Grabador.Cerrar()` (`eventos.go:100-104`) hace `g.cerrar.Do(func() { close(g.cola) })`, mientras que `Grabador.Observar(e)` (`eventos.go:87-93`) ejecuta `case g.cola <- e:`. Cualquier llamada a `Observar` concurrente con o posterior a `Cerrar` (por ejemplo, un `defer g.Cerrar()` registrado antes de `defer c.Cerrar()`, o un cierre ante `SIGTERM` mientras llega un `EvRegistro`, `EvDescarte` o `EvFin`) provoca **`panic: send on closed channel`**.
  2. Esto contrasta con el patrón seguro ya consolidado en el propio repositorio en `internal/api/api.go:199-241` (*«El canal nunca se cierra (los handlers podrían enviar después)»*) y `internal/ws/hub.go:72-94` (*«El canal no se cierra (un Publicar en curso podría escribir en él)»*).
  3. Además, `internal/cluster/eventos_test.go` no cubre: (a) concurrencia entre `Observar` y `Cerrar` bajo `-race` ni llamadas repetidas a `Cerrar()` (idempotencia); (b) propagación a `Cerrar()` de un error de `io.Writer` durante `Encode` o `Flush`; y (c) en `TestGrabadorNoBloquea` (`eventos_test.go:216-220`), se lanza `go io.Copy(io.Discard, pr)` y se cierra `pr` sin cerrar `pw` ni esperar la salida de esa goroutine antes de terminar el test.
- **Corrección sugerida:**
  - Proteger el cierre del canal en `Grabador` para eliminar por construcción el pánico: p. ej. añadir `mu sync.RWMutex` y `cerrado bool`; en `Observar`, tomar `RLock`, retornar (o contabilizar descarte) si `cerrado`, hacer el envío no bloqueante a `g.cola` y liberar `RUnlock`; en `Cerrar`, tomar `Lock`, marcar `cerrado = true` y cerrar `g.cola` una sola vez antes de `Unlock` y `<-g.hecho`. Así `Observar` sigue sin bloquear, todo lo aceptado antes de `Cerrar` se drena y escribe completo, y ningún `Observar` tardío o concurrente puede causar pánico.
  - Agregar tests con `-race` en `eventos_test.go` para `Observar` concurrente/posterior a `Cerrar`, `Cerrar` idempotente, retorno de error cuando `w.Write` falla, y cierre de `pw` + espera explícita de la goroutine lectora en `TestGrabadorNoBloquea`.

### H3 — `Coordinador.Terminar()` emite `EvFin` («terminó bien») incluso tras `abortar()` (`EvAbortado`) o en llamadas repetidas, y `EvAbortado` carece de pruebas · MAYOR

- **Dónde:** `internal/cluster/coordinador.go:556-581`, `internal/cluster/eventos.go:23-24`, `internal/cluster/eventos_test.go` y llamadores `cmd/benchmark/cluster.go:215`, `cmd/coordinator/api.go:98`
- **Evidencia:**
  1. `internal/cluster/eventos.go:23-24` documenta dos eventos terminales excluyentes: `EvAbortado = "abortado" // el entrenamiento falló` y `EvFin = "fin" // el entrenamiento terminó bien`.
  2. Sin embargo, `(*Coordinador).Terminar()` (`coordinador.go:576-581`) envía `Fin(MotivoCompleto)` y emite `EvFin` incondicionalmente, sin comprobar `c.fallo != nil` ni si `Terminar()` ya fue llamado antes. En `cmd/benchmark/cluster.go:215` (`defer c.Terminar()`) y `cmd/coordinator/api.go:98` (`c.Terminar()` durante el apagado), si el entrenamiento falló y `abortar()` ya emitió `EvAbortado`, la secuencia emitida termina con `EvAbortado` seguido de un `EvFin` contradictorio.
  3. En `internal/cluster/eventos_test.go` no existe ninguna prueba que ejercite `EvAbortado` ni que compruebe que mutar `e.Nodos` dentro del observador no altera `c.Estado().Nodos` (solo se muta `e.Pesos` en `recolector.observar`, `eventos_test.go:23-30`).
- **Corrección sugerida:**
  - Guardar un estado `terminado bool` (protegido por `muBucle`) en `Coordinador` y hacer que `Terminar()` retorne sin efecto si `c.fallo != nil` o `c.terminado` ya es `true` (marcando `c.terminado = true` antes de emitir `EvFin`).
  - Añadir un test en `eventos_test.go` donde caigan todos los nodos → verificar que se emite `EvDescarte` y `EvAbortado` con `Detalle != ""`, y que llamar `c.Terminar()` después no emite `EvFin` (y que llamar `c.Terminar()` dos veces en un entrenamiento exitoso emite un solo `EvFin`). Mutar también `e.Nodos` en `recolector.observar` para validar la copia de `Nodos`.

### H4 — Robustez en `medirObservador` / `ejecutarObservador`, divergencia de nombre `tarea` vs `asignacion` respecto de #64 y precisión de `proyeccion` · MENOR

- **Dónde:** `cmd/benchmark/cluster.go:397-454`, `internal/cluster/eventos.go:17` y `internal/cluster/eventos_test.go:165-182`
- **Evidencia:**
  1. En `cmd/benchmark/cluster.go:399`, `medirObservador` llama a `medirCluster(train, op)` sin poner antes `op.observador = nil`; si un llamador pasa `op.observador != nil`, la corrida de control «sin observador» también ejecutaría el observador.
  2. En `cmd/benchmark/cluster.go:433`, `Relativo: (b - a) / a` no guarda contra `a == 0` (produciría `NaN`/`+Inf`, que falla `json.MarshalIndent` en `escribir`); en `:418-422` se ignora en silencio cualquier error de `json.Marshal(e)`; y `ejecutarObservador` (`:445-449`) omite en `Protocolo` los campos `workers_por_nodo`, `tasa`, `lambda`, `calentamiento` y `estimador` que sí documenta `ejecutarCluster` (`:357-366`).
  3. El criterio 2 del issue #64 lista el evento `asignacion`, mientras `eventos.go:17` lo denomina `tarea` (emitido en cada despacho de shard por época, incluya o no la asignación inicial); conviene dejar nota explícita del cambio de nombre en el PR/issue para alinear a #65 y #69.
  4. En `internal/cluster/eventos_test.go:165-182`, `proyeccion` no verifica `sc.Err()` al salir de `for sc.Scan()`, formatea `e.Pesos` con `%v` en vez de `math.Float64bits`, y omite `e.Detalle` y los campos deterministas de `e.Nodos` (`Nombre`, `Workers`, `Sano`, `Carga`, `Motivo`).
- **Corrección sugerida:** Poner `op.observador = nil` al inicio de `medirObservador`, proteger `a > 0` en `Relativo`, contabilizar o fallar ante error de `json.Marshal`, completar `rep.Protocolo` en `ejecutarObservador`, comprobar `sc.Err()` y comparar los bits de `e.Pesos` en `proyeccion`, y documentar el nombre `tarea` respecto de #64.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-81. |

## Cómo explicarlo en la sustentación

El coordinador ya concentraba todo el estado del entrenamiento en una sola goroutine (el bucle de épocas protegido por `muBucle`), por lo que emite cada evento del protocolo (`registro`, `tarea`, `gradiente`, `epoca`, `reasignacion`, `descarte`, `abortado`, `fin`) con esquema versionado `v: 1` directamente en orden desde esa goroutine. Para garantizar que observar el entrenamiento en vivo no altere el resultado numérico (`AGENTS.md` §3.3), el observador solo recibe copias de los pesos y del estado de los nodos —probado bit a bit con 1 y 3 nodos incluso destruyendo los pesos recibidos con `NaN`— y nunca bloquea: `Grabador` (JSONL) y `ws.Hub` encolan en canales con búfer y descartan si se llenan. Una vez generado el reporte real de sobrecosto sobre `data/gold/train.bin` (**H1**), eliminado el riesgo de pánico al cerrar `Grabador` (**H2**) y asegurada la exclusión mutua entre `EvAbortado` y `EvFin` en `Terminar` (**H3**), la capa de eventos quedará lista para conectarse al WebSocket de la API en #65.
