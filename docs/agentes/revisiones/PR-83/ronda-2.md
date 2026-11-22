# Veredicto — PR-83 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/61-roles-cors` @ `3aacce7` (base: `integracion/tb2` @ `4e1842c`)
- **Fecha:** 2026-11-22
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H4**) quedaron resueltos y verificados con evidencia reproducible en los commits `8b1af63`, `6a0c187`, `33c8c2d` y `21f0801`, sin introducir defectos ni regresiones (`AGENTS.md` §3.2, §3.4–§3.6): `reports/bench_api_jorge-pc_jwt.json` queda versionado con la medición real contra Compose (MongoDB + Redis, 3 nodos, 16 clientes, 2000 peticiones por corrida, `autenticacion: "jwt"`, 0 errores, p95 **3,89 ms** en frío y **3,09 ms** en caché, ambos `< 100 ms`), y `apiReal()` en `cmd/informe/regiones.go` descarta `*_jwt.json` para no alterar la sección PC4 (**H1**); `leerJSON` lee el cuerpo acotado con `io.ReadAll(http.MaxBytesReader(w, r.Body, MaxCuerpo))` antes de decodificar, por lo que cualquier `POST` > 64 KiB —incluso no-JSON o *chunked*— responde `413` preservando CORS y manteniendo 401/403 antes del límite, y `POST /api/v1/entrenamientos` ya declara `"413"` en `docs/api/openapi.yaml` con guardia automática en `TestOpenAPISeguridad` (**H2**); `normalizarOrigen` + `puertoValido` rechazan en el arranque host vacío, comodines parciales y puertos vacíos, con cero inicial o fuera de `1..65535` (comprobado contra el binario `cmd/coordinator`), y `Rutas()` devuelve `slices.Clone(roles)` (**H3**); se añadieron tests de CORS en `429`/`500`, `Cache-Control` por ruta, CORS real en `TestEvaluacionConservaVaryDeCORS`, parser estricto de `securitySchemes.bearerAuth` en `seguridadDelOpenAPI` y rechazo explícito de `-token` + `-correo` juntos en `cmd/benchapi` (**H4**). Toda la verificación local (`gofmt`, `go vet`, `-tags integracion`, 25/25 paquetes Go con `-race`, 0 carreras, `go run ./cmd/informe` sin cambios) y los 7/7 jobs de GitHub Actions sobre `3aacce7` están en verde.

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/api ./cmd/coordinator ./cmd/benchapi
github.com/JorgeGarcia005/cc65-tf-dengue/internal/api -> [bytes compress/gzip context crypto/rand crypto/sha256 encoding/hex encoding/json errors fmt .../docs/api .../almacen .../auth .../cluster .../features .../ingest .../ml io log math net net/http net/url runtime/debug slices strconv strings sync sync/atomic time unicode/utf8]
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator -> [context encoding/json errors flag fmt .../almacen .../almacen/mongo .../almacen/redis .../api .../auth .../cluster .../gold .../ml io/fs log math net os os/signal path/filepath sync sync/atomic syscall time]
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi -> [bytes context encoding/json errors flag fmt .../maquina io log math math/rand/v2 net/http os path/filepath runtime slices strings sync time]

$ go test -race -count=1 -v -run 'MatrizRoles|CORS|Cabeceras|OpenAPI|ServicioExigeToken|PrepararSeguridad|MedirConLogin' ./internal/api ./cmd/coordinator ./cmd/benchapi
PASS (19/19: TestOpenAPICubreLasRutas, TestOpenAPIMapasEnLineaBienFormados, TestEvaluacionConservaVaryDeCORS,
TestMatrizRoles [11 rutas × 3 roles], TestMatrizRolesTokenAjenoYSinFirmador, TestCORSPreflightOrigenPermitido,
TestCORSPreflightOrigenNoPermitido, TestCORSRespuestasDecoradas, TestCORSOrigenesValidos, TestCabecerasSeguridad,
TestCabecerasLimiteDeCuerpo413, TestOpenAPISeguridad, TestMatrizRolesCoincideConADR,
TestMatrizRolesRutasDevuelveCopias, TestCORSEnRespuestas429Y500, TestCabecerasCacheSegunRuta,
TestServicioExigeTokenYAplicaCORS, TestPrepararSeguridad, TestMedirConLoginUsaJWT)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	12.561s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	7.707s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	13.411s

$ go test -race -count=1 -v ./cmd/informe && go run ./cmd/informe && git status --short
PASS (6/6, incl. TestFragmentosSalenDelFuente)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.261s
(git status --short vacío: sin drift en docs/informe/pc4/10-api.md)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.673s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	10.191s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	7.956s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.053s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	2.273s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.513s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.641s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.565s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	4.052s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.101s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.237s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.025s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.078s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	27.461s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.918s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.657s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.004s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.047s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.979s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.326s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.745s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.757s

$ go run ./cmd/benchapi -h
  -correo string  inicia sesión con este correo y la clave de $BENCH_CLAVE (por defecto $BENCH_CORREO)
  -salida string  reporte (por defecto reports/bench_api_<maquina>[_jwt].json)
  -token string   JWT para Authorization: Bearer (por defecto $BENCH_TOKEN)

