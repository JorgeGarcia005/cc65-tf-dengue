# Veredicto — PR-102 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/101-e2e-seguridad` @ `ca6c46e`
- **Fecha:** 2026-11-28
- **Decisión:** CAMBIOS

El PR resuelve con buen diseño general el issue `#101` (pasos 13 y 14 de `docs/guias/jose-70.md`): `cmd/seguridad` usa exclusivamente la biblioteca estándar más `internal/auth` (`AGENTS.md` §3.2), ejercita los controles de ADR-0007 y espera la liberación de la ventana del *rate limit* antes de terminar; la suite de Playwright recorre los flujos por rol sin ensuciar Vitest (`*.e2e.ts`) y genera las 12 capturas con nombre fijo; y tanto `gofmt`, `go vet`, `go test -race ./...` (26 suites, 0 carreras), `npx tsc -b` + `npx vitest run` (186 tests) como los 7 trabajos del CI de GitHub Actions pasan en verde.

Sin embargo, antes de que José ejecute la guía en `deez`, hay **dos hallazgos BLOQUEANTES** que afectan directamente la validez de las evidencias y capturas del informe TB2 (**H1** y **H2**), además de un hallazgo **MAYOR** (**H3**) y uno **MENOR** (**H4**).

## Verificación automática

```text
$ git rev-parse --short HEAD
ca6c46e

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	12.336s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	10.007s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.595s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.032s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.745s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.286s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	2.110s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.432s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/seguridad	26.507s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	5.823s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.089s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	6.188s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.125s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	39.536s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	19.520s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	12.503s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.711s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.226s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.052s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.143s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	10.864s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.055s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	15.123s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.796s

$ cd web && npm ci --no-audit --no-fund && npx tsc -b && npx vitest run
Test Files  12 passed (12)
     Tests  186 passed (186)

$ gh pr checks 102 -R JorgeGarcia005/cc65-tf-dengue
Docker (imagen, clúster y caída de un nodo)	pass	2m1s
Ecosistema de agentes                       	pass	5s
Go (gofmt, vet, test -race)                 	pass	1m35s
MongoDB y Redis (tests de integración)      	pass	56s
OpenAPI 3 válido                            	pass	22s
SPA (tipos, tests, build e imagen)          	pass	48s
Spin (modelo del protocolo y mutantes)      	pass	1m48s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (suites `internal/ml`, `internal/cluster`, `cmd/coordinator` y job Docker del CI en verde) |
| Latencia API p95 < 100 ms | ✅ (`cmd/benchapi` con JWT en verde; medición final en `deez` en paso 8 de `#70`) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni arquitectura. |
| Implementación | 8 | en riesgo hasta corregir H2 | `cmd/seguridad` automatiza con stdlib la verificación externa de ADR-0007, pero H2 permite dar `todos_ok: true` omitiendo casos cuando falla una petición auxiliar. |
| Interfaz | 5 | en riesgo hasta corregir H1 y H3 | Playwright cubre los roles, el modo demo y responsive, pero H1 toma `e2e-08-admin-entrenando.png` en la época 0 y H3 introduce una carrera asíncrona al validar la versión en Inicio. |
| Informe | 2 | en riesgo hasta corregir H1 y H2 | `reports/seguridad.json` y las capturas `e2e-08` / `e2e-09` se incrustan directamente en el informe final TB2 (`#70` / `#71`). |
| Video | 2 | no aplica | No afecta directamente al video. |

## Hallazgos

### H1 — El sondeo de época en Playwright concatena `"0 / 600"` → `600` y captura `e2e-08-admin-entrenando.png` en la época 0 · BLOQUEANTE

