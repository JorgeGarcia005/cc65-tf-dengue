# Auditoría integral TB2 — Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama auditada:** `integracion/tb2` @ `a435072` (base `develop` @ `396c4ce`, `main` @ `769fb21`)
- **Fecha:** 2026-11-27
- **Decisión:** **CAMBIOS** (núcleo técnico sólido y operativo de punta a punta, pero con brechas bloqueantes de cierre en Informe, correcciones PC4, participación en Git y Video antes de la entrega del dom 29/11/2026 23:30)

---

## 1. Resumen ejecutivo y puntaje estimado de rúbrica

El sistema distribuido en `integracion/tb2` compila limpio, pasa el detector de carreras en los 25 paquetes con tests (`go test -race -count=1 ./...` aislado: 0 carreras), pasa `tsc -b`, los 170 tests de Vitest y `vite build`, construye dos imágenes `distroless` sin privilegios (`cc65-tf` de 25,2 MB y `cc65-tf-web` de 16,0 MB) y levanta en Docker Compose (`DEMO=true`) los 7 contenedores (`coordinator`, `node` ×3, `mongo`, `redis`, `web`) sirviendo la SPA Vigía con CORS real, CSP estricta, JWT HS256, PBKDF2-SHA256, WebSocket RFC 6455 en biblioteca estándar, corte de nodo en vivo con reasignación de shard, sello `bit a bit ✓` y latencia `p95 < 100 ms`.

Sin embargo, al auditar el repositorio completo frente al enunciado TB2, `AGENTS.md` §3–§4 y las **6 observaciones del profesor a la PC4**, la entrega perdería hoy ~4,6 puntos si se presentara en este estado: no existe todavía `docs/informe/tb2/` (issue `#71`), las 6 observaciones a la PC4 siguen sin atenderse en el informe, faltan las pruebas E2E con Playwright y `reports/seguridad.json` (issue `#70`), un integrante tiene **0 commits** en todo el repositorio (`AGENTS.md` §4), y detectamos fallos reproducibles en `cerrarSesion` de la SPA (React Router v7), en la colección de Postman con `demo=true`, en la versión de modelo al reentrenar con los mismos pesos y en un timeout bajo `-race` de `cmd/benchapi`.

| Criterio (Rúbrica TB2) | Peso | Estado actual | Potencial tras cierre | Resumen |
|---|---:|---:|---:|---|
| **Planteamiento** | 3 | 2,7 / 3 | 3,0 / 3 | Problema, ODS 3, pregunta, ADR-0001..0007 y arquitectura muy bien definidos; falta documentar el pipeline completo de datos antes de `docker compose up` y prever el bloqueo WAF (`HTTP 418`) del portal MINSA (**B1**). |
| **Implementación** | 8 | 7,2 / 8 | 8,0 / 8 | Núcleo 100 % stdlib, determinismo bit a bit y tolerancia a fallos excelentes; brechas en hash de `Version` con `Generado` (**B2**), Postman `demo=true` con `404` (**B3**), 2 tests sensibles a carga en 2 vCPU con `-race` (**B4**) y benchmark sin JWT en `evaluacion.json` (**B5**). |
| **Interfaz** | 5 | 4,3 / 5 | 5,0 / 5 | Las 5 vistas (`Inicio`, `Mapa`, `Predicción`, `Evaluación`, `Clúster`) funcionan en vivo con ambos roles y CSP estricta; corregir bug de `cerrarSesion` en `App.tsx` (**B7**) y cerrar Playwright + capturas automáticas de `#70` (**B8**). |
| **Informe** | 2 | 0,7 / 2 | 2,0 / 2 | **BLOQUEANTE:** falta `docs/informe/tb2/` (**B10**), las 6 observaciones del profesor a la PC4 siguen abiertas (**B11**) y Marco Cabrera registra 0 commits en Git / Kanban incompleto (**B12**). |
| **Video** | 2 | 0,0 / 2 | 2,0 / 2 | **BLOQUEANTE:** pendiente de grabación y enlace (`#72`, **B13**); la plataforma con `DEMO=true` ya está lista para grabarlo. |
| **Total** | **20** | **14,9 / 20** | **20,0 / 20** | Código de producción muy maduro; el riesgo está concentrado en informe, pruebas de evidencia, Git Flow y sustentación. |

---

## 2. Verificación automática y recorrido en vivo (`DEMO=true`)

### 2.1 Backend Go (`gofmt`, `go vet`, `go test -race ./...`)

```text
$ gofmt -l .
(vacío · código 0)

$ go vet ./...
(vacío · código 0)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go test -race -count=1 ./...   # Corrida 1: en paralelo con npm ci / vitest en VM Linux de 2 vCPU
--- FAIL: TestMedirConLoginUsaJWT (15.70s)
    main_test.go:359: login: la API no responde en http://127.0.0.1:40561: Post "http://127.0.0.1:40561/api/v1/auth/login": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	23.835s
(los otros 24 paquetes con tests: ok · 0 carreras de datos)

$ go test -race -count=1 ./...   # Corrida 2: aislada en la misma VM de 2 vCPU
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	12.389s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	10.067s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.461s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.032s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.610s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.278s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.259s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.422s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.443s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.061s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.154s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.128s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	29.258s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.151s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.957s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.839s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.050s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.951s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.082s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.112s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.923s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.790s
(25/25 paquetes con tests en verde · 0 DATA RACE)
```

### 2.2 Frontend `web/` (`tsc`, `vitest`, `build`) y contrato OpenAPI

```text
$ cd web && npm ci --no-audit --no-fund && npx tsc -b && npx vitest run && npm run build
tsc exit: 0
Test Files  11 passed (11)
     Tests  170 passed (170)
  (stderr en src/pruebas/admin.test.tsx > un ticket vacío no abre el WebSocket: 4 avisos "not wrapped in act(...)")
dist/assets/index-BXu4YVOm.css   21.91 kB │ gzip:   5.03 kB
dist/assets/index-DOfSEIgY.js   527.48 kB │ gzip: 178.22 kB  (!) chunk > 500 kB
✓ built in 1.75s (0 ocurrencias de url(data: en CSS)

$ npx @redocly/cli@1.25.0 lint --extends=minimal docs/api/openapi.yaml
docs/api/openapi.yaml: validated in 120ms (0 errores, 8 warnings cosméticos)
```

### 2.3 Docker Compose en modo servicio (`DEMO=true`), recorrido de la SPA con ambos roles, `benchapi`, BD y Newman

