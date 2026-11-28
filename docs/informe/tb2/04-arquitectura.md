# 4. Arquitectura de la solución

La solución sigue las cuatro partes de la visión de arquitectura del enunciado, que funcionan como una cadena y que se desarrollaron por separado: **A**, el cargador concurrente; **B**, el motor de entrenamiento distribuido; **C**, la API que coordina el clúster y sirve las predicciones; y **D**, la interfaz web de visualización. Los datos pasan por tres capas (*medallion*): **bronze**, el CSV original sin tocar; **silver**, las filas limpias y validadas; y **gold**, la matriz numérica de entrenamiento, partida en shards para los nodos.

## 4.1 Vista general

La @fig:arquitectura muestra dónde se ejecuta cada componente y cómo se comunican. Todo corre en contenedores de Docker Compose (sección 5) salvo la SPA, que se descarga del contenedor `web` y se ejecuta en el navegador del usuario.

![Vista de despliegue de la solución](fig/arquitectura.svg){#fig:arquitectura}

Nota: Diagrama de despliegue UML 2.5. «contenedor» es un servicio de Docker Compose y la imagen de la que se crea. Las flechas continuas con puerto son conexiones de red, en el sentido de la petición; la de doble punta entre el coordinador y los nodos es una conexión TCP persistente por la que viajan mensajes en los dos sentidos (la abre cada nodo al registrarse). Las flechas sin puerto son lectura o escritura de archivos: del CSV a la Parte A y de la Parte A al volumen; las discontinuas, lectura del volumen en solo lectura. Fuente: `deploy/docker-compose.yml` y `deploy/docker-compose.api.yml`; diagrama en `docs/informe/tb2/diagramas/arquitectura.dot`.

La vista se lee de arriba abajo en el orden en que se usa el sistema:

1. **Preparación de los datos (una vez).** Los contenedores `loader`, `gold` y `entrenar` del perfil `datos` leen el CSV del MINSA, lo limpian, construyen la matriz gold y sus shards y entrenan un modelo inicial. No tienen red: solo leen y escriben archivos en `data/`.
2. **Clúster.** El coordinador y los N nodos montan `data/gold` en solo lectura. Cada nodo abre una conexión TCP al puerto 9000 del coordinador, se registra y carga su shard. A partir de ahí, en cada época el coordinador envía los pesos por esa conexión y recibe los parciales del gradiente.
3. **Servicio.** El coordinador publica la API en el puerto 8080 del anfitrión: predicciones, modelo, evaluación, estado del clúster, entrenamientos y un WebSocket con los eventos del entrenamiento. Guarda en MongoDB y consulta Redis antes de calcular una predicción.
4. **Interfaz.** El navegador descarga la SPA del contenedor `web` (puerto 8081) y llama a la API directamente en el puerto 8080. Son dos orígenes distintos, así que la API aplica una política CORS real (sección 15.2).

La @tab:fig1 compara esta vista con la arquitectura de referencia del enunciado (Fig. 1 y apartado 6 del enunciado).

Tabla: Arquitectura de referencia del enunciado frente a la arquitectura implementada {#tab:fig1}

| Elemento del enunciado | Cómo se implementa | Diferencia o añadido, y por qué |
|---|---|---|
| Clúster de nodos ML: goroutines y canales; comunica resultados al coordinador por TCP interno | `cmd/node`: un *worker pool* por nodo calcula un parcial por bloque; envía los parciales al coordinador por TCP en JSON por línea | El puerto 9000 no se publica en el anfitrión: solo existe dentro de la red `cluster` |
| API coordinadora: envía tareas a los nodos y unifica resultados | `cmd/coordinator`: *parameter server* síncrono; reduce los parciales en orden de bloque y actualiza los pesos | Coordinador y API en el mismo proceso: la API lanza entrenamientos y publica el modelo sin un salto de red adicional |
| API: endpoints, JWT, datos en tiempo real por WebSocket | `internal/api`, `internal/auth`, `internal/ws`, solo con la biblioteca estándar | El WebSocket usa un ticket de un solo uso en vez de enviar el JWT en la URL (sección 15.3) |
| Base de datos persistente (MongoDB) y caché (Redis) | `internal/almacen`: modelos, entrenamientos, predicciones y usuarios en MongoDB; predicciones en Redis | La escritura del registro de predicciones es asíncrona, para no sumar su latencia a la respuesta (sección 14.3) |
| Interfaz SPA con inicio de sesión, *dashboard*, resultados y panel de administración | `web/`: React y Vite, servida por `cmd/web` en Go | Agrega el mapa de riesgo y la evaluación del modelo, que responden la pregunta de impacto (sección 16) |
| Cargador concurrente (Parte A) | `cmd/loader` e `internal/ingest`: bloques de bytes y *worker pool* | Imagen propia (`cc65-tf-datos`), sin red, que se ejecuta una vez antes del servicio |

Nota: Elaboración propia a partir del enunciado del trabajo final (apartados 5 y 6).

## 4.2 Componentes, responsabilidades y tecnologías

La @tab:componentes lista los programas del sistema. Cada uno es un comando de Go en `cmd/` que usa paquetes de `internal/`; la lógica concurrente y numérica vive en los paquetes y se prueba sin red ni disco, y los comandos solo la conectan con la red y los archivos.

Tabla: Componentes del sistema, su responsabilidad y su patrón de concurrencia {#tab:componentes}

| Parte | Componente | Responsabilidad | Concurrencia | Dónde se ejecuta |
|---|---|---|---|---|
| A | `cmd/loader`, `internal/ingest` | Lee el CSV por bloques de bytes, limpia y valida cada fila, escribe la capa silver | *Worker pool* sobre bloques alineados a fin de línea; resultados en una ranura por bloque y concatenación en orden | Contenedor `loader` (imagen `cc65-tf-datos`) |
| A | `cmd/gold`, `internal/features`, `internal/gold` | Codifica las variables, separa entrenamiento y prueba y parte la matriz en shards | Secuencial (una pasada sobre silver) | Contenedor `gold` |
| A | `cmd/entrenar`, `internal/ml` | Entrena el modelo inicial (`modelo.json`) | *Worker pool* del gradiente | Contenedor `entrenar` |
| B | `cmd/node`, `internal/cluster`, `internal/ml` | Carga su shard y calcula, en cada época, un parcial del gradiente por bloque | *Worker pool* persistente; acumuladores privados por bloque | Contenedores `node` (N réplicas, imagen `cc65-tf`) |
| B y C | `cmd/coordinator`, `internal/cluster` | Registra los nodos, reparte los shards, reduce los parciales en orden y actualiza los pesos; reasigna el shard de un nodo caído | Una goroutine lectora por nodo; barrera por época con plazo por shard | Contenedor `coordinator` (imagen `cc65-tf`) |
| C | `internal/api`, `internal/auth`, `internal/ws` | API REST y WebSocket: predicción, modelo, evaluación, clúster, entrenamientos; JWT, CORS y cabeceras de seguridad | Una goroutine por petición (`net/http`); modelo vigente en un `atomic.Pointer`; *hub* de WebSocket con una goroutine escritora por cliente | Mismo proceso que el coordinador |
| C | `internal/almacen` | Persistencia en MongoDB y caché en Redis | Registro de predicciones con búfer y un consumidor; caché *cache-aside* con presupuesto de tiempo | Mismo proceso; MongoDB y Redis en sus contenedores |
| D | `web/`, `cmd/web` | SPA Vigía: inicio de sesión, inicio, mapa, predicción, evaluación y administración del clúster | Asíncrona en el navegador (React); `cmd/web` sirve archivos estáticos | Navegador del usuario; archivos desde el contenedor `web` |
| — | `cmd/benchmark`, `cmd/benchapi`, `cmd/evaluacion`, `cmd/informe`, `cmd/word` | Medición de rendimiento, evaluación del modelo y generación de este informe | Según la medición | Anfitrión (`go run`) |

Nota: Fuente: comentarios de paquete de cada carpeta de `cmd/` e `internal/`. La justificación de cada patrón está en las secciones 7.4, 8.3 y 9.

Las tecnologías son las que fija el enunciado y no más: **Go** para todo el backend, con la biblioteca estándar en el núcleo (carga, ML, protocolo TCP, API, JWT y WebSocket); los drivers oficiales de **MongoDB** y **Redis** (MongoDB, Inc., 2025; Redis Ltd., 2025), únicos módulos de terceros de `go.mod`; **React** con **Vite** para la SPA; y **Docker Compose** para el despliegue.

## 4.3 Flujo de datos de punta a punta

La @fig:flujo-datos sigue un caso desde el CSV del MINSA hasta la pantalla del usuario. Las acciones de cada parte están en su propio carril; las flechas discontinuas son datos que pasan de una parte a otra sin que la segunda espere a la primera.

![Flujo de datos de punta a punta](fig/flujo_datos.svg){#fig:flujo-datos}

Nota: Diagrama de actividades UML 2.5. Los rectángulos redondeados son acciones y los rectángulos rectos, objetos (archivos o colecciones). Fuente: `docs/informe/tb2/diagramas/flujo_datos.dot`.

1. **Bronze → silver.** `cmd/loader` parte el archivo en bloques de bytes que terminan en un fin de línea y los reparte entre los workers. Cada worker limpia las filas de su bloque y deja el resultado en la ranura del bloque; al final, las ranuras se concatenan en orden, así que el silver es idéntico al de una limpieza secuencial (sección 7.3).
2. **Silver → gold.** `cmd/gold` convierte cada fila en <!--v:.S.M.S.D-->31<!--/v--> variables numéricas, separa entrenamiento y prueba y parte el entrenamiento en shards de bloques contiguos, uno por nodo (sección 8.1).
3. **Gold → modelo.** En cada época, el coordinador envía los pesos a los nodos; cada nodo calcula un parcial del gradiente por cada bloque de su shard y los devuelve; el coordinador los suma en orden de bloque, aplica la regularización y actualiza los pesos (sección 9.3).
4. **Modelo → API.** Al terminar, el coordinador publica el modelo nuevo para la API (sin detenerla) y lo guarda en MongoDB con su versión. Una predicción busca primero en Redis y, si no está, calcula la probabilidad y la guarda en la caché (sección 12).
5. **API → SPA.** La SPA consulta la API por REST y recibe por WebSocket cada evento del entrenamiento (época, pérdida, nodos), que dibuja en tiempo real (sección 16.5).

## 4.4 Restricciones del trabajo y cómo se cumplen

La rúbrica pide que la solución «tenga en cuenta las restricciones del trabajo». La @tab:restricciones reúne las del enunciado y las que el equipo se impuso en `AGENTS.md`, con la forma de comprobar cada una. La mayoría se comprueba en cada Pull Request en el flujo de integración continua (CI) de GitHub Actions, que tiene siete trabajos.

Tabla: Restricciones, cómo se cumplen y cómo se comprueban {#tab:restricciones}

| Restricción | Origen | Cómo se cumple | Cómo se comprueba |
|---|---|---|---|
| Desarrollado íntegramente en Go y contenerizado | Enunciado | Todo el backend es Go; tres imágenes Docker (sección 5.1) | Trabajo «Docker» del CI: construye las imágenes y entrena en el clúster |
| Despliegue con Docker Compose de UI, API, clúster y bases de datos | Enunciado | `deploy/docker-compose.yml` y `deploy/docker-compose.api.yml` | `make servicio`; sección 5.7 |
| Más de 1 000 000 de registros procesados de forma concurrente | Enunciado y rúbrica | *Worker pool* sobre bloques del CSV completo | `reports/limpieza.json`: <!--v:miles .S.L.L.Entradas-->1 065 274<!--/v--> filas leídas |
| Goroutines y canales; sin condiciones de carrera | Enunciado, rúbrica y `AGENTS.md` | Acumuladores privados por bloque; estado compartido solo con canal, mutex o atómicos | `go test -race ./...` en el trabajo «Go» del CI |
| Comunicación TCP entre los nodos y el coordinador | Enunciado | Protocolo propio de JSON por línea (sección 9.2) | Tests de `internal/cluster`; modelo Promela verificado con Spin |
| API en menos de 100 ms | Enunciado | Predicción como producto punto, modelo en memoria, caché Redis | `cmd/benchapi` (sección 13): p95 de <!--v:dos (index .APIJWT.Corridas 0).P95-->3,89<!--/v--> ms |
| MongoDB persistente y Redis como caché | Enunciado | `internal/almacen` | Trabajo «MongoDB y Redis» del CI (tests de integración) |
| WebSocket para el tiempo real; JWT | Enunciado | `internal/ws` (RFC 6455) e `internal/auth` (HS256) | Tests de `internal/ws`, `internal/auth` e `internal/api` |
| Núcleo con biblioteca estándar; terceros solo para MongoDB y Redis; sin bibliotecas de ML | `AGENTS.md`, aprobado por el profesor (ADR-0003) | `go.mod` declara solo los dos drivers | Revisión del juez en cada PR; `go.mod` |
| Entrenamiento distribuido idéntico bit a bit al secuencial | `AGENTS.md` | Bloques de tamaño fijo y reducción en orden de bloque | Tests de equivalencia y `VERIFICAR=true` en el trabajo «Docker» del CI |
| Ninguna cifra escrita a mano en el informe | `AGENTS.md` | `cmd/informe` genera cifras, tablas, figuras y fragmentos de código | `TestInformeTB2AlDia` y `TestInformesSinAvisos` |
| Datos no versionados | `AGENTS.md` | `data/` en `.gitignore`; se versiona el script de descarga y el SHA-256 | `make data` verifica el checksum |
| GitHub con Git Flow y tablero Kanban; repositorio público | Enunciado | Ramas `feature/*`, PR a la rama de integración, *tags* por entregable | Sección 19 |

Nota: Los trabajos del CI están en `.github/workflows/ci.yml`. ADR: registro de decisiones de arquitectura, en `docs/decisiones/`.

## 4.5 Requisitos del enunciado y archivos del repositorio

La @tab:requisitos indica, para cada requisito del enunciado del trabajo final, el archivo del repositorio donde se implementa y la sección de este informe que lo explica. Sirve para ubicar cualquier parte del sistema durante la revisión o la sustentación.

Tabla: Requisitos del enunciado, archivos que los implementan y sección del informe {#tab:requisitos}

| Requisito del enunciado | Archivos principales | Sección |
|---|---|---|
| Cargar y procesar más de un millón de registros de forma concurrente | `internal/ingest/bloques.go`, `internal/ingest/limpieza.go`, `cmd/loader/main.go` | 7 |
| Entrenar un modelo de forma distribuida (regresión logística) | `internal/ml/logistica.go`, `internal/cluster/coordinador.go`, `internal/cluster/nodo.go` | 8, 9 |
| Comunicación TCP entre la API coordinadora y los nodos | `internal/cluster/protocolo/`, `cmd/node/main.go` | 9.2 |
| Topología del clúster justificada | `docs/decisiones/ADR-0005-protocolo-tcp.md` | 9.1 |
| API REST con Swagger que responda en menos de 100 ms | `internal/api/`, `docs/api/openapi.yaml`, `cmd/benchapi/` | 12, 13 |
| WebSocket con datos en tiempo real | `internal/ws/`, `internal/api/` | 15.3, 16.5 |
| Autenticación con JWT | `internal/auth/` | 15.1 |
| MongoDB y Redis | `internal/almacen/`, `docs/decisiones/ADR-0006-entidades-mongo-redis.md` | 14 |
| SPA con inicio de sesión, *dashboard*, resultados y panel de administración del clúster | `web/src/` | 16 |
| Despliegue con Docker Compose | `deploy/` | 5 |
| Evaluación experimental y respuesta a la pregunta de impacto | `cmd/evaluacion/`, `internal/evaluacion/`, `reports/evaluacion.json` | 17, 18 |
| Git Flow, Kanban y reporte de participación | Repositorio de GitHub, `docs/agentes/` | 19, anexo F |

Nota: Rutas relativas a la raíz del repositorio (anexo A).
