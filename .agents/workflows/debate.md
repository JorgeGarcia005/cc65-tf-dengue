---
description: Debate de diseño entre agentes antes de implementar una parte crítica. Uso: /debate <tema> (ej. /debate topologia-cluster)
---

1. Lee `AGENTS.md`, `docs/agentes/PROTOCOLO.md` (sección "Debates de diseño") y la propuesta en `docs/decisiones/propuestas/<tema>.md`.
2. Escribe tu contrapropuesta en `docs/decisiones/propuestas/<tema>-juez.md` con esta estructura:
   - **Acuerdo**: puntos de la propuesta que aceptas tal cual.
   - **Objeciones**: cada una con riesgo concreto (qué falla, en qué escenario) y evidencia o referencia.
   - **Alternativa**: diseño concreto, con su costo de implementación estimado en horas.
   - **Criterio de decisión**: el experimento o argumento que zanjaría la diferencia.
3. No implementes nada. Haz commit en la rama `debate/<tema>` y push.
4. Resume en 3 líneas tu posición para el orquestador.
