#!/bin/sh
# sdd-wait: espera, sem LLM, até uma condição valer, para que o agente (ou um
# shell em segundo plano) nunca faça polling à mão. Imprime uma linha no fim.
#
# Uso: sdd-wait.sh [--repo DONO/REPO] [--interval S] [--timeout S] CONDIÇÃO ALVO
#   pr-merged '#N'      espera o PR ser mergeado ou fechado
#   ci SHA|'#N'         espera os checks do commit/PR (delega ao sdd-ci.sh)
#   issue-closed '#N'   espera a issue ser fechada
#   merged-any          vigia: espera qualquer PR aberto (na hora em que começa) ser
#                       mergeado ou fechado; uma consulta por rodada (spec 016)
#   --interval  segundos entre consultas (padrão 30; 60 no merged-any)
#   --timeout   tempo máximo em segundos (padrão 3600)
# Códigos: 0 a condição valeu (PR mergeado, CI verde, issue fechada);
#          1 falhou (PR fechado sem merge, CI vermelho); 2 tempo esgotado;
#          3 uso/erro (no merged-any, também sem PR aberto para vigiar).
# Requer gh; só chamadas REST (gh api).
set -eu

repo="" interval="" timeout=3600
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --interval) interval="${2:?}"; shift 2 ;;
    --timeout) timeout="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,17p' "$0"; exit 0 ;;
    -*) echo "sdd-wait: opção desconhecida: $1" >&2; exit 3 ;;
    *) break ;;
  esac
done
cond="${1:-}" target="${2:-}"
if [ "$cond" = merged-any ] && [ $# -eq 1 ]; then
  : # o vigia não tem alvo
elif [ -z "$cond" ] || [ -z "$target" ] || [ $# -ne 2 ]; then
  echo "sdd-wait: uso: sdd-wait.sh [opções] pr-merged|ci|issue-closed ALVO | merged-any" >&2
  exit 3
fi
# Intervalo padrão: 60 s no vigia (spec 016 NFR-1, poupa a cota da API), 30 s no resto.
if [ -z "$interval" ]; then if [ "$cond" = merged-any ]; then interval=60; else interval=30; fi; fi
case "$interval$timeout" in *[!0-9]*) echo "sdd-wait: --interval e --timeout são números de segundos" >&2; exit 3 ;; esac

if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2>/dev/null)" || { echo "sdd-wait: sem remote origin; use --repo" >&2; exit 3; }
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi

if [ "$cond" = ci ]; then
  # O sdd-ci.sh já espera e resume; só repassa o código (0 verde, 1 vermelho, 2 tempo).
  rc=0
  sh "$(dirname "$0")/sdd-ci.sh" --repo "$repo" --timeout "$timeout" "$target" || rc=$?
  exit "$rc"
fi

if [ "$cond" = merged-any ]; then
  # Os PRs abertos agora; a cada rodada, uma consulta à lista: o que saiu dela é
  # consultado uma vez para saber se foi mergeado ou fechado (spec 016 FR-1, FR-2).
  list() { gh api "repos/$repo/pulls?state=open&per_page=100" --jq '.[].number'; }
  t="$(printf '\t')"
  watched="$(list)" || { echo "sdd-wait: não consegui listar os PRs de $repo" >&2; exit 3; }
  [ -n "$watched" ] || { echo "sdd-wait: nenhum PR aberto em $repo; nada a vigiar"; exit 3; }
  start="$(date +%s)"
  while :; do
    now=" $(list | tr '\n' ' ') "
    for n in $watched; do
      case "$now" in *" $n "*) continue ;; esac
      # Fora da lista (ou a lista falhou): só conclui com o PR de fato fechado.
      st="$(gh api "repos/$repo/pulls/$n" --jq '"\(.state)\t\(.merged_at // "-")\t\(.title)"' 2> /dev/null)" || continue
      case "$st" in closed"$t"*) ;; *) continue ;; esac
      st="${st#closed"$t"}"
      if [ "${st%%"$t"*}" = - ]; then echo "sdd-wait: #$n fechado sem merge"; exit 1; fi
      echo "sdd-wait: #$n mergeado: ${st#*"$t"}"; exit 0
    done
    if [ $(($(date +%s) - start)) -ge "$timeout" ]; then
      echo "sdd-wait: nenhum PR mergeado após ${timeout}s; tempo esgotado"
      exit 2
    fi
    [ "$interval" -eq 0 ] || sleep "$interval"
  done
fi

case "$cond" in
  pr-merged) path=pulls ;;
  issue-closed) path=issues ;;
  *) echo "sdd-wait: condição desconhecida: $cond" >&2; exit 3 ;;
esac
case "$target" in
  \#*[!0-9]* | \#) echo "sdd-wait: alvo deve ser '#N': $target" >&2; exit 3 ;;
  \#*) n="${target#\#}" ;;
  *) echo "sdd-wait: alvo deve ser '#N': $target" >&2; exit 3 ;;
esac

start="$(date +%s)"
while :; do
  st="$(gh api "repos/$repo/$path/$n" --jq '"\(.state) \(.merged_at // "-")"')"
  case "$cond:$st" in
    pr-merged:closed\ -) echo "sdd-wait: #$n fechado sem merge"; exit 1 ;;
    pr-merged:closed\ *) echo "sdd-wait: #$n mergeado"; exit 0 ;;
    issue-closed:closed*) echo "sdd-wait: #$n fechada"; exit 0 ;;
  esac
  if [ $(($(date +%s) - start)) -ge "$timeout" ]; then
    echo "sdd-wait: #$n ainda aberto após ${timeout}s; tempo esgotado"
    exit 2
  fi
  [ "$interval" -eq 0 ] || sleep "$interval"
done
