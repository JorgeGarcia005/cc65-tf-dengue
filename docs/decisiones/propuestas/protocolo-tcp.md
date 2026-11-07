# Propuesta — Protocolo coordinador–nodos por TCP

- **Issue:** #9 · **Autor:** Claude Code (implementador) · **Fecha:** 05/11/2026
- **Estado:** debatida; decisión final en
  [`ADR-0005-protocolo-tcp.md`](../ADR-0005-protocolo-tcp.md) (incorpora las 4 objeciones del juez)
- **Se implementa en:** PC4 (`cmd/coordinator`, `cmd/node`, `internal/cluster`)
- **ADR resultante:** `docs/decisiones/ADR-0005-protocolo-tcp.md` (el número 0002 que cita el
  issue ya lo usa `ADR-0002-worker-pool-gradiente.md`)

## 1. Qué hay que decidir

Cómo se comunican el coordinador (parameter server) y los nodos de ML para entrenar la
regresión logística por épocas, de modo que:

1. el gradiente distribuido sea **idéntico bit a bit** al secuencial (AGENTS.md §3.3);
2. no haya deadlock ni pérdida silenciosa de un shard si un nodo cae o se demora;
3. se pueda implementar solo con la biblioteca estándar (§3.2) y explicar en la sustentación.

Lo que ya está decidido y no se reabre: topología **estrella con parameter server síncrono**
(AGENTS.md §2 y §6), shards de **bloques contiguos** con `bloque_inicial` (ADR-0004, `internal/gold`)
y worker pool dentro de cada nodo (ADR-0002).

## 2. Restricción que manda: el orden de la suma

`ml.ReducirEnOrden` (`internal/ml/logistica.go:196`) acumula los parciales de izquierda a
derecha: `((p₀ + p₁) + p₂) + … + p_{B−1}`. La suma de `float64` **no es asociativa**, así que
cualquier otra agrupación da otros bits. En particular, si cada nodo sumara primero sus bloques y
el coordinador sumara los totales de los nodos,

```
(p₀ + … + p₆₁) + (p₆₂ + … + p₁₂₃) + (p₁₂₄ + … + p₁₈₆)   ≠   ((p₀ + p₁) + p₂) + … + p₁₈₆
```

y el resultado dependería del número de nodos. `TestShardsReproducenElGradienteGlobal`
(`internal/gold/gold_test.go:58`) ya fija la forma correcta: cada shard aporta **un parcial por
bloque**, el coordinador los coloca por índice global de bloque y reduce todo en orden.

**Propuesta:** el nodo devuelve el `ml.Parcial` de **cada bloque** de su shard, sin reducirlos.
El coordinador los guarda en `parciales[bloque_inicial + k]` y aplica `ReducirEnOrden` y
`Finalizar` sobre los `B` bloques. El orden de llegada de los nodos deja de importar.

Costo: más bytes por época que mandar un solo parcial por nodo. Con el gold actual (d = 31,
un `Parcial` = 32 + 2 = 34 `float64`, unos 62 bloques por shard) son del orden de
62 × 34 ≈ 2 100 números por nodo y época. Es una estimación por cálculo, no una medición; la
medición del tiempo de red frente al de cómputo queda para PC4 (`reports/bench_cluster_*.json`).

## 3. Transporte y framing

- **TCP**, una conexión persistente por nodo, iniciada por el nodo hacia `coordinator:9000`
  (en Compose, el nombre del servicio). El coordinador no necesita conocer las IP de los nodos y
  escalar es `docker compose up --scale node=N`.
- **JSON delimitado por salto de línea** (un mensaje = un objeto JSON + `\n`), con
  `json.Encoder` y `json.Decoder` sobre la conexión (con `bufio`). Se usa `Decoder` y **no**
  `bufio.Scanner`: el `Scanner` corta en 64 KiB por defecto y un mensaje `gradiente` puede
  acercarse a ese tamaño (≈ 2 100 números × ~20 caracteres).
- **Exactitud de los `float64` en JSON:** `encoding/json` escribe cada `float64` con la
  representación más corta que vuelve exactamente al mismo valor (`strconv`, precisión `-1`), así
  que la ida y vuelta conserva los bits. `NaN` e `±Inf` no se pueden serializar: si aparecen, el
  nodo responde `error` (un gradiente no finito es un fallo, no un dato). Lo exige un test en PC4.
- **Sobre de mensaje:** un solo tipo Go con campo `tipo` y campos `omitempty`, para que el
  decodificador no tenga que adivinar.

## 4. Mensajes

