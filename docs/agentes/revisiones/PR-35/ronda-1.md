# Veredicto — PR-35 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/25-protocolo-cluster` @ `5fac285`
- **Fecha:** 2026-11-14
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.183s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	2.044s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	2.232s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.543s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.624s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	2.000s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.616s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	3.865s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.610s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	7.478s
$ go test -race -count=1 -run "Equivalencia|ShardsReproducen|Parciales" -v ./...
=== RUN   TestParcialesDeUnShardRealHacenIdaYVueltaBitABit
    --- PASS: TestParcialesDeUnShardRealHacenIdaYVueltaBitABit/pipe (0.00s)
    --- PASS: TestParcialesDeUnShardRealHacenIdaYVueltaBitABit/tcp (0.01s)
--- PASS: TestParcialesDeUnShardRealHacenIdaYVueltaBitABit (0.02s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.328s
=== RUN   TestShardsReproducenElGradienteGlobal
--- PASS: TestShardsReproducenElGradienteGlobal (0.02s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.532s
=== RUN   TestEquivalencia
    --- PASS: TestEquivalencia/workers=1..64 (0.17s)
=== RUN   TestParcialesDevuelveUnParcialPorBloqueIgualAlSecuencial
--- PASS: TestParcialesDevuelveUnParcialPorBloqueIgualAlSecuencial (0.01s)
=== RUN   TestParcialesReutilizaLosWorkersConDatasetsDistintos
--- PASS: TestParcialesReutilizaLosWorkersConDatasetsDistintos (0.00s)
=== RUN   TestGradienteYParcialesConvivenEnElMismoPool
--- PASS: TestGradienteYParcialesConvivenEnElMismoPool (0.00s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	1.902s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia`, `TestShardsReproducenElGradienteGlobal`, `TestParcialesDevuelveUnParcialPorBloqueIgualAlSecuencial` y `TestParcialesDeUnShardRealHacenIdaYVueltaBitABit` en verde) |
| Latencia API p95 < 100 ms | no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Avanza PC4 (protocolo TCP + `ml.Pool.Parciales`) | Implementa `internal/cluster/protocolo` conforme a `ADR-0005` solo con stdlib (7 mensajes, framing JSON por línea con `json.Decoder`, rechazo previo de `NaN`/`±Inf` con `ErrNoFinito`, plazos con `SetReadDeadline`/`SetWriteDeadline`, límite `MaxMensaje` = 64 MiB y escritura serializada con mutex) y desacopla `ml.Pool` del dataset base mediante `Pool.Parciales(ds, w)` conservando la equivalencia bit a bit. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | Mantiene 2 / 2 | Documenta las 5 decisiones de implementación y la explicación para la sustentación en el PR. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes ni mayores. Las 5 decisiones señaladas en la descripción del PR (etiquetas JSON en `ml.Parcial` para evitar copias por época, `omitempty` en `epoca`/`shard`, `NuevoPool(nil, cfg, n)` para nodos sin dataset base inicial, `MaxMensaje = 64 MiB` más mutex en `Enviar`, e invalidación de la conexión tras error de lectura en `json.Decoder`) son acertadas, cumplen `AGENTS.md` §3 y respetan `ADR-0005`. Se deja una observación menor:

### H1 — Validación de rangos de `ConfigML` y `Parcial.Peso` en `Mensaje.Validar` · MENOR

- **Dónde:** `internal/cluster/protocolo/protocolo.go:125-128` y `internal/cluster/protocolo/protocolo.go:137-149`
- **Evidencia:** En `Validar()`, el caso `Asignacion` comprueba `m.Config == nil`, pero no valida que `m.Config.Lambda >= 0`, `m.Config.PesoPos >= 0` y `m.Config.PesoNeg >= 0`; si un par defectuoso o un test de nodos falsos envía valores negativos en `config`, el nodo entrará en pánico al invocar `ml.Config.normalizada()` (`internal/ml/logistica.go:74-76`) en lugar de rechazar el mensaje con `ErrInvalido`. De forma análoga, en `Gradiente` no se verifica `m.Parciales[k].Peso > 0`, lo que permitiría que parciales con peso cero o negativo lleguen a `ml.Finalizar` (`internal/ml/logistica.go:211`) y dividan por cero.
- **Corrección sugerida:** No bloquea este PR; puede añadirse en un PR posterior de `internal/cluster` (o si se toca `protocolo.go`) comprobando `m.Config.Lambda < 0 || m.Config.PesoPos < 0 || m.Config.PesoNeg < 0` en `Asignacion` y `m.Parciales[k].Peso <= 0` en `Gradiente` dentro de `Mensaje.Validar()`.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-35. |

## Cómo explicarlo en la sustentación

El coordinador y los nodos se comunican por conexiones TCP persistentes enviando un objeto JSON por línea, leído con `json.Decoder` para no truncar mensajes de más de 64 KiB como haría `bufio.Scanner`. En cada época, el nodo reutiliza un único `ml.Pool` sobre cualquiera de sus shards mediante `Pool.Parciales(ds, w)` y envía un `ml.Parcial` por bloque sin reducirlos localmente, porque la suma en punto flotante (`float64`) no es asociativa y solo reduciendo en el coordinador por índice global de bloque se garantiza un resultado idéntico bit a bit al secuencial. Además, `Conexion.Enviar` codifica el mensaje en memoria antes de escribir en el socket, de modo que si aparece un `NaN` o `±Inf` devuelve `ErrNoFinito` sin corromper el flujo TCP y permite al nodo notificar el fallo con un mensaje `error`.
