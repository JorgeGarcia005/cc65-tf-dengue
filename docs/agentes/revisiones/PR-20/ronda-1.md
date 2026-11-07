# Veredicto — PR-20 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/10-informe-pc3` @ `c9c9b26`
- **Fecha:** 2026-11-07
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.687s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	2.582s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	2.971s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	2.491s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	2.743s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	2.174s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.959s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	2.091s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.491s
$ go test -race -count=1 -run Equivalencia -v ./...
=== RUN   TestEquivalencia
--- PASS: TestEquivalencia (0.29s)
    --- PASS: TestEquivalencia/workers=1 (0.07s)
    --- PASS: TestEquivalencia/workers=2 (0.04s)
    --- PASS: TestEquivalencia/workers=4 (0.02s)
    --- PASS: TestEquivalencia/workers=8 (0.03s)
    --- PASS: TestEquivalencia/workers=16 (0.03s)
    --- PASS: TestEquivalencia/workers=64 (0.03s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	2.794s
$ go run ./cmd/informe && git status --short
(vacío = idempotente; 0 diferencias en docs/informe/pc3/ y fig/*.svg)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; CI de GitHub en verde en `c9c9b26`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia` con 1..64 workers + `reports/entrenamiento.json` con 16 workers idéntico en pesos, sesgo y pérdida por época al entrenamiento con 2 workers) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Problema, motivación alineada al ODS 3 (metas 3.3 y 3.d), pregunta de impacto social por departamento, semana epidemiológica y etapa de vida ([`02-problema.md`](docs/informe/pc3/02-problema.md)), objetivos ([`03-objetivos.md`](docs/informe/pc3/03-objetivos.md)), diagrama de arquitectura distinguiendo PC3 vs. PC4/TB2 ([`04-arquitectura.md`](docs/informe/pc3/04-arquitectura.md), [`arquitectura.svg`](docs/informe/pc3/fig/arquitectura.svg)) y mapeo completo del Entregable 1 (§4.2). |
| Implementación | 8 | Avance completo de PC3 | Añade [`features.SemanaDeAngulo`](internal/features/features.go#L136-L142) e `Indice` con test de inversión ([`TestSemanaDeAnguloInvierteLaCodificacion`](internal/features/features_test.go#L107-L130)) y [`agruparPorSemana`](cmd/entrenar/main.go#L197-L220) ([`TestAgruparPorSemana`](cmd/entrenar/main_test.go#L31-L60)) para responder la dimensión temporal de la pregunta de impacto social sin alterar pesos ni métricas del modelo. |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | 2 / 2 | Estructura completa a)–j) del enunciado en `docs/informe/pc3/01..13`, generación 100 % reproducible desde `reports/*.json` mediante [`cmd/informe`](cmd/informe/main.go) (tablas, resumen, conclusiones y 5 gráficos SVG sin doble eje Y), armado automático en Word (`.docx`) solo con stdlib mediante [`cmd/word`](cmd/word/main.go) (ecuaciones LaTeX a OMML nativo, tablas, figuras SVG y sangría francesa APA 7), sin códigos de alumno hardcodeados en el repo público y con declaración explícita del ecosistema multiagente en [`13-anexos.md`](docs/informe/pc3/13-anexos.md). |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

Sin hallazgos abiertos en esta ronda.

> **Nota operativa para el equipo humano antes de subir al Aula Virtual:**
> 1. Mover los issues correspondientes en el tablero Kanban de GitHub e insertar las 6 capturas de pantalla en los marcadores `[Pegar aquí la captura]` del capítulo 9 (`09-evidencias.md`) en el `.docx` final antes de correr `scripts/word-final.ps1` para actualizar la tabla de contenido.
> 2. Una vez fusionados los PRs de PC3 a `develop` y de `develop` a `main`, crear el tag `pc3` en `main`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión. |

## Cómo explicarlo en la sustentación

Ninguna cifra, tabla ni gráfico de resultados del informe está escrito a mano: los programas de ingesta, entrenamiento y benchmark escriben sus salidas en `reports/*.json`; `cmd/informe` los transforma de forma determinista en las secciones Markdown (`01-resumen.md`, `05-datos.md`, `06-limpieza.md`, `07-modelo.md`, `10-conclusiones.md`) y en gráficos SVG vectoriales; y `cmd/word` ensambla el `.docx` final usando únicamente la biblioteca estándar de Go (`archive/zip`, `encoding/xml`), traduciendo las ecuaciones LaTeX a OMML nativo de Word y conservando la carátula y estilos del equipo sin exponer códigos de alumno en el repositorio público. Además, para responder la dimensión temporal de la pregunta de impacto social, `features.SemanaDeAngulo` invierte con `atan2` la codificación cíclica `(semana_sin, semana_cos)` del conjunto de prueba, mostrando que el volumen de casos graves se concentra en las semanas 15–16 y 18–20 (demanda pico de camas), mientras que el riesgo calibrado por caso es mayor en las semanas 45–49 (17,18 % vs. 10,40 %).
