#!/bin/sh
# sdd-mark: edita os arquivos de status do SDD de forma segura.
#
# Uso:
#   sdd-mark.sh decide [--date AAAA-MM-DD] D1=a D2=b ...
#       marca as decisões como respondidas em specs/ANALYSIS.md; quando não
#       sobra nenhuma em aberto, move as specs Draft para Approved (cabeçalho e
#       specs/README.md) e marca a Fase 0 do ROADMAP.
#   sdd-mark.sh close [--date AAAA-MM-DD] vX.Y.Z
#       PR de fechamento da versão vX.Y.Z (a do go-release-manager, #198): marca as
#       tarefas da fase atual do ROADMAP (a primeira com tarefa aberta; num ROADMAP
#       antigo, a fase "→ `vX.Y.Z`") e grava "→ `vX.Y.Z`" no cabeçalho dela; move para
#       Done as specs citadas que não têm tarefa aberta em outra fase (as demais vão
#       para In Progress) e abre "## [X.Y.Z] - data" no CHANGELOG.
# Saída: uma linha por arquivo alterado e os avisos. Nunca deixa um arquivo vazio
# ou menor do que a edição permite; em erro, nada é gravado (códigos: 0 ok, 1 erro, 2 uso).
set -eu

die() { echo "sdd-mark: $*" >&2; exit 1; }
usage() { sed -n '2,15p' "$0"; exit 2; }

[ $# -gt 0 ] || usage
cmd="$1"; shift
date="$(date +%Y-%m-%d)"
args=""
while [ $# -gt 0 ]; do
  case "$1" in
    --date) date="${2:?}"; shift 2 ;;
    -h | --help) usage ;;
    -*) echo "sdd-mark: opção desconhecida: $1" >&2; exit 2 ;;
    *) args="$args $1"; shift ;;
  esac
done
[ -f specs/README.md ] || die "rode na raiz do repositório (specs/README.md não existe)"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
: > "$tmp/changed"; : > "$tmp/warn"
warn() { echo "aviso: $*" >> "$tmp/warn"; }

# stage ARQUIVO: grava $tmp/out sobre o ARQUIVO só se a saída não estiver vazia e
# mantiver o mesmo número de linhas (MIN_DELTA permite inserir linhas). Todas as
# edições são preparadas antes; commit_all copia só no fim.
stage() {
  sf="$1" delta="${2:-0}"
  [ -s "$tmp/out" ] || die "edição de $sf gerou arquivo vazio; nada gravado"
  base="$(current "$sf")"
  a="$(wc -l < "$base")" b="$(wc -l < "$tmp/out")"
  if [ "$b" -lt "$a" ] || [ "$b" -gt $((a + delta)) ]; then die "edição de $sf mudou $a→$b linhas; nada gravado"; fi
  if ! cmp -s "$base" "$tmp/out"; then
    mkdir -p "$tmp/staged/$(dirname "$sf")"
    mv "$tmp/out" "$tmp/staged/$sf"
    echo "$sf" >> "$tmp/changed"
  fi
}
# current ARQUIVO: a versão já preparada, se houver, senão a do disco.
current() { if [ -f "$tmp/staged/$1" ]; then echo "$tmp/staged/$1"; else echo "$1"; fi; }
commit_all() {
  sort -u "$tmp/changed" | while IFS= read -r cf; do cp "$tmp/staged/$cf" "$cf"; done
  sort -u "$tmp/changed" | sed 's/^/alterado: /'
  cat "$tmp/warn"
}

spec_file() { for s in specs/"$1"-*/spec.md; do [ -f "$s" ] && echo "$s" && return; done; }

# set_status NNN STATUS NOTA: troca o status no cabeçalho da spec e em specs/README.md.
set_status() {
  ss="$(spec_file "$1")"; [ -n "$ss" ] || { warn "spec $1 não existe"; return; }
  awk -v st="$2" -v note="$3" '!done && /^- \*\*Status:\*\*/ { print "- **Status:** " st (note == "" ? "" : " — " note); done = 1; next } { print }' \
    "$(current "$ss")" > "$tmp/out"
  stage "$ss"
  awk -F'|' -v id="$1" -v st="$2" 'BEGIN { OFS = "|" }
    { c = $2; gsub(/ /, "", c) }
    c == id && NF > 3 { $(NF - 1) = " " st " " } { print }' "$(current specs/README.md)" > "$tmp/out"
  stage specs/README.md
}

spec_status() { awk '/^- \*\*Status:\*\*/ { sub(/^- \*\*Status:\*\* */, ""); if ($0 ~ /^In Progress/) print "In Progress"; else { split($0, w, /[^A-Za-z]/); print w[1] } exit }' "$(current "$1")"; }

