# Veredicto — PR-98 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/97-informe-tb2-bloque2` @ `c79e131`
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

El PR completa el bloque 2 de `#71` (issue `#97`) redactando de forma rigurosa, verificable y 100 % sincronizada con el repositorio las secciones 1 a 5 del informe final TB2 (y las referencias APA 7 citadas en ellas):

1. **Cero cifras a mano (`AGENTS.md` §3.6):** más de 40 valores generados en línea (`<!--v:EXPR-->…<!--/v-->`) en el resumen (`01-resumen.md`), planteamiento y dataset (`02-problema.md`), matriz de objetivos (`03-objetivos.md`) y matriz de restricciones (`04-arquitectura.md`), respaldados por `.R.<reporte>` y `.APIJWT` en `cmd/informe/main.go:526-554` y verificados automáticamente por `TestInformeTB2AlDia`.
2. **Atención directa a las observaciones previas del profesor:**
   - *PC1 (análisis de papers):* la sección 2.6 y `@tab:relacionados` desglosan 9 referencias (MapReduce, Spark, *parameter server*, *ring all-reduce*, Goldberg, Huang et al. 2020, Bhatt et al. 2013, King y Zeng 2001 y Spin) especificando qué propone cada una, qué adopta el sistema y qué descarta con justificación técnica; todas las citas coinciden con `22-referencias.md`.
   - *PC3 (pregunta que resuelve el problema):* la sección 2.3 y `@tab:pregunta` enlazan cada cláusula de la pregunta con su evidencia y sección, retomándola en los objetivos (`@tab:objetivos`).
   - *PC4 #1 (Dockerfile, Compose, inicio del entrenamiento y scripts):* las secciones 5.1–5.7 incrustan del árbol fuente los tres `Dockerfile`, los cinco bloques de `deploy/docker-compose.yml` (solo comentarios `# informe:…`; `docker compose config` permanece idéntico), `deploy/docker-compose.api.yml`, `deploy/.env.example` y las funciones Go `conectar`, `ejecutar` y `(*entrenadorCluster).Entrenar`, junto con la máquina de estados UML 2.5 (`@fig:estados-coordinador`) que contrasta qué dispara el entrenamiento en el modo único (registro del `N`-ésimo nodo) frente al modo servicio (`POST /api/v1/entrenamientos` con rol `admin`).
3. **Arquitectura y mejoras de `cmd/word`:** diagrama de actividades UML 2.5 del flujo de datos (`@fig:flujo-datos`), comparación con la Fig. 1 del enunciado (`@tab:fig1`), tablas de componentes, restricciones y archivos (`@tab:componentes`, `@tab:restricciones`, `@tab:requisitos`), `<w:keepNext/>` en imágenes y capturas seguidas de `Nota:` (`TestNotaEnLaMismaPagina`) y sangría bilateral APA en citas en bloque (`cmd/word/md.go:317-319`).

Todas las comprobaciones automáticas (`gofmt`, `go vet`, `go test -race ./...` en las 25 suites con 0 carreras, `TestInformeTB2AlDia` y `TestInformesSinAvisos`) y los 7 trabajos del CI en GitHub Actions están en verde. Se deja únicamente una observación **MENOR *(no bloqueante)*** (**H1**) como previsión para los bloques 3–5 de `#71`.

## Verificación automática

