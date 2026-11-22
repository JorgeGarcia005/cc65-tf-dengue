# Veredicto — PR-54 · Ronda 3

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/53-api-bd-en-go` @ `a6400f8`
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

El único hallazgo pendiente de la ronda 2 (**H4**, menor) fue aceptado y resuelto con precisión en `e5ba397`, con respuesta registrada en `docs/agentes/revisiones/PR-54/respuesta-2.md` (`a6400f8`). La redacción de `docs/informe/pc4/11-bd.md:115` ahora distingue explícitamente la creación con `InsertOne` (`CrearEntrenamiento`, que rechaza un ID repetido) de las actualizaciones de avance y cierre con `ReplaceOne` y *upsert* (`GuardarEntrenamiento`), quedando totalmente alineada con `11.5` (`11-bd.md:236-237`) y con el código real en `internal/almacen/mongo/mongo.go` e `internal/api/entrenamientos.go`. Todas las verificaciones locales (`gofmt`, `go vet`, `go test -race -count=1 ./...` con 0 carreras, idempotencia de `cmd/informe` y conversión completa de las 16 secciones de PC4 a Word OOXML con 0 avisos) y los 6 jobs del CI de GitHub (run `44967003418` sobre `a6400f8`) están en verde, sin hallazgos abiertos. El PR queda listo para merge con aprobación humana.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/develop...origin/feature/53-api-bd-en-go
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.438s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.364s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.509s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.286s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.220s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.416s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.180s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.029s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.045s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.982s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.096s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.632s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.072s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.078s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.441s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.827s

$ go run ./cmd/informe && git diff --exit-code docs/informe
INFORME_IDEMPOTENT_OK (exit 0, sin cambios en docs/informe)

$ go test -race -count=1 -v ./cmd/informe ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.396s (6/6 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.277s (13/13 PASS)

$ go test -race -count=1 -overlay <audit_overlay.json> -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 16 capturas, 0 avisos, 92556 bytes

$ gh pr checks 54   # run 44967003418 sobre a6400f82acf47ed1368f6c42022c1bc151fd4207
Docker (imagen, clúster y caída de un nodo)	pass	1m59s
Ecosistema de agentes                       	pass	5s
Go (gofmt, vet, test -race)                 	pass	53s
MongoDB y Redis (tests de integración)      	pass	44s
OpenAPI 3 válido                            	pass	19s
Spin (modelo del protocolo y mutantes)      	pass	2m3s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes en verde localmente y en CI run `44967003418`) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; los tests de equivalencia pasan bajo `-race` en la suite y en el job Docker del CI) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`; `10.4` sigue regenerando su tabla desde `reports/bench_api_deez.json`) |
| Idempotencia `cmd/informe` y Word OOXML | ✅ (`git diff --exit-code docs/informe` exit 0; `TestAuditPC4WordXML` genera las 16 secciones con 16 capturas, 0 avisos, `CrearEntrenamiento`/`GuardarEntrenamiento` presentes y `1ACC0065_PC4_U&lt;código&gt;.zip` escapado en XML) |
| Restricciones `AGENTS.md` §3 | ✅ `go.mod` intacto (solo stdlib en `cmd/informe`), `data/` contiene solo `MANIFEST.sha256` y `README.md`, cero cifras o fragmentos copiados a mano |

## Verificación contra los criterios del issue #53

