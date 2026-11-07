# Veredicto — PR-<n> · Ronda <k>

- **Juez:** Antigravity · modelo: <modelo>
- **Rama revisada:** `<rama>` @ `<commit corto>`
- **Fecha:** AAAA-MM-DD
- **Decisión:** APROBADO | CAMBIOS | RECHAZADO

## Verificación automática

```text
$ gofmt -l .
<vacío = ok>
$ go vet ./...
<salida>
$ go test -race -count=1 ./...
<últimas líneas de la salida>
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ / ❌ |
| `go test -race ./...` | ✅ / ❌ (carreras: N) |
| Equivalencia secuencial = distribuido | ✅ / ❌ / no aplica |
| Latencia API p95 < 100 ms | ✅ / ❌ / no aplica |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | | |
| Implementación | 8 | | |
| Interfaz | 5 | | |
| Informe | 2 | | |
| Video | 2 | | |

## Hallazgos

### H1 — <título corto> · BLOQUEANTE | MAYOR | MENOR

- **Dónde:** `ruta/archivo.go:123`
- **Evidencia:** <salida de comando, cita del enunciado o razonamiento con escenario concreto>
- **Corrección sugerida:** <qué cambiar>

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) | RESUELTO / SE MANTIENE / RETIRADO | |

## Cómo explicarlo en la sustentación

<Dos o tres frases que un integrante podría decir para explicar este PR. Si no se puede,
es un hallazgo MAYOR.>
