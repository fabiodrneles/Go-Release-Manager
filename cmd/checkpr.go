package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fabiodrneles/go-release-manager/internal/config"
	"github.com/fabiodrneles/go-release-manager/internal/git"
	"github.com/fabiodrneles/go-release-manager/internal/prcheck"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var checkPRFlags struct {
	title, body, bodyFile, commitsFile, event, merged, output string
}

var checkPRCmd = &cobra.Command{
	Use:   "check-pr",
	Short: "Check a pull request's title and breaking changes before a squash merge.",
	Long: `Check that a pull request will not lose a breaking change when it is squash
merged. The title must be a Conventional Commit, and a breaking change announced
anywhere in the pull request (a "!" or a BREAKING CHANGE: line in the description
or in a commit) must also be in the title, because the title is what a squash
merge keeps. Doubtful cases are warnings and do not fail the check.

With --merged REF it checks the commit that landed on the main branch instead,
and fails when it lost a breaking change the pull request announced.

Exit code 0 means no errors; 1 means at least one error. The rules are the ones
the version calculation uses.`,
	Example: `  # In GitHub Actions (the check-pr Action does this for you)
  go-release-manager check-pr --github-event "$GITHUB_EVENT_PATH" --commits-file commits.json

  # By hand
  go-release-manager check-pr --title "feat: add a flag" --body-file pr.md

  # After the merge: did the squash commit keep the breaking change?
  go-release-manager check-pr --github-event event.json --commits-file commits.json --merged HEAD`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := checkPRFlags
		if f.output != "text" && f.output != "json" {
			return fmt.Errorf("invalid --output %q (use text or json)", f.output)
		}
		in, err := readPRInput(cmd.InOrStdin(), f.title, f.body, f.bodyFile, f.commitsFile, f.event)
		if err != nil {
			return err
		}
		if f.merged != "" {
			if in.Merged, err = git.CommitMessage(f.merged); err != nil {
				return fmt.Errorf("--merged %q is not a commit: %w", f.merged, err)
			}
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", config.FileName, err)
		}
		r := prcheck.Check(cfg, in)
		out := cmd.OutOrStdout()
		if f.output == "json" {
			if err := json.NewEncoder(out).Encode(r); err != nil {
				return err
			}
		} else {
			writePRReport(out, r, os.Getenv("GITHUB_ACTIONS") == "true")
		}
		if !r.OK {
			return errors.New("the pull request check failed")
		}
		return nil
	},
}

// readPRInput gathers the pull request data from the flags.
func readPRInput(stdin io.Reader, title, body, bodyFile, commitsFile, event string) (prcheck.Input, error) {
	var in prcheck.Input
	if event != "" {
		if title != "" || body != "" || bodyFile != "" {
			return in, errors.New("use either --github-event or --title/--body/--body-file, not both")
		}
		data, err := os.ReadFile(event)
		if err != nil {
			return in, fmt.Errorf("--github-event: %w", err)
		}
		var ev struct {
			PullRequest *struct {
				Title string  `json:"title"`
				Body  *string `json:"body"`
			} `json:"pull_request"`
		}
		if err := json.Unmarshal(data, &ev); err != nil {
			return in, fmt.Errorf("--github-event %s is not JSON: %w", event, err)
		}
		if ev.PullRequest == nil {
			return in, fmt.Errorf("--github-event %s has no pull_request: run it on a pull_request event", event)
		}
		in.Title = ev.PullRequest.Title
		if ev.PullRequest.Body != nil {
			in.Body = *ev.PullRequest.Body
		}
	} else {
		if title == "" {
			return in, errors.New("the pull request title is required: use --title or --github-event")
		}
		if body != "" && bodyFile != "" {
			return in, errors.New("use either --body or --body-file, not both")
		}
		in.Title, in.Body = title, body
		if bodyFile != "" {
			var data []byte
			var err error
			if bodyFile == "-" {
				data, err = io.ReadAll(stdin)
			} else {
				data, err = os.ReadFile(bodyFile)
			}
			if err != nil {
				return in, fmt.Errorf("--body-file: %w", err)
			}
			in.Body = string(data)
		}
	}
	if commitsFile != "" {
		data, err := os.ReadFile(commitsFile)
		if err != nil {
			return in, fmt.Errorf("--commits-file: %w", err)
		}
		if err := json.Unmarshal(data, &in.Commits); err != nil {
			return in, fmt.Errorf("--commits-file %s must be a JSON array of commit messages: %w", commitsFile, err)
		}
	}
	return in, nil
}

// writePRReport prints the findings for people and, on GitHub Actions, as
// annotations on the pull request.
func writePRReport(w io.Writer, r prcheck.Report, annotate bool) {
	_, _ = fmt.Fprintf(w, "Pull request: %s\n", r.Title)
	if r.Breaking {
		_, _ = fmt.Fprintf(w, "Breaking change announced in: %s\n", strings.Join(r.Sources, ", "))
	}
	for _, f := range r.Findings {
		if f.Severity == prcheck.Error {
			_, _ = fmt.Fprintf(w, "%s [%s] %s\n", color.RedString("✗"), f.Code, f.Message)
		} else {
			_, _ = fmt.Fprintf(w, "%s [%s] %s\n", color.YellowString("!"), f.Code, f.Message)
		}
		if annotate {
			_, _ = fmt.Fprintf(w, "::%s title=%s::%s\n", f.Severity, f.Code, escapeAnnotation(f.Message))
		}
	}
	if len(r.Findings) == 0 {
		_, _ = fmt.Fprintln(w, color.GreenString("✓ no problems found"))
		return
	}
	_, _ = fmt.Fprintf(w, "%s, %s\n", plural(r.Errors(), "error"), plural(r.Warnings(), "warning"))
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// escapeAnnotation encodes the characters GitHub reserves in workflow commands.
func escapeAnnotation(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

func init() {
	rootCmd.AddCommand(checkPRCmd)
	fl := checkPRCmd.Flags()
	fl.StringVar(&checkPRFlags.title, "title", "", "Pull request title")
	fl.StringVar(&checkPRFlags.body, "body", "", "Pull request description")
	fl.StringVar(&checkPRFlags.bodyFile, "body-file", "", `File with the pull request description ("-" for standard input)`)
	fl.StringVar(&checkPRFlags.commitsFile, "commits-file", "", "JSON file with an array of the pull request's commit messages")
	fl.StringVar(&checkPRFlags.event, "github-event", "", "GitHub Actions pull_request event file ($GITHUB_EVENT_PATH): reads the title and the description")
	fl.StringVar(&checkPRFlags.merged, "merged", "", "Commit that landed on the main branch (e.g. HEAD): check that it kept the breaking change")
	fl.StringVarP(&checkPRFlags.output, "output", "o", "text", "Result format: text or json")
}
