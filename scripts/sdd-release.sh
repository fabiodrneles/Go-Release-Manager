#!/bin/sh
# sdd-release: prepara o PR de fechamento de uma versão em um comando (sdd-kit,
# issue #142) e, depois do merge dele, cria a tag. Só REST (gh api); não faz merge.
#
# Uso: sdd-release.sh [--repo DONO/REPO] [--dry-run] [X.Y.Z]
#      sdd-release.sh [--repo DONO/REPO] [--dry-run] --tag [X.Y.Z]
#
# A versão vem do go-release-manager (`go-release-manager next` na origin/main,
# pelos Conventional Commits desde a última tag estável; sem o binário, usa
# `go run` com GRM_VERSION); o ROADMAP não fixa versão (#198). X.Y.Z força a versão
# (release-as) e só vale a pedido explícito do dono, com aviso se divergir da calculada. Sem GRM nem Go, e sem X.Y.Z, falha dizendo como instalar.
#
# Preparar (X.Y.Z):
#   1 recusa árvore suja; cria a branch chore/release-vX.Y.Z a partir de origin/main
#     (se ela já existe, de uma tentativa anterior, reaproveita e traz a origin/main);
#   2 CHANGELOG: insere "## [X.Y.Z] - <hoje>" logo abaixo de "## [Unreleased]", move
#     para ela o que estava em Unreleased e acrescenta um RASCUNHO com os PRs
#     mesclados na main depois da última tag, agrupados pelo prefixo Conventional
#     Commits: feat -> "### Adicionado", fix -> "### Corrigido", demais ->
#     "### Alterado" (chore, docs, ci e test ficam de fora). Cada linha é
#     "- <título sem o prefixo> (#PR)". É um rascunho: revise o texto depois;
#   3 sobe a versão nos arquivos listados em .sdd-release (opcional, veja abaixo);
#   4 commit "chore: release vX.Y.Z" e entrega com sdd-pr.sh --no-wait.
# Tag (--tag X.Y.Z), depois do merge do PR de fechamento: confere que a main tem a
# entrada do CHANGELOG; roda scripts/sdd-release-check.sh pre (se existir); se há
# .github/workflows/release-tag.yml, dispara o workflow (release-as=vX.Y.Z, ref=SHA
# da main); senão cria e envia a tag anotada vX.Y.Z na origin/main.
#   --dry-run  mostra o que faria, sem escrever (nem branch, arquivos, push ou API)
#
# .sdd-release (raiz do repositório; linhas vazias e com # são ignoradas):
#   CAMINHO        troca toda ocorrência da versão anterior (sem o "v") por X.Y.Z
#   CAMINHO TEXTO  TEXTO é literal e leva @V@ onde vai a versão; só ele é trocado
#                  (ex.: scripts/adopt.sh  SDD_KIT_REF:-v@V@). A versão anterior
#                  vem da última tag vN.N.N da origin/main. Falha se o arquivo não
#                  tiver o trecho: o fechamento nunca sobe a versão pela metade.
# Códigos: 0 ok; 1 falha de uma etapa; 3 uso/pré-condição.
set -eu

die() { echo "sdd-release: $*" >&2; exit 3; }
repo="" dry=0 tagmode=0 ver=""
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    --tag) tagmode=1; shift ;;
    -h | --help) sed -n '2,32p' "$0"; exit 0 ;;
    -*) die "opção desconhecida: $1" ;;
    *) [ -z "$ver" ] || die "versão repetida: $1"; ver="${1#v}"; shift ;;
  esac
done
here="$(cd "$(dirname "$0")" && pwd)"
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2> /dev/null)" || die "sem remote origin; use --repo"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
git fetch -q --tags origin main

# Próxima versão pelo go-release-manager, analisando a origin/main (sem tocar na
# árvore de trabalho: worktree descartável).
GRM_VERSION="${GRM_VERSION:-v1.1.0}"
grm_next() {
  wt="$(mktemp -d)"
  git worktree add -q --detach "$wt" origin/main 2> /dev/null || { rm -rf "$wt"; return 1; }
  if command -v go-release-manager > /dev/null 2>&1; then
    out="$(cd "$wt" && go-release-manager next 2> /dev/null)" || out=""
  elif command -v go > /dev/null 2>&1; then
    out="$(cd "$wt" && go run "github.com/fabiodrneles/go-release-manager@$GRM_VERSION" next 2> /dev/null)" || out=""
  else
    out=""
  fi
  git worktree remove --force "$wt" 2> /dev/null || rm -rf "$wt"
  printf '%s\n' "$out" | tail -n 1 | sed 's/^v//'
}
calc="$(grm_next || true)"
if [ -z "$ver" ]; then
  [ -n "$calc" ] || die "sem versão: instale o go-release-manager (go install github.com/fabiodrneles/go-release-manager@latest) ou passe X.Y.Z; sem commits feat/fix desde a última tag, não há o que lançar"
  ver="$calc"
  echo "sdd-release: versão v$ver calculada pelo go-release-manager"
