#!/bin/sh
# sdd-ci-comment: mantém UM comentário de resumo do CI num PR (spec 015 FR-3).
# Resume o CI do commit com `sdd-ci.sh --no-wait` (uma linha por check e só o fim
# do log das falhas) e cria o comentário marcado ou edita o que já existe. Quando o
# CI fica verde, edita o comentário existente para dizer isso; sem comentário e
# sem falha, não escreve nada. Rodar duas vezes não duplica nada (NFR-2).
#
# Uso: sdd-ci-comment.sh [--repo DONO/REPO] --pr N --sha SHA [--lines N]
# Ambiente: SDD_ENGINE=off desliga tudo, sem ler nem escrever (AC-7).
# Códigos: 0 feito ou nada a fazer; 3 uso/erro. Requer gh; só chamadas REST.
set -eu

marker='<!-- sdd-ci-summary -->'
repo="" pr="" sha="" lines=30
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --pr) pr="${2:?}"; shift 2 ;;
    --sha) sha="${2:?}"; shift 2 ;;
    --lines) lines="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,10p' "$0"; exit 0 ;;
    *) echo "sdd-ci-comment: argumento desconhecido: $1" >&2; exit 3 ;;
  esac
done

if [ "${SDD_ENGINE:-}" = off ]; then
  echo "sdd-ci-comment: SDD_ENGINE=off; nada a fazer"
  exit 0
fi
if [ -z "$pr" ] || [ -z "$sha" ]; then
  echo "sdd-ci-comment: --pr e --sha são obrigatórios" >&2
  exit 3
fi
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2>/dev/null)" || { echo "sdd-ci-comment: sem remote origin; use --repo" >&2; exit 3; }
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# Uma rodada antiga não sobrescreve o resumo de um commit mais novo.
head="$(gh api "repos/$repo/pulls/$pr" --jq .head.sha)"
if [ "$head" != "$sha" ]; then
  echo "sdd-ci-comment: $sha não é mais a ponta do PR #$pr; nada a fazer"
  exit 0
fi

rc=0
sh "$here/sdd-ci.sh" --repo "$repo" --no-wait --lines "$lines" "$sha" > "$tmp/summary" 2>&1 || rc=$?
short="$(printf %.7s "$sha")"
case "$rc" in
  0) title="CI verde em $short" ;;
  1) title="CI vermelho em $short" ;;
  2) # Checks ainda pendentes: só comenta se já há falha para mostrar.
    grep -q '^FALHA ' "$tmp/summary" || { echo "sdd-ci-comment: checks pendentes; nada a fazer"; exit 0; }
    title="CI vermelho em $short (há checks pendentes)" ;;
  *) cat "$tmp/summary" >&2; exit 3 ;;
esac

id="$(gh api --paginate "repos/$repo/issues/$pr/comments?per_page=100" \
  --jq ".[] | select(.body | contains(\"$marker\")) | .id" | sed -n 1p)"
if [ "$rc" -eq 0 ] && [ -z "$id" ]; then
  echo "sdd-ci-comment: CI verde e sem comentário anterior; nada a fazer"
  exit 0
fi

{
  echo "$marker"
  echo "### $title"
  echo
  echo '~~~~text'
  head -c 60000 "$tmp/summary"
  echo '~~~~'
  echo
  echo "_Resumo automático do \`sdd-ci-comment.sh\`; é editado a cada rodada do CI._"
} > "$tmp/body.md"

if [ -n "$id" ]; then
  gh api -X PATCH "repos/$repo/issues/comments/$id" -F "body=@$tmp/body.md" --silent
  echo "sdd-ci-comment: comentário $id do PR #$pr editado ($title)"
else
  gh api -X POST "repos/$repo/issues/$pr/comments" -F "body=@$tmp/body.md" --silent
  echo "sdd-ci-comment: comentário criado no PR #$pr ($title)"
fi
