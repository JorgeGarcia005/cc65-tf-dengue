# Veredicto — PR-80 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/60-login` @ `032dc1f` (base: `integracion/tb2` @ `0f9c0f5`)
- **Fecha:** 2026-11-22
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H4**) quedaron resueltos con evidencia verificable en el commit `b08d7d0` sin introducir regresiones ni dependencias de terceros en el núcleo (`AGENTS.md` §3.2): `limitador.Reservar(ip)` cierra la ventana TOCTOU contabilizando atómicamente `fallos + enCurso` bajo `l.mu` con liberación `sync.Once` vía `defer` (validado en `TestLoginRafagaConcurrente` y `TestLimitadorCuentaIntentosEnCurso`, además de reforzar `TestLoginRateLimit` para matar el mutante sin `intentoExitoso`); la poda del limitador corre a lo sumo una vez cada media ventana con tope duro `maxIPsLimitador = 8192` (`TestLimitadorMemoriaAcotada`); `SembrarUsuarios` aplica los mismos topes en bytes que `login` (`MaxBytesCorreo = 254`, `MaxBytesClave = 256`) y rechaza correos normalizados repetidos entre roles; y el hash señuelo se precalcula al crear el `Servidor`, `deploy/.env.example` usa `RandomNumberGenerator`, el 503 distingue falta de almacén de usuarios y los comentarios de paquete están actualizados. `gofmt`, `go vet` (estándar e integración), `go test -race -count=1 ./...` completo (25/25 paquetes con tests, 0 carreras) y los 7/7 jobs del CI en GitHub sobre `032dc1f` están en verde.

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

$ bash scripts/check-skills-sync.sh
skills: ok

$ git diff --check origin/integracion/tb2...HEAD -- ':!docs/agentes/revisiones/*'
(vacío = ok)

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/almacen ./internal/api ./cmd/usuarios
github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen -> [context errors fmt .../cluster .../features strconv strings sync time]
github.com/JorgeGarcia005/cc65-tf-dengue/internal/api -> [context crypto/rand crypto/sha256 encoding/hex encoding/json errors fmt .../docs/api .../almacen .../auth .../cluster .../features .../ingest .../ml io log math net net/http runtime/debug strconv strings sync sync/atomic time unicode/utf8]
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios -> [context flag fmt .../almacen .../almacen/mongo .../api io log os time]

$ go test -race -count=3 -v -run 'Login|Limitador|Sembrar|IPCliente' ./internal/api
PASS (3/3 repeticiones: TestLoginCorrectoDevuelveTokenVerificable, TestLoginCredencialesIncorrectasMismoMensaje,
TestLoginRateLimit, TestLoginRafagaConcurrente, TestLoginSinUsuariosDa503, TestLoginSinHashEnRespuestasNiLogs,
TestLoginValidacion, TestLoginSinFirmadorDa503, TestLimitadorPorIP, TestLimitadorCuentaIntentosEnCurso,
TestLimitadorMemoriaAcotada, TestIPCliente, TestSembrarUsuarios)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	62.675s

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	3.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	7.219s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.178s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.966s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.463s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.268s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.487s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	4.091s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.082s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.205s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.035s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.150s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	24.904s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	16.911s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.542s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.788s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.052s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.426s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.036s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.109s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.880s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.036s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.226s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.750s

