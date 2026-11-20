# Veredicto — PR-42 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/32-benchmark-cluster` @ `ec55b35`
- **Fecha:** 2026-11-20
- **Decisión:** CAMBIOS

El PR instrumenta el protocolo y el coordinador solo con biblioteca estándar, añade el modo `-cluster` en `cmd/benchmark` con verificación bit a bit contra el secuencial en cada repetición, y aporta la corrida sobre el gold real (`reports/bench_cluster_jorge-pc_gold.json`, 765 297 filas × 31 variables, coherente con `reports/split.json`). Sin embargo, presenta **3 hallazgos bloqueantes** (un deadlock en el cierre por error de `medirNodos`, pérdida/deltas negativos de bytes en `Coordinador.Tiempos()` al cerrar o caer nodos, y el criterio #4 de #32 —`speedup_cluster.svg`— sin implementar ni trasladado a #33 pese a `Cierra #32`), además de **4 hallazgos mayores** y **2 menores** sobre la medición en Windows, el cálculo de `SegCalculoMax` con varios shards por nodo y el manejo de `-nodos`/Karp-Flatt.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ pkgs=$(go list ./... | grep -Ev 'almacen/(mongo|redis)|cmd/coordinator')
$ go vet $pkgs
(vacío = ok; nota: go vet ./... y go test ./... completos requieren descargar
github.com/klauspost/compress@v1.16.7 para el driver de MongoDB y el proxy del
entorno local devolvió 403 Forbidden; se corrobora el módulo completo en el CI
de GitHub Actions, run 44683042216, todos los jobs en verde)

$ go test -race -count=1 $pkgs
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.203s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.156s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.399s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.326s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.573s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.629s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.045s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.068s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.363s

$ go test -race -count=1 -run Equivalencia -v ./internal/cluster/... ./internal/ml/... ./internal/gold/...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.49s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=1 (0.14s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=2 (0.12s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=3 (0.12s)
    --- PASS: TestEquivalenciaClusterSecuencialBitABit/nodos=5 (0.12s)
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.09s)
--- PASS: TestEquivalencia (0.20s)

$ go run ./cmd/benchmark -cluster -filas 40000 -features 12 -nodos 1,2,3 -reps-cluster 3 -epocas-cluster 5 -salida /tmp/bench_cluster_test.json
sintetico(filas=40000, features=12, semilla=2026)
 nodos     ms/época          IC95 ms  speedup    efic.      K-F   cómp. ms red+esp ms    JSON ms     KB/época
     1         2.88    2.87–2.89         1.00     1.00        —       2.75       0.13       0.05          3.4
     2         2.07    1.50–2.41         1.39     0.70    0.439       1.53       0.54       0.06          3.7
     3         1.83    1.81–1.86         1.58     0.53    0.451       1.06       0.77       0.06          4.1

$ gh pr checks 42
Docker (imagen, clúster y caída de un nodo)	pass	2m31s
Ecosistema de agentes	pass	5s
Go (gofmt, vet, test -race)	pass	47s
MongoDB y Redis (tests de integración)	pass	47s
OpenAPI 3 válido	pass	21s
```

Las comprobaciones adicionales de H1, H2, H4, H5, H6 y H7 se ejecutaron con `go test -race -count=1 -overlay` sobre pruebas temporales (sin modificar archivos del repositorio) y un script de verificación aritmética sobre `reports/bench_cluster_jorge-pc_gold.json`.

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (local en los 21 paquetes stdlib; módulo completo verde en CI run `44683042216`) |
| `go test -race ./...` | ✅ (carreras: 0 en los 21 paquetes stdlib locales y en la suite completa de CI) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` en verde y cada repetición del benchmark exige igualdad bit a bit) |
| Latencia API p95 < 100 ms | no aplica (no toca `internal/api`) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica el planteamiento ni la pregunta de impacto social. |
| Implementación | 8 | Parcial (bloqueado por H1 y H2) | El protocolo TCP, la instrumentación con stdlib y la verificación bit a bit sobre el gold real (1–5 nodos, 4,24× speedup) están bien encaminados, pero el deadlock en `medirNodos` (H1) y la pérdida de bytes en `Coordinador.Tiempos()` ante cierre/caída (H2) afectan la corrección concurrente. |
| Interfaz | 5 | no aplica | No toca API ni SPA. |
| Informe | 2 | Parcial (bloqueado por H3; afectado por H4 y H5) | Aporta `reports/bench_cluster_jorge-pc_gold.json` sin cifras a mano ni datos en el repo (`AGENTS.md` §3.6 y §3.7), pero falta la figura `docs/informe/pc4/fig/speedup_cluster.svg` exigida en #32 (H3) y el desglose de JSON/reducción en Windows necesita ajustes antes de llevarlo al informe (H4, H5). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H1 — Deadlock en `medirNodos` por orden invertido de `defer c.Cerrar()` y `defer wg.Wait()` · BLOQUEANTE

- **Dónde:** `cmd/benchmark/cluster.go:162-179`
- **Evidencia:**
  `medirNodos` registra los `defer` en este orden:
  ```go
  defer c.Cerrar()        // línea 162 → se ejecuta DESPUÉS de wg.Wait()
  var wg sync.WaitGroup
  defer wg.Wait()         // línea 164 → se ejecuta ANTES de c.Cerrar()
  ...
  if err := c.EsperarNodos(n); err != nil {
      return medicionCluster{}, err
  }
  defer c.Terminar()      // línea 179 → solo se registra si EsperarNodos tuvo éxito
  ```
  Como `defer` ejecuta en orden LIFO, `wg.Wait()` corre **antes** que `c.Cerrar()`. Si `net.Dial` falla en `i >= 1` (línea 168) o `c.EsperarNodos(n)` retorna error (línea 177), `defer c.Terminar()` nunca se registra: los goroutines `Nodo.Servir(conn)` ya lanzados quedan bloqueados sin plazo en `pc.Recibir(0)` (`internal/cluster/nodo.go:55`), mientras `medirNodos` bloquea para siempre en `wg.Wait()` sin llegar jamás a `c.Cerrar()`. Lo mismo sucede si `c.Terminar()` falla al escribir `fin` a un nodo (ignora el error de `Enviar` y no cierra conexiones).
  Reproducido con `go test -race -overlay` (`TestAuditDeferOrderDeadlock`, replicando las líneas 162–179 con `TimeoutRegistro = 100 ms` y 1 de 2 nodos):
  ```text
  === RUN   TestAuditDeferOrderDeadlock
      audit_juez_test.go:144: DEADLOCK CONFIRMADO: wg.Wait() bloqueado indefinidamente antes de ejecutar defer c.Cerrar()
  ```
  Contrasta con `internal/cluster/cluster_test.go:108-111`, donde `cp.c.Cerrar()` sí se llama antes de `cp.wg.Wait()`.
- **Corrección sugerida:** Declarar `var wg sync.WaitGroup` y `defer wg.Wait()` **antes** de `defer c.Cerrar()` para que el orden de salida (LIFO) sea siempre `c.Terminar()` (si se registró) → `c.Cerrar()` → `wg.Wait()` → `os.RemoveAll(dir)`. Agregar un test que verifique que `medirNodos` (o un helper equivalente) retorna sin colgarse cuando falla el registro o un nodo.

### H2 — `Coordinador.Tiempos()` pierde los bytes de conexiones cerradas y devuelve deltas negativos si cae un nodo · BLOQUEANTE

- **Dónde:** `internal/cluster/coordinador.go:200-205` y `internal/cluster/coordinador.go:583-595`
- **Evidencia:**
  `Coordinador.Tiempos()` promete acumular «los bytes de todas las conexiones con nodos» (`coordinador.go:568-569`), pero solo suma `pc.Bytes()` sobre las conexiones todavía presentes en el mapa `c.conns` (`coordinador.go:588`). En `c.leer` (`coordinador.go:200-205`), el `defer` ejecuta `delete(c.conns, id)` al cerrarse una conexión sin sumar sus bytes a ningún acumulado.
  Consecuencia comprobada con `go test -race -overlay`:
  ```text
  === RUN   TestAuditTiemposPierdeBytesAlCerrar
      audit_juez_test.go:28: ANTES de Cerrar:  env=1800 rec=16649 epocas=4
      audit_juez_test.go:29: DESPUÉS de Cerrar: env=0 rec=0 epocas=4
  === RUN   TestAuditTiemposBytesNegativosAlCaerNodo
      audit_juez_test.go:51: ANTES (5 ep):   env=2164 rec=20677
      audit_juez_test.go:52: DESPUÉS (+1 ep con caída): env=1482 rec=14504
      audit_juez_test.go:53: DELTA en esa época: env=-682 rec=-6173
  ```
  1) Cualquier llamador que lea `c.Tiempos()` después de `c.Terminar()` y `c.Cerrar()` obtiene `BytesEnviados = 0` y `BytesRecibidos = 0` (por eso `TestTiemposSeparanComputoReduccionYBytes` en `cluster_test.go:716-717` tuvo que leer antes de `Terminar()`).
  2) Si un nodo cae durante el entrenamiento y su shard se reasigna (comportamiento central de ADR-0005), el contador no es monótono y el cálculo por diferencias de `cmd/benchmark/cluster.go:206` produce valores **negativos**.