```text
$ API=:8080 DEMO=true GOLD=/tmp/gold ... docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml up --build -d
cc65-tf:     user=nonroot:nonroot bytes=25216609 (25,2 MB < 40 MB)
cc65-tf-web: user=nonroot:nonroot bytes=15978377 (16,0 MB)
7 contenedores arriba: coordinator (:8080), web (:8081), node ×3, mongo (healthy), redis (healthy).

Recorrido E2E en Google Chrome headless (Playwright) sobre http://localhost:8081:
- Sin sesión: /mapa redirige a http://localhost:8081/login.
- Login fallido: devuelve 401 y muestra «Correo o clave incorrectos.».
- Rol analista (analista@vigia.pe):
  · Tras login vuelve a /mapa (menú: Inicio, Mapa, Predicción, Evaluación; oculta Clúster).
  · /mapa: 25 departamentos en SVG, filtros de semana/etapa/medida operativos, 21 filas en tabla.
  · / (Inicio): 10 prioridades, semanas pico, etapas de vida, AUC 0.622; muestra aviso de versión si modelo != evaluacion.json.
  · /prediccion: validación de 4 campos vacíos; LORETO 2024 sem 16 edad 9 F → riesgo 21.2%, «sesgo + Σ aportes = logit +0.629 → σ(logit) = 0.652 ✓ cuadra», MISS en primera consulta y «desde la caché (Redis)» (HIT) al repetir; AREQUIPA muestra aviso de agrupación en «Otros».
  · /evaluacion: renderiza en orden roc, calibracion, confusion, escalabilidad, latencia.
  · /admin directo como analista: bloqueado con vista 403 («Esta vista no es para tu rol»).
  · Botón «Salir» desde /admin → redirige a /login, PERO conserva state.desde="/admin" (ver B7).
- Rol admin (admin@vigia.pe):
  · Menú completo: Inicio, Mapa, Predicción, Evaluación, Clúster.
  · /admin: pide ticket de un solo uso, abre WebSocket («En vivo»), lista los 3 nodos con 2 workers cada uno.
  · Entrenamiento en vivo (40 épocas, tasa 2, ritmo 80 ms) + clic en «Tumbar» sobre el 2.º nodo en época >= 4:
    descarta el nodo por error de envío/EOF, reasigna el shard 2 en la misma época (reasignaciones = 1),
    el contenedor del nodo se reconecta automáticamente como reserva («0 shards · 2 workers · cayó 1×»),
    completa 40 / 40 épocas y pinta sello data-sello="ok":
    «bit a bit ✓ Los 32 pesos anunciados por el WebSocket son idénticos (===) a los de GET /api/v1/modelo».
- Consola y seguridad del navegador: 0 pageerror, 0 violaciones de CSP.

$ BENCH_CLAVE=... go run ./cmd/benchapi -url http://localhost:8080 -correo analista@vigia.pe -exigir-p95 100ms
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria         2000       0     1707     8.21    19.70    29.51    41.71      0   2000  p95 < 100 ms: true
caliente     2000       0     1999     7.28    15.89    20.27    28.53   2000      0  p95 < 100 ms: true

$ bash scripts/evidencia-bd.sh
-- modelos: 2 documentos; entrenamientos: 1 documentos; predicciones: 1952 documentos (TTL 7776000s)
-- Redis: 2029 claves pred:387003a097f7:* con TTL 86396 s · OK: cadena completa

$ npx newman run docs/api/cc65-tf.postman_collection.json ... --env-var demo=true
43/44 aserciones en verde; FALLA «Cortar un nodo (solo con -demo)»:
  POST http://localhost:8080/api/v1/demo/nodos/nodo-2/cortar → 404 Not Found (esperaba 202; ver B3)
```

---

## 3. Contraste contra `AGENTS.md` §3 (Restricciones) y §4 (Git Flow)

| Regla | Estado | Evidencia (`archivo:línea` o comando) |
|---|---|---|
| **§3.1** Go en todo el backend y todo contenerizado con `docker compose up` | ⚠️ Parcial | `cmd/*` e `internal/*` son 100 % Go (incluido el servidor estático `cmd/web/main.go`); `deploy/docker-compose.yml` + `deploy/docker-compose.api.yml` levantan todo. **Brecha:** `README.md:70-72` y `deploy/README.md:25-28` no documentan la secuencia completa previa (`make data` → `cmd/loader` → `cmd/gold` → `cmd/entrenar` → `deploy/.env`), por lo que un `docker compose up` en limpio falla buscando `/datos/shard_0.bin` o `/datos/modelo.json` (**B1**). |
| **§3.2** Núcleo solo stdlib; terceros únicamente drivers de MongoDB y Redis | ✅ Cumple | `go list -f '{{.ImportPath}}: {{.Imports}}' ./...` confirma que solo `internal/almacen/mongo` (`go.mongodb.org/mongo-driver/v2`) e `internal/almacen/redis` (`github.com/redis/go-redis/v9`) importan paquetes externos. `ingest`, `ml`, `cluster`, `api`, `auth` (JWT + PBKDF2) y `ws` (RFC 6455) usan 100 % stdlib. |
| **§3.3** Determinismo y gradiente distribuido idéntico bit a bit al secuencial | ✅ Cumple | `internal/ml/logistica.go:198-204` (`ReducirEnOrden`), `internal/cluster/coordinador.go:367,588` (copia por `BloqueInicial`), exigido por `TestEquivalencia`, `TestShardsReproducenElGradienteGlobal`, `TestEquivalenciaClusterSecuencialBitABit` y `TestServicioDemoCortarNodoPorLaAPI`. |
| **§3.4** Sin condiciones de carrera (`go test -race`) | ⚠️ Con observación | 0 condiciones de carrera (`DATA RACE`) en los 25 paquetes. Sin embargo, 2 tests tienen plazos frágiles bajo `-race` en 2 vCPU (`cmd/benchapi/main_test.go:355` y `internal/ws/ws_test.go` con PR `#88` abierto; **B4**). |
| **§3.5** API de predicción `< 100 ms` (p95) con benchmark reproducible | ⚠️ Con observación | `cmd/benchapi` mide `p95 = 3,89 ms` fría / `3,09 ms` caliente en `reports/bench_api_jorge-pc_jwt.json` (y `19,70 ms` / `15,89 ms` en nuestra VM de 2 vCPU). **Brecha:** `cmd/evaluacion/main.go:33` y `reports/evaluacion.json` siguen citando `reports/bench_api_deez.json` (PC4 **sin JWT**; **B5**). |
| **§3.6** Ninguna cifra a mano en el informe (`reports/*.json`) | ⚠️ Pendiente TB2 | Se cumple en `docs/informe/pc3` y `pc4` vía `cmd/informe`, pero falta extender `cmd/informe` para generar `docs/informe/tb2/` desde `reports/evaluacion.json`, `reports/bench_api_*_jwt.json` y `reports/seguridad.json` (**B10**). |
| **§3.7** Datos no versionados (`data/` en `.gitignore` + `MANIFEST.sha256`) | ✅ Cumple | `.gitignore:2-4` ignora `data/*` salvo `data/MANIFEST.sha256` y `data/README.md`; ningún CSV ni `.bin` está versionado. |
| **§4** Git Flow, Kanban y commits propios de los **tres integrantes** | ❌ Incumple | `git shortlog -sne origin/integracion/tb2`: Jorge García (43 commits), José Villanueva (25 commits), **Marco Cabrera (0 commits en todo el repo y 0 en TB2; figura como `—` en `README.md:80`)**. Además, `#68` y `#69` ya fusionados siguen `OPEN` en *Backlog* del Kanban y faltan los merges `integracion/tb2 → develop → main` + tag `tb2` (**B12**). |

