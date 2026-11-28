# Veredicto — PR-104 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/103-informe-tb2-bloque3` @ `fda4d81`
- **Fecha:** 2026-11-28
- **Decisión:** APROBADO

El PR completa con rigor técnico y trazabilidad automática el bloque 3 del informe final TB2 (issue `#103`, secciones 6 a 11) más el candado del paso 16 en `scripts/paso70.sh`:

1. **Cero cifras a mano (`AGENTS.md` §3.6):** todas las tablas (`docs/informe/tb2/plantillas/*.tmpl`) y los valores en línea (`<!--v:EXPR-->…<!--/v-->`) de las secciones `06-datos.md` a `11-escalabilidad.md` salen de `reports/*.json` y coinciden byte a byte con el árbol fuente (`go run ./cmd/informe && git diff --exit-code` vacío y `TestInformeTB2AlDia` en verde).
2. **Atención directa a las observaciones 2 y 5 de la PC4:**
   - *PC4 obs. 2 (pruebas que sustentan los valores de las tablas):* la sección 11 incorpora la tabla de dispersión de las repeticiones (`@tab:cluster-muestras`), la estimación lineal del punto de equilibrio (`@tab:equilibrio-cluster`, cubierta por `TestEquilibrioClusterConElModeloLineal`), la función completa `medirNodos` extraída de `cmd/benchmark/cluster.go` y el reporte íntegro `reports/bench_cluster_deez_gold.json` (al igual que `reports/limpieza.json` en `07-limpieza.md:151-204`). En `09-cluster.md:68-85`, `regionProtocolo` construye, valida con `Mensaje.Validar` y serializa con `encoding/json` un ejemplo real de los 7 tipos del protocolo TCP (`TestRegionProtocoloEsDelProtocoloReal`).
   - *PC4 obs. 5 (validaciones de `spin/barrera.pml`):* la sección 10 documenta el alcance (`@tab:spin-alcance`), la correspondencia con Go (`@tab:spin-correspondencia`), las 6 aserciones y la propiedad LTL `ltl termina { <> terminado }` incrustadas desde `spin/barrera.pml` mediante marcadores `/* informe:… */` (comprobados con `bash spin/verificar.sh`: los mismos 10 chequeos, estados y errores de `reports/spin.json`), la ausencia de interbloqueos/carreras y el contraejemplo de cada uno de los 4 mutantes (`@tab:spin-mutantes` y `@tab:spin`).
3. **Evidencias, capturas, diagramas UML 2.5 y candado de `#70`:** ubica `evidencias/caida.txt` (`09-cluster.md:377`), `evidencias/cluster.txt` (`09-cluster.md:391`, retirado de `05-despliegue.md:637` para no duplicarlo), `evidencias/spin.txt` (`10-spin.md:242`) y las capturas `capturas/e2e-09-admin-nodo-caido.png` (`09-cluster.md:381`) y `capturas/e2e-10-admin-metricas.png` (`09-cluster.md:428`); incorpora 5 diagramas UML 2.5 en Graphviz (`epoca`, `flujo`, `nodo`, `topologia`, `worker_pool`) sin avisos en `cmd/word` (`TestInformesSinAvisos`); y bloquea el paso 16 de `scripts/paso70.sh:109-116` mientras queden secciones con `<!-- pendiente:`.

Todas las verificaciones automáticas (`gofmt`, `go vet`, `go test -race ./...` en las 26 suites con tests y 0 carreras, `spin/verificar.sh` y los 7 trabajos del CI en GitHub Actions) están en verde. Se dejan únicamente dos observaciones **MENORES *(no bloqueantes)*** (**H1** y **H2**) para pulir en los bloques 4–5 de `#71`.

## Verificación automática

