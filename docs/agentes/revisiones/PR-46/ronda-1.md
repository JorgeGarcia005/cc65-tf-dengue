# Veredicto — PR #46 · Ronda 1

**APROBADO.** El job Docker del PR midió **15 438 634 bytes (15,44 MB)** frente al límite de **40 000 000 bytes**. El usuario de la imagen es `nonroot:nonroot`. Se cumplen los cuatro criterios del issue #45. Queda un comentario de documentación de severidad menor.

| Dato | Valor |
|---|---|
| Juez | Codex (GPT-6), skill `juez-cc65` |
| Fecha | 20 de noviembre de 2026, Lima |
| Rama revisada | `fix/45-imagen-40mb` en `9a01833` |
| Base y alcance | `git diff origin/develop...origin/fix/45-imagen-40mb`, desde `08f8600`. Cambian `deploy/Dockerfile` y `deploy/README.md`. |
| Entregable | PC4, imagen del clúster. [PR #46](https://github.com/JorgeGarcia005/cc65-tf-dengue/pull/46) · [Issue #45](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/45) |

## Ejecutables de la imagen

En `deploy/Dockerfile:24`, el compilador recibe exactamente `./cmd/coordinator` y `./cmd/node`. La etapa final copia el directorio de salida a `/usr/local/bin/`, conserva `USER nonroot:nonroot` y arranca el coordinador por defecto. La documentación de los dos ejecutables y de las herramientas que se ejecutan en el host está en `deploy/README.md:9`.

| Consumidor | Comprobación |
|---|---|
| Compose principal y modo API | Resolví ambos con `docker compose config --format json`. Los únicos servicios con `image: cc65-tf` son `coordinator` y `node`, con los comandos de `deploy/docker-compose.yml:37` y `:62`. El archivo adicional de la API conserva ambos ejecutables. |
| Bases de datos | El Compose exclusivo de BD resuelve a `mongo:7.0.14` y `redis:7.4.1`. En los servicios de BD, los healthchecks y `scripts/evidencia-bd.sh` ejecutan `mongosh` en Mongo y `redis-cli` en Redis. |
| Scripts y Makefile | `scripts/evidencia-api.sh:59` ejecuta `benchapi` con Go en el host. `scripts/prueba-caida.sh` usa el cliente Docker del host para levantar Compose, leer logs, detener un nodo y esperar al coordinador. Revisé también los demás scripts versionados y los destinos del Makefile. |
| CI | `.github/workflows/ci.yml:121` genera el gold con Go en el runner. El entrenamiento usa los comandos de Compose y Python lee el informe en el runner. Ningún job invoca un tercer binario dentro de `cc65-tf`. |

## Verificación automática

| Chequeo | Resultado |
|---|---|
| `gofmt -l .` y `go vet ./...` | Ambos terminaron con código 0 y salida vacía. |
| `go test -race -count=1 ./...` | Pasó localmente, incluidos `cmd/coordinator`, `cmd/node`, `internal/cluster` e `internal/ml`. |
| Imagen en CI | `docker image inspect` devolvió `usuario=nonroot:nonroot bytes=15438634`. Pasaron las dos aserciones de `.github/workflows/ci.yml:118` y `:119`. La descripción del PR consigna 24 956 495 bytes en su medición local. Ambas cumplen el umbral. |
| Clúster en Compose | El job validó tres nodos, 60 épocas e `igual_al_secuencial: true` en el reporte. El coordinador confirmó que pesos e historia eran idénticos bit a bit al secuencial. |
| Caída de un nodo | El job detuvo `cc65-tf-node-1` en la época 20 o después. Hubo una reasignación en la misma época y se conservó la igualdad bit a bit hasta completar el entrenamiento. |
| CI del PR | Los seis jobs terminaron con éxito para `9a01833`. [Job Docker con el log de tamaño y entrenamiento](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44708903260/job/136195759487) · [Ejecución completa](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44708903260) |
| Latencia de predicción | Fuera de alcance. El diff modifica el empaquetado y su documentación. |

## Hallazgos

### H1 — El comentario de `make imagen` aún describe el empaquetado anterior · MENOR

En `Makefile:40` se lee «todos los cmd/». El destino construye el Dockerfile correcto. Basta con actualizar el comentario para que coincida con `deploy/README.md:9`.

## Puntaje de rúbrica en juego

| Criterio | Peso TB2 | Evaluación de este PR |
|---|---:|---|
| Planteamiento | 3 | Fuera del alcance del cambio. |
| Implementación | 8 | El issue evalúa el tamaño y el usuario de la imagen, la ejecución con tres nodos y la recuperación ante la caída de uno. El peso corresponde al criterio completo del entregable. |
| Interfaz | 5 | Fuera del alcance del cambio. |
| Informe | 2 | El README de despliegue documenta el contenido de la imagen. |
| Video | 2 | Fuera del alcance del cambio. |

## Seguimiento de rondas previas

Primera ronda del PR #46.

## Cómo explicarlo en la sustentación

Compose usa una misma imagen para dos procesos. El coordinador dirige el entrenamiento y cada réplica de `node` procesa sus shards. La preparación del gold y las herramientas de evaluación se ejecutan desde el host, por lo que empaquetar solo los dos procesos del clúster reduce el tamaño de la imagen.
