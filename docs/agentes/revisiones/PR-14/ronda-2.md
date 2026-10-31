# Veredicto — PR-14 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/4-perfil-datos` @ `42be54c`
- **Fecha:** 2026-10-31
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 -v -cover ./internal/ingest/...
=== RUN   TestParsearLinea
--- PASS: TestParsearLinea (0.00s)
=== RUN   TestEncabezadoReal
--- PASS: TestEncabezadoReal (0.02s)
=== RUN   TestPartirEnBloquesNoCortaLineas
--- PASS: TestPartirEnBloquesNoCortaLineas (0.03s)
=== RUN   TestMapearEnOrdenEsDeterministaYSinCarreras
--- PASS: TestMapearEnOrdenEsDeterministaYSinCarreras (0.02s)
=== RUN   TestUltimaLineaSinSaltoFinal
--- PASS: TestUltimaLineaSinSaltoFinal (0.00s)
=== RUN   TestPerfilConcurrenteIgualAlSecuencial
--- PASS: TestPerfilConcurrenteIgualAlSecuencial (8.43s)
=== RUN   TestPerfilMuestraReal
--- PASS: TestPerfilMuestraReal (0.00s)
=== RUN   TestEdadEnAnios
--- PASS: TestEdadEnAnios (0.00s)
=== RUN   TestPerfilCuentaErroresDeColumnas
--- PASS: TestPerfilCuentaErroresDeColumnas (0.00s)
PASS
coverage: 99.4% of statements
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	10.762s	coverage: 99.4% of statements
$ go test -race -count=1 ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.989s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	10.762s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	10.095s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0, cobertura `internal/ingest`: 99,4 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestPerfilConcurrenteIgualAlSecuencial` y `cmd/perfil` verifican igualdad exacta con el secuencial) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Valida con el dataset real (1 065 274 filas, 117,9 MB, sha256 en `data/MANIFEST.sha256`) el caso, la variable objetivo (A97.1/A97.2 vs. A97.0, 11,32 % positivos) y la decisión de entrenar desde 2011 (`ADR-0003`). |
| Implementación | 8 | Avance sólido de Parte A (en curso) | Partición por bloques de ~1 MiB alineados a `\n` (`PartirEnBloques`), worker pool genérico sin mutex con slots exclusivos (`MapearEnOrden`), parser RFC 4180 zero-copy (`ParsearLinea`) y reducción determinista (`PerfilarConcurrente`), 100 % stdlib y con cobertura de 99,4 %. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | Genera automáticamente `docs/informe/pc3/02-datos.md` (incluyendo ahora el conteo de `no_numerica`) desde `reports/perfil_bronze.json` mediante `cmd/informe` (`git diff` vacío tras `go run ./cmd/informe`). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos nuevos. Todos los hallazgos de la ronda 1 han sido resueltos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) | RESUELTO | Resuelto en `e94ec50`: `TestPerfilCuentaErroresDeColumnas` (`internal/ingest/perfil_test.go:118-132`) verifica `ErrColumnas=2` y `ErrComillas=1` en `PerfilarBloque`, y `TestEncabezadoReal` (`internal/ingest/csv_test.go:72-80`) verifica un encabezado de 15 columnas con nombre inválido (`semana_epi`) y otro en mayúsculas con espacios. |
| H2 (ronda 1) | RESUELTO | Resuelto en `e94ec50`: `cmd/informe/main.go:55, 112` y `docs/informe/pc3/02-datos.md:89` incorporan el campo `no_numerica` (`NoNum`), y `TestUltimaLineaSinSaltoFinal` (`internal/ingest/csv_test.go:139-158`) cubre `SepararEncabezado`, `PartirEnBloques` y `ParaCadaLinea` sin `\n` final. |

## Cómo explicarlo en la sustentación

Cargamos el CSV completo (117,9 MB, 1 065 274 filas) en memoria y `PartirEnBloques` calcula los rangos `[Ini, Fin)` de ~1 MiB buscando el siguiente `\n`, de modo que ninguna fila quede partida a la mitad y los límites de los 113 bloques dependan solo de los bytes del archivo, nunca del número de hilos. Luego `MapearEnOrden` reparte los índices `0..112` por un canal a un *worker pool* donde cada goroutine construye un `*Perfil` privado y lo escribe en su propio slot `res[k]` sin mutex; tras `wg.Wait()`, el hilo principal une `res[0..112]` en orden de índice y ordena las categorías con desempate determinista, obteniendo exactamente el mismo JSON que el secuencial con un *speedup* de 1,83× en 2 workers.
