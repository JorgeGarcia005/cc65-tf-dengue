# Veredicto — PR-77 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/59-evaluacion` @ `d82266e` (base: `integracion/tb2` @ `396c4ce`)
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El núcleo analítico de `internal/evaluacion` está sólidamente planteado y es 100 % stdlib: la curva ROC acumula el trapecio sobre la curva completa agrupando empates (exactamente equivalente a Mann-Whitney `ml.AUC` dentro de `1e-12`), la calibración por deciles aplica intervalos de Wilson al 95 % junto con Brier sobre probabilidades calibradas (`ml.Calibrar`), el cubo departamento × semana × grupo de edad acumula los 8 niveles en una sola pasada $O(N)$ determinista sobre un arreglo plano (sin mapas) y las cifras de escalabilidad y latencia se copian sin recalcular desde `reports/bench_cluster_deez_gold.json` y `reports/bench_api_deez.json`. Toda la verificación local pasa en verde (`gofmt`, `go vet`, `go test -race -count=1 ./...` en los 27 paquetes con 0 carreras). Sin embargo, el PR no puede quedar `APROBADO` en esta ronda por **2 hallazgos bloqueantes** (falta generar y versionar `reports/evaluacion.json` con los datos reales y contrastar el AUC contra `reports/entrenamiento.json` — **H1** —, y el PR no cuenta con ningún chequeo de CI reportado en GitHub porque `.github/workflows/ci.yml` aún no cubre `integracion/tb2` — **H2**), además de **3 hallazgos mayores** (pánicos y lectura cruzada entre filas cuando `test.bin` y `modelo.json` difieren en `D` — **H3** —, pánicos en `cmd/evaluacion` cuando `Prioridades` está vacía o `-top < 0` junto con serialización `"prioridades": null` — **H4** —, y omisión silenciosa de la validación de `-entrenamiento` cuando el JSON no trae `metricas_test.AUC`, camino además sin tests en `cmd/evaluacion` — **H5**) y **1 menor** (**H6**).

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...origin/feature/59-evaluacion
(vacío = ok)

$ bash scripts/check-skills-sync.sh && wc -m AGENTS.md
skills: ok
7781 AGENTS.md

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.429s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.325s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.552s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.288s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.284s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.221s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.421s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.166s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.030s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.120s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.105s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.346s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.105s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.054s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.004s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.081s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.373s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.029s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.908s

