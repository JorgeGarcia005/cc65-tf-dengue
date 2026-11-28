# 5. Despliegue con Docker

El enunciado pide desplegar con Docker Compose la interfaz, la API, el clúster de nodos y las bases de datos. Esta sección explica cada archivo que lo hace: las imágenes (5.1), los dos archivos de Compose (5.2 y 5.3), la red, los puertos y los volúmenes (5.4), qué ocurre al arrancar y qué inicia un entrenamiento (5.5), los archivos de ejecución (5.6) y la ejecución completa paso a paso (5.7). Los fragmentos de configuración se copian de los archivos del repositorio al generar el informe, así que coinciden con lo que se ejecuta.

## 5.1 Las tres imágenes

El sistema usa tres imágenes propias, una por tipo de proceso, y las imágenes oficiales de MongoDB y Redis (@tab:imagenes). Las tres propias siguen el mismo patrón de **construcción en varias etapas** (*multi-stage build*): una etapa con las herramientas de compilación (Go o Node) produce los binarios, y la imagen final copia solo esos binarios sobre `gcr.io/distroless/static-debian12:nonroot`, una imagen sin intérprete de órdenes (*shell*), sin gestor de paquetes y con un usuario sin privilegios. Así la imagen que se despliega es pequeña y expone muy poca superficie de ataque.