---

## 4. Contraste contra las 6 observaciones del profesor a la PC4

El entregable TB2 exige incluir los entregables previos **con sus correcciones** (`AGENTS.md` §6, issue `#71`). En `integracion/tb2` las 6 observaciones de la PC4 continúan abiertas porque `docs/informe/pc4/` no fue actualizado tras el cierre de la PC4 y `docs/informe/tb2/` aún no existe:

| # | Observación del profesor (PC4) | Estado actual en el repo (`archivo:línea`) | Qué falta exactamente para cerrarla en TB2 |
|---|---|---|---|
| **PC4-1** | **Dockerfile/Compose sin explicar** | `docs/informe/pc4/04-arquitectura.md:39-53` (§4.2) y `08-distribuido.md:389-395` (§8.6) solo resumen los comandos en prosa, sin mostrar ni explicar los archivos de despliegue. Además, `cmd/informe/regiones.go:81-101` solo extrae funciones/tipos Go, no archivos Docker/YAML. | Incluir fragmentos reales (extraídos automáticamente desde `deploy/Dockerfile`, `deploy/Dockerfile.web`, `deploy/docker-compose.yml` y `deploy/docker-compose.api.yml`) y explicar: (a) *multi-stage build* (2 etapas en backend, 3 en `web`), (b) `CGO_ENABLED=0` + `-trimpath -ldflags="-s -w"` para binarios estáticos `< 40 MB` sobre `distroless/static-debian12:nonroot` (`uid 65532`), (c) caché de capas (`COPY go.mod go.su[m] ./`) y `Dockerfile.web.dockerignore`, (d) aislamiento en red interna `cluster` (`:9000`, `:27017`, `:6379` no publicados) frente a puertos expuestos (`:8080`, `:8081` en orígenes distintos para CORS real), `healthcheck` + `service_healthy`, volúmenes `:ro` y escalado con `replicas: ${NODOS:-3}`. |
| **PC4-2** | **Faltan pruebas que sustenten las tablas** | En `05-datos.md`, `06-limpieza.md`, `07-modelo.md`, `08-distribuido.md:436-441` y `09-rendimiento.md` aparecen más de 12 tablas cuyos números vienen de `reports/*.json`, pero `12-evidencias.md:8-11` solo muestra una captura global de `go test -race ./...` (`ok paquete Xs`) sin el detalle `=== RUN / --- PASS` ni las salidas de consola de `cmd/perfil`, `cmd/loader`, `cmd/gold` y `cmd/benchmark` (loader y gradiente). | Agregar una tabla de trazabilidad **«Tabla del informe → comando / test Go → reporte JSON → evidencia en §Evidencias»** y respaldar las tablas con corridas explícitas (`go test -race -v -run ...`) de limpieza (`internal/ingest`), equivalencia y speedup (`internal/ml`), tolerancia a fallos (`internal/cluster`, los 4 tests de §8.8) y evaluación (`internal/evaluacion`). |
| **PC4-3** | **Contratos request/response y set de pruebas de la API** | `docs/informe/pc4/10-api.md:9-17` (§10.1) solo tiene una tabla resumen de 6 filas (faltan además las 6 rutas nuevas de TB2: `/api/v1/auth/login`, `/api/v1/riesgo`, `/api/v1/evaluacion`, `/api/v1/ws/ticket`, `/api/v1/ws`, `/api/v1/demo/nodos/{nombre}/cortar`). No muestra cuerpos JSON de ejemplo (entrada, salida 2xx y errores 400/401/403/409/413/429) ni un cuadro del set de pruebas. | (1) Documentar los contratos JSON `request`/`response` y cabeceras (`Authorization`, `X-Cache`, `ETag`, `Location`, `Retry-After`, `WWW-Authenticate`) de las **12 rutas** de `internal/api/api.go:318-333`. (2) Incluir la tabla estructurada del **set de pruebas de la API** (caso, ruta, rol/token, entrada, código esperado vs obtenido, test en `internal/api/*_test.go` / Newman / `scripts/evidencia-seguridad.sh` → `reports/seguridad.json`). (3) Corregir el fallo de Newman en `demo=true` (**B3**). |
| **PC4-4** | **Pruebas del cuadro de latencia** | `docs/informe/pc4/10-api.md:204-224` (§10.4) muestra el cuadro resumen (`bench_api_deez.json`, sin JWT), pero no muestra las pruebas que garantizan que la medición es válida ni la corrida autenticada de TB2. | (1) Actualizar la tabla y `reports/evaluacion.json` a una corrida con **JWT** sobre el mismo equipo y versión de modelo vigente (**B5**). (2) Sustentar el cuadro con: salida de `cmd/benchapi -correo ... -exigir-p95 100ms`, tests `TestSemillaRepetidaFallaLaCorridaFria`, `TestFallaSiLaCorridaCalienteNoSaleDeCache` y `TestMedirConLoginUsaJWT` (`cmd/benchapi/main_test.go`), evidencia de `X-Cache: MISS` (0 hits en las 2 000 frías) vs `X-Cache: HIT` (2 000 hits en las calientes) y ejecución completa en Newman/Postman con `< 100 ms`. |
| **PC4-5** | **Qué valida `spin/barrera.pml`** | `docs/informe/pc4/08-distribuido.md:397-428` (§8.7) enumera resultados en un párrafo denso y pega la tabla de conteo de estados, sin mostrar las fórmulas/`assert` de `spin/barrera.pml` ni delimitar qué queda dentro y fuera del modelo formal. | Añadir una tabla explícita **«Propiedad / Mutante en `spin/barrera.pml:línea` → Aserción o fórmula LTL → Equivalente en `coordinador.go:línea` → Qué error atrapa»**, seguida de un apartado **«Alcance del modelo formal (qué valida y qué NO valida)»**: Spin valida el plano de control de la barrera y reasignación (ausencia de *deadlock*, completitud de N shards, no envío a descartados, balanceo `|carga[k]-carga[j]| ≤ 1`, no descarte arbitrario de vivos y terminación `<> terminado`); **no** modela el handshake `asignacion`/`listo`, nodos de reserva tardíos, aritmética `float64` ni framing JSON (cubiertos por los tests de `internal/cluster` e `internal/ml`). |
| **PC4-6** | **Análisis de las capturas de BD** | `docs/informe/pc4/12-evidencias.md:59-67` (§12.4) contiene únicamente dos líneas introductorias y dos marcadores `<!-- captura: ... -->`, sin ningún análisis. Además, `scripts/evidencia-bd.sh:31` solo recorre `['modelos', 'entrenamientos', 'predicciones']` y **omite la colección `usuarios`** de TB2. | (1) Actualizar `scripts/evidencia-bd.sh` para verificar también `usuarios` (`_id` normalizado, `rol`, formato PBKDF2 sin exponer secretos) e imprimir un documento real por colección y un `TTL` + valor de Redis. (2) Debajo de cada captura en el informe, redactar el análisis cuantitativo: vínculo `entrenamientos._id` ↔ `version_modelo` ↔ `modelos._id` (`$push` en `entrenamientos[]`), comparación de `microsegundos` entre `desde_cache: false` y `desde_cache: true`, por qué `predicciones` puede tener menos documentos que peticiones bajo ráfaga (`BufferRegistros = 1024`, descarte no bloqueante) y por qué tras una caída con reconexión `entrenamientos.nodos` registra N+1 entradas. |

