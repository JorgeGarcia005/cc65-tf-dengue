# Veredicto — PR-15 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/5-loader-concurrente` @ `66c96f9`
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
=== RUN   TestUltimaLineaSinSaltoFinal
--- PASS: TestUltimaLineaSinSaltoFinal (0.00s)
=== RUN   TestParsearLineaTolerante
--- PASS: TestParsearLineaTolerante (0.00s)
=== RUN   TestLimpiarCamposValida
--- PASS: TestLimpiarCamposValida (0.00s)
=== RUN   TestCadaReglaSeDetecta
--- PASS: TestCadaReglaSeDetecta (0.00s)
=== RUN   TestUbigeoDeCincoDigitosSeCorrige
--- PASS: TestUbigeoDeCincoDigitosSeCorrige (0.00s)
=== RUN   TestEdadEnMesesYDias
--- PASS: TestEdadEnMesesYDias (0.00s)
=== RUN   TestSilverIdaYVuelta
--- PASS: TestSilverIdaYVuelta (0.00s)
=== RUN   TestLimpiezaConcurrenteIgualALaSecuencial
--- PASS: TestLimpiezaConcurrenteIgualALaSecuencial (8.41s)
=== RUN   TestMuestraRealPasaLaLimpieza
--- PASS: TestMuestraRealPasaLaLimpieza (0.00s)
=== RUN   TestPerfilConcurrenteIgualAlSecuencial
--- PASS: TestPerfilConcurrenteIgualAlSecuencial (8.83s)
=== RUN   TestPerfilMuestraReal
--- PASS: TestPerfilMuestraReal (0.00s)
=== RUN   TestEdadEnAnios
--- PASS: TestEdadEnAnios (0.00s)
=== RUN   TestPerfilCuentaErroresDeColumnas
--- PASS: TestPerfilCuentaErroresDeColumnas (0.00s)
PASS
coverage: 97.9% of statements
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	19.512s	coverage: 97.9% of statements
$ go test -race -count=1 ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	2.890s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	9.818s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	20.482s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0, cobertura `internal/ingest`: 97,9 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestLimpiezaConcurrenteIgualALaSecuencial` verifica igualdad `sha256` byte a byte del CSV silver y conteos con bloques de 2 KiB y 64 KiB y 1, 2, 4 y 8 workers; `cmd/loader` verifica `"silver_concurrente_igual_al_secuencial": true` con sha256 `192f2824...` sobre las 1 065 274 filas reales) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Documenta y cuantifica las 11 reglas de limpieza en orden de precedencia, la recuperación de 85 filas con comillas sin escapar en `localidad` (83 en Loreto) y la decisión epidemiológica de conservar las 148 787 filas idénticas (sin ID de paciente pueden ser casos reales distintos). |
| Implementación | 8 | Avance sólido de Parte A (en curso) | Implementa `LimpiarConcurrente` y `cmd/loader` (modos limpieza y `-bench`) usando solo stdlib, con buffers privados por bloque, concatenación en orden de índice de bloque e igualdad bit a bit garantizada frente a la ejecución secuencial (speedup 1,67× con 2 workers, IC95 bootstrap). Además incorpora las mejoras de cobertura de PR-14 (H1 y H2). |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | `cmd/informe` genera `docs/informe/pc3/03-limpieza.md` íntegramente desde `reports/limpieza.json` y `reports/bench_loader_*.json` (verificado con `go run ./cmd/informe` y `git diff` vacío), cumpliendo `AGENTS.md` §3.6. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H1 — `res.Reparadas` se incrementa antes de validar `LimpiarCampos` y ramas `Reparadas`/`Corregidas` sin cubrir en `LimpiarBloque` · MENOR

- **Dónde:** `internal/ingest/limpieza.go:229-239` e `internal/ingest/limpieza_test.go:103-139`
- **Evidencia:** En `LimpiarBloque` (`internal/ingest/limpieza.go:229-236`), `if reparada { res.Reparadas++ }` se ejecuta antes de `r, regla, corr := LimpiarCampos(campos)` y `if regla != OK { res.PorRegla[regla]++; return }`. Si una fila repara sus comillas pero incumple alguna regla posterior (p. ej. `EdadFueraDeRango`), se contabiliza simultáneamente como descartada y como reparada en el silver, a diferencia de `res.Corregidas++` (`limpieza.go:237-239`), que solo cuenta las filas que pasan `LimpiarCampos`. Además, `go tool cover -func` muestra `LimpiarBloque 91.3%` y `Objetivo 66.7%` porque `csvSintetico` no inyecta filas con comillas reparables ni ubigeos de 5 dígitos al probar `LimpiarConcurrente`.
- **Corrección sugerida:** Mover `if reparada { res.Reparadas++ }` debajo de la guarda `if regla != OK` en `LimpiarBloque`, e inyectar en `csvSintetico` (o en un test de `LimpiarBloque`) una fila con comillas reparables y otra con ubigeo de 5 dígitos, además de verificar `r.Objetivo() == 0` para `Clase == 0`.

### H2 — `parsearLaxo` acepta comillas sueltas en campos que no inician con comilla · MENOR

- **Dónde:** `internal/ingest/csv.go:74-103`
- **Evidencia:** El comentario de `parsearLaxo` (`internal/ingest/csv.go:74-75`) especifica: *"en un campo que empieza con comillas, solo la comilla seguida de ',' o de fin de línea cierra el campo"*. Sin embargo, en la rama `else` (`csv.go:95-103`, cuando el campo no empieza con `"`), no se comprueba `bytes.IndexByte(campo, '"') >= 0` como sí hace `ParsearLineaN` (`csv.go:160-162`). Como resultado, una línea con una comilla suelta en un campo sin entrecomillar (p. ej. `A,B"C,...`) falla en `ParsearLineaN` con `ErrComillas` y es aceptada silenciosamente por `parsearLaxo` como `reparada = true`.
- **Corrección sugerida:** Si solo se desea reparar campos entrecomillados con comillas internas sin escapar, mantener en la rama `else` de `parsearLaxo` la validación `if bytes.IndexByte(campo, '"') >= 0 { return campos, ErrComillas }` (o actualizar el comentario de `parsearLaxo` si se busca admitir también comillas en campos no entrecomillados).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Ronda 1 (sin rondas previas). Nota: este PR también resolvió los hallazgos menores H1 y H2 de PR-14. |

## Cómo explicarlo en la sustentación

El loader (`cmd/loader`) divide en memoria el CSV de 1 065 274 filas en 113 bloques de ~1 MiB alineados a `\n` y los procesa con el *worker pool* `MapearEnOrden`: cada worker ejecuta `LimpiarBloque` sobre un acumulador privado (`ResultadoBloque`), aplicando un parser CSV en dos fases (`ParsearLineaTolerante`, que recupera 85 filas reales con comillas sin escapar en `localidad`) y 11 reglas deterministas en orden fijo de prioridad. Al terminar los workers, el hilo principal concatena los buffers `Silver` y reduce los conteos y huellas `maphash` en orden estricto de índice de bloque (`0..112`), garantizando que el archivo `data/silver/dengue.csv` (1 065 260 filas válidas, solo 14 descartadas por `edad > 110`) sea idéntico bit a bit (`sha256`) con 1, 2 o más workers y logrando un *speedup* de 1,67× en 2 vCPU.
