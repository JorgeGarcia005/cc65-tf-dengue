/*
 * barrera.pml — modelo del protocolo coordinador–nodos (issue #9).
 * Propuesta: docs/decisiones/propuestas/protocolo-tcp.md
 *
 * Un coordinador (parameter server síncrono) y N nodos, uno por shard.
 * Cada época el coordinador manda los pesos a cada shard y espera el
 * gradiente de los N shards (barrera) antes de pasar a la siguiente.
 *
 * Fallos modelados:
 *   - CAIDAS:    un nodo deja de responder para siempre al recibir una época.
 *   - SOSPECHAS: el temporizador de época vence aunque el nodo esté vivo
 *                (nodo lento). Su respuesta puede llegar después.
 * En ambos casos el coordinador descarta al nodo y reasigna el shard a un
 * nodo sano; si no queda ninguno, aborta.
 *
 * El timeout real se modela con `timeout` de Promela (nada más puede
 * ejecutarse: nadie va a responder) y la sospecha falsa con una opción
 * no determinista que puede dispararse en cualquier momento.
 *
 * Verificación:
 *   spin -a barrera.pml
 *   gcc -O2 -DNOCLAIM -o pan-seg pan.c && ./pan-seg    # deadlocks y asserts
 *   gcc -O2 -o pan pan.c && ./pan -a -N termina        # LTL: toda corrida termina
 * (con el LTL incluido, pan desactiva la búsqueda de estados finales
 * inválidos; por eso la seguridad se verifica aparte con -DNOCLAIM)
 */

#define N         3   /* nodos = shards */
#define EPOCAS    3
#define CAIDAS    1   /* nodos que pueden caer en toda la corrida */
#define SOSPECHAS 1   /* timeouts falsos sobre nodos vivos */

mtype = { EPOCA, GRADIENTE, FIN };

/* (tipo, época, shard)
 * Enviar a un nodo no debe bloquear al coordinador aunque el nodo esté caído
 * (en TCP el mensaje va al búfer del kernel, y la escritura real lleva plazo).
 * Con capacidad N, Spin halló una corrida en que el coordinador quedaba
 * bloqueado enviando a un nodo caído con el canal lleno. A un nodo le pueden
 * quedar pendientes a lo sumo N mensajes de la época anterior más N de la
 * actual (luego vence el plazo y se lo descarta): 2N basta. */
chan aNodo[N] = [2*N] of { mtype, byte, byte };
chan aCoord   = [2*N] of { mtype, byte, byte };

byte caidas = 0;
bool terminado = false;
bool abortado = false;
byte epocasCompletas = 0;

proctype Nodo(byte id) {
	byte e, s;
end:
	do
	:: aNodo[id] ? EPOCA(e, s) ->
		if
		:: atomic { caidas < CAIDAS -> caidas++ }; goto caido
		:: aCoord ! GRADIENTE(e, s)
		fi
	:: aNodo[id] ? FIN(e, s) -> break
	od;
caido:
	skip
}

/* Dos pasadas: primero se descarta a TODOS los dueños de shards que no
 * llegaron y recién después se reasigna. En una sola pasada, un shard podía
 * reasignarse a un nodo caído que aún no se había descartado, y luego nunca
 * volvía a reasignarse (su dueño ya figuraba como descartado): ciclo de
 * timeouts sin progreso. Si no queda ningún nodo sano, aborta. */
inline reasignar() {
	for (s : 0 .. N-1) {
		if
		:: !listo[s] -> descartado[duenio[s]] = true
		:: else -> skip
		fi
	}
	for (s : 0 .. N-1) {
		if
		:: !listo[s] ->
			j = 0;
			do
			:: j < N && descartado[j] -> j++
			:: else -> break
			od;
			if
			:: j == N -> abortado = true; goto cierre
			:: else -> duenio[s] = j; aNodo[j] ! EPOCA(e, s)
			fi
		:: else -> skip
		fi
	}
}

proctype Coordinador() {
	byte e = 0, s, ep, r, j, sospechas = 0;
	byte duenio[N];
	bool listo[N];
	bool descartado[N];

	for (s : 0 .. N-1) {
		duenio[s] = s
	}

	do
	:: e < EPOCAS ->
		for (s : 0 .. N-1) {
			listo[s] = false;
			aNodo[duenio[s]] ! EPOCA(e, s)
		}
		r = 0;
		do
		:: r == N -> break
		:: aCoord ? GRADIENTE(ep, s) ->
			if
			:: ep == e && !listo[s] -> listo[s] = true; r++
			:: else -> skip   /* tardía o duplicada: se descarta */
			fi
		:: timeout -> reasignar()
		:: sospechas < SOSPECHAS -> sospechas++; reasignar()
		od;
		/* barrera: la época avanza solo con los N shards */
		for (s : 0 .. N-1) {
			assert(listo[s])
		}
		e++;
		epocasCompletas = e
	:: else -> break
	od;

cierre:
	for (j : 0 .. N-1) {
		if
		:: !descartado[j] -> aNodo[j] ! FIN(0, 0)
		:: else -> skip
		fi
	}
	/* sin pérdida de shards: si no se abortó, se completaron todas las épocas */
	assert(abortado || epocasCompletas == EPOCAS);
	terminado = true
}

init {
	byte i;
	atomic {
		for (i : 0 .. N-1) {
			run Nodo(i)
		}
		run Coordinador()
	}
}

ltl termina { <> terminado }
