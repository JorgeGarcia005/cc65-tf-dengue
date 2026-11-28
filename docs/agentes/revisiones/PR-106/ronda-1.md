# Veredicto — PR-106 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/105-informe-tb2-bloque4` @ `4da9ef8`
- **Fecha:** 2026-11-28
- **Decisión:** APROBADO

El PR cierra el bloque 4 del informe final TB2 (`#105`, parte de `#71`), completando sin marcadores `<!-- pendiente:` las secciones 12 (Parte C: API REST), 13 (latencia de la API), 14 (bases de datos), 15 (seguridad e integración SPA–API) y 16 (Parte D: interfaz web), además de incorporar el paso 15 (`contratos`) en la guía y control de `#70`:

1. **Observación 3 de la PC4 (código de la API, contratos *request/response* y set de pruebas):** la sección 12 incrusta desde el árbol fuente `api.Rutas()`, `(*Servidor).Handler()`, `(*Servidor).predecir()`, `(*Modelo).prediccion()` y `leerJSON`; genera automáticamente `@tab:contratos` cruzando `api.Rutas()` con `docs/api/openapi.yaml` (`cmd/informe/contratos.go`, verificado por `TestContratosCubrenLasRutas`) junto con los `struct` de Go (`Consulta`, `Prediccion`, `RespuestaLogin`, `ParametrosEntrenamiento`); detalla los cuatro conjuntos de datos de prueba (`@tab:postman` generada desde `cc65-tf.postman_collection.json`, generador en base mixta de `cmd/benchapi`, casos adversarios de `cmd/seguridad` y matriz de tests en Go) y añade el paso `contratos` en `scripts/evidencias-tb2.sh` con peticiones y respuestas reales sin exponer tokens, tickets ni claves.
2. **Observación 4 de la PC4 (pruebas que corroboran el cuadro de latencia):** la sección 13 genera `@tab:latencia` mediante `docs/informe/tb2/plantillas/latencia.tmpl` desde `reports/bench_api_*_jwt.json` (p95 de `3,89 ms` en caché fría y `3,09 ms` en caché caliente con 16 clientes concurrentes y JWT activo) y la respalda con tres evidencias independientes: `evidencias/benchapi.txt`, el *Collection Runner* de Postman (`@fig:postman-runner`) y el p95 medido dentro del servidor en el panel de administración durante el entrenamiento (`@fig:admin-entrenando`).
3. **Observación 6 de la PC4 (diagrama de entidades y qué evidencia `bd.txt`):** la sección 14 incorpora el diagrama de clases UML 2.5 de colecciones de MongoDB, documentos embebidos y caché Redis (`docs/informe/tb2/diagramas/entidades.dot` → `fig/entidades.svg`), incrusta los tipos BSON, `GuardarModelo` atómico (`$setOnInsert` + `$max` + `$push`), `crearIndices`, `ClavePrediccion` y el productor-consumidor no bloqueante (`registrar` / `registrador`), y desglosa punto por punto los seis bloques verificados por `scripts/evidencia-bd.sh`.
4. **Seguridad e interfaz (Partes C y D):** la sección 15 documenta `(*Firmador).Verificar`, `Comparar` (PBKDF2-HMAC-SHA256 con 600 000 iteraciones en tiempo constante), `Requiere`, `cors`, la CSP estricta de `cmd/web`, el *handshake* WebSocket con ticket de un solo uso + `Origin` (`conTicket`), `(*limitador).Reservar` y `@tab:seguridad` con el candado `<!-- falta: reports/seguridad.json -->` probado por `TestFaltaReporte` en `cmd/word` (verificado tanto con reporte presente como ausente). La sección 16 documenta las vistas por rol (`web/src/rutas.tsx#vistas`), el reductor puro del tiempo real, reconexión y accesibilidad, enlazando las capturas de Playwright y renumerando limpiamente los pasos 15 a 19 en `docs/guias/jose-70.md` y `scripts/paso70.sh`.

Todas las verificaciones automáticas (`gofmt`, `go vet`, `go test -race -count=1 ./...` en las 26 suites con 0 carreras, `tsc -b` + Vitest con 186/186 tests en `web/` y los 7 trabajos del CI en GitHub Actions) están en verde. Se dejan dos observaciones **MENORES** (**H1** y **H2**, no bloqueantes) que pueden incorporarse opcionalmente en el bloque 5 (`#108`).

## Verificación automática

