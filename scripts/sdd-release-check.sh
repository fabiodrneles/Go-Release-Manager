#!/bin/sh
# sdd-release-check: confere uma release Go antes e depois da tag (spec 009;
# template/go; issue sdd-kit#51, achado 10).
#
# Uso:
#   sdd-release-check.sh pre  vX.Y.Z [CMD]  simula o release: clone local da HEAD com a tag
#                                      temporária vX.Y.Z, CMD (padrão "make ci") e, se houver
#                                      .goreleaser.y*ml e goreleaser, "release --snapshot"
#   sdd-release-check.sh post vX.Y.Z   confere a release publicada: assets, checksums
#                                      e "go install MÓDULO@vX.Y.Z" + "--version"
# Saída: uma linha "ok|FALHA|pulado <etapa>" por etapa; numa falha, as últimas
# 20 linhas da saída da etapa. Códigos: 0 ok, 1 alguma etapa falhou, 2 uso.
# A tag temporária só existe num clone descartável; nada é publicado.
# shellcheck disable=SC2317 # funções chamadas por step
set -eu

[ $# -ge 2 ] || { sed -n '2,13p' "$0"; exit 2; }
mode="$1" v="$2" cmd="${3:-make ci}"
case "$v" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "sdd-release-check: versão inválida: $v" >&2; exit 2 ;; esac
tmp="$(mktemp -d)"
fail=0
step() { # step NOME COMANDO...: roda e imprime uma linha
  name="$1"; shift
  if "$@" > "$tmp/log" 2>&1; then echo "ok $name"; else echo "FALHA $name"; tail -n 20 "$tmp/log" | sed 's/^/  /'; fail=1; fi
}

case "$mode" in
pre)
  root="$(git rev-parse --show-toplevel)"
  ! git -C "$root" rev-parse -q --verify "refs/tags/$v" >/dev/null || { echo "sdd-release-check: a tag $v já existe" >&2; exit 2; }
  # Clone local, e não worktree: num worktree (.git é arquivo) o Go não grava a
  # versão da tag no binário, e a falha da v1.0.0 do go-release-manager passaria.
  # A tag só existe no clone: o repositório do dono não muda.
  wt="$tmp/wt"
  trap 'rm -rf "$tmp"' EXIT
  git clone -q --no-checkout "$root" "$wt"
  git -C "$wt" checkout -q --detach "$(git -C "$root" rev-parse HEAD)"
  git -C "$wt" tag "$v"
  [ -z "$(git -C "$root" status --porcelain)" ] || echo "aviso: há mudanças não commitadas; a simulação usa só a HEAD"
  cd "$wt"
  # -count=1: o cache de testes ignora a tag e esconderia justamente a falha procurada.
  step "$cmd na tag $v (checkout limpo)" env GOFLAGS="${GOFLAGS:+$GOFLAGS }-count=1" sh -c "$cmd"
  if ! ls .goreleaser.y*ml >/dev/null 2>&1; then
    echo "pulado goreleaser (sem .goreleaser.yml)"
  elif ! command -v goreleaser >/dev/null; then
    echo "pulado goreleaser (não instalado: go install github.com/goreleaser/goreleaser/v2@latest)"
  else
    step "goreleaser check" goreleaser check
    step "goreleaser release --snapshot" goreleaser release --snapshot --clean --skip=publish
  fi
  ;;
post)
  trap 'rm -rf "$tmp"' EXIT
  repo="$(git remote get-url origin | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
  if ! gh api "repos/$repo/releases/tags/$v" --jq '.assets[] | "\(.name)\t\(.id)"' > "$tmp/assets" 2>"$tmp/log"; then
    echo "FALHA release $v não encontrada em $repo"; exit 1
  fi
  n="$(wc -l < "$tmp/assets" | tr -d ' ')"
  if [ "$n" -gt 0 ]; then echo "ok release $v com $n asset(s)"; else echo "FALHA release $v sem assets"; fail=1; fi
  sums="$(grep -iE 'checksums?\.txt' "$tmp/assets" | cut -f1 | head -n 1 || true)"
  if [ -z "$sums" ]; then
    [ "$n" -eq 0 ] || { echo "FALHA sem arquivo de checksums"; fail=1; }
  else
    # Todo asset (exceto o próprio checksums) precisa constar do arquivo.
    check_sums() {
      # REST (gh release download usa GraphQL, bloqueado em algumas sessões).
      mkdir -p "$tmp/dl" &&
        gh api "repos/$repo/releases/assets/$(grep -F "$sums$(printf '\t')" "$tmp/assets" | cut -f2)" \
          -H 'Accept: application/octet-stream' > "$tmp/dl/$sums" &&
        cut -f1 "$tmp/assets" | grep -vxF "$sums" | while IFS= read -r a; do
          grep -qE "[[:space:]]\\*?$a\$" "$tmp/dl/$sums" || { echo "$a fora de $sums"; exit 1; }
        done
    }
    step "checksums cobrem os assets" check_sums
  fi
  mod="$(sed -n 's/^module //p' go.mod 2>/dev/null || true)"
  if [ -z "$mod" ]; then
    echo "pulado go install (sem go.mod)"
  else
    install() {
      GOBIN="$tmp/bin" GOFLAGS=-mod=mod go install "$mod@$v" &&
        bin="$(find "$tmp/bin" -type f | head -n 1)" && "$bin" --version | tee /dev/stderr | grep -qF "${v#v}"
    }
    step "go install $mod@$v e --version" install
  fi
  ;;
*) sed -n '2,13p' "$0"; exit 2 ;;
esac
exit "$fail"
