#!/bin/sh
# sdd-ci: espera os checks e os status de commit (ex.: Vercel) de um commit, PR
# ou branch e imprime uma linha por check; para cada check que falhou, imprime
# só o fim do log do job (ou o link, num status de commit).
#
# Uso: sdd-ci.sh [--repo DONO/REPO] [--min N] [--timeout S] [--lines N] [--no-wait] [REF]
#   REF      SHA, branch ou "#N" (PR); padrão: HEAD local
#   --min    número mínimo de checks esperados antes de concluir (padrão 1)
#   --lines  linhas do fim do log de cada job que falhou (padrão 30)
# Saída: "ok|FALHA|pendente <check>" (status de commit: "<contexto> (status)"),
#        e "== log: <check>" + trecho para falhas.
# Códigos: 0 tudo verde; 1 algum check falhou; 2 tempo esgotado; 3 uso/erro.
# Requer gh (só leitura) e jq embutido no gh (--jq).
set -eu

repo="" min=1 timeout=900 lines=30 wait=1 ref=""
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --min) min="${2:?}"; shift 2 ;;
    --timeout) timeout="${2:?}"; shift 2 ;;
    --lines) lines="${2:?}"; shift 2 ;;
    --no-wait) wait=0; shift ;;
    -h | --help) sed -n '2,14p' "$0"; exit 0 ;;
    -*) echo "sdd-ci: opção desconhecida: $1" >&2; exit 3 ;;
    *) ref="$1"; shift ;;
  esac
done

# DONO/REPO a partir do remote origin (funciona com URL https, ssh ou proxy).
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2>/dev/null)" || { echo "sdd-ci: sem remote origin; use --repo" >&2; exit 3; }
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi

case "$ref" in
  "") sha="$(git rev-parse HEAD)" ;;
  \#*) sha="$(gh api "repos/$repo/pulls/${ref#\#}" --jq .head.sha)" ;;
  *) sha="$(gh api "repos/$repo/commits/$ref" --jq .sha)" ;;
esac

runs() { gh api "repos/$repo/commits/$sha/check-runs?per_page=100" --jq "$1"; }
# Status de commit (API antiga, usada por Vercel, Netlify e outros serviços).
statuses() { gh api "repos/$repo/commits/$sha/status?per_page=100" --jq "$1"; }

start="$(date +%s)"
while :; do
  # shellcheck disable=SC2046 # a divisão em palavras é intencional
  set -- $(runs '[.total_count, ([.check_runs[] | select(.status != "completed")] | length)] | @tsv')
  total="${1:-0}" pending="${2:-0}"
  spending="$(statuses '[.statuses[] | select(.state == "pending")] | length')"
  pending=$((pending + ${spending:-0}))
  if [ "$total" -ge "$min" ] && [ "$pending" -eq 0 ]; then
    # Um check pode nascer entre as duas leituras (ex.: um workflow disparado pelo
    # próprio merge): confirma que nada está rodando antes de classificar.
    late="$(runs '[.check_runs[] | select(.status != "completed")] | length')"
    [ "${late:-0}" -eq 0 ] && break
  fi
  if [ "$wait" -eq 0 ] || [ $(($(date +%s) - start)) -ge "$timeout" ]; then
    runs '.check_runs[] | "\(if .status != "completed" then "pendente" elif .conclusion == "success" or .conclusion == "skipped" or .conclusion == "neutral" then "ok" else "FALHA" end) \(.name)"'
    statuses '.statuses[] | "\(if .state == "pending" then "pendente" elif .state == "success" then "ok" else "FALHA" end) \(.context) (status)"'
    echo "sdd-ci: $repo@$(printf %.7s "$sha"): $total check(s), $pending pendente(s); tempo esgotado"
    exit 2
  fi
  sleep "${SDD_CI_INTERVAL:-15}"
done

tab="$(printf '\t')"
runs '.check_runs[] | [(if .status != "completed" then "pendente" elif .conclusion == "success" or .conclusion == "skipped" or .conclusion == "neutral" then "ok" else "FALHA" end), .name, (.id | tostring), (.app.slug // "")] | @tsv' > "${TMPDIR:-/tmp}/sdd-ci.$$"
statuses '.statuses[] | [(if .state == "success" then "ok" else "FALHA" end), "\(.context) (status)", "status", (.target_url // "")] | @tsv' >> "${TMPDIR:-/tmp}/sdd-ci.$$"
trap 'rm -f "${TMPDIR:-/tmp}/sdd-ci.$$"' EXIT
failed=0
while IFS="$tab" read -r st name _ _; do echo "$st $name"; done < "${TMPDIR:-/tmp}/sdd-ci.$$"
while IFS="$tab" read -r st name id app; do
  [ "$st" = FALHA ] || continue
  failed=$((failed + 1))
  echo "== log: $name"
  if [ "$id" = status ]; then
    echo "(status de commit: veja ${app:-o serviço que o publicou})"
  elif [ "$app" = github-actions ]; then
    # O id do check run é o id do job no Actions. Tira o carimbo de hora de cada
    # linha, mostra primeiro as linhas de erro e depois o fim do log.
    gh api "repos/$repo/actions/jobs/$id" --jq '.steps[] | select(.conclusion == "failure") | "passo que falhou: \(.name)"' || true
    if ! log="$(gh api "repos/$repo/actions/jobs/$id/logs" 2>/dev/null)"; then
      # Log expirado ou bloqueado: as anotações do check trazem os erros reportados.
      echo "(log indisponível; anotações do check:)"
      gh api "repos/$repo/check-runs/$id/annotations" \
        --jq '.[] | select(.annotation_level == "failure") | "\(.path):\(.start_line): \(.message)"' | head -n "$lines" || true
      continue
    fi
    log="$(printf '%s\n' "$log" | sed -E 's/^[0-9T:.-]+Z //')"
    printf '%s\n' "$log" | grep -E '##\[error\]' | head -n 5 || true
    printf '%s\n' "$log" | grep -vE '^##\[(group|endgroup)\]' | tail -n "$lines"
  else
    echo "(check externo: veja $(runs ".check_runs[] | select(.id == $id) | .details_url"))"
  fi
done < "${TMPDIR:-/tmp}/sdd-ci.$$"

n="$(wc -l < "${TMPDIR:-/tmp}/sdd-ci.$$" | tr -d ' ')"
if [ "$failed" -eq 0 ]; then
  echo "sdd-ci: $repo@$(printf %.7s "$sha"): $n check(s) verdes"
  exit 0
fi
echo "sdd-ci: $repo@$(printf %.7s "$sha"): $failed de $n check(s) falharam"
exit 1