$ <prueba CLI contra cmd/coordinator -cors inválido y cmd/benchapi -token + -correo>
- http://:8081          -> origen CORS "http://:8081": falta el host, lleva comodín o credenciales
- http://*.example.com  -> origen CORS "http://*.example.com": falta el host, lleva comodín o credenciales
- http://localhost:     -> origen CORS "http://localhost:": el puerto debe estar entre 1 y 65535
- http://localhost:99999 -> origen CORS "http://localhost:99999": el puerto debe estar entre 1 y 65535
- http://localhost:0    -> origen CORS "http://localhost:0": el puerto debe estar entre 1 y 65535
- http://localhost:08081 -> origen CORS "http://localhost:08081": el puerto debe estar entre 1 y 65535
- BENCH_CORREO=a@b.pe BENCH_TOKEN=tok BENCH_CLAVE=x go run ./cmd/benchapi -url http://127.0.0.1:1:
  usa -token o -correo, no ambos (revisa $BENCH_TOKEN y $BENCH_CORREO) (exit status 1)

$ jq '{maquina, so, autenticacion, version_modelo, concurrencia, limite_p95_ms, corridas: [.corridas[] | {nombre, peticiones, errores, cache_hits, cache_misses, p50_ms, p95_ms, p99_ms, con_cache, cumple_limite}]}' reports/bench_api_jorge-pc_jwt.json
- fria:     2000 pet., 0 err., 0 HIT / 2000 MISS, p50=2.2801 ms, p95=3.8904 ms, p99=4.6391 ms, con_cache=true, cumple_limite=true
- caliente: 2000 pet., 0 err., 2000 HIT / 0 MISS, p50=1.7557 ms, p95=3.0943 ms, p99=3.8865 ms, con_cache=true, cumple_limite=true

$ API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config
CORS_ORIGENES: http://localhost:8081  ·  JWT_SECRETO: ""  ·  API_URL: http://localhost:8080

$ gh pr checks 83   # run 44990405068 sobre 3aacce7
Docker (imagen, clúster y caída de un nodo)	pass	2m39s
Ecosistema de agentes                       	pass	10s
Go (gofmt, vet, test -race)                 	pass	1m3s
MongoDB y Redis (tests de integración)      	pass	55s
OpenAPI 3 válido                            	pass	22s
SPA (tipos, tests, build e imagen)          	pass	53s
Spin (modelo del protocolo y mutantes)      	pass	1m52s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes con tests PASS) |
| CI GitHub Actions sobre `3aacce7` | ✅ (7/7 jobs PASS, run `44990405068`) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; suite completa en verde) |
| Latencia API p95 < 100 ms con JWT | ✅ `reports/bench_api_jorge-pc_jwt.json`: p95 **3,89 ms** en frío y **3,09 ms** en caché, 0 errores, 2000 pet./corrida, 16 clientes, Mongo + Redis |

## Verificación contra los criterios del issue #61

