# Veredicto — PR-16 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/6-features-gold` @ `48186a9`
- **Fecha:** 2026-10-31
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	2.012s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.981s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	2.184s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.869s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.017s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	10.004s
$ go test -race -count=1 -run "Equivalencia|ShardsReproducen" -v ./...
=== RUN   TestShardsReproducenElGradienteGlobal
--- PASS: TestShardsReproducenElGradienteGlobal (0.03s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.700s
=== RUN   TestEquivalencia
=== RUN   TestEquivalencia/workers=1
=== RUN   TestEquivalencia/workers=2
=== RUN   TestEquivalencia/workers=4
=== RUN   TestEquivalencia/workers=8
=== RUN   TestEquivalencia/workers=16
=== RUN   TestEquivalencia/workers=64
--- PASS: TestEquivalencia (0.16s)
PASS
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	1.718s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; cobertura: `features` 98,3 %, `gold` 84,6 %, `ingest` 97,4 %, `ml` 98,5 %) |
| Equivalencia secuencial = distribuido | ✅ (`TestShardsReproducenElGradienteGlobal` con 1, 2, 3, 5 y 8 shards; `TestEquivalencia` con 1–64 workers; `cmd/entrenar` verifica `"gradiente_pool_igual_al_secuencial": true` en `reports/entrenamiento.json`) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Cierra el diseño metodológico para PC3 (`ADR-0004`): exclusión justificada de `tipo_dx` para evitar fuga de información, estandarización ajustada solo en `train`, split 80/20 determinista por hash `splitmix64` (12,55 % vs. 12,58 % positivos) y corrección de prior de King y Zeng (2001) para responder la pregunta de impacto social en escala real. |
| Implementación | 8 | Avance completo de PC3 (Partes A y B) | Lectura concurrente del silver (`LeerSilverConcurrente` con `MapearEnOrden`, ida y vuelta exacta), transformación a 31 features (`internal/features`), formato binario `CC65GOLD` con `bloque0` y partición en shards de bloques contiguos (`internal/gold`), y entrenamiento real con `ml.Pool` sobre 765 297 filas (`cmd/entrenar`, AUC 0,6221 en test). 100 % stdlib. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | 2 / 2 (PC3 completo) | `cmd/informe` genera `docs/informe/pc3/04-modelo.md` íntegramente desde `reports/split.json`, `reports/entrenamiento.json` y `reports/bench_gradiente_*.json` sin ninguna cifra escrita a mano (`git status` limpio tras `go run ./cmd/informe`). |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes ni mayores. Se registran dos observaciones menores no bloqueantes (opcionales para futuros PR de PC4):

### H1 — Validación defensiva de `TamBloque` en `gold.Decodificar` · MENOR

- **Dónde:** `internal/gold/gold.go:88-118` y `internal/gold/gold.go:128-135`
- **Evidencia:** `Decodificar` valida la magia `"CC65GOLD"`, la versión, la longitud en bytes `tamCab + 8*(n*d+n)` y `ml.NuevoDataset(x, y, d)`, pero no comprueba que `tam > 0` (`uint32` en el offset 24). Si un archivo binario externo o corrupto trae `tam == 0`, `Decodificar` retorna sin error y una llamada posterior a `Particionar` ejecuta `ml.NumBloques(a.N, a.TamBloque)` (`(n + tam - 1) / tam`), provocando un pánico por división entre cero.
- **Corrección sugerida:** No bloquea este PR (pues `cmd/gold` ya valida `*tam >= 1` en `cmd/gold/main.go:59`). Cuando `cmd/node` cargue shards desde disco en PC4, conviene añadir `if tam <= 0 { return Archivo{}, errors.New("gold: tamaño de bloque inválido") }` en `Decodificar`.

### H2 — Benchmark de gradiente en §4.7 sobre el gold real cuando se corra en máquina multihilo · MENOR

- **Dónde:** `docs/informe/pc3/04-modelo.md:130` (`reports/bench_gradiente_ci-2vcpu.json`)
- **Evidencia:** La sección 4.7 del informe muestra el benchmark del gradiente ejecutado previamente sobre el dataset sintético (`sintetico(filas=1000000, features=40, semilla=2026)`). Ahora que `cmd/gold` genera `data/gold/train.bin` (765 297 filas, 31 variables, 187 bloques), `cmd/benchmark` también puede correrse apuntando al gold real o conservar ambos reportes (`reports/bench_gradiente_*.json`) al medir en la PC de un integrante con más núcleos.
- **Corrección sugerida:** Ninguna obligatoria en este PR; `cmd/informe` ya itera sobre todos los `reports/bench_gradiente_*.json` que existan.

## Seguimiento de rondas previas

Primera ronda de revisión para PR-16.

## Cómo explicarlo en la sustentación

Leemos el CSV limpio (`silver`, 1 065 260 filas) en paralelo con `LeerSilverConcurrente`, filtramos los casos desde 2011 (cuando el MINSA adoptó la clasificación OMS con signos de alarma) y separamos 80 % entrenamiento (765 297 filas) y 20 % prueba (190 819 filas) aplicando un hash `splitmix64` sobre el índice global de fila, de modo que el split sea determinista e independiente del número de workers o nodos. Ajustamos el esquema de 31 variables (edad estandarizada y 5 etapas de vida MINSA, sexo, semana epidemiológica en seno/coseno, año estandarizado y 21 columnas de departamento con `< 100` casos agrupados en `OTROS`, excluyendo `tipo_dx` para evitar fuga de información) usando **solo** el conjunto de entrenamiento, y guardamos la matriz en formato binario `CC65GOLD` dividida en 3 shards de bloques contiguos de 4 096 filas (`bloque_inicial` = 0, 62 y 124). Tanto `TestShardsReproducenElGradienteGlobal` como `cmd/entrenar` demuestran que reducir las sumas parciales en orden de índice de bloque produce un gradiente **idéntico bit a bit** al secuencial, entrenando 600 épocas en ~37 s (AUC 0,6221 en prueba) y estimando, tras la corrección de prior de King y Zeng (2001), los mayores riesgos calibrados en Madre de Dios (20,7 %), Loreto (19,8 %), San Martín (19,2 %), Ucayali (19,0 %) y en niños de 0 a 11 años (14,8 %).
