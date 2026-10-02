# Changelog

Todas as mudanças relevantes deste projeto. Formato [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/); versões [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

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
