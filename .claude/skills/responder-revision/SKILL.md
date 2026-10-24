---
name: responder-revision
description: Responde un veredicto del juez sobre un PR del Trabajo Final CC65, hallazgo por hallazgo, aceptando con commit o rebatiendo con evidencia. Úsala cuando aparezca un archivo ronda-<k>.md en docs/agentes/revisiones/PR-<n>/ para un PR tuyo.
---

# Responder revisión

## Procedimiento

1. Trae la revisión: `git fetch origin && git merge --ff-only origin/revision/PR-<n>` sobre tu
   rama del PR (o `git cherry-pick` del commit del veredicto si la rama divergió).
2. Lee `docs/agentes/revisiones/PR-<n>/ronda-<k>.md` completo antes de tocar código.
3. Para cada hallazgo, decide:
   - **ACEPTO**: corrígelo con un commit pequeño (`fix(<área>): <qué> (PR-<n> H<i>)`) y anota el hash.
   - **REBATO**: aporta evidencia que el juez pueda reproducir: un test nuevo que pasa, la
     salida de un comando, la cita del enunciado o de la teoría (Ben-Ari, apuntes del curso).
   - **ACLARO**: el hallazgo nace de un malentendido; explica y, si hace falta, mejora la
     documentación o el nombre para que no se repita.
4. Escribe `docs/agentes/revisiones/PR-<n>/respuesta-<k>.md` con esta tabla:

   | Hallazgo | Severidad | Decisión | Evidencia / commit |
   |---|---|---|---|
   | H1 | BLOQUEANTE | ACEPTO | `a1b2c3d` — test `TestReduccionOrdenada` agregado |

5. `make check` en verde, push de la rama del PR y aviso al humano para que el juez haga la
   ronda `k+1`.

## Reglas

- Ningún BLOQUEANTE se rebate sin un test o comando reproducible.
- No mezcles cambios no pedidos en los commits de respuesta.
- Si tras la ronda 3 queda un desacuerdo, escribe en la respuesta `ESCALADO AL ORQUESTADOR`
  con un resumen de ambas posiciones en 5 líneas como máximo.
