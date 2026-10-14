# Changelog

Todas as mudanças relevantes deste projeto. Formato [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/); versões [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [1.2.0] - 2026-10-14

### Adicionado

- the CLI and the Action speak English (#58)

## [1.1.1] - 2026-10-14

### Corrigido

- O binário baixado da release imprime no `--version` a versão no formato da tag, com o commit abreviado (`v1.1.1 (410e246)`). Antes, imprimia `1.1.0 (hash inteiro)`, diferente do `go install` e do `next` (#53).

### Documentação

- README reescrito em português e novo README em inglês (`README.en.md`), com prints da saída real, diagrama do cálculo da versão, instalação passo a passo por sistema com conferência do checksum, todos os inputs e outputs da Action, a saída JSON campo a campo, dicas e problemas comuns (#55).
- Os blocos `bash` dos dois READMEs rodam no CI (#55).

### Interno

- Testes de unidade para os pacotes internos e os comandos: a cobertura do código todo passa de 7,9% para 97,1% (#50).
- Arquivos do sdd-kit sincronizados para a v1.24.0. A sincronização semanal, que falhava, volta a funcionar, e o `make ci` passa a exigir 80% de cobertura (#51).

## [1.1.0] - 2026-10-02

### Adicionado

- `create --release-as vX.Y.Z`: força a versão (SemVer, maior que a última estável e ainda inexistente).
- `create --ref` e `next --ref`: analisam e criam a tag num commit específico.
- Saída JSON com `ref` e `forced`.
- Action com os inputs `release-as` e `ref`, e `version: source`.
- Workflow "Release tag": cria a tag pelo GitHub e publica a release no mesmo fluxo.

## [1.0.0] - 2026-10-02

Primeira versão estável. Cálculo de versão, CLI e pipeline cobertos por specs e testes ([`specs/`](specs/README.md)).

### Adicionado

- GitHub Action reutilizável (`uses: fabiodrneles/go-release-manager@v1.0.0`) com outputs `next`, `previous`, `increment` e `created`, testada em Linux, macOS e Windows.
- Integração documentada com o plugin `sdd-release` do sdd-kit.

### Corrigido

- `--version` mostra a versão do módulo quando o binário vem do `go install`.

## [0.12.0] - 2026-10-02

### Adicionado

- `--version`, com a versão embutida pelo GoReleaser.
- `next`: imprime só a próxima versão, para scripts e GitHub Actions.
- `create --output json`: resultado completo (`previous`, `next`, `increment`, `commits`, `created`).
- `go install github.com/fabiodrneles/go-release-manager@latest`.
- O release roda `make ci` antes de publicar, e os PRs rodam `goreleaser check`.

### Alterado

- `--dry-run` não exige mais credenciais: o token só é pedido antes do push da tag.
- Tags com sufixo (`-beta.N`, `-rc.N`) são publicadas como pré-release no GitHub.
- Erros saem numa linha só em stderr, sem o texto de uso do comando.
- Cores só num terminal; `NO_COLOR` é respeitado.
- README reescrito; os comandos dele rodam no CI.
- O caminho do módulo passa a ser `github.com/fabiodrneles/go-release-manager`.

### Removido

- O pacote `internal/provider`, que não era usado, e as dependências da API do GitHub.

## [0.11.0] - 2026-10-02

Primeira versão desenvolvida com o processo SDD do [sdd-kit](https://github.com/fabiodrneles/sdd-kit). Specs em [`specs/`](specs/README.md).

### Corrigido

- A versão é calculada a partir da última tag **estável**: depois de `v2.0.0-beta.1`, a estável é `v2.0.0` (antes `v2.1.0`) e a próxima beta é `v2.0.0-beta.2` (antes `v2.1.0-beta.1`).
- Um `.go-releaserc.yml` parcial não desativa mais `feat` e `fix`: as regras do arquivo sobrescrevem só os tipos listados.
- Tags que não são SemVer (ex.: `latest`) são ignoradas.
- A contagem de commits analisados não tem mais um item a mais.

### Alterado

- Commit incompatível enquanto a versão é `0.x` incrementa o minor (SemVer §4), em vez de ir para `1.0.0`.
- Valor de `release` inválido no `.go-releaserc.yml` é erro, com o tipo e o valor citados.

### Adicionado

- Testes de ponta a ponta em repositórios git reais e CI de PR (`make ci`).
- Processo SDD: constituição, specs, ROADMAP, `CLAUDE.md` e `AGENTS.md`.

### Removido

- O binário `go-release-manager.exe` versionado; o CI barra binários novos.
