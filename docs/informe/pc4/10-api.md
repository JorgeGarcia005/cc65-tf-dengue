# 10. API REST

## 10.1 Rutas y documentación

El coordinador, en modo servicio, expone la API con `net/http` de la biblioteca estándar. El contrato está
en `docs/api/openapi.yaml` (OpenAPI 3.0.3, validado en el CI) y se sirve con **Swagger UI** en `/docs`. Un
test exige que cada ruta del servidor esté documentada.

| Método y ruta | Qué hace | Respuestas |
|---|---|---|
| `GET /healthz` | Salud del servicio y versión del modelo publicado | 200 |
| `GET /api/v1/modelo` | Modelo publicado: versión, pesos por variable, umbral, departamentos | 200 |
| `POST /api/v1/predicciones` | Riesgo calibrado de que un caso sea con signos de alarma o grave, a partir de departamento, año, semana epidemiológica, edad y sexo | 200, 400 (consulta inválida, con mensaje en español) |
| `GET /api/v1/cluster` | Foto del clúster: nodos, estado, carga, época y pérdida | 200 |
| `POST /api/v1/entrenamientos` | Lanza un entrenamiento distribuido en los nodos (cuerpo opcional: épocas, tasa, λ) | 202 con `Location`, 409 si ya hay uno en curso, 503 si el clúster no tiene sus nodos |
| `GET /api/v1/entrenamientos/{id}` | Progreso o resultado: época, pérdida por época, nodos y versión del modelo publicado | 200, 404 |

La consulta se valida con las mismas reglas que la limpieza (sección 6) y se transforma con el mismo
esquema de variables guardado en el modelo, así que la API predice exactamente lo que el modelo aprendió.

## 10.2 Diseño para responder en menos de 100 ms

- **Sin bloqueos en la lectura.** El modelo publicado vive en un `atomic.Pointer`: cada predicción lo lee
  sin mutex, y un entrenamiento nuevo reemplaza el puntero entero cuando termina. Las predicciones siguen
  respondiendo con el modelo anterior mientras el clúster entrena.
- **Caché con presupuesto.** Redis guarda cada predicción con una clave que incluye la versión del modelo.
  Leer y escribir en la caché tienen un plazo fijo de 15 ms cada uno: si Redis está lento o caído, la API
  calcula y responde igual.
- **El registro no espera a la base.** Cada predicción se registra en MongoDB desde una goroutine aparte que
  consume un canal con búfer; si el búfer se llena, el registro se descarta y se cuenta, pero la respuesta
  nunca espera a Mongo.
- **Cierre ordenado.** Con `SIGTERM`, el servidor deja de aceptar conexiones, termina las peticiones en
  curso, cancela el entrenamiento y espera su escritura final, y recién después avisa `fin` a los nodos.

## 10.3 Latencia medida

`go run ./cmd/benchapi -exigir-p95 100ms` envía 2000 predicciones con 16 clientes concurrentes en dos
corridas: con la **caché fría** (consultas distintas: se calcula y se guarda en Redis) y con la **caché
caliente** (las mismas consultas, ya en Redis). Mide en el cliente, como Postman, hasta leer la respuesta
completa, y falla si el p95 de alguna corrida no baja de 100 ms o si la fría tuvo aciertos de caché o la
caliente no salió entera de Redis.

<!-- generado:api -->
| Corrida | Peticiones | Errores | HIT / MISS | Peticiones por segundo | p50 (ms) | p95 (ms) | p99 (ms) | Máx. (ms) | p95 < 100 ms |
|---|---:|---:|---|---:|---:|---:|---:|---:|---|
| Caché fría (consultas distintas) | 2 000 | 0 | 0 / 2 000 | 10 986 | 1,39 | 2,21 | 2,78 | 5,49 | sí |
| Caché caliente (en Redis) | 2 000 | 0 | 2 000 / 0 | 12 982 | 1,10 | 2,06 | 2,34 | 3,02 | sí |

Fuente: `reports/bench_api_deez.json` (`go run ./cmd/benchapi -exigir-p95 100ms` contra el clúster en Docker Compose con el gold real, 16 clientes concurrentes, Windows; modelo `e69395b36d2f`).
<!-- /generado -->

La colección `docs/api/cc65-tf.postman_collection.json` tiene una petición por ruta con tests de estado y de
tiempo menor a 100 ms; la predicción nueva debe responder `X-Cache: MISS` y la repetida `HIT`. Las capturas
de Postman y de Swagger están en la sección 12.3.
