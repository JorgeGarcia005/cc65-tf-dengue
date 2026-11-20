# spin/

Modelo Promela del protocolo coordinador–nodos, al día con el
[ADR-0005](../docs/decisiones/ADR-0005-protocolo-tcp.md) (issue #31; la primera versión es del
issue #9 y su diseño está en
[`docs/decisiones/propuestas/protocolo-tcp.md`](../docs/decisiones/propuestas/protocolo-tcp.md) §9).

| Archivo | Qué hace |
|---|---|
| `barrera.pml` | Barrera por época del parameter server síncrono con N = 3 nodos y 2 épocas: envío no bloqueante (descarta al nodo si `full(aNodo[j])`, canal de capacidad N), elección del nodo sano de menor carga (empate → menor índice), caída por EOF separada del cuelgue que solo detecta el plazo, plazo por shard, una sospecha falsa sobre un nodo vivo, descarte en dos pasadas (`cubrir`) y aborto si no quedan nodos sanos |
| `verificar.sh` | Corre el modelo y los tres mutantes y falla si algo no es lo esperado |

## Propiedades

- **Seguridad** (`pan` con `-DNOCLAIM`): sin deadlocks ni estados finales inválidos; la barrera
  avanza solo con los N shards; si no se aborta se completan todas las épocas; y **nunca se
  descarta a un nodo vivo y no sospechado** (variables fantasma `vivo[]` y `sospechado[]`), así que
  solo se aborta si todos los nodos cayeron o fueron sospechados.
- **Vivacidad** (LTL `termina`, `<> terminado`): toda corrida termina.

Con el LTL incluido, `pan` desactiva la búsqueda de estados finales inválidos; por eso la
seguridad se verifica aparte, compilando con `-DNOCLAIM`.

## Verificar

En Linux (Debian/Ubuntu: `sudo apt-get install -y spin gcc`):

```bash
cd spin && bash verificar.sh
```

o a mano:

```bash
cd spin
spin -a barrera.pml
gcc -O2 -DNOCLAIM -o pan-seg pan.c && ./pan-seg   # seguridad → errors: 0
gcc -O2 -o pan pan.c && ./pan -a -N termina       # LTL <> terminado → errors: 0
```

## Resultado

Spin 6.5.2, gcc 13, Ubuntu 24.04 (`bash verificar.sh`, 1 min 27 s):

| Corrida | Chequeo | Estados | Errores | Esperado |
|---|---|---:|---:|---|
| Modelo (canal de capacidad N) | seguridad | 18 450 303 | 0 | 0 |
| Modelo (canal de capacidad N) | termina | 18 306 811 | 0 | 0 |
| Modelo con canal de capacidad 1, sin asserts (`-A`) | seguridad | 26 708 282 | 0 | 0 |
| Modelo con canal de capacidad 1, sin asserts (`-A`) | termina | 26 370 789 | 0 | 0 |
| Mutante `MUT_BLOQUEANTE`: envío bloqueante (capacidad 1) | seguridad | 236 361 | 1 (estado final inválido) | > 0 |
| Mutante `MUT_UNA_PASADA`: al descartar se reasigna solo el shard que venció | seguridad | 35 317 | 1 (estado final inválido) | > 0 |
| Mutante `MUT_PLAZO_GLOBAL`: al vencer un plazo se descarta al dueño de todo shard pendiente | seguridad | 64 | 1 (descarta a un nodo vivo) | > 0 |

Cómo leerlo:

- **Envío no bloqueante.** Con capacidad N el envío bloqueante tampoco se traba en el modelo: a un
  nodo le quedan a lo sumo N mensajes pendientes. Por eso el mutante se prueba con capacidad 1, que
  representa un búfer TCP lleno: el envío bloqueante deja al coordinador trabado y el modelo
  correcto, en cambio, termina sin deadlocks. Con capacidad 1 el modelo correcto sí puede descartar
  a un nodo vivo cuyo canal está lleno (por eso esa corrida va con `-A`): es el costo que el
  ADR-0005 acepta, porque una escritura que vence su plazo cuenta como caída.
- **Dos pasadas.** Si al descartar a un nodo solo se reasigna el shard que venció, sus otros shards
  pendientes quedan con un dueño ya descartado y nadie los vuelve a asignar.
- **Plazo por shard.** Con un plazo global, el vencimiento del shard de un nodo colgado descarta
  también a un nodo vivo al que se le acababa de reasignar un shard (el escenario de la caída
  escalonada del ADR-0005, test 3).

Si hay un contraejemplo, `spin -t -p barrera.pml` (con los mismos `-D`) lo reproduce paso a paso.
Los archivos generados (`pan*`, `*.trail`, `_spin_nvr.tmp`) no se versionan.

Nota de modelado: en Spin, una etiqueta pegada a un `for` hace que un `goto` salte la
inicialización de la variable del bucle; por eso las etiquetas del coordinador van sobre una
sentencia propia (`skip;` o una asignación).
