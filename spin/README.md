# spin/

Modelo Promela del protocolo coordinador–nodos (issue #9). Diseño y resultados en
[`docs/decisiones/propuestas/protocolo-tcp.md`](../docs/decisiones/propuestas/protocolo-tcp.md) §9.

| Archivo | Qué modela |
|---|---|
| `barrera.pml` | Barrera por época del parameter server síncrono con N = 3 nodos, caída de un nodo y timeout falso, reasignación de shards y aborto |

## Verificar

En Linux (Debian: `sudo apt-get install -y spin gcc`):

```bash
cd spin
spin -a barrera.pml
gcc -O2 -DNOCLAIM -o pan-seg pan.c && ./pan-seg   # deadlocks y asserts → errors: 0
gcc -O2 -o pan pan.c && ./pan -a -N termina       # LTL <> terminado → errors: 0
```

Con el LTL incluido, `pan` desactiva la búsqueda de estados finales inválidos; por eso la
seguridad se verifica aparte, compilando con `-DNOCLAIM`.

Si hay un contraejemplo, `spin -t -p barrera.pml` lo reproduce paso a paso.
Los archivos generados (`pan*`, `*.trail`, `_spin_nvr.tmp`) no se versionan.
