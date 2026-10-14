#!/bin/sh
# sdd-on-release-merge: quando o PR de fechamento (branch chore/release-vX.Y.Z) é
# mergeado, dispara a release da versão uma única vez, rodando o
# `sdd-release.sh --tag X.Y.Z` (sdd-kit, spec 015 FR-5, NFR-1, NFR-2). Só REST
# (gh api). Usado pelo workflow sdd-on-release-merge.yml; a lógica fica aqui para
# ser testável.
#
# Uso: sdd-on-release-merge.sh [--repo DONO/REPO] NÚMERO_DO_PR
# Só age se o PR foi mesclado, é da branch chore/release-vX.Y.Z do próprio
# repositório (PR de fork é ignorado, NFR-1); qualquer outro PR sai 0 sem fazer
# nada. A versão X.Y.Z da branch vai como release-as para o sdd-release.sh --tag.
# Com SDD_ENGINE=off não lê nem escreve nada (AC-7).
# Idempotente (NFR-2): nada é feito se a tag vX.Y.Z já existe, ou se há uma
# execução do workflow release-tag.yml na fila, em andamento ou concluída com
# sucesso desde o merge do PR (a API não expõe os inputs da execução).
# Observação: um workflow_dispatch disparado com o GITHUB_TOKEN RODA (é a exceção
# documentada à regra de não recursão do GitHub); só push e eventos de PR não
# disparam outros workflows. Se o repositório não tem release-tag.yml, o
# sdd-release.sh cria a tag por push e o workflow precisa de contents: write.
# SDD_RELEASE_SH troca o script chamado (testes). Códigos: 0 ok; 1 falha ao
# disparar; 3 uso.
set -eu

die() { echo "sdd-on-release-merge: $*" >&2; exit 3; }
repo="${GITHUB_REPOSITORY:-}" pr=""
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    -h | --help) sed -n '2,19p' "$0"; exit 0 ;;
    -*) die "opção desconhecida: $1" ;;
    *) [ -z "$pr" ] || die "PR repetido: $1"; pr="${1#\#}"; shift ;;
  esac
done
if [ "${SDD_ENGINE:-}" = "off" ]; then echo "SDD_ENGINE=off: nada a fazer"; exit 0; fi
[ -n "$repo" ] || die "informe --repo DONO/REPO (ou GITHUB_REPOSITORY)"
printf '%s\n' "$pr" | grep -Eq '^[0-9]+$' || die "informe o número do PR"
here="$(cd "$(dirname "$0")" && pwd)"
release="${SDD_RELEASE_SH:-$here/sdd-release.sh}"

info="$(gh api "repos/$repo/pulls/$pr" \
  --jq '[.merged, .head.ref, (.head.repo.full_name // ""), (.merged_at // "")] | @tsv')" \
  || die "falha ao ler o PR #$pr"
tab="$(printf '\t')"
IFS="$tab" read -r merged ref headrepo merged_at <<EOT
$info
EOT
[ "$merged" = "true" ] || { echo "PR #$pr não foi mesclado: nada a fazer"; exit 0; }
printf '%s\n' "$ref" | grep -Eq '^chore/release-v[0-9]+\.[0-9]+\.[0-9]+$' \
  || { echo "PR #$pr ($ref) não é de fechamento: nada a fazer"; exit 0; }
[ "$headrepo" = "$repo" ] || { echo "PR #$pr vem de outro repositório: ignorado"; exit 0; }
tag="${ref#chore/release-}"
ver="${tag#v}"

if gh api "repos/$repo/git/ref/tags/$tag" > /dev/null 2>&1; then
  echo "a tag $tag já existe: nada a fazer"; exit 0
fi
runs="$(gh api "repos/$repo/actions/workflows/release-tag.yml/runs?per_page=30" \
  --jq ".workflow_runs[] | select(.created_at >= \"$merged_at\" and (.status == \"queued\" or .status == \"in_progress\" or .conclusion == \"success\")) | .id" 2> /dev/null || true)"
if [ -n "$runs" ]; then
  echo "já há uma execução do Release tag desde o merge ($(echo "$runs" | head -n 1)): nada a fazer"; exit 0
fi

echo "disparando a release $tag (PR #$pr)"
# O runner não tem as ferramentas do projeto: a checagem antes da tag é o CI da main (#184).
SDD_RELEASE_PRE=ci sh "$release" --repo "$repo" --tag "$ver" || { echo "sdd-on-release-merge: o sdd-release.sh falhou" >&2; exit 1; }

# A fase fechou: abre o épico da próxima fase do ROADMAP (já aprovada pelo dono) e
# grava o checkpoint nele, para um "continue" numa sessão nova achar o próximo passo (#191).
open_epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty' 2> /dev/null || true)"
# O épico da fase entregue (sem sub-issue aberta) é fechado aqui; antes, ficava aberto
# e impedia a abertura da próxima fase (#217).
if [ -n "$open_epic" ]; then
  left="$(gh api "repos/$repo/issues/$open_epic/sub_issues?per_page=100" --jq '[.[] | select(.state == "open")] | length' 2> /dev/null || echo '?')"
  if [ "$left" = 0 ]; then
    gh api -X POST "repos/$repo/issues/$open_epic/comments" -f body="Fase entregue na $tag (PR #$pr); todos os tickets fechados. Épico fechado pelo motor." > /dev/null || true
    if gh api -X PATCH "repos/$repo/issues/$open_epic" -f state=closed -f state_reason=completed > /dev/null; then
      echo "épico #$open_epic fechado: fase entregue na $tag"
      open_epic=""
    fi
  fi
fi
n=""
if [ -n "$open_epic" ]; then
  echo "épico #$open_epic já aberto: próxima fase não aberta"
elif next="$(sh "$here/sdd-next-phase.sh" 2> /dev/null)"; then
  t="$(printf '\t')"; n="${next%%"$t"*}"
else
  echo "nenhuma fase com tarefa aberta no ROADMAP: o dono escolhe a próxima"
fi
# Fase já entregue (épico fechado) com caixa ainda aberta no ROADMAP: o PR de
# fechamento foi mesclado sem marcar o ROADMAP; reabri-la duplicaria a fase (#292).
done_epic=""
if [ -n "$n" ]; then
  done_epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=closed&per_page=100" \
    --jq "[.[] | select((.title + \" \") | startswith(\"Fase $n \"))][0].number // empty" 2> /dev/null || true)"
fi
if [ -n "$done_epic" ]; then
  echo "sdd-on-release-merge: aviso: a Fase $n já foi entregue no épico #$done_epic, mas o ROADMAP ainda tem tarefa aberta; marque o ROADMAP (a fase não foi reaberta)" >&2
elif [ -n "$n" ]; then
  echo "abrindo a Fase $n do ROADMAP"
  if sh "${SDD_EPIC_SH:-$here/sdd-epic.sh}" --repo "$repo" "$n"; then
    sh "$here/sdd-checkpoint.sh" --repo "$repo" --ci save "release $tag disparada; Fase $n aberta pelo motor" \
      "primeiro ticket aberto do épico (sdd-resume.sh lista as issues)" || true
  else
    echo "sdd-on-release-merge: aviso: não consegui abrir a Fase $n" >&2
  fi
fi
