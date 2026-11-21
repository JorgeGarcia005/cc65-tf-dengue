# Veredicto — PR #40 · Ronda 1

**CAMBIOS** · 4 hallazgos bloqueantes y 6 mayores.

| Dato | Valor |
|---|---|
| Juez | Codex (GPT-6), skill `juez-cc65` |
| Fecha | 17 de noviembre de 2026, Lima |
| Rama y commit | `feature/29-mongo-redis` en `f2ff73a` |
| Base efectiva | `feature/28-api` en `f38b24d`, también ancestro común |
| Diff evaluado | `git diff origin/feature/28-api...origin/feature/29-mongo-redis` |
| Entregable | PC4, persistencia y caché. [PR #40](https://github.com/JorgeGarcia005/cc65-tf-dengue/pull/40) · [Issue #29](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/29) |

Los adaptadores funcionan con MongoDB y Redis reales. El problema principal aparece cuando una escritura falla o el coordinador se reinicia. En esos casos, la versión que la API anuncia y la que se puede recuperar del almacén pueden ser distintas.

## Hallazgos

### H1 — La API anuncia como publicado un modelo que Mongo no guardó · BLOQUEANTE

En `internal/api/api.go:334` el error de `GuardarModelo` solo llega al log. A continuación se sustituye el modelo activo y se guarda el entrenamiento como terminado con la nueva versión. Lo comprobé con un almacén que conservaba el modelo anterior y fallaba únicamente al escribir el nuevo. El entrenamiento quedó `terminado` y su `version_modelo` difería de la versión recuperable. Además, un error al guardar el estado final del entrenamiento solo se registra en `internal/api/entrenamientos.go:144`.

**Corrección sugerida.** Conservar el modelo anterior hasta confirmar la nueva publicación y definir cómo se recupera una escritura parcial de modelo y entrenamiento. Probar por separado el fallo de cada escritura y comprobar lo que verá un cliente después de reiniciar.

### H2 — El arranque sigue usando el modelo suministrado aunque haya uno vigente en el almacén · BLOQUEANTE

El ADR-0006, línea 24, especifica que al arrancar se prefiere Mongo y se recurre al archivo cuando la colección está vacía. En `internal/api/api.go:94` se publica directamente `op.Modelo`. El diff introduce `ModeloVigente` y `ModeloDeDocumento`, pero carece de un consumidor de producción que los conecte al arranque. Preparé dos versiones distintas, una en el almacén en memoria y otra como modelo de archivo. Al crear el servidor quedó activa la segunda. El test de reinicio reconstruye un modelo en `internal/api/almacen_test.go:132` y al nuevo servidor le comprueba únicamente el historial.

**Corrección sugerida.** Resolver el modelo inicial en la composición del coordinador o en el constructor. Distinguir colección vacía de error de lectura. Reiniciar con un archivo anterior y comprobar la versión y la predicción servidas por la API.

### H3 — Dos arranques en el mismo segundo pueden sobrescribir un entrenamiento · BLOQUEANTE

En `internal/api/entrenamientos.go:79` el ID combina el segundo UTC con un contador local que vuelve a uno tras cada reinicio. Mongo reemplaza el documento con ese ID mediante `upsert` en `internal/almacen/mongo/mongo.go:80`. Creé dos instancias con el mismo almacén dentro de un segundo. Sus primeros entrenamientos, de 3 y 9 épocas, recibieron el mismo ID. Al consultar el identificador del primero se recuperaron las 9 épocas del segundo.

**Corrección sugerida.** Usar un identificador independiente del reloj y del proceso, y distinguir la creación de las actualizaciones de progreso. Una colisión al crear debe preservar el entrenamiento existente. Probar dos instancias que compartan almacén.

### H4 — La evidencia del issue puede darse por válida con un entrenador simulado y registros anteriores · BLOQUEANTE

El último criterio del issue #29 exige registros tras un entrenamiento del clúster en Compose. `internal/api/bd_integracion_test.go:51` usa un entrenador falso con Mongo y Redis reales. En la línea 19 se documenta cómo escribir ese modelo sintético en la base de la aplicación. Además, `scripts/evidencia-bd.sh:24` verifica que haya algún modelo y alguna predicción. La comprobación de Redis en la línea 29 acepta claves de cualquier versión. Por ello el script puede salir con cero aun cuando falte el entrenamiento correspondiente. En esta revisión salió con cero y mostró claves de tres versiones. La descripción del PR reconoce que la ejecución distribuida queda pendiente.

**Corrección sugerida.** Mantener los modelos de prueba en una base de pruebas. Capturar el ID de una ejecución distribuida en Compose y verificar la cadena completa con ese ID, la versión publicada, una predicción de esa versión y su clave Redis con TTL positivo. El job `bd` de CI también puede ejecutar la prueba de integración HTTP. Actualmente solo incluye los adaptadores en `.github/workflows/ci.yml:63`.

### H5 — Una lectura lenta de Redis consume por sí sola el presupuesto de predicción · MAYOR

En `internal/api/api.go:246` la petición espera el GET y en la línea 265 espera el SET incluso si la lectura falló. El adaptador permite 200 ms por lectura o escritura y 2 s para conectar en `internal/almacen/redis/redis.go:27`. Probé 20 peticiones secuenciales al handler con una caché que demoraba 120 ms en la lectura y devolvía error de inmediato al escribir. Las 20 devolvieron 200 y el p95 fue **120,76 ms**. Es una medición del escenario de demora inducida. El p95 con Redis sano sigue pendiente del benchmark del issue #30.

**Corrección sugerida.** Reservar un presupuesto total para Redis dentro de los 100 ms, incluyendo conexión y reintentos, y pasar al cálculo local cuando se agote. Evitar una segunda espera tras un error de lectura. Medir acierto, fallo de clave y caché degradada.

### H6 — El cierre puede terminar antes de la última escritura del entrenamiento o tardar minutos en vaciar registros · MAYOR

El `WaitGroup` en `internal/api/api.go:96` cubre únicamente al registrador. El entrenamiento se ejecuta por separado y persiste su estado final en la línea 347. Con la cola vacía, `Cerrar` puede retornar mientras el entrenamiento cancelado aún está escribiendo. Si quien llama desconecta Mongo en ese momento, el historial persistido puede quedar `en_curso`. Con Mongo lento ocurre otro problema de cierre. La línea 121 crea un plazo nuevo de 2 s para cada inserción y el drenaje procesa hasta 1024 registros en serie. Si cada inserción agota su plazo, la espera puede superar 34 minutos.

**Corrección sugerida.** Detener la admisión, esperar la escritura terminal del entrenamiento antes de cerrar el almacén y dar al drenaje un plazo global. Cubrir con pruebas la cancelación con una escritura final pendiente y el cierre con la cola llena.

### H7 — El almacén en memoria comparte el esquema mutable del modelo · MAYOR

En `internal/almacen/memoria.go:24` y `:41` se copian los pesos, mientras que `Esquema` sigue apuntando al mismo objeto. Un consumidor de `ModeloVigente` puede modificar una media o ejecutar `Preparar`, que reconstruye índices y nombres, y alterar así el documento guardado y el esquema que usa otra predicción. El mutex protege el mapa de modelos, pero las referencias sobreviven al desbloqueo.

**Corrección sugerida.** Dar al almacén su propia copia del esquema al guardar y devolver una copia independiente al leer. Extender el contrato compartido con una mutación del resultado y verificar que una segunda lectura y la predicción original conserven sus valores.

### H8 — El error de conexión a Mongo incluye la URI completa · MAYOR

En `internal/almacen/mongo/mongo.go:43` se interpola `uri` en el error de `Ping`. Si el despliegue configura usuario y contraseña en `MONGO_URI`, un fallo de conexión incorpora ambos al error que recibe el coordinador. La URI local sin autenticación de Compose deja este caso fuera de la prueba habitual.

**Corrección sugerida.** Omitir las credenciales del diagnóstico y probar con valores ficticios que el mensaje de error no los incluya.

### H9 — La respuesta inicial del entrenamiento devuelve listas nulas · MAYOR

`internal/almacen/almacen.go:58` y `:59` copian las listas sobre un slice nulo. El entrenamiento inicial crea dos listas vacías, pero al copiarlo para la respuesta 202 se serializan como `"historia":null` y `"nodos":null`. Comprobé ambas propiedades en la copia que devuelve `iniciar`. El esquema añadido en `docs/api/openapi.yaml:220` y `:224` declara arreglos. Esto afecta al cliente que intenta recorrer el progreso al recibir su ID.

**Corrección sugerida.** Conservar los arreglos vacíos al copiar y verificar el JSON de la respuesta 202 y del GET antes de la primera época.

### H10 — El modo sin Mongo conserva todas las predicciones en RAM · MAYOR

En `internal/api/api.go:89` la ausencia de un almacén selecciona `NuevaMemoria`. En `internal/almacen/memoria.go:62` cada registro consumido se añade a un slice sin retención ni límite. El búfer de 1024 regula las solicitudes que esperan escritura, pero el historial ya escrito sigue creciendo con el tráfico, incluso durante el uso normal sin Mongo.

**Corrección sugerida.** Definir una retención acotada para el modo en memoria o usar un registrador que descarte el historial cuando se elige operar sin persistencia. Mantener la variante de inspección que necesitan las pruebas.

## Verificación automática

| Chequeo | Resultado |
|---|---|
| `gofmt -l .` y `go vet ./...` | Código 0 y salida vacía en ambos. |
| `go test -race -count=1 ./...` | Pasó en Windows, incluidos almacén, API, ML y clúster. |
| Integración local con `-race -tags integracion` | Pasaron los contratos de Mongo y Redis y la prueba HTTP contra ambos servicios de Compose. |
| Caché versionada | Pasó `TestCacheSeInvalidaAlPublicarModelo`. El handler toma una sola referencia al modelo para construir la clave y calcular el resultado. |
| Equivalencia | El diff apilado conserva ML y clúster. Sus pruebas están incluidas en la suite completa. |
| Latencia oficial | `go run ./cmd/benchapi` devolvió `directory not found`. El experimento de H5 mide únicamente el handler bajo demora controlada. |
| Drivers y datos | `go list -m all` pasó. Las únicas dependencias directas nuevas son los drivers de MongoDB y Redis. El diff no agrega datos del MINSA ni cifras al informe. |
| GitHub Actions | Los cuatro jobs están verdes, incluidos [Go con detector de carreras](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44214616506/job/134501215963) e [integración de adaptadores](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44214616506/job/134501215519). |

Extractos de las ejecuciones locales. Los servicios de Compose ya estaban en ejecución. Las pruebas de integración usaron sus bases temporales y el script consultó los registros existentes en `cc65_dengue`.

```text
$ go test -race -count=1 ./...
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen  2.592s
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/api      9.544s
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster  13.059s
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml       15.092s

$ go test -race -count=1 -tags integracion -v ./internal/almacen/... ./internal/api -run 'TestMongoCumpleElContrato|TestRedisCumpleElContrato|TestBDDePuntaAPunta'
--- PASS: TestMongoCumpleElContrato (0.12s)
--- PASS: TestRedisCumpleElContrato (1.51s)
--- PASS: TestBDDePuntaAPunta (0.13s)

$ bash scripts/evidencia-bd.sh
-- modelos: 1 documentos
-- entrenamientos: 1 documentos
-- predicciones: 2 documentos
-- claves pred:*: 3
```

Las comprobaciones adicionales de H1, H2, H3, H5 y H9 se ejecutaron con un overlay temporal de Go y `-race`. H1 y H2 usaron un almacén en memoria para controlar la versión recuperable y el fallo de escritura.

## Puntaje de rúbrica en juego

| Criterio | Peso TB2 | Aporte acreditado en esta ronda |
|---|---:|---|
| Planteamiento | 3 | Fuera del alcance del issue. |
| Implementación | 8 | Parcial. Los drivers, índices, TTL y la invalidación de caché funcionan. La recuperación de los datos y la evidencia de entrenamiento distribuido siguen abiertas. |
| Interfaz | 5 | Parcial en el contrato de entrenamiento que consumirá la SPA. H9 afecta el estado inicial. |
| Informe | 2 | El ADR documenta las entidades y el script muestra registros reales. H4 impide atribuirlos a la ejecución distribuida exigida. |
| Video | 2 | Fuera del alcance del issue. |

Los pesos corresponden al entregable completo. La rúbrica facilitada carece de una subdivisión numérica para este PR.

## Seguimiento de rondas previas

Primera ronda del PR #40.

## Cómo explicarlo en la sustentación

Mongo conserva los modelos y la historia por época. Redis reutiliza la respuesta de una consulta normalizada durante 24 horas y la versión del modelo forma parte de la clave. Al publicar nuevos pesos, la siguiente predicción usa otra clave. El registro de predicciones se envía a Mongo mediante una cola para que la respuesta HTTP pueda terminar antes de esa escritura.