$ go test -race -count=1 -v -cover ./internal/evaluacion ./cmd/evaluacion
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.800s	coverage: 85.9% of statements (7/7 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.285s	coverage: 47.4% of statements (2/2 PASS; ejecutar: 65.9%, main: 0.0%)

$ go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/evaluacion ./cmd/evaluacion
(100 % biblioteca estándar + paquetes internos del proyecto; 0 dependencias de terceros)

$ gh pr checks 77
no checks reported on the 'feature/59-evaluacion' branch
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 27/27 paquetes en verde en local) |
| CI de GitHub sobre el PR #77 | ❌ (0 checks reportados; ver **H2**) |
| Equivalencia secuencial = distribuido | no aplica (el PR no modifica `internal/ml` ni `internal/cluster`; `internal/ml` y `internal/cluster` pasan completos con `-race`) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`; copia `reports/bench_api_deez.json` sin recalcular) |
| Restricciones `AGENTS.md` §3 | ✅ solo stdlib en los paquetes nuevos y sin `data/` en el diff; ⚠️ pendiente `reports/evaluacion.json` real (**H1**) |

## Verificación contra los criterios del issue #59

| Criterio (#59) | Estado | Evidencia |
|---|---|---|
| 1. Puntos ROC y AUC por trapecio igual a `metricas_test.AUC` de `reports/entrenamiento.json` (±1e-6) | ⚠️ parcial (**H1**, **H5**) | `TestROCAUCIgualMannWhitney` verifica equivalencia trapecio = `ml.AUC` (±1e-12, con empates) y `TestGenerarAUCReferenciaDistinta` valida el rechazo en `Generar`; falta correrlo contra los datos reales (`0.6220849410863315` en `reports/entrenamiento.json:627`) y cubrir/endurecer la lectura de `-entrenamiento` en `cmd/evaluacion`. |
| 2. Curva de calibración por deciles (predicho calibrado vs observado) | ✅ | `Calibracion` (`curvas.go:94-112`) con IC de Wilson al 95 % (`curvas.go:125-135`) y `Brier` sobre `pc`; `TestCalibracionDeciles` PASS sobre 20 003 filas. |
| 3. Riesgo cruzado departamento × semana × grupo de edad; celdas con `< 30` casos marcadas `insuficiente` | ✅ (ajustes en **H3**, **H4**) | `Cruzar` (`riesgo.go:97-149`) y `TestCruzadoConteosYNiveles` verifican conteos directos, niveles «todos», `OTROS` y `Insuficiente: a.casos < minCasos`. |
| 4. Suma de casos esperados = suma de riesgos calibrados fila a fila (±1e-9) | ✅ | `TestCasosEsperados` verifica igualdad a `1e-9` tanto en el total nacional como en la suma por departamento, y `Fraccion == 1` (±1e-12). |
| 5. `go run ./cmd/evaluacion` dos veces → mismo sha256 de `reports/evaluacion.json` | ❌ pendiente (**H1**) | En prueba sintética de caja negra y en `TestEjecutarDeterminista` el sha256 coincide byte a byte, pero el artefacto real `reports/evaluacion.json` aún no fue generado ni versionado en la rama. |
| 6. Escalabilidad por nodos copiada desde `reports/bench_cluster_*_gold.json` (no recalculada) | ✅ | `LeerEscalabilidad` y `LeerLatencia` (`evaluacion.go:140-177`) leen directamente `reports/bench_cluster_deez_gold.json` y `reports/bench_api_deez.json`, verificado en `TestGenerarDeterminista` y `TestEjecutarFaltaArchivo`. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 (en curso) | Traduce operativamente la pregunta de impacto social (ODS 3) a casos esperados por departamento × semana × etapa de vida, con exclusión de `OTROS` y celdas `< 30` en prioridades. |
| Implementación | 8 | Parcial (pendiente **H1–H5**) | Paquete `internal/evaluacion` y CLI `cmd/evaluacion` deterministas, 100 % stdlib y libres de carreras; faltan el `reports/evaluacion.json` real (**H1**), CI verde (**H2**) y endurecer bordes que hoy provocan pánicos o validación omitida (**H3–H5**). |
| Interfaz | 5 | habilitador (pendiente **H1**, **H4**) | Define el contrato JSON (`version_esquema: 1`) que consumirán los issues #66, #67 y #68; requiere `reports/evaluacion.json` versionado y `"prioridades": []` en vez de `null`. |
| Informe | 2 | habilitador (pendiente **H1**) | Cumple «ninguna cifra a mano» (`AGENTS.md` §3.6) cuando `reports/evaluacion.json` real exista y quede trazado por sha256 de entradas. |
| Video | 2 | no aplica | Corresponde al cierre de TB2 (#72). |

## Hallazgos

### H1 — Falta generar y versionar `reports/evaluacion.json` con los datos reales · BLOQUEANTE

- **Dónde:** `reports/evaluacion.json` (ausente en el diff de `feature/59-evaluacion` @ `d82266e`), issue #59 (título y criterios 1 y 5) y descripción del PR #77.
- **Evidencia:**
  1. `git diff --stat origin/integracion/tb2...origin/feature/59-evaluacion` lista únicamente 7 archivos (`Makefile`, `cmd/evaluacion/{main.go,main_test.go}`, `internal/evaluacion/{curvas.go,evaluacion.go,evaluacion_test.go,riesgo.go}`); `reports/evaluacion.json` no existe en la rama.
  2. La propia descripción del PR deja abierta la casilla: `- [ ] Pendiente (Jorge): generar reports/evaluacion.json con los datos reales y subirlo a esta rama: go run ./cmd/evaluacion y luego sha256sum reports/evaluacion.json dos veces`.
  3. Sin este archivo real, nunca se ha verificado que `cmd/evaluacion` ejecute limpiamente sobre `data/gold/{test.bin,modelo.json}` reales ni que `auc_trapecio` coincida (±1e-6) con `metricas_test.AUC` (`0.6220849410863315` en `reports/entrenamiento.json:627`), y los issues dependientes (#66, #67, #68, #71) quedarían sin la fuente de datos versionada que exige `AGENTS.md` §3.6.
- **Corrección sugerida:** Correr `go run ./cmd/evaluacion` (o `make evaluacion`) en la máquina con `data/gold/` real, verificar dos corridas seguidas con `sha256sum reports/evaluacion.json` idéntico, añadir `reports/evaluacion.json` a la rama y marcar la casilla con el sha256 obtenido en la descripción del PR.

### H2 — El PR #77 no tiene ningún chequeo de CI reportado en GitHub · BLOQUEANTE

- **Dónde:** PR #77 (`baseRefName: "integracion/tb2"`) y `.github/workflows/ci.yml:3-7`.
- **Evidencia:**
  1. `gh pr checks 77` devuelve `no checks reported on the 'feature/59-evaluacion' branch` y `gh run list --branch feature/59-evaluacion` devuelve `[]`.
  2. En la base actual `integracion/tb2` (`396c4ce`), `.github/workflows/ci.yml:6-7` solo dispara el workflow de `pull_request` sobre `[main, develop]` (el PR #74 del issue #56 que añade `integracion/tb2` sigue abierto sin fusionar).
  3. Tanto `AGENTS.md` §5, `docs/agentes/PROTOCOLO.md` §7 como la skill `juez-cc65` establecen que un PR solo puede quedar `APROBADO` con el CI del PR en verde.
- **Corrección sugerida:** Fusionar primero el PR #74 (`#56`) en `integracion/tb2` y empujar los commits de corrección de esta ronda (o actualizar la rama) para que todos los jobs de `.github/workflows/ci.yml` corran y pasen en verde sobre el PR #77.

