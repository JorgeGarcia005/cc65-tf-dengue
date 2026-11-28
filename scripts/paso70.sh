#!/usr/bin/env bash
# Control de la guía docs/guias/jose-70.md (#70): revisa, EN ORDEN, cada paso
# y se detiene en el primero que no está hecho, con la instrucción exacta para
# hacerlo. Un paso solo cuenta si su evidencia salió bien («# resultado: OK»)
# y es POSTERIOR a la del paso anterior: correr los pasos desordenados, o
# repetir uno sin repetir los siguientes, hace que este control lo marque.
#
#   bash scripts/paso70.sh          # dónde estoy y qué sigue
#
# No modifica nada: solo lee. Sale con 0 cuando todo está completo.
set -uo pipefail
cd "$(dirname "$0")/.."

EVID=docs/informe/tb2/evidencias
CAPT=docs/informe/tb2/capturas
CORREO_JORGE=jg1536454@gmail.com
total=0
anterior=""     # archivo de la evidencia del paso anterior (para el orden)

ok()    { printf '  [✓] %s\n' "$1"; }
alto()  { # alto <n> <título> <motivo> <instrucciones…>
  local n=$1 titulo=$2 motivo=$3; shift 3
  printf '\n  [✗] Paso %s: %s\n\n  Falta: %s\n\n  Qué hacer (docs/guias/jose-70.md, paso %s):\n' "$n" "$titulo" "$motivo" "$n"
  for l in "$@"; do printf '    %s\n' "$l"; done
  printf '\n  No sigas con el paso siguiente hasta que este control marque [✓] el paso %s.\n' "$n"
  exit 1
}
mtime() { stat -c %Y "$1" 2>/dev/null || echo 0; }

# evidencia <n> <paso> <título> <orden>: el paso cuenta si su archivo terminó
# en «# resultado: OK» y es posterior al del paso anterior.
evidencia() {
  local n=$1 paso=$2 titulo=$3 orden=$4 f="$EVID/$2.txt"
  total=$((total + 1))
  [[ -f "$f" ]] || alto "$n" "$titulo" "no existe $f" "$orden"
  [[ "$(tail -1 "$f")" == "# resultado: OK" ]] ||
    alto "$n" "$titulo" "$f no terminó bien (ábrelo: el error está al final)" "Corrige la causa (sin editar el archivo) y vuelve a correr:" "$orden"
  if [[ -n "$anterior" && $(mtime "$f") -lt $(mtime "$anterior") ]]; then
    alto "$n" "$titulo" "$f es anterior a $anterior: se corrió fuera de orden" "Vuelve a correr este paso y luego todos los siguientes, en orden:" "$orden"
  fi
  anterior=$f
  ok "Paso $n: $titulo"
}

echo "Guía #70 — control de avance ($(date '+%Y-%m-%d %H:%M'))"

# --- Paso 1: rama, identidad y archivos locales -------------------------------
git fetch -q origin 2>/dev/null || true
rama=$(git rev-parse --abbrev-ref HEAD)
[[ "$rama" == feature/70-evidencias ]] ||
  alto 1 "Preparar la rama" "estás en la rama «$rama»" \
    "git fetch origin" "git switch -c feature/70-evidencias origin/integracion/tb2"
git merge-base --is-ancestor origin/integracion/tb2 HEAD ||
  alto 1 "Preparar la rama" "tu rama no tiene lo último de origin/integracion/tb2" \
    "git merge origin/integracion/tb2      (sin rebase; luego vuelve a correr este control)"
correo=$(git config user.email || true)
[[ -n "$correo" && "$correo" != "$CORREO_JORGE" ]] ||
  alto 1 "Preparar la rama" "git no tiene tu correo de GitHub (los commits deben salir a tu nombre)" \
    'git config user.name  "José Javier Villanueva Aramburú"' 'git config user.email "<tu correo de la cuenta JJva-06>"'
[[ -f data/bronze/vigilancia_dengue_2000_2025.csv ]] ||
  alto 1 "Preparar la rama" "falta el CSV en data/bronze/" "make data   (o copia el CSV que ya usaste en la PC3)"
[[ -f deploy/.env ]] && grep -q '^JWT_SECRETO=..*' deploy/.env &&
  grep -q '^VIGIA_ADMIN_CLAVE=.\{12,\}' deploy/.env && grep -q '^VIGIA_ANALISTA_CLAVE=.\{12,\}' deploy/.env ||
  alto 1 "Preparar la rama" "deploy/.env no existe o le falta JWT_SECRETO o una clave de 12 caracteres o más" \
    "cp deploy/.env.example deploy/.env" "completa JWT_SECRETO (openssl rand -base64 48), VIGIA_ADMIN_CLAVE y VIGIA_ANALISTA_CLAVE"
[[ -f entregables/plantilla.docx && -f entregables/codigos.txt ]] ||
  alto 1 "Preparar la rama" "falta entregables/plantilla.docx o entregables/codigos.txt (no se versionan)" \
    "pídele a Jorge la plantilla de la carátula y escribe tu código y el de Jorge en entregables/codigos.txt"
total=$((total + 1)); ok "Paso 1: rama feature/70-evidencias al día, identidad, CSV, deploy/.env y plantilla"