- **Dónde:** `web/e2e/vigia.e2e.ts:137-139, 145-147` y `web/src/paginas/Admin.tsx:348-352`
- **Evidencia:** En `Admin.tsx:348-352`, `<dd data-kpi="epoca">` renderiza `{estado.epoca}{entrenamiento ? \` / ${entrenamiento.epocas}\` : ''}`. Apenas `POST /api/v1/entrenamientos` responde `202` con `epocas = 600` (`Admin.tsx:232`), el texto del `<dd>` pasa a `"0 / 600"`. En `vigia.e2e.ts:138`, la condición de espera hace:
  ```ts
  await expect.poll(async () => Number((await epoca.textContent())?.replace(/\D/g, '') || 0), { timeout: 60_000 }).toBeGreaterThan(40)
  ```
  Al quitar todos los no-dígitos, `"0 / 600"` se convierte en `"0600"` y se evalúa como `600`:
  ```text
  $ node -e 'const txt = "0 / 600"; console.log("texto DOM:", JSON.stringify(txt), "-> replace(/\\D/g,\"\"):", Number(txt.replace(/\D/g, "") || 0), "-> parseInt:", Number.parseInt(txt, 10) || 0)'
  texto DOM: "0 / 600" -> replace(/\D/g,""): 600 -> parseInt: 0
  ```
  Por ello, `.toBeGreaterThan(40)` se cumple instantáneamente en la **época 0**, provocando que:
  1. `e2e-08-admin-entrenando.png` (`:139`) se capture antes de que corran las épocas, con `Época: 0 / 600`, `Pérdida: —` y el gráfico `Pérdida por época` vacío, contradiciendo lo requerido por `docs/guias/jose-70.md:105` (*«Panel de administración durante un entrenamiento (época, pérdida, nodos sanos, p95 de la API)»*).
  2. Se pulse `Tumbar` (`:145`) de inmediato en el arranque en vez de con el entrenamiento avanzado.
  3. Además, en `:146-147` se toma `e2e-09-admin-nodo-caido.png` en cuanto aparece `'salió del clúster'` (`EvDescarte`), sin esperar al frame siguiente de `EvReasignacion` (`reasignacion:`) o `EvRegistro` (`se registró`), pese a que `docs/guias/jose-70.md:106` exige mostrar *«la caída y el reingreso (o la reasignación de su shard)»*.
- **Corrección sugerida:** Parsear únicamente el entero previo a `'/'` sobre `[data-kpi="epoca"]` (p. ej. `Number.parseInt((await epoca.textContent()) ?? '', 10) || 0`), comprobar también que `[data-kpi="perdida"]` ya no sea `'—'` antes de capturar `e2e-08-admin-entrenando.png`, y antes de capturar `e2e-09-admin-nodo-caido.png` esperar a que `page.getByRole('log')` contenga además `/reasignacion:|se registró/`.

### H2 — `cmd/seguridad` omite casos en silencio cuando falla una petición auxiliar y puede escribir `todos_ok: true` sin probarlos · BLOQUEANTE

- **Dónde:** `cmd/seguridad/main.go:208-248` y `cmd/seguridad/main_test.go:66-84`
- **Evidencia:** En `(*Prueba).Correr`, seis bloques envuelven el registro de `p.caso(...)` en `if ..., err := ...; err == nil` sin rama `else`:
  - `CORS: origen de la SPA` (`main.go:208`) y `CORS: origen ajeno` (`:214`)
  - `cabecera nosniff` (`:221`) y `login sin caché` (`:225`)
  - `CSP de la SPA` (`:231`)
  - `WS de un origen ajeno` (`:239`), `WS con ticket válido` (`:246`) y `WS: ticket reutilizado` (`:247`), condicionados a `if t, err := p.ticket(admin); err == nil` (`:239, :243`).

  A diferencia de `esperarCodigo` (donde un error de red produce código `0` y `ok: false`), en estos bloques cualquier `err != nil` descarta el caso sin agregarlo a `p.casos`, por lo que `p.reporte()` (`:286-292`) calcula `TodosOK = true` sobre los casos restantes y el binario termina con código `0`. En concreto:
  1. Si el contenedor `web` (`http://localhost:8081`) está caído o en otro puerto cuando se corre `scripts/evidencias-tb2.sh seguridad`, `GET http://localhost:8081/` falla con `connection refused`, `"CSP de la SPA"` se omite, `reports/seguridad.json` se escribe con 23 casos y `"todos_ok": true`, y `scripts/paso70.sh:89` (`grep -q '"todos_ok": *true'`) da por válido el paso 13 sin haber probado la CSP.
  2. Si `POST /api/v1/ws/ticket` falla o devuelve distinto de `200`, `p.ticket(admin)` retorna `err != nil` y los 3 casos con ticket del WebSocket desaparecen del reporte dejando `"todos_ok": true`.
  3. En `CSP de la SPA` (`:231-235`) tampoco se comprueba `res.StatusCode == 200` (`cmd/web/main.go:115` añade la cabecera `Content-Security-Policy` incluso cuando responde `404` o `405`).
  4. En `cmd/seguridad/main_test.go:68-75`, `TestCasosContraLaAPI` deja `p.Web == ""` y solo comprueba `len(r.Casos) < 20`, por lo que no ejercita `"CSP de la SPA"` ni detecta casos omitidos.
- **Corrección sugerida:** Registrar siempre los casos previstos con `ok = false` y el detalle del error/estado en `obtenido` cuando `err != nil`, exigir `res.StatusCode == 200` junto con la directiva CSP en `"CSP de la SPA"`, y ampliar `cmd/seguridad/main_test.go` con un servidor de prueba para `p.Web` verificando tanto el conteo exacto de casos cuando todo pasa (`22` con `SinLimite: true`) como que un `p.Web` inalcanzable o sin CSP deja `r.TodosOK == false`.

