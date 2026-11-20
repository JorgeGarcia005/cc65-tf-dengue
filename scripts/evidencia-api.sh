#!/usr/bin/env bash
# Evidencia de los issues #28, #29 y #30: la API vive en el coordinador del clúster
# en Docker, con MongoDB y Redis, y se prueba desde el anfitrión. Requiere el gold real
# (scripts/evidencia-cluster.sh antes).
#
#   & "C:\Program Files\Git\bin\bash.exe" scripts/evidencia-api.sh
set -euo pipefail
cd "$(dirname "$0")/.."
out=evidencia/28
mkdir -p "$out"
dc=(docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml)
export PUERTO_API=${PUERTO_API:-18080}
url=http://localhost:$PUERTO_API
export API=:8080
paso() { printf '\n== %s\n' "$1"; }
trap '"${dc[@]}" down --remove-orphans >/dev/null 2>&1 || true' EXIT

{ paso "1. Clúster en modo servicio: API=:8080 docker compose (+ docker-compose.api.yml, puerto $PUERTO_API) up --build -d"
  git log -1 --oneline
  "${dc[@]}" up --build -d
  for _ in $(seq 1 120); do curl -fsS "$url/healthz" >/dev/null 2>&1 && break; sleep 1; done
  curl -sS -i "$url/healthz"; } 2>&1 | tee "$out/01-arranque.txt"

{ paso "2. Swagger UI y contrato desde el anfitrión"
  curl -sS -o /dev/null -w "GET /docs → %{http_code} (%{content_type})\n" "$url/docs"
  curl -sS -o /dev/null -w "GET /docs/openapi.yaml → %{http_code} (%{size_download} bytes)\n" "$url/docs/openapi.yaml"
  curl -sS "$url/docs/openapi.yaml" | head -15; } 2>&1 | tee "$out/02-docs.txt"

{ paso "3. Estado del clúster"
  for _ in $(seq 1 60); do curl -sS "$url/api/v1/cluster" | grep -q '"nombre"' && break; sleep 1; done
  curl -sS "$url/api/v1/cluster"; echo; } 2>&1 | tee "$out/03-cluster.txt"

{ paso "4. Entrenamiento distribuido vía API"
  curl -sS "$url/api/v1/modelo" | head -c 300; echo
  resp=$(curl -sS -i -X POST -H 'Content-Type: application/json' -d '{"epocas": 600}' "$url/api/v1/entrenamientos")
  echo "$resp"
  ubic=$(printf '%s' "$resp" | tr -d '\r' | awk -F': ' 'tolower($1)=="location"{print $2}')
  for _ in $(seq 1 600); do
    e=$(curl -sS "$url$ubic"); echo "$e" | grep -q '"estado":"en_curso"' || break; sleep 2
  done
  echo "$ubic" > "$out/.entrenamiento"
  echo "GET $ubic"; echo "$e"
  curl -sS "$url/api/v1/modelo" | head -c 300; echo; } 2>&1 | tee "$out/04-entrenamiento.txt"

{ paso "5. Predicción con el modelo publicado"
  curl -sS -w "\n(%{time_total} s)\n" -X POST -H 'Content-Type: application/json' \
    -d '{"departamento":"LORETO","ano":2024,"semana":10,"edad":30,"tipo_edad":"A","sexo":"F"}' "$url/api/v1/predicciones"
} 2>&1 | tee "$out/05-prediccion.txt"

ent_id=$(basename "$(cat "$out/.entrenamiento")")
{ paso "6. MongoDB y Redis: cadena del entrenamiento $ent_id (issue #29)"
  # Segunda consulta igual: debe salir de la caché de Redis.
  curl -sS -o /dev/null -w "repetida: %{http_code} en %{time_total} s\n" -X POST -H 'Content-Type: application/json' \
    -d '{"departamento":"LORETO","ano":2024,"semana":10,"edad":30,"tipo_edad":"A","sexo":"F"}' "$url/api/v1/predicciones"
  "${dc[@]}" logs coordinator | grep -E 'MongoDB|Redis' || true
  bash scripts/evidencia-bd.sh "$ent_id"; } 2>&1 | tee "$out/06-bd.txt"

{ paso "7. Latencia de la API con carga concurrente (issue #30): cmd/benchapi -exigir-p95 100ms"
  go run ./cmd/benchapi -url "$url" -exigir-p95 100ms
  if command -v npx >/dev/null 2>&1; then
    paso "7b. Colección de Postman con Newman (tests de estado y < 100 ms)"
    npx --yes newman run docs/api/cc65-tf.postman_collection.json --env-var "base_url=$url" || true
  else
    echo "(sin Node/npx: la colección se corre desde Postman con base_url=$url)"
  fi; } 2>&1 | tee "$out/07-latencia.txt"

{ paso "8. Cierre con SIGTERM: docker compose stop coordinator"
  "${dc[@]}" stop coordinator
  "${dc[@]}" ps -a
  "${dc[@]}" logs coordinator | tail -25; } 2>&1 | tee "$out/08-cierre.txt"

paso "Listo: evidencia en $out/"
