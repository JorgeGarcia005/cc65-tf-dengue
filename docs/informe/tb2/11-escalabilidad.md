# 11. Rendimiento y escalabilidad del clúster

La sección 8.5 midió cuánto se acelera el cálculo del gradiente con más goroutines dentro de un nodo. Esta sección mide cuánto se acelera el entrenamiento al **repartirlo entre nodos** que se comunican por TCP, y responde a la observación 2 de la PC4 («faltan más pruebas que sustenten los valores de las tablas»): cada cifra sale de `reports/bench_cluster_deez_gold.json`, se muestra con su intervalo de confianza y la dispersión de sus repeticiones (11.2), se descompone en cómputo, red y espera (11.3) y se acompaña del código que la produce (11.5).

## 11.1 Protocolo de medición

`go run ./cmd/benchmark -cluster -gold data/gold/train.bin` levanta, en un mismo proceso, un coordinador y N nodos que se comunican por TCP con el protocolo real de la sección 9, para N = 1 a <!--v:(index .Cluster.Mediciones .Ultimo).Nodos-->5<!--/v-->. Las condiciones son:

- **Datos:** el conjunto de entrenamiento real (<!--v:miles .S.M.S.Train.Filas-->765 297<!--/v--> filas), partido en N *shards* con la misma función que usa `cmd/gold` (`gold.Particionar`).
- **Un worker por nodo**, para que cada nodo se comporte como una máquina de un núcleo y el *speedup* refleje solo la distribución, no la concurrencia interna de la sección 8.
- **Red de *loopback*** (127.0.0.1): el costo de red medido es el mínimo posible. En un clúster real se sumaría la latencia de la red física (sección 11.4).
- **Calentamiento:** un entrenamiento descartado por cada N, que carga los *shards* en los nodos.
- **Repeticiones:** <!--v:.Cluster.Protocolo.Reps-->5<!--/v--> entrenamientos de <!--v:.Cluster.Protocolo.Epocas-->30<!--/v--> épocas por cada N. El tiempo por época de cada repetición es su duración total dividida entre sus épocas.
- **Verificación:** antes de registrar cada repetición, sus pesos y su historia de pérdida se comparan bit a bit con los del entrenamiento secuencial; si difieren, el benchmark se detiene con error.
- **Estimador:** media recortada al 20 % e intervalo de confianza del 95 % por *bootstrap*, que no suponen una distribución normal de los tiempos.
- **Máquina:** <!--v:.Cluster.Maquina.CPU-->AMD Ryzen 7 5800X 8-Core Processor<!--/v-->, <!--v:.Cluster.Maquina.NumCPU-->16<!--/v--> hilos, <!--v:.Cluster.Maquina.Go-->go1.26.10<!--/v-->, sistema <!--v:.Cluster.Maquina.SO-->windows<!--/v-->: el mismo procesador de los benchmarks de las secciones 7 y 8, para no mezclar procesadores.

El tiempo de cada época se separa en cuatro partes, que el coordinador mide con `(*Coordinador).Tiempos`:

- **Cómputo del nodo más lento:** el que decide cuándo se cumple la barrera. Si un nodo tiene varios *shards*, sus tiempos se suman, porque los calcula uno tras otro.
- **Reducción:** `ReducirEnOrden` y `Finalizar` en el coordinador.
- **Red y espera:** el resto de la época: envío de los pesos, serialización JSON, transmisión y espera en la barrera.
- **Bytes por época:** lo enviado y recibido por todas las conexiones, dividido entre las épocas.

La fracción serial se estima con la métrica de Karp y Flatt (1990), donde *S* es el *speedup* con *p* nodos:

$$e = \frac{1/S - 1/p}{1 - 1/p}$$

Si *e* se mantiene constante al aumentar *p*, lo que limita el *speedup* es una parte fija del trabajo que no se reparte (ley de Amdahl); si *e* crece con *p*, el límite es un costo que aumenta con los nodos, como la comunicación.

## 11.2 Resultados de 1 a 5 nodos

Con <!--v:(index .Cluster.Mediciones .Ultimo).Nodos-->5<!--/v--> nodos el entrenamiento es <!--v:dos (index .Cluster.Mediciones .Ultimo).Speedup-->4,80<!--/v--> veces más rápido que con uno (@tab:cluster y @fig:speedup-cluster), y todas las repeticiones produjeron el mismo modelo que el entrenamiento secuencial.

