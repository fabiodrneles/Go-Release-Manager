# CLAUDE.md

Guia rápido para agentes (Claude Code) trabalharem no go-release-manager sem redescobrir o projeto a cada sessão. O processo é o da skill [`sdd-delivery`](https://github.com/fabiodrneles/sdd-kit).

## Retomar o trabalho (sessão nova ou contexto perdido)

1. Leia o **comentário "Estado da fase"** mais recente no épico aberto (issues com o label `épico`): PRs, estado do CI, decisões e próximo passo.
2. Liste os **PRs abertos** e o CI de cada um, e as **issues abertas** da fase.
3. Continue do próximo passo registrado. Não refaça análise que já está em specs, issues ou PRs.

O estado do trabalho vive no GitHub, e não na conversa. Abra o ticket e o PR assim que a tarefa começar e terminar, e atualize o comentário de estado do épico a cada marco.

## O projeto

CLI em Go que lê os commits desde a última tag, aplica Conventional Commits e cria a próxima tag SemVer (com canais de pré-release). O GoReleaser publica os binários quando a tag chega ao GitHub.

| Caminho | O que tem |
|---|---|
| `main.go`, `cmd/` | Comandos cobra (`create`) e flags |
| `internal/semver` | Cálculo da próxima versão a partir dos commits |
| `internal/git` | Chamadas ao `git` (tags, log, push) |
| `internal/config` | Leitura do `.go-releaserc.yml` |
| `internal/auth` | Token via `GITHUB_TOKEN` ou `gh auth token` |
| `.goreleaser.yml`, `.github/workflows/release.yml` | Build e release disparados por tag `v*` |
| `specs/` | Constituição, specs `NNN-nome/spec.md`, `ROADMAP.md`, `ANALYSIS.md` |

## Comandos

```text
make ci     # a mesma verificação do CI (rode antes de todo push)
make docs   # markdownlint (os links são verificados no CI)
make sdd-check  # cada AC de spec In Progress/Done citado num teste ("NNN AC-n")
```

Numa sessão na web, o hook `.claude/hooks/session-start.sh` instala as dependências e as ferramentas do CI.

## Convenções

- **Idioma:** specs, issues, PRs e documentação em português; commits e código (identificadores) em inglês.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`; `!` para mudança incompatível).
- **Branch:** uma por ticket, `<tipo>/<nº-da-issue>-<descrição>`, a partir da `main`.
- **PR:** começa com `Closes #N · Épico #M · Spec NNN` e segue o template.
- **Arquivos de status** (status das specs, checkboxes do ROADMAP, CHANGELOG) só mudam no PR de fechamento da fase.
- **Merge, tag e release** são do dono, salvo delegação explícita para uma rodada.

## Armadilhas já conhecidas

- **As tags `v0.9.0-beta.1`..`v0.10.0-beta.5` não estão na `main`** (houve rollback). Veja a decisão D6 em `specs/ANALYSIS.md`.
- **Toolchain baixada pelo `GOTOOLCHAIN`** não traz o `covdata`: `go test -cover` falha em pacotes sem teste com `no such tool "covdata"`. Use uma instalação completa do Go (o CI usa `setup-go`).

## Economia de uso

- Leia trechos (`sed -n 'a,bp'`, `grep -n`) em vez de arquivos inteiros, e não releia o que já leu nesta sessão.
- Para conferir CI, peça só o resumo das conclusões dos checks. Para investigar uma falha, leia o fim do log do job que falhou.
- Junte a validação num comando só (`make ci`) em vez de rodar etapas avulsas.
- Detalhes vão nos PRs e nas issues; no chat, só o resumo e o próximo passo.
