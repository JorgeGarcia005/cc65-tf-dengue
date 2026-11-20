# Veredicto — PR-43 · Ronda 2

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/31-spin-adr5` @ `d7fbffe`
- **Fecha:** 2026-11-20
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l $(find . -name '*.go' -not -path './web/*')
(vacío = ok)

$ go vet ./internal/ml/... ./internal/cluster/... ./internal/ingest/... ./internal/gold/... ./internal/features/... ./internal/almacen ./internal/api/...
(vacío = ok)

$ go test -race -count=1 ./...   # paquetes del núcleo stdlib (sin driver externo de Mongo)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.066s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.158s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.391s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.014s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.035s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.015s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.449s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.840s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.645s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.031s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.016s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.049s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.114s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.495s

$ go test -race -count=1 -run "Equivalencia|ShardsReproducen" -v ./internal/ml ./internal/cluster ./internal/gold
PASS (TestEquivalencia workers=1..64; TestEquivalenciaClusterSecuencialBitABit nodos=1,2,3,5; TestEquivalenciaConMenosNodosQueShards; TestShardsReproducenElGradienteGlobal)

$ bash -n spin/verificar.sh && bash scripts/check-skills-sync.sh && python3 -m json.tool reports/spin.json >/dev/null
skills: ok

$ bash spin/verificar.sh   # ejecutado localmente en copia temporal con Spin 6.5.2 (gcc 12.2.0)
corrida                                          chequeo         estados errores esperado
modelo N=3 (canal de capacidad N)                seguridad      17949492       0 0
modelo N=3 (canal de capacidad N)                termina        17814453       0 0
modelo N=3, canal de capacidad 1                 seguridad      26025432       0 0
modelo N=3, canal de capacidad 1                 termina        25707167       0 0
modelo N=4, 1 época, sin sospechas               seguridad       4923889       0 0
modelo N=4, 1 época, sin sospechas               termina         4920817       0 0
mutante: envío bloqueante (capacidad 1)          seguridad        239247       1 >0
mutante: reasigna solo el shard vencido          seguridad         36613       1 >0
mutante: plazo global                            seguridad            70       1 >0
mutante: primer nodo sano (N=4)                  seguridad          5269       1 >0
spin: Spin Version 6.5.2 -- 6 December 2019
escrito reports/spin.json
RESULTADO: todo como se esperaba

$ gh pr checks 43   # run 44683537924 sobre d7fbffe
Docker (imagen, clúster y caída de un nodo)	pass	1m52s
Ecosistema de agentes                       	pass	10s
Go (gofmt, vet, test -race)                 	pass	49s
MongoDB y Redis (tests de integración)      	pass	46s
OpenAPI 3 válido                            	pass	24s
Spin (modelo del protocolo y mutantes)      	pass	1m42s
```

> Nota de entorno: en este contenedor sin acceso a `proxy.golang.org` (HTTP 403 sobre `github.com/klauspost/compress@v1.16.7`), `cmd/coordinator` e `internal/almacen/mongo` no se compilan localmente por el driver de MongoDB. Todos los paquetes del núcleo stdlib pasaron localmente con `-race` y la suite completa (`go test -race -count=1 ./...`), MongoDB/Redis, Docker, OpenAPI y Spin están en verde en el CI de GitHub (run `44683537924`, commit `d7fbffe`).

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (local en núcleo stdlib + CI completo verde, job `136112351674`) |
| `go test -race ./...` | ✅ (carreras: 0; núcleo stdlib local + suite completa en CI job `136112351674`) |
| Equivalencia secuencial = distribuido | ✅ (PR no modifica Go; `TestEquivalencia*` y `TestShardsReproducen*` en verde) |
| Latencia API p95 < 100 ms | no aplica |
| Spin (`bash spin/verificar.sh`) | ✅ (todas las 10 corridas reproducen bit a bit `reports/spin.json`, `spin/README.md` y CI job `136112351572`, sin `-A` y con detección verificada de búsquedas incompletas) |

