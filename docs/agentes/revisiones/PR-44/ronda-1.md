# Veredicto — PR-44 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/33-informe-pc4` @ `77e3bb0`
- **Fecha:** 2026-11-21
- **Decisión:** CAMBIOS

El PR genera de forma reproducible las 16 secciones del informe de la PC4 (`docs/informe/pc4/`), cierra las dos observaciones del profesor sobre la PC3 (pregunta que resuelve el problema repetida literalmente en Resumen, Problema, Objetivos y Conclusiones; y 9 funciones del algoritmo distribuido copiadas del fuente Go mediante `cmd/informe` y exigidas por `TestFragmentosSalenDelFuente`), integra las mediciones reales del Ryzen 7 5800X (clúster, API, Spin y `entrenamiento_cluster.json` verificado bit a bit con `cmd/entrenar -comparar`), y convierte a OOXML válido con 0 avisos en `cmd/word`. Toda la verificación técnica (`gofmt`, `go vet`, `go test -race ./...`, equivalencia secuencial = distribuido, idempotencia de `cmd/informe` y los 6 jobs del CI) está en verde. Sin embargo, se emiten **3 hallazgos mayores** y **2 menores** de exactitud documental y cumplimiento APA 7 antes de aprobar el merge: las secciones 4.2 y 14 contradicen `deploy/Dockerfile` ya fusionado en `develop` (PR #46), dos comandos de Docker Compose en 4.2 y 12.2 no producen el comportamiento que describen (`--scale node=N` y ausencia de `VERIFICAR=true`), y cuatro entradas de `15-referencias.md` carecen de cita en el cuerpo del informe.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	2.366s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	6.463s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.551s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.030s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.279s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.131s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.418s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.020s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.054s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.024s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.054s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.106s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	8.206s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.064s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.044s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.089s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	7.553s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.027s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	14.022s

$ go test -race -count=1 -v ./cmd/informe
=== RUN   TestEtiquetaBench
--- PASS: TestEtiquetaBench (0.00s)
=== RUN   TestPC4BenchmarksDeUnaMaquina
--- PASS: TestPC4BenchmarksDeUnaMaquina (0.00s)
=== RUN   TestFragmentosSalenDelFuente
--- PASS: TestFragmentosSalenDelFuente (0.09s)
=== RUN   TestExtraerFuncionYMetodo
--- PASS: TestExtraerFuncionYMetodo (0.02s)
=== RUN   TestRegionDesconocidaDaError
--- PASS: TestRegionDesconocidaDaError (0.00s)
PASS

$ go test -race -count=1 -run Equivalencia -v ./...
PASS (TestEquivalencia workers=1..64; TestEquivalenciaClusterSecuencialBitABit nodos=1,2,3,5; TestEquivalenciaConMenosNodosQueShards)

$ go run ./cmd/informe && git diff --exit-code docs/informe
(0 diferencias = idempotente)

$ go run ./cmd/entrenar -comparar reports/entrenamiento_cluster.json
reports/entrenamiento.json y reports/entrenamiento_cluster.json: mismos hiperparámetros, pesos, sesgo e historia de pérdida, idénticos bit a bit

$ go test -race -count=1 -overlay <audit_pc4_word> -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 24 figuras/capturas totales, 9 SVGs embebidos, 16 enlaces, 0 avisos

$ gh pr checks 44
Docker (imagen, clúster y caída de un nodo)	pass	1m48s
Ecosistema de agentes                       	pass	11s
Go (gofmt, vet, test -race)                 	pass	57s
MongoDB y Redis (tests de integración)      	pass	39s
OpenAPI 3 válido                            	pass	20s
Spin (modelo del protocolo y mutantes)      	pass	2m2s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; suite completa local + CI verde en run `44935849521`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` + `cmd/entrenar -comparar reports/entrenamiento_cluster.json` bit a bit, aun con 1 nodo caído y 1 reasignación) |
| Latencia API p95 < 100 ms | ✅ (no modifica `internal/api`; `reports/bench_api_deez.json` registra p95 = 2,21 ms fría y 2,06 ms caliente con 0 errores en 16 clientes) |
| Reproducibilidad del informe y Word XML | ✅ (`cmd/informe` idempotente, `TestFragmentosSalenDelFuente` en verde, 16 secciones PC4 convertidas a OOXML válido con 0 avisos) |

## Verificación contra los criterios del issue #33

| Criterio (#33) | Estado | Evidencia |
|---|---|---|
| Obs. 1: «Pregunta que resuelve el problema» explícita en 2.2, retomada con las mismas palabras en Resumen, Objetivos y Conclusiones, y en la tabla enunciado → sección | ✅ | Subtítulo en `02-problema.md:31`, cadena exacta (351 caracteres) verificada en `01-resumen.md:9`, `02-problema.md:36`, `03-objetivos.md:5` y `13-conclusiones.md:5`; mapeada en `04-arquitectura.md:54`. |
| Obs. 2: código fuente del algoritmo distribuido copiado del repositorio + pseudocódigo numerado «Algoritmo n» | ✅ | Algoritmos 1 y 2 en `08-distribuido.md:64-86`; 9 regiones `<!-- generado:fuente:... -->` verificadas por `TestFragmentosSalenDelFuente`. |
| Secciones nuevas (clúster, topología, protocolo + secuencia, fallos, Spin, rendimiento, API, Mongo/Redis) | ⚠️ ajustes en 4.2, 10, 14 y `epoca.svg` | Todas las secciones existen y son consistentes con la arquitectura, pero presentan discrepancias puntuales con `develop` y el código (**H1**, **H2**, **H4**, **H5**). |
| Evidencias con capturas reales (`grep -c 'PENDIENTE'` = 0) | ✅ (con nota operativa) | `grep -c 'PENDIENTE' docs/informe/pc4/*.md` = 0; `12-evidencias.md` deja 15 marcos `<!-- captura: ... -->` que se completan en el `.docx` (mismo flujo que PC3), salvo el comando de `12-evidencias.md:21` (**H2**). |
| Ninguna cifra a mano (`go run ./cmd/informe && git diff --exit-code docs/informe`) | ✅ | Salida limpia con 0 diferencias; `TestPC4BenchmarksDeUnaMaquina` y `TestFragmentosSalenDelFuente` en verde. |
| Trabajos relacionados (2.6) y bibliografía APA 7 con DOI/documento oficial | ⚠️ parcial | `02-problema.md:84-97` y `15-referencias.md` tienen DOIs/URLs oficiales y orden alfabético, pero 4 referencias de `15-referencias.md` no están citadas en el texto (**H3**). |
| Tabla «Correcciones a la PC3» | ✅ | `16-anexos.md:3-11`, referenciada en `04-arquitectura.md:70`. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Cierra la observación 1 de PC3 con subtítulo propio, tabla de trazabilidad en `02-problema.md:40-46`, repetición exacta en Resumen/Objetivos/Conclusiones y sección de trabajos relacionados (`02-problema.md:84-97`). |
| Implementación | 8 | Avance completo de PC4 (con ajustes en doc de despliegue H1–H2) | Integra en `cmd/benchmark` el seguimiento del PR #42 (`-recorte-cluster` validado, `nodosMedidos` y nota de Karp-Flatt), arregla SIGPIPE en `scripts/evidencia-bd.sh` y versiona las mediciones reales del Ryzen 7 5800X (`bench_cluster_deez_gold.json`, `bench_api_deez.json`, `entrenamiento_cluster.json`, `spin.json`). |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | Parcial (pendiente H1–H3) | Excelente automatización (`cmd/informe` + `TestFragmentosSalenDelFuente` + corrección de `"máx. "` en `figuras.go`), pero hoy contradice `deploy/Dockerfile` en `develop` (H1), imprime comandos de Compose inexactos (H2) y deja 4 referencias sin citar en APA 7 (H3). |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

### H1 — Las secciones 4.2 y 14 contradicen el `deploy/Dockerfile` ya fusionado en `develop` (PR #46) · MAYOR

- **Dónde:** `docs/informe/pc4/04-arquitectura.md:38-39` y `docs/informe/pc4/14-recomendaciones.md:19-21`
- **Evidencia:**
  1. `feature/33-informe-pc4` se ramificó antes de que entrara a `develop` el PR #46 (commit `9a01833`, fusionado en `80bb4f8`), que corrigió el fallo de CI por superar 40 MB compilando **únicamente** `./cmd/coordinator` y `./cmd/node` (`origin/develop:deploy/Dockerfile:24`, `deploy/README.md:9-11`), reduciendo la imagen a **15,44 MB** en CI (`25,0 MB` en local, ver `origin/develop:docs/agentes/revisiones/PR-46/ronda-1.md`).
  2. Pese a ello, `docs/informe/pc4/04-arquitectura.md:38-39` todavía afirma:
     > «Todo el backend se construye en **una sola imagen** Docker (`deploy/Dockerfile`: compilación estática de los binarios de `cmd/` sobre `distroless/static`, usuario sin privilegios)»
  3. Y `docs/informe/pc4/14-recomendaciones.md:19-21` recomienda como trabajo futuro exactamente lo que el PR #46 ya hizo:
     > «- **Vigilar el tamaño de la imagen.** La imagen única está cerca de su límite de 40 MB; antes de agregar otro binario conviene separar en una imagen aparte las herramientas que no se despliegan (benchmarks, generación del informe).»
  4. Al fusionar este PR en `develop` y etiquetar `pc4`, el informe contradeciría directamente al `deploy/Dockerfile` entregado en el mismo tag `pc4`.
- **Corrección sugerida:** Actualizar `04-arquitectura.md:38-39` para indicar que `deploy/Dockerfile` compila de forma estática los dos binarios que corren en contenedores (`coordinator` y `node`) sobre `distroless/static`, mientras las herramientas de carga, benchmark e informe se ejecutan en el anfitrión con `go run`. En `14-recomendaciones.md:19-21`, retirar la recomendación ya resuelta (o reemplazarla por una recomendación vigente para TB2, p. ej. incorporar la SPA en una imagen separada Nginx/static para mantener acotado el tamaño de la imagen del backend).

### H2 — Comandos de Docker Compose imprecisos en las secciones 4.2 y 12.2 (`--scale node=N` y omisión de `VERIFICAR=true`) · MAYOR

- **Dónde:** `docs/informe/pc4/04-arquitectura.md:40, 45` y `docs/informe/pc4/12-evidencias.md:21-23`
- **Evidencia:**
  1. En `04-arquitectura.md:40` se escribe «un coordinador, N réplicas del nodo (`--scale node=N`)». Sin embargo, `deploy/docker-compose.yml:40, 69` parametriza ambos servicios con `NODOS` (`-nodos=${NODOS:-3}` en el coordinador y `deploy.replicas: ${NODOS:-3}` en `node`), y `deploy/README.md:49-50` advierte explícitamente: *«Usa `NODOS=5` (levanta 5 y el coordinador espera 5). `--scale node=5` solo cambia cuántos se levantan.»* Ejecutar solo `--scale node=5` deja al coordinador esperando 3 nodos, no N.
  2. Tanto `04-arquitectura.md:45` como `12-evidencias.md:21-23` muestran el comando `docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator` y afirman que **«verifica contra el secuencial [dentro del contenedor]»**. En `deploy/docker-compose.yml:44`, el valor por defecto es `-verificar=${VERIFICAR:-false}`, y en `cmd/coordinator/main.go:257` la verificación interna solo corre cuando `op.verificar` es verdadero. Sin `VERIFICAR=true` (como sí hacen `.github/workflows/ci.yml:125`, `deploy/README.md:36` y `scripts/prueba-caida.sh:15`), el coordinador entrena y escribe el JSON **sin** verificar contra el secuencial dentro del contenedor.
- **Corrección sugerida:** Cambiar `(--scale node=N)` por `NODOS=N` en `04-arquitectura.md:40`, y anteponer `VERIFICAR=true` al comando de entrenamiento único en `04-arquitectura.md:45` y `12-evidencias.md:21` (o bien aclarar en ambos sitios que con `VERIFICAR=true` verifica dentro del contenedor y sin ella se compara luego con `cmd/entrenar -comparar`).

### H3 — Cuatro referencias en `15-referencias.md` no están citadas en el cuerpo del informe (APA 7) · MAYOR

- **Dónde:** `docs/informe/pc4/15-referencias.md:5-7, 38, 40-41, 46` (y citas en `10-api.md:6`, `11-bd.md:11`, `07-modelo.md:109`)
- **Evidencia:**
  En APA 7 (evaluado explícitamente en la rúbrica de Informe y en el criterio 6 del issue #33), toda fuente listada en Referencias debe tener al menos una cita en el texto. La auditoría de las 17 entradas de `docs/informe/pc4/15-referencias.md` contra `docs/informe/pc4/01..14,16*.md` muestra 4 entradas sin cita:
  - `Amdahl, G. M. (1967)` (`15-referencias.md:5-7`): **0 menciones** en todo `docs/informe/pc4/` (quedó huérfana de `docs/informe/pc3/11-recomendaciones.md:11`, que se reescribió para PC4).
  - `MongoDB, Inc. (2025)` (`15-referencias.md:38`): **0 citas** en el texto (`11-bd.md:11` menciona «los drivers oficiales» sin citarla).
  - `OpenAPI Initiative. (2020)` (`15-referencias.md:40-41`): **0 citas** en el texto (`10-api.md:6` escribe `OpenAPI 3.0.3` sin citarla).
  - `Redis Ltd. (2025)` (`15-referencias.md:46`): **0 citas** en el texto (`11-bd.md:11` no la cita).
  Adicionalmente, las citas parentéticas mezclan `y` (`(corrección de prior, King y Zeng, 2001)` en `07-modelo.md:109`, generada desde `cmd/informe/main.go:425`) con `&` (`(Dean & Ghemawat, 2008)`, `(Sergeev & Del Balso, 2018)`, `(Donovan & Kernighan, 2015)` en `02-problema.md:88,90` y `04-arquitectura.md:23`).
- **Corrección sugerida:** Citar `(OpenAPI Initiative, 2020)` en `10-api.md:6`, `(MongoDB, Inc., 2025)` y `(Redis Ltd., 2025)` en `11-bd.md:11`, y citar `(Amdahl, 1967)` en la discusión de fracción serial/límites de escalabilidad de `09-rendimiento.md` (o retirarla de `15-referencias.md` si no se cita). Unificar también el conector en citas parentéticas (`y` en español APA 7, o `&` en todas).

### H4 — `docs/informe/pc4/fig/epoca.svg` conserva el estilo discontinuo «Planificado» de la PC3 para el `Coordinador` · MENOR

- **Dónde:** `docs/informe/pc4/fig/epoca.svg:11` (frente a `docs/informe/pc4/fig/arquitectura.svg:10, 41, 84-87`)
- **Evidencia:**
  `arquitectura.svg` fija la convención visual del informe: relleno azul sólido (`fill="#cde2fb" stroke="#2a78d6" stroke-width="2"`) = *Implementado (PC3 y PC4)*, frente a borde gris discontinuo (`fill="#ffffff" stroke="#8a8984" stroke-width="1.5" stroke-dasharray="6 4"`) = *Planificado (TB2)*. En `docs/informe/pc4/fig/epoca.svg:11`, la caja `Coordinador` sigue usando el borde discontinuo heredado de la PC3, mientras `Nodo 0`, `Nodo 1` y `Nodo 2` (`epoca.svg:13,15,17`) están en azul sólido.
- **Corrección sugerida:** Cambiar el rectángulo de `Coordinador` en `docs/informe/pc4/fig/epoca.svg:11` a `fill="#cde2fb" stroke="#2a78d6" stroke-width="2"` para ser consistente con `arquitectura.svg`.

### H5 — Imprecisiones menores en la sección 10 sobre códigos de respuesta OpenAPI (10.1) y alcance de la corrida caliente de `benchapi` (10.3) · MENOR

- **Dónde:** `docs/informe/pc4/10-api.md:13-15` y `docs/informe/pc4/10-api.md:38-39`
- **Evidencia:**
  1. La tabla de rutas de `10-api.md:13-15` omite códigos de error declarados en `docs/api/openapi.yaml`: `413` en `POST /api/v1/predicciones` (`openapi.yaml:76`), `503` en `GET /api/v1/cluster` (`openapi.yaml:88`) y `400` en `POST /api/v1/entrenamientos` (`openapi.yaml:112`).
  2. `10-api.md:38-39` describe la corrida caliente de `cmd/benchapi` como *«las mismas consultas, ya en Redis»*, lo que sugiere repetir las 2000 consultas de la corrida fría. En realidad, `cmd/benchapi/main.go:92, 145-177` separa mitades disjuntas del espacio de combinaciones y ejecuta 2000 peticiones sobre un conjunto precalentado de `-calientes 32` consultas repetidas.
- **Corrección sugerida:** Incluir los códigos `413`, `503` y `400` en la tabla de `10-api.md:13-15` y precisar en `10-api.md:38-39` que la corrida caliente envía 2000 peticiones sobre un conjunto precalentado de 32 consultas ya en Redis (disjunto de las 2000 consultas frías).

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| — | — | Primera ronda de revisión del PR-44. |

## Cómo explicarlo en la sustentación

El informe de la PC4 se genera desde el repositorio con `cmd/informe`: reescribe tanto las secciones numéricas completas (Resumen, Datos, Limpieza, Modelo, Conclusiones y figuras SVG) a partir de `reports/*.json` como las regiones marcadas (`<!-- generado:... -->`) dentro de las secciones técnicas, copiando al vuelo 9 funciones del algoritmo distribuido desde los archivos Go y las tablas de Spin, escalabilidad del clúster y latencia de la API. Un test automatizado (`TestFragmentosSalenDelFuente`) rompe el CI si el código cambia y el informe no se regenera, y `TestPC4BenchmarksDeUnaMaquina` asegura que los benchmarks de PC3 incluidos en PC4 provengan de una sola máquina comparable. Tras corregir las tres discrepancias documentales (actualización respecto al Dockerfile del PR #46, flags exactos de Compose y citas APA 7), el equipo podrá generar los `.docx` finales con `cmd/word` (ya actualizado en `develop` por el PR #47) e insertar las 15 capturas en la sección 12 antes de etiquetar `pc4`.