elif [ -n "$calc" ] && [ "$calc" != "${ver#v}" ]; then
  echo "sdd-release: aviso: o go-release-manager calcula v$calc; usando v${ver#v} (release-as)" >&2
fi
ver="${ver#v}"
printf '%s\n' "$ver" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || die "versão inválida (use X.Y.Z): $ver"
tag="v$ver"
if git ls-remote --tags origin "$tag" | grep -q .; then die "a tag $tag já existe na origin"; fi
short() { printf '%s' "$1" | cut -c1-7; }

# ---- --tag: depois do merge do PR de fechamento ----
if [ "$tagmode" -eq 1 ]; then
  git show origin/main:CHANGELOG.md 2> /dev/null | grep -q "^## \[$ver\]" \
    || die "a origin/main não tem '## [$ver]' no CHANGELOG: o PR de fechamento já foi mesclado?"
  sha="$(git rev-parse origin/main)"
  wf=".github/workflows/release-tag.yml"
  if [ "${SDD_RELEASE_PRE:-}" = ci ]; then
    # No motor (runner sem as ferramentas do projeto): a verificação é o CI da main
    # verde exatamente no commit que vai receber a tag.
    if [ "$dry" -eq 1 ]; then
      echo "[dry-run] sh $here/sdd-ci.sh $sha (CI da main antes da tag)"
    else
      echo "sdd-release: esperando o CI da main em $(printf '%s' "$sha" | cut -c1-7) antes da tag"
      sh "$here/sdd-ci.sh" --repo "$repo" --timeout "${SDD_RELEASE_CI_TIMEOUT:-1800}" "$sha" \
        || { echo "sdd-release: o CI da main não está verde em $(printf '%s' "$sha" | cut -c1-7); nada foi criado" >&2; exit 1; }
    fi
  elif [ -f scripts/sdd-release-check.sh ]; then
    if [ "$dry" -eq 1 ]; then
      echo "[dry-run] sh scripts/sdd-release-check.sh pre $tag"
    else
      sh scripts/sdd-release-check.sh pre "$tag" || { echo "sdd-release: sdd-release-check.sh pre falhou; nada foi criado" >&2; exit 1; }
    fi
  fi
  if git cat-file -e "origin/main:$wf" 2> /dev/null; then
    if [ "$dry" -eq 1 ]; then
      echo "[dry-run] dispararia o workflow release-tag.yml (release-as=$tag, ref=$(short "$sha"))"
    else
      gh api -X POST "repos/$repo/actions/workflows/release-tag.yml/dispatches" \
        -f ref=main -f "inputs[release-as]=$tag" -f "inputs[ref]=$sha" > /dev/null \
        || { echo "sdd-release: falha ao disparar o workflow" >&2; exit 1; }
      echo "sdd-release: workflow Release tag disparado para $tag em $(short "$sha")"
    fi
  elif [ "$dry" -eq 1 ]; then
    echo "[dry-run] git tag -a $tag origin/main e git push origin $tag"
  else
    git tag -a "$tag" -m "$tag" "$sha"
    git push -q origin "$tag"
    echo "sdd-release: tag $tag criada e enviada em $(short "$sha")"
  fi
  [ ! -f scripts/sdd-release-check.sh ] || echo "sdd-release: com a release publicada, rode: sh scripts/sdd-release-check.sh post $tag"
  exit 0
fi

# ---- preparar ----
branch="chore/release-$tag"
if [ -n "$(git status --porcelain)" ]; then
  git status --short >&2
  die "árvore suja: faça commit (ou WIP) antes"
fi
# Tentativa anterior: a branch já existe (local ou na origin); é reaproveitada.
reuse=0
if git ls-remote --exit-code --heads origin "$branch" > /dev/null 2>&1 \
  || git rev-parse -q --verify "refs/heads/$branch" > /dev/null; then reuse=1; fi
