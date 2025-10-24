package main

import (
	"go-release-manager/cmd"
)

// --- CORREÇÃO: DECLARAÇÃO SEM INICIALIZAÇÃO ---
// Estas variáveis devem ser declaradas sem valor inicial para que o ldflags funcione.
// O GoReleaser preenche main.version e main.commit.
var (
	version string
	commit  string
)

// --- FIM DA CORREÇÃO ---

func main() {
	// O valor padrão ("dev" / "none") é setado AQUI,
	// após a declaração, para que o ldflags (GoReleaser) possa ter prioridade no build.
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "none"
	}

	cmd.SetVersionInfo(version, commit)
	cmd.Execute()
}
