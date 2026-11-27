# Veredicto — PR-86 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/68-evaluacion` @ `f555807` (base: `integracion/tb2` @ `c6b75ab`)
- **Fecha:** 2026-11-26
- **Decisión:** APROBADO

El PR reemplaza el marcador de `/evaluacion` por la vista completa de **Evaluación del modelo** (`web/src/paginas/Evaluacion.tsx`, `web/src/evaluacion/datos.ts`, `web/src/evaluacion/Graficos.tsx`, tipos en `web/src/api/tipos.ts`, estilos en `web/src/estilos/vigia.css` y 35 pruebas en `web/src/pruebas/evaluacion.test.tsx`), tocando únicamente 6 archivos bajo `web/src/` sin modificar Go ni agregar dependencias externas (`AGENTS.md` §3.2, §3.7). Todas las cifras, textos interpretativos (`lecturaAuc`, conteo de deciles dentro del IC 95 %) y archivos fuente de benchmarks (`fuente(doc, /bench_cluster_/)`, `fuente(doc, /bench_api_/)`) salen de `GET /api/v1/evaluacion` (`reports/evaluacion.json`) sin ninguna cifra escrita a mano (`AGENTS.md` §3.6), y la forma de toda respuesta `200` se valida antes de pintar mediante `esDocumentoEvaluacionCompleto` (cubriendo 14 casos malformados en tests, incluidos HTML de proxy, campos de texto como objeto, `nodos = 0`, `casos < 0` y versión de esquema inválida) y `puntoUmbral`/`proporcion()` sin `NaN`. Toda la verificación local (`gofmt`, `go vet ./...`, `go vet -tags integracion ./internal/...`, 24/25 paquetes Go con `-race` y 0 carreras en Windows —con la limitación conocida de `cmd/web` al limpiar `t.TempDir()` por `os.OpenRoot`, idéntica en `origin/integracion/tb2`—, `npx tsc -b`, 107/107 tests de Vitest incluidos los 35 de `npm test -- evaluacion`, `vite build` sin `url(data:`), la auditoría en vivo en Chrome headless contra `cmd/web` con CSP estricta `style-src 'self'` (0 errores de consola, 0 excepciones, 0 violaciones de CSP, 0 estilos inline, 0 solapamientos o recortes de etiquetas SVG y `docScrollW === docClientW` sin tarjetas desbordadas a `1280px`, `768px` y `375px`) y los 7/7 jobs de GitHub Actions en Linux sobre `f555807` están en verde. Se deja únicamente una observación **MENOR** no bloqueante (**H1**) sobre la búsqueda en objeto literal en `nombreCorrida` y el aviso de tamaño de chunk de Vite (`501,75 kB`).

## Verificación automática

