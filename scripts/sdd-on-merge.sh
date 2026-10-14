#!/bin/sh
# sdd-on-merge: quando um PR de ticket é mergeado, atualiza o checkpoint do épico
# aberto sem LLM (spec 015 FR-2): "feito" é o PR (#N título) e "próximo" é o
# sub-issue aberto de menor número do épico. Branch e commit saem do checkout
# (sdd-checkpoint.sh --ci); quem escreve é o próprio sdd-checkpoint.sh, então
# rodar duas vezes não duplica nem reescreve nada (NFR-2).
#
# Uso: sdd-on-merge.sh [--repo DONO/REPO] --pr N
# Ambiente: SDD_ENGINE=off desliga tudo, sem ler nem escrever (AC-7).
# PRs de release (chore/release-v*) são ignorados: têm o próprio fluxo.
# Códigos: 0 feito ou nada a fazer; 3 uso/erro. Requer gh e jq; só chamadas REST.
set -eu

repo="" pr=""
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --pr) pr="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,10p' "$0"; exit 0 ;;
    *) echo "sdd-on-merge: argumento desconhecido: $1" >&2; exit 3 ;;
  esac
done

if [ "${SDD_ENGINE:-}" = off ]; then
  echo "sdd-on-merge: SDD_ENGINE=off; nada a fazer"
  exit 0
fi
[ -n "$pr" ] || { echo "sdd-on-merge: --pr é obrigatório" >&2; exit 3; }
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2>/dev/null)" || { echo "sdd-on-merge: sem remote origin; use --repo" >&2; exit 3; }
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

gh api "repos/$repo/pulls/$pr" > "$tmp/pr.json" || { echo "sdd-on-merge: PR #$pr ilegível" >&2; exit 3; }
[ "$(jq -r '.merged' "$tmp/pr.json")" = true ] || { echo "sdd-on-merge: PR #$pr não foi mergeado; nada a fazer"; exit 0; }
case "$(jq -r '.head.ref' "$tmp/pr.json")" in
  chore/release-v*) echo "sdd-on-merge: PR de release; nada a fazer"; exit 0 ;;
esac
title="$(jq -r '.title' "$tmp/pr.json")"

epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty' 2>/dev/null || true)"
[ -n "$epic" ] || { echo "sdd-on-merge: nenhum épico aberto em $repo"; exit 0; }

# Tickets que o PR fecha ainda podem constar como abertos (o fechamento é assíncrono).
closed="$(jq -r '.body // ""' "$tmp/pr.json" | grep -oiE '(close[sd]?|fix(e[sd])?|resolve[sd]?) #[0-9]+' | grep -oE '[0-9]+' | paste -sd ' ' - || true)"
tab="$(printf '\t')"
gh api "repos/$repo/issues/$epic/sub_issues?per_page=100" \
  --jq '.[] | select(.state == "open") | "\(.number)\t\(.title)"' | sort -n > "$tmp/open" || : > "$tmp/open"
next=""
while IFS="$tab" read -r n t; do
  case " $closed " in *" $n "*) continue ;; esac
  next="#$n $t"
  break
done < "$tmp/open"
[ -n "$next" ] || next="nenhum ticket aberto; preparar o PR de fechamento da fase"

sh "$here/sdd-checkpoint.sh" --repo "$repo" --ci save "#$pr $title (mergeado)" "$next"
