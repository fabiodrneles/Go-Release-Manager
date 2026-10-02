# 002 — Interface de linha de comando

- **Prioridade:** P1
- **Status:** Draft — aguarda as decisões D4, D5 e D7 do dono
- **Código afetado:** `cmd/`, `main.go`, `internal/auth`, `internal/provider`, `README.md`
- **Resolve:** A3, A5, M1, M4, B3

## Contexto

A versão não é embutida no binário (A3), o README documenta uma flag que não existe (A5), o dry-run exige token (M1) e não há saída para máquinas (M4), o que impede o uso pelo plugin `sdd-release` do sdd-kit.

## Requisitos funcionais

- **FR-1** `--version` MUST imprimir a versão embutida pelo GoReleaser (`-X main.version`), ou `dev` num build local.
- **FR-2 (D5)** O token MUST ser exigido só quando o comando vai fazer push; `--dry-run` MUST funcionar sem credenciais.
- **FR-3 (D4)** `next` MUST imprimir só a próxima versão (ou nada, sem incremento) e sair com 0.
- **FR-4 (D4)** `create --output json` MUST imprimir um objeto com `previous`, `next`, `increment`, `commits` e `created`.
- **FR-5** Mensagens de erro MUST terminar com quebra de linha e ir para stderr; a saída JSON MUST ir sozinha para stdout.
- **FR-6** Cores MUST ser desativadas quando a saída não é um terminal ou quando `NO_COLOR` está definida.

## Critérios de aceite

- **AC-1** Dado um binário com `-X main.version=1.2.3`, quando `--version` roda, então imprime `1.2.3`.
- **AC-2** Dado um repositório com commits novos e nenhuma credencial, quando `create -d` roda, então sai com 0 e mostra a versão proposta.
- **AC-3** Dado `v1.0.0` e um commit `fix:`, quando `next` roda, então stdout é exatamente `v1.0.1\n`.
- **AC-4** Dado o mesmo histórico, quando `create -d --output json` roda, então stdout é um JSON válido com `"next":"v1.0.1"` e `"created":false`.
- **AC-5** Dado o README, quando o CI roda, então cada comando dos blocos `bash` funciona.

## Fora de escopo

- Criar a release no GitHub pela API (fica com o GoReleaser, D5).

## Decisões

- Pendentes: D4, D5 e D7 ([ANALYSIS.md §7](../ANALYSIS.md#7-decisões-em-aberto)).
