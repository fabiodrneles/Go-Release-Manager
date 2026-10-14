package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-release-manager",
	Short: "", // set in init
	Long:  "", // set in init
}

func init() {
	// Banner in bold cyan, tagline in yellow, description in blue.
	cTitle := color.New(color.FgCyan, color.Bold)

	cTagline := color.New(color.FgYellow)

	cDesc := color.New(color.FgBlue)

	asciiArt := cTitle.Sprintf(`
	   
   ██████╗ ███████╗██╗     ███████╗ █████╗ ███████╗███████╗
   ██╔══██╗██╔════╝██║     ██╔════╝██╔══██╗██╔════╝██╔════╝
   ██████╔╝█████╗  ██║     █████╗  ███████║███████╗█████╗  
   ██╔══██╗██╔══╝  ██║     ██╔══╝  ██╔══██║╚════██║██╔══╝  
   ██║  ██║███████╗███████╗███████╗██║  ██║███████║███████╗
   ╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝
`)

	tagline := cTagline.Sprint("\n  Automate your versioning and releases with Conventional Commits.")
	description := cDesc.Sprint(`
    This tool reads the latest tag, analyzes the commits since then and
    determines the next version increment (Major, Minor or Patch).`)

	rootCmd.Short = color.CyanString("A CLI that automates semantic versioning and releases.")

	rootCmd.Long = fmt.Sprintf("%s\n%s\n%s", asciiArt, tagline, description)
}

// Execute runs the CLI. version and commit come from the release build.
func Execute(version, commit string) {
	rootCmd.Version = version
	if commit != "" {
		rootCmd.Version = version + " (" + commit + ")"
	}
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	// Errors go to stderr once, with a trailing newline (spec 002 FR-5).
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error: %v", err))
		os.Exit(1)
	}
}
