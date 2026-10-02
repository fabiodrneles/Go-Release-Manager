package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var nextChannel string

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Imprime só a próxima versão (nada se não houver release).",
	Long: `Calcula a próxima versão sem criar tag e imprime só ela em stdout, para
uso em scripts e GitHub Actions. Sem mudanças que gerem release, não imprime
nada e sai com 0. As mensagens de diagnóstico vão para stderr.`,
	Example: `  VERSION=$(go-release-manager next)
  go-release-manager next -p rc`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := plan(nextChannel)
		if err != nil {
			return err
		}
		if !p.Release() {
			return nil
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), p.Next)
		return err
	},
}

func init() {
	rootCmd.AddCommand(nextCmd)
	nextCmd.Flags().StringVarP(&nextChannel, "pre-release", "p", "", "Canal de pré-release (ex: beta, rc)")
}