| Tipo | Sentido | Campos | Cuándo |
|---|---|---|---|
| `registro` | nodo → coord. | `nodo`, `version` (= 1), `workers` | al conectarse |
| `asignacion` | coord. → nodo | `shard`, `ruta`, `sha256`, `bloque_inicial`, `num_bloques`, `tam_bloque`, `d`, `config` (`lambda`, `peso_pos`, `peso_neg`) | tras el registro, o al reasignar un shard |
| `listo` | nodo → coord. | `shard`, `bloque_inicial`, `num_bloques`, `n`, `d` | cuando cargó y validó el shard |
| `epoca` | coord. → nodo | `epoca`, `shard`, `w` (d + 1 pesos) | al empezar cada época |
| `gradiente` | nodo → coord. | `epoca`, `shard`, `bloque_inicial`, `parciales` (lista de `{grad, perdida, peso}`, uno por bloque) | al terminar su cálculo |
| `error` | ambos | `epoca`, `shard`, `mensaje` | shard inválido, gradiente no finito, versión incompatible |
| `fin` | coord. → nodo | `motivo` (`completo` / `abortado`) | al terminar el entrenamiento |

- `listo` repite lo que el nodo leyó de la cabecera del archivo gold. El coordinador lo compara con
  la asignación: si no coincide, descarta al nodo. Así se detecta un volumen de datos equivocado
  antes de la primera época, no por un gradiente raro después.
- `epoca` y `gradiente` llevan **`epoca` y `shard`**. Con eso, el coordinador descarta respuestas
  tardías de épocas pasadas y duplicados de un shard reasignado (sección 6).

## 5. Una época, paso a paso

```mermaid
sequenceDiagram
  participant C as Coordinador
  participant N0 as Nodo 0 (shard 0)
  participant N1 as Nodo 1 (shard 1)
  participant N2 as Nodo 2 (shard 2)
  C->>N0: epoca{e, shard 0, w}
  C->>N1: epoca{e, shard 1, w}
  C->>N2: epoca{e, shard 2, w}
  par cálculo local con el worker pool
    N0->>C: gradiente{e, shard 0, bloques 0..61}
    N2->>C: gradiente{e, shard 2, bloques 124..186}
    N1->>C: gradiente{e, shard 1, bloques 62..123}
  end
  Note over C: barrera: los 3 shards de la época e<br/>parciales[0..186] → ReducirEnOrden → Finalizar
  C->>C: w ← w − η·∇L ; publica métricas (WebSocket)
```

**Concurrencia en el coordinador** (mismo estilo que ADR-0002):

- Una goroutine **lectora por conexión** decodifica mensajes y los manda a un único canal
  `respuestas`. Nadie más lee de esa conexión.
- La **escritura** a cada nodo la hace solo el bucle de épocas (una goroutine), así que no hay dos
  escritores sobre el mismo `net.Conn`. Toda escritura lleva plazo (`SetWriteDeadline`) y un error
  de escritura cuenta como caída del nodo: **enviar nunca puede bloquear el bucle de épocas**. Spin
  encontró una corrida en que el coordinador se quedaba bloqueado enviando a un nodo caído
  (sección 9).
- El **bucle de épocas** difunde `w`, recibe de `respuestas` con `select` y un temporizador de
  época, y guarda cada `gradiente` en el slot de su shard. Cuando están todos, reduce en orden.
  El estado de la época (slots, qué shard llegó) es local a esa goroutine: no hay mutex.
- La API HTTP solo lee el estado publicado (época, pérdida, pesos) a través de un canal o de
  una copia protegida por `sync.RWMutex`; nunca toca los slots.

**En el nodo:** al recibir `asignacion` carga el shard con `gold.Leer`, crea un `ml.Pool`
(ADR-0002) y responde `listo`. Por cada `epoca` calcula los parciales por bloque con el pool y
responde `gradiente`. Hay que exponer en `internal/ml` una función que devuelva los parciales sin
reducirlos; hoy `Pool.Gradiente` los reduce y finaliza.

## 6. Fallos: qué pasa si un nodo cae o se demora

| Situación | Cómo se detecta | Qué hace el coordinador |
|---|---|---|
| Nodo no se registra a tiempo | plazo de registro (`-timeout-registro`) sin los N shards cubiertos | aborta con error claro (no entrena con datos incompletos) |
| Nodo cae a mitad de época (EOF, reset) | la goroutine lectora recibe error de lectura | marca al nodo como descartado y **reasigna su shard** a un nodo sano: `asignacion` + `epoca` de la misma época |
| Nodo vivo pero colgado o lento | vence el temporizador de época (`-timeout-epoca`) | igual que si hubiera caído; si después llega su respuesta, se descarta por `epoca` / `shard` ya cubierto |
| Respuesta de una época anterior | `epoca` ≠ época actual | se descarta |
| Dos respuestas del mismo shard (por reasignación) | el slot del shard ya está lleno | se descarta la segunda: los bits son idénticos porque el cálculo es determinista |
| `error` de un nodo | mensaje explícito | se trata como caída de ese nodo |
| No quedan nodos sanos | todos descartados | `fin{abortado}` y error en la API; nunca espera para siempre |

**Reasignar en dos pasadas.** Cuando vence el plazo, el coordinador primero descarta a **todos**
los dueños de shards que no llegaron y recién después reasigna esos shards a nodos sanos. En una
sola pasada, un shard podía reasignarse a un nodo caído que todavía no se había descartado; como
después ese dueño figuraba como descartado, el shard no se volvía a reasignar y el coordinador
quedaba en un ciclo de timeouts sin avanzar. El error apareció al escribir el modelo Promela y Spin
lo confirma con un contraejemplo (sección 9, mutante 2).

