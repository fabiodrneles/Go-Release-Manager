# Squash merges sem perder quebras de compatibilidade

[English](squash-merges.en.md) · Guia de boas práticas do go-release-manager

Este guia explica por que um squash merge pode esconder uma quebra de compatibilidade, como evitar que isso aconteça e o que fazer quando já aconteceu.

## Sumário

- [O problema em um minuto](#o-problema-em-um-minuto)
- [O que o GitHub guarda em cada caso](#o-que-o-github-guarda-em-cada-caso)
- [Boas práticas](#boas-práticas)
- [Configuração passo a passo](#configuração-passo-a-passo)
- [Como escrever o PR](#como-escrever-o-pr)
- [Se a quebra já se perdeu](#se-a-quebra-já-se-perdeu)
- [Limites](#limites)
- [Perguntas frequentes](#perguntas-frequentes)

## O problema em um minuto

O go-release-manager calcula a versão pelas mensagens dos commits da `main`. Uma mudança incompatível sobe o major quando a mensagem tem `!` no cabeçalho (`feat!: ...`) ou um rodapé `BREAKING CHANGE: ...` no último parágrafo.

No squash merge, os commits do PR viram **um commit só**, e a mensagem dele é montada pelo GitHub. Se o aviso de quebra estava só na descrição do PR, ou num commit do meio, ele pode não chegar à `main`:

```text
PR "feat: drop --legacy"            → commit na main: "feat: drop --legacy (#42)"
  descrição: "BREAKING CHANGE: ..."     (a descrição ficou de fora)
  versão calculada: minor             ← deveria ser major
```

Ninguém percebe até a release sair com a versão errada, e quem depende do projeto atualiza sem saber que algo quebrou.

## O que o GitHub guarda em cada caso

Configuração em *Settings → General → Pull Requests → Allow squash merging → Default commit message*:

| Configuração | PR com vários commits | PR com um commit só |
|---|---|---|
| **Default message** (padrão do GitHub) | Título do PR + lista das mensagens dos commits | Mensagem do commit (o título do PR é ignorado) |
| **Pull request title** | Só o título do PR | Só o título do PR |
| **Pull request title and description** (recomendado) | Título + descrição do PR | Título + descrição do PR |

Consequências:

- **Em qualquer configuração, o título conta.** Se o título tem `!`, a quebra sobrevive sempre. É a regra mais simples e a mais segura.
- **Só com "title and description"** um rodapé `BREAKING CHANGE:` escrito na descrição chega à `main`.
- **No padrão do GitHub, o PR de um commit só** usa a mensagem do commit: editar o título do PR não muda nada.
- **Quem clica em "Squash and merge" pode editar a mensagem** na hora e apagar o aviso. Só a verificação depois do merge pega esse caso.

## Boas práticas

1. **Toda quebra vai no título com `!`:** `feat!: drop --legacy`, `fix(api)!: rename the status field`. Isso funciona em qualquer configuração.
2. **Explique a quebra num rodapé no fim da descrição:** `BREAKING CHANGE: --legacy was removed; use --mode=classic`. É ele que vira a nota de migração.
3. **Configure o squash para "Pull request title and description"**, para que a descrição chegue à `main`.
4. **Rode o `check-pr` em todo PR e torne-o obrigatório:** ele bloqueia o merge quando a quebra está anunciada em algum lugar e falta no título, e sugere o título certo.
5. **Mantenha a rede de segurança depois do merge:** o workflow de pós-merge abre uma issue se a quebra se perdeu mesmo assim.
6. **Antes da release, confira com `--explain`:** `go-release-manager create --dry-run --explain` mostra quais commits decidiram o incremento.
7. **Escreva `BREAKING CHANGE:` exatamente assim:** maiúsculas, singular e com dois-pontos. `BREAKING CHANGES:` ou `Breaking change:` não contam (o `check-pr` avisa).
8. **Prefira o squash a um merge commit sem padrão:** um merge commit como `Merge pull request #42` não segue Conventional Commits e é ignorado no cálculo.

## Configuração passo a passo

1. **GitHub:** *Settings → General → Pull Requests*, marque **Allow squash merging** e escolha **Default to pull request title and description**.
2. **Verificação no PR:** crie `.github/workflows/pr-check.yml`:

   ```yaml
   name: PR check
   on:
     pull_request:
       types: [opened, edited, synchronize, reopened]
   permissions:
     contents: read
     pull-requests: read
   jobs:
     check-pr:
       runs-on: ubuntu-latest
       steps:
         - uses: fabiodrneles/go-release-manager/check-pr@v1.3.0
   ```

   O evento `edited` faz o check rodar de novo quando alguém corrige o título.
3. **Torne o check obrigatório:** *Settings → Branches* (ou *Rules → Rulesets*), proteção da `main`, *Require status checks to pass*, e escolha **check-pr**.
4. **Rede de segurança:** copie o [post-merge-check.yml](../.github/workflows/post-merge-check.yml) deste repositório e troque `uses: ./check-pr` com `version: source` por `uses: fabiodrneles/go-release-manager/check-pr@v1.3.0` (o passo do `setup-go` deixa de ser necessário).
5. **Modelo de PR:** inclua no `.github/pull_request_template.md` a pergunta "Há quebra de compatibilidade? Se sim, `!` no título e `BREAKING CHANGE:` no fim da descrição".

## Como escrever o PR

Bom: a quebra está no título e explicada no fim da descrição.

```text
Título:    feat!: drop --legacy

Descrição:
Remove the --legacy flag, deprecated since v1.4.

BREAKING CHANGE: --legacy was removed; use --mode=classic instead.
```

Ruim: o título não tem `!`. O `check-pr` reprova e sugere `feat!: drop --legacy`.

```text
Título:    feat: drop --legacy

Descrição:
BREAKING CHANGE: --legacy was removed.
```

Ruim: o rodapé não está no último parágrafo. O `check-pr` avisa, porque a regra só lê o fim da mensagem.

```text
BREAKING CHANGE: --legacy was removed.

Also updated the docs.
```

Para conferir localmente antes de abrir o PR:

```text
go-release-manager check-pr --title "feat: drop --legacy" --body-file pr.md
```

## Se a quebra já se perdeu

Se o merge entrou sem a quebra (a rede de segurança abre uma issue), corrija **antes da release**, de um destes jeitos:

- **Commit de correção:** `git commit --allow-empty -m "feat!: drop --legacy" -m "BREAKING CHANGE: --legacy was removed"` na `main` (por um PR, se a `main` é protegida). O próximo cálculo vê a quebra.
- **Versão forçada:** `go-release-manager create --release-as v2.0.0` (ou o input `release-as` da Action) cria a tag com o major certo.

Se a release já saiu com a versão errada, publique logo a versão major correta e anote no CHANGELOG da versão anterior que ela continha uma quebra.

## Limites

- **Só quebras declaradas:** a ferramenta lê `!` e `BREAKING CHANGE:`. Uma quebra que ninguém anunciou exige comparar a API pública entre versões, o que ainda não é feito.
- **Configurações do repositório:** escolher a mensagem do squash e tornar o check obrigatório são passos de quem administra o repositório.
- **Push direto na `main`:** sem PR, não há o que verificar antes; vale o cálculo normal pelos commits.

## Perguntas frequentes

**Basta pôr `!` no título?** Para a versão, sim: o título sempre chega à `main` no squash. O rodapé `BREAKING CHANGE:` explica a quebra e entra nas notas.

**Uso merge commit em vez de squash. Preciso disso?** Os commits do PR entram como estão, então a quebra num commit chega à `main`. O `check-pr` continua útil para manter os títulos no padrão.

**O `check-pr` reprova PR sem quebra?** Só se o título não seguir Conventional Commits ou tiver um tipo desconhecido. Os avisos (rodapé fora do lugar, PR de um commit só) não reprovam.

**Funciona em runners macOS e Windows?** Sim. Se o job falhar em *Set up job* com "Failed to download archive", é uma instabilidade do GitHub ao baixar actions: re-execute o job.
