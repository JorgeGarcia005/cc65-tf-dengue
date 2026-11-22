# Veredicto — PR-83 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/61-roles-cors` @ `e9495ae` (base: `integracion/tb2` @ `4e1842c`)
- **Fecha:** 2026-11-22
- **Decisión:** CAMBIOS

El PR implementa la autorización por rol generada desde `Rutas()` (`internal/api/api.go`, `internal/api/seguridad.go`), con prueba automática bidireccional contra la tabla de `ADR-0007` (`TestMatrizRoles`, `TestMatrizRolesCoincideConADR`), fallo cerrado con 503 sin firmador y arranque bloqueado en modo servicio sin `JWT_SECRETO` (`cmd/coordinator/seguridad.go`), middleware CORS con lista blanca y preflight antes de la autenticación, cabeceras de seguridad (`nosniff`, `no-referrer`, `Cache-Control: no-store` por defecto preservando `private, no-cache` en `riesgo`/`evaluacion`), OpenAPI con `bearerAuth` y respuestas 401/403 (`TestOpenAPISeguridad`), colección Postman actualizada y soporte JWT en `cmd/benchapi`. Todo el núcleo de API/benchmark usa solo la biblioteca estándar (`AGENTS.md` §3.2), `gofmt`, `go vet`, los 25 paquetes Go con `-race` (0 carreras) y los 7/7 jobs de GitHub Actions están en verde. Sin embargo, se emiten **1 hallazgo bloqueante**, **2 mayores** y **1 menor** antes de aprobar: falta generar y versionar `reports/bench_api_<maquina>_jwt.json` exigido por el criterio 5 del issue #61 (`Cierra #61`) y `AGENTS.md` §3.5–§3.6 (**H1**); un `POST` de > 64 KiB con cuerpo no-JSON (como el comando de `scripts/evidencia-api.sh:62-63`) devuelve `400` en lugar de `413` y `docs/api/openapi.yaml` omite `413` en `POST /api/v1/entrenamientos` (**H2**); `normalizarOrigen` acepta orígenes malformados (`http://:8081`, `http://*.example.com`, `http://localhost:`, `http://localhost:99999`) y `Rutas()` expone sin clonar los slices mutables `rolesTodos`/`soloAdmin` (**H3**); y conviene completar la cobertura de tests de seguridad/OpenAPI, limpiar el CORS falso de `TestEvaluacionConservaVaryDeCORS` y aclarar la precedencia `-token` vs `-correo` en `cmd/benchapi` (**H4**).

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

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	21.313s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	14.422s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	11.406s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	2.701s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.651s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.259s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.415s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.412s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.046s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.159s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.022s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.066s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	26.774s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.960s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.997s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.944s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.043s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.966s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.068s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.963s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.302s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.752s

$ go test -race -count=1 -v -run 'MatrizRoles|CORS|Cabeceras|OpenAPI|ServicioExigeToken|PrepararSeguridad|MedirConLogin' ./internal/api ./cmd/coordinator ./cmd/benchapi
PASS (16/16: TestOpenAPICubreLasRutas, TestOpenAPIMapasEnLineaBienFormados, TestEvaluacionConservaVaryDeCORS,
TestMatrizRoles [11 rutas × 3 roles], TestMatrizRolesTokenAjenoYSinFirmador, TestCORSPreflightOrigenPermitido,
TestCORSPreflightOrigenNoPermitido, TestCORSRespuestasDecoradas, TestCORSOrigenesValidos, TestCabecerasSeguridad,
TestCabecerasLimiteDeCuerpo413, TestOpenAPISeguridad, TestMatrizRolesCoincideConADR,
TestServicioExigeTokenYAplicaCORS, TestPrepararSeguridad, TestMedirConLoginUsaJWT)

$ go test -race -count=1 -run 'Fragmentos' -v ./cmd/informe
--- PASS: TestFragmentosSalenDelFuente (0.15s)

$ go run ./cmd/benchapi -h
  -correo string  inicia sesión con este correo y la clave de $BENCH_CLAVE (por defecto $BENCH_CORREO)
  -salida string  reporte (por defecto reports/bench_api_<maquina>[_jwt].json)
  -token string   JWT para Authorization: Bearer (por defecto $BENCH_TOKEN)
  (resto de flags: -c, -calientes, -exigir-p95, -maquina, -n, -semilla, -timeout, -url)

$ API=:8080 docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml config
CORS_ORIGENES: http://localhost:8081  ·  API_URL: http://localhost:8080  (interpolación válida)

