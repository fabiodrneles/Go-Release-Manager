package cmd

import (
	"fmt"
	"log"
	"os"

	"go-release-manager/internal/git"
	"go-release-manager/internal/semver"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	token             string
	dryRun            bool
	preReleaseChannel string // <-- NOVA VARIÁVEL
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: color.CyanString("Cria e empurra uma nova tag semântica."),
	Long: color.WhiteString(`Analisa os commits desde a última tag, determina a próxima versão semântica,
cria e empurra a tag. O release do GitHub (com os binários) será criado automaticamente pela GitHub Action.`),

	// --- EXEMPLO ATUALIZADO ---
	Example: color.YellowString(`
  # Executa o comando (lê o token do GITHUB_TOKEN)
  # A ferramenta irá criar e empurrar a tag, acionando o GoReleaser no GitHub.
  go-release-manager create

  # Executa em modo "dry run" (simulação) para ver a tag que será criada
  go-release-manager create --dry-run

  # Cria uma pré-release (ex: v1.3.0-beta.1)
  go-release-manager create --pre-release beta
`),
	// --- FIM DA ATUALIZAÇÃO ---

	Run: func(cmd *cobra.Command, args []string) {

		// --- LÓGICA DE SEGURANÇA (INTACTA) ---
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
		if token == "" {
			log.Fatalf("%s", color.RedString("Erro: Token de acesso não fornecido.\nDefina-o via flag --token ou pela variável de ambiente GITHUB_TOKEN."))
		}
		// --- FIM DA LÓGICA DE SEGURANÇA ---

		// 1. Obter a última tag (INTACTA)
		latestTag, err := git.GetLatestTag()
		if err != nil {
			log.Fatalf(color.RedString("Erro ao obter a última tag: %v"), err)
		}
		log.Printf(color.GreenString("Última versão encontrada: %s"), latestTag)

		// 2. Obter commits (INTACTA)
		commits, err := git.GetCommitsSince(latestTag)
		if err != nil {
			log.Fatalf(color.RedString("Erro ao obter commits: %v"), err)
		}
		log.Printf("Analisando %d commits desde a tag %s...", len(commits), latestTag)

		// --- 3. DETERMINAR A PRÓXIMA VERSÃO (ATUALIZADO) ---
		if preReleaseChannel != "" {
			log.Printf(color.CyanString("Modo de pré-release ativado. Canal: %s"), preReleaseChannel)
		}

		// A chamada agora passa 'preReleaseChannel' e espera um 'err'.
		nextVersion, increment, err := semver.DetermineNextVersion(latestTag, commits, preReleaseChannel)
		if err != nil { // <-- NOVO TRATAMENTO DE ERRO
			log.Fatalf(color.RedString("Erro ao determinar a próxima versão: %v"), err)
		}

		if increment == semver.IncrementNone {
			log.Println(color.YellowString("Nenhuma mudança relevante encontrada (feat, fix, BREAKING CHANGE). Nenhum release será criado."))
			return
		}
		log.Printf(color.GreenString("Tipo de incremento: %s. Nova versão calculada: %s"), increment, nextVersion)

		// --- 4. SE FOR --dry-run (ATUALIZADO) ---
		// O dry-run agora mostra o canal de pré-release
		if dryRun {
			fmt.Println(color.CyanString("\n--- MODO DRY RUN (SIMULAÇÃO) ---"))
			fmt.Printf("Última tag encontrada: %s\n", latestTag)
			if preReleaseChannel != "" {
				fmt.Printf("Canal de pré-release: %s\n", preReleaseChannel)
			}
			fmt.Printf("Commits analisados: %d\n", len(commits))
			fmt.Printf("Decisão de incremento: %s\n", color.MagentaString(increment.String()))
			fmt.Printf("A nova tag a ser criada seria: %s\n", color.MagentaString(nextVersion))
			fmt.Println(color.CyanString("--- FIM DO DRY RUN ---"))
			return
		}

		// 5. Criar e empurrar a tag (INTACTA)
		log.Printf("Criando tag git '%s'...", nextVersion)
		if err := git.CreateTag(nextVersion); err != nil {
			log.Fatalf(color.RedString("Erro ao criar tag: %v"), err)
		}

		log.Printf("Empurrando tag '%s' para o repositório remoto...", nextVersion)
		if err := git.PushTag(nextVersion); err != nil {
			log.Fatalf(color.RedString("Erro ao empurrar tag: %v"), err)
		}

		// --- 6. CRIAR O RELEASE NO PROVEDOR (INTACTO, JÁ REMOVIDO) ---

		// --- NOVA MENSAGEM DE SUCESSO (INTACTA) ---
		log.Printf(color.GreenString("✅ Tag %s criada e empurrada com sucesso!"), nextVersion)
		log.Println(color.CyanString("A GitHub Action 'Release' foi acionada. Verifique seu repositório em alguns minutos para os binários."))
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// --- ATUALIZADO (Short/Long intactos) ---
	createCmd.Short = color.CyanString("Cria e empurra uma nova tag semântica.")
	createCmd.Long = color.WhiteString(
		`Analisa os commits desde a última tag, determina a próxima versão semântica,
cria e empurra a tag. O release do GitHub (com os binários) será criado automaticamente pela GitHub Action.
`)
	// --- EXEMPLO ATUALIZADO ---
	createCmd.Example = color.YellowString(
		`
  # Executa o comando (lê o token do GITHUB_TOKEN)
  # A ferramenta irá criar e empurrar a tag, acionando o GoReleaser no GitHub.
  go-release-manager create

  # Executa em modo "dry run" (simulação) para ver a tag que será criada
  go-release-manager create --dry-run

  # Cria uma pré-release (ex: v1.3.0-beta.1)
  go-release-manager create --pre-release beta

  # Simula uma pré-release
  go-release-manager create --dry-run --pre-release rc
`)
	// --- FIM DA ATUALIZAÇÃO ---

	// Flags (INTACTAS)
	createCmd.Flags().StringVarP(&token, "token", "t", "", "Token de Acesso Pessoal (PAT) do GitHub. (Padrão: variável de ambiente GITHUB_TOKEN)")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simula o processo sem criar tags ou releases")

	// --- NOVA FLAG ---
	createCmd.Flags().StringVar(&preReleaseChannel, "pre-release", "", "Cria uma pré-release com o canal especificado (ex: beta, rc). Gera tags como vX.Y.Z-canal.N")
}
