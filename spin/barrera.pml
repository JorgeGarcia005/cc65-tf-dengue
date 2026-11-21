/*
 * barrera.pml — modelo del protocolo coordinador–nodos al día con el
 * ADR-0005 (issue #31; versión anterior: issue #9).
 * Decisión: docs/decisiones/ADR-0005-protocolo-tcp.md
 *
 * Un coordinador (parameter server síncrono) y N nodos, un shard por nodo al
 * empezar. Cada época el coordinador manda los pesos de cada shard a su
 * dueño y espera los N gradientes (barrera) antes de pasar a la siguiente.
 *
 * Lo que modela, igual que internal/cluster/coordinador.go:
 *   - Envío no bloqueante: si el canal del nodo está lleno (el búfer TCP de un
 *     nodo que dejó de leer), el envío falla y el nodo se descarta en vez de
 *     bloquear el bucle de épocas. Canal de capacidad N.
 *   - Elección del nodo sano de menor carga (shards asignados); empate → menor
 *     índice.
 *   - Dos formas de caer: el nodo cierra la conexión (EOF, se detecta al
 *     instante) o se cuelga sin cerrarla (solo lo detecta su plazo).
 *   - Plazo por shard: vence el del shard cuyo dueño no respondió; se descarta
 *     solo a ese dueño. Además, una sospecha falsa: el plazo de un nodo vivo
 *     vence (nodo lento) y se lo descarta igual.
 *   - Dos pasadas: primero se descarta al nodo y después `cubrir` reasigna
 *     TODOS los shards pendientes cuyo dueño quedó descartado. Descartar a un
 *     nodo ya descartado no hace nada (como en el código).
 *   - Si no queda ningún nodo sano, se aborta (fin{abortado}).
 *
 * Propiedades:
 *   - seguridad (pan con -DNOCLAIM): sin deadlocks ni estados finales
 *     inválidos; la barrera solo avanza con los N shards; si no se aborta se
 *     completan todas las épocas; nunca se envía una época a un nodo
 *     descartado; entre los nodos sanos la carga difiere a lo sumo en 1; y
 *     nunca se descarta a un nodo vivo salvo por una sospecha o por un canal
 *     lleno (variables fantasma), así que solo se aborta si todos cayeron o
 *     fueron sospechados o saturados.
 *   - vivacidad (LTL `termina`): toda corrida termina.
 *
 * Mutantes (se compilan con -D; los corre verificar.sh y deben FALLAR):
 *   MUT_BLOQUEANTE  envío bloqueante (sin full(aNodo[j]))  → deadlock
 *   MUT_UNA_PASADA  al descartar se reasigna solo el shard que venció, no
 *                   todos los del nodo caído (sin `cubrir`) → no termina
 *   MUT_PLAZO_GLOBAL al vencer un plazo se descarta al dueño de todo shard
 *                   pendiente (plazo global)               → descarta a un nodo vivo
 *   MUT_PRIMER_SANO el shard va al primer nodo sano y no al de menor carga
 *                   (con N = 4)                            → rompe el balanceo
 *
 * Verificación: bash verificar.sh   (o los comandos de README.md)
 */

/* Parámetros (se pueden cambiar con -D; verificar.sh corre N = 3 y N = 4) */
#ifndef N
#define N         3   /* nodos = shards iniciales (3 o 4) */
#endif
#ifndef EPOCAS
#define EPOCAS    2
#endif
#ifndef CAIDAS
#define CAIDAS    2   /* nodos que pueden caer (EOF o colgarse) en la corrida */
#endif
#ifndef SOSPECHAS
#define SOSPECHAS 1   /* plazos que vencen sobre un nodo vivo (nodo lento) */
#endif

#ifndef CAP
#define CAP N
#endif

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

