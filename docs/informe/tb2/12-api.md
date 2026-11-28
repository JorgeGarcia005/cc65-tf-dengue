# 12. Parte C: API REST

El coordinador, en modo servicio, expone una API REST que usan la SPA (sección 16), Postman y los guiones de evidencia. Esta sección responde a la observación 3 de la PC4 («mejorar la explicación del código de la API y los contratos *request/response* de los endpoints; ¿cuáles son los datos del set de pruebas?»): cómo se arma el servidor (12.1), el contrato de cada endpoint con sus tipos en Go (12.2), la documentación OpenAPI (12.3), los datos de prueba (12.4) y las peticiones y respuestas reales del sistema desplegado (12.5). La latencia se mide en la sección 13, las bases de datos se describen en la 14 y los controles de seguridad en la 15.

## 12.1 Servidor HTTP y cadena de middlewares

La API usa solo `net/http` de la biblioteca estándar. Todas las rutas salen de una tabla única, `api.Rutas()`: de ella se arman el enrutador y la autorización de cada ruta, y contra ella comparan los tests la documentación OpenAPI y la matriz de roles. Así, una ruta no puede existir sin estar documentada ni quedar sin su regla de acceso.

<!-- generado:fuente:internal/api/api.go#Rutas -->
```go
// internal/api/api.go
// Rutas es la tabla única de la que salen el mux, la autorización y los tests
// del OpenAPI y de la matriz de roles.
func Rutas() []Ruta {
	pub := func(metodo, ruta string, doc bool, h func(*Servidor, http.ResponseWriter, *http.Request)) Ruta {
		return Ruta{Metodo: metodo, Ruta: ruta, Documentar: doc, Modo: Publica, manejar: h}
	}
	demo := func(r Ruta) Ruta { r.SoloDemo = true; return r }
	con := func(metodo, ruta string, roles []auth.Rol, h func(*Servidor, http.ResponseWriter, *http.Request)) Ruta {
		// Copia: quien recibe Rutas() no puede cambiar la política (PR-83 H3).
		return Ruta{Metodo: metodo, Ruta: ruta, Documentar: true, Modo: Bearer, Roles: slices.Clone(roles), manejar: h}
	}
	return []Ruta{
		pub("GET", "/healthz", true, (*Servidor).salud),
		pub("POST", "/api/v1/auth/login", true, (*Servidor).login),
		con("GET", "/api/v1/modelo", rolesTodos, (*Servidor).verModelo),
		con("POST", "/api/v1/predicciones", rolesTodos, (*Servidor).predecir),
		con("GET", "/api/v1/riesgo", rolesTodos, (*Servidor).verRiesgo),
		con("GET", "/api/v1/evaluacion", rolesTodos, (*Servidor).verEvaluacion),
		con("GET", "/api/v1/cluster", soloAdmin, (*Servidor).verCluster),
		con("POST", "/api/v1/entrenamientos", soloAdmin, (*Servidor).crearEntrenamiento),
		con("GET", "/api/v1/entrenamientos/{id}", soloAdmin, (*Servidor).verEntrenamiento),
		con("POST", "/api/v1/ws/ticket", soloAdmin, (*Servidor).emitirTicket),
		{Metodo: "GET", Ruta: "/api/v1/ws", Documentar: true, Modo: Ticket, Roles: slices.Clone(soloAdmin), manejar: (*Servidor).servirWS},
		demo(con("POST", "/api/v1/demo/nodos/{nombre}/cortar", soloAdmin, (*Servidor).cortarNodo)),
		pub("GET", "/docs", false, (*Servidor).swaggerUI),
		pub("GET", "/docs/openapi.yaml", false, (*Servidor).openapi),
	}
}
```
<!-- /generado -->

`Handler` envuelve el enrutador en una cadena de *middlewares*. Cada petición los atraviesa de afuera hacia adentro:

1. **`recuperar`:** convierte un pánico en una respuesta 500 sin detener el proceso.
2. **`cabecerasSeguridad`:** agrega `X-Content-Type-Options` y `Referrer-Policy` a toda respuesta y limita el cuerpo de todo POST a 64 KiB con `http.MaxBytesReader` (413 si se excede).
3. **`cors`:** responde el *preflight* `OPTIONS` y agrega `Access-Control-Allow-Origin` solo para los orígenes permitidos (sección 15.2).
4. **Enrutador** (`http.ServeMux` con patrones que incluyen el método, como `"POST /api/v1/predicciones"`): elige la ruta o responde 404 o 405.
5. **Autenticación de la ruta:** `proteger` exige un JWT válido y uno de los roles de la ruta (401 o 403); `conTicket` exige el ticket del WebSocket (sección 15).
6. **Handler** de la ruta.

