# PR-79 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/62-spa-base`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO (dependencia) · ACLARO | El CI (incluido el job `web` que cito en los criterios) no corre hasta fusionar #74 (#56); tras fusionarlo se actualiza esta rama. Sobre #60: este PR no lo necesita para sus criterios. El login se prueba con `fetch` simulado contra el contrato `{token, correo, rol, expira}` fijado en ADR-0007 (PR #75) y la integración real se verifica en #60. |
| H2 | MAYOR | ACEPTO | `75c9708` — (1) la caché `immutable` se pone solo cuando el asset existe (en `servirArchivo`); un 404 de `assets/` no la lleva (aserción nueva). (2) `validarAPI`: solo `http(s)://host[:puerto]`; rechaza `;`, `,`, espacios u otros caracteres en el host, host vacío, credenciales, query (incluida `?` vacía), fragmento, ruta y puertos fuera de 1–65535 (14 casos en `TestConfiguracionInvalida`, más `TestAPIURLValidas`). (3) `main` espera a que `Shutdown` termine (`<-cerrado`). (4) Los archivos se sirven con `os.OpenRoot(dir)` + `http.ServeContent`: `/index.html` responde 200 sin el 301 de `FileServer`, y una ruta con `\` (`/..%5csecreto.txt`) da 404 y no puede salir de `dir` ni en Windows. |
| H3 | MAYOR | ACEPTO | `d62b7eb` — `esSesion` exige token y correo no vacíos y `expira` finito; `entrar` valida la respuesta con `esSesion` **antes** de leerla (un 200 con HTML muestra «No se pudo entrar.» sin `TypeError`); `leerGuardada` borra una sesión vencida o corrupta; `alExpirar` recibe el token que falló y solo cierra si sigue siendo el token actual. Tests: `esSesion…` (6 casos), 401 integrado (borra `sessionStorage`, vuelve al login y muestra el aviso), 401 tardío de la sesión anterior que no cierra la nueva (comprobado: sin la guarda, el test falla), login 200 sin JSON, sesión guardada con token vacío. |
| H4 | MAYOR | ACEPTO | `de884d4` — `destinoTrasLogin(desde, rol)`: solo rutas internas (`/`, no `//`, `\` ni `/login`) y solo si el rol nuevo puede ver la vista; si no, al inicio. «Salir» navega a `/login` sin recordar la vista. `ConSesion` guarda `pathname + search`. «Entrar» exige `correo.trim()`. Tests: admin sale desde `/admin` → analista aterriza en el inicio; `//evil.example`, `/\evil.example`, `https://evil.example`, `/login` y `/admin` (analista) → inicio; la query de `/mapa?semana=20&…` se conserva; correo de solo espacios deja «Entrar» deshabilitado. |
| H5 | MENOR | ACEPTO | `1876948` — `.dockerignore` con `!cmd/web/*.go` y sin `*_test.go`; el Dockerfile ya no hace `go mod download` (comprobado: `GOMODCACHE` vacío + `GOPROXY=off` → `go build ./cmd/web` compila); Compose: `API_URL: ${API_URL:-http://localhost:${PUERTO_API:-8080}}`; CI: `trap 'docker rm -f web' EXIT`. `de884d4`: `<main tabIndex={-1}>` para el enlace «Saltar al contenido». `d62b7eb`: `config.ts` rechaza `api_url` vacía. |

## Verificación

```text
$ gofmt -l cmd/web && go vet ./cmd/web            (vacío)
$ go test -race -count=1 -cover ./cmd/web
ok   .../cmd/web   coverage: 69.9% of statements   (main sin test; nuevoServidor/validarAPI/servirArchivo cubiertos)
$ cd web && npx tsc -b && npx vitest run && npm run build
Tests  27 passed (27) · ✓ built
```
