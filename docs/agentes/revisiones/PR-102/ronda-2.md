# Veredicto — PR-102 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/101-e2e-seguridad` @ `1a1e48b`
- **Fecha:** 2026-11-28
- **Decisión:** APROBADO

Los **4 hallazgos** de la ronda 1 (2 bloqueantes, 1 mayor y 1 menor) quedaron íntegramente resueltos, con pruebas automatizadas y los 7 trabajos del CI de GitHub Actions en verde:

- **Sondeo real de épocas y recuperación de nodo en Playwright (H1):** `web/e2e/vigia.e2e.ts:147-150` parsea únicamente el entero previo a `'/'` sobre `[data-kpi="epoca"]` (`Number.parseInt((await epoca.textContent()) ?? '', 10) || 0 > 40`) y exige además que `[data-kpi="perdida"]` ya no sea `'—'` antes de capturar `e2e-08-admin-entrenando.png`. Luego (`vigia.e2e.ts:156-165`), cuenta las apariciones previas de `'se registró'` en la bitácora, pulsa `Tumbar`, espera `'salió del clúster'` y aguarda la recuperación efectiva (`'reasignacion:'` o un `'se registró'` nuevo) antes de tomar `e2e-09-admin-nodo-caido.png`.
- **Ningún caso de seguridad desaparece del reporte (H2):** `cmd/seguridad/main.go:142-146, 215-271, 341-356` incorpora `(*Prueba).fallo` para registrar como no cumplido (`ok: false`, `"no se pudo probar: …"`) cualquier caso cuya petición auxiliar falle, exige `res.StatusCode == 200` en `"CSP de la SPA"` y en `(*Prueba).ticket`, y valida el conteo exacto de casos (`22` con `SinLimite: true`, `24` en corrida completa) tanto en éxito (`TestCasosContraLaAPI`, `cmd/seguridad/main_test.go:86-104`) como ante fallos de la SPA (`TestUnCasoNoDesaparece`, `main_test.go:122-141`).
- **Comparación positiva de versión en Inicio con `E2E_EXIGIR_VERSION=1` (H3):** `web/e2e/vigia.e2e.ts:60-76` consulta `version_modelo` en `GET ${API}/healthz` (validando `/^[0-9a-f]{12}$/`), comprueba positivamente `expect(page.locator('[data-kpi="version"]')).toHaveText(servida)` y mantiene `toHaveCount(0)` sobre el aviso de desfase, eliminando la carrera entre `/api/v1/evaluacion` y `/api/v1/modelo`.
- **Cobertura de tipos sobre `e2e/` y validaciones defensivas (H4):** `web/tsconfig.json:18` incluye `"e2e"` y `"playwright.config.ts"` en `npx tsc -b` (sin afectar el *build* de Vite ni la imagen Docker), `cmd/seguridad/main.go:71` exige también `AdminCorreo` y `AnalistaCorreo`, e `internal/auth/jwt.go:88-91` rechaza `ahora == nil` en `NuevoFirmadorConReloj` (`TestFirmadorConRelojNil`, `internal/auth/jwt_test.go:156-160`).

## Verificación automática

