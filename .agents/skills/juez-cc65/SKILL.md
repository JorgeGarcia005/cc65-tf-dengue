---
name: juez-cc65
description: Revisa un Pull Request del Trabajo Final CC65 contra la rúbrica TB2, el enunciado y las restricciones de AGENTS.md, y emite un veredicto con evidencia en docs/agentes/revisiones/. Úsala cuando te pidan juzgar, revisar o auditar un PR o una rama.
---

# Juez CC65

Revisas como lo haría el profesor Carlos Jara en la sustentación: ¿funciona, es correcto en
concurrencia, cumple el enunciado y el equipo puede explicarlo?

## Entrada

Número de PR `<n>` y rama `<rama>`. Si te dan solo uno, deduce el otro con
`git branch -r` o el título del PR.

## Procedimiento

1. **Contexto.** Lee `AGENTS.md`, el issue enlazado en el PR y, si existe, la ronda anterior
   en `docs/agentes/revisiones/PR-<n>/` junto con la `respuesta-<k>.md` del implementador.
2. **Checkout limpio.** `git fetch origin && git switch --detach origin/<rama>`.
3. **Verificación automática.** Ejecuta y guarda la salida:
   - `make check` (gofmt, vet, `go test -race ./...`)
   - si el PR toca `internal/ml` o `internal/cluster`: el test de equivalencia
     (`go test -race -run Equivalencia ./...`)
   - si toca la API: `make bench-api` y revisa que p95 < 100 ms
4. **Revisión del diff** (`git diff origin/develop...origin/<rama>`) con la lista de abajo.
5. **Veredicto.** Copia `docs/agentes/revisiones/VEREDICTO.md` a
   `docs/agentes/revisiones/PR-<n>/ronda-<k>.md` y llénala. `k` = 1 + rondas existentes.
6. **Entrega.** En una rama `revision/PR-<n>` creada desde la rama del PR:
   `git add docs/agentes/revisiones/PR-<n>/ronda-<k>.md`,
   `git commit -m "docs(revision): veredicto PR-<n> ronda <k>"` y `git push -u origin HEAD`.
   Luego avisa al humano con el resumen de 3 líneas del veredicto.

## Lista de revisión

### Concurrencia y distribución (lo central del curso)
- [ ] ¿Hay estado compartido mutable? ¿Está protegido o se evitó con acumuladores privados?
- [ ] ¿Todos los goroutines terminan? (WaitGroup, cierre de canales, `context` con cancelación)
- [ ] ¿Puede haber deadlock? (envío a canal sin receptor, locks en orden distinto, TCP bloqueado)
- [ ] ¿La reducción es en orden fijo de bloque? ¿El test de equivalencia secuencial = distribuido pasa?
- [ ] ¿Qué pasa si un nodo cae o tarda? (timeout, reintento, error propagado, no cuelga)
- [ ] ¿El protocolo TCP define formato, framing y errores? ¿Está documentado?
- [ ] ¿El patrón usado (worker pool, pipeline, fan-out/fan-in, parameter server) está justificado?

### Corrección numérica y de datos
- [ ] Gradiente de la regresión logística correcto: ∇w = Σ (σ(xᵀw) − y)·x / N (más regularización si aplica)
- [ ] Sin fuga de datos (no usar en las features información posterior al diagnóstico)
- [ ] Desbalance de clases tratado y declarado (pesos o umbral), métricas adecuadas (AUC, recall)
- [ ] Limpieza reproducible: cada regla tiene conteo antes/después en `reports/`

### Restricciones de AGENTS.md §3
- [ ] Núcleo solo con stdlib; terceros solo drivers de BD
- [ ] Ninguna cifra a mano: los números del informe salen de `reports/*.json`
- [ ] `data/` fuera del repo

### Rúbrica TB2 (puntaje que arriesga este PR)
- [ ] Planteamiento (3): problema, dataset, diagrama de arquitectura, restricciones
- [ ] Implementación (8): >1M concurrente, entrenamiento paralelo, Docker, puertos, canales, API
- [ ] Interfaz (5): SPA con menú, parámetros desde la UI, tiempo real, impacto social
- [ ] Informe (2): estructura a)–j) del enunciado, APA7, Git Flow, Kanban
- [ ] Video (2): end to end, ≤ 6 min, los tres integrantes

### Sustentación
- [ ] ¿Un integrante podría explicar este PR en 1 minuto sin leer? Si no, pide una sección
      "Cómo explicarlo" en la descripción del PR.

## Reglas del veredicto

- Cada hallazgo: severidad (BLOQUEANTE / MAYOR / MENOR), ubicación `archivo:línea`,
  evidencia (salida de comando o cita), corrección sugerida.
- `APROBADO` solo sin BLOQUEANTES abiertos y con `make check` verde.
- En rondas 2 y 3: marca cada hallazgo previo como `RESUELTO`, `SE MANTIENE` (con nueva
  evidencia) o `RETIRADO` (si el rebate del implementador te convenció; dilo explícitamente).
- No repitas hallazgos ya retirados. No agregues hallazgos MENORES nuevos en la ronda 3.