```text
$ go version
go version go1.26.10 windows/amd64

$ git log --format="%h | %an <%ae> | %s" origin/integracion/tb2..HEAD
f555807 | José Javier Villanueva Aramburú <152805563+JJva-06@users.noreply.github.com> | fix(web): blinda la forma de la evaluación y ordena sus secciones (#68)
9f1130e | José Javier Villanueva Aramburú <152805563+JJva-06@users.noreply.github.com> | feat(web): vista de evaluación con ROC, calibración, matriz, escalabilidad y p95 (#68)
d8cdec4 | José Javier Villanueva Aramburú <152805563+JJva-06@users.noreply.github.com> | test(web): tipos, validación y lógica pura de la vista de evaluación (#68)

$ git diff --stat origin/integracion/tb2...HEAD
 web/src/api/tipos.ts                |  59 ++++++
 web/src/estilos/vigia.css           | 125 +++++++++++++
 web/src/evaluacion/Graficos.tsx     | 138 ++++++++++++++
 web/src/evaluacion/datos.ts         |  85 +++++++++
 web/src/paginas/Evaluacion.tsx      | 349 +++++++++++++++++++++++++++++++++++-
 web/src/pruebas/evaluacion.test.tsx | 310 ++++++++++++++++++++++++++++++++
 6 files changed, 1060 insertions(+), 6 deletions(-)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go vet -tags integracion ./internal/...
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	9.998s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.416s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	8.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.820s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	3.002s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	2.104s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	2.202s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	2.235s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.740s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	4.591s
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	7.504s   # limitación Windows: 4 tests fallan solo en TempDir RemoveAll cleanup (os.OpenRoot abierto); pasa en CI Linux
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.803s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.648s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	6.599s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	18.940s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	9.200s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	7.871s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.273s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.658s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.390s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.645s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.628s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.005s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.583s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	9.689s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	3.113s

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm test -- evaluacion && npm run build
added 116 packages, and audited 117 packages in 14s (0 vulnerabilities)
tsc -b: exit 0
Test Files  6 passed (6)
     Tests  107 passed (107)
 ✓ src/pruebas/evaluacion.test.tsx (35 tests) 631ms
✓ built in 905ms (dist/assets/index-81bEa5HI.css 18.27 kB; dist/assets/index-DoRCqBsf.js 501.75 kB)
Select-String 'url\(data:' dist/assets/*.css → vacío (ok)

$ Select-String '[0-9]+[.,][0-9]{2,}' web/src (excluyendo pruebas/)
web/src/evaluacion/Graficos.tsx:74  x={x(0.97)} y={y(0.06)}           # coordenadas SVG de la leyenda AUC
web/src/evaluacion/Graficos.tsx:85  const max = Math.max(0.01, ...)   # piso 1 % del eje de calibración
(0 cifras del modelo o benchmarks escritas a mano)

$ <auditoría en vivo: cmd/web (CSP estricta style-src 'self') sirviendo web/dist + reports/evaluacion.json real en Chrome headless>
- CSP: default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self' http://127.0.0.1:18980 ws://127.0.0.1:18980; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'
- Consola: 0 errores, 0 excepciones no capturadas, 0 violaciones CSP, 0 atributos [style] inline, 0 NaN/Infinity/undefined.
- @ 1280×900: docScrollW=1265 === docClientW=1265, badTarjetas=0, 4/4 tablas sin scroll interno (calibracion 525/525, confusion 559/559, escalabilidad 1141/1141, latencia 1141/1141), 4/4 SVGs con role="img" + aria-label, 0 solapamientos y 0 recortes de texto.
- @  768×1024: docScrollW=753 === docClientW=753, badTarjetas=0, 4/4 tablas sin scroll interno (629/629), 4/4 SVGs (560×467) con 0 solapamientos y 0 recortes.
- @  375×812: docScrollW=375 === docClientW=375, badTarjetas=0, tablas anchas desplazables dentro de .tabla-envoltura (calibracion 357/251, escalabilidad 449/251, latencia 548/251; confusion 251/251), 4/4 SVGs (251×209) con 0 solapamientos y 0 recortes.
- Edge case 200 HTML de proxy: muestra role="alert" («No se pudo cargar la evaluación: la evaluación no tiene el formato esperado (versión o campos)») manteniendo el h1 y sin romper la SPA.
- 4/4 <figcaption> contienen reports/*.json (evaluacion.json en ROC y calibración; bench_cluster_deez_gold.json vía evaluacion.json en speedup y eficiencia); matriz y latencia citan sus fuentes en .nota.

$ gh pr checks 86   # run 45934400081 sobre f555807
Docker (imagen, clúster y caída de un nodo)	pass	2m32s
Ecosistema de agentes                       	pass	6s
Go (gofmt, vet, test -race)                 	pass	1m30s
MongoDB y Redis (tests de integración)      	pass	51s
OpenAPI 3 válido                            	pass	22s
SPA (tipos, tests, build e imagen)          	pass	56s
Spin (modelo del protocolo y mutantes)      	pass	1m58s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (24/25 paquetes PASS en Windows con 0 carreras; `cmd/web` falla solo en `TempDir` cleanup en Windows por `os.OpenRoot`, idéntico en `origin/integracion/tb2`; **PASS en CI Linux**) |
| Equivalencia secuencial = distribuido | no aplica (no toca `internal/ml` ni `internal/cluster`; suite Go en verde) |
| Latencia API p95 < 100 ms | no aplica (no toca la API; la vista presenta `2,2 ms` fría / `2,1 ms` caliente frente al límite de `100 ms` desde `reports/evaluacion.json`) |
| SPA (`tsc -b`, Vitest, `vite build`, sin `url(data:)`, CSP y responsive en Chrome) | ✅ (107/107 tests PASS, 35/35 en `evaluacion.test.tsx`, 0 errores consola/CSP, 0 estilos inline, sin desbordes en `1280px`/`768px`/`375px`) |
| CI GitHub Actions (PR #86 @ `f555807`) | ✅ (7/7 jobs PASS, run `45934400081`) |

## Verificación contra los criterios del issue #68 y §4.5

| Criterio (#68 / §4.5) | Estado | Evidencia |
|---|---|---|
| 1. ROC, calibración, matriz de confusión, speedup/eficiencia por nodos y p95 de la API (`npm test -- evaluacion`) | ✅ | `web/src/paginas/Evaluacion.tsx`, `web/src/evaluacion/Graficos.tsx`, `web/src/evaluacion/datos.ts`; `evaluacion.test.tsx` (35/35 PASS contra `reports/evaluacion.json` real: ROC de 201 puntos `(0,0)→(1,1)` con AUC `0.622` y punto del umbral `(FPR=43.6%, TPR=61.9%)`, 10 deciles que suman `190 819` casos y Brier `0.108`, matriz 2×2 con `VP+FN+FP+VN = 190 819` y `VP+FN = 24 002`, speedup `4.804` / eficiencia `96.1%` con 5 nodos, latencia p95 `2.2 ms` / `2.1 ms` con «✓ cumple < 100 ms»). |
| 2. Pie de **cada** gráfico con el `reports/*.json` de origen | ✅ | Los 4 `<figcaption>` (ROC, calibración, speedup y eficiencia) y las notas de matriz y latencia citan `reports/*.json`; escalabilidad y latencia toman el archivo dinámicamente de `doc.fuentes` mediante `fuente()` y `<Fuente>`, verificado por test unitario y en DOM real. |
| 3. Validación de forma de respuestas `200`, cero `NaN`, responsive `1280/768/375 px`, accesibilidad, CSP sin estilos inline y cero cifras a mano (§4.5) | ✅ | `esDocumentoEvaluacionCompleto` rechaza 14 variantes malformadas (HTML de proxy, objetos donde se espera texto, `nodos=0`, `casos<0`, versión distinta); `puntoUmbral` usa `proporcion()`; Chrome headless contra `cmd/web` confirma `docScrollW === docClientW` en `1280`/`768`/`375 px`, 0 violaciones de CSP, 0 estilos inline, `<svg role="img" aria-label>`, `<figure>`/`<figcaption>`, `<th scope>` + `<caption>`, lectura honesta del AUC («modesta: sirve para priorizar, no para diagnosticar») y 0 cifras escritas a mano. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Presenta de forma transparente el desempeño real del modelo en el conjunto de prueba (`AUC = 0,622`, calibración en 10 deciles y matriz de confusión) con lectura honesta alineada al uso de priorización en salud pública (ODS 3). |
| Implementación | 8 | 8.0 / 8 | Muestra el speedup real (`4,804×`), eficiencia (`96,1 %`), fracción de red/espera (`3,4 %`) y latencia p95 de la API (`2,2 ms < 100 ms`) directamente desde `reports/evaluacion.json` y `doc.fuentes`, manteniendo el backend intacto y sin nuevas dependencias. |
| Interfaz | 5 | 5.0 / 5 | Vista de evaluación completa, ordenada según §4.2, accesible (`role="img"`, `aria-label`, `<figure>`/`<figcaption>`, `<th scope>`, `<caption>`), resiliente ante respuestas `200` malformadas, sin estilos inline bajo CSP estricta y sin desbordes horizontales en escritorio, tableta y móvil. |
| Informe | 2 | 2.0 / 2 | Trazabilidad explícita al `reports/*.json` en cada gráfico/tabla y Git Flow limpio (3 commits convencionales en español con autoría de José `JJva-06`). |
| Video | 2 | — | No corresponde a este PR (deja lista la vista de evaluación para las capturas de #70 y el bloque 4:20–5:20 del video #72). |

## Hallazgos

Sin hallazgos **BLOQUEANTES** ni **MAYORES**.

### H1 — Detalles menores no bloqueantes: consulta de propiedades heredadas en `nombreCorrida` y aviso de tamaño de chunk de Vite · MENOR *(no bloqueante)*

- **Dónde:** `web/src/paginas/Evaluacion.tsx:13` y salida de `npm run build` (`dist/assets/index-DoRCqBsf.js`)
- **Evidencia:**
  1. `const nombreCorrida = (n: string) => ({ fria: 'Fría', caliente: 'Caliente' })[n] ?? n` indexa un objeto literal con `Object.prototype`: si una API alterada devolviera un `nombre` igual a una propiedad heredada (p. ej. `"constructor"` o `"toString"`), devolvería una función en lugar de caer al respaldo `n`. En la API real solo se emiten `"fria"` y `"caliente"`, por lo que no afecta la operación normal.
  2. `vite build` advierte que el bundle JS alcanza `501,75 kB` (`170,31 kB` gzip, apenas sobre el umbral informativo de `500 kB`). Es una advertencia, no un error, y el build/imagen Docker en CI pasan en verde.
- **Corrección sugerida:** No requiere una ronda extra en este PR; si se realiza una pasada de pulido en #70, cambiar `nombreCorrida` a un `Map` o `switch`/`if` (eliminando la búsqueda en el prototipo) y, opcionalmente, dividir las rutas con `React.lazy` para bajar el chunk inicial por debajo de `500 kB`. Asimismo, mantener anotado para #71 regenerar `reports/evaluacion.json` con el benchmark de clúster definitivo (actualmente `escalabilidad.maquina.so` dice `windows`) y con `bench_api_jorge-pc_jwt.json` (el pie de la vista se actualizará solo desde `doc.fuentes`).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda oficial del juez sobre el PR #86 (la revisión informal previa en dos rondas quedó verificada independientemente sobre `f555807`). |

## Cómo explicarlo en la sustentación

> La vista de **Evaluación** consume `GET /api/v1/evaluacion` (`reports/evaluacion.json`), valida toda su estructura con `esDocumentoEvaluacionCompleto` antes de dibujar (mostrando un aviso accesible si llega HTML de proxy o un reporte incompleto) y presenta sin ninguna cifra escrita a mano las cuatro piezas experimentales: curva ROC de 201 puntos con `AUC = 0,622` y el punto exacto del umbral desde la matriz de confusión (lectura honesta: capacidad modesta, útil para priorizar y no para diagnosticar), calibración en 10 deciles con intervalos de Wilson al 95 % (`8/10` dentro del IC y `Brier = 0,108`), matriz de confusión 2×2 con sus métricas, escalabilidad del clúster de 1 a 5 nodos (`speedup 4,804`, eficiencia `96,1 %`, `3,4 %` de red y espera) y latencia de la API (`p95 = 2,2 ms` fría / `2,1 ms` caliente frente al límite de `100 ms`). Todos los gráficos son SVG hechos con `d3-scale` y atributos puros —sin estilos inline para cumplir la CSP `style-src 'self'` de `cmd/web`—, se adaptan sin scroll horizontal a 1280, 768 y 375 px, y citan al pie el `reports/*.json` de origen leído dinámicamente de `fuentes`.
