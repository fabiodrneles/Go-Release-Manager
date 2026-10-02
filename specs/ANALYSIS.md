# Análise e verificação do go-release-manager

Fase de descoberta (skill `sdd-delivery`), feita em 2026-10-02 sobre a `main` em `597683b`. O que está marcado como **verificado** foi executado; o que está como **inferido** vem da leitura do código.

## 1. Resumo executivo

O go-release-manager é uma CLI em Go (cerca de 750 linhas) que lê os commits desde a última tag, aplica Conventional Commits e cria a próxima tag SemVer, inclusive em canais de pré-release (`-p beta`). O GoReleaser publica os binários quando a tag chega ao GitHub.

O núcleo funciona no caso simples (`v1.0.0` + `fix:` → `v1.0.1`), mas **o cálculo depois de uma pré-release está errado**: a versão estável e a próxima pré-release pulam uma versão. O histórico do próprio repositório mostra o sintoma, com uma sequência de betas e nenhuma versão estável desde a `v0.4.3`. Não há nenhum teste nem CI de PR, um arquivo de configuração parcial desativa `feat` e `fix` em silêncio, e o README documenta uma flag que não existe.

A `main` foi revertida ("rollback") para antes da `--version`: as tags `v0.9.0-beta.1` a `v0.10.0-beta.5` apontam para commits que não estão mais na `main`.

## 2. O que foi verificado

| Comando | Resultado |
|---|---|
| `go build ./... && go vet ./...` | ok |
| `go test ./...` | nenhum arquivo de teste em nenhum pacote |
| `golangci-lint run` (v2.14.0) | 1 problema: SA6000 em `internal/semver/semver.go:47` |
| `v1.0.0` + `fix: b`; `create -d` | `v1.0.1` (correto), mas "Commits analisados: 2" para 1 commit |
| `v1.0.0` + `feat!: c`; `create -d -p beta` | `v2.0.0-beta.1` (correto) |
| `v2.0.0-beta.1` + `feat: c`; `create -d` | **`v2.1.0`**; esperado `v2.0.0` |
| `v2.0.0-beta.1` + `feat: c`; `create -d -p beta` | **`v2.1.0-beta.1`**; esperado `v2.0.0-beta.2` |
| `.go-releaserc.yml` só com `docs: patch`; commit `feat: b` | **"Nenhuma mudança relevante"**: `feat` deixou de gerar release |
| `create -d -t abc` (flag documentada no README) | `unknown shorthand flag: 't'` |
| tag não SemVer (`latest`) na ponta; `create -d` | erro `invalid semantic version` |
| `--version` | `unknown flag: --version` |
| `create -d` sem `GITHUB_TOKEN` e sem `gh` | falha por falta de token, embora o dry-run não use a rede |
| `git tag` e releases no GitHub | tags `v0.9.0-beta.1`..`v0.10.0-beta.5` fora da `main`; as releases `v0.5.0-beta.1`..`v0.8.0-beta.2` publicadas **sem** a marca de pré-release |
| `markdownlint-cli2` no README | 4 erros (MD004, MD026, MD047) |

## 3. Observações por severidade

### Críticas

- **C1** Cálculo errado depois de pré-release. `DetermineNextVersion` incrementa a partir da última tag, mesmo quando ela é uma pré-release (`internal/semver/semver.go:164-176`), e a busca da pré-release seguinte usa a base já incrementada (`semver.go:183-184`). Resultado: `v2.0.0-beta.1` → `v2.1.0` (estável) e → `v2.1.0-beta.1` (beta). É a causa das betas sucessivas `v0.6.0-beta.3` → `v0.7.0-beta.1` → `v0.8.0-beta.1`. → spec 001.
- **C2** Configuração parcial desativa os padrões. `yaml.Unmarshal` substitui a lista `releaseRules` inteira (`internal/config/config.go:63-65`), embora o comentário diga que o usuário só precisa definir o que muda. Um arquivo só com `docs` faz `feat` e `fix` não gerarem release, sem aviso. → spec 001.

### Altas

- **A1** Nenhum teste e nenhum CI de PR; só o workflow de release (`.github/workflows/release.yml`). → spec 003.
- **A2** Pré-releases publicadas como releases normais: o `.goreleaser.yml` não tem `prerelease: auto`, então a "Latest release" do GitHub é uma beta. → spec 003.
- **A3** A versão não é embutida: o `.goreleaser.yml` passa `-X main.version`, mas `main.go` não declara a variável, e não há `--version`. → spec 002.
- **A4** `main` atrás das tags publicadas: `v0.9.0-beta.1`..`v0.10.0-beta.5` estão em commits fora da `main`. Uma nova tag calculada na `main` parte da `v0.8.0-beta.2` (inferido de `git describe`). → decisão D6.
- **A5** O README documenta `-t/--token`, que não existe mais, e a instalação "só para Windows" com o `.exe` versionado. → spec 002.