---

## 5. Brechas detalladas por criterio de la rúbrica TB2

### Criterio 1 — Planteamiento (3 puntos)

#### B1 — Documentación de puesta en marcha incompleta y bloqueo WAF (`HTTP 418`) al descargar el CSV · MAYOR
- **Dónde:** `README.md:69-72`, `deploy/README.md:24-39`, `deploy/README.md:90-93`, `cmd/gold/main.go:189-212`, `scripts/descargar-datos.sh:8,17`
- **Evidencia:**
  1. `README.md:69-72` solo documenta `make check` y `make data`. `deploy/README.md:25` empieza en `go run ./cmd/gold -shards 3` omitiendo `go run ./cmd/loader` (que genera `data/silver/dengue.csv`) y omitiendo `go run ./cmd/entrenar` (que es el único comando que escribe `data/gold/modelo.json`, exigido por `coordinator` en modo servicio en `deploy/docker-compose.yml:50`).
  2. El modo sintético documentado en `deploy/README.md:37` (`go run ./cmd/gold -sintetico 300000 -shards 3 -salida /tmp/gold`) escribe `train.bin`, `test.bin` y `shard_0..2.bin`, pero **no escribe `esquema.json` ni `modelo.json`** (`cmd/gold/main.go:188-208`). Al levantar el modo servicio con `GOLD=/tmp/gold`, `coordinator` aborta al arrancar con `modelo inicial: open /datos/modelo.json: no such file or directory`.
  3. Al ejecutar `make data` desde esta VM en la nube, `datosabiertos.gob.pe` responde **`HTTP/1.1 418` (`Server: CW`, cookie `HWWAFSESID`, bloqueo WAF a IPs de datacenter / `curl` sin navegador)**:
     ```text
     $ curl -sSI "https://www.datosabiertos.gob.pe/sites/default/files/datos_abiertos_vigilancia_dengue_2000_2025_0.csv"
     HTTP/1.1 418 · Block-Event-Id: 08-nil · Server: CW
     ```
- **Corrección sugerida:**
  - Documentar en `README.md` y `deploy/README.md` la cadena exacta de comandos desde cero (`make data` → `cmd/loader` → `cmd/gold` → `cmd/entrenar` → `cp deploy/.env.example deploy/.env` → `docker compose ... up --build`), y advertir qué hacer si el portal del MINSA responde `418` fuera de una red doméstica peruana (descarga manual desde navegador en `data/bronze/vigilancia_dengue_2000_2025.csv` y verificación con `make data`).
  - Hacer que `cmd/gold -sintetico` (o un flag auxiliar) escriba también un `esquema.json` y un `modelo.json` sintéticos compatibles con las 31 variables para que el evaluador pueda levantar el **modo servicio completo + SPA** con un solo comando sin el CSV de 118 MB.

---

### Criterio 2 — Implementación (8 puntos)

#### B2 — `Version` del modelo incluye la marca temporal `Generado` en el SHA-256: dos entrenamientos con pesos idénticos producen versiones distintas · MAYOR
- **Dónde:** `internal/api/api.go:549`, `internal/api/modelo.go:33,65-76,104-107`, `cmd/entrenar/main.go:155-156`, `internal/almacen/almacen.go:53-55`, `web/src/paginas/Inicio.tsx:69,80-84`
- **Evidencia:**
  1. `internal/almacen/almacen.go:53-55` (y `docs/informe/pc4/11-bd.md:84-85`) afirman: *«Dos entrenamientos con los mismos pesos comparten versión y ambos quedan aquí (PR-40 ronda 2, H11)»*.
  2. Sin embargo, `(*Servidor).entrenar` (`internal/api/api.go:549`) llama `base.conPesos(w, time.Now().Format(time.RFC3339))`, y `conPesos` (`internal/api/modelo.go:66-74`) incluye `Generado: generado` en el struct **antes** de `b, err := json.Marshal(n); n.Version = version(b)`. Lo mismo ocurre en `cmd/entrenar/main.go:155-156`.
  3. Consecuencias reales verificadas:
     - En `reports/` coexisten **tres versiones distintas** para los mismos pesos del gold real: `e69395b36d2f` (`bench_api_deez.json`), `655c9dc770fa` (`evaluacion.json`) y `5cb7a4120775` (`bench_api_jorge-pc_jwt.json`).
     - En la demostración en vivo, apenas el admin lanza un entrenamiento de 600 épocas con los hiperparámetros por defecto (que converge bit a bit a los mismos pesos), `version_modelo` cambia por `time.Now()`, invalida toda la caché de Redis y hace aparecer en `Inicio` el banner de alerta: *«Esta evaluación es del modelo 655c9dc770fa; la API publica el <nuevo>. Regenera reports/evaluacion.json»*.
- **Corrección sugerida:** Calcular `version` sobre una representación canónica **sin `Generado`** (pesos + esquema + umbral + `ajuste_prior`), conservando `Generado` solo como metadato informativo. Así dos corridas deterministas con los mismos pesos producen exactamente el mismo `_id` en MongoDB, ejercitan de verdad el `$push` de `GuardarModelo` y no desfasan `reports/evaluacion.json` al reentrenar con los parámetros por defecto en la sustentación.

#### B3 — La colección de Postman falla en `demo=true` contra Docker Compose (`404` al cortar `nodo-2`) · MAYOR
- **Dónde:** `docs/api/cc65-tf.postman_collection.json:38-40,340-355,599-632`, `deploy/docker-compose.yml:58-70`, `cmd/node/main.go:34-36`
- **Evidencia:**
  - En Docker Compose, `node` se escala con `replicas: ${NODOS:-3}` sin `-nombre`, por lo que `cmd/node` registra `os.Hostname()` (el ID corto del contenedor, p. ej. `919f98fdfe3c`), nunca `"nodo-2"`.
  - La colección de Postman define `"nodo": "nodo-2"` (`línea 39`) y el request «Estado del clúster» (`líneas 340-355`) no actualiza `nodo` con `nodos[0].nombre`. Al correr Newman contra el Compose levantado con `DEMO=true`:
    ```text
    → Cortar un nodo (solo con -demo)
      POST http://localhost:8080/api/v1/demo/nodos/nodo-2/cortar [404 Not Found, 288B, 2ms]
      1. AssertionError: expected response to have status code 202 but got 404
    ```
  - Además, «Lanzar entrenamiento distribuido» (`línea 410`) deja corriendo un entrenamiento de 600 épocas en segundo plano justo antes de «Cortar un nodo», lo que es válido, pero si quedan ≤ 1 nodos sanos tras corridas repetidas sin reiniciar contenedores el entrenamiento posterior fallará.
