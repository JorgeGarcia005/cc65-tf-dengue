# Veredicto — PR-79 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/62-spa-base` @ `cf39c16` (base: `integracion/tb2` @ `b405959`)
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H5**) quedaron resueltos y verificados tanto en tests unitarios/integrados como en prueba de caja negra sobre el binario compilado de `cmd/web`. La base `integracion/tb2` @ `b405959` quedó integrada (`cf39c16`) y los **7/7 jobs del CI** en GitHub pasan en verde (run `44974692377`), incluido `SPA (tipos, tests, build e imagen)` que construye la imagen `cc65-tf-web` sin `go mod download`, verifica `User=nonroot:nonroot` y valida `/config.json`, la CSP y `/mapa` por `curl`. `cmd/web` sigue 100 % en biblioteca estándar (`AGENTS.md` §3.2), endurece `API_URL` contra inyección CSP, usa `os.OpenRoot` + `http.ServeContent` (sin `301` en `/index.html` ni `immutable` en `404`), drena `Shutdown` en `SIGTERM`, y la SPA (`web/`) pasa `tsc -b`, 27/27 tests en Vitest y `vite build` cerrando la sesión zombi, el `TypeError` en 200 no-JSON, la carrera de `401` tardío y la redirección segura tras el login.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ go list -f '{{.Imports}}' ./cmd/web
[context encoding/json errors flag fmt log net/http net/url os os/signal path strconv strings syscall time]

$ go test -race -count=1 -v -cover ./cmd/web
=== RUN   TestCabecerasYCSP          --- PASS (0.03s)
=== RUN   TestConfigJSON             --- PASS (0.00s)
=== RUN   TestRutasDelSPA            --- PASS (0.00s)
=== RUN   TestConfiguracionInvalida  --- PASS (0.00s)
=== RUN   TestAPIURLValidas          --- PASS (0.00s)
PASS
coverage: 69.9% of statements
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.061s	coverage: 69.9% of statements

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	3.173s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	7.630s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	3.421s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.039s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.517s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.274s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.457s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.116s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.200s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.044s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.080s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	6.066s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.660s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.438s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.041s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.136s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.658s

$ (cd web && npm ci --no-audit --no-fund && npm run typecheck && npm test && npm run build)
✓ src/pruebas/app.test.tsx (22 tests)
✓ src/pruebas/cliente.test.ts (5 tests)
Test Files  2 passed (2) · Tests  27 passed (27)
✓ built in 818ms (dist/index.html, assets/index-ChGXP6MQ.css, assets/index-relcYGPg.js, fuentes autoalojadas)

$ <prueba de caja negra sobre el binario compilado de cmd/web sirviendo web/dist>
REJECTED OK: http://a;b:8080 | http://:8080 | http://user:pass@localhost:8080?x=1#frag | http://localhost:8080?
GET /config.json          -> 200 OK · Cache-Control: no-store · {"api_url":"http://localhost:8080"}
GET /assets/no-existe.js  -> 404 Not Found · (sin Cache-Control: immutable)
GET /index.html           -> 200 OK · Cache-Control: no-cache (sin 301)
GET /mapa                 -> 200 OK · Cache-Control: no-cache · CSP connect-src 'self' http://localhost:8080 ws://localhost:8080
SIGTERM                   -> EXIT_CODE=0

$ gh pr checks 79   # run 44974692377 sobre cf39c16
Docker (imagen, clúster y caída de un nodo)	pass	2m21s
Ecosistema de agentes                       	pass	10s
Go (gofmt, vet, test -race)                 	pass	55s
MongoDB y Redis (tests de integración)      	pass	47s
OpenAPI 3 válido                            	pass	23s
SPA (tipos, tests, build e imagen)          	pass	49s   (job 137061834729: build 8.1s, nonroot:nonroot, curl /config.json y CSP ok)
Spin (modelo del protocolo y mutantes)      	pass	1m59s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 21/21 paquetes con tests en verde; `cmd/web` 69,9 %) |
| `tsc -b` + Vitest + `vite build` | ✅ (0 errores de tipos, 27/27 tests en verde, build en 818 ms) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; la suite completa pasa) |
| Latencia API p95 < 100 ms | no aplica (no toca la API del coordinador) |
| CI de GitHub en el PR #79 | ✅ (7/7 jobs en verde, incluido `SPA (tipos, tests, build e imagen)`, run `44974692377`) |

## Verificación contra los criterios del issue #62

