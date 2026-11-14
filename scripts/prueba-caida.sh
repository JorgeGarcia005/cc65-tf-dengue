#!/usr/bin/env bash
# Prueba de tolerancia a fallos en Docker: levanta el clúster, mata un nodo a
# mitad del entrenamiento y exige que el coordinador termine bien, que haya
# reasignado el shard del nodo caído y que los pesos sean idénticos bit a bit a
# los del entrenamiento secuencial (-verificar).
#
#   bash scripts/prueba-caida.sh
#
# Variables: NODOS (3), EPOCAS (600), MATAR_EN (época tras la que cae el nodo, 50),
# GOLD (carpeta con los shards) y las demás de deploy/docker-compose.yml.
set -euo pipefail

raiz="$(cd "$(dirname "$0")/.." && pwd)"
dc=(docker compose -f "$raiz/deploy/docker-compose.yml")
export NODOS="${NODOS:-3}" EPOCAS="${EPOCAS:-600}" VERIFICAR=true
matar_en="${MATAR_EN:-50}"
export PROGRESO="$matar_en"
[ "$NODOS" -ge 2 ] || { echo "ERROR: hacen falta al menos 2 nodos para sobrevivir a una caída"; exit 1; }

limpiar() { "${dc[@]}" down --remove-orphans >/dev/null 2>&1 || true; }
trap limpiar EXIT
limpiar
mkdir -p "$raiz/reports"

echo "== Levantando el coordinador y $NODOS nodos ($EPOCAS épocas)"
"${dc[@]}" up --build -d

coordinador="$("${dc[@]}" ps -aq coordinator)"
echo "== Esperando a la época $matar_en"
for _ in $(seq 1 600); do
  avance="$("${dc[@]}" logs coordinator 2>&1 || true)"
  if grep -q "época $matar_en:" <<<"$avance"; then break; fi
  if [ "$(docker inspect -f '{{.State.Running}}' "$coordinador")" != "true" ]; then
    "${dc[@]}" logs coordinator | tail -20
    echo "ERROR: el coordinador terminó antes de la época $matar_en"; exit 1
  fi
  sleep 0.5
done

nodos_vivos="$("${dc[@]}" ps -q node)"
victima="$(head -1 <<<"$nodos_vivos")"
nombre="$(docker inspect -f '{{.Name}}' "$victima")"
echo "== docker kill $nombre"
docker kill "$victima" >/dev/null

echo "== Esperando al coordinador"
codigo="$(docker wait "$coordinador")"
registro="$("${dc[@]}" logs --no-log-prefix coordinator 2>&1)"
grep -E "registrado|descartado|reasignado|entrenamiento|verificado|abortado|escrito" <<<"$registro" || true

fallo=0
exigir() { grep -q "$1" <<<"$registro" || { echo "ERROR: falta en el log del coordinador: $2"; fallo=1; }; }
[ "$codigo" = "0" ] || { echo "ERROR: el coordinador salió con código $codigo"; fallo=1; }
exigir "descartado: conexión perdida" "el descarte del nodo caído (¿terminó el entrenamiento antes de la caída? Sube EPOCAS)"
exigir "reasignado a" "la reasignación del shard del nodo caído"
exigir "verificado: pesos e historia idénticos bit a bit" "la verificación contra el secuencial"
[ "$fallo" = "0" ] || exit 1

echo "OK: cayó $nombre en la época $matar_en o después, su shard se reasignó en la misma época"
echo "    y los pesos finales son idénticos bit a bit a los del entrenamiento secuencial."
echo "    Reporte: reports/entrenamiento_cluster.json"
