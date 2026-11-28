# 15. Seguridad e integración entre la SPA y la API

La SPA y la API corren en orígenes distintos (`:8081` y `:8080`, sección 5), como en un despliegue real, así que la integración entre ambas tiene que resolver autenticación, autorización y política de orígenes cruzados (CORS). Todas las decisiones están en el ADR-0007 y se implementan con la biblioteca estándar de Go: JWT con `crypto/hmac`, claves con `crypto/pbkdf2` y WebSocket sobre `net/http`. Esta sección las describe (15.1 a 15.4) y presenta las pruebas que se hicieron desde fuera contra el sistema desplegado (15.5).

## 15.1 JWT, PBKDF2 y roles

**Inicio de sesión.** `POST /api/v1/auth/login` recibe correo y clave y, si son correctos, devuelve un **JWT** (*JSON Web Token*) firmado con HMAC-SHA256 que vence a los 60 minutos. El token lleva el correo (`sub`), el rol, el emisor y la audiencia, y la SPA lo envía en la cabecera `Authorization: Bearer <token>` de cada petición. Nunca se usa en *cookies*: así no hay riesgo de CSRF, porque el navegador no envía el token por su cuenta a otro sitio.

**Verificación del token.** `Verificar` rechaza cualquier token que no sea exactamente uno emitido por el servidor:

- el algoritmo es fijo (`HS256`): un token con `alg: none` o con otro algoritmo se rechaza, lo que evita el ataque clásico de cambiar el algoritmo en la cabecera;
- la firma se compara en tiempo constante (`hmac.Equal`), para no filtrar información por el tiempo de respuesta;
- se exigen el emisor, la audiencia, un rol conocido y que no haya vencido;
- el base64url se decodifica en modo estricto, de modo que cada token válido tiene una sola representación.

<!-- generado:fuente:internal/auth/jwt.go#(*Firmador).Verificar -->
```go
// internal/auth/jwt.go
// Verificar comprueba estructura, cabecera, firma (en tiempo constante),
// vigencia, emisor, audiencia y rol, en ese orden. La cabecera tiene que ser
// exactamente la que emite Firmar y el base64url se decodifica en modo
// estricto (sin bits sobrantes ni saltos de línea): cada token válido tiene una
// sola representación.
func (f *Firmador) Verificar(token string) (Reclamos, error) {
	if strings.ContainsAny(token, "\r\n") {
		return Reclamos{}, ErrMalformado
	}
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return Reclamos{}, ErrMalformado
	}
	if partes[0] != cabeceraB64 {
		return Reclamos{}, errorDeCabecera(partes[0])
	}
	firma, err := b64.DecodeString(partes[2])
	if err != nil {
		return Reclamos{}, ErrMalformado
	}
	if !hmac.Equal(firma, f.firma(partes[0]+"."+partes[1])) {
		return Reclamos{}, ErrFirma
	}
	carga, err := b64.DecodeString(partes[1])
	if err != nil {
		return Reclamos{}, ErrMalformado
	}
	var rec Reclamos
	if err := json.Unmarshal(carga, &rec); err != nil {
		return Reclamos{}, ErrMalformado
	}
	ahora := f.ahora().Unix()
	switch {
	case ahora >= rec.Exp:
		return Reclamos{}, ErrVencido
	case ahora < rec.Nbf:
		return Reclamos{}, ErrNoVigente
	case rec.Iss != Emisor || rec.Aud != Audiencia || rec.Sub == "" || !rec.Rol.Valido():
		return Reclamos{}, ErrReclamos
	}
	return rec, nil
}
```
<!-- /generado -->

El secreto de firma (`JWT_SECRETO`, al menos 32 bytes) se pasa por variable de entorno desde `deploy/.env`, que no se versiona. Sin él, el coordinador no arranca en modo servicio y las rutas protegidas responden 503: la API falla cerrada, nunca queda abierta.

**Claves.** Las claves de los usuarios se guardan como hash **PBKDF2-HMAC-SHA256** con sal aleatoria de 16 bytes y 600 000 iteraciones, la cantidad que recomienda OWASP para este algoritmo. La comparación es en tiempo constante, y un correo inexistente se compara contra un hash señuelo, de modo que la respuesta tarda lo mismo y dice lo mismo («credenciales incorrectas») exista o no el correo: no se puede averiguar qué correos están registrados.

