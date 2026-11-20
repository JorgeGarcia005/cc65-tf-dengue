# Veredicto — PR-43 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/31-spin-adr5` @ `bd460d2`
- **Fecha:** 2026-11-20
- **Decisión:** CAMBIOS

## Verificación automática

```text
$ gofmt -l $(find . -name '*.go' -not -path './web/*')
(vacío = ok)

$ go vet ./internal/ml/... ./internal/cluster/... ./internal/ingest/... ./internal/gold/... ./internal/features/... ./internal/almacen ./internal/api/...
(vacío = ok)

$ go test -race -count=1 ./internal/cluster/... ./internal/ml/... ./internal/estadistica ./internal/features ./internal/gold ./internal/ingest ./internal/maquina ./internal/almacen ./internal/api ./cmd/benchmark ./cmd/entrenar ./cmd/gold ./cmd/node ./cmd/revalidar ./cmd/word
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	6.876s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.069s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	6.642s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.064s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.030s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.049s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.307s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.022s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/api	4.562s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.148s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.024s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.201s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.559s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/revalidar	1.080s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.030s

$ go test -race -count=1 -run "Equivalencia|ShardsReproducen" -v ./internal/ml ./internal/cluster ./internal/gold
PASS (TestEquivalencia workers=1..64; TestEquivalenciaClusterSecuencialBitABit nodos=1,2,3,5; TestEquivalenciaConMenosNodosQueShards; TestShardsReproducenElGradienteGlobal)

$ bash -n spin/verificar.sh && bash scripts/check-skills-sync.sh
skills: ok

$ bash spin/verificar.sh   # ejecutado localmente con Spin 6.5.2 compilado desde fuente
corrida                                      chequeo           estados  errores esperado
modelo (canal de capacidad N)                seguridad        18450303        0 0
modelo (canal de capacidad N)                termina          18306811        0 0
modelo, canal de capacidad 1, sin asserts    seguridad        26708282        0 0
modelo, canal de capacidad 1, sin asserts    termina          26370789        0 0
mutante: envío bloqueante (capacidad 1)     seguridad          236361        1 >0
mutante: reasigna solo el shard vencido      seguridad           35317        1 >0
mutante: plazo global                        seguridad              64        1 >0
spin: Spin Version 6.5.2 -- 6 December 2019
RESULTADO: todo como se esperaba

$ gh pr checks 43
Docker (imagen, clúster y caída de un nodo)	pass	1m56s
Ecosistema de agentes                       	pass	12s
Go (gofmt, vet, test -race)                 	pass	45s
MongoDB y Redis (tests de integración)      	pass	46s
OpenAPI 3 válido                            	pass	24s
Spin (modelo del protocolo y mutantes)      	pass	1m59s
```

> Nota de entorno: en este contenedor sin acceso a `proxy.golang.org` (HTTP 403 sobre `github.com/klauspost/compress`), `go test ./...` completo no descarga el driver de MongoDB. Todos los paquetes del núcleo stdlib pasaron localmente con `-race` y la suite completa (`go test -race -count=1 ./...`) más la integración de Mongo/Redis, Docker y Spin están en verde en el CI de GitHub (run `44680546270`).

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ (local en núcleo stdlib + CI completo verde) |
| `go test -race ./...` | ✅ (carreras: 0; núcleo stdlib local + suite completa en CI job `136102721105`) |
| Equivalencia secuencial = distribuido | ✅ (PR no modifica Go; `TestEquivalencia*` y `TestShardsReproducen*` en verde) |
| Latencia API p95 < 100 ms | no aplica |
| Spin (`bash spin/verificar.sh`) | ✅ reproduce exactamente los números del README y de CI job `136102720960` (pero ver H1–H4) |

## Verificación contra Issue #31 y ADR-0005 (punto 6)

