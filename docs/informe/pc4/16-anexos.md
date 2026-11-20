# 16. Anexos

## 16.1 Correcciones a la PC3

| Observación del profesor (PC3, 18/20) | Dónde se corrigió |
|---|---|
| Elaborar la pregunta que resuelve el problema planteado | Sección 2.2, subtítulo «Pregunta que resuelve el problema», con una tabla que une cada parte de la pregunta con la sección que la responde; la misma pregunta se retoma en los Objetivos (3.1), el Resumen (1) y las Conclusiones (13) |
| Falta incluir el código fuente del algoritmo distribuido | Sección 8.4: pseudocódigo numerado (Algoritmos 1 y 2) y el código fuente del bucle de entrenamiento, la época del coordinador, la reasignación en dos pasadas, el plazo por shard, el cálculo en el nodo y la reducción en orden, copiado del repositorio al generar el informe (`go run ./cmd/informe`); un test (`TestFragmentosSalenDelFuente`) falla si el informe no coincide con el código |

Además, las secciones 4 (arquitectura y despliegue) y 8 (clúster) pasan de diseño a implementación, y la
sección 2.6 agrega el análisis de trabajos relacionados.

## 16.2 Enlaces y ubicación

| Anexo | Enlace o ubicación |
|---|---|
| Repositorio en GitHub | <https://github.com/JorgeGarcia005/cc65-tf-dengue> |
| Tablero Kanban del proyecto | <https://github.com/users/JorgeGarcia005/projects/4> |
| Reglas del proyecto para agentes | `AGENTS.md` |
| Protocolo multiagente | `docs/agentes/PROTOCOLO.md` |
| Veredictos del juez y respuestas | `docs/agentes/revisiones/` |
| Decisiones de diseño (ADR) | `docs/decisiones/` (ADR-0005: protocolo del clúster; ADR-0006: MongoDB y Redis) |
| Contrato de la API (OpenAPI) | `docs/api/openapi.yaml` (Swagger UI en `/docs`) |
| Colección de Postman | `docs/api/cc65-tf.postman_collection.json` |
| Despliegue | `deploy/Dockerfile`, `deploy/docker-compose.yml`, `deploy/docker-compose.api.yml` |
| Scripts de evidencia | `scripts/prueba-caida.sh`, `scripts/evidencia-api.sh`, `scripts/evidencia-bd.sh` |
| Modelo Promela y su verificación | `spin/barrera.pml`, `spin/verificar.sh`, `spin/README.md` |
| Reportes con todas las cifras del informe | `reports/*.json` |
| Fuente de este informe | `docs/informe/pc4/` (se genera con `go run ./cmd/informe` y `go run ./cmd/word`) |
| Versión entregada | Rama `main`, tag `pc4` (release desde `develop`) |
| Código fuente (además del repositorio) | `1ACC0065_PC4_UXXXXXXXX.zip`, generado con `git archive` desde el tag `pc4` |
| Reporte de participación | `1ACC0065-202620_Participación_PC4.docx`, en el formato del docente, en el Aula Virtual |
| Video | No corresponde a esta entrega (TB2) |

## 16.3 Uso de inteligencia artificial

El código, las pruebas y este informe se produjeron con un ecosistema de agentes
(`docs/agentes/PROTOCOLO.md`): Claude Code implementó y orquestó, y Antigravity revisó cada *pull request*
contra la rúbrica y dejó su veredicto con evidencia en `docs/agentes/revisiones/`; en algunas rondas de la
PC4 también se usó Codex como juez. Ningún cambio entró a `develop` sin el veredicto `APROBADO` del juez, el
CI en verde y la aprobación de un integrante del equipo, que corrió las mediciones con los datos reales,
revisó y es responsable de cada parte.
