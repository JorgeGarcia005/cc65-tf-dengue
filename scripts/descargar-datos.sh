#!/usr/bin/env bash
# Descarga el dataset de dengue del Portal Nacional de Datos Abiertos y verifica su sha256.
# La primera vez registra el checksum en data/MANIFEST.sha256 (que sí se versiona);
# las siguientes, falla si el archivo cambió en el portal.
set -euo pipefail

raiz="$(cd "$(dirname "$0")/.." && pwd)"
url="https://www.datosabiertos.gob.pe/sites/default/files/datos_abiertos_vigilancia_dengue_2000_2025_0.csv"
dir="$raiz/data/bronze"
archivo="$dir/vigilancia_dengue_2000_2025.csv"
manifiesto="$raiz/data/MANIFEST.sha256"

mkdir -p "$dir"

if [ ! -s "$archivo" ]; then
  echo "Descargando $url"
  curl -fL --retry 3 --retry-delay 5 -o "$archivo.part" "$url"
  mv -f "$archivo.part" "$archivo"
fi

suma="$(sha256sum "$archivo" | awk '{print $1}')"
filas="$(($(wc -l < "$archivo") - 1))"
rel="data/bronze/$(basename "$archivo")"

if [ -f "$manifiesto" ] && grep -q " $rel\$" "$manifiesto"; then
  esperado="$(grep " $rel\$" "$manifiesto" | awk '{print $1}')"
  if [ "$suma" != "$esperado" ]; then
    echo "ERROR: el sha256 no coincide con data/MANIFEST.sha256"
    echo "  esperado: $esperado"
    echo "  obtenido: $suma"
    echo "El portal pudo actualizar el archivo. Revísalo y actualiza el manifiesto en un PR."
    exit 1
  fi
  echo "OK: $rel ($filas filas) coincide con el manifiesto"
else
  echo "$suma  $rel" >> "$manifiesto"
  echo "Registrado en data/MANIFEST.sha256: $rel ($filas filas). Versiona ese archivo."
fi