<!-- generado:fuente:internal/api/api.go#(*Servidor).Handler -->
```go
// internal/api/api.go
// Handler arma la cadena de ADR-0007, de afuera hacia adentro: recuperar →
// cabeceras de seguridad → CORS → Opciones.Middleware → mux → autenticación
// y rol de la ruta → handler. La autenticación va por ruta (después de que el
// mux la elige) para que cada una aplique su propio modo.
func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	for _, r := range Rutas() {
		if r.SoloDemo && s.op.Demo == nil {
			continue // sin -demo la ruta no existe: 404 del mux
		}
		manejar := r.manejar
		var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { manejar(s, w, req) })
		switch r.Modo {
		case Bearer:
			h = s.proteger(h, r.Roles)
		case Ticket:
			h = s.conTicket(h, r.Roles)
		}
		mux.Handle(r.Metodo+" "+r.Ruta, h)
	}
	var h http.Handler = mux
	for i := len(s.op.Middleware) - 1; i >= 0; i-- {
		h = s.op.Middleware[i](h)
	}
	return s.recuperar(cabecerasSeguridad(s.cors(h)))
}
```
<!-- /generado -->

La autenticación va **después** del enrutador porque cada ruta tiene su propio modo (pública, JWT o ticket); CORS va **antes**, porque el navegador envía el *preflight* sin el token y debe recibir la respuesta aunque la ruta sea protegida.

**Diseño para responder rápido.** `predecir`, el endpoint que se mide en la sección 13, sigue cuatro decisiones:

- **Lectura sin bloqueos del modelo.** El modelo publicado vive en un `atomic.Pointer`. Cada predicción lo lee una sola vez, sin mutex, y un entrenamiento nuevo reemplaza el puntero completo al terminar. Mientras el clúster entrena, las predicciones siguen respondiendo con el modelo anterior.
- **Caché con presupuesto de tiempo** (*cache-aside*). Antes de calcular, busca la predicción en Redis con un plazo fijo; si Redis está lento o caído, calcula igual y responde. La clave incluye la versión del modelo (sección 14.2).
- **Registro sin espera.** El registro de la predicción en MongoDB se encola en un canal con búfer y lo escribe otra goroutine; la respuesta no espera a la base (sección 14.3).
- **Validación con las reglas de la limpieza.** La consulta se valida y transforma con el mismo esquema de variables del modelo publicado, así que la API predice exactamente lo que el modelo aprendió.

<!-- generado:fuente:internal/api/api.go#(*Servidor).predecir -->
```go
// internal/api/api.go
func (s *Servidor) predecir(w http.ResponseWriter, r *http.Request) {
	inicio := time.Now()
	var c Consulta
	if !leerJSON(w, r, &c, false) {
		return
	}
	m := s.ModeloActual() // una sola lectura: clave, cálculo y registro usan el mismo modelo
	reg, err := m.registro(c)
	if err != nil {
		responderError(w, http.StatusBadRequest, err.Error())
		return
	}
	cn := normalizada(reg)
	clave := almacen.ClavePrediccion(m.Version, cn)
	ctx := r.Context()

	// Cache-aside (ADR-0006) con presupuesto fijo (PR-40 H5): un fallo o una
	// demora de la caché no falla la predicción ni consume los 100 ms.
	cacheSana := s.op.Cache != nil
	if cacheSana {
		ctxLeer, cancelar := context.WithTimeout(ctx, PresupuestoLecturaCache)
		b, ok, err := s.op.Cache.Obtener(ctxLeer, clave)
		cancelar()
		if err != nil {
			cacheSana = false // no se intenta una segunda espera para guardar
			s.op.Log("api: caché no disponible al leer: %v", err)
		}
		var p Prediccion
		// Una entrada guardada antes de #67 no trae contribuciones: se recalcula
		// y se reemplaza, en vez de servirla incompleta.
		if ok && json.Unmarshal(b, &p) == nil && len(p.Contribuciones) > 0 {
			w.Header().Set("X-Cache", "HIT")
			responderBytes(w, http.StatusOK, b)
			s.latencias.anotar(time.Since(inicio), true)
			s.registrar(registroDe(cn, p, true, inicio))
			return
		}
	}
	p := m.prediccion(reg)
	b, err := json.Marshal(p)
	if err != nil {
		responderError(w, http.StatusInternalServerError, "error interno al codificar la respuesta")
		return
	}
	if s.op.Cache != nil {
		if cacheSana {
			ctxGuardar, cancelar := context.WithTimeout(ctx, PresupuestoEscrituraCache)
			if err := s.op.Cache.Guardar(ctxGuardar, clave, b, almacen.TTLPrediccion); err != nil {
				s.op.Log("api: caché no disponible al guardar: %v", err)
			}
			cancelar()
		}
		w.Header().Set("X-Cache", "MISS")
	}
	responderBytes(w, http.StatusOK, b)
	s.latencias.anotar(time.Since(inicio), false)
	s.registrar(registroDe(cn, p, false, inicio))
}
```
<!-- /generado -->