| Criterio (#62) | Estado | Evidencia |
|---|---|---|
| 1. `web/` con React + Vite + TypeScript; `npm run build` y `npm test` (Vitest) en verde en el CI (job `web`) | ✅ | Verificado localmente (`tsc -b`, 27/27 tests, `vite build`) y en GitHub Actions (job `SPA (tipos, tests, build e imagen)`, run `44974692377`). |
| 2. Login, cliente HTTP con token en cabecera, cierre de sesión al recibir 401, página 403 propia, menú según rol | ✅ | `web/src/api/cliente.ts`, `web/src/sesion/Sesion.tsx`, `web/src/App.tsx`, `web/src/paginas/Login.tsx`, `web/src/rutas.tsx`; 27/27 tests incluyendo 401 integrado, 401 tardío y cambio de rol tras Salir. |
| 3. `cmd/web` sirve `dist/`, `/config.json` con `API_URL` de entorno y CSP estricta | ✅ | `cmd/web/main.go` (100 % stdlib, `os.OpenRoot`, `validarAPI` estricta, `ServeContent`, apagado con `<-cerrado`), `cmd/web/main_test.go` (5/5 PASS) y prueba de caja negra. |
| 4. `deploy/Dockerfile.web` multi-etapa y servicio `web` en el Compose (`:8081` → `:8080`) | ✅ | `deploy/Dockerfile.web` (node → go sin `go mod download` → `distroless:nonroot`), `deploy/docker-compose.api.yml:25-35` y job `web` del CI (imagen probada con `curl`). |
| 5. Estilo Vigía (tipografías y paleta) con contraste WCAG AA en textos | ✅ | `web/src/estilos/vigia.css` (fuentes autoalojadas Instrument Serif + JetBrains Mono, contraste AA, enlace «Saltar al contenido» con `<main id="contenido" tabIndex={-1}>`, `prefers-reduced-motion`). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Alineado con `ADR-0007` (contrato `{token, correo, rol, expira}`, CSP estricta, orígenes separados `:8081`/`:8080`). |
| Implementación | 8 | Base web y contenedor listos | `cmd/web` en Go stdlib, imagen multi-etapa `distroless:nonroot`, Compose integrado y job `web` en CI. |
| Interfaz | 5 | Base sólida para #66–#69 | Login, roles, 403/404 propios, cierre en 401 sin carreras, navegación segura y accesibilidad base listos. |
| Informe | 2 | Aporta evidencia | Evidencia de CORS entre orígenes, CSP sin `unsafe-*` y contenedor nonroot en CI. |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | Base `integracion/tb2` @ `b405959` integrada en `cf39c16`; 7/7 jobs del CI en verde (run `44974692377`, incluido el job `web` probando build, tests e imagen Docker con `curl`); PR #75 (`ADR-0007`, contrato de login) aprobado en ronda 2. |
| H2 (ronda 1 · MAYOR) | RESUELTO | `75c9708` (`cmd/web/main.go:44-83, 132-167`): `Cache-Control: immutable` solo en `servirArchivo` cuando el asset existe (404 sin caché larga); `validarAPI` rechaza `;`, `,`, espacios, credenciales, query/ForceQuery, fragmento y puertos inválidos; `main` espera `<-cerrado`; `os.OpenRoot` + `http.ServeContent` evita el 301 en `/index.html` y bloquea `%5c`. Verificado en tests y caja negra. |
| H3 (ronda 1 · MAYOR) | RESUELTO | `d62b7eb` (`cliente.ts:72, 97-109`, `Sesion.tsx:8-18, 46-78`, `app.test.tsx:144-171, 212-227`): `esSesion` exige `token`/`correo` no vacíos y `expira` finito; `entrar` valida con `esSesion` antes de leer; `leerGuardada` borra sesiones vencidas/corruptas; `alExpirar(tokenFallido)` + `actual.current = null` en `salir()` evitan que un 401 tardío cierre una sesión nueva; prueba integrada de 401 en `app.test.tsx`. |
| H4 (ronda 1 · MAYOR) | RESUELTO | `de884d4` (`rutas.tsx:34-47`, `App.tsx:12, 26-29`, `Login.tsx:29, 37, 85`, `app.test.tsx:173-210`): `destinoTrasLogin` valida rutas internas seguras y permisos del rol nuevo; `Salir` navega a `/login` sin recordar la vista; `ConSesion` conserva `pathname + search`; «Entrar» exige `correo.trim()`. |
| H5 (ronda 1 · MENOR) | RESUELTO | `1876948` + `de884d4` + `d62b7eb`: `.dockerignore` con `!cmd/web/*.go` y sin `*_test.go`, `Dockerfile.web` sin `go mod download`, `API_URL` sobreescribible en `docker-compose.api.yml:32`, `trap` de limpieza en `.github/workflows/ci.yml:131`, `<main tabIndex={-1}>` en `App.tsx:60` y rechazo de `api_url` vacía en `config.ts:12`. |

## Cómo explicarlo en la sustentación

La SPA Vigía (React + Vite + TypeScript) se compila en una imagen multi-etapa independiente y la sirve `cmd/web` en Go stdlib sobre `distroless:nonroot` en `:8081`, mientras la API corre en `:8080`: al estar en orígenes distintos, la política CORS se ejercita de verdad y la URL de la API se inyecta en tiempo de ejecución vía `/config.json`. El token JWT vive en `sessionStorage` y viaja solo en la cabecera `Authorization` (evitando CSRF), mientras `cmd/web` aplica una CSP estricta derivada de una `API_URL` validada y sirve los estáticos confinados con `os.OpenRoot`; el cliente HTTP cierra sesión ante `401` sin ser afectado por respuestas tardías de sesiones previas, y el menú/redirección post-login se ajustan al rol manteniendo la autorización real en el servidor.
