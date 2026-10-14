# CLAUDE.md

Guia rápido para agentes (Claude Code) trabalharem no go-release-manager sem redescobrir o projeto a cada sessão. O processo é o da skill [`sdd-delivery`](https://github.com/fabiodrneles/sdd-kit).

## Retomar o trabalho (sessão nova ou contexto perdido)

Uma regra só: **faça o "Próximo" que o `sh scripts/sdd-resume.sh` imprime** (o hook de início de sessão já o roda), sem esperar instrução e sem escolher outra coisa. "Continue" quer dizer exatamente isso.

- O "Próximo" vem do checkpoint do épico aberto (o motor o atualiza a cada merge) ou, sem épico, da fase já aprovada no ROADMAP.
- Se o "Próximo" é "perguntar ao dono", pergunte e pare: não escreva spec, não abra épico nem fase por conta própria.
- Não refaça análise que já está em specs, issues ou PRs.

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
make linkcheck  # links quebrados nos .md (precisa do lychee)
make sdd-check  # cada AC de spec In Progress/Done citado num teste ("NNN AC-n")
```

Numa sessão na web, o hook `.claude/hooks/session-start.sh` instala as dependências e as ferramentas do CI.

## Convenções

- **Idioma:** a conversa segue o idioma em que o dono escreve; uma mensagem curta como "continue" não define idioma: siga o das mensagens anteriores do dono ou, numa sessão nova, o deste arquivo; specs, issues, PRs e documentação em português; commits e código (identificadores) em inglês.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`; `!` para mudança incompatível).
- **Branch:** uma por ticket, `<tipo>/<nº-da-issue>-<descrição>`, a partir da `main`.
- **PR:** começa com `Closes #N · Épico #M · Spec NNN` e segue o template.
- **Arquivos de status** (status das specs, checkboxes do ROADMAP, CHANGELOG) só mudam no PR de fechamento da fase.
- **Merge, tag e release** são do dono, salvo delegação: `sh scripts/sdd-auto-merge.sh on` liga o merge automático (todo PR com o CI verde e sem conflito, inclusive os do relé), `off` desliga e `status` mostra. Quando o dono pedir para ligar ou desligar, rode o comando.
- **Comandos do README:** um bloco ```` ```bash ```` com `<!-- doc-commands -->` na linha anterior roda no CI (`sh scripts/doc-commands.sh`); marque os exemplos que devem continuar funcionando.

## Armadilhas já conhecidas

- Ferramenta antiga antes no PATH ofusca a versão certa do Makefile, e o shellcheck quebra em locale que não é UTF-8 (use `LC_ALL=C.UTF-8`): `sh scripts/sdd-doctor.sh` aponta e imprime o `export` a usar.
- **As tags `v0.9.0-beta.1`..`v0.10.0-beta.5` não estão na `main`** (houve rollback). Veja a decisão D6 em `specs/ANALYSIS.md`.
- **Toolchain baixada pelo `GOTOOLCHAIN`** não traz o `covdata`: `go test -cover` falha em pacotes sem teste com `no such tool "covdata"`. Use uma instalação completa do Go (o CI usa `setup-go`).

## Economia de uso

Cada regra abaixo reduziu o gasto de sessões reais; aplique desde a primeira mensagem.

- **Checkpoint contínuo:** a sessão pode acabar a qualquer momento, sem aviso. Depois de cada passo (commit, PR aberto, CI verde, merge), faça push e rode `sh scripts/sdd-checkpoint.sh save "feito" "próximo passo" ["o que bloqueia"]`: ele atualiza um único comentário de checkpoint no épico aberto, com branch, commit e PRs. Nunca deixe mais de um passo só na máquina; trabalho a meio vai num commit WIP na branch do ticket. O hook Stop mantém o estado do git atualizado a cada resposta.
- Leia trechos (`sed -n 'a,bp'`, `grep -n`) em vez de arquivos inteiros, e não releia o que já leu, nem depois de editar.
- Junte leituras e checagens independentes num comando só.
- Saída longa vai para um arquivo; mostre só o código de saída e o fim: `make ci > /tmp/ci.log 2>&1; echo "exit $?"; tail -n 3 /tmp/ci.log`.
- Valide tudo com `make ci`, uma vez, antes do push.
- CI dos PRs: `sh scripts/sdd-ci.sh '#PR'` (uma linha por check e só o fim do log das falhas). Não assine os eventos do PR; se a sessão assinar sozinha, cancele.
- **Não espere nem acompanhe eventos de PR (motor de eventos):** abra o PR (`sh scripts/sdd-pr.sh --no-wait`), grave o checkpoint e encerre a resposta. Os workflows atualizam o checkpoint no merge, resumem o CI vermelho num comentário do PR, trazem a `main` para os PRs, abrem o PR de fechamento e disparam a release; volte só para julgamento (código, causa raiz, conflito real, revisão). Se a sessão assinar o PR sozinha, cancele. **Vigia do merge (spec 016):** antes de encerrar, deixe **um** `sh scripts/sdd-wait.sh merged-any` em segundo plano (a última linha do `sdd-pr.sh`); quando ele acordar a sessão, rode o `sdd-resume.sh` e faça o Próximo, sem esperar o dono avisar; se o Próximo depende do dono, religue o vigia e encerre com uma linha; tempo esgotado, religue em silêncio. Para outras esperas num shell, o mesmo `sdd-wait.sh` em segundo plano (`pr-merged '#N'`, `ci '#N'`, `issue-closed '#N'`), nunca um laço feito à mão.
- Entrega do ticket num comando: `sh scripts/sdd-pr.sh [--spec NNN] [--dry-run]` (merge da `main`, `make ci`, push, PR ou o já aberto, CI do PR e checkpoint; não faz merge).
- Ambiente antes de `make ci`: `sh scripts/sdd-doctor.sh [--check]` confere e conserta ferramentas nas versões do CI, locale UTF-8 e PATH (uma linha por item); rode quando o hook de sessão não rodou (ex.: repositório anexado no meio da sessão) ou o `make ci` falhar por ferramenta.
- Fechamento da versão num comando: `sh scripts/sdd-release.sh [--dry-run]` (a versão é sempre a do go-release-manager, nunca a do ROADMAP; X.Y.Z força só a pedido explícito do dono; branch `chore/release-vX.Y.Z`, rascunho do CHANGELOG pelos PRs mesclados, versões listadas em `.sdd-release`, commit e PR); com o PR mesclado, `sh scripts/sdd-release.sh --tag X.Y.Z` (dispara o *Release tag* ou cria a tag; não faz merge).
- Custo medido, sem LLM: `sh scripts/sdd-report.sh tokens [--since DATA]` (sessão), `ticket '#N'` (janela do PR) e `phase '#ÉPICO'` (tabela num único comentário do épico); toda alegação de economia cita esse número.
- **Uma sessão, um ticket (spec 018):** com o PR do ticket aberto e o checkpoint gravado, a sessão termina; o próximo ticket começa numa sessão nova a partir de `sh scripts/sdd-context.sh '#N'` (só as linhas citadas da spec, decisões, arquivos prováveis e a entrega), sem ler specs inteiras. `sh scripts/sdd-relay.sh` faz esse laço sem LLM no meio (um agente novo por ticket, espera do merge, próximo ticket; para no dono, no CI vermelho duas vezes ou no orçamento) e grava na issue o custo exato do ticket. Conversa longa com o dono não acumula trabalho de tickets; perto do teto de contexto (`SDD_SESSION_MAX_TOKENS`, hook `sdd-session-guard.sh`), faça commit WIP, grave o checkpoint e termine.
- Edição mecânica por script que falha se o trecho não existir (ex.: `assert old in s` antes do `replace` em Python), sem reler o arquivo.
- Checagem de mutação sem reler: copie o arquivo, quebre, rode o teste, restaure com `cp`.
- Confira ferramentas e rede antes de começar (o proxy pode bloquear downloads); tente o gerenciador de pacotes do sistema.
- Nas ferramentas do GitHub, peça só os campos necessários (`fields`, `minimal_output`, `perPage`).
- Branch de PR já mergeado: recomece da `main` num comando (`git fetch origin main && git checkout -B <branch> origin/main`).
- Subagentes só para buscas amplas de leitura, num modelo pequeno.
- "Estado da fase" só nos marcos e em poucas linhas; no chat, três linhas (feito, falta, bloqueia); detalhes nos PRs e nas issues.
