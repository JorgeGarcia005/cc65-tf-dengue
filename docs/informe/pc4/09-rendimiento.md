# 9. Rendimiento del clúster

## 9.1 Protocolo de medición

`go run ./cmd/benchmark -cluster -gold data/gold/train.bin` levanta en el mismo proceso un coordinador y
N nodos que hablan por TCP con el protocolo real, para N = 1 a 5. Cada nodo corre con **un worker**, como
una máquina de un núcleo; la red es la de *loopback*, así que el costo de red medido es el mínimo posible.
Por cada N se parte el gold en N shards, se descarta un entrenamiento de calentamiento (carga los shards) y
se miden cinco repeticiones. **Cada repetición se compara bit a bit con el entrenamiento secuencial antes
de registrar su tiempo**: si difiere, el benchmark falla. El tiempo por época se resume con la media
recortada y su intervalo de confianza del 95 % por *bootstrap*, y se separa en:

- **cómputo del nodo más lento**: el que dicta cuándo se cumple la barrera (si un nodo tiene varios
  shards, sus tiempos se suman, porque los calcula uno tras otro);
- **reducción** en el coordinador (`ReducirEnOrden` y `Finalizar`), que dura decenas de microsegundos;
- **red y espera**: el resto de la época, que incluye enviar los pesos, la serialización JSON, la
  transmisión y la espera en la barrera;
- **JSON del shard más pesado**, medido aparte, que es la parte de la serialización que cae en el camino
  crítico.

La fracción serial se estima con la métrica de Karp y Flatt (1990), donde S es el *speedup* con p nodos:

$$e = \frac{1/S - 1/p}{1 - 1/p}$$

Si e crece con p, el límite es la comunicación y no el cómputo.

## 9.2 Resultados

<!-- generado:cluster -->
| Nodos | ms por época (IC 95 %) | Speedup | Eficiencia | Karp-Flatt | Cómputo del nodo más lento (ms) | Red y espera (ms) | JSON del shard más pesado (ms) | KB por época |
|---:|---|---:|---:|---:|---:|---:|---:|---:|
| 1 | 92,00 (91,32–93,42) | 1,00 | 1,00 | — | 89,60 | 2,62 | 2,340 | 112,5 |
| 2 | 49,64 (49,23–51,13) | 1,85 | 0,93 | 0,079 | 48,49 | 1,45 | 1,200 | 113,3 |
| 3 | 35,38 (34,72–35,86) | 2,60 | 0,87 | 0,077 | 34,30 | 1,00 | 0,753 | 114,1 |
| 4 | 27,58 (27,53–27,63) | 3,34 | 0,83 | 0,066 | 26,69 | 0,89 | 0,572 | 114,8 |
| 5 | 22,48 (22,35–23,97) | 4,09 | 0,82 | 0,055 | 22,11 | 0,78 | 0,473 | 115,6 |

Fuente: `reports/bench_cluster_jorge-pc_gold.json` (AMD Ryzen 5 3600 6-Core Processor, 12 hilos, Windows, go1.27.3; 5 repeticiones de 30 épocas por número de nodos, 1 worker por nodo; cada repetición verificada bit a bit contra el secuencial).
<!-- /generado -->

![Speedup del entrenamiento distribuido por número de nodos](fig/speedup_cluster.svg)

## 9.3 Análisis

- **La época está dominada por el cómputo.** En todas las configuraciones, el cómputo del nodo más lento
  ocupa casi toda la época; la red y la espera son unos pocos milisegundos o menos. Por eso el *speedup*
  crece casi en línea con los nodos y la eficiencia se mantiene alta.
- **El tráfico no crece con los nodos.** Cada época viaja un parcial por bloque, y el número de bloques es
  fijo (depende de las filas, no de los nodos); con más nodos solo se agrega el envío de los pesos, que
  pesan poco. Por eso los bytes por época casi no cambian.
- **La fracción serial es baja y no crece.** Karp-Flatt se mantiene en unas centésimas: lo que no se
  reparte (difundir los pesos, la barrera y la reducción) es pequeño frente al cómputo. La eficiencia baja
  levemente porque ese costo fijo pesa más cuando la época se acorta.
- **El límite, en una sola máquina.** Los nodos comparten la CPU del equipo, así que a partir de tantos
  nodos como núcleos físicos el *speedup* deja de crecer; en un clúster real cada nodo tendría su propia
  máquina y la red agregaría latencia, pero el tráfico por época seguiría siendo del mismo orden.
