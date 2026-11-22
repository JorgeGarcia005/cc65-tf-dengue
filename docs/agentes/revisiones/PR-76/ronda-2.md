# Veredicto — PR-76 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/58-auth` @ `238fc90` (base: `integracion/tb2` @ `b405959`)
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H6**) fueron resueltos y verificados con evidencia directa. La base `integracion/tb2` @ `b405959` quedó integrada (`238fc90`) y los 6 jobs del CI en GitHub pasan en verde (run `44974678960`). `internal/auth` sigue usando exclusivamente la biblioteca estándar de Go 1.24 (`AGENTS.md` §3.2), cierra la maleabilidad de firmas y cabeceras en `Verificar` (`base64.RawURLEncoding.Strict()`, rechazo de `\r`/`\n`, comparación exacta contra `cabeceraB64` y clasificación estricta en `errorDeCabecera`), exige vida ≥ 1 s en `NuevoFirmador`, fija los largos exactos de sal (16 bytes) y clave derivada (32 bytes) en `Comparar` con `base64.RawStdEncoding.Strict()`, y clona defensivamente `roles` en `Requiere`. La suite completa con `-race` pasa en verde con 0 carreras y 94,1 % de cobertura en `internal/auth`.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ go list -f '{{.Imports}}' ./internal/auth
[bytes context crypto/hmac crypto/pbkdf2 crypto/rand crypto/sha256 crypto/subtle encoding/base64 encoding/json errors fmt net/http slices strconv strings sync time]

$ go test -race -count=1 -v -cover ./internal/auth
=== RUN   TestClaveIdaYVuelta                  --- PASS (0.03s)
=== RUN   TestClaveSalDistinta                 --- PASS (0.01s)
=== RUN   TestClaveFormatoPorDefecto           --- PASS (4.66s)
=== RUN   TestClaveGuardadoInvalido            --- PASS (0.00s)
=== RUN   TestClaveSenueloNoVerifica           --- PASS (7.02s)
=== RUN   TestJWTIdaYVuelta                    --- PASS (0.00s)
=== RUN   TestJWTRechazos (21 subtests)        --- PASS (0.01s)
=== RUN   TestJWTSecretoCorto                  --- PASS (0.00s)
=== RUN   TestJWTFirmarRolInvalido             --- PASS (0.00s)
=== RUN   TestMiddleware (9 subtests)          --- PASS (0.00s)
=== RUN   TestMiddlewareRequiereSinAutenticar  --- PASS (0.00s)
=== RUN   TestMiddlewareRolesNoSeAlteran       --- PASS (0.00s)
=== RUN   TestMiddlewareConcurrente            --- PASS (0.01s)
PASS
coverage: 94.1% of statements
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.778s	coverage: 94.1% of statements

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.345s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.151s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.657s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.255s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.212s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.410s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.135s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.023s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.074s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.918s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	7.976s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.826s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.084s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.017s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.344s

