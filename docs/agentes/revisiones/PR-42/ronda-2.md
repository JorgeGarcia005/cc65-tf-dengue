# Veredicto — PR-42 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/32-benchmark-cluster` @ `1cc94a6`
- **Fecha:** 2026-11-20
- **Decisión:** APROBADO

Los **9 hallazgos de la ronda 1** (3 bloqueantes, 4 mayores y 2 menores) quedaron **resueltos** con pruebas reproducibles y con el reporte del gold real regenerado (`reports/bench_cluster_jorge-pc_gold.json`) más su figura SVG (`docs/informe/pc4/fig/speedup_cluster.svg`). Todas las comprobaciones locales (`gofmt`, `go vet`, `go test -race -count=1`, `TestEquivalencia*`, `go run ./cmd/informe`, pruebas de auditoría con `-overlay` y verificación aritmética de JSON y SVG) y los 6 jobs del CI de GitHub Actions (`44686506017`) están en verde. Se registran únicamente 2 observaciones **MENORES** no bloqueantes y notas de pulido para el informe PC4 (#33).

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ pkgs=$(go list ./... | grep -Ev 'almacen/(mongo|redis)|cmd/coordinator')
$ go vet $pkgs
(vacío = ok; nota: el proxy del entorno local devuelve 403 al descargar
github.com/klauspost/compress@v1.16.7 para el driver de MongoDB; el módulo
completo está verificado en el CI de GitHub Actions, run 44686506017)

$ go test -race -count=1 $pkgs
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	4.809s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.014s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.160s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.417s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.033s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.381s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.773s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.886s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.043s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.037s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.047s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.123s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.015s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.602s

$ go test -race -count=1 -run Equivalencia -v ./internal/cluster/... ./internal/ml/... ./internal/gold/...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.47s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=1 (0.13s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=2 (0.10s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=3 (0.11s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=5 (0.12s)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.09s)
--- PASS: TestEquivalencia (0.20s)

$ go run ./cmd/informe && git diff docs/informe/
(salida de diff vacía: docs/informe/pc4/fig/speedup_cluster.svg reproducible bit a bit
 y docs/informe/pc3/ sin cambios)

$ go run ./cmd/benchmark -cluster -filas 40000 -features 12 -nodos 3,1,2,2 -reps-cluster 3 -epocas-cluster 5 -salida /tmp/bench_r2_test.json
 nodos     ms/época          IC95 ms  speedup    efic.      K-F   cómp. ms red+esp ms JSON/shard ms     KB/época
     1         2.93    2.90–2.96         1.00     1.00        —       2.78       0.14          0.06          3.4
     2         2.32    1.51–2.91         1.26     0.63    0.584       1.62       0.69          0.03          3.7
     3         2.18    2.05–2.28         1.34     0.45    0.617       1.44       0.73          0.03          4.1

$ go test -race -count=1 -v -overlay <audit_r2> -run TestAuditR2 ./internal/cluster
--- PASS: TestAuditR2BytesNoBajanAlCaerNodoEnMedio (0.08s)
    antes=(934,12591) tras_caida=(1431,16791) final=(1466,16791) reasign=1
--- PASS: TestAuditR2SegCalculoMaxDosNodosCuatroShards (0.08s)
    2 nodos x 4 shards: epoca=0.026052s calcMax=0.009522s calcSuma=0.016706s ratio=0.570

$ gh pr checks 42
Docker (imagen, clúster y caída de un nodo)	pass	2m1s
Ecosistema de agentes	pass	4s
Go (gofmt, vet, test -race)	pass	46s
MongoDB y Redis (tests de integración)	pass	49s
OpenAPI 3 válido	pass	17s
Spin (modelo del protocolo y mutantes)	pass	1m28s
```

Además, un script de verificación sobre `reports/bench_cluster_jorge-pc_gold.json` y `docs/informe/pc4/fig/speedup_cluster.svg` confirmó: coincidencia exacta de filas y features con `reports/split.json` (`765 297 × 31`), `speedup`, `eficiencia`, `karp_flatt`, `seg_red_y_espera` y `fraccion_red_y_espera` exactos dentro de `1e-12`, `seg_json_shard_mas_pesado < seg_red_y_espera` en las cinco configuraciones (`2,340 < 2,622 ms` hasta `0,473 < 0,783 ms`), `seg_json_cpu_todos_los_shards` estable (`2,195–2,340 ms`), y las cinco coordenadas SVG (`cx, cy`) idénticas al JSON.

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (local en los 21 paquetes stdlib; módulo completo verde en CI run `44686506017`) |
| `go test -race ./...` | ✅ (carreras: 0 en los 21 paquetes stdlib locales y en la suite completa de CI) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` en verde y verificada bit a bit en cada repetición del benchmark) |
| Latencia API p95 < 100 ms | no aplica (no toca `internal/api`) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica el planteamiento ni la pregunta de impacto social. |
| Implementación | 8 | Completo para el alcance de #32 | Protocolo TCP e instrumentación 100 % stdlib (`go.mod` intacto), sin carreras ni deadlocks, contadores de bytes monótonos incluso ante caída de nodos, cómputo por nodo correcto con múltiples shards y equivalencia bit a bit exigida en cada repetición. |
| Interfaz | 5 | no aplica | No toca API ni SPA. |
| Informe | 2 | Completo para el alcance de #32 | `reports/bench_cluster_jorge-pc_gold.json` sin cifras a mano ni datos en el repo (`AGENTS.md` §3.6–§3.7), desglose coherente sin cuantización de 0,5 ms y figura `docs/informe/pc4/fig/speedup_cluster.svg` generada reproduciblemente por `cmd/informe`. |
| Video | 2 | no aplica | Entregable TB2. |

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1 · BLOQUEANTE) | RESUELTO | `cmd/benchmark/cluster.go:190-192` registra `defer wg.Wait()` antes que `defer c.Cerrar()`, asegurando la salida LIFO `Terminar → Cerrar → wg.Wait`. Verificado por `TestBenchmarkClusterSinTodosLosNodosVuelveConError` (`cmd/benchmark/main_test.go:81-98`). |
| H2 (ronda 1 · BLOQUEANTE) | RESUELTO | `internal/cluster/coordinador.go:123, 206-210, 601-606` acumula `pc.Bytes()` en `bytesEnvCerradas`/`bytesRecCerradas` bajo `muConns` antes de `delete(c.conns, id)`. Verificado al cerrar en `TestTiemposConservaLosBytesDeConexionesCerradas` (`cluster_test.go:736-752`) y ante caída de nodo a mitad del entrenamiento en `TestAuditR2BytesNoBajanAlCaerNodoEnMedio` (`reasign=1`, totales estrictamente crecientes). |
| H3 (ronda 1 · BLOQUEANTE) | RESUELTO | `cmd/informe/figuras.go:148-208` y `cmd/informe/main.go:516-572` generan `docs/informe/pc4/fig/speedup_cluster.svg` desde `reports/bench_cluster_*.json` sin alterar `docs/informe/pc3/`. Verificado `go run ./cmd/informe` con diff vacío y coordenadas exactas respecto al gold JSON. |
| H4 (ronda 1 · MAYOR) | RESUELTO | `cmd/benchmark/cluster.go:265-318` mide parciales reales (`ml.NuevoPool(nil, cfg, 1).Parciales`) con `json.NewDecoder` y `pasadasPorMedicion`, separando `seg_json_cpu_todos_los_shards` y `seg_json_shard_mas_pesado`. En el gold regenerado (`1cc94a6`), el CPU total queda plano (`2,19–2,34 ms`) y el shard más pesado (`2,34 → 0,47 ms`) queda siempre por debajo de `seg_red_y_espera` (`2,62 → 0,78 ms`). |
| H5 (ronda 1 · MAYOR) | RESUELTO | `internal/cluster/cluster_test.go:725` y `cmd/benchmark/main_test.go:51` aceptan `>= 0` para `SegReduccion` (resolución de reloj en Windows) conservando la cota `< SegEpoca`. |
| H6 (ronda 1 · MAYOR) | RESUELTO | `internal/cluster/coordinador.go:113, 282, 322-327, 527` acumula `c.calcNodo[j] += m.MicrosCalculo` por nodo y toma el máximo entre nodos. Verificado con 1 nodo × 3 shards (`TestTiemposSumanLosShardsDeUnMismoNodo`) y 2 nodos × 4 shards (`TestAuditR2SegCalculoMaxDosNodosCuatroShards`, `calcMax / calcSuma = 0,570`). |
| H7 (ronda 1 · MAYOR) | RESUELTO | `cmd/benchmark/cluster.go:116-121, 133-141` clona, ordena y deduplica `op.nodos`, y solo define `KarpFlatt` cuando `base.Nodos == 1 && p > 1`. Verificado por `TestBenchmarkClusterOrdenaNodosYKarpFlattSoloContraUnNodo` (`main_test.go:102-124`) y ejecución CLI con `-nodos 3,1,2,2`. |
| H8 (ronda 1 · MENOR) | RESUELTO | `cmd/benchmark/main.go:95` fija `-recorte-cluster` por defecto en `0.20` (recorta 1 extremo por lado con 5 repeticiones; reflejado en el gold JSON) y `cluster.go:91-92, 350` aclara que el desglose usa medias aritméticas. |
| H9 (ronda 1 · MENOR) | RESUELTO | `internal/cluster/protocolo/protocolo.go:82` documenta `(MicrosCalculo, opcional)` y `protocolo_test.go:356, 455-493` cubre `us_calculo negativo` y `TestConexionCuentaBytesEnviadosYRecibidos` bajo `-race`. |

## Hallazgos (nuevos, no bloqueantes)

### H10 — `-recorte-cluster` no se valida al inicio del CLI ni en `medirCluster` · MENOR

- **Dónde:** `cmd/benchmark/main.go:95-99` y `cmd/benchmark/cluster.go:106-108, 252`
- **Evidencia:**
  `main.go:97-99` valida `*recorte` en `[0, 0.5)`, pero el nuevo flag `-recorte-cluster` no se valida antes de entrenar. Al ejecutar `go run ./cmd/benchmark -cluster -filas 1000 -features 4 -nodos 1 -reps-cluster 2 -epocas-cluster 1 -recorte-cluster 0.6`, el benchmark corre la referencia secuencial, el calentamiento y todas las repeticiones del primer nodo antes de fallar con pánico en `estadistica.ValidarRecorte`:
  ```text
  panic: estadistica: recorte = 0.6 (debe estar en [0, 0.5))
  ```
- **Corrección sugerida:** Validar `op.recorte >= 0 && op.recorte < 0.5` al inicio de `medirCluster` (y/o en `main.go` junto a `-recorte`) para fallar rápido con un error limpio. Opcional antes del merge o en un ajuste menor posterior.

### H11 — Metadatos `protocolo.nodos` y `protocolo.karp_flatt` no reflejan la normalización de H7 · MENOR

- **Dónde:** `cmd/benchmark/cluster.go:344, 351`
- **Evidencia:**
  Al ejecutar con `-nodos 3,1,2,2`, `mediciones` queda ordenado y deduplicado (`[1, 2, 3]`), pero `rep.Protocolo["nodos"]` guarda el slice original `[3, 1, 2, 2]`. Además, `rep.Protocolo["karp_flatt"]` mantiene `"p = nodos / nodos de la base"` aunque `medirCluster:138` ahora solo define Karp-Flatt cuando `base.Nodos == 1`. No afecta al reporte del gold real (`nodos = [1, 2, 3, 4, 5]`).
- **Corrección sugerida:** Guardar en `Protocolo["nodos"]` la lista normalizada (o los `Nodos` de `med`) y redactar `Protocolo["karp_flatt"]` como `e = (1/S − 1/p) / (1 − 1/p), definido solo contra base de 1 nodo`.

### Notas para el informe PC4 (#33) y el merge (informativas)

- **Etiqueta y leyenda en `figuras.go` (preexistente desde PC3):** `cmd/informe/figuras.go:222` ejecuta `strings.Replace(fmt.Sprintf("máx. %.2f×", mejor), ".", ",", 1)`, lo que reemplaza el punto de `máx.` y genera `máx, 4.09×` en `docs/informe/pc4/fig/speedup_cluster.svg:36` (igual que en `pc3/fig/speedup_*.svg`). Además, `cpuCorta` (`figuras.go:368-376`) no acorta `"AMD Ryzen 5 3600 6-Core Processor"`, por lo que la leyenda sobresale del recuadro blanco de `290 px` (`speedup_cluster.svg:37, 41`). Conviene pulirlo al redactar #33.
- **Descripción del PR #42:** Antes del *squash merge*, conviene marcar como cumplido el cuarto criterio de aceptación (`speedup_cluster.svg`) y anotar que el comentario inicial del PR corresponde a la corrida de la ronda 1 (`100,30 ms`, `4,24×`), reemplazada por el gold regenerado (`92,00 ms`, `4,09×`) en `1cc94a6`.

## Cómo explicarlo en la sustentación

El benchmark levanta el coordinador y de 1 a 5 nodos reales sobre TCP en loopback con 1 worker por nodo (emulando máquinas de un núcleo), exige igualdad bit a bit de pesos e historia contra el secuencial en cada repetición, y descompone cada época en cómputo del nodo más lento, reducción y red más espera. Sobre el dataset gold real (765 297 filas, 187 bloques, media recortada al 20 % sobre 5 repeticiones de 30 épocas), escalar de 1 a 5 nodos reduce la época de 92,00 ms a 22,48 ms (speedup 4,09×, eficiencia 82 %, Karp-Flatt 0,055–0,079), con ~97 % del tiempo en cómputo y ~112–116 KB/época porque siempre viajan los 187 parciales por bloque. Además, medir la serialización JSON con los parciales reales y pasadas agrupadas muestra que el costo total de CPU es casi constante (~2,2–2,3 ms) y que en el camino crítico se divide entre nodos (`2,34 ms → 0,47 ms`), quedando por debajo de la red y espera total (`2,62 ms → 0,78 ms`), de modo que la sobrecarga de red + barrera en loopback es de solo ~0,25–0,32 ms por época.
