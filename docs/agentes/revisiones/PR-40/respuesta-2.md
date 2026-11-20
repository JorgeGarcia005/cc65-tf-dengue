# Respuesta — PR #40 · Ronda 2

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-19 · **Base:** `feature/29-mongo-redis`

Acepto los cinco hallazgos abiertos y los cinco quedan corregidos. Seguí el mismo método de la
ronda 1: cada test reprodujo el defecto antes de la corrección (H5 y H8) o fue confirmado con un
mutante (H1 y H11).

**Nota sobre la base revisada.** La ronda 2 evaluó `8700802`. Después llegó a esta rama
`5f5784e`, desde otra sesión: integra `feature/28-api` (ya mergeado como #39, con `cmd/coordinator`
y el Compose del #37), conecta MongoDB y Redis en el modo servicio del coordinador y mueve `mongo` y
`redis` a `deploy/docker-compose.yml`. Eso cierra la observación de la ronda 1 sobre el Compose
principal y permite cerrar H4 en esta ronda.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `7333e9d`: el resumen final (`epoca`, `perdida`, `historia`, `nodos`, `reasignaciones`, `fin`) viaja en **la misma escritura que publica el modelo**: un `UpdateOne` con `upsert`, `$setOnInsert`, `$push` y `$max`, atómico sobre el documento. `reconciliar` lo reconstruye completo. `TestEstadoFinalNoGuardadoSeReconciliaAlReiniciar` usa ahora 7 épocas (no múltiplo de 10) y un nodo, compara `epoca`, `historia`, `perdida`, `nodos`, `reasignaciones`, `fin` y `estado` antes y después del reinicio, y exige `historia = [0.69 0.5]`, 7 épocas y 1 nodo. Mutante «reconciliar sin restaurar métricas» → el test falla |
| H4 | BLOQUEANTE | ACEPTO | Ejecución distribuida real en Compose: `bash scripts/evidencia-api.sh`. Levanta el coordinador, 3 nodos, Mongo y Redis, entrena 600 épocas vía `POST /api/v1/entrenamientos`, predice dos veces la misma consulta y corre `scripts/evidencia-bd.sh <id>`, que sale con 0. La salida está abajo |
| H5 | MAYOR | ACEPTO | `a207dfd`: `ContextTimeoutEnabled: true` en el cliente go-redis. Con eso, el plazo de la petición (15 ms) acota conexión, espera del pool, E/S y reintentos, y deja de aplicarse el `ReadTimeout` de 200 ms. `contrato.ProxyLento` demora 120 ms cada respuesta de un Redis real. `TestRedisRespetaElPlazoDelContexto`: antes, `lectura 0: <nil> en 121.0283ms`; ahora cada lectura y escritura vuelve en < 60 ms con error. `TestRedisLentoNoRompeElPresupuesto` (HTTP, adaptador real): **p95 16,7 ms** con 120 ms de demora (control sin demora: 1,06 ms), frente a los 122,289 ms de la ronda. Mutante sin la opción → falla |
| H8 | MAYOR | ACEPTO | `a863fdd`: el diagnóstico usa `destino(uri)`, que muestra solo esquema y hosts. `limpiar` quita del texto del driver la URI completa, el userinfo y los valores de la consulta: `tlsCertificateKeyFilePassword`, `sslClientCertificateKeyPassword` y `authMechanismProperties` siempre, y también su forma decodificada. `TestErrorDeConexionNoExponeCredenciales` agrega marcadores ficticios en `tlsCertificateKeyFilePassword` y en `AWS_SESSION_TOKEN`; antes fallaba con `…tlsCertificateKeyFilePassword=clave-tls-ficticia…` en el mensaje. Tests: `TestDestino` y `TestLimpiarQuitaValoresDeLaConsulta` |
| H11 | MAYOR | ACEPTO | `7333e9d`: `modelos.entrenamientos` es una lista de resúmenes y `GuardarModelo` **agrega**: nunca reemplaza el vínculo. `ModeloDeEntrenamiento` busca por `entrenamientos.id`, con índice. El contrato (memoria y **Mongo real**) publica la misma versión desde dos entrenamientos y recupera ambos resúmenes por ID; `publicado` avanza con `$max` y el vigente conserva los dos. Mutantes «memoria reemplaza» y «Mongo con `$set` en vez de `$push`» → el contrato falla |

## Mutantes de esta ronda

| Mutante | Hallazgo | Tests que fallan |
|---|---|---:|
| `reconciliar` sin restaurar época, historia, pérdida, nodos ni reasignaciones | H1 | 1 |
| `Memoria.GuardarModelo` reemplaza la versión existente | H11 | 1 |
| `mongo.GuardarModelo` con `$set: {entrenamientos: …}` | H11 | 1 |
| Cliente Redis sin `ContextTimeoutEnabled` | H5 | 1 |

## H4: entrenamiento distribuido con las bases conectadas

`bash scripts/evidencia-api.sh` sobre esta rama, con el gold real (765 297 filas, 3 shards). Extracto
del log del coordinador y paso 6 (`evidencia-bd.sh <id>`):

```text
coordinator-1  | MongoDB conectado (base cc65_dengue): modelos, entrenamientos y predicciones persistentes
coordinator-1  | Redis conectado en redis:6379: caché de predicciones
coordinator-1  | 3 shards, 765297 filas, 31 variables, 187 bloques de 4096 filas
coordinator-1  | api: modelo inicial 0647084f0d6b, del archivo
coordinator-1  | 3 nodos registrados: la API acepta entrenamientos
coordinator-1  | entrenamiento vía API: 600 épocas en 5.9 s, pérdida 0.693147 → 0.670065
coordinator-1  | api: entrenamiento ent-20261120T045028-0bb4ffadee9f2849 terminado en 5.885s; modelo publicado 9505e2eabdf2

== 6. MongoDB y Redis: cadena del entrenamiento ent-20261120T045028-0bb4ffadee9f2849 (issue #29)
repetida: 200 en 0.002117 s
== MongoDB (cc65_dengue)
-- modelos: 2 documentos; índices {"_id":1}, {"publicado":-1}, {"entrenamientos.id":1}
-- entrenamientos: 1 documentos; índices {"_id":1}, {"inicio":-1}, {"estado":1}
-- predicciones: 2 documentos; índices {"_id":1}, {"fecha":-1} TTL 7776000s, {"version_modelo":1,"consulta.departamento":1}
== Cadena del entrenamiento ent-20261120T045028-0bb4ffadee9f2849
   estado terminado, 600/600 épocas, pérdida final 0.6700654760947148, 3 nodos, 0 reasignaciones
   modelo 9505e2eabdf2 publicado 2026-11-20T04:50:33.933Z
   predicciones con esa versión: 2 (1 desde la caché)
== Redis
   claves pred:9505e2eabdf2:*: 1; pred:9505e2eabdf2:LORETO:2024:10:30:F con TTL 86397 s
OK: cadena completa para la versión 9505e2eabdf2
```

Los dos documentos de `modelos` son el modelo inicial del archivo (`0647084f0d6b`, guardado al
arrancar por H2 de la ronda 1) y el publicado por el entrenamiento. La pérdida final del clúster,
`0.6700654760947148`, es **idéntica** al último valor de `perdida_por_epoca` (época 600) en
`reports/entrenamiento.json` (entrenamiento secuencial), como exige AGENTS.md §3.3.

## Verificación

```text
$ gofmt -l (archivos Go)                       → vacío
$ go vet ./... ; go vet -tags integracion ./... → sin salida
$ go test -race -count=1 ./...                  → todos ok
$ go test -race -count=1 -tags integracion ./internal/almacen/... ./internal/api
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	5.694s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	3.838s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	6.665s
```
