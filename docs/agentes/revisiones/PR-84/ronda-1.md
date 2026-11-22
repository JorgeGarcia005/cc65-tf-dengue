# Veredicto — PR-84 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/65-ws-ticket-demo` @ `11aaa88` (base: `integracion/tb2` @ `99d25d6`)
- **Fecha:** 2026-11-22
- **Decisión:** CAMBIOS

El PR conecta el hub del WebSocket (`internal/ws`) al observador del coordinador (`cmd/coordinator/api.go`) e incorpora el endpoint de emisión de tickets (`POST /api/v1/ws/ticket`), el handshake protegido por `Origin` → consumo atómico de ticket → rol (`GET /api/v1/ws`), el cierre ordenado con código `1001` al recibir `SIGTERM` y el modo demo (`-demo` / `DEMO=true`: `POST /api/v1/demo/nodos/{nombre}/cortar`, `cluster.Coordinador.CortarNodo` y pausa `ritmo_ms` entre épocas fuera del bucle de cálculo). Todo el núcleo permanece en la biblioteca estándar (`AGENTS.md` §3.2), `gofmt`, `go vet`, los 25 paquetes Go con `-race` (0 carreras de datos), la equivalencia secuencial = distribuido bit a bit (incluso tras cortar un nodo), el p95 < 100 ms con JWT y los 7/7 jobs de CI están en verde. Sin embargo, se emiten **2 hallazgos bloqueantes**, **1 mayor** y **1 menor** antes de aprobar: `CortarNodo` no retira el nombre de `c.nombres` ni trata `net.ErrClosed`, devolviendo **HTTP 500** (`use of closed network connection`) al cortar dos veces en reposo, en concurrencia o tras la reconexión automática de `cmd/node` (**H1**); `servirWS` suscribe al cliente en el `Hub` *después* de enviar `101 Switching Protocols` y `TestServicioDemoCortarNodoPorLaAPI` asume un orden fijo de registro, fallando de forma reproducible bajo `GOMAXPROCS=1 go test -race -count=20` (**H2**); los dos nuevos `POST` declaran `413` en OpenAPI pero aceptan cuerpos de 70 KB sin rechazarlos y `conTicket` omite `Cache-Control: no-store` (**H3**); y la colección de Postman lee `pm.collectionVariables.get("demo")` en lugar de `pm.variables.get("demo")`, ignorando `--env-var "demo=true"` en Newman (**H4**).

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
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.833s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.713s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.817s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.281s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.254s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.418s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.411s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.057s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.148s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.057s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	34.580s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	18.778s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.558s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.990s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.040s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.112s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.088s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.876s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.674s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.769s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalencia (0.53s) [workers=1,2,4,8,16,64]

$ go test -race -count=1 -run 'Ticket|WSOrigen|WSCerrar|Demo|MatrizRoles|OpenAPISeguridad' -v ./internal/api
PASS (12/12: TestMatrizRoles [14 rutas × 3 roles, incl. ticket], TestMatrizRolesTokenAjenoYSinFirmador,
TestOpenAPISeguridad, TestMatrizRolesCoincideConADR, TestMatrizRolesRutasDevuelveCopias,
TestTicketSoloAdminUnUsoYVence, TestTicketTopeDePendientesY429, TestTicketConsumoAtomico,
TestWSOrigenFueraDeLaListaDa403SinGastarElTicket, TestWSCerrarEnvia1001,
TestDemoSinFlagNoExiste, TestDemoCortarNodoYRitmo)

$ go test -race -count=1 -run 'CortarNodo' -v ./internal/cluster
--- PASS: TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial (0.17s)

$ go test -race -count=1 -run 'Demo' -v ./cmd/coordinator
--- PASS: TestServicioDemoCortarNodoPorLaAPI (0.96s)
--- PASS: TestServicioSinDemoNoExponeElCorte (0.02s)

$ GOMAXPROCS=1 go test -race -count=20 -failfast -timeout=60s -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator
--- FAIL: TestServicioDemoCortarNodoPorLaAPI (30.09s)
    demo_test.go:106: no llegó el registro de los nodos por el WebSocket
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	41.187s   # ver H2

