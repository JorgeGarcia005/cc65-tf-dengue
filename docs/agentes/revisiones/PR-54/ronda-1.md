# Veredicto — PR-54 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/53-api-bd-en-go` @ `f6eaa87`
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR atiende la exigencia de la rúbrica de la PC4 incorporando el funcionamiento en Go de la API (sección `10.3`) y el diseño de entidades en Go de MongoDB y Redis (sección `11.4`), extraídos automáticamente del repositorio mediante `cmd/informe` (que se amplía para extraer también declaraciones de tipo `fuente:RUTA#Tipo`), además de actualizar las referencias cruzadas en `4.3` y `12.3`, añadir el marco de captura del historial de commits en `12.6` y precisar el nombre del `.zip` por integrante en `16.2`. Todas las verificaciones automatizadas pasan en verde (`gofmt`, `go vet`, `go test -count=1 ./...` local, `go run ./cmd/informe` idempotente, conversión completa de PC4 a Word OOXML con 16 secciones, 12 SVG, 16 capturas y 0 avisos, y los 6 jobs del CI en GitHub con `-race` en el run `44965093304`). Sin embargo, se emite **1 hallazgo mayor** y **2 menores** antes de aprobar: la sección `11.4` afirma que cada colección se modela con un `struct` con etiquetas `bson` y muestra `crearIndices` sobre `a.modelos`, pero **omite el `struct` de la colección `modelos`** (`almacen.Modelo` y su subdocumento `ResumenEntrenamiento`) e imprecisa cómo se persiste (**H1**); el umbral del test `TestFragmentosSalenDelFuente` queda en `8` sin proteger las secciones nuevas (**H2**); y el segundo commit (`f6eaa87`) no figura en la descripción del PR ni en el texto introductorio de `12.6` (**H3**).

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/develop...origin/feature/53-api-bd-en-go
(vacío = ok)

$ go run ./cmd/informe && git diff --exit-code docs/informe
INFORME_DIFF_EXIT=0 (sin cambios en las regiones generadas)

$ go test -race -count=1 -v ./cmd/informe ./cmd/word
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1

$ $env:CGO_ENABLED="1"; go test -race -count=1 -v ./cmd/informe ./cmd/word
# runtime/cgo
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe [build failed]
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word [build failed]

