# Veredicto — PR-82 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/66-mapa` @ `94e5258` (base: `integracion/tb2` @ `0f9c0f5`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR implementa la vista principal del mapa coroplético de riesgo por departamento, semana epidemiológica y grupo de edad (`web/src/mapa/`, `web/src/paginas/Mapa.tsx`), incorpora el GeoJSON departamental sin modificar con licencia MPL-2.0 y sha256 verificado, añade `GET /api/v1/riesgo` y `GET /api/v1/evaluacion` con precompresión gzip y ETag en `internal/api/evaluacion.go` usando solo stdlib (`AGENTS.md` §3.2), actualiza OpenAPI/Postman/Compose y corrige la incrustación `data:` de fuentes en Vite bajo la CSP estricta de `cmd/web` (#79). Toda la verificación automática local y los 7 jobs del CI en GitHub pasan en verde, sin carreras de datos ni cifras escritas a mano en `web/src/`. Sin embargo, al auditar la lógica contra el `reports/evaluacion.json` real y el contrato HTTP de `ADR-0007`, se emiten **2 hallazgos bloqueantes**, **2 mayores** y **1 menor**: el parámetro `?depto=` de la URL no se normaliza ni valida, haciendo que `/mapa?depto=piura` muestre las cifras de `OTROS` (8 casos, 0,0 % observado) bajo el título «Piura» en lugar de los 41 319 casos reales de Piura (**H1**); la curva semanal de `PanelDepto` calcula el máximo Y incluyendo celdas `insuficiente` que no grafica (aplastando hasta 10× la curva en 32/126 combinaciones reales como Callao y Lima), interpola con una recta continua sobre huecos de hasta 36 semanas sin datos suficientes (51/126 curvas) y deja 41/126 gráficas completamente en blanco sin aviso (**H2**); `servirDocumento` sobrescribe `Vary: Origin` (rompiendo CORS de `ADR-0007`), emite el mismo ETag fuerte para `plano` y `gzip`, acepta `gzip;q=0.0` como válido y admite `null` en campos requeridos (**H3**); la escala de quintiles fabrica cortes artificiales `[0, 1]` en las 23/318 combinaciones sin departamentos suficientes (y cortes repetidos en 24), y faltan tests para la reproducción de semanas y la actualización de leyenda y panel exigidos por #66 (**H4**); más ajustes menores de rendimiento SVG, foco de teclado e higiene (**H5**). (La autorización por rol `analista`/`admin` en `Rutas()` queda correctamente diferida a `#60`/`#61`, que aún no se fusionan en `integracion/tb2`.)

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
web/src/mapa/MapaPeru.tsx:102: new blank line at EOF.

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.Imports}}' ./internal/api
[bytes compress/gzip context crypto/rand crypto/sha256 encoding/hex encoding/json errors fmt github.com/JorgeGarcia005/cc65-tf-dengue/docs/api github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster github.com/JorgeGarcia005/cc65-tf-dengue/internal/features github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml io log math net net/http runtime/debug strings sync sync/atomic time]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	3.531s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	8.842s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.512s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.628s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.269s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.206s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.421s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.056s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.136s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.024s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.057s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	9.168s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.133s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.304s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.956s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.042s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.008s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.925s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.040s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.691s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.768s

$ go test -race -count=1 -run 'Evaluacion|AceptaGzip|OpenAPI' -v ./internal/api ./cmd/coordinator
PASS (7/7 tests: OpenAPI, reporte real 1929.5 KB → 324.2 KB gzip / riesgo 1907.4 KB → 316.5 KB gzip, 304, 503, inválidos, LeerEvaluacion)