- **Corrección sugerida:** En el script `test` de «Estado del clúster» (`cc65-tf.postman_collection.json:346`), guardar el primer nodo sano conectado: `const sano = pm.response.json().nodos.find(n => n.sano); if (sano) pm.collectionVariables.set("nodo", sano.nombre);`.

#### B4 — Dos pruebas unitarias sensibles a planificación de CPU bajo `-race` en entornos de 2 vCPU · MAYOR
- **Dónde:** `cmd/benchapi/main_test.go:346-359`, `internal/ws/ws_test.go` (PR `#88` abierto)
- **Evidencia:**
  1. En `cmd/benchapi/main_test.go:355`, `TestMedirConLoginUsaJWT` fija `timeout: 5 * time.Second` en el cliente HTTP. Dentro del mismo test se ejecutan múltiples derivaciones PBKDF2-HMAC-SHA256 de **600 000 iteraciones** instrumentadas con `-race` (`auth.Hash`, `HashSenuelo` en segundo plano y `auth.Comparar` en `POST /api/v1/auth/login`). Incluso aislado en 2 vCPU tarda `7,32 s`; cuando coincide con la compilación/ejecución paralela de `cmd/benchmark` e `internal/api`, el `POST /api/v1/auth/login` superó los 5 s y falló con `Client.Timeout exceeded while awaiting headers (15.70s)`.
  2. El PR `#88` (`fix/ws-test-intermitente`, que sube el ping de `TestHubConcurrenteSinFugas` de 5 ms a 25 ms tras fallos intermitentes en el CI de GitHub Actions de 2 vCPU) sigue `OPEN` sin revisar ni fusionar a `integracion/tb2`.
- **Corrección sugerida:** Subir `timeout` en `TestMedirConLoginUsaJWT` (`cmd/benchapi/main_test.go:355`) a `20 * time.Second` (solo acota el plazo de red del test bajo `-race`, no el requisito de 100 ms en producción) y revisar/fusionar el PR `#88` en `integracion/tb2`.

#### B5 — `reports/evaluacion.json` cita el benchmark de latencia de PC4 sin JWT y falta `reports/seguridad.json` · MAYOR
- **Dónde:** `cmd/evaluacion/main.go:33`, `reports/evaluacion.json:1`, `reports/bench_api_deez.json`, `reports/bench_api_jorge-pc_jwt.json`, `cmd/informe/regiones.go:307-323`
- **Evidencia:**
  - `cmd/evaluacion/main.go:33` tiene por defecto `-api reports/bench_api_deez.json`, que fue medido el `2026-11-21` **sin autenticación JWT** (en PC4). Como consecuencia, la vista `/evaluacion` de la SPA (`Evaluacion.tsx:308-355`) muestra hoy en pantalla las cifras sin JWT (`2,21 ms` / `2,06 ms` de `bench_api_deez.json`) en lugar de la latencia con verificación Bearer JWT exigida en TB2 (`ADR-0007` §Consecuencias).
  - Asimismo, el script `scripts/evidencia-seguridad.sh` y su salida `reports/seguridad.json` comprometidos en `ADR-0007:85-86` e issue `#70` aún no existen en el repositorio.
- **Corrección sugerida:** Correr `cmd/benchapi` con JWT sobre la máquina de referencia del informe con el modelo vigente, pasar ese `bench_api_*_jwt.json` a `cmd/evaluacion` y `cmd/informe` (o mostrar en el informe TB2 la comparación sin JWT vs con JWT) e implementar `scripts/evidencia-seguridad.sh` → `reports/seguridad.json` (`#70`).

#### B6 — Cobertura directa de `cmd/loader` y `cmd/perfil`, y métrica `RegistrosDescartados` invisible en operación · MENOR
- **Dónde:** `cmd/loader/main.go`, `cmd/perfil/main.go`, `internal/api/api.go:229,381-383`
- **Evidencia:** `cmd/loader` y `cmd/perfil` son los únicos binarios ejecutables de `cmd/` sin `*_test.go` (aunque su lógica en `internal/ingest` sí está cubierta). Por otro lado, durante nuestra corrida de `cmd/benchapi` (4 048 predicciones a ~1 800 RPS), el búfer asíncrono `BufferRegistros = 1024` descartó ~2 102 escrituras a MongoDB como está diseñado (`ADR-0006`), pero `(*Servidor).RegistrosDescartados()` nunca se incluye en `/healthz` ni en `/api/v1/cluster` ni en logs, dificultando auditar el descarte en caliente.
- **Corrección sugerida:** Agregar un test de humo CLI en `cmd/loader` y `cmd/perfil` sobre `internal/ingest/testdata/muestra.csv`, e incluir `predicciones_descartadas` (o log al cerrar) si se desea trazabilidad directa en las capturas de BD.

---

### Criterio 3 — Interfaz (5 puntos)

#### B7 — «Salir» recuerda la ruta anterior en React Router v7 e incumple el contrato de `App.tsx:25` · MAYOR
- **Dónde:** `web/src/App.tsx:8-14,25-29`, `web/src/pruebas/app.test.tsx:182-192`
- **Evidencia:**
  - En `web/src/App.tsx:25-29`:
    ```tsx
    // Salir a propósito no recuerda la vista: el próximo usuario empieza en el inicio.
    const cerrarSesion = () => {
      navegar('/login', { replace: true })
      salir()
    }
    ```
  - En `react-router-dom` v7 (`^7.18.6`), `navegar(...)` envuelve la actualización de ruta en una transición de baja prioridad (`startTransition`), mientras `salir()` ejecuta `setSesion(null)` como actualización urgente síncrona. En ese render intermedio `sesion` ya es `null` pero `ubicacion.pathname` sigue siendo la vista previa (`/admin`, `/mapa`, `/prediccion`…), por lo que `ConSesion` (`App.tsx:12`) renderiza `<Navigate to="/login" replace state={{ desde: ubicacion.pathname + ubicacion.search }} />` y **pisa** la navegación sin estado.
  - **Reproducido en vivo con Playwright:** al pulsar «Salir» estando en `/admin` y entrar luego como `admin@vigia.pe`, el navegador aterrizó en `http://localhost:8081/admin` en vez de `http://localhost:8081/`. El test de `app.test.tsx:182-192` no lo detectó porque entraba como `analista` tras salir de `/admin`, caso en el que `destinoTrasLogin('/admin', 'analista')` devuelve `'/'` por falta de rol y no porque `desde` estuviera limpio.