### H3 — Carrera asíncrona en `E2E_EXIGIR_VERSION=1` puede dar por buena una versión desfasada en Inicio · MAYOR

- **Dónde:** `web/e2e/vigia.e2e.ts:60-68` y `web/src/paginas/Inicio.tsx:23-36, 69, 80-84`
- **Evidencia:** En `Inicio.tsx:23-36`, `GET /api/v1/evaluacion` (`setDoc`) y `GET /api/v1/modelo` (`setModelo`, con `.catch(() => undefined)`) se disparan en paralelo. La tabla `Las 10 prioridades` se pinta en cuanto resuelve `/api/v1/evaluacion` (`Inicio.tsx:50`), mientras que el aviso `"Esta evaluación es del modelo …"` (`Inicio.tsx:69, 80-84`) solo existe cuando `modelo !== null`. En `vigia.e2e.ts:63-67`:
  ```ts
  await expect(tabla.getByRole('row')).toHaveCount(11)
  if (process.env.E2E_EXIGIR_VERSION === '1') {
    await expect(page.getByText(/Esta evaluación es del modelo/)).toHaveCount(0)
  }
  ```
  En Playwright, `toHaveCount(0)` se resuelve con éxito en la primera comprobación en que haya 0 coincidencias. Si `/api/v1/evaluacion` responde unos milisegundos antes que `/api/v1/modelo` (por ejemplo con `304 Not Modified` gracias a `ETag`) o si `/api/v1/modelo` falla, `modelo` sigue siendo `null` cuando aparece la tabla y `toHaveCount(0)` aprueba en falso sin haber contrastado las versiones, debilitando el control descrito en `docs/guias/jose-70.md:94`.
- **Corrección sugerida:** Con `E2E_EXIGIR_VERSION === '1'`, leer `version_modelo` de `GET ${API}/healthz` (mediante `request.get` de Playwright o esperando la respuesta `200` de `/api/v1/modelo`) y comprobar positivamente `await expect(page.locator('[data-kpi="version"]')).toHaveText(versionServida)`, manteniendo después `toHaveCount(0)` sobre el aviso de desfase.

### H4 — `npm run typecheck` no cubre `web/e2e/` ni `playwright.config.ts`, y validaciones defensivas menores · MENOR

- **Dónde:** `web/tsconfig.json:18`, `web/package.json:10`, `cmd/seguridad/main.go:71-73` e `internal/auth/jwt.go:88-95`
- **Evidencia:**
  1. `web/tsconfig.json:18` declara `"include": ["src", "vite.config.ts"]`, de modo que `npx tsc -b` (en `.github/workflows/ci.yml:120` y en `scripts/evidencias-tb2.sh:205`) no verifica los tipos de `web/playwright.config.ts` ni de `web/e2e/vigia.e2e.ts`.
  2. En `cmd/seguridad/main.go:71`, la guardia inicial comprueba `cred.Secreto == "" || cred.AdminClave == "" || cred.AnalistaClave == ""` pero no `cred.AdminCorreo == "" || cred.AnalistaCorreo == ""`, aunque el mensaje de la línea 72 indica `faltan JWT_SECRETO, VIGIA_ADMIN_* o VIGIA_ANALISTA_*`.
  3. En `internal/auth/jwt.go:88-95`, `NuevoFirmadorConReloj` asigna `f.ahora = ahora` sin validar `ahora == nil` (lo que causaría pánico en `Firmar` o `Verificar`).
- **Corrección sugerida:** Agregar un `web/tsconfig.e2e.json` (`"include": ["e2e", "playwright.config.ts"]`, `"types": ["node"]`) e invocarlo en `typecheck` (o como referencia de proyecto), validar ambos correos en `cmd/seguridad/main.go:71` y tratar `ahora == nil` en `NuevoFirmadorConReloj` (error o `time.Now`).

## Seguimiento de rondas previas

Primera ronda; no aplica.

## Cómo explicarlo en la sustentación

La descripción del PR incluye una explicación clara y concisa para la sustentación: los controles de seguridad de ADR-0007 (JWT alterado, vencido o de otro secreto, RBAC, lista blanca de CORS, CSP estricta, tickets de un solo uso para WebSocket y límites 413/429) se verifican con un cliente externo en Go (`cmd/seguridad`) que deja cada caso esperado vs. obtenido en `reports/seguridad.json`, mientras que Playwright recorre automáticamente los flujos de analista y administrador contra el clúster desplegado y genera con nombre fijo las 12 capturas del informe.
