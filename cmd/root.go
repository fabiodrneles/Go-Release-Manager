package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// --- NOVO (Intacto) ---
var (
	appVersion = "dev"
	appCommit  = "none"
)

// SetVersionInfo é chamada pelo main.go para passar as informações de build
func SetVersionInfo(version, commit string) {
	appVersion = version
	appCommit = commit
}

// --- FIM DO NOVO ---

var rootCmd = &cobra.Command{
	Use:     "go-release-manager",
	Short:   "",
	Long:    "",
	Version: " ",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().NFlag() == 0 && len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

func init() {
	color.NoColor = false
	cTitle := color.New(color.FgCyan, color.Bold)
	cTagline := color.New(color.FgYellow)
	cDesc := color.New(color.FgBlue)

	asciiArt := cTitle.Sprintf(`
			ЯΣᄂΣΛƧΣ MΛПΛGΣЯ
`) // O seu ASCII art está aqui

	tagline := cTagline.Sprint("\n  Automatize seu versionamento e releases com Conventional Commits.")
	description := cDesc.Sprint(`
    Esta ferramenta lê a última tag, analisa os commits desde então e
    determina o próximo incremento de versão (Major, Minor ou Patch).`)

	rootCmd.Short = color.CyanString("Uma CLI para automatizar o versionamento semântico e releases.")
	rootCmd.Long = fmt.Sprintf("%s\n%s\n%s", asciiArt, tagline, description)

	// Adiciona a flag --version/-v
	var showVersion bool
	// Note que a flag está declarada como Persistente para funcionar em qualquer subcomando
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Exibe a versão da aplicação")

	// Usamos PersistentPreRunE para verificar a flag *antes* de executar qualquer comando.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("go-release-manager version %s (commit: %s)\n", appVersion, appCommit)
			os.Exit(0) // Sai com sucesso após exibir a versão
		}
		return nil // Continua a execução normal se a flag não foi usada
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Ocorreu um erro: '%s'"), err)
		os.Exit(1)
	}
}
