#!/bin/bash
# Prepara o ambiente do Claude Code na web para rodar `make ci` sem instalar
# nada no meio do trabalho. Só roda em sessões remotas.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(pwd)}"

if [ -f go.mod ]; then
  go mod download
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