| Criterio (#31 / ADR-0005 §6) | Estado | Evidencia |
|---|---|---|
| Envío no bloqueante (`full(aNodo[j])`), canal de capacidad `N` | ⚠️ parcial | Presente en `spin/barrera.pml:49,122,218`, pero inalcanzable con `CAP=N` y ejecutado con `-A` (sin `assert`) en `CAP=1` (**H1**). |
| Elección del nodo sano por menor carga, empate por menor índice | ⚠️ parcial | Implementado en `spin/barrera.pml:200-206`, pero matemáticamente inalcanzable con `N=3` (**H3**). |
| EOF modelado aparte del plazo, con plazo por shard | ⚠️ parcial | Modelado en `barrera.pml:75,148-156,174-175`, pero la difusión de época 1 envía `EPOCA` a nodos ya descartados (**H2**). |
| LTL `termina` sin errores y mutantes con contraejemplo (`verificar.sh`) | ⚠️ parcial | Los 3 mutantes producen contraejemplo y `ci.yml:30-39` los corre en CI, pero `verificar.sh` acepta búsquedas truncadas y `$errores` vacío (**H4**). |
| `spin/README.md` y tabla del informe con estados reales | ✅ / diferido a #33 | `spin/README.md:45-53` coincide exactamente con Spin 6.5.2 (CI y local); postergar la tabla del informe a #33 (`docs/informe/pc4/`) es razonable (**H5**). |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Plus formal de PC4 (requiere correcciones H1–H4) | Actualiza `spin/barrera.pml` hacia el ADR-0005 y agrega el job `spin` en `.github/workflows/ci.yml`, pero hoy no verifica las aserciones de seguridad cuando `full(aNodo[j])` dispara y envía `EPOCA` a nodos ya descartados en época 1. |
| Interfaz | 5 | no aplica | No toca API ni SPA. |
| Informe | 2 | Mantiene 2 / 2 (con ajuste menor H5) | Documenta el modelo y los resultados reales en `spin/README.md`, base para la sección de verificación formal en #33; Git Flow, tamaño del PR y trazabilidad de IA correctos. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

### H1 — El envío no bloqueante (`full(aNodo[j])`) es inalcanzable con `CAP=N` y con `CAP=1` se ejecuta con `-A` (todos los `assert` desactivados) · BLOQUEANTE

- **Dónde:** `spin/barrera.pml:93-98`, `spin/barrera.pml:121-124`, `spin/barrera.pml:217-222`, `spin/verificar.sh:43-44` y `spin/README.md:49-50, 57-62`
- **Evidencia:**
  1. En la corrida principal (`CAP = N`), `pan-seg` reporta explícitamente que las dos ramas `full(aNodo[j])` jamás se ejecutan en los 18 450 303 estados:
     ```text
     unreached in proctype Coordinador
     	barrera.pml:122, state 28, "carga[j] = 0"
     	barrera.pml:122, state 29, "assert((!(vivo[j])||sospechado[j]))"
     	barrera.pml:122, state 31, "(1)"
     	barrera.pml:122, state 32, "(!(descartado[j]))"
     	barrera.pml:122, state 32, "else"
     	barrera.pml:218, state 168, "carga[j] = 0"
     	barrera.pml:218, state 169, "assert((!(vivo[j])||sospechado[j]))"
     	(6 of 230 states)
     ```
  2. Para ejercitar `full(aNodo[j])`, `spin/verificar.sh:44` corre `-DCAP=1` con la bandera **`-A`** (`./pan-seg -A` y `./pan -a -A -N termina`).
  3. En Spin, `-A` **desactiva todas las aserciones del modelo**, no únicamente el `assert` de `DESCARTAR` (`:96`): también apaga `assert(listo[k])` (barrera por época, `barrera.pml:234`), `assert(abortado || epocasCompletas == EPOCAS)` (sin pérdida de shards, `barrera.pml:250`) y `assert(!vivo[k] || sospechado[k])` (`barrera.pml:255`).
  4. En consecuencia, ninguna ejecución que realmente recorra el descarte y reintento por canal lleno (verificación 6(a) del ADR-0005 y criterio 1 del issue #31) comprueba las propiedades de barrera ni de conservación de shards.
- **Corrección sugerida:** Registrar explícitamente la saturación del canal en una variable fantasma (por ejemplo, `bool saturado[N]` puesta a `true` solo en las ramas `full(aNodo[j])` de `:122` y `:218`, o marcar `sospechado[j] = true` en esas ramas) y comprobar `assert(!vivo[jj] || sospechado[jj] || saturado[jj])` en `:96` y `:255`. Con ello, eliminar `-A` de `spin/verificar.sh:44` para que la corrida `-DCAP=1` verifique **todas** las aserciones de seguridad y vivacidad mientras ejercita `full(aNodo[j])`.

### H2 — Al iniciar la época 1, el coordinador envía `EPOCA` a nodos ya descartados (diverge de `coordinador.go` y del ADR-0005) · BLOQUEANTE

- **Dónde:** `spin/barrera.pml:10, 93-98, 113-127` (frente a `internal/cluster/coordinador.go:276, 310-350, 409-424`)
- **Evidencia:**
  1. Cuando un nodo `j` entrega el gradiente de un shard `s` en la época 0 (`listo[s] = true`) y luego es descartado en la misma época, `DESCARTAR(j)` (`barrera.pml:93-98`) no limpia `duenio[s]` y `cubrir` (`barrera.pml:189`) solo reasigna los shards con `!listo[k]`. Así, la época 0 cierra con `duenio[s] == j` y `descartado[j] == true`.
  2. Al abrir la época 1 (`barrera.pml:113-126`), el bucle de difusión ejecuta `j = duenio[s]` y `aNodo[j] ! EPOCA(e, s, j)` **sin comprobar `!descartado[j]`** (y lo mismo puede suceder dentro de un mismo bucle si `j` se descarta por `full(aNodo[j])` en un shard previo y otro shard todavía apunta a `j`).
  3. **Contraejemplo reproducible con Spin 6.5.2:** Al añadir `assert(!descartado[j])` justo antes de `aNodo[j] ! EPOCA(e, s, j)` en `barrera.pml:123`, `./pan-seg` falla en **0,45 s** (profundidad 263, 695 643 estados):
     ```text
     pan:1: assertion violated  !(descartado[j]) (at depth 263)
     ...
     262: proc 4 (Coordinador:1) probe_a.pml:117 (state 24) [j = duenio[s]]
     263: proc 4 (Coordinador:1) probe_a.pml:123 (state 34) [(nfull(aNodo[j]))]
     spin: probe_a.pml:123, Error: assertion violated
     ```
     En esa traza, en la época 0 el nodo 0 es sospechado y su shard pasa al nodo 1; el nodo 1 completa el shard 1 y muere en el shard 0 (`vivo[1] = 0`, `descartado[1] = 1`); al arrancar la época 1 (`epocasCompletas = 1`), el coordinador envía `EPOCA(1, 1, 1)` al nodo 1 ya muerto y descartado. Cuando el nodo descartado sigue vivo (sospecha falsa), ese envío espurio incluso reactiva al nodo descartado en la época 1, permitiéndole emitir mensajes o consumir `caidas`.
  4. En `internal/cluster/coordinador.go:415-424`, `descartar(j)` cierra la conexión y deja `c.duenio[s] = -1` para **todos** los shards de `j` (incluidos los ya listos), y cada época empieza directamente con `c.cubrir()` (`coordinador.go:276`), por lo que jamás se escribe a un nodo descartado.
- **Corrección sugerida:** Omitir el envío en `barrera.pml:113-126` cuando `descartado[j]` es verdadero (o limpiar `duenio[k]` al descartar y unificar el inicio de época con `cubrir`, igual que `coordinador.go:276`) y conservar `assert(!descartado[j])` antes de cada envío de `EPOCA`. Además de alinear el modelo con `coordinador.go`, esto elimina transiciones espurias y reduce el espacio de estados.

### H3 — Con `N = 3`, la selección por menor carga (`carga[k] < carga[mejor]`) nunca llega a activarse · MAYOR

- **Dónde:** `spin/barrera.pml:43, 200-206` y `spin/README.md:10, 45-54`
- **Evidencia:**
  1. Con `N = 3`, cada nodo sano parte con `carga[k] = 1` y nunca pierde shards mientras sigue sano ($\text{carga}[k] \ge 1$). Para que `carga[k] < carga[mejor]` (`barrera.pml:203`) sea verdadero se necesitan al menos dos nodos sanos `mejor < k` con $\text{carga}[\text{mejor}] \ge 2$ y $\text{carga}[k] \ge 1$, más al menos un shard pendiente de asignar: mínimo $2 + 1 + 1 = 4$ shards, imposible con `N = 3`.
  2. **Prueba exhaustiva con Spin 6.5.2:** Al insertar `assert(mejor == N || !(carga[k] < carga[mejor]))` dentro de la rama de `barrera.pml:203`, `./pan-seg` recorrió los **18 450 303 estados** completos en 24,5 s con **`errors: 0`**. Es decir, en el modelo actual el criterio de menor carga nunca se diferencia de elegir el primer nodo sano (`j = 0`, el diseño previo al ADR-0005).
  3. **Factibilidad verificada con `N = 4`:** Parametrizando `N`, `EPOCAS` y `SOSPECHAS` y añadiendo la guarda del shard 3, una corrida dedicada con `-DN=4 -DEPOCAS=1 -DSOSPECHAS=0` y `assert(descartado[k] || carga[k] <= 2)` cuando quedan 2 nodos sanos completó **4 876 616 estados en 3,94 s** (1,17 GB de RAM, `errors: 0`), confirmando que verificar el balanceo real en Spin cabe holgadamente en CI.
- **Corrección sugerida:** Añadir una corrida corta con `N = 4, EPOCAS = 1, CAIDAS = 2, SOSPECHAS = 0` en `verificar.sh` que verifique el balanceo (`carga[k] <= 2` con 2 nodos sanos; opcionalmente con un mutante de primer nodo sano que falle esa aserción), **o bien** —si se decide mantener `barrera.pml` fijo en `N = 3`— documentar explícitamente en `spin/README.md` que con `N = 3` Spin solo ejercita el desempate por índice y que el balanceo con `N = 4` se demuestra en `TestBalanceoTrasDobleCaidaReparteDosYDos` (`internal/cluster/cluster_test.go:364-387`).

### H4 — `spin/verificar.sh` aprueba búsquedas truncadas y mutantes con `$errores` vacío · MAYOR

- **Dónde:** `spin/verificar.sh:30-35` (usado como gate en `.github/workflows/ci.yml:38`)
- **Evidencia:**
  1. `pan` sale con **código 0** y `errors: 0` aunque trunque la exploración por profundidad o memoria insuficiente. Comprobado sobre el `pan-seg` del PR:
     ```text
     $ ./pan-seg -m10; echo "EXIT_CODE=$?"
     error: max search depth too small
     ...
     State-vector 128 byte, depth reached 9, errors: 0
     EXIT_CODE=0
     ```
     Como `spin/verificar.sh:33` solo evalúa `[ "$errores" != 0 ]`, una corrida incompleta (`max search depth too small`, `Search not completed`, `out of memory`) pasaría en verde en CI.
  2. En `spin/verificar.sh:34`:
     ```bash
     if [ "$esperado" = ">0" ] && [ "$errores" != 0 ]; then return 0; fi
     ```
     Si `$errores` resulta vacío (`""`), `[ "" != 0 ]` es verdadero en bash y la función retorna `0`, aceptando el mutante aunque `pan` no haya reportado ningún contraejemplo.
- **Corrección sugerida:** Comprobar que `$errores` sea un entero (`[[ "$errores" =~ ^[0-9]+$ ]]`), fallar cuando `$esperado = 0` si `$salida` contiene `Search not completed|max search depth too small|out of memory`, y evaluar mutantes con `[ "$errores" -gt 0 ]`.

### H5 — Precisión documental sobre el contraejemplo de `MUT_PLAZO_GLOBAL` y trazabilidad de la tabla del informe · MENOR

- **Dónde:** `spin/README.md:53, 65-67` y criterio 5 del issue #31 (`AGENTS.md` §3.6)
- **Evidencia:**
  1. `spin/README.md:65-67` afirma que en `MUT_PLAZO_GLOBAL` el vencimiento de un nodo colgado descarta a un nodo vivo *"al que se le acababa de reasignar un shard (el escenario de la caída escalonada del ADR-0005, test 3)"*. Sin embargo, al reproducir el contraejemplo de 64 estados (`spin -t -p -DMUT_PLAZO_GLOBAL barrera.pml`, 81 pasos), `caidas = 0` y no hay ninguna reasignación previa: una sospecha falsa dispara sobre el shard 0 nada más difundir la época 0 y la línea 166 descarta de inmediato al nodo 1 vivo y no sospechado. Precisarlo evita contradicciones si el profesor pide correr el `.trail` en vivo.
  2. Diferir la actualización de la tabla del informe al issue #33 (`docs/informe/pc4/`) tiene sentido porque `docs/informe/pc3/` corresponde a la entrega ya cerrada. Como recordatorio para #33 (y `AGENTS.md` §3.6: *"Ninguna cifra a mano en el informe"*), convendrá que `verificar.sh` pueda emitir un `reports/spin.json` que consuma `cmd/informe` en lugar de transcribir estados a mano.
- **Corrección sugerida:** Ajustar las líneas 65–67 de `spin/README.md` para distinguir la motivación en el sistema real (test 3 del ADR-0005) del contraejemplo mínimo de 64 estados que encuentra Spin; opcionalmente añadir una bandera para volcar JSON si se desea adelantar el insumo de #33.

## Seguimiento de rondas previas

No aplica (primera ronda de revisión del PR-43).

## Cómo explicarlo en la sustentación

Modelamos en Promela (`spin/barrera.pml`) el coordinador síncrono del ADR-0005 con caídas por EOF separadas de nodos colgados, plazo individual por shard, reasignación en dos pasadas al nodo sano de menor carga y envío no bloqueante ante búfer lleno, y lo verificamos en cada push mediante GitHub Actions (`spin/verificar.sh`). Spin explora exhaustivamente los estados posibles para comprobar ausencia de *deadlock*, cumplimiento de la barrera, conservación de shards, no descarte injustificado de nodos vivos y terminación LTL (`<> terminado`), mientras que tres mutantes (envío bloqueante, reasignación en una pasada y plazo global) producen contraejemplo y demuestran que el modelo sí distingue cada decisión del ADR del defecto que evita. Una vez corregidos el uso de `-A` en la corrida de canal saturado y el envío a nodos ya descartados al inicio de la época 1, toda la máquina de estados quedará alineada con `internal/cluster/coordinador.go` con todas las aserciones activas.
