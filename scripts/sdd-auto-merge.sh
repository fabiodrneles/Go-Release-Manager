#!/bin/sh
# sdd-auto-merge: liga e desliga o merge automático do repositório, num comando.
#
# Uso: sdd-auto-merge.sh [--repo DONO/REPO] on|off|status
#      sdd-auto-merge.sh [--repo DONO/REPO] merge --sha SHA --branch BRANCH
#   on      liga: todo PR com o CI verde e sem conflito é mesclado sozinho
#   off     desliga (padrão): todo merge é do dono
#   status  mostra se está ligado
#   merge   o que o workflow sdd-auto-merge.yml roda quando o CI de um PR termina:
#           mescla o PR da BRANCH, se a cabeça dele for SHA, todos os checks
#           estiverem verdes e não houver conflito; depois dispara os workflows do
#           motor que um merge do GITHUB_TOKEN não dispara (checkpoint, fase, release)
# A chave é a variável SDD_AUTO_MERGE do repositório, gravada pelo gh do usuário.
# Códigos: 0 ok ou nada a fazer; 1 falha do GitHub; 3 uso. Requer gh.
set -eu

usage() { sed -n '2,14p' "$0"; exit 3; }
repo=""
[ "${1:-}" != --repo ] || { repo="${2:?}"; shift 2; }
cmd="${1:-}"
[ $# -eq 0 ] || shift
say() { echo "sdd-auto-merge: $*"; }
here="$(cd "$(dirname "$0")" && pwd)"
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2> /dev/null || true)"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
[ -n "$repo" ] || { echo "sdd-auto-merge: repositório desconhecido (use --repo)" >&2; exit 3; }
var="repos/$repo/actions/variables"

case "$cmd" in
  on | off)
    if gh api "$var/SDD_AUTO_MERGE" --silent 2> /dev/null; then
      gh api -X PATCH "$var/SDD_AUTO_MERGE" -f name=SDD_AUTO_MERGE -f value="$cmd" --silent
    else
      gh api -X POST "$var" -f name=SDD_AUTO_MERGE -f value="$cmd" --silent
    fi || { echo "sdd-auto-merge: não consegui gravar a chave em $repo (o gh tem acesso de admin?)" >&2; exit 1; }
    if [ "$cmd" = on ]; then say "ligado em $repo: todo PR com o CI verde e sem conflito é mesclado sozinho"
    else say "desligado em $repo: todo merge é do dono"; fi
    exit 0 ;;
  status)
    v="$(gh api "$var/SDD_AUTO_MERGE" --jq .value 2> /dev/null)" || v=off
    case "$v" in on) ;; *) v=off ;; esac
    say "$v em $repo"; exit 0 ;;
  merge) ;;
  *) usage ;;
esac

sha="" branch=""
while [ $# -gt 0 ]; do
  case "$1" in
    --sha) sha="${2:?}"; shift 2 ;;
    --branch) branch="${2:?}"; shift 2 ;;
    *) usage ;;
  esac
done
if [ -z "$sha" ] || [ -z "$branch" ]; then usage; fi
[ "${SDD_AUTO_MERGE:-off}" = on ] || { say "desligado (SDD_AUTO_MERGE=${SDD_AUTO_MERGE:-off}): o merge é do dono"; exit 0; }

pr="$(gh api "repos/$repo/pulls?state=open&head=${repo%%/*}:$branch&per_page=1" --jq '.[0] | select(.) | "\(.number) \(.head.sha)"')"
[ -n "$pr" ] || { say "nenhum PR aberto da branch $branch"; exit 0; }
num="${pr%% *}" head="${pr#* }"
[ "$head" = "$sha" ] || { say "PR #$num: a cabeça mudou ($head); espera o CI dela"; exit 0; }
rc=0; sh "$here/sdd-ci.sh" --repo "$repo" --no-wait "$sha" > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 0 ] || { say "PR #$num: nem todos os checks de $sha estão verdes; não mesclo"; exit 0; }
mergeable="$(gh api "repos/$repo/pulls/$num" --jq '.mergeable // "null"')"
[ "$mergeable" = true ] || { say "PR #$num: o GitHub não diz que é mesclável ($mergeable); não mesclo"; exit 0; }
gh api -X PUT "repos/$repo/pulls/$num/merge" -f merge_method=merge -f sha="$sha" --silent \
  || { echo "sdd-auto-merge: o GitHub recusou o merge do PR #$num" >&2; exit 1; }
say "PR #$num mesclado (SDD_AUTO_MERGE=on)"

# Um merge feito pelo GITHUB_TOKEN não gera os eventos que os workflows do motor
# esperam; com um token pessoal (SDD_ENGINE_TOKEN), eles disparam sozinhos.
if [ -z "${GITHUB_ACTIONS:-}" ] || [ -n "${SDD_ENGINE_TOKEN_SET:-}" ]; then exit 0; fi
ref="${DEFAULT_BRANCH:-main}"
dispatch() {
  if gh api -X POST "repos/$repo/actions/workflows/$1/dispatches" -f ref="$ref" -f "inputs[$2]=$3" --silent; then
    say "disparado $1 ($2 $3)"
  else
    say "aviso: não consegui disparar $1"
  fi
}
case "$branch" in
  chore/release-v*) dispatch sdd-on-release-merge.yml pr "$num" ;;
  *)
    dispatch sdd-on-merge.yml pr "$num"
    for i in $(gh api "repos/$repo/pulls/$num" --jq '.body // ""' | grep -oiE '(closes|fixes|resolves) #[0-9]+' | grep -oE '[0-9]+' | sort -u); do
      dispatch sdd-on-phase-done.yml issue "$i"
    done ;;
esac
