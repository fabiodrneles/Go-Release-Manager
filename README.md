# Go Release Manager

[![CI](https://github.com/fabiodrneles/go-release-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/fabiodrneles/go-release-manager/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/fabiodrneles/go-release-manager?label=release)](https://github.com/fabiodrneles/go-release-manager/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/fabiodrneles/go-release-manager)](go.mod)

Versionamento semântico automático a partir dos seus commits, **sem a complexidade**.

O `go-release-manager` é uma CLI leve, escrita em Go, que lê os commits desde a última versão estável, aplica o [Conventional Commits](https://www.conventionalcommits.org/pt-br/v1.0.0/) e calcula (ou cria) a próxima tag [SemVer](https://semver.org/lang/pt-BR/). Funciona sozinha ou como o "cérebro" de um pipeline com o [GoReleaser](https://goreleaser.com/).

## Por que usar

- **Faz uma coisa bem:** calcula a próxima tag. Build, changelog e release ficam com o GoReleaser.
- **Binário único:** sem `node_modules`, sem plugins.
- **Feito para automação:** `next` imprime só a versão e `--output json` entrega o resultado completo.
- **Seguro:** `--dry-run` não escreve nada e não precisa de credenciais.

## Instalação

Baixe o binário do seu sistema (Linux, macOS ou Windows) na [página de Releases](https://github.com/fabiodrneles/go-release-manager/releases), confira o `checksums.txt` e coloque o binário no `PATH`. Com Go instalado:

```text
go install github.com/fabiodrneles/go-release-manager@latest
```

## Uso

Todos os exemplos abaixo rodam num repositório git com commits no padrão Conventional Commits.

```bash
# Próxima versão, sem criar nada (não precisa de token)
go-release-manager create --dry-run

# Só a versão, para scripts e GitHub Actions
go-release-manager next

# Próxima pré-release do canal rc (v1.3.0-rc.1, v1.3.0-rc.2, ...)
go-release-manager next --pre-release rc

# Resultado completo em JSON
go-release-manager create --dry-run --output json

# Versão do binário
go-release-manager --version
```

Para seguir uma versão definida no planejamento ou criar a tag num commit que não é o `HEAD`:

```text
go-release-manager create --release-as v2.0.0      # força a versão (maior que a última e ainda inexistente)
go-release-manager create --ref 1a2b3c4            # analisa e cria a tag nesse commit
```

Para criar e empurrar a tag, rode `go-release-manager create` com `GITHUB_TOKEN` definido ou depois de `gh auth login`. O push da tag dispara o workflow de release do seu repositório.

## Como a versão é calculada

| Commits desde a última versão estável | Próxima versão |
|---|---|
| `fix:` | patch (`v1.2.3` → `v1.2.4`) |
| `feat:` | minor (`v1.2.3` → `v1.3.0`) |
| `feat!:` ou rodapé `BREAKING CHANGE:` | major (`v1.2.3` → `v2.0.0`); em `0.x`, minor |
| `docs:`, `chore:`, `test:`, ... | nenhuma |

- A base é sempre a maior tag **estável** alcançável a partir de `HEAD`. Tags que não são SemVer (ex.: `latest`) são ignoradas.
- Com `--pre-release CANAL`, o contador continua: depois de `v2.0.0-beta.1`, vem `v2.0.0-beta.2`; a versão estável seguinte é `v2.0.0`.
- Sem nenhuma tag, a primeira versão parte de `v0.0.0`.

## Configuração (opcional)

Um `.go-releaserc.yml` na raiz muda as regras **só dos tipos listados**; os demais mantêm o padrão. Valores aceitos em `release`: `major`, `minor`, `patch`, `none`.

```yaml
releaseRules:
  - type: docs
    release: patch
  - type: perf
    release: patch
```

## GitHub Action

A Action instala o binário da release (conferindo o checksum) e expõe o resultado como outputs: `next`, `previous`, `increment` e `created`.

```yaml
name: Release
on:
  workflow_dispatch:
permissions:
  contents: write
jobs:
  tag:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - id: version
        uses: fabiodrneles/go-release-manager@v1.0.0
        with:
          create: true # sem isto, só calcula
      - run: echo "Tag criada: ${{ steps.version.outputs.next }}"
```

| Input | Padrão | O que faz |
|---|---|---|
| `version` | `latest` | Versão do go-release-manager a instalar |
| `create` | `false` | `true` cria e empurra a tag |
| `pre-release` | | Canal de pré-release (`beta`, `rc`) |
| `working-directory` | `.` | Repositório a analisar |
| `token` | `github.token` | Token para baixar a release e empurrar a tag |

O `checkout` precisa de `fetch-depth: 0` para enxergar as tags. A tag criada com o `GITHUB_TOKEN` padrão **não dispara** outros workflows (regra do GitHub); para disparar o seu workflow de release, use um token de app ou PAT em `token`, ou chame o build no mesmo workflow.

### Com o sdd-kit

No processo do [sdd-kit](https://github.com/fabiodrneles/sdd-kit), o plugin `sdd-release` usa esta ferramenta para propor a versão da próxima release a partir dos commits da fase e preparar o PR de fechamento.

## Desenvolvimento

O projeto segue Spec Driven Development com o [sdd-kit](https://github.com/fabiodrneles/sdd-kit): specs em [`specs/`](specs/README.md), um PR por ticket e `make ci` antes de todo push. Veja o [CONTRIBUTING](CONTRIBUTING.md).

Inspirado na filosofia do semantic-release, com foco em simplicidade, desempenho nativo e o ecossistema Go.

## Licença

[MIT](LICENSE)