- **Corrección sugerida:** Añadir campos acumulados (p. ej. `bytesEnvCerrados`, `bytesRecCerrados int64` protegidos por `c.muConns`) y en el `defer` de `c.leer`, bajo `c.muConns.Lock()` antes de `delete(c.conns, id)`, acumular `pc.Bytes()`. En `Tiempos()`, inicializar `BytesEnviados`/`BytesRecibidos` con esos acumulados y sumar las conexiones activas. Cubrir con tests en `internal/cluster/cluster_test.go` ambos casos: lectura tras `Terminar()+Cerrar()` y caída de un nodo en medio del entrenamiento.

### H3 — Criterio de aceptación #4 del issue #32 (`speedup_cluster.svg`) sin implementar ni registrado en #33 pese a `Cierra #32` · BLOQUEANTE

- **Dónde:** Descripción del PR #42 (`Cierra #32`), `cmd/informe/main.go:473-515`, `cmd/informe/figuras.go`
- **Evidencia:**
  El issue #32 (`gh issue view 32`) define explícitamente cuatro criterios de aceptación verificables; el cuarto es:
  > `Figura SVG de speedup por nodos generada por cmd/informe — verificar con: go run ./cmd/informe && test -f docs/informe/pc4/fig/speedup_cluster.svg`
  El PR declara `Cierra #32` (cierre automático al hacer merge) y marca ese criterio como pendiente con la justificación «se hace en #33 con el JSON del gold real». Sin embargo:
  1. `reports/bench_cluster_jorge-pc_gold.json` **ya está versionado en este PR** (commit `ec55b35`), eliminando el bloqueo que justificaba diferir la figura.
  2. Al revisar el cuerpo del issue #33 (`gh issue view 33`), **ninguno de sus criterios menciona `speedup_cluster.svg` ni `reports/bench_cluster_*.json`**.
  3. `cmd/informe/main.go:473-515` solo genera las figuras de `docs/informe/pc3/fig/` y `docs/informe/pc4/fig/speedup_cluster.svg` no existe. Fusionar el PR cerraría #32 dejando un criterio verificable sin cumplir y sin rastro en el tablero.
