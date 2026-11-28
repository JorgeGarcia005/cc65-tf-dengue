# 10. Verificación formal con Spin

Los tests de la sección 9 prueban el protocolo en las situaciones que alguien pensó en escribir. Un protocolo concurrente puede fallar en un orden de eventos que nadie imaginó: un nodo que cae justo cuando vence el plazo de otro, una reasignación que llega a un canal lleno. Para esos casos se usa **verificación de modelos** (*model checking*): se describe el protocolo en Promela y Spin (Holzmann, 2003) recorre **todos** los estados alcanzables del modelo, con todas las intercalaciones posibles de los procesos, buscando uno que viole una propiedad. Esta sección responde a la observación 5 de la PC4 («¿en qué consisten las validaciones de `spin/barrera.pml`?»): qué se modela (10.1 y 10.2), qué comprueba cada aserción y cada propiedad LTL (10.3 y 10.4), cómo se descartan interbloqueos y carreras (10.5), qué error reintroduce cada mutante (10.6) y la evidencia de la ejecución (10.7).

## 10.1 Qué se modela y qué no

El modelo representa la **coordinación** del clúster: quién recibe cada *shard*, cuándo se descarta un nodo, cuándo avanza la barrera y cuándo termina el entrenamiento. Es lo que más fácil se equivoca en un protocolo distribuido y lo que menos cubren los tests.

