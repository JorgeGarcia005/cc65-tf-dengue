# ADR-0007 — Seguridad de la API (JWT, roles, CORS) y tiempo real (WebSocket con stdlib)

- **Estado:** Propuesto (22/11/2026) · Issue #57 · TB2
- **Relacionados:** ADR-0003 (solo stdlib salvo drivers de BD), ADR-0005 (protocolo y estado del
  clúster), ADR-0006 (colección `usuarios`); issues #58 (auth), #60 (login), #61 (roles y CORS), #62 (SPA), #63 (WebSocket), #64–#65 (eventos y ticket), #67 (riesgo y evaluación), #69 (panel en vivo), #70 (evidencias de seguridad)

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
| Token | **JWT HS256** firmado con `crypto/hmac` + `crypto/sha256`, codificado en `base64.RawURLEncoding` **estricto** (RFC 7515; sin bits sobrantes ni saltos de línea). Cabecera fija `{"alg":"HS256","typ":"JWT"}` exigida byte a byte: un `alg` distinto (incluido `none`) es error de algoritmo y cualquier otra diferencia (`typ`, `crit`…) es token malformado. La firma se compara con `hmac.Equal` (tiempo constante). Reclamos `sub` (correo), `rol`, `iss` = `cc65-tf`, `aud` = `vigia`, `iat`, `nbf`, `exp` = `iat` + 60 min (vida mínima 1 s: `exp` va en segundos enteros) | RS256: dos claves para un solo emisor y un solo verificador (el mismo proceso) no aportan nada; librerías JWT de terceros: prohibidas |
| Secreto | `JWT_SECRETO` por variable de entorno, ≥ 32 bytes. En modo servicio el coordinador **no arranca** sin él. Rotarlo invalida todas las sesiones | Secreto en el repo o generado al azar en cada arranque (cerraría sesiones en cada reinicio sin avisar) |
| Contraseñas | **PBKDF2-HMAC-SHA256** (`crypto/pbkdf2`, stdlib desde Go 1.24), 600 000 iteraciones (OWASP 2023), sal aleatoria de 16 bytes, clave de 32 bytes. Formato guardado `pbkdf2-sha256$<iter>$<sal b64>$<clave b64>`. Comparación con `crypto/subtle.ConstantTimeCompare` | bcrypt/argon2: están en `golang.org/x/crypto`, fuera de stdlib |
| Usuarios | Colección `usuarios` de ADR-0006 (`_id` = correo **normalizado** con `strings.ToLower(strings.TrimSpace(…))` al guardar y al buscar, porque `_id` distingue mayúsculas; `hash`, `rol`, `creado`) + implementación en memoria para tests. Alta con `cmd/usuarios` leyendo correo y clave de variables de entorno: ninguna clave en el repo | Registro abierto desde la SPA: no tiene sentido para un sistema de salud pública |
| Login | `POST /api/v1/auth/login` → `200 {token, correo, rol, expira}` (`expira` en segundos Unix). Clave errada y correo inexistente devuelven **el mismo 401 con el mismo mensaje** y el mismo costo (se compara contra un hash señuelo) para no revelar qué correos existen. **Rate limit** por IP, tomada del host de `r.RemoteAddr` con `net.SplitHostPort` (no de `X-Forwarded-For`, falsificable sin un proxy de confianza): ventana deslizante de 1 minuto sobre los intentos **fallidos**; el 6.º fallo dentro del minuto → 429 con `Retry-After` (segundos hasta que salga el fallo más viejo). Las entradas sin fallos en el último minuto se borran del mapa | Bloquear la cuenta tras N fallos: permite a un tercero bloquear al admin a propósito |
| Autorización | Cada `Ruta` declara su **modo de autenticación**: `publica`, `bearer` (JWT en `Authorization: Bearer`, con lista de `Roles`) o `ticket` (solo `GET /api/v1/ws`, con `Roles`). En modo `bearer`, sin token o token inválido → **401**; rol no incluido → **403**. El modo `ticket` no pasa por el middleware Bearer (ver «Autenticar el WebSocket»). La matriz de abajo sale de `Rutas()` y un test la recorre por modo: `publica` con y sin token, `bearer` × {sin token, analista, admin}, `ticket` × {sin ticket, ticket de analista, ticket de admin} | Comprobaciones dentro de cada handler: fácil olvidar una |
| Dónde vive el token | En la cabecera `Authorization`, nunca en cookies → **no hay CSRF**. La SPA lo guarda en `sessionStorage` (se borra al cerrar la pestaña) y una **CSP estricta** sin `unsafe-inline` reduce el riesgo de XSS | Cookie `HttpOnly`: obliga a defender CSRF y a usar credenciales en CORS |
| CORS | Middleware propio. Lista blanca `CORS_ORIGENES` (p. ej. `http://localhost:8081`). Si el `Origin` está en la lista, **toda** respuesta (2xx, 4xx y 5xx, incluidos 401, 403 y 429) lleva `Access-Control-Allow-Origin: <ese origen>`, `Vary: Origin` y `Access-Control-Expose-Headers: X-Cache, Retry-After, Location` (sin esto el navegador oculta `X-Cache` de la predicción, el `Retry-After` del login y el `Location` del entrenamiento). Nunca `*` ni `Access-Control-Allow-Credentials`. Preflight `OPTIONS` de un origen permitido → 204 con `Access-Control-Allow-Methods: GET, POST`, `Access-Control-Allow-Headers: Authorization, Content-Type` y `Access-Control-Max-Age: 600`; de un origen no permitido → 204 **sin** cabeceras `Access-Control-Allow-*` (el navegador bloquea). El preflight se responde en el middleware, antes del `ServeMux`: nunca llega a la autenticación (el navegador no manda `Authorization` en él) ni al 405 del mux | `Access-Control-Allow-Origin: *`: cualquier sitio podría usar la API con un token robado |
| Cabeceras | `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `Cache-Control: no-store` por defecto en respuestas autenticadas (los documentos con ETag de `/api/v1/riesgo` y `/api/v1/evaluacion` usan `private, no-cache`: el navegador revalida y ninguna caché compartida guarda copia); `http.MaxBytesReader` de 64 KiB en los POST (413). **Orden de la cadena** en `Handler()`, de afuera hacia adentro: `recuperar` (pánico → 500) → cabeceras de seguridad → **CORS** (responde el preflight y decora toda respuesta de un origen permitido) → `ServeMux` → autenticación y rol de la ruta elegida → handler. La autenticación envuelve cada handler al registrarlo en el mux (sale de `Rutas()`), así cada ruta aplica su modo; sin firmador, las rutas protegidas responden 503 (la API falla cerrada) y el coordinador en modo servicio no arranca sin `JWT_SECRETO` | — |
| Tiempo real | **WebSocket implementado con stdlib** (`internal/ws`): handshake RFC 6455 tomando la conexión con `http.NewResponseController(w).Hijack()` (funciona aunque un middleware envuelva el `ResponseWriter`); justo después se **limpian los plazos heredados** del `http.Server` (`ReadTimeout` 15 s y `WriteTimeout` 30 s de `NuevoHTTP`, que `Hijack` no quita) con `SetDeadline(time.Time{})`, y se sigue leyendo del `bufio.Reader` que devuelve el hijack (puede traer bytes ya leídos). Frames de texto del servidor sin máscara, lectura de frames enmascarados del cliente, ping/pong y cierre. Por conexión: **una goroutine lectora y una única escritora** (mensajes del hub, ping periódico y frame de cierre), cada escritura con plazo propio de 10 s; un cliente que no responde termina con error de escritura y se da de baja. Mensaje máximo del cliente: 64 KiB. Sin extensiones ni subprotocolos; el servidor no fragmenta | **SSE**: más simple, pero el enunciado pide WebSocket y la rúbrica evalúa la implementación. `golang.org/x/net/websocket` o gorilla: terceros |
| Autenticar el WebSocket | El navegador no puede enviar `Authorization` en el handshake. La SPA pide `POST /api/v1/ws/ticket` (modo `bearer`, rol admin; un analista recibe 403 aquí) y obtiene un **ticket aleatorio de 32 bytes, de un solo uso, que vence a los 30 s** y guarda el rol de quien lo pidió; luego abre `GET /api/v1/ws?ticket=…` (modo `ticket`). Orden de las comprobaciones en el handshake: (1) `Origin` ausente o fuera de `CORS_ORIGENES` → **403**, *antes* de tocar el almacén de tickets, para que otro sitio no pueda quemar ni sondear un ticket (evita *cross-site WebSocket hijacking*); (2) consumo **atómico** del ticket bajo mutex (buscar, borrar y validar vencimiento en la misma sección crítica): inexistente, vencido o ya usado → **401**, así dos handshakes concurrentes con el mismo ticket no pasan ambos; (3) rol del ticket distinto de admin → **403** (defensa en profundidad: el paso de emisión ya lo filtra) | JWT en la URL: queda en logs e historial con 60 min de validez |
| Productor lento | El hub publica con envío **no bloqueante** a un canal con búfer por cliente; si se llena, descarta el mensaje y lo cuenta. El entrenamiento **nunca espera** al front. El observador solo lee copias de los pesos: no cambia el resultado (test bit a bit) | Envío bloqueante: un navegador lento frenaría la barrera de épocas |
| Despliegue del front | **Imagen `web` aparte** servida por `cmd/web` (Go stdlib, distroless) en `:8081`; la API en `:8080`. Son orígenes distintos, así que la política CORS se ejercita de verdad. `cmd/web` envía la CSP y arma su `connect-src 'self' <API_URL> <WS_URL>` a partir de `API_URL` (http→ws, https→wss), y entrega `/config.json` con `API_URL` en tiempo de ejecución | Embeber la SPA en el coordinador: mismo origen, el CORS no se probaría; nginx: otro lenguaje en el backend |

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
| `GET /api/v1/ws?ticket=…` (modo `ticket`) | 403 si el `Origin` no está en la lista; 401 sin ticket válido | 403 (el ticket guarda el rol; defensa en profundidad) | ✓ |
| `POST /api/v1/demo/nodos/{nombre}/cortar` (solo con `-demo`; sin el flag, 404) | 401 | 403 | ✓ |

Modo demo (#65): `-demo` (o `DEMO=true`) registra la ruta de corte y habilita `ritmo_ms` (pausa
de 0–2000 ms entre épocas, fuera del bucle del coordinador: no cambia el resultado). Cortar cierra
la conexión TCP del nodo; el coordinador lo descarta y reasigna por el camino normal de una caída
(ADR-0005), y un test exige que los pesos sigan iguales bit a bit al secuencial. El observador del
coordinador publica cada `cluster.Evento` en JSON al hub del WebSocket.

Swagger UI queda público para la sustentación; el OpenAPI declara `bearerAuth` y las respuestas
401/403/429, de modo que desde `/docs` se puede autorizar con un token y probar las rutas.

## Consecuencias

- El costo de verificar un JWT (un HMAC-SHA256 sobre ~300 bytes) es de microsegundos; aun así el p95
  se **vuelve a medir con token** (`cmd/benchapi` → `reports/bench_api_<maquina>_jwt.json`, #61).
- Los tokens son sin estado: cerrar sesión borra el token del navegador, pero el token sigue válido
  hasta su `exp` (60 min). Para este alcance se acepta; revocar exigiría una lista negra en Redis.
- PBKDF2 con 600 000 iteraciones cuesta del orden de cientos de milisegundos por login: es el costo
  buscado contra fuerza bruta y no afecta a la ruta de predicción.
- El servidor WebSocket es un subconjunto de RFC 6455 suficiente para navegadores actuales. Las
  **extensiones** se negocian en el handshake: los navegadores ofrecen
  `Sec-WebSocket-Extensions: permessage-deflate`, y el servidor la **declina omitiendo** esa cabecera
  en el `101` (RFC 6455 §9.1); el handshake no se rechaza por eso. Ya conectado, lo que no
  implementa se cierra con el código que corresponde, nunca se ignora en silencio: bits `RSV` ≠ 0
  sin extensión negociada, frame sin máscara u opcode reservado → `1002`; mensaje binario o
  fragmentado → `1003`; más de 64 KiB → `1009`.
- `http.Server.Shutdown` no cierra ni espera las conexiones tomadas con `Hijack`. Por eso el
  `Servidor` de la API cierra el hub en `Cerrar()` (al recibir SIGTERM): da de baja a todos los
  clientes, cada escritora manda el cierre `1001` (*going away*) y cierra su socket, y la
  lectora termina. Un test verifica que no quedan goroutines después del cierre (#64/#65).
- Las pruebas de seguridad (#70) dejan `reports/seguridad.json` con el resultado esperado vs obtenido
  de cada caso (CORS permitido/denegado, sin token, token vencido, rol ajeno, WS sin ticket, 429).
