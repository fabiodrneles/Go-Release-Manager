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



