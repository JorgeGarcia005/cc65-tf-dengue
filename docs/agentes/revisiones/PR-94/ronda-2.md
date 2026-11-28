# Veredicto — PR-94 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/93-despliegue` @ `68c3e14`
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

Los **2 hallazgos** de la ronda 1 (1 mayor y 1 menor) quedaron resueltos y verificados de extremo a extremo en contenedores reales:

- **Construcción autónoma de `cc65-tf-datos` en el camino «Sin el CSV» (H1):** `deploy/docker-compose.yml:35-40, 112, 121, 133` define el ancla YAML `x-imagen-datos: &imagen-datos` con `image: cc65-tf-datos` y `build: { context: .., dockerfile: deploy/Dockerfile.datos }` compartida por `loader`, `gold` y `entrenar`; `README.md:54, 87` y `Makefile:36` incorporan `mkdir -p data reports` y `--build` en el primer `run --no-deps`; y `.github/workflows/ci.yml:173-175` elimina previamente `cc65-tf-datos` para probar exactamente ese comando en un entorno limpio. Verificado localmente borrando la imagen y ejecutando el flujo completo en Docker: construye `cc65-tf-datos` (`15,3 MB`, `nonroot:nonroot`), genera el gold sintético con `esquema.json` de 31 variables y produce `data/gold/modelo.json` de 32 pesos sin errores.
- **Invariante de finitud en `features.NuevoEsquema` y comentario de `deploy/Dockerfile` (H2):** `internal/features/features.go:90-92` rechaza `NaN` y `±Inf` tanto en `mediaEdad` como en `mediaAno` (cubierto en `internal/features/features_test.go:180-187`), y `deploy/Dockerfile:2-5` remite a `deploy/Dockerfile.datos` (`cc65-tf-datos`) para los binarios de la Parte A.

## Verificación automática

```text
$ git rev-parse --short HEAD
68c3e14

$ git diff --check origin/integracion/tb2...origin/feature/93-despliegue
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && bash -n scripts/descargar-datos.sh scripts/evidencia-bd.sh
skills: ok

$ bash scripts/descargar-datos.sh
Descargando https://www.datosabiertos.gob.pe/sites/default/files/datos_abiertos_vigilancia_dengue_2000_2025_0.csv
ERROR: el portal respondió HTTP 418.
Descárgalo con un navegador desde una red doméstica o de la universidad:
  https://www.datosabiertos.gob.pe/sites/default/files/datos_abiertos_vigilancia_dengue_2000_2025_0.csv
guárdalo como data/bronze/vigilancia_dengue_2000_2025.csv y vuelve a correr este script: verificará el sha256.
(código de salida 1; sin archivos .part residuales)

$ docker image rm -f cc65-tf-datos
$ USUARIO="$(id -u):$(id -g)" docker compose -f deploy/docker-compose.yml --profile datos run --rm --no-deps --build gold /usr/local/bin/gold -sintetico 60000 -shards 3
Image cc65-tf-datos Built
gold sintético: 60000 filas, 31 variables, 15 bloques de 4096 filas, 3 shards en data/gold
$ USUARIO="$(id -u):$(id -g)" docker compose -f deploy/docker-compose.yml --profile datos run --rm --no-deps entrenar /usr/local/bin/entrenar -epocas 20 -salida reports/entrenamiento_sintetico.json
2026/11/28 03:00:48 escrito data/gold/modelo.json
user=nonroot:nonroot size=15342161
modelo.json OK: 32 pesos, 31 variables

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.742s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.907s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.950s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.032s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.634s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.277s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.226s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.419s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.354s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.055s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.154s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.080s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.294s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.719s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.535s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.787s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.099s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.032s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.078s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.801s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.194s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.757s

$ (en web/) npx tsc -b && npx vitest run && npm run build && ! grep -q "url(data:" dist/assets/*.css
Test Files  11 passed (11)
     Tests  170 passed (170)
✓ built in 953ms · sin url(data: en CSS

$ gh pr checks 94 -R JorgeGarcia005/cc65-tf-dengue   # run 46098700442 sobre 68c3e14
Docker (imagen, clúster y caída de un nodo)	pass	2m2s
Ecosistema de agentes                      	pass	9s
Go (gofmt, vet, test -race)                	pass	1m50s
MongoDB y Redis (tests de integración)     	pass	52s
OpenAPI 3 válido                           	pass	21s
SPA (tipos, tests, build e imagen)         	pass	1m1s
Spin (modelo del protocolo y mutantes)     	pass	2m0s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`26/26` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`) |
| Imagen `cc65-tf-datos` y perfil `datos` | ✅ (`15,3 MB`, `nonroot:nonroot`, `network_mode: none`; flujo con y sin CSV verificado desde cero) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (`170/170` tests en verde tras retirar `Pendiente.tsx`, sin `url(data:` en CSS) |
| CI GitHub Actions (PR #94 @ `68c3e14`) | ✅ (`7/7` jobs en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | `README.md`, `deploy/README.md` y `data/README.md` explican de principio a fin las tres imágenes, los puertos y la contingencia verificada por SHA-256 ante el `HTTP 418` del portal MINSA. |
| Implementación | 8 | 8.0 / 8 | Parte A íntegramente contenerizada en Go sobre `distroless/static-debian12:nonroot` sin red (`network_mode: none`), y gold sintético completo (`esquema.json` + `modelo.json` de 32 pesos) reproducible en un clon limpio. |
| Interfaz | 5 | sin cambios | Retira el marcador huérfano `web/src/paginas/Pendiente.tsx`. |
| Informe | 2 | 2.0 / 2 | `scripts/evidencia-bd.sh` audita `usuarios` por rol y formato PBKDF2-SHA256 sin exponer correos ni hashes, y los documentos del proyecto reflejan consistentemente el equipo de dos integrantes. |
| Video | 2 | sin cambios | Alinea la regla de participación a los dos integrantes activos del equipo. |

## Hallazgos

Sin hallazgos abiertos (**0 bloqueantes, 0 mayores, 0 menores**).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — `gold` y `entrenar` sin `build:` en `deploy/docker-compose.yml` · MAYOR | RESUELTO | Resuelto en `6735952` con el ancla `x-imagen-datos` en los tres servicios, `--build` y `mkdir -p data reports` en `README.md` / `Makefile`, y prueba limpia en `.github/workflows/ci.yml:173-175`. |
| H2 (ronda 1) — Validación de medias en `features.NuevoEsquema` y comentario en `deploy/Dockerfile` · MENOR | RESUELTO | Resuelto en `62a8618` (`internal/features/features.go:90-92`, `features_test.go:180-187` y `deploy/Dockerfile:2-5`). |

## Cómo explicarlo en la sustentación

Toda la arquitectura se despliega con Docker Compose usando tres imágenes mínimas sobre `distroless/static-debian12:nonroot` (binarios Go estáticos con `CGO_ENABLED=0`, sin shell y sin privilegios de `root`): `cc65-tf-datos` ejecuta por única vez la Parte A sin acceso a red (`network_mode: none`) encadenando `loader` → `gold` → `entrenar` con `service_completed_successfully`; `cc65-tf` ejecuta el coordinador y las `N` réplicas de nodos ML sobre la red interna junto a MongoDB y Redis; y `cc65-tf-web` sirve la SPA con cabeceras de seguridad y CSP estricta. Si el firewall del portal de datos abiertos bloquea la descarga automatizada con `HTTP 418`, el script guía la descarga manual verificada contra `data/MANIFEST.sha256`, o permite generar en contenedores un `gold` y `modelo.json` sintéticos de 31 variables para levantar y probar toda la infraestructura desde cero sin el CSV.
