#!/usr/bin/env bash
# Revalida en esta máquina los reportes versionados (issue #23, AGENTS.md §3.3 y §3.6).
#
# 1. Verifica el sha256 del CSV contra data/MANIFEST.sha256.
# 2. Copia la referencia (reports/*.json de HEAD) a una carpeta temporal.
# 3. Regenera perfil → loader → gold → entrenar con los valores por defecto.
# 4. Compara campo por campo con cmd/revalidar.
# 5. Entrena con -workers 1 y -workers 16 y exige pesos idénticos bit a bit.
# 6. Restaura reports/ y data/gold/modelo.json: nunca sobrescribe lo aprobado.
#
# Uso: bash scripts/revalidar.sh   (desde la raíz del repo, con el CSV en data/bronze/)
set -euo pipefail
cd "$(dirname "$0")/.."

tmp=$(mktemp -d)
trap 'git restore --source=HEAD -- reports/ 2>/dev/null || true; rm -rf "$tmp"' EXIT

echo "== 1. sha256 del CSV"
sha256sum -c data/MANIFEST.sha256

echo "== 2. referencia de HEAD ($(git rev-parse --short HEAD))"
mkdir -p "$tmp/ref"
for f in perfil_bronze limpieza split entrenamiento; do
  git show "HEAD:reports/$f.json" > "$tmp/ref/$f.json"
done
cp data/gold/modelo.json "$tmp/modelo.json" 2>/dev/null || true

echo "== 3. pipeline completo"
go run ./cmd/perfil
go run ./cmd/loader
go run ./cmd/gold
go run ./cmd/entrenar
echo "-- git diff --stat reports/"
git diff --stat -- reports/

echo "== 4. comparación campo por campo"
estado=0
go run ./cmd/revalidar -ref "$tmp/ref" -nuevo reports || estado=1

echo "== 5. entrenar -workers 1 vs -workers 16"
mkdir -p "$tmp/w1" "$tmp/w16"
go run ./cmd/entrenar -workers 1  -salida "$tmp/w1/entrenamiento.json"  -modelo "$tmp/w1/modelo.json"
go run ./cmd/entrenar -workers 16 -salida "$tmp/w16/entrenamiento.json" -modelo "$tmp/w16/modelo.json"
go run ./cmd/revalidar -ref "$tmp/w1" -nuevo "$tmp/w16" entrenamiento.json || estado=1

[ -f "$tmp/modelo.json" ] && cp "$tmp/modelo.json" data/gold/modelo.json
echo "== máquina: $(go env GOOS)/$(go env GOARCH), $(go version | awk '{print $3}'), $(nproc) CPU lógicas"
if [ $estado -eq 0 ]; then echo "RESULTADO: reproducible"; else echo "RESULTADO: hay diferencias (abrir issue fix/ con esta salida)"; fi
exit $estado