El cálculo es el mismo producto escalar y la misma sigmoide del entrenamiento (sección 8.2), con la corrección de *prior* para devolver el riesgo en la escala de las tasas observadas, y la contribución de cada variable al logit, que la SPA muestra al analista:

<!-- generado:fuente:internal/api/modelo.go#(*Modelo).prediccion -->
```go
// internal/api/modelo.go
func (m *Modelo) prediccion(r ingest.Registro) Prediccion {
	d := m.Esquema.D()
	x := make([]float64, d)
	m.Esquema.Llenar(r, x)
	z := m.Pesos[d]
	contribuciones := make([]Contribucion, d)
	for j, v := range x {
		// La conversión explícita redondea el producto (la especificación de Go
		// permite fusionar x*y+z en una FMA): así Σ aportes reproduce z exacto.
		aporte := float64(m.Pesos[j] * v)
		if aporte == 0 {
			aporte = 0 // -0 (peso negativo × 0) no cambia z, pero ensucia el JSON (PR-85 H3)
		}
		z += aporte
		contribuciones[j] = Contribucion{Variable: m.Esquema.Nombres[j], Valor: v, Peso: m.Pesos[j], Aporte: aporte}
	}
	p := ml.Sigmoide(z)
	pr := Prediccion{
		Logit: z, Sesgo: m.Pesos[d], Contribuciones: contribuciones,
		Riesgo:                ml.Calibrar(p, m.AjustePrior),
		ProbabilidadPonderada: p,
		Umbral:                m.Umbral,
		Etiqueta:              "sin signos de alarma",
		DepartamentoModelo:    features.OtrosDepartamentos,
		VersionModelo:         m.Version,
	}
	if p >= m.Umbral {
		pr.Clase, pr.Etiqueta = 1, "con signos de alarma o grave"
	}
	for _, dep := range m.Esquema.Departamentos {
		if dep == r.Departamento {
			pr.DepartamentoModelo = dep
		}
	}
	return pr
}
```
<!-- /generado -->

## 12.2 Contratos de los endpoints

La @tab:contratos resume el contrato de cada endpoint: quién puede llamarlo, qué recibe, qué devuelve y con qué códigos responde. Se genera desde la tabla de rutas del código y desde el OpenAPI, de modo que no puede quedar desactualizada respecto de ninguno de los dos.