last="$(git describe --tags --abbrev=0 --match 'v[0-9]*' origin/main 2> /dev/null)" || die "origin/main sem tag vN.N.N anterior"
old="${last#v}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git show origin/main:CHANGELOG.md > "$tmp/CHANGELOG.md" 2> /dev/null || die "origin/main sem CHANGELOG.md"
grep -q '^## \[Unreleased\]' "$tmp/CHANGELOG.md" || die "CHANGELOG.md sem '## [Unreleased]'"
if grep -q "^## \[$ver\]" "$tmp/CHANGELOG.md"; then die "o CHANGELOG já tem a versão $ver"; fi

# PRs mesclados na main depois da data do commit da última tag (UTC, comparável).
since="$(TZ=UTC git log -1 --date=format-local:%Y-%m-%dT%H:%M:%SZ --format=%cd "$last^{commit}")"
# Paginação explícita (page=N): o --paginate do gh segue links repositories/{id}, que
# alguns proxies recusam. A lista vem pela atualização mais recente, então para na
# primeira página que já alcança PRs atualizados antes da última tag.
: > "$tmp/prs"
page=1
while :; do
  gh api "repos/$repo/pulls?state=closed&base=main&sort=updated&direction=desc&per_page=100&page=$page" \
    --jq "(.[] | select(.merged_at != null and .merged_at > \"$since\") | \"PR\t\(.number)\t\(.title)\"), \"N\t\(length)\", \"OLD\t\([.[] | select(.updated_at < \"$since\")] | length)\"" \
    > "$tmp/page" || die "falha ao listar os PRs de $repo"
  sed -n 's/^PR\t//p' "$tmp/page" >> "$tmp/prs"
  n="$(sed -n 's/^N\t//p' "$tmp/page")"
  stale="$(sed -n 's/^OLD\t//p' "$tmp/page")"
  if [ "${n:-0}" -lt 100 ] || [ "${stale:-0}" -gt 0 ] || [ "$page" -ge 50 ]; then break; fi
  page=$((page + 1))
done
: > "$tmp/add"; : > "$tmp/fix"; : > "$tmp/alt"
sort -n "$tmp/prs" | while IFS="$(printf '\t')" read -r num title; do
  type="$(printf '%s\n' "$title" | sed -n -E 's/^([a-z]+)(\([^)]*\))?!?: .*/\1/p')"
  text="$(printf '%s\n' "$title" | sed -E 's/^[a-z]+(\([^)]*\))?!?: //')"
  case "$type" in
    chore | docs | ci | test) continue ;;
    feat) f="$tmp/add" ;;
    fix) f="$tmp/fix" ;;
    *) f="$tmp/alt" ;;
  esac
  echo "- $text (#$num)" >> "$f"
done

# CHANGELOG: o conteúdo de Unreleased vira o da versão; os itens do rascunho entram
# no fim da seção de mesmo título (ou numa seção nova).
awk '/^## \[Unreleased\]/ { print; exit } { print }' "$tmp/CHANGELOG.md" > "$tmp/head"
awk '/^## \[Unreleased\]/ { u = 1; next } u && /^## \[/ { exit } u { print }' "$tmp/CHANGELOG.md" > "$tmp/body"
awk '/^## \[Unreleased\]/ { u = 1; next } u && /^## \[/ { r = 1 } r { print }' "$tmp/CHANGELOG.md" > "$tmp/rest"
awk -v fa="$tmp/add" -v ff="$tmp/fix" -v fl="$tmp/alt" '
function load(file, key,   l) { while ((getline l < file) > 0) items[key] = items[key] l "\n"; close(file) }
function flush() { if (cur in items) { printf "%s", items[cur]; delete items[cur] } printf "%s", held; held = "" }
BEGIN { load(fa, "### Adicionado"); load(ff, "### Corrigido"); load(fl, "### Alterado")
        order[1] = "### Adicionado"; order[2] = "### Corrigido"; order[3] = "### Alterado" }
/^[[:space:]]*$/ { held = held "\n"; next }
/^### / { flush(); cur = $0; print; next }
{ printf "%s", held; held = ""; print }
END { flush()
      for (i = 1; i <= 3; i++) if (order[i] in items) { print ""; print order[i]; print ""; printf "%s", items[order[i]] } }
' "$tmp/body" > "$tmp/body.new"
{
  cat "$tmp/head"
  echo
  echo "## [$ver] - $(date +%Y-%m-%d)"
  echo
  cat "$tmp/body.new"
  if [ -s "$tmp/rest" ]; then echo; cat "$tmp/rest"; fi
} | cat -s > "$tmp/CHANGELOG.out"

