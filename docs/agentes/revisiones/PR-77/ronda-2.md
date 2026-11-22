# Veredicto — PR-77 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/59-evaluacion` @ `4538f2b` (base: `integracion/tb2` @ `b405959`)
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Todos los hallazgos de la ronda 1 (**H1–H6**) quedaron cerrados y verificados con evidencia directa. `reports/evaluacion.json` (1 975 761 bytes, SHA-256 `ddc12d9b49b5411a9a8abb942f6200758629feedf3fededa9ebad22befd36fbf`) ya está versionado en la rama con los datos reales de `data/gold/` (`190 819` filas de test, `24 002` positivos, `auc_trapecio = 0.6220849410863555` contra `metricas_test.AUC = 0.6220849410863315` de `reports/entrenamiento.json`, Δ = `2,40e-14 < 1e-6`), con todas las invariantes de conservación, prioridades, benchmarks copiados y checksums SHA-256 verificadas. La base `integracion/tb2` @ `b405959` quedó integrada (`4538f2b`), los 6 jobs del CI en GitHub pasan en verde (run `44974681855`), las validaciones de dimensiones, prioridades vacías/`-top < 0`, `-entrenamiento` sin `metricas_test.AUC` y guardas numéricas quedaron implementadas y probadas, y la suite completa con `-race` pasa en verde con 0 carreras.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/integracion/tb2...HEAD
(vacío = ok)

$ go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/evaluacion ./cmd/evaluacion
.../internal/evaluacion: [cmp crypto/sha256 encoding/hex encoding/json errors fmt .../internal/features .../internal/ingest .../internal/ml math slices strings]
.../cmd/evaluacion: [encoding/json flag fmt .../internal/api .../internal/evaluacion .../internal/gold log os path/filepath]

$ sha256sum reports/evaluacion.json reports/entrenamiento.json reports/bench_cluster_deez_gold.json reports/bench_api_deez.json
ddc12d9b49b5411a9a8abb942f6200758629feedf3fededa9ebad22befd36fbf  reports/evaluacion.json
ce550ebe97664971b1469168e291c37aac0f96a35d9df520eac55d18cd71a17e  reports/entrenamiento.json
fbf04b010cc4fd8d638a8f42aaa7e13b7f0705be39a1b8f503fdffdf16e65d61  reports/bench_cluster_deez_gold.json
c3facf00eaa44bd2b5b3b974ff0d514a469513e11d62f4d9167591b82d8773df  reports/bench_api_deez.json

$ python3 <auditoría de invariantes sobre reports/evaluacion.json>
filas_test: 190819 · positivos_test: 24002 · brier_calibrado: 0.10778136616581856
auc_trapecio: 0.6220849410863555 · metricas_test.AUC (ent): 0.6220849410863315 · diff: 2.398e-14
nac (TODOS/0/todas): casos=190819, pos=24002, casos_esperados=23918.450521258692
  hojas   (4771): diff casos=0, diff pos=0, diff esp=2.547e-11
  deptos  (  21): diff casos=0, diff pos=0, diff esp=4.729e-11
  semanas (  52): diff casos=0, diff pos=0, diff esp=1.455e-11
  etapas  (   5): diff casos=0, diff pos=0, diff esp=1.091e-11
  calibrac(  10): diff casos=0, diff esp=4.002e-11
prioridades: 15/15 válidas (>= 30 casos, sin TODOS/OTROS, ordenadas desc.); top 1: PIURA sem 20 adulto_30_59 (161.86)
escalabilidad & latencia_api: todos los campos coinciden bit a bit con sus JSON fuente

$ go test -race -count=1 -v -cover ./internal/evaluacion ./cmd/evaluacion
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.751s	coverage: 86.6% of statements (11/11 PASS)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.596s	coverage: 58.5% of statements (4/4 PASS)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.369s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.168s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.708s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.030s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.623s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.258s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.221s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.413s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.162s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.031s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.051s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.857s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	7.818s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.271s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.235s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.071s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.987s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.240s

$ gh pr checks 77   # run 44974681855 sobre 4538f2b
Docker (imagen, clúster y caída de un nodo)	pass	2m29s
Ecosistema de agentes                       	pass	6s
Go (gofmt, vet, test -race)                 	pass	56s
MongoDB y Redis (tests de integración)      	pass	41s
OpenAPI 3 válido                            	pass	19s
Spin (modelo del protocolo y mutantes)      	pass	1m57s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; 22/22 paquetes con tests en verde; 86,6 % en `internal/evaluacion`, 58,5 % en `cmd/evaluacion`) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; la suite completa pasa) |
| Latencia API p95 < 100 ms | ✅ (copiada verbatim de `reports/bench_api_deez.json`: p95 fría `2,21 ms`, caliente `2,06 ms`) |
| CI de GitHub en el PR #77 | ✅ (6/6 jobs en verde, run `44974681855`) |

## Verificación contra los criterios del issue #59

