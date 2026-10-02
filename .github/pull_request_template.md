<!-- markdownlint-disable-file MD041 -- o corpo do PR não tem título; o título é o do próprio PR -->
<!-- Título no formato Conventional Commits, ex.: "feat: add --watch mode" -->

Closes # · Épico # · Spec <!-- ex.: Closes #12 · Épico #10 · Spec 002 (use "Spec —" se nenhuma spec for afetada) -->

<!-- Se este PR depende de outro ainda não mergeado (PR empilhado), diga aqui:
> PR empilhado sobre #NN. -->

## O que muda

<!-- O problema e a solução, do ponto de vista de quem usa o go-release-manager. -->

## Specs e critérios de aceite

<!-- Ex.: 003 FR-8, AC-6. Se o comportamento mudou, a spec foi atualizada neste PR? -->

## Como foi testado

<!-- Testes novos ou alterados e o que mais foi verificado manualmente. -->

## Checklist

- [ ] `make ci` passa localmente
- [ ] Testes cobrem os critérios de aceite deste PR
- [ ] Spec atualizada (se o comportamento mudou)
- [ ] README/docs atualizados (se a interface mudou)
- [ ] Mudança incompatível sinalizada com `!` no título e explicada acima