```text
$ git rev-parse --short HEAD
4da9ef8

$ git diff --check origin/integracion/tb2...origin/feature/105-informe-tb2-bloque4
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash -n scripts/evidencias-tb2.sh && bash -n scripts/paso70.sh && bash scripts/check-skills-sync.sh
skills: ok

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	12.382s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	10.142s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	7.067s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.897s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.290s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	2.456s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.438s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/seguridad	73.161s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.416s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.070s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	7.887s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.032s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.055s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	31.124s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	23.420s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	15.453s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	6.233s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.089s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.168s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.056s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.083s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	9.467s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.053s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	16.336s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.837s

$ (cd web && npm ci --no-audit --no-fund >/dev/null && npx tsc -b && npx vitest run)
Test Files  12 passed (12)
     Tests  186 passed (186)

$ gh pr checks 106 -R JorgeGarcia005/cc65-tf-dengue   # run 46163088910 sobre 4da9ef8
Docker (imagen, clúster y caída de un nodo)	pass	2m56s
Ecosistema de agentes                       	pass	9s
Go (gofmt, vet, test -race)                 	pass	1m39s
MongoDB y Redis (tests de integración)      	pass	59s
OpenAPI 3 válido                            	pass	21s
SPA (tipos, tests, build e imagen)          	pass	1m12s
Spin (modelo del protocolo y mutantes)      	pass	1m50s
```