- **Corrección sugerida:** Implementar en este PR la generación de `docs/informe/pc4/fig/speedup_cluster.svg` desde `reports/bench_cluster_*.json` en `cmd/informe` (reutilizando `lienzo` en `cmd/informe/figuras.go` con eje X lineal `1..5` nodos y recta ideal `y = x`) y verificar `go run ./cmd/informe && test -f docs/informe/pc4/fig/speedup_cluster.svg`. Como alternativa, si el equipo y el orquestador deciden moverlo a #33, agregar explícitamente ese criterio verificable en el cuerpo del issue #33 antes de fusionar.

### H4 — `seg_serializacion_json` supera en 1,8×–2,3× a `seg_red_y_espera` en el gold real, sufre cuantización de 0,5 ms en Windows y usa payload sintético sesgado · MAYOR

- **Dónde:** `cmd/benchmark/cluster.go:78, 228-258, 262, 290` y `reports/bench_cluster_jorge-pc_gold.json:53-54, 83-84, 113-114, 143-144, 173-174`
- **Evidencia:**
  En `reports/bench_cluster_jorge-pc_gold.json` (Windows), los valores registrados para 1–5 nodos son:
  | Nodos | `seg_red_y_espera` (ms) | `seg_serializacion_json` (ms) | Relación JSON / (red+espera) |
  |---:|---:|---:|---:|
  | 1 | 2,734 | 2,0003 | 0,73× |
  | 2 | 1,382 | 2,5125 | **1,82×** |
  | 3 | 1,166 | 2,5213 | **2,16×** |
  | 4 | 0,871 | 1,9979 | **2,29×** |
  | 5 | 0,776 | 1,5225 | **1,96×** |

  Se combinan tres problemas verificados:
  1. **Unidades mezcladas en `desglose_por_epoca` e `imprimirCluster`:** aunque `protocolo.desglose` aclara «JSON medido aparte (CPU)», `serializacionPorEpoca` mide la **suma secuencial en 1 hilo de los N shards** (comparable a `seg_calculo_todos_los_nodos`), mientras `seg_red_y_espera` mide **reloj de pared en el camino crítico** (donde los `N` nodos serializan en paralelo y las `N` lectoras del coordinador decodifican solapadas con el cómputo del nodo más lento). Mostrar `red+esp ms = 1,17` junto a `JSON ms = 2,52` en la misma tabla produce una contradicción aparente en la sustentación.
  2. **Cuantización del reloj de Windows (~0,5 ms):** `serializacionPorEpoca` (`cluster.go:243-256`) cronometra una sola pasada (~2 ms) 5 veces sin agrupar, ignorando `pasadasPorMedicion(..., medicionMinima)` ya implementado en el mismo paquete (`cmd/benchmark/main.go:258`). Por eso los cinco valores en el JSON del gold son múltiplos exactos de 0,5 ms (`2,00`, `2,51`, `2,52`, `2,00`, `1,52 ms`, una oscilación artificial de 65 % para los mismos 187 bloques).
  3. **Payload sintético dependiente de `N` en sentido inverso y decodificador distinto:** en `cluster.go:236`, `1.0 / float64(3+j+k)` reinicia `k` en 0 para cada shard; al medir su tamaño con `TestAuditSerializacionYReduccionFormaGold` (765 297 × 31, 187 bloques), los bytes sintéticos **bajan** de `126 810 B` (1 nodo) a `119 645 B` (5 nodos) —mientras el tráfico real del gold **sube** de `115 225 B` a `118 388 B`—, y además usa `json.Unmarshal` (`cluster.go:251`) en vez del `json.NewDecoder` que usa `protocolo.Conexion.Recibir` (`protocolo.go:266`).
