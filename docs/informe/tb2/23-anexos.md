# 23. Anexos

## Anexo A. Enlaces

Tabla: Enlaces y ubicación de los entregables {#tab:enlaces}

| Qué | Dónde |
|---|---|
| Repositorio en GitHub | <https://github.com/JorgeGarcia005/cc65-tf-dengue> |
| Versión entregada | Rama `main`, etiqueta `tb2` |
| Tablero Kanban | <https://github.com/users/JorgeGarcia005/projects/4> |
| Conjunto de datos | <https://www.datosabiertos.gob.pe/dataset/vigilancia-epidemiol%C3%B3gica-de-dengue> (checksum en `data/MANIFEST.sha256`) |
| Despliegue | `deploy/Dockerfile`, `deploy/Dockerfile.web`, `deploy/docker-compose.yml`, `deploy/docker-compose.api.yml` |
| Contrato de la API | `docs/api/openapi.yaml` (Swagger UI en `/docs`) |
| Colección de Postman | `docs/api/cc65-tf.postman_collection.json` |
| Modelo Promela | `spin/barrera.pml` y `spin/verificar.sh` |
| Decisiones de diseño | `docs/decisiones/ADR-0001` a `ADR-0007` |
| Reglas y protocolo multiagente | `AGENTS.md`, `docs/agentes/PROTOCOLO.md` y `docs/agentes/revisiones/` |
| Reportes con todas las cifras | `reports/*.json` |
| Evidencias y capturas de este informe | `docs/informe/tb2/evidencias/` y `docs/informe/tb2/capturas/` |
| Fuente del informe | `docs/informe/tb2/` (`go run ./cmd/informe` y `go run ./cmd/word`) |
| Video de la exposición | Aula Virtual, junto con este informe |

Nota: Elaboración propia.

## Anexo B. Matriz de cumplimiento del enunciado y la rúbrica

Tabla: Cumplimiento de los criterios de la rúbrica de la TB2 {#tab:rubrica}

| Criterio (puntos) | Qué pide | Dónde se cumple |
|---|---|---|
| Planteamiento (3) | Caso, pregunta de impacto social y su respuesta con evidencia | Secciones 2 y 3; respuesta en la sección 18 |
| Implementación (8) | Carga concurrente, modelo distribuido, clúster en Docker, API con base de datos, integración segura | Secciones 5 a 15 |
| Interfaz (5) | SPA integrada con la API: login, consulta, configuración del entrenamiento y tiempo real | Sección 16 (y 15 para JWT y CORS) |
| Informe (2) | Informe final con la PC3 y la PC4 corregidas | Todo el documento; correcciones en el Anexo C |
| Video (2) | Video de hasta 6 minutos con los integrantes | Entregado en el Aula Virtual |

Nota: Elaboración propia a partir de la rúbrica de la TB2.

Los requisitos técnicos del enunciado, con los archivos que los implementan y la sección que los explica, están en la @tab:requisitos de la sección 4.

## Anexo C. Atención a las observaciones de la PC3 y la PC4

Tabla: Observaciones del profesor en las entregas anteriores y dónde se atienden {#tab:observaciones}

| Entrega | Observación | Dónde se atiende |
|---|---|---|
| PC1 y PC2 | Todos los integrantes deben tener commits en GitHub | Evidencia de commits por autor, sección 19.1 |
| PC2 | Explicar cómo se aplica cada patrón de concurrencia | Tablas de patrones con su aplicación: @tab:patrones-carga y @tab:patrones-gradiente |
| PC3 | Elaborar la pregunta que resuelve el problema | Sección 2.2; respuesta completa en la sección 18 |
| PC3 | Incluir el código fuente del algoritmo distribuido | Sección 9.3: algoritmos del coordinador y del nodo con su código fuente, copiado del repositorio al generar el informe |
| PC4, 1 | Explicar el Dockerfile y el Compose: cómo se configuran la API, el clúster y las bases de datos, qué inicia el entrenamiento y los archivos de ejecución | Sección 5 completa |
| PC4, 2 | Más pruebas que sustenten los valores de las tablas | Dispersión de las repeticiones (@tab:cluster-muestras), código del benchmark y reporte completo (sección 11.5); evidencias con fecha, máquina y commit en todo el informe |
| PC4, 3 | Explicar el código de la API, los contratos *request/response* y el set de pruebas de los endpoints | Sección 12: código, @tab:contratos, tipos de Go, @tab:postman y peticiones y respuestas reales (12.5) |
| PC4, 4 | Pruebas que corroboren el cuadro de latencia | Sección 13.3: salida de `cmd/benchapi`, Postman y p95 en vivo |
| PC4, 5 | ¿En qué consisten las validaciones de `spin/barrera.pml`? | Sección 10: alcance, cada aserción y la propiedad LTL con su fragmento, mutantes con su contraejemplo |
| PC4, 6 | Imágenes de las bases de datos: qué evidencian y su análisis | Sección 14.4: qué prueba cada parte de la evidencia |

Nota: Elaboración propia a partir de la retroalimentación del Aula Virtual.

## Anexo D. Historial de versiones

Tabla: Versiones entregadas del proyecto {#tab:versiones}

| Etiqueta | Entrega | Contenido principal |
|---|---|---|
| `pc3` | PC3, 07/11/2026 | Caso y pregunta; limpieza y perfil del dataset; diseño del modelo distribuido; *worker pool* del gradiente; speedup secuencial frente a concurrente; modelo en Promela |
| `pc4` | PC4, 21/11/2026 | Clúster por TCP en Docker con tolerancia a fallos; API con OpenAPI y latencia medida; MongoDB y Redis; verificación de Spin al día con el código |
| `tb2` | TB2, 29/11/2026 | SPA con login, mapa de riesgo, predicción, evaluación y panel del clúster en tiempo real; seguridad de la integración; evaluación experimental y respuesta a la pregunta; informe final |

Nota: Las fechas, etiquetas y Pull Requests de cada versión se ven en la evidencia de la sección 19.1.

## Anexo E. Catálogo de pruebas

<!-- generado:pruebas -->
Tabla: Catálogo de pruebas automáticas del repositorio {#tab:catalogo-pruebas}

| Paquete o carpeta | Pruebas | Benchmarks | Qué cubren |
|---|---:|---:|---|
| `cmd/benchapi` | 10 | 0 | Generador de consultas, percentiles y verificación de la caché |
| `cmd/benchmark` | 6 | 0 | Benchmarks que separan tiempos y verifican los pesos |
| `cmd/coordinator` | 19 | 0 | Arranque en modo entrenamiento y servicio, opciones y verificación contra el secuencial |
| `cmd/entrenar` | 3 | 0 | Entrenamiento sobre el gold y comparación bit a bit de reportes |
| `cmd/evaluacion` | 4 | 0 | Evaluación determinista y AUC de referencia |
| `cmd/gold` | 1 | 0 | Construcción del gold y de los shards |
| `cmd/informe` | 17 | 0 | Informe al día con los reportes y el código; regiones generadas |
| `cmd/node` | 6 | 0 | Conexión al coordinador, reintentos y reconexión tras una caída |
| `cmd/revalidar` | 5 | 0 | Comparación campo por campo de los reportes de referencia |
| `cmd/seguridad` | 3 | 0 | Casos de seguridad contra una API de prueba |
| `cmd/usuarios` | 1 | 0 | Alta de usuarios desde variables de entorno |
| `cmd/web` | 5 | 0 | Servidor de la SPA: CSP, configuración y rutas |
| `cmd/word` | 27 | 0 | Conversión del informe a Word en formato APA |
| `internal/almacen` | 8 | 0 | Contrato común de las implementaciones del almacén |
| `internal/almacen/mongo` | 4 | 0 | MongoDB real: índices, publicación atómica y reconciliación |
| `internal/almacen/redis` | 2 | 0 | Redis real: contrato de la caché y plazo de cada operación |
| `internal/api` | 86 | 0 | Rutas, contratos, matriz de roles, CORS, cabeceras, caché, registro asíncrono y cierre ordenado |
| `internal/auth` | 14 | 0 | JWT (algoritmo fijo, firma, vencimiento) y PBKDF2 |
| `internal/cluster` | 32 | 0 | Protocolo, caídas, plazos, reasignación, balanceo y equivalencia bit a bit por TCP |
| `internal/cluster/protocolo` | 19 | 0 | Validación y límites de los mensajes |
| `internal/estadistica` | 7 | 0 | Media recortada e intervalo de confianza por bootstrap |
| `internal/evaluacion` | 11 | 0 | AUC, calibración, cubo de riesgo y casos esperados |
| `internal/features` | 7 | 0 | Codificación de variables y split determinista |
| `internal/gold` | 5 | 0 | Formato binario del gold y shards que reproducen el gradiente global |
| `internal/ingest` | 20 | 0 | Reglas de limpieza, reparaciones, bloques alineados y limpieza concurrente igual a la secuencial |
| `internal/maquina` | 3 | 0 | Descripción del equipo de cada benchmark |
| `internal/ml` | 30 | 1 | Gradiente (diferencias finitas), worker pool igual bit a bit al secuencial, entrenamiento |
| `internal/ws` | 13 | 0 | Handshake RFC 6455, frames, límites y hub que no bloquea |
| `web/src/pruebas` (Vitest, 12 archivos) | 156 | — | Guardas de ruta, cliente de la API, vistas, reductor del panel, réplica y reconexión del WebSocket |
| `web/e2e` (Playwright) | 9 | — | Flujos por rol contra el sistema desplegado, con capturas |
| **Total** | **533** | **1** | |

Nota: Funciones `Test*` y `Fuzz*` de los archivos `*_test.go` y casos `it(…)`/`test(…)` de la SPA, contados por `cmd/informe`; varias pruebas de Go recorren tablas de casos, así que el número de casos ejecutados es mayor. Además: `spin/verificar.sh` (sección 10), la colección de Postman (sección 12.4), `cmd/seguridad` (sección 15.5) y las pruebas en Docker del CI (sección 9).
<!-- /generado -->

## Anexo F. Reporte de participación

El reporte de participación de los integrantes se entrega en el formato del docente, como archivo aparte en el Aula Virtual. Los commits de cada integrante se pueden comprobar en el repositorio (sección 19.1).

## Anexo G. Glosario

Tabla: Términos usados en el informe {#tab:glosario}

| Término | Significado en este informe |
|---|---|
| Bronze, silver, gold | Capas de datos: archivo original, filas limpias y validadas, y matriz numérica para el modelo |
| Bloque | Grupo de filas de tamaño fijo (por ejemplo, 4 096) que es la unidad de trabajo del cálculo del gradiente |
| *Shard* | Conjunto de bloques contiguos que procesa un nodo del clúster |
| *Worker pool* | Número fijo de goroutines que toman tareas de un canal |
| *Parameter server* | Proceso que guarda los pesos del modelo, reparte el trabajo y agrega los resultados |
| Barrera | Punto en que un proceso espera a que terminen todos los demás antes de seguir |
| Igual bit a bit | Mismo valor en cada bit de cada número en coma flotante, no solo un valor parecido |
| *Speedup* | Tiempo de la versión de referencia dividido entre el tiempo de la versión paralela |
| Eficiencia | *Speedup* dividido entre el número de workers o de nodos |
| Punto de equilibrio | Tamaño de los datos desde el cual la versión paralela es más rápida |
| Riesgo calibrado | Probabilidad estimada por el modelo, corregida para que coincida con las tasas observadas |
| Casos esperados | Suma del riesgo calibrado de un grupo de casos: cuántos se espera que tengan signos de alarma |
| AUC | Área bajo la curva ROC: probabilidad de que un caso positivo reciba más riesgo que uno negativo |
| p95 | Percentil 95 de la latencia: el tiempo que no supera el 95 % de las peticiones |
| JWT | *JSON Web Token*: credencial firmada que la SPA envía en cada petición |
| CORS | Política del navegador que controla qué orígenes pueden leer las respuestas de la API |
| CSP | *Content Security Policy*: lista de orígenes desde los que la página puede cargar recursos |
| Mutante | Versión del modelo Promela con un error a propósito, que Spin debe detectar |
| LTL | Lógica temporal lineal: lenguaje para propiedades sobre ejecuciones completas |
| ODS 3 | Objetivo de Desarrollo Sostenible 3: salud y bienestar |
| DIRESA | Dirección Regional de Salud |

Nota: Elaboración propia.
