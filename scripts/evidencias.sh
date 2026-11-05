#!/usr/bin/env bash
# Genera las evidencias del informe en docs/informe/pc3/evidencias/:
# salida de go test -race, historial de Git Flow, Pull Requests, tablero Kanban
# y commits por integrante. Requiere go (con cgo para -race), git y gh con
# permiso read:project. Las secciones del informe las incluyen tal cual.
set -euo pipefail

raiz="$(cd "$(dirname "$0")/.." && pwd)"
dir="$raiz/docs/informe/pc3/evidencias"
repo="JorgeGarcia005/cc65-tf-dengue"
proyecto=4
mkdir -p "$dir"
cd "$raiz"

echo "go test -race"
{
  echo "\$ go test -race -count=1 ./...   # $(go version | awk '{print $3, $4}')"
  CGO_ENABLED=1 go test -race -count=1 ./... 2>&1 | grep -v 'no test files'
} > "$dir/go-test-race.txt"

echo "git log --graph"
{
  echo "\$ git log --graph --oneline --decorate --all"
  git log --graph --oneline --decorate --all --color=never \
    --exclude='refs/remotes/origin/dependabot/*' --all | head -n 45
} > "$dir/gitflow.txt"

echo "Pull Requests"
{
  echo "| PR | Rama | Estado | Título |"
  echo "|---:|---|---|---|"
  gh pr list --repo "$repo" --state all --limit 100 \
    --json number,title,headRefName,state,author \
    --jq '.[] | select(.author.login != "app/dependabot") |
          "| #\(.number) | `\(.headRefName)` | \(.state | ascii_downcase) | \(.title | gsub("\\|"; "/")) |"'
} > "$dir/prs.md"

echo "Kanban"
{
  echo "| Columna | Issue | Título |"
  echo "|---|---:|---|"
  gh project item-list "$proyecto" --owner "${repo%%/*}" --limit 200 --format json \
    --jq '.items | sort_by(.content.number) | .[] |
          "| \(.status // "Sin columna") | #\(.content.number) | \(.content.title | gsub("\\|"; "/")) |"'
} > "$dir/kanban.md"

echo "commits por autor"
{
  echo "Commits por autor en todas las ramas, sin merges (identidades unidas con \`.mailmap\`):"
  echo
  echo "| Autor | Commits |"
  echo "|---|---:|"
  # git log y no git shortlog: shortlog lee la entrada estándar si no hay terminal.
  git log --all --no-merges --format='%aN' | sort | uniq -c | sort -rn |
    awk '{n = $1; $1 = ""; sub(/^ /, ""); printf "| %s | %s |\n", $0, n}'
} > "$dir/commits.md"

echo "listo: $dir"
