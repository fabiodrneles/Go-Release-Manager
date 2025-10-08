package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-release-manager",
	Short: "Uma CLI para automatizar o versionamento semântico e releases.",
	Long: `go-release-manager automatiza o processo de criação de tags e releases no GitHub
analisando seu histórico de commits baseado no padrão Conventional Commits.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Ocorreu um erro: '%s'", err)
		os.Exit(1)
	}
}
