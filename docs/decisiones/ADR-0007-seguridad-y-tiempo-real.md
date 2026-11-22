# ADR-0007 — Seguridad de la API (JWT, roles, CORS) y tiempo real (WebSocket con stdlib)

- **Estado:** Propuesto (22/11/2026) · Issue #57 · TB2
- **Relacionados:** ADR-0003 (solo stdlib salvo drivers de BD), ADR-0005 (protocolo y estado del
  clúster), ADR-0006 (colección `usuarios`), issues #58–#65

## Contexto

El entregable TB2 pide «integrar el frontend con el API concentradora (configuración de políticas de
seguridad de API)», y el enunciado describe una SPA conectada al coordinador por **REST + WebSocket**,
con login, **JWT** y **CORS**. La API de la PC4 no tiene autenticación: cualquiera puede lanzar un
entrenamiento. La SPA tiene dos tipos de usuario:

- **Analista** (DIRESA, epidemiólogo): consulta el mapa de riesgo, la evaluación y predice casos.
- **Admin** (el equipo): además opera el clúster (entrenar, ver el entrenamiento en vivo, tumbar un
  nodo en la demostración).

Restricciones: API y protocolo solo con la biblioteca estándar (AGENTS §3.2); p95 de predicción
< 100 ms (§3.5); la seguridad no puede alterar el entrenamiento (§3.3).

## Decisiones

