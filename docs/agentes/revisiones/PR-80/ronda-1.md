# Veredicto — PR-80 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/60-login` @ `48edfee` (base: `integracion/tb2` @ `0f9c0f5`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR implementa el repositorio de usuarios (`internal/almacen/usuarios.go`, `Memoria` y colección `usuarios` en `mongo.Almacen` con `_id` = correo normalizado y contrato compartido probado contra MongoDB 7.0.14 en el job `bd` del CI), el endpoint `POST /api/v1/auth/login` en Go puro (`internal/api/login.go`, `limite.go`, `usuarios.go`) con hash señuelo PBKDF2-HMAC-SHA256, respuesta `{token, correo, rol, expira}` alineada con la SPA (`web/src/api/cliente.ts`), `Cache-Control: no-store`, siembra por variables de entorno (`cmd/usuarios` y `cmd/coordinator/seguridad.go`), OpenAPI 3 válido y colección de Postman. Todas las verificaciones locales (`gofmt`, `go vet`, `go vet -tags integracion`, `go test -race -count=1 ./...`) y los 7 jobs del CI en GitHub pasan en verde, y el núcleo cumple `AGENTS.md` §3.2 (solo biblioteca estándar; el driver oficial de MongoDB queda encapsulado en `internal/almacen/mongo` / `cmd/usuarios`). Sin embargo, se emiten **3 hallazgos mayores** y **1 menor** antes de aprobar: existe una ventana TOCTOU concurrente entre `Bloqueado` y `Fallo` (~605 ms por PBKDF2 de 600 000 iteraciones) que permite evadir el límite de 5 intentos con ráfagas concurrentes, sin ningún test concurrente del limitador/login y con una aserción en `TestLoginRateLimit` que no mata el mutante de eliminar `Exito` (**H1**); `limitador.Fallo` degrada a $O(N^2)$ bajo el mutex global y crece sin cota real cuando llegan ≥ 1 024 IPs activas en el mismo minuto (**H2**); `SembrarUsuarios` no valida los largos máximos de `correo` y `clave` que luego rechaza `login` (creando cuentas que no pueden iniciar sesión) y sobrescribe en silencio al `admin` si ambos roles comparten correo (**H3**); y hay detalles menores en el doble PBKDF2 del primer login respecto del tope de 1 s en Postman, el comando PowerShell no criptográfico de `deploy/.env.example` y comentarios/mensajes (**H4**).

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

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/almacen ./internal/api ./cmd/usuarios
github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen -> [context errors fmt .../cluster .../features strconv strings sync time]
github.com/JorgeGarcia005/cc65-tf-dengue/internal/api -> [context crypto/rand crypto/sha256 encoding/hex encoding/json errors fmt .../docs/api .../almacen .../auth .../cluster .../features .../ingest .../ml io log math net net/http runtime/debug strconv strings sync sync/atomic time unicode/utf8]
github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios -> [context flag fmt .../almacen .../almacen/mongo .../api io log os time]

$ go run ./cmd/usuarios -h
Uso: usuarios [-mongo URI] [-base NOMBRE]

Lee VIGIA_ADMIN_CORREO/VIGIA_ADMIN_CLAVE y VIGIA_ANALISTA_CORREO/VIGIA_ANALISTA_CLAVE
y crea o actualiza esas cuentas (claves de ≥ 12 caracteres, guardadas con PBKDF2).

  -base string
    	base de datos (default "cc65_dengue")
  -mongo string
    	URI de MongoDB (por defecto $MONGO_URI) (default "mongodb://localhost:27017")

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	4.068s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	10.669s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	5.184s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.034s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.639s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.282s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.225s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.419s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.477s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.058s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.198s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.032s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.082s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	17.506s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.319s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.558s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.047s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.084s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.318s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.575s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.755s

$ go test -count=1 -v -run 'Senuelo' ./internal/auth
--- PASS: TestClaveSenueloNoVerifica (1.21s)   # ~605 ms por PBKDF2 de 600 000 iteraciones sin -race