$ go test -race -count=1 -run 'Fragmentos' -v ./cmd/informe
--- PASS: TestFragmentosSalenDelFuente (0.17s)

$ API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config | grep -E "DEMO|CORS_ORIGENES|API_URL"
      CORS_ORIGENES: http://localhost:8081
      DEMO: "false"
      API_URL: http://localhost:8080
$ API=:8080 DEMO=true docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config | grep "DEMO:"
      DEMO: "true"

$ <pruebas en vivo contra binarios compilados cmd/coordinator -demo + 2 procesos cmd/node>
- DEMO=invalido: rc=1, stderr='DEMO="invalido": use true o false' (falla rápido ✓)
- POST /api/v1/ws/ticket (70 KB): HTTP 200 (emite ticket en vez de 413; ver H3)
- POST /api/v1/demo/nodos/no-existe/cortar (70 KB): HTTP 404 (en vez de 413; ver H3)
- GET /api/v1/ws sin ticket (401) y Origin ajeno (403): Cache-Control ausente (ver H3)
- Doble corte en reposo sobre nodo-2 (-reconexiones=0): 1.º → 202; 2.º → HTTP 500 use of closed network connection (ver H1)
- Cortes sobre nodo-1 (-reconexiones=3, por defecto en Compose): 1.º → 202; siguientes 4 → 202, 500, 500, 500 (ver H1)
- cmd/benchapi -url http://127.0.0.1:18091 -correo analista@vigia.pe -n 500 -c 8 -exigir-p95 100ms:
  fria: 500 pet., 0 err., p50=0.61 ms, p95=1.92 ms (cumple=True) · caliente: 500 pet., 0 err., p50=0.54 ms, p95=1.57 ms (cumple=True)
- SIGTERM con cliente WS activo: opcode=0x8, close_code=1001, coord_rc=0, elapsed=0.004s (✓)

