# 001 — Cálculo da próxima versão

- **Prioridade:** P0
- **Status:** Approved — decisões respondidas pelo dono em 2026-10-02
- **Código afetado:** `internal/semver`, `internal/config`, `internal/git`
- **Resolve:** C1, C2, M2, M3, B1

## Contexto

O cálculo erra depois de uma pré-release (C1), uma configuração parcial desativa os tipos padrão (C2), uma tag não SemVer derruba o cálculo (M2) e a contagem de commits vem com um a mais (M3). Nada disso tem teste.

## Requisitos funcionais

- **FR-1 (D1)** A base do cálculo MUST ser a última tag SemVer **estável** alcançável a partir de `HEAD`.
- **FR-2 (D1)** Com `-p CANAL`, se já existe `vBASE-CANAL.N` para a próxima versão estável, o resultado MUST ser `vBASE-CANAL.(N+1)`; senão, `vBASE-CANAL.1`.
- **FR-3 (D2)** Enquanto a base for `0.x`, um commit incompatível MUST incrementar o minor.
- **FR-4 (D3)** As regras do `.go-releaserc.yml` MUST sobrescrever as padrão só para os tipos listados. Um valor de `release` desconhecido MUST ser erro.
- **FR-5** Tags que não são SemVer MUST ser ignoradas na busca da última versão.
- **FR-6** O número de commits analisados MUST ser o número real de commits desde a base.

## Critérios de aceite

- **AC-1** Dado `v2.0.0-beta.1` sobre `v1.0.0` e um commit `feat:`, quando `create -d` roda, então a versão proposta é `v2.0.0`.
- **AC-2** Dado o mesmo histórico, quando `create -d -p beta` roda, então a versão proposta é `v2.0.0-beta.2`.
- **AC-3** Dado `v0.3.0` e um commit `feat!:`, quando `create -d` roda, então a versão proposta é `v0.4.0`.
- **AC-4** Dado um `.go-releaserc.yml` só com `docs: patch` e um commit `feat:` sobre `v1.0.0`, quando `create -d` roda, então a versão proposta é `v1.1.0`.
- **AC-5** Dado `release: "minr"` no `.go-releaserc.yml`, quando `create -d` roda, então sai com código diferente de zero e cita o tipo e o valor inválidos.
- **AC-6** Dada a tag `latest` sobre `v1.0.0` e um commit `fix:`, quando `create -d` roda, então a versão proposta é `v1.0.1`.
- **AC-7** Dado um commit desde `v1.0.0`, quando `create -d` roda, então "Commits analisados" é 1.
- **AC-8** Dado um repositório sem tags e um commit `feat:`, quando `create -d` roda, então a versão proposta é `v0.1.0`.

## Fora de escopo

- Monorepos e tags com prefixo de pacote.

## Decisões

- D1, D2 e D3 respondidas pelo dono em 2026-10-02 conforme as recomendações de [ANALYSIS.md §7](../ANALYSIS.md#7-decisões-respondidas-pelo-dono-em-2026-10-02).
