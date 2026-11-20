# Veredicto — PR-37 · Ronda 1

- **Juez:** Antigravity · modelo: Gemini
- **Rama revisada:** `feature/27-docker-cluster` @ `49c0abd`
- **Fecha:** 2026-11-19
- **Decisión:** APROBADO

## Verificación automática

```text
$ gofmt -l .
(vacío = ok)
$ go vet ./...
(vacío = ok)
$ go test -race -count=1 ./...
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.066s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	1.773s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.013s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.167s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/informe	[no test files]
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/loader	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	1.406s
?   	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/perfil	[no test files]
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.041s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	5.901s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	2.774s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.028s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.014s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.048s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.339s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.026s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	8.420s

$ go test -race -count=1 -run Equivalencia -v ./...
=== RUN   TestEquivalenciaClusterSecuencialBitABit
--- PASS: TestEquivalenciaClusterSecuencialBitABit (0.46s) (nodos=1,2,3,5)
=== RUN   TestEquivalenciaConMenosNodosQueShards
--- PASS: TestEquivalenciaConMenosNodosQueShards (0.09s)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster	1.565s
=== RUN   TestEquivalencia
--- PASS: TestEquivalencia (0.20s) (workers=1,2,4,8,16,64)
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	1.218s

$ go test -race -count=5 ./cmd/coordinator ./cmd/node ./cmd/entrenar ./cmd/gold
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/coordinator	4.540s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/node	2.909s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.017s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/gold	1.765s

$ CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/cc65-out/ ./cmd/...
(9 binarios estáticos compilados sin error; tamaño total sin comprimir 27M)

$ go run ./cmd/entrenar -comparar reports/entrenamiento.json
reports/entrenamiento.json y reports/entrenamiento.json: mismos hiperparámetros, pesos, sesgo e historia de pérdida, idénticos bit a bit

$ bash -n scripts/prueba-caida.sh && bash scripts/check-skills-sync.sh
skills: ok

$ gh pr checks 37
Docker (imagen, clúster y caída de un nodo)	pass	1m56s
Ecosistema de agentes                       	pass	9s
Go (gofmt, vet, test -race)                 	pass	52s
```

