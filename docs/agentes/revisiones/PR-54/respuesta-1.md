# Respuesta a la ronda 1 — PR #54 (API y entidades en Go)

Veredicto: `ronda-1.md` (rama `revision/PR-54` @ `3b464c0`). Los tres hallazgos se aceptan y se
corrigen en `6dd14b2`.

| # | Severidad | Decisión | Corrección |
|---|---|---|---|
| H1 | Mayor | ACEPTO | `11-bd.md` §11.4 sigue el orden de la tabla 11.2: primero `modelos` con las regiones `almacen.go#Modelo` y `almacen.go#ResumenEntrenamiento`, luego `(*Almacen).GuardarModelo` (`mongo.go`) con el párrafo que explica el `UpdateOne` atómico con *upsert*: `$setOnInsert` (pesos y esquema solo si la versión es nueva), `$push` (resumen) y `$max` (`publicado`). La frase inicial se matiza: el driver codifica y decodifica el `struct`; `entrenamientos` se guarda con `ReplaceOne` y *upsert*. `11-bd.md` pasa de 5 a 8 fragmentos. |
| H2 | Menor | ACEPTO | `TestFragmentosSalenDelFuente` exige un mínimo por sección (`08-distribuido.md` ≥ 9, `10-api.md` ≥ 4, `11-bd.md` ≥ 8): borrar los bloques de 10.3 u 11.4 hace fallar el test. `extraerFuncion` documenta los tipos y recorre todas las `gd.Specs`: dentro de un bloque `type ( … )` devuelve `type Nombre …` con su comentario (nuevo `TestExtraerTipoDentroDeUnBloque`). |
| H3 | Menor | ACEPTO | `12-evidencias.md` §12.6 menciona el historial de commits de `develop` con los commits de cada integrante. La descripción del PR #54 suma la viñeta de `f6eaa87` (marco de 12.6 y nombre del zip en 16.2). |

Verificación: `gofmt -l cmd` vacío, `go vet ./cmd/informe`, `go run ./cmd/informe` idempotente y
`go test ./cmd/informe ./cmd/word` en verde.
