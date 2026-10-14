package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var nextChannel, nextRef string

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Print only the next version (nothing when there is no release).",
	Long: `Compute the next version without creating a tag and print only that version on
stdout, for scripts and GitHub Actions. When no change calls for a release, print
nothing and exit with 0. Diagnostic messages go to stderr.`,
	Example: `  VERSION=$(go-release-manager next)
  go-release-manager next -p rc`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := plan(planOptions{channel: nextChannel, ref: nextRef})
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
	nextCmd.Flags().StringVarP(&nextChannel, "pre-release", "p", "", "Pre-release channel (e.g. beta, rc)")
	nextCmd.Flags().StringVar(&nextRef, "ref", "", "Commit or branch to analyze (default: HEAD)")
}
