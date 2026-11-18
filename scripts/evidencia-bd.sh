#!/usr/bin/env bash
# Evidencia para el informe (issue #29): qué hay en MongoDB y en Redis después
# de usar la API. Muestra, por colección, el número de documentos, los índices
# y el documento más reciente; y en Redis las claves pred:* con su TTL.
#
# Uso (con las bases levantadas: docker compose -f deploy/docker-compose.bd.yml up -d):
#   bash scripts/evidencia-bd.sh
set -euo pipefail
cd "$(dirname "$0")/.."
COMPOSE=(docker compose -f deploy/docker-compose.bd.yml)
BASE=${BASE:-cc65_dengue}

echo "== MongoDB ($BASE)"
"${COMPOSE[@]}" exec -T mongo mongosh --quiet "$BASE" --eval '
for (const c of ["modelos", "entrenamientos", "predicciones"]) {
  const col = db.getCollection(c);
  print(`-- ${c}: ${col.countDocuments()} documentos`);
  if (!db.getCollectionNames().includes(c)) { print("   (la colección aún no existe)"); continue; }
  print("   índices: " + col.getIndexes().map(i => JSON.stringify(i.key) + (i.expireAfterSeconds ? ` TTL ${i.expireAfterSeconds}s` : "")).join(", "));
  const orden = c === "predicciones" ? {fecha: -1} : c === "modelos" ? {publicado: -1} : {inicio: -1};
  const ult = col.find({}, {pesos: 0, esquema: 0, historia: 0}).sort(orden).limit(1).toArray();
  if (ult.length) printjson(ult[0]);
}
const fallar = db.getCollection("modelos").countDocuments() === 0 || db.getCollection("predicciones").countDocuments() === 0;
quit(fallar ? 1 : 0);
' || { echo "FALTA: no hay modelos o predicciones en Mongo"; estado=1; }

echo "== Redis"
claves=$("${COMPOSE[@]}" exec -T redis redis-cli --scan --pattern 'pred:*' | head -20)
n=$("${COMPOSE[@]}" exec -T redis redis-cli --scan --pattern 'pred:*' | wc -l)
echo "-- claves pred:*: $n"
for k in $(echo "$claves" | head -5); do
  echo "   $k  (TTL $("${COMPOSE[@]}" exec -T redis redis-cli TTL "$k") s)"
done
[ "$n" -gt 0 ] || { echo "FALTA: no hay claves pred:* en Redis"; estado=1; }
exit "${estado:-0}"
