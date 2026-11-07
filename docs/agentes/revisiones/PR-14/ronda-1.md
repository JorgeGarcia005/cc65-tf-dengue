# Veredicto — PR-14 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/4-perfil-datos` @ `4b7fa61`
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
--- PASS: TestEncabezadoReal (0.00s)
=== RUN   TestPartirEnBloquesNoCortaLineas
--- PASS: TestPartirEnBloquesNoCortaLineas (0.02s)
=== RUN   TestMapearEnOrdenEsDeterministaYSinCarreras
--- PASS: TestMapearEnOrdenEsDeterministaYSinCarreras (0.02s)
=== RUN   TestPerfilConcurrenteIgualAlSecuencial
--- PASS: TestPerfilConcurrenteIgualAlSecuencial (3.84s)
=== RUN   TestPerfilMuestraReal
--- PASS: TestPerfilMuestraReal (0.00s)
=== RUN   TestEdadEnAnios
--- PASS: TestEdadEnAnios (0.00s)
PASS
coverage: 95.6% of statements
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.080s	coverage: 95.6% of statements
$ go test -race -count=1 ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.989s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	3.406s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	10.095s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0, cobertura `internal/ingest`: 95,6 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestPerfilConcurrenteIgualAlSecuencial` verifica igualdad exacta de JSON para bloques de 4 KiB y 64 KiB con 1, 2, 4 y 8 workers; `cmd/perfil` verifica `"concurrente_igual_a_secuencial": true` sobre las 1 065 274 filas reales) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Valida con el dataset real (1 065 274 filas, 117,9 MB, sha256 en `data/MANIFEST.sha256`) el caso, la variable objetivo (A97.1/A97.2 vs. A97.0, 11,32 % positivos) y la decisión de entrenar desde 2011 (`ADR-0003`). |
| Implementación | 8 | Avance sólido de Parte A (en curso) | Partición por bloques de ~1 MiB alineados a `\n` (`PartirEnBloques`), worker pool genérico sin mutex con slots exclusivos (`MapearEnOrden`), parser RFC 4180 zero-copy (`ParsearLinea`) y reducción determinista (`PerfilarConcurrente`), 100 % stdlib y con speedup de 1,83× en 2 workers. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | Genera automáticamente `docs/informe/pc3/02-datos.md` desde `reports/perfil_bronze.json` mediante `cmd/informe` (comprobado con `go run ./cmd/informe` + `git diff` vacío), cumpliendo `AGENTS.md` §3.6 (cero cifras a mano). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H1 — Ramas de error `ErrColumnas` en `PerfilarBloque` y nombre inválido en `ValidarEncabezado` sin test directo · MENOR

- **Dónde:** `internal/ingest/perfil.go:98-101`, `internal/ingest/perfil_test.go:46-48`, `internal/ingest/csv.go:47-51` y `internal/ingest/csv_test.go:69-71`
- **Evidencia:** En `csvSintetico` (`internal/ingest/perfil_test.go:46-48`) solo se inyectan filas defectuosas con comillas sin cerrar (`ErrComillas`), por lo que la rama `default: p.FilasMalas++; p.ErrColumnas++` en `PerfilarBloque` (`internal/ingest/perfil.go:98-101`) no se ejecuta durante los tests. Del mismo modo, en `TestEncabezadoReal` (`internal/ingest/csv_test.go:69`), `ValidarEncabezado([]byte("a,b,c"))` retorna en `ParsearLinea` por `ErrColumnas`, sin ejercitar el bucle que compara el nombre de cada una de las 15 columnas (`internal/ingest/csv.go:47-51`).
- **Corrección sugerida:** Inyectar en `csvSintetico` (o en un test puntual) al menos una línea con distinto número de columnas para verificar `p.ErrColumnas > 0`, y añadir un caso en `TestEncabezadoReal` con 15 columnas donde una tenga nombre distinto al esperado.

### H2 — Campo `no_numerica` omitido en la plantilla de `cmd/informe` y corte sin `\n` final en `bloques.go` · MENOR

- **Dónde:** `cmd/informe/main.go:52-55, 103-110` e `internal/ingest/bloques.go:27-28, 45-47, 94-95`
- **Evidencia:** `ingest.Histograma` exporta `NoNum` (`"no_numerica": 0` en `reports/perfil_bronze.json:1640`), pero la estructura anónima `Edad` en `cmd/informe/main.go:52-55` y la sección 2.4 de la plantilla (`cmd/informe/main.go:103-110`) no lo incluyen (aunque en el CSV real vale 0). Por otro lado, en `TestPartirEnBloquesNoCortaLineas` (`internal/ingest/csv_test.go:76-78`) todas las filas terminan en `\r\n`, dejando sin cubrir las guardas para un archivo cuya última línea no termine en `\n` en `SepararEncabezado` (`bloques.go:27`), `PartirEnBloques` (`bloques.go:46`) y `ParaCadaLinea` (`bloques.go:95`).
- **Corrección sugerida:** Incluir `NoNum int `json:"no_numerica"`` en `cmd/informe/main.go` (mostrándolo al pie de la tabla 2.4) y probar `PartirEnBloques`/`ParaCadaLinea` con un buffer sin `\n` al final.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Ronda 1 (sin rondas previas). |

## Cómo explicarlo en la sustentación

Cargamos el CSV completo (117,9 MB, 1 065 274 filas) en memoria y `PartirEnBloques` calcula los rangos `[Ini, Fin)` de ~1 MiB buscando el siguiente `\n`, de modo que ninguna fila quede partida a la mitad y los límites de los 113 bloques dependan solo de los bytes del archivo, nunca del número de hilos. Luego `MapearEnOrden` reparte los índices `0..112` por un canal a un *worker pool* donde cada goroutine construye un `*Perfil` privado y lo escribe en su propio slot `res[k]` sin mutex; tras `wg.Wait()`, el hilo principal une `res[0..112]` en orden de índice y ordena las categorías con desempate determinista, obteniendo exactamente el mismo JSON que el secuencial con un *speedup* de 1,83× en 2 workers.
