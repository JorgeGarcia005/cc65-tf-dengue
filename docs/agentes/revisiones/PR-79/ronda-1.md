# Veredicto — PR-79 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/62-spa-base` @ `57e32cb` (base: `integracion/tb2` @ `396c4ce`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR implementa la base de la SPA Vigía en `web/` (React 19 + Vite 8 + TypeScript, 14 pruebas con Vitest y tipografías autoalojadas con contraste WCAG AA verificado), el servidor estático `cmd/web` **exclusivamente con la biblioteca estándar de Go** (`AGENTS.md` §3.2), el build multi-etapa `deploy/Dockerfile.web` sobre `distroless:nonroot` (15,6 MB), el servicio `web` en `deploy/docker-compose.api.yml` (`:8081`, origen distinto al de la API en `:8080` para ejercitar CORS real) y el job `web` en `.github/workflows/ci.yml`. Las verificaciones locales (`gofmt`, `go vet`, `go test -race ./...`, `go test -race -count=10 ./cmd/web`, `npm ci && typecheck && test && build`, `docker build` + smoke test de `/config.json` y CSP sobre el contenedor y cálculo de contrastes WCAG AA) pasan en verde. Sin embargo, se emiten **1 hallazgo bloqueante**, **3 mayores** y **1 menor** antes de aprobar: el PR carece de corridas de CI en GitHub porque `#56` (PR #74) sigue sin fusionar en `integracion/tb2` —pese a que la descripción marca criterios como verificados por el job `web` del CI— y su dependencia `#60` (contrato de login) y `#57` (ADR-0007) siguen abiertas (**H1**); `cmd/web` marca respuestas `404` de `/assets/*` como `public, max-age=31536000, immutable`, valida `API_URL` de forma incompleta permitiendo inyectar `;` en la cabecera `Content-Security-Policy`, no espera a `srv.Shutdown` al recibir `SIGTERM` y redirige `GET /index.html` con `301` (**H2**); en la sesión del frontend, `esSesion` acepta `token: ""` (sesión zombi que nunca cierra ante `401`), `entrar()` accede a `r.token` antes de validar con `esSesion`, existe carrera lógica ante un `401` tardío tras salir o cambiar de cuenta, y falta prueba de integración de cierre por `401` en `app.test.tsx` (**H3**); la redirección post-login envía a un `analista` a un `403` si un `admin` cerró sesión desde `/admin`, no valida rutas protocol-relative (`//...`) y descarta `search` (**H4**); y hay detalles menores de fragilidad en `.dockerignore`, `Dockerfile.web`, Compose y accesibilidad (**H5**).

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.Imports}}' ./cmd/web
[context encoding/json errors flag fmt log net/http net/url os os/signal path path/filepath strings syscall time]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	7.756s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	16.515s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.579s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.047s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.288s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.409s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.472s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.186s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.027s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.056s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.074s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.451s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.103s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.061s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.085s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.271s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.681s

$ go test -race -count=10 -coverprofile=/tmp/web_cmd.cov ./cmd/web && go tool cover -func=/tmp/web_cmd.cov
PASS (10/10 iteraciones: TestCabecerasYCSP, TestConfigJSON, TestRutasDelSPA, TestConfiguracionInvalida)
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web/main.go:28:	main		0.0%
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web/main.go:57:	nuevoServidor	97.4%
total:								(statements)	64.4%

$ npm --prefix web ci --no-audit --no-fund && npm --prefix web run typecheck && npm --prefix web test && npm --prefix web run build
✓ src/pruebas/app.test.tsx (10 tests)
✓ src/pruebas/cliente.test.ts (4 tests)
Test Files  2 passed (2) · Tests  14 passed (14)
✓ built in 2.92s (web/dist/index.html sin scripts ni estilos en línea)

$ docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config --quiet
compose config OK

$ docker build -f deploy/Dockerfile.web -t cc65-tf-web-juez .
$ docker image inspect cc65-tf-web-juez --format 'user={{.Config.User}} size={{.Size}}'
user=nonroot:nonroot size=15614798
# Smoke test sobre el contenedor en ejecución:
# /config.json -> {"api_url":"http://localhost:8080"}
# /mapa -> 200 OK, Content-Security-Policy: default-src 'self'; ..., <div id="raiz">

