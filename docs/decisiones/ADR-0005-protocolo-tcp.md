# ADR-0005 — Protocolo coordinador–nodos por TCP

- **Estado:** Aceptado (07/11/2026) · Issue #9 · Se implementa en PC4
- **Debate:** propuesta del implementador
  ([`propuestas/protocolo-tcp.md`](propuestas/protocolo-tcp.md)) y contrapropuesta del juez
  ([`propuestas/protocolo-tcp-juez.md`](propuestas/protocolo-tcp-juez.md)). Sintetiza el orquestador.

## Contexto

El parameter server (coordinador) y los nodos de ML deben entrenar por épocas con un gradiente
**idéntico bit a bit** al secuencial (AGENTS.md §3.3), sin deadlock ni pérdida silenciosa de un
shard cuando un nodo cae o se demora, y solo con la biblioteca estándar (§3.2). La topología
(estrella con parameter server síncrono), los shards de bloques contiguos (ADR-0004) y el worker
pool por nodo (ADR-0002) ya estaban decididos.

## Decisiones

| Tema | Decisión | Origen |
|---|---|---|
| Qué envía el nodo | Un `ml.Parcial` **por bloque**, sin reducir; el coordinador los ubica por índice global de bloque y aplica `ReducirEnOrden` + `Finalizar` | Propuesta §2; el juez acepta |
| Transporte | TCP, una conexión persistente iniciada por el nodo hacia `coordinator:9000` | Propuesta §3; el juez acepta |
| Framing | JSON por línea con `json.Encoder`/`json.Decoder` (no `bufio.Scanner`, que corta en 64 KiB); `NaN`/`±Inf` → mensaje `error` | Propuesta §3; el juez acepta |
| Mensajes | `registro`, `asignacion`, `listo`, `epoca`, `gradiente`, `error`, `fin`; `epoca` y `gradiente` llevan `epoca` y `shard` | Propuesta §4 |
| Concurrencia del coordinador | Una goroutine lectora por conexión → canal `respuestas`; **un solo escritor** (el bucle de épocas); estado de la época local a esa goroutine, sin mutex; la API lee una copia bajo `sync.RWMutex` | Propuesta §5 |
| Envíos | Todo envío pasa por `enviar(j, msg) error` con `SetWriteDeadline`; un error de escritura es una caída del nodo `j` | Propuesta §5 (Spin, mutante 1) |
| Fallos | Reasignar solo los shards faltantes **en la misma época**; descarte en **dos pasadas** (primero se descartan todos los dueños vencidos, después se reasigna) | Propuesta §6 (Spin, mutante 2); el juez acepta |
| **Plazo por shard** | Cada envío de `epoca` para el shard `s` fija `vence[s] = ahora + timeoutEpoca`. El temporizador se arma con el menor `vence[s]` pendiente; al dispararse solo se descarta al dueño de un shard **cuyo propio plazo venció** | Juez, objeción 1 (ajuste del orquestador: un plazo por shard en vez de reiniciar un temporizador global) |
| Caída por EOF | Se reasignan **todos** los shards pendientes del nodo caído (`duenio[s] == i && !listo[s]`), no solo uno | Juez, objeción 1 |
| Reintento al asignar | `asignarA(s)` recorre los nodos sanos hasta que un envío funcione; si falla el envío a `j`, descarta a `j` y prueba el siguiente; si no queda ninguno, `fin{abortado}`. Se usa tanto al difundir la época como al reasignar | Juez, objeción 2 |
| Elección del nodo sano | El de **menor `carga[j]`** (shards asignados); empate → menor índice. Mantiene el reparto determinista; los pesos no dependen de esta elección porque la reducción usa el índice global de bloque | Juez, objeción 4 |
| Estado del nodo | `shards map[int]gold.Archivo` y **un único `ml.Pool`** por proceso. `internal/ml` expone `Pool.Parciales(ds *Dataset, w []float64) []Parcial`, que reutiliza los mismos workers para cualquier shard | Juez, objeción 3 |
| Recarga de shards | El coordinador lleva `cargado[j][s]`; envía `asignacion` antes de `epoca` solo si `!cargado[j][s]`. Un `listo` que llega en medio de una época solo marca `cargado[j][s] = true`. TCP conserva el orden, así que el nodo siempre procesa `asignacion` antes que la `epoca` del mismo shard | Juez, objeción 3 |

## Opciones descartadas

- **Reducción local en el nodo** (un parcial por nodo): menos tráfico, pero el resultado dependería
  del número de nodos (la suma de `float64` no es asociativa).
- **Parameter server asíncrono** y **ring all-reduce**: no reproducibles bit a bit; detalle en la
  propuesta §7.
- **gRPC**: no es stdlib. **`gob`**: más compacto, pero el JSON se audita con `tcpdump` y en los
  logs, útil para depurar y para la sustentación.
- **Seguir sin el shard caído**, **esperar indefinidamente** o **replicar siempre cada shard**:
  rompen la equivalencia, arriesgan un deadlock o duplican el cómputo (propuesta §6).
- **Reiniciar el temporizador global al reasignar** (variante del juez): también resuelve la
  objeción 1, pero un plazo por shard es más fácil de razonar y de probar, y no alarga el plazo de
  los nodos que no tuvieron nada que ver con la caída.

## Consecuencias

- Costo estimado de los ajustes del debate: ~1,5 h adicionales en `internal/cluster` (estimación
  del juez).
- `internal/ml` cambia en PC4: `Pool` deja de quedar atado a un único `*Dataset` y gana
  `Parciales`; `Pool.Gradiente` se conserva para el loader y el benchmark.
- El informe de PC3 (sección 8) cita este ADR como decisión tomada tras el debate.

## Verificación en PC4

1. **Equivalencia:** coordinador + 1, 2, 3 y 5 nodos en el mismo proceso por `127.0.0.1`; pesos
   idénticos bit a bit a `ml.Entrenar(ml.NuevoSecuencial(...))`, con `-race` (propuesta §8).
2. **Ida y vuelta JSON** de `Parcial`, incluidos subnormales y `-0`; `NaN` produce `error`.
3. **Caída escalonada (EOF + plazo)**, criterio 1 del juez: N = 3, `timeoutEpoca = 200 ms`; el
   nodo 1 se cuelga, el nodo 2 cierra la conexión a los 160 ms. El entrenamiento **no aborta**,
   el nodo 0 termina los shards 1 y 2 y los pesos son idénticos al secuencial.
4. **Balanceo tras doble caída**, criterio 2 del juez: N = 4; caen los nodos 2 y 3 en la época 0;
   en las épocas siguientes `carga[0] == 2` y `carga[1] == 2`.
5. **Nodos falsos:** cierra la conexión, no responde, responde dos veces o con una época vieja;
   el resultado es igual al secuencial o el entrenamiento aborta con error, nunca se cuelga.
6. **Modelo Spin:** actualizar `spin/barrera.pml` para que (a) el envío sea no bloqueante y
   descarte al nodo si su canal está lleno (`full(aNodo[j])`), en lugar de depender de un búfer
   de `2N`; (b) la elección del nodo sano use la menor carga; y (c) se modele la caída por EOF
   separada del vencimiento del plazo. Volver a correr seguridad, LTL y los mutantes.