<!-- generado:contratos -->
Tabla: Contratos de los endpoints de la API {#tab:contratos}

| Método y ruta | Rol | Qué hace | Cuerpo de la petición | Respuesta correcta | Códigos documentados |
|---|---|---|---|---|---|
| `GET /healthz` | público | Comprobación de vida | — | — | 200 |
| `POST /api/v1/auth/login` | público | Inicia sesión y devuelve un JWT (ADR-0007) | `Login` | `Sesion` | 200, 400, 401, 413, 429, 503 |
| `GET /api/v1/modelo` | analista, admin | Modelo publicado (pesos por variable, umbral y versión) | — | `Modelo` | 200, 401, 403 |
| `POST /api/v1/predicciones` | analista, admin | Riesgo de dengue con signos de alarma o grave para un caso | `Consulta` | `Prediccion` | 200, 400, 401, 403, 413 |
| `GET /api/v1/riesgo` | analista, admin | Riesgo cruzado por departamento, semana y grupo de edad (mapa) | — | `DocumentoEvaluacion` | 200, 304, 401, 403, 503 |
| `GET /api/v1/evaluacion` | analista, admin | Evaluación experimental completa (reports/evaluacion.json) | — | `DocumentoEvaluacion` | 200, 304, 401, 403, 503 |
| `GET /api/v1/cluster` | admin | Estado del clúster y métricas del panel admin (nodos, carga, CPU, latencia, época y pérdida) | — | `EstadoCluster` | 200, 401, 403, 503 |
| `POST /api/v1/entrenamientos` | admin | Lanza un entrenamiento distribuido | `ParametrosEntrenamiento` | `Entrenamiento` | 202, 400, 401, 403, 409, 413, 503 |
| `GET /api/v1/entrenamientos/{id}` | admin | Estado de un entrenamiento | — | `Entrenamiento` | 200, 401, 403, 404 |
| `POST /api/v1/ws/ticket` | admin | Ticket de un solo uso para abrir el WebSocket (30 s) | — | — | 200, 401, 403, 413, 429 |
| `GET /api/v1/ws` | admin (ticket) | WebSocket con los eventos del entrenamiento en vivo | — | — | 101, 401, 403 |
| `POST /api/v1/demo/nodos/{nombre}/cortar` | admin | Corta la conexión de un nodo (solo con -demo) | — | — | 202, 401, 403, 404, 413 |

Nota: Rol y modo de autenticación tomados de `api.Rutas()`, la tabla de la que salen el enrutador y la autorización; resumen, esquemas y códigos, de `docs/api/openapi.yaml`. «Ticket»: el WebSocket se autentica con un ticket de un uso (sección 15.3). Los esquemas están en `components/schemas` del OpenAPI y en los `struct` de Go de la sección 12.2. Generado por `cmd/informe`.
<!-- /generado -->

Todos los errores tienen la misma forma, `{"error": "<mensaje en español>"}`, con el código HTTP que corresponde: 400 si la consulta es inválida (por ejemplo, un departamento que no existe o una semana fuera de 1–53), 401 sin token o con uno inválido, 403 si el rol no tiene acceso, 404 si el recurso no existe, 409 si ya hay un entrenamiento en curso, 413 si el cuerpo pasa de 64 KiB, 429 si se superan los intentos de login y 503 si el servicio corre sin clúster o sin secreto JWT.

**Tipos de la petición y la respuesta.** Los esquemas del OpenAPI corresponden a estos `struct` de Go, que `encoding/json` lee y escribe directamente. La consulta de una predicción:

<!-- generado:fuente:internal/api/modelo.go#Consulta -->
```go
// internal/api/modelo.go
// Consulta es el cuerpo de POST /api/v1/predicciones. Usa los mismos campos y
// reglas que el CSV del MINSA (ingest.LimpiarCampos).
type Consulta struct {
	Departamento string   `json:"departamento"`
	Ano          int      `json:"ano"`
	Semana       int      `json:"semana"`
	Edad         *float64 `json:"edad"`      // puntero: distingue 0 (recién nacido) de ausente o null
	TipoEdad     string   `json:"tipo_edad"` // A (años, por defecto), M (meses) o D (días)
	Sexo         string   `json:"sexo"`
}
```
<!-- /generado -->

La respuesta de una predicción:

<!-- generado:fuente:internal/api/modelo.go#Prediccion -->
```go
// internal/api/modelo.go
// Prediccion es la respuesta de POST /api/v1/predicciones.
type Prediccion struct {
	// Riesgo es la probabilidad calibrada de dengue con signos de alarma o
	// grave: la escala de la tasa observada.
	Riesgo float64 `json:"riesgo"`
	// ProbabilidadPonderada es σ(w·x+b), la salida del modelo entrenado con
	// pesos por clase; es la que se compara con el umbral.
	ProbabilidadPonderada float64 `json:"probabilidad_ponderada"`
	Clase                 int     `json:"clase"`
	Etiqueta              string  `json:"etiqueta"`
	Umbral                float64 `json:"umbral"`
	DepartamentoModelo    string  `json:"departamento_modelo"`
	VersionModelo         string  `json:"version_modelo"`
	// Logit es z = b + Σ w_j·x_j; ProbabilidadPonderada = σ(Logit). Sesgo más
	// los aportes, sumados en el orden de Contribuciones, dan Logit bit a bit
	// (es la misma suma que hace el modelo, #67).
	Logit          float64        `json:"logit"`
	Sesgo          float64        `json:"sesgo"`
	Contribuciones []Contribucion `json:"contribuciones"`
}
```
<!-- /generado -->

El inicio de sesión devuelve el token y su vencimiento:

<!-- generado:fuente:internal/api/login.go#RespuestaLogin -->
```go
// internal/api/login.go
// RespuestaLogin es el contrato que espera la SPA (#62).
type RespuestaLogin struct {
	Token  string `json:"token"`
	Correo string `json:"correo"`
	Rol    string `json:"rol"`
	Expira int64  `json:"expira"` // segundos Unix
}
```
<!-- /generado -->

Y un entrenamiento se lanza con estos parámetros (todos opcionales: si faltan, se usan los valores por defecto del servidor, y `ritmo_ms`, una pausa entre épocas para la demostración, solo se acepta en modo demo):

<!-- generado:fuente:internal/api/entrenamientos.go#ParametrosEntrenamiento -->
```go
// internal/api/entrenamientos.go
// ParametrosEntrenamiento es el cuerpo de POST /api/v1/entrenamientos. D lo
// completa la API con el número de variables del modelo publicado.
type ParametrosEntrenamiento struct {
	Epocas int     `json:"epocas"`
	Tasa   float64 `json:"tasa"`
	Lambda float64 `json:"lambda"`
	// RitmoMs es la pausa entre épocas, solo en modo demo (#65): deja ver el
	// entrenamiento en vivo. No cambia el resultado numérico.
	RitmoMs int `json:"ritmo_ms,omitempty"`
	D       int `json:"-"`
}
```
<!-- /generado -->

**Lectura estricta del cuerpo.** `leerJSON` rechaza con 400 un cuerpo que no es JSON, que trae campos desconocidos (`DisallowUnknownFields`) o que tiene más de un objeto, y con 413 uno demasiado grande. El mensaje de error dice qué campo falló y de qué tipo debía ser, en español:

<!-- generado:fuente:internal/api/api.go#leerJSON -->
```go
// internal/api/api.go
// leerJSON decodifica el cuerpo de forma estricta y responde el error en
// español. Devuelve false si ya respondió. Si opcional, un cuerpo vacío es
// válido y deja v como estaba (con sus valores por defecto).
func leerJSON(w http.ResponseWriter, r *http.Request, v any, opcional bool) bool {
	// Se lee el cuerpo entero (≤ MaxCuerpo) antes de decodificar: así un cuerpo
	// grande da 413 aunque no sea JSON. Si se decodificara en flujo, el error de
	// sintaxis del primer byte saldría antes de llegar al límite (PR-83 H2).
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxCuerpo))
	if err != nil {
		var grande *http.MaxBytesError
		if errors.As(err, &grande) {
			responderError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("el cuerpo supera %d bytes", MaxCuerpo))
		} else {
			responderError(w, http.StatusBadRequest, "no se pudo leer el cuerpo")
		}
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(cuerpo))
	dec.DisallowUnknownFields()
	err = dec.Decode(v)
	if err == nil {
		// Decoder.More solo mira dentro de un arreglo u objeto: para exigir que
		// el documento terminó hay que intentar leer otro valor y obtener EOF.
		var resto json.RawMessage
		if dec.Decode(&resto) != io.EOF {
			responderError(w, http.StatusBadRequest, "hay contenido después del objeto JSON")
			return false
		}
		return true
	}
	if errors.Is(err, io.EOF) && opcional {
		return true // sin cuerpo: v conserva los valores por defecto
	}
	var (
		sintaxis *json.SyntaxError
		tipo     *json.UnmarshalTypeError
		msg      string
	)
	switch {
	case errors.As(err, &sintaxis), errors.Is(err, io.ErrUnexpectedEOF):
		msg = "JSON mal formado"
	case errors.Is(err, io.EOF):
		msg = "falta el cuerpo JSON"
	case errors.As(err, &tipo):
		msg = fmt.Sprintf("el campo %q debe ser de tipo %s", tipo.Field, nombreTipo(tipo.Type.Kind().String()))
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		msg = "campo desconocido " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	default:
		msg = "cuerpo inválido: " + err.Error()
	}
	responderError(w, http.StatusBadRequest, msg)
	return false
}
```
<!-- /generado -->

## 12.3 OpenAPI y Swagger

El contrato completo está en `docs/api/openapi.yaml` (OpenAPI 3.0.3; OpenAPI Initiative, 2020): rutas, parámetros, esquemas de petición y respuesta con ejemplos, códigos de error y el esquema de seguridad `bearerAuth`. El coordinador lo sirve en `/docs/openapi.yaml` y lo muestra con Swagger UI en `/docs` (@fig:swagger). Tres controles lo mantienen al día:

- `TestOpenAPICubreLasRutas` compara las rutas del OpenAPI con `api.Rutas()`: falla si una ruta del código no está documentada o si el documento tiene una que el código no sirve.
- `TestOpenAPISeguridad` exige que toda ruta protegida declare `bearerAuth` en el OpenAPI y que las públicas no lo declaren.
- El trabajo «OpenAPI 3 válido» del CI valida el archivo contra la especificación en cada Pull Request.

<!-- captura: Swagger UI de la API con el botón de autorización Bearer {#fig:swagger} [capturas/e2e-11-swagger.png] -->

## 12.4 Datos de prueba de los endpoints

Los endpoints se prueban con cuatro conjuntos de datos, cada uno con un propósito distinto:

1. **Colección de Postman** (`docs/api/cc65-tf.postman_collection.json`, @tab:postman): un caso por endpoint y por respuesta relevante, con aserciones sobre el código, el tiempo y el contenido. Se ejecuta completa con el *Collection Runner* (sección 12.5).
2. **Consultas de `cmd/benchapi`** (sección 13): <!--v:miles (index .APIJWT.Corridas 0).Peticiones-->2 000<!--/v--> consultas distintas para la caché fría y un conjunto pequeño, repetido, para la caliente. Se generan recorriendo en base mixta departamento × año × semana × edad × sexo, de modo que no hay dos iguales y todas son válidas para el modelo publicado (los departamentos se leen de `GET /api/v1/modelo`).
3. **Casos de `cmd/seguridad`** (sección 15.5): peticiones diseñadas para fallar (sin token, token alterado o vencido, rol ajeno, origen no permitido, cuerpo grande, demasiados intentos) y comprobar que fallan con el código correcto.
4. **Tests de Go** (`internal/api/*_test.go`): cada ruta × {sin token, analista, admin} (`TestMatrizRoles`), consultas inválidas campo por campo (`TestPrediccionEntradaInvalidaDa400EnEspanol`) y la predicción de la API comparada con el cálculo directo del modelo (`TestPrediccionCoincideConElModeloEntrenado`).

<!-- generado:postman -->
Tabla: Casos de prueba de la colección de Postman {#tab:postman}

| # | Caso | Petición | Cuerpo | Aserciones |
|---:|---|---|---|---|
| 1 | Salud | `GET /healthz` | — | HTTP 200; responde en menos de 100 ms; estado ok |
| 2 | Login (JWT) | `POST /api/v1/auth/login` | `{"clave":"{{clave}}","correo":"{{correo}}"}` | HTTP 200; responde en menos de 1 s; devuelve token, rol y vencimiento |
| 3 | Modelo sin token (401) | `GET /api/v1/modelo` | — | HTTP 401 sin Authorization; pide un token Bearer; responde en menos de 100 ms |
| 4 | Modelo publicado | `GET /api/v1/modelo` | — | HTTP 200; responde en menos de 100 ms; tiene versión y variables |
| 5 | Predicción nueva (caché: MISS) | `POST /api/v1/predicciones` | `{"departamento": "PIURA", "ano": 2024, "semana": {{semana_prueba}}, "edad": {{edad_prueba}}, "tipo_edad": "A", "sexo": "M"}` | HTTP 200; responde en menos de 100 ms; riesgo entre 0 y 1; se calcula, no sale de la caché |
| 6 | Predicción repetida (caché: HIT) | `POST /api/v1/predicciones` | `{"departamento": "PIURA", "ano": 2024, "semana": {{semana_prueba}}, "edad": {{edad_prueba}}, "tipo_edad": "A", "sexo": "M"}` | HTTP 200; responde en menos de 100 ms; sale de la caché de Redis |
| 7 | Predicción inválida (sin edad) | `POST /api/v1/predicciones` | `{"ano":2024,"departamento":"LORETO","semana":10,"sexo":"F","tipo_edad":"A"}` | HTTP 400; responde en menos de 100 ms; mensaje de error en español |
| 8 | Estado del clúster | `GET /api/v1/cluster` | — | HTTP 200; responde en menos de 100 ms; nodos es un arreglo |
| 9 | Lanzar entrenamiento distribuido | `POST /api/v1/entrenamientos` | `{"epocas":600}` | HTTP 202; responde en menos de 100 ms; queda en curso con un ID |
| 10 | Consultar el entrenamiento | `GET /api/v1/entrenamientos/{{entrenamiento_id}}` | — | HTTP 200; responde en menos de 100 ms; es el entrenamiento lanzado |
| 11 | Riesgo cruzado (mapa) | `GET /api/v1/riesgo` | — | HTTP 200; responde en menos de 100 ms; cubo y prioridades |
| 12 | Evaluación experimental | `GET /api/v1/evaluacion` | — | HTTP 200; responde en menos de 100 ms; métricas, ROC y escalabilidad |
| 13 | Ticket del WebSocket (admin) | `POST /api/v1/ws/ticket` | — | HTTP 200; responde en menos de 100 ms; ticket de un solo uso que vence a los 30 s |
| 14 | WebSocket sin ticket (401) | `GET /api/v1/ws` | — | HTTP 401 sin ticket; responde en menos de 100 ms |
| 15 | Cortar un nodo (solo con -demo) | `POST /api/v1/demo/nodos/{{nodo}}/cortar` | — | responde en menos de 100 ms |

Nota: Generado por `cmd/informe` desde `docs/api/cc65-tf.postman_collection.json`. Las variables entre llaves las completa la colección: el login guarda el token, el lanzamiento del entrenamiento guarda su identificador y el caso 5 elige al azar la semana y la edad (`{{semana_prueba}}`, `{{edad_prueba}}`), para que en cada corrida la primera predicción sea nueva y la repetida salga de la caché.
<!-- /generado -->

El generador de consultas de `cmd/benchapi`:

<!-- generado:fuente:cmd/benchapi/main.go#(generador).consulta -->
```go
// cmd/benchapi/main.go
func (g generador) consulta(k uint64) []byte {
	sexo := []string{"F", "M"}[k%2]
	k /= 2
	edad := k % edades
	k /= edades
	semana := 1 + k%semanas
	k /= semanas
	ano := g.anoMin + int(k%g.anos())
	k /= g.anos()
	depto := g.deptos[k%uint64(len(g.deptos))]
	b, _ := json.Marshal(map[string]any{
		"departamento": depto, "ano": ano, "semana": semana, "edad": edad, "tipo_edad": "A", "sexo": sexo,
	})
	return b
}
```
<!-- /generado -->

## 12.5 Evidencias

**Peticiones y respuestas reales.** El paso `contratos` del guion de evidencias inicia sesión como analista y como admin contra el sistema desplegado y registra cada petición con su respuesta: el estado, las cabeceras relevantes y el cuerpo (recortado cuando es largo). Los tokens, el ticket y las claves se reemplazan por una descripción. Se ve, en orden: el login; una predicción calculada y la misma repetida, que sale de Redis (`X-Cache`); una consulta sin edad (400); el modelo, el riesgo del mapa y la evaluación; una petición sin token (401); el analista en el clúster (403); el admin en el clúster (200) y el ticket del WebSocket.

<!-- evidencia: evidencias/contratos.txt -->

**Entrenamiento lanzado por la API.** `POST /api/v1/entrenamientos` responde 202 con la ubicación del recurso (`Location`); `GET` sobre esa ubicación devuelve el progreso y, al terminar, la versión del modelo publicado:

<!-- evidencia: evidencias/entrenamiento.txt -->

**Postman.** La @fig:postman-prediccion muestra una predicción con la autorización Bearer, el cuerpo, la respuesta 200 y su tiempo; la ejecución completa de la colección se muestra en la sección 13.3.

<!-- captura: Predicción en Postman: autorización Bearer, cuerpo, respuesta 200 y tiempo de respuesta {#fig:postman-prediccion} [capturas/m-postman-prediccion.png] -->
