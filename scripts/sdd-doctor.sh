#!/bin/sh
# Confere (e conserta) o ambiente local para o `make ci` se comportar como o CI,
# sob demanda, mesmo quando o hook de início de sessão não rodou (ex.: segundo
# repositório anexado no meio da sessão). Rode na raiz do projeto.
#
# Uso: sdd-doctor.sh [--check]
#   --check   só relata; nunca instala nem altera nada
#
# Uma linha por item: `ok <item>`, `corrigido <item>` ou `FALHA <item>: <como corrigir>`.
# Sai com 0 se tudo terminou ok e com 1 se sobrou alguma FALHA.
set -u

check=0
case "${1:-}" in
  "") ;;
  --check) check=1 ;;
  -h | --help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
  *) echo "uso: sdd-doctor.sh [--check]" >&2; exit 3 ;;
esac

rc=0
tmp="${TMPDIR:-/tmp}"
ok() { echo "ok $*"; }
fixed() { echo "corrigido $*"; }
bad() { echo "FALHA $*"; rc=1; }
have() { command -v "$1" >/dev/null 2>&1; }
# Valor de uma variável `NOME := valor` do Makefile (vazio se não houver).
mkvar() { sed -n "s/^$1 *:\\{0,1\\}= *//p" Makefile 2>/dev/null | head -n 1; }
# Grava uma linha no arquivo de ambiente da sessão, sem repetir.
envline() {
  [ "$check" -eq 0 ] && [ -n "${CLAUDE_ENV_FILE:-}" ] || return 1
  grep -qxF "$1" "$CLAUDE_ENV_FILE" 2>/dev/null || echo "$1" >> "$CLAUDE_ENV_FILE"
}
# Erro que só uma variável de ambiente resolve: com CLAUDE_ENV_FILE o próximo
# comando da sessão já vem certo (corrigido); sem ele, FALHA com a linha a usar.
needs_env() { # item, linha de export
  if envline "$2"; then
    fixed "$1: $2 (gravado em CLAUDE_ENV_FILE; vale a partir do próximo comando)"
  else
    bad "$1: $2"
  fi
}
# Ferramenta obrigatória no PATH.
need() { # comando, como instalar
  if have "$1"; then ok "$1"; else bad "$1: $2"; fi
}

# --- Locale: o shellcheck quebra fora de UTF-8 ------------------------------
loc="${LC_ALL:-${LC_CTYPE:-${LANG:-}}}"
case "$loc" in
  *[Uu][Tt][Ff]-8* | *[Uu][Tt][Ff]8*) ok "locale ($loc)" ;;
  *)
    want_loc="C.UTF-8"
    [ "$(uname -s)" = Darwin ] && want_loc="en_US.UTF-8"
    needs_env "locale (${loc:-vazio} não é UTF-8; o shellcheck quebra)" "export LC_ALL=$want_loc"
    ;;
esac

# --- Raiz do próprio sdd-kit: shellcheck e actionlint nas versões do CI -----
if [ -f Makefile ] && grep -q '^SHELLCHECK_VERSION' Makefile; then
  bin="$HOME/.local/bin"
  plat=""
  aplat=""
  if [ "$(uname -s)-$(uname -m)" = "Linux-x86_64" ]; then
    plat="linux.x86_64"
    aplat="linux_amd64"
  fi
  can_download() { [ "$check" -eq 0 ] && [ -n "$plat" ] && have curl && have tar && mkdir -p "$bin"; }

  want="$(mkvar SHELLCHECK_VERSION)"
  sc="$(command -v shellcheck || echo "$bin/shellcheck")"
  if "$sc" --version 2>/dev/null | grep -q "version: ${want#v}$"; then
    ok "shellcheck ${want#v}"
  elif can_download &&
    curl -fsSL "https://github.com/koalaman/shellcheck/releases/download/${want}/shellcheck-${want}.${plat}.tar.xz" | tar -xJ -C "$tmp" &&
    cp "$tmp/shellcheck-${want}/shellcheck" "$bin/"; then
    fixed "shellcheck ${want#v} (em $bin)"
    envline "export PATH=\"$bin:\$PATH\"" || true
  else
    bad "shellcheck: ausente ou diferente de ${want#v}; instale a ${want} (https://github.com/koalaman/shellcheck/releases) e ponha antes no PATH"
  fi

  want="$(mkvar ACTIONLINT_VERSION)"
  al="$(command -v actionlint || echo "$bin/actionlint")"
  if "$al" --version 2>/dev/null | grep -qx "${want#v}"; then
    ok "actionlint ${want#v}"
  elif can_download &&
    curl -fsSL "https://github.com/rhysd/actionlint/releases/download/${want}/actionlint_${want#v}_${aplat}.tar.gz" | tar -xz -C "$bin" actionlint; then
    fixed "actionlint ${want#v} (em $bin)"
    envline "export PATH=\"$bin:\$PATH\"" || true
  else
    bad "actionlint: ausente ou diferente de ${want#v}; instale a ${want} (https://github.com/rhysd/actionlint/releases) e ponha antes no PATH"
  fi
fi

# --- Markdown: markdownlint-cli2 via npx (cache), lychee opcional -----------
if have npx; then
  ok "npx"
  mdv="$(mkvar MARKDOWNLINT_VERSION)"
  [ -n "$mdv" ] || mdv="$(sed -n 's/.*markdownlint-cli2@\([0-9][0-9.]*\).*/\1/p' Makefile 2>/dev/null | head -n 1)"
  if [ -n "$mdv" ] && [ "$check" -eq 0 ]; then
    if npx --yes "markdownlint-cli2@$mdv" --help >/dev/null 2>&1; then
      ok "markdownlint-cli2 $mdv"
    else
      bad "markdownlint-cli2 $mdv: o npx não baixou (rede/proxy?); rode 'npx --yes markdownlint-cli2@$mdv --help'"
    fi
  fi
