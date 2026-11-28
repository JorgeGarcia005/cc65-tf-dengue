# Veredicto — PR-92 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/91-metricas-cluster` @ `7ad8153`
- **Fecha:** 2026-11-27
- **Decisión:** CAMBIOS

El PR incorpora las métricas exigidas por el enunciado TB2 §6 en el panel administrador («uso CPU, latencia, número de nodos, logs»): medición de CPU del proceso en Unix con solo stdlib (`syscall.Getrusage(RUSAGE_SELF)`), latencia por nodo y época con desglose en cálculo y red, búfer circular de `1024` latencias de predicción en la API sin asignaciones en caliente (`p95 = 4,30 ms` fría y `3,94 ms` caliente a `~8 800 RPS` con JWT), actualización de OpenAPI y regeneración idempotente de `docs/informe/pc4/`. La suite con `-race`, los tests de equivalencia bit a bit y los 7 jobs del CI pasan en verde. No obstante, se emiten **2 hallazgos bloqueantes** y **2 menores**: en `Coordinador.despachar` la marca `envioNodo[j]` se toma **después** de `enviarTarea`, perdiendo el orden *happens-before* frente al cálculo del nodo, reportando `LatenciaMs < CalculoMs` y haciendo fallar `TestMetricasPorNodoTrasEntrenar` en **7 de 30 corridas** (**H1**); y en la SPA `conGradiente` compara `EvGradiente.epoca` (**0-based**: `0, 1, …`) directamente con `MetricasNodo.epoca` (**1-based**: `1, 2, …`), duplicando `calculoMs`, anulando `usoCpu` a `—` y dejando que el sondeo de `/api/v1/cluster` pise las métricas en vivo (**H2**).

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./... && go vet -tags integracion ./...
(vacío = ok)

$ go run ./cmd/informe && git diff --exit-code docs/informe
(sin diferencias)

$ GOOS=windows go build ./internal/cluster ./cmd/node && GOOS=darwin go build ./internal/cluster ./cmd/node
(compila limpio en Linux, Windows y macOS)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	11.134s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	9.322s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	6.508s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.033s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/evaluacion	1.658s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.250s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.222s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.408s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/usuarios	3.253s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/web	1.047s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.150s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	26.531s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth	12.126s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.521s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.887s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/evaluacion	1.894s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.067s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.191s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	12.355s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ws	1.750s

$ go test -race -count=1 -run Equivalencia -v ./...
--- PASS: TestEquivalenciaClusterSecuencialBitABit (1.36s) [nodos=1, 2, 3, 5]
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.17s)
--- PASS: TestEquivalencia (0.46s) [workers=1, 2, 4, 8, 16, 64]

$ go test -count=30 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
--- FAIL: TestMetricasPorNodoTrasEntrenar (0.02s)
    metricas_test.go:36: métricas de b: {Epoca:4 Shards:1 CalculoMs:0.416 LatenciaMs:0.018 RedMs:0 UsoCPU:0xc00015a8d0} (épocas 4)
--- FAIL: TestMetricasPorNodoTrasEntrenar (0.02s)
    metricas_test.go:36: métricas de b: {Epoca:4 Shards:1 CalculoMs:0.407 LatenciaMs:0.021 RedMs:0 UsoCPU:0xc000012df0} (épocas 4)
--- FAIL: TestMetricasPorNodoTrasEntrenar (0.03s)
    metricas_test.go:53: gradiente sin métricas: {V:1 Tipo:gradiente T:2026-11-28 02:11:58.307298207 +0000 UTC m=+0.177091627 Epoca:2 Nodo:a Shard:0xc000012c78 Perdida:0 Pesos:[] Micros:357 MicrosCPU:359 MicrosIdaVuelta:27 Detalle: Nodos:[]}
(... 7 fallos en 30 corridas; ver H1 ...)
FAIL	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	0.744s