- **Corrección sugerida:**
  1) Usar `pasadasPorMedicion(..., medicionMinima)` en `serializacionPorEpoca` para eliminar la cuantización de 0,5 ms en Windows.
  2) Construir los parciales de prueba a partir de parciales reales (p. ej. una pasada de `ml.ParcialBloque` con los pesos de referencia, o al menos indexando `k` por índice global de bloque) y decodificar con `json.NewDecoder`.
  3) Distinguir explícitamente en el esquema/tabla que esa métrica es **CPU total acumulada en los N shards** (y opcionalmente añadir el costo por shard del camino crítico, que sí es acotado por `seg_red_y_espera`). Regenerar `reports/bench_cluster_jorge-pc_gold.json` tras el ajuste.

### H5 — Tests frágiles en Windows (`SegReduccion > 0`) evidenciado por `seg_reduccion = 0` en el reporte gold · MAYOR

- **Dónde:** `internal/cluster/cluster_test.go:724`, `cmd/benchmark/main_test.go:50`, `reports/bench_cluster_jorge-pc_gold.json:52` y `:142`
- **Evidencia:**
  En el propio `reports/bench_cluster_jorge-pc_gold.json` (generado en Windows con 187 bloques × 31 variables y 150 épocas por configuración), `seg_reduccion` valió **exactamente `0`** en 1 nodo (línea 52) y 4 nodos (línea 142), `0,00000667 s` (= `1,00 ms / 150`, 2 ticks de 0,5 ms) en 2 y 3 nodos, y `0,00001019 s` (= `1,53 ms / 150`, 3 ticks) en 5 nodos. Esto ocurre porque `ReducirEnOrden + Finalizar` sobre 187 bloques tarda decenas de microsegundos (en Linux con `-race` medimos `129,6 µs/época` en `TestAuditSerializacionYReduccionFormaGold`, y sin `-race` ~10–20 µs), muy por debajo del salto de ~0,5 ms del reloj de Go en Windows (`cmd/benchmark/main.go:250-252`).
  Sin embargo, ambos tests nuevos imponen `tm.SegReduccion > 0` (`cluster_test.go:724`, solo 20 bloques × 6 variables × 8 épocas) y `d.Reduccion > 0` (`main_test.go:50`, 30 bloques × 12 variables × 8 épocas), lo que pasa en Linux CI pero resulta intermitente/falla en Windows, donde trabaja el equipo (`PROTOCOLO.md`).
