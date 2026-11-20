# Veredicto — PR-40 · Ronda 3

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/29-mongo-redis` @ `18b85b8`
- **Fecha:** 2026-11-19
- **Decisión:** APROBADO

## Verificación automática

> **Nota del entorno local:** el proxy del sandbox del VM devolvió `403 Forbidden` al descargar la dependencia transitiva `github.com/klauspost/compress@v1.16.7` desde `proxy.golang.org`, lo que impidió compilar localmente los dos paquetes que importan el driver de MongoDB (`internal/almacen/mongo` y `cmd/coordinator`). Todo el resto de paquetes se verificó localmente con `-race`, y la suite completa (incluidos `cmd/coordinator`, `internal/almacen/mongo`, `internal/almacen/redis`, los tests de integración contra `mongo:7.0.14` y `redis:7.4.1`, y el clúster en Docker) se verificó en verde en el CI de GitHub Actions ([run 44677694904](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44677694904)).

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./internal/almacen ./internal/api ./internal/features ./internal/ml/... ./internal/cluster/... ./internal/ingest/... ./internal/gold/...
(vacío = ok)

$ go test -race -count=1 ./internal/almacen ./internal/api ./internal/features ./internal/ml/... ./internal/cluster/...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.328s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.268s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.545s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.616s

$ go test -race -count=1 -run Equivalencia -v ./internal/ml/... ./internal/cluster/...
--- PASS: TestEquivalencia (0.33s)  (workers=1, 2, 4, 8, 16, 64)
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.51s)  (nodos=1, 2, 3, 5)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.09s)

$ go test -race -count=3 -v -run 'TestMemoriaCumpleElContrato|TestEstadoFinalNoGuardadoSeReconciliaAlReiniciar|TestCacheSeInvalidaAlPublicarModelo|TestCacheLentaNoConsumeElPresupuesto|TestCerrarEsperaLaEscrituraFinalDelEntrenamiento|TestCerrarConColaLlenaRespetaElPlazo|TestArranque|TestColision|TestModeloNoGuardado' ./internal/almacen ./internal/api
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	(3/3 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	6.564s (3/3 PASS)
```

Extracto de GitHub Actions ([run 44677694904](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44677694904), commit `18b85b8`):

