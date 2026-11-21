# Veredicto del PR #47 · Ronda 2

**APROBADO.** H1, H2 y H3 quedan resueltos en `534e0a4`. Se cumplen los tres criterios del issue #24. La revisión del diff completo no arrojó hallazgos nuevos. [Issue #24](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/24)

| Dato | Valor |
|---|---|
| Juez y fecha | Codex (GPT-6), skill `juez-cc65` · 20 de noviembre de 2026 |
| Diff revisado | `git diff origin/develop...origin/feature/24-word-plantilla`, desde `80bb4f8` hasta `08cf279` |
| Código probado | `534e0a4`. El único cambio posterior en la rama del PR es la incorporación de la [respuesta del implementador](respuesta-1.md) en `08cf279`. |

## Seguimiento de la ronda 1

| Hallazgo | Estado | Evidencia en `534e0a4` |
|---|---|---|
| H1 · BLOQUEANTE · `>` dentro de un atributo | **RESUELTO** | `cmd/word/main.go:447` lee la raíz con `RawToken` y en la línea 476 usa la posición en bytes del lector para insertar las declaraciones. `TestRaizConMayorQueEnUnAtributo` genera el DOCX, vuelve a leer la raíz y recupera el valor completo `Nivel > 1` junto con el contenido esperado (`cmd/word/plantilla_test.go:188`). |
| H2 · MAYOR · El test pasaba sin cargar el Markdown | **RESUELTO** | La prueba cambia al directorio temporal, pasa `01-x.md` como ruta relativa y falla ante cualquier aviso (`cmd/word/plantilla_test.go:51`). Cuenta elementos por URI y nombre local en el XML extraído del DOCX (línea 89). En una copia temporal omití la llamada `d.Convertir(s)` de `cmd/word/main.go:226`. El test de namespaces falló con cero figuras, ecuaciones y enlaces. En el código revisado pasó con una figura, dos elementos `oMath` y un enlace. |
| H3 · MAYOR · Prefijo existente asociado a otro URI | **RESUELTO** | `cmd/word/main.go:472` rechaza explícitamente el conflicto. `TestPrefijoDeLaPlantillaConOtroURI` comprueba que el error identifica `xmlns:pic`, el URI de la plantilla `urn:plantilla:portada` y el URI de imágenes OOXML (`cmd/word/plantilla_test.go:212`). La plantilla mínima con los prefijos ausentes sí genera el DOCX y declara cada URI una sola vez (línea 166). |

## Verificación automática

| Comando o comprobación | Resultado |
|---|---|
| `gofmt -l .`, `go vet ./...`, `git diff --check origin/develop...origin/feature/24-word-plantilla` | Los tres terminaron con código 0 y salida vacía. |
| `go test -race -count=1 ./...` | Pasaron todos los paquetes con pruebas. `cmd/word` pasó en 1,822 s. |
| `go test -race -count=1 -v -run 'TestPlantilla\|TestRaiz\|TestPrefijo\|TestXMLBienFormado' ./cmd/word` | Pasaron los cinco casos de `plantilla_test.go`, incluida la conservación de una única sección con sus atributos de página (línea 150). |
| CI de `08cf279` | Seis comprobaciones en verde. [Ejecución de GitHub Actions](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44827731334) |

El diff de código se limita a `cmd/word` y sus pruebas. La equivalencia del gradiente distribuido y la latencia p95 de la API quedan fuera del alcance del PR.

## Comparación con la plantilla actual

Ejecuté los generadores de `80bb4f8` y `534e0a4` desde la raíz con `-plantilla entregables/plantilla.docx -fuentes docs/informe/pc3 -codigos U00000000` y directorios de salida distintos. Ambos generaron las 13 secciones. El SHA-256 de la plantilla local fue `a189aa59a7d6bab202b4cc07832fb4b3a87a507fb2d830fa25a5c76c7ab5991a`.

Extraje `word/document.xml` de ambos DOCX. El de `develop` mide 881 185 bytes y el del PR, 881 256. Al insertar en la raíz del primero el espacio y la declaración `xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main"`, los dos archivos coincidieron **byte por byte**. Un segundo lector XML (`.NET XmlDocument`) abrió la salida del PR y contó nueve elementos de figura en el URI de imágenes OOXML.

## Puntaje de rúbrica en juego

| Criterio | Peso TB2 | Evaluación |
|---|---:|---|
| Informe | 2 | Cumplido el alcance del issue #24 para el generador Word. El puntaje del informe completo corresponde a la evaluación del entregable. |
| Planteamiento, implementación, interfaz y video | 3, 8, 5 y 2 | Fuera del alcance del cambio. |

## Cómo explicarlo en la sustentación

La plantilla conserva sus propiedades de página y el generador añade a la raíz los namespaces que necesita el contenido nuevo. Si un prefijo ya designa otro vocabulario, el generador señala el conflicto. Las pruebas abren el DOCX y cuentan las figuras, ecuaciones y enlaces por su URI para comprobar que Word puede identificarlos.
