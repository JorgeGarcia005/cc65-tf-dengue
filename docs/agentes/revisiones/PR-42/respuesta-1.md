# Respuesta — PR #42 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-20 · **Base:** `feature/32-benchmark-cluster` @ `6655118`

Acepto los nueve hallazgos. Para H1 escribí primero el test y confirmé el deadlock con el orden
anterior: el test se cuelga y vence a los 10 s.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `fc3790d`: `defer wg.Wait()` se registra antes que `defer c.Cerrar()`, así que la salida es Terminar → Cerrar → esperar a los nodos. Test `TestBenchmarkClusterSinTodosLosNodosVuelveConError` (se conecta 1 de 2 nodos, plazo de registro 300 ms): vuelve con error. Mutante con el orden viejo: `FAIL ... (10.00s): medirCluster quedó colgado` |
| H2 | BLOQUEANTE | ACEPTO | `214e82d`: al cerrarse una conexión, la lectora suma sus bytes a `bytesEnvCerradas`/`bytesRecCerradas` (bajo `muConns`) antes de borrarla; `Tiempos()` parte de esos acumulados. Test `TestTiemposConservaLosBytesDeConexionesCerradas`: tras `Terminar()` + `Cerrar()` los totales no bajan |
| H3 | BLOQUEANTE | ACEPTO | `0dbbb9b`: `cmd/informe` genera `docs/informe/pc4/fig/speedup_cluster.svg` desde `reports/bench_cluster_*.json` (prefiere el gold real), con eje X lineal de nodos y la recta ideal; `figuraSpeedupEjes` generaliza la figura existente sin cambiar las de la PC3 (`go run ./cmd/informe` deja `docs/informe/pc3` sin cambios) |
| H4 | MAYOR | ACEPTO | `fc3790d`: la serialización usa los **parciales reales** de cada shard (una pasada de `ml.Pool.Parciales` con los pesos de referencia), decodifica con `json.NewDecoder` como `protocolo.Conexion`, agrupa k pasadas con `pasadasPorMedicion` (sin cuantización de 0,5 ms) y se separa en `seg_json_cpu_todos_los_shards` (CPU sumada) y `seg_json_shard_mas_pesado` (camino crítico, incluido en red y espera); la tabla impresa muestra este último. **El JSON del gold real se regenera** en la PC de Jorge con el código nuevo (ver abajo) |
| H5 | MAYOR | ACEPTO | `214e82d` y `fc3790d`: las aserciones de la reducción pasan a `>= 0` (dura decenas de µs; en Windows puede medirse 0), conservando `< SegEpoca` |
| H6 | MAYOR | ACEPTO | `214e82d`: el cómputo se acumula por nodo durante la época (`calcNodo[j] += us`) y se toma el máximo entre nodos. Test `TestTiemposSumanLosShardsDeUnMismoNodo`: con un nodo y 3 shards, el cómputo del más lento es igual a la suma |
| H7 | MAYOR | ACEPTO | `fc3790d`: `-nodos` se copia, ordena y deduplica; Karp-Flatt solo se define con base de 1 nodo. Test `TestBenchmarkClusterOrdenaNodosYKarpFlattSoloContraUnNodo` (`3,1,2,2` → 1, 2, 3; base 2 → sin Karp-Flatt) |
| H8 | MENOR | ACEPTO | `fc3790d`: `-recorte-cluster` por defecto 0,20 (con 5 repeticiones recorta una por lado); `protocolo.desglose` aclara que el desglose son medias aritméticas |
| H9 | MENOR | ACEPTO | `214e82d`: caso `us_calculo negativo` en la tabla de mensajes inválidos, `TestConexionCuentaBytesEnviadosYRecibidos` bajo `-race` (envío y lectura concurrente de `Bytes()`), y `MicrosCalculo` en la tabla de mensajes de `protocolo.go` |

## Verificación

```text
$ gofmt -l . ; go vet ./cmd/benchmark ./cmd/informe ./internal/cluster/...   → vacío
$ go test -race -count=1 ./cmd/benchmark ./internal/cluster/...
ok  	.../cmd/benchmark	5.039s
ok  	.../internal/cluster	6.238s
ok  	.../internal/cluster/protocolo	3.738s
$ go run ./cmd/benchmark -cluster -filas 200000 -features 31 -nodos 3,1,2 -reps-cluster 3 -epocas-cluster 10
 nodos     ms/época          IC95 ms  speedup    efic.      K-F   cómp. ms red+esp ms JSON/shard ms     KB/época
     1        26.30   24.63–28.28        1.00     1.00        —      25.11       1.19          0.75         32.7
     2        12.91   12.86–12.95        2.04     1.02   -0.019      12.33       0.57          0.40         33.4
     3        16.22   16.13–16.32        1.62     0.54    0.425       9.39       6.82          0.27         34.1
```

(En la nube hay 2 núcleos: con 3 nodos de 1 worker compiten por la CPU; por eso la cifra válida es la de
la PC de Jorge.)

**Gold real regenerado con el código nuevo** (PC de Jorge: AMD Ryzen 5 3600, 12 CPU lógicas, Windows,
Go 1.27.3; `go run ./cmd/benchmark -cluster -gold data/gold/train.bin` y `go run ./cmd/informe`):

| Nodos | ms/época | Speedup | Eficiencia | Karp-Flatt | Cómputo del nodo más lento (ms) | Red y espera (ms) | JSON del shard más pesado (ms) | JSON CPU todos los shards (ms) | KB/época |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 92,00 | 1,00 | 1,00 | — | 89,60 | 2,62 | 2,340 | 2,340 | 112,5 |
| 2 | 49,64 | 1,85 | 0,93 | 0,079 | 48,49 | 1,45 | 1,200 | 2,253 | 113,3 |
| 3 | 35,38 | 2,60 | 0,87 | 0,077 | 34,30 | 1,00 | 0,753 | 2,195 | 114,1 |
| 4 | 27,58 | 3,34 | 0,83 | 0,066 | 26,69 | 0,89 | 0,572 | 2,229 | 114,8 |
| 5 | 22,48 | 4,09 | 0,82 | 0,055 | 22,11 | 0,78 | 0,473 | 2,287 | 115,6 |

Con parciales reales y k pasadas agrupadas, el JSON del shard más pesado queda por debajo de red y
espera en todas las configuraciones, como corresponde a una parte de ella, y el total de CPU es casi
constante (~2,2–2,3 ms), porque los 187 bloques se serializan igual sin importar cuántos nodos haya.