proctype Coordinador() {
	byte e = 0, s, x, ep, j, k, r, mejor, sospechas = 0;
	byte duenio[N];
	byte carga[N];
	bool listo[N];
	bool descartado[N];

/* descarta a j; si ya estaba descartado no hace nada (como descartar()) */
#define DESCARTAR(jj) \
	if \
	:: !descartado[jj] -> descartado[jj] = true; carga[jj] = 0; \
	   assert(!vivo[jj] || sospechado[jj] || saturado[jj]) /* nunca a un nodo vivo sin motivo */ \
	:: else -> skip \
	fi

	/* asignar(s): al nodo sano de menor carga (empate → menor índice); si el
	 * envío falla porque su canal está lleno, lo descarta y prueba el
	 * siguiente; sin nodos sanos, aborta. */
	for (k : 0 .. N-1) {
		duenio[k] = k;
		carga[k] = 1
	}

	do
	:: e < EPOCAS ->
		for (s : 0 .. N-1) {
			listo[s] = false
		}
		for (x : 0 .. N-1) {
			s = x;
			/* la época se difunde al dueño actual; si su canal está lleno se
			 * lo descarta y cubrir() reasigna */
			j = duenio[s];
			if
			:: descartado[j] -> skip  /* lo reasigna cubrir(), como en coordinador.go */
			:: else ->
#ifdef MUT_BLOQUEANTE
				aNodo[j] ! EPOCA(e, s, j)
#else
				if
				:: full(aNodo[j]) -> saturado[j] = true; DESCARTAR(j)
				:: nfull(aNodo[j]) -> assert(!descartado[j]); aNodo[j] ! EPOCA(e, s, j)
				fi
#endif
			fi
		}
		goto cubrir;
barrera:
		r = 0;
		for (k : 0 .. N-1) {
			if
			:: listo[k] -> r++
			:: else -> skip
			fi
		}
		if
		:: r == N -> goto avanzar
		:: else -> skip
		fi;
		do
		/* gradiente del dueño actual y sano; tardíos o duplicados se ignoran */
		:: aCoord ? GRADIENTE(ep, s, j) ->
			if
			:: ep == e && !listo[s] && duenio[s] == j && !descartado[j] -> listo[s] = true; goto barrera
			:: else -> skip
			fi
		/* EOF: el nodo cayó; se descarta y se cubren todos sus pendientes */
		:: aCoord ? EOF(ep, s, j) -> DESCARTAR(j); goto cubrir
		/* vence el plazo del shard x: su dueño no respondió (cayó o se colgó) */
		:: (!listo[0] && !vivo[duenio[0]] && !descartado[duenio[0]]) -> x = 0; goto vencer
		:: (!listo[1] && !vivo[duenio[1]] && !descartado[duenio[1]]) -> x = 1; goto vencer
		:: (!listo[2] && !vivo[duenio[2]] && !descartado[duenio[2]]) -> x = 2; goto vencer
#if N > 3
		:: (!listo[3] && !vivo[duenio[3]] && !descartado[duenio[3]]) -> x = 3; goto vencer
#endif
		/* sospecha falsa: vence el plazo de un nodo vivo (lento) */
		:: (sospechas < SOSPECHAS && !listo[0] && !descartado[duenio[0]]) -> x = 0; sospechas++; sospechado[duenio[0]] = true; goto vencer
		:: (sospechas < SOSPECHAS && !listo[1] && !descartado[duenio[1]]) -> x = 1; sospechas++; sospechado[duenio[1]] = true; goto vencer
		:: (sospechas < SOSPECHAS && !listo[2] && !descartado[duenio[2]]) -> x = 2; sospechas++; sospechado[duenio[2]] = true; goto vencer
#if N > 3
		:: (sospechas < SOSPECHAS && !listo[3] && !descartado[duenio[3]]) -> x = 3; sospechas++; sospechado[duenio[3]] = true; goto vencer
#endif
		od;

vencer:
#ifdef MUT_PLAZO_GLOBAL
		/* plazo global: al vencer, se descarta al dueño de todo shard pendiente */
		k = 0;
		do
		:: k < N ->
			if
			:: !listo[k] -> j = duenio[k]; DESCARTAR(j)
			:: else -> skip
			fi;
			k++
		:: else -> break
		od;
#else
		/* plazo por shard: solo al dueño del shard cuyo plazo venció */
		j = duenio[x];
		DESCARTAR(j);
#endif
#ifdef MUT_UNA_PASADA
		/* una pasada: se reasigna solo el shard que venció */
		s = x;
		goto asignar;
#endif
		goto cubrir;

cubrir:
		/* segunda pasada: todo shard pendiente con dueño descartado */
		s = N;
		for (k : 0 .. N-1) {
			if
			:: s == N && !listo[k] && descartado[duenio[k]] -> s = k
			:: else -> skip
			fi
		}
		if
		:: s == N -> goto barrera
		:: else -> skip
		fi;

asignar:
		/* nodo sano de menor carga; empate → menor índice */
		mejor = N;
		for (k : 0 .. N-1) {
			if
#ifdef MUT_PRIMER_SANO
			:: !descartado[k] && mejor == N -> mejor = k
#else
			:: !descartado[k] && (mejor == N || carga[k] < carga[mejor]) -> mejor = k
#endif
			:: else -> skip
			fi
		}
		if
		:: mejor == N -> abortado = true; goto cierre
		:: else -> skip
		fi;
		j = mejor;
#ifdef MUT_BLOQUEANTE
		aNodo[j] ! EPOCA(e, s, j);
		duenio[s] = j;
		carga[j]++;
#else
		if
		:: full(aNodo[j]) -> saturado[j] = true; DESCARTAR(j); goto asignar
		:: nfull(aNodo[j]) -> assert(!descartado[j]); aNodo[j] ! EPOCA(e, s, j);
			duenio[s] = j;
			carga[j]++
		fi;
#endif
#ifdef MUT_UNA_PASADA
		goto barrera;
#endif
		goto cubrir;

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
	:: else -> break
	od;

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
}

init {
	byte i;
	atomic {
		for (i : 0 .. N-1) {
			vivo[i] = true
		}
		for (i : 0 .. N-1) {
			run Nodo(i)
		}
		run Coordinador()
	}
}

ltl termina { <> terminado }