```text
$ git rev-parse --short HEAD
1a1e48b

$ git diff --check origin/integracion/tb2...origin/feature/101-e2e-seguridad
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && bash -n scripts/evidencias-tb2.sh scripts/paso70.sh
skills: ok
bash syntax: ok

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.725s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.782s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.498s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.720s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.273s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.742s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.413s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/seguridad	57.592s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.260s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	4.329s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.102s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	27.184s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.259s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.360s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.906s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.039s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.295s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.082s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.469s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	12.734s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.794s

$ cd web && npm ci --no-audit --no-fund && npx tsc -b && npx tsc --listFilesOnly | grep -E "e2e|playwright" && npx vitest run
.../web/e2e/vigia.e2e.ts
.../web/playwright.config.ts
Test Files  12 passed (12)
     Tests  186 passed (186)

$ gh pr checks 102 -R JorgeGarcia005/cc65-tf-dengue
Docker (imagen, clúster y caída de un nodo)	pass	1m57s
Ecosistema de agentes                       	pass	6s
Go (gofmt, vet, test -race)                 	pass	1m29s
MongoDB y Redis (tests de integración)      	pass	53s
OpenAPI 3 válido                            	pass	20s
SPA (tipos, tests, build e imagen)          	pass	54s
Spin (modelo del protocolo y mutantes)      	pass	1m59s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`27/27` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (suites `internal/ml`, `internal/cluster`, `cmd/coordinator` y job Docker del CI en verde) |
| Latencia API p95 < 100 ms | ✅ (`cmd/benchapi` con JWT en verde; medición final en `deez` en el paso 8 de `#70`) |
| SPA (`tsc -b` sobre `src` + `e2e` + `playwright.config.ts`, Vitest) | ✅ (`186/186` tests en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni arquitectura. |
| Implementación | 8 | 8.0 / 8 (controles ADR-0007 verificados externamente) | `cmd/seguridad` prueba 24 casos reales contra la API y la SPA usando solo stdlib + `internal/auth`, sin casos omitidos en silencio y esperando la liberación del *rate limit*. |
| Interfaz | 5 | 5.0 / 5 (pruebas E2E por rol y demostración en vivo) | Playwright recorre los flujos de analista y administrador, valida la versión servida, espera épocas reales y recuperación de nodo tras caída, verifica el sello bit a bit y comprueba responsive a 375 px. |
| Informe | 2 | pasos 13 y 14 de `#70` desbloqueados | `scripts/evidencias-tb2.sh seguridad\|e2e`, `scripts/paso70.sh` y `docs/guias/jose-70.md` quedan listos para generar `reports/seguridad.json` y las 12 capturas `e2e-*.png` en `deez`. |
| Video | 2 | no aplica | No afecta directamente al video. |

## Hallazgos

Sin hallazgos abiertos en esta ronda.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `d2247fe`: `web/e2e/vigia.e2e.ts:147-150` lee la época con `Number.parseInt` sobre `[data-kpi="epoca"]` y exige `[data-kpi="perdida"]` ≠ `'—'` antes de `e2e-08`; en `:156-165` espera `'salió del clúster'` y después `'reasignacion:'` o un `'se registró'` adicional antes de `e2e-09`. |
| H2 (ronda 1 · BLOQUEANTE) | RESUELTO | `d41ac86`: `cmd/seguridad/main.go:142-146, 215-271, 341-356` registra todo fallo auxiliar con `p.fallo` y exige `200` en `"CSP de la SPA"` y en `ticket`. Verificado con `TestCasosContraLaAPI` (`22` casos exactos) y `TestUnCasoNoDesaparece` (`cmd/seguridad/main_test.go:86-104, 122-141`). |
| H3 (ronda 1 · MAYOR) | RESUELTO | `d2247fe`: `web/e2e/vigia.e2e.ts:60-76` contrasta positivamente `[data-kpi="version"]` contra `version_modelo` de `GET ${API}/healthz` cuando `E2E_EXIGIR_VERSION=1`, además de comprobar la ausencia del aviso de desfase. |
| H4 (ronda 1 · MENOR) | RESUELTO | `ffc2051` y `d41ac86`: `web/tsconfig.json:18` incluye `e2e` y `playwright.config.ts` en `tsc -b`; `cmd/seguridad/main.go:71` exige los dos correos; `internal/auth/jwt.go:88-91` devuelve error si `ahora == nil` (`TestFirmadorConRelojNil`). |

## Cómo explicarlo en la sustentación

Los controles de seguridad de ADR-0007 (JWT ausente, mal formado, alterado, vencido o de otro secreto, autorización por rol, lista blanca de CORS, CSP estricta sin `unsafe-*`, tickets de un solo uso para WebSocket y límites 413/429 con liberación tras `Retry-After`) se prueban desde fuera con un cliente en Go stdlib (`cmd/seguridad`) que registra los 24 casos esperado vs. obtenido en `reports/seguridad.json` sin descartar ninguno ante fallos de red. En paralelo, Playwright automatiza 9 flujos de punta a punta por rol contra el sistema desplegado —incluidos el entrenamiento en vivo, la caída y recuperación de un nodo, el sello de equivalencia bit a bit y el diseño responsive a 375 px— dejando las 12 capturas con nombre fijo para el informe TB2.
