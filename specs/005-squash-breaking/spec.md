# 005 — Quebras de compatibilidade em squash merges

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.3.0`
- **Código afetado:** `cmd/`, `internal/semver`, `internal/prcheck`, `internal/git`, `check-pr/action.yml`, `.github/workflows/`
- **Origem:** pergunta de um leitor sobre o `BREAKING CHANGE:` em squash merges (épico #60)

## Contexto

Num squash merge, a mensagem do commit que entra na `main` é a que vale para a versão. Se o aviso de quebra (`!` depois do tipo ou o rodapé `BREAKING CHANGE:`) estava só na descrição do PR, ou num commit do PR que o mantenedor reescreveu, ele some e a versão sai menor do que deveria (um Minor no lugar de um Major). O cálculo em si (spec 001) está certo; o que falta é garantir que o aviso chegue ao commit.

## Requisitos funcionais

- **FR-1** `check-pr` MUST ler o título (`--title`), a descrição (`--body` ou `--body-file`) e os commits do PR (`--commits-file`, um JSON com a mensagem de cada commit); `--github-event ARQUIVO` MUST ler o título e a descrição do evento `pull_request` do GitHub Actions.
- **FR-2** O título MUST ser um Conventional Commit (`tipo(escopo)!: descrição`) de um tipo conhecido: os das regras (padrão e `.go-releaserc.yml`).
- **FR-3** Uma quebra anunciada em qualquer lugar do PR (o `!` do título, uma linha `BREAKING CHANGE:` ou `BREAKING-CHANGE:` na descrição, o `!` ou o rodapé de qualquer commit) MUST exigir o `!` no título. A falha MUST sugerir o título corrigido.
- **FR-4** Três situações duvidosas MUST gerar aviso, sem reprovar: o rodapé fora da última parte do texto (que a versão não conta), uma variante do marcador no começo de uma linha (`BREAKING CHANGES:`, `Breaking change:`) e um PR de um commit só com a quebra apenas fora dele (o squash padrão do GitHub usa a mensagem desse commit, e não o título).
- **FR-5** Com `--merged REF`, `check-pr` MUST comparar o PR com a mensagem do commit que entrou na `main` e MUST reprovar quando o PR anunciou uma quebra e esse commit não conta como quebra pelas mesmas regras do cálculo da versão.
- **FR-6** `check-pr` MUST sair com 0 sem erros (avisos não reprovam) e com 1 com pelo menos um erro; `--output json` MUST trazer `ok`, `title`, `breaking`, `sources`, `suggested_title` e `findings` (`severity`, `code`, `message`).
- **FR-7** A detecção de quebra MUST usar o mesmo código do cálculo da versão, para os dois nunca divergirem.
- **FR-8** `create --explain` e `next --explain` MUST listar os commits que determinaram o incremento e o motivo de cada um; o JSON MUST trazer `reasons`.
- **FR-9** A Action `check-pr/action.yml` MUST instalar o binário, buscar os commits do PR pela API e rodar o `check-pr`; o repositório MUST usá-la no workflow `pr-check.yml`.
- **FR-10** O workflow `post-merge-check.yml` MUST rodar a verificação do FR-5 a cada push na `main` e abrir uma issue quando a quebra se perdeu.
- **FR-11** O README (PT e EN) MUST explicar o problema, a configuração do GitHub, o `check-pr`, a Action e a rede de segurança; o modelo de PR MUST perguntar se há quebra de compatibilidade.

## Critérios de aceite

- **AC-1** Dado o título `feat: add flag` e nenhuma quebra, quando `check-pr` roda, então passa, sem erros.
- **AC-2** Dado o título `add flag`, ou o tipo `wip`, quando `check-pr` roda, então reprova com `title-not-conventional`.
- **AC-3** Dado o título `feat: add flag` e a descrição com a linha `BREAKING CHANGE: flags renamed` no fim, quando `check-pr` roda, então reprova com `breaking-not-in-title` e sugere `feat!: add flag`.
- **AC-4** Dado um commit do PR com `fix!: x` ou com o rodapé `BREAKING CHANGE:`, e o título sem `!`, quando `check-pr` roda, então reprova com `breaking-not-in-title`.
- **AC-5** Dado o título `feat(cli)!: rename flags`, quando `check-pr` roda, então passa e o resultado traz `breaking: true`.
- **AC-6** Dada a linha `BREAKING CHANGE:` no meio da descrição, com o `!` no título, quando `check-pr` roda, então passa com o aviso `footer-misplaced`.
- **AC-7** Dada a linha `BREAKING CHANGES: x` (plural) na descrição, quando `check-pr` roda, então avisa `breaking-typo`.
- **AC-8** Dado um PR de um commit só, com a quebra apenas na descrição e o `!` no título, quando `check-pr` roda, então passa com o aviso `single-commit-squash`.
- **AC-9** Dado o PR com quebra e `--merged HEAD` num commit `feat: x` sem marcador, quando `check-pr` roda, então reprova com `breaking-lost-in-merge`; com `feat!: x`, passa.
- **AC-10** Dados `--github-event` e `--commits-file`, quando `check-pr` roda, então lê o título e a descrição do evento e as mensagens do arquivo.
- **AC-11** Dado `--output json`, quando `check-pr` roda, então a saída é um JSON com os campos do FR-6.
- **AC-12** Dado o título ausente, `--github-event` junto de `--title`, ou um `--commits-file` inválido, quando `check-pr` roda, então sai com erro e uma mensagem clara.
- **AC-13** Dado um repositório com `feat:`, `fix!:` e `docs:` desde a última tag, quando `create -d --explain` roda, então lista o `fix!:` como o que decidiu o Major e o JSON traz `reasons`.
- **AC-14** Dado um PR deste repositório, quando o `pr-check.yml` roda, então usa a Action `check-pr` e reprova um título que perde a quebra.
- **AC-15** Dado um commit na `main` que perdeu a quebra do seu PR, quando o `post-merge-check.yml` roda, então abre uma issue com o motivo.

## Fora de escopo

- Detectar quebras que ninguém declarou (por exemplo, pela diferença da API pública de um módulo Go): uma segunda fase.
- Alterar as configurações de merge do repositório no GitHub.