case "$cmd" in
decide)
  [ -n "$args" ] || usage
  f=specs/ANALYSIS.md; [ -f "$f" ] || die "$f não existe"
  for kv in $args; do
    id="${kv%%=*}" ans="${kv#*=}"
    case "$kv" in D[0-9]*=?*) ;; *) die "esperado Dn=resposta, veio: $kv" ;; esac
    grep -qE "^\| *$id *\|" "$(current "$f")" || die "$id não está na tabela de decisões de $f"
    awk -v id="$id" -v ans="$ans" '
      { c = $0; sub(/^\| */, "", c); sub(/ *\|.*/, "", c) }
      /^\|/ && c == id && $0 !~ /respondida/ {
        sub(/ *\| *$/, ""); r = ans; if (r ~ /^[a-z]$/) r = "(" r ")"
        print $0 " — **respondida: " r "** |"; next }
      { print }' "$(current "$f")" > "$tmp/out"
    stage "$f"
  done
  open="$(grep -E '^\| *D[0-9]+ *\|' "$(current "$f")" | grep -vc respondida || true)"
  if [ "$open" -gt 0 ]; then
    warn "$open decisão(ões) ainda em aberto: specs continuam Draft"
  else
    # O título da seção não muda: âncoras que apontam para ela continuariam válidas.
    for s in specs/[0-9][0-9][0-9]-*/spec.md; do
      [ -f "$s" ] || continue
      [ "$(spec_status "$s")" = Draft ] || continue
      set_status "$(basename "$(dirname "$s")" | cut -c1-3)" Approved "decisões respondidas pelo dono em $date"
    done
    for s in specs/[0-9][0-9][0-9]-*/spec.md; do
      [ -f "$s" ] || continue
      awk '/^## Decis/ { on = 1; next } /^## / { on = 0 } on && /Pendente/ { found = 1 } END { exit !found }' "$s" &&
        warn "$s: a seção \"Decisões\" ainda diz \"Pendente\"; revise o texto à mão"
    done
    if [ -f specs/ROADMAP.md ]; then
      awk '/^## Fase 0/ { on = 1 } /^## / && !/^## Fase 0/ { on = 0 } on { sub(/^- \[ \]/, "- [x]") } { print }' \
        "$(current specs/ROADMAP.md)" > "$tmp/out"; stage specs/ROADMAP.md
    fi
  fi
  commit_all
  ;;
close)
  # shellcheck disable=SC2086 # $args é uma lista de palavras
  set -- $args
  [ $# -eq 1 ] || usage
  v="$1"; case "$v" in v[0-9]*.[0-9]*.[0-9]*) ;; *) die "versão inválida: $v (use vX.Y.Z)" ;; esac
  r=specs/ROADMAP.md; [ -f "$r" ] || die "$r não existe"
  # A fase: a que já aponta para a versão (ROADMAP antigo) ou a primeira com tarefa aberta.
  h="$(grep -E "^## Fase .*\`$v\`" "$r" | head -n 1)" || true
  [ -n "$h" ] || h="$(awk '/^## Fase [0-9]+/ { c = $0; next } /^## / { c = "" } c != "" && /^- \[ \]/ { print c; exit }' "$r")"
  [ -n "$h" ] || die "nenhuma fase do ROADMAP tem tarefa aberta nem aponta para \`$v\`"
  # Specs citadas nas tarefas da fase, e specs com tarefa aberta em outra fase.
  awk -v h="$h" '/^## / { on = ($0 == h) } on && /^- \[/' "$r" | grep -oE '(^| )[0-9]{3} ' | tr -d ' ' | sort -u > "$tmp/phase"
  awk -v h="$h" '/^## / { on = ($0 == h) } !on && /^- \[ \]/' "$r" | grep -oE '(^| )[0-9]{3} ' | tr -d ' ' | sort -u > "$tmp/open"
  # Marca as tarefas e registra a versão entregue no cabeçalho (se ainda não tem).
  awk -v h="$h" -v v="\`$v\`" '/^## / { on = ($0 == h); if (on && index($0, v) == 0) $0 = $0 " → " v } on { sub(/^- \[ \]/, "- [x]") } { print }' "$r" > "$tmp/out"
  stage "$r"
  [ -s "$tmp/phase" ] || warn "nenhuma tarefa da fase cita uma spec (NNN FR-n); status das specs não mudou"
  while IFS= read -r nnn; do
    if grep -qx "$nnn" "$tmp/open"; then
      set_status "$nnn" "In Progress" "parte entregue na \`$v\`"
      warn "$nnn tem tarefas abertas em outra fase: ficou In Progress (o sdd-check --strict exige todos os ACs citados)"
    else
      set_status "$nnn" Done "entregue na \`$v\`"
    fi
    s="$(spec_file "$nnn")"
    [ -z "$s" ] || ! grep -q '^## Estado atual' "$s" || warn "$s tem \"Estado atual\": revise o texto à mão"
  done < "$tmp/phase"
  c=CHANGELOG.md
  if [ ! -f "$c" ]; then
    warn "$c não existe"
  elif grep -qF "## [${v#v}]" "$c"; then
    warn "$c já tem a seção [${v#v}]"
  else
    grep -q '^## \[Unreleased\]' "$c" || die "$c sem a seção [Unreleased]"
    body="$(awk '/^## \[Unreleased\]/ { on = 1; next } /^## / { on = 0 } on && NF' "$c")"
    [ -n "$body" ] || warn "[Unreleased] está vazio: escreva as entradas da versão em $c"
    awk -v h="## [${v#v}] - $date" '{ print } /^## \[Unreleased\]/ { print ""; print h }' "$c" > "$tmp/out"
    stage "$c" 2
    ! grep -qE '^\[Unreleased\]: ' "$c" || warn "$c tem links de comparação no rodapé: atualize-os"
  fi
  commit_all
  [ ! -f scripts/sdd-check.sh ] || sh scripts/sdd-check.sh | tail -n 1
  ;;
*) usage ;;
esac