### H3 — Falta validar `in.Test.D == in.Esquema.D()` en `Generar` y `Cruzar` (pánico y lectura cruzada de filas) · MAYOR

- **Dónde:** `internal/evaluacion/evaluacion.go:71,78`, `internal/evaluacion/riesgo.go:55,69-70,97` y `cmd/evaluacion/main.go:98`.
- **Evidencia:**
  1. `Generar` (`evaluacion.go:71`) valida `len(in.Pesos) != in.Esquema.D()+1`, pero nunca comprueba `in.Test.D != in.Esquema.D()` (a diferencia de `cmd/entrenar/main.go:104`, que sí exige `esq.D() == test.D`). Tampoco `Cruzar` valida `ds == nil`, `esq == nil`, `ds.D != esq.D()`, `len(y) != ds.N` o `len(pc) != ds.N`, ni `NuevosGrupos` valida `esq == nil` o límites de columnas.
  2. Reproducción de caja negra contra `cmd/evaluacion` pasando un `modelo.json` con `D=31` y un `test.bin` con `D=33`:
     ```text
     panic: runtime error: index out of range [33] with length 32
       internal/ml.Puntaje(...)  internal/ml/logistica.go:150
       internal/evaluacion.Generar(...)  internal/evaluacion/evaluacion.go:78
     ```
  3. Con un `test.bin` de `D=29 < 31`, como `ds.Fila(i)` (`internal/ml/logistica.go:55`) devuelve `ds.X[i*D : (i+1)*D]` conservando la capacidad hasta el final de `ds.X`, `ml.Puntaje` usa `w[29]` como sesgo, `x[g.jDepto:g.jDepto+g.nDepto]` (`riesgo.go:70`) **lee columnas pertenecientes a la fila `i+1`** en las filas `0..N-2` y finalmente entra en pánico en la última fila:
     ```text
     panic: runtime error: slice bounds out of range [:31] with capacity 29
       internal/evaluacion.Grupos.De(...)  internal/evaluacion/riesgo.go:70
       internal/evaluacion.Cruzar(...)     internal/evaluacion/riesgo.go:106
     ```
