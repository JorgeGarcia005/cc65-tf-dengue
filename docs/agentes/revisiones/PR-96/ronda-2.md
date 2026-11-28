# Veredicto — PR-96 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/95-informe-tb2-base` @ `febee68`
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

Los **3 hallazgos** de la ronda 1 (2 mayores y 1 menor) quedaron íntegramente resueltos y verificados con pruebas automatizadas y el CI en verde:

- **Medición de celdas, índices y `partirFila` sobre el texto visible (H1):** `cmd/word/md.go:594-599` introduce `(*documento).textoPlano`, que limpia comentarios `<!--v:…-->`, resuelve `@fig:`/`@tab:` sin duplicar avisos (`resolverCon(..., false)`, `md.go:728-744`), reduce `[texto](url)` y `<url>` a su texto visible y quita las marcas de formato antes de poblar `d.entradas` (`md.go:342`), calcular `anchosColumnas` (`md.go:461-467`) y escribir `wp:docPr descr="…"` (`md.go:630`). Además, `partirFila` (`md.go:557-579`) ignora barras `|` dentro de comentarios `<!-- … -->`, admitiendo *pipelines* de `text/template` dentro de celdas. Cubierto por `TestTextoPlanoEnCeldasEIndices` (`cmd/word/apa_test.go:197-236`).
- **Validación en el CI de `docs/informe/tb2/*.md` y `pc4/*.md` sin avisos (H2):** `TestInformesSinAvisos` (`cmd/word/plantilla_test.go:258-273`) convierte todas las secciones reales de `docs/informe/tb2` y `docs/informe/pc4` con `plantillaMinima` exigiendo `err == nil` y cero avisos, y `cmd/word/main.go:51, 78-80` aborta con código de salida distinto de cero ante cualquier aviso salvo `-permitir-avisos`.
- **Lenguajes/BOM en `regionArchivo`, unión de tabla con `Nota.` y precisión en `@fig:arquitectura` (H3):** `cmd/informe/regiones.go:165, 173-175, 196` incorpora `.ps1` (`powershell`) y `.env`/`.env.*` (`bash`) con comentario de ruta y elimina BOM UTF-8 (`\uFEFF`) y CRLF (cubierto por `TestRegionArchivoLenguajes`, `cmd/informe/archivo_test.go:94-106`); `cmd/word/md.go:225-229, 507-510` suprime el párrafo vacío tras `</w:tbl>` cuando le sigue `Nota: `; y `docs/informe/tb2/04-arquitectura.md:11` distingue con exactitud las conexiones de red (incluido el TCP persistente iniciado por cada nodo) del flujo y lectura de archivos en volúmenes.

## Verificación automática

```text
$ git rev-parse --short HEAD
febee68

$ git diff --check origin/integracion/tb2...origin/feature/95-informe-tb2-base
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && bash -n scripts/figuras.sh && go run ./cmd/informe && git diff --exit-code
skills: ok
2026/11/28 03:50:48 escrito docs/informe/tb2/fig/casos_por_ano.svg
2026/11/28 03:50:48 escrito docs/informe/tb2/fig/riesgo_departamento.svg
2026/11/28 03:50:48 escrito docs/informe/tb2/fig/riesgo_semana.svg
2026/11/28 03:50:48 escrito docs/informe/tb2/fig/speedup_gradiente.svg
2026/11/28 03:50:48 escrito docs/informe/tb2/fig/speedup_loader.svg
2026/11/28 03:50:48 escrito docs/informe/tb2/fig/speedup_cluster.svg
(sin diferencias en el árbol de trabajo)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.516s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.683s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.385s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.587s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.265s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.280s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.410s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.018s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.352s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.051s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	3.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.052s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.170s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.000s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.784s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.046s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.175s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.071s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.799s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.391s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.802s

$ gh pr checks 96 -R JorgeGarcia005/cc65-tf-dengue   # run 46104675974 sobre febee68
7/7 jobs en verde (Ecosistema, Spin, OpenAPI 3, MongoDB/Redis, Go -race, SPA, Docker)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`25/25` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`) |
| `cmd/informe` y `cmd/word` (APA 7, referencias, índices y secciones TB2/PC4) | ✅ (`TestInformesSinAvisos`, `TestInformeTB2AlDia` y `TestTextoPlanoEnCeldasEIndices` en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Diagrama de despliegue UML 2.5 (`docs/informe/tb2/fig/arquitectura.svg`) y su nota técnica alineados fielmente con Docker Compose y el código real. |
| Implementación | 8 | sin cambios | No toca el núcleo concurrente/distribuido; solo stdlib en `cmd/informe` y `cmd/word`. |
| Interfaz | 5 | sin cambios | No modifica `web/`. |
| Informe | 2 | base lista (bloque 1 de `#71`) | Deja lista y protegida en CI toda la infraestructura del informe final TB2 (APA 7, referencias cruzadas, índices con `PAGEREF`, regiones de archivos no Go, valores en línea desde `reports/*.json`, gráficos académicos y esqueleto de 23 secciones). |
| Video | 2 | sin cambios | No afecta el video. |

## Hallazgos

Sin hallazgos abiertos en esta ronda.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · MAYOR) | RESUELTO | `b6ce199`: `(*documento).textoPlano` (`cmd/word/md.go:594-599`) limpia `<!--v:…-->`, resuelve `@fig:`/`@tab:` sin repetir avisos y reduce enlaces a su texto visible para `anchosColumnas`, `d.entradas` y `descr`; `partirFila` (`md.go:557-579`) no parte por `\|` dentro de `<!-- … -->`. Verificado en `TestTextoPlanoEnCeldasEIndices`. |
| H2 (ronda 1 · MAYOR) | RESUELTO | `661f821`: `TestInformesSinAvisos` (`cmd/word/plantilla_test.go:258-273`) valida `docs/informe/tb2` y `docs/informe/pc4` con cero avisos en `go test`, y `cmd/word/main.go:78-80` falla si hay avisos salvo `-permitir-avisos`. |
| H3 (ronda 1 · MENOR) | RESUELTO | `ebc8dae` y `b6ce199`: soporte de `.ps1` y `.env.*` + recorte de BOM/CRLF en `cmd/informe/regiones.go` (`TestRegionArchivoLenguajes`), supresión del párrafo vacío entre `</w:tbl>` y `Nota.` (`cmd/word/md.go:225-229, 507-510`), y nota precisa en `docs/informe/tb2/04-arquitectura.md:11`. |

## Cómo explicarlo en la sustentación

Todo el informe final se construye de forma reproducible desde el repositorio usando únicamente la biblioteca estándar de Go: `cmd/informe` lee los reportes `reports/*.json` para evaluar cada cifra incrustada en el texto o en las tablas (`<!--v:EXPR-->`), dibuja los gráficos científicos en SVG blanco y negro y extrae directamente del árbol fuente las funciones Go, los `Dockerfile`, los archivos de Docker Compose, scripts y el modelo Promela (`TestInformeTB2AlDia` falla en el CI si algún fragmento o cifra se desfasa). A continuación, `cmd/word` convierte las 23 secciones Markdown a OpenXML aplicando el formato APA 7 (tablas de tres líneas sin relleno medidas sobre el texto visible, pies numerados encima, referencias cruzadas en dos pasadas e índices de figuras y tablas con `PAGEREF`), mientras `TestInformesSinAvisos` impide que llegue al entregable cualquier referencia o etiqueta rota.
