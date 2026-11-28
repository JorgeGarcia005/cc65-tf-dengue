# Veredicto — PR-96 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/95-informe-tb2-base` @ `dc2fd2e`
- **Fecha:** 2026-11-27
- **Decisión:** CAMBIOS

El PR construye con solo stdlib la infraestructura del informe final TB2 (bloque 1 de `#71`): formato APA 7 para tablas (tres reglas horizontales sin relleno, pie con «**Tabla N**» en negrita y título en cursiva encima con `<w:keepNext/>`, y «*Nota.*» debajo) y figuras/capturas con numeración compartida, resolución en dos pasadas de referencias cruzadas `@fig:id` y `@tab:id` (incluidas hacia adelante), índices de figuras y tablas mediante campos `PAGEREF` actualizados también en `scripts/word-final.ps1`, incrustación verificada de archivos y fragmentos no Go (`archivo:RUTA` y `archivo:RUTA#NOMBRE`) más cifras en línea `<!--v:EXPR-->…<!--/v-->` en `cmd/informe`, gráficos SVG científicos en blanco y negro con marcadores diferenciados y coma decimal, primer diagrama UML 2.5 de despliegue en Graphviz (`docs/informe/tb2/diagramas/arquitectura.dot`), opción `-quitar` para la carátula y esqueleto completo de las 23 secciones de `docs/informe/tb2/`. Todas las verificaciones automáticas (`gofmt`, `go vet`, `go test -race ./...`) y los 7 jobs del CI pasan en verde. Antes de fusionar esta base sobre la que se escribirán los bloques 2–5 de `#71`, se solicitan **2 ajustes mayores** (**H1** y **H2**) y **1 menor** (**H3**): `textoVisible` y `partirFila` en `cmd/word/md.go` no descuentan comentarios `<!--v:…-->` ni enlaces al calcular anchos de columnas o poblar los índices de figuras/tablas (**H1**), y falta un test en `cmd/word` que convierta `docs/informe/tb2/*.md` en el CI exigiendo cero avisos de referencias o etiquetas (**H2**).

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && bash -n scripts/figuras.sh
skills: ok

$ go run ./cmd/informe && git diff --exit-code
2026/11/28 03:38:21 escrito docs/informe/tb2/fig/casos_por_ano.svg
2026/11/28 03:38:21 escrito docs/informe/tb2/fig/riesgo_departamento.svg
2026/11/28 03:38:21 escrito docs/informe/tb2/fig/riesgo_semana.svg
2026/11/28 03:38:21 escrito docs/informe/tb2/fig/speedup_gradiente.svg
2026/11/28 03:38:21 escrito docs/informe/tb2/fig/speedup_loader.svg
2026/11/28 03:38:21 escrito docs/informe/tb2/fig/speedup_cluster.svg
(sin diferencias en el árbol de trabajo)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.730s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.696s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.599s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.029s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.879s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.267s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.266s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.408s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.361s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.050s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.226s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.085s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	39.501s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	24.748s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.565s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.673s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.043s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.938s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.080s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.802s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.223s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.799s

