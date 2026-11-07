# Veredicto — PR-15 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/5-loader-concurrente` @ `589fca8`
- **Fecha:** 2026-10-31
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -coverprofile=coverage.out ./internal/ingest/... && go tool cover -func=coverage.out
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	coverage: 98.8% of statements
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/bloques.go:24:		SepararEncabezado	100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/bloques.go:34:		PartirEnBloques		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/bloques.go:60:		MapearEnOrden		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/bloques.go:90:		ParaCadaLinea		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/csv.go:42:		ValidarEncabezado	100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/csv.go:56:		ParsearLinea		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/csv.go:65:		ParsearLineaTolerante	100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/csv.go:78:		parsearLaxo		90.9%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/csv.go:129:		ParsearLineaN		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:79:	Objetivo		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:88:	LimpiarCampos		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:148:	soloDigitos		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:158:	AgregarSilver		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:178:	agregarCampo		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:192:	bytesNecesitaComillas	100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:218:	LimpiarBloque		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/limpieza.go:267:	LimpiarConcurrente	100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:59:		TramoEdad		75.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:69:		NuevoPerfil		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:86:		PerfilarBloque		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:134:		EdadEnAnios		100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:151:		Unir			100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:179:		Cerrar			100.0%
github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest/perfil.go:200:		PerfilarConcurrente	100.0%
total:										(statements)		98.8%
$ go test -race -count=1 ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	3.058s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	10.112s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	20.415s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0, cobertura `internal/ingest`: 98,8 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestLimpiezaConcurrenteIgualALaSecuencial` y `reports/limpieza.json` con `"silver_concurrente_igual_al_secuencial": true`, sha256 `192f2824...`) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Documenta y cuantifica las 11 reglas de limpieza en orden de precedencia, la recuperación de 85 filas con comillas sin escapar en `localidad` y la conservación justificada de las 148 787 filas idénticas. |
| Implementación | 8 | Avance sólido de Parte A (en curso) | Loader concurrente 100 % stdlib con `MapearEnOrden`, acumuladores privados por bloque, equivalencia bit a bit (`sha256`) verificada frente al secuencial, benchmark con IC95 bootstrap (speedup 1,67× en 2 vCPU) y 98,8 % de cobertura en `internal/ingest`. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (en curso) | `cmd/informe` genera `docs/informe/pc3/03-limpieza.md` desde `reports/limpieza.json` y `reports/bench_loader_*.json` sin cifras a mano (`AGENTS.md` §3.6). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos nuevos. Los dos hallazgos menores de la Ronda 1 quedaron resueltos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) | RESUELTO | En `c29d964`, `if reparada { res.Reparadas++ }` se movió después de `if regla != OK` en `LimpiarBloque` (`internal/ingest/limpieza.go:235-237`), y se agregó `TestLimpiarBloqueContadores` (`internal/ingest/limpieza_test.go:152-193`), subiendo `LimpiarBloque` y `Objetivo` al 100 % de cobertura y `internal/ingest` al 98,8 %. |
| H2 (ronda 1) | RESUELTO | En `c29d964`, la rama `else` de `parsearLaxo` (`internal/ingest/csv.go:104-106`) rechaza campos sin entrecomillar que contengan `"`, verificado por `TestParsearLaxoNoAceptaComillasEnCampoSinComillas` (`internal/ingest/csv_test.go:199-204`). |

## Cómo explicarlo en la sustentación

El loader (`cmd/loader`) divide en memoria el CSV de 1 065 274 filas en 113 bloques de ~1 MiB alineados a `\n` y los procesa con el *worker pool* `MapearEnOrden`: cada worker ejecuta `LimpiarBloque` sobre un acumulador privado (`ResultadoBloque`), aplicando un parser CSV en dos fases (`ParsearLineaTolerante`, que recupera 85 filas reales con comillas sin escapar en `localidad`) y 11 reglas deterministas en orden fijo de prioridad. Al terminar los workers, el hilo principal concatena los buffers `Silver` y reduce los conteos y huellas `maphash` en orden estricto de índice de bloque (`0..112`), garantizando que el archivo `data/silver/dengue.csv` (1 065 260 filas válidas, solo 14 descartadas por `edad > 110`) sea idéntico bit a bit (`sha256`) con 1, 2 o más workers y logrando un *speedup* de 1,67× en 2 vCPU.