$ BENCH_CLAVE=... go run ./cmd/benchapi -url http://127.0.0.1:18089 -correo analista@vigia.pe -exigir-p95 100ms
corrida      pet.    err.      RPS   p50 ms   p95 ms   p99 ms   máx ms    HIT   MISS
fria         2000       0     8828     1.33     4.30     6.91     9.52      0      0  p95 < 100 ms: true
caliente     2000       0     8669     1.40     3.94     6.59    12.75      0      0  p95 < 100 ms: true

$ (en web/) npm ci && npx tsc -b && npx vitest run && npm run build && ! grep -q "url(data:" dist/assets/*.css
Test Files  12 passed (12)
     Tests  179 passed (179)
✓ built in 758ms · sin url(data: en CSS

$ gh pr checks 92 -R JorgeGarcia005/cc65-tf-dengue   # run 46079155586 sobre 7ad8153
7/7 jobs en verde (Ecosistema, Spin, OpenAPI 3, MongoDB/Redis, Go -race, SPA, Docker)
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet (`./...` y `-tags integracion`) | ✅ |
| `go test -race ./...` | ⚠️ (pasa en una corrida con `-race`, pero `TestMetricasPorNodoTrasEntrenar` falla `7/30` veces sin `-race` por **H1**; carreras de memoria: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` idéntico bit a bit) |
| Latencia API p95 < 100 ms | ✅ (`4,30 ms` fría / `3,94 ms` caliente con JWT y 16 clientes concurrentes) |
| SPA (`tsc -b`, Vitest, `vite build`, CSP) | ✅ (`179/179` tests existentes en verde, sin `url(data:` en CSS; falta cubrir el caso real 0-based/1-based de **H2**) |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | sin cambios | No altera el planteamiento ni la partición de datos. |
| Implementación | 8 | Pendiente **H1** | Núcleo solo con stdlib (`syscall.Getrusage`), `latencias` O(1) sin contención apreciable en `/predicciones` y equivalencia bit a bit intacta; falta corregir el instante de medición en `despachar` (**H1**). |
| Interfaz | 5 | Pendiente **H2** | Cubre CPU, latencia con desglose, nodos sanos y p95 de la API en el panel admin, pero el desfase `0-based` vs `1-based` en `epoca` corrompe las métricas en vivo al cruzarlas con `/api/v1/cluster` (**H2**). |
| Informe | 2 | al día | Regiones extraídas en `docs/informe/pc4/08-distribuido.md` y `10-api.md` sincronizadas con `cmd/informe` (recordar regenerarlas tras aplicar **H1**). |
| Video | 2 | mejora la demo (tras **H1** y **H2**) | En la demostración en vivo (`ritmo_ms = 20`, sondeo cada `2 s`), corregir **H2** evita que los KPIs de cálculo/CPU de los nodos dupliquen valores o muestren `CPU —` entre épocas. |

## Hallazgos

### H1 — `despachar` registra `envioNodo[j]` *después* de `enviarTarea`, perdiendo el orden *happens-before* frente al nodo, reportando `LatenciaMs < CalculoMs` y haciendo fallar `TestMetricasPorNodoTrasEntrenar` · BLOQUEANTE

- **Dónde:** `internal/cluster/coordinador.go:53-56, 459-468` y `internal/cluster/metricas_test.go:35, 52`
- **Evidencia:**
  1. En `Coordinador.despachar` (`coordinador.go:459-468`), el coordinador primero escribe `asignacion`/`epoca` por el socket TCP (`c.enviarTarea(j, s)`) y recién al retornar ejecuta `ahora := time.Now()` y asigna `c.envioNodo[j] = ahora`.
  2. En cuanto `Write` entrega los bytes al socket loopback, el kernel despierta en otro núcleo a la goroutine/proceso del nodo (`internal/cluster/nodo.go:128-132`), que toma `ini := time.Now()`, ejecuta `n.pool.Parciales` y escribe `gradiente` **en paralelo** al retorno de `enviarTarea`. Si el coordinador sufre una pausa de planificación antes de la línea 464, `c.envioNodo[j]` queda registrado cuando el nodo ya lleva calculando —o ya terminó— su shard.
  3. Al correr sin el sobrecoste de ThreadSanitizer (como corre el binario estático `CGO_ENABLED=0` en producción), el defecto se reproduce de inmediato en **7 de 30 ejecuciones**:
     ```text
     $ go test -count=30 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster
     --- FAIL: TestMetricasPorNodoTrasEntrenar (0.02s)
         metricas_test.go:36: métricas de b: {Epoca:4 Shards:1 CalculoMs:0.416 LatenciaMs:0.018 RedMs:0 UsoCPU:0xc00015a8d0} (épocas 4)
     --- FAIL: TestMetricasPorNodoTrasEntrenar (0.03s)
         metricas_test.go:53: gradiente sin métricas: {V:1 Tipo:gradiente ... Epoca:2 Nodo:a ... Micros:357 MicrosCPU:359 MicrosIdaVuelta:27}
     ```
     El nodo calculó durante `416 µs`, pero el coordinador midió apenas `18 µs` de ida y vuelta (`27 µs` frente a `357 µs` en `EvGradiente`). Además, tomar la marca después de `enviarTarea` excluye la codificación JSON y escritura TCP del coordinador que `Admin.tsx:423-424` y `docs/api/openapi.yaml:498-499` atribuyen a `latencia_ms` / `red_ms`.
- **Corrección sugerida:**
  - En cada intento del bucle de `despachar` (`internal/cluster/coordinador.go:459`), tomar `inicioEnvio := time.Now()` **antes** de `c.enviarTarea(j, s)` y, tras un envío exitoso, registrar `if _, ok := c.envioNodo[j]; !ok { c.envioNodo[j] = inicioEnvio }` (dejando `c.vence[s] = time.Now().Add(c.op.TimeoutEpoca)` después del envío como hoy, y manteniendo `max(lat-calc, 0)` en `cerrarMetricas` como defensa ante deriva de reloj entre máquinas físicas distintas).
  - Ajustar el comentario de `MetricasNodo` (`coordinador.go:54`) para indicar que `LatenciaMs` cuenta desde que el coordinador inicia el envío de la primera tarea de la época al nodo, regenerar `docs/informe/pc4/08-distribuido.md` con `go run ./cmd/informe`, y verificar estabilidad con `go test -count=100 -run TestMetricasPorNodoTrasEntrenar ./internal/cluster`.

### H2 — Desfase *off-by-one* entre `EvGradiente.epoca` (`0-based`) y `MetricasNodo.epoca` (`1-based`) duplica `calculoMs`, anula `usoCpu` a `—` y deja que `/api/v1/cluster` pise las métricas en vivo · BLOQUEANTE

- **Dónde:** `web/src/admin/estado.ts:165-186, 229-236` (frente a `internal/cluster/coordinador.go:403, 415, 643, 772` e `internal/cluster/eventos.go:37`, y en `web/src/pruebas/metricasNodo.test.ts:56-61`, `web/src/pruebas/admin.test.tsx:150, 168`)
- **Evidencia:**
  1. En Go, durante la $E$-ésima época en curso ($E \ge 1$), `c.epoca` vale **`E - 1`** (`0, 1, 2, …`): `EvGradiente` (`coordinador.go:643`) emite `Epoca: E - 1` (como se aprecia también en la salida real de `TestMetricasPorNodoTrasEntrenar`: para 4 épocas emite `Epoca: 0..3`). Recién al cumplirse la barrera se ejecuta `c.epoca++` (`coordinador.go:403`) y después `c.cerrarMetricas()` (`coordinador.go:415, 772`), por lo que `MetricasNodo.epoca` en `GET /api/v1/cluster` vale **`E`** (`1, 2, 3, …`).
  2. En la SPA, `vistaDe` (`estado.ts:166`) guarda `epoca: m.epoca` (**1-based**), mientras `conGradiente` (`estado.ts:176, 183`) guarda `epoca: ev.epoca` (**0-based**) y compara directamente `n.vista?.epoca === ev.epoca` y `previo.vista.epoca > n.metricas.epoca` como si usaran la misma numeración.
  3. Durante el uso real del panel (que sondea `GET /api/v1/cluster` cada `SONDEO_MS = 2000 ms`, `Admin.tsx:173-190`), esto provoca tres fallos visibles:
     - **Suma cruzada entre dos épocas distintas (`calculoMs` duplicado):** cuando `/api/v1/cluster` responde tras cerrarse la época $k$ (p. ej. al abrir el panel con `metricas.epoca = 600`, o en cualquiera de las pausas `ritmo_ms = 20` del modo demo), `conInstantanea` fija `vista = { epoca: k, calculoMs: C, ... }`. Al llegar enseguida el primer `EvGradiente` de la época $k+1$ en curso, trae `ev.epoca === k` (`(k+1) - 1`). En `conGradiente` (`estado.ts:176`), `const misma = n.vista?.epoca === ev.epoca` da **`true`** (`k === k`), por lo que suma el `calculo_ms` completo de la época $k$ con el shard de la época $k+1$ (`calculoMs` supera a `latenciaMs` y fuerza `redMs = 0`).
     - **Pérdida de CPU (`CPU —`) en la época siguiente a una instantánea:** cuando `n.vista` provino de `GET /api/v1/cluster` al abrir el panel (`n.microsCpuEpoca === undefined`), en ese mismo `EvGradiente` con `ev.epoca === k` (`misma === true`), `previa` es `undefined` y la línea 179 evalúa `cpuUs = null`, mostrando **`CPU —`** aunque el nodo sí informó `micros_cpu`. Y si `microsCpuEpoca` quedó de una época anterior en `previo` (porque `conInstantanea` hace `{ ...previo, ...n, vista }` sin reiniciar `microsCpuEpoca`), suma la CPU de dos épocas distintas.
     - **Sobreescritura de datos vivos por una instantánea vieja:** mientras corre la época $k+1$, sus `EvGradiente` fijan `previo.vista.epoca = k`; si llega la respuesta de `/api/v1/cluster` correspondiente a la época $k$ recién cerrada (`n.metricas.epoca = k`), `!(previo.vista.epoca > n.metricas.epoca)` evalúa `!(k > k) === true` y pisa la época $k+1$ en curso con la época $k$ anterior.
  4. Los tests `metricasNodo.test.ts:58-59` y `admin.test.tsx:150, 168` no atraparon el problema porque alimentaron `gradiente` con `epoca: 9` y `epoca: 8` (1-based) tras `metricas.epoca = 7`, cuando el coordinador real emite `epoca: 7` para la 8.ª época.
- **Corrección sugerida:**
  - Normalizar la época en `conGradiente` (`web/src/admin/estado.ts:176, 183`) a numeración 1-based (`const epoca = ev.epoca + 1`, documentando que `EvGradiente.epoca` es 0-based como en `internal/cluster/eventos.go:37`, mientras `MetricasNodo.epoca` es 1-based). Con `epoca = ev.epoca + 1`, todo `EvGradiente` de la época $E$ lleva `epoca = E`, mientras que cualquier instantánea de `/api/v1/cluster` anterior al cierre de $E$ trae `n.metricas.epoca <= E - 1 < E`, y cuando $E$ ya se cerró ningún `EvGradiente` futuro vuelve a traer `E` (traerá $\ge E + 1$). Además, en `conInstantanea` (`estado.ts:234`), limpiar `microsCpuEpoca: undefined` cuando `vista` proviene de `vistaDe(n.metricas)` (y considerar también el caso en que el coordinador se haya reiniciado y `c.epocas < previo.vista.epoca`).
  - Actualizar `web/src/pruebas/admin.test.tsx` y `web/src/pruebas/metricasNodo.test.ts` usando la numeración real del coordinador (`metricas.epoca = 7` seguido de `gradiente` de la 8.ª época con `epoca: 7`), verificando que reemplaza sin sumar con la época 7 y mantiene `usoCpu` válido.

### H3 — `MicrosCPU == 0` se codifica igual que «sin soporte de CPU» y anula el uso de CPU de toda la época en shards sub-milisegundo · MENOR

- **Dónde:** `internal/cluster/protocolo/protocolo.go:110`, `internal/cluster/nodo.go:133-135`, `internal/cluster/coordinador.go:634-638` y `internal/cluster/metricas_test.go:39, 52`
- **Evidencia:** `protocolo.Mensaje.MicrosCPU` es `int64` con `omitempty` y `coordinador.go:634` exige `m.MicrosCPU > 0` (si vale `0`, marca `c.cpuNodo[j] = -1`; en `estado.ts:179`, `!ev.micros_cpu` también convierte `0` en `null`). Sobre el gold real (~763k filas) cada shard consume decenas de milisegundos de CPU, pero en shards muy pequeños o si el kernel cuantiza `getrusage` a `0 µs` en un cálculo ultrarrápido, un nodo Unix válido reportaría `0` y haría fallar `metricas_test.go:52` (`g.MicrosCPU <= 0`) o mostraría `CPU —`.
- **Corrección sugerida:** Usar `*int64` en `MicrosCPU` (presente también cuando vale `0` en Unix y `nil` cuando `cpuProceso()` devuelve `false`), aceptando `>= 0` en el coordinador y en TypeScript (`ev.micros_cpu !== undefined`), o bien relajar la guarda a `>= 0` donde corresponda documentando la convención.

### H4 — `docs/api/openapi.yaml` no menciona los campos nuevos de `cluster.Evento` en `/api/v1/ws` ni declara `required` en `MetricasAPI` / `MetricasNodo` · MENOR

- **Dónde:** `docs/api/openapi.yaml:274-279, 478-501`
- **Evidencia:** La descripción de `GET /api/v1/ws` enumera los campos especiales de `cluster.Evento`, pero no menciona `micros_cpu` ni `micros_ida_vuelta` en `gradiente`; y en `MetricasAPI` y `MetricasNodo` todos los campos salvo `uso_cpu` son siempre emitidos por Go y exigidos por las guardas `esMetricasAPI` / `esMetricasNodo` (`web/src/admin/estado.ts:136-148, 322-331`), pero figuran como opcionales en el esquema OpenAPI.
- **Corrección sugerida:** Añadir una mención breve de `micros_cpu` y `micros_ida_vuelta` en la descripción de `/api/v1/ws` y agregar `required` con los campos fijos en `MetricasAPI` y `MetricasNodo`.

## Seguimiento de rondas previas

Primera ronda para este PR (no aplica).

## Cómo explicarlo en la sustentación

Cada nodo lee el tiempo de CPU acumulado de su proceso con `syscall.Getrusage(RUSAGE_SELF)` justo antes y después de calcular los parciales de su shard en el worker pool, y envía esa diferencia junto con el tiempo de cómputo puro en el mensaje `gradiente` sin tocar los pesos ni el orden de reducción. En el coordinador, el bucle de épocas mide para cada nodo la latencia de extremo a extremo desde el envío de su primera tarea de la época hasta la recepción de su último gradiente, desglosándola en cómputo y sobrecosto de red/serialización/cola, y calcula el uso de CPU sobre la capacidad de sus `workers`. En paralelo, la API registra en `O(1)` la duración de las últimas `1024` predicciones en un búfer circular de tamaño fijo protegido por mutex —calculando `p50` y `p95` fuera de la sección crítica al consultar `GET /api/v1/cluster`— para que el panel administrador contraste en vivo el `p95` interno frente al umbral de `100 ms`.
