#!/bin/sh
# sdd-on-phase-done: quando o último ticket aberto do épico aberto é fechado, abre o
# PR de fechamento da versão com o sdd-release.sh, sem LLM (spec 015 FR-4).
# Confere, só por REST, que a issue é sub-issue do épico aberto e que nenhuma
# outra sub-issue está aberta. Idempotente (NFR-2): se já há PR aberto de uma branch
# chore/release-v*, não faz nada.
#
# Uso: sdd-on-phase-done.sh [--repo DONO/REPO] ISSUE
# Ambiente: SDD_ENGINE=off desliga tudo, sem ler nem escrever (AC-7).
# Em CI, defina a identidade do git (github-actions[bot]) antes; veja o workflow.
# Códigos: 0 feito ou nada a fazer; 1 falha do sdd-release.sh; 3 uso. Requer gh.
set -eu

repo="" issue=""
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,11p' "$0"; exit 0 ;;
    -*) echo "sdd-on-phase-done: opção desconhecida: $1" >&2; exit 3 ;;
    *) issue="${1#\#}"; shift ;;
  esac
done
if [ "${SDD_ENGINE:-}" = off ]; then
  echo "sdd-on-phase-done: SDD_ENGINE=off; nada a fazer"
  exit 0
fi
case "$issue" in '' | *[!0-9]*) echo "sdd-on-phase-done: informe o número da issue fechada" >&2; exit 3 ;; esac
here="$(cd "$(dirname "$0")" && pwd)"
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2> /dev/null)" || { echo "sdd-on-phase-done: sem remote origin; use --repo" >&2; exit 3; }
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi

epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty')" || epic=""
if [ -z "$epic" ]; then
  echo "sdd-on-phase-done: nenhum épico aberto; nada a fazer"
  exit 0
fi
subs="$(gh api --paginate "repos/$repo/issues/$epic/sub_issues?per_page=100" --jq '.[] | "\(.number) \(.state)"')" \
  || { echo "sdd-on-phase-done: falha ao listar as sub-issues do épico #$epic" >&2; exit 3; }
if ! printf '%s\n' "$subs" | grep -q "^$issue "; then
  echo "sdd-on-phase-done: #$issue não é sub-issue do épico #$epic; nada a fazer"
  exit 0
fi
if printf '%s\n' "$subs" | grep -q ' open$'; then
  echo "sdd-on-phase-done: o épico #$epic ainda tem tickets abertos; nada a fazer"
  exit 0
fi
existing="$(gh api --paginate "repos/$repo/pulls?state=open&per_page=100" \
  --jq '.[] | select(.head.ref | startswith("chore/release-v")) | .number')" || existing=""
if [ -n "$existing" ]; then
  echo "sdd-on-phase-done: já há PR de fechamento aberto (#$(printf '%s\n' "$existing" | head -n 1)); nada a fazer"
  exit 0
fi
echo "sdd-on-phase-done: último ticket do épico #$epic fechado; preparando o PR de fechamento"
# O runner do motor não tem as ferramentas do projeto: o CI do PR verifica, e sem
# SDD_ENGINE_TOKEN o PR do GITHUB_TOKEN precisa que o CI seja disparado (#182).
SDD_PR_NO_CI=1
export SDD_PR_NO_CI
if [ -z "${SDD_ENGINE_TOKEN_SET:-}" ]; then SDD_PR_DISPATCH_CI=1; export SDD_PR_DISPATCH_CI; fi
exec sh "$here/sdd-release.sh" --repo "$repo"