$ <prueba en vivo contra binario cmd/coordinator -api 127.0.0.1:18089 con JWT en memoria>
- Sin JWT_SECRETO: rechaza arrancar («JWT_SECRETO es obligatorio en modo servicio…»)
- POST 70 KB de 'a' (comando exacto de scripts/evidencia-api.sh:62-63): HTTP/1.1 400 Bad Request {"error":"JSON mal formado"}  (ver H2)
- -cors http://:8081 | http://*.example.com | http://localhost: | http://localhost:99999: arrancó sin rechazar (ver H3)
- BENCH_CLAVE=… go run ./cmd/benchapi -url http://127.0.0.1:18089 -correo analista@vigia.pe -n 500 -c 8 -exigir-p95 100ms:
  fria: 500 pet., 0 err., p95=3.21 ms (p95 < 100 ms: true) · caliente: 500 pet., 0 err., p95=2.80 ms (p95 < 100 ms: true)

$ gh pr checks 83   # run 44989016430 sobre e9495ae
Docker (imagen, clúster y caída de un nodo)	pass	1m55s
Ecosistema de agentes                       	pass	4s
Go (gofmt, vet, test -race)                 	pass	1m6s
MongoDB y Redis (tests de integración)      	pass	54s
OpenAPI 3 válido                            	pass	19s
SPA (tipos, tests, build e imagen)          	pass	53s
Spin (modelo del protocolo y mutantes)      	pass	1m51s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes con tests PASS) |
| CI GitHub Actions (7 jobs, incl. Mongo/Redis e imagen Docker) | ✅ (run `44989016430`) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; suite completa PASS) |
| Latencia API p95 < 100 ms con JWT | ⚠️ Código y prueba sintética en memoria pasan (`p95` 3,21 ms / 2,80 ms), pero **falta `reports/bench_api_<maquina>_jwt.json`** medido contra Compose real (**H1**) |

## Verificación contra los criterios del issue #61