<!-- generado:fuente:internal/auth/clave.go#Comparar -->
```go
// internal/auth/clave.go
// Comparar indica si clave corresponde al hash guardado. La comparación final es
// en tiempo constante; un hash mal formado nunca verifica.
func Comparar(guardado, clave string) bool {
	if strings.ContainsAny(guardado, "\r\n") {
		return false
	}
	partes := strings.Split(guardado, "$")
	if len(partes) != 4 || partes[0] != prefijo {
		return false
	}
	iter, err := strconv.Atoi(partes[1])
	if err != nil || iter < 1 || iter > maxIteraciones {
		return false
	}
	sal, err1 := b64Clave.DecodeString(partes[2])
	esperado, err2 := b64Clave.DecodeString(partes[3])
	// Largos fijos: una clave derivada más larga multiplicaría el costo
	// (PBKDF2 hace iter × ⌈largo/32⌉ rondas) y una truncada sería trivial de adivinar.
	if err1 != nil || err2 != nil || len(sal) != bytesSal || len(esperado) != bytesClave {
		return false
	}
	k, err := pbkdf2.Key(sha256.New, clave, sal, iter, bytesClave)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(k, esperado) == 1
}
```
<!-- /generado -->

**Roles.** Hay dos: **analista**, que consulta el riesgo, predice casos y ve la evaluación; y **admin**, que además ve el clúster y lanza entrenamientos. El rol de cada ruta está en `api.Rutas()` (columna «Rol» de la @tab:contratos). `Autenticar` valida el token y guarda sus datos en el contexto de la petición; `Requiere` comprueba el rol y responde 401 sin token válido y 403 con un rol sin acceso. `TestMatrizRoles` prueba cada ruta con cada rol y sin token.

<!-- generado:fuente:internal/auth/middleware.go#Requiere -->
```go
// internal/auth/middleware.go
// Requiere deja pasar solo a los roles dados: sin reclamos en el contexto → 401;
// rol no incluido → 403. Se coloca detrás de Autenticar.
func Requiere(roles ...Rol) func(http.Handler) http.Handler {
	permitidos := slices.Clone(roles) // el llamador no puede cambiar la política después
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec, ok := ReclamosDe(r.Context())
			if !ok {
				noAutenticado(w, `Bearer realm="vigia"`, "falta el token de acceso")
				return
			}
			if !slices.Contains(permitidos, rec.Rol) {
				escribirError(w, http.StatusForbidden, "tu rol no tiene acceso a este recurso")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```
<!-- /generado -->

Los usuarios se crean con `cmd/usuarios`, que lee correos y claves de variables de entorno: ninguna clave está en el repositorio.

## 15.2 CORS, CSP y cabeceras de seguridad

**CORS.** El navegador solo deja que la SPA de `:8081` lea las respuestas de la API en `:8080` si la API lo autoriza. El *middleware* `cors` compara el encabezado `Origin` con una lista blanca exacta (`CORS_ORIGENES`, validada al arrancar): si coincide, responde con ese origen en `Access-Control-Allow-Origin`; si no, no agrega nada y el navegador bloquea la respuesta. Nunca usa `*` ni `Allow-Credentials`, y agrega `Vary: Origin` para que ninguna caché intermedia sirva una respuesta de un origen a otro. El *preflight* (`OPTIONS`) se responde antes de la autenticación, porque el navegador lo envía sin el token.

<!-- generado:fuente:internal/api/seguridad.go#(*Servidor).cors -->
```go
// internal/api/seguridad.go
// cors decora toda respuesta (2xx, 4xx y 5xx) de un origen permitido y
// responde el preflight aquí, antes de la autenticación y del 405 del mux: el
// navegador no manda Authorization en él. Nunca «*» ni Allow-Credentials.
func (s *Servidor) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")
		origen := r.Header.Get("Origin")
		permitido := origen != "" && slices.Contains(s.op.OrigenesCORS, origen)
		if permitido {
			h.Set("Access-Control-Allow-Origin", origen)
			h.Set("Access-Control-Expose-Headers", corsExpuestas)
		}
		if r.Method == http.MethodOptions && origen != "" && r.Header.Get("Access-Control-Request-Method") != "" {
			if permitido {
				h.Set("Access-Control-Allow-Methods", corsMetodos)
				h.Set("Access-Control-Allow-Headers", corsCabeceras)
				h.Set("Access-Control-Max-Age", corsMaxAge)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```
<!-- /generado -->

**Cabeceras de la API.** Toda respuesta lleva `X-Content-Type-Options: nosniff` (el navegador no reinterpreta el tipo de contenido) y `Referrer-Policy: no-referrer`; las respuestas autenticadas, `Cache-Control: no-store`, para que ninguna caché guarde datos de un usuario.