$ gh pr checks 76   # run 44974678960 sobre 238fc90
Docker (imagen, clúster y caída de un nodo)	pass	2m19s
Ecosistema de agentes                       	pass	9s
Go (gofmt, vet, test -race)                 	pass	47s
MongoDB y Redis (tests de integración)      	pass	42s
OpenAPI 3 válido                            	pass	20s
Spin (modelo del protocolo y mutantes)      	pass	1m58s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 21/21 paquetes con tests en verde; 94,1 % de cobertura en `internal/auth`) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; la suite completa pasa) |
| Latencia API p95 < 100 ms | no aplica (la integración en la API y el benchmark con JWT corresponden a #60/#61) |
| CI de GitHub en el PR #76 | ✅ (6/6 jobs en verde, run `44974678960`) |

## Verificación contra los criterios del issue #58

| Criterio (#58) | Estado | Evidencia |
|---|---|---|
| 1. `Firmar`/`Verificar`: token alterado, `alg: none`, `alg: HS512`, `exp` vencido, `nbf` futuro, `aud`/`iss` ajenos → error tipado | ✅ | `internal/auth/jwt.go:60-161` y `TestJWTRechazos` (`jwt_test.go:60-117`, 21 casos, incluyendo bits sobrantes en firma, `\n`/`\r`, `typ` distinto, `ALG`, `crit`, cabecera `null`). |
| 2. `Hash`/`Comparar` PBKDF2-SHA256 (≥ 600 000 iteraciones, sal de 16 bytes) | ✅ | `internal/auth/clave.go:16-69` (600 000 iteraciones, sal 16 bytes, clave 32 bytes, `base64.RawStdEncoding.Strict()`, `subtle.ConstantTimeCompare`); `clave_test.go:11-86` decodifica sal/clave y rechaza truncados/sobredimensionados. |
| 3. Middleware `Autenticar` + `Requiere(roles...)`: sin token 401, rol ajeno 403, válido pasa con `sub` y `rol` en el contexto | ✅ | `internal/auth/middleware.go:26-60` y `middleware_test.go:11-140` (`TestMiddleware` verifica `sub` y `rol` y `error="invalid_token"` en vencido; `TestMiddlewareRolesNoSeAlteran` y `TestMiddlewareConcurrente` con 128 goroutines mixtas). |
| 4. Secreto < 32 bytes → error al construir | ✅ | `internal/auth/jwt.go:74-83` (`ErrSecretoCorto` y vida ≥ 1 s) y `TestJWTSecretoCorto` (`jwt_test.go:127-145`). |
| 5. `make check` y CI en verde | ✅ | `gofmt`, `go vet`, `go test -race -count=1 ./...` y 6/6 jobs del CI en verde (run `44974678960`). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Alineado con `ADR-0007` y `AGENTS.md` §3.2 (100 % stdlib). |
| Implementación | 8 | Base sólida para TB2 | JWT HS256 estricto, PBKDF2-SHA256 acotado y middlewares inmutables/sin carreras listos para #60 y #61. |
| Interfaz | 5 | Habilitador | Define el contrato de `Authorization: Bearer`, `401` (`WWW-Authenticate`) y `403` que consumirá la SPA. |
| Informe | 2 | Aporta evidencia | Cobertura 94,1 %, cero dependencias externas y pruebas deterministas con `-race`. |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y probados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | Base `integracion/tb2` @ `b405959` integrada en `238fc90`; 6/6 jobs del CI en verde (run `44974678960`). |
| H2 (ronda 1 · MAYOR) | RESUELTO | `3744ac2` (`jwt.go:60, 107-109, 117, 124`): rechaza `\r`/`\n`, usa `base64.RawURLEncoding.Strict()` en las tres partes y añade casos de bits sobrantes y saltos de línea en `TestJWTRechazos` (`jwt_test.go:95-97, 121-125`). |
| H3 (ronda 1 · MAYOR) | RESUELTO | `3744ac2` (`jwt.go:62-63, 114-116, 147-161`): exige `partes[0] == cabeceraB64` y clasifica diferencias vía `errorDeCabecera` (`alg` distinto → `ErrAlgoritmo`; `typ`, `ALG`, `crit`, `null` → `ErrMalformado`), cubierto en `jwt_test.go:98-101`. |
| H4 (ronda 1 · MAYOR) | RESUELTO | `aa88478` (`jwt.go:78-81`): `NuevoFirmador` exige `duracion >= time.Second`; probado en `jwt_test.go:131-144`. |
| H5 (ronda 1 · MAYOR) | RESUELTO | `860e58a` (`clave.go:25, 46-68`): `Comparar` rechaza `\r`/`\n`, usa `base64.RawStdEncoding.Strict()`, exige `len(sal) == 16` y `len(esperado) == 32` y fija `bytesClave` en `pbkdf2.Key`; cubierto en `clave_test.go:41-45, 58-74`. |
| H6 (ronda 1 · MENOR) | RESUELTO | `6429a6f` (`middleware.go:45, 53`, `middleware_test.go:37-76, 93-140`): `Requiere` clona `roles` (`slices.Clone`), `TestMiddleware` valida `sub`, `rol` y `error="invalid_token"` en vencido, y `TestMiddlewareConcurrente` prueba 128 goroutines mixtas validando estado HTTP y reclamos. |

## Cómo explicarlo en la sustentación

`internal/auth` concentra la criptografía y el control de acceso usando solo la biblioteca estándar de Go 1.24: `Firmador` es inmutable entre goroutines y emite/verifica JWT HS256 con `base64.RawURLEncoding.Strict()`, cabecera fija exigida byte a byte y firma en tiempo constante (`hmac.Equal`), rechazando `alg: none`, tokens vencidos o manipulados; `clave.go` deriva contraseñas con `crypto/pbkdf2` (600 000 iteraciones, sal de 16 bytes y clave fija de 32 bytes) y usa un hash señuelo (`sync.OnceValue`) para no filtrar correos existentes; y los middlewares `Autenticar` y `Requiere` separan autenticación (`401` con `WWW-Authenticate`) de autorización por rol (`403`) propagando los reclamos en `context.Context` sin estado mutable compartido.
