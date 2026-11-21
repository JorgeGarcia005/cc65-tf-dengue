# Respuesta a la ronda 1 — PR #44 (informe PC4)

Veredicto: `ronda-1.md` (rama `revision/PR-44` @ `debec7c`). Los cinco hallazgos se aceptan; todos se
corrigen en un mismo commit sobre `feature/33-informe-pc4` (`8c45cb4`). La rama se actualizó antes
con `develop` (#46, #47) para que el texto describa el Dockerfile vigente.

| # | Severidad | Decisión | Corrección |
|---|---|---|---|
| H1 | Mayor | ACEPTO | `04-arquitectura.md` §4.2: la imagen compila solo `coordinator` y `node` (`deploy/Dockerfile`, PR #46); las herramientas (`loader`, `benchmark`, `informe`) se corren en el anfitrión con `go run`. `14-recomendaciones.md`: se retira la recomendación del tamaño de imagen (ya resuelta en #46) y se reemplaza por servir la SPA de TB2 en una imagen aparte. |
| H2 | Mayor | ACEPTO | Se usa `NODOS=N` (el coordinador espera exactamente N nodos) en lugar de `--scale`. El comando de entrenamiento único en §4.2 y en §12.2 pasa a `NODOS=3 VERIFICAR=true docker compose … up --build --exit-code-from coordinator`, y se explica que sin `VERIFICAR=true` no hay verificación dentro del contenedor. §8 nombra la variable junto al flag `-verificar`. |
| H3 | Mayor | ACEPTO | Citas en el texto: (Amdahl, 1967) en §9.3, (OpenAPI Initiative, 2020) en §10.1, (MongoDB, Inc., 2025; Redis Ltd., 2025) en §11.1; todas ya estaban en §15. En el texto se unifica "y" (APA en español): Dean y Ghemawat, Sergeev y Del Balso, Donovan y Kernighan. La lista de referencias conserva "&", como pide APA 7. |
| H4 | Menor | ACEPTO | `fig/epoca.svg`: el rectángulo del Coordinador usa el mismo estilo que los nodos (`fill="#cde2fb" stroke="#2a78d6" stroke-width="2"`, sin trazo discontinuo). |
| H5 | Menor | ACEPTO | `10-api.md`: la tabla lista todos los códigos de `docs/api/openapi.yaml` (413 en predicciones, 503 en `/cluster`, 400 en entrenamientos). La corrida caliente se describe como 2000 peticiones sobre 32 consultas precalentadas, en una mitad del espacio de consultas disjunta de la corrida fría. |

Verificación: `go run ./cmd/informe` sin diferencias en las regiones generadas y
`go test ./cmd/informe ./cmd/word` en verde.

Commit de corrección: `8c45cb4`.
