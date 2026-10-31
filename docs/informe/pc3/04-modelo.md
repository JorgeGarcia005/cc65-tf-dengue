<!-- Generado por cmd/informe desde reports/split.json, reports/entrenamiento.json y reports/bench_gradiente_*.json. No editar a mano. -->

# 4. Modelo de ML distribuido y paralelización del cálculo (Parte B)

## 4.1 Datos de entrenamiento

- Se usan los casos desde **2011** (la categoría *con signos de alarma* se aplica de forma
  sistemática desde ese año, ver §2.3). Quedan fuera 109 144 de las
  1 065 260 filas del silver.
- **Split determinista** 20,00 % prueba: cada fila va a prueba según un hash (splitmix64)
  de su posición en el silver, así que el split no depende del número de workers ni de nodos.

| Conjunto | Filas | Positivos | % positivos |
|---|---:|---:|---:|
| Entrenamiento | 765 297 | 96 012 | 12,55 |
| Prueba | 190 819 | 24 002 | 12,58 |

**31 variables**, todas conocidas al notificar el caso: edad estandarizada y etapa de vida MINSA
(5), sexo, estacionalidad de la semana epidemiológica (seno y coseno), tendencia anual y departamento
(one-hot; los de menos de 100 casos en entrenamiento se agrupan en OTROS). Se excluye `tipo_dx`
(confirmado/probable) porque los casos graves se confirman en laboratorio con más frecuencia: sería
una fuga de información del objetivo (ADR-0004). La media y desviación de edad y año se calculan
**solo con el entrenamiento**.

## 4.2 Modelo: regresión logística con pesos por clase

$$L(w) = \frac{1}{S}\sum_i s_i\,\big[\log(1+e^{z_i}) - y_i z_i\big] + \frac{\lambda}{2}\lVert w\rVert^2,\qquad z_i = x_i^\top w + b$$

$$\nabla_w L = \frac{1}{S}\sum_i s_i\,(\sigma(z_i) - y_i)\,x_i + \lambda w,\qquad S = \sum_i s_i$$

Los pesos por clase ($s_i$ = 3,985 para positivos y 0,572 para negativos)
igualan el aporte de ambas clases ante el desbalance. λ = 0.0001.

**Por qué se puede distribuir:** el gradiente es una **suma sobre filas**. Si las filas se parten en
bloques $B_1,\dots,B_K$, cada bloque aporta su suma parcial $g_k$ y el gradiente total es
$\frac{1}{S}\sum_k g_k + \lambda w$. Un worker, un hilo o un nodo puede calcular cualquier $g_k$;
solo hace falta **sumar los parciales siempre en el mismo orden de bloque** para que el resultado sea
idéntico bit a bit (la suma en punto flotante no es asociativa).

## 4.3 Paralelización en Go dentro de un nodo

| Patrón | Dónde | Para qué |
|---|---|---|
| *Worker pool* persistente | `ml.Pool` | Los workers viven todo el entrenamiento; no se crean goroutines por época |
| *Fan-out* por canal | `tareas chan int` | Reparte índices de bloque; cada bloque lo procesa un solo worker |
| Acumuladores privados | búfer local + slot `parciales[k]` | Nadie escribe donde otro escribe: sin mutex y sin *false sharing* |
| Barrera | `sync.WaitGroup` | Espera los 187 bloques de la época antes de reducir |
| *Fan-in* ordenado | `ml.ReducirEnOrden` | Un solo goroutine suma en orden de bloque → determinismo |

El gradiente del pool es **idéntico bit a bit** al secuencial
sobre los datos reales (verificado por `cmd/entrenar`); `TestEquivalencia` lo exige con 1 a 64 workers.

## 4.4 Distribución en el clúster (diseño para PC4)

El entrenamiento de 765 297 filas se reparte en **3 shards** de bloques
contiguos (4096 filas por bloque, 187 bloques):

| Shard | Bloque inicial | Bloques | Filas |
|---:|---:|---:|---:|
| 0 | 0 | 62 | 253 952 |
| 1 | 62 | 62 | 253 952 |
| 2 | 124 | 63 | 257 393 |

En cada época el coordinador (parameter server) envía los pesos a los nodos por TCP; cada nodo calcula
con su worker pool los parciales de sus bloques y los devuelve; el coordinador los concatena **en orden
de shard** (que es el orden global de bloque), aplica `Finalizar` y actualiza los pesos. El test
`TestShardsReproducenElGradienteGlobal` comprueba que con 1, 2, 3, 5 u 8 shards el gradiente es
idéntico bit a bit al secuencial. El protocolo TCP se define en #9 (ADR pendiente).

## 4.5 Resultados del entrenamiento

