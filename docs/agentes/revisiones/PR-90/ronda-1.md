# Veredicto — PR-90 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `fix/89-auditoria` @ `93a7078` (base: `integracion/tb2` @ `a435072`)
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

El PR cierra el issue `#89` resolviendo de forma limpia, mínima y bien probada cuatro hallazgos de la auditoría integral TB2 (`docs/agentes/revisiones/auditoria-tb2/ronda-1.md`):
1. **B2 (`internal/api/modelo.go`, `internal/api/version_test.go`):** `versionDe` calcula el SHA-256 (primeros 12 hex) sobre una estructura anónima canónica con los cuatro campos que determinan la predicción (`Pesos`, `Esquema`, `Umbral`, `AjustePrior`), dejando `Generado` fuera y unificando la ruta de `CargarModelo` y `conPesos` (que antes también diferían en formato/orden de claves frente al JSON identado de `cmd/entrenar`). Verificado con `TestVersionNoDependeDeGenerado` y de extremo a extremo sobre Docker Compose (`mongo:7.0.14` + `redis:7.4.1`): dos entrenamientos deterministas separados por `> 1 s` publican exactamente la misma versión (`46fade6c6fa4`), conservan el `X-Cache: HIT` en Redis y acumulan ambos entrenamientos (`num_entrenamientos: 2`) en el mismo documento de `db.modelos` vía `$push`.
2. **B7 (`web/src/sesion/Sesion.tsx`, `web/src/App.tsx`, `web/src/pruebas/app.test.tsx`):** `Marco` cierra la sesión con `salir(true)`, fijando `salidaVoluntaria = true` y `sesion = null` en el mismo render síncrono para que `ConSesion` emita `<Navigate to="/login" replace state={undefined} />` sin competir con `startTransition` de React Router v7, mientras que los cierres por `401`/vencimiento (`salir()`) y los accesos directos sin sesión mantienen `state={{ desde }}`. Verificado con `173/173` tests en Vitest (incluidos los nuevos casos de salida voluntaria desde `/admin` y `/mapa` con el mismo rol y recuerdo de destino sin sesión).
3. **B3 (`docs/api/cc65-tf.postman_collection.json`):** «Estado del clúster» guarda `sano.nombre` en la variable de colección `nodo`. Verificado en vivo con Newman (`--env-var demo=true`) contra Docker Compose (`3` réplicas `node` con nombres de contenedor `0e6f379c4d5b`, `579d0a4efe92`, `907b2fa6d816`): **44/44 aserciones en verde** (`POST /api/v1/demo/nodos/0e6f379c4d5b/cortar → 202 Accepted` en `6 ms`).
4. **B4 (`cmd/benchapi/main_test.go`):** `TestMedirConLoginUsaJWT` amplía el plazo de red del cliente HTTP del test de `5 s` a `20 s` para absorber las derivaciones PBKDF2 de `600 000` iteraciones bajo `-race` en máquinas de `2 vCPU`, sin tocar la verificación de `p95 < 100 ms`. Verificado con `-race -count=3` (`7,09 s`–`10,26 s` por corrida bajo carga, `0` fallos).