| Tema | Decisión | Alternativa descartada y por qué |
|---|---|---|
| Token | **JWT HS256** firmado con `crypto/hmac` + `crypto/sha256`. Cabecera fija `{"alg":"HS256","typ":"JWT"}`: se rechaza cualquier otro `alg` (incluido `none`). Reclamos `sub` (correo), `rol`, `iss` = `cc65-tf`, `aud` = `vigia`, `iat`, `nbf`, `exp` = `iat` + 60 min | RS256: dos claves para un solo emisor y un solo verificador (el mismo proceso) no aportan nada; librerías JWT de terceros: prohibidas |
| Secreto | `JWT_SECRETO` por variable de entorno, ≥ 32 bytes. En modo servicio el coordinador **no arranca** sin él. Rotarlo invalida todas las sesiones | Secreto en el repo o generado al azar en cada arranque (cerraría sesiones en cada reinicio sin avisar) |
| Contraseñas | **PBKDF2-HMAC-SHA256** (`crypto/pbkdf2`, stdlib desde Go 1.24), 600 000 iteraciones (OWASP 2023), sal aleatoria de 16 bytes, clave de 32 bytes. Formato guardado `pbkdf2-sha256$<iter>$<sal b64>$<clave b64>`. Comparación con `crypto/subtle.ConstantTimeCompare` | bcrypt/argon2: están en `golang.org/x/crypto`, fuera de stdlib |
| Usuarios | Colección `usuarios` de ADR-0006 (`_id` = correo, `hash`, `rol`, `creado`) + implementación en memoria para tests. Alta con `cmd/usuarios` leyendo correo y clave de variables de entorno: ninguna clave en el repo | Registro abierto desde la SPA: no tiene sentido para un sistema de salud pública |
| Login | `POST /api/v1/auth/login`. Clave errada y correo inexistente devuelven **el mismo 401 con el mismo mensaje** y el mismo costo (se compara contra un hash señuelo) para no revelar qué correos existen. **Rate limit**: token bucket por IP, 5 intentos por minuto → 429 con `Retry-After` | Bloquear la cuenta tras N fallos: permite a un tercero bloquear al admin a propósito |
| Autorización | Cada `Ruta` declara `Roles`. Un único middleware lee `Authorization: Bearer <jwt>`: sin token o token inválido → **401**; rol no incluido → **403**. La matriz de abajo sale de `Rutas()` y un test la recorre completa | Comprobaciones dentro de cada handler: fácil olvidar una |
| Dónde vive el token | En la cabecera `Authorization`, nunca en cookies → **no hay CSRF**. La SPA lo guarda en `sessionStorage` (se borra al cerrar la pestaña) y una **CSP estricta** sin `unsafe-inline` reduce el riesgo de XSS | Cookie `HttpOnly`: obliga a defender CSRF y a usar credenciales en CORS |
| CORS | Middleware propio. Lista blanca `CORS_ORIGENES` (p. ej. `http://localhost:8081`); si el `Origin` está en la lista, responde ese origen exacto con `Vary: Origin`; nunca `*`; sin `Access-Control-Allow-Credentials`. Preflight `OPTIONS` → 204 con `Allow-Methods` (`GET, POST`), `Allow-Headers` (`Authorization, Content-Type`) y `Max-Age` 600. Origen fuera de la lista → sin cabeceras CORS (el navegador bloquea) | `Access-Control-Allow-Origin: *`: cualquier sitio podría usar la API con un token robado |
| Cabeceras | `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `Cache-Control: no-store` en respuestas autenticadas; `http.MaxBytesReader` de 64 KiB en los POST (413) | — |
| Tiempo real | **WebSocket implementado con stdlib** (`internal/ws`): handshake RFC 6455 con `http.Hijacker`, frames de texto del servidor sin máscara, lectura de frames enmascarados del cliente, ping/pong y cierre. Sin extensiones ni subprotocolos; el servidor no fragmenta. Mensaje máximo del cliente: 64 KiB | **SSE**: más simple, pero el enunciado pide WebSocket y la rúbrica evalúa la implementación. `golang.org/x/net/websocket` o gorilla: terceros |
| Autenticar el WebSocket | El navegador no puede enviar `Authorization` en el handshake. La SPA pide `POST /api/v1/ws/ticket` (con JWT, rol admin) y recibe un **ticket aleatorio de 32 bytes, de un solo uso, que vence a los 30 s**; abre `GET /api/v1/ws?ticket=…`. Además se valida que `Origin` esté en `CORS_ORIGENES` (evita *cross-site WebSocket hijacking*) | JWT en la URL: queda en logs e historial con 60 min de validez |
| Productor lento | El hub publica con envío **no bloqueante** a un canal con búfer por cliente; si se llena, descarta el mensaje y lo cuenta. El entrenamiento **nunca espera** al front. El observador solo lee copias de los pesos: no cambia el resultado (test bit a bit) | Envío bloqueante: un navegador lento frenaría la barrera de épocas |
| Despliegue del front | **Imagen `web` aparte** servida por `cmd/web` (Go stdlib, distroless) en `:8081`; la API en `:8080`. Son orígenes distintos, así que la política CORS se ejercita de verdad. `cmd/web` envía la CSP y entrega `/config.json` con `API_URL` en tiempo de ejecución | Embeber la SPA en el coordinador: mismo origen, el CORS no se probaría; nginx: otro lenguaje en el backend |

### Matriz ruta × rol

| Ruta | Público | Analista | Admin |
|---|:-:|:-:|:-:|
| `GET /healthz` | ✓ | ✓ | ✓ |
| `GET /docs`, `GET /docs/openapi.yaml` | ✓ | ✓ | ✓ |
| `POST /api/v1/auth/login` | ✓ (rate limit) | ✓ | ✓ |
| `GET /api/v1/modelo` | 401 | ✓ | ✓ |
| `POST /api/v1/predicciones` | 401 | ✓ | ✓ |
| `GET /api/v1/riesgo` | 401 | ✓ | ✓ |
| `GET /api/v1/evaluacion` | 401 | ✓ | ✓ |
| `GET /api/v1/cluster` | 401 | 403 | ✓ |
| `POST /api/v1/entrenamientos` | 401 | 403 | ✓ |
| `GET /api/v1/entrenamientos/{id}` | 401 | 403 | ✓ |
| `POST /api/v1/ws/ticket` | 401 | 403 | ✓ |
| `GET /api/v1/ws?ticket=…` | 401 sin ticket válido | 403 (el ticket guarda el rol) | ✓ |
| `POST /api/v1/demo/nodos/{nombre}/cortar` (solo con `-demo`; sin el flag, 404) | 401 | 403 | ✓ |

Swagger UI queda público para la sustentación; el OpenAPI declara `bearerAuth` y las respuestas
401/403/429, de modo que desde `/docs` se puede autorizar con un token y probar las rutas.

## Consecuencias

- El costo de verificar un JWT (un HMAC-SHA256 sobre ~300 bytes) es de microsegundos; aun así el p95
  se **vuelve a medir con token** (`cmd/benchapi` → `reports/bench_api_<maquina>_jwt.json`, #61).
- Los tokens son sin estado: cerrar sesión borra el token del navegador, pero el token sigue válido
  hasta su `exp` (60 min). Para este alcance se acepta; revocar exigiría una lista negra en Redis.
- PBKDF2 con 600 000 iteraciones cuesta del orden de cientos de milisegundos por login: es el costo
  buscado contra fuerza bruta y no afecta a la ruta de predicción.
- El servidor WebSocket es un subconjunto de RFC 6455 suficiente para navegadores actuales; lo que no
  implementa (extensiones como `permessage-deflate`, frames fragmentados del cliente) se rechaza con
  el código de cierre correspondiente, nunca se ignora en silencio.
- Las pruebas de seguridad (#70) dejan `reports/seguridad.json` con el resultado esperado vs obtenido
  de cada caso (CORS permitido/denegado, sin token, token vencido, rol ajeno, WS sin ticket, 429).
