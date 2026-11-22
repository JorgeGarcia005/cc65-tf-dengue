# Veredicto — PR-85 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/67-inicio-prediccion` @ `2272d03` (base: `integracion/tb2` @ `00e1400`)
- **Fecha:** 2026-11-22
- **Decisión:** APROBADO

Los tres hallazgos de la ronda 1 (**H1**, **H2** y **H3**) quedaron resueltos con evidencia reproducible en los commits `d1b1960`, `0849eef` y `d8a468d`, sin introducir defectos, carreras ni cifras escritas a mano (`AGENTS.md` §3.2, §3.6): `esPrediccion`, `esModelo` y `esEvaluacion` validan la forma de todas las respuestas `200` antes de pintar (mostrando `role="alert"` ante HTML de proxy o JSON incompleto en vez de dejar la SPA en blanco) y `proporcion()` elimina las divisiones sin guarda (**H1**); las tablas de `Inicio` viven dentro de `.tabla-envoltura`, el grid usa `min-width: 0` + `minmax(min(420px, 100%), 1fr)` y los campos del formulario llevan `width: 100%; min-width: 0`, eliminando por completo los desbordes horizontales en `1280px`, `768px` y `375px` tanto en `Inicio` como en `Predicción` (**H2**); y se resolvieron los ajustes de higiene en Go y SPA: cero apariciones de `"aporte":-0` en 100 consultas reales del modelo de 31 variables (`2 805` bytes/resp.), `GOARCH=arm64` sin instrucciones FMA en `(*Modelo).prediccion` ni en `TestPrediccionCoincideConElModeloEntrenado`, eliminación de `sigmoide` y del parámetro sin efecto `departamentos`, `role="radiogroup"` con borde de error en `<fieldset aria-invalid>`, texto exacto del conteo de variables con aporte 0 y fragmento de `docs/informe/pc4/10-api.md` regenerado (**H3**). Toda la verificación local (`gofmt`, `go vet`, `-tags integracion`, 25/25 paquetes Go con `-race` y 0 carreras, 72/72 tests de Vitest, `tsc -b`, `vite build` sin `url(data:)`, auditoría en vivo en Chrome headless con CSP estricta y `cmd/benchapi` con `p95 = 4,06 ms` fría / `4,82 ms` caliente) y los 7/7 jobs de GitHub Actions sobre `2272d03` están en verde. Se deja únicamente una observación **MENOR** no bloqueante (**H1-R2**) sobre el ancho mínimo de la primera columna de `.contribucion` en móvil estrecho (`375px`), opcional para pulir en #70.

## Verificación automática

