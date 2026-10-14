#!/bin/sh
# sdd-phase-status: monta o comentário "Estado da fase" do épico aberto a partir
# do GitHub (sub-issues, PRs que as fecham, CI de cada PR) e das decisões em
# aberto de specs/ANALYSIS.md.
#
# Uso: sdd-phase-status.sh [--repo DONO/REPO] [--epic N] [--next "texto"] [--post]
#   --epic  épico a usar (padrão: o épico aberto mais recente, label "épico")
#   --next  próximo passo (padrão: o primeiro ticket aberto sem PR, ou o fechamento)
#   --post  publica o comentário no épico; sem ela, só imprime o Markdown
# Saída sem --post: o Markdown do comentário. Com --post: uma linha com a URL.
set -eu

repo="" epic="" next="" post=0
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --epic) epic="${2:?}"; shift 2 ;;
    --next) next="${2:?}"; shift 2 ;;
    --post) post=1; shift ;;
    -h | --help) sed -n '2,11p' "$0"; exit 0 ;;
    *) echo "sdd-phase-status: opção desconhecida: $1" >&2; exit 2 ;;
  esac
done
if [ -z "$repo" ]; then
  repo="$(git remote get-url origin | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
if [ -z "$epic" ]; then
  epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty')"
  [ -n "$epic" ] || { echo "sdd-phase-status: nenhum épico aberto em $repo" >&2; exit 1; }
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
tab="$(printf '\t')"

gh api "repos/$repo/issues/$epic/sub_issues?per_page=100" \
  --jq '.[] | [.number, .state, .title] | @tsv' > "$tmp/subs"
# PRs recentes com "Closes/Fixes/Resolves #N" no corpo: número, estado, sha, N.
gh api "repos/$repo/pulls?state=all&per_page=100" --jq '.[] |
  [.number, (if .merged_at then "mergeado" elif .state == "open" then "aberto" else "fechado" end),
   .head.sha, .base.ref,
   ((.body // "") | [scan("(?i)(?:closes|fixes|resolves) #([0-9]+)")[0]] | join(","))] | @tsv' > "$tmp/prs"

# shellcheck disable=SC2016 # $c, $n... são variáveis do jq
ci() { # resumo do CI de um sha: "verde (5)", "FALHOU: a, b", "pendente (2/5)"
  gh api "repos/$repo/commits/$1/check-runs?per_page=100" --jq '
    [.check_runs[]] as $c | ($c | length) as $n
    | ([$c[] | select(.status != "completed")] | length) as $p
    | [$c[] | select(.status == "completed" and (.conclusion | IN("success", "skipped", "neutral") | not)) | .name] as $f
    | if $n == 0 then "sem checks" elif ($f | length) > 0 then "FALHOU: " + ($f | join(", "))
      elif $p > 0 then "pendente (\($p)/\($n))" else "verde (\($n))" end'
}

{
  echo "## Estado da fase ($(date +%Y-%m-%d))"
  echo
  echo "| Ticket | PR | Base | Estado | CI |"
  echo "|---|---|---|---|---|"
  while IFS="$tab" read -r n st t; do
    pr="$(awk -F'\t' -v n="$n" '{ split($5, a, ","); for (i in a) if (a[i] == n) { print; exit } }' "$tmp/prs")"
    if [ -z "$pr" ]; then
      [ "$st" = closed ] && s="fechado sem PR" || s="sem PR"
      echo "| #$n $t | — | — | $s | — |"
      [ -n "$next" ] || [ "$st" = closed ] || next="#$n $t"
      continue
    fi
    IFS="$tab" read -r pn ps sha base _ <<EOF
$pr
EOF
    c="—"; [ "$ps" = fechado ] || c="$(ci "$sha")"
    echo "| #$n $t | #$pn | \`$base\` | $ps | $c |"
  done < "$tmp/subs"
  echo
  if [ -f specs/ANALYSIS.md ]; then
    # Pendente: linha Dn sem "respondida", numa seção cujo título não diz "respondidas".
    d="$(awk -F'|' '/^## / { ans = /respondidas/ } /^\| *D[0-9]+ *\|/ && !ans && !/respondida/ { gsub(/ /, "", $2); print $2 }' \
      specs/ANALYSIS.md | paste -sd, -)"
    echo "**Decisões pendentes:** ${d:-nenhuma}"
    echo
  fi
  main="$(ci "$(gh api "repos/$repo/commits/main" --jq .sha)")"
  echo "CI da \`main\`: $main."
  echo
  [ -n "$next" ] || next="PR de fechamento da fase"
  echo "**Próximo passo:** $next"
} > "$tmp/body.md"

if [ "$post" -eq 1 ]; then
  gh api "repos/$repo/issues/$epic/comments" -F body=@"$tmp/body.md" --jq .html_url
else
  cat "$tmp/body.md"
fi
