# Veredicto — PR-84 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/65-ws-ticket-demo` @ `00f25b4` (base: `integracion/tb2` @ `99d25d6`)
- **Fecha:** 2026-11-22
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H4**) quedaron resueltos y verificados con evidencia reproducible en los commits `6cda7a8`, `7b0b631` y `8618835`, sin introducir defectos ni regresiones (`AGENTS.md` §3.2–§3.6): `CortarNodo` retira el nombre de `c.nombres` bajo `c.muConns` antes de cerrar la conexión y salta `net.ErrClosed`, eliminando por completo los HTTP 500 en cortes repetidos, concurrentes o tras la reconexión automática de `cmd/node` (**H1**, verificado en test con `-race` y en vivo contra binarios compilados con `-reconexiones=0` y `-reconexiones=3`); `servirWS` suscribe al cliente en `ws.Hub` *antes* de `ws.Aceptar` (y desuscribe si el handshake falla) y `TestServicioDemoCortarNodoPorLaAPI` acepta los `EvRegistro` en cualquier orden, pasando 20/20 corridas bajo `GOMAXPROCS=1 go test -race -count=20` junto con 20/20 de `TestWSSuscritoAntesDelHandshake` (**H2**); `descartarCuerpo` aplica el límite `MaxCuerpo` (64 KiB → `413`) en `emitirTicket` y `cortarNodo` preservando CORS y manteniendo 401/403 antes que 413, y `conTicket` fija `Cache-Control: no-store` en toda respuesta del handshake (**H3**); y la colección de Postman lee `pm.variables.get("demo")` respetando `--env-var "demo=true"` (**H4**). Toda la verificación local (`gofmt`, `go vet`, `-tags integracion`, 25/25 paquetes Go con `-race`, 0 carreras, equivalencia secuencial = distribuido bit a bit, p95 < 100 ms con JWT, `go run ./cmd/informe` sin diff) y los 7/7 jobs de GitHub Actions sobre `00f25b4` están en verde.

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/api ./internal/cluster ./internal/ws ./cmd/coordinator
internal/api     -> solo stdlib + paquetes internos (docs/api, almacen, auth, cluster, features, ingest, ml, ws)
internal/cluster -> solo stdlib + paquetes internos (protocolo, gold, ml)
internal/ws      -> solo stdlib
cmd/coordinator  -> solo stdlib + paquetes internos (almacen/mongo y almacen/redis permitidos por ADR-0003)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.739s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.972s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.725s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.029s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.834s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.277s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.231s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.416s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.415s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.053s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.156s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	27.711s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.897s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	10.182s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.955s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.082s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.133s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.073s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.195s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.534s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.754s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (1.68s) [nodos=1,2,3,5]
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.26s)
--- PASS: TestEquivalencia (0.36s) [workers=1,2,4,8,16,64]

$ go test -race -count=1 -run 'Ticket|WSOrigen|WSCerrar|WSSuscrito|Demo|MatrizRoles|CabecerasLimiteDeCuerpo413|OpenAPISeguridad' -v ./internal/api
PASS (14/14: TestMatrizRoles [14 rutas × 3 roles, incl. ticket y demo], TestMatrizRolesTokenAjenoYSinFirmador,
TestCabecerasLimiteDeCuerpo413 [incl. /ws/ticket y /demo/nodos/{nombre}/cortar], TestOpenAPISeguridad,
TestMatrizRolesCoincideConADR, TestMatrizRolesRutasDevuelveCopias, TestTicketSoloAdminUnUsoYVence,
TestTicketTopeDePendientesY429, TestTicketConsumoAtomico, TestWSOrigenFueraDeLaListaDa403SinGastarElTicket,
TestWSCerrarEnvia1001, TestDemoSinFlagNoExiste, TestDemoCortarNodoYRitmo, TestWSSuscritoAntesDelHandshake)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.524s