```text
$ go version
go version go1.24.3 linux/amd64

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go list -f '{{.Imports}}' ./internal/api
[bytes compress/gzip context crypto/rand crypto/sha256 encoding/base64 encoding/hex encoding/json errors fmt github.com/JorgeGarcia005/cc65-tf-dengue/docs/api github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster github.com/JorgeGarcia005/cc65-tf-dengue/internal/features github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws io log math net net/http net/url runtime/debug slices strconv strings sync sync/atomic time unicode/utf8]

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.764s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.607s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.446s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.030s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.592s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.256s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.216s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.412s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.447s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.049s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.146s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.027s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.043s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.391s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	13.040s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.406s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.791s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.047s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.942s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.610s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.784s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.751s

$ go test -race -count=1 -run 'TestPrediccionContribucionesSumanElLogit|TestCacheViejaSinContribucionesSeRecalcula|TestOpenAPI|TestPrediccionCoincideConElModeloEntrenado' -v ./internal/api
PASS (6/6: TestPrediccionCoincideConElModeloEntrenado, TestOpenAPICubreLasRutas, TestOpenAPIMapasEnLineaBienFormados, TestPrediccionContribucionesSumanElLogit, TestCacheViejaSinContribucionesSeRecalcula, TestOpenAPISeguridad)

$ go test -race -count=1 -run Fragmentos -v ./cmd/informe
--- PASS: TestFragmentosSalenDelFuente (0.17s)

$ GOARCH=arm64 go test -c -o /tmp/api_arm64.test ./internal/api && go tool objdump -s 'TestPrediccionCoincideConElModeloEntrenado|prediccion' /tmp/api_arm64.test | grep -E '^(TEXT|.*FMADD|.*FMSUB|.*FNMADD|.*FNMSUB)'
TEXT github.com/JorgeGarcia005/cc65-tf-dengue/internal/api.(*Modelo).prediccion(SB) .../internal/api/modelo.go
TEXT github.com/JorgeGarcia005/cc65-tf-dengue/internal/api.TestPrediccionCoincideConElModeloEntrenado(SB) .../internal/api/api_test.go
(0 instrucciones FMA en ambas funciones)

$ (en web/) npm ci --silent && npx tsc -b && npx vitest run && npm run build && ! grep -l 'url(data:' dist/assets/*.css
Test Files  5 passed (5)
     Tests  72 passed (72)
✓ built in 3.64s
NO url(data: OK

$ (en web/) npm test -- inicio && npm test -- prediccion
✓ src/pruebas/inicio.test.tsx (12 tests)
✓ src/pruebas/prediccion.test.tsx (14 tests)

$ grep -rnE '[0-9]+[.,][0-9]{2,}' web/src --include='*.ts*' | grep -v pruebas
(vacío = ok)

$ rg -n 'sigmoide' web/src/
(vacío = ok)

$ <auditoría en vivo: cmd/coordinator con modelo real de 31 variables (reports/entrenamiento.json), reports/evaluacion.json, JWT y CORS + cmd/web con CSP estricta>
- 100 consultas variadas contra el modelo real de 31 variables: negZero=0 (0 apariciones de ":-0"), sumMismatch=0 (100/100 exactas en JS), 2805 bytes/resp.
- go run ./cmd/benchapi -url http://127.0.0.1:18180 -token <jwt> -n 1000 -c 16 -exigir-p95 100ms:
  fria:     1000 pet., 0 err., 8020 RPS, p50=1.49 ms, p95=4.06 ms, p99=5.77 ms  (p95 < 100 ms: true)
  caliente: 1000 pet., 0 err., 8029 RPS, p50=1.35 ms, p95=4.82 ms, p99=7.19 ms  (p95 < 100 ms: true)
- Chrome headless contra cmd/web (style-src 'self'): 0 errores de consola, 0 violaciones CSP.
  Inicio:
    @ 1280px: docScrollW=1265 === docClientW=1265, badTarjetas=0, tablas 1141/1141 y 525/525
    @  768px: docScrollW=753  === docClientW=753,  badTarjetas=0, tablas 629/629 y 629/629 (1 columna, sin scroll)
    @  375px: docScrollW=375  === docClientW=375,  badTarjetas=0, tablas desplazables dentro de .tabla-envoltura (624/251 y 366/251)
    10 prioridades exactas, enlace #1 → /mapa?depto=PIURA&semana=20&etapa=adulto_30_59&metrica=esperados
  Predicción (errores de validación):
    fieldset[role="radiogroup"][aria-invalid="true"]: border 2px solid rgb(179, 18, 63), padding-left 10px
    @ 1280px, 768px, 375px: docScrollW === docClientW, badTarjetas=0
  Predicción (resultado MADRE DE DIOS, 65 A, M, sem 14, 2024):
    riesgo 19.0%, «✓ cuadra», «Modelo 585644bd3f51 · 24 variables con aporte 0 no se muestran»
    @ 1280px: docScrollW=1265 === docClientW=1265, badTarjetas=0 (.contribucion: nombre 213px, eje 227px, valor 64px)
    @  768px: docScrollW=753  === docClientW=753,  badTarjetas=0 (.contribucion: nombre 264px, eje 281px, valor 64px)
    @  375px: docScrollW=375  === docClientW=375,  badTarjetas=0 (.contribucion: nombre 190px, eje 1px, valor 64px; ver H1-R2)

$ gh pr checks 85   # run 44996827447 sobre 2272d03
Docker (imagen, clúster y caída de un nodo)	pass	1m58s
Ecosistema de agentes                       	pass	8s
Go (gofmt, vet, test -race)                 	pass	1m3s
MongoDB y Redis (tests de integración)      	pass	56s
OpenAPI 3 válido                            	pass	23s
SPA (tipos, tests, build e imagen)          	pass	52s
Spin (modelo del protocolo y mutantes)      	pass	2m6s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (25/25 paquetes con tests PASS, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; suite completa en verde) |
| Latencia API p95 < 100 ms | ✅ (`4,06 ms` fría / `4,82 ms` caliente con modelo real de 31 variables, JWT y contribuciones; `2 805` bytes/resp.) |
| SPA (`tsc -b`, Vitest, `vite build`, sin `url(data:)`, CSP en Chrome) | ✅ (72/72 tests PASS, 0 errores consola/CSP, sin desbordes a `1280px`/`768px`/`375px`) |
| CI GitHub Actions (PR #85 @ `2272d03`) | ✅ (7/7 jobs PASS, run `44996827447`) |

## Verificación contra los criterios del issue #67

| Criterio (#67) | Estado | Evidencia |
|---|---|---|
| 1. Inicio: top 10 en el mismo orden que `evaluacion.json` y KPIs del modelo con su versión (`npm test -- inicio`) | ✅ | `web/src/paginas/Inicio.tsx`, `web/src/inicio/datos.ts`; `inicio.test.tsx` (12/12 PASS con `reports/evaluacion.json` real), aviso si el modelo publicado difiere, validación de forma `esEvaluacion` y `proporcion()` sin `NaN`, sin cifras escritas a mano. |
| 2. Predicción: validación del formulario, probabilidad calibrada, contribución por variable (suman el logit), latencia y acierto de caché (`npm test -- prediccion`) | ✅ | `web/src/paginas/Prediccion.tsx`, `web/src/prediccion/datos.ts`, `internal/api/modelo.go`, `internal/api/api.go`; `prediccion.test.tsx` (14/14 PASS), `TestPrediccionContribucionesSumanElLogit` y `TestCacheViejaSinContribucionesSeRecalcula` PASS, 100/100 sumas exactas y 0 `-0` en vivo con el modelo real de 31 variables. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | La portada responde directamente la pregunta de impacto ODS 3 (dónde, cuándo y a qué grupo de edad reforzar) con cifras reales de `reports/evaluacion.json`, sin cifras escritas a mano. |
| Implementación | 8 | 8.0 / 8 | `POST /api/v1/predicciones` devuelve contribuciones exactas bit a bit sin FMA ni ceros negativos (`-0`), invalida caché vieja sin contribuciones, conserva `p95 < 5 ms` y mantiene el núcleo 100 % stdlib (`AGENTS.md` §3.2). |
| Interfaz | 5 | 5.0 / 5 | Inicio y Predicción claros, accesibles (`role="radiogroup"`, `aria-invalid`, `aria-live`), resilientes ante respuestas `200` malformadas, sin desbordes horizontales en escritorio/tableta/móvil y con 0 violaciones de CSP estricta. |
| Informe | 2 | 2.0 / 2 | `docs/informe/pc4/10-api.md` sincronizado con `(*Modelo).prediccion` (`TestFragmentosSalenDelFuente` en verde) y `docs/api/openapi.yaml` validado en CI. |
| Video | 2 | — | No corresponde a este PR (queda listo el flujo completo Inicio → Mapa → Predicción para grabarlo). |

## Hallazgos

Sin hallazgos **BLOQUEANTES** ni **MAYORES**.

### H1-R2 — Barra de `.contribucion` comprimida a `1px` en móvil estrecho (`375px`) · MENOR *(no bloqueante)*

- **Dónde:** `web/src/estilos/vigia.css:723-729`
- **Evidencia:** Medido en Chrome headless sobre el resultado de `MADRE DE DIOS` a `375px`: la página y la tarjeta no desbordan (`docScrollW = 375 === docClientW = 375`, `badTarjetas = 0`, `valorRightVsBorder = -10px`), pero como el ancho interior de `.tarjeta` es `251px` y `.contribucion` fija `grid-template-columns: minmax(190px, 1.5fr) 1.6fr 64px` con `gap: 10px` (`190 + 64 + 20 = 274px`), la pista central `.contribucion-eje` queda en `1px` y `.contribucion-valor` invade `24px` del relleno derecho de la tarjeta (en `1280px` y `768px` el alineamiento es exacto: `ejeW = 227px` y `281px`, `valorRightVsContent = 0`).
- **Corrección sugerida:** No requiere una ronda extra en este PR; en la pasada de pulido/capturas de TB2 (#70), reducir el mínimo de la primera columna en pantallas `< 480px` (p. ej. `@media (max-width: 480px) { .contribucion { grid-template-columns: minmax(110px, 1fr) 1fr 58px; gap: 6px; } }` o apilar el nombre en una línea arriba) para que la barra siga visible también en teléfonos de `375px`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) | RESUELTO | `d1b1960` agrega `esPrediccion`, `esModelo` (`web/src/prediccion/datos.ts:88-106`), `esEvaluacion` y `proporcion()` (`web/src/inicio/datos.ts:36-49`), valida `typeof m?.version === 'string'` en `Inicio.tsx:35` y añade 10 pruebas en `inicio.test.tsx` y `prediccion.test.tsx` (HTML de proxy, JSON sin `contribuciones`, `logit` no numérico, `200 {}` en `/api/v1/modelo`, sin `metricas_test`, `prioridades: null`, `version_esquema: 2` y denominadores en `0`). |
| H2 (ronda 1) | RESUELTO | `0849eef` envuelve ambas tablas de `Inicio.tsx` en `.tabla-envoltura`, añade `min-width: 0` en los grids, sube el corte a `minmax(min(420px, 100%), 1fr)` y fija `width: 100%; min-width: 0` en los campos del formulario. Verificado en Chrome headless a `1280px`, `768px` y `375px`: `docScrollW === docClientW` y `0` tarjetas desbordadas en `Inicio` y `Predicción`. |
| H3 (ronda 1) | RESUELTO | `d8a468d` normaliza `if aporte == 0 { aporte = 0 }` en `internal/api/modelo.go:249-251` (verificado: `0` apariciones de `:-0` en 100 consultas reales, `TestFragmentosSalenDelFuente` en verde) y elimina FMA en `internal/api/api_test.go:161` (verificado con `GOARCH=arm64` + `objdump`: `0` instrucciones `FMADD`/`FMSUB`). `d1b1960` y `0849eef` retiran `sigmoide` y el parámetro sin uso `departamentos`, añaden `role="radiogroup"` + `fieldset[aria-invalid='true']` (verificado en Vitest y Chrome: borde `2px solid rgb(179, 18, 63)`) y corrigen el texto del conteo de variables en `0`. La regeneración de `reports/bench_api_jorge-pc_jwt.json` queda anotada para la corrida final de evidencias (#70/#71). |

## Cómo explicarlo en la sustentación

> La vista de **Inicio** responde la pregunta de impacto ODS 3 sin recalcular cifras en el navegador: consume `GET /api/v1/evaluacion`, valida la estructura del JSON recibido y muestra en el orden exacto de `reports/evaluacion.json` las 10 combinaciones departamento × semana × grupo de edad con más casos esperados con signos de alarma (enlazadas al mapa ya filtrado), las 5 semanas pico nacionales, los grupos de edad por riesgo y los KPIs del modelo con aviso si la API publica otra versión. La vista de **Predicción** valida con las mismas reglas que la API, llama a `POST /api/v1/predicciones` midiendo la latencia y leyendo `X-Cache` (expuesta por CORS), e ilustra el aporte `peso × valor` de cada variable activa verificando en el navegador que `sesgo + Σ aportes` reproduce bit a bit el `logit` calculado en Go (con redondeo explícito `float64(m.Pesos[j] * v)` sin FMA y normalización de `-0`, manteniendo `p95 < 5 ms`).
