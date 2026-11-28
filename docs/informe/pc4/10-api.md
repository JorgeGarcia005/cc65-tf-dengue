# 10. API REST

## 10.1 Rutas y documentación

El coordinador, en modo servicio, expone la API con `net/http` de la biblioteca estándar. El contrato está
en `docs/api/openapi.yaml` (OpenAPI 3.0.3; OpenAPI Initiative, 2020; validado en el CI) y se sirve con **Swagger UI** en `/docs`. Un
test exige que cada ruta del servidor esté documentada.

| Método y ruta | Qué hace | Respuestas |
|---|---|---|
| `GET /healthz` | Salud del servicio y versión del modelo publicado | 200 |
| `GET /api/v1/modelo` | Modelo publicado: versión, pesos por variable, umbral, departamentos | 200 |
| `POST /api/v1/predicciones` | Riesgo calibrado de que un caso sea con signos de alarma o grave, a partir de departamento, año, semana epidemiológica, edad y sexo | 200, 400 (consulta inválida, con mensaje en español), 413 (cuerpo demasiado grande) |
| `GET /api/v1/cluster` | Foto del clúster: nodos, estado, carga, época y pérdida | 200, 503 (el servicio corre sin clúster) |
| `POST /api/v1/entrenamientos` | Lanza un entrenamiento distribuido en los nodos (cuerpo opcional: épocas, tasa, λ) | 202 con `Location`, 400 (parámetros inválidos), 409 si ya hay uno en curso, 503 si no hay clúster |
| `GET /api/v1/entrenamientos/{id}` | Progreso o resultado: época, pérdida por época, nodos y versión del modelo publicado | 200, 404 |

La consulta se valida con las mismas reglas que la limpieza (sección 6) y se transforma con el mismo
esquema de variables guardado en el modelo, así que la API predice exactamente lo que el modelo aprendió.

## 10.2 Diseño para responder en menos de 100 ms

- **Sin bloqueos en la lectura.** El modelo publicado vive en un `atomic.Pointer`: cada predicción lo lee
  sin mutex, y un entrenamiento nuevo reemplaza el puntero entero cuando termina. Las predicciones siguen
  respondiendo con el modelo anterior mientras el clúster entrena.
- **Caché con presupuesto.** Redis guarda cada predicción con una clave que incluye la versión del modelo.
  Leer y escribir en la caché tienen un plazo fijo de 15 ms cada uno: si Redis está lento o caído, la API
  calcula y responde igual.
- **El registro no espera a la base.** Cada predicción se registra en MongoDB desde una goroutine aparte que
  consume un canal con búfer; si el búfer se llena, el registro se descarta y se cuenta, pero la respuesta
  nunca espera a Mongo.
- **Cierre ordenado.** Con `SIGTERM`, el servidor deja de aceptar conexiones, termina las peticiones en
  curso, cancela el entrenamiento y espera su escritura final, y recién después avisa `fin` a los nodos.

## 10.3 Funcionamiento en Go

La API es un `http.Server` de la biblioteca estándar. `Handler` arma el enrutador con la tabla única
`Rutas()` (la misma que el test compara con el OpenAPI) usando los patrones con método de `net/http`
(`"POST /api/v1/predicciones"`) y envuelve todo con `recuperar`, que convierte un pánico en un 500 sin
tumbar el proceso:

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

`predecir` es el camino que mide el benchmark. Lee el modelo vigente una sola vez desde el
`atomic.Pointer` (clave, cálculo y registro usan el mismo modelo aunque otro entrenamiento lo reemplace en
ese instante), valida y normaliza la consulta, intenta la caché con un plazo fijo (*cache-aside*) y, si no
está, calcula, guarda en Redis y responde. El registro en MongoDB se encola al final y no se espera:

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

El cálculo es el mismo producto punto y la misma sigmoide del entrenamiento, con los pesos publicados y la
calibración del riesgo (sección 7):

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

`registrar` es la mitad productora del registro asíncrono: un `select` con `default` sobre un canal con
búfer. Si el canal está lleno, el registro se cuenta como descartado y el handler sigue; una sola goroutine,
`registrador`, consume el canal y escribe en MongoDB con su propio plazo:

<!-- generado:fuente:internal/api/api.go#(*Servidor).registrar -->
```go
// internal/api/api.go
// registrar encola sin bloquear.
func (s *Servidor) registrar(r almacen.RegistroPrediccion) {
	select {
	case s.registros <- r:
	default:
		s.descartados.Add(1)
	}
}
```
<!-- /generado -->

## 10.4 Latencia medida

`go run ./cmd/benchapi -exigir-p95 100ms` envía, con 16 clientes concurrentes, dos corridas de 2000
peticiones: con la **caché fría** (2000 consultas distintas: se calcula y se guarda en Redis) y con la
**caché caliente** (2000 peticiones sobre 32 consultas precalentadas, que ya están en Redis). Las dos
corridas usan mitades disjuntas del espacio de consultas, así que ninguna consulta fría estaba en la caché. Mide en el cliente, como Postman, hasta leer la respuesta
completa, y falla si el p95 de alguna corrida no baja de 100 ms o si la fría tuvo aciertos de caché o la
caliente no salió entera de Redis.

<!-- generado:api -->
| Corrida | Peticiones | Errores | HIT / MISS | Peticiones por segundo | p50 (ms) | p95 (ms) | p99 (ms) | Máx. (ms) | p95 < 100 ms |
|---|---:|---:|---|---:|---:|---:|---:|---:|---|
| Caché fría (consultas distintas) | 2 000 | 0 | 0 / 2 000 | 10 986 | 1,39 | 2,21 | 2,78 | 5,49 | sí |
| Caché caliente (en Redis) | 2 000 | 0 | 2 000 / 0 | 12 982 | 1,10 | 2,06 | 2,34 | 3,02 | sí |

Fuente: `reports/bench_api_deez.json` (`go run ./cmd/benchapi -exigir-p95 100ms` contra el clúster en Docker Compose con el gold real, 16 clientes concurrentes, Windows; modelo `e69395b36d2f`).
<!-- /generado -->

La colección `docs/api/cc65-tf.postman_collection.json` tiene una petición por ruta con tests de estado y de
tiempo menor a 100 ms; la predicción nueva debe responder `X-Cache: MISS` y la repetida `HIT`. Las capturas
de Postman y de Swagger están en la sección 12.3.