- **Corrección sugerida:** Marcar mediante un ref (p. ej. `saliendoManual.current = true`) en `cerrarSesion` para que `ConSesion` navegue a `/login` **sin `state`** cuando el cierre fue voluntario (y sí conserve `state={{ desde }}` cuando `sesion` cae por 401/expiración o entrada directa sin sesión), agregando un test en `app.test.tsx` donde un usuario salga desde `/prediccion` (o `admin` desde `/admin`) y el siguiente login aterrice en `'/'`.

#### B8 — Faltan las pruebas E2E automatizadas (Playwright) y las capturas reproducibles del front (`#70`) · MAYOR
- **Dónde:** Issue `#70` abierto; `web/package.json` (sin script E2E); ausencia de `docs/informe/fig/tb2/`
- **Evidencia:** Las pruebas de componentes en Vitest + jsdom (`170/170`) son muy completas, pero aún no existe la suite Playwright contra el stack levantado ni la generación automática de capturas con nombre fijo en `docs/informe/fig/tb2/` que exige el plan del equipo (`#70`) para las evidencias de funcionalidad y seguridad del informe final.
- **Corrección sugerida:** Incorporar la suite Playwright de `#70` cubriendo los flujos de ambos roles (login inválido/429, analista en las 4 vistas + bloqueo 403 en `/admin`, admin con entrenamiento en vivo por WebSocket + corte de nodo + sello `bit a bit ✓`) y guardando las capturas de `docs/informe/fig/tb2/`.

#### B9 — Avisos `act(...)` en `admin.test.tsx`, bundle inicial `> 500 kB` y convención decimal distinta al informe · MENOR
- **Dónde:** `web/src/pruebas/admin.test.tsx` (`un ticket vacío no abre el WebSocket`), `web/vite.config.ts`, `web/src/mapa/datos.ts:115-120`
- **Evidencia:**
  1. `npx vitest run` imprime 4 advertencias `An update to Admin inside a test was not wrapped in act(...)` en `admin.test.tsx` porque `cargarModelo()` y `GET /api/v1/cluster` resuelven promesas asíncronas después de terminar el test de ticket vacío.
  2. `vite build` advierte chunk único de `527,48 kB` (`178,22 kB` gzip) porque `peru_departamental_simple.geojson` y todas las rutas se importan de forma estática en `rutas.tsx:3-7`.
  3. `datos.ts:115-120` usa `Intl.NumberFormat('es-PE')` (punto decimal y coma de miles: `21.2%`, `32,078`), mientras `cmd/informe` formatea con coma decimal y espacio fino (`20,65 %`, `32 078`). Es válido en el Perú (estándar oficial BCRP/SUNAT vs estilo académico), pero conviene tenerlo claro o unificarlo.
- **Corrección sugerida:** Esperar la carga inicial en el test de `admin.test.tsx` antes de finalizar la aserción; opcionalmente usar `React.lazy` por ruta en `rutas.tsx` para partir el GeoJSON fuera del login inicial.

---

### Criterio 4 — Informe (2 puntos)

#### B10 — No existe el informe final TB2 (`docs/informe/tb2/`) ni su generación en `cmd/informe` y `cmd/word` · BLOQUEANTE
- **Dónde:** `docs/informe/` (solo existen `pc3/` y `pc4/`), `cmd/informe/main.go:473-539`, `cmd/word/main.go:38-40`, `Makefile:57-60` (issue `#71` abierto)
- **Evidencia:**
  - `ls docs/informe/` muestra únicamente `pc3` y `pc4`.
  - `cmd/informe` no lee `reports/evaluacion.json`, `reports/bench_api_*_jwt.json`, `reports/bench_observador_*.json` ni `reports/seguridad.json`, ni genera secciones de TB2.
  - `Makefile:58-59` invoca `go run ./cmd/word` sin argumentos, cuyos valores por defecto en `cmd/word/main.go:38-40` siguen siendo `-fuentes docs/informe/pc3` y `-entregable PC3`.
- **Corrección sugerida:** Construir `docs/informe/tb2/` con la estructura completa `a)–j)` del enunciado TB2 (integrando PC3 + PC4 corregidos + arquitectura y UX de la SPA, seguridad JWT/PBKDF2/CORS/CSP/WebSocket con código extraído del fuente, evaluación experimental completa desde `reports/evaluacion.json`, respuesta operativa a la pregunta de impacto social, evidencias E2E y de seguridad), actualizar `cmd/informe`, `cmd/word` y `Makefile` para apuntar a `tb2` y verificar idempotencia (`go run ./cmd/informe && git diff --exit-code docs/informe`).

#### B11 — Las 6 observaciones del profesor a la PC4 siguen sin incorporarse en el informe · BLOQUEANTE
- **Dónde:** `docs/informe/pc4/{04-arquitectura,08-distribuido,10-api,11-bd,12-evidencias,16-anexos}.md`, `cmd/informe/regiones.go:81-101`, `scripts/evidencia-bd.sh:31`
- **Evidencia:** Ver el desglose completo en la **Sección 4** de esta auditoría (PC4-1 a PC4-6). Si el informe final TB2 hereda las secciones de PC4 sin atender estas 6 observaciones explícitas del docente y sin una tabla «Observación PC4 → Dónde se corrigió» en el anexo (análoga al anexo 16.1 de PC3), perderá puntaje directo en Informe e Implementación.
- **Corrección sugerida:** Atender las 6 observaciones según la Sección 4, ampliar `cmd/informe/regiones.go` para admitir inclusión verificada de archivos completos o bloques de `deploy/*` y `spin/barrera.pml` (p. ej. `<!-- generado:archivo:deploy/Dockerfile -->`), ampliar `scripts/evidencia-bd.sh` con `usuarios` y documentar la tabla de correcciones a la PC4 en el anexo del informe TB2.

#### B12 — Tercer integrante con 0 commits en Git, Kanban desactualizado y ramas pendientes de cierre · BLOQUEANTE
- **Dónde:** Historial Git (`origin/integracion/tb2`), `README.md:80`, tablero Kanban de GitHub (issues `#68`–`#73`), PR `#88` y PRs Dependabot `#1`, `#2`, `#48`
- **Evidencia:**
  1. `git shortlog -sne origin/integracion/tb2`:
     ```text
     41  Jorge Rolando García Roca <jg1536454@gmail.com>
     25  José Javier Villanueva Aramburú <152805563+JJva-06@users.noreply.github.com>
      2  JorgeGarcia005 <211437968+JorgeGarcia005@users.noreply.github.com>
      1  Claude (agente implementador) <noreply@anthropic.com>
     ```
     **Marco Antonio Luciano Cabrera Díaz tiene 0 commits** en todo el repositorio y 0 en `origin/develop..origin/integracion/tb2`, y en `README.md:80` aparece `—` en la columna GitHub, incumpliendo `AGENTS.md` §4 (*«Los tres integrantes deben tener commits propios en cada entregable»*) y el criterio de trabajo colaborativo / Git Flow de la rúbrica.
  2. En el tablero Kanban (`TF CC65`), los issues `#68` y `#69` continúan `OPEN` y en *Backlog* pese a que sus PRs `#86` y `#87` ya fueron aprobados y fusionados en `integracion/tb2`.
  3. Siguen abiertas ramas antiguas (`origin/feature/33-informe-pc4-jose`, `origin/evidencia/27-gold-real`, `origin/feature/7-rl-secuencial`, `origin/chore/13-juez-windows`, `origin/debate/protocolo-tcp`) y el release final `integracion/tb2 → develop → main` con tag `tb2` está pendiente (`#73`).