> *Nota sobre `entidades.svg`:* se verificó con `dot -Tsvg:cairo docs/informe/tb2/diagramas/entidades.dot` (`Graphviz 2.42.2`, `Liberation Serif`) que `docs/informe/tb2/fig/entidades.svg` conserva exactamente el mismo lienzo (`width="862" height="593" viewBox="0 0 862 593"`) y pasa `TestInformesSinAvisos` en `cmd/word`. Asimismo, se probó `go run ./cmd/informe` y `go test ./cmd/informe ./cmd/word` tanto sin `reports/seguridad.json` (rama `<!-- falta: reports/seguridad.json -->`) como con un `reports/seguridad.json` sintético de `cmd/seguridad` (rama `{{- with index .R "seguridad" -}}`).

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`26/26` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite y en el job Docker del CI) |
| Latencia API p95 < 100 ms | ✅ (`@tab:latencia` generada desde `reports/bench_api_jorge-pc_jwt.json`: p95 = `3,89 ms` fría y `3,09 ms` caliente con JWT y 16 clientes; no modifica `internal/api`) |
| Sincronía de contratos, informe, Word y SPA (`cmd/informe`, `cmd/word`, `web`) | ✅ (`TestContratosCubrenLasRutas`, `TestInformeTB2AlDia`, `TestFaltaReporte`, `TestInformesSinAvisos` y `186/186` tests de Vitest en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | Cubierto en los bloques 2 y 3 (secciones 1 a 6). |
| Implementación | 8 | documenta Parte C completa | Explica y respalda con código real la cadena de *middlewares*, contratos REST/OpenAPI, medición de latencia < 100 ms con JWT, diseño e índices de MongoDB/Redis, escritura asíncrona no bloqueante y seguridad (JWT HS256, PBKDF2, CORS, CSP, ticket WS y límites). |
| Interfaz | 5 | documenta Parte D completa | Cubre usuarios y decisiones UX, matriz de rutas por rol (`web/src/rutas.tsx`), vistas de impacto social, configuración de entrenamiento desde la UI, eventos en tiempo real por WebSocket + modo réplica y accesibilidad/diseño adaptable, enlazando las 12 capturas de Playwright y las 2 de Postman. |
| Informe | 2 | avance muy sólido (secciones 1–16 y 22 listas) | Cierra las observaciones 3, 4 y 6 de la PC4 sin cifras a mano, añade `TestContratosCubrenLasRutas` y el candado `<!-- falta: … -->` para `cmd/word -final`, y actualiza en orden la guía y el control de `#70`. |
| Video | 2 | sin cambios | No afecta el video. |

## Hallazgos

### H1 — Normalización de barra en `.APIJWTArchivo` en Windows nativo e idempotencia de `X-Cache` al reejecutar `contratos` · MENOR

- **Dónde:** `cmd/informe/main.go:561-565` (`docs/informe/tb2/plantillas/latencia.tmpl:9`) y `scripts/evidencias-tb2.sh:264-288`
- **Evidencia:**
  1. En `cmd/informe/main.go:561-565`, `filepath.Glob("reports/bench_api_*_jwt.json")` asigna `d.APIJWTArchivo = a` sin `filepath.ToSlash(a)` (a diferencia de `tablaCluster` en `cmd/informe/regiones.go:476` y `tablaAPI` en `cmd/informe/regiones.go:539`). En Linux, WSL (donde se ejecuta toda la guía `docs/guias/jose-70.md`) y el CI pasa en verde, pero si se corre `go test ./cmd/informe` en Windows nativo (`GOOS=windows`, PowerShell), `filepath.Glob` devuelve `reports\bench_api_jorge-pc_jwt.json` con barra invertida y `TestInformeTB2AlDia` falla contra `docs/informe/tb2/13-latencia.md:27`.
  2. En `scripts/evidencias-tb2.sh:283-288`, `p_contratos` envía una consulta fija (`{"departamento":"LORETO","ano":2025,"semana":9,"edad":7,"tipo_edad":"A","sexo":"F"}`) y `muestra` (`264-273`) valida el código HTTP (`200`), pero no el encabezado `X-Cache`. En la primera ejecución sobre el servicio levantado produce `MISS` seguido de `HIT`, pero si se reejecuta `bash scripts/evidencias-tb2.sh contratos` sin reiniciar Redis (`TTLPrediccion = 24h` en `internal/almacen/almacen.go:167`), ambas respuestas devuelven `X-Cache: HIT` con estado `OK`, contradiciendo el encabezado del paso y `12-api.md:437`.
- **Corrección sugerida:** Usar `d.APIJWTArchivo = filepath.ToSlash(a)` en `cmd/informe/main.go:565` y, en `p_contratos` (`scripts/evidencias-tb2.sh:283-288`), garantizar que la primera predicción sea nueva (p. ej. borrando antes su clave en Redis o eligiendo una edad en días `tipo_edad: "D"` no cacheada) y verificar `X-Cache: MISS` en la primera llamada y `X-Cache: HIT` en la segunda (opcional en `#108` o `#70`).

### H2 — Matices menores en la nota de `@tab:contratos`, descripción de `@fig:admin-caida` en §16.5 y registro duplicado en `sintesis.go` · MENOR

- **Dónde:** `cmd/informe/contratos.go:78,106-111,129` (`docs/informe/tb2/12-api.md:210,211,219,221`), `docs/informe/tb2/16-interfaz.md:80` y `cmd/informe/sintesis.go:234,242`
- **Evidencia:**
  1. En `cmd/informe/contratos.go:78,106-111`, la columna `Respuesta correcta` de `@tab:contratos` imprime `—` cuando la respuesta `2xx` de `docs/api/openapi.yaml` usa un `type: object` en línea en vez de `$ref` (`GET /healthz`, `POST /api/v1/ws/ticket`, `POST /api/v1/demo/nodos/{nombre}/cortar`), igual que en `GET /api/v1/ws` (`101` sin cuerpo JSON); además, el esquema OpenAPI del login se llama `Sesion` (`12-api.md:211`) y su tipo en Go `RespuestaLogin` (`12-api.md:280`).
  2. En `docs/informe/tb2/16-interfaz.md:80`, se describe `@fig:admin-caida` como *«donde se ve la caída de un nodo y la reasignación de su shard»*, mientras que con `ritmo_ms = 20` en `web/e2e/vigia.e2e.ts:152-165` el corte puede ocurrir en la pausa entre épocas y registrar caída + reconexión (como ya contempla `docs/guias/jose-70.md:106` y se observó en PR-104 H1).
  3. En `cmd/informe/sintesis.go:234,242`, `funciones["derefB"]` se asigna dos veces dentro de `init()`.
- **Corrección sugerida:** Aclarar en la `Nota:` de `regionContratos()` (`cmd/informe/contratos.go:129`) que `—` en respuesta indica ausencia de esquema con nombre en `components/schemas` (objetos breves en línea o cambio de protocolo `101`) y la correspondencia `Sesion` ↔ `RespuestaLogin`; alinear `16-interfaz.md:80` con `docs/guias/jose-70.md:106`; y dejar una sola asignación de `derefB` en `sintesis.go` (opcional en `#108`).

## Seguimiento de rondas previas

Primera ronda para este PR.

## Cómo explicarlo en la sustentación

Las secciones 12 a 16 documentan de punta a punta el plano de servicio y visualización (Partes C y D) respondiendo con evidencia verificable a las observaciones 3, 4 y 6 de la PC4: la tabla de contratos se genera directamente desde `api.Rutas()` y `docs/api/openapi.yaml` para que nunca pueda desfasarse del código (`TestContratosCubrenLasRutas`) y se acompaña de los tipos Go, los cuatro conjuntos de datos de prueba y trazas HTTP reales sin secretos (`evidencias/contratos.txt`). La latencia menor a 100 ms se demuestra bajo carga concurrente con JWT activo mediante `cmd/benchapi` (`p95 = 3,89 ms` en caché fría y `3,09 ms` en caliente), Postman y el medidor interno en vivo; la persistencia en MongoDB y la caché en Redis se explican con su diagrama UML de entidades, escritura asíncrona no bloqueante e inspección línea por línea de la cadena entrenamiento → modelo → predicciones → caché; y la integración SPA–API cubre RBAC, CORS, CSP, WebSocket con ticket de un uso y pruebas externas automatizadas con `cmd/seguridad` y Playwright.
