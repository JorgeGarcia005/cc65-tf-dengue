# Respuesta — PR #39 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-17 · **Base:** `feature/28-api`

Acepto los siete hallazgos. Seis quedan corregidos en esta ronda. Cada uno tiene un test que primero
reprodujo el defecto y ahora pasa. H2 se acepta, pero depende del PR #37, que todavía no se mergea.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `35acda3`: `Consulta.Edad` es `*float64`. Sin edad o con `null` responde 400 con «falta la edad»; con `0` explícito responde 200. Test: `TestPrediccionEdadAusenteONulaDa400` |
| H2 | BLOQUEANTE | ACEPTO | `4f3bbe1` (tras mergear `develop` con el #37): modo servicio `-api :8080` en `cmd/coordinator` y puerto 8080 en Compose. Ver «Actualización» al final |
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

## Actualización (2026-11-19): H2 implementado

El #37 y el #38 ya están en `develop`; la rama los incorpora con un merge (sin reescribir la historia
del PR). Commit `4f3bbe1`:

- `cmd/coordinator -api :8080` pasa a **modo servicio**: carga el gold y `modelo.json`, sirve la API de
  inmediato y entrena en los nodos del clúster (TCP) con cada `POST /api/v1/entrenamientos`. El modelo que
  publica sale del mismo `ml.EntrenarFalible` que el modo de entrenamiento único.
- `entrenadorCluster` implementa `api.Entrenador` sobre `cluster.Coordinador`: revisa `ctx` antes de cada
  época (la cancela `Servidor.Cerrar`), informa el progreso y **rechaza un λ distinto** al del clúster,
  porque la regularización viaja a los nodos al registrarse. Antes de que estén todos los nodos, el
  entrenamiento falla con «el clúster todavía espera a que se registren sus nodos».
- `signal.NotifyContext(SIGTERM, SIGINT)` + `api.Servir`: cierre ordenado, cancela el entrenamiento en
  curso y envía `fin` a los nodos.
- `deploy/docker-compose.yml`: `-api=${API:-}` y `-modelo=/datos/modelo.json`; `deploy/docker-compose.api.yml`
  publica el puerto (`PUERTO_API`, 8080 por defecto) solo en modo servicio. Sin `API` sigue el modo de
  entrenamiento único y no publica puertos (el CI del clúster no cambia).

Tests nuevos en `cmd/coordinator/api_test.go`, con el clúster real por TCP (3 nodos):

| Test | Qué exige |
|---|---|
| `TestServicioEntrenaEnElClusterYPublicaElModelo` | `/healthz` antes de los nodos; 3 nodos en `GET /api/v1/cluster`; `POST` → 202 → `terminado` con 25 épocas; modelo publicado con la versión del entrenamiento y pesos y sesgo **idénticos bit a bit** al secuencial; `/docs` y `/docs/openapi.yaml` 200; tras cancelar (SIGTERM) termina sin error y los nodos reciben `fin` |
| `TestServicioRechazaUnLambdaDistintoAlDelCluster` | `fallido` con «lambda=0.001» |
| `TestServicioSinNodosRegistradosNoEntrena` | `fallido` sin nodos; las predicciones siguen con el modelo inicial |
| `TestServicioRechazaModeloYShardsIncompatibles` | error claro y el listener HTTP queda cerrado |

Mutantes: sin la comprobación de λ falla el segundo test; sin la de nodos registrados, el tercero.

```text
$ gofmt -l . ; go vet ./...        → vacío
$ go test -race -count=1 ./...     → todos ok
$ go test -race -count=3 ./cmd/coordinator
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	33.805s
```

### Verificación desde el anfitrión (Docker Desktop, PC de Jorge, gold real)

`bash scripts/evidencia-api.sh` en `8bad2d1`: Windows/amd64, 12 CPU lógicas, Docker 29.9.3, gold real
(765 297 filas de train, 31 variables, 3 shards). El puerto del anfitrión es 18080 porque el 8080 estaba
ocupado en esa PC (`PUERTO_API`). Extractos:

```text
== 2. Swagger UI y contrato desde el anfitrión
GET /docs → 200 (text/html; charset=utf-8)
GET /docs/openapi.yaml → 200 (8243 bytes)

== 3. Estado del clúster
{"epocas":0,"perdida":0,"nodos":[{"nombre":"372ac67cf223","remoto":"172.18.0.5:51614","workers":12,"sano":true,"carga":0}, … 3 nodos …],"reasignaciones":0}

== 4. Entrenamiento distribuido vía API
HTTP/1.1 202 Accepted
Location: /api/v1/entrenamientos/ent-1
{"id":"ent-1","estado":"en_curso","epocas":600,"tasa":2,"lambda":0.0001,"epoca":0,"perdida":0,"inicio":"2026-11-20T04:16:17Z"}
GET /api/v1/entrenamientos/ent-1
{"id":"ent-1","estado":"terminado","epocas":600,"tasa":2,"lambda":0.0001,"epoca":600,"perdida":0.6700654760947148,"inicio":"2026-11-20T04:16:17Z","fin":"2026-11-20T04:16:26Z","version_modelo":"9bb1594a398f"}

== 5. Predicción con el modelo publicado
{"riesgo":0.1734997592400519,"probabilidad_ponderada":0.5940450096064652,"clase":1,"etiqueta":"con signos de alarma o grave","umbral":0.5,"departamento_modelo":"LORETO","version_modelo":"9bb1594a398f"}
(0.002655 s)

== 6. Cierre con SIGTERM: docker compose stop coordinator
cc65-tf-coordinator-1   Exited (0)
cc65-tf-node-1          Exited (0)   (node-2 y node-3 igual)
coordinator-1  | … 3 nodos registrados: la API acepta entrenamientos
coordinator-1  | … entrenamiento vía API: 600 épocas en 9.6 s, pérdida 0.693147 → 0.670065
coordinator-1  | … api: entrenamiento ent-1 terminado en 9.627s; modelo publicado 9bb1594a398f
coordinator-1  | … coordinador detenido
```

La pérdida final (0.670065) es la misma del entrenamiento del clúster en modo único que `cmd/entrenar
-comparar` declaró idéntico bit a bit al secuencial (evidencia del #27). El SIGTERM cierra el coordinador
con código 0 y los nodos terminan con `fin` (código 0). La latencia p95 y Postman se miden en el #30.
