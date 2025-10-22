package cmd

import (
	"context"
	"fmt"
	"log"

	"go-release-manager/internal/git"
	"go-release-manager/internal/provider"
	"go-release-manager/internal/semver"

	"github.com/fatih/color" // Importar
	"github.com/spf13/cobra"
)

var (
	token   string
	dryRun  bool
	repoURL string
)

var createCmd = &cobra.Command{
	Use: "create",
	// Deixar em branco, será preenchido no init()
	Short:   "",
	Long:    "",
	Example: "",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Obter a última tag
		latestTag, err := git.GetLatestTag()
		if err != nil {
			log.Fatalf(color.RedString("Erro ao obter a última tag: %v"), err)
		}
		log.Printf(color.GreenString("Última versão encontrada: %s"), latestTag)

		// 2. Obter commits desde a última tag
		commits, err := git.GetCommitsSince(latestTag)
		if err != nil {
			log.Fatalf(color.RedString("Erro ao obter commits: %v"), err)
		}
		log.Printf("Analisando %d commits desde a tag %s...", len(commits), latestTag)

		// 3. Determinar a próxima versão
		nextVersion, increment, changelog := semver.DetermineNextVersion(latestTag, commits)
		if increment == semver.IncrementNone {
			log.Println(color.YellowString("Nenhuma mudança relevante encontrada (feat, fix, BREAKING CHANGE). Nenhum release será criado."))
			return
		}
		log.Printf(color.GreenString("Tipo de incremento: %s. Nova versão calculada: %s"), increment, nextVersion)

		// 4. Se for --dry-run, apenas mostrar o que seria feito
		if dryRun {
			fmt.Println(color.CyanString("\n--- MODO DRY RUN (SIMULAÇÃO) ---"))
			fmt.Printf("A nova tag a ser criada seria: %s\n", color.MagentaString(nextVersion))
			fmt.Println("Changelog gerado:")
			fmt.Println(changelog)
			fmt.Println(color.CyanString("--- FIM DO DRY RUN ---"))
			return
		}

		// 5. Criar e empurrar a tag
		log.Printf("Criando tag git '%s'...", nextVersion)
		if err := git.CreateTag(nextVersion); err != nil {
			log.Fatalf(color.RedString("Erro ao criar tag: %v"), err)
		}

		log.Printf("Empurrando tag '%s' para o repositório remoto...", nextVersion)
		if err := git.PushTag(nextVersion); err != nil {
			log.Fatalf(color.RedString("Erro ao empurrar tag: %v"), err)
		}

		// 6. Criar o release no provedor
		log.Printf("Criando release '%s' no GitHub...", nextVersion)
		owner, repoName, err := git.GetCurrentRepo()
		if err != nil {
			log.Fatalf(color.RedString("Não foi possível extrair o dono e nome do repositório da URL remota: %v"), err)
		}

		ctx := context.Background()
		githubProvider := provider.NewGitHubProvider(ctx, token, owner, repoName)

		releaseURL, err := githubProvider.CreateRelease(ctx, nextVersion, changelog)
		if err != nil {
			log.Fatalf(color.RedString("Erro ao criar release no GitHub: %v"), err)
		}

		log.Printf(color.GreenString("✅ Release %s criado com sucesso! Acesse em: %s"), nextVersion, releaseURL)
	},
}

// A função init() já existia, vamos adicionar as strings de ajuda nela
func init() {
	rootCmd.AddCommand(createCmd)

	// --- ADIÇÕES AQUI ---
	// Preenche os campos de ajuda com as cores
	createCmd.Short = color.CyanString("Cria uma nova versão, tag e release.")
	createCmd.Long = color.WhiteString(`Analisa os commits desde a última tag, determina a próxima versão semântica,
cria e empurra a tag, e finalmente cria um release no provedor git (ex: GitHub).`)
	createCmd.Example = color.YellowString(`
  # Executa o comando passando o token do GitHub via flag
  go-release-manager create --token "seu_token_aqui"

  # Executa em modo "dry run" (simulação) para ver o que será feito
  go-release-manager create --token "seu_token_aqui" --dry-run
`)
	// --- FIM DAS ADIÇÕES ---

	// As flags originais continuam aqui
	createCmd.Flags().StringVarP(&token, "token", "t", "", "Token de Acesso Pessoal (PAT) do GitHub (obrigatório)")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simula o processo sem criar tags ou releases")
	createCmd.MarkFlagRequired("token")
}
