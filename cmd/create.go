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
	createRef         string
	releaseAs         string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: color.CyanString("Create and push the next semantic version tag."),
	Long: color.WhiteString(`Analyze the commits since the latest stable tag, determine the next semantic
version, then create and push the tag. The GitHub release (with the binaries) is
built by GoReleaser in the GitHub Action the tag triggers.`),
	Example: color.YellowString(`
  # Create and push the next tag (reads GITHUB_TOKEN or the 'gh' token)
  go-release-manager create

  # Simulate, with no credentials and without writing anything
  go-release-manager create -d

  # Create a pre-release
  go-release-manager create -p beta

  # JSON result for automation
  go-release-manager create -d --output json
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output != "text" && output != "json" {
			return fmt.Errorf("invalid --output %q (use text or json)", output)
		}
		p, err := plan(planOptions{channel: preReleaseChannel, ref: createRef, releaseAs: releaseAs})
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		if !p.Release() {
			log.Println(color.YellowString("No relevant change found (feat, fix, BREAKING CHANGE, etc.). No release will be created."))
			return report(out, p)
		}
		if p.Forced {
			log.Printf(color.GreenString("Version given with --release-as: %s"), p.Next)
		} else {
			log.Printf(color.GreenString("Increment: %s. Next version: %s"), p.Increment, p.Next)
		}

		if dryRun {
			return report(out, p)
		}

		// Credentials are required only to push (spec 002 FR-2).
		if _, err := auth.GetToken(); err != nil {
			return fmt.Errorf("no access token: set GITHUB_TOKEN or log in with `gh auth login` (%w)", err)
		}
		log.Printf("Creating git tag '%s'...", p.Next)
		if err := git.CreateTag(p.Next, p.Ref); err != nil {
			return fmt.Errorf("failed to create the tag: %w", err)
		}
		log.Printf("Pushing tag '%s' to the remote...", p.Next)
		if err := git.PushTag(p.Next); err != nil {
			return fmt.Errorf("failed to push the tag: %w", err)
		}
		p.Created = true
		log.Printf(color.GreenString("✅ Tag %s created and pushed."), p.Next)
		log.Println(color.CyanString("The tag push triggers your release workflow. Check your repository in a few minutes for the binaries."))
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
	_, _ = fmt.Fprintln(w, color.CyanString("\n--- DRY RUN ---"))
	_, _ = fmt.Fprintf(w, "Latest tag: %s\n", shown(p.Previous))
	if p.Channel != "" {
		_, _ = fmt.Fprintf(w, "Pre-release channel: %s\n", p.Channel)
	}
	_, _ = fmt.Fprintf(w, "Commits analyzed: %d\n", p.Commits)
	_, _ = fmt.Fprintf(w, "Increment: %s\n", color.MagentaString(p.Increment))
	_, _ = fmt.Fprintf(w, "Tag that would be created: %s\n", color.MagentaString(p.Next))
	_, _ = fmt.Fprintln(w, color.CyanString("--- END OF DRY RUN ---"))
	return nil
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Simulate: create no tag and need no credentials")
	createCmd.Flags().StringVarP(&preReleaseChannel, "pre-release", "p", "", "Create a pre-release on this channel (e.g. beta, rc)")
	createCmd.Flags().StringVarP(&output, "output", "o", "text", "Result format: text or json")
	createCmd.Flags().StringVar(&createRef, "ref", "", "Commit or branch to analyze and tag (default: HEAD)")
	createCmd.Flags().StringVar(&releaseAs, "release-as", "", "Use this version (vX.Y.Z) instead of the computed one")
}
