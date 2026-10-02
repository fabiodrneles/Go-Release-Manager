# Specs — go-release-manager

Este diretório organiza o desenvolvimento em **Spec Driven Development (SDD)**: nenhuma mudança de comportamento entra no código sem uma spec que a descreva e critérios de aceite que a verifiquem.

## Fluxo

```text
spec.md (O QUÊ / POR QUÊ)  →  revisão  →  testes a partir dos critérios de aceite  →  implementação  →  status: Done
```

1. **Especificar** — requisitos (`FR-*`), não funcionais (`NFR-*`) e critérios de aceite (`AC-*`) no formato Dado/Quando/Então (comportamento visto pelo usuário) ou EARS — `QUANDO <gatilho>, O SISTEMA DEVE <resposta>` (requisitos de sistema).
2. **Resolver decisões** — itens em aberto são respondidos pelo dono antes de implementar.
3. **Testar primeiro** — cada `AC-*` vira ao menos um teste automatizado.
4. **Implementar** — o PR referencia os IDs (ex.: `002 FR-1, AC-2`) e, se o comportamento mudou, acrescenta uma linha `ADDED`, `MODIFIED` ou `REMOVED` na seção "Mudanças" da spec.
5. **Fechar** — no PR de fechamento da fase (e não em cada PR de ticket), atualizar o status abaixo, o [ROADMAP](ROADMAP.md) e o `CHANGELOG.md`.

Convenções: `MUST`/`SHOULD`/`MAY` seguem a RFC 2119. Prioridades: **P0** (bloqueia uso real), **P1** (confiabilidade), **P2** (polimento).

## Documentos

| Documento | Conteúdo |
|---|---|
| [ANALYSIS.md](ANALYSIS.md) | Relatório da verificação e decisões em aberto |
| [constitution.md](constitution.md) | Princípios inegociáveis do projeto |
| [ROADMAP.md](ROADMAP.md) | Tarefas por fase, ligadas às specs |

## Specs

| ID | Spec | Prioridade | Status |
|---|---|---|---|
| 001 | [Cálculo da próxima versão](001-versioning/spec.md) | P0 | Approved |
| 002 | [Interface de linha de comando](002-cli/spec.md) | P1 | Approved |
| 003 | [CI e pipeline de release](003-release-pipeline/spec.md) | P0 | Approved |

Status possíveis: `Draft` → `Approved` → `In Progress` → `Done`.