- **Corrección sugerida:** Asegurar que Marco realice y firme con su cuenta de GitHub commits reales y defendibles en los entregables pendientes asignados a él (`#71` informe TB2 / `#72` presentación y enlace de video) y enlazar su usuario en `README.md:80` y `.mailmap`; cerrar y mover `#68` y `#69` a *Hecho* en el Kanban; limpiar ramas obsoletas y ejecutar el flujo de release `#73` al cerrar `#70`–`#72`.

---

### Criterio 5 — Video (2 puntos)

#### B13 — Video demostrativo `≤ 6:00` con los tres integrantes pendiente (`#72`) · BLOQUEANTE
- **Dónde:** Issue `#72` abierto; sección de anexos del informe TB2
- **Evidencia:** El video aún no se ha grabado ni enlazado. La infraestructura de demostración (`DEMO=true`, pausa entre épocas `ritmo_ms`, botón «Tumbar», WebSocket en vivo, sello `bit a bit ✓` y réplica `.jsonl` de respaldo) funcionó al 100 % en nuestra prueba en vivo.
- **Corrección sugerida:** Grabar el video cuidando estrictamente el tope de **6:00 minutos**, con participación visible y equilibrada de los **tres integrantes**, recorriendo extremo a extremo: (1) problema y pregunta ODS 3, (2) arquitectura y Docker Compose, (3) login + rol analista (`Inicio`, `Mapa`, `Predicción` MISS→HIT en `< 100 ms`, `Evaluación` y bloqueo `403` en `/admin`), (4) rol admin (`/admin` en vivo por WebSocket, lanzamiento con `ritmo_ms`, «Tumbar» un nodo mostrando reasignación y sello `bit a bit ✓`), y (5) respuesta cuantitativa a la pregunta de impacto social.

---

## 6. Riesgos para la sustentación

1. **R1 — ¿Por qué tras «Tumbar» un nodo en vivo aparece como conectado otra vez en 1 segundo y MongoDB reporta `4 nodos` si `NODOS=3`?**
   - *Qué ocurre en el código:* `CortarNodo` (`internal/cluster/coordinador.go:250-279`) cierra el socket TCP; `cmd/node` tiene reconexión automática (`-reconexiones 3`, pausa `1s` en `cmd/node/main.go:40,55-71`) y vuelve a enviar `registro`. El coordinador agrega una nueva entrada a `c.nodos` con carga `0` (nodo de reserva, `coordinador.go:544-555`), dejando 4 entradas en `c.nodos` (la conexión caída `sano: false` y la reconectada `sano: true`). En cambio, la SPA agrupa por `nombre` (`web/src/admin/estado.ts`), mostrando `0 shards · 2 workers · cayó 1×` con punto verde.
   - *Riesgo ante el profesor:* Si un integrante dice «el contenedor murió» o no sabe explicar por qué `entrenamientos.nodos` tiene longitud 4 en MongoDB mientras la UI muestra 3 nodos, parecerá una contradicción. Deben explicar con claridad la diferencia entre `docker kill` (`scripts/prueba-caida.sh`, muerte del proceso) y `CortarNodo` (`DEMO=true`, corte de conexión TCP donde el proceso reconecta como reserva sin recuperar su shard salvo nueva caída).

2. **R2 — ¿Por qué tras correr `benchapi` hay ~2 029 claves en Redis pero menos documentos (~1 946) en `db.predicciones` de MongoDB?**
   - *Qué ocurre en el código:* `(*Servidor).registrar` (`internal/api/api.go:288-294`) usa un canal con búfer `BufferRegistros = 1024` y `select` no bloqueante (`default: s.descartados.Add(1)`). A ~1 800–12 000 peticiones/s de `benchapi`, MongoDB no drena a la misma velocidad que la memoria/caché y el exceso se descarta deliberadamente (`ADR-0006`) para no acoplar la latencia `p95` de predicción al disco de Mongo.
   - *Riesgo ante el profesor:* Si en la captura de BD aparece `predicciones: 1946` después de lanzar `4 000` peticiones en `benchapi` y nadie lo explica, el profesor puede objetar pérdida de datos no controlada.

3. **R3 — Banner amarillo de desfase en `Inicio` después de entrenar en vivo en la sustentación (`B2`):**
   - Si en plena demostración el equipo entrena en `/admin` y luego vuelve a `Inicio` para mostrar las prioridades, aparecerá el cartel amarillo *«Esta evaluación es del modelo 655c9dc770fa; la API publica el <nuevo>. Regenera reports/evaluacion.json»*, incluso si entrenaron 600 épocas con `tasa=2` obteniendo exactamente los mismos pesos. Corregir **B2** elimina ese tropiezo cuando se entrena con los parámetros por defecto; si se entrena con menos épocas (p. ej. 40 para la demo rápida), el equipo debe saber explicar que `evaluacion.json` es un artefacto offline sobre las 190 819 filas de `test.bin` (`cmd/evaluacion`), mientras `/prediccion` sí usa en caliente el modelo recién publicado en el `atomic.Pointer`.

4. **R4 — Pregunta trampa sobre alcance de Spin (`spin/barrera.pml`):**
   - El profesor observó en PC4 *«qué valida `spin/barrera.pml`»*. Si pregunta *«¿Spin demuestra que el gradiente distribuido es idéntico bit a bit al secuencial?»*, la respuesta correcta es **NO**: Spin valida propiedades de seguridad y vivacidad del protocolo de coordinación (barrera de N shards, no bloqueo con canal lleno, plazo por shard frente a plazo global, dos pasadas y balanceo por menor carga); la identidad numérica IEEE-754 bit a bit la garantizan los límites fijos de bloque + `ReducirEnOrden` y la verifican los tests de Go (`TestEquivalenciaClusterSecuencialBitABit` y `cmd/entrenar -comparar`).