# --- Pasos 2 a 12: evidencias por script, en orden ----------------------------
S="bash scripts/evidencias-tb2.sh"
evidencia 2  entorno       "Entorno y dataset"                         "$S entorno"
evidencia 3  parte-a       "Parte A en Docker"                         "$S parte-a"
evidencia 4  cluster       "Clúster con VERIFICAR=true"                "$S cluster"
evidencia 5  caida         "Caída de un nodo"                          "$S caida"
evidencia 6  servicio      "Modo servicio arriba"                      "$S servicio"
evidencia 7  entrenamiento "Entrenamiento por la API"                  "$S entrenamiento"
evidencia 8  benchapi      "Latencia de la API con JWT"                "$S benchapi"
[[ $(mtime reports/bench_api_deez_jwt.json) -ge $(mtime "$EVID/servicio.txt") ]] ||
  alto 8 "Latencia de la API con JWT" "reports/bench_api_deez_jwt.json no es de esta corrida" "$S benchapi"
evidencia 9  bd            "MongoDB y Redis"                           "$S bd"
evidencia 10 evaluacion    "Evaluación del modelo servido"             "$S evaluacion"
evidencia 11 spin          "Spin"                                      "$S spin     (en WSL, con: sudo apt-get install -y spin gcc)"
evidencia 12 pruebas       "Pruebas de Go y de la SPA"                 "$S pruebas  (en WSL: go test -race necesita gcc)"

# --- Pasos 13 y 14: seguridad y Playwright (con el sistema arriba) -----------
evidencia 13 seguridad     "Seguridad (reports/seguridad.json)"        "$S seguridad"
grep -q '"todos_ok": *true' reports/seguridad.json 2>/dev/null ||
  alto 13 "Seguridad" "reports/seguridad.json no tiene todos_ok: true" "Avísale a Jorge con el archivo: es un hallazgo, no se corrige a mano."
evidencia 14 e2e           "Playwright: flujos por rol y capturas"     "$S e2e"

# --- Paso 15: capturas manuales -------------------------------------------------
faltan=()
while IFS= read -r c; do
  [[ -f "docs/informe/tb2/$c" ]] || faltan+=("$c")
done < <(grep -ho '\[capturas/[^]]*\.png\]' docs/informe/tb2/[0-9][0-9]-*.md | tr -d '[]' | sort -u)
if ((${#faltan[@]})); then
  alto 15 "Capturas manuales" "${#faltan[@]} capturas que el informe pide y no están en $CAPT/" \
    "Tómalas como dice la tabla del paso 15 de la guía, con el nombre exacto:" "${faltan[@]}"
fi
total=$((total + 1)); ok "Paso 15: todas las capturas del informe están en $CAPT/"

# --- Paso 16: informe final -----------------------------------------------------
# Candado: el paso 16 arma el Word final, así que no se puede correr mientras el
# informe tenga secciones por escribir (las escribe Jorge por bloques, con PR y
# juez). Se abre solo cuando integracion/tb2 trae todo el texto y tu rama lo
# tiene (git merge origin/integracion/tb2; el paso 1 lo exige).
pendientes=$(grep -l '<!-- pendiente:' docs/informe/tb2/*.md 2>/dev/null | xargs -r -n1 basename | sort | tr '\n' ' ')
if [[ -n "$pendientes" ]]; then
  total=$((total + 1))
  alto 16 "Informe final" "el informe aún tiene secciones por escribir: $pendientes" \
    "Esto no lo haces tú: Jorge termina esas secciones y te avisa." \
    "Cuando te avise:  git fetch origin && git merge origin/integracion/tb2" \
    "y vuelve a correr:  bash scripts/paso70.sh"
fi
evidencia 16 informe       "Informe final (cmd/informe y cmd/word -final)" "QUITAR=\"<texto que te indique Jorge>\" $S informe"

# --- Paso 17: commits a tu nombre ------------------------------------------------
if [[ -n "$(git status --porcelain -- docs/informe reports)" ]]; then
  alto 17 "Commits" "hay cambios sin commit en docs/informe o reports" \
    "git add docs/informe reports" 'git commit -m "docs(informe): evidencias y capturas de la TB2 en deez (#70)"'
fi
propios=$(git log --format='%ae' origin/integracion/tb2..HEAD | grep -cv "^$CORREO_JORGE$" || true)
((propios > 0)) || alto 17 "Commits" "no hay commits tuyos en la rama" "git add docs/informe reports && git commit -m \"docs(informe): evidencias y capturas de la TB2 en deez (#70)\""
total=$((total + 1)); ok "Paso 17: cambios en commits a tu nombre"

# --- Paso 18: entrega a Jorge (sin push) ----------------------------------------
[[ -f jose-70.bundle && $(mtime jose-70.bundle) -ge $(git log -1 --format=%ct) ]] && git bundle verify -q jose-70.bundle 2>/dev/null ||
  alto 18 "Entrega" "falta jose-70.bundle o es anterior a tu último commit" \
    "git bundle create jose-70.bundle origin/integracion/tb2..feature/70-evidencias" \
    "Envíale a Jorge jose-70.bundle y el .docx de entregables/. NO hagas push ni merge."
total=$((total + 1)); ok "Paso 18: jose-70.bundle listo para Jorge"

echo
echo "Todo completo ($total pasos). Envía jose-70.bundle y el .docx a Jorge."
