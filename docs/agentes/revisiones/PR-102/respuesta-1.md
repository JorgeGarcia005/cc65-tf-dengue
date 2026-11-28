# Respuesta — PR-102 · Ronda 1

- **Implementador:** Claude Code (Claude Opus 5.5)
- **Veredicto respondido:** `ronda-1.md` (CAMBIOS)

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `d2247fe` — la época se lee con `Number.parseInt` sobre `[data-kpi="epoca"]` («71 / 600» → 71) y además se exige `[data-kpi="perdida"]` ≠ «—» antes de `e2e-08`. Tras tumbar, se espera «salió del clúster» **y** después «reasignacion:» o un «se registró» nuevo (conteo antes/después) antes de `e2e-09`. Corrida local: `e2e-08` muestra «71 / 600» y pérdida 0.222621; `e2e-09` muestra el corte en la época 80, «reasignacion: el shard 2 pasa a nodo-3» y el reingreso. |
| H2 | BLOQUEANTE | ACEPTO | `d41ac86` — cada bloque registra el caso como no cumplido (`no se pudo probar: …`) si la petición auxiliar falla; la CSP exige `200`; `ticket` exige `200`. `TestCasosContraLaAPI` ahora usa una SPA simulada con CSP y exige **22** casos exactos (`SinLimite`); `TestUnCasoNoDesaparece`: SPA inalcanzable, sin CSP o 404 → 22 casos y `todos_ok=false`. |
| H3 | MAYOR | ACEPTO | `d2247fe` — con `E2E_EXIGIR_VERSION=1`, `version_modelo` de `GET /healthz` (fixture `request`) y `expect([data-kpi="version"]).toHaveText(servida)`, luego `toHaveCount(0)` del aviso. Corrida local con el gold sintético: falla con `Expected "97ec3d4f7151" / Received "655c9dc770fa"`, como debe. |
| H4 | MENOR | ACEPTO | `ffc2051` — `tsconfig.json` incluye `e2e` y `playwright.config.ts` (`tsc --listFilesOnly` los lista; el CI corre `npx tsc -b`). `d41ac86` — correos obligatorios en `cmd/seguridad`; `NuevoFirmadorConReloj(…, nil)` devuelve error (`TestFirmadorConRelojNil`). |

## Verificación

```text
$ go test -race ./cmd/seguridad ./internal/auth
ok  	…/cmd/seguridad
ok  	…/internal/auth
$ cd web && npx tsc -b && npx vitest run
Tests  186 passed (186)
$ npx playwright test   # stack local: coordinador -api, 3 nodos, cmd/web, gold sintético
8 passed; swagger ✘ (cdn.jsdelivr.net bloqueado por el proxy de este entorno)
$ E2E_EXIGIR_VERSION=1 npx playwright test -g inicio
✘ Expected "97ec3d4f7151" Received "655c9dc770fa"   # esperado: el gold sintético no es el evaluado
```
