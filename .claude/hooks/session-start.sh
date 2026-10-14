#!/bin/bash
# Prepara o ambiente do Claude Code na web para rodar `make ci` sem instalar
# nada no meio do trabalho. Só roda em sessões remotas.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(pwd)}"

# Spec 014: o ponto de retomada, a branch, os PRs e as issues do épico aberto entram no contexto da sessão.
sh scripts/sdd-resume.sh 2>/dev/null || true

if [ -f go.mod ]; then
  go mod download
  # A toolchain que o GOTOOLCHAIN baixa para a versão do go.mod não traz o
  # covdata, e `go test -coverprofile ./...` falha num pacote sem testes
  # ("no such tool covdata"). Compila o que falta na própria toolchain.
  tooldir="$(go env GOTOOLDIR)"
  if [ ! -x "$tooldir/covdata" ]; then
    chmod u+w "$tooldir"
    go build -o "$tooldir/covdata" cmd/covdata
  fi
fi

# golangci-lint na mesma versão do CI (lida do Makefile).
want="$(sed -n 's/^GOLANGCI_LINT_VERSION *:= *//p' Makefile)"
gobin="$(go env GOPATH)/bin"
if ! "$gobin/golangci-lint" version 2>/dev/null | grep -q "version ${want#v} "; then
  GOBIN="$gobin" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$want"
fi
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
  echo "export PATH=\"$gobin:\$PATH\"" >> "$CLAUDE_ENV_FILE"
fi
