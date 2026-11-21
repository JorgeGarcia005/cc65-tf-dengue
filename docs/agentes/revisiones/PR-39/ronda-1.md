# Veredicto — PR #39 · Ronda 1

**CAMBIOS** · 2 hallazgos bloqueantes y 5 mayores.

| Dato | Valor |
|---|---|
| Juez | Codex (GPT-6), con la skill `juez-cc65` |
| Fecha | 2026-11-17, Lima |
| Entregable | PC4, parte C |
| Revisión | `feature/28-api` en `f281f92`, contra `develop` en `3c11132` |
| Alcance | [PR #39](https://github.com/JorgeGarcia005/cc65-tf-dengue/pull/39), implementación del [issue #28](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/28) |

## Hallazgos

### H1 — Una predicción sin edad usa la de un recién nacido · BLOQUEANTE

En `internal/api/modelo.go:84`, el campo numérico recibe cero tanto si el cliente envía `0` como si omite `edad` o envía `null`. OpenAPI exige una edad entera en `docs/api/openapi.yaml:143`. Probé las tres variantes en el handler con el modelo de prueba del PR, departamento LORETO, año 2024, semana 10 y sexo F. Las tres devolvieron HTTP 200 y exactamente el mismo riesgo, aunque en dos de ellas se desconoce la edad.

**Corrección sugerida.** Distinguir la presencia de la edad al decodificar. Cubrir los tres casos con una prueba que exija 400 y mensaje en español para los dos primeros casos inválidos y 200 para una edad explícita de cero.

### H2 — El coordinador de Compose todavía no sirve la API · BLOQUEANTE

En `docs/api/openapi.yaml:15` se anuncia el servidor de Compose en el puerto 8080. En este commit, `git ls-tree -r --name-only HEAD -- cmd/coordinator deploy` devuelve una lista vacía. La prueba de Swagger en `internal/api/api_test.go:306` usa un servidor de prueba. El test de cierre en `internal/api/api_test.go:386` cancela un contexto directamente. El equipo puede demostrar el handler y el drenaje de una petición, pero el checkout carece del servicio para probar la URL publicada, el entrenador real y la recepción de SIGTERM. La descripción del PR reconoce la dependencia del PR #37.

**Corrección sugerida.** Integrar la API en el coordinador y en Compose al incorporar la dependencia. Verificar desde el host `/docs`, `/docs/openapi.yaml` y un entrenamiento consultable por ID. Verificar también que el coordinador reciba SIGTERM y cierre el servicio.

### H3 — Un esquema de modelo incompleto puede devolver HTTP 200 sin predicción · MAYOR

`CargarModelo` acepta las desviaciones de edad y año ausentes o iguales a cero en `internal/api/modelo.go:45`. La transformación divide por ellas en `internal/features/features.go:111` y `:120`. En el modelo de prueba fijé la media de edad en 19 y la desviación en cero antes de cargar el JSON. La carga tuvo éxito. Una consulta válida de edad 19 produjo un resultado no finito y el handler devolvió HTTP 200 con el cuerpo vacío. En `internal/api/api.go:247` se escribe el estado antes de codificar y se descarta el error de codificación.

**Corrección sugerida.** Validar que ambas desviaciones sean positivas al cargar el modelo. Preparar el JSON antes de emitir el código de éxito y cubrir el caso de un artefacto inválido con un test.

### H4 — El cuerpo opcional de entrenamiento resulta obligatorio · MAYOR

En `docs/api/openapi.yaml:95` el cuerpo de `POST /api/v1/entrenamientos` es opcional. En `internal/api/api.go:188` se exige decodificar un objeto aun después de cargar los parámetros predeterminados. Probé el handler con un entrenador configurado y sin entrenamiento activo. Una petición sin cuerpo obtuvo HTTP 400 con el mensaje «falta el cuerpo JSON». Un cliente generado a partir de Swagger puede emitir legítimamente esa petición.

**Corrección sugerida.** Aceptar la ausencia de cuerpo y aplicar los valores predeterminados documentados. Probar que la petición devuelve 202 y que el entrenador recibe esos valores. Si el equipo decide exigir un objeto, debe actualizar el contrato y su ejemplo.

### H5 — El lector JSON acepta contenido sobrante después del objeto · MAYOR

Después de decodificar la consulta, `internal/api/api.go:261` usa `Decoder.More()` para comprobar que terminó el documento. Ese método comprueba si quedan elementos dentro de un arreglo u objeto. También devuelve falso ante un delimitador de cierre sobrante, por lo que el handler puede procesar una petición mal formada a pesar de anunciar validación estricta.

**Corrección sugerida.** Intentar una segunda decodificación y exigir `io.EOF`. Cubrir el espacio en blanco final válido, el contenido sobrante y el límite de tamaño del cuerpo.

### H6 — El esquema OpenAPI de la unidad de edad contiene una propiedad inválida · MAYOR

En `docs/api/openapi.yaml:149` la descripción de `tipo_edad` contiene una coma dentro de un mapa YAML en línea. Al parsear el archivo obtuve `"description":"Años","meses o días.":null`. La segunda clave queda fuera de los campos admitidos para ese esquema. La prueba de cobertura en `internal/api/api_test.go:361` verifica métodos, rutas y el archivo embebido, por lo que pasa aunque un validador de OpenAPI rechace el esquema.

**Corrección sugerida.** Entrecomillar la descripción completa o usar un mapa YAML en bloque. Validar el documento como OpenAPI 3.0.3 además de comparar las rutas.

### H7 — El clúster vacío devuelve una lista de nodos nula · MAYOR

`internal/api/api.go:179` serializa directamente la vista del coordinador. Antes del primer registro, la copia de la lista en `internal/cluster/coordinador.go:546` es un slice nulo. Comprobé que el handler devuelve HTTP 200 con `"nodos":null`. El esquema en `docs/api/openapi.yaml:188` declara un arreglo. Un panel que recorra los nodos según ese contrato fallará justo al arrancar el clúster.

**Corrección sugerida.** Serializar la colección vacía como `[]` y cubrir el estado anterior al primer registro con una prueba de la respuesta JSON.

## Verificación automática

| Chequeo | Resultado |
|---|---|
| `gofmt -l .` y `go vet ./...` | Ambos finalizaron con código 0 y salida vacía. |
| `go test -race -count=1 ./...` | La repetición completa pasó. En la primera ejecución con acceso a localhost falló la prueba de caída escalonada del clúster. Ninguna de estas ejecuciones informó una carrera. |
| `go test -race -count=1 -run Equivalencia -v ./...` | Pasó con 1, 2, 3 y 5 nodos, con menos nodos que shards y con 1 a 64 workers en las configuraciones del test. |
| Modelo local | `TestCargarModeloReal` pasó con el artefacto local, versión `0647084f0d6b`. |
| CI del PR | Pasaron [Go con detector de carreras](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44207172953/job/134475994890) y [Ecosistema de agentes](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44207172953/job/134475994553). |
| Latencia de predicción p95 < 100 ms | Pendiente de medición. `go run ./cmd/benchapi` devolvió `directory not found`. El benchmark y Postman corresponden al [issue #30](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/30). |

Extractos de las ejecuciones locales con Go 1.26.10 en Windows. La primera ejecución dentro del sandbox había bloqueado las conexiones a localhost. Las siguientes tuvieron acceso a los sockets de prueba.

```text
$ go test -race -count=1 ./...   (primera ejecución con localhost)
--- FAIL: TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano (0.72s)
    cluster_test.go:345: el entrenamiento abortó: ml: época 1: cluster: no quedan nodos sanos para el shard 2 en la época 1

$ go test -race -count=1 ./...   (repetición completa, código 0)
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/api      3.117s
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster  5.723s
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml       7.802s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.49s)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.11s)
--- PASS: TestEquivalencia (0.20s)
```

El diff contra `develop` mantiene intacto `internal/cluster`. Los casos adicionales de H1, H3, H4 y H7 se observaron llamando a `Servidor.Handler()` con `httptest.NewRecorder` mediante un overlay temporal de Go.

## Puntaje de rúbrica en juego

| Criterio | Peso | Evaluación de esta ronda |
|---|---:|---|
| Planteamiento | 3 | Fuera del alcance del diff. |
| Implementación | 8 | Crédito parcial. El módulo usa la biblioteca estándar, reutiliza las variables del entrenamiento y conserva la equivalencia. Los bloqueantes impiden acreditar la API desplegada con validación completa. |
| Interfaz | 5 | Aporta el contrato HTTP que consumirá la SPA. La interoperabilidad queda pendiente por H4, H6 y H7. La interfaz de usuario se evalúa en TB2. |
| Informe | 2 | Fuera del alcance del diff. |
| Video | 2 | Fuera del alcance del diff. |

La rúbrica asigna estos pesos al entregable completo. El issue carece de un reparto de puntos por endpoint que permita calcular una nota numérica independiente para este PR.

## Seguimiento de rondas previas

Primera ronda del PR #39.

## Cómo explicarlo en la sustentación

La API transforma cada consulta con el esquema guardado durante el entrenamiento. Usa la probabilidad ponderada para decidir la clase y aplica la calibración para devolver el riesgo. Mientras el entrenamiento trabaja en segundo plano, las peticiones siguen usando el modelo publicado. El servidor sustituye el puntero al modelo cuando los pesos nuevos están listos y protege el historial de entrenamientos con un mutex.