$ (en web/) npm ci && npx tsc -b && npm test && npm run build && ! grep -l 'url(data:' dist/assets/*.css
Test Files  3 passed (3)
     Tests  37 passed (37)
✓ built in 1.75s
SPA OK

$ sha256sum web/src/datos/peru_departamental_simple.geojson
3d34d5450a1b1f059eebdf7386e5b5d071a5063bf3679fcbbbc5314493791290  web/src/datos/peru_departamental_simple.geojson

$ grep -rnE '[0-9]+[.,][0-9]{2,}' web/src --include='*.ts*' | grep -v pruebas
(vacío = ok)

$ gh pr checks 82
7/7 pass (Ecosistema de agentes, Go, MongoDB y Redis, OpenAPI 3 válido, SPA, Spin, Docker)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`) |
| Latencia API p95 < 100 ms | no aplica (no modifica `POST /api/v1/predicciones`; `cmd/benchapi` tests ✅; `/api/v1/riesgo` y `/api/v1/evaluacion` sirven buffers precomprimidos en memoria) |
| SPA (`tsc -b`, Vitest, `vite build`, sin `url(data:)`) | ✅ (37/37 tests, 0 vulnerabilidades, build limpio) |
| CI GitHub (PR #82) | ✅ (7/7 jobs en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 2.5 / 3 | Responde directamente la pregunta de impacto ODS 3 con datos de `reports/evaluacion.json`, pero H1 y H2 distorsionan/falsean resultados en casos reales. |
| Implementación | 8 | 7.5 / 8 | Backend 100 % stdlib, precompresión gzip, ETag y sin carreras; H3 debe corregirse para no romper `Vary: Origin` de CORS (`ADR-0007`) ni negociación HTTP. |
| Interfaz | 5 | 3.5 / 5 | Coroplético accesible, teclado y tabla alternativa muy bien encaminados, pero H1 (URL `?depto=`), H2 (curva 10× aplastada, huecos puenteados y 41 gráficas vacías) y H4 (leyenda `[0, 1]` y tests de reproducción) arriesgan la demostración en vivo. |
| Informe | 2 | — | No corresponde a este PR. |
| Video | 2 | — | No corresponde a este PR. |

## Hallazgos

### H1 — Parámetro `?depto=` sin validar ni normalizar: muestra cifras de `OTROS` bajo el nombre de un departamento real o inexistente · BLOQUEANTE

- **Dónde:** `web/src/paginas/Mapa.tsx:13-23, 179-181, 212`, `web/src/mapa/PanelDepto.tsx:19, 30-37`, `web/src/mapa/datos.ts:20-23` y `web/src/mapa/MapaPeru.tsx:61, 66`
- **Evidencia:**
  1. `filtros()` (`Mapa.tsx:13-23`) valida `semana`, `etapa` y `metrica`, pero devuelve `depto: p.get('depto')` crudo, sin normalizar ni comprobar que sea un departamento válido.
  2. Al abrir `/mapa?depto=piura` (minúsculas), `PanelDepto` llama a `indice.grupo("piura")` (`datos.ts:21-23`), que solo busca coincidencias exactas en `this.modelo` (mayúsculas) y cae silenciosamente en `OTROS`. El panel muestra el encabezado **«Piura»**, el aviso falso *«Con pocos casos en el entrenamiento, el modelo lo agrupa con otros departamentos»* y las cifras de `OTROS` del `reports/evaluacion.json` real (`semana=0, etapa=todas`: **8 casos**, tasa observada **0,0 %**, riesgo calibrado **10,9 %**, **0,9** casos esperados, `insuficiente: true`), en lugar de las cifras reales de `PIURA` (**41 319 casos**, tasa observada **13,4 %**, riesgo calibrado **13,3 %**, **5 510,4** casos esperados, `insuficiente: false`).
  3. De igual manera, cualquier valor arbitrario (`/mapa?depto=Marte`) se presenta como un departamento válido con el título «Marte» y las cifras/curva de `OTROS`.
  4. Al hacer clic en «Otros (agrupados)» en la tabla (`Mapa.tsx:212`, que escribe `?depto=OTROS`), `MapaPeru.tsx:61,66` compara `x.nombre === seleccionado` y no resalta ninguno de los 5 departamentos agrupados (`APURIMAC`, `AREQUIPA`, `HUANCAVELICA`, `MOQUEGUA`, `TACNA`), mientras `PanelDepto.tsx:33-37` despliega la nota en singular (*«lo agrupa con otros departamentos»*) bajo «Otros (agrupados)».
- **Corrección sugerida:**
  - En `filtros()` (`Mapa.tsx:13-23`), normalizar `p.get('depto')` con `.trim().toUpperCase()` y aceptarlo únicamente si existe en el conjunto de `NOMBDEP` de `DEPARTAMENTOS` o es `OTROS`; en caso contrario devolver `null`.
  - Cuando `seleccionado === OTROS`, resaltar en `MapaPeru.tsx` los `<path>` con `x.grupo === OTROS`, y ajustar la nota de `PanelDepto.tsx:33-37` para distinguir si `depto === OTROS` (mencionando los 5 departamentos agrupados) frente a un departamento individual agrupado.
  - Añadir pruebas en `web/src/pruebas/mapa.test.tsx` para `/mapa?depto=piura` (muestra los datos reales de `PIURA`), `/mapa?depto=NOEXISTE` (no abre panel con datos de `OTROS`) y clic en `OTROS` desde la tabla.

### H2 — Curva semanal en `PanelDepto`: escala Y distorsionada hasta 10× por celdas excluidas, puenteo de huecos de hasta 36 semanas y 41/126 gráficas vacías sin aviso · BLOQUEANTE

- **Dónde:** `web/src/mapa/PanelDepto.tsx:21-25, 67-77`
- **Evidencia:** (auditado con Node directamente sobre las 126 combinaciones `departamento × etapa` del `reports/evaluacion.json` real)
  1. **Escala Y inflada hasta 10× por celdas que no se dibujan (`PanelDepto.tsx:22` vs `:25`):** `maximo` incluye todas las celdas `c` (también las `insuficiente`), pero `linea` filtra `c && !c.insuficiente`. En `reports/evaluacion.json`, **32 de 126** combinaciones tienen `maxAll > 2 × maxSuf`:
     - `CALLAO / adulto_30_59`: las 12 semanas suficientes tienen máximo `0.100` (10,0 %), pero la semana 36 (`casos: 1`, `insuficiente: true`, `tasa_observada: 1.0`) sube `maximo` a `1.0` (**10×**), aplastando toda la curva trazada en el 10 % inferior del SVG.
     - `ICA / adolescente_12_17`: `maxSuf = 0.138` vs `maxAll = 1.0` (**7,25×**); `LIMA / nino_0_11`: `0.177` vs `1.0` (**5,65×**); `LIMA / adulto_mayor_60_mas`: `0.195` vs `1.0` (**5,12×**).
  2. **Puenteo lineal sobre semanas insuficientes o vacías (`PanelDepto.tsx:24-25, 70-71`):** al aplanar todos los puntos `!c.insuficiente` en un único `<polyline>`, SVG conecta con una recta continua semanas separadas por meses sin datos suficientes. En `reports/evaluacion.json`, **51 de 126** curvas tienen huecos internos, llegando a **36 semanas consecutivas** en `MADRE DE DIOS / nino_0_11` (une directamente la semana 8 con la semana 45), contradiciendo el criterio de `#66` (*«celdas `insuficiente` señaladas»*) y la regla de no inventar certeza.
  3. **41 de 126 curvas (32,5 %) completamente en blanco sin explicación (`PanelDepto.tsx:25, 67-77`):** **39** combinaciones tienen 0 semanas con ≥ `min_casos` (p. ej. `AMAZONAS` en 4 de 5 etapas de edad) y **2** (`CAJAMARCA / adulto_mayor_60_mas`, semana 16; `JUNIN / adolescente_12_17`, semana 46) tienen exactamente 1 semana suficiente, que un `<polyline>` de un punto no renderiza en SVG. En todos esos casos el recuadro de la curva queda vacío sin aviso.
- **Corrección sugerida:**
  - Calcular `maximo` exclusivamente sobre las celdas `!c.insuficiente`.
  - Dividir la curva en tramos contiguos de semanas `!c.insuficiente` (p. ej. usando `<path>` con subcaminos `M`/`L` por cada tramo consecutivo y `<circle>` para semanas suficientes aisladas), sin trazar líneas rectas sobre semanas insuficientes/ausentes (opcionalmente marcando los huecos insuficientes de forma discreta).
  - Si ninguna semana alcanza `min_casos`, mostrar un mensaje explícito en lugar de un `<svg>` vacío (p. ej. *«Ninguna semana individual alcanza {min_casos} casos con este filtro»*).
  - Añadir pruebas en `web/src/pruebas/mapa.test.tsx` que cubran: (a) una celda `insuficiente` con tasa `1.0` no aplasta la escala Y; (b) un hueco `insuficiente` intermedio divide la curva en dos subcaminos; (c) una curva con 0 o 1 semana suficiente muestra aviso o punto visible.

### H3 — `servirDocumento` sobrescribe `Vary: Origin` (rompe CORS de `ADR-0007`), comparte ETag fuerte entre `plano` y `gzip`, acepta `gzip;q=0.0` y permite `null` en campos obligatorios · MAYOR

- **Dónde:** `internal/api/evaluacion.go:39, 55-59, 80-93, 101-118`
- **Evidencia:**
  1. **Sobrescritura de `Vary` y `Cache-Control` sin `private` (`evaluacion.go:82-83`):** `cab.Set("Vary", "Accept-Encoding")` borra cualquier valor previo de `Vary`. Según `ADR-0007` (líneas 32–33), el middleware CORS corre fuera del `ServeMux` y añade `Vary: Origin` a toda respuesta de un origen permitido; al ejecutarse el handler, `cab.Set` elimina `Origin` y rompe el contrato de caché CORS. Además, `cab.Set("Cache-Control", "no-cache")` carece de `private` (RFC 9111 §5.2.2.7), autorizando a cachés compartidas a almacenar respuestas de rutas autenticadas (`analista`, `admin`).
  2. **Mismo ETag fuerte para `identity` y `gzip` (`evaluacion.go:39, 81, 90-93`):** `d.etag` es un ETag fuerte (`"<hex>"`) calculado sobre `d.plano`, pero se envía idéntico cuando la respuesta va con `Content-Encoding: gzip`. Por RFC 9110 §8.8.3.3, representaciones con distinta codificación de contenido no son idénticas byte a byte y no deben compartir un ETag fuerte.
  3. **Parser de `Accept-Encoding` y cabeceras repetidas (`evaluacion.go:84, 90, 110-118`):** `strings.ReplaceAll(strings.TrimSpace(q), " ", "") != "q=0"` devuelve `true` para `gzip;q=0.0`, `gzip;q=0.000` o `gzip;Q=0`, enviando `gzip` aunque el cliente lo haya rechazado explícitamente (RFC 9110 §12.4.2). Además, `r.Header.Get` solo inspecciona la primera cabecera si el cliente envía múltiples líneas `Accept-Encoding` o `If-None-Match`.
  4. **Campos requeridos en `null` (`evaluacion.go:55-59`) y slice sin clonar (`:39`):** `len(campos[k]) == 0` acepta `{"version_esquema":1,"version_modelo":null,"riesgo":null,"prioridades":null}` (`len("null") == 4`), lo que arranca la API con un documento que rompe el SPA al construir `IndiceRiesgo`. Además `nuevoDocumento` guarda `plano: b` sin clonar el slice recibido en `Opciones.Evaluacion`.
- **Corrección sugerida:**
  - Usar `cab.Add("Vary", "Accept-Encoding")` en lugar de `cab.Set` y `Cache-Control: private, no-cache` (documentando la convivencia con el `no-store` general de `#61` para permitir 304). Cubrir en `evaluacion_test.go` que un middleware previo con `Vary: Origin` conserva ambos valores.
  - Emitir ETag débil (`W/"<hex>"`) o ETags diferenciados por codificación (`etag` / `etagGz`) según la representación negociada.
  - Parsear `q=` de forma insensible a mayúsculas con `strconv.ParseFloat` (rechazando `q <= 0`) y combinar cabeceras repetidas con `r.Header.Values(...)`; ampliar `TestAceptaGzipYETag` con `gzip;q=0.0`, `gzip;q=0.000` y `gzip;Q=0`.
  - En `cargarEvaluacion`, validar los tipos JSON básicos (`version_modelo` string no vacío, `riesgo` objeto, `prioridades` arreglo — rechazando `null`) y usar `slices.Clone(b)` en `nuevoDocumento`.

### H4 — Leyenda con quintiles fabricados `[0, 1]` o intervalos degenerados y falta de pruebas de reproducción, leyenda y actualización del panel · MAYOR

- **Dónde:** `web/src/mapa/datos.ts:58-66`, `web/src/mapa/MapaPeru.tsx:85-100` y `web/src/pruebas/mapa.test.tsx:103-146`
- **Evidencia:**
  1. **Escala de respaldo `[0, 1]` y cortes repetidos en datos reales (`datos.ts:64`, `MapaPeru.tsx:85-94`):** en el `reports/evaluacion.json` real (318 combinaciones `semana × etapa`):
     - En **23** combinaciones (p. ej. `semana=32..36, etapa=adulto_mayor_60_mas`), 0 departamentos del modelo tienen datos suficientes; `escalaPara` usa `domain([0, 1])` y `Leyenda` imprime cortes inventados (`< 20,0 %`, `20,0 % – 40,0 %`, `40,0 % – 60,0 %`, `60,0 % – 80,0 %`, `≥ 80,0 %`, o `< 0,2`, `0,2 – 0,4`… en `casos esperados`) que no salen de `reports/evaluacion.json`. Además, si `OTROS` superara `min_casos` con 0 departamentos del modelo suficientes, se colorearía contra ese dominio ficticio `[0, 1]`.
     - En **12** combinaciones con un solo departamento suficiente (p. ej. `semana=1, etapa=adulto_mayor_60_mas`, riesgo `20,5 %`), `scaleQuantile` genera cuatro cortes idénticos (`20,5 %`), mostrando tres clases degeneradas `20,5 % – 20,5 %` (**24** combinaciones en total muestran cortes formateados repetidos).
  2. **Criterio del issue #66 sin cobertura en tests (`mapa.test.tsx`):** el criterio 2 de `#66` exige *«Filtros semana (1–53, con reproducción), grupo de edad y métrica (riesgo calibrado / casos esperados) actualizan mapa, leyenda y panel»*. Ningún test en `web/src/pruebas/mapa.test.tsx` prueba el botón **«Reproducir semanas» / «Pausar»**, ni verifica que la **leyenda** o el **panel del departamento** se actualicen al cambiar semana, grupo de edad o métrica.
- **Corrección sugerida:**
  - En `escalaPara` / `Leyenda`, devolver estado vacío cuando `valores.length === 0` y mostrar solo la entrada rayada («Sin departamentos con datos suficientes para este filtro») sin fabricar cortes `[0, 1]`; cuando los cortes formateados se repitan (menos de 5 valores distintos), colapsar clases redundantes en lugar de imprimir `X – X`.
  - Añadir en `web/src/pruebas/mapa.test.tsx` pruebas para: (a) reproducción semanal con `vi.useFakeTimers()` (avanza semanas, se pausa, se detiene en 52 y reinicia en 1 desde 52); (b) actualización de leyenda y cifras del panel al cambiar filtros; (c) filtro sin departamentos suficientes sin leyenda ficticia `[0, 1]`.

### H5 — Rendimiento de proyección SVG, foco de teclado en departamento seleccionado, código muerto e higiene · MENOR

- **Dónde:** `web/src/mapa/MapaPeru.tsx:25-37, 102`, `web/src/mapa/datos.ts:31-33`, `web/src/paginas/Mapa.tsx:63-74`, `web/src/estilos/vigia.css:407-415`, `web/src/datos/LEEME.md:9`, `internal/api/api.go:5-13` y `cmd/coordinator/main.go:316-322`
- **Evidencia:**
  1. `MapaPeru.tsx:25-37`: `camino(f)` recalcula la proyección GeoJSON → SVG path de los 25 polígonos (140 KB) en cada re-render, incluido cada movimiento del cursor o foco (`setSobre`), en vez de memoizar `{ nombre, d }` una sola vez.
  2. `MapaPeru.tsx:102`: línea en blanco extra al final del archivo (`git diff --check` falla con `new blank line at EOF`).
  3. `datos.ts:31-33`: `IndiceRiesgo.semanas()` no se invoca en ningún archivo (código muerto).
  4. `Mapa.tsx:63-74`: el `useEffect` de reproducción incluye `params` en sus dependencias, destruyendo y recreando el `setInterval` cada 700 ms (y reiniciando el contador al seleccionar un departamento durante la reproducción); la actualización funcional `setParams((prev) => ...)` evita recrear el temporizador.
  5. `vigia.css:407-415`: `.depto:focus-visible` quita `outline` y usa `stroke-width: 2.4`, pero `.depto-sel` (posterior) fija `stroke-width: 3`, dejando sin indicador visual distinto de foco al departamento seleccionado al tabular.
  6. `web/src/datos/LEEME.md:9`: apunta a `src/pruebas/mapa.test.ts` cuando el archivo es `src/pruebas/mapa.test.tsx`.
  7. `internal/api/api.go:5-13`: el comentario de paquete omite `GET /api/v1/riesgo` y `GET /api/v1/evaluacion`.
  8. `cmd/coordinator/main.go:316-322` + `internal/api/api.go:130-134`: ante un archivo `-evaluacion` existente pero vacío (0 bytes), `leerEvaluacion` registra `"evaluación cargada … (0 bytes)"` y `api.Nuevo` lo trata en silencio como ausente (503) en vez de rechazarlo como inválido.
- **Corrección sugerida:** Aplicar estos ajustes junto con las correcciones de H1–H4.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-82. |

## Cómo explicarlo en la sustentación

La vista del mapa responde la pregunta de impacto social (ODS 3): a partir del cubo `departamento × semana × grupo de edad` calculado en `reports/evaluacion.json` y servido precomprimido con `gzip` y `ETag` (304) por la API en Go stdlib (`/api/v1/riesgo`), la SPA pinta los 25 departamentos del Perú con `d3-geo` en quintiles de riesgo calibrado o de casos esperados con signos de alarma —la medida que anticipa demanda de camas—, manteniendo los filtros compartibles en la URL y señalando con rayado las celdas con pocos casos para no inventar certeza. Antes de integrarlo, hay que corregir la validación del parámetro `?depto=` en la URL (para no mostrar datos de `OTROS` bajo el nombre de un departamento real en minúsculas), el escalado y corte de huecos en la curva semanal de `PanelDepto`, la preservación de `Vary: Origin` / negociación HTTP en `internal/api/evaluacion.go` y la degradación de la leyenda + pruebas de reproducción semanal.