- **Corrección sugerida:** En `evaluacion.Generar`, añadir `in.Test.D != in.Esquema.D()` a la validación inicial devolviendo un error descriptivo con ambas dimensiones; en `Cruzar`, validar `ds == nil || esq == nil || ds.D != esq.D() || len(y) != ds.N || len(pc) != ds.N` (y `esq == nil` más rangos en `NuevosGrupos`). Cubrir con tests unitarios para `D < esq.D()` y `D > esq.D()`.

### H4 — Pánico en `cmd/evaluacion` con `Prioridades` vacía o `-top < 0`, y JSON `"prioridades": null` · MAYOR

- **Dónde:** `cmd/evaluacion/main.go:44-47` e `internal/evaluacion/riesgo.go:155,167`.
- **Evidencia:**
  1. `cmd/evaluacion/main.go:46-47` indexa directamente `rep.Prioridades[0]` sin verificar `len(rep.Prioridades) > 0`. Al ejecutar `go run ./cmd/evaluacion` sobre un `test.bin` sintético de 2000 filas con el valor por defecto `-min-casos 30` (o con `-top 0`), el programa escribe el JSON con `"prioridades": null` y acto seguido cae con:
     ```text
     panic: runtime error: index out of range [0] with length 0
       main.main()  cmd/evaluacion/main.go:46
     ```
  2. Con `-top -1`, `Prioridades` (`riesgo.go:167`) ejecuta `out[:min(n, len(out))]` con límite negativo y entra en pánico:
     ```text
     panic: runtime error: slice bounds out of range [:-1]
       internal/evaluacion.Prioridades(...)  internal/evaluacion/riesgo.go:167
     ```
  3. Como `out` se declara `var out []Celda` (`riesgo.go:155`), una lista vacía se serializa como `"prioridades": null` en lugar de `[]`, fragilizando a los consumidores TypeScript (#66, #67).
- **Corrección sugerida:**
  - En `Prioridades` (`riesgo.go:155,167`): inicializar `out := make([]Celda, 0)` y devolver `out` vacío si `n <= 0` (o rechazar `in.TopN < 0` en `Generar`).
  - En `cmd/evaluacion/main.go:44-47`: imprimir la parte de `top: ...` solo cuando `len(rep.Prioridades) > 0` (en caso contrario informar `sin combinaciones con ≥ minCasos`).
  - Añadir un test que verifique `Prioridades` con `n <= 0` y sin celdas suficientes (y que serialice `[]`, no `null`).

### H5 — La verificación contra `-entrenamiento` se omite en silencio si el JSON carece de `metricas_test.AUC` y no tiene cobertura de tests · MAYOR

- **Dónde:** `cmd/evaluacion/main.go:76-89`, `internal/evaluacion/evaluacion.go:90` y `cmd/evaluacion/main_test.go:56-58,81`.
- **Evidencia:**
  1. En `cmd/evaluacion/main.go:77-89`, cuando `o.entrenamiento != ""`, se lee el JSON en `struct{ Test struct{ AUC float64 } }` y se pasa `aucRef = e.Test.AUC`. Si el archivo indicado no tiene `metricas_test` (por ejemplo, pasando `-entrenamiento reports/entrenamiento_cluster.json`, archivo real del repositorio), `aucRef` queda en `0`, `Generar` (`evaluacion.go:90`) interpreta `0` como «no comparar», **omite la validación en silencio**, sale con código `0` y además registra `reports/entrenamiento_cluster.json` dentro de `fuentes` como si hubiera contrastado el AUC.
  2. En `cmd/evaluacion/main_test.go:56-58`, `preparar` deja `o.entrenamiento` vacío (`""`), por lo que el bloque `main.go:77-89` nunca se ejecuta en la suite (`ejecutar` tiene 65.9 % de cobertura y `TestEjecutarDeterminista` solo comprueba `len(rep.Fuentes) != 4`).
- **Corrección sugerida:** En `cmd/evaluacion/main.go:77-89`, validar que `e.Test.AUC` exista y esté en `(0, 1]` (p. ej. con `*float64` o `if e.Test.AUC <= 0 { return ..., fmt.Errorf("%s: falta metricas_test.AUC", o.entrenamiento) }`). En `cmd/evaluacion/main_test.go`, escribir un `entrenamiento.json` temporal y probar los tres casos: AUC coincidente (5 fuentes, OK), AUC divergente (error) y JSON sin `metricas_test.AUC` (error).

### H6 — Comentario impreciso en `Calibracion`, guardas numéricas defensivas y marcador `PR-<n>` · MENOR

- **Dónde:** `internal/evaluacion/curvas.go:51,91-92,101,115-122`, `internal/evaluacion/riesgo.go:130,142-143` y descripción del PR #77.
- **Evidencia:**
  1. El comentario de `Calibracion` (`curvas.go:91-92`) dice *«(el último absorbe el resto)»*, pero `ini, fin := b*len(idx)/k, (b+1)*len(idx)/k` (`curvas.go:101`) reparte el resto `n % k` entre `n % k` tramos (p. ej., para `n = 20003, k = 10`, los tramos con 2001 filas son `b ∈ {3, 6, 9}`, comportamiento preferible al descrito).
  2. En funciones exportadas del paquete: en `ROC` (`curvas.go:51`), un `NaN` en `p` hace que `p[idx[j]] == p[idx[i]]` sea `false` incluso con `j == i` (bucle infinito); en `Brier` (`curvas.go:121`), un slice vacío devuelve `NaN`; y en `Cruzar` (`riesgo.go:130,142-143`), si un llamador directo pasa `pc` todo en `0` (`nac.suma == 0`), `Fraccion` y `RazonNacional` quedan en `NaN` y rompen `json.Marshal`.
  3. En la sección «Uso de IA» de la descripción del PR #77 figura aún el marcador literal `docs/agentes/revisiones/PR-<n>/ronda-1.md` en lugar de `PR-77`.
- **Corrección sugerida:** Alinear el comentario de `Calibracion` con el reparto real, agregar guardas simples contra valores no finitos en `ROC`, `len(y) == 0` en `Brier` y `nac.suma == 0` en `Cruzar`, y actualizar `PR-<n>` → `PR-77` en la descripción del PR.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-77. |

## Cómo explicarlo en la sustentación

`internal/evaluacion` y `cmd/evaluacion` construyen el artefacto determinista `reports/evaluacion.json` que alimenta tanto la SPA de TB2 como el informe final sin cifras a mano: sobre el conjunto de prueba (`test.bin`) calcula la curva ROC verificando que su AUC por trapecio coincida con Mann-Whitney y con `reports/entrenamiento.json` (±1e-6), la curva de calibración por deciles con intervalos de Wilson y Brier sobre las probabilidades calibradas (corrección de prior de King y Zeng), y un cubo cruzado departamento × semana epidemiológica × etapa de vida que agrega todos los marginales en una sola pasada determinista. Para responder la pregunta de impacto social (dónde y cuándo reforzar camas y cerco), las prioridades se ordenan por **casos esperados con signos de alarma** (suma del riesgo calibrado de los casos notificados), descartando `OTROS` y celdas con menos de 30 casos (`insuficiente`), y se copian sin recalcular las mediciones de escalabilidad del clúster y latencia p95 de la API junto con el `sha256` de cada archivo fuente. Una vez generado `reports/evaluacion.json` sobre los datos reales (**H1**), habilitado el CI en `integracion/tb2` (**H2**) y cubiertas las validaciones de bordes (**H3–H5**), el paquete queda listo para integrarse con el mapa y las vistas de evaluación.
