# Veredicto — PR-86 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/68-evaluacion` @ `7f1426e` (base: `integracion/tb2` @ `c6b75ab`)
- **Fecha:** 2026-11-27
- **Decisión:** APROBADO

Tras el veredicto de la ronda 1 (`29a8677`, `APROBADO` con una observación menor no bloqueante `H1`) y la respuesta documentada en `a838488`, la revisión humana del equipo incorporó el commit `7f1426e` (`fix(web): nombre de corrida sin prototipo y cortes de AUC bien citados (#68)`), que resuelve de inmediato el punto de código de `H1` y corrige una imprecisión bibliográfica en la lectura del AUC:

- **`H1.1` cerrado en código y con prueba dedicada:** `nombreCorrida` (`web/src/paginas/Evaluacion.tsx:14-23`) pasó de indexar un objeto literal a un `switch` explícito (`'fria' | 'caliente' | default`), evitando que claves heredadas de `Object.prototype` (`"constructor"`, `"toString"`) devuelvan una función. El test nuevo `latencia: un nombre de corrida desconocido (incluso «constructor») se muestra tal cual (PR-86 H1)` (`web/src/pruebas/evaluacion.test.tsx:174-181`) verifica el caso en el DOM renderizado.
- **Cita rigurosa de Hosmer y Lemeshow en `lecturaAuc`:** `web/src/evaluacion/datos.ts:77-88` cita ahora con exactitud los tramos de Hosmer, Lemeshow y Sturdivant (*Applied Logistic Regression*, 3.ª ed., 2013, cap. 5: `0,5` sin discriminación; `0,5–0,7` pobre; `0,7–0,8` aceptable; `0,8–0,9` excelente; `≥ 0,9` sobresaliente), aclara que el corte interno en `0,6` es propio del proyecto para distinguir «apenas mejor que el azar» dentro del tramo pobre, e indica explícitamente para `[0,6; 0,7)` (`AUC = 0,622`): `«capacidad de discriminación modesta (pobre según Hosmer y Lemeshow, < 0,7): sirve para priorizar, no para diagnosticar»`. Los cinco tramos quedan cubiertos en `web/src/pruebas/evaluacion.test.tsx:208-216`.
- **Sin regresiones:** los 25 paquetes de Go pasan con `-race` en Linux (`0` carreras, incluido `cmd/web`), `npx tsc -b` y Vitest pasan al 100 % (`108/108` tests en 6 archivos, `36/36` en `evaluacion.test.tsx`), `npm run build` compila limpio sin `url(data:` en el CSS (compatible con `style-src 'self'`), y los **7/7 jobs del CI de GitHub** sobre `7f1426e` están en verde (`run 46068092012`).

## Verificación automática