| Criterio (#53) | Estado | Evidencia |
|---|---|---|
| 1. Sección 10.3 «Funcionamiento en Go» con fragmentos extraídos del fuente: enrutador, handler de predicción, cálculo del riesgo y encolado del registro | ✅ | `docs/informe/pc4/10-api.md:35-178` extrae `(*Servidor).Handler`, `(*Servidor).predecir`, `(*Modelo).prediccion` y `(*Servidor).registrar` mediante `<!-- generado:fuente:... -->`, fieles a `internal/api/api.go` e `internal/api/modelo.go`. |
| 2. Sección 11.4 «Entidades en Go» con los `struct` con etiquetas `bson` de las colecciones, creación de índices, clave de Redis (todo extraído del fuente) y párrafo de registros que remite a 12.4 | ✅ | `docs/informe/pc4/11-bd.md:36-228` documenta las tres colecciones en el orden de `11.2` (`Modelo`, `ResumenEntrenamiento`, `(*Almacen).GuardarModelo`, `Entrenamiento`, `ConsultaNormalizada`, `RegistroPrediccion`), `(*Almacen).crearIndices`, `ClavePrediccion`, la precisión `InsertOne`/`ReplaceOne` (`:115`) y el párrafo `**Registros.**` (`:224-228`) que remite a `12.4`. |
| 3. `cmd/informe` extrae también tipos (`fuente:RUTA#Tipo`), con test | ✅ | `cmd/informe/regiones.go:103-168` extrae declaraciones de tipo individuales y dentro de bloques `type ( … )`; `cmd/informe/regiones_test.go:15-45, 57-60, 72-83` valida mínimos por sección (`08-distribuido.md` ≥ 9, `10-api.md` ≥ 4, `11-bd.md` ≥ 8) y ambos formatos de tipo. |
| 4. La tabla 4.3 y las referencias cruzadas apuntan a las secciones nuevas | ✅ | `docs/informe/pc4/04-arquitectura.md:73-74` apunta a `10.3` y `11.4`; `docs/informe/pc4/12-evidencias.md:55` apunta a `10.4`; `rg` sobre `docs/informe/pc4/` confirma cero referencias obsoletas. |
| 5. `go run ./cmd/informe` idempotente; `go test ./cmd/informe ./cmd/word` en verde | ✅ | `INFORME_IDEMPOTENT_OK`, `cmd/informe` (6/6 PASS `-race`), `cmd/word` (13/13 PASS `-race`) y `TestAuditPC4WordXML` (16 secciones, 16 capturas, 0 avisos). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | La tabla `4.3` (`04-arquitectura.md:73-74`) enlaza API en Go y entidades/registros con `10.3`, `11.4`, `12.3` y `12.4`. |
| Implementación | 8 | sin cambios en backend | Solo extiende `cmd/informe` con paquetes de la biblioteca estándar (`go/ast`, `go/token`). |
| Interfaz | 5 | no aplica | Entregable TB2. |
| Informe | 2 | 2 / 2 (en el alcance del issue #53) | Las secciones `10.3` y `11.4` muestran código Go real extraído del repositorio para API, colecciones MongoDB, índices y clave Redis, sin inconsistencias internas y listo para generar el `.docx` con `cmd/word`. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos. Los cuatro hallazgos acumulados de las rondas 1 y 2 (**H1–H4**) quedaron resueltos y verificados contra el código fuente, los tests con `-race` y el CI completo en verde.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H4 (ronda 2 · MENOR) — Matiz de redacción entre `11.4` y `11.5` sobre creación frente a actualización en `entrenamientos` | **RESUELTO** | Commit `e5ba397` actualiza `docs/informe/pc4/11-bd.md:115` a: *«Un entrenamiento (colección `entrenamientos`) se crea con `InsertOne` (`CrearEntrenamiento`, que rechaza un ID repetido) y cada avance o cierre lo reescribe entero con `ReplaceOne` y *upsert* (`GuardarEntrenamiento`):»*. Contrastado con `internal/almacen/mongo/mongo.go:84-90, 128-131, 250-260` (`InsertOne` → `almacen.ErrDuplicado`; `ReplaceOne` con `SetUpsert(true)`) e `internal/api/entrenamientos.go:104-109, 138, 161, 200, 234`, y totalmente coherente con `11.5` (`11-bd.md:236-237`). |
| H1–H3 (ronda 1) — `Modelo` + `ResumenEntrenamiento` + `GuardarModelo` en `11.4`, mínimos por sección y `type ( … )` en `cmd/informe`, y descripción/texto de `f6eaa87` | **RESUELTO** | Resueltos en ronda 2 (`6dd14b2`, `fbe4b3a`) y confirmados sin regresiones en `a6400f8` (`TestFragmentosSalenDelFuente` 6/6, `cmd/word` 13/13, `TestAuditPC4WordXML` 0 avisos, CI 6/6 en verde). |

## Cómo explicarlo en la sustentación

Las secciones `10.3` y `11.4` incorporan directamente desde el repositorio —sin copiar ni pegar a mano— el código Go que sustenta la API REST y las bases de datos de la PC4: el enrutador único `Handler`, el camino crítico `predecir` (lectura sin bloqueo del modelo vía `atomic.Pointer`, *cache-aside* en Redis con plazo fijo de 15 ms por operación y encolado no bloqueante), el cálculo calibrado `(*Modelo).prediccion` y el productor `registrar`, junto con los `struct` con etiquetas `bson` de las tres colecciones (`Modelo` + `ResumenEntrenamiento`, `Entrenamiento`, `ConsultaNormalizada` + `RegistroPrediccion`), el `UpdateOne` atómico con *upsert* de `GuardarModelo`, la separación entre `CrearEntrenamiento` (`InsertOne` sin *upsert*) y `GuardarEntrenamiento` (`ReplaceOne`), `crearIndices` (con TTL de 90 días) y `ClavePrediccion`. `cmd/informe` se amplió para extraer también declaraciones de tipo (`fuente:RUTA#Tipo`, incluso dentro de `type ( … )`) y `TestFragmentosSalenDelFuente` verifica por sección que el informe permanezca sincronizado con el código fuente.
