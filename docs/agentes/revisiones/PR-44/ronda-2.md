# Veredicto — PR-44 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/33-informe-pc4` @ `6bd970c`
- **Fecha:** 2026-11-21
- **Decisión:** APROBADO

Los cinco hallazgos de la ronda 1 (**H1–H5**) fueron aceptados y resueltos de forma completa y verificable en los commits `b5fbe0d` (fusión de `develop` con PR #46 y PR #47) y `8c45cb4`, con respuesta documentada en `docs/agentes/revisiones/PR-44/respuesta-1.md` (`6bd970c`). La auditoría sobre el diff completo no encontró hallazgos nuevos: todas las verificaciones locales (`gofmt`, `go vet`, `go test -race -count=1 ./...`, equivalencia secuencial = distribuido, idempotencia de `cmd/informe`, `TestFragmentosSalenDelFuente`, auditoría OOXML con el `cmd/word` actualizado del PR #47 y trazabilidad 17/17 de citas APA 7) y los 6 jobs del CI de GitHub (run `44938025551`) están en verde. El PR queda listo para merge con aprobación humana.

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)

$ go vet ./...
(vacío = ok)

$ git diff --check origin/develop...origin/feature/33-informe-pc4
(vacío = ok)

$ bash scripts/check-skills-sync.sh
skills: ok

$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchapi	3.121s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	8.817s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	2.585s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.038s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.279s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.223s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.444s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.274s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/docs/api	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.046s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/contrato	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo	4.082s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	5.411s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	9.541s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	4.601s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.041s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.023s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.075s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	6.828s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	13.338s

