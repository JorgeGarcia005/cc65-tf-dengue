# ADR-0002 — Worker pool para el gradiente dentro de un nodo

- **Estado:** Aceptado (24/10/2026) · Issue #8

## Contexto

Cada época del descenso de gradiente recorre todas las filas. Dentro de un nodo queremos
usar todos los núcleos sin romper el determinismo bit a bit exigido en AGENTS.md §3.

## Decisión

**Worker pool persistente + slots por bloque + barrera por época + reducción ordenada.**

1. Los workers se crean una vez y viven todo el entrenamiento (no se crean goroutines por época).
2. Un canal sin búfer reparte índices de bloque (fan-out).
3. Cada worker acumula el bloque en un **búfer privado** y lo copia a `parciales[k]` al terminar.
4. `sync.WaitGroup` es la barrera de fin de época.
5. Un solo goroutine reduce `parciales[0..B-1]` en orden de índice (fan-in determinista).

## Alternativas descartadas

| Alternativa | Por qué no |
|---|---|
| Mutex sobre un acumulador global | Serializa la parte caliente y el orden de suma depende de la planificación → resultados no reproducibles |
| `atomic` sobre floats | Go no tiene suma atómica de `float64`; con CAS el orden de suma también es no determinista |
| Un acumulador por worker (no por bloque) | Más barato en memoria, pero el resultado cambia con el número de workers |
| Acumular directo en `parciales[k]` | Correcto, pero **false sharing**: los slots de bloques vecinos comparten línea de caché; con 2 workers el pool resultó más lento que el secuencial. Se corrigió con el búfer privado (ver `reports/bench_gradiente_*.json`) |

## Consecuencias

- `TestEquivalencia` exige igualdad bit a bit con 1, 2, 4, 8, 16 y 64 workers, y corre con `-race`.
- El protocolo distribuido (PC4) reutiliza la misma idea: cada nodo envía sus parciales por
  bloque y el coordinador los reduce en orden global de bloque, así el número de nodos
  tampoco cambia el resultado.
- Con menos filas que un bloque (`TamBloque` = 4096) no hay nada que repartir: el punto de
  equilibrio queda por encima de ese tamaño.
