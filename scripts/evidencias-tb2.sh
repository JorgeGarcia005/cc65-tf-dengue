#!/usr/bin/env bash
# Evidencias del informe TB2 (#70): ejecuta cada paso de la guía
# docs/guias/jose-70.md y guarda en docs/informe/tb2/evidencias/<paso>.txt lo
# que el informe muestra, con una cabecera (fecha, máquina, commit) y, solo si
# el paso salió bien, la última línea «# resultado: OK». Cuando la salida
# completa es larga (logs de Docker), va aparte en <paso>.log y el .txt trae
# las líneas que importan. cmd/word incrusta los .txt en el informe y
# scripts/paso70.sh los usa para no dejar avanzar al paso siguiente.
#
# Las evidencias NUNCA se editan a mano: si un paso falla, se corrige la causa
# y se vuelve a correr el mismo paso.
#
# Uso (desde la raíz del repo, en WSL o Git Bash, con Docker Desktop):
#   bash scripts/evidencias-tb2.sh <paso>
# Pasos, en orden: entorno parte-a cluster caida servicio entrenamiento
#                  benchapi bd evaluacion spin pruebas … informe; y apagar
#
# Variables: PUERTO_API (8080), PUERTO_WEB (8081), NODOS (3), QUITAR (texto de
# la carátula que cmd/word borra, p. ej. el nombre de un exintegrante).
set -euo pipefail
cd "$(dirname "$0")/.."

EVID=docs/informe/tb2/evidencias
mkdir -p "$EVID" reports data
PUERTO_API=${PUERTO_API:-8080}
PUERTO_WEB=${PUERTO_WEB:-8081}
export NODOS=${NODOS:-3}
URL=http://localhost:$PUERTO_API
BASE=(docker compose --ansi never -f deploy/docker-compose.yml)
SERV=(docker compose --ansi never -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml)
# En Linux/WSL los contenedores escriben en data/ y reports/ con el usuario propio.
if [[ "$(uname -s)" == Linux ]]; then export USUARIO="${USUARIO:-$(id -u):$(id -g)}"; fi

falla() { echo "ERROR: $*" >&2; exit 1; }

# cabecera <paso> <descripción>
cabecera() {
  echo "# evidencia: $1 — $2"
  echo "# fecha: $(date '+%Y-%m-%d %H:%M:%S %z')"
  echo "# máquina: $(hostname) ($(uname -sm))"
  echo "# commit: $(git rev-parse --short HEAD) ($(git rev-parse --abbrev-ref HEAD))"
  echo
}

# registrar <paso> <descripción> <función>: corre la función, guarda todo en
# evidencias/<paso>.txt y agrega «# resultado: OK» solo si terminó bien.
registrar() {
  local paso=$1 desc=$2 fn=$3 archivo="$EVID/$1.txt" tmp
  tmp=$(mktemp)
  cabecera "$paso" "$desc" >"$tmp"
  echo "== $desc"
  # Fuera de un «if»: así set -e sí detiene la función en la primera orden que
  # falle (dentro de la condición de un if, bash lo desactiva).
  set +e
  ( set -eo pipefail; "$fn" ) 2>&1 | tee -a "$tmp"
  local estado=${PIPESTATUS[0]}
  set -e
  if [[ $estado -eq 0 ]]; then
    echo "# resultado: OK" >>"$tmp"
    mv "$tmp" "$archivo"
    echo "== OK → $archivo"
  else
    echo "# resultado: FALLÓ" >>"$tmp"
    mv "$tmp" "$archivo"
    falla "el paso «$paso» falló; revisa $archivo, corrige la causa y vuelve a correrlo"
  fi
}

# Variables de deploy/.env sin ejecutarlo (puede tener caracteres de shell).
env_de() { sed -n "s/^$1=//p" deploy/.env | tail -1 | tr -d '\r'; }

login() {
  curl -sS -X POST -H 'Content-Type: application/json' \
    -d "{\"correo\":\"$1\",\"clave\":\"$2\"}" "$URL/api/v1/auth/login" |
    sed -n 's/.*"token":"\([^"]*\)".*/\1/p'
}

# --- Pasos ---------------------------------------------------------------------

p_entorno() {
  echo "-- herramientas"
  go version
  echo "node $(node -v)"
  docker version --format 'docker cliente {{.Client.Version}}, servidor {{.Server.Version}} ({{.Server.Os}}/{{.Server.Arch}})'
  docker compose version
  echo "-- CPU"
  if command -v lscpu >/dev/null; then lscpu | grep -E 'Model name|^CPU\(s\)'; else echo "$PROCESSOR_IDENTIFIER, $NUMBER_OF_PROCESSORS hilos"; fi
  echo "-- dataset (sha256 contra data/MANIFEST.sha256)"
  sha256sum -c data/MANIFEST.sha256
  echo "-- rama y último commit"
  git log -1 --format='%h %s (%an)'
}

