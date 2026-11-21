# Veredicto — PR-51 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/50-figuras-cluster` @ `1a8a98a`
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR responde a la observación del profesor del 21/11 incorporando al informe de la PC4 las tres figuras SVG solicitadas (`topologia.svg` en 8.1, `flujo.svg` y `nodo.svg` en 8.4), la explicación de cómo interactúan los nodos con el coordinador y la remisión desde la sección 4. Toda la verificación automática (`gofmt`, `go vet`, `go test -race ./...`, idempotencia de `cmd/informe`, `cmd/word` con los 12 SVG embebidos en OOXML válido y 0 avisos, y los 6 jobs del CI en GitHub) pasa en verde, y el mapeo con `deploy/docker-compose*.yml` e `internal/cluster/nodo.go` es fiel. Sin embargo, la auditoría visual renderizada y el cruce paso a paso con `internal/cluster/coordinador.go` y `cmd/coordinator/{main,api}.go` detectan **2 hallazgos mayores** y **1 menor** que conviene corregir antes de generar el `.docx` final: colisiones visibles de texto sobre flechas y bordes en `nodo.svg`, `flujo.svg` y `topologia.svg` (**H1**), la omisión de la rama de aborto del `cubrir()` inicial y la mezcla entre modo único y modo servicio en `Fin` dentro de `flujo.svg` (**H2**), y un subíndice Unicode ausente en Arial en `nodo.svg` (**H3**).

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/develop...origin/feature/50-figuras-cluster
(vacío = ok)

$ go test -race -count=1 -p 4 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	6.855s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	7.100s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	5.928s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.864s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	2.016s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	2.100s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	3.891s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	2.769s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	2.480s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	2.136s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	8.026s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	7.951s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	10.606s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.673s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.549s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.612s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.642s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.152s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	2.000s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	11.767s

