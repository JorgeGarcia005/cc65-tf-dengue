# deploy/

Despliegue del clúster de entrenamiento con Docker Compose: un **coordinador** (parameter
server) y **N nodos de ML** que se conectan a él por TCP (`coordinator:9000`, red interna).
MongoDB, Redis, la API y la SPA se agregan a este mismo archivo en los issues #28 a #30 y en TB2.

## Imagen

`deploy/Dockerfile` construye **una sola imagen** (`cc65-tf`) con los dos comandos que corren en
contenedores, `coordinator` y `node`. El resto de `cmd/` (`gold`, `entrenar`, `benchapi`,
`informe`…) se ejecuta en el host con `go run`:

- dos etapas: `golang:1.24` compila y `gcr.io/distroless/static-debian12:nonroot` ejecuta;
- binarios estáticos (`CGO_ENABLED=0`) con `-trimpath -ldflags="-s -w"`;
- sin shell ni gestor de paquetes, y con usuario sin privilegios (`nonroot`, uid 65532).

```bash
docker build -f deploy/Dockerfile -t cc65-tf .
docker image inspect cc65-tf --format '{{.Config.User}} {{.Size}}'
```

## Entrenar en el clúster

```bash
go run ./cmd/gold -shards 3     # una vez: escribe data/gold/shard_0..2.bin
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
