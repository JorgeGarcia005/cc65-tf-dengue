# Veredicto — PR-51 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/50-figuras-cluster` @ `e81aa43`
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Los tres hallazgos de la ronda 1 (**H1–H3**) fueron aceptados y resueltos de forma completa y verificable en `c5d5e54`, con respuesta documentada en `docs/agentes/revisiones/PR-51/respuesta-1.md` (`e81aa43`). La nueva auditoría visual sobre los tres SVG renderizados en Chromium/Arial a escala 2× confirma cero colisiones de texto contra flechas o bordes (**H1**), la alineación exacta de `flujo.svg` con los dos `if err := c.cubrir()` de `GradienteErr` y la distinción entre modo único y modo API en `Fin` (**H2**), y la eliminación de `U+2096`/`U+208B` en `nodo.svg` mediante `<tspan>` con `K−1` en mayúscula (**H3**). No aparecieron hallazgos nuevos: todas las verificaciones locales (`gofmt`, `go vet`, `go test -race -count=1 ./...`, idempotencia de `cmd/informe`, `cmd/word` con 16 secciones, 12 SVG válidos, 12 PNG y 0 avisos) y los 6 jobs del CI en GitHub (run `44952541126`) están en verde. El PR queda listo para merge con aprobación humana.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/develop...HEAD
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.395s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.499s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.474s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.281s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.126s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.419s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.156s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.026s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.081s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.947s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.237s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.284s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.043s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.077s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.335s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.546s

$ go run ./cmd/informe && git diff --exit-code docs/informe
DIFF_EXIT=0 (sin cambios en las regiones generadas)

$ go test -race -count=1 -v ./cmd/informe ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.134s (5/5 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.124s (13/13 PASS)

$ go test -race -count=1 -overlay <audit_overlay.json> -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 12 SVGs, 12 PNGs, 0 avisos

$ gh pr checks 51   # run 44952541126 sobre e81aa43
Docker (imagen, clúster y caída de un nodo)	pass	2m41s
Ecosistema de agentes                       	pass	8s
Go (gofmt, vet, test -race)                 	pass	1m0s
MongoDB y Redis (tests de integración)      	pass	44s
OpenAPI 3 válido                            	pass	17s
Spin (modelo del protocolo y mutantes)      	pass	2m7s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; suite completa local + CI verde en run `44952541126`) |
| Equivalencia secuencial = distribuido | no aplica (el PR solo agrega figuras y texto del informe; tests de equivalencia en verde dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no toca `internal/api`) |
| Idempotencia `cmd/informe` y Word OOXML | ✅ (`DIFF_EXIT=0`; 16 secciones, 12 SVGs XML válidos, 12 PNGs, 0 avisos, y pies de las 3 figuras presentes en `word/document.xml`) |

## Verificación contra los criterios del issue #50

