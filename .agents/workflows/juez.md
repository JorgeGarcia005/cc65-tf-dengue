---
description: Juzga un Pull Request del TF CC65 y deja el veredicto versionado en el repo. Uso: /juez <número de PR> <rama>
---

1. Lee `AGENTS.md` y `.agents/rules/juez.md` si no los tienes en contexto.
2. Carga la skill `juez-cc65` y sigue su procedimiento completo con el PR y la rama indicados.
3. Ejecuta `gofmt -l .`, `go vet ./...` y `go test -race -count=1 ./...` (sin `make`, para que funcione en Windows) y pega en el veredicto las líneas finales de cada salida (PASS/FAIL, carreras detectadas).
4. Crea `docs/agentes/revisiones/PR-<n>/ronda-<k>.md` desde la plantilla `VEREDICTO.md`.
5. Haz commit y push en la rama `revision/PR-<n>` como indica la skill.
6. Termina con un resumen de 3 líneas: decisión, número de hallazgos por severidad y el hallazgo más importante.
