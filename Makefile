# Verificação local igual à do CI: rode `make ci` antes de todo push.
GOLANGCI_LINT_VERSION := v2.14.0
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80

# Artefatos do make ci (cobertura) ficam dentro de .git/, que o git nunca versiona:
# nada solto na árvore depois do make ci, sem depender do .gitignore (#180).
SDD_OUT := $(shell git rev-parse --git-path sdd-out 2>/dev/null || echo .sdd-out)

.DEFAULT_GOAL := ci

.PHONY: ci
ci: lint test build no-binaries go-version doc-commands ## Tudo o que o CI verifica

.PHONY: lint
lint: ## golangci-lint (prefere o do GOPATH/bin, que o sdd-doctor.sh instala na versão certa)
	$(or $(wildcard $(shell go env GOPATH)/bin/golangci-lint),golangci-lint) run

.PHONY: test
test: ## Testes com race detector e cobertura mínima (COVERAGE_MIN)
	@# -count=1: sem ele, com o cache quente e um package main em -coverpkg=./..., o go reaproveita
	@# metadados de cobertura de uma versão antiga do main.go e a cobertura sai menor que a real (#172).
	@# sdd-cover-guard.sh confere o perfil e, se ainda vier misturado, refaz com um cache frio.
	@mkdir -p $(SDD_OUT)
	sh scripts/sdd-cover-guard.sh $(SDD_OUT)/coverage.out -- go test -count=1 -race -coverpkg=./... -coverprofile=$(SDD_OUT)/coverage.out ./...
	@total="$$(go tool cover -func=$(SDD_OUT)/coverage.out | awk '/^total:/ { sub("%", "", $$3); print $$3 }')"; \
		awk -v t="$$total" -v m="$(COVERAGE_MIN)" 'BEGIN { printf "cobertura: %.1f%% (mínimo %s%%)\n", t, m; exit !(t + 0 >= m + 0) }' || \
		{ echo "cobertura abaixo do mínimo; suba os testes ou ajuste COVERAGE_MIN no Makefile" >&2; exit 1; }

.PHONY: build
build: ## Compila tudo
	go build ./...

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3

.PHONY: linkcheck
linkcheck: ## Links quebrados nos .md (precisa do lychee; o CI sempre verifica)
	@command -v lychee >/dev/null || { echo "lychee não instalado: https://lychee.cli.rs" >&2; exit 1; }
	lychee --config lychee.toml --no-progress './**/*.md'

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (falha se houver aviso)
	sh scripts/sdd-check.sh --strict

.PHONY: no-binaries
no-binaries: ## Nenhum binário versionado (spec 003 AC-3)
	sh scripts/check-binaries.sh

.PHONY: go-version
go-version: ## Workflows e Dockerfile seguem a versão do go.mod
	sh scripts/check-go-version.sh

.PHONY: doc-commands
doc-commands: ## Os blocos bash do README funcionam (spec 002 AC-5)
	sh scripts/doc-commands.sh
