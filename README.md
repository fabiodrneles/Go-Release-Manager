# Go-Release-Manager

Uma ferramenta de linha de comando (CLI) escrita em Go para automatizar completamente o processo de versionamento semântico e criação de releases no GitHub.

## O Problema que Resolve

Em projetos de software, o processo de criar uma nova versão é manual, repetitivo e sujeito a erros. Esta ferramenta automatiza todo o fluxo, lendo seu histórico de commits e fazendo o trabalho pesado por você.

## Pré-requisitos

1.  **Go** instalado (versão 1.18+). https://go.dev/dl/
2.  **Git** instalado e configurado.
3.  Um repositório Git com um remote `origin` apontando para o GitHub.

## Instalação

- Baixe o executável na ultima release, o arquivo go-release-manager.exe ( para sistema windows apenas até o momento ).
- Coloque o executável dentro do seu projeto , aquele no qual quer construir a release.
- No terminal rode o comando ```.\go-release-manager.exe``` para executar o programa.
- Depois para criar uma release rode o comando ```.\go-release-manager.exe create --token "ghp_coloqueseutokendogithubaqui"```

<img width="1402" height="801" alt="image" src="https://github.com/user-attachments/assets/6dd6ec3f-4610-450e-b3ce-c58853ea0a9f" />

<img width="710" height="580" alt="image" src="https://github.com/user-attachments/assets/58ff37ae-1ce0-4f1c-8b67-7b85050bdd05" />


## Deve seguir o Conventional Commits para que o programa funcione adequadamente!



# Go Release Manager

[![](https://img.shields.io/github/actions/workflow/status/fabiodrneles/go-release-manager/release.yml?branch=main&label=Release&style=flat-square)](https://github.com/fabiodrneles/go-release-manager/actions/workflows/release.yml)
[![](https://img.shields.io/github/v/release/fabiodrneles/go-release-manager?style=flat-square&label=Última+Versão)](https://github.com/fabiodrneles/go-release-manager/releases)
[![](https://img.shields.io/badge/go-1.21%2B-blue?style=flat-square)](https://go.dev/)

Gerenciamento de versão e release totalmente automatizado, **sem a complexidade.**

O `go-release-manager` é uma CLI leve e ultrarrápida, escrita em Go, que implementa o Versionamento Semântico e o Conventional Commits. Ela é projetada para ser simples e funcionar perfeitamente com o **GoReleaser** e **GitHub Actions**.

---

## Por que o Go Release Manager?

O ecossistema de automação de releases é dominado por ferramentas complexas que exigem um ecossistema de plugins, múltiplas dependências e configurações extensas. O `go-release-manager` é diferente.

* **Simples e Focado:** Sem plugins. Sem `node_modules`. Enquanto concorrentes tentam fazer tudo (analisar, gerar changelog, publicar), o `go-release-manager` adota o Princípio da Responsabilidade Única: ele faz **uma coisa** perfeitamente: **determinar a próxima tag de versão**.
* **Feito para GoReleaser:** Esta ferramenta é o "cérebro" perfeito para o seu `goreleaser.yml`. Deixe o `go-release-manager` calcular a tag e deixe o `GoReleaser` fazer o build.
* **Rápido e Portátil:** É um binário Go único e nativo. Ele é executado instantaneamente, tornando seu pipeline de CI mais rápido.
* **CLI Ergonômica:** Construído para ser usado tanto em pipelines de CI quanto localmente por desenvolvedores. Com flags curtas e intuitivas como `-d` (dry-run) e `-p` (pre-release), testar seu próximo release é trivial.

## Como Funciona?

O fluxo de trabalho é projetado para máxima automação com o mínimo de configuração:

1.  Um desenvolvedor (ou um bot) faz um `git push` com commits (ex: `feat:`, `fix:`) para o branch principal.
2.  Uma GitHub Action é acionada e executa `go-release-manager create`.
3.  A ferramenta analisa os commits desde a última tag.
4.  Ela determina a próxima versão semântica (ex: `v1.2.3` ou `v1.3.0-beta.1`).
5.  Ela cria e empurra a nova tag Git para o seu repositório.
6.  O seu workflow `release.yml` (que escuta por *tags*) é **automaticamente acionado** por esse push da tag.
7.  O `GoReleaser` vê a nova tag, constrói seus binários, gera o changelog e publica o Release no GitHub.

## Recursos

* **Análise de Conventional Commits:** Entende `feat:`, `fix:`, e `BREAKING CHANGE` (ambos no cabeçalho `!` e no rodapé `BREAKING CHANGE:`).
* **Canais de Pré-Release:** Suporte completo para criar versões de pré-release (ex: `beta`, `rc`) com incremento automático (`.1`, `.2`, `.3`).
* **Modo de Simulação (Dry Run):** Veja qual versão seria criada sem fazer alterações no repositório.
* **Autenticação Flexível:** Lê o token da flag `-t` ou da variável de ambiente `GITHUB_TOKEN`.

## Instalação e Uso

### 1. Download (Recomendado)

Baixe o binário apropriado para seu sistema operacional (Windows, macOS, Linux) na nossa [**página de Releases**](https://github.com/fabiodrneles/go-release-manager/releases).

Coloque o binário em um local no seu `PATH` (ex: `/usr/local/bin` ou `C:\Program Files\go-release-manager`) para que ele possa ser chamado de qualquer lugar.

### 2. Uso na Linha de Comando (Local)

Use o `go-release-manager` para testar ou criar releases manualmente.

```bash
# Ajuda: veja todos os comandos e flags
$ go-release-manager create --help

Usage:
  go-release-manager create [flags]

Flags:
  -d, --dry-run        Simula o processo sem criar tags ou releases
  -h, --help           help for create
  -p, --pre-release string   Cria uma pré-release com o canal especificado (ex: beta, rc)
  -t, --token string       Token de Acesso Pessoal (PAT) do GitHub. (Padrão: env GITHUB_TOKEN)
```

#### Este projeto é inspirado pela filosofia do semantic-release, mas reimaginado com um foco em simplicidade, performance nativa e integração com o ecossistema Go.