**CSP de la SPA.** `cmd/web`, el servidor de la SPA, envía una *Content Security Policy* estricta: `default-src 'self'`, scripts y estilos solo del propio origen, sin `unsafe-inline` ni `unsafe-eval`, conexiones solo a la API configurada y `frame-ancestors 'none'` (la SPA no se puede incrustar en otra página). Si un atacante lograra inyectar HTML, el navegador no ejecutaría sus scripts. La URL de la API se valida al arrancar para que no pueda colar otra directiva en la política.

<!-- generado:fuente:cmd/web/main.go#nuevoServidor -->
```go
// cmd/web/main.go
// nuevoServidor valida la carpeta y la URL de la API y arma el handler.
func nuevoServidor(dir, apiURL string) (http.Handler, error) {
	raiz, err := os.OpenRoot(dir) // ninguna ruta pedida puede salir de dir (ni con «\» en Windows)
	if err != nil {
		return nil, fmt.Errorf("web: %w", err)
	}
	if info, err := raiz.Stat("index.html"); err != nil || info.IsDir() {
		return nil, fmt.Errorf("web: falta index.html en %s", dir)
	}
	origenAPI, err := validarAPI(apiURL)
	if err != nil {
		return nil, err
	}
	origenWS := strings.Replace(origenAPI, "http", "ws", 1) // http→ws, https→wss
	config, _ := json.Marshal(map[string]string{"api_url": origenAPI})
	csp := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src 'self' " + origenAPI + " " + origenWS,
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cab := w.Header()
		cab.Set("Content-Security-Policy", csp)
		cab.Set("X-Content-Type-Options", "nosniff")
		cab.Set("Referrer-Policy", "no-referrer")
		cab.Set("X-Frame-Options", "DENY")
		cab.Set("Cross-Origin-Opener-Policy", "same-origin")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			cab.Set("Allow", "GET, HEAD")
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		limpia := path.Clean("/" + r.URL.Path)
		if limpia == "/config.json" {
			cab.Set("Content-Type", "application/json; charset=utf-8")
			cab.Set("Cache-Control", "no-store")
			_, _ = w.Write(config)
			return
		}
		if !strings.ContainsRune(limpia, '\\') && limpia != "/" {
			if servirArchivo(w, r, raiz, strings.TrimPrefix(limpia, "/")) {
				return
			}
		}
		// Un archivo con extensión que no existe es un 404 real (sin caché larga);
		// una ruta sin extensión (/mapa, /admin) es del router del SPA → index.html.
		if path.Ext(limpia) != "" {
			http.NotFound(w, r)
			return
		}
		servirArchivo(w, r, raiz, "index.html")
	}), nil
}
```
<!-- /generado -->

La sesión se guarda en `sessionStorage`, que se borra al cerrar la pestaña.

## 15.3 WebSocket con ticket de un uso

El panel de administración recibe los eventos del entrenamiento por WebSocket (sección 16.5). El protocolo se implementó con la biblioteca estándar, sobre `net/http` y `Hijack` (subconjunto de RFC 6455: *handshake*, *frames* de texto, *ping/pong* y cierre), porque el enunciado pide WebSocket y no hay un paquete estándar que lo implemente.

**El problema.** El navegador no permite enviar la cabecera `Authorization` al abrir un WebSocket, así que el JWT no puede viajar como en las demás peticiones. Ponerlo en la URL lo dejaría en los registros del servidor y del *proxy*.

**La solución: un ticket de un uso.**

1. La SPA pide `POST /api/v1/ws/ticket` con su JWT (solo admin) y recibe un ticket aleatorio que vence a los 30 segundos.
2. Abre `GET /api/v1/ws?ticket=…`. El servidor **consume** el ticket (un segundo uso da 401) y además comprueba que el `Origin` del navegador esté en la lista de CORS (403 si no). Esta segunda comprobación evita el *Cross-Site WebSocket Hijacking*: una página de otro sitio no puede abrir el WebSocket con la sesión del usuario.
3. Si la conexión se corta, la SPA pide un ticket **nuevo** para reconectarse, porque el anterior ya se gastó.