Tabla: Alcance del modelo barrera.pml {#tab:spin-alcance}

| Se modela | No se modela | Dónde se verifica lo que no se modela |
|---|---|---|
| Un coordinador y N nodos (3 o 4), un *shard* por nodo al empezar | Los valores de los pesos y de los gradientes | La equivalencia bit a bit se prueba con tests en Go (sección 9.5) |
| Los mensajes `epoca`, `gradiente`, `fin` y el cierre de conexión (EOF) | `registro`, `asignacion`, `listo` y `error` | Tests de registro y carga en `internal/cluster` |
| Envío no bloqueante: si el canal del nodo está lleno, se descarta al nodo | El tamaño real del búfer TCP | Se verifican los dos extremos: capacidad N y capacidad 1 |
| Dos formas de caer (EOF y cuelgue) y una sospecha falsa sobre un nodo vivo | Tiempos reales: el vencimiento de un plazo es una elección no determinista | Los plazos reales se prueban con tests en Go (`TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano`) |
| Plazo por *shard*, reasignación en dos pasadas y elección del nodo sano de menor carga | El orden de la reducción dentro de la barrera | `TestEquivalencia` y `TestShardsReproducenElGradienteGlobal` |
| Dos épocas (una con N = 4), hasta dos caídas | Seiscientas épocas | Para el protocolo, las épocas solo se distinguen por qué nodos ya se descartaron; dos épocas cubren el paso de una a la siguiente en ese estado |

Nota: Elaboración propia a partir de `spin/barrera.pml`.

Así, Spin **no** prueba que el gradiente distribuido sea igual al secuencial; prueba que, para cualquier orden de llegadas, caídas y vencimientos de plazo, el coordinador nunca se bloquea, nunca avanza la barrera sin todos los *shards*, nunca pierde un *shard*, nunca descarta a un nodo sano sin motivo y siempre termina. La igualdad de los números se prueba en Go. Las dos verificaciones se complementan.

## 10.2 El modelo barrera.pml

**Canales y estado.** Cada nodo tiene un canal de entrada `aNodo[j]` (la conexión TCP del coordinador hacia él) y todos comparten `aCoord` (las goroutines lectoras que reenvían al bucle de épocas). Las variables «fantasma» guardan lo que el coordinador **no** sabe, como si un nodo sigue vivo, y solo se usan para escribir las propiedades:

<!-- generado:archivo:spin/barrera.pml#canales -->
```promela
/* spin/barrera.pml */
mtype = { EPOCA, GRADIENTE, FIN, EOF };

/* (tipo, época, shard, nodo) */
chan aNodo[N] = [CAP] of { mtype, byte, byte, byte };
chan aCoord   = [3*N] of { mtype, byte, byte, byte };

/* variables fantasma: lo que el coordinador NO sabe, para las propiedades */
bool vivo[N];          /* el nodo no cayó */
bool sospechado[N];    /* su plazo venció estando vivo */
bool saturado[N];      /* un envío lo encontró con el canal lleno (equivale a
                        * que venza el plazo de escritura: cuenta como caída) */
byte caidas = 0;

bool terminado = false;
bool abortado = false;
byte epocasCompletas = 0;
```
<!-- /generado -->

**Nodo.** En cada época, un nodo puede responder el gradiente, caer cerrando la conexión (el coordinador lee EOF) o colgarse sin responder ni cerrar (solo lo detecta un plazo). Spin explora las tres opciones en cada época de cada nodo, hasta `CAIDAS` caídas en total. La etiqueta `end:` le indica a Spin que esperar un mensaje en ese punto es un estado final válido:

<!-- generado:archivo:spin/barrera.pml#nodo -->
```promela
/* spin/barrera.pml */
proctype Nodo(byte id) {
	byte e, s, x;
end:
	do
	:: aNodo[id] ? EPOCA(e, s, x) ->
		if
		:: aCoord ! GRADIENTE(e, s, id)
		/* cae y el SO cierra la conexión: el coordinador lee EOF */
		:: atomic { caidas < CAIDAS -> caidas++; vivo[id] = false }; aCoord ! EOF(0, 0, id); goto muerto
		/* se cuelga: no responde ni cierra; solo lo detecta el plazo */
		:: atomic { caidas < CAIDAS -> caidas++; vivo[id] = false }; goto muerto
		fi
	:: aNodo[id] ? FIN(e, s, x) -> break
	od;
muerto:
	skip
}
```
<!-- /generado -->

**Coordinador.** El proceso `Coordinador` sigue el Algoritmo 1 de la sección 9.3: envía la época, espera eventos en la barrera, descarta, cubre los *shards* pendientes y asigna cada uno al nodo sano de menor carga. La @tab:spin-correspondencia relaciona cada parte del modelo con el código Go que representa.

Tabla: Correspondencia entre el modelo Promela y el código Go {#tab:spin-correspondencia}

| Modelo (`barrera.pml`) | Código Go (`internal/cluster/coordinador.go`) |
|---|---|
| `chan aNodo[N]` con `full()` / `nfull()` | Escritura con `SetWriteDeadline`: si el nodo no lee, el envío falla |
| `chan aCoord` | Canal de eventos que alimentan las goroutines `leer` |
| `DESCARTAR(j)` | `(*Coordinador).descartar` |
| Etiqueta `cubrir:` | `(*Coordinador).cubrir` |
| Etiqueta `asignar:` | `(*Coordinador).elegir` y `despachar` |
| Opciones de vencimiento de plazo en la barrera | `(*Coordinador).vencer` con el plazo propio de cada *shard* |
| Etiqueta `avanzar:` | Fin de `GradienteErr`: `ReducirEnOrden` y `Finalizar` |
| Etiqueta `cierre:` | `avisarFin` y `abortar` |

Nota: Elaboración propia.

## 10.3 Aserciones

Una aserción `assert(c)` es una condición que debe valer **cada vez** que la ejecución pasa por ese punto, en cualquier intercalación. El modelo tiene seis.

**1. Nunca se descarta a un nodo vivo sin motivo.** Al descartar, el nodo debe haber caído, o haber sido sospechado (su plazo venció estando vivo, como ocurre con un nodo lento), o haber tenido el canal lleno. Si el coordinador descartara a un nodo sano por un error de lógica, por ejemplo por el plazo de **otro** *shard*, la aserción fallaría.

<!-- generado:archivo:spin/barrera.pml#descartar -->
```promela
/* spin/barrera.pml */
/* descarta a j; si ya estaba descartado no hace nada (como descartar()) */
#define DESCARTAR(jj) \
	if \
	:: !descartado[jj] -> descartado[jj] = true; carga[jj] = 0; \
	   assert(!vivo[jj] || sospechado[jj] || saturado[jj]) /* nunca a un nodo vivo sin motivo */ \
	:: else -> skip \
	fi
```
<!-- /generado -->

**2. Nunca se envía una época a un nodo descartado.** Antes de cada envío, `assert(!descartado[j])`. Junto con el envío no bloqueante, garantiza que el coordinador no espera respuesta de un nodo que ya dio por perdido.

<!-- generado:archivo:spin/barrera.pml#envio -->
```promela
/* spin/barrera.pml */
#ifdef MUT_BLOQUEANTE
				aNodo[j] ! EPOCA(e, s, j)
#else
				if
				:: full(aNodo[j]) -> saturado[j] = true; DESCARTAR(j)
				:: nfull(aNodo[j]) -> assert(!descartado[j]); aNodo[j] ! EPOCA(e, s, j)
				fi
#endif
```
<!-- /generado -->

**3. La barrera avanza solo con todos los *shards*.** Al pasar a la época siguiente, `assert(listo[k])` para cada *shard*: ningún camino del modelo permite actualizar los pesos con un gradiente incompleto.

**4. Balanceo.** Entre los nodos sanos, la carga (número de *shards*) difiere a lo sumo en uno. Comprueba que las reasignaciones repartan el trabajo y no lo amontonen en un nodo, que se volvería el más lento de cada época.

<!-- generado:archivo:spin/barrera.pml#barrera -->
```promela
/* spin/barrera.pml */
avanzar:
		/* barrera: la época avanza solo con los N shards. (Una etiqueta pegada
		 * a un for de Spin salta su inicialización: por eso el skip.) */
		skip;
		for (k : 0 .. N-1) {
			assert(listo[k])
		}
		/* balanceo: entre los nodos sanos, la carga difiere a lo sumo en 1 */
		for (k : 0 .. N-1) {
			for (j : 0 .. N-1) {
				assert(descartado[k] || descartado[j] || carga[k] <= carga[j] + 1)
			}
		}
		e++;
		epocasCompletas = e
```
<!-- /generado -->

**5. No se pierden *shards*.** Al terminar, o se abortó o se completaron todas las épocas.

**6. Solo se aborta con motivo.** Si el entrenamiento abortó, ningún nodo estaba vivo y sin sospecha ni canal lleno: el coordinador no se rinde mientras quede un nodo sano.

<!-- generado:archivo:spin/barrera.pml#cierre -->
```promela
/* spin/barrera.pml */
cierre:
	skip;
	for (k : 0 .. N-1) {
		if
		:: !descartado[k] && nfull(aNodo[k]) -> aNodo[k] ! FIN(0, 0, k)
		:: descartado[k] || full(aNodo[k]) -> skip
		fi
	}
	/* sin pérdida de shards: si no se abortó, se completaron todas las épocas */
	assert(abortado || epocasCompletas == EPOCAS);
	/* solo se aborta si no quedaba ningún nodo vivo y no sospechado */
	if
	:: abortado ->
		for (k : 0 .. N-1) {
			assert(!vivo[k] || sospechado[k] || saturado[k])
		}
	:: else -> skip
	fi;
	terminado = true
```
<!-- /generado -->

## 10.4 Propiedades LTL

Las aserciones hablan de puntos concretos de la ejecución. Que el entrenamiento **siempre termine** es una propiedad sobre ejecuciones completas, y se expresa en lógica temporal lineal (LTL):

<!-- generado:archivo:spin/barrera.pml#ltl -->
```promela
/* spin/barrera.pml */
ltl termina { <> terminado }
```
<!-- /generado -->

La fórmula $\Diamond\,\mathit{terminado}$ se lee «en algún momento futuro, `terminado` es verdadero». Spin la verifica convirtiendo su negación ($\Box\,\neg\mathit{terminado}$, «nunca termina») en un autómata de Büchi y buscando un ciclo de aceptación: una ejecución infinita en la que el coordinador nunca llega a `terminado`, por ejemplo porque dos nodos se pasan un *shard* indefinidamente o porque la barrera espera un gradiente que nunca llegará. Si no existe, toda ejecución termina, completa o abortada. La verificación se hace **sin** suponer equidad entre procesos (no se usa la opción `-f` de `pan`): Spin también considera ejecuciones en las que algún proceso casi nunca avanza, y aun así no encuentra una que no termine.

## 10.5 Ausencia de carreras y de interbloqueo

**Interbloqueo.** La verificación de seguridad (`pan` compilado con `-DNOCLAIM`) busca, además de violaciones de aserciones, **estados finales inválidos**: estados en los que ningún proceso puede avanzar y alguno no está en su final ni en una etiqueta `end`. Es la definición de interbloqueo en Spin. El caso típico es el coordinador esperando en la barrera mientras el nodo que debía responder está bloqueado, o esperando enviar a un canal lleno. Con 0 errores, ninguna intercalación del modelo llega a un estado así.

**Carreras.** En el modelo, el estado del coordinador (dueños, cargas, *shards* listos) es local a su proceso, y los nodos solo se comunican con él por canales. Las únicas variables globales son las fantasma, que representan hechos que el coordinador real no puede leer (si un nodo sigue vivo); el modelo las usa para decidir cuándo puede vencer un plazo y para escribir las propiedades, los nodos las modifican dentro de bloques `atomic` y Spin explora todas las intercalaciones de esos accesos. Esto refleja el diseño del código, donde solo la goroutine del bucle de épocas modifica el estado de la época (sección 9.2). En el código Go, la ausencia de carreras sobre la memoria la comprueba el detector de carreras: todos los tests de `internal/cluster`, incluidos los de caídas concurrentes, corren con `go test -race` en cada Pull Request.

**Tamaño del espacio de estados.** Se verifica con tres configuraciones (@tab:spin): N = 3 con canal de capacidad N (el caso normal), N = 3 con capacidad 1 (el caso más exigente para el envío no bloqueante, porque el canal se llena con un solo mensaje) y N = 4 con una época (necesario para que el balanceo tenga más de una opción). Spin informa en cada corrida si la búsqueda quedó incompleta por memoria o profundidad, y `verificar.sh` falla en ese caso: una búsqueda truncada con 0 errores no prueba nada.

## 10.6 Mutantes

Que Spin no encuentre errores solo tiene valor si es capaz de encontrarlos. Para demostrarlo, el modelo incluye cuatro **mutantes**: variantes que reintroducen, cada una, un error de diseño real del proyecto. Se activan con una opción del preprocesador y `verificar.sh` exige que **cada uno dé un contraejemplo**.

Tabla: Mutantes del modelo, error que reintroducen y contraejemplo que encuentra Spin {#tab:spin-mutantes}

| Mutante (`-D`) | Qué cambia | Origen del error | Contraejemplo de Spin |
|---|---|---|---|
| `MUT_BLOQUEANTE` (con capacidad 1) | El envío espera a que haya espacio en el canal en lugar de descartar al nodo | Primer diseño de la PC3 | Estado final inválido (interbloqueo): el coordinador queda bloqueado enviando a un nodo que dejó de leer su canal |
| `MUT_UNA_PASADA` | Al vencer un plazo se reasigna solo ese *shard*, no todos los del nodo caído | Primer diseño de la PC3 | Estado final inválido: los otros *shards* del nodo caído quedan sin dueño y la barrera espera para siempre |
| `MUT_PLAZO_GLOBAL` | Un solo plazo para toda la época: al vencer, se descarta a los dueños de todos los *shards* pendientes | Objeción del juez en el debate de diseño de la PC4 | Violación de la aserción 1: se descarta a un nodo vivo que acababa de recibir un *shard* reasignado |
| `MUT_PRIMER_SANO` (N = 4) | Los *shards* huérfanos van al primer nodo sano, no al de menor carga | Objeción del juez en el debate de diseño de la PC4 | Violación de la aserción 4 (balanceo): un nodo sano acumula al menos dos *shards* más que otro |

Nota: Elaboración propia. El tipo de contraejemplo es el que informa `pan` para cada mutante; el número de estados explorados hasta encontrarlo está en la @tab:spin.

Los dos primeros errores los encontró este mismo modelo en la PC3, antes de escribir el código del clúster; los dos últimos los señaló el juez al revisar el diseño del protocolo y se corrigieron antes de implementarlo (ADR-0005). Mantenerlos como mutantes asegura que, si un cambio futuro debilitara el modelo, la verificación lo detectaría: un mutante que deja de fallar hace fallar a `verificar.sh`.

## 10.7 Evidencia de la verificación

`spin/verificar.sh` genera el verificador (`spin -a`), lo compila con `gcc`, lo ejecuta para cada configuración y cada mutante, comprueba el resultado esperado y escribe `reports/spin.json`, del que sale la @tab:spin. El CI ejecuta el mismo guion en cada Pull Request.

<!-- generado:plantilla:spin -->
Tabla: Verificaciones de Spin sobre el modelo del protocolo y sus mutantes {#tab:spin}

| Corrida | Opciones | Chequeo | Estados | Errores | Esperado | Cumple |
|---|---|---|---:|---:|---|:---:|
| modelo N=3 (canal de capacidad N) | — | seguridad | 17 949 492 | 0 | 0 | sí |
| modelo N=3 (canal de capacidad N) | — | termina | 17 814 453 | 0 | 0 | sí |
| modelo N=3, canal de capacidad 1 | `-DCAP=1` | seguridad | 26 025 432 | 0 | 0 | sí |
| modelo N=3, canal de capacidad 1 | `-DCAP=1` | termina | 25 707 167 | 0 | 0 | sí |
| modelo N=4, 1 época, sin sospechas | `-DN=4 -DEPOCAS=1 -DSOSPECHAS=0` | seguridad | 4 923 889 | 0 | 0 | sí |
| modelo N=4, 1 época, sin sospechas | `-DN=4 -DEPOCAS=1 -DSOSPECHAS=0` | termina | 4 920 817 | 0 | 0 | sí |
| mutante: envío bloqueante (capacidad 1) | `-DMUT_BLOQUEANTE -DCAP=1` | seguridad | 239 247 | 1 | > 0 (contraejemplo) | sí |
| mutante: reasigna solo el shard vencido | `-DMUT_UNA_PASADA` | seguridad | 36 613 | 1 | > 0 (contraejemplo) | sí |
| mutante: plazo global | `-DMUT_PLAZO_GLOBAL` | seguridad | 70 | 1 | > 0 (contraejemplo) | sí |
| mutante: primer nodo sano (N=4) | `-DMUT_PRIMER_SANO -DN=4 -DEPOCAS=1 -DSOSPECHAS=0` | seguridad | 5 269 | 1 | > 0 (contraejemplo) | sí |

Nota: «seguridad» busca violaciones de aserciones e interbloqueos (estados finales inválidos); «termina» verifica la propiedad LTL de terminación. Spin Version 6.5.2 -- 6 December 2019, gcc 14.2.0. Fuente: `reports/spin.json`, generado por `bash spin/verificar.sh`.
<!-- /generado -->

<!-- evidencia: evidencias/spin.txt -->
