# Veredicto — PR-76 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/58-auth` @ `4a9a6ca` (base: `integracion/tb2` @ `396c4ce`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR implementa `internal/auth` exclusivamente con la biblioteca estándar de Go 1.24 (`crypto/hmac`, `crypto/sha256`, `crypto/pbkdf2`, `crypto/subtle`, `net/http`), sin dependencias de terceros ni estado compartido mutable en `Firmador` o `HashSenuelo`, y pasa en verde la suite local completa con `-race` en los 26 paquetes (`0` carreras, `92,9 %` de cobertura en `internal/auth`). Sin embargo, se emite **1 hallazgo bloqueante**, **4 mayores** y **1 menor** antes de aprobar: el CI de GitHub Actions no ha corrido sobre el PR porque `integracion/tb2` aún no incluye el PR #74 (**H1**); `Verificar` acepta firmas con el último carácter alterado o con `\n` embebido por decodificación base64url no estricta (**H2**); la verificación de cabecera contradice `jwt.go:58` y ADR-0007 al ignorar `c.Typ` y aceptar `ALG` en mayúsculas o `crit` desconocido (**H3**); `NuevoFirmador` admite duraciones `< 1s` que producen tokens nacidos vencidos (**H4**); `Comparar` no acota `len(esperado)` ni `len(sal)`, permitiendo superar el tope `maxIteraciones` y aceptar hashes truncados a 1 byte (**H5**); y `Requiere(roles ...Rol)` conserva el slice del llamador sin clonarlo mientras las pruebas de middleware no comprueban `rol` en el contexto (**H6**).

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ git diff --stat origin/integracion/tb2...origin/feature/58-auth
 internal/auth/clave.go           |  79 +++++++++++++++++++++
 internal/auth/clave_test.go      |  56 +++++++++++++++
 internal/auth/jwt.go             | 145 +++++++++++++++++++++++++++++++++++++++
 internal/auth/jwt_test.go        | 126 ++++++++++++++++++++++++++++++++++
 internal/auth/middleware.go      |  82 ++++++++++++++++++++++
 internal/auth/middleware_test.go | 105 ++++++++++++++++++++++++++++
 6 files changed, 593 insertions(+)

$ git diff --check origin/integracion/tb2...origin/feature/58-auth
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.Imports}}' ./internal/auth
[bytes context crypto/hmac crypto/pbkdf2 crypto/rand crypto/sha256 crypto/subtle encoding/base64 encoding/json errors fmt net/http slices strconv strings sync time]

$ go test -race -count=1 -v -cover ./internal/auth
--- PASS: TestClaveIdaYVuelta (0.04s)
--- PASS: TestClaveSalDistinta (0.02s)
--- PASS: TestClaveFormatoPorDefecto (10.09s)
--- PASS: TestClaveGuardadoInvalido (0.00s)
--- PASS: TestClaveSenueloNoVerifica (14.29s)
--- PASS: TestJWTIdaYVuelta (0.00s)
--- PASS: TestJWTRechazos (0.01s)  # 14/14 subtests PASS
--- PASS: TestJWTSecretoCorto (0.00s)
--- PASS: TestJWTFirmarRolInvalido (0.00s)
--- PASS: TestMiddleware (0.00s)   # 8/8 subtests PASS
--- PASS: TestMiddlewareRequiereSinAutenticar (0.00s)
--- PASS: TestMiddlewareConcurrente (0.02s)
PASS
coverage: 92.9% of statements
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	25.569s	coverage: 92.9% of statements

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.446s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.216s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.724s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.034s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.303s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.240s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.426s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.177s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.025s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.054s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.151s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	14.066s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.653s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.992s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.086s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.926s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.267s

$ gh pr checks 76
no checks reported on the 'feature/58-auth' branch

$ go test -race -count=1 -overlay <scratch/overlay.json> -run '^TestAudit' -v ./internal/auth
=== RUN   TestAuditSubSecondDuration
    audit_scratch_test.go:29: duracion=500ms -> iat=1800000000 exp=1800000000 Verificar=token vencido (errors.Is ErrVencido=true)