| Criterio (#59) | Estado | Evidencia |
|---|---|---|
| 1. Puntos ROC y AUC por trapecio igual a `metricas_test.AUC` de `reports/entrenamiento.json` (±1e-6) | ✅ | `reports/evaluacion.json` (`auc_trapecio = 0.6220849410863555` vs `0.6220849410863315`, Δ = `2,40e-14`), `TestROCAUCIgualMannWhitney`, `TestGenerarAUCReferenciaDistinta`, `TestEjecutarAUCReferencia`. |
| 2. Curva de calibración por deciles (predicho calibrado vs observado) | ✅ | 10 tramos en `reports/evaluacion.json` con IC de Wilson al 95 % y `brier_calibrado = 0.1078`; `TestCalibracionDeciles`. |
| 3. Riesgo cruzado departamento × semana × grupo de edad sobre test; celdas `< 30` marcadas `insuficiente` | ✅ | `6 224` celdas (`4 771` hojas + marginales) en `reports/evaluacion.json`, `15` prioridades válidas con ≥ 30 casos y sin `OTROS`; `TestCruzadoConteosYNiveles`, `TestPrioridades`. |
| 4. Suma de casos esperados = suma de riesgos calibrados fila a fila (±1e-9) | ✅ | Verificado en `TestCasosEsperados` y sobre los 190 819 casos reales de `reports/evaluacion.json` (diff ≤ `4,73e-11` en hojas, marginales y deciles). |
| 5. `go run ./cmd/evaluacion` dos veces → mismo sha256 de `reports/evaluacion.json` | ✅ | `56dc101` (`ddc12d9b49b5411a9a8abb942f6200758629feedf3fededa9ebad22befd36fbf` en dos corridas reales) y `TestEjecutarDeterminista`. |
| 6. Escalabilidad por nodos copiada desde `reports/bench_cluster_*_gold.json` (no recalculada) | ✅ | Verificado campo por campo contra `reports/bench_cluster_deez_gold.json` y `reports/bench_api_deez.json` más checksums SHA-256 en `fuentes`. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Responde con cifras reproducibles la pregunta de impacto social (ODS 3) por departamento × semana × etapa de vida. |
| Implementación | 8 | Completo para evaluación | 100 % stdlib, determinista, sin carreras, con bordes validados y `reports/evaluacion.json` real versionado. |
| Interfaz | 5 | Habilitador listo | Contrato JSON estable (`version_esquema: 1`, `"prioridades": []` nunca `null`) listo para #66, #67 y #68. |
| Informe | 2 | Cumple §3.6 | Cero cifras a mano; todas las métricas y prioridades salen de `reports/evaluacion.json` con trazabilidad SHA-256. |
| Video | 2 | no aplica | Corresponde a #72. |

## Hallazgos

Ninguno. Todos los hallazgos de la ronda 1 quedaron cerrados y verificados sin introducir defectos nuevos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `56dc101`: `reports/evaluacion.json` generado sobre `data/gold/` real (SHA-256 `ddc12d9b49b5411a9a8abb942f6200758629feedf3fededa9ebad22befd36fbf`) con todas las invariantes y checksums verificados. *(Nota menor no bloqueante: marcar la casilla «Pendiente (Jorge)» en la descripción del PR antes del merge.)* |
| H2 (ronda 1 · BLOQUEANTE) | RESUELTO | Base `integracion/tb2` @ `b405959` integrada en `4538f2b`; 6/6 jobs del CI en verde (run `44974681855`). |
| H3 (ronda 1 · MAYOR) | RESUELTO | `75d3d31`: `Generar` (`evaluacion.go:71-77`), `NuevosGrupos` (`riesgo.go:56-67`) y `Cruzar` (`riesgo.go:102-104`) validan dimensiones y nulos; probado en `TestGenerarDimensionesDistintas` (`evaluacion_test.go:270-294`). |
| H4 (ronda 1 · MAYOR) | RESUELTO | `75d3d31` + `e9c2f46`: `Prioridades` (`riesgo.go:168-171`) siempre devuelve slice no nulo (`[]` en JSON), `cmd/evaluacion` (`main.go:44-56`) valida `-top >= 0` y evita indexar `Prioridades[0]` vacío; cubierto en `TestPrioridadesVaciasSeSerializanComoLista` y `TestEjecutarParametrosInvalidos`. |
| H5 (ronda 1 · MAYOR) | RESUELTO | `e9c2f46`: `cmd/evaluacion/main.go:88-98` exige `metricas_test.AUC` presente en `(0, 1]` cuando `-entrenamiento` no está vacío; cubierto en `TestEjecutarAUCReferencia` (`main_test.go:95-127`, 5 casos). |
| H6 (ronda 1 · MENOR) | RESUELTO | `75d3d31` + `1348021` y descripción del PR: comentario de `Calibracion` alineado (`curvas.go:96-99`), guardas en `ROC` (`:35-39`), `Brier` (`:122-125`) y `Cruzar` (`riesgo.go:138-143`), `TestGuardasNumericas`, `TestCruzarRiesgoNacionalCero` y marcador `PR-77` actualizado. |

## Cómo explicarlo en la sustentación

`internal/evaluacion` y `cmd/evaluacion` construyen de forma determinista `reports/evaluacion.json` sobre los 190 819 casos reales de prueba (`test.bin`) con trazabilidad SHA-256 de sus cinco fuentes: calcula la curva ROC comprobando que su AUC por trapecio (`0,6221`) coincide con Mann-Whitney y `reports/entrenamiento.json` (`Δ = 2,4×10⁻¹⁴`), verifica la calibración por deciles con intervalos de Wilson y Brier (`0,1078`), y arma el cubo departamento × semana × etapa de vida priorizando por **casos esperados con signos de alarma** (excluyendo `OTROS` y celdas `< 30` casos), donde destaca Piura en semana 20 para `adulto_30_59` (`161,9` casos esperados), además de copiar sin recalcular la escalabilidad del clúster y la latencia p95 de la API.