$ go test -count=1 -v ./cmd/informe ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	0.433s (5/5 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	0.427s (13/13 PASS)

$ go test -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	1.895s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	3.661s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.251s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	0.633s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	0.673s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	0.676s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.115s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	0.626s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	0.712s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	0.652s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	3.267s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.406s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	3.841s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	1.387s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	0.477s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	0.567s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	0.515s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	1.165s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	0.483s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	2.719s

$ go test -count=1 -overlay <audit_overlay.json> -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 12 SVGs, 12 PNGs, 16 capturas, 0 avisos

$ gh pr checks 54   # run 44965093304 sobre f6eaa87
Docker (imagen, clúster y caída de un nodo)	pass	1m55s
Ecosistema de agentes                       	pass	11s
Go (gofmt, vet, test -race)                 	pass	53s   (todos los paquetes ok, 0 carreras)
MongoDB y Redis (tests de integración)      	pass	37s
OpenAPI 3 válido                            	pass	22s
Spin (modelo del protocolo y mutantes)      	pass	2m5s
```

> Limitación del entorno local (Windows): `go env CGO_ENABLED` está en `0` y no hay `gcc` en `%PATH%`, por lo que `go test -race` falla al invocar el compilador de C (no por un fallo de test ni por una carrera). Se corrió `go test -count=1 ./...` localmente (25/25 paquetes en verde) y se verificó el log del job `Go (gofmt, vet, test -race)` del CI en GitHub (run `44965093304`, job `137032265280`), donde todos los paquetes pasaron con `-race` y 0 carreras.

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; suite local sin `-race` por falta de `gcc` en Windows + CI `Go (gofmt, vet, test -race)` en verde en run `44965093304`) |
| Equivalencia secuencial = distribuido | no aplica (el PR no modifica `internal/ml` ni `internal/cluster`; los tests de equivalencia pasan dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (el PR no modifica `internal/api`, solo extrae su código al informe) |
| Idempotencia `cmd/informe` y Word OOXML | ✅ (`INFORME_DIFF_EXIT=0`; 16 secciones, 12 SVGs, 16 capturas, 0 avisos, `1ACC0065_PC4_U&lt;código&gt;.zip` escapado correctamente en XML) |

## Verificación contra los criterios del issue #53

| Criterio (#53) | Estado | Evidencia |
|---|---|---|
| 1. Sección 10.3 «Funcionamiento en Go» con fragmentos extraídos del fuente: enrutador, handler de predicción, cálculo del riesgo y encolado del registro | ✅ | `docs/informe/pc4/10-api.md:35-179` extrae `(*Servidor).Handler`, `(*Servidor).predecir`, `(*Modelo).prediccion` y `(*Servidor).registrar` mediante `<!-- generado:fuente:... -->`; la prosa coincide fielmente con `internal/api/api.go`, `internal/api/modelo.go` y `internal/api/api_test.go:362-383`. |
| 2. Sección 11.4 «Entidades en Go» con los `struct` con etiquetas `bson` de las colecciones, creación de índices, clave de Redis (todo extraído del fuente) y párrafo de registros que remite a 12.4 | ⚠️ incompleto (**H1**) | `docs/informe/pc4/11-bd.md:36-153` incluye `Entrenamiento`, `ConsultaNormalizada`, `RegistroPrediccion`, `(*Almacen).crearIndices`, `ClavePrediccion` y el párrafo de registros (`:148-152`), pero **omite `almacen.Modelo` (y `ResumenEntrenamiento`)**, es decir, falta la colección `modelos` de las tres colecciones descritas en 11.2. |
| 3. `cmd/informe` extrae también tipos (`fuente:RUTA#Tipo`), con test | ✅ (ajustes menores en **H2**) | `cmd/informe/regiones.go:137-152` extrae `*ast.GenDecl` de `token.TYPE` y `cmd/informe/regiones_test.go:54-57` lo prueba con `RegistroPrediccion`; quedan mejoras menores en el piso de `TestFragmentosSalenDelFuente` y el comentario/soporte de bloques `type (...)` (**H2**). |
| 4. La tabla 4.3 y las referencias cruzadas apuntan a las secciones nuevas | ✅ | `docs/informe/pc4/04-arquitectura.md:73-74` apunta a `10.3` y `11.4`; `docs/informe/pc4/12-evidencias.md:55` actualiza `10.3` → `10.4`; búsqueda en todo `docs/informe/pc4/*.md` confirma cero referencias obsoletas a `10.3` o `11.4`. |
| 5. `go run ./cmd/informe` idempotente; `go test ./cmd/informe ./cmd/word` en verde | ✅ | `INFORME_DIFF_EXIT=0`, `cmd/informe` (5/5 PASS), `cmd/word` (13/13 PASS) y `TestAuditPC4WordXML` (16 secciones, 12 SVGs, 16 capturas, 0 avisos). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | La tabla `4.3` (`04-arquitectura.md:73-74`) mapea con precisión los puntos de API en Go y entidades/registros hacia `10.3`, `11.4`, `12.3` y `12.4`. |
| Implementación | 8 | sin cambios en backend | No toca lógica concurrente ni de red; solo extiende `cmd/informe` (herramienta de documentación) con stdlib (`go/ast`, `go/token`). |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | Parcial (pendiente **H1**) | La sección `10.3` queda sólida y fiel al código real; la sección `11.4` deja sin mostrar el `struct` de la colección `modelos` pese a describir tres colecciones en `11.2` y crear sus índices en `crearIndices` (**H1**). |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

### H1 — La sección 11.4 omite el `struct` de la colección `modelos` (`almacen.Modelo` y `ResumenEntrenamiento`) e imprecisa su escritura en MongoDB · MAYOR

- **Dónde:** `docs/informe/pc4/11-bd.md:38-41, 65-66, 99-113` (contrastado con `docs/informe/pc4/11-bd.md:14-24`, `internal/almacen/almacen.go:19-47` e `internal/almacen/mongo/mongo.go:92-126`)
- **Evidencia:**
  1. El criterio 2 del issue #53 exige mostrar *«los struct con etiquetas bson de las colecciones»*, y la tabla de la sección `11.2` (`11-bd.md:16-20`) define **tres** colecciones en `cc65_dengue`: **`modelos`**, **`entrenamientos`** y **`predicciones`**.
  2. La sección `11.4` abre declarando (`11-bd.md:38-39`): *«Cada colección es un `struct` de Go con etiquetas `bson`; el driver lo serializa tal cual, sin mapeos intermedios.»*, luego señala (`11-bd.md:65-66`) que la versión del modelo *«es lo que une las tres colecciones»*, y más abajo (`11-bd.md:110-113`) muestra `(*Almacen).crearIndices` creando los índices `{publicado: -1}` y `{"entrenamientos.id": 1}` sobre `a.modelos`.
  3. Sin embargo, `11.4` solo extrae `Entrenamiento` (`entrenamientos`), `ConsultaNormalizada` y `RegistroPrediccion` (`predicciones`), **omitiendo por completo `almacen.Modelo`** (`internal/almacen/almacen.go:19-33`, colección `modelos`, donde `Version` es `_id`, `Publicado` sustenta el índice `{publicado: -1}` y `Entrenamientos []ResumenEntrenamiento` enlaza los entrenamientos) y su subdocumento **`almacen.ResumenEntrenamiento`** (`internal/almacen/almacen.go:35-47`, donde vive `bson:"id"` del índice `entrenamientos.id`). En revisión o sustentación, el hueco entre la tabla `11.2` (que encabeza con `modelos`), los índices de `a.modelos` en `crearIndices` y la ausencia de `Modelo` en `11.4` es inmediato.
  4. Adicionalmente, la frase *«el driver lo serializa tal cual, sin mapeos intermedios»* (`11-bd.md:38-39`) aplica tal cual a `entrenamientos`, `predicciones` y la lectura (`Decode`) de `modelos`, pero al **escribir** un modelo `(*Almacen).GuardarModelo` (`internal/almacen/mongo/mongo.go:95-113`) arma un `UpdateOne` atómico con `$setOnInsert`, `$max` sobre `publicado` y `$push` sobre `entrenamientos` para que dos entrenamientos con los mismos pesos no se pisen entre sí (justamente lo que promete el primer punto de `11.5`). Precisarlo evita una contradicción si preguntan en la sustentación cómo se publica sin sobrescribir resúmenes.
- **Corrección sugerida:**
  - Incorporar en `docs/informe/pc4/11-bd.md` (antes de `Entrenamiento`, siguiendo el mismo orden de la tabla `11.2`: `modelos`, `entrenamientos`, `predicciones`) las regiones generadas `<!-- generado:fuente:internal/almacen/almacen.go#Modelo -->` y `<!-- generado:fuente:internal/almacen/almacen.go#ResumenEntrenamiento -->`, y regenerar con `go run ./cmd/informe`.
  - Matizar `11-bd.md:38-39` indicando que cada colección tiene un `struct` de Go con etiquetas `bson` que el driver codifica y decodifica directamente (y que, en `modelos`, la publicación usa un `UpdateOne` atómico con `$setOnInsert`, `$max` y `$push` sobre esos mismos campos BSON para acumular resúmenes sin pisarlos).

### H2 — `TestFragmentosSalenDelFuente` mantiene el umbral de 8 fragmentos y `extraerFuncion` deja el comentario/caso agrupado sin actualizar · MENOR

- **Dónde:** `cmd/informe/regiones_test.go:37-41` y `cmd/informe/regiones.go:103-105, 140`
- **Evidencia:**
  1. `cmd/informe/regiones_test.go:39` comprueba `if fragmentos < 8`, pensado cuando solo la sección `08-distribuido.md` tenía fragmentos (9 bloques). Hoy el informe de PC4 contiene **18** bloques `<!-- generado:fuente:... -->` (9 en `08-distribuido.md`, 4 en `10-api.md` y 5 en `11-bd.md`; serán 20 al resolver **H1**). Si alguien borrara por accidente los bloques de `10.3` o `11.4`, el test seguiría pasando porque `9 >= 8`.
  2. El comentario de `extraerFuncion` en `cmd/informe/regiones.go:103-104` todavía dice solo *«una función (con su comentario)»* (mientras el comentario del paquete en `:13-15` ya documenta tipos), y la condición `len(gd.Specs) != 1` en `regiones.go:140` omitiría declaraciones agrupadas `type ( ... )` (actualmente no existe ninguna en el repo, por lo que no falla hoy).
- **Corrección sugerida:**
  - Actualizar `TestFragmentosSalenDelFuente` para reflejar el total esperado tras **H1** (p. ej. exigir al menos 18–20 fragmentos en total, o comprobar un mínimo por sección en `08-distribuido.md`, `10-api.md` y `11-bd.md`).
  - Actualizar el comentario de `extraerFuncion` en `regiones.go:103-104` para mencionar también tipos (y, opcionalmente, iterar sobre `gd.Specs` en vez de filtrar `len(gd.Specs) != 1`).

### H3 — El segundo commit (`f6eaa87`) no figura en la descripción del PR ni en el texto introductorio de `12.6` · MENOR

- **Dónde:** descripción del PR #54, `docs/informe/pc4/12-evidencias.md:76-81` y `docs/informe/pc4/16-anexos.md:31`
- **Evidencia:**
  El commit `f6eaa87` (`docs(informe): marco para el historial de commits y nombre del zip por integrante (#53)`) añade `<!-- captura: Historial de commits de la rama develop en GitHub -->` en `12-evidencias.md:81` y ajusta el formato del zip a `` `1ACC0065_PC4_U<código>.zip` (uno por integrante) `` en `16-anexos.md:31`. Ambos cambios son válidos y se convierten sin avisos en `cmd/word`, pero la sección `## Cambios` del PR #54 no los menciona y el texto de `12-evidencias.md:76-77` sigue anunciando solo *«Historial de pull requests de la PC4 […] y las ramas con Git Flow»* sin mencionar el historial de commits de `develop` (donde se verifican los commits de los tres integrantes).
- **Corrección sugerida:**
  Añadir una viñeta en la descripción del PR #54 para los ajustes de `12.6` y `16.2`, y mencionar también el historial de commits de los integrantes en `docs/informe/pc4/12-evidencias.md:76-77`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-54. |

## Cómo explicarlo en la sustentación

La sección `10.3` muestra en Go puro (`net/http`) el camino crítico de la API extraído directamente del código fuente: el enrutador único `Rutas()` verificado contra OpenAPI y protegido con `recuperar`, el handler `predecir` (lectura sin bloqueo del modelo en `atomic.Pointer`, *cache-aside* en Redis con presupuesto fijo de 15 ms por operación y encolado no bloqueante), el cálculo del riesgo calibrado en `(*Modelo).prediccion`, y el productor no bloqueante `registrar` con canal con búfer hacia el consumidor `registrador`. La sección `11.4` hace lo mismo para persistencia y caché (`struct` con etiquetas `bson`, creación de índices incluido el TTL de 90 días, y construcción de la clave versionada de Redis) remitiendo los registros reales a `12.4`, mientras `cmd/informe` extiende `fuente:RUTA#Tipo` para que ningún fragmento se copie a mano. Al incorporar el `struct` faltante de la colección `modelos` (`almacen.Modelo` y `ResumenEntrenamiento`, **H1**), las tres colecciones de MongoDB quedarán documentadas de principio a fin y listas para generar el `.docx` final con `cmd/word`.
