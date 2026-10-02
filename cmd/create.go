package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/fabiodrneles/go-release-manager/internal/auth"
	"github.com/fabiodrneles/go-release-manager/internal/git"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	dryRun            bool
	preReleaseChannel string
	output            string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: color.CyanString("Cria e empurra uma nova tag semântica."),
	Long: color.WhiteString(`Analisa os commits desde a última tag estável, determina a próxima versão
semântica, cria e empurra a tag. O release do GitHub (com os binários) é criado
pelo GoReleaser na GitHub Action disparada pela tag.`),
	Example: color.YellowString(`
  # Cria e empurra a próxima tag (lê GITHUB_TOKEN ou o token do 'gh')
  go-release-manager create

  # Simula, sem credenciais e sem escrever nada
  go-release-manager create -d

  # Cria uma pré-release
  go-release-manager create -p beta

  # Resultado em JSON para automação
  go-release-manager create -d --output json
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output != "text" && output != "json" {
			return fmt.Errorf("--output inválido: %q (use text ou json)", output)
		}
		p, err := plan(preReleaseChannel)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		if !p.Release() {
			log.Println(color.YellowString("Nenhuma mudança relevante encontrada (feat, fix, BREAKING CHANGE, etc.). Nenhum release será criado."))
			return report(out, p)
		}
		log.Printf(color.GreenString("Tipo de incremento: %s. Nova versão calculada: %s"), p.Increment, p.Next)

		if dryRun {
			return report(out, p)
		}

		// Credenciais só são exigidas quando há push (spec 002 FR-2).
		if _, err := auth.GetToken(); err != nil {
			return fmt.Errorf("token de acesso não fornecido: defina GITHUB_TOKEN ou faça login com `gh auth login` (%w)", err)
		}
		log.Printf("Criando tag git '%s'...", p.Next)
		if err := git.CreateTag(p.Next); err != nil {
			return fmt.Errorf("erro ao criar tag: %w", err)
		}
		log.Printf("Empurrando tag '%s' para o repositório remoto...", p.Next)
		if err := git.PushTag(p.Next); err != nil {
			return fmt.Errorf("erro ao empurrar tag: %w", err)
		}
		p.Created = true
		log.Printf(color.GreenString("✅ Tag %s criada e empurrada com sucesso!"), p.Next)
		log.Println(color.CyanString("A GitHub Action 'Release' foi acionada. Verifique seu repositório em alguns minutos para os binários."))
		return report(out, p)
	},
}

// report writes the result: JSON alone on stdout, or the dry-run summary.
func report(w io.Writer, p Plan) error {
	if output == "json" {
		enc := json.NewEncoder(w)
		return enc.Encode(p)
	}
	if !dryRun || !p.Release() {
		return nil
	}
	_, _ = fmt.Fprintln(w, color.CyanString("\n--- MODO DRY RUN (SIMULAÇÃO) ---"))
	_, _ = fmt.Fprintf(w, "Última tag encontrada: %s\n", shown(p.Previous))
	if p.Channel != "" {
		_, _ = fmt.Fprintf(w, "Canal de pré-release: %s\n", p.Channel)
	}
	_, _ = fmt.Fprintf(w, "Commits analisados: %d\n", p.Commits)
	_, _ = fmt.Fprintf(w, "Decisão de incremento: %s\n", color.MagentaString(p.Increment))
	_, _ = fmt.Fprintf(w, "A nova tag a ser criada seria: %s\n", color.MagentaString(p.Next))
	_, _ = fmt.Fprintln(w, color.CyanString("--- FIM DO DRY RUN ---"))
	return nil
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Simula o processo sem criar tags ou releases")
	createCmd.Flags().StringVarP(&preReleaseChannel, "pre-release", "p", "", "Cria uma pré-release com o canal especificado (ex: beta, rc)")
	createCmd.Flags().StringVarP(&output, "output", "o", "text", "Formato do resultado: text ou json")
}