$ gh pr checks 80   # run 44979562107 sobre 032dc1f
Docker (imagen, clúster y caída de un nodo)	pass	1m55s
Ecosistema de agentes                       	pass	11s
Go (gofmt, vet, test -race)                 	pass	1m1s
MongoDB y Redis (tests de integración)      	pass	48s
OpenAPI 3 válido                            	pass	20s
SPA (tipos, tests, build e imagen)          	pass	56s
Spin (modelo del protocolo y mutantes)      	pass	1m57s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes con tests PASS) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; suite completa en verde) |
| Latencia API p95 < 100 ms | no aplica (`POST /api/v1/predicciones` sin cambios; p95 con JWT se mide en #61) |
| CI GitHub Actions sobre `032dc1f` | ✅ (7/7 jobs PASS, incl. `bd` MongoDB/Redis y `OpenAPI 3 válido`) |

## Verificación contra los criterios del issue #60

| Criterio (#60) | Estado | Evidencia |
|---|---|---|
| 1. Repositorio `usuarios` en memoria y en Mongo con índice único `correo` | ✅ | `internal/almacen/usuarios.go`, `internal/almacen/mongo/mongo.go` (`_id` = correo normalizado), `internal/almacen/contrato/contrato.go` (probado en CI `MongoDB y Redis` en verde). |
| 2. `POST /api/v1/auth/login`: 200 + token; clave errada y correo inexistente → mismo 401 con el mismo mensaje | ✅ | `internal/api/login.go:75-94`, `TestLoginCorrectoDevuelveTokenVerificable` y `TestLoginCredencialesIncorrectasMismoMensaje` PASS. |
| 3. 6.º intento fallido en un minuto desde la misma IP → 429 con `Retry-After` | ✅ | `internal/api/limite.go:73-103`, `TestLoginRateLimit` (exactamente 55 s y borrado tras login exitoso) y `TestLoginRafagaConcurrente` (20 simultáneos → 5 × 401 y 15 × 429 con `Retry-After`). |
| 4. Ninguna respuesta ni log contiene `hash` | ✅ | `Usuario.Hash` con `json:"-"`, `TestLoginSinHashEnRespuestasNiLogs` y `TestUsuarioNoSerializaElHash` PASS. |
| 5. `cmd/usuarios` crea admin y analista desde variables de entorno (sin claves en el repo) | ✅ | `cmd/usuarios/main.go`, `go run ./cmd/usuarios -h`, `go test -race ./cmd/usuarios` PASS, `deploy/.env.example` con claves vacías. |
| 6. OpenAPI documenta la ruta | ✅ | `docs/api/openapi.yaml`, `TestOpenAPICubreLasRutas` PASS y job `OpenAPI 3 válido` en verde. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | Mantiene 3 / 3 | Alineado con `ADR-0006` y `ADR-0007`. |
| Implementación | 8 | Avanza seguridad de API | Colección `usuarios` en Mongo/memoria, login JWT HS256 con hash señuelo PBKDF2, rate limit atómico y acotado en memoria sin TOCTOU, siembra sin secretos en el repo y OpenAPI/Postman completos. |
| Interfaz | 5 | Coherente con la SPA | Contrato `{token, correo, rol, expira}` y cabecera `Retry-After` en `429` alineados con `web/src/api/cliente.ts` y `web/src/sesion/Sesion.tsx`. |
| Informe | 2 | sin cambios | Evidencia de seguridad y latencia con JWT en #61 y #70. |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron resueltos y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · MAYOR) | RESUELTO | `b08d7d0` (`internal/api/limite.go:73-103`, `internal/api/login.go:49-57`, `internal/api/login_test.go:133-179, 256-277`): `limitador.Reservar(ip)` verifica y reserva `len(f)+enCurso[ip]` atómicamente bajo `l.mu`, con liberación `sync.Once` en `defer` (también ante pánico) y `Retry-After: 1` cuando los cupos están en curso. `TestLoginRafagaConcurrente` (×3 con `-race`) verifica exactamente 5 × 401 y 15 × 429 en 20 peticiones simultáneas; `TestLoginRateLimit` ahora prueba que el borrado en `intentoExitoso` evita el 429 ante nuevos fallos. |
| H2 (ronda 1 · MAYOR) | RESUELTO | `b08d7d0` (`internal/api/limite.go:16-17, 105-122`, `internal/api/login_test.go:279-301`): la poda global corre a lo sumo cada `ventana/2` (`ultimaPoda`) y el mapa aplica tope duro `maxIPsLimitador = 8192` con desalojo $O(1)$ para IPs nuevas; cubierto por `TestLimitadorMemoriaAcotada`. |
| H3 (ronda 1 · MAYOR) | RESUELTO | `b08d7d0` (`internal/api/login.go:17-20, 63-65`, `internal/api/usuarios.go:30-46`, `internal/api/login_test.go:338-343`): `SembrarUsuarios` valida `MaxBytesCorreo` (254) y `MaxBytesClave` (256) en bytes —coherente con `login` y su mensaje 400— y rechaza correos normalizados repetidos entre roles (`vistos`); cubierto en `TestSembrarUsuarios` (257 bytes, 150 «ñ» = 300 bytes, correo > 254 bytes y correo repetido). |
| H4 (ronda 1 · MENOR) | RESUELTO | `b08d7d0` (`internal/api/api.go:8, 130-134`, `internal/api/login.go:41-48`, `deploy/.env.example:6-7`, `internal/almacen/mongo/mongo.go:1-4`, `internal/api/login_test.go:181-190`): `Nuevo` precalienta `auth.HashSenuelo()` (que usa `sync.OnceValue` en memoria y termina tras una derivación), `deploy/.env.example` usa `RandomNumberGenerator`, el 503 diferencia ausencia de almacén (`TestLoginSinUsuariosDa503`) y los comentarios de paquete en `api.go` y `mongo.go` quedan actualizados. |

## Cómo explicarlo en la sustentación

Las cuentas de la SPA se guardan en la colección `usuarios` de MongoDB usando el correo normalizado como `_id` único y un hash PBKDF2-HMAC-SHA256 de 600 000 iteraciones etiquetado con `json:"-"` para que nunca se serialice; las claves se siembran únicamente por variables de entorno (`cmd/usuarios` o el coordinador al arrancar) y nunca entran al repositorio. Al recibir `POST /api/v1/auth/login`, el limitador por IP (`RemoteAddr`, ignorando `X-Forwarded-For`) reserva atómicamente el cupo bajo mutex contando fallos vigentes más intentos en curso para impedir ráfagas concurrentes mientras corre PBKDF2, mantiene memoria acotada con poda espaciada y tope duro, compara siempre contra el hash real o un hash señuelo precalculado para responder el mismo `401` sin revelar correos registrados, y emite el JWT HS256 de 60 minutos con `{token, correo, rol, expira}`.