## Verificación contra Issue #31 y ADR-0005 (punto 6)

| Criterio (#31 / ADR-0005 §6) | Estado | Evidencia |
|---|---|---|
| Envío no bloqueante (`full(aNodo[j])`), canal de capacidad `N` | ✅ | `spin/barrera.pml:63, 75-76, 141-142, 248-249`. Con `-DCAP=1` y todas las aserciones activas (sin `-A`), `pan-seg` recorre ambas ramas `full(aNodo[j])` (26 025 432 estados, 0 errores; 255/256 estados alcanzados en `Coordinador`). |
| Elección del nodo sano por menor carga, empate por menor índice | ✅ | `spin/barrera.pml:232, 267-271`. La corrida `-DN=4 -DEPOCAS=1 -DSOSPECHAS=0` comprueba el invariante de balanceo `carga[k] <= carga[j] + 1` (4 923 889 estados, 0 errores) y `MUT_PRIMER_SANO` lo viola en 5 269 estados. |
| EOF modelado aparte del plazo, con plazo por shard | ✅ | `spin/barrera.pml:91-93, 134-145, 168-182, 200-201`. Cada envío de `EPOCA` verifica `assert(!descartado[j])`; `MUT_PLAZO_GLOBAL` falla en 70 estados al descartar un nodo vivo y no sospechado. |
| LTL `termina` sin errores y mutantes con contraejemplo (`verificar.sh`) | ✅ | `spin/verificar.sh` (ejecutado en CI job `136112351572` y localmente) valida `seguridad` y `termina` para las 3 configuraciones y exige contraejemplo en los 4 mutantes, rechazando búsquedas truncadas y `$errores` no numéricos. |
| `spin/README.md` y trazabilidad para la tabla del informe | ✅ | `spin/README.md:49-80` coincide exactamente con `reports/spin.json` (generado por `verificar.sh` para consumo de `cmd/informe` en #33, cumpliendo `AGENTS.md` §3.6). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Plus formal de PC4 completo | Modelo `spin/barrera.pml` alineado con el ADR-0005 e `internal/cluster/coordinador.go`, verificado con seguridad + LTL + 4 mutantes en CI en cada push sin desactivar aserciones. |
| Interfaz | 5 | no aplica | No toca API ni SPA. |
| Informe | 2 | Mantiene 2 / 2 | `spin/README.md` documenta las propiedades, contraejemplos y alcance de `N=3` vs `N=4`, y `reports/spin.json` entrega cifras generadas por código listas para #33. Git Flow y trazabilidad multiagente completos. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos abiertos.

## Seguimiento de rondas previas

| Hallazgo | Estado | Nota |
|---|---|---|
| H1 (ronda 1) — `full(aNodo[j])` inalcanzable con `CAP=N` y ejecutado con `-A` en `CAP=1` · BLOQUEANTE | **RESUELTO** | Commits `040522d` y `026b75e`: variable fantasma `saturado[N]` (`spin/barrera.pml:75-76`) puesta únicamente en las dos ramas `full(aNodo[j])` (`:141, 248`), aserciones actualizadas a `!vivo \|\| sospechado \|\| saturado` (`:112, 291`) y eliminación total de `-A` en `spin/verificar.sh:28-34`. Comprobado localmente con Spin 6.5.2 sobre `-DCAP=1`: 26 025 432 estados, 0 errores, `assertion violations +`, `invalid end states +`, con 255/256 estados de `Coordinador` alcanzados (solo queda sin alcanzar el `else` de `DESCARTAR` en `:141`, matemáticamente imposible porque la guarda previa garantiza `!descartado[j]`) y 0 estados sin alcanzar en `Nodo` e `init`. |
| H2 (ronda 1) — Envío de `EPOCA` a nodos ya descartados al abrir la época 1 · BLOQUEANTE | **RESUELTO** | Commit `040522d`: la difusión de época omite los shards cuyo dueño ya está descartado (`spin/barrera.pml:134-136`) para que `cubrir` los reasigne igual que `internal/cluster/coordinador.go:276, 415-424`, y ambos envíos de `EPOCA` incorporan `assert(!descartado[j])` (`:142, 249`). Verificado con 0 errores en las tres corridas del modelo (y la corrida `N=3, CAP=N` bajó de 18 450 303 a 17 949 492 estados al eliminar esas transiciones espurias). |
| H3 (ronda 1) — Con `N=3`, la selección por menor carga nunca se diferenciaba del primer nodo sano · MAYOR | **RESUELTO** | Commits `040522d`, `026b75e` y `0f73e04`: parámetros configurables por `-D` (`spin/barrera.pml:49-60`), guardas `#if N > 3` para el shard 3 (`:173-175, 180-182`), aserción de balanceo `carga[k] <= carga[j] + 1` en `avanzar` (`:267-271`), corrida dedicada `-DN=4 -DEPOCAS=1 -DSOSPECHAS=0` (4 923 889 estados seguridad / 4 920 817 termina, 0 errores) y cuarto mutante `MUT_PRIMER_SANO` (`:229-231`, `verificar.sh:64`) que viola la aserción de balanceo en profundidad 380 tras 5 269 estados. Documentado en `spin/README.md:77-80`. |
| H4 (ronda 1) — `spin/verificar.sh` aprobaba búsquedas truncadas y `$errores` vacío · MAYOR | **RESUELTO** | Commit `026b75e` (`spin/verificar.sh:37-55`): exige entero en `$errores` (`^[0-9]+$`), rechaza salidas con `Search not completed\|max search depth too small\|out of memory` cuando `esperado=0`, y evalúa mutantes con `[ "$errores" -gt 0 ]`. Probado empíricamente en copia temporal contra tres casos negativos: `./pan-seg -m50` falla con `ERROR: búsqueda incompleta` y código 1; salida corrupta sin `errors:` falla incluso en mutante con `ERROR: no se pudo leer el número de errores de pan` y código 1; mutante sin fallos falla con `ERROR: el mutante no dio contraejemplo` y código 1. |
| H5 (ronda 1) — Precisión documental sobre el contraejemplo de `MUT_PLAZO_GLOBAL` y reporte JSON para #33 · MENOR | **RESUELTO** | Commits `026b75e` y `0f73e04`: `spin/README.md:72-76` distingue con precisión el contraejemplo mínimo de Spin (verificado con `spin -t -p -DMUT_PLAZO_GLOBAL barrera.pml`: 70 estados, 87 pasos, `caidas = 0`, sospecha falsa sobre shard 0 que descarta al nodo 1 vivo y no sospechado) de su motivación en el sistema real (`TestCaidaEscalonadaPorEOFYPlazoNoDescartaAlNodoSano`, ADR-0005 test 3). Además `verificar.sh:68-75` genera `reports/spin.json` (válido y exactamente igual a la tabla del README y al CI) listo para `cmd/informe` en #33. |

## Cómo explicarlo en la sustentación

Modelamos en Promela (`spin/barrera.pml`) el coordinador síncrono del ADR-0005 con caídas por EOF separadas de nodos colgados, plazo individual por shard, reasignación en dos pasadas al nodo sano de menor carga y envío no bloqueante ante búfer lleno, ejecutado automáticamente en CI (`spin/verificar.sh`) y volcado a `reports/spin.json`. Spin verifica de forma exhaustiva sin desactivar aserciones tres configuraciones —capacidad `N` (17,9 millones de estados), capacidad `1` para forzar saturación de búfer (26,0 millones) y `N = 4` para ejercitar el balanceo (4,9 millones)— probando ausencia de *deadlock*, barrera completa, equilibrio de carga (`<= 1`), cero envíos a nodos descartados, cero descartes injustificados de nodos vivos y terminación LTL (`<> terminado`). A la vez, cuatro mutantes que eliminan una sola decisión del ADR (envío bloqueante, reasignación en una pasada, plazo global y primer nodo sano) producen contraejemplo, demostrando que el modelo distingue cada decisión del defecto que corrige.