5. **R5 — Defensa estadística y epidemiológica (`AUC = 0,6221`, probabilidad ponderada vs riesgo calibrado y paradoja semanas 15–20 vs 45–49):**
   - Todo el equipo debe poder explicar sin dudar:
     1. Por qué `probabilidad_ponderada` ($\sigma(z) \approx 0{,}65$) decide la clase frente al umbral `0,5` (modelo entrenado con pesos balanceados $+3{,}985 / -0{,}572$ para ganar sensibilidad `Recall = 61,9 %` ante prevalencia de `12,58 %`), mientras `riesgo` ($\approx 21{,}2\,\%$) resta $\ln(s_+/s_-)$ al logit (King y Zeng, 2001) para devolver una probabilidad bien calibrada (`Brier = 0,108`, 8/10 deciles dentro del IC 95 % de Wilson).
     2. Por qué **camas** se priorizan por **casos esperados** ($\sum \hat{p}_i$, pico en semanas 15–16 y 18–20 en Piura, Loreto, Lima y La Libertad) mientras la **proporción individual** $\hat{p}_i$ sube en semanas 45–49 (sesgo de notificación fuera de brote + estacionalidad amazónica).

6. **R6 — Defensa individual de los tres integrantes (especialmente del núcleo concurrente y de seguridad en stdlib):**
   - Al haberse generado el código con apoyo intensivo de IA y tener Marco 0 commits hasta hoy, en la sustentación de semana 15 el profesor preguntará de forma dirigida sobre piezas no triviales hechas desde cero sin librerías: (a) handshake `Hijack`, limpieza de `SetDeadline`, enmascaramiento XOR y ping/pong en `internal/ws/ws.go`, (b) ticket atómico de un solo uso y chequeo de `Origin` previo al consumo en `internal/api/tiemporeal.go`, (c) `ConstantTimeCompare` y `HashSenuelo` en `internal/auth/clave.go` + `internal/api/login.go`, y (d) relaciones *happens-before* sin mutex en los slots de `internal/ml/concurrente.go`.

7. **R7 — Acceso por IP o red del campus en el día de la sustentación:**
   - `cmd/web` fija la CSP `connect-src 'self' <API_URL> <WS_URL>` al arrancar (`cmd/web/main.go:98-111`) y el coordinador exige coincidencia exacta en `CORS_ORIGENES` (`internal/api/seguridad.go:121`). Si abren la SPA usando `http://127.0.0.1:8081` o `http://<IP-LAN>:8081` dejando la configuración por defecto (`http://localhost:8081` y `http://localhost:8080`), **tanto CORS como la CSP y el handshake WebSocket bloquearán todas las llamadas**. Llevar siempre la pareja sincronizada (`CORS_ORIGENES` y `API_URL`) y un `.jsonl` grabado para el modo «Réplica sin clúster» como plan de contingencia.

---

## 7. Lista priorizada de acciones (ruta crítica al dom 29/11/2026 23:30)

1. **[P0 · BLOQUEANTE · #71] Crear `docs/informe/tb2/` y actualizar `cmd/informe`, `cmd/word` y `Makefile` (B10, B11):**
   - Incorporar las secciones nuevas de TB2 (SPA, seguridad ADR-0007 con código Go extraído del fuente, evaluación experimental desde `reports/evaluacion.json` y respuesta final a la pregunta) + las **6 correcciones a las observaciones de la PC4** (explicación línea a línea de `Dockerfile`, `Dockerfile.web` y `docker-compose*.yml`; trazabilidad de pruebas por tabla; contratos JSON request/response y tabla de pruebas de las 12 rutas; pruebas del cuadro de latencia con JWT; alcance exacto y fórmulas de `spin/barrera.pml`; análisis cuantitativo de las capturas de MongoDB + Redis incluyendo `usuarios`) + tabla de correcciones PC4 en anexos.
   - *Verificar con:* `go run ./cmd/informe && git diff --exit-code docs/informe && go test -race ./cmd/informe ./cmd/word`.

2. **[P0 · BLOQUEANTE · #4 AGENTS.md / #71 / #72] Participación real de los tres integrantes en Git (B12):**
   - Que Marco configure su cuenta GitHub, complete su enlace en `README.md:80` y realice sus propios commits y PRs en las tareas bajo su responsabilidad (`#71` / `#72`).
   - *Verificar con:* `git shortlog -sne origin/develop..origin/integracion/tb2` mostrando commits propios de Jorge, José y Marco.

3. **[P0 · BLOQUEANTE · #70] Pruebas E2E con Playwright, capturas reproducibles y `scripts/evidencia-seguridad.sh` → `reports/seguridad.json` (B5, B8):**
   - Automatizar los flujos de `analista` y `admin` con capturas en `docs/informe/fig/tb2/`, actualizar `scripts/evidencia-bd.sh` para incluir la colección `usuarios`, y generar `reports/seguridad.json` (CORS permitido/denegado, 401 sin token/vencido, 403 por rol, WebSocket sin ticket/origen ajeno, 413 cuerpo grande, 429 rate limit).
   - *Verificar con:* `npx playwright test` y `bash scripts/evidencia-seguridad.sh`.

4. **[P0 · BLOQUEANTE · #72] Grabar y verificar el video `≤ 6:00` con los tres integrantes (B13):**
   - Cubrir problema, arquitectura, flujo de ambos roles en vivo (`DEMO=true`, corte de nodo y sello `bit a bit ✓`), evaluación y conclusiones; enlazarlo en el anexo del informe.

5. **[P1 · MAYOR · Código corto y seguro] Corregir los 4 defectos puntuales hallados en la auditoría antes de congelar código:**
   - **B2 (`internal/api/modelo.go`):** Excluir `Generado` del cálculo de `version(b)` para que pesos idénticos conserven la misma `version_modelo` y regenerar `modelo.json` + `bench_api_*_jwt.json` + `reports/evaluacion.json` con la misma versión y con JWT (**B5**).
   - **B7 (`web/src/App.tsx`):** Evitar que `ConSesion` pise `cerrarSesion()` con `state={{ desde }}` en React Router v7 + test en `app.test.tsx`.
   - **B3 (`docs/api/cc65-tf.postman_collection.json`):** Guardar `sano.nombre` en la variable `nodo` al consultar `/api/v1/cluster` para que Newman pase 44/44 en `demo=true`.
   - **B4 (`cmd/benchapi/main_test.go` + PR `#88`):** Subir el `timeout` de `TestMedirConLoginUsaJWT` bajo `-race` y fusionar el PR `#88` (`fix/ws-test-intermitente`).

6. **[P1 · MAYOR · #1 Planteamiento / Despliegue] Completar `README.md`, `deploy/README.md` y modo sintético completo (B1):**
   - Documentar la cadena completa de comandos desde cero, la contingencia ante `HTTP 418` del portal MINSA y generar `modelo.json` en `cmd/gold -sintetico` para poder levantar el modo servicio sin el CSV real.

7. **[P2 · CIERRE · #73] Higiene de Kanban, ramas y release final `tb2` (B12, B6, B9):**
   - Cerrar y mover `#68` y `#69` (y luego `#70`–`#72`) a *Hecho* en el Kanban, limpiar ramas sueltas/Dependabot, abrir y aprobar los PRs `integracion/tb2 → develop` y `develop → main`, etiquetar `tb2` y ensayar las 7 preguntas de sustentación de la Sección 6.
