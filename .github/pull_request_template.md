## Qué cambia

Cierra #<issue>

<!-- Una o dos frases. -->

## Cómo explicarlo en la sustentación

<!-- 2-3 frases que cualquier integrante pueda decir sin leer. Si no se puede, el PR es demasiado grande. -->

## Criterios de aceptación

- [ ] <criterio del issue> — verificado con: `<comando o test>`

## Verificación

- [ ] `make check` en verde (gofmt, vet, `go test -race ./...`)
- [ ] Si toca ML/clúster: test de equivalencia secuencial = distribuido en verde
- [ ] Si produce cifras para el informe: salen de `reports/*.json`, no a mano
- [ ] Sin secretos ni datos en el diff

## Rúbrica que mueve

- [ ] Planteamiento (3) · [ ] Implementación (8) · [ ] Interfaz (5) · [ ] Informe (2) · [ ] Video (2)

## Uso de IA

- **Generado por:** Claude Code · Antigravity · humano
- **Modelo:**
- **Revisado por el humano:** <qué leyó / probó el integrante>
- **Veredicto del juez:** `docs/agentes/revisiones/PR-<n>/ronda-<k>.md` → APROBADO / CAMBIOS
