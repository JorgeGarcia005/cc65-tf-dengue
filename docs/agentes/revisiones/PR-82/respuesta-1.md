# PR-82 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/66-mapa`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `6613351` — `normalizarDepto` pasa `?depto=` a mayúsculas y lo acepta solo si existe en el GeoJSON o es `OTROS`; si no, `null` (no se abre el panel). Con `depto=OTROS` el mapa resalta los departamentos agrupados y el panel los nombra; un departamento agrupado nombra a los demás del grupo. Tests: `?depto=piura` → panel de PIURA con sus cifras y sin nota de agrupado, `?depto=NOEXISTE` → sin panel, clic en «Otros (agrupados)» de la tabla → resalta los agrupados y no PIURA. |
| H2 | BLOQUEANTE | ACEPTO | `6613351` — `tramos(curva, f)` parte la curva en semanas consecutivas con datos suficientes; `Serie` dibuja un subcamino `M…L…` por tramo y un `<circle>` por semana aislada, sin unir huecos. El máximo del eje Y sale solo de lo que se dibuja: una semana con 1 caso al 100 % ya no aplasta la curva. Si ninguna semana alcanza `min_casos`, se dice explícitamente en lugar de mostrar un recuadro vacío. Tests: `tramos` con insuficiente al 100 % y huecos → 3 tramos y máximo 0,2; TUMBES (semana aislada) → 1 punto; LORETO → 1 subcamino. Verificado en Chrome con el JSON real (Madre de Dios, niños, semana 20): la línea se corta entre las semanas 8 y 45. |
| H3 | MAYOR | ACEPTO | `28a3c0a` — `Vary` con `Add` (test con un middleware que pone `Vary: Origin`: se conservan los dos); `Cache-Control: private, no-cache`; ETag distinto por representación (`"…"` y `"…-gzip"`; el de gzip no valida la plana); `Accept-Encoding` y `If-None-Match` combinando todas sus líneas, con `q` leído por `strconv.ParseFloat` (`gzip;q=0.0`, `q=0.000`, `Q=0` → sin gzip; `*` y `x-gzip` considerados); `cargarEvaluacion` exige `version_modelo` texto no vacío, `riesgo` objeto y `prioridades` arreglo (rechaza `null`); `nuevoDocumento` clona el slice. |
| H4 | MAYOR | ACEPTO | `6613351` — `escalaPara` ya no inventa un dominio `[0, 1]`: sin departamentos suficientes devuelve `vacia` y la leyenda lo dice; la leyenda muestra, para cada color **usado**, el mínimo y el máximo **reales** (no hay clases vacías ni «X – X»). Tests: escala vacía, un solo departamento suficiente → una clase con su valor, y que leyenda y panel cambien con semana y medida; reproducción con temporizadores falsos (avanza, se pausa, se detiene en 52 y desde 52 reinicia en 1). |
| H5 | MENOR | ACEPTO | `6613351` + `28a3c0a` — los caminos SVG se proyectan una sola vez por módulo; un único `setInterval` mientras dura la reproducción (lee semana y función de cambio por `useRef`); `.depto-sel:focus-visible` con trazo propio; sin línea en blanco final; `IndiceRiesgo.semanas()` eliminado; LEEME apunta a `mapa.test.tsx`; el comentario de paquete de `api` lista las dos rutas; un `-evaluacion` que existe pero está vacío detiene el arranque (`TestLeerEvaluacion`). Además, «Madre de Dios» con la preposición en minúscula. |

## Verificación

```text
$ make check                                   en verde
$ cd web && npx tsc -b && npx vitest run       Tests 46 passed (46)
$ npm run build                                ✓ (sin data: en el CSS)
```
