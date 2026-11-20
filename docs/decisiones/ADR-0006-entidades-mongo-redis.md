# ADR-0006 — Entidades en MongoDB y caché de predicciones en Redis

- **Estado:** Propuesto (17/11/2026) · Issue #29 · PC4
- **Relacionados:** ADR-0003 (drivers de MongoDB y Redis permitidos), ADR-0005 (estado del clúster),
  PR #39 (`internal/api`)

## Contexto

El enunciado de la PC4 pide «implementación de bases de datos (diseño de entidades, registros)»
con MongoDB como almacén persistente y Redis como caché. La API (#28) guardaba el modelo y el
historial de entrenamientos solo en memoria, así que un reinicio del coordinador los perdía. Además,
cada predicción se recalcula aunque la misma consulta se repita, y la latencia exigida es
p95 < 100 ms (AGENTS.md §3.5).

Restricciones: los únicos terceros son los drivers oficiales
(`go.mongodb.org/mongo-driver/v2`, `github.com/redis/go-redis/v9`), y la API debe poder probarse
sin contenedores.

## Decisiones

| Tema | Decisión |
|---|---|
| Acceso | `internal/almacen` define las interfaces `Almacen` (persistente) y `Cache`. Hay tres implementaciones: `almacen.Memoria` (tests y modo sin BD), `almacen/mongo` y `almacen/redis`. La API solo conoce las interfaces |
| Modelo publicado | Se guarda en Mongo al publicarse. Al arrancar, el coordinador toma el vigente de Mongo y, si no hay ninguno, el de `data/gold/modelo.json` |
| Entrenamientos | Se escriben en Mongo (upsert por `_id`) al iniciar, cada 10 épocas y al terminar. El estado en vivo sigue en memoria; Mongo responde por los entrenamientos de ejecuciones anteriores |
| Registro de predicciones | Se escribe en segundo plano por un canal con búfer (capacidad 1024). Si el búfer está lleno, el registro se descarta y se cuenta: **la latencia de la predicción no depende de Mongo** |
| Caché | Redis, patrón *cache-aside*. La clave incluye la versión del modelo, así que publicar un modelo nuevo invalida la caché sin borrar nada; las claves viejas expiran por TTL |
| Fallos de Redis | Un error de la caché no falla la predicción: se calcula y se registra el error |

### Colecciones de MongoDB (base `cc65_dengue`)

| Colección | `_id` | Campos | Índices |
|---|---|---|---|
| `modelos` | `version` (12 hex del sha256 del modelo) | `pesos` [D+1], `esquema` (departamentos, medias, desviaciones, nombres), `umbral`, `ajuste_prior`, `generado`, `entrenamiento_id`, `publicado` | `{publicado: -1}`, para obtener el vigente |
| `entrenamientos` | `id` (`ent-<fecha>-<n>`) | `estado` (`en_curso`/`terminado`/`fallido`), `epocas`, `tasa`, `lambda`, `epoca`, `perdida`, `historia` [pérdida por época], `nodos` [nombre, remoto, workers, sano, carga, motivo], `reasignaciones`, `inicio`, `fin`, `version_modelo`, `error` | `{inicio: -1}`, `{estado: 1}` |
| `predicciones` | ObjectId | `fecha`, `version_modelo`, `consulta` {departamento, ano, semana, edad_anios, sexo}, `riesgo`, `clase`, `desde_cache`, `microsegundos` | `{fecha: -1}`, `{version_modelo: 1, "consulta.departamento": 1}` y TTL de 90 días sobre `fecha` |
| `usuarios` (TB2) | `correo` | `hash` (bcrypt o PBKDF2), `rol` (`admin`/`analista`), `creado` | Único por `_id`. Se implementa con JWT en TB2 |

Las predicciones se agrupan por `version_modelo` y `consulta.departamento` para que el panel admin de
TB2 muestre la demanda por departamento. El TTL evita que la colección crezca sin límite.

### Claves de Redis

| Clave | Valor | TTL |
|---|---|---|
| `pred:<version_modelo>:<departamento>:<ano>:<semana>:<edad_anios>:<sexo>` | JSON de la predicción | 24 h |

La consulta se normaliza antes de armar la clave (departamento en mayúsculas, edad convertida a
años, sexo en mayúscula). Así, `"loreto"` y `"LORETO"`, o `12 M` y `1 A`, comparten la misma entrada.

## Consecuencias

- La API se prueba completa con `almacen.Memoria` y una caché en memoria con TTL. Los adaptadores
  reales tienen tests de integración con la etiqueta `integracion`
  (`go test -race -tags integracion ./internal/almacen/...`), que corren contra
  `deploy/docker-compose.bd.yml`.
- `mongo` y `redis` son servicios de `deploy/docker-compose.yml`, en la red interna del clúster.
  `deploy/docker-compose.api.yml` (modo servicio) le pasa al coordinador `MONGO_URI` y `REDIS_ADDR`
  y espera a que ambos estén sanos. `deploy/docker-compose.bd.yml` queda solo para los tests de
  integración locales, porque publica los puertos en `127.0.0.1`.
- `go.mod` agrega solo los dos drivers y sus dependencias transitivas.