```text
$ git rev-parse --short HEAD
fda4d81

$ git diff --check origin/integracion/tb2...origin/feature/103-informe-tb2-bloque3
(vacío = ok)

$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ bash scripts/check-skills-sync.sh && bash -n scripts/paso70.sh
skills: ok

$ grep -l '<!-- pendiente:' docs/informe/tb2/*.md | xargs -r -n1 basename | sort | tr '\n' ' '
12-api.md 13-latencia.md 14-bd.md 15-seguridad.md 16-interfaz.md 17-evaluacion.md 18-impacto.md 19-gestion.md 20-conclusiones.md 21-recomendaciones.md 23-anexos.md

$ go run ./cmd/informe && git status -s
2026/11/28 15:59:24 escrito docs/informe/tb2/fig/casos_por_ano.svg
2026/11/28 15:59:24 escrito docs/informe/tb2/fig/riesgo_departamento.svg
2026/11/28 15:59:24 escrito docs/informe/tb2/fig/riesgo_semana.svg
2026/11/28 15:59:24 escrito docs/informe/tb2/fig/speedup_gradiente.svg
2026/11/28 15:59:24 escrito docs/informe/tb2/fig/speedup_loader.svg
2026/11/28 15:59:24 escrito docs/informe/tb2/fig/speedup_cluster.svg
(árbol limpio)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	29.817s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	21.509s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	7.182s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.037s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.908s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.332s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	2.139s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.431s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.025s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/seguridad	81.540s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.370s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.057s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	9.785s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.243s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.257s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	44.588s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	17.062s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	13.972s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.741s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.158s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	2.235s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.234s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	9.451s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.158s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	16.142s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.840s

$ cp reports/spin.json /tmp/spin.json.orig && bash spin/verificar.sh && diff -u <(jq .corridas /tmp/spin.json.orig) <(jq .corridas reports/spin.json) && git checkout -- reports/spin.json && rm /tmp/spin.json.orig
RESULTADO: todo como se esperaba
(sin diferencias en corridas/estados/errores)

$ gh pr checks 104 -R JorgeGarcia005/cc65-tf-dengue   # run 46161111840 sobre fda4d81
Docker (imagen, clúster y caída de un nodo)	pass	2m30s
Ecosistema de agentes                       	pass	11s
Go (gofmt, vet, test -race)                 	pass	2m18s
MongoDB y Redis (tests de integración)      	pass	52s
OpenAPI 3 válido                            	pass	18s
SPA (tipos, tests, build e imagen)          	pass	1m0s
Spin (modelo del protocolo y mutantes)      	pass	1m46s
```

> *Nota sobre `scripts/figuras.sh --verificar`:* en esta VM (`Ubuntu 24.04`, `libcairo2 1.18.0`) Cairo exporta los contornos de glifos con redondeo ligeramente distinto al contenedor con que se generaron los `.svg` (como advierte `scripts/figuras.sh:6-7`); se verificó que los 5 diagramas nuevos tienen exactamente el mismo `width`, `height` y `viewBox` que produce `dot -Tsvg:cairo` y pasan `TestInformesSinAvisos` y `TestAltoMaximo` en `cmd/word`.

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ✅ (`26/26` suites con tests en verde, carreras: 0) |
| Equivalencia secuencial = distribuido | no aplica (no modifica `internal/ml` ni `internal/cluster`; pasa dentro de la suite y en el job Docker del CI) |
| Latencia API p95 < 100 ms | no aplica (no modifica `internal/api`) |
| Sincronía de informe, protocolo, Spin y Word (`cmd/informe`, `cmd/word`, `spin/verificar.sh`) | ✅ (`TestInformeTB2AlDia`, `TestRegionProtocoloEsDelProtocoloReal`, `TestEquilibrioClusterConElModeloLineal`, `TestInformesSinAvisos` y `spin/verificar.sh` en verde) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3.0 / 3 | Completa el perfil exploratorio de la capa bronze (sección 6) y sustenta empíricamente el corte en 2011, la codificación de variables y la prevención de fuga de datos. |
| Implementación | 8 | documenta Partes A, B, C (clúster) y Spin | Explica con ecuaciones exactas, diagramas UML 2.5, código fuente incrustado, ejemplos reales del protocolo TCP, verificación formal con Spin y análisis de escalabilidad/equilibrio los dos niveles de paralelismo (goroutines + nodos TCP). |
| Interfaz | 5 | enlaza capturas admin | Reserva con nombre exacto `e2e-09-admin-nodo-caido.png` y `e2e-10-admin-metricas.png` y documenta el costo del observador WebSocket (`@tab:observador`). |
| Informe | 2 | avance sólido (secciones 1–11 y 22 listas) | Atiende de lleno las observaciones 2 y 5 de la PC4, mantiene cero cifras a mano, cumple APA 7 y protege el armado final en `scripts/paso70.sh`. |
| Video | 2 | sin cambios | No afecta el video. |

## Hallazgos

### H1 — Detalles puntuales de texto y notas en `07-limpieza.md`, `09-cluster.md` y `cluster.tmpl` · MENOR *(no bloqueante)*