$ go test -race -count=1 -v ./cmd/informe ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	1.268s (5/5 PASS: TestEtiquetaBench, TestPC4BenchmarksDeUnaMaquina, TestFragmentosSalenDelFuente, TestExtraerFuncionYMetodo, TestRegionDesconocidaDaError)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.170s (13/13 PASS, incluidos los 5 casos de plantilla_test.go del PR #47)

$ go test -race -count=1 -run Equivalencia -v ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	2.693s (TestEquivalenciaClusterSecuencialBitABit nodos=1,2,3,5; TestEquivalenciaConMenosNodosQueShards)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	1.769s (TestEquivalencia workers=1,2,4,8,16,64)

$ go run ./cmd/informe && git diff --exit-code docs/informe
INFORME_IDEMPOTENTE=OK

$ go run ./cmd/entrenar -comparar reports/entrenamiento_cluster.json
reports/entrenamiento.json y reports/entrenamiento_cluster.json: mismos hiperparámetros, pesos, sesgo e historia de pérdida, idénticos bit a bit

$ go test -race -count=1 -overlay /tmp/audit_pc4_overlay.json -run TestAuditPC4WordXML -v ./cmd/word
PC4 Word OK: 16 secciones, 9 SVGs, 9 PNGs, 0 avisos

$ gh pr checks 44   # run 44938025551 sobre 6bd970c
Docker (imagen, clúster y caída de un nodo)	pass	2m28s
Ecosistema de agentes                       	pass	6s
Go (gofmt, vet, test -race)                 	pass	45s
MongoDB y Redis (tests de integración)      	pass	41s
OpenAPI 3 válido                            	pass	22s
Spin (modelo del protocolo y mutantes)      	pass	1m54s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; suite completa local + CI verde en run `44938025551`) |
| Equivalencia secuencial = distribuido | ✅ (`TestEquivalencia*` + `cmd/entrenar -comparar reports/entrenamiento_cluster.json` bit a bit) |
| Latencia API p95 < 100 ms | ✅ (no modifica `internal/api`; `reports/bench_api_deez.json` registra p95 = 2,21 ms fría y 2,06 ms caliente, 0 errores en 16 clientes) |
| Reproducibilidad del informe y Word XML | ✅ (`cmd/informe` idempotente, `TestFragmentosSalenDelFuente` en verde, 16 secciones PC4 + 9 SVGs/PNGs convertidos a OOXML válido con 0 avisos tras la fusión del PR #47) |
| Citas y referencias APA 7 | ✅ (17/17 entradas de `15-referencias.md` citadas en el cuerpo; conector `y` unificado en prosa y `&` en la lista de referencias) |

## Verificación contra los criterios del issue #33

| Criterio (#33) | Estado | Evidencia |
|---|---|---|
| Obs. 1: «Pregunta que resuelve el problema» explícita en 2.2, retomada con las mismas palabras en Resumen, Objetivos y Conclusiones, y en la tabla enunciado → sección | ✅ | Subtítulo en `02-problema.md:31`; cadena exacta (351 caracteres) verificada en `01-resumen.md`, `02-problema.md`, `03-objetivos.md` y `13-conclusiones.md`; mapeada en `04-arquitectura.md:57`. |
| Obs. 2: código fuente del algoritmo distribuido copiado del repositorio + pseudocódigo numerado «Algoritmo n» | ✅ | Algoritmos 1 y 2 en `08-distribuido.md`; regiones `<!-- generado:fuente:... -->` verificadas por `TestFragmentosSalenDelFuente`. |
| Secciones nuevas (clúster, topología, protocolo + secuencia, fallos, Spin, rendimiento, API, Mongo/Redis) | ✅ | Consistentes con `deploy/Dockerfile`, `deploy/docker-compose.yml`, `docs/api/openapi.yaml`, `cmd/benchapi` y `fig/arquitectura.svg` tras resolver H1, H2, H4 y H5. |
| Evidencias con capturas reales (`grep -c 'PENDIENTE'` = 0) | ✅ (con nota operativa) | `grep -c 'PENDIENTE' docs/informe/pc4/*.md` = 0; `12-evidencias.md` deja los 15 marcos `<!-- captura: ... -->` (con comandos exactos tras H2) para pegar las capturas en el `.docx` final antes de subir la entrega. |
| Ninguna cifra a mano (`go run ./cmd/informe && git diff --exit-code docs/informe`) | ✅ | 0 diferencias al regenerar; `TestPC4BenchmarksDeUnaMaquina` y `TestFragmentosSalenDelFuente` en verde. |
| Trabajos relacionados (2.6) y bibliografía APA 7 con DOI/documento oficial | ✅ | `02-problema.md:84-97` y `15-referencias.md` con DOIs/URLs oficiales y correspondencia exacta 17/17 en el cuerpo tras H3. |
| Tabla «Correcciones a la PC3» | ✅ | `16-anexos.md:3-11`, referenciada en `04-arquitectura.md:73`. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | 3 / 3 | Observación 1 de PC3 resuelta de punta a punta (Resumen, 2.2 con tabla de trazabilidad, Objetivos y Conclusiones), más sección de trabajos relacionados (`02-problema.md:84-97`). |
| Implementación | 8 | Avance completo de PC4 | Cierra el seguimiento del PR #42 en `cmd/benchmark` (`-recorte-cluster` validado, `nodosMedidos`, nota Karp-Flatt), corrige SIGPIPE en `scripts/evidencia-bd.sh`, alinea la documentación con el `Dockerfile` del PR #46 y versiona las mediciones reales del Ryzen 7 5800X. |
| Interfaz | 5 | no aplica | Corresponde a TB2. |
| Informe | 2 | 2 / 2 (a falta de insertar las 15 capturas en el `.docx` final) | Generación 100 % reproducible desde el código y `reports/*.json`, 0 avisos en `cmd/word` (ya con las correcciones del PR #47), citas APA 7 17/17 consistentes, despliegue y API exactos, Git Flow y trazabilidad multiagente completos. |
| Video | 2 | no aplica | Corresponde a TB2. |

## Hallazgos

Sin hallazgos abiertos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — Secciones 4.2 y 14 contradecían `deploy/Dockerfile` de `develop` (PR #46) · MAYOR | **RESUELTO** | `b5fbe0d` integró `develop` (#46, #47) y `8c45cb4` actualizó `docs/informe/pc4/04-arquitectura.md:38-41` (la imagen compila estáticamente solo `coordinator` y `node` sobre `distroless/static`; herramientas en anfitrión con `go run`, exacto con `deploy/Dockerfile:24`) y reemplazó la recomendación resuelta de `14-recomendaciones.md:19-21` por servir la SPA de TB2 en su propia imagen estática. |
| H2 (ronda 1) — Comandos de Docker Compose imprecisos en 4.2 y 12.2 (`--scale node=N` y falta de `VERIFICAR=true`) · MAYOR | **RESUELTO** | `8c45cb4` eliminó toda referencia a `--scale` (0 apariciones en `docs/informe/pc4/`), fijó `NODOS=N` en `04-arquitectura.md:42-43` y `NODOS=3 VERIFICAR=true docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator` en `04-arquitectura.md:48` y `12-evidencias.md:21-24`, precisando la diferencia con y sin `VERIFICAR=true` también en `08-distribuido.md:363-365`. |
| H3 (ronda 1) — Cuatro referencias sin citar en `15-referencias.md` y mezcla de `y`/`&` en el cuerpo · MAYOR | **RESUELTO** | `8c45cb4` citó `(Amdahl, 1967)` en `09-rendimiento.md:53`, `OpenAPI Initiative, 2020` en `10-api.md:6` y `(MongoDB, Inc., 2025; Redis Ltd., 2025)` en `11-bd.md:11-12`, logrando 17/17 correspondencia verificada entre `15-referencias.md` y el cuerpo; unificó `y` en todas las citas parentéticas en español (`02-problema.md:88,90`, `04-arquitectura.md:23`) manteniendo `&` únicamente en `15-referencias.md` conforme a APA 7. |
| H4 (ronda 1) — `fig/epoca.svg` conservaba estilo discontinuo «Planificado» para `Coordinador` · MENOR | **RESUELTO** | `8c45cb4` actualizó `docs/informe/pc4/fig/epoca.svg:11` a `fill="#cde2fb" stroke="#2a78d6" stroke-width="2"` (sin `stroke-dasharray`), consistente con los nodos (`:13,15,17`) y con la convención *Implementado (PC3 y PC4)* de `fig/arquitectura.svg`. |
| H5 (ronda 1) — Omisión de códigos OpenAPI en 10.1 y descripción imprecisa de corrida caliente en 10.3 · MENOR | **RESUELTO** | `8c45cb4` incorporó `413`, `503` y `400` en `docs/informe/pc4/10-api.md:13-15` (100 % alineado con `docs/api/openapi.yaml:66-114`) y precisó en `10-api.md:37-40` que la corrida caliente envía 2000 peticiones sobre 32 consultas precalentadas en Redis, en una mitad del espacio de consultas disjunta de la corrida fría. |

## Cómo explicarlo en la sustentación

El informe de la PC4 se genera desde el repositorio con `cmd/informe`: reconstruye las secciones numéricas y gráficos SVG desde `reports/*.json` y reescribe las regiones marcadas (`<!-- generado:... -->`) copiando las funciones reales del algoritmo distribuido desde el código Go y las tablas de Spin, clúster y API. Un test automatizado (`TestFragmentosSalenDelFuente`) rompe el CI si el código cambia y nadie regenera el informe, y `TestPC4BenchmarksDeUnaMaquina` garantiza que los resultados de PC3 incluidos en PC4 provengan de una sola máquina comparable. Con las cinco correcciones de la ronda 1 aplicadas y el generador `cmd/word` ya actualizado en la rama, el equipo puede armar directamente los `.docx` finales, insertar las 15 capturas en la sección 12 y cerrar el release `pc4`.