### Médias

- **M1** O token é exigido até no `--dry-run`, e o pacote `internal/provider` não é usado: o push vai pelas credenciais do git (`cmd/create.go:52-56`). → spec 002.
- **M2** Tag não SemVer na ponta (`latest`, `nightly`) derruba o cálculo; `git describe --tags` não filtra. → spec 001.
- **M3** "Commits analisados" conta um a mais: o `split` por NUL deixa um item vazio no fim (`internal/git/git.go:56`). → spec 001.
- **M4** Não há saída para automação: a versão calculada só aparece no meio do texto colorido, o que impede o uso pelo plugin `sdd-release` e por uma GitHub Action. → spec 002.
- **M5** `go-release-manager.exe` (10 MB) versionado no git. → spec 003.

### Baixas

- **B1** Regex recompilada a cada parágrafo (SA6000) e ramo morto `if v.Major() == 0` com os dois lados iguais (`semver.go:166-171`).
- **B2** Comentários de histórico no código ("Intacto", "NOVO PACOTE IMPORTADO").
- **B3** Mensagens de erro do cobra sem quebra de linha no fim (`cmd/root.go`).

## 4. Pontos positivos (manter)

- Responsabilidade única e clara: calcular e criar a próxima tag, deixando o build para o GoReleaser.
- Parse de rodapé `BREAKING CHANGE:` e de `!` no cabeçalho, ambos corretos no caso estável.
- Dry-run e canais de pré-release com flags curtas.
- Regras configuráveis por tipo de commit em `.go-releaserc.yml`.
- GoReleaser com binários para Linux, macOS e Windows.

## 5. Avaliação do README

Bem escrito e com boa proposta de valor ("por que o go-release-manager"), mas desatualizado: a seção de instalação no topo duplica e contradiz a do meio (só Windows, `--token`), a ajuda mostrada não bate com a real, e não há como verificar os exemplos automaticamente.

## 6. Melhorias recomendadas (priorizadas)

1. **Fase 1 (P0):** testes do cálculo de versão e correção de C1, C2, M2 e M3; CI de PR; remover o `.exe`.
2. **Fase 2 (P1):** `--version`, token só quando faz push, saída para automação (`--output json`), README correto, `prerelease: auto` no GoReleaser.
3. **Fase 3 (P2):** GitHub Action reutilizável e integração com o plugin `sdd-release` do sdd-kit.

## 7. Decisões (respondidas pelo dono em 2026-10-02)

| ID | Pergunta | Opções | Recomendação |
|---|---|---|---|
| D1 | Como calcular a versão depois de uma pré-release? | (a) a base é a última tag **estável**; pré-releases do mesmo canal e da mesma base só incrementam o contador (`v2.0.0-beta.1` → `v2.0.0-beta.2`; estável → `v2.0.0`), como no semantic-release; (b) manter o comportamento atual e documentar | **(a)**: é o comportamento esperado do SemVer e corrige C1 — **respondida: (a)** |
| D2 | Commit incompatível enquanto a versão é `0.x`? | (a) manter: vai para `1.0.0`; (b) incrementar o minor enquanto for `0.x` (SemVer §4), com uma opção para sair do `0.x` | **(b)**: evita uma `1.0.0` acidental; a `1.0.0` vira uma decisão explícita — **respondida: (b)** |
| D3 | Como combinar o `.go-releaserc.yml` com as regras padrão? | (a) mesclar por tipo: o arquivo só sobrescreve os tipos que lista; (b) substituir tudo, mas avisar quando `feat` ou `fix` ficarem sem regra | **(a)**: é o que o comentário do código promete e corrige C2 — **respondida: (a)** |
| D4 | Saída para automação? | (a) `--output json` no `create` (versão, incremento, commits, tag criada ou não); (b) um subcomando `next` que imprime só a versão; (c) as duas | **(c)**: `next` para shell e Actions, JSON para o plugin — **respondida: (c)** |
| D5 | Token e o pacote `internal/provider`? | (a) exigir token só quando há push e remover o `provider`; (b) usar o `provider` para criar a release no GitHub | **(a)**: o GoReleaser já cria a release; menos código e dry-run sem credenciais — **respondida: (a)** |
| D6 | O que fazer com as tags fora da `main` (`v0.9.0-beta.1`..`v0.10.0-beta.5`)? | (a) manter as tags e seguir da `main`, com a primeira versão estável da Fase 1 em `v0.11.0`, acima de todas; (b) apagar essas tags (ação do dono) | **(a)**: não reescreve nada publicado e evita conflito de versão — **respondida: (a)** |
| D7 | Idioma das mensagens da CLI? | (a) manter em português; (b) inglês, com o README em português e inglês | **(a)** agora; inglês pode virar ticket na Fase 3 — **respondida: (a)** |
