# Veredicto — PR-18 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/17-bench-gold-real` @ `3977e8e`
- **Fecha:** 2026-11-07
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.986s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	2.799s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	2.843s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	2.785s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	2.315s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.479s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	2.177s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.127s
$ go test -race -count=1 -run "Equivalencia|ShardsReproducen|TamBloqueCero|PasadasPorMedicion" -v ./...
=== RUN   TestPasadasPorMedicionSuperaLaResolucionDelReloj
--- PASS: TestPasadasPorMedicionSuperaLaResolucionDelReloj (0.05s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.430s
=== RUN   TestDecodificarRechazaTamBloqueCero
--- PASS: TestDecodificarRechazaTamBloqueCero (0.02s)
=== RUN   TestShardsReproducenElGradienteGlobal
--- PASS: TestShardsReproducenElGradienteGlobal (0.08s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	2.038s
=== RUN   TestEquivalencia
=== RUN   TestEquivalencia/workers=1
=== RUN   TestEquivalencia/workers=2
=== RUN   TestEquivalencia/workers=4
=== RUN   TestEquivalencia/workers=8
=== RUN   TestEquivalencia/workers=16
=== RUN   TestEquivalencia/workers=64
--- PASS: TestEquivalencia (0.51s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	2.402s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia`, `TestShardsReproducenElGradienteGlobal`, `equivalente_bit_a_bit: true` en los 4 reportes nuevos de gradiente y `silver_igual_al_secuencial: true` en los 2 reportes nuevos del loader) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica el planteamiento ni el diseño metodológico. |
| Implementación | 8 | Avance sólido de PC3 (Partes A y B) y blindaje para PC4 | Resuelve H1 (`gold.Decodificar` rechaza `tam_bloque < 1`) y H2 (`cmd/benchmark -gold`), corrige la medición de pasadas cortas bajo relojes de baja resolución en Windows (`pasadasPorMedicion` ≥ 20 ms), unifica la detección de hardware en `internal/maquina` y aporta mediciones reales en una CPU de 8 núcleos / 16 hilos (Ryzen 7 5800X) tanto en Windows 11 como en Debian 13 (WSL2). |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (PC3 completo y multi-plataforma) | `docs/informe/pc3/03-limpieza.md` y `04-modelo.md` regenerados exactamente por `go run ./cmd/informe` (`git diff` vacío) a partir de `reports/*.json`, distinguiendo el sistema operativo en la cabecera de cada tabla. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes, mayores ni menores. Los dos hallazgos menores diferidos de `PR-16` quedan resueltos con sus respectivos tests:

- **PR-16 H1 (`RESUELTO`):** `internal/gold/gold.go:100-102` valida `tam < 1` en `Decodificar` antes de construir el `Archivo`, evitando cualquier división por cero posterior en `Particionar`. Cubierto por `TestDecodificarRechazaTamBloqueCero` (`internal/gold/gold_test.go:56-65`).
- **PR-16 H2 (`RESUELTO`):** `cmd/benchmark/main.go:88,107-116` incorpora el flag `-gold` que lee el archivo binario real (`data/gold/train.bin`, 765 297 filas × 31 variables, bloques de 4 096 filas) y genera `reports/bench_gradiente_<host>_gold.json`.

## Seguimiento de rondas previas

Primera ronda de revisión para PR-18 (cierra las observaciones diferidas H1 y H2 de PR-16).

## Cómo explicarlo en la sustentación

En una PC de 8 núcleos físicos y 16 hilos SMT (AMD Ryzen 7 5800X), una época de gradiente sobre el dataset gold real (765 297 filas × 31 variables, 187 bloques de 4 096 filas) baja de ~67,5 ms en secuencial a ~10,1 ms con 8 workers (speedup 6,7×, eficiencia ~84 %) y a ~6,8 ms con 16 workers (speedup ~10,0×, eficiencia ~62 %), manteniendo **igualdad bit a bit** con la versión secuencial tanto en Windows 11 como en Debian 13 (WSL2). La caída de eficiencia de 8 a 16 workers se explica porque los 16 hilos comparten 8 núcleos físicos mediante SMT, y con 32 workers el beneficio marginal ya se satura; además, el análisis de punto de equilibrio demuestra que por debajo de 10 000 filas (3 bloques) no conviene paralelizar porque el costo de coordinación supera el cómputo. En el loader (`cmd/loader -bench`), la limpieza en memoria de las 1 065 274 filas baja de ~550 ms a ~165 ms (speedup 3,25×–3,48×), limitada antes que el gradiente por el ancho de banda de memoria y las asignaciones del armado del CSV `silver`.