$ go test -race -count=1 -v ./cmd/informe ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	2.473s (5/5 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	2.017s (13/13 PASS)

$ go run ./cmd/informe && git diff --exit-code docs/informe
DIFF_EXIT=0 (sin cambios en las regiones generadas)

$ go test -race -count=1 -overlay <audit_overlay.json> -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 12 SVGs, 12 PNGs, 0 avisos

$ gh pr checks 51   # run 44949853947 sobre 1a8a98a
Docker (imagen, clúster y caída de un nodo)	pass	1m50s
Ecosistema de agentes                       	pass	10s
Go (gofmt, vet, test -race)                 	pass	56s
MongoDB y Redis (tests de integración)      	pass	45s
OpenAPI 3 válido                            	pass	22s
Spin (modelo del protocolo y mutantes)      	pass	2m1s
```

> Nota del entorno local (Windows): al lanzar `go test -race -count=1 ./...` con paralelismo por defecto (todos los paquetes a la vez), `TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano` (cuyo plazo es 200 ms para 3 shards de 60 ms) falló por contención de CPU; al correr la suite completa con `-p 4` y `./internal/cluster` en solitario pasó sin errores ni carreras, igual que el CI de GitHub (el PR no toca código Go).

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; suite local `-p 4` + CI verde en run `44949853947`) |
| Equivalencia secuencial = distribuido | no aplica (el PR solo agrega figuras y texto del informe; tests de equivalencia en verde dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no toca `internal/api`) |
| Idempotencia `cmd/informe` y Word OOXML | ✅ (`DIFF_EXIT=0`; 16 secciones, 12 SVGs válidos, 12 PNGs, 0 avisos) |

## Verificación contra los criterios del issue #50

| Criterio (#50) | Estado | Evidencia |
|---|---|---|
| 1. `fig/topologia.svg` en 8.1 con estrella, N nodos, red `cluster`, puertos/protocolos reales (`deploy/docker-compose*.yml`), volumen `ro` y quién inicia cada conexión | ✅ (ajuste visual menor en **H1**) | `08-distribuido.md:20` y `fig/topologia.svg:15-107` coinciden con `deploy/docker-compose.yml:24,38,40,49,52,54-55,63,66,69,75-101` y `deploy/docker-compose.api.yml:10-18` (TCP `:9000` solo interna, HTTP `:8080` único puerto publicado en modo servicio, `mongo :27017`, `redis :6379`, volumen `../data/gold → /datos (ro)`). |
| 2. `fig/nodo.svg` en 8.4 (Algoritmo 2) fiel a `internal/cluster/nodo.go` | ⚠️ colisión visual (**H1**) y glifo Unicode (**H3**) | El contenido técnico coincide con `cmd/node/main.go:38,56-86`, `internal/cluster/nodo.go:37-132` e `internal/ml/concurrente.go:82-135`, pero la flecha vertical de retorno atraviesa la etiqueta `listo · gradiente · error` (**H1**) y `pₖ₋₁` usa un carácter fuera de Arial (**H3**). |
| 3. `fig/flujo.svg` en 8.4 que coincide paso a paso con el Algoritmo 1 | ⚠️ colisiones visuales (**H1**) y omisión/ambigüedad (**H2**) | Cubre el bucle `ml.EntrenarFalible` y la época (`GradienteErr`), pero omite la salida de aborto del `cubrir()` inicial, mezcla los modos único y API en `Fin` (**H2**), y presenta dos solapamientos de texto con los bordes de `cubrir() · 2.ª pasada` (**H1**). |
| 4. Texto que explica cómo interactúan los nodos con el coordinador; ningún nodo habla con otro | ✅ | `08-distribuido.md:22-30` («Cómo interactúan los nodos»), `08-distribuido.md:107-111` (interior del nodo) y remisión en `04-arquitectura.md:10-11`. |
| 5. `go run ./cmd/informe` sin cambios, `go test ./cmd/informe ./cmd/word` verde y Word con las tres figuras | ✅ | `DIFF_EXIT=0`, `cmd/informe` (5/5) y `cmd/word` (13/13) en verde, `TestAuditPC4WordXML` confirma 12 SVGs embebidos (9 previos + 3 nuevos) con 0 avisos y XML válido. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | La sección 4 remite con claridad a las vistas detalladas de red, flujo e interior del nodo en la sección 8 (`04-arquitectura.md:10-11`). |
| Implementación | 8 | sin cambios en código | No modifica binarios ni paquetes Go; documenta fielmente el clúster desplegado en PC4. |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | Parcial (pendiente **H1–H2**) | Cierra directamente la observación del profesor del 21/11 y se integra limpio en `cmd/word`, pero requiere ajustar las colisiones visuales en las figuras (**H1**) y alinear por completo `flujo.svg` con el código real (**H2**) para quedar defendible sin objeciones en el `.docx` final. |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

### H1 — Colisiones visibles de texto sobre flechas y bordes en los SVG renderizados · MAYOR

- **Dónde:** `docs/informe/pc4/fig/nodo.svg:114, 119`, `docs/informe/pc4/fig/flujo.svg:75-78, 108` y `docs/informe/pc4/fig/topologia.svg:42, 47`
- **Evidencia:**
  Al renderizar los tres SVG en Chromium/Arial (la misma fuente declarada en `font-family="Arial, Helvetica, sans-serif"`) y verificar las coordenadas exactas del XML:
  1. **`nodo.svg:114, 119` (flecha atraviesa el texto):** la polilínea de retorno `points="168,463 70,463 70,149"` (`stroke-width="1.6"`) sube por **`x = 70`**. La etiqueta `<text x="78" y="300" transform="rotate(-90 78 300)" text-anchor="middle">listo · gradiente · error</text>` (`font-size="11"`) coloca la línea base en `x = 78`, pero al rotar `-90°` la altura ascendente de los caracteres (~8,5 px) se proyecta hacia la **izquierda** (`x ≈ 69,5..78`), por lo que el trazo negro de la flecha corta directamente las letras `listo · gradiente · error` (compárese con `flujo.svg:97, 109`, donde la línea va en `x = 35` y el texto rotado `-90°` va a la izquierda en `x = 27`, sin tocarla).
  2. **`flujo.svg:75, 108` (etiqueta choca con el borde del recuadro):** `<text x="372" y="572" text-anchor="middle">sin nodo sano</text>` (`font-size="10.5"`, ancho ~69 px → borde derecho en `x ≈ 406,5`) invade el borde vertical izquierdo en **`x = 405`** de `<rect x="405" y="540" width="395" height="50" .../>`, haciendo que la «o» final de «sano» pise la línea azul del recuadro.
  3. **`flujo.svg:75, 78` (texto pegado al borde inferior del recuadro):** en ese mismo rectángulo (`y = 540`, `height = 50` → borde inferior en **`y = 590`**), la tercera línea `(empate: menor índice) · envía asignacion si hace falta, y epoca` tiene línea base en **`y = 587`**, dejando solo 3 px hasta el trazo; los descendentes (`p`, `,`, `y`) tocan la línea azul inferior. (Además, en `flujo.svg:89-91`, el tronco vertical desde `Esperar` baja en `x = 600` y el ramal central cae en `x = 602`, dejando un quiebre visible de 2 px.)
  4. **`topologia.svg:42, 47` (etiqueta roza la flecha):** en el enlace superior `node-1 → coordinator` (`(180,98)→(298,172)`), `<text x="196" y="118" transform="rotate(32 196 118)">TCP</text>` queda debajo de la línea con la parte superior de las letras tocando el trazo, a diferencia de las otras dos etiquetas `TCP` (`topologia.svg:48-49`), que van limpiamente por encima de sus flechas.
- **Corrección sugerida:**
  - En `nodo.svg:119`: desplazar `listo · gradiente · error` a `x="62"` (`transform="rotate(-90 62 300)"`, a la izquierda de la flecha) o a `x="84"` (`transform="rotate(-90 84 300)"`, a la derecha con margen limpio).
  - En `flujo.svg:75-78`: aumentar la altura del rectángulo `cubrir() · 2.ª pasada` de `50` a `56` (sobra espacio hasta `height="640"`), y mover `sin nodo sano` hacia la izquierda (p. ej. `x="358"` con el codo en `x="332"`, o en dos líneas `sin nodo` / `sano`) para separarlo del borde `x="405"`. Opcionalmente alinear `flujo.svg:90` a `x="600"` (o el tronco en `602`).
  - En `topologia.svg:47`: ubicar `TCP` por encima de la flecha superior (p. ej. `x="208" y="108" transform="rotate(32 208 108)"`), consistente con las líneas 48–49.

### H2 — `flujo.svg` omite la ruta de aborto del `cubrir()` inicial y mezcla modo único con modo servicio en `Fin` · MAYOR

- **Dónde:** `docs/informe/pc4/fig/flujo.svg:22-24, 40-42, 86` (contrastado con `docs/informe/pc4/08-distribuido.md:79-82, 168-170`, `internal/cluster/coordinador.go:289-291`, `cmd/coordinator/main.go:275-276` y `cmd/coordinator/api.go:24-27, 95, 116-138`)
- **Evidencia:**
  El criterio 3 del issue #50 exige que `flujo.svg` coincida paso a paso con el Algoritmo 1:
  1. **Aborto en la difusión inicial:** el Algoritmo 1 paso 2 (`08-distribuido.md:79-82`) establece: *«Si un envío falla, descartar al nodo y probar el siguiente; si no queda ninguno, abortar»*, y el código copiado justo debajo de `flujo.svg` (`internal/cluster/coordinador.go:289-291`, `08-distribuido.md:168-170`) ejecuta antes del bucle `for pendientes > 0`:
     ```go
     if err := c.cubrir(); err != nil {
         return ml.Gradiente{}, c.abortar(err)
     }
     ```
     Sin embargo, en `flujo.svg:22-24, 86`, el bloque `Difundir la época e (todos los shards pendientes · cubrir())` tiene una única salida hacia `¿quedan shards pendientes?` y carece de transición `sin nodo sano → Abortar` (esa flecha solo sale del `cubrir() · 2.ª pasada` al final de la columna derecha, línea 98). Si todos los nodos cayeron entre épocas o fallan los envíos iniciales, el diagrama no refleja cómo se alcanza `Abortar`.
  2. **Confusión entre modo único y modo servicio en `Fin` (`flujo.svg:40-42`):** el nodo terminal `Fin` dice `fin(completo) a los nodos · modelo y reports/*.json`. En el código real:
     - En **modo entrenamiento único** (`cmd/coordinator/main.go:275-276`), el coordinador guarda `reports/entrenamiento_cluster.json` (y verifica si `-verificar`) y luego llama a `c.Terminar()` (`fin(completo)`).
     - En **modo servicio / API** (`cmd/coordinator/api.go:24-27, 95, 116-138`), terminar `ml.EntrenarFalible` en un `POST /api/v1/entrenamientos` publica el modelo en MongoDB y en el `atomic.Pointer` **sin enviar `fin(completo)`**, pues los nodos siguen conectados para entrenamientos futuros y solo reciben `fin` al llegar `SIGTERM`.
     Juntar ambas cosas en `Fin` puede convertirse en una trampa en la sustentación si el profesor pregunta si un entrenamiento lanzado por la API apaga los nodos al terminar.
- **Corrección sugerida:**
  1. Añadir en `flujo.svg` la salida `sin nodo sano` desde `Difundir la época e` hacia `Abortar` (por ejemplo, una flecha que baje por el canal central entre ambas columnas y empalme con la llegada a `Abortar`, o un rombo/anotación equivalente de error de `cubrir()`), de modo que coincida exactamente con los dos `if err := c.cubrir()` de `GradienteErr`.
  2. Precisar el texto de `Fin` en `flujo.svg:42` para distinguir ambos modos sin recargarlo, por ejemplo: `modo único: reports/*.json y fin(completo) · API: publica modelo` (o `devuelve w y pérdida · modo único: reporte y fin(completo); API: publica modelo`).

### H3 — Subíndice Unicode `pₖ₋₁` en `nodo.svg` fuera de la cobertura de Arial e inconsistente con `K−1` · MENOR

- **Dónde:** `docs/informe/pc4/fig/nodo.svg:67, 80` (comparar con `docs/informe/pc4/fig/worker_pool.svg:51`)
- **Evidencia:**
  La línea 67 de `nodo.svg` nombra los bloques con mayúscula (`bloques 0 … K−1`), pero la línea 80 escribe `pₖ₋₁` usando los caracteres Unicode `U+2096` (`ₖ`, *latin subscript small letter k*) y `U+208B` (`₋`). La fuente `Arial` declarada en `nodo.svg:1` no incluye `U+2096` en Windows: en el renderizado en Chromium ya se aprecia sustitución de fuente y cambio de caja (`k` minúscula frente a `K`), y al importar el SVG en Word o LibreOffice los glifos ausentes en Arial suelen renderizarse como recuadros vacíos (`□`). Por ese motivo `docs/informe/pc4/fig/worker_pool.svg:51` ya había evitado `U+2096`.
- **Corrección sugerida:**
  Sustituir `pₖ₋₁` en `nodo.svg:80` por `<tspan>` estándar de SVG usando `K−1` en mayúscula (p. ej. `p<tspan dy="2.5" font-size="7.5">K−1</tspan>`, y de forma opcional el mismo `<tspan>` para `p₀`, `p₁`, `p₂`), garantizando consistencia con la línea 67 y cobertura completa en Arial en Word y LibreOffice.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-51. |

## Cómo explicarlo en la sustentación

La sección 4 da el mapa general del sistema y remite a la sección 8 para el detalle del clúster en tres vistas complementarias: la topología de red en estrella sobre la red interna `cluster` de Docker (donde cada nodo abre una única conexión TCP persistente a `coordinator:9000`, los nodos nunca se hablan entre sí y solo se publica el puerto HTTP `:8080` en modo servicio), el diagrama de flujo del entrenamiento que refleja `ml.EntrenarFalible` y `GradienteErr` (difusión, espera de eventos o plazos por shard, reasignación en dos pasadas y barrera con reducción en orden), y el interior de un nodo (`(*Nodo).Servir`: una sola goroutine de red y un único `ml.Pool` compartido por todos sus shards que devuelve un parcial por bloque sin reducir). Una vez corregidos los solapamientos visuales en los SVG y las dos precisiones de `flujo.svg`, las tres figuras se integran automáticamente en el Word con `cmd/word` (12 SVG, 0 avisos).
