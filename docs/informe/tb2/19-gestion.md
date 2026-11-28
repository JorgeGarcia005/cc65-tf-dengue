# 19. Gestión del proyecto

El proyecto se gestionó con Git Flow en GitHub (19.1), un tablero Kanban (19.2) y un ecosistema de agentes de IA con revisión cruzada, en el que los integrantes aprueban cada integración (19.3). La sección termina con la participación de cada integrante (19.4).

## 19.1 Git Flow

El repositorio (<https://github.com/JorgeGarcia005/cc65-tf-dengue>) sigue Git Flow, con las reglas de `AGENTS.md` §4:

Tabla: Ramas del repositorio y su uso {#tab:ramas}

| Rama | Para qué | Cómo se integra |
|---|---|---|
| `main` | Solo versiones entregadas | Desde `develop` en cada entrega, con etiqueta (`pc3`, `pc4`, `tb2`) |
| `develop` | Integración de los entregables | Por Pull Request desde las ramas de trabajo |
| `integracion/tb2` | Integración de la TB2 mientras la PC4 estaba en revisión: `develop` y `main` quedaron congelados para no alterar la entrega calificada | Por Pull Request; pasa a `develop` y `main` en la entrega `tb2` |
| `feature/<issue>-<tema>` | Una funcionalidad o una sección del informe | Pull Request con veredicto del juez y aprobación humana |
| `fix/<issue>-<tema>` | Una corrección | Igual que `feature/` |
| `revision/PR-<n>` | Veredictos del juez sobre un Pull Request | Se integran al Pull Request revisado |

Nota: Elaboración propia a partir de `AGENTS.md`.

**Reglas que se cumplieron en todo el proyecto:**

- **Nada entra directo** a `develop`, `main` ni `integracion/tb2`: todo pasa por un Pull Request.
- **Un Pull Request por issue**, con la plantilla del repositorio: qué cambia, cómo explicarlo en la sustentación, criterios de aceptación con el comando que los verifica, verificación, rúbrica que mueve y uso de IA.
- **Commits en español**, en imperativo y con el formato convencional (`feat(ml): agrega reducción ordenada`).
- **Integración continua** en cada Pull Request: `gofmt`, `go vet` y `go test -race`; pruebas de la SPA; MongoDB y Redis reales; Spin; validación del OpenAPI; imagen de Docker con el clúster y la prueba de caída de un nodo. Un Pull Request con el CI en rojo no se integra.
- **Integración con *merge commit*** en la TB2, para que la historia conserve cada Pull Request como una unidad.

La evidencia siguiente, tomada del repositorio al cerrar el informe, muestra las ramas principales, las etiquetas de cada entrega, los Pull Requests integrados y los commits de cada autor:

<!-- evidencia: evidencias/gestion.txt -->

## 19.2 Tablero Kanban

Cada issue vivió en el tablero «TF CC65» de GitHub Projects, con cuatro columnas: **Backlog**, **En curso**, **En revisión** y **Hecho**. Un issue pasa a «En revisión» cuando se abre su Pull Request y a «Hecho» cuando se integra. Los issues de la TB2 se agruparon en el *milestone* «TB2» y se ordenaron por dependencias: seguridad y datos primero, luego el WebSocket y el mapa, después las vistas, las pruebas de punta a punta y, al final, el informe y la entrega (@fig:kanban).

<!-- captura: Tablero Kanban del proyecto al cierre del informe {#fig:kanban} [capturas/m-kanban.png] -->

## 19.3 Ecosistema multiagente y uso de IA

El proyecto usó agentes de IA con roles separados y un protocolo escrito (`docs/agentes/PROTOCOLO.md`), para que ningún agente aprobara su propio trabajo y cada decisión quedara con evidencia:

Tabla: Roles del ecosistema multiagente {#tab:roles-agentes}

| Rol | Quién | Qué hace |
|---|---|---|
| Orquestador | Claude (chat del proyecto) | Divide el trabajo en issues con criterios de aceptación verificables y decide cuando hay desacuerdo |
| Implementador | Claude Code | Escribe el código, las pruebas y la sección del informe; abre el Pull Request y responde al juez |
| Juez | Antigravity (Gemini) | Revisa cada Pull Request contra la rúbrica, el enunciado y `AGENTS.md`, y emite un veredicto con evidencia |
| Equipo humano | Jorge y José | Aprueban cada integración, ejecutan las mediciones y las evidencias en la máquina de referencia y defienden el trabajo en la sustentación |

Nota: Elaboración propia a partir de `AGENTS.md` §5.

**El ciclo de revisión.** El juez escribe su veredicto en `docs/agentes/revisiones/PR-<n>/ronda-<k>.md`: puntaje por criterio, hallazgos con su severidad (bloqueante, mayor o menor) y evidencia reproducible (archivo y línea, comando o cita del enunciado), y una decisión: APROBADO, CAMBIOS o RECHAZADO. El implementador responde cada hallazgo en `respuesta-<k>.md`, aceptándolo con el commit que lo corrige o rebatiéndolo con evidencia. Hay como máximo tres rondas; un Pull Request se integra solo con veredicto APROBADO, el CI en verde y la aprobación de un integrante. La @tab:revisiones resume todas las revisiones del proyecto.

<!-- generado:revisiones -->
Tabla: Revisiones del juez sobre los Pull Requests del proyecto {#tab:revisiones}

| Indicador | Valor |
|---|---:|
| Pull Requests con veredicto del juez | 41 |
| Rondas de revisión en total | 76 |
| Aprobados en la primera ronda | 10 |
| Pull Requests que necesitaron 2 rondas | 27 |
| Pull Requests que necesitaron 3 rondas | 4 |
| Terminados con veredicto APROBADO | 41 |
| Hallazgos de severidad bloqueante | 31 |
| Hallazgos de severidad mayor | 64 |
| Hallazgos de severidad menor | 48 |

Nota: Cuenta los archivos `ronda-<k>.md` de `docs/agentes/revisiones/PR-<n>/`: la decisión de cada ronda y los hallazgos con su severidad. Generado por `cmd/informe`.
<!-- /generado -->

Dos Pull Requests de la TB2 se integraron sin veredicto del juez, solo con revisión humana y el CI en verde: el #88, que corrigió una prueba intermitente del WebSocket, y el #100, la guía de evidencias para José. Se declaran aquí para que la tabla refleje exactamente lo ocurrido.

**Qué aportó la revisión.** Los hallazgos bloqueantes fueron errores que habrían llegado a la entrega: un modelo anunciado como publicado que la base de datos no había guardado, una carrera en una prueba que daba por buena una versión desfasada, un guion de seguridad que podía omitir casos sin avisarlo, entre otros. Varios de ellos se convirtieron en pruebas automáticas que hoy los impiden. Las decisiones de diseño caras de revertir, como el protocolo del clúster, pasaron además por un debate escrito entre implementador y juez antes de escribir código (ADR-0005).

**Uso de IA declarado.** Cada Pull Request indica qué agente lo produjo, con qué modelo, qué revisó el integrante y el veredicto del juez. Los integrantes ejecutaron en su máquina todas las mediciones y evidencias del informe con guiones que registran fecha, máquina y commit, de modo que ninguna cifra la produjo un agente sin ejecución real.

## 19.4 Participación

El equipo comenzó con tres integrantes; desde el 27/11/2026 son Jorge García y José Villanueva, porque el tercer integrante se retiró del curso (el enunciado admite equipos de dos a tres).

Tabla: Responsabilidades de cada integrante {#tab:participacion}

| Integrante | Responsabilidades principales |
|---|---|
| Jorge Rolando García Roca | Coordinación del proyecto y de los agentes; revisión y aprobación de los Pull Requests; arquitectura, clúster, API y SPA; informe |
| José Javier Villanueva Aramburú | Ejecución de las mediciones y de las evidencias en la máquina de referencia; pruebas de punta a punta y capturas; armado y revisión del documento final del informe |

Nota: Elaboración propia. Los commits de cada integrante se ven en la evidencia de la sección 19.1; el reporte de participación se entrega en un archivo aparte (Anexo F).
