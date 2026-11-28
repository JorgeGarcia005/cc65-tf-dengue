# Veredicto — PR-94 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/93-despliegue` @ `28e6c8a`
- **Fecha:** 2026-11-27
- **Decisión:** CAMBIOS

El PR cierra casi por completo la brecha B1 (y su parte de B11 y B12) de la auditoría integral TB2: incorpora la imagen multi-etapa `cc65-tf-datos` (`deploy/Dockerfile.datos`, `15,3 MB`, `nonroot:nonroot`, binarios estáticos sobre `distroless/static-debian12`) y el perfil `datos` en `deploy/docker-compose.yml` (`loader` → `gold` → `entrenar` encadenados con `service_completed_successfully` y `network_mode: none`), genera `esquema.json` de 31 variables en `cmd/gold -sintetico` (`features.NuevoEsquema`) para que `cmd/entrenar` produzca un `modelo.json` de 32 pesos sin el CSV del MINSA, maneja con un mensaje accionable el `HTTP 418` del WAF de `datosabiertos.gob.pe` en `scripts/descargar-datos.sh`, verifica `usuarios` y hashes PBKDF2-SHA256 sin filtrar secretos en `scripts/evidencia-bd.sh`, actualiza la guía de despliegue y refleja el retiro de Marco (equipo de 2 integrantes, admitido por el enunciado). Todas las verificaciones automáticas y los 7 jobs del CI pasan en verde. Se solicita un único ajuste **MAYOR** (**H1**) y dos detalles **MENORES** (**H2**) antes de aprobar: en `deploy/docker-compose.yml` solo `loader` declara `build:`, por lo que el primer comando del camino **«Sin el CSV»** en `README.md:86` (`docker compose … --profile datos run --rm --no-deps gold …`) falla en una máquina limpia intentando descargar `cc65-tf-datos` de Docker Hub incluso si se añade `--build`.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && bash -n scripts/descargar-datos.sh && bash -n scripts/evidencia-bd.sh
skills: ok

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	10.965s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.472s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.524s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.640s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.253s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.234s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.409s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.187s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.051s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.141s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.038s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	25.605s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	11.830s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.843s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.866s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.046s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.995s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.078s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.317s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	12.121s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.745s

$ bash scripts/descargar-datos.sh
Descargando https://www.datosabiertos.gob.pe/sites/default/files/datos_abiertos_vigilancia_dengue_2000_2025_0.csv
ERROR: el portal respondió HTTP 418.
Descárgalo con un navegador desde una red doméstica o de la universidad:
  https://www.datosabiertos.gob.pe/sites/default/files/datos_abiertos_vigilancia_dengue_2000_2025_0.csv
guárdalo como data/bronze/vigilancia_dengue_2000_2025.csv y vuelve a correr este script: verificará el sha256.
(código de salida 1; sin archivos .part residuales)

$ docker compose -f deploy/docker-compose.yml --profile datos run --rm --no-deps gold /usr/local/bin/gold -sintetico 1000 -shards 1 -salida /tmp/test_gold
Image cc65-tf-datos Pulling
Error response from daemon: pull access denied for cc65-tf-datos, repository does not exist or may require 'docker login'
(falla igual pasando --build, porque gold no declara build: en docker-compose.yml; ver H1)

$ docker compose -f deploy/docker-compose.yml --profile datos build loader && docker image inspect cc65-tf-datos --format 'User={{.Config.User}} Size={{.Size}}'
User=nonroot:nonroot Size=15342025
(tras construir loader manualmente, gold -sintetico y entrenar dentro de cc65-tf-datos generan modelo.json con 32 pesos y 31 nombres)

