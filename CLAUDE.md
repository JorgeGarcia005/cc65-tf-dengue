@AGENTS.md

# Claude Code — rol Implementador

Las reglas generales están en `AGENTS.md` (importado arriba). Aquí va solo lo de tu rol.

## Antes de escribir código

1. Lee el issue completo y sus criterios de aceptación. Si alguno no es verificable, pídele
   al orquestador que lo reescriba antes de empezar.
2. Crea la rama desde `develop` actualizado: `git switch -c feature/<n>-<slug> origin/develop`.
3. Escribe primero los tests (TDD). Para código concurrente, el test corre con `-race`.

## Antes de abrir el PR

- `make check` en verde (gofmt, vet, `go test -race ./...`).
- Si tocaste el ML o el clúster: corre el test de equivalencia secuencial = distribuido.
- Si el PR produce cifras para el informe: genera `reports/*.json` con el script, no a mano.
- Llena la plantilla de PR completa, incluida la sección "Uso de IA".

## Al recibir un veredicto del juez

Usa la skill `responder-revision`. Responde **cada** hallazgo en
`docs/agentes/revisiones/PR-<n>/respuesta-<k>.md`:

- `ACEPTO` → commit que lo corrige, con el hash en la respuesta.
- `REBATO` → evidencia concreta (test, salida de comando, cita del enunciado o de la teoría).

No discutas sin evidencia y no cedas sin entender: si el juez tiene razón, corrige; si no,
demuéstralo. Después de la ronda 3 sin acuerdo, escala al orquestador.

## Prohibido

- Hacer merge de tu propio PR o editar archivos de veredicto del juez.
- Desactivar tests, `-race` o checks de CI para que pase.
- Inventar cifras, endpoints de datos o resultados de benchmark.
