# Respuesta — PR #40 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-17 · **Base:** `feature/29-mongo-redis`

Acepto los diez hallazgos. Nueve quedan corregidos en esta ronda y H4 se corrige en parte; lo que
falta de H4 depende del PR #37. Escribí los tests antes de cada corrección. Los de la API no
compilaban sin la implementación, así que confirmé con **mutantes** que detectan el defecto: reintroduje
cada error y comprobé que su test falla. La tabla de mutantes está al final.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `e4413b0` + `25fa1af`: el modelo nuevo **solo se publica si `GuardarModelo` tuvo éxito**; si falla, el entrenamiento queda `fallido` y sigue publicado el anterior. El estado final se guarda con 3 intentos. Si aun así se pierde, al arrancar `reconciliar` revisa los entrenamientos que quedaron `en_curso`: los marca `terminado` si su modelo está guardado (`ModeloDeEntrenamiento`) y `fallido` («interrumpido») si no. Tests: `TestModeloNoGuardadoNoSePublica`, `TestEstadoFinalNoGuardadoSeReconciliaAlReiniciar`, `TestEntrenamientoInterrumpidoQuedaFallidoAlReiniciar` |
| H2 | BLOQUEANTE | ACEPTO | `25fa1af`: `Nuevo` arranca con el modelo vigente del almacén. Si no hay, usa el del archivo y lo guarda como vigente. Si la lectura falla, **no arranca**: distingue «colección vacía» de «error». Tests: `TestArranqueUsaElModeloVigenteDelAlmacen` (compara la versión y la predicción servidas), `TestArranqueSinModeloEnAlmacenGuardaElDelArchivo`, `TestArranqueConErrorDeLecturaFalla` |
| H3 | BLOQUEANTE | ACEPTO | `e4413b0` + `25fa1af`: el ID es `ent-<fecha>-<16 hex aleatorios>`, sin contador local. La creación usa `CrearEntrenamiento` (inserción, sin *upsert*): un `_id` repetido devuelve `ErrDuplicado`, se reintenta con otro ID y, si sigue chocando, responde 503 sin tocar el existente. Las actualizaciones de progreso siguen con `GuardarEntrenamiento`. Tests: contrato `CrearEntrenamientoNoSobrescribe` (memoria y **Mongo real**, clave duplicada), `TestEntrenamientosDeDosInstanciasNoColisionan`, `TestColisionAlCrearPreservaElExistente` |
| H4 | BLOQUEANTE | ACEPTO (completado; ver «Actualización») | `790e5f6`: se quitó `EVIDENCIA_BASE` y `TestBDDePuntaAPunta` usa siempre una base temporal que borra al terminar. `scripts/evidencia-bd.sh <id>` verifica la **cadena** de un entrenamiento (terminado → `version_modelo` → modelo con ese `_id` y `entrenamiento_id` → predicciones de esa versión → claves `pred:<versión>:*` con TTL > 0) y sale con 1 si falta un eslabón (probado con una cadena completa, una clave faltante y una base vacía). El job `bd` del CI ahora también corre la prueba HTTP de punta a punta. Borré la base local `cc65_dengue` que había mezclado modelos sintéticos. **Pendiente:** correr el entrenamiento distribuido en Compose; ver abajo |
| H5 | MAYOR | ACEPTO | `25fa1af`: la caché tiene un presupuesto fijo: 15 ms para leer y 15 ms para guardar (`PresupuestoLecturaCache` y `PresupuestoEscrituraCache`), con contextos que el cliente go-redis respeta, incluidas la conexión y los reintentos. Tras un error de lectura no se intenta guardar. Test `TestCacheLentaNoConsumeElPresupuesto`: con lectura de 120 ms, y también con escritura de 120 ms, el p95 de 20 peticiones queda < 100 ms y no hay intento de guardar después del error. El p95 con Redis sano se mide en el #30 |
| H6 | MAYOR | ACEPTO | `25fa1af`: `Cerrar` deja de admitir entrenamientos (503). El `Add` del `WaitGroup` ocurre bajo el mismo mutex que lee `Cerrar`, así nunca corre en paralelo con su `Wait`. Luego `Cerrar` cancela el entrenamiento y **espera su escritura final**, y vacía el registro con un **plazo global** (`PlazoCierre`, 5 s por defecto): al vencer, cancela las escrituras pendientes y cuenta las perdidas como descartadas. Tests: `TestCerrarEsperaLaEscrituraFinalDelEntrenamiento` (escritura final de 300 ms) y `TestCerrarConColaLlenaRespetaElPlazo` (cola llena con un almacén bloqueado, plazo de 300 ms; si vuelve el defecto, falla a los 3 s en lugar de colgarse) |
| H7 | MAYOR | ACEPTO | `dbe7dc8`: `features.Esquema.Copia` y `almacen.Modelo.Copia`. La memoria copia el modelo al guardar y al leer. El contrato muta tanto el esquema que se guardó como el que se leyó, y exige que una segunda lectura no cambie |
| H8 | MAYOR | ACEPTO | `e10a7a6`: los errores de `Conectar` muestran la URI con `***` en lugar de usuario y contraseña, y se limpian los mensajes del driver. Test `TestErrorDeConexionNoExponeCredenciales` con credenciales ficticias en tres casos: puerto cerrado, URI mal formada y SRV inexistente. Falló con el código anterior: `mongo: sin conexión a mongodb://usuario-ficticio:secreto-ficticio@…` |
| H9 | MAYOR | ACEPTO | `dbe7dc8`: `Entrenamiento.Copia` siempre devuelve `historia` y `nodos` no nulos. Tests: `TestCopiaConservaListasVacias` y `TestRutasEntrenamientoNuevoTraeListasVacias` (respuesta 202) |
| H10 | MAYOR | ACEPTO | `7ac1070`: `almacen.Memoria` guarda el registro de predicciones en un búfer circular: retiene las últimas 10 000 (`MaxPrediccionesMemoria`) y `NuevaMemoriaConLimite` sirve para los tests. Test `TestMemoriaRetieneSoloLasUltimasPredicciones` |

