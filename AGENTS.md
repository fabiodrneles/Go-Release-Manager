# AGENTS.md

Instruções para agentes de código (Codex, Copilot, Cursor, Claude Code e outros) que trabalham no go-release-manager. O guia completo está no [CLAUDE.md](CLAUDE.md); este arquivo resume o essencial.

## Processo: Spec Driven Development

- Nenhuma mudança de comportamento entra sem uma spec em [`specs/`](specs/README.md) com critérios de aceite (`AC-*`).
- Cada `AC-*` vira ao menos um teste automatizado.
- Uma tarefa = uma issue = uma branch (`<tipo>/<nº-da-issue>-<descrição>`) = um PR que começa com `Closes #N`.
- Decisões em aberto, merge, tag e release são do dono do repositório: pare e pergunte.
- O estado do trabalho fica no comentário "Estado da fase" do épico aberto (label `épico`); leia-o ao retomar.

## Verificação

Rode antes de todo push e só envie com tudo verde:

```text
make ci
```

## Convenções

- Commits em Conventional Commits, em inglês (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`).
- Specs, issues, PRs e documentação em português.
- Nunca desative ou pule um teste para o CI ficar verde: corrija a causa.
