#!/bin/sh
# sdd-sync: atualiza os arquivos que o sdd-kit gerencia neste repositório para
# outra versão do kit (sdd-kit, spec 005). Usado pelo workflow sdd-sync.yml.
#
# Uso: sdd-sync.sh [--version vX.Y.Z] [--kit DIR] [--summary ARQUIVO]
#   --version  versão alvo (padrão: última release do sdd-kit no GitHub)
#   --kit      checkout local do kit, em vez de baixar (testes)
#   --summary  onde escrever o resumo em Markdown (padrão: saída padrão)
#
# Regras (spec 005 FR-1..4):
#   - arquivo que o kit não mudou → intocado, mesmo se alterado localmente;
#   - arquivo gerenciado sem alteração local → atualizado;
#   - arquivo gerenciado alterado localmente → atualizado e listado como conflito,
#     para revisão no PR (nunca em silêncio);
#   - arquivo novo no template → criado; arquivo que o repositório já tinha e não é
#     gerenciado → intocado;
#   - já na versão alvo → nada muda.
# Requer jq e curl (ou --kit).
set -eu

version="" kit="" summary=""
while [ $# -gt 0 ]; do
  case "$1" in
    --version) version="${2:?}"; shift 2 ;;
    --kit) kit="${2:?}"; shift 2 ;;
    --summary) summary="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "sdd-sync: opção desconhecida: $1" >&2; exit 2 ;;
  esac
done

state=.sdd-kit.json
[ -f "$state" ] || { echo "sdd-sync: $state não existe; adote o kit primeiro" >&2; exit 2; }
current="$(jq -r .version "$state")"
lang="$(jq -r .lang "$state")"
project="$(jq -r '.project // empty' "$state")"
owner="$(jq -r '.owner // empty' "$state")"
repo="$(jq -r '.repo // empty' "$state")"

if [ -z "$version" ]; then
  version="$(curl -fsSL https://api.github.com/repos/fabiodrneles/sdd-kit/releases/latest | jq -r .tag_name)"
fi

out() { if [ -n "$summary" ]; then cat >> "$summary"; else cat; fi; }
[ -z "$summary" ] || : > "$summary"

if [ "$version" = "$current" ]; then
  echo "sdd-sync: já na versão $current; nada a fazer"
  exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
if [ -z "$kit" ]; then
  curl -fsSL "https://github.com/fabiodrneles/sdd-kit/archive/refs/tags/$version.tar.gz" | tar -xz -C "$tmp"
  kit="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n1)"
fi

# Gera o template da versão alvo num diretório limpo, com os mesmos valores.
mkdir "$tmp/new"
SDD_KIT_REF="$version" sh "$kit/scripts/adopt.sh" --lang "$lang" \
  ${project:+--project "$project"} --owner "$owner" --repo "$repo" "$tmp/new" > /dev/null

hash() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1"; else shasum -a 256 "$1"; fi | cut -d' ' -f1; }

: > "$tmp/updated"; : > "$tmp/created"; : > "$tmp/conflicts"; : > "$tmp/managed"
(cd "$tmp/new" && find . -type f ! -name .sdd-kit.json | sed 's#^\./##' | LC_ALL=C sort) > "$tmp/list"
while IFS= read -r rel; do
  new="$tmp/new/$rel"
  old_hash="$(jq -r --arg f "$rel" '.files[$f] // empty' "$state")"
  if [ ! -e "$rel" ]; then
    mkdir -p "$(dirname "$rel")"
    cp -p "$new" "$rel"
    echo "$rel" >> "$tmp/created"
    echo "$rel" >> "$tmp/managed"
  elif [ -n "$old_hash" ]; then
    echo "$rel" >> "$tmp/managed"
    # O kit não mudou este arquivo: mantém o que o repositório tiver, alterado ou não.
    [ "$(hash "$new")" != "$old_hash" ] || continue
    cmp -s "$new" "$rel" && continue
    [ "$(hash "$rel")" = "$old_hash" ] || echo "$rel" >> "$tmp/conflicts"
    cp -p "$new" "$rel"
    echo "$rel" >> "$tmp/updated"
  fi
done < "$tmp/list"

# Novo estado: versão alvo e o hash do conteúdo do kit (não do arquivo local),
# para que uma alteração local continue detectável na próxima sincronização.
{
  printf '{\n  "kit": "sdd-kit",\n  "version": "%s",\n  "lang": "%s",\n  "project": "%s",\n  "owner": "%s",\n  "repo": "%s",\n  "files": {' \
    "$version" "$lang" "$project" "$owner" "$repo"
  sep=""
  while IFS= read -r rel; do
    printf '%s\n    "%s": "%s"' "$sep" "$rel" "$(hash "$tmp/new/$rel")"
    sep=","
  done < "$tmp/managed"
  printf '\n  }\n}\n'
} > "$tmp/state"
mv "$tmp/state" "$state"

{
  echo "Atualiza os arquivos do [sdd-kit](https://github.com/fabiodrneles/sdd-kit) de \`$current\` para \`$version\`."
  echo
  echo "- Atualizados: $(wc -l < "$tmp/updated" | tr -d ' ')"
  echo "- Criados: $(wc -l < "$tmp/created" | tr -d ' ')"
  if [ -s "$tmp/conflicts" ]; then
    echo
    echo "### Conflitos: arquivos alterados neste repositório"
    echo
    echo "A versão nova do kit substituiu as alterações locais abaixo. Revise o diff e restaure o que precisar:"
    echo
    # shellcheck disable=SC2016 # crases são Markdown, não substituição de comando
    while IFS= read -r rel; do printf -- '- `%s`\n' "$rel"; done < "$tmp/conflicts"
  fi
} | out
