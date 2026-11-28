# Respuesta — PR-96 · Ronda 1

- **Implementador:** Claude Code (Claude Opus 5.5)
- **Veredicto respondido:** `ronda-1.md` (CAMBIOS)

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MAYOR | ACEPTO | `b6ce199` — `(*documento).textoPlano` (sin `<!--v:…-->`, referencias resueltas sin repetir avisos, enlaces y `<url>` reducidos a su texto, sin marcas) usado en los anchos de columna, en los títulos de los índices y en `descr` de las imágenes; `partirFila` no parte por `\|` dentro de `<!-- … -->`. Test `TestTextoPlanoEnCeldasEIndices`: título con valor y `@fig:` en el índice, pipeline `\| cuatro` en una celda, anchos iguales a los del texto visible. |
| H2 | MAYOR | ACEPTO | `661f821` — `TestInformesSinAvisos` convierte `docs/informe/tb2` y `docs/informe/pc4` con la plantilla mínima y exige cero avisos (comprobado: con `@fig:typo` agregado falla con «referencia a una etiqueta que no existe»). `cmd/word` termina con error si hay avisos, salvo `-permitir-avisos`. |
| H3.1 | MENOR | ACEPTO | `ebc8dae` — `.ps1` (powershell) y `.env`/`.env.*` con comentario de ruta; BOM y CRLF quitados en `regionArchivo`. Test `TestRegionArchivoLenguajes`. |
| H3.2 | MENOR | ACEPTO | `b6ce199` — sin párrafo vacío entre `</w:tbl>` y la `Nota.` que la sigue (comprobado en `TestTextoPlanoEnCeldasEIndices`). |
| H3.3 | MENOR | ACEPTO | `ebc8dae` — la nota de `@fig:arquitectura` distingue conexiones de red (con puerto, en el sentido de la petición; la TCP de doble punta la abre el nodo, `cmd/node/main.go:77`) de lectura/escritura de archivos y lectura del volumen. |

## Verificación

```text
$ gofmt -l . && go vet ./cmd/... ./internal/...      # vacío (drivers con stubs en este entorno)
$ go test -race ./cmd/word ./cmd/informe
ok  	…/cmd/word
ok  	…/cmd/informe
$ go run ./cmd/informe && git status --short            # sin cambios
```