| Criterio (#61) | Estado | Evidencia |
|---|---|---|
| 1. Test tabla desde `Rutas()`: ruta × {sin token, analista, admin} → 401/403/2xx según ADR-0007 | ✅ | `TestMatrizRoles` (11 rutas × 3 roles), `TestMatrizRolesTokenAjenoYSinFirmador` (401 + 503 sin firmador) y `TestMatrizRolesCoincideConADR` PASS. *(Detalle de inmutabilidad en **H3**.)* |
| 2. Preflight origen permitido → 204 con `Access-Control-Allow-*` y `Vary: Origin`; ajeno → sin `Access-Control-Allow-Origin` | ✅ | `TestCORSPreflightOrigenPermitido`, `TestCORSPreflightOrigenNoPermitido`, `TestCORSRespuestasDecoradas` PASS. *(Validación de orígenes malformados en **H3** y cobertura 429/500 en **H4**.)* |
| 3. `X-Content-Type-Options`, `Referrer-Policy` y límite de cuerpo en POST (413) | ⚠️ | `TestCabecerasSeguridad` y `TestCabecerasLimiteDeCuerpo413` pasan con JSON válido largo, pero un POST de > 64 KiB no-JSON (como `scripts/evidencia-api.sh:62-63`) responde `400` en vez de `413` (**H2**). |
| 4. OpenAPI declara `bearerAuth` y 401/403 en toda ruta protegida | ✅ | `TestOpenAPISeguridad` y job CI `OpenAPI 3 válido` PASS. *(Falta `"413"` en `POST /api/v1/entrenamientos`, **H2**.)* |
| 5. `cmd/benchapi` con token → `reports/bench_api_<maquina>_jwt.json` con p95 < 100 ms y 0 errores | ❌ | `cmd/benchapi -h` y `TestMedirConLoginUsaJWT` pasan, pero el archivo `reports/bench_api_<maquina>_jwt.json` no está generado ni versionado (**H1**). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | Alineado con `ADR-0007` (actualizado con el orden real de la cadena en `Handler()`). |
| Implementación | 8 | Parcial (pendiente **H1–H3**) | Autorización por rol, CORS, cabeceras y `JWT_SECRETO` obligatorio en stdlib con 0 carreras, pero falta el reporte real de p95 con JWT (**H1**), el límite 413 falla en payloads no-JSON y rompe `evidencia-api.sh` (**H2**), y hay huecos de validación/inmutabilidad en `normalizarOrigen`/`Rutas()` (**H3**). |
| Interfaz | 5 | Habilita la integración real de la SPA | CORS expone `X-Cache`, `Retry-After` y `Location` y permite `Authorization` y `Content-Type` tal como consume `web/src/api/cliente.ts` (`credentials: 'omit'`). |
| Informe | 2 | Parcial (bloqueado por **H1**) | `docs/informe/pc4/10-api.md` queda sincronizado (`TestFragmentosSalenDelFuente` ✅), pero falta `reports/bench_api_<maquina>_jwt.json` para respaldar el p95 con JWT sin cifras a mano (`AGENTS.md` §3.6). |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

### H1 — Falta generar y versionar `reports/bench_api_<maquina>_jwt.json` (criterio #5 de #61) · BLOQUEANTE

- **Dónde:** `reports/bench_api_*_jwt.json` (ausente en `feature/61-roles-cors` @ `e9495ae`), issue #61 (criterio 5) y descripción del PR #83 (`Cierra #61`)
- **Evidencia:**
  1. El criterio 5 del issue #61 exige: *«`cmd/benchapi` con token → `reports/bench_api_<maquina>_jwt.json` con p95 < 100 ms y 0 errores — verificar con: `go run ./cmd/benchapi -h` y el json»*.
  2. `ls reports/bench_api_*.json` solo contiene `reports/bench_api_deez.json` (PC4, sin JWT); `git diff --stat origin/integracion/tb2...HEAD` no añade ningún `reports/bench_api_*_jwt.json`.
  3. La descripción del PR deja abierta la casilla `- [ ]` (*«Pendiente de la medición en la máquina de Jorge con el Compose (Mongo + Redis), igual que en #81»*) pese a declarar `Cierra #61`. Como en PR-81 H1 (`reports/bench_observador_*.json`), no se puede aprobar el cierre del issue sin la evidencia medida y versionada (`AGENTS.md` §3.5 y §3.6).
  4. *Aviso preventivo:* `cmd/informe/regiones.go:307-315` (`apiReal()`) hace `filepath.Glob("reports/bench_api_*.json")`, ordena y lee `archivos[0]`. Si el hostname de la medición ordenase antes que `deez` alfabéticamente, `TestFragmentosSalenDelFuente` en `cmd/informe` compararía `docs/informe/pc4/10-api.md` contra el archivo nuevo en vez de `bench_api_deez.json`. Conviene verificar `go test -race ./cmd/informe` tras añadir el JSON (o excluir `*_jwt.json` en `apiReal()` si `pc4/10-api.md` debe seguir citando el reporte PC4).
- **Corrección sugerida:** Tras aplicar H2–H4, ejecutar `scripts/evidencia-api.sh` (o `BENCH_CLAVE=… go run ./cmd/benchapi -url http://localhost:8080 -correo analista@vigia.pe -exigir-p95 100ms`) contra el despliegue Compose real con MongoDB y Redis, versionar `reports/bench_api_<maquina>_jwt.json` (verificando `autenticacion: "jwt"`, `errores: 0`, `con_cache: true` y `p95_ms < 100` en `fria` y `caliente`), comprobar que `go test -race ./...` sigue en verde y marcar la casilla del PR.

### H2 — Un `POST` de > 64 KiB con cuerpo no-JSON devuelve `400` en vez de `413`, rompe `scripts/evidencia-api.sh:62-63` y falta `"413"` en OpenAPI para `/api/v1/entrenamientos` · MAYOR

- **Dónde:** `internal/api/api.go:572-619` (`leerJSON`), `internal/api/seguridad.go:83-93` (`cabecerasSeguridad`), `scripts/evidencia-api.sh:62-63`, `internal/api/seguridad_test.go:294-324` y `docs/api/openapi.yaml:205-218`
- **Evidencia:**
  1. `scripts/evidencia-api.sh:62-63` comprueba el límite de 64 KiB enviando 70 000 bytes `'a'` con `--data-binary @-` y anuncia `(quiere 413)`:
     ```bash
     echo "POST de 70 KB a /api/v1/predicciones → $(head -c 70000 /dev/zero | tr '\0' 'a' |
       codigo -X POST -H "Authorization: Bearer $analista" -H 'Content-Type: application/json' --data-binary @- "$url/api/v1/predicciones") (quiere 413)"
     ```
  2. Reproduciendo exactamente ese comando contra el binario compilado `cmd/coordinator` en modo servicio con un token de analista válido, la respuesta fue **`400 Bad Request`**, no `413`:
     ```text
     HTTP/1.1 400 Bad Request
     Cache-Control: no-store
     Content-Type: application/json; charset=utf-8
     ...
     {"error":"JSON mal formado"}
     ```
  3. Causa raíz: `cabecerasSeguridad` (`seguridad.go:88-90`) solo envuelve `r.Body = http.MaxBytesReader(w, r.Body, MaxCuerpo)` y `leerJSON` (`api.go:573-575`) vuelve a envolver `r.Body` en `http.MaxBytesReader` y decodifica con `json.NewDecoder`. Cuando el cuerpo no empieza como JSON válido, `json.NewDecoder` lee únicamente el primer bloque (512 bytes), falla en el byte 0 con `*json.SyntaxError` y nunca alcanza el byte 65 536 para disparar `*http.MaxBytesError`. `TestCabecerasLimiteDeCuerpo413` (`seguridad_test.go:309`) no lo detectó porque únicamente envió un objeto JSON sintácticamente válido al inicio (`{"x":"aaaa..."}`).
  4. Además, `docs/api/openapi.yaml:18` afirma *«Todo POST admite como máximo 64 KiB (413)»* y `TestCabecerasLimiteDeCuerpo413` (`seguridad_test.go:312`) exige `413` en `POST /api/v1/entrenamientos`, pero `docs/api/openapi.yaml:205-218` omite `"413"` en esa operación (sí está declarado en `/api/v1/auth/login:68` y `/api/v1/predicciones:127`).
- **Corrección sugerida:**
  - En `leerJSON` (`internal/api/api.go:572`), comprobar `if r.ContentLength > MaxCuerpo` antes de decodificar y responder inmediatamente `413` (`el cuerpo supera 65536 bytes`) sin leer bytes del cuerpo (así se conservan el orden 401/403 antes de 413 en rutas protegidas, las cabeceras de seguridad y CORS, y `scripts/evidencia-api.sh:62-63` devolverá `413`).
  - Añadir en `TestCabecerasLimiteDeCuerpo413` un caso con cuerpo no-JSON de > 64 KiB y verificar que la respuesta `413` conserva `Access-Control-Allow-Origin` para un origen permitido.
  - Añadir `"413": {$ref: "#/components/responses/Error"}` en `POST /api/v1/entrenamientos` (`docs/api/openapi.yaml:205-218`).

### H3 — `normalizarOrigen` acepta orígenes CORS malformados y `Rutas()` expone sin copiar los slices mutables `rolesTodos` / `soloAdmin` · MAYOR

- **Dónde:** `internal/api/seguridad.go:31-34, 66-79` y `internal/api/api.go:288-308`
- **Evidencia:**
  1. En `internal/api/seguridad.go:73`, la validación de host comprueba `u.Host == "" || u.Hostname() == "*" || u.User != nil`, lo que deja pasar orígenes inválidos que no coincidirán nunca con un origen real (o inducen a pensar que hay comodines). Probado arrancando el binario `cmd/coordinator -cors <valor>` con `JWT_SECRETO` válido:
     - `http://:8081` (`u.Host == ":8081"`, `u.Hostname() == ""`) → **aceptado**, el coordinador arrancó (`exit=124` por timeout de espera de nodos).
     - `http://*.example.com` (`u.Hostname() == "*.example.com"` ≠ `"*"`) → **aceptado**, el coordinador arrancó.
     - `http://localhost:` (dos puntos sin puerto) → **aceptado**, el coordinador arrancó.
     - `http://localhost:99999` (puerto fuera de 1–65535) → **aceptado**, el coordinador arrancó.
     Un error de despliegue como `CORS_ORIGENES=http://${HOST}:8081` con `HOST` vacío o un comodín `http://*.vigia.pe` pasaría silenciosamente el arranque en vez de fallar rápido.
  2. En `internal/api/api.go:292-304`, la función exportada `Rutas()` devuelve `Ruta{..., Roles: roles}` referenciando directamente los slices globales `rolesTodos` y `soloAdmin` (`seguridad.go:32-33`) sin `slices.Clone(roles)`. Si un llamador externo muta `r.Roles`, corrompe la matriz global para futuras llamadas a `Handler()` / `Rutas()` (y genera una condición de carrera si concurre con `Handler()`), rompiendo el criterio defensivo que el propio proyecto aplica en `auth.Requiere` (`internal/auth/middleware.go:45`) y en `api.Nuevo` (`internal/api/api.go:148`).
- **Corrección sugerida:**
  - En `normalizarOrigen` (`seguridad.go:66-79`), rechazar `u.Hostname() == ""`, `strings.Contains(u.Hostname(), "*")`, `strings.HasSuffix(u.Host, ":")` y validar que si `p := u.Port(); p != ""` el número esté en `1..65535` (y añadir esos casos a `TestCORSOrigenesValidos`).
  - En `Rutas()` (`api.go:293`), usar `Roles: slices.Clone(roles)` (y añadir una aserción de que mutar el slice devuelto por `Rutas()` no altera una llamada posterior).

### H4 — Cobertura puntual de tests de seguridad/OpenAPI, limpieza de `TestEvaluacionConservaVaryDeCORS` y precedencia `-token` / `-correo` en `cmd/benchapi` · MENOR

- **Dónde:** `internal/api/seguridad_test.go:224-292, 336-382`, `internal/api/evaluacion_test.go:80-94` y `cmd/benchapi/main.go:156-166`
- **Evidencia:**
  1. `ADR-0007` (fila CORS) exige expresamente que también `429` y `5xx` lleven `Access-Control-Allow-Origin`, `Vary: Origin` y `Access-Control-Expose-Headers` (imprescindible para que `web/src/api/cliente.ts:75` lea `Retry-After` en un `429` de login), y que `/api/v1/riesgo` y `/api/v1/evaluacion` usen `Cache-Control: private, no-cache` en lugar de `no-store`. `TestCORSRespuestasDecoradas` (`seguridad_test.go:224-247`) no cubre `429` ni `500` (pánico recuperado por `recuperar`), y `TestCabecerasSeguridad` (`:268-292`) no verifica `riesgo`/`evaluacion` (`private, no-cache` + ambos valores de `Vary`) ni `login` (`no-store`).
  2. `TestEvaluacionConservaVaryDeCORS` (`evaluacion_test.go:80-94`) mantiene un middleware CORS falso que añade `Vary: Origin`, duplicando ahora el que ya pone el middleware real `Servidor.cors`.
  3. En `seguridadDelOpenAPI` (`seguridad_test.go:350-381`), no se verifica `sc.Err()` tras `for sc.Scan()`, y `esquemaOK` se activa con `n == 4` (`bearerAuth`) + `n == 6` (`scheme: bearer`) sin exigir que esté bajo `securitySchemes:` ni verificar `type: http` (pese al mensaje de error en `:387`).
  4. En `cmd/benchapi/main.go:156`, si `$BENCH_CORREO` está en el entorno y el usuario pasa `-token <jwt>`, `op.correo != ""` sobrescribe `-token` (o falla pidiendo `$BENCH_CLAVE`) en lugar de priorizar `-token` explícito o rechazar la combinación.
- **Corrección sugerida:** Añadir un caso de `429` (comprobando `Access-Control-Expose-Headers` y `Retry-After`) y un caso de pánico `500` en `TestCORSRespuestasDecoradas`, comprobar `riesgo`/`evaluacion` y `login` en `TestCabecerasSeguridad`, sustituir el middleware falso de `TestEvaluacionConservaVaryDeCORS` por `OrigenesCORS` real, comprobar `sc.Err()` + `securitySchemes` + `type: http` en `seguridadDelOpenAPI`, y en `cmd/benchapi/main.go` hacer que `-token` tenga prioridad cuando `op.token == ""` o avisar si ambos se pasan a la vez.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-83. |

## Cómo explicarlo en la sustentación

Todas las rutas HTTP salen de una tabla única (`api.Rutas()`) que declara si cada endpoint es público o `Bearer` y qué roles permite (`analista` o `admin`). Al construir el `Handler()`, la cadena de middlewares en stdlib aplica `recuperar → cabecerasSeguridad → cors → ServeMux → Autenticar + Requiere → handler`: el preflight `OPTIONS` se responde antes de la autenticación para los orígenes de `CORS_ORIGENES`, toda respuesta (incluidos 401/403/404/429/500) conserva cabeceras de seguridad y CORS, y si falta `JWT_SECRETO` el coordinador no arranca en modo servicio y la API falla cerrada con 503. Además, `TestMatrizRoles` recorre esa tabla contra {sin token, analista, admin} y `TestMatrizRolesCoincideConADR` verifica que el código no diverja de la matriz de `ADR-0007`. Una vez generado y versionado el reporte real `reports/bench_api_<maquina>_jwt.json` (**H1**), corregido el rechazo `413` en cuerpos no-JSON > 64 KiB (**H2**) y endurecida la validación de orígenes y la copia defensiva en `Rutas()` (**H3**), el cierre de #61 quedará completamente respaldado.
