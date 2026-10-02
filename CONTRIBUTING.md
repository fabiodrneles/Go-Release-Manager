# Contribuindo com o go-release-manager

Obrigado pelo interesse! Este guia descreve como o projeto é desenvolvido: o processo é o mesmo para quem mantém o projeto e para quem contribui pela primeira vez.

## Princípios

O projeto segue **Spec Driven Development (SDD)**: nenhuma mudança de comportamento entra no código sem uma spec que a descreva e critérios de aceite que a verifiquem.

- As specs ficam em [`specs/`](specs/README.md); os princípios inegociáveis, em [`specs/constitution.md`](specs/constitution.md).
- Cada critério de aceite (`AC-*`) vira ao menos um teste automatizado.
- O CI precisa estar verde para qualquer merge.

## Fluxo de trabalho

```text
spec → ticket → branch → testes + código → PR → revisão → merge → fechamento da fase
```

### 1. Fases e tickets

O trabalho é organizado em **fases** (ver [`specs/ROADMAP.md`](specs/ROADMAP.md)). Cada fase tem uma issue **épico** (label `épico`), e cada tarefa é uma **sub-issue** do épico, com:

- contexto (o problema);
- o que fazer;
- critérios de aceite verificáveis;
- spec(s) afetada(s).

Labels: `fase-N`, `tipo:feature|docs|ci|teste|chore` (defeitos usam o label padrão `bug`) e prioridade `P1` (alta) a `P3` (baixa).

Mudanças pequenas e óbvias (erro de digitação, link quebrado) podem ir direto para PR, sem ticket.

### 2. Branches

Uma branch por ticket, criada a partir da `main` (ou da branch da fase em andamento, quando a fase anterior ainda não foi mergeada):

```text
<tipo>/<nº-da-issue>-<descrição-curta>
```

Exemplos: `feat/12-login`, `fix/31-timeout-na-api`, `docs/7-guia`, `ci/6-release`.

### 3. Commits

[Conventional Commits](https://www.conventionalcommits.org/pt-br/), em inglês, no imperativo:

```text
feat: add login endpoint
fix: retry API call on timeout
docs: document configuration
test: cover empty input
ci: run tests on every PR
chore: update dependencies
```

Mudanças incompatíveis usam `!` (`feat!: ...`) e explicam o impacto no corpo do commit. O changelog das releases é gerado a partir desses prefixos.

### 4. Pull requests

- Um PR por ticket. A descrição começa com `Closes #nº · Épico #nº · Spec NNN`; o template já traz essa linha.
- Diga quais specs e critérios de aceite o PR implementa e **como foi testado**.
- Se o comportamento mudou, atualize a spec no mesmo PR.
- Mantenha o PR focado: o que não é do ticket vira outro ticket.
- PRs que dependem de outro PR ainda não mergeado são **empilhados** (base = branch do PR anterior) e dizem isso no topo da descrição.

### 5. Revisão e merge

- Todo PR precisa de CI verde e da aprovação de um mantenedor (ver [`CODEOWNERS`](.github/CODEOWNERS)).
- Os PRs de uma fase são revisados e mergeados na **ordem sugerida no épico**.
- PRs de fase inteira (que outros PRs usam como base) são mergeados com **merge commit**, para que os PRs empilhados sejam redirecionados à `main` sem rebase. Os demais podem usar **squash**.

### 6. Fechamento da fase

Depois que todos os PRs da fase forem mergeados, um **PR de fechamento** atualiza o status das specs, o [`ROADMAP.md`](specs/ROADMAP.md) e o `CHANGELOG.md`. Assim, os PRs da fase não entram em conflito por editarem os mesmos arquivos. Em seguida, cria-se a tag da versão.

O processo completo está na skill [`sdd-delivery`](https://github.com/fabiodrneles/sdd-kit), que o Claude Code aplica neste repositório.

## Ambiente de desenvolvimento

Antes de todo push, rode a mesma verificação do CI:

```text
make ci
```

## Reportando bugs

Abra uma issue com o modelo **Bug**: versão, como reproduzir, o que esperava e o que aconteceu.
