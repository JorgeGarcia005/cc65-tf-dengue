# deploy/

Despliegue completo con Docker Compose. La guía paso a paso (datos → modelo inicial → sistema
completo) está en el [README de la raíz](../README.md#cómo-ejecutarlo); aquí va el detalle.

| Archivo | Qué levanta |
|---|---|
| `docker-compose.yml` | Clúster: `coordinator` (parameter server, `:9000` interno) y `node` × `NODOS`, `mongo`, `redis` en la red interna `cluster`. Perfil `datos`: `loader`, `gold` y `entrenar` (Parte A). |
| `docker-compose.api.yml` | Modo servicio: publica la API (`:8080`), conecta MongoDB y Redis, lee `deploy/.env` y agrega la SPA `web` (`:8081`). |
| `docker-compose.bd.yml` | Solo MongoDB y Redis en `127.0.0.1`, para correr la API desde el anfitrión o los tests de integración. |

## Imágenes

| Imagen | Dockerfile | Binarios | Para qué |
|---|---|---|---|
| `cc65-tf` | `Dockerfile` | `coordinator`, `node` | Clúster y API. El CI exige < 40 MB y usuario `nonroot`. |
| `cc65-tf-datos` | `Dockerfile.datos` | `loader`, `gold`, `entrenar` | Parte A: CSV → silver → gold + shards → `modelo.json`. Se usa una vez. |
| `cc65-tf-web` | `Dockerfile.web` | `web` + `dist/` de la SPA | SPA servida por `cmd/web` con CSP estricta. |

Las tres compilan en una etapa `golang:1.24` (la SPA, antes, en `node:22`) y corren sobre
`gcr.io/distroless/static-debian12:nonroot`: binarios estáticos (`CGO_ENABLED=0`,
`-trimpath -ldflags="-s -w"`), sin shell ni gestor de paquetes y con usuario sin privilegios
(uid 65532). La capa de `go mod download` se reutiliza mientras no cambie `go.mod`.

```bash
docker build -f deploy/Dockerfile -t cc65-tf .
docker image inspect cc65-tf --format '{{.Config.User}} {{.Size}}'
```

## Parte A en Docker (perfil `datos`)

```bash
docker compose -f deploy/docker-compose.yml --profile datos up entrenar
```

`entrenar` depende de `gold` y `gold` de `loader` (`condition: service_completed_successfully`), así
que Compose los corre en orden y se detiene si uno falla. Trabajan en `/trabajo` con `../data` y
`../reports` montados, por eso usan las mismas rutas por defecto que `go run`, y no tienen red
(`network_mode: none`). `NODOS` fija los shards de `gold`; `EPOCAS`, `TASA` y `LAMBDA`, el
entrenamiento inicial.

## Entrenar en el clúster

```bash
go run ./cmd/gold -shards 3     # una vez (o el perfil datos): escribe data/gold/shard_0..2.bin
docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator
go run ./cmd/entrenar -comparar reports/entrenamiento_cluster.json
```

El coordinador espera a los nodos, entrena y escribe `reports/entrenamiento_cluster.json`. El
último comando exige que pesos, sesgo y pérdida por época sean **idénticos bit a bit** a los de
`reports/entrenamiento.json` (entrenamiento en un solo proceso, mismas épocas y tasa).

Sin el CSV del MINSA se puede probar todo con datos sintéticos de la misma forma (31 variables):

```bash
go run ./cmd/gold -sintetico 300000 -shards 3 -salida /tmp/gold
GOLD=/tmp/gold EPOCAS=100 VERIFICAR=true docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator
```

El gold sintético incluye un `esquema.json` de demostración con las 31 variables del real, así que
`go run ./cmd/entrenar -gold /tmp/gold -salida /tmp/entrenamiento.json -modelo /tmp/gold/modelo.json`
escribe el `modelo.json` que necesita el modo servicio (`GOLD=/tmp/gold API=:8080 … up`).

| Variable | Qué hace | Por defecto |
|---|---|---|
| `NODOS` | Nodos que se levantan y que el coordinador espera antes de empezar | `3` |
| `EPOCAS`, `TASA`, `LAMBDA` | Hiperparámetros del entrenamiento | `600`, `2`, `0.0001` |
| `VERIFICAR` | Al terminar, repite en secuencial con `train.bin` y exige pesos idénticos | `false` |
| `PROGRESO` | Registra la pérdida cada tantas épocas | `50` |
| `WORKERS` | Goroutines del worker pool de cada nodo (`0` = núcleos del contenedor) | `0` |
| `GOLD` | Carpeta del anfitrión con los shards (se monta de solo lectura) | `../data/gold` |
| `USUARIO` | `uid:gid` del coordinador, que escribe en `reports/` | `65532:65532` |

- **Número de nodos.** Usa `NODOS=5` (levanta 5 y el coordinador espera 5). `--scale node=5` solo
  cambia cuántos se levantan. Los shards se reparten por menor carga, así que no hace falta que
  haya tantos shards como nodos, pero para que los 5 trabajen genera 5 (`cmd/gold -shards 5`).
- **Permisos en Linux.** El coordinador corre como uid 65532 y no puede escribir en un `reports/`
  del usuario del anfitrión. Lánzalo con `USUARIO="$(id -u):$(id -g)"`. En Docker Desktop
  (Windows, macOS) no hace falta.
- **Orden de arranque.** Los contenedores arrancan a la vez; cada nodo reintenta la conexión
  hasta que el coordinador escucha (`cmd/node -espera`).
- **Fin del despliegue.** El coordinador guarda el reporte y recién entonces envía `fin` a los
  nodos. Con `--exit-code-from`, Compose detiene todo cuando el primer contenedor termina; como
  el reporte ya está escrito, el coordinador ignora ese `SIGTERM` y sale con su propio código.

## Tolerancia a fallos

```bash
bash scripts/prueba-caida.sh        # o: make prueba-caida
```

Levanta el clúster, mata un nodo con `docker kill` a mitad del entrenamiento y exige que el
coordinador termine con código 0, que el log muestre el descarte y la reasignación del shard, y que
los pesos finales sean idénticos bit a bit a los secuenciales. Un nodo que se reinicia vuelve a
registrarse y queda de reserva.

## Ver el protocolo

Los mensajes son JSON por línea (ADR-0005), legibles tal cual:

```bash
docker compose -f deploy/docker-compose.yml logs -f coordinator
```

## SPA Vigía (TB2, #62)

`deploy/Dockerfile.web` construye la imagen `cc65-tf-web`: compila la SPA de `web/` (React + Vite)
y la sirve con `cmd/web`, un servidor estático en Go stdlib sobre distroless (`nonroot`). Envía una
CSP estricta (sin `unsafe-inline`), `X-Content-Type-Options`, `Referrer-Policy` y
`frame-ancestors 'none'`, y entrega `/config.json` con la URL de la API tomada de `API_URL` al
arrancar (ADR-0007). En modo servicio se levanta junto a la API:

```bash
API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml up --build
# SPA: http://localhost:8081 · API: http://localhost:8080 (otro origen: CORS real)
```

`PUERTO_WEB` cambia el puerto de la SPA en el anfitrión (8081 por defecto). Para desarrollar sin
Docker: `cd web && npm ci && npm run dev` (usa `web/public/config.json`).

## Login y cuentas (TB2, #60)

El modo servicio lee de `deploy/.env` (no versionado; plantilla en `deploy/.env.example`) el
secreto `JWT_SECRETO` (≥ 32 bytes) y las cuentas `VIGIA_ADMIN_*` y `VIGIA_ANALISTA_*`, que el
coordinador crea o actualiza en MongoDB al arrancar (claves guardadas con PBKDF2-SHA256). También
se pueden crear desde el anfitrión con `go run ./cmd/usuarios -mongo <URI>`. `JWT_SECRETO` es
obligatorio: sin él el coordinador no arranca en modo servicio (#61).

## Roles y CORS (TB2, #61)

Las rutas `/api/v1/*`, salvo el login, exigen `Authorization: Bearer <jwt>`. El analista ve el
modelo, las predicciones, el riesgo y la evaluación; el admin, además, el clúster y los
entrenamientos (matriz en ADR-0007; sin token → 401, rol sin acceso → 403). `CORS_ORIGENES` es la
lista blanca de orígenes del navegador (por defecto, la SPA en `http://localhost:8081`): si entras
a la SPA por IP, agrega `http://<ip>:8081`. El p95 con token se mide con
`BENCH_CLAVE=<clave> go run ./cmd/benchapi -correo analista@vigia.pe -exigir-p95 100ms`, que
escribe `reports/bench_api_<maquina>_jwt.json`.

## Tiempo real y modo demo (TB2, #65)

El panel admin sigue el entrenamiento por WebSocket: la SPA pide `POST /api/v1/ws/ticket` (con su
JWT de admin) y abre `GET /api/v1/ws?ticket=…`; el ticket vale una vez y vence a los 30 s, y el
`Origin` debe estar en `CORS_ORIGENES`. Con `DEMO=true` (o `-demo`) el coordinador expone
`POST /api/v1/demo/nodos/{nombre}/cortar`, que corta la conexión de un nodo para mostrar la
reasignación, y acepta `ritmo_ms` (0–2000) en `POST /api/v1/entrenamientos` para que las épocas se
vean pasar. Sin el modo demo, esa ruta responde 404 y `ritmo_ms` se rechaza.