| Criterio (#50) | Estado | Evidencia |
|---|---|---|
| 1. `fig/topologia.svg` en 8.1 con estrella, N nodos, red `cluster`, puertos/protocolos reales (`deploy/docker-compose*.yml`), volumen `ro` y quién inicia cada conexión | ✅ | `08-distribuido.md:20` y `fig/topologia.svg:15-107` coinciden con `deploy/docker-compose.yml:24,38,40,49,52,54-55,63,66,69,75-101` y `deploy/docker-compose.api.yml:10-18`; las tres etiquetas `TCP` (`:47-49`) quedan limpiamente por encima de sus flechas. |
| 2. `fig/nodo.svg` en 8.4 (Algoritmo 2) fiel a `internal/cluster/nodo.go` | ✅ | Fiel a `cmd/node/main.go:38,56-86` e `internal/cluster/nodo.go:37-132`; etiqueta de retorno separada de la flecha (`nodo.svg:119`) y subíndices `p0 … pK−1` construidos con `<tspan>` en Arial (`nodo.svg:79-80`), consistentes con `bloques 0 … K−1` (`:67`). |
| 3. `fig/flujo.svg` en 8.4 que coincide paso a paso con el Algoritmo 1 | ✅ | Cubre `ml.EntrenarFalible` y `GradienteErr` con las dos rutas de aborto de `cubrir()` (`flujo.svg:88,99-100,110` ↔ `internal/cluster/coordinador.go:289-291,308-310`), puente limpio en el cruce con `no`, distinción explícita entre modo único y modo API en `Fin` (`flujo.svg:40-43` ↔ `cmd/coordinator/main.go:275-276`, `cmd/coordinator/api.go:24-27,95,131-137`) y renderizado sin solapamientos. |
| 4. Texto que explica cómo interactúan los nodos con el coordinador; ningún nodo habla con otro | ✅ | `08-distribuido.md:22-30` («Cómo interactúan los nodos»), `08-distribuido.md:107-111` (interior del nodo) y remisión en `04-arquitectura.md:10-11`. |
| 5. `go run ./cmd/informe` sin cambios, `go test ./cmd/informe ./cmd/word` verde y Word con las tres figuras | ✅ | `DIFF_EXIT=0`, `cmd/informe` (5/5) y `cmd/word` (13/13) en verde, y `TestAuditPC4WordXML` confirma 16 secciones, 12 SVGs válidos (9 previos + 3 nuevos), 12 PNGs y 0 avisos. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | La sección 4 remite con claridad a las vistas detalladas de red, flujo e interior del nodo en la sección 8 (`04-arquitectura.md:10-11`). |
| Implementación | 8 | sin cambios en código | No modifica binarios ni paquetes Go; documenta con fidelidad el clúster desplegado en PC4. |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | 2 / 2 (en el alcance del issue #50) | Cierra la observación del profesor del 21/11 con tres diagramas limpios, fieles al código real e integrados sin avisos en el `.docx` generado por `cmd/word`. |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

Sin hallazgos abiertos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — Colisiones visibles de texto sobre flechas y bordes en los SVG renderizados · MAYOR | **RESUELTO** | En `c5d5e54`: `nodo.svg:119` movió `listo · gradiente · error` a `x="62"` (a la izquierda de la flecha en `x="70"`); `flujo.svg:76-79,87-96,110` amplió `cubrir() · 2.ª pasada` a `height="56"` (borde inferior en `y=596`, 10 px libres bajo `y=586`), trasladó la etiqueta de aborto al canal vertical en `x="326" y="520"` lejos de todos los bordes y alineó el eje de `¿quedan shards pendientes?` / `Esperar` / ramas en `x="602"`; `topologia.svg:47` subió `TCP` a `(208,108)` por encima de su flecha. Verificado visualmente en renderizado Chromium a escala 2× sin solapamientos ni recortes. |
| H2 (ronda 1) — `flujo.svg` omitía la ruta de aborto del `cubrir()` inicial y mezclaba modo único con modo servicio en `Fin` · MAYOR | **RESUELTO** | En `c5d5e54`: `flujo.svg:88,99-100,110` agregó la salida `(305,258) → (332,258) → (332,602) → (307,602)` desde `Difundir la época e` hacia `Abortar`, compartiendo el canal con `cubrir() · 2.ª pasada` bajo la etiqueta `sin nodo sano (cualquier cubrir)` y con un salto semicircular (`A6,6`) donde cruza la flecha `no`; y `flujo.svg:40-43` separa explícitamente ambos modos en `Fin` (`modo único: reports/*.json y fin(completo) a los nodos` / `modo API: publica el modelo; los nodos siguen conectados`). |
| H3 (ronda 1) — Subíndice Unicode `pₖ₋₁` en `nodo.svg` fuera de Arial e inconsistente con `K−1` · MENOR | **RESUELTO** | En `c5d5e54`: `nodo.svg:79-80` reemplazó `p₀`, `p₁`, `p₂` y `pₖ₋₁` por `<tspan dy="2.5" font-size="7.5">` con dígitos ASCII y `K−1` en mayúscula (`p<tspan…>0</tspan>` … `p<tspan…>K−1</tspan>`). Auditoría de glifos confirma 0 apariciones de `U+2096`/`U+208B` y renderizado uniforme en Arial. |

## Cómo explicarlo en la sustentación

La sección 4 muestra el mapa general del sistema y remite a la sección 8 para el detalle del clúster en tres vistas: la topología de red en estrella sobre la red interna `cluster` de Docker (`topologia.svg`: cada nodo inicia una única conexión TCP persistente a `coordinator:9000`, los nodos nunca se hablan entre sí y solo se publica HTTP `:8080` en modo servicio), el diagrama de flujo de `ml.EntrenarFalible` y `GradienteErr` (`flujo.svg`: difusión, espera de eventos o plazos por shard, reasignación en dos pasadas, aborto ante falta de nodos sanos en cualquier `cubrir()`, barrera con reducción en orden y diferenciación entre modo único y modo API al terminar), y el interior de un nodo (`nodo.svg`: una sola goroutine de red en `(*Nodo).Servir` y un único `ml.Pool` compartido por todos sus shards que devuelve un parcial por bloque sin reducir). Las tres figuras se embeben automáticamente como SVG válidos en el Word generado por `cmd/word` (12 SVG en total, 0 avisos).
