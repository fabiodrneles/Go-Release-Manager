# 004 — Controle da versão e do commit da tag

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.1.0`
- **Código afetado:** `cmd/`, `internal/git`, `action.yml`

## Contexto

No exemplo-automacao-page-object, a versão de cada fase vinha do ROADMAP (`v0.1.0`, `v0.2.0`), mas os commits (`test:`, `ci:`, `docs:`) não geravam release pela regra padrão. As tags tinham de ficar nos commits de fechamento de cada fase, e não no topo da `main`. Sem como forçar a versão nem escolher o commit, a tag foi criada fora da ferramenta.

## Requisitos funcionais

- **FR-1** `create --release-as vX.Y.Z` MUST usar a versão informada no lugar da calculada. Ela MUST ser SemVer, maior que a última tag estável alcançável e ainda não existir.
- **FR-2** `create --ref REF` e `next --ref REF` MUST analisar os commits até `REF`, e não até `HEAD`; `create` MUST criar a tag em `REF`.
- **FR-3** A saída JSON MUST trazer `ref` (o commit da tag, abreviado) e `forced` (`true` quando a versão veio de `--release-as`).
- **FR-4** A GitHub Action MUST aceitar os inputs `release-as` e `ref`.
- **FR-5** Um workflow `release-tag.yml` (manual) MUST criar a tag com a Action e, quando a tag for nova, rodar o release no mesmo fluxo. Uma tag criada com o `GITHUB_TOKEN` não dispara outro workflow.

## Critérios de aceite

- **AC-1** Dado `v1.0.0` e um commit `docs:`, quando `create -d --release-as v1.1.0` roda, então a versão proposta é `v1.1.0` e o JSON traz `"forced":true`.
- **AC-2** Dado `v1.0.0`, quando `create -d --release-as v0.9.0` roda, então sai com erro; com `--release-as v1.0.0` (já existe), também.
- **AC-3** Dado `v1.0.0`, um `fix:` (commit A) e depois um `feat:`, quando `create --ref A` roda contra um remoto, então cria `v1.0.1` apontando para A, analisando 1 commit.
- **AC-4** Dado o workflow de teste da Action, quando roda com `release-as`, então o output `next` é a versão informada.

## Fora de escopo

- Gerar CHANGELOG a partir dos commits.