$ (en web/) npx tsc -b && npx vitest run && npm run build && ! grep -q "url(data:" dist/assets/*.css
Test Files  11 passed (11)
     Tests  170 passed (170)
✓ built in 1.35s · sin url(data: en CSS

$ gh pr checks 94 -R JorgeGarcia005/cc65-tf-dengue   # run 46080295751 sobre 28e6c8a
7/7 jobs en verde (Ecosistema, Spin, OpenAPI 3, MongoDB/Redis, Go -race, SPA, Docker)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`25/25` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`) |
| Imagen `cc65-tf-datos` y perfil `datos` | ⚠️ (`15,3 MB`, `nonroot:nonroot`, `network_mode: none`; falta `build:` en `gold` y `entrenar` para el flujo sin CSV, ver **H1**) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (`170/170` tests en verde tras retirar `Pendiente.tsx`, sin `url(data:` en CSS) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | `README.md`, `deploy/README.md` y `data/README.md` documentan con claridad las 3 imágenes, los puertos y la contingencia ante el `HTTP 418` del portal MINSA. |
| Implementación | 8 | Pendiente **H1** | La Parte A queda íntegramente contenerizada en Go sobre `distroless:nonroot` y sin red; compartir `build:` en `gold` y `entrenar` (**H1**) cierra el único tropiezo al probar desde cero sin el CSV. |
| Interfaz | 5 | sin cambios | Solo elimina el componente huérfano `web/src/paginas/Pendiente.tsx`. |
| Informe | 2 | mejora evidencias | `scripts/evidencia-bd.sh` incorpora la verificación de `usuarios` (conteo por rol y formato `pbkdf2-sha256$<iter>$…` sin exponer correos ni hashes) y el repo refleja consistentemente el equipo de 2 integrantes. |
| Video | 2 | sin cambios | No afecta el video salvo alinear la regla a los dos integrantes activos. |

## Hallazgos

### H1 — `gold` y `entrenar` no declaran `build:` en `deploy/docker-compose.yml`, por lo que el comando «Sin el CSV» de `README.md` falla en un clon limpio · MAYOR

- **Dónde:** `deploy/docker-compose.yml:107-142` y `README.md:85-88` (evidenciado también en `.github/workflows/ci.yml:172`)
- **Evidencia:**
  1. En `deploy/docker-compose.yml:107-142`, únicamente `loader` tiene `build: { context: .., dockerfile: deploy/Dockerfile.datos }`, mientras `gold` (`:119-121`) y `entrenar` (`:131-133`) solo tienen `image: cc65-tf-datos` (a diferencia de `coordinator` y `node` en las líneas `25-60`, que sí comparten el ancla YAML `x-imagen: &imagen` con `image` + `build`).
  2. Si un evaluador clona el repositorio, corre `make data`, recibe `HTTP 418` del portal MINSA y sigue las instrucciones de contingencia `**Sin el CSV**` de `README.md:85-88`:
     ```bash
     docker compose -f deploy/docker-compose.yml --profile datos run --rm --no-deps gold /usr/local/bin/gold -sintetico 300000 -shards 3
     ```
     como `--no-deps` omite `loader` y `gold` no define `build:`, Docker Compose intenta descargar `cc65-tf-datos` desde Docker Hub y **falla incluso agregando `--build`**:
     ```text
     Image cc65-tf-datos Pulling
     Error response from daemon: pull access denied for cc65-tf-datos, repository does not exist or may require 'docker login'
     ```
     Por esta misma razón, en `.github/workflows/ci.yml:172` fue necesario anteponer `docker compose -f deploy/docker-compose.yml --profile datos build loader`, paso que no aparece en `README.md`.
  3. Adicionalmente, en Linux conviene recordar o incluir `mkdir -p data/gold reports` (o dejar que `gold` cree `data/gold` sobre `../data` ya existente con `USUARIO="$(id -u):$(id -g)"`): si `data/` o `reports/` no existieran en el anfitrión antes del `docker compose … run`, el demonio de Docker los crearía como `root:root` y el usuario sin privilegios recibiría `permission denied`.
- **Corrección sugerida:**
  - Definir en `deploy/docker-compose.yml` un ancla `x-imagen-datos: &imagen-datos` (análoga a `x-imagen: &imagen`) con `image: cc65-tf-datos` y `build: { context: .., dockerfile: deploy/Dockerfile.datos }`, y reutilizar `<<: *imagen-datos` en `loader`, `gold` y `entrenar`. Con ello, `docker compose -f deploy/docker-compose.yml --profile datos run --rm --no-deps --build gold …` (e incluso sin `--build` cuando la imagen aún no existe localmente) construye `cc65-tf-datos` automáticamente desde cero.
  - Añadir `--build` al primer `docker compose … run` de `README.md:86` (y asegurar `mkdir -p data reports` en `Makefile` `datos-docker` por si `reports/` no existiera en algún entorno limpio).

### H2 — Validación de medias en `features.NuevoEsquema` y comentario histórico en `deploy/Dockerfile` · MENOR

- **Dónde:** `internal/features/features.go:86-90` y `deploy/Dockerfile:2-4`
- **Evidencia:**
  1. `NuevoEsquema` rechaza `NaN`, `<= 0` e `Inf` en `desvEdad` y `desvAno`, pero no comprueba finitud (`!math.IsNaN` y `!math.IsInf`) en `mediaEdad` ni `mediaAno`. Aunque `api.CargarModelo` (`internal/api/modelo.go:88-89`) sí lo valida al leer `modelo.json`, comprobarlo en `NuevoEsquema` mantiene el invariante del paquete `features` desde su constructor.
  2. El encabezado de `deploy/Dockerfile:2-4` conserva la frase *«Los demás comandos de cmd/ (gold, entrenar, benchapi, informe…) corren en el host con `go run`»*, que conviene matizar ahora que `loader`, `gold` y `entrenar` cuentan con su propia imagen `cc65-tf-datos` en `deploy/Dockerfile.datos`.
- **Corrección sugerida:** Comprobar finitud de `mediaEdad` y `mediaAno` en `NuevoEsquema`, y apuntar `deploy/Dockerfile:2-4` hacia `deploy/Dockerfile.datos` para los binarios de la Parte A.

## Seguimiento de rondas previas

Primera ronda para este PR (atiende los puntos B1 y parte de B11/B12 de `docs/agentes/revisiones/auditoria-tb2/ronda-1.md`).

## Cómo explicarlo en la sustentación

Toda la arquitectura se despliega con Docker Compose usando tres imágenes mínimas sobre `distroless/static-debian12:nonroot` (binarios Go estáticos con `CGO_ENABLED=0`, sin shell y sin privilegios de `root`): `cc65-tf-datos` ejecuta por única vez la Parte A sin acceso a red (`network_mode: none`) encadenando `loader` → `gold` → `entrenar` con `service_completed_successfully`; `cc65-tf` ejecuta el coordinador y las `N` réplicas de nodos ML sobre la red interna junto a MongoDB y Redis; y `cc65-tf-web` sirve la SPA con cabeceras de seguridad y CSP estricta. Si el firewall del portal de datos abiertos bloquea la descarga automatizada con `HTTP 418`, el script indica la descarga manual verificada contra `data/MANIFEST.sha256`, o permite generar un `gold` y `modelo.json` sintéticos de 31 variables para levantar y probar toda la infraestructura de extremo a extremo sin el CSV.