- **Dónde:** `docs/informe/tb2/07-limpieza.md:210`, `docs/informe/tb2/09-cluster.md:379-381` y `docs/informe/tb2/plantillas/cluster.tmpl:9`
- **Evidencia:**
  1. En `docs/informe/tb2/07-limpieza.md:210`, el párrafo introductorio dice *«con 1, 2, 4, 8 y 16 workers»*, mientras que la `@tab:bench-loader` (`07-limpieza.md:217-222`, generada desde `reports/bench_loader_deez-wsl-debian.json`) y la `@fig:speedup-loader` incluyen también la fila de **32 workers** (`2 × runtime.NumCPU()` en `cmd/loader/main.go:141`, igual que `08-modelo.md:216` indica *«de 1 a 32 workers»*).
  2. En `docs/informe/tb2/09-cluster.md:379-381`, el texto y el título de `@fig:admin-caida` (`capturas/e2e-09-admin-nodo-caido.png`) dicen *«caída y reasignación de su shard»*. Sin embargo, en `web/e2e/vigia.e2e.ts:152-165` (donde el entrenamiento corre con `Pausa (ms, demo) = 20`), si el corte cae durante la pausa entre dos épocas, la bitácora registra `salió del clúster` seguido de la reconexión `se registró` en lugar de `reasignacion:`, tal como ya contempla `docs/guias/jose-70.md:106` (*«la caída y el reingreso (o la reasignación de su shard)»*).
  3. En `docs/informe/tb2/plantillas/cluster.tmpl:9` (`11-escalabilidad.md:40-46`), para 1 nodo `ms por época` es `72,35` (media recortada al 20 %), mientras que `Cómputo del nodo más lento` (`70,85`) + `Red y espera` (`2,36`) suman `73,21` porque las columnas de desglose son medias aritméticas de las 5 repeticiones (`desglose_por_epoca.seg_epoca = 73,22 ms`, explicado en `reports/bench_cluster_deez_gold.json:240`).
- **Corrección sugerida (opcional en los bloques 4–5):** Cambiar `07-limpieza.md:210` a *«de 1 a <!--v:(index .LoaderRef.Mediciones 5).Workers-->32<!--/v--> workers»* (o *«1 a 32 workers»*); matizar `09-cluster.md:379-381` como *«caída y recuperación (reasignación de su shard o reconexión)»*; y añadir en la `Nota:` de `cluster.tmpl:9` que las columnas de desglose son medias aritméticas sobre las repeticiones.

### H2 — Asignación duplicada en `cmd/informe/sintesis.go` y plantilla `coeficientes.tmpl` aún sin referenciar · MENOR *(no bloqueante)*

- **Dónde:** `cmd/informe/sintesis.go:232-240` y `docs/informe/tb2/plantillas/coeficientes.tmpl:1-9`
- **Evidencia:**
  1. En `cmd/informe/sintesis.go:232-233` y `239-240`, `funciones["num"]` y `funciones["inv"]` se registran dos veces con la misma definición dentro de `init()`.
  2. `docs/informe/tb2/plantillas/coeficientes.tmpl` se añade en este PR, pero ningún archivo `.md` incluye todavía `<!-- generado:plantilla:coeficientes -->` (`grep -l "plantilla:coeficientes" docs/informe/tb2/*.md` vacío).
- **Corrección sugerida (opcional en los bloques 4–5):** Eliminar las dos líneas repetidas en `cmd/informe/sintesis.go:232-233` y enlazar `<!-- generado:plantilla:coeficientes -->` en el bloque 5 (`17-evaluacion.md` o `18-impacto.md`), o retirarla si no se usará.

## Seguimiento de rondas previas

Primera ronda para este PR.

## Cómo explicarlo en la sustentación

Las secciones 6 a 11 documentan el núcleo concurrente y distribuido del sistema sin una sola cifra escrita a mano: cada tabla e indicador se genera desde `reports/*.json` y cada fragmento de código Go o Promela se extrae directamente del repositorio (`TestInformeTB2AlDia`). Para responder a las observaciones 2 y 5 de la PC4, la sección 9 genera y valida mensajes reales de los siete tipos del protocolo TCP, la sección 10 desglosa las seis aserciones, la fórmula LTL de terminación y los cuatro mutantes de `spin/barrera.pml` con sus contraejemplos, y la sección 11 sustenta la escalabilidad de 1 a 5 nodos mostrando la dispersión de las repeticiones, el desglose entre cómputo y red, el punto de equilibrio y el código completo del benchmark junto con su reporte JSON.