$ gh pr checks 96 -R JorgeGarcia005/cc65-tf-dengue   # run 46103341217 sobre dc2fd2e
7/7 jobs en verde (Ecosistema, Spin, OpenAPI 3, MongoDB/Redis, Go -race, SPA, Docker)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`25/25` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`) |
| `cmd/informe` y `cmd/word` (APA 7, referencias e índices) | ⚠️ (pasan los tests unitarios actuales, pero faltan cubrir `<!--v:…-->` en celdas/pies y validar `docs/informe/tb2/*.md` sin avisos en CI; ver **H1** y **H2**) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | mejora diagrama | Incorpora el diagrama de despliegue UML 2.5 (`docs/informe/tb2/fig/arquitectura.svg`) alineado con los servicios, redes, puertos y volúmenes reales de Docker Compose. |
| Implementación | 8 | sin cambios | No toca el núcleo concurrente/distribuido; solo stdlib en `cmd/informe` y `cmd/word`. |
| Interfaz | 5 | sin cambios | No modifica `web/`. |
| Informe | 2 | Pendiente **H1** y **H2** | Deja lista la estructura de 23 secciones (cubriendo `a)`–`j)` del enunciado y las observaciones de PC3/PC4), formato APA 7, regiones de archivos no Go y gráficos en blanco y negro; corregir **H1** y **H2** blinda la escritura de los bloques 2–5 de `#71`. |
| Video | 2 | sin cambios | No afecta el video. |

## Hallazgos

### H1 — `textoVisible` y `partirFila` no descuentan comentarios `<!--v:…-->`, enlaces `[texto](url)` ni referencias cruzadas al medir columnas de tablas o poblar los índices de figuras y tablas · MAYOR

- **Dónde:** `cmd/word/md.go:338`, `cmd/word/md.go:507-513`, `cmd/word/md.go:537-557`, `cmd/word/md.go:568-570` y `cmd/word/md.go:592`
- **Evidencia:**
  1. El propósito de este PR (`#95` y `AGENTS.md` §3.6) es que toda cifra y tabla de los bloques 2–5 de `#71` provenga de `reports/*.json` mediante valores en línea `<!--v:EXPR-->valor<!--/v-->`. Mientras `(*documento).inline` (`cmd/word/md.go:623`) sí ejecuta `s = d.resolver(reComentario.ReplaceAllString(s, ""))` al generar las corridas XML, `textoVisible` (`cmd/word/md.go:568-570`) solo elimina `"**"`, `` "`" `` y `"*"`.
  2. **Deformación de anchos en tablas APA (`cmd/word/md.go:508`):** `anchosColumnas` calcula `v := textoVisible(f[j])` sobre el Markdown crudo. Una celda numérica como `| <!--v:dos (index .Cluster.Mediciones .Ultimo).Speedup-->4,80<!--/v--> |`, cuyo texto real en Word es `"4,80"` (4 caracteres), deja el token `".Ultimo).Speedup-->4,80<!--/v-->"` de 32 caracteres sin espacios, asignándole un mínimo de `32 × 110 + 260 = 3 780 dxa` (~41 % del ancho total `9 122 dxa`) y comprimiendo las columnas de texto de la tabla. Lo mismo ocurre con enlaces `[texto](https://…)` en tablas de anexos/matrices (medidos por la longitud de la URL completa en vez de `texto`) y con referencias `@fig:`/`@tab:` sin resolver antes de medir.
  3. **Fuga de marcado HTML en índices y `descr` (`cmd/word/md.go:338, 592`):** En `(*documento).pie`, `Titulo: textoVisible(d.resolver(titulo))` no aplica `reComentario.ReplaceAllString`, por lo que si un título de tabla o figura lleva un valor generado (p. ej. `Tabla: Resultados con <!--v:…-->5<!--/v--> nodos`), el cuerpo muestra *«Resultados con 5 nodos»* pero el **Índice de tablas / Índice de figuras** (`cmd/word/main.go:414`) imprime literalmente `<!--v:…-->5<!--/v-->` (al igual que `wp:docPr descr="…"` en `md.go:592`).
  4. **Barras `|` en expresiones de plantilla dentro de celdas (`cmd/word/md.go:537-557`):** `partirFila` ignora `|` dentro de `` `…` ``, pero no dentro de `<!--…-->`. Si una celda usa un *pipeline* de `text/template` (`<!--v:.S.M.E.Test.AUC | cuatro-->0,6221<!--/v-->`), `partirFila` parte la celda por la mitad.
- **Corrección sugerida:**
  - Crear un método `(*documento).textoPlano(s string) string` (o limpiar antes de `textoVisible`) que aplique `reComentario.ReplaceAllString(s, "")`, `d.resolver(...)`, reemplace `[texto](url)` por `texto` y `<url>` por `url`, y elimine `**`, `*` y `` ` ``; usarlo en `pie` (`md.go:338`), `figura` (`md.go:592`) y antes de `anchosColumnas` en `tabla` (`md.go:456, 508`).
  - En `partirFila` (`md.go:537-557`), no partir por `|` cuando se esté dentro de `<!-- … -->`, y añadir un test en `cmd/word/apa_test.go` con celdas y pies que contengan `<!--v:…-->` y `[texto](url)`.

### H2 — Falta un test en `cmd/word` que valide las secciones reales de `docs/informe/tb2/*.md` sin avisos en el CI · MAYOR

- **Dónde:** `cmd/word/main.go:72-74` y `cmd/word/apa_test.go` (comparar con `TestInformeTB2AlDia` en `cmd/informe/regiones_test.go:88-112`)
- **Evidencia:**
  1. Como `entregables/plantilla.docx` no se versiona (`entregables/` está en `.gitignore`), `make check` y el job `Go` del CI nunca ejecutan `cmd/word` sobre `docs/informe/tb2/*.md`.
  2. Además, en `cmd/word/main.go:72-74` los `avisos` (referencia `@fig:…` o `@tab:…` a una etiqueta inexistente, `{#…}` repetida, figura `.svg` o archivo incluido que no se puede leer, o bloque `mermaid`) solo imprimen `log.Printf("aviso: %s", a)` y terminan con código `0`. En los bloques 2–5 de `#71` (23 archivos escritos en varios PRs), cualquier errata `@fig:typo` dejaría el texto literal `@fig:typo` en el `.docx` final sin que fallen ni el CI ni `make informe`.
- **Corrección sugerida:**
  - Agregar en `cmd/word` un test (p. ej. `TestInformeTB2SinAvisos`, reutilizando `plantillaMinima(raizSoloWR)` de `cmd/word/plantilla_test.go`) que convierta todas las secciones `../../docs/informe/tb2/[0-9][0-9]-*.md` y exija `err == nil` y `len(avisos) == 0` (verificado hoy: tanto `tb2` como `pc4` producen `0` avisos).
  - Opcionalmente, hacer que `cmd/word/main.go` falle con código distinto de cero cuando `len(avisos) > 0` (o salvo flag explícito), para que `make informe` tampoco genere silenciosamente un `.docx` con referencias rotas.

### H3 — Detalles de `lenguajeDe`/BOM en `regionArchivo`, párrafo vacío antes de `Nota.` en tablas y redacción de la nota de `@fig:arquitectura` · MENOR

- **Dónde:** `cmd/informe/regiones.go:160-191`, `cmd/word/md.go:496` y `docs/informe/tb2/04-arquitectura.md:11`
- **Evidencia:**
  1. En `cmd/informe/regiones.go:160-179`, `lenguajeDe` busca `filepath.Ext(base)`, por lo que `deploy/.env.example` (`Ext` = `".example"`) y `scripts/word-final.ps1` (`Ext` = `".ps1"`) caen en `{"text", "", ""}` sin cabecera `# ruta`. Asimismo, `regionArchivo` (`:191`) no recorta un posible BOM UTF-8 inicial (`\uFEFF`, presente en `scripts/word-final.ps1:1` para PowerShell 5.1) antes de incrustar el archivo en el `.md`.
  2. En `cmd/word/md.go:496`, `tabla` añade siempre `<w:p><w:pPr><w:spacing w:before="0" w:after="120"/></w:pPr></w:p>` tras `</w:tbl>`, lo que intercala un renglón vacío completo entre el borde inferior de la tabla APA y `Nota.` (`md.go:353`), a diferencia de `figura` (`md.go:591`), donde `Nota.` aparece inmediatamente debajo.
  3. En `docs/informe/tb2/04-arquitectura.md:11`, la nota de `@fig:arquitectura` dice *«las flechas indican quién inicia la comunicación y su protocolo»*, pero en `docs/informe/tb2/diagramas/arquitectura.dot:60, 63-67` la arista `coord -> nodo` es bidireccional (`dir=both`, y en el código quien marca por TCP hacia `:9000` es `node`) y las flechas con `data/bronze` y `data/gold` representan flujo de archivos en volumen, no inicio de comunicación por red.
- **Corrección sugerida:** Soportar `.ps1` y prefijo `.env.` + `strings.TrimPrefix(..., "\uFEFF")` en `regiones.go`; omitir o compactar el párrafo vacío tras `</w:tbl>` cuando siga `Nota: `; y matizar la nota de `04-arquitectura.md:11` distinguiendo conexiones de red y lectura/escritura de volúmenes.

## Seguimiento de rondas previas

Primera ronda para este PR.

## Cómo explicarlo en la sustentación

Todo el informe final se construye de forma reproducible desde el repositorio usando únicamente la biblioteca estándar de Go: `cmd/informe` lee los reportes `reports/*.json` para evaluar cada cifra incrustada en el texto (`<!--v:EXPR-->`), dibuja los gráficos científicos en SVG blanco y negro y extrae directamente del árbol fuente las funciones Go, los `Dockerfile`, los archivos de Docker Compose y el modelo Promela (`TestInformeTB2AlDia` falla en el CI si algún fragmento o cifra se desfasa). A continuación, `cmd/word` convierte las 23 secciones Markdown a OpenXML aplicando el formato APA 7 (tablas de tres líneas sin relleno, pies numerados encima, referencias cruzadas en dos pasadas e índices de figuras y tablas con `PAGEREF`).
