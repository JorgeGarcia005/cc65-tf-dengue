# Veredicto — PR #40 · Ronda 2

**CAMBIOS.** De los diez hallazgos de la primera ronda, seis quedan resueltos y cuatro se mantienen. Se añade un hallazgo mayor sobre la trazabilidad de dos entrenamientos que producen la misma versión. Quedan abiertos **2 bloqueantes y 3 mayores**.

| Dato | Valor |
|---|---|
| Juez | Codex (GPT-6), skill `juez-cc65` |
| Fecha | 19 de noviembre de 2026, Lima |
| Rama revisada | `feature/29-mongo-redis` en `8700802` |
| Diff evaluado | `git diff origin/feature/28-api...origin/feature/29-mongo-redis`. La punta de la base al entregar es `007bc25`. El ancestro común siguió en `f38b24d` durante toda la revisión. |
| Antecedentes | [Ronda 1](ronda-1.md) · [Respuesta del implementador](respuesta-1.md) · [Issue #29](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/29) |

## Seguimiento de la ronda 1

| Hallazgo | Estado | Evidencia de esta ronda |
|---|---|---|
| H1 · Persistencia al publicar | **SE MANTIENE** parcialmente · BLOQUEANTE | Si falla la escritura del modelo, la API conserva el anterior. Si falla la escritura terminal del entrenamiento, el reinicio recupera la versión pero pierde las métricas finales. Véase H1 abajo. |
| H2 · Modelo de arranque | **RESUELTO** | `internal/api/api.go:140` consulta el almacén. Pasaron las tres pruebas de arranque en `internal/api/revision40_test.go:128` y siguientes. Cubren el riesgo y la versión servidos, el almacén vacío y el error de lectura. |
| H3 · Colisión de ID | **RESUELTO** | La creación usa inserción en Mongo y distingue clave duplicada en `internal/almacen/mongo/mongo.go:199`. Pasaron la prueba de colisión forzada y el contrato con Mongo real. Se conserva el entrenamiento original. |
| H4 · Evidencia de Compose | **SE MANTIENE** parcialmente · BLOQUEANTE | El script vincula las entidades por ID y versión y rechazó la base vacía. Sigue pendiente la cadena de una ejecución distribuida del PR. |
| H5 · Demora de Redis | **SE MANTIENE** · MAYOR | El doble de prueba respeta el contexto. Con el adaptador de producción, el p95 llegó a 122,289 ms ante 120 ms de demora en las respuestas de Redis. |
| H6 · Cierre | **RESUELTO** para los dos defectos de la ronda 1 | `internal/api/api.go:180` espera al entrenamiento antes de cerrar el almacén y aplica un plazo común al drenaje del registrador. Pasaron `TestCerrarEsperaLaEscrituraFinalDelEntrenamiento` y `TestCerrarConColaLlenaRespetaElPlazo`. |
| H7 · Esquema mutable | **RESUELTO** | `internal/features/features.go:96` copia el esquema y `internal/almacen/almacen.go:32` lo incorpora a la copia del modelo. Pasaron la prueba de independencia y el contrato de almacén. |
| H8 · Credenciales en errores | **SE MANTIENE** parcialmente · MAYOR | Las pruebas de usuario y contraseña en la autoridad de la URI pasan. El diagnóstico todavía incluye opciones sensibles de la cadena de consulta. |
| H9 · Arreglos vacíos | **RESUELTO** | `internal/almacen/almacen.go:65` conserva listas vacías. Pasaron `TestCopiaConservaListasVacias` y la comprobación de la respuesta HTTP 202 en `internal/api/almacen_test.go:179`. |
| H10 · Memoria del registro | **RESUELTO** | `internal/almacen/memoria.go:23` limita el registro predeterminado a 10 000 predicciones. Pasó `TestMemoriaRetieneSoloLasUltimasPredicciones`. |

## Hallazgos abiertos

### H1 — La recuperación marca el entrenamiento como terminado con métricas incompletas · BLOQUEANTE

`internal/api/entrenamientos.go:132` persiste el progreso cada diez épocas. Si fallan las tres escrituras terminales, la reconciliación en la línea 210 solo reconstruye estado, versión y fecha de fin. Repetí el escenario de `TestEstadoFinalNoGuardadoSeReconciliaAlReiniciar`, con dos épocas y un nodo. Esperé a que el primer servidor cerrara con las tres escrituras fallidas, rehabilité el almacén y consulté el mismo ID en un servidor nuevo.

| Respuesta HTTP | Época | Historia | Pérdida | Nodos |
|---|---:|---|---:|---:|
| Antes del cierre, estado `terminado` | 2 | `[0.69, 0.5]` | 0.5 | 1 |
| Tras el reinicio, estado `terminado` | 0 | `[]` | 0 | 0 |

**Corrección sugerida.** Guardar un resumen terminal recuperable junto a la publicación del modelo, incluida la historia y la foto del clúster, o usar una operación equivalente que mantenga ambos documentos consistentes. Comparar todos esos campos antes y después del reinicio, también cuando el total de épocas no sea múltiplo de diez.

### H4 — Falta la evidencia de un entrenamiento distribuido con las bases conectadas · BLOQUEANTE

La prueba HTTP ahora usa una base temporal y el CI la ejecuta junto a los adaptadores. El entrenador sigue siendo simulado en `internal/api/bd_integracion_test.go:46`. En `8700802`, `git ls-tree -r --name-only HEAD -- cmd/coordinator deploy` devuelve solo `deploy/docker-compose.bd.yml`. Con ambos servicios sanos, `bash scripts/evidencia-bd.sh` terminó con código 1 y el mensaje `FALTA: entrenamiento terminado`. El script responde correctamente ante la falta de datos.

**Corrección sugerida.** El PR #37 ya está fusionado y la punta de `feature/28-api` contiene el coordinador y el Compose de la API. Incorporar la integración al PR #40, entrenar con nodos, consultar una predicción y guardar la salida de `bash scripts/evidencia-bd.sh <id>` para ese entrenamiento. [PR #37](https://github.com/JorgeGarcia005/cc65-tf-dengue/pull/37)

### H5 — El adaptador Redis real excede el presupuesto de lectura · MAYOR

`internal/api/api.go:327` pasa un contexto de 15 ms. En `internal/almacen/redis/redis.go:26` el cliente mantiene un plazo de lectura de 200 ms y deja `ContextTimeoutEnabled` en su valor predeterminado. En la versión fijada, `go-redis/v9@v9.7.3/redis.go:642` usa un contexto de fondo para los plazos de E/S cuando esa opción está desactivada.

Medí el endpoint HTTP con `redis.Conectar`, Redis 7.4.1 y una conexión TCP local que podía demorar la entrega de respuestas. Calenté la misma consulta hasta obtener `X-Cache: HIT` y ejecuté 20 peticiones secuenciales por serie con el cálculo de p95 de `internal/api/revision40_test.go:236`. El control dio **1,0921 ms**. Al añadir 120 ms a cada respuesta de Redis, el p95 fue **122,289 ms**. Son mediciones de este ensayo, distintas del benchmark oficial del issue #30.

**Corrección sugerida.** Configurar el cliente para aplicar el plazo de la petición a la E/S y verificar que el total incluya conexión, esperas de pool y reintentos. Conservar una prueba del adaptador concreto con respuestas demoradas que compruebe el retorno al cálculo local dentro del presupuesto.

### H8 — El error de conexión de Mongo conserva los parámetros de la URI · MAYOR

En `internal/almacen/mongo/mongo.go:44` el error de ping incluye la URI que devuelve `sinCredenciales`. En las líneas 151–166 se oculta el usuario y la contraseña anteriores a `@`, pero se conserva la cadena de consulta. El driver fijado reconoce `tlsCertificateKeyFilePassword` y su alias en `mongo-driver/v2@v2.2.2/x/mongo/driver/connstring/connstring.go:596`. Una URI configurada con esa opción puede dejar la contraseña del archivo de clave privada en el diagnóstico.

**Corrección sugerida.** Construir el diagnóstico con el host y el puerto, y revisar también el texto procedente del driver. Añadir una prueba con un marcador ficticio en una opción sensible y verificar su ausencia en el mensaje completo.

### H11 — Al guardar dos veces la misma versión se pierde el vínculo con el primer entrenamiento · MAYOR · Nuevo

Dos entrenamientos deterministas con los mismos pesos pueden terminar en el mismo segundo. `internal/api/api.go:439` usa esa precisión en `generado`. Al coincidir el resto del modelo se obtiene la misma versión. En `internal/almacen/mongo/mongo.go:93`, el reemplazo por versión sobrescribe el único `entrenamiento_id`. La nueva consulta de reconciliación en la línea 188 ya no encuentra el modelo del primer entrenamiento, y el script exige ese vínculo en `scripts/evidencia-bd.sh:46`.

Lo comprobé con el almacén en memoria, dos ID distintos y las publicaciones separadas por 250 ms dentro del mismo segundo. Ambas produjeron `97064c5c6f8f`. Tras guardar la segunda, `ModeloDeEntrenamiento` devolvió `false` para la primera y `true` para la segunda.

**Corrección sugerida.** Conservar la relación de cada entrenamiento con la versión que produjo, incluso cuando varios comparten pesos y clave de caché. Extender el contrato de Mongo y memoria para recuperar ambos por ID después de guardar la segunda publicación.

## Verificación automática

| Chequeo | Resultado |
|---|---|
| `gofmt -l .`, `go vet ./...` | Código 0 y salida vacía. |
| `go test -race -count=1 ./...` | Pasó. API 5,186 s, clúster 5,962 s y ML 9,463 s. El diff apilado conserva los paquetes de clúster y ML. |
| `go test -race -count=1 -tags integracion ./internal/almacen/... ./internal/api` | Pasó contra MongoDB y Redis locales en Docker. Mongo 4,243 s, Redis 3,199 s y API 5,554 s. |
| `go run ./cmd/benchapi` | `directory not found` en el commit revisado. El p95 oficial de PC4 queda pendiente. |
| CI del commit | Los cuatro jobs finalizaron con éxito. [Ejecución de GitHub Actions](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44223481597) |

Los ensayos adicionales de recuperación, demora y doble publicación se ejecutaron con `go test -race -count=1` mediante un overlay temporal, usando los mismos tipos y handlers del PR.

## Puntaje de rúbrica en juego

| Criterio | Peso TB2 | Evaluación de esta ronda |
|---|---:|---|
| Planteamiento | 3 | Fuera del alcance del diff. |
| Implementación | 8 | Parcial. Se acreditan los adaptadores, la carga del modelo vigente y la protección contra colisiones de ID. Siguen abiertas la integridad del historial y la ejecución distribuida con ambas bases. |
| Interfaz | 5 | El contrato inicial de `historia` y `nodos` queda verificado. La SPA corresponde a otro entregable. |
| Informe | 2 | El script valida una cadena concreta. Falta la captura de esa cadena tras el entrenamiento distribuido. |
| Video | 2 | Fuera del alcance del diff. |

Los pesos son los de la rúbrica del entregable completo. El issue carece de un reparto de puntos por hallazgo.

## Cómo explicarlo en la sustentación

Al arrancar, el coordinador consulta la versión vigente en MongoDB. Cada entrenamiento tiene su propio ID y la API publica los pesos después de confirmar su escritura. Redis reutiliza predicciones por versión. Para demostrar la ejecución de punta a punta, el equipo debe poder partir del ID de un entrenamiento real y encontrar su historia, su modelo y una predicción de esa versión en ambas bases.
