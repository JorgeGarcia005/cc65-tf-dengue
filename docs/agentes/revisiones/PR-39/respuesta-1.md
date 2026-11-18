# Respuesta — PR #39 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-17 · **Base:** `feature/28-api`

Acepto los siete hallazgos. Seis quedan corregidos en esta ronda. Cada uno tiene un test que primero
reprodujo el defecto y ahora pasa. H2 se acepta, pero depende del PR #37, que todavía no se mergea.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `35acda3`: `Consulta.Edad` es `*float64`. Sin edad o con `null` responde 400 con «falta la edad»; con `0` explícito responde 200. Test: `TestPrediccionEdadAusenteONulaDa400` |
| H2 | BLOQUEANTE | ACEPTO (depende del #37) | Pendiente. Ver abajo |
| H3 | MAYOR | ACEPTO | `e38a347`: `CargarModelo` exige `desv_edad` y `desv_ano` positivas y medias, pesos y ajuste finitos. `responder` codifica antes de escribir el código y, si falla, devuelve 500 con mensaje. Tests: `TestPrediccionModeloConDesviacionNulaNoSeCarga` y `TestPrediccionRespuestaNoCodificableDa500` |
| H4 | MAYOR | ACEPTO | `f3a2a12`: el cuerpo de `POST /api/v1/entrenamientos` es opcional; sin cuerpo responde 202 con los valores por defecto documentados. Test: `TestRutasEntrenamientoSinCuerpoUsaDefectos` |
| H5 | MAYOR | ACEPTO | `f3a2a12`: después del objeto se intenta otra lectura y se exige `io.EOF`, en lugar de usar `Decoder.More`. Rechaza `}`, `{}`, `[]` y texto sobrante; acepta espacio en blanco al final. Test: `TestPrediccionContenidoSobranteDa400` |
| H6 | MAYOR | ACEPTO | `9309822`: las descripciones en línea van entre comillas. `TestOpenAPIMapasEnLineaBienFormados` exige que cada elemento de un mapa en línea sea `clave: valor` (antes fallaba en `openapi.yaml:149`). Además, el nuevo job `openapi` del CI valida el documento como OpenAPI 3 con `@redocly/cli@1.25.0 lint --extends=minimal` |
| H7 | MAYOR | ACEPTO | `fbd3736`: `GET /api/v1/cluster` devuelve `"nodos":[]` antes del primer registro. Test: `TestRutasClusterVacioDevuelveArregloDeNodos` |

## H2: integración en el coordinador y en Compose

Coincido en que el criterio «Swagger UI servido en `/docs` dentro de Compose» no se cumple en este
checkout. `cmd/coordinator`, `cmd/node`, `deploy/Dockerfile` y `deploy/docker-compose.yml` se crean en el
PR #37 (issue #27), que sigue abierto y sin veredicto:

```text
$ gh pr view 37 --json state --jq .state
OPEN
$ git ls-tree -r --name-only origin/develop -- cmd/coordinator deploy
(vacío)
```

Crear esos archivos aquí duplicaría el trabajo del #37 y provocaría conflictos de merge. Propongo
seguir la corrección que sugiere el propio veredicto: «integrar la API en el coordinador y en Compose
al incorporar la dependencia». Cuando el #37 entre a `develop`, haré en **este mismo PR**:

1. Rebase sobre `develop`.
2. En `cmd/coordinator`: implementar `api.Entrenador` sobre `cluster.Coordinador` y `ml.EntrenarFalible`,
   servir la API con `api.Servir` y `signal.NotifyContext(ctx, SIGTERM, SIGINT)`, y exponer el puerto
   8080 en Compose.
3. Verificar desde el host `/docs`, `/docs/openapi.yaml`, `POST /api/v1/entrenamientos` +
   `GET /api/v1/entrenamientos/{id}` y el cierre con `docker compose stop coordinator`, y pegar las
   salidas en `respuesta-2.md`.

Por eso **el PR no debe mergearse antes de esa ronda**, aunque los demás hallazgos queden resueltos.

## Sobre la falla intermitente de `TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano`

El veredicto la registra en la primera ejecución dentro del sandbox y luego la ve pasar. Este PR no
toca `internal/cluster` (`git diff --stat origin/develop -- internal/cluster` vacío). Si vuelve a
aparecer fuera del sandbox, corresponde abrir un issue `fix/` aparte con la salida.

## Verificación

```text
$ gofmt -l . (archivos Go del repo)   → vacío
$ go vet ./...                          → sin salida
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.870s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.805s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.804s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	3.702s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	6.296s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.194s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.823s
(resto de paquetes: ok; ninguna carrera detectada)
```

El job `openapi` no pudo correr localmente porque no hay Node ni Docker en la máquina de José. Su
resultado queda en el CI de este push.