```text
$ git rev-parse --short HEAD
c79e131

$ git diff --check origin/integracion/tb2...origin/feature/97-informe-tb2-bloque2
(vacío = ok)

$ docker compose -f deploy/docker-compose.yml config --quiet
(salida 0: los marcadores # informe:… no alteran la configuración efectiva de Compose)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && go run ./cmd/informe && git diff --exit-code
skills: ok
2026/11/28 04:10:48 escrito docs/informe/tb2/fig/casos_por_ano.svg
2026/11/28 04:10:48 escrito docs/informe/tb2/fig/riesgo_departamento.svg
2026/11/28 04:10:48 escrito docs/informe/tb2/fig/riesgo_semana.svg
2026/11/28 04:10:48 escrito docs/informe/tb2/fig/speedup_gradiente.svg
2026/11/28 04:10:48 escrito docs/informe/tb2/fig/speedup_loader.svg
2026/11/28 04:10:48 escrito docs/informe/tb2/fig/speedup_cluster.svg
(sin diferencias en el árbol de trabajo)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	12.623s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	10.610s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.947s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.056s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.650s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.282s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.823s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.429s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.442s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.056s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	4.384s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.030s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.129s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	29.320s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.316s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.921s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.129s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.074s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.097s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.079s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.878s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.806s

$ gh pr checks 98 -R JorgeGarcia005/cc65-tf-dengue   # run 46106171564 sobre c79e131
7/7 jobs en verde (Ecosistema, Spin, OpenAPI 3, MongoDB/Redis, Go -race, SPA, Docker)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`25/25` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`) |
| Sincronía de cifras, regiones y referencias (`cmd/informe`, `cmd/word`) | ✅ (`TestInformeTB2AlDia`, `TestInformesSinAvisos` y `TestNotaEnLaMismaPagina` en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Cubre íntegramente problema, impacto social ODS 3, análisis crítico de bibliografía, dataset, diagramas UML de despliegue, flujo y estados, comparación con la Fig. 1 del enunciado y matriz de restricciones verificables. |
| Implementación | 8 | documenta Docker | Documenta con exactitud los tres `Dockerfile`, los dos archivos de Compose, redes, puertos, volúmenes y el código de arranque/entrenamiento en ambos modos (observación 1 de la PC4). |
| Interfaz | 5 | sin cambios | No modifica `web/`. |
| Informe | 2 | avance sólido (secciones 1–5 y 22) | Estructura, estilo académico APA 7, trazabilidad automática a `reports/*.json` y cero referencias cruzadas rotas. |
| Video | 2 | sin cambios | No afecta el video. |

## Hallazgos

### H1 — Tope de alto proporcional en `figura` para diagramas verticales futuros y nota de contingencia en §5.7 · MENOR *(no bloqueante)*

- **Dónde:** `cmd/word/md.go:638-641` y `docs/informe/tb2/05-despliegue.md:624-627`
- **Evidencia:**
  1. En `cmd/word/md.go:638-639`, `figura` escala siempre al ancho completo del texto (`cx = 9122 × 635` EMU) y calcula `cy = cx × h / w` sin cota superior de altura. En `docs/informe/tb2/fig/flujo_datos.svg` (`833pt × 923pt`), `cy` llega a `10 108 dxa` (~`17,8 cm`) y, unido por `<w:keepNext/>` con el pie APA superior y `Nota.` inferior (`~11 728 dxa`), todavía cabe dentro del alto útil de una página carta (`~12 960 dxa`). Sin embargo, si en los bloques 3–5 se incorpora un diagrama más vertical (p. ej. un diagrama de secuencia del protocolo TCP en la sección 9.2 con `h / w > 1,25`), la cadena `<w:keepNext/>` superará el alto útil de página y Word recortará la figura o romperá la unión con su nota.
  2. En `docs/informe/tb2/05-despliegue.md:624-627` (§5.7 *Ejecución paso a paso*), en Linux conviene mencionar `mkdir -p data reports` antes del paso 2 (como en `README.md:54` y `Makefile:36`, para que el demonio de Docker no cree esas carpetas como `root`) y una línea recordando el camino manual / `gold -sintetico` si `make data` recibe `HTTP 418` del WAF del portal MINSA (`README.md:65-68, 86-88`).
- **Corrección sugerida (opcional en los bloques 3–5):** Acotar `cy` en `cmd/word/md.go:639` (p. ej. `altoMax = 9500 * emuPorDxa`; si `cy > altoMax`, escalar `cx = cx * altoMax / cy` y `cy = altoMax` conservando la proporción), y añadir el apunte de `mkdir -p data reports` / `HTTP 418` al insertar las capturas de la sección 5.7 en `#70`.

## Seguimiento de rondas previas

Primera ronda para este PR.

## Cómo explicarlo en la sustentación

Las secciones 1 a 5 establecen el hilo conductor del proyecto desde la carga hospitalaria por dengue con signos de alarma en el Perú hasta la arquitectura contenerizada en Go: la sección 4 contrasta el despliegue real en tres imágenes `distroless:nonroot` con la Fig. 1 del enunciado y vincula cada restricción con la prueba del CI que la verifica, mientras la sección 5 responde la observación 1 de la PC4 incrustando directamente los `Dockerfile`, los servicios de Docker Compose y las funciones del coordinador para mostrar cómo el entrenamiento arranca al registrarse el último nodo en el modo único y mediante `POST /api/v1/entrenamientos` en el modo servicio.
