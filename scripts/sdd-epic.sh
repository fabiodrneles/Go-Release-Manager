#!/bin/sh
# sdd-epic: cria o épico de uma fase do ROADMAP e os tickets como sub-issues
#.
#
# Uso: sdd-epic.sh [--repo DONO/REPO] [--tickets DIR] [--dry-run] FASE
#   Lê "## Fase FASE — <nome> (Pn)" (o fechamento acrescenta "→ `vX.Y.Z`") e as tarefas "- [ ] **Tn** <título> — <IDs>".
#   --tickets DIR  DIR/Tn.md, se existir, substitui o corpo gerado do ticket. As
#                  primeiras linhas podem ser "title: <título>" e "labels: a,b",
#                  seguidas de uma linha em branco.
#   --dry-run      só mostra o plano e grava os corpos em ./.sdd-epic/ (nada no GitHub)
# Ordem (templates.md): labels → épico → tickets com "Épico #M" → sub-issues →
# corpo do épico com os números. Saída: "épico #M" e uma linha "Tn #n título".
set -eu

repo="" dir="" dry=0 phase=""
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --tickets) dir="${2:?}"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    -h | --help) sed -n '2,14p' "$0"; exit 0 ;;
    -*) echo "sdd-epic: opção desconhecida: $1" >&2; exit 2 ;;
    *) phase="$1"; shift ;;
  esac
done
[ -n "$phase" ] || { sed -n '2,14p' "$0"; exit 2; }
r=specs/ROADMAP.md
[ -f "$r" ] || { echo "sdd-epic: $r não existe" >&2; exit 1; }
if [ -z "$repo" ]; then
  url="$(git remote get-url origin)"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi

header="$(grep -E "^## Fase $phase( |$)" "$r" | head -n 1)"
[ -n "$header" ] || { echo "sdd-epic: Fase $phase não está em $r" >&2; exit 1; }
name="$(printf '%s\n' "$header" | sed -E 's/^## Fase [0-9]+ — //; s/ *\(P[0-9]\).*//; s/ *→.*//')"
version="$(printf '%s\n' "$header" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -n 1 || true)"
prio="$(printf '%s\n' "$header" | grep -oE '\(P[0-9]\)' | tr -d '()' || true)"
title="Fase $phase — $name${version:+ ($version)}"

out=.sdd-epic; [ "$dry" -eq 1 ] || out="$(mktemp -d)"
mkdir -p "$out"
[ "$dry" -eq 1 ] || trap 'rm -rf "$out"' EXIT

# Tarefas abertas da fase: "Tn<TAB>título<TAB>IDs".
tab="$(printf '\t')"
awk -v h="$header" '$0 == h { on = 1; next } /^## / { on = 0 }
  on && /^- \[ \] \*\*T[0-9]+\*\*/ {
    t = $0; sub(/^- \[ \] \*\*/, "", t); id = t; sub(/\*\*.*/, "", id); sub(/^T[0-9]+\*\* */, "", t)
    ids = ""; i = index(t, " — "); if (i) { ids = substr(t, i + 5); t = substr(t, 1, i - 1) }
    print id "\t" t "\t" ids }' "$r" > "$out/tasks"
[ -s "$out/tasks" ] || { echo "sdd-epic: a Fase $phase não tem tarefas abertas" >&2; exit 1; }

