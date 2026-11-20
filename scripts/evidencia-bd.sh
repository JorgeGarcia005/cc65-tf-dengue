#!/usr/bin/env bash
# Evidencia para el informe (issue #29): verifica en MongoDB y Redis la cadena
# completa de UN entrenamiento (PR-40 H4):
#
#   entrenamiento terminado → su version_modelo → el modelo con ese _id y
#   entrenamiento_id → predicciones de esa versión → claves pred:<versión>:*
#   en Redis con TTL positivo.
#
# Además muestra cuántos documentos e índices tiene cada colección.
#
# Uso (con las bases levantadas y después de entrenar y predecir con la API):
#   bash scripts/evidencia-bd.sh ent-20261117T183000-1a2b3c4d5e6f7a8b
#   bash scripts/evidencia-bd.sh            # el último entrenamiento terminado
#
# Sale con 1 si algún eslabón falta. La evidencia del informe debe venir de un
# entrenamiento del clúster en Compose, no de los tests (que usan otra base).
set -euo pipefail
cd "$(dirname "$0")/.."
# Las bases del despliegue (modo servicio del clúster, scripts/evidencia-api.sh).
COMPOSE=(docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml)
BASE=${BASE:-cc65_dengue}
ID=${1:-}
if [[ -n "$ID" && ! "$ID" =~ ^[A-Za-z0-9-]+$ ]]; then
  echo "ID inválido: $ID" >&2
  exit 2
fi

echo "== MongoDB ($BASE)"
salida=$("${COMPOSE[@]}" exec -T mongo mongosh --quiet "$BASE" --eval "
const pedido = '$ID';
for (const c of ['modelos', 'entrenamientos', 'predicciones']) {
  if (!db.getCollectionNames().includes(c)) { print('-- ' + c + ': (no existe)'); continue; }
  const col = db.getCollection(c);
  print('-- ' + c + ': ' + col.countDocuments() + ' documentos; índices ' +
    col.getIndexes().map(i => JSON.stringify(i.key) + (i.expireAfterSeconds ? ' TTL ' + i.expireAfterSeconds + 's' : '')).join(', '));
}
const ent = pedido
  ? db.entrenamientos.findOne({_id: pedido})
  : db.entrenamientos.find({estado: 'terminado'}).sort({fin: -1}).limit(1).toArray()[0];
if (!ent) { print('FALTA: entrenamiento ' + (pedido || 'terminado')); quit(1); }
print('== Cadena del entrenamiento ' + ent._id);
print('   estado ' + ent.estado + ', ' + ent.epoca + '/' + ent.epocas + ' épocas, pérdida final ' + ent.perdida +
  ', ' + (ent.nodos || []).length + ' nodos, ' + ent.reasignaciones + ' reasignaciones');
if (ent.estado !== 'terminado' || !ent.version_modelo) { print('FALTA: el entrenamiento no terminó con un modelo'); quit(1); }
const v = ent.version_modelo;
const mod = db.modelos.findOne({_id: v});
if (!mod || mod.entrenamiento_id !== ent._id) { print('FALTA: modelo ' + v + ' guardado por ' + ent._id); quit(1); }
print('   modelo ' + v + ' publicado ' + mod.publicado.toISOString());
const n = db.predicciones.countDocuments({version_modelo: v});
const hits = db.predicciones.countDocuments({version_modelo: v, desde_cache: true});
if (n === 0) { print('FALTA: predicciones con la versión ' + v); quit(1); }
print('   predicciones con esa versión: ' + n + ' (' + hits + ' desde la caché)');
print('VERSION=' + v);
") || { echo "$salida"; exit 1; }
echo "$salida" | grep -v '^VERSION='
version=$(echo "$salida" | sed -n 's/^VERSION=//p')

echo "== Redis"
claves=$("${COMPOSE[@]}" exec -T redis redis-cli --scan --pattern "pred:${version}:*")
n=$(printf '%s' "$claves" | grep -c . || true)
if [ "$n" -eq 0 ]; then
  echo "FALTA: claves pred:${version}:* en Redis"
  exit 1
fi
primera=$(printf '%s\n' "$claves" | head -1)
ttl=$("${COMPOSE[@]}" exec -T redis redis-cli TTL "$primera" | tr -d '\r')
echo "   claves pred:${version}:*: $n; $primera con TTL ${ttl} s"
if [ "$ttl" -le 0 ]; then
  echo "FALTA: TTL positivo en $primera"
  exit 1
fi
echo "OK: cadena completa para la versión $version"