Todo el núcleo permanece en stdlib (`AGENTS.md` §3.2), `gofmt`, `go vet`, las 25 suites de Go con `-race` (`0` carreras), la equivalencia secuencial = distribuido bit a bit, `cmd/benchapi` con JWT (`p95 = 14,51 ms` fría / `14,78 ms` caliente `< 100 ms`) y los `7/7` jobs del CI en GitHub Actions están en verde. Se deja una única nota operativa **MENOR *(no bloqueante)*** (**H1**) para los cierres de `#70`/`#71`/`#73`.

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ git log --format="%h | %an <%ae> | %s" origin/integracion/tb2..HEAD
93a7078 | JorgeGarcia005 <jg1536454@gmail.com> | fix: versión del modelo sin fecha, «Salir» sin destino previo, Postman y plazo de test (#89)

$ git diff --check origin/integracion/tb2...origin/fix/89-auditoria
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/api ./internal/cluster ./internal/ml ./internal/ingest
internal/api     -> solo stdlib + paquetes internos (docs/api, almacen, auth, cluster, features, ingest, ml, ws)
internal/cluster -> solo stdlib + paquetes internos (protocolo, gold, ml)
internal/ml      -> solo stdlib
internal/ingest  -> solo stdlib

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.799s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.906s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.399s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.038s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.648s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.260s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.228s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.417s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.380s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.068s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.148s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.294s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.693s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.430s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.841s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.042s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.940s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.086s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.607s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.037s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.149s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.754s

$ go test -race -count=1 -v -run TestVersionNoDependeDeGenerado ./internal/api
--- PASS: TestVersionNoDependeDeGenerado (0.00s)

$ go test -race -count=3 -v -run TestMedirConLoginUsaJWT ./cmd/benchapi
--- PASS: TestMedirConLoginUsaJWT (10.26s)
--- PASS: TestMedirConLoginUsaJWT (8.97s)
--- PASS: TestMedirConLoginUsaJWT (7.09s)

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.88s) [nodos=1, 2, 3, 5]
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.16s)
--- PASS: TestEquivalencia (0.53s) [workers=1, 2, 4, 8, 16, 64]

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm run build && ! grep -q "url(data:" dist/assets/*.css
Test Files  11 passed (11)
     Tests  173 passed (173)
✓ built in 1.44s · sin url(data: en CSS

$ <Docker Compose E2E: coordinator -demo + 3 réplicas node + mongo:7.0.14 + redis:7.4.1 + web>
Nodos en Compose: 0e6f379c4d5b, 579d0a4efe92, 907b2fa6d816 (os.Hostname)
Entrenamiento 1: {"id":"ent-20261127T224036-9704d650f17881e9","estado":"terminado","version_modelo":"46fade6c6fa4"}
Entrenamiento 2: {"id":"ent-20261127T224037-4d3716d2b5583ec9","estado":"terminado","version_modelo":"46fade6c6fa4"}
Redis antes/después de reentrenar: X-Cache: MISS -> X-Cache: HIT
MongoDB db.modelos: {"_id":"46fade6c6fa4","generado":"2026-11-27T22:40:36Z","num_entrenamientos":2}

$ BENCH_CLAVE=... go run ./cmd/benchapi -url http://localhost:18080 -correo analista@vigia.pe -exigir-p95 100ms
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria         2000       0     2120     7.00    14.51    17.92    28.11      0   2000  p95 < 100 ms: true
caliente     2000       0     2185     6.54    14.78    20.93    29.72   2000      0  p95 < 100 ms: true

$ npx newman run docs/api/cc65-tf.postman_collection.json ... --env-var demo=true
→ Cortar un nodo (solo con -demo)
  POST http://localhost:18080/api/v1/demo/nodos/0e6f379c4d5b/cortar [202 Accepted, 274B, 6ms]
  ✓  HTTP 202 en modo demo · ✓  responde en menos de 100 ms
requests: 15/15 · assertions: 44 executed, 0 failed

$ gh pr checks 90 -R JorgeGarcia005/cc65-tf-dengue
Docker (imagen, clúster y caída de un nodo)	pass	2m20s
Ecosistema de agentes                       	pass	7s
Go (gofmt, vet, test -race)                 	pass	1m46s
MongoDB y Redis (tests de integración)      	pass	51s
OpenAPI 3 válido                            	pass	20s
SPA (tipos, tests, build e imagen)          	pass	57s
Spin (modelo del protocolo y mutantes)      	pass	1m46s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (25/25 suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` bit a bit) |
| Latencia API p95 < 100 ms | ✅ (`14,51 ms` fría / `14,78 ms` caliente sobre Docker Compose con JWT, MongoDB y Redis; `44/44` en Newman con `demo=true`) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (`173/173` tests en verde, sin `url(data:` en CSS) |
| CI GitHub Actions (PR #90 @ `93a7078`) | ✅ (`7/7` jobs en verde, run `46076714694`) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | No toca planteamiento ni partición; mantiene la coherencia entre `cmd/evaluacion` y `api.CargarModelo` vía `versionDe`. |
| Implementación | 8 | 8.0 / 8 | `versionDe` canoniza la huella del modelo sin `Generado` (preservando Redis y activando la acumulación en `entrenamientos[]` de MongoDB), Newman pasa `44/44` con `demo=true` en Compose y `TestMedirConLoginUsaJWT` queda estable bajo `-race`. |
| Interfaz | 5 | 5.0 / 5 | Cierra la carrera de transición entre `salir` y React Router v7: «Salir» limpia el destino previo incluso entrando con el mismo rol, mientras un 401 o acceso directo sin sesión sí recuerda la vista. |
| Informe | 2 | sin cambios | Fragmentos de `docs/informe/pc4/10-api.md` intactos (`TestFragmentosSalenDelFuente` en verde); pendiente `#71` para el informe final TB2. |
| Video | 2 | mejora la demo | Reentrenar en vivo con los hiperparámetros de referencia ya no invalida Redis ni dispara el cartel de evaluación desfasada tras regenerar `evaluacion.json`. |

## Hallazgos

Sin hallazgos **BLOQUEANTES** ni **MAYORES**.

### H1 — Recordatorios operativos para el cierre de `#70` / `#71` / `#73` · MENOR *(no bloqueante)*

- **Dónde:** `reports/evaluacion.json`, PR `#88` (`internal/ws/ws_test.go`) y `web/src/sesion/Sesion.tsx:48,59,88`
- **Evidencia:**
  1. Como advierte con transparencia la descripción del PR, al excluir `Generado` y pasar a la serialización canónica de `versionDe`, el `modelo.json` del gold real cambia de identificador una sola vez. Hasta correr `cmd/entrenar`, `cmd/benchapi` (con JWT) y `cmd/evaluacion` juntos en `#70`/`#71`, `reports/evaluacion.json` conserva el id anterior (`655c9dc770fa`).
  2. Cuando `go test -race ./...` corrió en paralelo con `npm ci` + Vitest en esta VM de `2 vCPU`, `TestHubConcurrenteSinFugas` reprodujo el corte por ping de `5 ms` (`ws_test.go:329: recibir: EOF`), que pasó limpio al correr sin esa contienda externa y cuyo ajuste ya está listo en el PR `#88` (`fix/ws-test-intermitente`).
  3. En `Sesion.tsx:59,88`, `salidaVoluntaria` permanece en `true` hasta el próximo `entrar()` o `salir(false)`. En el uso normal y en equipos compartidos es seguro (pulsar «Atrás» tras «Salir» tampoco arrastra `desde`), por lo que no requiere cambios en este PR.
- **Corrección sugerida:** No bloquea el merge de este PR: (a) fusionar también el PR `#88` en `integracion/tb2`, y (b) en `#70`/`#71` regenerar `data/gold/modelo.json` + `reports/bench_api_*_jwt.json` + `reports/evaluacion.json` para alinear las tres fuentes bajo la nueva versión determinista.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| B2 (auditoría TB2) | RESUELTO | `versionDe` hashea `{Pesos, Esquema, Umbral, AjustePrior}` sin `Generado`; verificado en `TestVersionNoDependeDeGenerado` y en Docker Compose con MongoDB/Redis. |
| B3 (auditoría TB2) | RESUELTO | «Estado del clúster» actualiza `nodo` con el primer nodo sano; Newman pasa `44/44` con `demo=true` en Compose. |
| B4 (auditoría TB2, parte `benchapi`) | RESUELTO | Plazo de red de `TestMedirConLoginUsaJWT` subido a `20 s` (`3/3` corridas en verde bajo `-race` y contienda). La parte de `ws_test.go` vive en el PR `#88`. |
| B7 (auditoría TB2) | RESUELTO | `salir(true)` + `salidaVoluntaria` en `ConSesion`; verificado en `app.test.tsx` (`173/173` en Vitest). |

## Cómo explicarlo en la sustentación

La versión del modelo se calcula como los primeros 12 caracteres hexadecimales del SHA-256 de una representación JSON canónica de lo único que define sus predicciones —pesos, esquema de variables, umbral y ajuste de calibración—, dejando fuera la fecha `generado`. Como el descenso de gradiente distribuido con reducción en orden fijo de bloque es determinista bit a bit, reentrenar con los mismos hiperparámetros produce exactamente la misma versión: las predicciones cacheadas en Redis bajo `pred:<version>:<hash>` siguen siendo válidas (`HIT`) y MongoDB agrupa los entrenamientos equivalentes en el arreglo `entrenamientos[]` del mismo documento mediante `$push` atómico. En la SPA, cerrar sesión marca `salidaVoluntaria` en el mismo cambio de estado en que `sesion` pasa a `null`, evitando que la transición asíncrona de React Router v7 recuerde la vista anterior tras pulsar «Salir», sin perder el retorno automático cuando un usuario entra sin sesión o su token vence.
