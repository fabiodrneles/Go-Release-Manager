#!/bin/sh
# sdd-check: rastreabilidade entre specs, testes e ROADMAP (sdd-kit, spec 006 FR-2).
#
#   1. o status de cada spec é o mesmo no cabeçalho e em specs/README.md;
#   2. todo AC-n de spec In Progress ou Done é citado em algum arquivo
#      fora de specs/ e dos .md, no formato "NNN AC-n" ou "NNN/AC-n";
#   3. todo ID citado no ROADMAP ("NNN FR-n", "NNN NFR-n", "NNN AC-n") existe;
#   4. toda linha ADDED ou MODIFIED da seção "Mudanças" de uma spec aponta para
#      um ID que existe nela (REMOVED pode citar um ID que já saiu).
#
# Uso: sdd-check.sh [--strict] [DIRETÓRIO]
# Sem --strict, só avisa e sai com 0; com --strict, sai com 1 se houver aviso.
set -eu

strict=0
root=.
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) strict=1 ;;
    -h | --help) sed -n '2,12p' "$0"; exit 0 ;;
    -*) echo "sdd-check: opção desconhecida: $1" >&2; exit 2 ;;
    *) root="$1" ;;
  esac
  shift
done
cd "$root"

if [ ! -f specs/README.md ]; then
  echo "sdd-check: specs/README.md não existe; nada a verificar"
  exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
: > "$tmp/warnings"
warn() { echo "aviso: $*" >> "$tmp/warnings"; }

# Onde procurar citações de AC: tudo fora de specs/ que não seja Markdown.
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  git ls-files -co --exclude-standard
else
  find . -type f | sed 's#^\./##'
fi | grep -v '^specs/' | grep -v '\.md$' | grep -v '^\.git/' > "$tmp/files" || true

status_of() {
  awk '/^- \*\*Status:\*\*/ {
    sub(/^- \*\*Status:\*\* */, "")
    if ($0 ~ /^In Progress/) print "In Progress"
    else { split($0, w, /[^A-Za-z]/); print w[1] }
    exit
  }' "$1"
}

readme_status_of() {
  awk -F'|' -v id="$1" '{ c = $2; gsub(/ /, "", c) } c == id {
    s = $(NF - 1); gsub(/^ +| +$/, "", s); print s; exit
  }' specs/README.md
}

cites() { # cites NNN N -> número de arquivos que citam "NNN AC-N" ou "NNN/AC-N"
  tr '\n' '\0' < "$tmp/files" |
    xargs -0 grep -lE "(^|[^0-9])$1[ /]AC-$2([^0-9]|\$)" /dev/null 2>/dev/null | wc -l | tr -d ' '
}

echo "| Spec | Status | AC | Arquivos que citam |"
echo "|---|---|---|---|"
for spec in specs/[0-9][0-9][0-9]-*/spec.md; do
  [ -f "$spec" ] || continue
  nnn="$(basename "$(dirname "$spec")" | cut -c1-3)"
  st="$(status_of "$spec")"
  rst="$(readme_status_of "$nnn")"
  if [ -z "$rst" ]; then
    warn "$nnn: spec ausente de specs/README.md"
  elif [ "$st" != "$rst" ]; then
    warn "$nnn: status '$st' no cabeçalho e '$rst' em specs/README.md"
  fi
  awk '/^## Mudanças/ { on = 1; next } /^## / { on = 0 } on' "$spec" |
    grep -oE '^- (ADDED|MODIFIED) (FR|NFR|AC)-[0-9]+' | sed 's/^- [A-Z]* //' | sort -u > "$tmp/changes" || true
  while read -r id; do
    grep -qE "\*\*$id( [^*]*)?\*\*" "$spec" || warn "$nnn: \"Mudanças\" cita $id, que não existe na spec"
  done < "$tmp/changes"
  case "$st" in "In Progress" | Done) ;; *) continue ;; esac
  grep -oE '\*\*AC-[0-9]+( [^*]*)?\*\*' "$spec" | sed 's/^\*\*AC-\([0-9]*\).*/\1/' | sort -n -u > "$tmp/acs"
  while read -r n; do
    k="$(cites "$nnn" "$n")"
    echo "| $nnn | $st | AC-$n | $k |"
    [ "$k" -gt 0 ] || warn "$nnn AC-$n sem teste que o cite (\"$nnn AC-$n\")"
  done < "$tmp/acs"
done

if [ -f specs/ROADMAP.md ]; then
  for ref in $(grep -oE '[0-9]{3} (FR|NFR|AC)-[0-9]+' specs/ROADMAP.md | tr ' ' '_' | sort -u); do
    nnn="${ref%%_*}"
    id="${ref#*_}"
    spec=""
    for f in specs/"$nnn"-*/spec.md; do [ -f "$f" ] && spec="$f" && break; done
    if [ -z "$spec" ]; then
      warn "ROADMAP cita $nnn $id, mas a spec $nnn não existe"
    elif ! grep -qE "\*\*$id( [^*]*)?\*\*" "$spec"; then
      warn "ROADMAP cita $nnn $id, que não existe em $spec"
    fi
  done
fi

echo
cat "$tmp/warnings"
warnings="$(wc -l < "$tmp/warnings" | tr -d ' ')"
if [ "$warnings" -eq 0 ]; then
  echo "sdd-check: tudo rastreável"
  exit 0
fi
echo "sdd-check: $warnings aviso(s)"
[ "$strict" -eq 0 ] || exit 1
exit 0