=== RUN   TestAuditSignatureMalleability
    audit_scratch_test.go:49: tok[-4:]="aATs" tokMut[-4:]="aATt" equal=false Verificar(tokMut) err=<nil> sub="ana@diresa.pe"
    audit_scratch_test.go:56: tokNL con '\n' en firma: Verificar(tokNL) err=<nil> sub="ana@diresa.pe"
=== RUN   TestAuditHeaderValidation
    audit_scratch_test.go:79: caso="typ distinto" -> err=<nil> sub="ana@diresa.pe" rol="analista"
    audit_scratch_test.go:79: caso="sin typ y ALG mayúscula" -> err=<nil> sub="ana@diresa.pe" rol="analista"
    audit_scratch_test.go:79: caso="crit no soportado" -> err=<nil> sub="ana@diresa.pe" rol="analista"
    audit_scratch_test.go:91: cabecera=null -> err=algoritmo de firma no permitido: "" (IsMalformado=false IsAlgoritmo=true)
=== RUN   TestAuditRequiereSliceAliasing
    audit_scratch_test.go:115: tras mutar slice del llamador roles[0]=RolAnalista: admin->403, analista->204
=== RUN   TestAuditPBKDF2KeyLenMultiplication
    audit_scratch_test.go:136: iter=10000: len(esperado)=32 -> 43.833733ms; len(esperado)=320 (10 bloques) -> 442.384321ms (ratio=10.1x)
    audit_scratch_test.go:147: hash truncado a 1 byte con iter=1 verifica=true
PASS
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (`gofmt -l .`, `go vet ./...` y `scripts/check-skills-sync.sh` sin errores) |
| `go test -race ./...` | ✅ local (carreras: 0 en los 26 paquetes, incluidos `cmd/coordinator` e `internal/almacen/mongo`) |
| CI del PR en GitHub Actions | ❌ sin checks reportados (`gh pr checks 76` vacío — **H1**) |
| Restricción stdlib (`AGENTS.md` §3.2) | ✅ (cero importaciones externas; `go.mod` intacto) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; sus tests pasan dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no conecta todavía el middleware a `internal/api`; se medirá en #61) |

## Verificación contra los criterios del issue #58

