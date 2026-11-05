# 9. Evidencias de la implementación y pruebas

## 9.1 Organización del código

| Paquete | Responsabilidad |
|---|---|
| `internal/ingest` | Lectura por bloques de bytes, parseo CSV tolerante, reglas de limpieza, perfil del dataset y *worker pool* de la Parte A |
| `internal/features` | Esquema de variables, estandarización ajustada solo en entrenamiento y split determinista |
| `internal/gold` | Formato binario de la matriz de entrenamiento y partición en shards de bloques contiguos |
| `internal/ml` | Regresión logística secuencial de referencia, *worker pool* del gradiente, reducción ordenada y métricas |
| `internal/estadistica` | Media recortada, mediana e intervalos de confianza por *bootstrap* para los benchmarks |
| `internal/maquina` | Descripción del equipo en cada reporte de benchmark |
| `cmd/*` | Programas: `loader`, `perfil`, `gold`, `entrenar`, `benchmark` e `informe` |

## 9.2 Pruebas automatizadas

Las pruebas se escribieron antes que el código (TDD) en la lógica concurrente y numérica, y **todas se
ejecutan con el detector de carreras** (`go test -race`), en local y en la integración continua de GitHub
en cada Pull Request. Las más importantes para esta entrega:

| Test | Qué garantiza |
|---|---|
| `TestEquivalencia` | El gradiente del *worker pool* es idéntico bit a bit al secuencial con 1 a 64 workers |
| `TestShardsReproducenElGradienteGlobal` | Repartir los bloques en 1, 2, 3, 5 u 8 shards y reducir en orden da los mismos bits que el secuencial: la base del clúster |
| `TestReducirEnOrdenIgualASecuencial` | La reducción ordenada coincide con la suma secuencial |
| `TestLimpiezaConcurrenteIgualALaSecuencial` | El silver concurrente es idéntico byte a byte al de un solo worker |
| `TestMapearEnOrdenEsDeterministaYSinCarreras` | El mapeo concurrente por bloques conserva el orden y no tiene carreras |
| `TestPoolLlamadasConcurrentes` | El *worker pool* es seguro aunque varias goroutines lo usen a la vez |
| `TestGradienteDiferenciasFinitas` | El gradiente analítico coincide con el numérico: la matemática es correcta |
| `TestBloquesCubrenTodoSinSolaparse` | Los bloques de bytes cubren el archivo completo sin solaparse ni cortar líneas |
| `TestDecodificarRechazaTamBloqueCero` | Un archivo gold con tamaño de bloque cero se rechaza en vez de provocar una división entre cero |

Salida de `go test -race` sobre todo el repositorio:

<!-- incluir-codigo: evidencias/go-test-race.txt -->

## 9.3 Reproducibilidad de los resultados

Todas las cifras de este informe se pueden regenerar desde el repositorio con el dataset verificado por su
checksum:

```text
make data                                   # descarga el CSV y verifica su sha256
go run ./cmd/loader                         # bronze → silver + reports/limpieza.json
go run ./cmd/gold                           # silver → gold, split y shards
go run ./cmd/entrenar                       # entrenamiento + reports/entrenamiento.json
go run ./cmd/loader -bench                  # speedup de la limpieza
go run ./cmd/benchmark -gold data/gold/train.bin   # speedup del gradiente
go run ./cmd/informe                        # tablas, figuras y secciones con cifras
```

## 9.4 Git Flow

El repositorio sigue Git Flow: `main` solo recibe *releases* por entregable, `develop` integra, y cada
tarea vive en una rama `feature/<issue>-<descripción>` que entra a `develop` únicamente por Pull Request,
después de pasar la integración continua y la revisión del agente juez.

<!-- incluir: evidencias/prs.md -->

Historial de ramas (`git log --graph`, extracto):

<!-- incluir-codigo: evidencias/gitflow.txt -->

<!-- incluir: evidencias/commits.md -->

## 9.5 Tablero Kanban

Cada tarea es un issue con criterios de aceptación verificables y se sigue en el tablero Kanban del
proyecto en GitHub (columnas *Backlog*, *En curso*, *En revisión* y *Hecho*). Estado del tablero al
generar este informe:

<!-- incluir: evidencias/kanban.md -->

## 9.6 Ecosistema multiagente y uso responsable de IA

El desarrollo usa tres roles de IA con responsabilidades separadas, más la aprobación humana:

| Rol | Quién | Qué hace |
|---|---|---|
| Orquestador | Claude (chat del proyecto) | Divide el trabajo en issues con criterios verificables y decide si hay desacuerdo |
| Implementador | Claude Code | Escribe código, pruebas y secciones del informe; abre el Pull Request |
| Juez | Antigravity (Gemini) | Revisa cada Pull Request contra la rúbrica y las reglas del proyecto, con evidencia (archivo y línea, salida de pruebas) |
| Equipo | Jorge García y José Villanueva | Aprueban el *merge*, ejecutan las mediciones y entienden y defienden cada parte |

Cada veredicto del juez y cada respuesta del implementador quedan versionados en
`docs/agentes/revisiones/PR-<n>/`. Un hallazgo solo cuenta si trae evidencia, y ningún agente puede
aprobar su propio trabajo. Cada Pull Request declara qué agente lo generó y qué revisó el integrante.
