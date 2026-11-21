# Veredicto — PR #47 · Ronda 1

**CAMBIOS** · 1 hallazgo bloqueante y 2 mayores. Con la plantilla actual se cumple la equivalencia solicitada. La inserción de namespaces introduce una regresión con un atributo XML válido, y el test de la plantilla mínima pasa incluso cuando omite toda la sección de prueba.

| Dato | Valor |
|---|---|
| Juez | Codex (GPT-6), skill `juez-cc65` |
| Fecha | 20 de noviembre de 2026, Lima |
| Rama revisada | `feature/24-word-plantilla` en `676c12a` |
| Base y diff | `git diff origin/develop...origin/feature/24-word-plantilla`, con base `80bb4f8`. Cambian `cmd/word/main.go` y `cmd/word/plantilla_test.go`. |
| Entregable | Generador del informe Word. [Issue #24](https://github.com/JorgeGarcia005/cc65-tf-dengue/issues/24) · [PR #47](https://github.com/JorgeGarcia005/cc65-tf-dengue/pull/47) |

## Comparación con la plantilla actual

Ejecuté los generadores de ambos commits desde el mismo directorio, con `-plantilla entregables/plantilla.docx -fuentes docs/informe/pc3`, las mismas 13 secciones, un código de salida ficticio y directorios de salida temporales distintos. El SHA-256 de la plantilla usada fue `a189aa59a7d6bab202b4cc07832fb4b3a87a507fb2d830fa25a5c76c7ab5991a`.

| Parte del DOCX | Bytes | SHA-256 |
|---|---:|---|
| `word/document.xml` de `develop` | 881 185 | `b0a28da6d51b0c961b83bb1823c605a6d5ef56d52dc53f7c6b2583f03c5ef8c3` |
| `word/document.xml` del PR | 881 256 | `1fd24ee5310ee04e7684c758c3db103c56615db8ceca4890fa55987b1d2e341b` |

Al insertar en el XML de `develop` exactamente el espacio y la declaración `xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main"` antes del cierre de la etiqueta inicial de `w:document`, el resultado **coincidió byte a byte** con el del PR. Se conservaron espacios, saltos de línea, atributos y contenido. Un lector XML independiente de Go también abrió el XML del PR y encontró nueve elementos de figura en el namespace de imagen de OOXML.

## Criterios del issue #24

| Criterio | Resultado comprobado |
|---|---|
| Aceptar `w:sectPr` con atributos | Cumplido. `cmd/word/main.go:163` contiene la expresión pedida y `TestPlantillaConSectPrConAtributos` comprueba que se conserva el atributo y queda una sola sección. |
| Declarar los namespaces de figuras y matemáticas en la raíz cuando faltan | Cumplido en la plantilla mínima del issue y en la plantilla actual. Los casos válidos de raíz descritos en H1 y H3 quedan abiertos. |
| Generar y probar el DOCX mínimo con contenido que use los namespaces | Con una ruta relativa al Markdown obtuve un DOCX con una figura, dos elementos `m:oMath` y un enlace. El test versionado pasa con esos tres recuentos en cero en la ejecución local. Véase H2. |

## Hallazgos

### H1 — Un carácter `>` dentro de un atributo de la raíz impide generar el informe · BLOQUEANTE

`cmd/word/main.go:443` toma el primer `>` como fin de la etiqueta. En un valor entre comillas ese carácter forma parte del atributo. En una copia temporal de la plantilla actual añadí a la raíz `xmlns:upc="urn:plantilla:upc" mc:Ignorable="upc" upc:nota="Nivel > 1"`. Comprobé que un lector XML aceptaba la entrada y ejecuté ambos generadores con las mismas fuentes. `develop` escribió el informe y terminó con código 0. El PR insertó la declaración dentro de la nota y terminó con código 1 y `document.xml mal formado: XML syntax error on line 1: attribute name without = in element`.

**Corrección sugerida.** Obtener el elemento raíz y el final de su etiqueta inicial mediante un lector XML que exponga la posición en bytes, e insertar en esa posición conservando el resto del documento. Añadir el caso de atributo entrecomillado a las pruebas y repetir la comparación exacta con la plantilla actual.

### H2 — El test de namespaces pasa sin leer el Markdown que contiene la figura · MAYOR

`cmd/word/plantilla_test.go:49` devuelve la ruta absoluta de `01-x.md`. La llamada en la línea 94 descarta los avisos de `armar`. El conversor intenta abrir `filepath.Join(".", ruta)` en `cmd/word/md.go:112` y continúa ante el error. Repetí la llamada del test e inspeccioné el XML extraído del ZIP:

| Entrada de la prueba | Avisos | Figuras | Ecuaciones `oMath` | Enlaces |
|---|---:|---:|---:|---:|
| Ruta de `seccionConTodo(t)`, igual que en el PR | 1, «no se pudo leer …01-x.md» | 0 | 0 | 0 |
| Mismo archivo, desde su carpeta y con ruta relativa | 0 | 1 | 2 | 1 |

**Corrección sugerida.** Ejecutar el test desde la carpeta temporal y pasar el nombre relativo, o admitir rutas absolutas explícitamente. Exigir cero avisos y buscar los elementos por URI y nombre local en el XML generado. Así el test fallará si deja de incorporar texto, figuras o enlaces aunque la raíz siga declarando los prefijos.

### H3 — Un prefijo existente con otro URI cambia el significado de la figura · MAYOR

En `cmd/word/main.go:447` basta con que exista `xmlns:pic` para omitir la declaración requerida. En una plantilla mínima con `xmlns:pic="urn:plantilla:portada"` y el Markdown cargado correctamente, `armar` terminó con éxito y cero avisos. El lector XML mostró que el elemento generado `pic:pic` pertenecía a `urn:plantilla:portada`. En el control pertenecía a `http://schemas.openxmlformats.org/drawingml/2006/picture`. El documento puede pasar la validación sintáctica aunque la figura quede fuera del vocabulario de imágenes de Word.

**Corrección sugerida.** Comprobar también el URI asociado. Si hay conflicto, conservar la vinculación de la carátula y declarar el URI correcto en el contenido generado, o rechazar la plantilla con un diagnóstico explícito. Probar ambos vínculos por su nombre expandido.

## Verificación automática

```text
$ go test -race -count=1 -v -run '^(TestPlantillaConSectPrConAtributos|TestPlantillaSinNamespacesDeFiguras|TestXMLBienFormadoRechazaPrefijosSinDeclarar)$' ./cmd/word
--- PASS: TestPlantillaConSectPrConAtributos (0.01s)
--- PASS: TestPlantillaSinNamespacesDeFiguras (0.01s)
--- PASS: TestXMLBienFormadoRechazaPrefijosSinDeclarar (0.00s)
PASS

$ go test -race -count=1 ./...
ok  github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word     1.877s
ok  github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml  9.831s
```

La suite completa, `gofmt -l .`, `go vet ./...` y `git diff --check` terminaron con código 0. Los seis jobs del CI pasaron. El diff se limita al generador Word y sus pruebas. La equivalencia distribuida y el p95 de la API quedan fuera del alcance de este cambio. [CI del commit revisado](https://github.com/JorgeGarcia005/cc65-tf-dengue/actions/runs/44818060327)

## Puntaje de rúbrica en juego

| Criterio | Peso TB2 | Evaluación de esta ronda |
|---|---:|---|
| Planteamiento | 3 | Fuera del alcance. |
| Implementación | 8 | Fuera del alcance del sistema distribuido. |
| Interfaz | 5 | Fuera del alcance. |
| Informe | 2 | Avance parcial. La salida del informe actual conserva su contenido. Para acreditar la compatibilidad del generador quedan las correcciones H1–H3. |
| Video | 2 | Fuera del alcance. |

## Seguimiento de rondas previas

Primera ronda del PR #47.

## Cómo explicarlo en la sustentación

El generador conserva las propiedades de página de la carátula, aun cuando Word agrega atributos a `w:sectPr`. Las ecuaciones y figuras dependen de que cada prefijo apunte al vocabulario OOXML correcto. Por eso la prueba debe abrir el DOCX y localizar la ecuación y la figura en el XML, además de revisar las declaraciones de la raíz.