else
  bad "npx: instale o Node.js (necessário para o markdownlint do make ci)"
fi
if have lychee; then ok "lychee"; else ok "lychee (opcional, ausente: o CI verifica os links)"; fi

# --- Linguagem do projeto, detectada pelos arquivos -------------------------
if [ -f go.mod ]; then
  need go "instale o Go (a versão do go.mod; https://go.dev/dl/)"
  if have go; then
    # A toolchain baixada pelo GOTOOLCHAIN não traz o covdata.
    tooldir="$(go env GOTOOLDIR)"
    if [ -x "$tooldir/covdata" ]; then
      ok "covdata"
    elif [ "$check" -eq 1 ]; then
      bad "covdata: ausente em $tooldir; rode sh scripts/sdd-doctor.sh (compila: go build -o $tooldir/covdata cmd/covdata)"
    elif chmod u+w "$tooldir" 2>/dev/null && go build -o "$tooldir/covdata" cmd/covdata >/dev/null 2>&1; then
      fixed "covdata (compilado em $tooldir)"
    else
      bad "covdata: não compilou; rode 'go build -o $tooldir/covdata cmd/covdata' com permissão de escrita (ou use uma instalação completa do Go)"
    fi

    # golangci-lint na versão do Makefile, no GOPATH/bin.
    want="$(mkvar GOLANGCI_LINT_VERSION)"
    gobin="$(go env GOPATH)/bin"
    if [ -n "$want" ]; then
      if "$gobin/golangci-lint" version 2>/dev/null | grep -q "version ${want#v} "; then
        ok "golangci-lint ${want#v}"
      elif [ "$check" -eq 1 ]; then
        bad "golangci-lint: falta a ${want} em $gobin; rode sh scripts/sdd-doctor.sh (GOBIN=$gobin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$want)"
      elif GOBIN="$gobin" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$want" >/dev/null 2>&1; then
        fixed "golangci-lint ${want#v} (instalado em $gobin)"
      else
        bad "golangci-lint: a instalação falhou; rode 'GOBIN=$gobin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$want'"
      fi
      # Uma cópia antiga antes no PATH esconde a certa, a menos que o Makefile
      # já prefira o GOPATH/bin.
      first="$(command -v golangci-lint 2>/dev/null || true)"
      if [ -x "$gobin/golangci-lint" ] && [ -n "$first" ] && [ "$first" != "$gobin/golangci-lint" ]; then
        if grep -q 'go env GOPATH' Makefile 2>/dev/null; then
          ok "golangci-lint em $first fica de lado: o make lint usa $gobin"
        else
          needs_env "golangci-lint em $first esconde o de $gobin" "export PATH=\"$gobin:\$PATH\""
        fi
      fi
    fi
  fi
fi

if [ -f package.json ]; then
  need node "instale o Node.js (a versão do CI)"
  need npm "instale o npm junto com o Node.js"
  if [ -d node_modules ]; then
    ok "node_modules"
  elif [ "$check" -eq 1 ] || [ ! -f package-lock.json ]; then
    bad "node_modules: ausente; rode 'npm ci --no-audit --no-fund'"
  elif have npm && npm ci --no-audit --no-fund >/dev/null 2>&1; then
    fixed "node_modules (npm ci)"
  else
    bad "node_modules: o 'npm ci' falhou; rode 'npm ci --no-audit --no-fund' e leia o erro"
  fi
fi

if [ -f pom.xml ] || [ -f build.gradle ] || [ -f build.gradle.kts ]; then
  need java "instale o JDK (a versão do CI)"
  if [ -f pom.xml ]; then
    if [ -x ./mvnw ] || have mvn; then ok "maven"; else bad "maven: instale o Maven ou adicione o mvnw"; fi
  else
    if [ -x ./gradlew ] || have gradle; then ok "gradle"; else bad "gradle: instale o Gradle ou adicione o gradlew"; fi
  fi
fi

if [ -f pyproject.toml ]; then
  need python3 "instale o Python 3 (a versão do CI)"
  if have python3; then
    if python3 -c 'import pytest' >/dev/null 2>&1; then
      ok "dependências de desenvolvimento"
    elif [ "$check" -eq 1 ]; then
      bad "dependências de desenvolvimento: faltam; rode make deps (pip install -e \".[dev]\")"
    elif python3 -m pip install -q -e ".[dev]" >/dev/null 2>&1; then
      fixed "dependências de desenvolvimento (pip install -e .[dev])"
    else
      bad "dependências de desenvolvimento: o pip falhou; rode 'python3 -m pip install -e \".[dev]\"'"
    fi
  fi
fi

if [ -f Cargo.toml ]; then
  need cargo "instale o Rust (https://rustup.rs)"
  if have cargo; then
    if cargo clippy --version >/dev/null 2>&1 && cargo fmt --version >/dev/null 2>&1 && cargo llvm-cov --version >/dev/null 2>&1; then
      ok "clippy, rustfmt e cargo-llvm-cov"
    elif [ "$check" -eq 1 ]; then
      bad "clippy/rustfmt/cargo-llvm-cov: falta algum; rode make deps"
    elif make deps >/dev/null 2>&1; then
      fixed "clippy, rustfmt e cargo-llvm-cov (make deps)"
    else
      bad "clippy/rustfmt/cargo-llvm-cov: o 'make deps' falhou; rode-o e leia o erro"
    fi
  fi
fi

if ls ./*.sln ./*.csproj >/dev/null 2>&1; then
  need dotnet "instale o .NET SDK (a versão do CI)"
fi

exit "$rc"
