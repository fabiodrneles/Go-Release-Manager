package cmd

import (
	"fmt"
	"log"

	// "os" // <-- REMOVIDO (movido para o pacote auth)

	"go-release-manager/internal/auth"   // <-- NOVO PACOTE IMPORTADO
	"go-release-manager/internal/config" // Importação existente
	"go-release-manager/internal/git"
	"go-release-manager/internal/semver"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// token string // <-- Já removido
	dryRun            bool
	preReleaseChannel string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: color.CyanString("Cria e empurra uma nova tag semântica."),
	Long: color.WhiteString(`Analisa os commits desde a última tag, determina a próxima versão semântica,
cria e empurra a tag. O release do GitHub (com os binários) será criado automaticamente pela GitHub Action.`),

	// --- EXEMPLO ATUALIZADO (com nova autenticação) ---
	Example: color.YellowString(`
  # Executa o comando (lê GITHUB_TOKEN ou token do 'gh')
  go-release-manager create

  # Simula o processo (dry-run)
  go-release-manager create -d

  # Cria uma pré-release
  go-release-manager create -p beta

  # Simula uma pré-release
  # (Autenticação é automática via GITHUB_TOKEN ou 'gh auth login')
  go-release-manager create -d -p rc
`),
	// --- FIM DA ATUALIZAÇÃO ---

	Run: func(cmd *cobra.Command, args []string) {

		// --- LÓGICA DE AUTENTICAÇÃO (ATUALIZADA) ---
		// Tenta GITHUB_TOKEN, e se falhar, tenta 'gh auth token'
		// O token em si não é usado diretamente aqui, mas o 'gh' configura o git.
		// A verificação é crucial para falhar rápido se nenhuma auth estiver disponível.
		_, err := auth.GetToken()
		if err != nil {
			log.Fatalf("%s", color.RedString("Erro: Token de acesso não fornecido.\nDefina-o pela variável de ambiente GITHUB_TOKEN, ou faça login com o GitHub CLI (`gh auth login`).\nErro original: %v", err))
		}
		// --- FIM DA LÓGICA DE AUTENTICAÇÃO ---

		// --- CARREGAR CONFIGURAÇÃO (Intacto) ---
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf(color.RedString("Erro ao carregar configuração .go-releaserc.yml: %v"), err)
		}
		// --- FIM DO CARREGAMENTO ---

		// 1. Obter a última tag (Intacto)
		latestTag, err := git.GetLatestTag()
		if err != nil {
			log.Fatalf(color.RedString("Erro ao obter a última tag: %v"), err)
		}
		log.Printf(color.GreenString("Última versão encontrada: %s"), latestTag)

		// 2. Obter commits (Intacto)
		commits, err := git.GetCommitsSince(latestTag)
		if err != nil {
			log.Fatalf(color.RedString("Erro ao obter commits: %v"), err)
		}
		log.Printf("Analisando %d commits desde a tag %s...", len(commits), latestTag)

		// 3. DETERMINAR A PRÓXIMA VERSÃO (Intacto)
		if preReleaseChannel != "" {
			log.Printf(color.CyanString("Modo de pré-release ativado. Canal: %s"), preReleaseChannel)
		}

		nextVersion, increment, err := semver.DetermineNextVersion(cfg, latestTag, commits, preReleaseChannel)
		if err != nil {
			log.Fatalf(color.RedString("Erro ao determinar a próxima versão: %v"), err)
		}

		if increment == semver.IncrementNone {
			log.Println(color.YellowString("Nenhuma mudança relevante encontrada (feat, fix, BREAKING CHANGE, etc.). Nenhum release será criado."))
			return
		}
		log.Printf(color.GreenString("Tipo de incremento: %s. Nova versão calculada: %s"), increment, nextVersion)

		// 4. SE FOR --dry-run (INTACTO)
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

		// --- LÓGICA RESTANTE (INTACTA) ---

		log.Printf(color.GreenString("✅ Tag %s criada e empurrada com sucesso!"), nextVersion)
		log.Println(color.CyanString("A GitHub Action 'Release' foi acionada. Verifique seu repositório em alguns minutos para os binários."))
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// --- ATUALIZADO (Exemplos com nova auth) ---
	createCmd.Example = color.YellowString(
		`
  # Executa o comando (lê GITHUB_TOKEN ou token do 'gh')
  go-release-manager create

  # Simula o processo (dry-run)
  go-release-manager create -d

  # Cria uma pré-release
  go-release-manager create -p beta

  # Simula uma pré-release
  # (Autenticação é automática via GITHUB_TOKEN ou 'gh auth login')
  go-release-manager create -d -p rc
`)
	// --- FIM DA ATUALIZAÇÃO ---

	// --- FLAGS (Intactas) ---
	// Flag de Token (REMOVIDA)

	// Flag de Dry-Run (Intacta)
	createCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Simula o processo sem criar tags ou releases")

	// Flag de Pré-Release (Intacta)
	createCmd.Flags().StringVarP(&preReleaseChannel, "pre-release", "p", "", "Cria uma pré-release com o canal especificado (ex: beta, rc)")
}
