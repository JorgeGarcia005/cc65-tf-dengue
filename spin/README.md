# spin/

Modelo Promela del protocolo coordinador–nodos, al día con el
[ADR-0005](../docs/decisiones/ADR-0005-protocolo-tcp.md) (issue #31; la primera versión es del
issue #9 y su diseño está en
[`docs/decisiones/propuestas/protocolo-tcp.md`](../docs/decisiones/propuestas/protocolo-tcp.md) §9).

| Archivo | Qué hace |
|---|---|
| `barrera.pml` | Barrera por época del parameter server síncrono (N = 3 nodos y 2 épocas por defecto; `-DN=4` para el balanceo): envío no bloqueante (descarta al nodo si `full(aNodo[j])`, canal de capacidad N), elección del nodo sano de menor carga (empate → menor índice), caída por EOF separada del cuelgue que solo detecta el plazo, plazo por shard, una sospecha falsa sobre un nodo vivo, descarte en dos pasadas (`cubrir`), la época nunca se envía a un nodo descartado (como `coordinador.go`) y aborto si no quedan nodos sanos |
| `verificar.sh` | Corre el modelo y los cuatro mutantes, falla si algo no es lo esperado o si una búsqueda queda incompleta, y escribe `reports/spin.json` |

## Propiedades

- **Seguridad** (`pan` con `-DNOCLAIM`): sin deadlocks ni estados finales inválidos; la barrera
  avanza solo con los N shards; si no se aborta se completan todas las épocas; nunca se envía una
  época a un nodo descartado; entre los nodos sanos la carga difiere a lo sumo en 1; y nunca se
  descarta a un nodo vivo salvo por una sospecha (plazo vencido) o por un canal lleno (variables
  fantasma `vivo[]`, `sospechado[]` y `saturado[]`), así que solo se aborta si todos los nodos
  cayeron, fueron sospechados o se saturaron.
- **Vivacidad** (LTL `termina`, `<> terminado`): toda corrida termina.

Con el LTL incluido, `pan` desactiva la búsqueda de estados finales inválidos; por eso la
seguridad se verifica aparte, compilando con `-DNOCLAIM`. Ninguna corrida usa `-A`: todas las
aserciones están activas.

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

Spin 6.5.2, gcc 13.3, Ubuntu 24.04 (`bash verificar.sh`, 1 min 57 s; también corre en el CI, job
«Spin»). Las cifras quedan en `reports/spin.json`:

| Corrida | Chequeo | Estados | Errores | Esperado |
|---|---|---:|---:|---|
| Modelo N = 3 (canal de capacidad N) | seguridad | 17 949 492 | 0 | 0 |
| Modelo N = 3 (canal de capacidad N) | termina | 17 814 453 | 0 | 0 |
| Modelo N = 3, canal de capacidad 1 | seguridad | 26 025 432 | 0 | 0 |
| Modelo N = 3, canal de capacidad 1 | termina | 25 707 167 | 0 | 0 |
| Modelo N = 4, 1 época, sin sospechas | seguridad | 4 923 889 | 0 | 0 |
| Modelo N = 4, 1 época, sin sospechas | termina | 4 920 817 | 0 | 0 |
| Mutante `MUT_BLOQUEANTE`: envío bloqueante (capacidad 1) | seguridad | 239 247 | 1 (estado final inválido) | > 0 |
| Mutante `MUT_UNA_PASADA`: al descartar se reasigna solo el shard que venció | seguridad | 36 613 | 1 (estado final inválido) | > 0 |
| Mutante `MUT_PLAZO_GLOBAL`: al vencer un plazo se descarta al dueño de todo shard pendiente | seguridad | 70 | 1 (descarta a un nodo vivo) | > 0 |
| Mutante `MUT_PRIMER_SANO`: el shard va al primer nodo sano (N = 4) | seguridad | 5 269 | 1 (rompe el balanceo) | > 0 |

Cómo leerlo:

- **Envío no bloqueante.** Con capacidad N, a un nodo le quedan a lo sumo N mensajes pendientes y
  la rama `full(aNodo[j])` no se alcanza. Por eso hay una corrida con capacidad 1, que representa un
  búfer TCP lleno: ahí la rama sí se recorre (con todas las aserciones activas) y el modelo termina
  sin deadlocks, mientras que el mutante con envío bloqueante deja al coordinador trabado. Un nodo
  vivo con el canal lleno se descarta: es el costo que acepta el ADR-0005, porque una escritura que
  vence su plazo cuenta como caída (`saturado[]`).
- **Dos pasadas.** Si al descartar a un nodo solo se reasigna el shard que venció, sus otros shards
  pendientes quedan con un dueño ya descartado y nadie los vuelve a asignar.
- **Plazo por shard.** El contraejemplo mínimo que encuentra Spin (70 estados, sin caídas): tras
  difundir la época 0, una sospecha falsa vence el plazo del shard 0; con un plazo global se descarta
  también al dueño del shard 1, que está vivo y no fue sospechado. En el sistema real el mismo
  defecto aparece en la caída escalonada (ADR-0005, test 3: un nodo vivo al que se le acaba de
  reasignar un shard), que prueba `TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano`.
- **Menor carga.** Con N = 3 la menor carga solo decide por desempate: hacen falta al menos 4
  shards para que difiera de «primer nodo sano». La corrida con N = 4 sí lo distingue: elegir el
  primer nodo sano rompe el balanceo (la carga de dos nodos sanos difiere en más de 1). En Go lo prueba
  además `TestBalanceoTrasDobleCaidaReparteDosYDos`.

Si hay un contraejemplo, `spin -t -p barrera.pml` (con los mismos `-D`) lo reproduce paso a paso.
Los archivos generados (`pan*`, `*.trail`, `_spin_nvr.tmp`) no se versionan.

Nota de modelado: en Spin, una etiqueta pegada a un `for` hace que un `goto` salte la
inicialización de la variable del bucle; por eso las etiquetas del coordinador van sobre una
sentencia propia (`skip;` o una asignación).
