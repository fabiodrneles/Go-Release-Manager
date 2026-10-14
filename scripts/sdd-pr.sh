#!/bin/sh
# sdd-pr: entrega a branch do ticket com um comando (sdd-kit, issue #137):
# atualiza com a main, roda o CI local, envia, abre o PR (ou reaproveita o que já
# existe), espera o CI do PR e grava o checkpoint. Só REST (gh api); não faz merge.
#
# Uso: sdd-pr.sh [--repo DONO/REPO] [--spec NNN|—] [--title "feat: ..."]
#                [--body-file ARQ] [--no-wait] [--dry-run]
#   --spec       número da spec do PR (padrão "—")
#   --title      título do PR (padrão: assunto do primeiro commit da branch que não é merge)
#   --body-file  texto que entra depois da linha "Closes #N · Épico #M · Spec NNN"
#                no lugar das seções vazias de .github/pull_request_template.md
#   --no-wait    não espera o CI do PR (sdd-ci.sh)
#   --dry-run    mostra o que faria, sem merge, CI, push nem escrita na API
# Passos: 1 branch <tipo>/<N>-<desc> e árvore limpa; 2 merge de origin/main (no
# conflito, aborta e lista os arquivos); 3 make ci (só as últimas 30 linhas se
# falhar); 4 git push; 5 PR; 6 sdd-ci.sh <SHA enviado>; 7 sdd-checkpoint.sh save;
# 8 imprime o comando do vigia do merge (sdd-wait.sh merged-any, spec 016).
# Códigos: 0 ok; 1 falha de uma etapa (ou do CI do PR); 2 tempo esgotado no
# sdd-ci.sh; 3 uso/pré-condição.
set -eu

die() { echo "sdd-pr: $*" >&2; exit 3; }
repo="" spec="—" title="" bodyfile="" wait=1 dry=0
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --spec) spec="${2:?}"; shift 2 ;;
    --title) title="${2:?}"; shift 2 ;;
    --body-file) bodyfile="${2:?}"; shift 2 ;;
    --no-wait) wait=0; shift ;;
    --dry-run) dry=1; shift ;;
    -h | --help) sed -n '2,20p' "$0"; exit 0 ;;
    *) die "opção desconhecida: $1" ;;
  esac
done
[ -z "$bodyfile" ] || [ -f "$bodyfile" ] || die "arquivo não existe: $bodyfile"
here="$(cd "$(dirname "$0")" && pwd)"
log="${TMPDIR:-/tmp}/sdd-pr-ci.log"

# 1. Branch do ticket e árvore limpa.
branch="$(git rev-parse --abbrev-ref HEAD)"
case "$branch" in main | master | HEAD) die "estou em $branch; entregue a branch do ticket" ;; esac
n="$(printf '%s\n' "$branch" | sed -n -E 's#^[^/]+/([0-9]+)-.*#\1#p')"
# Branches sem ticket: fechamento (sdd-release.sh, chore/release-vX.Y.Z) e
# sincronização do kit (sdd-sync, chore/sync-*); a primeira linha sai sem Closes.
case "$branch" in chore/release-v[0-9]* | chore/sync-*) n="" ;; *) [ -n "$n" ] || die "a branch '$branch' não segue <tipo>/<nº-da-issue>-<descrição>" ;; esac
if [ -n "$(git status --porcelain)" ]; then
  git status --short >&2
  die "árvore suja: faça commit (ou WIP) antes"
fi
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2> /dev/null)" || die "sem remote origin; use --repo"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
owner="${repo%%/*}"
git fetch -q origin main

# 2. Merge de origin/main (sem rebase).
if [ "$dry" -eq 1 ]; then
  echo "[dry-run] git merge origin/main ($(git rev-list --count HEAD..origin/main) commit(s) novo(s))"
elif ! git merge -q --no-edit origin/main > "$log" 2>&1; then
  conflicts="$(git diff --name-only --diff-filter=U)"
  git merge --abort 2> /dev/null || true
  echo "sdd-pr: conflito ao mesclar origin/main; arquivos:" >&2
  printf '%s\n' "$conflicts" | sed 's/^/  /' >&2
  exit 1
fi

# 3. CI local.
if [ "$dry" -eq 1 ]; then
  echo "[dry-run] make ci > $log"
elif [ -n "${SDD_PR_NO_CI:-}" ]; then
  # No motor (workflow sem as ferramentas do projeto), quem verifica é o CI do PR.
  echo "sdd-pr: make ci local pulado (SDD_PR_NO_CI); o CI do PR verifica"