```text
$ git rev-parse --short HEAD
7f1426e

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.931s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.738s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.903s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.038s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.924s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.272s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.227s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.418s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.384s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.067s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.162s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.030s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.045s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	28.149s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.693s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.775s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.828s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.041s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.004s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.033s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.091s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.050s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.449s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.752s

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm test -- evaluacion && npm run build && (grep -F "url(data:" dist/assets/*.css || echo "SIN_URL_DATA")
Test Files  6 passed (6)
     Tests  108 passed (108)
src/pruebas/evaluacion.test.tsx (36 tests) -> 36 passed (36)
✓ built in 1.14s · SIN_URL_DATA

$ gh pr checks 86 -R JorgeGarcia005/cc65-tf-dengue   # run 46068092012 sobre 7f1426e
Docker (imagen, clúster y caída de un nodo)	pass	2m32s
Ecosistema de agentes                      	pass	10s
Go (gofmt, vet, test -race)                	pass	1m22s
MongoDB y Redis (tests de integración)     	pass	59s
OpenAPI 3 válido                           	pass	24s
SPA (tipos, tests, build e imagen)         	pass	58s
Spin (modelo del protocolo y mutantes)     	pass	1m53s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...`) | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 25/25 paquetes con tests en verde en Linux) |
| Equivalencia secuencial = distribuido | no aplica (solo modifica `web/src/`; `internal/ml` e `internal/cluster` siguen en verde) |
| Latencia API p95 < 100 ms | no aplica (no modifica la API; `cmd/benchapi` e `internal/api` en verde) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (108/108 tests, 36/36 en `evaluacion.test.tsx`, 0 `url(data:` en CSS) |
| CI GitHub Actions (PR #86 @ `7f1426e`) | ✅ (7/7 jobs PASS) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Presenta con rigor estadístico y bibliográfico el desempeño real en prueba (`AUC = 0,622`, pobre según Hosmer y Lemeshow `< 0,7`, útil para priorizar vigilancia y no para diagnóstico individual) junto con calibración y matriz de confusión. |
| Implementación | 8 | 8.0 / 8 | Expone el speedup (`4,804×`), la eficiencia (`96,1 %`), la fracción de red/espera (`3,4 %`) y el `p95` de la API (`2,2 ms < 100 ms`) tomados directamente de `reports/evaluacion.json` y `doc.fuentes`, sin tocar el núcleo Go ni sumar dependencias. |
| Interfaz | 5 | 5.0 / 5 | Vista completa, accesible (`role="img"`, `aria-label`, `<figure>`/`<figcaption>`, `<th scope>`, `<caption>`), validada ante respuestas malformadas y nombres de corrida arbitrarios, compatible con CSP estricta y responsive en `1280`/`768`/`375 px`. |
| Informe | 2 | 2.0 / 2 | Cada gráfico y tabla cita su `reports/*.json` de origen (sin cifras a mano), cita de Hosmer y Lemeshow corregida y commits convencionales de José y Jorge. |
| Video | 2 | — | No corresponde a este PR (deja lista la vista de evaluación para capturas y demostración). |

## Hallazgos

Sin hallazgos abiertos (**0 bloqueantes, 0 mayores, 0 menores**).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1.1 (ronda 1) — `nombreCorrida` indexaba un objeto literal expuesto a propiedades de `Object.prototype` · MENOR | RESUELTO | Resuelto en `7f1426e`: `switch` explícito en `web/src/paginas/Evaluacion.tsx:14-23` y test con `"constructor"` en `web/src/pruebas/evaluacion.test.tsx:174-181`. |
| H1.2 (ronda 1) — Aviso informativo de Vite por chunk JS de `~502 kB` · MENOR | RETIRADO (ACLARADO) | En `respuesta-1.md` se aclara que es un aviso informativo transversal a toda la SPA y que el build + imagen Docker pasan en CI; no bloquea ni arriesga puntos. |
| H1.3 (ronda 1) — Regenerar `reports/evaluacion.json` con los benchmarks definitivos en #71 · MENOR | RESUELTO (TRAZADO A #71) | La vista ya toma dinámicamente los nombres de archivo desde `doc.fuentes` (`fuente()` y `<Fuente>`), verificado en `evaluacion.test.tsx:183-203`; la regeneración del artefacto corresponde a #71. |

## Cómo explicarlo en la sustentación

> La vista de **Evaluación** consume `GET /api/v1/evaluacion` (`reports/evaluacion.json`), valida toda la forma del documento con `esDocumentoEvaluacionCompleto` antes de renderizar y dibuja sin ninguna cifra escrita a mano las cinco evidencias experimentales: curva ROC de 201 puntos (`AUC = 0,622`, tramo pobre `< 0,7` en la escala de Hosmer y Lemeshow: sirve para priorizar recursos epidemiológicos y no para diagnosticar pacientes), calibración en 10 deciles con intervalos de Wilson al 95 % (`8/10` dentro del IC, `Brier = 0,108`), matriz de confusión 2×2 con sus métricas, escalabilidad del clúster de 1 a 5 nodos (`speedup 4,804`, eficiencia `96,1 %`, `3,4 %` de red y espera) y latencia `p95` de la API (`2,2 ms` frente al límite de `100 ms`), citando al pie de cada figura el `reports/*.json` leído dinámicamente desde `fuentes`.
