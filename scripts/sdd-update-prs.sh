#!/bin/sh
# sdd-update-prs: quando a main muda, traz a main para os PRs abertos que estão
# atrás dela (merge, nunca rebase) e avisa uma vez os que têm conflito real
# (sdd-kit, spec 015 FR-6, NFR-1, NFR-2). Só REST (gh api). Usado pelo workflow
# sdd-update-prs.yml; a lógica fica aqui para ser testável.
#
# Uso: sdd-update-prs.sh [--repo DONO/REPO] [--base main]
# Uma linha por PR: "atualizado #N", "em dia #N" ou "conflito #N (avisado)".
# Só PRs cuja branch é do próprio repositório; PRs de fork são ignorados (NFR-1).
# Com SDD_ENGINE=off não lê nem escreve nada (AC-7). O comentário de conflito leva
# o marcador <!-- sdd-update-prs:conflict --> e não é repetido (NFR-2).
# O mergeable_state do GitHub é calculado sob demanda: se vier vazio, espera
# SDD_UPDATE_WAIT segundos (padrão 3) e lê de novo, até 4 vezes.
#
# CI depois do update-branch: um push feito com o GITHUB_TOKEN não dispara o
# pull_request, então, sem o segredo SDD_ENGINE_TOKEN (SDD_ENGINE_TOKEN_SET vazio),
# o script espera a cabeça do PR mudar (até 10 leituras, SDD_UPDATE_WAIT entre elas)
# e dispara o workflow de CI (SDD_CI_WORKFLOW, padrão ci.yml) na branch do PR por
# workflow_dispatch, que roda mesmo com o GITHUB_TOKEN; os checks caem na cabeça e
# aparecem no PR. Com o SDD_ENGINE_TOKEN, o push dele já dispara o CI (#174).
# Códigos: 0 ok; 1 algum PR falhou ao atualizar; 3 uso.
set -eu

die() { echo "sdd-update-prs: $*" >&2; exit 3; }
repo="${GITHUB_REPOSITORY:-}" base=main
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --base) base="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,23p' "$0"; exit 0 ;;
    *) die "opção desconhecida: $1" ;;
  esac
done
if [ "${SDD_ENGINE:-}" = "off" ]; then echo "SDD_ENGINE=off: nada a fazer"; exit 0; fi
[ -n "$repo" ] || die "informe --repo DONO/REPO (ou GITHUB_REPOSITORY)"
marker='<!-- sdd-update-prs:conflict -->'
wait_s="${SDD_UPDATE_WAIT:-3}"
ci_wf="${SDD_CI_WORKFLOW:-ci.yml}"
failf="$(mktemp)"
rm -f "$failf"

# Número, SHA da cabeça e repositório da cabeça (- se apagado) dos PRs abertos.
prs="$(gh api --paginate "repos/$repo/pulls?state=open&base=$base&per_page=100" \
  --jq '.[] | "\(.number) \(.head.sha) \(.head.repo.full_name // "-") \(.head.ref)"')"

printf '%s\n' "$prs" | while read -r n sha hrepo ref; do
  [ -n "$n" ] || continue
  [ "$hrepo" = "$repo" ] || { echo "ignorado #$n (branch de fork)"; continue; }
  # O GET do PR dispara o cálculo de mergeabilidade; releia enquanto vier vazio.
  state=""
  for _ in 1 2 3 4; do
    state="$(gh api "repos/$repo/pulls/$n" --jq '.mergeable_state // ""')"
    case "$state" in "" | unknown) sleep "$wait_s" ;; *) break ;; esac
  done
  if [ "$state" = "dirty" ]; then
    if ! gh api --paginate "repos/$repo/issues/$n/comments?per_page=100" --jq '.[].body' | grep -qF "$marker"; then
      gh api -X POST "repos/$repo/issues/$n/comments" --silent -f body="$marker
A \`$base\` mudou e este PR tem conflito com ela. Resolva localmente: \`git merge origin/$base\`, corrija os arquivos em conflito e envie."
    fi
    echo "conflito #$n (avisado)"
    continue
  fi
  behind="$(gh api "repos/$repo/compare/$base...$sha" --jq '.behind_by')"
  if [ "${behind:-0}" -eq 0 ]; then echo "em dia #$n"; continue; fi
  # Merge da main na branch, só se a cabeça ainda é a que lemos.
  if gh api -X PUT "repos/$repo/pulls/$n/update-branch" --silent -f expected_head_sha="$sha"; then
    if [ -n "${SDD_ENGINE_TOKEN_SET:-}" ]; then echo "atualizado #$n"; continue; fi
    # O update-branch é assíncrono: espera a cabeça nova para o CI rodar nela.
    moved=""
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      [ "$(gh api "repos/$repo/pulls/$n" --jq '.head.sha')" != "$sha" ] && { moved=1; break; }
      sleep "$wait_s"
    done
    note=""
    [ -n "$moved" ] || note=", cabeça ainda não mudou"
    if err="$(gh api -X POST "repos/$repo/actions/workflows/$ci_wf/dispatches" --silent -f ref="$ref" 2>&1)"; then
      echo "atualizado #$n (CI disparado$note)"
    else
      echo "atualizado #$n (CI não disparado: $(printf '%s' "$err" | head -n 1))"
    fi
  else
    echo "falhou #$n (update-branch)"
    : > "$failf"
  fi
done
if [ -f "$failf" ]; then rm -f "$failf"; exit 1; fi