<!-- generado:plantilla:cluster -->
Tabla: Tiempo por época del entrenamiento distribuido según el número de nodos {#tab:cluster}

| Nodos | ms por época | IC 95 % (ms) | Speedup | Eficiencia | Karp-Flatt | Cómputo del nodo más lento (ms) | Red y espera (ms) | Red y espera (%) | KB por época | Igual bit a bit |
|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|:---:|
| 1 | 72,35 | [70,91; 76,89] | 1,00 | 1,00 | — | 70,85 | 2,36 | 3,22 | 112,5 | sí |
| 2 | 40,79 | [40,57; 41,04] | 1,77 | 0,89 | 0,128 | 39,50 | 1,29 | 3,15 | 113,3 | sí |
| 3 | 25,67 | [23,94; 28,30] | 2,82 | 0,94 | 0,032 | 24,97 | 0,90 | 3,47 | 114,1 | sí |
| 4 | 18,49 | [18,22; 18,79] | 3,91 | 0,98 | 0,007 | 17,84 | 0,64 | 3,46 | 114,8 | sí |
| 5 | 15,06 | [14,89; 15,70] | 4,80 | 0,96 | 0,010 | 14,69 | 0,51 | 3,37 | 115,6 | sí |

Nota: AMD Ryzen 7 5800X 8-Core Processor, 16 hilos, go1.26.10; 5 repeticiones de 30 épocas por número de nodos, 1 worker por nodo, nodos y coordinador en la misma máquina comunicados por TCP. Media recortada al 20 % e IC 95 % por *bootstrap*. Fuente: `reports/bench_cluster_deez_gold.json`, generado por `cmd/benchmark -cluster`.
<!-- /generado -->

![Speedup del entrenamiento distribuido según el número de nodos](fig/speedup_cluster.svg){#fig:speedup-cluster}

Nota: La línea punteada es el speedup ideal (igual al número de nodos). Fuente: `reports/bench_cluster_deez_gold.json`.

Para que se pueda juzgar la estabilidad de cada media, la @tab:cluster-muestras muestra la dispersión de las repeticiones: la diferencia entre la repetición más rápida y la más lenta es pequeña frente a la diferencia entre configuraciones, así que el orden de los resultados no depende de una repetición afortunada.

<!-- generado:plantilla:cluster-muestras -->
Tabla: Dispersión de las repeticiones de cada medición del clúster {#tab:cluster-muestras}

| Nodos | Repeticiones | Mínimo (ms) | Mediana (ms) | Máximo (ms) | Desviación (ms) | Media recortada (ms) |
|---:|---:|---:|---:|---:|---:|---:|
| 1 | 5 | 70,62 | 71,89 | 78,49 | 3,14 | 72,35 |
| 2 | 5 | 40,51 | 40,78 | 41,05 | 0,24 | 40,79 |
| 3 | 5 | 23,93 | 25,02 | 28,43 | 2,20 | 25,67 |
| 4 | 5 | 18,20 | 18,53 | 18,86 | 0,27 | 18,49 |
| 5 | 5 | 14,87 | 15,12 | 15,98 | 0,45 | 15,06 |

Nota: Cada repetición es un entrenamiento de 30 épocas cuyos pesos se comparan bit a bit con los del entrenamiento secuencial antes de registrarse. Fuente: `reports/bench_cluster_deez_gold.json` (`seg_por_epoca` de cada medición).
<!-- /generado -->

## 11.3 Speedup, eficiencia y desglose del tiempo

**La época está dominada por el cómputo.** En todas las configuraciones, el cómputo del nodo más lento ocupa casi toda la época, y la red y la espera son una fracción pequeña y estable (columna «Red y espera (%)» de la @tab:cluster). Por eso el *speedup* crece casi en línea con los nodos y la eficiencia se mantiene alta.

**El tráfico casi no crece con los nodos.** En cada época viaja un parcial por bloque y el número de bloques es fijo: depende de las filas, no de los nodos. Agregar un nodo solo suma un envío más de los pesos, que es pequeño. Por eso los kilobytes por época casi no cambian entre 1 y <!--v:(index .Cluster.Mediciones .Ultimo).Nodos-->5<!--/v--> nodos.

**La fracción serial es baja y no crece.** La métrica de Karp-Flatt baja de <!--v:tres (derefF (index .Cluster.Mediciones 1).KarpFlatt)-->0,128<!--/v--> con <!--v:(index .Cluster.Mediciones 1).Nodos-->2<!--/v--> nodos a <!--v:tres (derefF (index .Cluster.Mediciones .Ultimo).KarpFlatt)-->0,010<!--/v--> con <!--v:(index .Cluster.Mediciones .Ultimo).Nodos-->5<!--/v-->: lo que no se reparte (enviar los pesos, la barrera y la reducción) es pequeño frente al cómputo y no aumenta con los nodos, así que el protocolo no es el cuello de botella. El valor más alto con dos nodos no se repite con más nodos, así que no indica un costo que crezca con ellos; con tan pocos nodos el denominador de la fórmula es pequeño y una variación pequeña del *speedup* mueve mucho la estimación. Por la ley de Amdahl (Amdahl, 1967), si esa fracción se mantuviera, el *speedup* tendría un techo teórico de 1/*e* ≈ <!--v:uno (inv (derefF (index .Cluster.Mediciones .Ultimo).KarpFlatt))-->98,0<!--/v-->; en la práctica, el límite llega antes por el hardware, como se explica a continuación.

**Eficiencia cercana a 1 y el efecto de la caché.** Con algunos números de nodos la eficiencia llega casi al ideal. Una explicación probable, que este benchmark no aísla, es la caché: al repartir el conjunto en más *shards*, cada nodo trabaja sobre menos datos, que caben mejor en la caché del núcleo que lo ejecuta, y esa ganancia compensa el costo de la comunicación.

**El límite en una sola máquina.** Los nodos del benchmark comparten la CPU del mismo equipo. Con más nodos que núcleos físicos, dos nodos compiten por el mismo núcleo y el *speedup* deja de crecer. En un clúster real cada nodo tendría su propia máquina: desaparece esa competencia, pero se suma la latencia de la red física, que se analiza en el punto de equilibrio.

## 11.4 Punto de equilibrio del clúster

Distribuir tiene un costo fijo por época: enviar los pesos, serializar y recibir los parciales y esperar en la barrera. Solo conviene si el cómputo que se ahorra al repartir supera ese costo. La @tab:equilibrio-cluster lo estima con los valores medidos: con *p* nodos se ahorra la fracción 1 − 1/*p* del cómputo de un nodo, que es proporcional a las filas, y se paga la red y espera medida con *p* nodos.

<!-- generado:plantilla:equilibrio-cluster -->
Tabla: Estimación del punto de equilibrio del clúster según el número de nodos {#tab:equilibrio-cluster}

| Nodos | Cómputo ahorrado por época (ms) | Red y espera por época (ms) | Conviene distribuir desde (filas) |
|---:|---:|---:|---:|
| 2 | 35,43 | 1,29 | 27 793 |
| 3 | 47,23 | 0,90 | 14 548 |
| 4 | 53,14 | 0,64 | 9 224 |
| 5 | 56,68 | 0,51 | 6 927 |

Nota: Estimación con un modelo lineal a partir de la @tab:cluster: el cómputo de una época con un nodo es proporcional a las filas, repartirlo en *p* nodos ahorra la fracción 1 − 1/*p* de ese cómputo y conviene cuando el ahorro supera la red y espera medida con *p* nodos. La red y espera se midió con el conjunto completo; con menos filas viajan menos bloques, así que el equilibrio real está en las filas indicadas o en menos. Calculado por `cmd/informe` (`EquilibrioCluster`) desde `reports/bench_cluster_deez_gold.json`.
<!-- /generado -->

El conjunto de entrenamiento real está muy por encima de ese punto, por lo que distribuirlo conviene con cualquier número de nodos medido. Dos consecuencias para un despliegue real:

1. **Con red física**, la red y espera crece con la latencia entre máquinas (cada época tiene al menos un viaje de ida y vuelta por nodo) y el punto de equilibrio sube en la misma proporción, porque el modelo es lineal. La regla práctica sale de comparar las dos columnas centrales de la @tab:equilibrio-cluster: distribuir conviene mientras la red y espera de una época sea pequeña frente al cómputo que se ahorra. En una red lenta, en la que ambas se acercaran, habría que reducir la comunicación, por ejemplo con varias épocas por mensaje; eso cambiaría el algoritmo (dejaría de ser descenso por lotes completos) y rompería la equivalencia con el secuencial.
2. **Con más datos**, el cómputo crece con las filas mientras el tráfico crece solo con el número de bloques, mucho más despacio. El clúster se vuelve más conveniente cuanto más grande es el conjunto.

## 11.5 Evidencias de la medición

Las cifras de la @tab:cluster las produce `medirNodos`, que se incluye completo. Se puede ver dónde se descarta el calentamiento, dónde se compara cada repetición bit a bit con el entrenamiento secuencial y cómo se calcula cada parte del desglose:

<!-- generado:fuente:cmd/benchmark/cluster.go#medirNodos -->
```go
// cmd/benchmark/cluster.go
// medirNodos levanta el clúster con n nodos y mide op.reps repeticiones.
func medirNodos(train gold.Archivo, n int, cfg ml.Config, ent ml.Entrenamiento, ref ml.Resultado, op opcionesCluster) (medicionCluster, error) {
	dir, err := os.MkdirTemp("", "cc65-bench-cluster-")
	if err != nil {
		return medicionCluster{}, err
	}
	defer os.RemoveAll(dir)
	partes, err := gold.Particionar(train, n)
	if err != nil {
		return medicionCluster{}, err
	}
	var shards []cluster.Shard
	for s, p := range partes {
		ruta := filepath.Join(dir, fmt.Sprintf("shard_%d.bin", s))
		if err := gold.Escribir(ruta, p); err != nil {
			return medicionCluster{}, err
		}
		desc, _, err := cluster.LeerShard(ruta)
		if err != nil {
			return medicionCluster{}, err
		}
		desc.Ruta = ruta
		shards = append(shards, desc)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return medicionCluster{}, err
	}
	tReg := op.timeoutRegistro
	if tReg <= 0 {
		tReg = 30 * time.Second
	}
	c, err := cluster.Nuevo(ln, cluster.Opciones{
		Shards: shards, D: train.D, Config: cfg,
		TimeoutRegistro: tReg, TimeoutEpoca: 5 * time.Minute, Observador: op.observador,
	})
	if err != nil {
		ln.Close()
		return medicionCluster{}, err
	}
	// Orden de salida (LIFO): Terminar (si se llegó a registrar) → Cerrar,
	// que corta las conexiones → esperar a los nodos. Al revés, un error
	// antes de Terminar dejaba a los nodos bloqueados leyendo (PR-42 H1).
	var wg sync.WaitGroup
	defer wg.Wait()
	defer c.Cerrar()
	conectar := n
	if op.conectar > 0 {
		conectar = min(op.conectar, n)
	}
	for i := range conectar {
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			return medicionCluster{}, err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			(&cluster.Nodo{Nombre: fmt.Sprintf("nodo-%d", i), Workers: op.workersNodo}).Servir(conn)
		}()
	}
	if err := c.EsperarNodos(n); err != nil {
		return medicionCluster{}, err
	}
	defer c.Terminar()

	op.logf("%d nodos: calentamiento (carga de shards) y %d repeticiones de %d épocas", n, op.reps, ent.Epocas)
	if _, err := ml.EntrenarFalible(c, train.D, ent); err != nil {
		return medicionCluster{}, err
	}
	var segEpoca []float64
	var suma desglose
	var bytes float64
	for r := range op.reps {
		antes := c.Tiempos()
		ini := time.Now()
		res, err := ml.EntrenarFalible(c, train.D, ent)
		total := time.Since(ini).Seconds()
		if err != nil {
			return medicionCluster{}, err
		}
		if !mismosBits(ref.W, res.W) || !mismosBits(ref.Historia, res.Historia) {
			return medicionCluster{}, fmt.Errorf("repetición %d: los pesos del clúster difieren de los secuenciales", r)
		}
		d := c.Tiempos()
		e := float64(d.Epocas - antes.Epocas)
		segEpoca = append(segEpoca, total/e)
		suma.Epoca += (d.SegEpoca - antes.SegEpoca) / e
		suma.CalculoNodoMax += (d.SegCalculoMax - antes.SegCalculoMax) / e
		suma.CalculoNodos += (d.SegCalculoSuma - antes.SegCalculoSuma) / e
		suma.Reduccion += (d.SegReduccion - antes.SegReduccion) / e
		bytes += float64(d.BytesEnviados-antes.BytesEnviados+d.BytesRecibidos-antes.BytesRecibidos) / e
	}
	k := float64(op.reps)
	des := desglose{
		Epoca: suma.Epoca / k, CalculoNodoMax: suma.CalculoNodoMax / k,
		CalculoNodos: suma.CalculoNodos / k, Reduccion: suma.Reduccion / k,
	}
	des.RedYEspera = math.Max(0, des.Epoca-des.CalculoNodoMax-des.Reduccion)
	des.FraccionRed = des.RedYEspera / des.Epoca
	des.SerializacionTotal, des.SerializacionShard, err = serializacionPorEpoca(partes, ref.W, cfg)
	if err != nil {
		return medicionCluster{}, err
	}
	return medicionCluster{
		Nodos: n, SegPorEpoca: estadistica.Resumir(segEpoca, op.recorte), Desglose: des,
		BytesPorEpoca: bytes / k, Equivalente: true, Repeticiones: op.reps, EpocasPorRepeticion: ent.Epocas,
	}, nil
}
```
<!-- /generado -->

`TestBenchmarkClusterSeparaTiemposYVerificaPesos` (en `cmd/benchmark`) comprueba en cada Pull Request que el benchmark separa los tiempos correctamente y que falla si los pesos del clúster no son los secuenciales. El protocolo de medición y la máquina quedan registrados en el propio reporte, del que salen las tablas de esta sección:

<!-- generado:archivo:reports/bench_cluster_deez_gold.json -->
```json
{
  "generado": "2026-11-21T02:32:21-05:00",
  "fuente": "gold(data/gold/train.bin, filas=765297, features=31)",
  "maquina": {
    "arquitectura": "amd64",
    "cpu": "AMD Ryzen 7 5800X 8-Core Processor",
    "go": "go1.26.10",
    "gomaxprocs": 16,
    "num_cpu": 16,
    "so": "windows"
  },
  "protocolo": {
    "calentamiento": "1 entrenamiento descartado (carga de shards)",
    "desglose": "medias aritméticas: red y espera = época − cómputo del nodo más lento (sus shards en serie) − reducción; JSON medido aparte (CPU de un hilo, k pasadas agrupadas): total de los shards y shard más pesado (incluido en red y espera)",
    "epocas_por_repeticion": 30,
    "estimador": "media recortada 20 % + IC95 bootstrap (2000)",
    "karp_flatt": "e = (1/S − 1/p) / (1 − 1/p), p = nodos; solo si la base es 1 nodo",
    "lambda": 0.0001,
    "nodos": [
      1,
      2,
      3,
      4,
      5
    ],
    "red": "TCP en 127.0.0.1 (loopback), protocolo JSON por línea real",
    "repeticiones": 5,
    "tasa": 2,
    "verificacion": "cada repetición se compara bit a bit con el secuencial antes de registrarse",
    "workers_por_nodo": 1
  },
  "mediciones": [
    {
      "nodos": 1,
      "seg_por_epoca": {
        "n": 5,
        "media_recortada": 0.0723484388888889,
        "recorte": 0.2,
        "mediana": 0.07189381,
        "desviacion": 0.0031430428580220353,
        "min": 0.07062479666666667,
        "max": 0.07848579,
        "ic95_inf": 0.07090520666666666,
        "ic95_sup": 0.07688568666666666
      },
      "speedup": 1,
      "eficiencia": 1,
      "desglose_por_epoca": {
        "seg_epoca": 0.07322452199999997,
        "seg_calculo_nodo_mas_lento": 0.07085118666666661,
        "seg_calculo_todos_los_nodos": 0.07085118666666661,
        "seg_reduccion": 0.000016659333333333335,
        "seg_red_y_espera": 0.002356676000000032,
        "seg_json_cpu_todos_los_shards": 0.0019608,
        "seg_json_shard_mas_pesado": 0.0019608,
        "fraccion_red_y_espera": 0.03218424560012877
      },
      "bytes_por_epoca": 115224.88,
      "equivalente_bit_a_bit": true,
      "repeticiones": 5,
      "epocas_por_repeticion": 30
    },
    {
      "nodos": 2,
      "seg_por_epoca": {
        "n": 5,
        "media_recortada": 0.040790400000000004,
        "recorte": 0.2,
        "mediana": 0.04077850666666667,
        "desviacion": 0.00023995595402536294,
        "min": 0.04051105,
        "max": 0.04105375,
        "ic95_inf": 0.04056563,
        "ic95_sup": 0.04103575777777778
      },
      "speedup": 1.7736633837591416,
      "eficiencia": 0.8868316918795708,
      "karp_flatt": 0.1276096796682782,
      "desglose_por_epoca": {
        "seg_epoca": 0.04078720000000001,
        "seg_calculo_nodo_mas_lento": 0.039500693333333316,
        "seg_calculo_todos_los_nodos": 0.07835448666666664,
        "seg_reduccion": 0,
        "seg_red_y_espera": 0.0012865066666666938,
        "seg_json_cpu_todos_los_shards": 0.0019765207792207793,
        "seg_json_shard_mas_pesado": 0.001054757142857143,
        "fraccion_red_y_espera": 0.03154192164862245
      },
      "bytes_por_epoca": 116015.23333333332,
      "equivalente_bit_a_bit": true,
      "repeticiones": 5,
      "epocas_por_repeticion": 30
    },
    {
      "nodos": 3,
      "seg_por_epoca": {
        "n": 5,
        "media_recortada": 0.02566983888888889,
        "recorte": 0.2,
        "mediana": 0.025022836666666666,
        "desviacion": 0.0022041211969228112,
        "min": 0.023932223333333332,
        "max": 0.028425013333333336,
        "ic95_inf": 0.02393607222222222,
        "ic95_sup": 0.028297645555555556
      },
      "speedup": 2.8184220088815866,
      "eficiencia": 0.9394740029605289,
      "karp_flatt": 0.03221270458189262,
      "desglose_por_epoca": {
        "seg_epoca": 0.02587335066666664,
        "seg_calculo_nodo_mas_lento": 0.024968833333333326,
        "seg_calculo_todos_los_nodos": 0.07275232666666666,
        "seg_reduccion": 0.000006668666666666666,
        "seg_red_y_espera": 0.0008978486666666462,
        "seg_json_cpu_todos_los_shards": 0.0018767368279569894,
        "seg_json_shard_mas_pesado": 0.0006462354838709677,
        "fraccion_red_y_espera": 0.03470167734492038
      },
      "bytes_por_epoca": 116806.6,
      "equivalente_bit_a_bit": true,
      "repeticiones": 5,
      "epocas_por_repeticion": 30
    },
    {
      "nodos": 4,
      "seg_por_epoca": {
        "n": 5,
        "media_recortada": 0.018486844444444445,
        "recorte": 0.2,
        "mediana": 0.018526166666666666,
        "desviacion": 0.0002736321822812514,
        "min": 0.018196959999999998,
        "max": 0.018862976666666666,
        "ic95_inf": 0.01822328222222222,
        "ic95_sup": 0.018794797777777778
      },
      "speedup": 3.9135093664203255,
      "eficiencia": 0.9783773416050814,
      "karp_flatt": 0.007366843539612071,
      "desglose_por_epoca": {
        "seg_epoca": 0.01849023800000002,
        "seg_calculo_nodo_mas_lento": 0.017843113333333334,
        "seg_calculo_todos_los_nodos": 0.0688407733333333,
        "seg_reduccion": 0.0000067,
        "seg_red_y_espera": 0.0006404246666666861,
        "seg_json_cpu_todos_los_shards": 0.0019282630780072642,
        "seg_json_shard_mas_pesado": 0.0005017069767441861,
        "fraccion_red_y_espera": 0.03463582603245482
      },
      "bytes_por_epoca": 117596.96666666667,
      "equivalente_bit_a_bit": true,
      "repeticiones": 5,
      "epocas_por_repeticion": 30
    },
    {
      "nodos": 5,
      "seg_por_epoca": {
        "n": 5,
        "media_recortada": 0.015060525555555556,
        "recorte": 0.2,
        "mediana": 0.015116816666666666,
        "desviacion": 0.0004472948622093591,
        "min": 0.014872356666666666,
        "max": 0.01598261,
        "ic95_inf": 0.014894413333333334,
        "ic95_sup": 0.015697151111111113
      },
      "speedup": 4.803845564486351,
      "eficiencia": 0.9607691128972702,
      "karp_flatt": 0.010208198456865998,
      "desglose_por_epoca": {
        "seg_epoca": 0.015203838666666664,
        "seg_calculo_nodo_mas_lento": 0.014690846666666677,
        "seg_calculo_todos_los_nodos": 0.06946221999999998,
        "seg_reduccion": 0,
        "seg_red_y_espera": 0.0005129919999999864,
        "seg_json_cpu_todos_los_shards": 0.001897065272934606,
        "seg_json_shard_mas_pesado": 0.00041778510638297876,
        "fraccion_red_y_espera": 0.033740952613808314
      },
      "bytes_por_epoca": 118388.33333333334,
      "equivalente_bit_a_bit": true,
      "repeticiones": 5,
      "epocas_por_repeticion": 30
    }
  ]
}
```
<!-- /generado -->