## Mutantes

Cada fila reintroduce el defecto en el código corregido y corre el test que lo cubre
(`go test -timeout 90s -run <test> ./internal/api`):

| Mutante | Hallazgo | Tests que fallan |
|---|---|---:|
| `Cerrar` sin `wgEntrenar.Wait()` | H6 | 1 |
| Drenaje sin plazo (`AfterFunc(time.Hour, …)`) | H6 | 1 |
| Lectura de caché sin presupuesto | H5 | 1 |
| Publicar aunque `GuardarModelo` falle | H1 | 1 |
| Arranque sin `reconciliar` | H1 | 1 |
| Arranque con `op.Modelo` sin consultar el almacén | H2 | 3 |
| Crear con *upsert* (`GuardarEntrenamiento`) | H3 | 1 |

## H4: lo que falta

El criterio pide la evidencia **después de un entrenamiento del clúster en Compose**. Para eso,
`cmd/coordinator` tiene que implementar `api.Entrenador` sobre `internal/cluster` y servir la API.
Ese archivo y el Compose del clúster llegan con el PR #37 (issue #27), que sigue abierto. Es la
misma dependencia del H2 del PR #39. Cuando el #37 entre, en el PR #39 (H2) o en este:

1. conecto la API en el coordinador con `almacen/mongo` y `almacen/redis`;
2. lanzo `POST /api/v1/entrenamientos` contra el clúster en Compose y hago predicciones con el
   modelo publicado;
3. corro `bash scripts/evidencia-bd.sh <id>` con el ID de ese entrenamiento y pego la salida en la
   ronda siguiente.

## Observación: Mongo y Redis en el Compose principal

AGENTS.md §2 y §3.1 piden que `deploy/docker-compose.yml` levante también `mongo` y `redis`, y que todo
se despliegue con `docker compose up`. Por ahora están en un archivo aparte,
`deploy/docker-compose.bd.yml`, porque `deploy/docker-compose.yml` lo crea el PR #37 (rama de Jorge) y
tocarlo aquí crearía un conflicto. El equipo decidió **dejarlo así por ahora**. Al integrar el #37, se
agregan `mongo` y `redis` al Compose principal, en el mismo archivo o con `include:`, y el coordinador
los usa por la red interna. **PC4 no se cierra sin ese cambio.**

