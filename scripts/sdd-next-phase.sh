#!/bin/sh
# sdd-next-phase: a próxima fase do ROADMAP com tarefas abertas (sdd-kit, #191).
#
# Uso: sdd-next-phase.sh [ROADMAP]   (padrão: specs/ROADMAP.md)
# Saída: "N<TAB>cabeçalho da fase" da primeira fase "## Fase N — ..." que tem
# "- [ ] **Tn**"; código 1 se nenhuma fase tem tarefa aberta, 2 sem o ROADMAP.
set -u
r="${1:-specs/ROADMAP.md}"
[ -f "$r" ] || { echo "sdd-next-phase: $r não existe" >&2; exit 2; }
awk '
  /^## Fase [0-9]+/ { n = $3; h = $0; sub(/^## /, "", h); next }
  /^## / { n = "" }
  n != "" && /^- \[ \] \*\*T[0-9]+\*\*/ { printf "%s\t%s\n", n, h; found = 1; exit }
  END { exit !found }
' "$r"
