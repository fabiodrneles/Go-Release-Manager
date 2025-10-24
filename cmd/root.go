package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// --- NOVO ---
// Variáveis para armazenar a informação de versão vinda do main.go
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
	Use:   "go-release-manager",
	Short: "", // Será preenchido no init
	Long:  "", // Será preenchido no init
	// --- NOVO ---
	// Desabilita o comando padrão 'version' do Cobra, pois criaremos o nosso
	Version: " ", // Um espaço em branco desabilita
	RunE: func(cmd *cobra.Command, args []string) error {
		// Se nenhuma flag for passada (além de --version), mostra a ajuda
		if cmd.Flags().NFlag() == 0 && len(args) == 0 {
			return cmd.Help()
		}
		// Se --version foi passado, o PersistentPreRun já tratou.
		// Se outro comando foi chamado (ex: 'create'), ele será executado.
		return nil
	},
	// --- FIM DO NOVO ---
}

func init() {
	color.NoColor = false
	cTitle := color.New(color.FgCyan, color.Bold)
	cTagline := color.New(color.FgYellow)
	cDesc := color.New(color.BgBlue)

	// ASCII Art (Intacto)
	asciiArt := cTitle.Sprintf(`
			ЯΣᄂΣΛƧΣ MΛПΛGΣЯ
`) // Use ` em vez de ' para strings multi-linha

	tagline := cTagline.Sprint("\n  Automatize seu versionamento e releases com Conventional Commits.")
	description := cDesc.Sprint(`
    Esta ferramenta lê a última tag, analisa os commits desde então e
    determina o próximo incremento de versão (Major, Minor ou Patch).`)

	rootCmd.Short = color.CyanString("Uma CLI para automatizar o versionamento semântico e releases.")
	rootCmd.Long = fmt.Sprintf("%s\n%s\n%s", asciiArt, tagline, description)

	// --- NOVO ---
	// Adiciona a flag --version/-v
	var showVersion bool
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Exibe a versão da aplicação")

	// Usamos PersistentPreRunE para verificar a flag *antes* de executar qualquer comando.
	// Se -v ou --version for encontrado, ele imprime a versão e *sai* do programa.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("go-release-manager version %s (commit: %s)\n", appVersion, appCommit)
			os.Exit(0) // Sai com sucesso após exibir a versão
		}
		return nil // Continua a execução normal se a flag não foi usada
	}
	// --- FIM DO NOVO ---
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Ocorreu um erro: '%s'"), err)
		os.Exit(1)
	}
}
