# Go-Release-Manager

Uma ferramenta de linha de comando (CLI) escrita em Go para automatizar completamente o processo de versionamento semântico e criação de releases no GitHub.

## O Problema que Resolve

Em projetos de software, o processo de criar uma nova versão é manual, repetitivo e sujeito a erros. Esta ferramenta automatiza todo o fluxo, lendo seu histórico de commits e fazendo o trabalho pesado por você.

## Pré-requisitos

1.  **Go** instalado (versão 1.18+).
2.  **Git** instalado e configurado.
3.  Um repositório Git com um remote `origin` apontando para o GitHub.

## Instalação

```bash
go install [github.com/seu-usuario/go-release-manager@latest](https://github.com/seu-usuario/go-release-manager@latest)