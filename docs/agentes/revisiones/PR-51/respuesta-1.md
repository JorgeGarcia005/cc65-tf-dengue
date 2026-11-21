# Respuesta a la ronda 1 — PR #51 (figuras del clúster)

Veredicto: `ronda-1.md` (rama `revision/PR-51` @ `d29a9f1`). Los tres hallazgos se aceptan y se
corrigen en `c5d5e54`. Las tres figuras se volvieron a renderizar (rsvg) para comprobar que ningún texto toca
una flecha o un borde.

| # | Severidad | Decisión | Corrección |
|---|---|---|---|
| H1 | Mayor | ACEPTO | `nodo.svg`: `listo · gradiente · error` pasa a `x=62`, a la izquierda de la flecha (`x=70`). `flujo.svg`: el recuadro de `cubrir()` mide 56 px de alto; la etiqueta de aborto se movió al canal vertical, lejos del borde; el tronco de `Esperar` y las tres ramas quedan alineados en `x=602`. `topologia.svg`: la etiqueta `TCP` superior va por encima de su flecha, como las otras dos. |
| H2 | Mayor | ACEPTO | `flujo.svg`: `Difundir la época e` tiene ahora su salida `sin nodo sano` hacia `Abortar` por el mismo canal que la del `cubrir()` de la segunda pasada (etiqueta «sin nodo sano (cualquier cubrir)»), igual que los dos `if err := c.cubrir()` de `GradienteErr`; donde ese canal cruza la flecha `no` se dibuja un salto. `Fin` distingue los modos: «modo único: reports/*.json y fin(completo) a los nodos» y «modo API: publica el modelo; los nodos siguen conectados». |
| H3 | Menor | ACEPTO | `nodo.svg`: `p₀ … pₖ₋₁` se reemplaza por `p<tspan>0</tspan> … p<tspan>K−1</tspan>` con dígitos y `K` mayúscula, sin caracteres fuera de Arial y consistente con «bloques 0 … K−1». |

Verificación: `xmllint --noout` sobre los SVG, `go run ./cmd/informe` sin cambios en las regiones
generadas, `go test ./cmd/informe ./cmd/word` en verde.
