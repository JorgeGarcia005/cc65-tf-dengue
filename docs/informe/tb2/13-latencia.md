# 13. Latencia de la API

El enunciado exige que la API de predicción responda en menos de 100 ms. Esta sección describe cómo se mide (13.1), qué se obtuvo con la autenticación JWT activa (13.2) y las tres evidencias independientes que lo respaldan (13.3), en respuesta a la observación 4 de la PC4 («faltan pruebas que corroboren el cuadro de latencia»).

## 13.1 Método

`cmd/benchapi` es un cliente de carga escrito en Go que mide `POST /api/v1/predicciones` contra el sistema completo desplegado con Docker Compose: coordinador con la API, MongoDB, Redis y los nodos.

- **Autenticación real.** Inicia sesión como analista una vez y envía el JWT en cada petición, así que cada medición incluye la verificación de la firma y del rol. El login no se mide. La clave se lee de una variable de entorno, nunca de un argumento, para que no quede en el historial de la terminal.
- **Dos corridas.** *Caché fría*: consultas todas distintas, que la API calcula y guarda en Redis. *Caché caliente*: un conjunto pequeño de consultas que se precalientan y luego se repiten, de modo que todas salen de Redis. Las dos corridas usan mitades disjuntas del espacio de consultas, así que ninguna consulta fría estaba en la caché.
- **Concurrencia.** <!--v:.APIJWT.Concurrencia-->16<!--/v--> clientes simultáneos, para medir la API bajo carga y no con peticiones aisladas.
- **Dónde se mide.** En el cliente, desde que se envía la petición hasta leer la respuesta completa, igual que la ve Postman o la SPA. Incluye la red local, el *parseo* HTTP, la autenticación, la caché y el cálculo.
- **Verificación.** Con `-exigir-p95 100ms`, el programa falla si el percentil 95 de alguna corrida no baja de 100 ms, si alguna petición falla, si la corrida fría tuvo aciertos de caché o si la caliente no salió entera de Redis (encabezado `X-Cache`). Así, una medición que no cumple las condiciones no llega al informe.

Se reporta el percentil 95 (p95) y no la media porque el requisito es sobre la experiencia de casi todos los usuarios: el p95 es el tiempo que no supera el 95 % de las peticiones, y no se ve afectado por unas pocas muy rápidas que bajarían la media.

## 13.2 Resultados

<!-- generado:plantilla:latencia -->
Tabla: Latencia de POST /api/v1/predicciones con JWT, medida en el cliente {#tab:latencia}

| Corrida | Peticiones | Errores | HIT / MISS | Peticiones por segundo | p50 (ms) | p95 (ms) | p99 (ms) | Máx. (ms) | p95 < 100 ms |
|---|---:|---:|---|---:|---:|---:|---:|---:|:---:|
| Caché fría (consultas distintas) | 2 000 | 0 | 0 / 2 000 | 6 530 | 2,28 | 3,89 | 4,64 | 6,42 | sí |
| Caché caliente (en Redis) | 2 000 | 0 | 2 000 / 0 | 8 153 | 1,76 | 3,09 | 3,89 | 6,76 | sí |

Nota: `go run ./cmd/benchapi -exigir-p95 100ms` con 16 clientes concurrentes contra el sistema en Docker Compose (windows/amd64), con un JWT del analista en cada petición; modelo `5cb7a4120775`. La latencia va desde que el cliente envía la petición hasta que lee la respuesta completa. Fuente: `reports/bench_api_jorge-pc_jwt.json`.
<!-- /generado -->

**Análisis.** En las dos corridas el p95 queda muy por debajo del límite, incluso con el JWT verificado en cada petición. La caché caliente es más rápida porque evita el cálculo del modelo y la escritura en Redis, pero la diferencia es pequeña: el cálculo de una predicción es un producto escalar de unas decenas de variables, así que el tiempo lo dominan la red local y el protocolo HTTP. Esto significa que el sistema cumple el requisito aunque Redis no esté disponible, porque la API calcula igual cuando la caché no responde dentro de su presupuesto (sección 12.1).

Tres decisiones de diseño explican estos tiempos: el modelo se lee de un `atomic.Pointer` sin mutex, el registro en MongoDB no se espera (sección 14.3) y los documentos grandes (`/riesgo` y `/evaluacion`) se sirven comprimidos y con `ETag`, de modo que una SPA que ya los tiene recibe un 304 sin cuerpo.

## 13.3 Evidencias

**Salida de `cmd/benchapi`.** El paso `benchapi` del guion de evidencias ejecuta la medición en la máquina de referencia y guarda su salida, de la que sale la @tab:latencia:

<!-- evidencia: evidencias/benchapi.txt -->

**Postman.** La colección completa (@tab:postman) se ejecutó con el *Collection Runner*. La @fig:postman-runner muestra todas las aserciones en verde, incluida la de «responde en menos de 100 ms» de cada endpoint, y el tiempo de cada petición.

<!-- captura: Ejecución de la colección de Postman con el Collection Runner: aserciones y tiempo de cada petición {#fig:postman-runner} [capturas/m-postman-runner.png] -->

**Latencia en vivo.** La API también mide su propia latencia: anota el tiempo de cada predicción y el panel de administración muestra el p95 de las últimas peticiones mientras el sistema está en uso (@fig:admin-entrenando, sección 16.4). Es una tercera medición, tomada dentro del servidor y con el clúster entrenando al mismo tiempo.
