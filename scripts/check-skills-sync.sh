#!/usr/bin/env bash
# Verifica que cada skill exista con el mismo contenido en .agents/skills (Antigravity)
# y en .claude/skills (Claude Code). La fuente canónica es .agents/skills.
# Uso: scripts/check-skills-sync.sh [--fix]
set -euo pipefail

raiz="$(cd "$(dirname "$0")/.." && pwd)"
origen="$raiz/.agents/skills"
destino="$raiz/.claude/skills"
fix="${1:-}"
fallas=0

[ -d "$origen" ] || { echo "skills: no existe $origen"; exit 1; }

for dir in "$origen"/*/; do
  nombre="$(basename "$dir")"
  if [ "$fix" = "--fix" ]; then
    mkdir -p "$destino/$nombre"
    cp -f "$dir"SKILL.md "$destino/$nombre/SKILL.md"
    continue
  fi
  if ! cmp -s "$dir/SKILL.md" "$destino/$nombre/SKILL.md" 2>/dev/null; then
    echo "skills: '$nombre' difiere o falta en .claude/skills (corre: scripts/check-skills-sync.sh --fix)"
    fallas=1
  fi
done

if [ -d "$destino" ]; then
  for dir in "$destino"/*/; do
    nombre="$(basename "$dir")"
    if [ ! -d "$origen/$nombre" ]; then
      echo "skills: '$nombre' existe en .claude/skills pero no en .agents/skills (la fuente canónica)"
      fallas=1
    fi
  done
fi

[ "$fix" = "--fix" ] && { echo "skills: sincronizadas"; exit 0; }
[ "$fallas" -eq 0 ] && echo "skills: ok"
exit "$fallas"