p_parte_a() {
  echo "-- construcción de la imagen cc65-tf-datos (salida en parte-a.log)"
  "${BASE[@]}" --profile datos build loader >"$EVID/parte-a.log" 2>&1
  echo "-- docker compose --profile datos up entrenar (loader → gold → entrenar)"
  "${BASE[@]}" --profile datos up --no-build entrenar >>"$EVID/parte-a.log" 2>&1 || true
  local fallos=0 c
  for s in loader gold entrenar; do
    c=$("${BASE[@]}" --profile datos ps -a --format '{{.ExitCode}}' "$s" | tail -1)
    echo
    echo "-- $s (código de salida: $c; últimas líneas de su log)"
    "${BASE[@]}" --profile datos logs --no-color --no-log-prefix --tail 12 "$s"
    [[ "$c" == 0 ]] || fallos=1
  done
  echo
  echo "-- archivos generados"
  ls -l data/silver/dengue.csv data/gold/train.bin data/gold/test.bin data/gold/modelo.json data/gold/shard_*.bin
  return $fallos
}

p_cluster() {
  "${BASE[@]}" down --remove-orphans >/dev/null 2>&1 || true
  "${BASE[@]}" build coordinator >"$EVID/cluster.log" 2>&1
  echo "-- NODOS=$NODOS VERIFICAR=true docker compose up --exit-code-from coordinator (log completo en cluster.log)"
  VERIFICAR=true "${BASE[@]}" up --no-build --exit-code-from coordinator >>"$EVID/cluster.log" 2>&1
  echo "-- log del coordinador"
  "${BASE[@]}" logs --no-color --no-log-prefix coordinator |
    grep -E "escuchando|shards|registr|época|epoca|entrenamiento distribuido|verificado|escrito" | head -60
  "${BASE[@]}" logs --no-color coordinator | grep -q "verificado: pesos e historia idénticos" ||
    { echo "no aparece «verificado» en el log del coordinador"; return 1; }
  "${BASE[@]}" down --remove-orphans >/dev/null 2>&1
}

p_caida() {
  echo "-- bash scripts/prueba-caida.sh (log completo en caida.log)"
  bash scripts/prueba-caida.sh >"$EVID/caida.log" 2>&1 || { tail -n 40 "$EVID/caida.log"; return 1; }
  tail -n 30 "$EVID/caida.log"
}

p_servicio() {
  [[ -f deploy/.env ]] || { echo "falta deploy/.env (cp deploy/.env.example deploy/.env y completarlo)"; return 1; }
  [[ -n "$(env_de JWT_SECRETO)" ]] || { echo "deploy/.env sin JWT_SECRETO"; return 1; }
  "${SERV[@]}" down --remove-orphans >/dev/null 2>&1 || true
  API=:8080 "${SERV[@]}" build >"$EVID/servicio.log" 2>&1
  API=:8080 DEMO=true PUERTO_API=$PUERTO_API PUERTO_WEB=$PUERTO_WEB "${SERV[@]}" up --no-build -d >>"$EVID/servicio.log" 2>&1
  echo "-- esperando a que los $NODOS nodos se registren"
  for _ in $(seq 1 180); do
    "${SERV[@]}" logs --no-color coordinator 2>/dev/null | grep -q "la API acepta entrenamientos" && break
    sleep 1
  done
  "${SERV[@]}" logs --no-color coordinator | grep -q "la API acepta entrenamientos" ||
    { echo "los nodos no se registraron en 3 minutos"; "${SERV[@]}" logs --no-color coordinator; return 1; }
  echo "-- docker compose ps"
  "${SERV[@]}" ps --format 'table {{.Service}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
  echo "-- log de arranque del coordinador"
  "${SERV[@]}" logs --no-color --no-log-prefix coordinator | head -40
  echo "-- GET /healthz"
  curl -sS -i "$URL/healthz" | tr -d '\r'
}