<!-- generado:fuente:internal/api/tiemporeal.go#(*Servidor).conTicket -->
```go
// internal/api/tiemporeal.go
// conTicket autentica el handshake del WebSocket (modo Ticket, ADR-0007). El
// orden importa: (1) Origin fuera de la lista → 403 antes de tocar los
// tickets, para que otro sitio no pueda quemar ni sondear uno (cross-site
// WebSocket hijacking); (2) ticket inexistente, vencido o usado → 401;
// (3) rol sin acceso → 403 (defensa en profundidad: la emisión ya lo filtra).
func (s *Servidor) conTicket(h http.Handler, roles []auth.Rol) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respuestas de error del handshake (401/403/400/426): como toda respuesta
		// autenticada, sin copia en cachés (PR-84 H3). No afecta al 101.
		w.Header().Set("Cache-Control", "no-store")
		if !s.origenPermitido(r.Header.Get("Origin")) {
			responderError(w, http.StatusForbidden, "origen no permitido para el WebSocket")
			return
		}
		rec, ok := s.tickets.Consumir(r.URL.Query().Get("ticket"))
		if !ok {
			responderError(w, http.StatusUnauthorized, "ticket inválido, vencido o ya usado: pide otro en POST /api/v1/ws/ticket")
			return
		}
		if !slices.Contains(roles, rec.Rol) {
			responderError(w, http.StatusForbidden, "tu rol no tiene acceso a este recurso")
			return
		}
		h.ServeHTTP(w, r.WithContext(auth.ConReclamos(r.Context(), rec)))
	})
}
```
<!-- /generado -->

**ws:// frente a wss://.** En el despliegue local la conexión es `ws://`, sin cifrado, igual que la API usa `http://`. En un despliegue en internet, la SPA, la API y el WebSocket irían detrás de un *proxy* con TLS (`https://` y `wss://`): sin cifrado, el JWT y el ticket viajarían en texto plano y podrían capturarse en la red. El código no cambia, porque la SPA arma la URL del WebSocket a partir de la de la API (`http` → `ws`, `https` → `wss`), y el navegador rechaza abrir `ws://` desde una página servida por `https://`.

## 15.4 Límites de intentos y de tamaño

- **Intentos de login.** Tras 5 intentos fallidos en un minuto desde la misma IP, el siguiente recibe 429 con `Retry-After`, sin consultar la base. Un login correcto borra los fallos de esa IP. La comprobación y la reserva del intento ocurren en la misma sección crítica, así que varias peticiones simultáneas no pueden pasar el límite juntas. El mapa de IPs tiene un tamaño máximo, para que no crezca sin límite ante muchas IPs distintas.
- **Tamaño del cuerpo.** Todo POST se lee con `http.MaxBytesReader` hasta 64 KiB; más allá, 413. Evita que un cliente agote la memoria del servidor con un cuerpo enorme.
- **Tamaño de los mensajes del WebSocket.** El servidor rechaza *frames* sin máscara del cliente y mensajes de más de 64 KiB, cerrando con el código de RFC 6455 correspondiente.

<!-- generado:fuente:internal/api/limite.go#(*limitador).Reservar -->
```go
// internal/api/limite.go
// Reservar decide si la IP puede intentar ahora. Si puede, reserva un cupo y
// devuelve la función que lo libera con el resultado del intento (llamarla
// exactamente una vez). Si no, devuelve cuánto esperar: hasta que salga de la
// ventana el fallo que hace sobrar, o 1 s si los cupos están ocupados por
// intentos que aún no terminan.
func (l *limitador) Reservar(ip string) (liberar func(resultadoIntento), espera time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.ahora()
	f := l.vigentes(ip, t)
	if len(f)+l.enCurso[ip] >= l.max {
		if len(f) >= l.max {
			return nil, f[len(f)-l.max].Add(l.ventana).Sub(t)
		}
		return nil, time.Second
	}
	l.enCurso[ip]++
	var una sync.Once
	return func(r resultadoIntento) { una.Do(func() { l.liberar(ip, r) }) }, 0
}
```
<!-- /generado -->

## 15.5 Evidencias

`cmd/seguridad` prueba cada control desde fuera, contra el sistema desplegado, como lo haría un atacante o un navegador: con tokens válidos, alterados, vencidos o firmados con otro secreto, con cada rol, con el origen de la SPA y con uno ajeno, y con el WebSocket con y sin ticket. Para cada caso registra lo esperado y lo obtenido (@tab:seguridad). Los tokens vencidos o con otro secreto los fabrica el propio programa con el mismo formato que la API.

<!-- generado:plantilla:seguridad -->
Tabla: Controles de seguridad probados desde fuera contra el sistema en ejecución {#tab:seguridad}

| # | Caso | Petición | Esperado | Obtenido | Cumple |
|---:|---|---|---|---|:---:|
| — | Se genera en el paso 13 de la guía de #70 | — | — | — | — |

Nota: Falta `reports/seguridad.json`; `cmd/informe` completa esta tabla al regenerar el informe.

<!-- falta: reports/seguridad.json -->
<!-- /generado -->

<!-- evidencia: evidencias/seguridad.txt -->

Desde la SPA, las pruebas de punta a punta de la sección 16 comprueban los mismos controles con un navegador real: el login fallido muestra un mensaje genérico (@fig:login) y el analista que intenta entrar al panel de administración recibe la pantalla 403 (@fig:analista-403).