# ACs de "002 FR-1, AC-1 a AC-3; 003 AC-2" → "- [ ] 002 AC-1 a AC-3", "- [ ] 003 AC-2".
acs() { printf '%s\n' "$1" | tr ';' ',' | tr ',' '\n' | awk '
  { for (i = 1; i <= NF; i++) if ($i ~ /^[0-9][0-9][0-9]$/) cur = $i }
  /AC-[0-9]+/ { s = $0; sub(/^ *([0-9][0-9][0-9] )?/, "", s); sub(/ +$/, "", s); print "- [ ] " cur " " s }'; }

: > "$out/plan"
while IFS="$tab" read -r id t ids; do
  f="$out/$id.md" tt="$id: $t" labels="fase-$phase,tipo:feature${prio:+,$prio}"
  if [ -n "$dir" ] && [ -f "$dir/$id.md" ]; then
    tt="$(sed -n 's/^title: //p' "$dir/$id.md" | head -n 1)"; [ -n "$tt" ] || tt="$id: $t"
    l="$(sed -n 's/^labels: //p' "$dir/$id.md" | head -n 1)"; [ -z "$l" ] || labels="fase-$phase,$l"
    awk 'h == 0 && /^(title|labels): / { next } h == 0 && /^$/ { h = 1; next } { h = 1; print }' "$dir/$id.md" > "$f"
  else
    {
      echo "## Contexto"; echo; echo "Tarefa $id da Fase $phase em \`specs/ROADMAP.md\`."; echo
      echo "## O que fazer"; echo; echo "- $t"; echo
      echo "## Critérios de aceite"; echo
      a="$(acs "$ids")"; if [ -n "$a" ]; then echo "$a"; else echo "- [ ] <critério verificável>"; fi
    } > "$f"
  fi
  { echo; echo "**Spec(s):** ${ids:-—} · **Épico:** #EPIC"; } >> "$f"
  printf '%s\t%s\t%s\n' "$id" "$tt" "$labels" >> "$out/plan"
done < "$out/tasks"

if [ "$dry" -eq 1 ]; then
  echo "épico: \"$title\" labels=épico,fase-$phase ($repo)"
  while IFS="$tab" read -r id tt labels; do echo "$id \"$tt\" labels=$labels"; done < "$out/plan"
  echo "sdd-epic: plano em $out/ (nada criado)"
  exit 0
fi

# Labels que faltam (criar issue com label inexistente falha na hora de anexar).
gh api "repos/$repo/labels?per_page=100" --paginate --jq '.[].name' > "$out/have"
cut -f3 "$out/plan" | tr ',' '\n' | { cat; printf 'épico\nfase-%s\n' "$phase"; } | sort -u |
  while IFS= read -r l; do
    grep -qxF "$l" "$out/have" || { gh api "repos/$repo/labels" -f name="$l" --silent && echo "label criada: $l"; }
  done

epic="$(gh api "repos/$repo/issues" -f title="$title" -f body="(em preparação)" \
  -f 'labels[]=épico' -f "labels[]=fase-$phase" --jq .number)"
echo "épico #$epic"
: > "$out/list"
while IFS="$tab" read -r id tt labels; do
  sed -i.bak "s/#EPIC/#$epic/" "$out/$id.md"
  set --; for l in $(echo "$labels" | tr ',' ' '); do set -- "$@" -f "labels[]=$l"; done
  res="$(gh api "repos/$repo/issues" -f title="$tt" -F body=@"$out/$id.md" "$@" --jq '"\(.number) \(.id)"')"
  n="${res% *}" iid="${res#* }"
  gh api "repos/$repo/issues/$epic/sub_issues" -F sub_issue_id="$iid" --silent
  echo "$id #$n $tt"
  echo "$((1 + $(wc -l < "$out/list"))). #$n $tt" >> "$out/list"
done < "$out/plan"

{
  echo "## Objetivo"; echo; echo "Fase $phase do ROADMAP: $name.${version:+ Versão: \`$version\`.}"; echo
  echo "## Tarefas (ordem sugerida de revisão e merge)"; echo
  echo "As tarefas são as sub-issues deste épico. Ordem sugerida:"; echo
  cat "$out/list"; echo
  echo "## Critério de pronto"; echo
  echo "- Todos os PRs mergeados com CI verde."
  echo "- PR de fechamento mergeado (specs, ROADMAP, CHANGELOG)."
  [ -z "$version" ] || echo "- Tag \`$version\` publicada pelo dono."
} > "$out/epic.md"
gh api -X PATCH "repos/$repo/issues/$epic" -F body=@"$out/epic.md" --silent
echo "sdd-epic: épico #$epic com $(wc -l < "$out/list" | tr -d ' ') sub-issue(s)"
