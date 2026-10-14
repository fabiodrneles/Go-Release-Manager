#!/bin/sh
# sdd-sync: atualiza os arquivos que o sdd-kit gerencia neste repositório para
# outra versão do kit (sdd-kit, spec 005). Usado pelo workflow sdd-sync.yml.
#
# Uso: sdd-sync.sh [--version vX.Y.Z] [--kit DIR] [--summary ARQUIVO] [--only-workflows]
#   --version  versão alvo (padrão: última release do sdd-kit no GitHub)
#   --kit      checkout local do kit, em vez de baixar (testes)
#   --summary  onde escrever o resumo em Markdown (padrão: saída padrão)
#   --only-workflows  aplica só os workflows gerenciados (.github/workflows/) da
#              versão alvo e não mexe no estado; para quem o PR da sincronização
#              deixou de fora (ver SDD_SYNC_SKIP_WORKFLOWS)
#
# Ambiente: SDD_SYNC_SKIP_WORKFLOWS=1 (o workflow define quando o token não tem a
#   permissão `workflows`, que o GITHUB_TOKEN nunca tem): os arquivos de
#   .github/workflows/ não são escritos e o resumo os lista (spec 005 FR-5).
#
# Regras (spec 005 FR-1..4):
#   - arquivo que o kit não mudou → intocado, mesmo se alterado localmente;
#   - arquivo gerenciado sem alteração local → atualizado;
#   - arquivo gerenciado alterado localmente → atualizado e listado como conflito,
#     para revisão no PR (nunca em silêncio);
#   - arquivo novo no template → criado; arquivo que o repositório já tinha e não é
#     gerenciado → intocado;
#   - modelos do projeto (specs/*.md, CHANGELOG.md; spec 012) → criados se faltam,
#     nunca alterados, e fora do estado;
#   - já na versão alvo → nada muda.
# Requer jq e curl (ou --kit).
set -eu

# Este script está entre os arquivos que a sincronização atualiza, e o sh lê o
# script aos poucos: roda uma cópia para não continuar lendo a versão nova.
if [ -z "${SDD_SYNC_COPY:-}" ]; then
  copy="$(mktemp)"
  cp "$0" "$copy"
  SDD_SYNC_COPY="$copy" exec sh "$copy" "$@"
fi

version="" kit="" summary="" only_wf=0
while [ $# -gt 0 ]; do
  case "$1" in
    --version) version="${2:?}"; shift 2 ;;
    --kit) kit="${2:?}"; shift 2 ;;
    --summary) summary="${2:?}"; shift 2 ;;
    --only-workflows) only_wf=1; shift ;;
    -h | --help) sed -n '2,28p' "$0"; exit 0 ;;
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

# Com SKIP_WORKFLOWS a execução continua na mesma versão: workflows deixados de
# fora antes seguem pendentes (o estado não registra o hash deles) e voltam à lista.
if [ "$version" = "$current" ] && [ "$only_wf" = 0 ] && [ "${SDD_SYNC_SKIP_WORKFLOWS:-}" != 1 ]; then
  echo "sdd-sync: já na versão $current; nada a fazer"
  exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp" "$SDD_SYNC_COPY"' EXIT
if [ -z "$kit" ]; then
  curl -fsSL "https://github.com/fabiodrneles/sdd-kit/archive/refs/tags/$version.tar.gz" | tar -xz -C "$tmp"
  kit="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n1)"
fi

# Gera o template da versão alvo num diretório limpo, com os mesmos valores.
mkdir "$tmp/new"
SDD_KIT_REF="$version" sh "$kit/scripts/adopt.sh" --lang "$lang" \
  ${project:+--project "$project"} --owner "$owner" --repo "$repo" "$tmp/new" > /dev/null

hash() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1"; else shasum -a 256 "$1"; fi | cut -d' ' -f1; }