```text
Go (gofmt, vet, test -race):
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	1.822s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.016s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.126s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	4.856s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	7.233s

MongoDB y Redis (tests de integración):
--- PASS: TestErrorDeConexionNoExponeCredenciales (3.00s)
--- PASS: TestDestino (0.00s)
--- PASS: TestLimpiarQuitaValoresDeLaConsulta (0.00s)
--- PASS: TestMongoCumpleElContrato (0.04s)
--- PASS: TestRedisCumpleElContrato (1.50s)
--- PASS: TestRedisRespetaElPlazoDelContexto (0.63s)
--- PASS: TestBDDePuntaAPunta (0.04s)
    bd_integracion_test.go:106: p95 con Redis directo 558.48µs; con 120 ms de demora 16.37744ms
--- PASS: TestRedisLentoNoRompeElPresupuesto (0.80s)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (`gofmt -l .` vacío; `go vet ./...` en verde en CI) |
| `go test -race ./...` | ✅ (carreras: 0; todos los paquetes en verde en CI y locales sin driver de Mongo) |
| `go test -race -tags integracion` (Mongo 7.0.14 + Redis 7.4.1) | ✅ (`internal/almacen/...` e `internal/api` en verde en CI) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia`, `TestEquivalenciaClusterSecuencialBitABit` y `TestEquivalenciaConMenosNodosQueShards` en verde; pérdida final de `evidencia-api.sh` = `0.6700654760947148`, idéntica a `reports/entrenamiento.json` época 600) |
| Latencia API p95 < 100 ms | ✅ parcial para caché (con Redis real directo: `558.48 µs`; con Redis demorado 120 ms: `16.38 ms` en `TestRedisLentoNoRompeElPresupuesto`); el benchmark oficial `cmd/benchapi` y Postman corresponden al issue #30 (`cmd/` no contiene `benchapi` todavía) |
| CI del PR (`gh pr checks 40`) | ✅ 5/5 en verde: `Ecosistema de agentes` (10s), `OpenAPI 3 válido` (19s), `MongoDB y Redis (tests de integración)` (43s), `Go (gofmt, vet, test -race)` (52s) y `Docker (imagen, clúster y caída de un nodo)` (2m21s) |
| Restricciones AGENTS.md §3 | ✅ `go.mod` solo agrega `go.mongodb.org/mongo-driver/v2` y `github.com/redis/go-redis/v9` (ADR-0003); `data/` sin archivos de datos versionados; sin cifras manuales en `docs/informe/` |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Completo para el alcance del issue #29 | Interfaces `Almacen` y `Cache` desacopladas de los drivers, implementaciones en memoria, MongoDB y Redis validadas por contrato común bajo `-race`, persistencia atómica de modelo + resumen de entrenamiento, invalidación por versión en Redis con presupuesto acotado (`ContextTimeoutEnabled: true`), integración completa en `cmd/coordinator` y `deploy/docker-compose.yml` con cierre ordenado, y evidencia end-to-end con el clúster real. Queda solo el benchmark/Postman (#30) para cerrar la parte técnica de PC4. |
| Interfaz | 5 | Completo para el contrato de PC4 | `docs/api/openapi.yaml` actualizado (`X-Cache`, `historia`, `nodos`, `reasignaciones`) y validado en CI con `@redocly/cli@1.25.0`. La SPA entra en TB2. |
| Informe | 2 | Completo para la trazabilidad de BD | `docs/decisiones/ADR-0006-entidades-mongo-redis.md` actualizado con el esquema final (`modelos.entrenamientos`, índices, TTL de 90 días y claves Redis 24 h) y evidencia reproducible en `scripts/evidencia-api.sh` + `scripts/evidencia-bd.sh`. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos. Los cinco hallazgos pendientes de la ronda 2 (H1, H4, H5, H8 y H11) y la observación preventiva sobre el arranque del coordinador quedaron resueltos con evidencia reproducible y CI completo en verde.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (rondas 1–2 · BLOQUEANTE) — Publicación sin persistencia y reconciliación con métricas incompletas | RESUELTO | Commit `7333e9d`: `almacen.ResumenEntrenamiento` (`internal/almacen/almacen.go:39-47`) guarda `id`, `epoca`, `perdida`, `historia`, `nodos`, `reasignaciones` y `fin` dentro del documento de `modelos` en la misma escritura atómica `UpdateOne` con `upsert` (`internal/almacen/mongo/mongo.go:95-113`). `entrenar` captura `est` y `fin` una sola vez y pasa el resumen a `GuardarModelo` (`internal/api/api.go:436-459`); `reconciliar` (`internal/api/entrenamientos.go:223-229`) restaura todos los campos con listas no nulas. Verificado en `TestEstadoFinalNoGuardadoSeReconciliaAlReiniciar` (`internal/api/revision40_test.go:97-125`, 7 épocas y 1 nodo) y en `TestMongoCumpleElContrato/EnCursoYModeloDeEntrenamiento` en CI. |
| H4 (rondas 1–2 · BLOQUEANTE) — Falta de evidencia distribuida en Compose con MongoDB y Redis conectados | RESUELTO | Commits `5f5784e`, `5349207`, `e9cd2e7`, `7333e9d`: `cmd/coordinator/bd.go:17-60` y `cmd/coordinator/main.go:88-90,121-143` conectan Mongo y Redis en modo `-api` y los cierran con `defer cerrarBD()` después de `ejecutarAPI` (cuando `srv.Cerrar()` ya terminó). `mongo:7.0.14` y `redis:7.4.1` viven en `deploy/docker-compose.yml:75-97` y `deploy/docker-compose.api.yml:11-18` los enlaza con `depends_on: service_healthy`. `scripts/evidencia-bd.sh:46-47` verifica la cadena por `_id` y `entrenamientos.id`, y `scripts/evidencia-api.sh:50-56` la ejecuta tras entrenar 600 épocas en el clúster real (`0.6700654760947148`, idéntico a `reports/entrenamiento.json`). |
| H5 (rondas 1–2 · MAYOR) — El adaptador Redis real excedía el presupuesto de lectura de 15 ms | RESUELTO | Commit `a207dfd`: `internal/almacen/redis/redis.go:34` habilita `ContextTimeoutEnabled: true` (verificado en `go-redis/v9@v9.7.3/redis.go:643`, donde con `true` se pasa el `ctx` de la petición en lugar de `context.Background()`). Con `contrato.ProxyLento` (`internal/almacen/contrato/proxy.go:15-74`, demora de 120 ms sobre Redis real), en CI pasaron `TestRedisRespetaElPlazoDelContexto` (`internal/almacen/redis/redis_integracion_test.go:31-60`) y `TestRedisLentoNoRompeElPresupuesto` (`internal/api/bd_integracion_test.go:80-110`) con **p95 = 16.38 ms** (control directo `558.48 µs`), bajando desde los 122.29 ms de la ronda 2. |
| H8 (rondas 1–2 · MAYOR) — El diagnóstico de conexión a Mongo conservaba parámetros sensibles de la URI | RESUELTO | Commit `a863fdd`: `destino(uri)` (`internal/almacen/mongo/mongo.go:181-187`) muestra únicamente esquema y hosts (sin userinfo, base ni query string), y `limpiar` (`mongo.go:201-235`) depura del error del driver la URI completa, usuario/clave y valores de consulta (incluyendo `tlsCertificateKeyFilePassword`, `sslClientCertificateKeyPassword`, `authMechanismProperties`, su forma decodificada con `url.QueryUnescape` y valores `CLAVE:valor`). Verificado en CI con `TestErrorDeConexionNoExponeCredenciales`, `TestDestino` y `TestLimpiarQuitaValoresDeLaConsulta` (`internal/almacen/mongo/mongo_test.go:13-62`). |
| H11 (ronda 2 · MAYOR) — Guardar dos veces la misma versión perdía el vínculo con el primer entrenamiento | RESUELTO | Commit `7333e9d`: `Modelo.Entrenamientos` es una lista de resúmenes; `Memoria.GuardarModelo` (`internal/almacen/memoria.go:34-49`) agrega sin sobrescribir y adelanta `Publicado`; `mongo.Almacen.GuardarModelo` (`internal/almacen/mongo/mongo.go:95-113`) usa `$setOnInsert`, `$max` y `$push/$each` de forma atómica con índice `entrenamientos.id` (`mongo.go:64`). Verificado en `contrato.Almacen` (`internal/almacen/contrato/contrato.go:169-188`) sobre memoria y MongoDB real en CI: tras dos publicaciones de la misma versión se recuperan ambos resúmenes por ID y `ModeloVigente` conserva los dos. |
| H2, H3, H6, H7, H9, H10 (ronda 1) — Arranque desde almacén, colisión de ID, cierre con drenaje acotado, copia profunda de esquema, listas vacías y retención en RAM | RESUELTO | Confirmados sin regresiones en la suite local (`3/3` corridas con `-race`) y en CI (5/5 jobs en verde). |
| Observación preventiva (`cmd/coordinator/api.go`) — Modelo inicial incompatible se persistía en almacén vacío antes de fallar el arranque | RESUELTO | Commit `a560490`: `ejecutarAPI` valida `base.Esquema.D() != p.d` antes de llamar a `api.Nuevo` (`cmd/coordinator/api.go:50-53`) y mantiene la comprobación posterior para un modelo vigente ya guardado en Mongo (`api.go:64-68`). Cubierto por `TestServicioNoPersisteUnModeloIncompatible` (`cmd/coordinator/api_test.go:302-314`), en verde en CI. |

## Cómo explicarlo en la sustentación

La API accede a la persistencia y a la caché únicamente mediante las interfaces `Almacen` y `Cache` (`internal/almacen`), lo que permite probar toda la lógica sin contenedores usando memoria y validar MongoDB y Redis con el mismo contrato de pruebas. MongoDB guarda tres colecciones (`modelos`, `entrenamientos` y `predicciones` con TTL de 90 días): al terminar un entrenamiento, una sola escritura atómica (`UpdateOne` con `$setOnInsert`, `$max` y `$push`) publica el modelo junto con el resumen completo del entrenamiento, de modo que dos entrenamientos con los mismos pesos no se pisan y un reinicio tras un fallo de escritura puede reconciliar el historial sin perder métricas. Redis implementa *cache-aside* con claves que incluyen la versión del modelo (invalidación automática al publicar nuevos pesos), TTL de 24 horas y `ContextTimeoutEnabled: true` para que una demora de Redis caiga al cálculo local en ~16 ms sin violar la meta de p95 < 100 ms, mientras que el registro de predicciones en MongoDB va por un canal con búfer que nunca bloquea la respuesta HTTP.
