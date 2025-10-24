package main

import (
	"go-release-manager/cmd"
)

// --- NOVO ---
// Estas variáveis serão preenchidas pelo GoReleaser durante o build
// usando a flag -ldflags (ex: -X main.version=v1.2.3).
// Veja a configuração em .goreleaser.yml
var (
	version = "dev"  // Valor padrão se não for buildado com GoReleaser
	commit  = "none" // Valor padrão
	// date    = "unknown" // Você pode adicionar a data também se quiser
)

// --- FIM DO NOVO ---

func main() {
	// --- NOVO ---
	// Passa a versão para o pacote cmd para que ele possa exibi-la
	cmd.SetVersionInfo(version, commit)
	// --- FIM DO NOVO ---

	cmd.Execute()
}