**Por qué reasignar no rompe el determinismo:** los shards son archivos deterministas en un
volumen compartido (de solo lectura), y la reducción depende solo del índice global de bloque, no
de qué nodo calculó cada bloque. Si el nodo 2 cae y el nodo 0 calcula también el shard 2, el
arreglo `parciales[0..186]` es el mismo y el gradiente sale con los mismos bits.

**Alternativas para fallos, descartadas:**

- *Seguir sin el shard caído* (gradiente con menos datos): rompe la equivalencia con el
  secuencial y cambia el modelo según qué nodo cayó.
- *Esperar indefinidamente a que el nodo vuelva:* posible deadlock si no vuelve nunca.
- *Replicar cada shard en dos nodos siempre:* duplica el cómputo para cubrir un caso raro; la
  reasignación bajo demanda cuesta lo mismo solo cuando hace falta.

## 7. Alternativas de topología (descartadas)

| Alternativa | Por qué no |
|---|---|
| Parameter server **asíncrono** (cada nodo actualiza cuando termina) | Gradientes calculados con pesos viejos y orden de actualización dependiente de la red: no reproducible ni equivalente al secuencial |
| **Ring all-reduce** entre nodos | Los nodos necesitan conocerse entre sí y el orden de suma depende del anillo; con 3 a 8 nodos y unos KB por época la red no es el cuello de botella que justificaría la complejidad |
| Que cada nodo **reduzca localmente** y mande un solo parcial | Menos tráfico, pero rompe la equivalencia bit a bit (sección 2) |
| **gRPC** o `net/rpc` + `gob` | gRPC no es stdlib (§3.2). `gob` sí lo es, pero JSON se puede leer en un `tcpdump` y en los logs, lo que ayuda a depurar y a mostrar el protocolo en la sustentación |

## 8. Verificación prevista (PC4)

1. **Equivalencia:** test que levanta el coordinador y 1, 2, 3 y 5 nodos en el mismo proceso
   (conexiones reales por `127.0.0.1`) y compara los pesos tras varias épocas con
   `ml.Entrenar(ml.NuevoSecuencial(...))`: idénticos bit a bit. Con `-race`.
2. **Ida y vuelta JSON:** test de que `Parcial` → JSON → `Parcial` conserva los bits, incluidos
   subnormales y `-0`, y de que `NaN` produce `error`.
3. **Fallos:** tests con un nodo falso que cierra la conexión, que no responde o que responde
   dos veces o con una época vieja; en todos, el resultado final es igual al secuencial o el
   entrenamiento aborta con error, nunca se cuelga.
4. **Modelo formal (plus):** `spin/barrera.pml` modela la barrera con caída y sospecha de nodos;
   Spin verifica ausencia de deadlock y que toda corrida termina (sección 9).

## 9. Modelo Promela

`spin/barrera.pml` modela un coordinador y N = 3 nodos durante 3 épocas, con canales con búfer,
hasta una caída de nodo y una sospecha falsa (timeout de un nodo que estaba vivo). Propiedades:

- **Ausencia de deadlock:** ningún estado final inválido (los nodos esperan en una etiqueta `end`).
- **Barrera:** `assert` de que una época avanza solo con los N shards recibidos.
- **Terminación:** `ltl termina { <> terminado }`.
- **Sin pérdida de shards:** si el entrenamiento no se abortó, se completaron todas las épocas.

**Resultado** (Spin 6.5.2, gcc 14.2, Debian 13; comandos en `spin/README.md`):

| Verificación | Estados | Errores |
|---|---:|---:|
| Seguridad: deadlocks y `assert` (`-DNOCLAIM`) | 87 711 almacenados | 0 |
| LTL `termina` (`-a`) | 171 332 visitados | 0 |
| Mutante 1: canal de nodo con capacidad N (envío que bloquea) | — | 1 (ciclo de aceptación) |
| Mutante 2: `reasignar` en una sola pasada | — | 1 (ciclo de aceptación) |

Los mutantes muestran que el modelo sí distingue el diseño correcto de los dos errores encontrados:
el envío que bloquea al coordinador (sección 5) y la reasignación en una pasada (sección 6).

**Límites del modelo:** N, épocas y fallos son pequeños (3, 3, 1 + 1) para que la exploración sea
exhaustiva; no modela el contenido numérico (eso lo cubre el test de equivalencia) ni la
reconexión de un nodo descartado.

## 10. Preguntas para el juez

1. ¿Es aceptable el costo de mandar un parcial por bloque frente a la alternativa de redefinir el
   "secuencial de referencia" como una suma jerárquica por shard?
2. ¿JSON o `gob`? La propuesta elige JSON por legibilidad; `gob` sería más compacto.
3. ¿Reasignar en la misma época o reiniciar la época completa? Reasignar solo el shard faltante es
   más barato; reiniciar es más simple de implementar.
