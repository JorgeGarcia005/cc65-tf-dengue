#!/usr/bin/env bash
# Dibuja los diagramas UML del informe TB2 con Graphviz:
# docs/informe/tb2/diagramas/<nombre>.dot → docs/informe/tb2/fig/<nombre>.svg
# Los SVG se versionan, así que regenerar el informe no exige Graphviz; este
# script solo hace falta al cambiar un diagrama. Falla si un SVG no coincide
# con su .dot cuando se llama con --verificar (con la misma versión de
# Graphviz: otra versión cambia detalles del SVG).
set -euo pipefail
cd "$(dirname "$0")/.."
command -v dot >/dev/null || { echo "falta Graphviz (dot): apt install graphviz / winget install graphviz" >&2; exit 1; }
# El SVG se genera con cairo: el texto queda como trazos (glifos), así el
# diagrama se ve igual en Word, LibreOffice y el navegador aunque no tengan
# la fuente. La fuente es Liberation Serif, con las mismas métricas que Times
# New Roman (apt install fonts-liberation).
fc-match "Liberation Serif" 2>/dev/null | grep -q Liberation || { echo "falta la fuente Liberation Serif (apt install fonts-liberation)" >&2; exit 1; }
verificar=false
[[ "${1:-}" == "--verificar" ]] && verificar=true
estado=0
for fuente in docs/informe/tb2/diagramas/*.dot; do
  nombre=$(basename "$fuente" .dot)
  destino=docs/informe/tb2/fig/$nombre.svg
  if $verificar; then
    if ! cmp -s <(dot -Tsvg:cairo "$fuente") "$destino"; then
      echo "desactualizado: $destino (corra make figuras)" >&2
      estado=1
    fi
  else
    dot -Tsvg:cairo "$fuente" -o "$destino"
    echo "escrito $destino"
  fi
done
exit $estado