elif ! make ci > "$log" 2>&1; then
  tail -n 30 "$log"
  echo "sdd-pr: make ci falhou (log completo em $log)" >&2
  exit 1
fi

# 4. Push.
if [ "$dry" -eq 1 ]; then
  echo "[dry-run] git push -u origin $branch"
else
  git push -q -u origin "$branch"
fi

# 5. PR: reaproveita o aberto para a branch; senão cria.
found="$(gh api "repos/$repo/pulls?state=open&head=$owner:$branch&per_page=1" --jq '.[0] | select(.) | "\(.number) \(.html_url)"')"
if [ -n "$found" ]; then
  pr="${found%% *}"
  echo "sdd-pr: PR #$pr já existe: ${found#* }"
else
  [ -n "$title" ] || title="$(git log origin/main..HEAD --no-merges --reverse --format=%s | head -n 1)"
  [ -n "$title" ] || die "sem commit na branch; passe --title"
  epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty' 2> /dev/null || true)"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  {
    # Branch numerada como o próprio épico aberto: "Refs", para o merge não fechá-lo.
    verb="Closes"
    [ "$n" != "$epic" ] || verb="Refs"
    echo "${n:+$verb #$n · }Épico ${epic:+#}${epic:-—} · Spec $spec"
    echo
    if [ -n "$bodyfile" ]; then
      cat "$bodyfile"
    elif [ -f .github/pull_request_template.md ]; then
      # Seções do template (a partir do primeiro "## "), sem comentários HTML
      # (de uma linha ou de várias) e sem as linhas que ficaram vazias por isso.
      awk '/^## /{on=1} on' .github/pull_request_template.md | awk '
        { line = $0; out = ""; touched = inc
          while (length(line)) {
            if (inc) {
              i = index(line, "-->")
              if (!i) { line = ""; break }
              line = substr(line, i + 3); inc = 0
            } else {
              i = index(line, "<!--")
              if (!i) { out = out line; line = ""; break }
              out = out substr(line, 1, i - 1); line = substr(line, i + 4); inc = 1; touched = 1
            }
          }
          if (!(touched && out ~ /^[[:space:]]*$/)) print out }' | cat -s
    fi
  } > "$tmp/body"
  if [ "$dry" -eq 1 ]; then
    echo "[dry-run] criaria o PR '$title' ($owner:$branch -> main) com o corpo:"
    sed 's/^/  | /' "$tmp/body"
    echo "[dry-run] depois esperaria o CI do PR e gravaria o checkpoint"
    exit 0
  fi
  out="$(gh api "repos/$repo/pulls" -f title="$title" -f head="$branch" -f base=main -F body=@"$tmp/body" --jq '"\(.number) \(.html_url)"')"
  pr="${out%% *}"
  echo "sdd-pr: PR #$pr aberto: ${out#* }"
fi
if [ "$dry" -eq 1 ]; then
  echo "[dry-run] esperaria o CI do PR #$pr e gravaria o checkpoint"
  exit 0
fi
# Push e PR feitos com o GITHUB_TOKEN não disparam o pull_request: o motor pede o
# CI da branch por workflow_dispatch (SDD_PR_DISPATCH_CI, workflow SDD_CI_WORKFLOW).
if [ -n "${SDD_PR_DISPATCH_CI:-}" ]; then
  if gh api -X POST "repos/$repo/actions/workflows/${SDD_CI_WORKFLOW:-ci.yml}/dispatches" --silent -f ref="$branch"; then
    echo "sdd-pr: CI disparado na branch $branch"
  else
    echo "sdd-pr: aviso: não consegui disparar o CI na branch $branch" >&2
  fi
fi

# 6. CI do PR.
rc=0
if [ "$wait" -eq 1 ]; then
  # Sem REF, o sdd-ci.sh espera o HEAD local (o SHA enviado): logo depois do
  # push, "#PR" ainda pode ler o head anterior do PR e dar um falso verde.
  sh "$here/sdd-ci.sh" --repo "$repo" || rc=$?
fi

# 7. Checkpoint (falha ignorada).
sh "$here/sdd-checkpoint.sh" --repo "$repo" save "PR #$pr aberto/atualizado ($branch)" "CI do PR #$pr e merge" || true
# 8. O vigia do merge, sem LLM (spec 016 FR-3): a última linha é o comando a deixar
# em segundo plano; o merge acorda a sessão, sem o dono precisar avisar.
rel="${here#"$(pwd)"/}"
echo "vigia: sh $rel/sdd-wait.sh merged-any"
exit "$rc"