- **Corrección sugerida:** Relajar la aserción de reducción en ambos tests a `>= 0` (manteniendo `< SegEpoca`). Si se desea reportar un `seg_reduccion` no nulo y fiable en Windows dentro del JSON del benchmark, medir `ReducirEnOrden + Finalizar` agrupado con `pasadasPorMedicion` (o documentar en `protocolo.desglose` que la reducción por época dura decenas de microsegundos y en Windows puede caer bajo la resolución del reloj).

### H6 — `SegCalculoMax` calcula el máximo por *shard* y no por *nodo* cuando un nodo tiene varios shards · MAYOR

- **Dónde:** `internal/cluster/coordinador.go:515`, `internal/cluster/coordinador.go:570, 574` y `cmd/benchmark/cluster.go:74`
- **Evidencia:**
  El comentario de `Tiempos` (`coordinador.go:570`) y el campo `seg_calculo_nodo_mas_lento` (`cluster.go:74`) definen `SegCalculoMax` como el cómputo del **nodo más lento** de cada época. Sin embargo, `atender` (`coordinador.go:515`) actualiza `c.calcMax = max(c.calcMax, m.MicrosCalculo)` por cada shard recibido. Como cada nodo procesa sus shards **en serie** en una sola goroutine (`internal/cluster/nodo.go:54-68`), si `nodos < shards` o tras la caída de un nodo que transfiere shards a un sobreviviente, el cómputo del nodo más lento es la suma de los shards de ese nodo, no el máximo de un shard aislado.
  Comprobado con `go test -race -overlay` (`TestAuditSegCalculoMaxConVariosShardsPorNodo`, 2 nodos × 4 shards, 40 ms por shard = 80 ms de cómputo/nodo/época):
  ```text
  2 nodos × 2 shards (40ms/shard → 80ms/nodo/época): seg_epoca=102.191ms, seg_calc_max=40.000ms, seg_calc_suma=160.000ms, red_y_espera_aparente=62.183ms
  ```
  Con 1 shard por nodo sin caídas ambos coinciden, pero con 2 shards por nodo `SegCalculoMax` subestima a la mitad el cómputo del nodo más lento y atribuye ~40 ms de cómputo real a `red_y_espera`.
- **Corrección sugerida:** Acumular por índice de nodo durante la época (`c.calcPorNodo[j] += m.MicrosCalculo`) y tomar el máximo entre nodos al cerrar la época. Coincide exactamente con el caso 1:1 del benchmark y corrige el caso general (`nodos < shards` o tras reasignaciones).

### H7 — `-nodos` sin ordenar toma `out[0]` como base y Karp-Flatt es matemáticamente incorrecto si `base.Nodos > 1` · MAYOR

- **Dónde:** `cmd/benchmark/cluster.go:110-121` y `cmd/benchmark/cluster.go:291`
- **Evidencia:**
  1. El comentario en `cluster.go:110-111` indica «contra la configuración de 1 nodo (o la menor medida)», pero `base := out[0]` asume que `-nodos` viene ordenado de menor a mayor. Con `nodos: []int{3, 2, 1}` (`TestAuditOrdenNodosYKarpFlatt`), `medirCluster` tomó `3` como base y produjo resultados inválidos:
     ```text
     nodos=3 speedup=1.0000 efic=1.0000 kf=<nil>
     nodos=2 speedup=2.0157 efic=3.0236 kf=<nil>
     nodos=1 speedup=1.0720 efic=3.2161 kf=<nil>
     ```
  2. Incluso con `-nodos` ordenado pero sin `1` (p. ej. `-nodos 2,4`), las líneas 114–118 aplican `e = (1/s - 1/p) / (1 - 1/p)` con `p = out[i].Nodos / base.Nodos` y `s = T(base)/T(n)`. Bajo la ley de Amdahl $T(n) = T(1)(f + (1-f)/n)$, esa fórmula evalúa a $\frac{f}{f + (1-f)/n_0} \neq f$: por ejemplo, para $n_0 = 2$ y una fracción serial real $f = 0,05$, devuelve `0,0952` (casi el doble de $f$).