$ # Contraste WCAG 2.1 sobre --papel (#f4eedf) y --tarjeta (#fffdf6)
tinta (#111111) sobre papel: 16.31:1 · tinta-2 (#3d3a33): 9.80:1 · azul (#2b4edb): 5.64:1 · error (#b3123f): 5.90:1
tinta sobre amarillo (#f5d547): 13.03:1 · sobre rosa-suave (#f7c4d4): 12.41:1 · sobre #d6def9: 14.10:1

$ gh pr checks 79
no checks reported on the 'feature/62-spa-base' branch

$ gh run list --branch feature/62-spa-base --limit 10 --json databaseId,workflowName,status,conclusion
[]
```

| Chequeo | Resultado |
|---|---|
| `gofmt` / `go vet` | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Solo biblioteca estándar en `cmd/web` (`AGENTS.md` §3.2) | ✅ |
| SPA (`tsc -b`, Vitest 14/14, `vite build`, WCAG AA) | ✅ |
| Imagen `Dockerfile.web` (`nonroot`, 15,6 MB, CSP, `/config.json`) | ✅ (verificado localmente por el juez) |
| CI del PR en GitHub | ❌ (sin checks reportados; base `integracion/tb2` aún sin `#56`) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`) |
| Latencia API p95 < 100 ms | no aplica (no toca `internal/api`) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---:|---|---|
| Planteamiento | 3 | fuera de alcance | Cubierto en PC3/PC4 y ADR-0007 (`#57`). |
| Implementación | 8 | parcial (bloqueado por H1; mejoras H2–H3) | Cumple contenedor separado en Go stdlib (`nonroot`) y separación de orígenes (`:8081` vs `:8080`), pero faltan el CI verde, correcciones de caché/CSP/shutdown en `cmd/web` y endurecimiento de sesión. |
| Interfaz | 5 | base sólida (mejoras H3–H4) | Establece React + Vite + TS, estilo Vigía con contraste AA real, fuentes autoalojadas, login, menú por rol y vistas 403/404; las vistas funcionales llegan en `#66`–`#69`. |
| Informe | 2 | fuera de alcance | Corresponde a `#71`; solo documenta `deploy/README.md`. |
| Video | 2 | fuera de alcance | Corresponde a `#72`. |

## Hallazgos

### H1 — Sin ejecución de CI en GitHub (base `integracion/tb2` sin `#56`), dependencias `#57`/`#60` abiertas y afirmación inexacta en el PR · BLOQUEANTE

- **Dónde:** `.github/workflows/ci.yml:3-7`, descripción del PR #79 (*Criterios de aceptación*) e issue #62 (*Depende de #60*).
- **Evidencia:**
  1. `gh pr checks 79` devuelve `no checks reported on the 'feature/62-spa-base' branch` y `gh run list --branch feature/62-spa-base` devuelve `[]`. En `integracion/tb2` (`396c4ce`) el workflow `.github/workflows/ci.yml:3-7` aún solo escucha `branches: [main, develop]`, porque `#56` (PR #74) sigue abierto.
  2. El criterio de aceptación #1 del issue #62 exige *«`npm run build` y `npm test` (Vitest) en verde en el CI — verificar con: job nuevo `web`»*, y `juez-cc65` prohíbe emitir `APROBADO` sin el CI del PR en verde. Aun así, el cuerpo del PR marcó como verificados con *«el job `web` del CI»* tanto el build/tests de `web/` como la imagen Docker, cuando el job nunca se ejecutó en GitHub Actions.
  3. El issue #62 depende de `#60` (`POST /api/v1/auth/login`) y se apoya en `ADR-0007` (`#57`, PR #75), ambos todavía abiertos. Cualquier ajuste en el contrato JSON de login (`expira`, claves, `Retry-After`) antes de cerrar `#60` desalinearía la SPA.
- **Corrección sugerida:**
  - Fusionar `#56` (PR #74) y `#57` (PR #75) en `integracion/tb2`, alinear/validar el contrato de login con `#60`, actualizar `feature/62-spa-base` sobre `integracion/tb2` y verificar que el workflow completo de CI —incluido el job `web`— corra y pase en verde en GitHub.
  - Ajustar la descripción del PR para reflejar evidencia real de ejecución una vez corra el CI.

### H2 — `cmd/web`: `404` de `/assets/*` cacheado como `immutable` por 1 año, validación de `API_URL` permite inyección en CSP, apagado sin esperar `Shutdown` y `301` en `/index.html` · MAYOR

- **Dónde:** `cmd/web/main.go:42-53, 61-66, 94-120` y `cmd/web/main_test.go:79-120`
- **Evidencia:**
  1. **`404` cacheado como inmutable por 1 año (`main.go:101-118`):** `Cache-Control: public, max-age=31536000, immutable` se escribe *antes* de comprobar con `os.Stat` que el archivo exista. Probado contra el binario compilado (`go build ./cmd/web`):
     ```text
     GET /assets/no-existe.js -> 404 | CC='public, max-age=31536000, immutable' | CL='19'
     ```
     Según RFC 9111 §3, un `404` con `public, max-age=31536000, immutable` es almacenable en caché por navegadores y proxies durante un año.
  2. **Inyección en `Content-Security-Policy` por validación laxa de `API_URL` (`main.go:61-66`):** `url.Parse` permite `;` en `u.Host` (sub-delim RFC 3986), hostname vacío (`:8080`), credenciales (`user:pass@`), query y fragmento. Probado contra el binario compilado:
     ```text
     ACCEPTED API_URL='http://a;b:8080' -> CSP=...; connect-src 'self' http://a;b:8080 ws://a;b:8080; ...
     ACCEPTED API_URL='http://:8080' -> body={"api_url":"http://:8080"}
     ACCEPTED API_URL='http://user:pass@localhost:8080?x=1#frag' -> body={"api_url":"http://localhost:8080"}
     ```
     El `;` rompe la directiva `connect-src` e inyecta una nueva directiva CSP (`b:8080`), y las otras dos incumplen el formato anunciado `http(s)://host[:puerto]`.
  3. **Apagado elegante incompleto en `main()` (`main.go:42-53`):** `srv.Shutdown(c)` corre en una goroutine aparte sin sincronización de cierre. Según `go doc net/http.Server.Shutdown`: *«When Shutdown is called, Serve, ListenAndServe, and ListenAndServeTLS immediately return ErrServerClosed. Make sure the program doesn't exit and waits instead for Shutdown to return.»* Al recibir `SIGTERM`, `main()` termina inmediatamente en cuanto `ListenAndServe` devuelve `ErrServerClosed`, cortando peticiones en vuelo antes de que `Shutdown` las drene.
  4. **`GET /index.html` devuelve `301` y divergencia `os.Stat` vs `http.FileServer` (`main.go:105-112`):** `GET /index.html` entra en `:106`, ejecuta `:108` (`cab.Set("Cache-Control", "no-cache")`, línea sin cobertura en `main_test.go`), y delega a `http.FileServer`, que siempre responde `301 Moved Permanently` a `./` (`GET /index.html -> 301 | Loc='./'`). Además, `os.Stat` valida sobre `limpia` pero `archivos.ServeHTTP(w, r)` recibe `r.URL.Path` original (en Windows, `\` no lo limpia `path.Clean`, generando discrepancia entre `os.Stat` y `http.Dir`).
- **Corrección sugerida:**
  - Mover el `Cache-Control: public, max-age=31536000, immutable` *dentro* del bloque `if err == nil && !info.IsDir()`, solo cuando el asset existe (nunca en `404`), y agregar una aserción en `TestRutasDelSPA` de que `GET /assets/no-existe.js` no devuelve `immutable`.
  - Endurecer `API_URL`: exigir `u.Hostname() != ""`, `u.User == nil`, `u.RawQuery == ""`, `u.Fragment == ""`, `u.Opaque == ""` y rechazar `;`, `,` o espacios en `u.Host` (añadir estos casos a `TestConfiguracionInvalida`).
  - Esperar a que termine `srv.Shutdown(c)` mediante un canal `cerrado := make(chan struct{})` cerrado al volver de `Shutdown` y bloqueando en `<-cerrado` cuando `ListenAndServe` retorne `http.ErrServerClosed`.
  - Servir archivos existentes con `http.ServeFile(w, r, rutaArchivo)` (o `os.OpenRoot(dir)` + `http.ServeContent`) en vez de pasar el `r` original a `http.FileServer`, evitando el `301` de `/index.html` y asegurando consistencia con `limpia` incluso en Windows.

### H3 — Sesión y cliente HTTP: `esSesion` acepta `token: ""` (sesión zombi sin cierre en 401), `TypeError` si login devuelve 200 no-JSON, carrera lógica en 401 tardío y falta prueba integrada de 401 · MAYOR

- **Dónde:** `web/src/api/cliente.ts:62-80, 97-106`, `web/src/sesion/Sesion.tsx:8-15, 48-72` y `web/src/pruebas/app.test.tsx`
- **Evidencia:**
  1. **`esSesion` permite `token: ""`, `correo: ""` y `expira: NaN/Infinity` (`cliente.ts:97-106`):** solo valida `typeof s.token === 'string'` y `typeof s.expira === 'number'`. Si una sesión entra con `token: ""`, `cliente.ts:47` (`if (t)`) omite la cabecera `Authorization` y `cliente.ts:72` (`if (resp.status === 401 && t)`) **nunca llama a `alExpirar`**, dejando una sesión «autenticada» zombi que no expira ante respuestas `401`.
  2. **Acceso a `r.token` antes de validar con `esSesion` (`Sesion.tsx:63-65` + `cliente.ts:62-80`):** si `POST /api/v1/auth/login` responde `200` con cuerpo vacío o HTML (p. ej. un proxy mal configurado), `cliente.ts:80` devuelve `undefined as T`, por lo que `Sesion.tsx:64` (`r.token`) lanza `TypeError` antes de alcanzar el `if (!esSesion(nueva))` de `:65`.
  3. **Carrera lógica en `401` tardío (`cliente.ts:72` + `Sesion.tsx:52-56`):** si una petición enviada con el token `t1` responde `401` después de que el usuario pulsó **Salir** (o inició sesión con otro token `t2`), `t` sigue siendo `t1` (truthy) y dispara `alExpirar()`, mostrando un falso aviso *«Tu sesión venció»* o cerrando la sesión nueva `t2`.
  4. **Sin test integrado de cierre por `401` en `app.test.tsx`:** `cliente.test.ts:20-26` solo prueba `crearCliente` aislado con un mock `alExpirar = vi.fn()`. Ningún test en `app.test.tsx` ejercita `ProveedorSesion` + `App` ante un `401` (borrado de `sessionStorage`, redirección a `/login` y aparición del `role="status"` con *«Tu sesión venció. Vuelve a entrar.»*).
- **Corrección sugerida:**
  - En `esSesion` (`cliente.ts:97-106`), exigir `s.token.trim().length > 0`, `s.correo.trim().length > 0` y `Number.isFinite(s.expira)`.
  - En `Sesion.tsx:64-65`, comprobar `if (!esSesion(r)) throw new Error('Respuesta de login inválida.')` *antes* de leer propiedades de `r`; además, en `leerGuardada` (`Sesion.tsx:8-15`), eliminar `sessionStorage.removeItem(CLAVE)` si la entrada guardada está vencida o corrupta.
  - Pasar el token que falló a `alExpirar?.(t)` y en `Sesion.tsx:53` comprobar `if (actual.current?.token !== tokenFallido) return`.
  - Añadir una prueba en `app.test.tsx` que realice una llamada con `useSesion().api`, reciba `401`, y verifique el borrado de `sessionStorage`, la vuelta a `/login` y el aviso visible.

### H4 — Flujo post-login y roles: `admin` sale desde `/admin` y el siguiente `analista` aterriza en `403`, falta validación de `desde` (`//...`) y se pierde `search` · MAYOR

- **Dónde:** `web/src/App.tsx:9-14, 49-51` y `web/src/paginas/Login.tsx:22, 28, 36, 84`
- **Evidencia:**
  1. **Cambio de rol tras pulsar «Salir» (`App.tsx:12, 49` + `Login.tsx:22, 36`):** si un `admin` está en `/admin` y hace clic en **Salir**, `setSesion(null)` provoca que `ConSesion` redirija a `/login` guardando `state: { desde: '/admin' }`. Cuando luego inicia sesión un `analista` (flujo habitual al demostrar roles en la sustentación), `Login.tsx:36` navega a `desde` (`'/admin'`), mostrando inmediatamente la pantalla **403 «Esta vista no es para tu rol»** tras un inicio de sesión exitoso.
  2. **Ruta `desde` sin validar (`App.tsx:12` + `Login.tsx:22, 36`):** si se accede sin sesión a una URL con doble barra (`//evil.example`), `ubicacion.pathname` conserva `//evil.example` y se pasa sin filtrar a `Navigate`/`navegar`. Además, guardar solo `ubicacion.pathname` descarta `ubicacion.search` (que usarán los filtros compartibles en `#66`/`#67`).
  3. **Validación de espacios en correo (`Login.tsx:84`):** `disabled={enviando || !correo || !clave}` deja habilitado el botón si `correo` contiene solo espacios (`"   "`), enviando `correo: ""` tras `correo.trim()`.
- **Corrección sugerida:**
  - En `Login.tsx`, validar que `desde` sea una ruta interna segura (empiece por `/` y no por `//`, distinta de `/login`) y comprobar si el rol recién autenticado tiene permiso sobre la vista destino (`VISTAS`): si no tiene permiso (o si se llegó por clic explícito en **Salir**), navegar a `/`.
  - Conservar `ubicacion.pathname + ubicacion.search` en `ConSesion` (solo cuando no sea cierre explícito) y cambiar `!correo` por `!correo.trim()` en `Login.tsx:84`. Cubrir con test en `app.test.tsx` el escenario *admin en `/admin` → Salir → login como analista → aterriza en `/`*.

### H5 — Detalles menores en `Dockerfile.web`, `.dockerignore`, Compose, CI y accesibilidad · MENOR

- **Dónde:** `deploy/Dockerfile.web.dockerignore:6`, `deploy/Dockerfile.web:22-23`, `deploy/docker-compose.api.yml:31`, `.github/workflows/ci.yml:131-136`, `web/src/App.tsx:26, 54` y `web/src/config.ts:11`
- **Evidencia:**
  1. `deploy/Dockerfile.web.dockerignore:6` lista literalmente `!cmd/web/main.go`: si se añade un segundo `.go` en `cmd/web/`, `docker build` lo excluirá y fallará. Con `!cmd/web/*.go` y `cmd/web/*_test.go` evita esa fragilidad.
  2. `deploy/Dockerfile.web:22-23` ejecuta `RUN go mod download` (8,6 s medidos en `docker build`), descargando `mongo-driver` y `go-redis` pese a que `cmd/web` es 100 % stdlib.
  3. `deploy/docker-compose.api.yml:31` fija `API_URL: http://localhost:${PUERTO_API:-8080}`, impidiendo sobreescribir `API_URL` desde el entorno (p. ej. si se accede por IP en red local durante la sustentación); mejor `${API_URL:-http://localhost:${PUERTO_API:-8080}}`.
  4. `.github/workflows/ci.yml:131-136` no registra `trap 'docker rm -f web || true' EXIT` antes de los `curl`.
  5. `web/src/App.tsx:54`: `<main id="contenido">` carece de `tabIndex={-1}` para asegurar el traslado de foco desde el enlace «Saltar al contenido»; y `web/src/config.ts:11` acepta `{"api_url": ""}` (cadena vacía).
- **Corrección sugerida:** Aplicar estos ajustes puntuales junto con las correcciones de H2–H4.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-79. |

## Cómo explicarlo en la sustentación

La SPA Vigía (React + Vite + TypeScript) se compila en una imagen multi-etapa independiente y la sirve `cmd/web` en Go stdlib sobre `distroless:nonroot` en el puerto `:8081`, mientras la API del coordinador corre en `:8080`: al ser orígenes distintos, la política CORS del servidor se ejercita de verdad y la URL de la API se inyecta al arrancar mediante `/config.json` sin recompilar el frontend. El token JWT se guarda en `sessionStorage` y viaja únicamente en la cabecera `Authorization` (eliminando CSRF), mientras `cmd/web` aplica una CSP estricta sin scripts en línea y con fuentes autoalojadas para mitigar XSS; el menú y las guardas de ruta filtran por rol (`admin` vs `analista`) mostrando un `403` propio, dejando siempre la autorización real en la API. Una vez que el CI corra en verde sobre `integracion/tb2` y se corrijan el caché en `404`, la validación de `API_URL`/apagado en `cmd/web` y los bordes de sesión/redirección post-login, la base quedará lista para montar las vistas `#66`–`#69`.
