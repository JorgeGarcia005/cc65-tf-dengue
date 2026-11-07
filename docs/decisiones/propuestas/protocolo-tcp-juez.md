# Contrapropuesta del Juez — Protocolo coordinador–nodos por TCP

- **Issue:** #9 · **Autor:** Antigravity (Juez) · **Fecha:** 07/11/2026
- **En respuesta a:** [`docs/decisiones/propuestas/protocolo-tcp.md`](protocolo-tcp.md) y modelo [`spin/barrera.pml`](../../../spin/barrera.pml)
- **Decisión del debate:** De acuerdo con la arquitectura base; se formulan **4 objeciones** con escenarios de fallo concretos y una alternativa de ajuste quirúrgico antes de pasar a `ADR-0005-protocolo-tcp.md`.

---

## 1. Acuerdo

Acepto tal cual los siguientes puntos de la propuesta y las respuestas a las 3 preguntas planteadas en [`protocolo-tcp.md:200-206`](protocolo-tcp.md#L200-L206):

1. **Un `ml.Parcial` por bloque sin reducción local en el nodo (Pregunta 1, [`protocolo-tcp.md:24-44`](protocolo-tcp.md#L24-L44)):**
   Es la única opción que cumple [`AGENTS.md` §3.3](../../../AGENTS.md) (*"los bloques de datos tienen límites fijos (independientes del número de workers o nodos) y las sumas parciales se reducen en orden de índice de bloque"*). Redefinir el secuencial como una suma jerárquica por shard haría que el resultado dependiera del número de nodos $N$, violando `ADR-0002` ([`docs/decisiones/ADR-0002-worker-pool-gradiente.md:32-34`](../ADR-0002-worker-pool-gradiente.md#L32-L34)) y [`TestShardsReproducenElGradienteGlobal`](../../../internal/gold/gold_test.go#L58-L95). Enviar ~2 108 `float64` (~42 KB en JSON) por nodo y época en red local/Docker toma $< 1\text{ ms}$, despreciable frente al cómputo de ~250 000 filas.
2. **Transporte TCP con conexión saliente del nodo y framing JSON por línea con `json.Encoder`/`json.Decoder` (Pregunta 2, [`protocolo-tcp.md:48-60`](protocolo-tcp.md#L48-L60)):**
   De acuerdo en preferir JSON sobre `encoding/gob`. En la biblioteca estándar de Go, `strconv.FormatFloat(..., 'g', -1, 64)` preserva exactamente los 64 bits de cualquier `float64` finito en el viaje de ida y vuelta (incluidos subnormales y `-0.0`, que serializa como `"-0"` y `strconv.ParseFloat` reconstruye con el bit de signo intacto), permite auditar el tráfico con `tcpdump`/logs en la sustentación y evita el límite de 64 KiB de `bufio.Scanner`.
3. **Reasignación en la misma época y descarte en dos pasadas (Pregunta 3 y hallazgos de Spin, [`protocolo-tcp.md:132-142, 184-195`](protocolo-tcp.md#L132-L142), [`spin/barrera.pml:35-42, 70-92`](../../../spin/barrera.pml#L35-L92)):**
   De acuerdo en reasignar únicamente los shards faltantes dentro de la misma época y en los dos hallazgos verificados con Spin:
   - **Mutante 1 (envío bloqueante):** toda escritura del coordinador debe llevar `SetWriteDeadline` para que un socket con búfer lleno de un nodo caído no bloquee el bucle de épocas.
   - **Mutante 2 (reasignación en dos pasadas):** al vencer el plazo se debe marcar primero como descartados a **todos** los dueños que no entregaron su shard y recién en una segunda pasada reasignar a nodos sanos.

---

## 2. Objeciones

### Objeción 1 — Descarte falso de un nodo sano cuando concurren una caída por EOF y el timeout de época

- **Evidencia:** [`protocolo-tcp.md:109, 125-126, 132-137`](protocolo-tcp.md#L125-L137) y [`spin/barrera.pml:70-76, 118-119`](../../../spin/barrera.pml#L70-L76).
- **Qué falla y en qué escenario:**
  La tabla de §6 ([`protocolo-tcp.md:125`](protocolo-tcp.md#L125)) establece que cuando un nodo cae a mitad de época con EOF/reset, la goroutine lectora lo detecta al instante y el coordinador reasigna inmediatamente su shard a un nodo sano (`duenio[s] = j`). Sin embargo, el bucle de épocas tiene un único temporizador de época (`-timeout-epoca`, [`protocolo-tcp.md:109, 126`](protocolo-tcp.md#L109)) y, cuando este vence, la primera pasada de `reasignar()` ([`spin/barrera.pml:71-76`](../../../spin/barrera.pml#L71-L76)) descarta a ciegas al dueño actual de todo shard incompleto:
  ```promela
  for (s : 0 .. N-1) {
      if :: !listo[s] -> descartado[duenio[s]] = true :: else -> skip fi
  }
  ```
  **Escenario concreto ($N = 3$, `-timeout-epoca = 5 s`):**
  1. En $t = 0\text{ s}$ inicia la época $e$.
  2. En $t = 1\text{ s}$, el Nodo 0 entrega `shard 0` (`listo[0] = true`). El Nodo 1 queda colgado (`listo[1] = false`).
  3. En $t = 4.5\text{ s}$, el Nodo 2 muere con `EOF`. El coordinador lo detecta de inmediato, marca `descartado[2] = true`, reasigna el `shard 2` al Nodo 0 (`duenio[2] = 0`) y le envía `asignacion(shard 2)` + `epoca(e, shard 2)`.
  4. En $t = 5.0\text{ s}$ (solo $0.5\text{ s}$ después, mientras el Nodo 0 aún lee `shard 2` de disco con `gold.Leer` y calcula sus bloques), vence el temporizador de la época porque el Nodo 1 sigue colgado.
  5. Se ejecuta `reasignar()`: como `listo[2] == false` y `duenio[2] == 0`, la primera pasada ejecuta `descartado[0] = true`.
  6. **Resultado:** el coordinador **descarta al Nodo 0 (el único nodo sano que quedaba)** y aborta el entrenamiento (`j == N`).
- **Por qué Spin no lo detectó:** [`spin/barrera.pml`](../../../spin/barrera.pml) no modela la detección inmediata por EOF separada del timeout global, y fija `SOSPECHAS = 1` ([`spin/barrera.pml:31, 119`](../../../spin/barrera.pml#L31)). Además, si un nodo caído por EOF ya era dueño de **más de un shard** (por una reasignación en una época previa), [`protocolo-tcp.md:125`](protocolo-tcp.md#L125) dice *"reasigna su shard"* en singular; debe reasignar **todos** los shards $s$ tales que `duenio[s] == i && !listo[s]`.

### Objeción 2 — Manejo incompleto del error de escritura (Mutante 1) en el modelo Spin y durante `reasignar()`

- **Evidencia:** [`protocolo-tcp.md:104-108, 190`](protocolo-tcp.md#L104-L108) y [`spin/barrera.pml:35-42, 87, 108`](../../../spin/barrera.pml#L35-L42).
- **Qué falla y en qué escenario:**
  1. En [`spin/barrera.pml:42`](../../../spin/barrera.pml#L42), el bloqueo hallado por el Mutante 1 se evitó subiendo el búfer del canal `aNodo[N]` de `[N]` a `[2*N]`, en lugar de modelar un envío no bloqueante que descarta al nodo si el canal está lleno (`full(aNodo[j])`).
  2. En el diseño en Go ([`protocolo-tcp.md:106-107`](protocolo-tcp.md#L106-L107)), se indica que *"un error de escritura cuenta como caída del nodo"*, pero hay dos puntos de escritura y ninguno detalla la recuperación inmediata:
     - **Al difundir al inicio de la época ([`barrera.pml:108`](../../../spin/barrera.pml#L108)):** si `Encode` falla con error/deadline al enviar `epoca` a `duenio[s]`, marcar `descartado[duenio[s]] = true` sin reasignar ese shard en el acto obliga a esperar en vano todo el `-timeout-epoca`.
     - **Dentro de la segunda pasada de `reasignar()` ([`barrera.pml:87`](../../../spin/barrera.pml#L87)):** si al reasignar el shard `s` al nodo sano `j` falla la escritura hacia `j`, marcar `descartado[j] = true` y avanzar al siguiente shard `s+1` deja a `s` asignado a un nodo muerto (`duenio[s] = j`) sin haberle enviado la tarea a otro nodo sano.

### Objeción 3 — Sobrescritura del shard propio, multiplicación de `ml.Pool` y mensajes `listo` en medio de una época

- **Evidencia:** [`protocolo-tcp.md:66-69, 109-111, 115-118, 125`](protocolo-tcp.md#L66-L69) e [`internal/ml/concurrente.go:33-68`](../../../internal/ml/concurrente.go#L33-L68).
- **Qué falla y en qué escenario:**
  - En §5 ([`protocolo-tcp.md:115-116`](protocolo-tcp.md#L115-L116)) se define que el nodo, *"al recibir `asignacion` carga el shard con `gold.Leer`, crea un `ml.Pool` (ADR-0002) y responde `listo`"*.
  - Cuando al Nodo 0 (dueño de `shard 0`) se le reasigna `shard 2` en la época $e$, en las épocas siguientes ($e+1, e+2, \dots$) el coordinador le pedirá calcular **ambos** shards (`duenio[0] = 0` y `duenio[2] = 0`, [`spin/barrera.pml:87, 108`](../../../spin/barrera.pml#L87)).
  - **Riesgo A (estado del nodo):** Si `asignacion` sobrescribe un único `gold.Archivo` / `ml.Pool` en el nodo, en la época $e+1$ el Nodo 0 no tendrá `shard 0`. Y si en cambio guarda un `map[int]*ml.Pool` creando un `ml.Pool` nuevo por cada shard asignado, un nodo con $k$ shards lanzará $k \times \text{workers}$ goroutines permanentes compitiendo por los mismos núcleos (rompiendo el invariante de `ADR-0002`).
  - **Riesgo B (`listo` huérfano y recarga desde disco):** Al enviar `asignacion` + `epoca` seguidos en una reasignación ([`protocolo-tcp.md:125`](protocolo-tcp.md#L125)), el nodo responderá `listo` en medio de la época mientras el coordinador espera `gradiente`. La propuesta no especifica que el bucle de épocas ignore `listo` tardíos ni evita volver a enviar `asignacion` (y releer ~60 MB de disco) si un shard se vuelve a enviar a un nodo que ya lo había cargado.

### Objeción 4 — Concentración de todos los shards reasignados en el nodo `j = 0`

- **Evidencia:** [`spin/barrera.pml:80-88`](../../../spin/barrera.pml#L80-L88) y [`protocolo-tcp.md:163`](protocolo-tcp.md#L163).
- **Qué falla y en qué escenario:**
  En [`spin/barrera.pml:80-84`](../../../spin/barrera.pml#L80-L84), la segunda pasada busca siempre el primer nodo no descartado desde `j = 0`:
  ```promela
  j = 0;
  do :: j < N && descartado[j] -> j++ :: else -> break od;
  ```
  Con $N = 5$ nodos (caso exigido por [`protocolo-tcp.md:163`](protocolo-tcp.md#L163) en `TestEquivalencia`) y 2 nodos caídos, **todos** los shards huérfanos recaen en el Nodo 0 (que pasa a calcular 3 shards = 60 % de los bloques), mientras los Nodos 1 y 2 calculan solo 1 shard cada uno (20 %). Al ser una barrera síncrona, la duración de cada época la dicta el nodo más cargado ($\max_j \text{carga}(j)$), degradando el speedup y aumentando el riesgo de que el Nodo 0 supere `-timeout-epoca` en la siguiente época y dispare una cascada de descartes.

---

## 3. Alternativa

Mantener íntegra la arquitectura de la propuesta (estrella síncrona, TCP con JSON por línea, un `ml.Parcial` por bloque, reducción global con `ReducirEnOrden` y descarte en dos pasadas) incorporando estos **4 ajustes concretos**:

1. **Plazo por asignación (`enviadoEn[s]`) o reinicio del timer al reasignar (resuelve Objeción 1):**
   - El coordinador registra `enviadoEn[s] = time.Now()` cada vez que envía `epoca` para el shard `s` (al inicio de la época o al reasignarlo) y reinicia el `time.Timer` de época al hacer una reasignación por caída temprana (EOF/error).
   - En la primera pasada de `reasignar()` tras vencer el temporizador, solo se marca `descartado[duenio[s]] = true` si `!listo[s] && time.Since(enviadoEn[s]) >= timeoutEpoca`. Así, un nodo sano que acaba de recibir un shard reasignado por el EOF de otro nodo cuenta con su propio plazo completo para procesarlo.
   - Cuando un nodo `i` cae por EOF/error de lectura, se reasignan **todos** los shards `s` con `duenio[s] == i && !listo[s]`.
2. **Reintento inmediato ante error de escritura (resuelve Objeción 2):**
   - Encapsular el envío a un nodo `j` en un helper con `SetWriteDeadline`. Si falla al enviar (sea en la difusión inicial de la época o dentro de la segunda pasada de `reasignar()`), marcar `descartado[j] = true` de inmediato y, en un bucle interno sobre los candidatos sanos restantes, reasignar el shard a otro nodo sano hasta que el envío sea exitoso (o abortar si no quedan nodos sanos).
3. **Mapa de shards `map[int]gold.Archivo` y pool único reutilizable en el nodo (resuelve Objeción 3):**
   - El nodo mantiene `shards map[int]gold.Archivo` en memoria y **un único pool** de `workers` goroutines (añadiendo en `internal/ml` un método `Pool.Parciales(ds *Dataset, w []float64) []Parcial` o cambiando el dataset activo bajo el mutex del `Pool`).
   - El coordinador lleva `tieneShard[nodo][shard] bool`: solo envía `asignacion` antes de `epoca` si `!tieneShard[j][s]`. Si el bucle de épocas recibe un mensaje `listo`, simplemente marca `tieneShard[j][s] = true` y continúa esperando `gradiente`.
4. **Reasignación al nodo sano con menor cantidad de shards asignados (resuelve Objeción 4):**
   - En la segunda pasada de `reasignar()`, en lugar de elegir el primer `j` con `!descartado[j]`, elegir el nodo sano `j` que minimice `carga[j]` (cantidad de shards actualmente asignados a `j`), desempatando por menor índice `j` para conservar el determinismo.

- **Costo de implementación estimado:** **1,5 h** adicionales dentro de la implementación de `internal/cluster` (~40 líneas de Go sin dependencias externas).

---

## 4. Criterio de decisión

Dos pruebas concretas y reproducibles zanjarían cualquier diferencia entre la propuesta original y esta contrapropuesta:

1. **Contraejemplo de fallo escalonado (EOF + timeout) en test unitario de `internal/cluster`:**
   Con $N = 3$ nodos y `-timeout-epoca = 200 ms`:
   - El Nodo 0 responde `shard 0` en $t = 20\text{ ms}$ y tarda $80\text{ ms}$ por shard.
   - El Nodo 1 queda colgado (no responde nunca).
   - El Nodo 2 cierra su conexión (`EOF`) en $t = 160\text{ ms}$.
   - **Criterio:** El entrenamiento **no debe abortar**: debe descartar a los Nodos 1 y 2, permitir que el Nodo 0 termine los shards 1 y 2 y devolver un vector de pesos idéntico bit a bit a `ml.NuevoSecuencial`. (Sin el ajuste 1, en $t = 200\text{ ms}$ el timeout descarta al Nodo 0 cuando lleva apenas $40\text{ ms}$ calculando el `shard 2` y aborta la corrida).
2. **Test de balanceo tras doble caída con $N = 4$ nodos:**
   Al caer los Nodos 2 y 3 en la época 0, verificar que en las épocas siguientes los Nodos 0 y 1 reciben **2 shards cada uno** (`carga[0] == 2`, `carga[1] == 2`) en lugar de quedar `3` en el Nodo 0 y `1` en el Nodo 1.