# Troca literal (sem regex) de $2 por $3 em $1, saída em $4; 1 se não havia ocorrência.
replace() {
  awk -v o="$2" -v n="$3" '
    { line = $0; out = ""
      while ((i = index(line, o)) > 0) { out = out substr(line, 1, i - 1) n; line = substr(line, i + length(o)); hit = 1 }
      print out line }
    END { exit hit ? 0 : 1 }' "$1" > "$4"
}
# Troca cada @V@ de $1 por $2.
subv() {
  s="$1" r=""
  while :; do
    case "$s" in *@V@*) ;; *) break ;; esac
    r="$r${s%%@V@*}$2"
    s="${s#*@V@}"
  done
  printf '%s' "$r$s"
}
bumps="$tmp/bumps"; : > "$bumps"
n=0
if git cat-file -e origin/main:.sdd-release 2> /dev/null; then
  git show origin/main:.sdd-release > "$tmp/sddrelease"
  while read -r path pat; do
    case "$path" in '' | '#'*) continue ;; esac
    git show "origin/main:$path" > "$tmp/src" 2> /dev/null || die ".sdd-release: '$path' não existe na origin/main"
    if [ -z "${pat:-}" ]; then
      o="$old" nw="$ver"
    else
      case "$pat" in *@V@*) ;; *) die ".sdd-release: '$path': o texto precisa de @V@" ;; esac
      o="$(subv "$pat" "$old")" nw="$(subv "$pat" "$ver")"
    fi
    n=$((n + 1))
    replace "$tmp/src" "$o" "$nw" "$tmp/bump.$n" || die ".sdd-release: '$path' não tem '$o'"
    echo "$path" >> "$bumps"
  done < "$tmp/sddrelease"
fi

if [ "$dry" -eq 1 ]; then
  echo "[dry-run] versão anterior $last; PRs mesclados desde $since: $(grep -c . "$tmp/prs" || true)"
  echo "[dry-run] criaria a branch $branch a partir de origin/main"
  echo "[dry-run] CHANGELOG.md, bloco novo:"
  diff "$tmp/CHANGELOG.md" "$tmp/CHANGELOG.out" | sed -n 's/^> /  | /p'
  sed 's/^/[dry-run] subiria a versão em /' "$bumps"
  echo "[dry-run] commit 'chore: release $tag' e sdd-pr.sh --no-wait"
  exit 0
fi

if [ "$reuse" -eq 1 ]; then
  echo "sdd-release: $branch já existe; reaproveitando e trazendo a origin/main"
  if git ls-remote --exit-code --heads origin "$branch" > /dev/null 2>&1; then
    git fetch -q origin "$branch"
    git checkout -q -B "$branch" FETCH_HEAD
  else
    git checkout -q "$branch"
  fi
  git merge -q --no-edit -X theirs origin/main
else
  git checkout -q -B "$branch" origin/main
fi
cp "$tmp/CHANGELOG.out" CHANGELOG.md
git add CHANGELOG.md
i=0
while read -r path; do
  i=$((i + 1))
  cp "$tmp/bump.$i" "$path"
  git add "$path"
done < "$bumps"
if [ "$reuse" -eq 1 ] && git diff --cached --quiet; then
  echo "sdd-release: nada novo a commitar em $branch"
else
  git commit -q -m "chore: release $tag"
fi
echo "sdd-release: $branch pronta (CHANGELOG com rascunho: revise o texto antes do merge)"
# Custo da fase (spec 017 FR-5): a tabela do sdd-report.sh phase do épico aberto,
# que também fica num comentário do épico; sem ela, o PR sai sem a seção.
bodyarg=""
epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty' 2> /dev/null || true)"
if [ -n "$epic" ] && [ -f "$here/sdd-report.sh" ] \
  && sh "$here/sdd-report.sh" --repo "$repo" phase "#$epic" > "$tmp/report" 2> /dev/null; then
  { printf '## Custo da fase\n\n%s\n\n' "Do \`sdd-report.sh phase\` do épico #$epic (sem LLM):"
    grep '^|' "$tmp/report"; } > "$tmp/prbody"
  bodyarg="$tmp/prbody"
fi
sh "$here/sdd-pr.sh" --repo "$repo" --no-wait --title "chore: release $tag" ${bodyarg:+--body-file "$bodyarg"}
