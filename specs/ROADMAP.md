# Roadmap e tarefas

Cada tarefa referencia a spec e os critérios de aceite que ela fecha. Ordem sugerida = ordem da lista. As versões continuam a numeração já publicada (D6).

## Fase 0 — Decisões (antes de codar)

- [x] Responder as decisões D1–D7 de [ANALYSIS.md](ANALYSIS.md) e mover as specs para `Approved`.

## Fase 1 — Funcionar de verdade (P0) → `v0.11.0`

- [x] **T1** CI de PR, remover o `.exe` e corrigir o lint — 003 FR-1, FR-4, AC-1, AC-3
- [x] **T2** Testes com repositório git real e correção do cálculo depois de pré-release — 001 FR-1, FR-2, AC-1, AC-2, AC-8
- [x] **T3** `0.x` e commits incompatíveis — 001 FR-3, AC-3
- [x] **T4** Mescla do `.go-releaserc.yml` e validação dos valores — 001 FR-4, AC-4, AC-5
- [x] **T5** Ignorar tags não SemVer e corrigir a contagem de commits — 001 FR-5, FR-6, AC-6, AC-7

## Fase 2 — Confiável (P1) → `v0.12.0`

- [x] **T6** `--version` e versão embutida — 002 FR-1, AC-1
- [x] **T7** Token só quando há push; remover `internal/provider` — 002 FR-2, AC-2
- [x] **T8** Subcomando `next` e `--output json` — 002 FR-3, FR-4, FR-5, AC-3, AC-4
- [x] **T9** README correto e comandos verificados no CI; cores com `NO_COLOR` — 002 FR-6, AC-5
- [x] **T10** Release com `make ci` antes e `prerelease: auto` — 003 FR-2, FR-3, AC-2
- [x] **T10b** Caminho do módulo do GitHub para o `go install` funcionar (achado durante o T9, #26)

## Fase 3 — Profissional (P2) → `v1.0.0`

- [x] **T11** GitHub Action reutilizável — 003 FR-5, AC-4
- [x] **T12** Integração com o plugin `sdd-release` do sdd-kit (documentação e exemplo)
- [x] **T13** `--version` com a versão do módulo no `go install` (#32)

## Fase 4 — Controle da release (P1) → `v1.1.0`

- [x] **T14** `create --release-as` — 004 FR-1, FR-3, AC-1, AC-2
- [x] **T15** `--ref`: analisar e criar a tag num commit — 004 FR-2, AC-3
- [x] **T16** Action com `release-as` e `ref`, e workflow `release-tag.yml` — 004 FR-4, FR-5, AC-4
