#!/bin/sh
# sdd-checkpoint: o ponto de retomada do trabalho, num comentário do épico aberto
# (sdd-kit, spec 014). Um único comentário, marcado com <!-- sdd-checkpoint -->,
# é atualizado a cada passo; o hook de início de sessão o mostra, e uma sessão
# nova continua dele sem o dono precisar explicar nada.
#
# Uso: sdd-checkpoint.sh [--repo DONO/REPO] [--ci] save "FEITO" "PRÓXIMO" ["BLOQUEIA"]
#      sdd-checkpoint.sh [--repo DONO/REPO] [--ci] auto   (hook Stop: só o estado do git)
#      sdd-checkpoint.sh [--repo DONO/REPO] [--ci] show   (hook SessionStart)
# O estado do git (branch, commit, alterações locais, commits não enviados) e os
# PRs abertos entram sozinhos. "auto" mantém o feito e o próximo do último save.
# --ci (workflow, sem máquina de trabalho): a linha da branch traz só branch e commit,
# sem alterações locais nem commits não enviados (spec 015 FR-2).
# Sem épico aberto ou sem gh, avisa e sai com 0: nunca bloqueia a sessão.
set -eu

repo="" ci=0
while [ "${1:-}" = --repo ] || [ "${1:-}" = --ci ]; do
  if [ "$1" = --ci ]; then ci=1; shift; else repo="${2:?}"; shift 2; fi
done
cmd="${1:-}"
[ $# -eq 0 ] || shift
case "$cmd" in save | auto | show) ;; *) sed -n '2,14p' "$0"; exit 2 ;; esac
[ "$cmd" != save ] || [ $# -ge 2 ] || { sed -n '2,14p' "$0"; exit 2; }

command -v gh >/dev/null 2>&1 || { echo "sdd-checkpoint: gh ausente"; exit 0; }
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2>/dev/null || true)"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
[ -n "$repo" ] || { echo "sdd-checkpoint: repositório desconhecido (use --repo)"; exit 0; }

marker='<!-- sdd-checkpoint -->'
epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty' 2>/dev/null || true)"
[ -n "$epic" ] || { echo "sdd-checkpoint: nenhum épico aberto em $repo"; exit 0; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
gh api --paginate "repos/$repo/issues/$epic/comments?per_page=100" \
  --jq ".[] | select(.body | startswith(\"$marker\")) | {id, body}" > "$tmp/found" 2>/dev/null || : > "$tmp/found"
id="$(jq -rs 'last | .id // empty' "$tmp/found")"
jq -js 'last | .body // ""' "$tmp/found" > "$tmp/old"

if [ "$cmd" = show ]; then
  if [ -n "$id" ]; then
    echo "Checkpoint do épico #$epic ($repo); continue a partir dele:"
    grep -v "^$marker" "$tmp/old"
  else
    echo "Épico aberto #$epic ($repo) sem checkpoint; leia o último \"Estado da fase\" dele."
  fi
  exit 0
fi

field() { sed -n "s/^- \*\*$1:\*\* //p" "$tmp/old" | head -n 1; }
if [ "$cmd" = save ]; then
  done_="$1" next="$2" block="${3:-nada}"
else
  done_="$(field Feito)" next="$(field Próximo)" block="$(field Bloqueia)"
  [ -n "$next" ] || { echo "sdd-checkpoint: sem checkpoint anterior; use save"; exit 0; }
fi

branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '?')"
sha="$(git rev-parse --short HEAD 2>/dev/null || echo '?')"
dirty="$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')"
ahead="$(git rev-list --count '@{upstream}..HEAD' 2>/dev/null || echo 'sem upstream')"
# shellcheck disable=SC2016 # crases são Markdown no texto do jq
prs="$(gh api "repos/$repo/pulls?state=open&per_page=20" --jq '.[] | "#\(.number) \(.title) (`\(.head.ref)`)"' 2>/dev/null | paste -sd ';' - | sed 's/;/; /g')"

{
  echo "$marker"
  echo "## Checkpoint (retome daqui)"
  echo
  if [ "$ci" -eq 1 ]; then
    echo "- **Branch:** \`$branch\` em \`$sha\`"
  else
    echo "- **Branch:** \`$branch\` em \`$sha\`; alterações locais: $dirty; commits não enviados: $ahead"
  fi
  echo "- **PRs abertos:** ${prs:-nenhum}"
  echo "- **Feito:** $done_"
  echo "- **Próximo:** $next"
  echo "- **Bloqueia:** ${block:-nada}"
} > "$tmp/new"
# Sem mudança, nada é escrito (o hook Stop roda a cada resposta). Compara só até
# o Bloqueia: o servidor pode acrescentar um rodapé ao comentário.
if [ -n "$id" ] && [ "$(tr -d '\r' < "$tmp/old" | sed '/^- \*\*Bloqueia:\*\*/q')" = "$(cat "$tmp/new")" ]; then
  exit 0
fi
if [ -n "$id" ]; then
  gh api -X PATCH "repos/$repo/issues/comments/$id" -F body=@"$tmp/new" --silent
else
  gh api "repos/$repo/issues/$epic/comments" -F body=@"$tmp/new" --silent
fi
echo "sdd-checkpoint: épico #$epic atualizado"
