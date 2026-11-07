# 13. Anexos

| Anexo | Enlace o ubicación |
|---|---|
| Repositorio en GitHub | <https://github.com/JorgeGarcia005/cc65-tf-dengue> |
| Tablero Kanban del proyecto | <https://github.com/users/JorgeGarcia005/projects/4> |
| Reglas del proyecto para agentes | `AGENTS.md` |
| Protocolo multiagente | `docs/agentes/PROTOCOLO.md` |
| Veredictos del juez y respuestas | `docs/agentes/revisiones/` |
| Decisiones de diseño (ADR) | `docs/decisiones/` |
| Propuesta del protocolo coordinador–nodos | `docs/decisiones/propuestas/protocolo-tcp.md` |
| Modelo Promela y cómo verificarlo | `spin/barrera.pml`, `spin/README.md` |
| Reportes con todas las cifras del informe | `reports/*.json` |
| Fuente de este informe | `docs/informe/pc3/` (se genera con `go run ./cmd/informe` y `go run ./cmd/word`) |
| Versión entregada | Rama `main`, tag `pc3` (release desde `develop`) |
| Código fuente (además del repositorio) | `1ACC0065_PC3_UXXXXXXXX.zip`, generado con `git archive` desde el tag `pc3` |
| Reporte de participación | `1ACC0065-202620_Participación_PC3.docx`, en el formato del docente, en el Aula Virtual |
| Video | No corresponde a esta entrega (TB2) |

**Uso de inteligencia artificial.** El código, las pruebas y este informe se produjeron con un
ecosistema de agentes (`docs/agentes/PROTOCOLO.md`): Claude Code implementó y orquestó, y Antigravity
(Gemini) revisó cada *pull request* contra la rúbrica y dejó su veredicto con evidencia en
`docs/agentes/revisiones/`. Ningún cambio entró a `develop` sin el veredicto `APROBADO` del juez, el CI en
verde y la aprobación de un integrante del equipo, que revisó y es responsable de cada parte.
