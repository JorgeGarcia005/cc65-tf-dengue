# data/

No se versiona (ver `.gitignore`), salvo este README y `MANIFEST.sha256`.

| Capa | Contenido | Quién la genera |
|---|---|---|
| `bronze/` | CSV original del MINSA, sin tocar | `make data` |
| `silver/` | Registros limpios y validados | `cmd/loader` |
| `gold/` | Matriz de features, split train/test, un shard por nodo, `esquema.json` y `modelo.json` | `cmd/gold` y `cmd/entrenar` |

`MANIFEST.sha256` fija el archivo exacto usado en los resultados del informe.

En Docker, `docker compose -f deploy/docker-compose.yml --profile datos up entrenar` corre
`loader`, `gold` y `entrenar` en orden y escribe aquí lo mismo (ver el README de la raíz).
