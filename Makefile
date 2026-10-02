# Verificação local igual à do CI: rode `make ci` antes de todo push.
GOLANGCI_LINT_VERSION := v2.14.0

.DEFAULT_GOAL := ci

.PHONY: ci
ci: lint test build ## Tudo o que o CI verifica

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run

.PHONY: test
test: ## Testes com race detector e cobertura
	go test -race -cover ./...

.PHONY: build
build: ## Compila tudo
	go build ./...

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (falha se houver aviso)
	sh scripts/sdd-check.sh --strict