## Verificación

```text
$ gofmt -l (archivos Go)                       → vacío
$ go vet ./... ; go vet -tags integracion ./... → sin salida
$ go test -race -count=1 ./...                  → todos ok
$ docker compose -f deploy/docker-compose.bd.yml up -d --wait
$ go test -race -count=1 -tags integracion ./internal/almacen/... ./internal/api
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.411s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	3.242s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.751s
```

## Actualización (2026-11-19): H4 completo y bases en el Compose principal

**Implementador:** Claude Code (Claude Opus 5.5). El #37 y el #39 ya están en `develop` y la rama los
incorpora con merges (sin reescribir la historia del PR); frente a `develop` el diff es solo el del #29.

- `5f5784e`: el coordinador en modo servicio (`-api`) se conecta a MongoDB y Redis con `-mongo`, `-base`
  y `-redis` (por defecto `$MONGO_URI` y `$REDIS_ADDR`) y se los pasa a la API (`api.Opciones.Almacen` y
  `.Cache`). Las conexiones se cierran después de `Servidor.Cerrar`, cuando la API ya no escribe. Si el
  modelo vigente del almacén no tiene las variables de los shards, no arranca.
- `mongo` y `redis` pasan a `deploy/docker-compose.yml` (red interna, volumen `mongo-datos`,
  healthchecks). `deploy/docker-compose.api.yml` le pasa al coordinador las direcciones y espera a que
  ambos estén sanos. `docker-compose.bd.yml` queda solo para los tests de integración locales (puertos en
  `127.0.0.1`). Con esto se cumple la observación de esta respuesta: «PC4 no se cierra sin ese cambio».
- `scripts/evidencia-bd.sh` usa los archivos del despliegue y `scripts/evidencia-api.sh` lo corre (paso 6)
  con el ID del entrenamiento que acaba de lanzar por la API.

### Evidencia del criterio (Docker Desktop, PC de Jorge, gold real, `5f5784e`)

Entrenamiento del **clúster en Compose** lanzado con `POST /api/v1/entrenamientos`, dos predicciones
iguales (la segunda desde Redis) y la cadena verificada con `scripts/evidencia-bd.sh <id>`:

```text
coordinator-1  | MongoDB conectado (base cc65_dengue): modelos, entrenamientos y predicciones persistentes
coordinator-1  | Redis conectado en redis:6379: caché de predicciones
coordinator-1  | entrenamiento vía API: 600 épocas en 9.9 s, pérdida 0.693147 → 0.670065
coordinator-1  | api: entrenamiento ent-20261120T044332-8f37755939e669da terminado en 9.865s; modelo publicado 2e9958f48052
repetida: 200 en 0.003018 s

== MongoDB (cc65_dengue)
-- modelos: 2 documentos; índices {"_id":1}, {"publicado":-1}, {"entrenamiento_id":1}
-- entrenamientos: 1 documentos; índices {"_id":1}, {"inicio":-1}, {"estado":1}
-- predicciones: 2 documentos; índices {"_id":1}, {"fecha":-1} TTL 7776000s, {"version_modelo":1,"consulta.departamento":1}
== Cadena del entrenamiento ent-20261120T044332-8f37755939e669da
   estado terminado, 600/600 épocas, pérdida final 0.6700654760947148, 3 nodos, 0 reasignaciones
   modelo 2e9958f48052 publicado 2026-11-20T04:43:42.428Z
   predicciones con esa versión: 2 (1 desde la caché)
== Redis
   claves pred:2e9958f48052:*: 1; pred:2e9958f48052:LORETO:2024:10:30:F con TTL 86397 s
OK: cadena completa para la versión 2e9958f48052
```

Los 2 modelos son el inicial (`modelo.json`, guardado como vigente al primer arranque) y el del
entrenamiento. La pérdida final es la misma del clúster en modo único, declarado idéntico bit a bit al
secuencial en la evidencia del #27. El cierre con `docker compose stop coordinator` sale con código 0.

CI de `5f5784e`: Go (`-race`), MongoDB y Redis (integración), Docker (clúster y caída), OpenAPI y
agentes, todos en verde.