| Criterio (#61) | Estado | Evidencia |
|---|---|---|
| 1. Test tabla desde `Rutas()`: ruta × {sin token, analista, admin} → 401/403/2xx según ADR-0007 | ✅ | `TestMatrizRoles` (11 rutas × 3 roles), `TestMatrizRolesTokenAjenoYSinFirmador` (401 + 503 sin firmador), `TestMatrizRolesCoincideConADR` y `TestMatrizRolesRutasDevuelveCopias` PASS. |
| 2. Preflight origen permitido → 204 con `Access-Control-Allow-*` y `Vary: Origin`; ajeno → sin `Access-Control-Allow-Origin` | ✅ | `TestCORSPreflightOrigenPermitido`, `TestCORSPreflightOrigenNoPermitido`, `TestCORSRespuestasDecoradas`, `TestCORSEnRespuestas429Y500` y `TestCORSOrigenesValidos` PASS. |
| 3. `X-Content-Type-Options`, `Referrer-Policy` y límite de cuerpo en POST (413) | ✅ | `TestCabecerasSeguridad`, `TestCabecerasLimiteDeCuerpo413` (JSON grande, 70 000 bytes no-JSON con CORS y 401 antes que 413) y `TestCabecerasCacheSegunRuta` PASS. |
| 4. OpenAPI declara `bearerAuth` y 401/403 en toda ruta protegida | ✅ | `TestOpenAPISeguridad` (valida `components.securitySchemes.bearerAuth` `type: http`/`scheme: bearer`, 401/403 en rutas protegidas y 413 en todo POST) y job CI `OpenAPI 3 válido` PASS. |
| 5. `cmd/benchapi` con token → `reports/bench_api_<maquina>_jwt.json` con p95 < 100 ms y 0 errores | ✅ | `reports/bench_api_jorge-pc_jwt.json` (`autenticacion: "jwt"`, p95 3,89 ms frío / 3,09 ms caliente, 0 errores, `cumple_limite: true`), `cmd/benchapi -h` y `TestMedirConLoginUsaJWT` PASS. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Alineado con `ADR-0007` (matriz verificada automáticamente por `TestMatrizRolesCoincideConADR` y orden real de middlewares documentado). |
| Implementación | 8 | Cierra seguridad de API y p95 con JWT | Autorización por rol desde tabla única, fallo cerrado (503 sin firmador y arranque bloqueado sin `JWT_SECRETO`), CORS estricto por lista blanca, cabeceras + 413 en cualquier cuerpo > 64 KiB y p95 < 100 ms demostrado con JWT en Compose real, todo en stdlib con 0 carreras. |
| Interfaz | 5 | Habilita integración completa de la SPA | CORS expone `X-Cache`, `Retry-After` y `Location` en toda respuesta (incluidos 401, 403, 413, 429 y 500) y atiende preflights antes de la autenticación, coincidiendo con `web/src/api/cliente.ts`. |
| Informe | 2 | Respaldado sin cifras a mano | `docs/informe/pc4/10-api.md` sincronizado (`TestFragmentosSalenDelFuente` ✅ y `apiReal()` protegido contra `*_jwt.json`), y `reports/bench_api_jorge-pc_jwt.json` disponible para la evaluación TB2 (`AGENTS.md` §3.6). |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `8b1af63` añade `reports/bench_api_jorge-pc_jwt.json` medido contra Compose completo (Mongo + Redis, 3 nodos, 16 clientes, 2000 pet./corrida, `autenticacion: "jwt"`, 0 errores, p95 3,89 ms frío / 3,09 ms caliente, `cumple_limite: true`); `6a0c187` actualiza `apiReal()` en `cmd/informe/regiones.go:307-323` para excluir `*_jwt.json` (`TestFragmentosSalenDelFuente` PASS y `go run ./cmd/informe` sin diff). |
| H2 (ronda 1 · MAYOR) | RESUELTO | `33c8c2d` lee el cuerpo acotado con `io.ReadAll(http.MaxBytesReader(w, r.Body, MaxCuerpo))` antes de decodificar en `internal/api/api.go:575-589` (cubre tanto cuerpos con `Content-Length` como *chunked*, JSON o no-JSON), añade pruebas de 70 000 bytes no-JSON con CORS y 401 previo a 413 (`internal/api/seguridad_test.go:321-338`), agrega `"413"` a `POST /api/v1/entrenamientos` (`docs/api/openapi.yaml:218`) y exige `"413"` en todo `POST` desde `TestOpenAPISeguridad` (`seguridad_test.go:431-433`). |
| H3 (ronda 1 · MAYOR) | RESUELTO | `33c8c2d` endurece `normalizarOrigen` + `puertoValido` (`internal/api/seguridad.go:67-91`) rechazando host vacío, comodines parciales, `:` sin puerto y puertos `0`, con cero inicial o `> 65535` (verificado en `TestCORSOrigenesValidos` y directamente contra el binario `cmd/coordinator`), y clona `Roles` con `slices.Clone(roles)` en `Rutas()` (`internal/api/api.go:295-296`, cubierto por `TestMatrizRolesRutasDevuelveCopias`). |
| H4 (ronda 1 · MENOR) | RESUELTO | `33c8c2d` añade `TestCORSEnRespuestas429Y500` y `TestCabecerasCacheSegunRuta` (`internal/api/seguridad_test.go:535-578`), reemplaza el middleware falso por `OrigenesCORS` real en `TestEvaluacionConservaVaryDeCORS` (`internal/api/evaluacion_test.go:80-88`) y exige `sc.Err()`, `securitySchemes` y `type: http` + `scheme: bearer` en `seguridadDelOpenAPI` (`seguridad_test.go:395-407`); `21f0801` hace explícito el error si se pasan `-token` y `-correo` juntos (`cmd/benchapi/main.go:156-160`, `main_test.go:381-384`). |

## Cómo explicarlo en la sustentación

Todas las rutas HTTP salen de una tabla única (`api.Rutas()`, con copia defensiva de roles) que declara si cada endpoint es público o `Bearer` y qué roles admite (`analista` o `admin`). `Handler()` aplica en stdlib la cadena `recuperar → cabecerasSeguridad → cors → ServeMux → Autenticar + Requiere → handler`: el preflight `OPTIONS` se responde antes de la autenticación solo para orígenes estrictamente validados de `CORS_ORIGENES`, toda respuesta (incluidos 401, 403, 404, 413, 429 y 500) conserva cabeceras de seguridad y CORS, todo `POST` > 64 KiB se rechaza con 413 sin importar si es JSON válido, y sin `JWT_SECRETO` el coordinador no arranca y la API falla cerrada con 503. Además, `TestMatrizRoles` y `TestMatrizRolesCoincideConADR` garantizan que código y `ADR-0007` no diverjan, `TestOpenAPISeguridad` audita `bearerAuth`, 401/403 y 413 en el contrato, y `reports/bench_api_jorge-pc_jwt.json` demuestra que verificar el JWT mantiene el p95 en 3,89 ms en frío y 3,09 ms en caché (< 100 ms, 0 errores) sobre el clúster en Docker Compose.
