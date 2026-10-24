# Protocolo del ecosistema multiagente

Cómo colaboran el orquestador, el implementador (Claude Code), el juez (Antigravity) y el
equipo humano. Todo queda versionado en el repo: sirve como evidencia del uso responsable
de IA para el informe y para el DD (ABET SO4).

## Por qué así

Un solo agente se equivoca con confianza. Separar quien escribe de quien revisa, y obligar a
que cada objeción traiga evidencia, sube la calidad sin convertir la revisión en opinión.
El humano conserva la última palabra y, sobre todo, entiende lo que se entrega.

## Flujo de un issue

```
Orquestador        Implementador (Claude Code)      Juez (Antigravity)        Humano
    │ issue + criterios ─►│                               │                       │
    │                     │ feature/<n>-…  + tests        │                       │
    │                     │ make check ✓  → PR a develop  │                       │
    │                     │──────────── aviso ──────────────────────────────────►│
    │                     │                               │◄── /juez <n> <rama> ──│
    │                     │                               │ ronda-1.md (push)     │
    │                     │◄─────── veredicto ────────────│                       │
    │                     │ respuesta-1.md + commits      │                       │
    │                     │            ... hasta 3 rondas ...                     │
    │◄── desacuerdo ──────│ (solo si hay ESCALADO)        │                       │
    │ decisión escrita    │                               │                       │
    │                     │                               │      APROBADO + CI ✓ ─►│ merge
```

## Paso a paso

1. **Issue.** El orquestador lo crea con: contexto, criterios de aceptación verificables
   (cada uno con su comando o test), parte de la rúbrica que mueve y estimado en horas.
   Va al tablero Kanban en *Backlog*.
2. **Implementación.** Claude Code mueve el issue a *En curso*, trabaja en
   `feature/<n>-<slug>`, abre el PR hacia `develop` con la plantilla completa y mueve el
   issue a *En revisión*.
3. **Juicio.** Un humano abre Antigravity en su PC, sobre el repo, y ejecuta
   `/juez <n> <rama>`. El juez deja `docs/agentes/revisiones/PR-<n>/ronda-1.md` en la rama
   `revision/PR-<n>`.
4. **Respuesta.** Claude Code trae el veredicto, responde con la skill `responder-revision`
   y sube `respuesta-1.md` más los commits de corrección.
5. **Rondas.** Se repite hasta `APROBADO` o hasta la ronda 3.
6. **Escalamiento.** Si tras la ronda 3 hay desacuerdo, el orquestador lee ambas posiciones,
   decide y lo registra al final de la última respuesta (`## Decisión del orquestador`).
7. **Merge.** Requiere: veredicto `APROBADO`, CI verde y aprobación de un integrante
   humano en GitHub. Se fusiona con *squash* a `develop` y el issue pasa a *Hecho*.

## Debates de diseño (antes de implementar)

Para decisiones caras de revertir (topología del clúster, protocolo TCP, esquema de Mongo,
estrategia de caché) se debate antes de escribir código:

1. El orquestador o el implementador escribe `docs/decisiones/propuestas/<tema>.md`.
2. El juez responde con `/debate <tema>` en `<tema>-juez.md`.
3. El orquestador sintetiza en `docs/decisiones/ADR-XXXX-<tema>.md` (contexto, opciones,
   decisión, consecuencias) citando ambos documentos.

## Reglas comunes

- **Evidencia o no cuenta:** archivo:línea, comando reproducible, test o cita del
  enunciado/rúbrica/bibliografía.
- **Nadie se aprueba a sí mismo.** El juez no edita código; el implementador no edita
  veredictos.
- **Trazabilidad de IA:** cada PR declara qué agente lo generó, qué modelo, y qué verificó
  el humano. Nada se entrega que un integrante no pueda explicar.
- **Sin secretos en el repo:** `.env` local; en Compose, variables con valores de ejemplo
  en `deploy/.env.example`.

## Puesta en marcha (una vez por integrante)

1. Clonar el repo y abrirlo en Antigravity como workspace. Antigravity lee `AGENTS.md`,
   `.agents/rules/`, `.agents/skills/` y `.agents/workflows/` automáticamente.
2. Verificar: en el chat de Antigravity, escribir `/juez` debe sugerir el workflow.
3. Tener Go instalado (la misma versión que `go.mod`) y `make` (en Windows: Git Bash o WSL).
4. En GitHub: **Projects → New project → Board** llamado `TF CC65`, columnas
   *Backlog / En curso / En revisión / Hecho*, y vincularlo al repo (la rúbrica lo exige).
