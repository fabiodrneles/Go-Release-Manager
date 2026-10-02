package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-release-manager",
	Short: "", // Será preenchido no init
	Long:  "", // Será preenchido no init
}

func init() {
	color.NoColor = false
	// --- 1. Definir os estilos de cor ---
	// Cor para o Título ASCII (Ciano, Negrito)
	cTitle := color.New(color.FgCyan, color.Bold)

	// Cor para o Slogan (Amarelo)
	cTagline := color.New(color.FgYellow)

	// Cor para a descrição (Branco Padrão)
	cDesc := color.New(color.FgBlue)

	asciiArt := cTitle.Sprintf(`
	   
   ██████╗ ███████╗██╗     ███████╗ █████╗ ███████╗███████╗
   ██╔══██╗██╔════╝██║     ██╔════╝██╔══██╗██╔════╝██╔════╝
   ██████╔╝█████╗  ██║     █████╗  ███████║███████╗█████╗  
   ██╔══██╗██╔══╝  ██║     ██╔══╝  ██╔══██║╚════██║██╔══╝  
   ██║  ██║███████╗███████╗███████╗██║  ██║███████║███████╗
   ╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝
`)

	// --- 3. Criar Slogan e Descrição ---
	tagline := cTagline.Sprint("\n  Automatize seu versionamento e releases com Conventional Commits.")
	description := cDesc.Sprint(`
    Esta ferramenta lê a última tag, analisa os commits desde então e
    determina o próximo incremento de versão (Major, Minor ou Patch).`)

	// --- 4. Definir o Short e Long do rootCmd ---

	// Short: Uma única linha, como antes
	rootCmd.Short = color.CyanString("Uma CLI para automatizar o versionamento semântico e releases.")

	// Long: A versão "profissional" com ASCII art
	rootCmd.Long = fmt.Sprintf("%s\n%s\n%s", asciiArt, tagline, description)
}

// Execute runs the CLI. version and commit come from the release build.
func Execute(version, commit string) {
	rootCmd.Version = version
	if commit != "" {
		rootCmd.Version = version + " (" + commit + ")"
	}
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	// Errors go to stderr once, with a trailing newline (spec 002 FR-5).
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Erro: %v", err))
		os.Exit(1)
	}
}