p_entrenamiento() {
  local admin resp ubic e version
  admin=$(login "$(env_de VIGIA_ADMIN_CORREO)" "$(env_de VIGIA_ADMIN_CLAVE)")
  [[ -n "$admin" ]] || { echo "no se pudo iniciar sesión como admin (¿servicio arriba? ¿deploy/.env?)"; return 1; }
  echo "login admin: token recibido (no se muestra)"
  local auth=(-H "Authorization: Bearer $admin")
  echo "-- POST /api/v1/entrenamientos (mismos parámetros que el modelo inicial)"
  resp=$(curl -sS -i -X POST "${auth[@]}" -H 'Content-Type: application/json' \
    -d '{"epocas":600,"tasa":2,"lambda":0.0001}' "$URL/api/v1/entrenamientos" | tr -d '\r')
  echo "$resp"
  ubic=$(printf '%s\n' "$resp" | awk -F': ' 'tolower($1)=="location"{print $2}')
  [[ -n "$ubic" ]] || { echo "la API no devolvió Location"; return 1; }
  for _ in $(seq 1 900); do
    e=$(curl -sS "${auth[@]}" "$URL$ubic")
    grep -q '"estado":"en_curso"' <<<"$e" || break
    sleep 2
  done
  echo "-- GET $ubic (sin la historia de la pérdida)"
  sed 's/"historia":\[[^]]*\]/"historia":[…]/' <<<"$e"
  echo
  grep -q '"estado":"terminado"' <<<"$e" || { echo "el entrenamiento no terminó bien"; return 1; }
  version=$(curl -sS "$URL/healthz" | sed -n 's/.*"version_modelo":"\([^"]*\)".*/\1/p')
  echo "version_modelo publicada: $version"
}

p_benchapi() {
  BENCH_CORREO=$(env_de VIGIA_ANALISTA_CORREO) BENCH_CLAVE=$(env_de VIGIA_ANALISTA_CLAVE) \
    go run ./cmd/benchapi -url "$URL" -maquina deez -exigir-p95 100ms -salida reports/bench_api_deez_jwt.json
}

p_bd() {
  bash scripts/evidencia-bd.sh
}

p_evaluacion() {
  go run ./cmd/evaluacion -api reports/bench_api_deez_jwt.json
  local api arch
  api=$(curl -sS "$URL/healthz" | sed -n 's/.*"version_modelo":"\([^"]*\)".*/\1/p')
  arch=$(sed -n 's/.*"version_modelo": *"\([^"]*\)".*/\1/p' reports/evaluacion.json | head -1)
  echo "versión del modelo en la API: $api"
  echo "versión evaluada (reports/evaluacion.json): $arch"
  [[ -n "$api" && "$api" == "$arch" ]] || { echo "la evaluación no es del modelo que sirve la API"; return 1; }
}

p_spin() {
  bash spin/verificar.sh
}

p_pruebas() {
  echo "-- go test -race ./... (requiere gcc: en WSL)"
  go test -race -count=1 ./... 2>&1 | grep -v 'no test files'
  echo "-- SPA: tipos y tests (Vitest)"
  ( cd web && npm ci --no-audit --no-fund >/dev/null && npx tsc -b && npx vitest run )
}

p_informe() {
  echo "-- go run ./cmd/informe (cifras, figuras y fragmentos desde reports/)"
  go run ./cmd/informe 2>&1 | grep -v "^.* escrito "
  echo "-- el informe coincide con el repositorio y no tiene referencias rotas"
  go test -count=1 ./cmd/informe ./cmd/word
  echo "-- go run ./cmd/word -final (falla si falta una captura o una evidencia)"
  if [[ -n "${QUITAR:-}" ]]; then go run ./cmd/word -final -quitar "$QUITAR"; else go run ./cmd/word -final; fi
  ls -l entregables/1ACC0065_TB2_*.docx
}

p_apagar() {
  "${SERV[@]}" down --remove-orphans
}

paso=${1:-}
case "$paso" in
  entorno)       registrar entorno "Herramientas, CPU y dataset" p_entorno ;;
  parte-a)       registrar parte-a "Parte A en Docker: loader → gold → entrenar" p_parte_a ;;
  cluster)       registrar cluster "Entrenamiento único en el clúster con VERIFICAR=true" p_cluster ;;
  caida)         registrar caida "Caída de un nodo a mitad del entrenamiento" p_caida ;;
  servicio)      registrar servicio "Modo servicio: contenedores y arranque del coordinador" p_servicio ;;
  entrenamiento) registrar entrenamiento "Entrenamiento distribuido lanzado por la API" p_entrenamiento ;;
  benchapi)      registrar benchapi "Latencia de la API con JWT (cmd/benchapi)" p_benchapi ;;
  bd)            registrar bd "Cadena del entrenamiento en MongoDB y Redis" p_bd ;;
  evaluacion)    registrar evaluacion "Evaluación del modelo en el conjunto de prueba" p_evaluacion ;;
  spin)          registrar spin "Verificación del protocolo con Spin" p_spin ;;
  pruebas)       registrar pruebas "Pruebas de Go (-race) y de la SPA" p_pruebas ;;
  informe)       registrar informe "Informe final: cmd/informe y cmd/word -final" p_informe ;;
  apagar)        p_apagar ;;
  *) echo "uso: bash scripts/evidencias-tb2.sh <entorno|parte-a|cluster|caida|servicio|entrenamiento|benchapi|bd|evaluacion|spin|pruebas|informe|apagar>" >&2; exit 2 ;;
esac
