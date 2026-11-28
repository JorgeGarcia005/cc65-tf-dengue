# Respuesta a la ronda 1 — PR #94

Implementador: Claude Code (Claude Opus 5.5). Revisado por Jorge.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MAYOR | ACEPTO | `6735952` — ancla `x-imagen-datos` (`image` + `build`) en `loader`, `gold` y `entrenar`; `--build` en el primer `run` del README; `mkdir -p data reports` en el README (Linux) y en `make datos-docker`; el CI borra `cc65-tf-datos` y corre el comando del README tal cual |
| H2 | MENOR | ACEPTO | `62a8618` — `NuevoEsquema` rechaza medias `NaN` o `±Inf` (test con los tres casos); el encabezado de `deploy/Dockerfile` apunta a `deploy/Dockerfile.datos` para la Parte A |

## H1 — verificación

```text
$ docker compose -f deploy/docker-compose.yml --profile datos config --format json | jq '.services | {loader, gold, entrenar} | map_values({image, dockerfile: .build.dockerfile})'
loader   cc65-tf-datos deploy/Dockerfile.datos
gold     cc65-tf-datos deploy/Dockerfile.datos
entrenar cc65-tf-datos deploy/Dockerfile.datos
```

El paso del CI «Parte A en Docker (perfil datos) sin el CSV» ahora parte de `docker image rm -f cc65-tf-datos` y usa `run --rm --no-deps --build gold …`, el mismo comando del README: si un servicio vuelve a quedarse sin `build:`, el CI falla al intentar el `pull`.

## Verificación

- `gofmt -l .` vacío; `go test ./internal/features ./cmd/gold` en verde; `go test ./cmd/informe`.
- `docker compose config` válido para el clúster, el modo servicio y el perfil `datos`.
