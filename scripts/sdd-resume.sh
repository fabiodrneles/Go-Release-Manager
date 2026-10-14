#!/bin/sh
# sdd-resume: a rotina "Retomar o trabalho" num comando só (sdd-kit, spec 014):
# mostra o checkpoint do épico aberto, entra na branch dele (se a árvore estiver
# limpa), lista os PRs abertos com o resumo do CI e as sub-issues abertas do épico.
#
# Uso: sdd-resume.sh [--repo DONO/REPO]
# Só leitura no GitHub (REST). Nunca descarta alterações locais: com árvore suja
# ou commits não enviados na branch, não troca de branch e diz por quê.
# Sem gh, sem rede ou sem épico aberto, avisa e sai com 0: nunca bloqueia a sessão.
# A saída sempre tem uma linha "Próximo:" (#195): a do checkpoint, a fase já aprovada
# no ROADMAP ou "perguntar ao dono". A sessão faz o Próximo e nada além.
set -u

repo=""
[ "${1:-}" != --repo ] || { repo="${2:?}"; shift 2; }
[ $# -eq 0 ] || { sed -n '2,9p' "$0"; exit 2; }
here="$(cd "$(dirname "$0")" && pwd)"
ask="Próximo: perguntar ao dono o que fazer (nenhuma fase aprovada em aberto); não comece nada antes da resposta"

command -v gh >/dev/null 2>&1 || { echo "sdd-resume: gh ausente"; exit 0; }
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2>/dev/null || true)"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
[ -n "$repo" ] || { echo "sdd-resume: repositório desconhecido (use --repo)"; exit 0; }

# 1. Checkpoint (a detecção do épico fica no sdd-checkpoint.sh).
out="$(sh "$here/sdd-checkpoint.sh" --repo "$repo" show 2>/dev/null)" || true
printf '%s\n' "$out"
epic="$(printf '%s\n' "$out" | sed -n '1s/^[^#]*#\([0-9][0-9]*\) (.*$/\1/p')"
if [ -z "$epic" ]; then
  # Sem épico aberto (a fase anterior fechou): o próximo passo vem do ROADMAP (#191).
  if next="$(sh "$here/sdd-next-phase.sh" 2> /dev/null)"; then
    t="$(printf '\t')"; n="${next%%"$t"*}"
    echo "Próximo: abrir a ${next#*"$t"} (já aprovada no ROADMAP): sh scripts/sdd-epic.sh $n"
  else
    echo "$ask"
  fi
  exit 0
fi

# 2. Branch do checkpoint.
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "sdd-resume: fora de um repositório git; sem troca de branch"
elif ! git fetch -q origin 2>/dev/null; then
  echo "sdd-resume: sem acesso ao origin (offline?); sem troca de branch"
else
  # shellcheck disable=SC2016 # crases são Markdown no checkpoint
  want="$(printf '%s\n' "$out" | sed -n 's/^- \*\*Branch:\*\* `\([^`]*\)` em .*$/\1/p' | head -n 1)"
  cur="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '?')"
  if [ -z "$want" ] || [ "$want" = "?" ] || [ "$want" = HEAD ]; then
    echo "sdd-resume: o checkpoint não nomeia uma branch; fico em $cur"
  elif [ "$want" = "$cur" ]; then
    behind="$(git rev-list --count "$want..origin/$want" 2>/dev/null || echo 0)"
    echo "sdd-resume: já em $cur (atrás do origin: $behind)"
  elif ! git rev-parse -q --verify "refs/remotes/origin/$want" >/dev/null; then
    echo "sdd-resume: $want não existe no origin; fico em $cur"
  elif [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    echo "sdd-resume: árvore com alterações locais; não troco de $cur para $want"
  elif git rev-parse -q --verify "refs/heads/$want" >/dev/null &&
    [ "$(git rev-list --count "origin/$want..$want" 2>/dev/null || echo 0)" -gt 0 ]; then
    echo "sdd-resume: $want local tem commits não enviados; não troco de $cur"
  elif git checkout -q -B "$want" "origin/$want" 2>/dev/null; then
    echo "sdd-resume: agora em $want ($(git rev-parse --short HEAD))"
  else
    echo "sdd-resume: não consegui entrar em $want; fico em $cur"
  fi
fi

# 3. PRs abertos, com o resumo do CI.
echo "PRs abertos:"
prs="$(gh api "repos/$repo/pulls?state=open&per_page=30" --jq '.[] | "\(.number)\t\(.title)"' 2>/dev/null || true)"
if [ -z "$prs" ]; then
  echo "  nenhum"
else
  tab="$(printf '\t')"
  printf '%s\n' "$prs" | while IFS="$tab" read -r n title; do
    ci="$(sh "$here/sdd-ci.sh" --repo "$repo" --no-wait "#$n" 2>/dev/null || true)"
    bad="$(printf '%s\n' "$ci" | sed -n 's/^FALHA //p' | paste -sd ',' - | sed 's/,/, /g')"
    np="$(printf '%s\n' "$ci" | grep -c '^pendente ' || true)"
    no="$(printf '%s\n' "$ci" | grep -c '^ok ' || true)"
    if [ -n "$bad" ]; then s="CI vermelho: $bad"
    elif [ "$np" -gt 0 ]; then s="CI pendente ($np)"
    elif [ "$no" -gt 0 ]; then s="CI verde ($no)"
    else s="sem checks"; fi
    echo "  #$n $title: $s"
  done
fi

# 4. Sub-issues abertas do épico.
echo "Issues abertas do épico #$epic:"
subs="$(gh api "repos/$repo/issues/$epic/sub_issues?per_page=100" --jq '.[] | select(.state == "open") | "  #\(.number) \(.title)"' 2>/dev/null || true)"
printf '%s\n' "${subs:-  nenhuma}"
# Checkpoint sem "Próximo" (ou épico sem checkpoint): perguntar, nunca adivinhar.
printf '%s\n' "$out" | grep -q 'Próximo' || echo "$ask"
exit 0
