# Respuesta a la ronda 2 — PR #54

Veredicto: `ronda-2.md` (rama `revision/PR-54` @ `2a958c7`), `APROBADO` con un hallazgo menor.

| # | Severidad | Decisión | Corrección |
|---|---|---|---|
| H4 | Menor | ACEPTO | `11-bd.md` §11.4: un entrenamiento se crea con `InsertOne` (`CrearEntrenamiento`, que rechaza un ID repetido con `ErrDuplicado`) y cada avance o cierre lo reescribe con `ReplaceOne` y *upsert* (`GuardarEntrenamiento`), en `e5ba397`. Cambio de una línea de texto; `go test ./cmd/informe` en verde. |
