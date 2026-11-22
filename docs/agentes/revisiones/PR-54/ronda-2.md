# Veredicto — PR-54 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/53-api-bd-en-go` @ `fbe4b3a`
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Los tres hallazgos de la ronda 1 (**H1–H3**) fueron aceptados y resueltos de forma verificable en `6dd14b2`, con respuesta registrada en `docs/agentes/revisiones/PR-54/respuesta-1.md` (`fbe4b3a`). La sección `11.4` ahora sigue el orden exacto de la tabla `11.2` (`modelos` → `entrenamientos` → `predicciones`) incorporando `almacen.Modelo`, `almacen.ResumenEntrenamiento` y `(*Almacen).GuardarModelo` con la explicación del `UpdateOne` atómico (`$setOnInsert`, `$push`, `$max`) (**H1**); `TestFragmentosSalenDelFuente` protege mínimos por sección (`08-distribuido.md` ≥ 9, `10-api.md` ≥ 4, `11-bd.md` ≥ 8) y `extraerFuncion` soporta bloques `type ( … )` con test propio (**H2**); y tanto `12.6` como la descripción del PR #54 reflejan los cambios de `f6eaa87` (**H3**). Todas las verificaciones locales (`gofmt`, `go vet`, `go test -race -count=1 ./...` con 0 carreras, idempotencia de `cmd/informe` y conversión completa de PC4 a Word OOXML con 16 secciones, 12 SVGs, 16 capturas y 0 avisos) y los 6 jobs del CI en GitHub (run `44966461206`) están en verde. Se deja solo **1 observación menor** no bloqueante de redacción en `11-bd.md:115` (**H4**); el PR queda listo para merge con aprobación humana.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/develop...origin/feature/53-api-bd-en-go
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.316s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.162s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.497s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.261s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.201s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.427s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.154s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.020s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.047s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.865s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	7.932s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.931s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.051s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.045s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.076s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	8.141s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.609s

$ go run ./cmd/informe && git diff --exit-code docs/informe
INFORME_DIFF_EXIT=0 (sin cambios en las regiones generadas)

