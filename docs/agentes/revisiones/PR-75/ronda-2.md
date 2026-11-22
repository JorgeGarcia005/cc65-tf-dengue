# Veredicto — PR-75 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/57-adr-seguridad` @ `85511b9` (base: `integracion/tb2` @ `b405959`)
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H5**) fueron resueltos con evidencia verificable. La base `integracion/tb2` ya incluye el PR #74 (`b405959`), los 6 jobs del CI en GitHub pasan en verde sobre `85511b9` (run `44974675104`), y `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md` (actualizado en `5235430`) cierra sin ambigüedades la política CORS (`Access-Control-Expose-Headers`, `Vary: Origin`, preflight `OPTIONS` y orden de la cadena de middlewares), el ciclo de vida de conexiones secuestradas en `internal/ws` (`http.NewResponseController(w).Hijack()`, `SetDeadline(time.Time{})`, lectora + escritora única, cierre con `1001` en `SIGTERM` y declinación de `permessage-deflate` conforme a RFC 6455 §9.1), los tres modos de autenticación por `Ruta` (`publica`, `bearer`, `ticket`) y el orden de comprobaciones en `GET /api/v1/ws`. La verificación local completa (`gofmt`, `go vet`, `go test -race -count=1 ./...`, `scripts/check-skills-sync.sh`) pasa en verde con 0 carreras y sin hallazgos nuevos.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh && wc -m AGENTS.md
skills: ok
8020 AGENTS.md

$ grep -n ADR-0007 docs/decisiones/ADR-0006*
37:| `usuarios` (TB2) | `correo` | `hash` (PBKDF2-SHA256, ver ADR-0007), `rol` (`admin`/`analista`), `creado` | Único por `_id`. Se implementa con JWT en TB2 (ADR-0007) |

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.362s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.447s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.479s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.272s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.212s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.411s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.141s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.023s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.474s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.896s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.191s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.166s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.046s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.074s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.923s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.251s