| Chequeo | Resultado |
|---|---|
| gofmt / vet | ✅ |
| `go test -race ./...` | ✅ (carreras: 0; además `-count=5` verde en los 4 paquetes `cmd/` tocados) |
| Equivalencia secuencial = distribuido | ✅ (tests Go locales + CI Docker `-verificar` en corrida normal y con `docker kill`) |
| Latencia API p95 < 100 ms | no aplica (la API HTTP pertenece al issue #28) |

## Verificación contra los criterios de aceptación del Issue #27

| Criterio (#27) | Estado | Evidencia |
|---|---|---|
| `cmd/node` se conecta, reintenta y carga shards de volumen `:ro` | ✅ | `cmd/node/main.go:52-87`, `cmd/node/main_test.go`, `deploy/docker-compose.yml:56` (`:/datos:ro`); `go test -race -count=5 ./cmd/node` en verde. |
| `cmd/coordinator` escucha `:9000`, espera `-nodos N`, entrena y escribe `reports/entrenamiento_cluster.json` | ✅ | `cmd/coordinator/main.go:67-102,148-217`, `cmd/coordinator/main_test.go`; `go test -race -count=5 ./cmd/coordinator` en verde. |
| Imagen única con todos los `cmd/`, `nonroot`, < 40 MB | ✅ | `deploy/Dockerfile`, `.dockerignore`; CI job `Docker` (run `43634568981`): `usuario=nonroot:nonroot bytes=24024972` (24,0 MB). |
| `docker compose up --build --scale node=3` entrena y sale con código 0 | ✅ en CI (sintético) · ⚠️ pendiente en gold real | CI ejecuta Compose con 3 nodos sobre gold sintético `400 000 × 31` (`igual_al_secuencial: true`, código 0). `deploy/docker-compose.yml:50,58-59` no fija `container_name` y usa `${NODOS:-3}`, así que `--scale node=3` funciona directo. **Pendiente humano (Jorge):** correr en su PC con `data/gold` real antes de cerrar #27 / redactar PC4. |
| Pesos del clúster idénticos bit a bit a `reports/entrenamiento.json` | ✅ mecanismo · ⚠️ pendiente en gold real | `cmd/entrenar/main.go:81-95,279-328` (`compararReportes` con `math.Float64bits`, probado en `TestCompararReportesExigeElMismoModeloBitABit` y contra `reports/entrenamiento.json` real) + `-verificar` en verde en CI. **Pendiente humano (Jorge):** ejecutar `go run ./cmd/entrenar -comparar reports/entrenamiento_cluster.json` tras entrenar sobre el gold real. |
| Matar un nodo (`docker kill`) no aborta y da los mismos pesos | ✅ | `scripts/prueba-caida.sh` ejecutado en CI: `época 60: nodo b5f308fd26a2 descartado: conexión perdida: EOF`, `época 60: shard 2 reasignado a 4f5203b663ea`, `1000 épocas`, `reasignaciones: 1`, `verificado: pesos e historia idénticos bit a bit al secuencial`. |
| Paso de CI que construye la imagen | ✅ | `.github/workflows/ci.yml:60-95`, job `Docker (imagen, clúster y caída de un nodo)` en verde. |

## Puntaje de rúbrica en juego

| Criterio | Peso | Este PR lo deja en | Comentario |
|---|---|---|---|
| Planteamiento | 3 | no aplica | No modifica planteamiento ni EDA. |
| Implementación | 8 | Avanza PC4 (despliegue Docker, puertos y tolerancia a fallos del clúster) | Cumple `AGENTS.md` §3: stdlib puro (`go.mod` sin terceros), imagen distroless `nonroot` de 24 MB, Compose en red interna (`cluster`, puerto `9000` sin publicar al anfitrión), determinismo bit a bit verificado y prueba automatizada de `docker kill`. |
| Interfaz | 5 | no aplica | API y SPA corresponden a issues posteriores (#28–#30 y TB2). |
| Informe | 2 | Mantiene 2 / 2 | Sin cifras a mano; las cifras sintéticas del CI quedan declaradas fuera del informe. |
| Video | 2 | no aplica | Entregable TB2. |

## Hallazgos

Sin hallazgos bloqueantes ni mayores. Se dejan dos observaciones menores no bloqueantes:

### H1 — Validación defensiva de flags (`-nodos`, `-epocas`) y cabecera de `train.bin` en `-verificar` · MENOR

- **Dónde:** `cmd/coordinator/main.go:148-159`, `cmd/coordinator/main.go:193-194`, `cmd/coordinator/main.go:196-205`
- **Evidencia:** A diferencia de `cmd/gold/main.go:64`, `ejecutar()` no valida `op.nodos < 1` ni `op.epocas < 1`. Si se invocara con `-epocas 0`, `ml.EntrenarFalible` devuelve `len(res.Historia) == 0` y `cmd/coordinator/main.go:194` produce *panic* al indexar `res.Historia[0]` (y con `-epocas < 0`, `make([]float64, 0, e.Epocas)` falla en `internal/ml/logistica.go:304`). De forma similar, con `-verificar`, si `train.bin` en `-gold` tuviera distinto `D`, `N` o `TamBloque` que los shards (p. ej. un archivo de una corrida previa), `ml.Entrenar` haría *panic* en `validarPesos` (`internal/ml/logistica.go:248`) en lugar de devolver un error descriptivo.
- **Corrección sugerida:** Al inicio de `ejecutar()`, rechazar `op.nodos < 1 || op.epocas < 1` cerrando `ln`; y tras `gold.Leer(train.bin)` comprobar `train.D != d || train.N != len(y) || train.TamBloque != tam` antes de correr el secuencial. Puede atenderse aquí o en un PR de seguimiento.

### H2 — `signal.Ignore(SIGTERM)` solo se activa cuando `guardar` tiene éxito · MENOR

- **Dónde:** `cmd/coordinator/main.go:88-98`, `cmd/coordinator/main.go:197-201`, `cmd/coordinator/main.go:214-216`
- **Evidencia:** Si `escribirJSON(*salida, rep)` falla (escenario frecuente en Linux si se olvida `USUARIO="$(id -u):$(id -g)"` y `/reports` devuelve `permission denied`), `guardar` retorna error antes de llegar a `signal.Ignore(syscall.SIGTERM)` en la línea 96. Sin embargo, la línea 215 igualmente llama a `c.Terminar()`, lo que envía `fin` a los nodos; al salir el primer nodo, Compose con `--exit-code-from coordinator` envía `SIGTERM` al coordinador y podría matarlo (código 143) antes de que `log.Fatal(err)` imprima el mensaje de permisos. Lo mismo ocurre en la línea 199 cuando `-verificar` no encuentra `train.bin`.
- **Corrección sugerida:** Llamar a `signal.Ignore(syscall.SIGTERM)` antes de enviar `fin` (o evitar llamar a `c.Terminar()` cuando `guardar` falla, dejando que `defer c.Cerrar()` corte las conexiones después de que el coordinador registre el error). No bloquea el merge porque el código de salida sigue siendo distinto de 0 y en el camino exitoso `guardar` protege el cierre.

## Pronunciamiento sobre las decisiones consultadas en el PR

1. **Guardar reporte antes de `fin` + `signal.Ignore(SIGTERM)`:** De acuerdo (con la nota menor H2 para el caso de error al escribir). El orden queda cubierto por `TestElReporteSeGuardaAntesDeQueLosNodosRecibanFin`.
2. **`cmd/gold -sintetico` (31 variables, sin tocar `split.json` ni `esquema.json`):** De acuerdo. Replica la dimensión exacta del gold real (`variables: 31` en `reports/entrenamiento.json`) para validar Docker y CI sin versionar datos (`AGENTS.md` §3.7).
3. **`-verificar` en el coordinador:** De acuerdo. Permite probar equivalencia bit a bit dentro del propio contenedor distroless sin depender de Go en tiempo de ejecución (con la nota menor H1).
4. **Reconexión del nodo (`-reconexiones 3`) y no reintentar tras `ErrAbortado`:** De acuerdo. `Nodo.Servir` libera la conexión y el `ml.Pool` mediante `defer` antes de reintentar, evitando fugas de goroutines; cubierto por tests en `cmd/node/main_test.go`.
5. **Pesos por clase desde `a.Y` descartando `a.X`:** De acuerdo. Mantiene baja la memoria del coordinador y calcula los mismos `PesoPos`/`PesoNeg` que `train.bin`.
6. **`NODOS` frente a `--scale`:** De acuerdo. Fijar `-nodos=${NODOS:-3}` y `replicas: ${NODOS:-3}` sin `container_name` hace compatibles tanto `--scale node=3` como `NODOS=5`, y está claramente explicado en `deploy/README.md`.
7. **Permisos en Linux (`USUARIO=uid:gid`):** De acuerdo. Conserva `USER nonroot:nonroot` en la imagen y resuelve la escritura en el volumen montado `../reports` mediante `USUARIO` y los targets `make cluster` / `make prueba-caida`.
8. **Alcance del Compose (Mongo, Redis, API y SPA en #28–#30 y TB2):** De acuerdo. Mantiene el PR enfocado en #27.

**Nota para el merge (Jorge):** Este veredicto aprueba el código y la infraestructura de #27. Antes de dar por cerrado el issue en el informe PC4, ejecuta en tu PC con el gold real en `data/gold`:
```bash
docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator
go run ./cmd/entrenar -comparar reports/entrenamiento_cluster.json
bash scripts/prueba-caida.sh
```
(en Linux/WSL anteponiendo `USUARIO="$(id -u):$(id -g)"` o con `make cluster` y `make prueba-caida`).

## Seguimiento de rondas previas

No aplica (primera ronda de revisión del PR #37).

## Cómo explicarlo en la sustentación

Una sola imagen distroless sin privilegios (24 MB, binarios estáticos Go sin librerías externas) empaqueta todos los comandos del proyecto, y `docker-compose.yml` levanta el coordinador y *N* nodos de cómputo en una red interna con los shards montados en solo lectura. Los nodos reintentan la conexión TCP hasta que el coordinador abre el puerto `:9000`, y el coordinador guarda `reports/entrenamiento_cluster.json` antes de enviar `fin` para no interrumpirse bajo `--exit-code-from`. Si matamos un contenedor nodo con `docker kill`, el coordinador detecta el `EOF`, reasigna su shard en la misma época y produce pesos e historia idénticos bit a bit al entrenamiento secuencial.