$ go test -race -count=1 -v ./cmd/informe ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.326s (6/6 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.341s (13/13 PASS)

$ go test -race -count=1 -overlay <audit_overlay.json> -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 12 SVGs, 12 PNGs, 16 capturas, 0 avisos

$ gh pr checks 54   # run 44966461206 sobre fbe4b3a
Docker (imagen, clúster y caída de un nodo)	pass	2m0s
Ecosistema de agentes                       	pass	11s
Go (gofmt, vet, test -race)                 	pass	52s
MongoDB y Redis (tests de integración)      	pass	40s
OpenAPI 3 válido                            	pass	21s
Spin (modelo del protocolo y mutantes)      	pass	1m56s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes en verde localmente en Linux + CI verde en run `44966461206`) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; los tests de equivalencia pasan con `-race` dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`; `10.4` sigue generando la tabla desde `reports/bench_api_deez.json`) |
| Idempotencia `cmd/informe` y Word OOXML | ✅ (`INFORME_DIFF_EXIT=0`; 16 secciones, 12 SVGs, 12 PNGs, 16 capturas, 0 avisos; `$setOnInsert`, `$push`, `$max` renderizan como código literal sin disparar `<m:oMath>`, y `1ACC0065_PC4_U&lt;código&gt;.zip` escapa correctamente en XML) |

## Verificación contra los criterios del issue #53

| Criterio (#53) | Estado | Evidencia |
|---|---|---|
| 1. Sección 10.3 «Funcionamiento en Go» con fragmentos extraídos del fuente: enrutador, handler de predicción, cálculo del riesgo y encolado del registro | ✅ | `docs/informe/pc4/10-api.md:35-179` extrae `(*Servidor).Handler`, `(*Servidor).predecir`, `(*Modelo).prediccion` y `(*Servidor).registrar` mediante `<!-- generado:fuente:... -->`, coincidiendo con `internal/api/api.go` y `internal/api/modelo.go`. |
| 2. Sección 11.4 «Entidades en Go» con los `struct` con etiquetas `bson` de las colecciones, creación de índices, clave de Redis (todo extraído del fuente) y párrafo de registros que remite a 12.4 | ✅ | `docs/informe/pc4/11-bd.md:36-229` cubre las tres colecciones en el orden de `11.2`: `Modelo`, `ResumenEntrenamiento` y `(*Almacen).GuardarModelo` (`modelos`), `Entrenamiento` (`entrenamientos`), `ConsultaNormalizada` y `RegistroPrediccion` (`predicciones`), más `(*Almacen).crearIndices`, `ClavePrediccion` y el párrafo `**Registros.**` (`:224-228`) que remite a `12.4`. |
| 3. `cmd/informe` extrae también tipos (`fuente:RUTA#Tipo`), con test | ✅ | `cmd/informe/regiones.go:103-168` extrae tipos individuales y dentro de `type ( … )`; `cmd/informe/regiones_test.go:15-45, 57-60, 72-83` exige mínimos por sección (`08-distribuido.md` ≥ 9, `10-api.md` ≥ 4, `11-bd.md` ≥ 8) y cubre ambos formatos de declaración de tipo. |
| 4. La tabla 4.3 y las referencias cruzadas apuntan a las secciones nuevas | ✅ | `docs/informe/pc4/04-arquitectura.md:73-74` apunta a `10.3` y `11.4`; `docs/informe/pc4/12-evidencias.md:55` apunta a `10.4`; `rg` sobre `docs/informe/pc4/` confirma cero referencias obsoletas a `10.3` o `11.4`. |
| 5. `go run ./cmd/informe` idempotente; `go test ./cmd/informe ./cmd/word` en verde | ✅ | `INFORME_DIFF_EXIT=0`, `cmd/informe` (6/6 PASS), `cmd/word` (13/13 PASS) y `TestAuditPC4WordXML` (16 secciones, 12 SVGs, 12 PNGs, 16 capturas, 0 avisos). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | La tabla `4.3` (`04-arquitectura.md:73-74`) enlaza con exactitud API en Go y entidades/registros con `10.3`, `11.4`, `12.3` y `12.4`. |
| Implementación | 8 | sin cambios en backend | Solo amplía `cmd/informe` (herramienta de documentación) usando únicamente stdlib (`go/ast`, `go/token`). |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | 2 / 2 (en el alcance del issue #53) | Las secciones `10.3` y `11.4` muestran en Go real el camino crítico de la API y las tres colecciones + índices + clave Redis extraídos del repositorio, verificados por test y convertidos sin avisos a Word. |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

### H4 — Matiz de redacción entre `11.4` y `11.5` sobre creación frente a actualización en `entrenamientos` · MENOR

- **Dónde:** `docs/informe/pc4/11-bd.md:115` (contrastado con `docs/informe/pc4/11-bd.md:236-237`, `internal/almacen/almacen.go:143-147` e `internal/almacen/mongo/mongo.go:84-90, 128-131, 250-260`)
- **Evidencia:** En `11.4`, la línea añadida en `6dd14b2` dice *«Un entrenamiento (colección `entrenamientos`) se guarda entero con `ReplaceOne` y *upsert*:»* (`11-bd.md:115`), mientras que en `11.5` se afirma *«Los entrenamientos se crean con inserción, no con *upsert*: un ID repetido devuelve error y se reintenta con otro.»* (`11-bd.md:236-237`). En el código real ambas operaciones existen en momentos distintos: la creación usa `CrearEntrenamiento` con `InsertOne` sin *upsert* (`mongo.go:251-260`, llamado desde `internal/api/entrenamientos.go:106`), y las actualizaciones de progreso y estado final usan `GuardarEntrenamiento` con `ReplaceOne` y *upsert* (`mongo.go:84-90, 129-131`, llamado desde `internal/api/entrenamientos.go:200, 234`).
- **Corrección sugerida:** No bloquea el merge; si se pule antes de generar el `.docx` final (o en un retoque menor), precisar `11-bd.md:115` como: *«Un entrenamiento (colección `entrenamientos`) se crea con `InsertOne` y sus actualizaciones de progreso y cierre se guardan enteras con `ReplaceOne` y *upsert*:»*.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — La sección 11.4 omite el `struct` de la colección `modelos` (`almacen.Modelo` y `ResumenEntrenamiento`) e imprecisa su escritura en MongoDB · MAYOR | **RESUELTO** | En `6dd14b2`, `docs/informe/pc4/11-bd.md:38-113` incorpora `Modelo` (`almacen.go#Modelo`), `ResumenEntrenamiento` (`almacen.go#ResumenEntrenamiento`) y `(*Almacen).GuardarModelo` (`mongo.go#(*Almacen).GuardarModelo`) encabezando `11.4` en el orden de `11.2`, matiza la frase inicial (`:38-39`) y explica `$setOnInsert`, `$push` y `$max` (`:82-85`). |
| H2 (ronda 1) — `TestFragmentosSalenDelFuente` mantiene el umbral de 8 fragmentos y `extraerFuncion` deja el comentario/caso agrupado sin actualizar · MENOR | **RESUELTO** | En `6dd14b2`, `cmd/informe/regiones_test.go:22,28,39-44` verifica mínimos por sección (`08-distribuido.md` ≥ 9, `10-api.md` ≥ 4, `11-bd.md` ≥ 8); `cmd/informe/regiones.go:103-105,144-164` documenta tipos y recorre todas las `gd.Specs` en bloques `type ( … )`, verificado por `TestExtraerTipoDentroDeUnBloque` (`regiones_test.go:72-83`). |
| H3 (ronda 1) — El segundo commit (`f6eaa87`) no figura en la descripción del PR ni en el texto introductorio de `12.6` · MENOR | **RESUELTO** | En `6dd14b2`, `docs/informe/pc4/12-evidencias.md:76-77` añade explícitamente el historial de commits de `develop` con los commits de cada integrante, y la descripción del PR #54 incluye la viñeta de `f6eaa87` (`12.6` y `16.2`). |

## Cómo explicarlo en la sustentación

Las secciones `10.3` y `11.4` incorporan directamente desde el repositorio —sin copiar ni pegar a mano— el código Go que sustenta la API REST y la persistencia de la PC4: `Handler` (enrutador único verificado contra OpenAPI y protegido por `recuperar`), `predecir` (lectura sin bloqueo del modelo vía `atomic.Pointer`, *cache-aside* en Redis con presupuesto de 15 ms por operación y encolado no bloqueante), `(*Modelo).prediccion` (producto punto, sigmoide y calibración) y `registrar` (canal con búfer hacia el consumidor `registrador`), junto con los `struct` con etiquetas `bson` de las tres colecciones (`Modelo` + `ResumenEntrenamiento`, `Entrenamiento`, `ConsultaNormalizada` + `RegistroPrediccion`), el `UpdateOne` atómico con *upsert* de `GuardarModelo`, `crearIndices` (incluido el TTL de 90 días) y `ClavePrediccion`. `cmd/informe` se amplía para extraer también declaraciones de tipos (`fuente:RUTA#Tipo`, incluso dentro de `type ( … )`) y `TestFragmentosSalenDelFuente` garantiza por sección que el informe no se desincronice del código fuente.
