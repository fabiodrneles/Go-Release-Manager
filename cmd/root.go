package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "go-release-manager",
	// Deixe os campos em branco (ou com texto simples)
	// Eles serão preenchidos pela função init()
	Short: "",
	Long:  "",
}

// A MÁGICA ESTÁ AQUI: init() é executado após a inicialização dos pacotes.
// Aqui é o local seguro para usar a biblioteca de cores.
func init() {
	rootCmd.Short = color.CyanString("Uma CLI para automatizar o versionamento semântico e releases.")
	rootCmd.Long = color.WhiteString(`
go-release-manager automatiza o processo de criação de tags e releases no GitHub
analisando seu histórico de commits baseado no padrão Conventional Commits.

Esta ferramenta lê a última tag, analisa os commits desde então e
determina o próximo incremento de versão (Major, Minor ou Patch).`)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// A cor no Execute() estava correta e pode ficar
		fmt.Fprintf(os.Stderr, color.RedString("Ocorreu um erro: '%s'"), err)
		os.Exit(1)
	}
}
