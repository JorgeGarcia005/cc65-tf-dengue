# 9. Clúster distribuido

La sección 8.2 mostró que el gradiente es una suma de parciales por bloque que cualquier proceso puede calcular. Esta sección describe cómo se reparte ese cálculo entre máquinas: la topología (9.1), el protocolo TCP (9.2), los algoritmos del coordinador y del nodo con su código fuente (9.3), qué pasa cuando un nodo falla (9.4), la evidencia de que el resultado distribuido es idéntico al secuencial (9.5) y las métricas que el sistema registra de cada nodo (9.6). El protocolo se verifica formalmente en la sección 10 y su rendimiento se mide en la sección 11.

## 9.1 Topología en estrella con parameter server síncrono

El clúster tiene un **coordinador**, que hace de *parameter server*: guarda los pesos del modelo, reparte el trabajo y agrega los resultados; y **N nodos de cómputo**, que guardan cada uno un *shard* (un conjunto de bloques contiguos del conjunto de entrenamiento) y calculan sus parciales. La @tab:shards muestra el reparto con tres nodos.

<!-- generado:plantilla:shards -->
Tabla: Shards del conjunto de entrenamiento {#tab:shards}

| Shard | Bloque inicial | Bloques | Filas |
|---:|---:|---:|---:|
| 0 | 0 | 62 | 253 952 |
| 1 | 62 | 62 | 253 952 |
| 2 | 124 | 63 | 257 393 |

Nota: Bloques de 4096 filas (187 en total), contiguos en cada shard. Fuente: `reports/split.json`.
<!-- /generado -->

![Topología del clúster: estrella con el coordinador al centro, en la red interna de Docker](fig/topologia.svg){#fig:topologia}

Nota: Diagrama de despliegue UML 2.5. Elaboración propia a partir de `deploy/docker-compose.yml`.

**Cómo interactúan.** Todos los contenedores comparten la red interna `cluster` y se encuentran por nombre de servicio (@fig:topologia). El coordinador escucha en el puerto TCP 9000, que no se publica fuera de esa red. Cada nodo abre **una sola conexión TCP persistente** con `coordinator:9000` y por ella se registra, recibe su *shard* y los pesos de cada época, y devuelve sus parciales. Los nodos no se conocen entre sí: toda la coordinación pasa por el centro de la estrella. Los archivos de los *shards* llegan a cada nodo por un volumen de solo lectura, de modo que por la red viajan solo pesos y parciales, del orden de cien kilobytes por época (@tab:cluster).

**Por qué esta topología.** La @tab:topologias compara las alternativas que se evaluaron en la PC3.

Tabla: Alternativas de topología para el entrenamiento distribuido {#tab:topologias}

| Alternativa | Decisión | Motivo |
|---|---|---|
| Estrella con *parameter server* síncrono | **Elegida** | Cada época es una barrera: el coordinador envía los pesos, espera los parciales de todos los *shards* y actualiza. Reproduce exactamente el descenso de gradiente secuencial |
| *Parameter server* asíncrono | Descartada | Los nodos calcularían con pesos de épocas anteriores y el orden de las actualizaciones dependería de la red: el resultado no sería reproducible ni equivalente al secuencial |
| *Ring all-reduce* entre nodos | Descartada | Cada nodo tendría que conocer a sus vecinos y el orden de la suma dependería del anillo; con pocos nodos y mensajes pequeños, la red no es el cuello de botella que *all-reduce* resuelve |
| Reducción local en cada nodo | Descartada | Cada nodo enviaría un solo vector en lugar de uno por bloque, con menos tráfico, pero el orden de la suma cambiaría con el número de nodos y se perdería la equivalencia bit a bit |

Nota: Elaboración propia; decisión registrada en el ADR-0005.

## 9.2 Protocolo TCP

Una época es un ***fan-out* distribuido** (el coordinador envía los pesos a todos los nodos) seguido de un ***fan-in* distribuido** (el coordinador recoge y agrega los parciales), con la condición de que la agregación respeta el orden de bloque (@fig:epoca).

![Una época del entrenamiento distribuido: envío de los pesos, cálculo en cada nodo, recepción y reducción en orden de bloque](fig/epoca.svg){#fig:epoca}

Nota: Diagrama de actividades UML 2.5 con particiones. Elaboración propia a partir de `(*Coordinador).GradienteErr` y `(*Nodo).Servir`.

Los mensajes son **objetos JSON, uno por línea**, sobre la conexión TCP (@tab:mensajes). Se eligió JSON por línea en lugar de un formato binario porque se puede leer y depurar con herramientas comunes, y porque `encoding/json` escribe cada `float64` con la representación decimal más corta que vuelve exactamente al mismo valor: los bits de los pesos y de los parciales se conservan en la red, condición necesaria para la equivalencia de la sección 9.5. Cada mensaje tiene un tamaño máximo (`MaxMensaje`) y se valida al recibirlo (`Mensaje.Validar`): tipo conocido, campos obligatorios presentes, enteros no negativos y ningún NaN ni infinito.

Tabla: Mensajes del protocolo entre el coordinador y los nodos {#tab:mensajes}

| Mensaje | Sentido | Campos | Cuándo |
|---|---|---|---|
| `registro` | nodo → coordinador | Nombre del nodo, versión del protocolo, número de workers | Al conectarse |
| `asignacion` | coordinador → nodo | *Shard*, ruta y SHA-256 del archivo, bloque inicial, número y tamaño de bloques, variables, hiperparámetros | La primera vez que el nodo recibe ese *shard* |
| `listo` | nodo → coordinador | Lo que el nodo leyó en la cabecera del archivo | Tras cargar y validar el *shard* |
| `epoca` | coordinador → nodo | Número de época, *shard* y pesos $w$ | Al inicio de cada época |
| `gradiente` | nodo → coordinador | Época, *shard*, bloque inicial, **un parcial por bloque** y el tiempo de cálculo | Al terminar el cálculo |
| `error` | ambos | Época, *shard* y descripción | *Shard* inválido, pesos de largo incorrecto, gradiente no finito, versión incompatible |
| `fin` | coordinador → nodo | Motivo: `completo` o `abortado` | Al terminar el entrenamiento |

Nota: Elaboración propia a partir de `internal/cluster/protocolo`.

El nodo devuelve un parcial **por bloque** y no la suma de su *shard*: si cada nodo sumara sus bloques y el coordinador sumara los totales de los nodos, el orden de las sumas cambiaría con el número de nodos (sección 8.4). El coordinador coloca cada parcial en el *slot* de su índice global de bloque (`bloque_inicial` + posición) y reduce todos en orden, así que el orden de llegada de los nodos no importa.

Los ejemplos siguientes los genera `cmd/informe` con los tipos reales del paquete `protocolo`, los valida con `Mensaje.Validar` y los serializa con `encoding/json`, igual que en la red. Para que quepan en la página usan un modelo de juguete de tres variables y un *shard* de dos bloques de cuatro filas; los parciales del mensaje `gradiente` los calcula `ml.ParcialBloque` sobre esos datos con los pesos de la época 0. Los campos en cero no viajan (por eso el mensaje de la época 0 no lleva el campo `epoca`) y la ruta es la del volumen de solo lectura dentro del contenedor del nodo. En el clúster real, `w` y cada `grad` tienen <!--v:add .S.M.S.D 1-->32<!--/v--> componentes y un *shard* tiene decenas de bloques.

<!-- generado:protocolo -->
```json
// registro (nodo → coordinador)
{"tipo":"registro","nodo":"node-2","version":1,"workers":16}
// asignacion (coordinador → nodo)
{"tipo":"asignacion","shard":1,"ruta":"/datos/shard_1.bin","sha256":"75ca917c38ed1e797c255aafb490410ed96dc2c442eed5668987611c5aa7657d","bloque_inicial":2,"num_bloques":2,"tam_bloque":4,"d":3,"config":{"lambda":0.0001,"peso_pos":1.3333333333333333,"peso_neg":0.8}}
// listo (nodo → coordinador)
{"tipo":"listo","shard":1,"bloque_inicial":2,"num_bloques":2,"n":8,"d":3}
// epoca (coordinador → nodo)
{"tipo":"epoca","shard":1,"w":[0,0,0,0]}
// gradiente (nodo → coordinador)
{"tipo":"gradiente","shard":1,"bloque_inicial":2,"parciales":[{"grad":[-2.0933333333333333,-0.2666666666666666,0.8,-0.5333333333333332],"perdida":2.9574279703890998,"peso":4.266666666666667},{"grad":[-1.12,-0.2666666666666666,0.8,0.5333333333333334],"perdida":2.5877494740904625,"peso":3.7333333333333334}],"us_calculo":52,"us_cpu":41}
// error (ambos)
{"tipo":"error","epoca":7,"shard":1,"mensaje":"llegaron 3 pesos, el shard necesita 4"}
// fin (coordinador → nodo)
{"tipo":"fin","motivo":"completo"}
```
<!-- /generado -->

**Concurrencia en el coordinador.** Una goroutine lectora por conexión convierte cada mensaje recibido en un evento y lo envía a un único canal. Solo el bucle de épocas escribe a los nodos, así que nunca hay dos escritores sobre la misma conexión, y solo esa goroutine modifica el estado de la época (dueño de cada *shard*, plazos y parciales recibidos). La API lee una copia publicada de ese estado, protegida por su propio mutex, sin detener la época. Toda escritura en la red lleva un plazo (`SetWriteDeadline`): un nodo lento o colgado no puede bloquear el envío a los demás.

## 9.3 Algoritmo del coordinador y del nodo

**Bucle de entrenamiento.** Es el mismo para la versión secuencial, el *worker pool* y el clúster: solo cambia el calculador que se le pasa. Por eso el clúster actualiza los pesos con exactamente las mismas operaciones que el entrenamiento secuencial.

<!-- generado:fuente:internal/ml/logistica.go#EntrenarFalible -->
```go
// internal/ml/logistica.go
// EntrenarFalible es Entrenar para un calculador que puede fallar. Es el único
// bucle de entrenamiento del proyecto: secuencial, worker pool y clúster
// actualizan los pesos con exactamente las mismas operaciones. Si una época
// falla devuelve el error junto con los pesos y la historia hasta la época
// anterior.
func EntrenarFalible(calc CalculadorFalible, d int, e Entrenamiento) (Resultado, error) {
	w := make([]float64, d+1)
	historia := make([]float64, 0, e.Epocas)
	for epoca := range e.Epocas {
		g, err := calc.GradienteErr(w)
		if err != nil {
			return Resultado{W: w, Historia: historia}, fmt.Errorf("ml: época %d: %w", epoca, err)
		}
		historia = append(historia, g.Perdida)
		for j := range w {
			w[j] -= e.Tasa * g.Grad[j]
		}
	}
	return Resultado{W: w, Historia: historia}, nil
}
```
<!-- /generado -->

**Algoritmo 1. Una época en el coordinador** (`(*Coordinador).GradienteErr`):

1. Recibir los pesos $w$ de la época y marcar todos los *shards* como pendientes.
2. **Cubrir:** para cada *shard* pendiente sin una tarea en curso, elegir su dueño (el actual si está sano; si no, el nodo sano con menos *shards*, y ante empate el de menor índice), enviarle `asignacion` si todavía no lo cargó y `epoca`, y fijar el plazo propio de ese *shard*. Si un envío falla, descartar al nodo y probar con el siguiente; si no queda ninguno, abortar.
3. Mientras falten *shards*, esperar el siguiente evento o el plazo más próximo.
4. Si llega `gradiente` del dueño actual y de esta época, copiar sus parciales en los *slots* de su índice global de bloque y marcar el *shard* como listo. Las respuestas tardías, duplicadas o de nodos descartados se ignoran.
5. Si un nodo cae (error de lectura o fin de la conexión), descartarlo: sus *shards* pendientes quedan sin dueño.
6. Si vence un plazo, descartar solo a los dueños de los *shards* **cuyo propio plazo venció**.
7. Después de cada evento, volver a cubrir: se reasignan todos los *shards* sin dueño. Repetir desde el paso 3.
8. **Barrera:** con los parciales de todos los bloques, reducir en orden de bloque y finalizar, $\nabla_w L = \frac{1}{S}\sum_{k} g_k + \lambda w$.
9. Devolver el gradiente a `EntrenarFalible`, que actualiza $w \leftarrow w - \eta\,\nabla_w L$.

![Algoritmo del entrenamiento distribuido: bucle de épocas y una época en el coordinador, con la reasignación de shards](fig/flujo.svg){#fig:flujo}

Nota: Diagrama de actividades UML 2.5. A la izquierda, el bucle de entrenamiento, el mismo del secuencial; a la derecha, lo que hace el coordinador dentro de una época. Elaboración propia.

<!-- generado:fuente:internal/cluster/coordinador.go#(*Coordinador).GradienteErr -->
```go
// internal/cluster/coordinador.go
// GradienteErr implementa ml.CalculadorFalible: ejecuta una época en el clúster
// y devuelve el gradiente completo, idéntico bit a bit al secuencial.
func (c *Coordinador) GradienteErr(w []float64) (ml.Gradiente, error) {
	c.muBucle.Lock()
	defer c.muBucle.Unlock()
	if c.fallo != nil {
		return ml.Gradiente{}, c.fallo
	}
	if len(w) != c.op.D+1 {
		return ml.Gradiente{}, fmt.Errorf("cluster: len(w) = %d, se esperaban %d", len(w), c.op.D+1)
	}
	ini := time.Now()
	clear(c.calcNodo)
	clear(c.envioNodo)
	clear(c.ultimoNodo)
	clear(c.shardsNodo)
	clear(c.cpuNodo)
	c.calcSuma = 0
	c.w = w
	for s := range c.listo {
		c.listo[s], c.enviado[s] = false, false
	}
	pendientes := len(c.op.Shards)
	if err := c.cubrir(); err != nil {
		return ml.Gradiente{}, c.abortar(err)
	}
	for pendientes > 0 {
		plazo := time.NewTimer(time.Until(c.proximoVencimiento()))
		select {
		case ev := <-c.eventos:
			plazo.Stop()
			if c.atender(ev) {
				pendientes--
			}
		case <-plazo.C:
			c.vencer(time.Now())
		case <-c.hecho:
			plazo.Stop()
			return ml.Gradiente{}, ErrCerrado
		}
		// Segunda pasada: recién después de procesar el suceso completo (que
		// pudo descartar a varios nodos) se reasignan los shards sin dueño.
		if err := c.cubrir(); err != nil {
			return ml.Gradiente{}, c.abortar(err)
		}
	}
	// Barrera cumplida: están los parciales de los B bloques. Reducción en
	// orden global de bloque, igual que la versión secuencial.
	iniRed := time.Now()
	g := ml.Finalizar(ml.ReducirEnOrden(c.parciales, c.op.D), c.op.Config, w)
	fin := time.Now()
	c.epoca++
	c.muEstado.Lock()
	c.tiempos.Epocas++
	c.tiempos.SegEpoca += fin.Sub(ini).Seconds()
	c.tiempos.SegReduccion += fin.Sub(iniRed).Seconds()
	var calcMax int64
	for _, us := range c.calcNodo {
		calcMax = max(calcMax, us)
	}
	c.tiempos.SegCalculoMax += float64(calcMax) / 1e6
	c.tiempos.SegCalculoSuma += float64(c.calcSuma) / 1e6
	c.muEstado.Unlock()
	c.cerrarMetricas()
	c.publicar(g.Perdida)
	if c.op.Observador != nil {
		c.emitir(Evento{Tipo: EvEpoca, Epoca: c.epoca, Perdida: g.Perdida, Pesos: append([]float64(nil), w...)})
	}
	return g, nil
}
```
<!-- /generado -->

La elección del dueño de un *shard* (paso 2) y el descarte de un nodo (pasos 5 y 6) son:

<!-- generado:fuente:internal/cluster/coordinador.go#(*Coordinador).elegir -->
```go
// internal/cluster/coordinador.go
// elegir devuelve el nodo sano con menos shards asignados; si empatan, el de
// menor índice. Devuelve -1 si no hay nodos sanos.
func (c *Coordinador) elegir() int {
	mejor, cargaMejor := -1, 0
	for j, n := range c.nodos {
		if !n.sano {
			continue
		}
		if carga := c.carga(j); mejor < 0 || carga < cargaMejor {
			mejor, cargaMejor = j, carga
		}
	}
	return mejor
}
```
<!-- /generado -->

<!-- generado:fuente:internal/cluster/coordinador.go#(*Coordinador).descartar -->
```go
// internal/cluster/coordinador.go
// descartar saca a un nodo del clúster: cierra su conexión y deja sin dueño
// todos sus shards. Los que no habían llegado en esta época quedan pendientes
// de reenvío; cubrir los reasigna.
func (c *Coordinador) descartar(j int, motivo string) {
	n := c.nodos[j]
	if !n.sano {
		return
	}
	n.sano, n.motivo = false, motivo
	n.conn.Cerrar()
	for s, d := range c.duenio {
		if d != j {
			continue
		}
		c.duenio[s] = -1
		if !c.listo[s] {
			c.enviado[s] = false
		}
	}
	c.op.Log("época %d: nodo %s descartado: %s", c.epoca, n.nombre, motivo)
	c.publicarNodos()
	if c.op.Observador != nil {
		c.emitir(Evento{Tipo: EvDescarte, Epoca: c.epoca, Nodo: n.nombre, Detalle: motivo, Nodos: c.vistaNodos()})
	}
}
```
<!-- /generado -->

**Algoritmo 2. El nodo** (`(*Nodo).Servir`): registrarse; por cada `asignacion`, leer el archivo del *shard*, verificar su SHA-256 y su cabecera y responder `listo`; por cada `epoca`, calcular con su *worker pool* un parcial por bloque del *shard* y responder `gradiente`; terminar al recibir `fin` (@fig:nodo).

![Ciclo de vida de un nodo: conexión, registro, carga del shard y cálculo de cada época](fig/nodo.svg){#fig:nodo}

Nota: Máquina de estados UML 2.5. Elaboración propia a partir de `cmd/node` y `(*Nodo).Servir`.

<!-- generado:fuente:internal/cluster/nodo.go#(*Nodo).Servir -->
```go
// internal/cluster/nodo.go
// Servir registra el nodo por conn y atiende al coordinador hasta `fin`.
// Devuelve nil si el entrenamiento terminó completo, ErrAbortado si el
// coordinador abortó, o el error de conexión.
func (n *Nodo) Servir(conn net.Conn) error {
	if n.Log == nil {
		n.Log = func(string, ...any) {}
	}
	pc := protocolo.Nueva(conn)
	defer pc.Cerrar()
	defer func() {
		if n.pool != nil {
			n.pool.Cerrar()
			n.pool = nil
		}
	}()
	n.shards = map[int]gold.Archivo{}
	workers := max(n.Workers, 1)
	if err := pc.Enviar(protocolo.Mensaje{Tipo: protocolo.Registro, Nodo: n.Nombre, Version: protocolo.Version, Workers: workers}, 0); err != nil {
		return err
	}
	for {
		m, err := pc.Recibir(0)
		if err == io.EOF {
			return errors.New("cluster: el coordinador cerró la conexión")
		}
		if err != nil {
			return err
		}
		var resp protocolo.Mensaje
		switch m.Tipo {
		case protocolo.Asignacion:
			resp = n.cargar(m, workers)
		case protocolo.Epoca:
			resp = n.calcular(m)
		case protocolo.Fin:
			n.Log("fin del entrenamiento: %s", m.Motivo)
			if m.Motivo == protocolo.MotivoAbortado {
				return ErrAbortado
			}
			return nil
		default:
			resp = fallo(m, "mensaje inesperado %q", m.Tipo)
		}
		err = pc.Enviar(resp, 0)
		if errors.Is(err, protocolo.ErrNoFinito) {
			// Un gradiente con NaN o ±Inf no es un dato: se informa como error.
			err = pc.Enviar(fallo(m, "gradiente no finito"), 0)
		}
		if err != nil {
			return err
		}
	}
}
```
<!-- /generado -->

<!-- generado:fuente:internal/cluster/nodo.go#(*Nodo).calcular -->
```go
// internal/cluster/nodo.go
// calcular devuelve un parcial por bloque del shard, sin reducir.
func (n *Nodo) calcular(m protocolo.Mensaje) protocolo.Mensaje {
	a, ok := n.shards[m.Shard]
	if !ok {
		return fallo(m, "shard %d no cargado", m.Shard)
	}
	if len(m.W) != a.D+1 {
		return fallo(m, "llegaron %d pesos, el shard necesita %d", len(m.W), a.D+1)
	}
	cpu0, okCPU := cpuProceso()
	ini := time.Now()
	parciales := n.pool.Parciales(a.Dataset, m.W)
	r := protocolo.Mensaje{Tipo: protocolo.Gradiente, Epoca: m.Epoca, Shard: m.Shard,
		BloqueInicial: a.BloqueInicial, Parciales: parciales, MicrosCalculo: time.Since(ini).Microseconds()}
	if cpu1, ok := cpuProceso(); okCPU && ok {
		cpu := max(cpu1-cpu0, 0) // solo métrica: no entra en el gradiente
		r.MicrosCPU = &cpu
	}
	return r
}
```
<!-- /generado -->

Dentro del nodo, una sola goroutine atiende la conexión: lee un mensaje, lo procesa y responde, en orden. El paralelismo está en el `ml.Pool` de la sección 8.3, que se crea una sola vez por proceso y se comparte entre todos los *shards* que el nodo reciba. Así, el sistema combina los dos niveles que pide el enunciado: **concurrencia** dentro de cada nodo (goroutines sobre los bloques de su *shard*) y **distribución** entre nodos (procesos en contenedores distintos que se comunican por TCP).

## 9.4 Tolerancia a fallos

El coordinador distingue cinco fallas y responde a cada una sin detener el entrenamiento mientras quede al menos un nodo sano (@tab:fallas).

Tabla: Fallas que detecta el coordinador y cómo responde {#tab:fallas}

| Falla | Cómo se detecta | Qué hace el coordinador | Test |
|---|---|---|---|
| El nodo cae y el sistema cierra la conexión | Fin de la conexión o error en la goroutine lectora | Lo descarta y reasigna **todos** sus *shards* pendientes en la misma época | `TestCortarNodoEnLaEpocaKReasignaYEsIgualAlSecuencial` |
| El nodo se cuelga sin cerrar la conexión | Vence el plazo **de su *shard*** | Descarta solo a ese dueño; un nodo que acaba de recibir un *shard* reasignado conserva su plazo completo | `TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano` |
| Falla el envío a un nodo | Error o plazo vencido en la escritura | Lo cuenta como caída y prueba con el siguiente nodo de menor carga | `TestSinNodosSanosElEntrenamientoAbortaConErrSinNodos` |
| El nodo responde con datos inválidos o de otra época | Validación del mensaje y de la forma de sus parciales | Descarta al nodo, o ignora la respuesta si es tardía | `TestNodoFalsoConGradienteDeFormaIncorrectaSeDescarta` |
| No queda ningún nodo sano | — | Aborta con un error explícito y envía `fin` con motivo `abortado`: nunca espera indefinidamente | `TestSinNodosParaUnShardIlegibleElNodoRecibeFinAbortado` |

Nota: Elaboración propia. Todos los tests están en `internal/cluster/cluster_test.go` y `eventos_test.go` y se ejecutan con `go test -race`.

Un nodo que se registra tarde, o que se reconecta tras una caída, queda **de reserva** (sin *shards*) y hereda trabajo si otro nodo cae (`TestNodoQueSeRegistraTardeQuedaDeReserva`). Cuando hay que reasignar varios *shards* huérfanos, van al nodo sano con menos carga, no todos al primero (`TestBalanceoTrasDobleCaidaReparteDosYDos`). La reasignación **no cambia el resultado**: los *shards* son archivos fijos en un volumen compartido, y la reducción depende solo del índice de bloque, no de qué nodo calculó cada uno.

**Evidencia en Docker.** `scripts/prueba-caida.sh` levanta el clúster con los datos reales, espera a que el entrenamiento llegue a una época intermedia, mata un nodo con `docker kill` y exige tres cosas en el log del coordinador: que descartó al nodo por conexión perdida, que reasignó su *shard* y que los pesos finales son idénticos bit a bit a los del entrenamiento secuencial.

<!-- evidencia: evidencias/caida.txt -->

El mismo comportamiento se ve en el panel de administración de la SPA, que recibe los eventos del clúster por WebSocket: la @fig:admin-caida muestra la bitácora después de tumbar un nodo desde el panel, con su caída y su recuperación: la reasignación de su *shard* si el corte ocurre durante una época, o su reconexión como nodo de reserva si ocurre en la pausa entre dos épocas.

<!-- captura: Bitácora del panel de administración tras tumbar un nodo: caída y recuperación (reasignación de su shard o reconexión) {#fig:admin-caida} [capturas/e2e-09-admin-nodo-caido.png] -->

## 9.5 Distribuido igual a secuencial

La propiedad central del clúster es que entrena **exactamente el mismo modelo** que la versión secuencial: no un modelo parecido, sino los mismos bits en cada peso y en cada valor de la pérdida. Se comprueba en tres niveles:

1. **Tests unitarios por TCP.** `TestEquivalenciaClusterSecuencialBitABit` levanta un coordinador y varios nodos en el mismo proceso, comunicados por TCP real, entrena y compara bit a bit con `ml.Entrenar` sobre `ml.Secuencial`; `TestEquivalenciaConMenosNodosQueShards` repite la comparación cuando un nodo atiende varios *shards*.
2. **Docker con los datos reales.** Con `VERIFICAR=true`, el coordinador repite el entrenamiento en secuencial dentro de su contenedor y falla si los pesos o la historia de la pérdida difieren en un solo bit. Es lo que muestra la evidencia siguiente, y la misma orden se ejecuta en cada Pull Request en el trabajo «Docker» del CI.
3. **Benchmark.** Cada repetición del benchmark del clúster (sección 11) se compara con el entrenamiento secuencial antes de registrarse (columna «Igual bit a bit» de la @tab:cluster).

<!-- evidencia: evidencias/cluster.txt -->

El resultado del último entrenamiento en Docker queda en `reports/entrenamiento_cluster.json`. El guion de evidencias ejecuta la prueba de caída (sección 9.4) después del entrenamiento con verificación, así que ese reporte corresponde a la corrida en la que se mató un nodo: <!--v:entero (index .R "entrenamiento_cluster").nodos_pedidos-->3<!--/v--> nodos, <!--v:entero (index .R "entrenamiento_cluster").epocas-->600<!--/v--> épocas, <!--v:entero (index .R "entrenamiento_cluster").cluster.reasignaciones-->1<!--/v--> reasignaciones de *shard* y `igual_al_secuencial` = `<!--v:(index .R "entrenamiento_cluster").igual_al_secuencial-->true<!--/v-->`.

## 9.6 Métricas del clúster: CPU, latencia y red

Además de los pesos, el coordinador registra en cada época cuatro métricas por nodo, que la API expone en `GET /api/v1/cluster` y el panel de administración muestra en vivo:

- **Cálculo (ms):** lo que el nodo informa que tardó en calcular sus parciales (`us_calculo` del mensaje `gradiente`), sin red ni serialización.
- **Latencia (ms):** desde que el coordinador empieza a enviar la tarea de la época al nodo hasta que recibe su último gradiente.
- **Red (ms):** la diferencia entre latencia y cálculo: transmisión, serialización JSON y cola.
- **Uso de CPU:** tiempo de CPU (usuario + sistema) que el proceso del nodo gastó durante el cálculo, leído con `getrusage`, dividido entre el tiempo de cálculo por el número de workers del nodo. Un valor cercano a 1 indica que sus workers estuvieron ocupados todo el cálculo.

<!-- generado:fuente:internal/cluster/coordinador.go#MetricasNodo -->
```go
// internal/cluster/coordinador.go
// MetricasNodo resume la última época en que un nodo calculó al menos un shard.
// LatenciaMs va de que el coordinador empieza a enviar la primera tarea de la
// época al nodo hasta que recibe su último gradiente; CalculoMs es lo que el
// nodo dice que tardó en calcular; RedMs, la diferencia (red, JSON y cola).
// UsoCPU es la CPU del proceso del nodo durante el cálculo sobre la capacidad
// de sus workers (0..1); nil si el nodo no la informa (p. ej. fuera de Linux).
type MetricasNodo struct {
	// Epoca cuenta desde 1 (las épocas ya cerradas, como Estado.Epocas); los
	// eventos llevan la época en curso desde 0 (Evento.Epoca).
	Epoca      int      `json:"epoca"`
	Shards     int      `json:"shards"`
	CalculoMs  float64  `json:"calculo_ms"`
	LatenciaMs float64  `json:"latencia_ms"`
	RedMs      float64  `json:"red_ms"`
	UsoCPU     *float64 `json:"uso_cpu,omitempty"`
}
```
<!-- /generado -->

La @fig:admin-metricas muestra estas métricas al terminar un entrenamiento con tres nodos. La medición de la CPU no afecta al modelo: se toma antes y después del cálculo y no entra en el gradiente.

<!-- captura: Topología del clúster en el panel de administración con el uso de CPU y la latencia de cada nodo {#fig:admin-metricas} [capturas/e2e-10-admin-metricas.png] -->

**Costo de observar el entrenamiento.** Publicar cada evento del clúster por WebSocket podría frenar el bucle de épocas. Para evitarlo, los eventos se publican en un *hub* con búfer que descarta mensajes si un cliente no los lee a tiempo, en lugar de bloquear al coordinador. La @tab:observador mide el costo: la diferencia es de pocos puntos porcentuales, cambia de signo entre configuraciones y es pequeña frente al tiempo de la época; las dos corridas producen el mismo modelo bit a bit.

<!-- generado:plantilla:observador -->
Tabla: Costo de publicar las métricas en vivo durante el entrenamiento {#tab:observador}

| Nodos | ms por época sin observador | ms por época con observador | Diferencia (%) |
|---:|---:|---:|---:|
| 1 | 99,31 | 96,77 | -2,55 |
| 3 | 34,11 | 34,49 | 1,11 |
| 5 | 22,53 | 23,99 | 6,45 |

Nota: AMD Ryzen 5 3600 6-Core Processor, 12 CPU lógicas, windows; datos: gold(data/gold/train.bin, filas=765297, features=31). «Con observador»: cada evento del clúster se serializa a JSON y se publica en el *hub* de WebSocket con un suscriptor conectado. Se publicaron 3 792 eventos y se descartaron 0. Las dos corridas se compararon bit a bit con el entrenamiento secuencial. Fuente: `reports/bench_observador_jorge-pc_gold.json`.
<!-- /generado -->

El desglose del tiempo de cada época entre cálculo, red y espera, y los bytes que viajan por época, se analizan con el número de nodos en la sección 11.3.