- **Corrección sugerida:** Copiar, ordenar y deduplicar `op.nodos` antes del bucle en `medirCluster` para garantizar que `base` sea el mínimo; definir `KarpFlatt` únicamente cuando `base.Nodos == 1` (o emplear la ecuación exacta respecto a $n_0$, y actualizar `protocolo.karp_flatt` acorde). Añadir tests unitarios para `-nodos` desordenado y para `base.Nodos > 1`.

### H8 — El estimador declarado «media recortada 10 %» con `n = 5` no recorta ningún valor y `desglose_por_epoca` no aplica recorte · MENOR

- **Dónde:** `cmd/benchmark/main.go:83, 92`, `cmd/benchmark/cluster.go:202-212, 220, 288` y `reports/bench_cluster_jorge-pc_gold.json:16, 27, 36-38`
- **Evidencia:**
  Con los valores por defecto `-reps-cluster 5` y `-recorte 0.10`, `estadistica.MediaRecortada` calcula `k := int(float64(5) * 0.10) = 0` (`internal/estadistica/estadistica.go:35`): descarta **0** observaciones en cada extremo (es la media aritmética simple), aunque `protocolo.estimador` afirma «media recortada 10 % + IC95 bootstrap (2000)». Además, si un usuario ejecuta con `-reps-cluster 10` (donde sí recorta 1 por extremo), `seg_por_epoca.media_recortada` recorta valores atípicos mientras `desglose_por_epoca` (`cluster.go:202-212`) promedia sin recorte.
- **Corrección sugerida:** O bien reflejar en `protocolo.estimador` los elementos efectivamente recortados (`k = ⌊n·recorte⌋`), o subir el recorte por defecto del modo `-cluster` a `0.20` (con `n = 5`, recorta 1 por extremo) o `-reps-cluster` a `10`; y, si se aplica recorte, usar el mismo criterio o explicitar que `desglose_por_epoca` reporta medias aritméticas.

### H9 — Faltan pruebas unitarias en `internal/cluster/protocolo` para `us_calculo` negativo y `Conexion.Bytes()` · MENOR

- **Dónde:** `internal/cluster/protocolo/protocolo.go:82, 149-151, 212-214` y `internal/cluster/protocolo/protocolo_test.go`
- **Evidencia:**
  El PR añade `MicrosCalculo` (`us_calculo`), la regla `m.MicrosCalculo < 0` en `Mensaje.Validar()` y el método `Conexion.Bytes()`, pero `internal/cluster/protocolo/protocolo_test.go` no fue modificado: no hay prueba unitaria de que `us_calculo < 0` retorne `ErrInvalido`, ni de ida y vuelta de `us_calculo`, ni de exactitud y acceso concurrente a `Conexion.Bytes()`. Además, la tabla de campos del comentario en `protocolo.go:82` no lista `MicrosCalculo` (opcional).
- **Corrección sugerida:** Añadir el caso `"gradiente con us_calculo negativo"` en `TestMensajeDeTipoDesconocidoOIncompletoSeRechaza`, un test de `Conexion.Bytes()` bajo `-race` en `protocolo_test.go`, y mencionar `MicrosCalculo` (opcional) en `protocolo.go:82`.

## Seguimiento de rondas previas

Primera ronda del PR #42.

## Cómo explicarlo en la sustentación

El benchmark levanta el coordinador y de 1 a 5 nodos reales sobre TCP en loopback con 1 worker por nodo (emulando máquinas de un núcleo), valida bit a bit los pesos y la pérdida de cada repetición contra el entrenamiento secuencial, y mide el tiempo por época, speedup, eficiencia y fracción serial de Karp-Flatt. En el dataset gold real (765 297 filas, 187 bloques), pasar de 1 a 5 nodos baja la época de 100,3 ms a 23,7 ms (speedup 4,24×, eficiencia 85 %), con más del 96 % del tiempo en cómputo y ~113–116 KB por época porque siempre viajan los 187 parciales por bloque. Nota para la defensa: el equipo debe estar listo para explicar (1) por qué Karp-Flatt baja de 0,076 en 3 nodos a 0,045 en 5 nodos (en una misma CPU, el cómputo total sube ~11 % al pasar de 1 a varios núcleos por frecuencia turbo y ancho de banda de memoria, y ese costo fijo se amortiza al crecer `p`, mientras la red en el camino crítico baja al repartirse la serialización de los 187 bloques), y (2) la diferencia entre el tiempo CPU total de JSON y la red+espera en el camino crítico (H4).
