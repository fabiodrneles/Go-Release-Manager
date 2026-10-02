# Constituição do go-release-manager

Princípios que toda spec e todo PR devem respeitar. Mudá-los exige uma spec própria.

1. **Testado.** Todo critério de aceite tem teste automatizado; o CI bloqueia merge vermelho.
2. **Falhar alto.** Erros são reportados com contexto e código de saída diferente de zero; nada é descartado em silêncio, inclusive regras de configuração.
3. **SemVer correto.** A versão calculada segue o SemVer 2.0.0 e o Conventional Commits 1.0.0; cada regra de cálculo tem teste com um repositório git real.
4. **Responsabilidade única.** A ferramenta calcula e cria a tag; build, changelog e release no GitHub ficam com o GoReleaser.
5. **Seguro por padrão.** `--dry-run` nunca escreve no repositório nem exige credenciais; nada é empurrado sem o pedido explícito do comando.
6. **Feito para automação.** Toda informação mostrada a pessoas também está disponível numa saída estável para máquinas.
7. **Documentação verificada.** Cada comando do README funciona como está escrito.