Descenso de gradiente *full-batch*: 600 épocas, tasa 2, 2 workers,
36,92 s (61,5 ms por época). Pérdida: 0,6931 →
0,6701.

| Métrica (prueba, umbral 0.5) | Valor |
|---|---:|
| AUC-ROC | 0,6221 |
| Recall (sensibilidad) | 0,619 |
| Precisión | 0,170 |
| F1 | 0,266 |
| Exactitud | 0,571 |
| VP / FN / FP / VN | 14 859 / 9 143 / 72 713 / 94 104 |

El poder predictivo es **moderado**: sin comorbilidades ni datos clínicos, solo con edad, sexo, lugar y
momento, el modelo ordena el riesgo mejor que el azar pero no lo determina. Es una limitación de los
datos abiertos disponibles, declarada y aceptada (ADR-0003).

**Variables más influyentes** (razón de odds = $e^{w}$; > 1 aumenta el riesgo):

| Variable | Peso | Razón de odds |
|---|---:|---:|
| `depto_ayacucho` | -0,909 | 0,40 |
| `depto_cusco` | -0,763 | 0,47 |
| `depto_madre_de_dios` | +0,716 | 2,05 |
| `depto_loreto` | +0,640 | 1,90 |
| `depto_san_martin` | +0,631 | 1,88 |
| `depto_ucayali` | +0,572 | 1,77 |
| `depto_ica` | -0,385 | 0,68 |
| `depto_lima` | -0,338 | 0,71 |
| `depto_ancash` | -0,310 | 0,73 |
| `depto_la_libertad` | -0,301 | 0,74 |

## 4.6 Primera respuesta a la pregunta de impacto social

Riesgo de dengue con signos de alarma o grave en el conjunto de prueba. El riesgo calibrado corrige el
efecto de los pesos por clase (corrección de prior, King y Zeng, 2001) para que esté en la escala real.

| Departamento | Casos (prueba) | Tasa observada | Riesgo calibrado |
|---|---:|---:|---:|
| madre_de_dios | 5 913 | 19,70 % | 20,65 % |
| loreto | 17 548 | 19,47 % | 19,78 % |
| san_martin | 10 646 | 19,36 % | 19,17 % |
| ucayali | 9 085 | 19,72 % | 19,03 % |
| cajamarca | 5 923 | 15,09 % | 14,28 % |
| junin | 5 158 | 14,25 % | 13,60 % |
| huanuco | 2 148 | 13,59 % | 13,36 % |
| piura | 41 319 | 13,37 % | 13,34 % |

| Etapa de vida | Casos (prueba) | Tasa observada | Riesgo calibrado |
|---|---:|---:|---:|
| nino_0_11 | 32 078 | 14,66 % | 14,79 % |
| adolescente_12_17 | 25 914 | 13,61 % | 13,55 % |
| joven_18_29 | 43 650 | 12,82 % | 12,67 % |
| adulto_mayor_60_mas | 17 313 | 12,66 % | 12,52 % |
| adulto_30_59 | 71 864 | 11,11 % | 11,08 % |

## 4.7 Speedup del cálculo del gradiente

**Intel(R) Xeon(R) Processor @ 2.10GHz** · 2 CPU lógicas (`bench_gradiente_ci-2vcpu.json`; sintetico(filas=1000000, features=40, semilla=2026))

| Configuración | Workers | Media recortada (ms) | IC 95 % (ms) | Speedup | Eficiencia | Bit a bit |
|---|---:|---:|---|---:|---:|:---:|
| secuencial | 1 | 127,2 | [121,0, 136,3] | 1,00× | 100,00 % | ✔ |
| pool-01 | 1 | 127,0 | [121,5, 134,5] | 1,00× | 100,17 % | ✔ |
| pool-02 | 2 | 87,5 | [81,8, 94,2] | 1,45× | 72,68 % | ✔ |
| pool-04 | 4 | 77,4 | [74,1, 80,8] | 1,64× | 41,10 % | ✔ |

Punto de equilibrio con 2 workers: la concurrencia conviene desde **10 000 filas**.

| Filas | Bloques | Secuencial (ms) | Concurrente (ms) | Speedup |
|---:|---:|---:|---:|---:|
| 1 000 | 1 | 0,1 | 0,1 | 0,94× |
| 4 096 | 1 | 0,7 | 0,7 | 0,97× |
| 10 000 | 3 | 1,2 | 0,9 | 1,33× |
| 30 000 | 8 | 3,7 | 2,7 | 1,37× |
| 100 000 | 25 | 11,2 | 7,6 | 1,46× |
| 300 000 | 74 | 34,2 | 23,9 | 1,43× |
| 1 000 000 | 245 | 121,3 | 96,8 | 1,25× |

