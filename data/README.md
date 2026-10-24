# data/

No se versiona (ver `.gitignore`), salvo este README y `MANIFEST.sha256`.

| Capa | Contenido | Quién la genera |
|---|---|---|
| `bronze/` | CSV original del MINSA, sin tocar | `make data` |
| `silver/` | Registros limpios y validados | `cmd/loader` |
| `gold/` | Matriz de features + shards por nodo | `cmd/loader` |

`MANIFEST.sha256` fija el archivo exacto usado en los resultados del informe.
