# 003 — CI e pipeline de release

- **Prioridade:** P0
- **Status:** Done — entregue na `v1.0.0`
- **Código afetado:** `.github/workflows/`, `.goreleaser.yml`, `Makefile`, `action.yml`
- **Resolve:** A1, A2, A4, M5

## Contexto

Não há CI de PR (A1); pré-releases saem como releases normais (A2); a `main` está atrás das tags publicadas (A4); um binário de 10 MB está versionado (M5). O plugin `sdd-release` do sdd-kit vai usar esta ferramenta numa GitHub Action.

## Requisitos funcionais

- **FR-1** Todo PR MUST rodar `make ci` (lint, testes com race, build) e a verificação dos specs.
- **FR-2** O workflow de release MUST rodar `make ci` antes do GoReleaser.
- **FR-3** O GoReleaser MUST marcar como pré-release toda tag com sufixo (`prerelease: auto`).
- **FR-4** Nenhum binário MAY ser versionado; o `.gitignore` MUST cobri-los.
- **FR-5** O repositório MUST oferecer uma GitHub Action (`action.yml`) que instala a versão pedida a partir das releases e expõe a próxima versão como output.

## Critérios de aceite

- **AC-1** Dado um PR, quando o CI roda, então o job `make ci` é obrigatório e passa.
- **AC-2** Dado o `.goreleaser.yml`, quando `goreleaser check` roda no CI, então passa e `release.prerelease` é `auto`.
- **AC-3** Dado o repositório, quando o CI roda, então `git ls-files` não lista nenhum `.exe` nem binário.
- **AC-4** Dado um workflow de teste que usa a Action local, quando roda num repositório com `v1.0.0` e um `fix:`, então o output `next` é `v1.0.1`.

## Decisões

- D6 respondida pelo dono em 2026-10-02 conforme as recomendações de [ANALYSIS.md §7](../ANALYSIS.md#7-decisões-respondidas-pelo-dono-em-2026-10-02).
