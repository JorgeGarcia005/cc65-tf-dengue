# Respuesta — PR #47 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-20 · **Base:** `feature/24-word-plantilla`

Acepto los tres hallazgos; los tres quedan corregidos en `534e0a4`. Antes de corregir, cada test
nuevo reprodujo el defecto sobre `676c12a`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `534e0a4`: `declararNamespaces` (`cmd/word/main.go`) lee la raíz con `xml.Decoder.RawToken` y toma el fin de la etiqueta de `InputOffset`, no del primer `>`. `TestRaizConMayorQueEnUnAtributo` (raíz con `upc:nota="Nivel > 1"`): antes `armar: document.xml mal formado: XML syntax error on line 1: attribute name without = in element`; ahora genera el documento y conserva `upc:nota = "Nivel > 1"`. Con la plantilla real más `xmlns:upc="urn:plantilla:upc" upc:nota="Nivel > 2"` en la raíz, `develop` y este PR escriben el informe (13 secciones, 71 KB) |
| H2 | MAYOR | ACEPTO | `534e0a4`: el helper `generar` corre desde la carpeta de la sección (`t.Chdir`) con ruta relativa, como `main`, y hace `t.Fatalf` si `armar` deja avisos. `exigirContenido` cuenta por nombre expandido `{pic}pic`, `{a}graphic`, `{wp}inline`, `{asvg}svgBlip` (1 c/u), `{m}oMath` (2) y `{w}hyperlink` (1). Mutante «`armar` no convierte las secciones» → fallan 3 tests |
| H3 | MAYOR | ACEPTO | `534e0a4`: si la plantilla declara un prefijo de `nsDocumento` con otro URI, `armar` falla con `la plantilla declara xmlns:pic="urn:plantilla:portada" y el contenido generado lo usa para "http://schemas.openxmlformats.org/drawingml/2006/picture"`. Elegí rechazar con diagnóstico (una de las dos correcciones sugeridas): la carátula oficial usa los URI estándar y redeclarar en cada figura agregaría código sin caso real. `TestPrefijoDeLaPlantillaConOtroURI`: antes `armar aceptó xmlns:pic con otro URI`. `TestPlantillaSinNamespacesDeFiguras` comprueba ahora cada declaración por prefijo **y** URI |

## Equivalencia con la plantilla actual

Binario de `develop` (`origin/develop`) contra este PR, ambos desde la raíz del repo con
`entregables/plantilla.docx` y `docs/informe/pc3`: el `document.xml` es idéntico salvo la etiqueta
`<w:document>`, que gana solo `xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main"`
(23 → 24 declaraciones).

## Verificación

```text
$ gofmt -l (archivos Go)              → vacío
$ go vet ./...                         → sin salida
$ go test -race -count=1 ./...         → todos ok
$ git diff --check                     → sin salida
```