$ gh pr checks 84   # run 44992031806 sobre 11aaa88
Docker (imagen, clúster y caída de un nodo)	pass	1m47s
Ecosistema de agentes                       	pass	10s
Go (gofmt, vet, test -race)                 	pass	1m3s
MongoDB y Redis (tests de integración)      	pass	46s
OpenAPI 3 válido                            	pass	20s
SPA (tipos, tests, build e imagen)          	pass	46s
Spin (modelo del protocolo y mutantes)      	pass	2m5s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ⚠️ 25/25 paquetes PASS en corrida normal (carreras de datos: 0), pero `TestServicioDemoCortarNodoPorLaAPI` falla con `GOMAXPROCS=1 go test -race -count=20` por carrera de orden/suscripción (**H2**) |
| CI GitHub Actions (7 jobs) | ✅ (run `44992031806`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia`, `TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial` y `TestServicioDemoCortarNodoPorLaAPI`, bit a bit) |
| Latencia API p95 < 100 ms | ✅ (verificado en vivo con `cmd/benchapi` + JWT sobre binario con `-demo`: p95 1,92 ms / 1,57 ms) |

## Verificación contra los criterios del issue #65

| Criterio (#65) | Estado | Evidencia |
|---|---|---|
| 1. `POST /api/v1/ws/ticket` (admin) → ticket de un solo uso que vence a los 30 s; reuso o vencido → 401; analista → 403 | ✅ | `TestTicketSoloAdminUnUsoYVence`, `TestTicketConsumoAtomico` (64 goroutines → 1 válido), `TestTicketTopeDePendientesY429` y `TestMatrizRoles` PASS. *(Detalle de límite 413 en **H3**.)* |
| 2. `GET /api/v1/ws` rechaza `Origin` fuera de la lista de CORS con 403 | ⚠️ | `TestWSOrigenFueraDeLaListaDa403SinGastarElTicket` PASS (403 sin quemar el ticket), pero `servirWS` suscribe al `Hub` después de escribir `101` (**H2**) y `conTicket` omite `Cache-Control: no-store` (**H3**). |
| 3. `ritmo` y `POST /api/v1/demo/nodos/{nombre}/cortar` solo existen con `-demo` (sin el flag → 404) | ⚠️ | `TestDemoSinFlagNoExiste`, `TestDemoCortarNodoYRitmo` y `TestServicioSinDemoNoExponeElCorte` PASS, pero cortar un nodo dos veces en reposo/concurrencia o tras reconexión de `cmd/node` devuelve **500** (**H1**) y `TestServicioDemoCortarNodoPorLaAPI` es flaky (**H2**). |
| 4. Cortar un nodo en la época k → el shard se reasigna y los pesos finales son iguales al secuencial | ✅ | `TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial` PASS (compara pesos e historia bit a bit). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | `ADR-0007` documenta con claridad el modo demo, los tickets y el orden del handshake; `TestMatrizRolesCoincideConADR` con `pendientesADR` vacío ✅. |
| Implementación | 8 | Parcial (pendiente **H1–H3**) | Diseño limpio en stdlib (0 carreras de datos, cierre `1001` verificado en 4 ms, equivalencia bit a bit intacta), pero **H1** produce HTTP 500 en el flujo real de Compose con `cmd/node` y **H2** introduce carrera de pérdida de eventos tras `101` + test E2E flaky. |
| Interfaz | 5 | Habilita el panel en vivo (#69) | El contrato WS + ticket + demo queda listo para la SPA, pero arreglar **H1** (doble clic / reconexión) y **H2** (no perder eventos tras `onopen`) es clave para una demostración sin fallos. |
| Informe | 2 | sin cambios | `docs/informe/pc4/10-api.md` regenerado correctamente (`TestFragmentosSalenDelFuente` ✅). |
| Video | 2 | Habilita la demo en vivo | Permite demostrar caída y reasignación de nodo con `ritmo_ms`; **H1** y **H2** evitan sorpresas al grabar o sustentar. |

## Hallazgos

### H1 — `CortarNodo` no retira el nombre de `c.nombres` ni trata `net.ErrClosed`: HTTP 500 al cortar dos veces en reposo, en concurrencia o tras reconexión de `cmd/node` · BLOQUEANTE

- **Dónde:** `internal/cluster/coordinador.go:212-239, 250-265` (`leer`, `CortarNodo`), `internal/api/tiemporeal.go:160-170` (`cortarNodo`), `internal/cluster/eventos_test.go:378`
- **Evidencia:**
  1. `CortarNodo` (`coordinador.go:250-265`) busca `nombre` en `c.nombres`, obtiene `pc = c.conns[id]`, suelta `c.muConns` y llama a `pc.Cerrar()`, pero **no elimina `id` de `c.nombres`**. La limpieza solo ocurre en el `defer` de `c.leer` (`coordinador.go:212-221`), que únicamente corre **después** de que `c.entregar(evento{id: id, msg: m, err: err})` (`coordinador.go:235`) sea consumido del canal sin búfer `c.eventos`.
  2. Cuando el servicio está en reposo (sin entrenamiento activo) o en la ventana entre `pc.Cerrar()` y el `defer` de `leer` durante un entrenamiento, `c.nombres[id]` sigue apuntando a la conexión ya cerrada. Una segunda llamada a `CortarNodo(nombre)` (doble clic en la SPA, dos peticiones concurrentes o repetir el corte) vuelve a llamar `pc.Cerrar()` sobre el socket cerrado, que devuelve `net.ErrClosed`, y `cortarNodo` (`tiemporeal.go:167-169`) responde **HTTP 500** filtrando la dirección TCP interna.
  3. Más grave aún en el despliegue real de Docker Compose: `cmd/node` corre con `-reconexiones=3` por defecto (`cmd/node/main.go:40, 56-71`). Al cortar `nodo-1`, el contenedor se reconecta de inmediato con el mismo nombre (`id` nuevo), mientras la entrada vieja sigue en `c.nombres`. Como `for id, n := range c.nombres` itera un mapa en orden aleatorio, cortes posteriores caen sobre la conexión ya cerrada (500) y dejan viva la nueva.
  4. Reproducción en vivo contra el binario compilado `cmd/coordinator -api 127.0.0.1:18091 -demo` + 2 `cmd/node`:
     ```text
     [PROBE 5a] 1st cut nodo-2 (idle, -reconexiones=0) -> HTTP 202, body={"estado":"cortado","nodo":"nodo-2"}
     [PROBE 5b] 2nd cut nodo-2 (idle, -reconexiones=0) -> HTTP 500, body={"error":"no se pudo cortar el nodo: close tcp 127.0.0.1:19091->127.0.0.1:42890: use of closed network connection"}
     [PROBE 6]  nodo-1 (idle, -reconexiones=3) 1st cut  -> HTTP 202 {"estado":"cortado","nodo":"nodo-1"}
                next 4 cuts -> [(202, ...), (500, "...use of closed network connection"), (500, "..."), (500, "...")]
     ```
  5. `TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial` (`eventos_test.go:378`) no detectó el fallo porque solo repite `CortarNodo("nodo-2")` al finalizar todo el entrenamiento, cuando `leer` ya había salido de `c.entregar` y ejecutado su `defer`.
- **Corrección sugerida:**
  - En `CortarNodo` (`internal/cluster/coordinador.go:250-265`), iterar en bucle: bajo `c.muConns.Lock()`, localizar `id` con `n == nombre`, guardar `pc = c.conns[id]` y hacer **`delete(c.nombres, id)`** antes de soltar el mutex (conservando `c.conns[id]` para que `Tiempos()` y `Cerrar()` no pierdan la conexión); fuera del mutex, llamar `err := pc.Cerrar()`. Si `errors.Is(err, net.ErrClosed)` (p. ej. porque `descartar` o `Cerrar` cerró `pc` en paralelo), continuar el bucle para buscar otra conexión viva con ese nombre o terminar devolviendo `ErrNodoDesconocido`. Registrar el log solo cuando `pc.Cerrar()` devuelve `nil`.
  - Añadir tests con `-race` en `internal/cluster/eventos_test.go`: (a) dos llamadas seguidas en reposo y llamadas concurrentes con el mismo nombre → exactamente una devuelve `nil` y el resto `ErrNodoDesconocido`; (b) nodo que se reconecta con el mismo nombre tras ser cortado → el siguiente `CortarNodo` corta la nueva conexión sin devolver `net.ErrClosed`.

### H2 — Carrera de suscripción tras `101` en `servirWS` y test E2E flaky `TestServicioDemoCortarNodoPorLaAPI` · BLOQUEANTE

- **Dónde:** `internal/api/tiemporeal.go:151-157` (`servirWS`) y `cmd/coordinator/demo_test.go:103-109` (`TestServicioDemoCortarNodoPorLaAPI`)
- **Evidencia:**
  1. En `internal/api/tiemporeal.go:151-157`:
     ```go
     conn, err := ws.Aceptar(w, r, nil) // ya escribe HTTP/1.1 101 Switching Protocols
     if err != nil {
         return
     }
     s.op.Hub.Servir(conn, s.op.Hub.Suscribir(), IntervaloPingWS)
     ```
     El cliente se suscribe al `Hub` **después** de que el servidor ya envió `101 Switching Protocols`. Un cliente (la SPA al dispararse `ws.onopen`, o un test tras `eventosWS`) que inmediatamente lance un entrenamiento o registre nodos puede perder los primeros eventos emitidos entre el `101` y `Hub.Suscribir()`.
  2. En `cmd/coordinator/demo_test.go:103-109`, inmediatamente después de leer el `101`, el test lanza 3 goroutines concurrentes (`lanzarNodos`, `cmd/coordinator/main_test.go:49-64`) y espera sus `EvRegistro` en orden fijo `nodo-0`, `nodo-1`, `nodo-2`:
     ```go
     for i := range 3 {
         esperarEvento(t, evs, "el registro de los nodos", func(e cluster.Evento) bool {
             return e.Tipo == cluster.EvRegistro && e.Nodo == fmt.Sprintf("nodo-%d", i)
         })
     }
     ```
     Si `nodo-1` o `nodo-2` se registra antes que `nodo-0` (o si algún registro ocurre antes de `Hub.Suscribir()`), `esperarEvento` descarta los eventos que no coinciden con `i=0` y luego la iteración siguiente espera 30 s un evento ya descartado.
  3. Reproducción directa con `GOMAXPROCS=1`:
     ```text
     $ GOMAXPROCS=1 go test -race -count=20 -failfast -timeout=60s -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator
     --- FAIL: TestServicioDemoCortarNodoPorLaAPI (30.09s)
         demo_test.go:106: no llegó el registro de los nodos por el WebSocket
     FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	41.187s
     ```
- **Corrección sugerida:**
  - En `internal/api/tiemporeal.go:151-157` (`servirWS`), suscribir **antes** de `ws.Aceptar` y desuscribir si el handshake falla:
    ```go
    cliente := s.op.Hub.Suscribir()
    conn, err := ws.Aceptar(w, r, nil)
    if err != nil {
        s.op.Hub.Desuscribir(cliente)
        return
    }
    s.op.Hub.Servir(conn, cliente, IntervaloPingWS)
    ```
    Así, al llegar el `101` al cliente, ya está en `Hub` y no pierde eventos (y si `Hub.Cerrar()` corrió entretanto, `Hub.Servir` cierra con `1001`).
  - En `cmd/coordinator/demo_test.go:105-109`, acumular los nodos registrados en un `map[string]bool` aceptando los 3 `EvRegistro` en cualquier orden hasta recibir `nodo-0`, `nodo-1` y `nodo-2`. Verificar con `GOMAXPROCS=1 go test -race -count=20 -run TestServicioDemoCortarNodoPorLaAPI ./cmd/coordinator`.

### H3 — Los dos nuevos `POST` declaran `413` en OpenAPI pero nunca rechazan cuerpos > 64 KiB, y `conTicket` omite `Cache-Control: no-store` · MAYOR

- **Dónde:** `internal/api/tiemporeal.go:109-126, 134-147, 160-174`, `docs/api/openapi.yaml:18, 264, 319`, `internal/api/seguridad_test.go:325-347`
- **Evidencia:**
  1. `docs/api/openapi.yaml:18` establece *«Todo POST admite como máximo 64 KiB (413)»*, y el PR declara `"413": {$ref: "#/components/responses/Error"}` tanto en `POST /api/v1/ws/ticket` (`openapi.yaml:264`) como en `POST /api/v1/demo/nodos/{nombre}/cortar` (`openapi.yaml:319`).
  2. Como `emitirTicket` (`tiemporeal.go:134-147`) y `cortarNodo` (`tiemporeal.go:160-174`) nunca leen `r.Body`, el `http.MaxBytesReader` puesto por `cabecerasSeguridad` (`seguridad.go:107`) no se dispara jamás. Reproducido en vivo contra `cmd/coordinator -demo` con un cuerpo de 70 000 bytes `'a'` y JWT de admin:
     ```text
     [PROBE 2] POST /api/v1/ws/ticket (70KB) -> HTTP 200, body={"ticket":"...","expira":...}
     [PROBE 3] POST /api/v1/demo/nodos/no-existe/cortar (70KB) -> HTTP 404, body={"error":"no hay un nodo conectado llamado \"no-existe\""}
     ```
     (Sobre un nodo existente devuelve `202` y ejecuta el corte.) `TestCabecerasLimiteDeCuerpo413` (`seguridad_test.go:325-341`) solo prueba `/predicciones`, `/entrenamientos` y `/auth/login`, dejando fuera los dos nuevos `POST`.
  3. Además, en `conTicket` (`tiemporeal.go:109-126`), las respuestas HTTP `401` y `403` de `GET /api/v1/ws` salen sin `Cache-Control: no-store` (verificado en vivo: `Cache-Control=None` tanto en `401` como en `403`), rompiendo la política uniforme de `ADR-0007` (fila Cabeceras: *«`Cache-Control: no-store` por defecto en respuestas autenticadas»*).
- **Corrección sugerida:**
  - Añadir un helper corto (p. ej. `descartarCuerpo(w, r) bool`) que haga `_, err := io.Copy(io.Discard, r.Body)` (usando el `http.MaxBytesReader` ya puesto por `cabecerasSeguridad`) y, ante `*http.MaxBytesError`, responda `413` (`el cuerpo supera 65536 bytes`); invocarlo al inicio de `emitirTicket` y `cortarNodo` (tras pasar la autenticación, manteniendo `401`/`403` antes de `413`). Añadir ambas rutas a `TestCabecerasLimiteDeCuerpo413`.
  - Fijar `w.Header().Set("Cache-Control", "no-store")` al principio de `conTicket` (no afecta al `101` escrito tras `Hijack` y cubre `401`/`403`/`400`/`426`) y comprobarlo en test.

### H4 — La colección de Postman usa `pm.collectionVariables.get("demo")` e ignora `--env-var "demo=true"` en Newman · MENOR

- **Dónde:** `docs/api/cc65-tf.postman_collection.json:5, 34-36, 606, 617`
- **Evidencia:**
  1. La descripción de la colección (`:5`) y `scripts/evidencia-api.sh:100-101` ejecutan Newman pasando sobreescrituras con `--env-var`.
  2. La línea `606` del test *«Cortar un nodo (solo con -demo)»* consulta `const demo = pm.collectionVariables.get("demo") === "true";`. En Postman/Newman, `pm.collectionVariables` solo lee el ámbito de colección (fijo en `"false"` en `:35`) e ignora las variables de entorno (`--env-var "demo=true"` o un Environment de Postman), mientras que `{{nodo}}` en `:617` sí resuelve el ámbito de entorno. Por tanto, ejecutar Newman con `--env-var "demo=true" --env-var "nodo=<nombre>"` contra un servidor con `DEMO=true` falla el test exigiendo `404` ante un `202`.
- **Corrección sugerida:** Sustituir `pm.collectionVariables.get("demo")` por `pm.variables.get("demo")` en `docs/api/cc65-tf.postman_collection.json:606`, que respeta `--env-var` / Environment y cae a la variable de colección `"false"` por defecto.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-84. |

## Cómo explicarlo en la sustentación

El navegador no puede añadir `Authorization: Bearer` al handshake del WebSocket; por eso la SPA (con JWT de admin) solicita un ticket aleatorio de 32 bytes y un solo uso en `POST /api/v1/ws/ticket` (válido 30 s, tope de 128 pendientes) y abre `GET /api/v1/ws?ticket=…`. El servidor valida primero el `Origin` contra `CORS_ORIGENES` (403 sin consumir el ticket para impedir *cross-site WebSocket hijacking*), luego consume el ticket de forma atómica bajo mutex (401 si no existe, venció o ya se usó) y verifica el rol admin (403) antes del `Hijack` RFC 6455; conectado, el observador del coordinador publica cada `cluster.Evento` en JSON al `ws.Hub` sin bloquear el entrenamiento y al recibir `SIGTERM` despide a los clientes con cierre `1001`. Cuando el coordinador arranca con `-demo` (`DEMO=true`), se habilitan `ritmo_ms` (pausa entre épocas fuera del bucle del coordinador) y `POST /api/v1/demo/nodos/{nombre}/cortar`, que cierra la conexión TCP del nodo para que la caída y la reasignación sigan el camino normal del protocolo manteniendo los pesos idénticos bit a bit al secuencial. Una vez corregido el borrado atómico en `CortarNodo` (**H1**), la suscripción previa al `101` junto con el test E2E determinista (**H2**), el límite `413` y `Cache-Control: no-store` (**H3**) y la lectura de `demo` en Postman (**H4**), el PR quedará listo para aprobarse.
