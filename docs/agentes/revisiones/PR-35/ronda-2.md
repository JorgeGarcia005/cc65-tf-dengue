# Veredicto — PR-35 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini 4 Argon (High)
- **Rama revisada:** `feature/25-protocolo-cluster` @ `5d9853e`
- **Fecha:** 2026-11-14
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	2.003s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.961s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	2.073s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.729s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.572s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	2.140s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.868s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.562s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.469s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	10.885s
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
| Implementación | 8 | Avanza PC4 (protocolo TCP + `ml.Pool.Parciales`) | Cierra también el hallazgo menor `H1` validando rangos no negativos en `ConfigML` y `Peso > 0` en cada `ml.Parcial` dentro de `Mensaje.Validar`. |
| Interfaz | 5 | no aplica | No modifica la SPA ni la API HTTP. |
| Informe | 2 | Mantiene 2 / 2 | Trazabilidad completa de ronda 1 y respuesta 1 registrada en `docs/agentes/revisiones/PR-35/`. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — Validación de rangos de `ConfigML` y `Parcial.Peso` en `Mensaje.Validar` | RESUELTO | En `11dc87c` (`internal/cluster/protocolo/protocolo.go:131-134, 158-160`), `Mensaje.Validar` rechaza con `ErrInvalido` valores negativos en `ConfigML` (`Lambda`, `PesoPos`, `PesoNeg`) y parciales con `Peso <= 0`, con 4 casos nuevos en `TestMensajeDeTipoDesconocidoOIncompletoSeRechaza` (`internal/cluster/protocolo/protocolo_test.go:358-361`). |

## Cómo explicarlo en la sustentación

El coordinador y los nodos se comunican por conexiones TCP persistentes enviando un objeto JSON por línea, leído con `json.Decoder` para no truncar mensajes de más de 64 KiB como haría `bufio.Scanner`. En cada época, el nodo reutiliza un único `ml.Pool` sobre cualquiera de sus shards mediante `Pool.Parciales(ds, w)` y envía un `ml.Parcial` por bloque sin reducirlos localmente, porque la suma en punto flotante (`float64`) no es asociativa y solo reduciendo en el coordinador por índice global de bloque se garantiza un resultado idéntico bit a bit al secuencial. Además, `Conexion.Enviar` codifica el mensaje en memoria antes de escribir en el socket, de modo que si aparece un `NaN` o `±Inf` devuelve `ErrNoFinito` sin corromper el flujo TCP y permite al nodo notificar el fallo con un mensaje `error`.
