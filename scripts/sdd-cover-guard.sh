#!/bin/sh
# sdd-cover-guard: válvula de resfriamento do cache do Go para o gate de cobertura
# (sdd-kit, spec 010 FR-1, #172).
#
# Uso: sdd-cover-guard.sh PERFIL -- COMANDO...
#   Roda COMANDO (o go test que grava PERFIL) e confere o perfil antes de o gate
#   calcular a cobertura. Um perfil é coerente quando, em cada arquivo, os blocos não
#   se sobrepõem e nenhum passa da última linha do arquivo. Com o cache quente, o go
#   pode misturar no perfil blocos de uma versão antiga de um arquivo (Go 1.25+ com
#   -coverpkg e um package main), e eles contam como não cobertos.
#   Perfil incoerente: avisa numa linha, roda COMANDO de novo num GOCACHE frio e
#   descartável e confere outra vez. Sai com o código do COMANDO, ou 1 se o perfil
#   continuar incoerente com o cache frio. Perfil coerente: nada além do COMANDO.
# Variável: SDD_COVER_GUARD=off desliga a conferência.
set -u

if [ $# -lt 3 ] || [ "$2" != "--" ]; then
  echo "uso: sdd-cover-guard.sh PERFIL -- COMANDO..." >&2
  exit 2
fi
prof="$1"
shift 2

# Imprime o primeiro arquivo do perfil com blocos incoerentes; código 0 se achou.
incoherent() {
  [ -f "$prof" ] || return 1
  mod="$(go list -m -f '{{.Path}}' 2> /dev/null || true)"
  # arquivo, início (linha*1e6+coluna), fim, linha final; blocos repetidos (um por
  # binário de teste) contam uma vez.
  awk 'NR > 1 {
    i = index($1, ":"); f = substr($1, 1, i - 1); r = substr($1, i + 1)
    if ((f, r) in seen) next
    seen[f, r] = 1
    split(r, b, ","); split(b[1], s, "."); split(b[2], e, ".")
    printf "%s %d %d %d\n", f, s[1] * 1000000 + s[2], e[1] * 1000000 + e[2], e[1]
  }' "$prof" | sort -k1,1 -k2,2n | awk -v mod="$mod" '
    $1 != f { f = $1; end = 0 }
    $2 < end { print f; found = 1; exit }
    { if ($3 > end) end = $3
      if ($4 > last[f]) last[f] = $4 }
    END {
      if (found) exit 0
      for (f in last) {
        p = f
        if (mod != "" && index(f, mod "/") == 1) p = substr(f, length(mod) + 2)
        n = 0
        while ((getline line < p) > 0) n++
        close(p)
        if (n > 0 && last[f] > n) { print f; exit 0 }
      }
      exit 1
    }'
}

"$@" || exit $?
[ "${SDD_COVER_GUARD:-on}" = off ] && exit 0
f="$(incoherent)" || exit 0

echo "sdd-cover-guard: o perfil tem blocos de versões diferentes de $f (cache quente do go); refazendo com um cache frio" >&2
cold="$(mktemp -d)"
GOCACHE="$cold" "$@"
rc=$?
chmod -R u+w "$cold" 2> /dev/null
rm -rf "$cold"
[ "$rc" -eq 0 ] || exit "$rc"
if f="$(incoherent)"; then
  echo "sdd-cover-guard: o perfil continua incoerente em $f mesmo com o cache frio" >&2
  exit 1
fi
echo "sdd-cover-guard: perfil refeito com o cache frio" >&2
exit 0