$ gh pr checks 75   # run 44974675104 sobre 85511b9
Docker (imagen, clúster y caída de un nodo)	pass	1m59s
Ecosistema de agentes                       	pass	10s
Go (gofmt, vet, test -race)                 	pass	52s
MongoDB y Redis (tests de integración)      	pass	53s
OpenAPI 3 válido                            	pass	21s
Spin (modelo del protocolo y mutantes)      	pass	1m51s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 20/20 paquetes con tests en verde sobre Go 1.24.3) |
| Equivalencia secuencial = distribuido | no aplica (solo documentación; la suite completa de `internal/ml` e `internal/cluster` pasa con `-race`) |
| Latencia API p95 < 100 ms | no aplica (solo ADR; la remedición con JWT queda en #61) |
| CI de GitHub en el PR #75 | ✅ (6/6 jobs en verde, run `44974675104`) |

## Verificación contra los criterios del issue #57

| Criterio (#57) | Estado | Evidencia |
|---|---|---|
| 1. `ADR-0007` con contexto, opciones, decisión y consecuencias (JWT HS256 stdlib, PBKDF2 `crypto/pbkdf2`, roles admin/analista, CORS lista blanca, ticket WS + `Origin`, WebSocket RFC 6455 stdlib vs SSE, imagen `web` aparte) | ✅ | `docs/decisiones/ADR-0007-seguridad-y-tiempo-real.md:23-37, 60-80` cubre todos los temas, incluyendo CORS completo, cadena de middlewares, `Hijack` + `SetDeadline` + cierre `1001`, declinación de extensiones RFC 6455 §9.1 y ticket atómico. |
| 2. Matriz ruta × rol completa (todas las rutas de `Rutas()` más las nuevas) | ✅ | `ADR-0007:41-55` cubre las 8 rutas de `internal/api/api.go:240-250` y las 6 rutas TB2, con modos de autenticación (`publica`, `bearer`, `ticket`) coherentes con `:30` y `:35`. |
| 3. Enlazado desde `docs/decisiones/` y ADR-0006 (colección `usuarios`) | ✅ | `docs/decisiones/ADR-0006-entidades-mongo-redis.md:37` enlaza a ADR-0007. |
| Dependencia `#56 (T0)` y CI en verde | ✅ | Base `integracion/tb2` @ `b405959` integrada en `85511b9`; 6/6 checks en verde. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Decisiones de seguridad y tiempo real documentadas con opciones descartadas y consecuencias, alineadas con `AGENTS.md` §3.2, §3.3 y §3.6. |
| Implementación | 8 | Contrato completo TB2 | Deja un contrato sin contradicciones para #58, #60–#65 y #67–#70. |
| Interfaz | 5 | Contrato completo de integración | Origen separado `:8081` / `:8080`, `Access-Control-Expose-Headers` (`X-Cache`, `Retry-After`, `Location`) y CSP con `connect-src` dinámico. |
| Informe | 2 | Trazabilidad lista | Sustenta la sección de integración segura del informe final (#71). |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron cerrados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | PR #74 fusionado en `integracion/tb2` (`b405959`), rama actualizada en `85511b9` y 6/6 jobs del CI en verde (run `44974675104`). |
| H2 (ronda 1 · MAYOR) | RESUELTO | `5235430` (`ADR-0007:32-33`): añade `Access-Control-Expose-Headers: X-Cache, Retry-After, Location`, `Vary: Origin` en 2xx/4xx/5xx, nombres normativos `Access-Control-Allow-*`/`Max-Age`, `OPTIONS` no permitido → 204 sin cabeceras CORS antes del `ServeMux`, y el orden exacto de middlewares en `Handler()`. |
| H3 (ronda 1 · MAYOR) | RESUELTO | `5235430` (`ADR-0007:34, 68-78`): fija `http.NewResponseController(w).Hijack()`, `SetDeadline(time.Time{})`, lectura desde el `bufio.Reader` del hijack, 1 lectora + 1 única escritora con plazo de 10 s, cierre del hub con `1001` en `Servidor.Cerrar()` y declinación de `permessage-deflate` omitiendo `Sec-WebSocket-Extensions` en el `101` (RFC 6455 §9.1). |
| H4 (ronda 1 · MAYOR) | RESUELTO | `5235430` (`ADR-0007:30, 35, 54`): define los modos `publica`, `bearer` y `ticket` por `Ruta`, el test tabla por modo y el orden estricto en `GET /api/v1/ws` (`Origin` → 403 antes de tocar tickets; consumo atómico bajo mutex → 401; rol ≠ admin → 403). |
| H5 (ronda 1 · MENOR) | RESUELTO | `5235430` (`ADR-0007:5, 25, 28, 29, 37`) y descripción del PR: issues exactos, `hmac.Equal` + base64url estricto, normalización de correo, IP desde `RemoteAddr` + ventana deslizante de 1 min sobre fallos, `connect-src` desde `API_URL` y marcador `PR-75` actualizado. |

## Cómo explicarlo en la sustentación

El sistema separa dos roles —analista de salud pública para consultar riesgo, evaluación y predicciones, y administrador para operar el clúster— y protege la API REST solo con la biblioteca estándar de Go (`crypto/hmac`, `crypto/sha256` y `crypto/pbkdf2` de Go 1.24), declarando por cada ruta su modo de autenticación (`publica`, `bearer` o `ticket`), una lista blanca estricta de CORS con `Access-Control-Expose-Headers` entre la SPA (`:8081`) y la API (`:8080`), y el orden exacto de middlewares para que el preflight `OPTIONS` y los errores `401`/`403`/`429` funcionen en el navegador. Para el entrenamiento en vivo se implementa un servidor WebSocket RFC 6455 propio sobre `http.NewResponseController(w).Hijack()`, limpiando los plazos heredados de `http.Server`, validando `Origin` antes de consumir un ticket atómico de 30 s, cerrando las conexiones con `1001` en `SIGTERM` y publicando por canales con búfer sin bloqueo para no frenar la barrera del clúster ni alterar los pesos bit a bit.