$ gh pr checks 80
Docker (imagen, clúster y caída de un nodo)	pass	2m3s
Ecosistema de agentes	pass	5s
Go (gofmt, vet, test -race)	pass	55s
MongoDB y Redis (tests de integración)	pass	45s  # TestMongoCumpleElContrato/{GuardarYBuscarConCorreoNormalizado,ReemplazaLaClave,RechazaInvalidos,Concurrente} PASS
OpenAPI 3 válido	pass	19s
SPA (tipos, tests, build e imagen)	pass	56s
Spin (modelo del protocolo y mutantes)	pass	1m44s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| CI GitHub Actions (7 jobs, incl. `bd` MongoDB/Redis y `openapi`) | ✅ |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`) |
| Latencia API p95 < 100 ms | no aplica (`POST /api/v1/predicciones` sin cambios; p95 con JWT se mide en #61; `TestMedirSeparaCacheFriaYCaliente` ✅) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | Alineado con ADR-0006 y ADR-0007. |
| Implementación | 8 | Parcial (pendiente **H1–H3**) | Cubre colección `usuarios` en Mongo/memoria, login con JWT HS256 + hash señuelo PBKDF2, siembra sin secretos en el repo y OpenAPI/Postman; quedan por corregir la ventana TOCTOU y cobertura concurrente del rate limit (**H1**), el acotamiento/poda con IPs activas (**H2**) y la validación de siembra (**H3**). |
| Interfaz | 5 | Coherente con la SPA | El contrato `{token, correo, rol, expira}` y `Retry-After` en `429` coinciden con `web/src/api/cliente.ts` y `web/src/sesion/Sesion.tsx`. |
| Informe | 2 | sin cambios | Las cifras/evidencias de seguridad se generan en #61 y #70. |
| Video | 2 | no aplica | Entregable TB2 final. |

## Hallazgos

### H1 — Ventana TOCTOU concurrente en el rate limit del login, ausencia de prueba concurrente y aserción que no verifica `Exito` · MAYOR

- **Dónde:** `internal/api/login.go:45-80`, `internal/api/limite.go:50-80` y `internal/api/login_test.go:110-136, 178-200`
- **Evidencia:**
  1. En `internal/api/login.go:46-79`, el handler consulta `s.limite.Bloqueado(ip)` (adquiere y suelta `l.mu`), luego consulta la base y ejecuta `auth.Comparar(hash, p.Clave)` —que corre PBKDF2-HMAC-SHA256 con 600 000 iteraciones— y recién al terminar llama a `s.limite.Fallo(ip)`. En esta máquina Linux x86_64 sin `-race`, `go test -count=1 -v -run 'Senuelo' ./internal/auth` tarda **1,21 s** para dos derivaciones (**~605 ms** por petición).
  2. Esa ventana de cientos de milisegundos crea una carrera lógica *check-then-act* (TOCTOU): si un cliente lanza $K$ peticiones `POST /api/v1/auth/login` concurrentes desde la misma IP (p. ej. 20 goroutines en paralelo), todas observan `len(f) < 5` en `Bloqueado(ip)` antes de que termine la primera derivación PBKDF2, todas ejecutan las 600 000 iteraciones en paralelo y todas responden `401` (cero `429`), burlando el límite de 5 intentos por minuto y saturando la CPU del mismo proceso que sirve `POST /api/v1/predicciones` (p95 < 100 ms, `AGENTS.md` §3.5).
  3. Aunque la descripción del PR destaca que el contador «lo comparten muchas peticiones concurrentes, así que lo protege un mutex», `internal/api/login_test.go` no contiene **ninguna prueba concurrente** de `limitador` ni de `POST /api/v1/auth/login` bajo `-race`.
  4. Además, en `TestLoginRateLimit` (`internal/api/login_test.go:129-135`), la comprobación tras el login exitoso no prueba que `s.limite.Exito(ip)` haya borrado los fallos: los 5 fallos ocurrieron en `t = 0, 1, 2, 3, 4 s`; al avanzar 55 s hasta `t = 60 s`, el fallo de `t = 0 s` ya venció y solo quedan 4 fallos (`1..4 s`). Como `4 < 5`, `s.limite.Bloqueado("127.0.0.1")` devuelve `false` **incluso si se borra por completo la línea `s.limite.Exito(ip)`** (`login.go:80`).
- **Corrección sugerida:**
  - Contabilizar los intentos **en curso** por IP (`enCurso map[string]int`) bajo `l.mu`: ofrecer una reserva atómica (p. ej. `Reservar(ip) (bloqueado bool, espera time.Duration)`) que verifique `len(vigentes) + enCurso[ip] >= max` antes de correr `Usuario` / `Comparar` y libere el cupo al terminar (con `defer` ante error/pánico, convirtiéndolo en fallo con `Fallo` o limpiando los fallos con `Exito`). Si los cupos están ocupados por intentos en curso sin 5 fallos cerrados aún, devolver `429` con `Retry-After: 1` (o el vencimiento del fallo más viejo si ya hay ≥ 5).
  - Añadir un test concurrente con `-race` que lance p. ej. 20 goroutines simultáneas con clave errada desde la misma IP y compruebe que exactamente `MaxFallosLogin` (5) obtienen `401` y las demás `429` con `Retry-After`.
  - Reforzar `TestLoginRateLimit` para que mate el mutante de eliminar `Exito`: tras el login exitoso en `t = 60 s`, enviar 1 fallo adicional y verificar que un siguiente intento **no** recibe `429` (sin `Exito`, 4 + 1 = 5 fallos bloquearían de inmediato).

### H2 — Poda $O(N^2)$ bajo mutex y crecimiento sin tope real en `limitador.Fallo` con ≥ 1 024 IPs activas en la misma ventana · MAYOR

- **Dónde:** `internal/api/limite.go:14, 63-73` y `internal/api/login_test.go:192-199`
- **Evidencia:**
  1. `Fallo` (`limite.go:67-71`) hace:
     ```go
     if len(l.fallos) >= maxIPsAntesPoda {
         for otra := range l.fallos {
             l.vigentes(otra, t)
         }
     }
     ```
  2. `TestLimitadorPorIPYPoda` solo ejercita el caso donde todas las IPs ya expiraron (`ahora = t0.Add(2 * VentanaFallos)`). Si `1 024` IPs distintas fallan dentro del mismo minuto, ninguna está vencida: el bucle no borra ninguna entrada, `len(l.fallos)` queda `>= 1024`, y a partir de la IP 1 025 **cada llamada a `Fallo` recorre el mapa entero bajo `l.mu.Lock()`** ($O(N)$ por fallo → $O(N^2)$ acumulado bloqueando todos los logins) mientras `l.fallos` sigue creciendo sin límite duro de memoria. Esto contradice la garantía declarada en el PR («la memoria del limitador no crece sin límite»).
- **Corrección sugerida:**
  - Espaciar las podas globales con una marca temporal (p. ej. `ultimaPoda time.Time`, podando a lo sumo una vez por `ventana/2` o `ventana`) para evitar el barrido $O(N)$ en cada petición cuando hay ≥ 1 024 IPs activas.
  - Si tras podar vencidas el mapa sigue en un tope máximo (p. ej. `maxIPs`), aplicar una política acotada explícita (desalojar la entrada más antigua o limitar el tamaño del mapa) y añadir un test que pruebe `> maxIPsAntesPoda` IPs activas dentro de la misma ventana.

### H3 — `SembrarUsuarios` omite los topes de longitud que exige `login` y sobrescribe en silencio si `admin` y `analista` comparten correo · MAYOR

- **Dónde:** `internal/api/usuarios.go:28-48`, `internal/api/login.go:17-20, 56-59` y `docs/api/openapi.yaml:189-190`
- **Evidencia:**
  1. `login.go:56` rechaza con `400` toda petición con `len(correo) > maxLargoCorreo` (254 bytes) o `len(p.Clave) > maxLargoClave` (256 bytes). En contraste, `SembrarUsuarios` (`usuarios.go:35`) solo valida `utf8.RuneCountInString(clave) < MinLargoClave` (12 runas) y no impone tope superior a `correo` ni a `clave`. Si se siembra una clave de > 256 bytes (o de 150 caracteres UTF-8 multibyte, 300 bytes) o un correo de > 254 bytes, `cmd/usuarios` / `prepararSeguridad` la persisten con éxito y esa cuenta queda permanentemente bloqueada con `400` en `POST /api/v1/auth/login`. (Además, `login.go:57` dice «máximo 254 y 256 caracteres» cuando valida bytes con `len`.)
  2. Si `VIGIA_ADMIN_CORREO` y `VIGIA_ANALISTA_CORREO` normalizan al mismo correo, el bucle de `SembrarUsuarios` (`usuarios.go:30-46`) guarda primero `admin`, luego sobrescribe el mismo `_id` con `analista` sin error, y devuelve `["<correo> (admin)", "<correo> (analista)"]`, reportando ambas cuentas sembradas cuando la de `admin` fue destruida.
- **Corrección sugerida:**
  - Validar en `SembrarUsuarios` los mismos topes máximos (`maxLargoCorreo` y `maxLargoClave`) que aplica `login` —unificando el criterio de bytes/runas entre `usuarios.go`, `login.go` y el texto de error— y rechazar con error claro si dos roles de `VariablesSiembra` normalizan al mismo correo.
  - Cubrir ambos casos en `TestSembrarUsuarios`.

### H4 — Doble PBKDF2 en el primer login frente al tope de 1 s de Postman, generador PowerShell no criptográfico y detalles de documentación · MENOR

- **Dónde:** `internal/api/login.go:41-44, 70`, `docs/api/cc65-tf.postman_collection.json:71`, `deploy/.env.example:7`, `internal/api/api.go:6-13`, `internal/almacen/mongo/mongo.go:1-3`
- **Evidencia:**
  1. `login.go:70` inicializa `auth.HashSenuelo()` de forma perezosa (`sync.OnceValue`). Como una derivación PBKDF2 de 600 000 iteraciones tarda ~605 ms en esta máquina sin `-race` (`TestClaveSenueloNoVerifica = 1.21s` para dos derivaciones), la **primera** petición `POST /api/v1/auth/login` tras arrancar el coordinador ejecuta dos derivaciones (~1,2 s) y puede hacer fallar la aserción de Postman `pm.expect(pm.response.responseTime).to.be.below(1000)` (`cc65-tf.postman_collection.json:71`).
  2. `deploy/.env.example:7` propone generar `JWT_SECRETO` en PowerShell con `[Convert]::ToBase64String((1..48 | % { Get-Random -Max 256 }))`. En Windows PowerShell 5.1 (predeterminado en Windows), `Get-Random` usa `System.Random` sembrado con `TickCount` (no criptográfico), inadecuado para un secreto JWT.
  3. El comentario de paquete de `internal/api/api.go:6-13` no incluye `POST /api/v1/auth/login`; el de `internal/almacen/mongo/mongo.go:1-3` no menciona `almacen.Usuarios` ni la colección `usuarios`; y `login.go:41-42` atribuye siempre el `503` a `(falta JWT_SECRETO)` incluso cuando `Firmador != nil` y lo que es `nil` es `Usuarios`.
- **Corrección sugerida:**
  - Precalentar `auth.HashSenuelo()` al crear el `Servidor` (o en `prepararSeguridad` cuando `Firmador != nil`) para que el primer login pague solo una derivación PBKDF2.
  - Cambiar el ejemplo de PowerShell en `deploy/.env.example:7` a `$b = New-Object byte[] 48; [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b); [Convert]::ToBase64String($b)`.
  - Actualizar los comentarios de paquete en `api.go` y `mongo.go` y diferenciar el mensaje 503 si falta `Usuarios`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-80. |

## Cómo explicarlo en la sustentación

La colección `usuarios` de MongoDB (ADR-0006) guarda cada cuenta usando el correo normalizado en minúsculas como `_id` (único por construcción), el rol (`admin` o `analista`) y el hash PBKDF2-HMAC-SHA256 de 600 000 iteraciones etiquetado con `json:"-"` para que jamás pueda serializarse en una respuesta HTTP; las cuentas se siembran desde variables de entorno (`cmd/usuarios` o `prepararSeguridad` al arrancar) sin versionar claves en el repositorio. Al recibir `POST /api/v1/auth/login`, el coordinador valida el JSON de forma estricta, aplica un limitador por IP (tomada de `RemoteAddr`, ignorando `X-Forwarded-For`) con ventana deslizante de un minuto, ejecuta siempre un PBKDF2 completo (contra un hash señuelo si el correo no existe) para devolver el mismo `401` y no revelar correos registrados, y emite un JWT HS256 de 60 minutos con el contrato `{token, correo, rol, expira}` de la SPA. Tras cerrar la ventana TOCTOU concurrente del limitador (**H1**), acotar la poda bajo alta concurrencia de IPs (**H2**) y alinear las validaciones de siembra con el login (**H3**), la autenticación quedará lista para enchufar el middleware de roles y CORS en #61.