$ go test -race -count=1 -run 'CortarNodo' -v ./internal/cluster
--- PASS: TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial (0.21s)
--- PASS: TestCortarNodoEnReposoConcurrenteYTrasReconectar (0.11s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	1.335s

$ go test -race -count=1 -run 'Demo' -v ./cmd/coordinator
--- PASS: TestServicioDemoCortarNodoPorLaAPI (0.95s)
--- PASS: TestServicioSinDemoNoExponeElCorte (0.02s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.007s

$ GOMAXPROCS=1 go test -race -count=20 -failfast -timeout=180s -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	21.085s
$ GOMAXPROCS=1 go test -race -count=20 -failfast -timeout=60s -run TestWSSuscritoAntesDelHandshake ./internal/api
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	6.923s
$ GOMAXPROCS=1 go test -race -count=20 -failfast -timeout=60s -run TestCortarNodoEnReposoConcurrenteYTrasReconectar ./internal/cluster
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	2.395s

$ go test -race -count=1 -run 'Fragmentos' -v ./cmd/informe && go run ./cmd/informe && git status --short
--- PASS: TestFragmentosSalenDelFuente (0.16s)
(git status --short vacío: sin drift en docs/informe/pc4/10-api.md)

$ API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config | grep -E "DEMO|CORS_ORIGENES|API_URL"
      CORS_ORIGENES: http://localhost:8081
      DEMO: "false"
      API_URL: http://localhost:8080
$ API=:8080 DEMO=true docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config | grep "DEMO:"
      DEMO: "true"

$ <pruebas en vivo contra binarios compilados cmd/coordinator -demo + 2 procesos cmd/node>
- [PROBE 1] DEMO=invalido -> rc=1, stderr='DEMO="invalido": use true o false' (falla rápido ✓)
- [PROBE 2] POST /api/v1/ws/ticket (70KB) -> HTTP 413, ACAO=http://localhost:8081, body={"error":"el cuerpo supera 65536 bytes"} (✓)
- [PROBE 3] POST /api/v1/demo/nodos/no-existe/cortar (70KB) -> HTTP 413, ACAO=http://localhost:8081, body={"error":"el cuerpo supera 65536 bytes"} (✓)
- [PROBE 4] WS 401 Cache-Control=no-store | WS 403 Cache-Control=no-store (✓)
- [PROBE 5a] 1st cut nodo-2 (idle, -reconexiones=0) -> HTTP 202, body={"estado":"cortado","nodo":"nodo-2"} (✓)
- [PROBE 5b] 2nd cut nodo-2 (idle, -reconexiones=0) -> HTTP 404, body={"error":"no hay un nodo conectado llamado \"nodo-2\""} (✓)
- [PROBE 6] nodo-1 (-reconexiones=3) 6 sequential cuts -> [202, 202, 202, 202, 404, 404] (4 conexiones cortadas + 2×404, 0×500 ✓)
- [PROBE 7] cmd/benchapi -correo analista@vigia.pe -n 500 -c 8 -exigir-p95 100ms:
  fria: 500 pet., 0 err., p50=0.34 ms, p95=1.58 ms (cumple=True) · caliente: 500 pet., 0 err., p50=0.50 ms, p95=2.16 ms (cumple=True)
- [PROBE 8] SIGTERM with active WS: opcode=0x8, close_code=1001, coord_rc=0, elapsed=0.002s (✓)

$ gh pr checks 84   # run 44993292453 sobre 00f25b4
Docker (imagen, clúster y caída de un nodo)	pass	1m55s
Ecosistema de agentes                       	pass	9s
Go (gofmt, vet, test -race)                 	pass	1m1s
MongoDB y Redis (tests de integración)      	pass	55s
OpenAPI 3 válido                            	pass	21s
SPA (tipos, tests, build e imagen)          	pass	54s
Spin (modelo del protocolo y mutantes)      	pass	1m59s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes PASS + 20/20 bajo `GOMAXPROCS=1 -race` en E2E, handshake WS y `CortarNodo`) |
| CI GitHub Actions sobre `00f25b4` | ✅ (7/7 jobs PASS, run `44993292453`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia`, `TestEquivalenciaClusterSecuencialBitABit`, `TestEquivalenciaConMenosNodosQueShards`, `TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial` y `TestServicioDemoCortarNodoPorLaAPI`, bit a bit) |
| Latencia API p95 < 100 ms | ✅ (verificado en vivo con `cmd/benchapi` + JWT sobre binario con `-demo`: p95 1,58 ms frío / 2,16 ms caliente, 0 errores) |

## Verificación contra los criterios del issue #65

| Criterio (#65) | Estado | Evidencia |
|---|---|---|
| 1. `POST /api/v1/ws/ticket` (admin) → ticket de un solo uso que vence a los 30 s; reuso o vencido → 401; analista → 403 | ✅ | `TestTicketSoloAdminUnUsoYVence`, `TestTicketConsumoAtomico` (64 goroutines → 1 válido), `TestTicketTopeDePendientesY429`, `TestMatrizRoles` y `TestCabecerasLimiteDeCuerpo413` PASS; 70 KB → 413 con CORS verificado en vivo. |
| 2. `GET /api/v1/ws` rechaza `Origin` fuera de la lista de CORS con 403 | ✅ | `TestWSOrigenFueraDeLaListaDa403SinGastarElTicket`, `TestWSSuscritoAntesDelHandshake` (20×20 bajo `GOMAXPROCS=1 -race`) y `TestWSCerrarEnvia1001` PASS; `Cache-Control: no-store` en 401/403 verificado en test y en vivo. |
| 3. `ritmo` y `POST /api/v1/demo/nodos/{nombre}/cortar` solo existen con `-demo` (sin el flag → 404) | ✅ | `TestDemoSinFlagNoExiste`, `TestDemoCortarNodoYRitmo`, `TestServicioSinDemoNoExponeElCorte`, `TestServicioDemoCortarNodoPorLaAPI` (20/20 bajo `GOMAXPROCS=1 -race`) y `TestCortarNodoEnReposoConcurrenteYTrasReconectar` PASS; doble corte y reconexiones verificados en vivo sin 500. |
| 4. Cortar un nodo en la época k → el shard se reasigna y los pesos finales son iguales al secuencial | ✅ | `TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial` y `TestServicioDemoCortarNodoPorLaAPI` PASS (comparan pesos, sesgo e historia bit a bit). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | `ADR-0007` completo (incluye modo demo, tickets y orden del handshake) y `TestMatrizRolesCoincideConADR` con `pendientesADR` vacío ✅. |
| Implementación | 8 | Cierra tiempo real WS y modo demo | 100 % stdlib en el núcleo, 0 carreras de datos, cierre ordenado `1001` en 2 ms, suscripción sin pérdida de eventos tras `101`, corte robusto ante doble clic y reconexiones, límite `413` + `no-store` consistentes, equivalencia bit a bit intacta y p95 < 100 ms con JWT. |
| Interfaz | 5 | Habilita el panel en vivo (#69) | Contrato WS (ticket + `Origin` + eventos JSON sin pérdida tras `onopen`) y botón «Tumbar nodo» (`202`/`404` sin 500) listos para conectar desde la SPA. |
| Informe | 2 | Sincronizado sin cifras a mano | `docs/informe/pc4/10-api.md` regenerado y verificado (`TestFragmentosSalenDelFuente` PASS, `go run ./cmd/informe` sin diff). |
| Video | 2 | Habilita la demostración en vivo | `DEMO=true` + `ritmo_ms` + corte limpio de nodo permiten grabar y sustentar la caída y reasignación en tiempo real sin fallos ni sorpresas. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `6cda7a8` actualiza `CortarNodo` (`internal/cluster/coordinador.go:250-279`) para retirar `delete(c.nombres, id)` bajo `c.muConns` antes de soltarlo (conservando `c.conns[id]` para `Tiempos()` y `Cerrar()`), iterar en bucle saltando `errors.Is(err, net.ErrClosed)` y registrar el log solo si `err == nil`; añade `TestCortarNodoEnReposoConcurrenteYTrasReconectar` (`internal/cluster/eventos_test.go:390-441`, 20/20 con `GOMAXPROCS=1 -race`). Verificado en vivo: doble corte en reposo (`-reconexiones=0`) → `202` y `404`; 6 cortes sobre nodo con `-reconexiones=3` → `[202, 202, 202, 202, 404, 404]` (0 errores `500`). |
| H2 (ronda 1 · BLOQUEANTE) | RESUELTO | `7b0b631` mueve `s.op.Hub.Suscribir()` antes de `ws.Aceptar` en `servirWS` (`internal/api/tiemporeal.go:158-168`, con `Desuscribir` si el handshake falla y cierre `1001` en `Hub.Servir` si el hub se cierra entretanto), agrega `TestWSSuscritoAntesDelHandshake` (`internal/api/tiemporeal_test.go:353-369`, 20/20 corridas de 20 handshakes bajo `GOMAXPROCS=1 -race`) y acumula `EvRegistro` en un mapa sin orden fijo en `TestServicioDemoCortarNodoPorLaAPI` (`cmd/coordinator/demo_test.go:107-113`), pasando 20/20 en `GOMAXPROCS=1 go test -race -count=20` (21,09 s). |
| H3 (ronda 1 · MAYOR) | RESUELTO | `7b0b631` agrega `descartarCuerpo` (`internal/api/tiemporeal.go:193-205`) invocado al inicio de `emitirTicket` (`:139`) y `cortarNodo` (`:172`) tras la autenticación (401/403 antes que 413), incluye ambas rutas en `TestCabecerasLimiteDeCuerpo413` (`internal/api/seguridad_test.go:329-330`) y fija `Cache-Control: no-store` al inicio de `conTicket` (`tiemporeal.go:114`, verificado en `TestTicketSoloAdminUnUsoYVence` en `tiemporeal_test.go:162-171`). Confirmado en vivo: 70 KB → `413` con `Access-Control-Allow-Origin` en ambos `POST`, y `Cache-Control: no-store` en WS `401` y `403`. |
| H4 (ronda 1 · MENOR) | RESUELTO | `8618835` cambia `pm.collectionVariables.get("demo")` por `pm.variables.get("demo")` en `docs/api/cc65-tf.postman_collection.json:606`, respetando `--env-var "demo=true"` / Environment y cayendo al valor de colección `"false"` por defecto; JSON válido y cubierto por `TestColeccionPostmanCubreLasRutas`. |

## Cómo explicarlo en la sustentación

El navegador no puede añadir `Authorization: Bearer` al handshake del WebSocket; por eso la SPA (con JWT de admin) solicita un ticket aleatorio de 32 bytes y un solo uso en `POST /api/v1/ws/ticket` (válido 30 s, tope de 128 pendientes) y abre `GET /api/v1/ws?ticket=…`. El servidor aplica `Cache-Control: no-store`, valida primero el `Origin` contra `CORS_ORIGENES` (`403` sin consumir el ticket para impedir *cross-site WebSocket hijacking*), consume el ticket de forma atómica bajo mutex (`401` si no existe, venció o ya se usó) y verifica el rol admin (`403`); luego suscribe al cliente en `ws.Hub` *antes* de enviar `101 Switching Protocols` para no perder eventos tras `onopen`, recibe cada `cluster.Evento` en JSON sin bloquear el entrenamiento y despide a los clientes con cierre `1001` ante `SIGTERM`. Cuando el coordinador arranca con `-demo` (`DEMO=true`), habilita `ritmo_ms` (pausa entre épocas fuera del bucle del coordinador) y `POST /api/v1/demo/nodos/{nombre}/cortar`, que retira atómicamente el nombre del nodo y cierra su socket TCP (`202` sobre conexión viva, `404` si ya no existe, incluso ante doble clic o reconexiones) para que la caída y la reasignación sigan el camino normal del protocolo manteniendo los pesos idénticos bit a bit al secuencial.