| Criterio (#58) | Estado | Evidencia |
|---|---|---|
| 1. `Firmar`/`Verificar`: token alterado, `alg: none`, `alg: HS512`, `exp` vencido, `nbf` futuro, `aud`/`iss` ajenos, rol desconocido, `sub` vacío → error tipado | ⚠️ parcial (**H2**, **H3**, **H4**) | `go test -race -run 'JWT' ./internal/auth` pasa (14/14 subtests), pero `Verificar` acepta firmas alteradas en los 2 bits sobrantes del último carácter o con `\n` embebido (**H2**), ignora `c.Typ` y acepta `ALG`/`crit` (**H3**), y `NuevoFirmador` admite duraciones `< 1s` que nacen vencidas (**H4**). |
| 2. `Hash`/`Comparar` PBKDF2-SHA256 (≥ 600 000 iteraciones, sal de 16 bytes): correcta verifica, otra no, dos hashes difieren | ⚠️ parcial (**H5**) | `go test -race -run 'Clave' ./internal/auth` pasa (600 000 iteraciones, ~0,43 s sin `-race`), pero `Comparar` no acota `len(esperado)` ni `len(sal)` (elude `maxIteraciones` y acepta hashes de 1 byte) y `TestClaveFormatoPorDefecto` no verifica la longitud decodificada de la sal de 16 bytes (**H5**). |
| 3. Middleware `Autenticar` + `Requiere(roles...)`: sin token 401, rol ajeno 403, válido pasa con `sub` y `rol` en el contexto | ⚠️ parcial (**H6**) | `go test -race -run 'Middleware' ./internal/auth` pasa (401 con `WWW-Authenticate: Bearer`, 403 con `Cache-Control: no-store`, 64 goroutines), pero `Requiere` no clona `roles` y `TestMiddleware` solo verifica `visto.Sub`, nunca `visto.Rol` (**H6**). |
| 4. Secreto < 32 bytes → error al construir | ✅ | `TestJWTSecretoCorto` verifica `ErrSecretoCorto` (`jwt.go:71-73`, `jwt_test.go:112-119`). |
| 5. `make check` en verde y CI | ⚠️ parcial (**H1**) | Local completo en verde (`gofmt`, `vet`, `skills`, `go test -race ./...`), pero el CI de GitHub Actions no ha corrido en PR #76 (**H1**). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | El diseño de seguridad se documenta en ADR-0007 (PR #75, aún abierto). |
| Implementación | 8 | Parcial (pendiente **H1–H5**) | Núcleo criptográfico y middlewares implementados solo con stdlib y sin carreras; falta cerrar la validación estricta de firma/cabecera/duración/PBKDF2 y contar con el CI en verde en `integracion/tb2`. |
| Interfaz | 5 | no aplica | Corresponde a #62–#70. |
| Informe | 2 | no aplica | Corresponde a #71. |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

### H1 — El CI del PR no ha corrido en GitHub Actions (`integracion/tb2` aún no incluye PR #74) · BLOQUEANTE

- **Dónde:** PR #76 (`baseRefName: integracion/tb2`) y `.github/workflows/ci.yml:3-7` en `origin/integracion/tb2` (`396c4ce`)
- **Evidencia:**
  1. `gh pr checks 76` devuelve `no checks reported on the 'feature/58-auth' branch` (`statusCheckRollup: []`).
  2. La rama base `origin/integracion/tb2` sigue en `396c4ce` (igual que `pc4`), donde `.github/workflows/ci.yml:4-7` solo dispara el flujo para `branches: [main, develop]`. El cambio que activa el CI en `integracion/tb2` vive en el PR #74 (`feature/56-integracion-tb2`), que sigue abierto.
  3. La propia descripción del PR #76 se apoya en el CI (*«el CI corre la suite completa»*) y las reglas de `juez-cc65` y `AGENTS.md` §5 exigen el CI del PR en verde para emitir `APROBADO`. (Nota adicional de orden: el issue #58 declara `Depende de #57 (T1)` y el código cita `ADR-0007` en `jwt.go:2,25` y `clave.go:15,78`, documento que aún está en el PR #75 abierto).
- **Corrección sugerida:** Fusionar primero el PR #74 (`#56`) —y en orden el PR #75 (`#57`)— en `integracion/tb2`, y luego actualizar el PR #76 (al subir los commits de corrección de esta ronda) para que GitHub Actions ejecute todos los jobs sobre la rama y `gh pr checks 76` quede en verde.

### H2 — `Verificar` acepta tokens con la firma alterada por decodificación base64url no estricta y `\r`/`\n` embebidos · MAYOR

- **Dónde:** `internal/auth/jwt.go:103, 114-121`
- **Evidencia:**
  1. Un HMAC-SHA256 tiene 32 bytes (256 bits). En base64url sin relleno ocupa 43 caracteres (`42 × 6 = 252` bits + 4 bits útiles y **2 bits sobrantes** en el último carácter).
  2. `base64.RawURLEncoding.DecodeString` (`jwt.go:114`) no usa `.Strict()` y, por diseño de `encoding/base64`, ignora bits sobrantes distintos de cero así como cualquier `\r` o `\n` embebido.
  3. Reproducido con overlay (`TestAuditSignatureMalleability`): tomando un token válido emitido por `f.Firmar`, al alterar el último carácter de la firma (`"aATs"` → `"aATt"`, cadena distinta) `f.Verificar(tokMut)` devolvió **`err=<nil>`** (`sub="ana@diresa.pe"`). Al insertar `\n` dentro de `partes[2]`, `f.Verificar(tokNL)` también devolvió **`err=<nil>`**. Esto contradice el criterio 1 del issue #58 (*«token alterado → error tipado»*) y hace que cada token válido admita múltiples representaciones distintas como válidas.
- **Corrección sugerida:** Rechazar `\r` y `\n` en el token (p. ej. `if strings.ContainsAny(token, "\r\n") { return Reclamos{}, ErrMalformado }`) y usar `base64.RawURLEncoding.Strict()` al decodificar las tres partes (o validar `partes[2]` con `.Strict()` y comparar en tiempo constante). Añadir en `TestJWTRechazos` un caso con el último carácter de la firma mutado en sus bits sobrantes y otro con `\n` en la firma.

### H3 — La validación de la cabecera contradice `jwt.go:58` y ADR-0007 (`c.Typ` nunca se comprueba; acepta `ALG` y `crit`; `null` devuelve `ErrAlgoritmo`) · MAYOR

- **Dónde:** `internal/auth/jwt.go:58-59, 103-113`
- **Evidencia:**
  1. `internal/auth/jwt.go:58` afirma: *«La cabecera es siempre la misma: se precalcula y se exige tal cual al decodificar.»* y `ADR-0007` (PR #75) especifica: *«Cabecera fija `{"alg":"HS256","typ":"JWT"}`: se rechaza cualquier otro `alg` (incluido `none`).»*
  2. Sin embargo, `jwt.go:107-113` decodifica en `var c struct{ Alg, Typ string }` sin etiquetas JSON, nunca lee `c.Typ` y nunca compara con `cabeceraB64`.
  3. Reproducido con overlay (`TestAuditHeaderValidation`): `Verificar` devolvió **`err=<nil>`** para tokens firmados con cabecera `{"alg":"HS256","typ":"NO-JWT"}`, `{"ALG":"HS256"}` (sin `typ` y clave en mayúscula por la coincidencia insensible a mayúsculas de `encoding/json`) y `{"alg":"HS256","typ":"JWT","crit":["x-req"]}` (parámetro crítico desconocido según RFC 7515 §4.1.11). Además, una cabecera JSON `null` (`bnVsbA`) devolvió `ErrAlgoritmo: ""` en lugar de `ErrMalformado`.
- **Corrección sugerida:** En el camino normal comprobar `partes[0] == cabeceraB64` (lo que hace verdadera la documentación de `jwt.go:58` y evita decodificar/parsear JSON en cada petición válida). Solo cuando `partes[0] != cabeceraB64`, decodificar con `.Strict()` para diferenciar un `"alg"` explícito distinto de `"HS256"` (`ErrAlgoritmo`) de cualquier otra cabecera inválida (`typ` distinto o ausente, `crit`, `ALG`, `null` → `ErrMalformado`). Cubrir estos casos en `TestJWTRechazos`.

### H4 — `NuevoFirmador` acepta duraciones sub-segundo (`0 < duracion < 1s`) y emite tokens que nacen vencidos · MAYOR

- **Dónde:** `internal/auth/jwt.go:74-76, 86-87, 131`
- **Evidencia:** `NuevoFirmador` comprueba únicamente `duracion <= 0` (`jwt.go:74`), pero `Firmar` calcula `Exp: t + int64(f.duracion/time.Second)` (`jwt.go:87`). Reproducido con overlay (`TestAuditSubSecondDuration`): `NuevoFirmador(secretoPrueba, 500*time.Millisecond)` retorna `err=<nil>`, `Firmar` produce `iat=1800000000, exp=1800000000`, y `Verificar` en el mismo instante `ahora` falla de inmediato con `ErrVencido` (`ahora >= rec.Exp` en `jwt.go:131`).
- **Corrección sugerida:** Rechazar `duracion < time.Second` en `NuevoFirmador` (p. ej. `"la duración del token debe ser de al menos 1 segundo"`, acorde con la resolución en segundos de `NumericDate` en RFC 7519) y agregar el caso `500*time.Millisecond` en `TestJWTSecretoCorto` (o `TestJWTConstructor`).

### H5 — `Comparar` no acota `len(esperado)` ni `len(sal)`, eludiendo `maxIteraciones` y aceptando hashes truncados a 1 byte · MAYOR

- **Dónde:** `internal/auth/clave.go:18, 52-58` y `internal/auth/clave_test.go:29-41`
- **Evidencia:**
  1. `clave.go:18` fija `maxIteraciones = 10_000_000` (*«evita que uno manipulado congele el login»*), pero `clave.go:55-58` solo rechaza `len(sal) == 0 || len(esperado) == 0` y pasa `len(esperado)` sin cota a `pbkdf2.Key(sha256.New, clave, sal, iter, len(esperado))`. Como PBKDF2-HMAC-SHA256 ejecuta `iter × ceil(len(esperado)/32)` iteraciones, en `TestAuditPBKDF2KeyLenMultiplication` (con `iter=10000`), `len(esperado)=32` tardó `43,8 ms` mientras `len(esperado)=320` (10 bloques) tardó `442,4 ms` (**10,1×** más CPU para el mismo `iter`). Un hash manipulado con `iter = maxIteraciones` y una clave derivada larga burla el tope arbitrariamente.
  2. Además, en el mismo test, un hash truncado a 1 byte con `iter=1` devolvió `Comparar(...) == true`, y `clave.go:52` usa `base64.RawStdEncoding` sin `.Strict()` ni rechazo de `\r`/`\n`.
  3. En `clave_test.go:34-37`, `TestClaveFormatoPorDefecto` comprueba `partes[0]` y `partes[1]`, pero no decodifica `partes[2]` ni `partes[3]` para acreditar la sal de 16 bytes exigida en el criterio 2 del issue #58 ni la clave de 32 bytes de ADR-0007.
- **Corrección sugerida:** En `Comparar`, exigir `len(sal) == bytesSal && len(esperado) == bytesClave` (o al menos acotar `len(sal)` a `[bytesSal, 64]` y fijar `len(esperado) == bytesClave`), usar `base64.RawStdEncoding.Strict()` y rechazar `\r`/`\n`. En `clave_test.go`, verificar en `TestClaveFormatoPorDefecto` que la sal decodificada mida 16 bytes y la clave 32 bytes, y agregar en `TestClaveGuardadoInvalido` casos con clave/sal truncada o sobredimensionada.

### H6 — `Requiere(roles ...Rol)` conserva el slice del llamador sin clonarlo y las pruebas de middleware no comprueban `rol` en el contexto · MENOR

- **Dónde:** `internal/auth/middleware.go:44-52` y `internal/auth/middleware_test.go:18-62, 86-105`
- **Evidencia:**
  1. Mientras `NuevoFirmador` clona defensivamente el slice `secreto` (`jwt.go:77`, `bytes.Clone`), `Requiere(roles ...Rol)` captura `roles` directamente. Reproducido con overlay (`TestAuditRequiereSliceAliasing`): al pasar `roles := []Rol{RolAdmin}` como `Requiere(roles...)` y mutar luego `roles[0] = RolAnalista`, el middleware cambió de política en caliente (`admin → 403`, `analista → 204`); una mutación concurrente con `ServeHTTP` generaría condición de carrera.
  2. El criterio 3 del issue #58 pide comprobar que un token válido *«pasa con `sub` y `rol` en el contexto»*, pero `TestMiddleware` (`middleware_test.go:57-61`) solo verifica `visto.Sub != c.sub` y no afirma `visto.Rol`. Asimismo, `TestMiddlewareConcurrente` (`middleware_test.go:86-105`) envía el mismo token en las 64 goroutines a un handler vacío sin inspeccionar `ReclamosDe(r.Context())` ni combinar peticiones con roles distintos y rechazos concurrentes.
- **Corrección sugerida:** Clonar el slice al construir el middleware (`permitidos := slices.Clone(roles)`). En `TestMiddleware`, verificar también `visto.Rol == c.rol` y añadir un caso de token vencido (401 con `error="invalid_token"`); en `TestMiddlewareConcurrente`, mezclar tokens de `admin`, `analista` y tokens inválidos verificando estado HTTP y `ReclamosDe(r.Context())` en cada goroutine.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-76. |

## Cómo explicarlo en la sustentación

`internal/auth` concentra la criptografía y el control de acceso usando solo la biblioteca estándar de Go 1.24: `Firmador` es inmutable entre goroutines y firma JWT HS256 (`cabecera.carga.firma`) con `crypto/hmac` y `crypto/sha256`, comparando la firma en tiempo constante y validando algoritmo, vigencia (`nbf`/`exp`), emisor, audiencia y rol; `clave.go` guarda las contraseñas con `crypto/pbkdf2` (600 000 iteraciones, sal aleatoria de 16 bytes) y prepara un hash señuelo con `sync.OnceValue` para que un correo inexistente tarde lo mismo que una clave errada; y los middlewares `Autenticar` y `Requiere` desacoplan la verificación del token (401 con `WWW-Authenticate: Bearer`) de la autorización por rol (403) pasando los reclamos por `context.Context`. Tras aplicar los ajustes de validación estricta (**H2–H6**) y correr el CI en `integracion/tb2` (**H1**), el paquete quedará listo como base para el login (#60) y la matriz de roles (#61).
