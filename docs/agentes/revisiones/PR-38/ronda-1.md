# Veredicto — PR-38 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/23-revalidar-hardware-jose` @ `b504ecb`
- **Fecha:** 2026-11-19
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.063s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.011s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.014s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.547s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.546s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.019s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.043s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	3.770s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.011s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	7.700s

$ go run ./cmd/revalidar -ref reports -nuevo reports
perfil_bronze.json   IGUAL (salvo generado, workers, speedup y seg_*)
limpieza.json        IGUAL (salvo generado, workers, speedup y seg_*)
split.json           IGUAL (salvo generado, workers, speedup y seg_*)
entrenamiento.json   IGUAL (salvo generado, workers, speedup y seg_*)

$ tmp=$(mktemp -d) && git show 736f4dd:reports/entrenamiento.json > "$tmp/entrenamiento.json" \
  && go run ./cmd/revalidar -ref "$tmp" -nuevo reports entrenamiento.json; rm -rf "$tmp"
entrenamiento.json   DIFIERE en 1 campos:
    riesgo_por_semana_epidemiologica: solo en el nuevo

$ bash -n scripts/revalidar.sh && bash scripts/check-skills-sync.sh
skills: ok
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; CI de GitHub en verde: `Ecosistema de agentes` y `Go (gofmt, vet, test -race)`) |
| Equivalencia secuencial = distribuido | no aplica en el código nuevo (no modifica `internal/ml` ni `internal/cluster`; valida reproducibilidad bit a bit sobre `reports/*.json` e independiza el resultado del hardware y del número de workers) |
| Latencia API p95 < 100 ms | no aplica |

## Verificación contra el Issue #23 y restricciones de `AGENTS.md`

- **Criterios de aceptación de #23:**
  - Comentario con `git diff --stat reports/`, verificación de `sha256sum -c data/MANIFEST.sha256`, comparación campo por campo con `cmd/revalidar` y prueba `-workers 1` vs `-workers 16` publicado en `#23` (`#issuecomment-6962404605`) por José (`JJva-06`, Ryzen 7 5800X, 16 CPU lógicas, `windows/amd64`, `go1.26.10`).
  - Se comprobó de forma independiente con `cmd/revalidar` que el `reports/entrenamiento.json` actual (generado en la PC de José con 16 workers en #20) coincide bit a bit en pesos, sesgo, pérdida por época, métricas y riesgo por departamento y etapa con el `reports/entrenamiento.json` de #16 (`736f4dd`, generado en otra máquina con 2 workers), difiriendo únicamente en el campo `riesgo_por_semana_epidemiologica` agregado después.
  - No se sobrescribió ningún archivo de `reports/` ni se versionó nada de `data/` (el diff agrega solo 3 archivos, 285 líneas: `cmd/revalidar/main.go`, `cmd/revalidar/main_test.go`, `scripts/revalidar.sh`).
- **Auditoría de campos ignorados (`cmd/revalidar/main.go:74-80`):** inspeccionando todas las claves de los cuatro reportes reales en `reports/`, `ignorado()` filtra exclusivamente campos de tiempo, conteo de workers y speedup (`perfil_bronze.json`: `generado`, `seg_concurrente`, `seg_secuencial`, `speedup`, `workers`; `limpieza.json`: `generado`, `seg_lectura_disco`, `seg_limpieza_concurrente`, `seg_limpieza_secuencial`, `speedup`, `workers`; `split.json`: `generado`, `seg_lectura_silver`; `entrenamiento.json`: `generado`, `seg_entrenamiento`, `seg_por_epoca`, `workers`), sin falsos positivos en campos anidados.
- **Restricciones (`AGENTS.md` §3 y §4):** solo biblioteca estándar (`go.mod` intacto), comparación IEEE-754 bit a bit mediante `math.Float64bits`, PR acotado (< 400 líneas), rama `feature/23-revalidar-hardware-jose` → `develop`, commit convencional en español con autoría de José y declaración de uso de IA completa.

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni arquitectura. |
| Implementación | 8 | Refuerza PC3 / avanza PC4 | Agrega verificación automatizada y reutilizable (`cmd/revalidar` + `scripts/revalidar.sh`) de reproducibilidad bit a bit entre máquinas y entre 1 y 16 workers (`AGENTS.md` §3.3), con pruebas unitarias en verde y solo stdlib. |
| Interfaz | 5 | no aplica | No modifica API ni SPA. |
| Informe | 2 | Mantiene 2 / 2 | Deja evidencia reproducible y trazable (código + comentario en #23) de que ninguna cifra de `reports/*.json` depende del hardware donde se ejecutó (`AGENTS.md` §3.6), y suma commits de José en PC4. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes ni mayores. Se deja una observación menor (no bloquea el merge):

### H1 — Restauración de `data/gold/modelo.json` fuera del `trap EXIT` · MENOR

- **Dónde:** `scripts/revalidar.sh:16` y `scripts/revalidar.sh:46`
- **Evidencia:** En `scripts/revalidar.sh:16`, el `trap ... EXIT` restaura `reports/` (`git restore --source=HEAD -- reports/`) y elimina `$tmp`, mientras que la restauración de `data/gold/modelo.json` (`[ -f "$tmp/modelo.json" ] && cp "$tmp/modelo.json" data/gold/modelo.json`) ocurre recién en la línea 46. Si el script se interrumpe (por ejemplo, `Ctrl-C` o fallo en el paso 5 después de que `go run ./cmd/entrenar` en la línea 32 ya reescribió `data/gold/modelo.json`), `reports/` se restaura pero `data/gold/modelo.json` queda con la copia regenerada. (Adicionalmente, el archivo quedó con modo `100644` en git frente a `100755` de `scripts/check-skills-sync.sh` y `scripts/descargar-datos.sh`, aunque su uso documentado es `bash scripts/revalidar.sh`.)
- **Corrección sugerida:** No requiere bloquear este PR; si se vuelve a tocar el script, mover la restauración de `$tmp/modelo.json` dentro del `trap` de la línea 16 antes de `rm -rf "$tmp"` y opcionalmente aplicar `git update-index --chmod=+x scripts/revalidar.sh`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR #38. |

## Cómo explicarlo en la sustentación

Para demostrar en la práctica el determinismo exigido por el proyecto (`AGENTS.md` §3.3 y §3.6), reejecutamos todo el pipeline (`perfil` → `loader` → `gold` → `entrenar`) en la PC de José (16 CPU lógicas, Windows) y comparamos cada campo de los reportes JSON contra la referencia versionada con `cmd/revalidar`, que ignora únicamente metadatos de ejecución (`generado`, `workers`, `speedup` y `seg_*`) y exige igualdad bit a bit (`math.Float64bits`) en todos los números flotantes. Tanto la comparación entre distintos equipos como la corrida con `-workers 1` frente a `-workers 16` arrojaron pesos, sesgo, pérdida por época, métricas y riesgo por grupo idénticos bit a bit, confirmando que fijar los límites de bloque y reducir en orden de índice hace el resultado independiente del hardware y del grado de paralelismo.