# Spec 005 FR-5: sem a permissão `workflows`, o push que cria ou muda um workflow é
# recusado; nesse modo os workflows ficam de fora e são listados no resumo.
skip_wf() { [ "${SDD_SYNC_SKIP_WORKFLOWS:-}" = 1 ] && case "$1" in .github/workflows/*) true ;; *) false ;; esac; }

: > "$tmp/updated"; : > "$tmp/created"; : > "$tmp/conflicts"; : > "$tmp/managed"; : > "$tmp/skipped"
(cd "$tmp/new" && find . -type f ! -name .sdd-kit.json | sed 's#^\./##' | LC_ALL=C sort) > "$tmp/list"
while IFS= read -r rel; do
  new="$tmp/new/$rel"
  if [ "$only_wf" = 1 ]; then
    case "$rel" in .github/workflows/*) ;; *) continue ;; esac
    jq -e --arg f "$rel" '.files | has($f)' "$tmp/new/.sdd-kit.json" > /dev/null || continue
    if ! cmp -s "$new" "$rel"; then
      mkdir -p "$(dirname "$rel")"
      cp -p "$new" "$rel"
      echo "$rel" >> "$tmp/updated"
    fi
    # Registra o hash do kit: o arquivo deixa de estar pendente na próxima sincronização.
    jq --arg f "$rel" --arg h "$(hash "$new")" '.files[$f] = $h' "$state" > "$tmp/state" && cp "$tmp/state" "$state"
    continue
  fi
  # Spec 012 FR-2: gerenciado é só o que a adoção da versão alvo registra; os
  # modelos do projeto (specs, CHANGELOG) são criados se faltam e nunca mudam.
  if ! jq -e --arg f "$rel" '.files | has($f)' "$tmp/new/.sdd-kit.json" > /dev/null; then
    if [ ! -e "$rel" ]; then
      mkdir -p "$(dirname "$rel")"
      cp -p "$new" "$rel"
      echo "$rel" >> "$tmp/created"
    fi
    continue
  fi
  old_hash="$(jq -r --arg f "$rel" '.files[$f] // empty' "$state")"
  if [ ! -e "$rel" ]; then
    if skip_wf "$rel"; then echo "$rel" >> "$tmp/skipped"; continue; fi
    echo "$rel" >> "$tmp/managed"
    mkdir -p "$(dirname "$rel")"
    cp -p "$new" "$rel"
    echo "$rel" >> "$tmp/created"
  elif [ -n "$old_hash" ]; then
    echo "$rel" >> "$tmp/managed"
    # O kit não mudou este arquivo: mantém o que o repositório tiver, alterado ou não.
    [ "$(hash "$new")" != "$old_hash" ] || continue
    cmp -s "$new" "$rel" && continue
    if skip_wf "$rel"; then echo "$rel" >> "$tmp/skipped"; continue; fi
    [ "$(hash "$rel")" = "$old_hash" ] || echo "$rel" >> "$tmp/conflicts"
    cp -p "$new" "$rel"
    echo "$rel" >> "$tmp/updated"
  fi
done < "$tmp/list"

if [ "$only_wf" = 1 ]; then
  {
    echo "Workflows do [sdd-kit](https://github.com/fabiodrneles/sdd-kit) \`$version\` aplicados: $(wc -l < "$tmp/updated" | tr -d ' ')."
    echo
    # shellcheck disable=SC2016 # crases são Markdown, não substituição de comando
    while IFS= read -r rel; do printf -- '- `%s`\n' "$rel"; done < "$tmp/updated"
  } | out
  exit 0
fi

cp "$state" "$tmp/oldstate"
# Novo estado: versão alvo e o hash do conteúdo do kit (não do arquivo local),
# para que uma alteração local continue detectável na próxima sincronização.
{
  printf '{\n  "kit": "sdd-kit",\n  "version": "%s",\n  "lang": "%s",\n  "project": "%s",\n  "owner": "%s",\n  "repo": "%s",\n  "files": {' \
    "$version" "$lang" "$project" "$owner" "$repo"
  sep=""
  while IFS= read -r rel; do
    # Workflow deixado de fora mantém o hash anterior: segue pendente até ser aplicado.
    if grep -qxF "$rel" "$tmp/skipped"; then h="$(jq -r --arg f "$rel" '.files[$f]' "$tmp/oldstate")"; else h="$(hash "$tmp/new/$rel")"; fi
    printf '%s\n    "%s": "%s"' "$sep" "$rel" "$h"
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
  if [ -s "$tmp/skipped" ]; then
    echo
    echo "### Workflows não aplicados"
    echo
    echo "O token desta execução não tem a permissão \`workflows\`, e o GitHub recusa o push que cria ou muda arquivos em \`.github/workflows/\`. Estes arquivos ficaram de fora deste PR:"
    echo
    # shellcheck disable=SC2016 # crases são Markdown, não substituição de comando
    while IFS= read -r rel; do printf -- '- `%s`\n' "$rel"; done < "$tmp/skipped"
    echo
    echo "Para aplicá-los, escolha um caminho:"
    echo
    echo "1. configure o segredo \`SDD_SYNC_TOKEN\` (token fine-grained com *Contents*, *Pull requests* e *Workflows* em escrita) e rode o workflow de novo; ou"
    echo "2. localmente, na branch deste PR: \`sh scripts/sdd-sync.sh --only-workflows\`, depois commit e push."
  fi
} | out