Tabla: Imágenes del despliegue {#tab:imagenes}

| Imagen | Archivo | Qué contiene | Servicios que la usan |
|---|---|---|---|
| `cc65-tf` | `deploy/Dockerfile` | `coordinator` y `node` | `coordinator`, `node` (N réplicas) |
| `cc65-tf-datos` | `deploy/Dockerfile.datos` | `loader`, `gold` y `entrenar` (Parte A) | `loader`, `gold`, `entrenar` (perfil `datos`) |
| `cc65-tf-web` | `deploy/Dockerfile.web` | `web` (servidor estático en Go) y el *build* de la SPA | `web` |
| `mongo:7.0.14` | Oficial | MongoDB | `mongo` |
| `redis:7.4.1` | Oficial | Redis | `redis` |

Nota: Las versiones de MongoDB y Redis están fijadas para que el despliegue sea reproducible. Fuente: `deploy/`.

**Imagen del clúster.** La primera etapa (`construccion`) descarga los módulos de Go en una capa propia, que Docker reutiliza mientras `go.mod` no cambie, y compila los dos binarios con `CGO_ENABLED=0` (binarios estáticos, sin la biblioteca de C, que distroless no trae), `-trimpath` (sin rutas de la máquina que compiló) y `-ldflags="-s -w"` (sin tabla de símbolos). Copia también `docs/api`, porque el contrato OpenAPI se incrusta en el binario del coordinador con `go:embed` para servir Swagger. La segunda etapa solo copia `/out/` y declara el usuario `nonroot` y el puerto 9000 del protocolo de los nodos. El comando por defecto es el coordinador; los nodos lo cambian en Compose.

<!-- generado:archivo:deploy/Dockerfile -->
```dockerfile
# deploy/Dockerfile
# syntax=docker/dockerfile:1
# Imagen del clúster: coordinator y node en una imagen mínima (< 40 MB en el CI).
# La Parte A (loader, gold, entrenar) tiene su propia imagen, cc65-tf-datos
# (deploy/Dockerfile.datos); las herramientas del informe y los benchmarks
# (benchapi, evaluacion, informe, word…) corren en el host con `go run`.
#
#   docker build -f deploy/Dockerfile -t cc65-tf .
#
# Etapa 1: compila. Etapa 2: solo los binarios sobre distroless (sin shell ni
# gestor de paquetes) y con usuario sin privilegios.

FROM golang:1.24-bookworm AS construccion
WORKDIR /src
# Primero el módulo: mientras no cambien las dependencias, esta capa se reutiliza.
COPY go.mod go.su[m] ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
# El contrato OpenAPI se embebe en el binario (go:embed en docs/api).
COPY docs/api ./docs/api
# CGO_ENABLED=0: binarios estáticos, sin libc (requisito de distroless/static).
# -trimpath: sin rutas de la máquina que compiló. -s -w: sin tabla de símbolos
# ni datos de depuración (binarios más pequeños).
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/coordinator ./cmd/node

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=construccion /out/ /usr/local/bin/
USER nonroot:nonroot
# 9000: TCP coordinador–nodos (ADR-0005).
EXPOSE 9000
CMD ["/usr/local/bin/coordinator"]
```
<!-- /generado -->

**Imagen de la Parte A.** Es igual en estructura, con los tres comandos de la preparación de datos. Su directorio de trabajo es `/trabajo`, donde Compose monta `data/` y `reports/` del anfitrión: así las rutas por defecto de los tres comandos (`data/…`, `reports/…`) son las mismas en el contenedor que con `go run` en el anfitrión. Va separada de `cc65-tf` porque solo se usa una vez, antes del servicio, y porque así la imagen del clúster queda mínima.

<!-- generado:archivo:deploy/Dockerfile.datos -->
```dockerfile
# deploy/Dockerfile.datos
# syntax=docker/dockerfile:1
# Imagen de la Parte A (carga y preparación de datos): loader, gold y entrenar.
# Va aparte de la imagen del clúster (cc65-tf) para que esa siga mínima (< 40 MB
# en el CI) y porque solo se usa una vez, antes de levantar el servicio.
#
#   docker build -f deploy/Dockerfile.datos -t cc65-tf-datos .
#
# La usan los servicios loader, gold y entrenar del perfil «datos» de
# deploy/docker-compose.yml: CSV bronze → silver → gold + shards → modelo.json.
# El directorio de trabajo es /trabajo, donde el Compose monta ../data y
# ../reports, así que las rutas por defecto de los tres comandos (data/…,
# reports/…) son las mismas que en el anfitrión.

# Etapa 1: compila los tres comandos (solo biblioteca estándar: no necesitan
# los drivers de MongoDB y Redis, pero go.sum los lista y se descargan igual).
FROM golang:1.24-bookworm AS construccion
WORKDIR /src
COPY go.mod go.su[m] ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY docs/api ./docs/api
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/loader ./cmd/gold ./cmd/entrenar

# Etapa 2: solo los binarios, sin shell y sin privilegios.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=construccion /out/ /usr/local/bin/
WORKDIR /trabajo
USER nonroot:nonroot
CMD ["/usr/local/bin/loader"]
```
<!-- /generado -->

**Imagen de la SPA.** Tiene tres etapas: Node compila la SPA de React a archivos estáticos (`npm ci` instala exactamente las versiones de `package-lock.json`), Go compila `cmd/web` y la imagen final reúne los dos. `cmd/web` sirve los archivos con una política de seguridad de contenido (CSP) estricta y entrega `/config.json` con la dirección de la API (variable `API_URL`), de modo que la misma imagen funciona en `localhost` o con la IP de otra máquina sin recompilar.

<!-- generado:archivo:deploy/Dockerfile.web -->
```dockerfile
# deploy/Dockerfile.web
# syntax=docker/dockerfile:1
# Imagen de la SPA Vigía (#62): el build de React lo sirve cmd/web, un servidor
# estático en Go stdlib con CSP estricta y /config.json (ADR-0007). Va aparte de
# la imagen del clúster y en otro origen que la API, así que el CORS es real.
#
#   docker build -f deploy/Dockerfile.web -t cc65-tf-web .
#
# Usa deploy/Dockerfile.web.dockerignore (no el .dockerignore de la raíz, que
# excluye web/).

# Etapa 1: build de la SPA.
FROM node:22-bookworm-slim AS front
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# Etapa 2: servidor estático en Go.
FROM golang:1.24-bookworm AS servidor
WORKDIR /src
# cmd/web solo usa la biblioteca estándar: no hace falta descargar los drivers
# de MongoDB y Redis que figuran en go.mod.
COPY go.mod go.su[m] ./
COPY cmd/web ./cmd/web
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/web ./cmd/web

# Etapa 3: solo el binario y los archivos estáticos, sin shell y sin privilegios.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=servidor /out/web /usr/local/bin/web
COPY --from=front /web/dist /web
USER nonroot:nonroot
EXPOSE 8081
CMD ["/usr/local/bin/web", "-dir", "/web", "-escucha", ":8081"]
```
<!-- /generado -->

## 5.2 El clúster: docker-compose.yml

`deploy/docker-compose.yml` define el clúster de entrenamiento: el coordinador, los N nodos, MongoDB, Redis y los tres servicios de la Parte A. Empieza con dos *anclas* de YAML que dicen qué imagen usa cada servicio y cómo construirla; los servicios las reutilizan con `<<: *imagen`, de modo que un `docker compose up --build` construye las imágenes que falten.

<!-- generado:archivo:deploy/docker-compose.yml#imagenes -->
```yaml
# deploy/docker-compose.yml
x-imagen: &imagen
  image: cc65-tf
  build:
    context: ..
    dockerfile: deploy/Dockerfile

# Imagen de la Parte A (perfil «datos»): los tres servicios declaran cómo
# construirla, así `run --no-deps gold …` o `run --no-deps entrenar …` la
# construyen en un clon limpio en vez de buscarla en Docker Hub (PR-94 H1).
x-imagen-datos: &imagen-datos
  image: cc65-tf-datos
  build:
    context: ..
    dockerfile: deploy/Dockerfile.datos
```
<!-- /generado -->

**Coordinador.** El servicio `coordinator` ejecuta el binario con sus parámetros como banderas de línea de órdenes; cada una toma su valor de una variable de entorno con un valor por defecto (`${NODOS:-3}`), así que el mismo archivo sirve para cualquier número de nodos o de épocas sin editarlo (@tab:variables). Monta el gold en `/datos` **en solo lectura** y `reports/` del anfitrión, donde escribe el reporte del entrenamiento. `expose: 9000` abre el puerto del protocolo solo dentro de la red `cluster`. Corre con el uid y gid de `USUARIO` para que los archivos que escribe en `reports/` pertenezcan al usuario del anfitrión.

<!-- generado:archivo:deploy/docker-compose.yml#coordinator -->
```yaml
# deploy/docker-compose.yml
coordinator:
  <<: *imagen
  user: "${USUARIO:-65532:65532}"
  command:
    - /usr/local/bin/coordinator
    - -escucha=:9000
    - -gold=/datos
    - -nodos=${NODOS:-3}
    - -epocas=${EPOCAS:-600}
    - -tasa=${TASA:-2}
    - -lambda=${LAMBDA:-0.0001}
    - -verificar=${VERIFICAR:-false}
    - -progreso=${PROGRESO:-50}
    - -timeout-registro=${TIMEOUT_REGISTRO:-120s}
    - -timeout-epoca=${TIMEOUT_EPOCA:-60s}
    - -salida=/reports/entrenamiento_cluster.json
    - -api=${API:-}
    - -modelo=/datos/modelo.json
  volumes:
    - ${GOLD:-../data/gold}:/datos:ro
    - ../reports:/reports
  expose:
    - "9000"
  networks: [cluster]
```
<!-- /generado -->

**Nodos.** El servicio `node` se replica `NODOS` veces (`deploy.replicas`). Cada réplica se conecta a `coordinator:9000`, nombre que resuelve el DNS interno de Docker, y monta el mismo gold en solo lectura: el coordinador no envía los datos por la red, solo le dice a cada nodo qué shard cargar. `depends_on: [coordinator]` solo ordena el arranque; si el coordinador aún no escucha, el nodo reintenta la conexión (sección 5.5). `WORKERS` fija las goroutines del *worker pool* de cada nodo (0 = una por núcleo).

<!-- generado:archivo:deploy/docker-compose.yml#node -->
```yaml
# deploy/docker-compose.yml
node:
  <<: *imagen
  # Sin nombre de contenedor fijo: `--scale node=N` crea node-1 … node-N.
  command:
    - /usr/local/bin/node
    - -coordinador=coordinator:9000
    - -workers=${WORKERS:-0}
  volumes:
    - ${GOLD:-../data/gold}:/datos:ro
  depends_on: [coordinator]
  deploy:
    replicas: ${NODOS:-3}
  networks: [cluster]
```
<!-- /generado -->

**Bases de datos.** MongoDB guarda sus datos en el volumen con nombre `mongo-datos`, que sobrevive a `docker compose down`; Redis se configura solo como caché, sin persistencia en disco y con desalojo LRU al llegar a 128 MB. Cada una tiene una prueba de salud (`healthcheck`) que el coordinador espera en el modo servicio. Ninguna publica puertos en el anfitrión.

<!-- generado:archivo:deploy/docker-compose.yml#bases -->
```yaml
# deploy/docker-compose.yml
mongo:
  image: mongo:7.0.14
  volumes: [mongo-datos:/data/db]
  healthcheck:
    test: ["CMD", "mongosh", "--quiet", "--eval", "db.adminCommand('ping').ok"]
    interval: 5s
    timeout: 5s
    retries: 12
  networks: [cluster]

redis:
  image: redis:7.4.1
  # Solo caché: sin persistencia en disco y con desalojo LRU si se llena.
  command: ["redis-server", "--save", "", "--appendonly", "no", "--maxmemory", "128mb", "--maxmemory-policy", "allkeys-lru"]
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 3s
    retries: 12
  networks: [cluster]
```
<!-- /generado -->

**Parte A.** Los servicios `loader`, `gold` y `entrenar` pertenecen al perfil `datos`: no se levantan con un `up` normal, solo con `--profile datos`. Se encadenan con `depends_on: condition: service_completed_successfully`, de modo que `gold` empieza cuando `loader` terminó bien, y `entrenar` cuando terminó `gold`. Usan `network_mode: none`: no tienen red, porque solo transforman archivos.

<!-- generado:archivo:deploy/docker-compose.yml#datos -->
```yaml
# deploy/docker-compose.yml
loader:
  <<: *imagen-datos
  profiles: [datos]
  user: "${USUARIO:-65532:65532}"
  command: ["/usr/local/bin/loader"]
  volumes:
    - ../data:/trabajo/data
    - ../reports:/trabajo/reports
  network_mode: none
gold:
  <<: *imagen-datos
  profiles: [datos]
  user: "${USUARIO:-65532:65532}"
  command: ["/usr/local/bin/gold", "-shards=${NODOS:-3}"]
  volumes:
    - ../data:/trabajo/data
    - ../reports:/trabajo/reports
  network_mode: none
  depends_on:
    loader:
      condition: service_completed_successfully
entrenar:
  <<: *imagen-datos
  profiles: [datos]
  user: "${USUARIO:-65532:65532}"
  command: ["/usr/local/bin/entrenar", "-epocas=${EPOCAS:-600}", "-tasa=${TASA:-2}", "-lambda=${LAMBDA:-0.0001}"]
  volumes:
    - ../data:/trabajo/data
    - ../reports:/trabajo/reports
  network_mode: none
  depends_on:
    gold:
      condition: service_completed_successfully
```
<!-- /generado -->

Tabla: Variables de entorno del clúster {#tab:variables}

| Variable | Valor por defecto | Para qué |
|---|---|---|
| `NODOS` | 3 | Réplicas del servicio `node` y nodos que el coordinador espera; también el número de shards que genera `gold` |
| `EPOCAS` | 600 | Épocas de descenso de gradiente en el modo de entrenamiento único |
| `TASA` | 2 | Tasa de aprendizaje |
| `LAMBDA` | 0.0001 | Regularización L2; viaja a los nodos al registrarse |
| `VERIFICAR` | false | Repite el entrenamiento en secuencial dentro del contenedor y falla si los pesos no son idénticos bit a bit |
| `PROGRESO` | 50 | Cada cuántas épocas se registra la pérdida en el log |
| `TIMEOUT_REGISTRO` | 120s | Espera máxima a que se registren los N nodos |
| `TIMEOUT_EPOCA` | 60s | Plazo de cada shard en una época; vencido, el shard se reasigna |
| `WORKERS` | 0 | Goroutines del *worker pool* de cada nodo (0 = núcleos disponibles) |
| `GOLD` | `../data/gold` | Carpeta del anfitrión con los shards y el modelo |
| `USUARIO` | 65532:65532 | uid:gid de los contenedores que escriben en `data/` y `reports/` |
| `API` | vacío | Vacío: entrena una vez y termina. Con una dirección (p. ej. `:8080`): modo servicio (sección 5.5) |

Nota: Fuente: comentario inicial y servicios de `deploy/docker-compose.yml`.

## 5.3 La API, las bases de datos y la SPA: docker-compose.api.yml

El modo servicio se arma combinando el archivo anterior con `deploy/docker-compose.api.yml`. Compose fusiona los dos: este archivo **agrega** al servicio `coordinator` lo que solo necesita como API (el puerto publicado, las variables de MongoDB, Redis, JWT y CORS, y la espera a que las bases estén sanas) y define el servicio `web`. Así el modo de entrenamiento único no publica puertos ni depende de las bases de datos.

<!-- generado:archivo:deploy/docker-compose.api.yml -->
```yaml
# deploy/docker-compose.api.yml
# Modo servicio: publica la API del coordinador en el anfitrión y la conecta a
# MongoDB (persistencia) y Redis (caché). Se usa junto con docker-compose.yml y
# la variable API (ver ese archivo):
#   API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml up --build
# La SPA (servicio web, #62) queda en http://localhost:8081 y llama a la API en
# http://localhost:8080 desde el navegador: son orígenes distintos (CORS real).
# El modo de entrenamiento único no publica puertos ni usa las bases, así que
# no choca con otro servicio que ya use el 8080 en la máquina.
services:
  coordinator:
    ports:
      - "${PUERTO_API:-8080}:8080"
    environment:
      MONGO_URI: mongodb://mongo:27017
      REDIS_ADDR: redis:6379
      # reports/ ya está montado en /reports (docker-compose.yml): mapa y evaluación (#66).
      EVALUACION: /reports/evaluacion.json
      # Login (ADR-0007): secreto del JWT y cuentas iniciales, desde deploy/.env
      # (no versionado; ver deploy/.env.example). Sin JWT_SECRETO el coordinador no arranca (#61).
      JWT_SECRETO: ${JWT_SECRETO:-}
      VIGIA_ADMIN_CORREO: ${VIGIA_ADMIN_CORREO:-}
      VIGIA_ADMIN_CLAVE: ${VIGIA_ADMIN_CLAVE:-}
      VIGIA_ANALISTA_CORREO: ${VIGIA_ANALISTA_CORREO:-}
      VIGIA_ANALISTA_CLAVE: ${VIGIA_ANALISTA_CLAVE:-}
      # Orígenes del navegador que pueden llamar a la API (CORS, ADR-0007): la SPA.
      # Para entrar por IP en la sustentación: CORS_ORIGENES=http://<ip>:8081
      CORS_ORIGENES: ${CORS_ORIGENES:-http://localhost:${PUERTO_WEB:-8081}}
      # Modo demo (#65): DEMO=true habilita «Tumbar nodo» (POST /api/v1/demo/nodos/{nombre}/cortar)
      # y ritmo_ms en los entrenamientos, para la sustentación. Apagado por defecto.
      DEMO: ${DEMO:-false}
      # Réplica del panel admin (#69): EVENTOS=/reports/eventos_demo.jsonl graba cada
      # evento del clúster en ../reports para reproducirlo luego sin clúster. Vacío = no graba.
      EVENTOS: ${EVENTOS:-}
    depends_on:
      mongo:
        condition: service_healthy
      redis:
        condition: service_healthy

  # SPA Vigía: imagen aparte servida por cmd/web (ADR-0007). API_URL es la
  # dirección de la API vista desde el navegador del usuario, no desde la red
  # de Docker.
  web:
    image: cc65-tf-web
    build:
      context: ..
      dockerfile: deploy/Dockerfile.web
    environment:
      # Para entrar desde otra máquina (p. ej. por IP en la sustentación): API_URL=http://<ip>:8080
      API_URL: ${API_URL:-http://localhost:${PUERTO_API:-8080}}
    ports:
      - "${PUERTO_WEB:-8081}:8081"
    depends_on: [coordinator]
```
<!-- /generado -->

Los secretos no están en el repositorio. Compose los lee de `deploy/.env`, que no se versiona; el repositorio trae la plantilla `deploy/.env.example`. Sin `JWT_SECRETO` el coordinador no arranca, y las claves de las cuentas iniciales deben tener al menos 12 caracteres (sección 15.1).

<!-- generado:archivo:deploy/.env.example -->
```bash
# deploy/.env.example
# Copia este archivo como deploy/.env (no se versiona) y cambia los valores.
# docker compose lo lee solo al levantar el modo servicio.

# Secreto del JWT HS256 (ADR-0007), obligatorio: sin él el coordinador no arranca.
# Al menos 32 bytes aleatorios. Genera uno con
#   openssl rand -base64 48
# o en PowerShell (generador criptográfico; Get-Random NO sirve para secretos):
#   $b = New-Object byte[] 48; [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b); [Convert]::ToBase64String($b)
JWT_SECRETO=

# Cuentas que el coordinador crea o actualiza al arrancar (claves de ≥ 12 caracteres).
VIGIA_ADMIN_CORREO=admin@vigia.pe
VIGIA_ADMIN_CLAVE=
VIGIA_ANALISTA_CORREO=analista@vigia.pe
VIGIA_ANALISTA_CLAVE=

# Orígenes del navegador permitidos por CORS (ADR-0007), separados por coma. Por
# defecto, la SPA en http://localhost:8081. Si entras por IP, agrega http://<ip>:8081.
# CORS_ORIGENES=http://localhost:8081

# Modo demo (#65): habilita «Tumbar nodo» y la pausa entre épocas (ritmo_ms) para la
# sustentación. Déjalo en false fuera de la demostración.
# DEMO=true
```
<!-- /generado -->

Tabla: Variables del modo servicio {#tab:variables-api}

| Variable | Valor por defecto | Para qué |
|---|---|---|
| `PUERTO_API` | 8080 | Puerto del anfitrión que publica la API |
| `PUERTO_WEB` | 8081 | Puerto del anfitrión que publica la SPA |
| `JWT_SECRETO` | — (obligatorio) | Clave HS256 con la que se firman los tokens |
| `VIGIA_ADMIN_CORREO`, `VIGIA_ADMIN_CLAVE` | — | Cuenta de administrador que se crea o actualiza al arrancar |
| `VIGIA_ANALISTA_CORREO`, `VIGIA_ANALISTA_CLAVE` | — | Cuenta de analista |
| `CORS_ORIGENES` | `http://localhost:8081` | Orígenes del navegador que pueden llamar a la API |
| `API_URL` | `http://localhost:8080` | Dirección de la API vista desde el navegador; la SPA la lee de `/config.json` |
| `DEMO` | false | Habilita «Tumbar nodo» y la pausa entre épocas (`ritmo_ms`) para la demostración |
| `EVENTOS` | vacío | Graba los eventos del clúster en un archivo `.jsonl` para reproducirlos en el panel |
| `MONGO_URI`, `REDIS_ADDR`, `EVALUACION` | Fijas en el archivo | Conexión a las bases de datos y ruta de `evaluacion.json` dentro del contenedor |

Nota: Fuente: `deploy/docker-compose.api.yml` y `deploy/.env.example`.

## 5.4 Red, puertos y volúmenes

Todos los servicios del clúster están en la red `cluster`, una red puente (*bridge*) de Docker en la que se encuentran por su nombre de servicio (`coordinator`, `mongo`, `redis`). Solo se publican en el anfitrión los dos puertos que usa el navegador; el protocolo de los nodos y las bases de datos no son alcanzables desde fuera (@tab:puertos).

Tabla: Puertos de cada servicio {#tab:puertos}

| Servicio | Puerto | Protocolo | Publicado en el anfitrión | Quién se conecta |
|---|---|---|---|---|
| `coordinator` | 9000 | TCP, JSON por línea | No (`expose`) | Los nodos |
| `coordinator` | 8080 | HTTP: REST y WebSocket | Sí, en modo servicio (`PUERTO_API`) | El navegador (SPA), Postman, `cmd/benchapi` |
| `web` | 8081 | HTTP: archivos estáticos | Sí (`PUERTO_WEB`) | El navegador |
| `mongo` | 27017 | Protocolo de MongoDB | No | El coordinador |
| `redis` | 6379 | Protocolo de Redis | No | El coordinador |

Nota: `deploy/docker-compose.bd.yml` publica MongoDB y Redis solo en `127.0.0.1`, para los tests de integración en el anfitrión; no es parte del despliegue.

Tabla: Volúmenes y montajes {#tab:volumenes}

| Origen en el anfitrión | Destino | Modo | Servicios | Qué contiene |
|---|---|---|---|---|
| `data/gold` (`GOLD`) | `/datos` | Solo lectura | `coordinator`, `node` | Shards, `train.bin` y `modelo.json` |
| `reports/` | `/reports` | Lectura y escritura | `coordinator` | Reporte del entrenamiento, `evaluacion.json`, eventos grabados |
| `data/`, `reports/` | `/trabajo/data`, `/trabajo/reports` | Lectura y escritura | `loader`, `gold`, `entrenar` | Capas bronze, silver y gold, y sus reportes |
| Volumen `mongo-datos` | `/data/db` | Lectura y escritura | `mongo` | Base de datos `cc65_dengue` |

Nota: Fuente: `deploy/docker-compose.yml` y `deploy/docker-compose.api.yml`.

## 5.5 Secuencia de arranque y qué inicia el entrenamiento

El coordinador tiene dos modos, según la variable `API` (@fig:estados-coordinador):

- **Entrenamiento único** (`API` vacía): carga los shards, espera a que se registren `NODOS` nodos, entrena `EPOCAS` épocas, verifica contra el secuencial si `VERIFICAR=true`, escribe `reports/entrenamiento_cluster.json`, avisa `fin` a los nodos y termina. **Lo que inicia el entrenamiento es el registro del último nodo.** Con `--exit-code-from coordinator`, Compose detiene los demás contenedores y devuelve el código de salida del coordinador, lo que permite usarlo en el CI.
- **Servicio** (`API=:8080`): lee el modelo inicial, conecta con MongoDB y Redis, crea las cuentas de `deploy/.env`, carga los shards y abre la API **antes** de que lleguen los nodos: la salud, el modelo y las predicciones responden desde el primer momento. Cuando los N nodos se registran, la API acepta entrenamientos. **Lo que inicia un entrenamiento es una petición `POST /api/v1/entrenamientos`** de un usuario con rol de administrador, normalmente desde el panel de la SPA, con las épocas, la tasa y λ en el cuerpo. Al terminar, el modelo nuevo se publica sin detener la API y se guarda en MongoDB.

![Estados del coordinador en sus dos modos de arranque](fig/estados_coordinador.svg){#fig:estados-coordinador}

Nota: Máquina de estados UML 2.5; las transiciones se leen «evento [guarda] / acción». Fuente: `cmd/coordinator/main.go` y `cmd/coordinator/api.go`; diagrama en `docs/informe/tb2/diagramas/estados_coordinador.dot`.

En el modo servicio, el orden de arranque de los contenedores lo fijan las dependencias de Compose: MongoDB y Redis primero, hasta que su prueba de salud responde; luego el coordinador; luego los nodos y la SPA. Un nodo puede arrancar antes de que el coordinador escuche, así que no depende de ese orden: reintenta la conexión hasta un plazo y, si la pierde a mitad de un entrenamiento, vuelve a registrarse.

<!-- generado:fuente:cmd/node/main.go#conectar -->
```go
// cmd/node/main.go
func conectar(op opciones, logf func(string, ...any)) (net.Conn, error) {
	limite := time.Now().Add(op.espera)
	for {
		conn, err := net.DialTimeout("tcp", op.coordinador, 5*time.Second)
		if err == nil {
			return conn, nil
		}
		if time.Now().Add(op.reintento).After(limite) {
			return nil, fmt.Errorf("no se pudo conectar a %s en %v: %w", op.coordinador, op.espera, err)
		}
		logf("coordinador no disponible (%v); reintento en %v", err, op.reintento)
		time.Sleep(op.reintento)
	}
}
```
<!-- /generado -->

El modo único está en `ejecutar`: prepara el clúster, espera los nodos, entrena con `ml.EntrenarFalible` (que admite que un nodo falle) y, con `-verificar`, repite el entrenamiento en secuencial y compara los pesos bit a bit.

<!-- generado:fuente:cmd/coordinator/main.go#ejecutar -->
```go
// cmd/coordinator/main.go
// ejecutar corre un entrenamiento completo sobre ln. guardar recibe el reporte
// (también el de un entrenamiento abortado, con lo avanzado) antes de que los
// nodos reciban `fin`: mientras el coordinador verifica y escribe, los nodos
// siguen vivos y nadie da el despliegue por terminado.
func ejecutar(ln net.Listener, op opciones, logf func(string, ...any), guardar func(reporte) error) (reporte, error) {
	if guardar == nil {
		guardar = func(reporte) error { return nil }
	}
	c, p, err := prepararCluster(ln, op, logf)
	if err != nil {
		return reporte{}, err
	}
	defer c.Cerrar()
	if err := c.EsperarNodos(op.nodos); err != nil {
		return reporte{}, err
	}
	d, cfg := p.d, p.cfg
	pos, neg := cfg.PesoPos, cfg.PesoNeg

	ent := ml.Entrenamiento{Epocas: op.epocas, Tasa: op.tasa}
	ini := time.Now()
	res, errEnt := ml.EntrenarFalible(&conProgreso{CalculadorFalible: c, cada: op.progreso, logf: logf}, d, ent)
	seg := time.Since(ini).Seconds()

	rep := reporte{
		Generado: time.Now().Format(time.RFC3339), NodosPedidos: op.nodos, Shards: len(p.shards),
		Epocas: len(res.Historia), Tasa: op.tasa, Lambda: op.lambda, PesoPos: pos, PesoNeg: neg,
		FilasTrain: len(p.y), D: d, TamBloque: p.tam, Bloques: p.bloques,
		SegTotal: seg, SegPorEpoca: seg / float64(max(len(res.Historia), 1)), Cluster: c.Estado(),
		Historia: res.Historia, Pesos: res.W[:d], Sesgo: res.W[d],
	}
	if errEnt != nil {
		return rep, errors.Join(errEnt, guardar(rep))
	}
	logf("entrenamiento distribuido: %d épocas en %.1f s (%.1f ms/época), pérdida %.6f → %.6f, %d reasignaciones",
		op.epocas, seg, 1000*rep.SegPorEpoca, res.Historia[0], res.Historia[len(res.Historia)-1], rep.Cluster.Reasignaciones)

	if op.verificar {
		train, err := gold.Leer(filepath.Join(op.gold, "train.bin"))
		if err != nil {
			c.Terminar()
			return rep, fmt.Errorf("-verificar necesita train.bin: %w", err)
		}
		ini := time.Now()
		ref := ml.Entrenar(ml.NuevoSecuencial(train.Dataset, cfg), d, ent)
		rep.SegSecuencial = time.Since(ini).Seconds()
		igual := mismosBits(ref.W, res.W) && mismosBits(ref.Historia, res.Historia)
		rep.Verificado = &igual
		if !igual {
			err := errors.New("los pesos del clúster difieren de los secuenciales")
			defer c.Terminar()
			return rep, errors.Join(err, guardar(rep))
		}
		logf("verificado: pesos e historia idénticos bit a bit al secuencial (%.1f s)", rep.SegSecuencial)
	}
	err = guardar(rep)
	c.Terminar()
	return rep, err
}
```
<!-- /generado -->

En el modo servicio, la API llama a `Entrenar` de `entrenadorCluster` en cada `POST /api/v1/entrenamientos`. Antes de empezar comprueba que los nodos estén registrados y que el λ pedido sea el del clúster, porque la regularización se envía a los nodos al registrarse.

<!-- generado:fuente:cmd/coordinator/api.go#(*entrenadorCluster).Entrenar -->
```go
// cmd/coordinator/api.go
// Entrenar corre p.Epocas épocas en el clúster y devuelve los pesos con el
// sesgo al final. La regularización viaja a los nodos al registrarse
// (protocolo.ConfigML), así que un λ distinto al del clúster se rechaza en vez
// de entrenar en silencio con otro.
func (e *entrenadorCluster) Entrenar(ctx context.Context, p api.ParametrosEntrenamiento, progreso func(int, float64)) ([]float64, error) {
	if !e.listo.Load() {
		if err := e.sinNodos.Load(); err != nil {
			return nil, fmt.Errorf("el clúster no tiene sus nodos: %w", *err)
		}
		return nil, errors.New("el clúster todavía espera a que se registren sus nodos")
	}
	if p.D != e.d {
		return nil, fmt.Errorf("el modelo tiene %d variables y el clúster %d", p.D, e.d)
	}
	if p.Lambda != e.lambda {
		return nil, fmt.Errorf("el clúster está configurado con lambda=%g (flag -lambda del coordinador); se pidió %g", e.lambda, p.Lambda)
	}
	calc := &conContexto{ctx: ctx, calc: e.calc, progreso: progreso, cada: e.progreso, logf: e.logf,
		ritmo: time.Duration(p.RitmoMs) * time.Millisecond}
	ini := time.Now()
	res, err := ml.EntrenarFalible(calc, e.d, ml.Entrenamiento{Epocas: p.Epocas, Tasa: p.Tasa})
	if err != nil {
		return nil, err
	}
	e.logf("entrenamiento vía API: %d épocas en %.1f s, pérdida %.6f → %.6f",
		len(res.Historia), time.Since(ini).Seconds(), res.Historia[0], res.Historia[len(res.Historia)-1])
	return res.W, nil
}
```
<!-- /generado -->

## 5.6 Archivos de ejecución: Makefile y scripts

Las órdenes largas están en el `Makefile` y en `scripts/`, para que cualquier integrante (y el CI) ejecute exactamente lo mismo (@tab:make y @tab:scripts).

Tabla: Objetivos del Makefile {#tab:make}

| Objetivo | Qué ejecuta | Para qué |
|---|---|---|
| `make check` | `gofmt`, `go vet` y `go test -race ./...`, y la sincronía de las skills de los agentes | Lo que el CI exige antes de abrir un Pull Request |
| `make data` | `scripts/descargar-datos.sh` | Descarga el CSV del MINSA y verifica su SHA-256 |
| `make datos-docker` | `docker compose --profile datos up --build entrenar` | Parte A en Docker: CSV → silver → gold y shards → modelo inicial |
| `make cluster` | `docker compose up --build --exit-code-from coordinator` | Entrenamiento único en el clúster |
| `make servicio` | `API=:8080 docker compose -f … -f deploy/docker-compose.api.yml up --build` | Sistema completo: SPA, API, clúster, MongoDB y Redis |
| `make prueba-caida` | `scripts/prueba-caida.sh` | Tolerancia a fallos: mata un nodo a mitad del entrenamiento |
| `make bench` | `go run ./cmd/benchmark` | Speedup secuencial frente a concurrente |
| `make bench-api` | `go run ./cmd/benchapi` | Latencia de la API con carga concurrente |
| `make evaluacion` | `go run ./cmd/evaluacion` | Métricas del modelo en el conjunto de prueba |
| `make imagen` | `docker build -f deploy/Dockerfile -t cc65-tf .` | Solo la imagen del clúster |
| `make web` | `npm ci`, tipos, tests y *build* de la SPA | Lo mismo que el trabajo «SPA» del CI |
| `make figuras` | `scripts/figuras.sh` | Diagramas UML del informe con Graphviz |
| `make informe` | `go run ./cmd/informe` y `go run ./cmd/word` | Cifras, figuras y fragmentos del informe, y el documento Word |

Nota: Fuente: `Makefile` (cada objetivo lleva un comentario `##` con su descripción).

Tabla: Scripts del repositorio {#tab:scripts}

| Script | Qué hace |
|---|---|
| `scripts/descargar-datos.sh` | Descarga el dataset del portal; la primera vez registra su SHA-256 en `data/MANIFEST.sha256` y luego falla si el archivo cambió |
| `scripts/prueba-caida.sh` | Levanta el clúster, mata un nodo a mitad del entrenamiento y exige que el coordinador reasigne su shard y que los pesos sean idénticos bit a bit al secuencial |
| `scripts/evidencia-api.sh` | Levanta el modo servicio con secretos de un solo uso, inicia sesión y prueba la API desde el anfitrión con JWT |
| `scripts/evidencia-bd.sh` | Comprueba en MongoDB y Redis la cadena completa de un entrenamiento: modelo, entrenamiento, predicciones y claves en caché |
| `scripts/revalidar.sh` | Regenera en la máquina actual perfil, limpieza, gold y entrenamiento, los compara campo por campo con los reportes versionados y exige pesos idénticos con 1 y 16 workers |
| `spin/verificar.sh` | Verifica el modelo Promela del protocolo y sus mutantes con Spin |
| `scripts/figuras.sh` | Dibuja los diagramas UML del informe |
| `scripts/word-final.ps1` | Abre el Word generado, actualiza los índices y exporta el PDF (Windows) |
| `scripts/check-skills-sync.sh` | Comprueba que las skills de los dos agentes sean idénticas |

Nota: Fuente: comentario inicial de cada script.

## 5.7 Ejecución paso a paso

Desde un clon limpio del repositorio, con Docker y Compose v2, el sistema completo se levanta con cuatro órdenes. En Linux se crean antes las carpetas (`mkdir -p data reports`; si no existen, Docker las crea como `root`) y se antepone `USUARIO="$(id -u):$(id -g)"` para que los contenedores escriban en ellas con el usuario propio; en PowerShell, las variables se fijan antes con `$env:API=":8080"`.

1. **Datos.** `make data` descarga el CSV a `data/bronze/` y verifica su SHA-256. El portal responde `HTTP 418` a clientes sin navegador y a centros de datos; en ese caso se descarga el archivo con un navegador, se guarda como `data/bronze/vigilancia_dengue_2000_2025.csv` y `make data` solo verifica el checksum.
2. **Parte A.** `docker compose -f deploy/docker-compose.yml --profile datos up entrenar` ejecuta `loader`, `gold` y `entrenar` en orden y deja el gold, los shards y `modelo.json` en `data/gold/`.
3. **Secretos.** `cp deploy/.env.example deploy/.env` y completar `JWT_SECRETO` y las claves de las dos cuentas.
4. **Sistema.** `API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml up --build` levanta siete contenedores: el coordinador, tres nodos, MongoDB, Redis y la SPA. La SPA queda en <http://localhost:8081>, la API en <http://localhost:8080> y Swagger en <http://localhost:8080/docs>.

Las salidas siguientes se guardaron con `scripts/evidencias-tb2.sh` en la máquina de referencia: cada una lleva la fecha, la máquina y el commit con que se ejecutó, y ninguna se editó a mano (la salida completa de Docker queda en el archivo `.log` del mismo nombre).

**Parte A (paso 2).** Los tres servicios terminan con código 0 y dejan el gold, los shards y el modelo inicial en `data/gold/`.

<!-- evidencia: evidencias/parte-a.txt -->

**Clúster con verificación.** El modo de entrenamiento único con `VERIFICAR=true` registra los nodos, entrena y confirma que los pesos son idénticos bit a bit al entrenamiento secuencial. Su salida se presenta en la sección 9.5, junto con las demás pruebas de esa propiedad, y la prueba de caída de un nodo en la sección 9.4.

**Sistema completo (paso 4).** `docker compose ps` muestra los siete contenedores; MongoDB y Redis en estado `healthy`, que es lo que espera el coordinador antes de arrancar. El log del coordinador sigue el orden de la @fig:estados-coordinador: la API responde antes de que lleguen los nodos y acepta entrenamientos cuando se registran todos.

<!-- evidencia: evidencias/servicio.txt -->
