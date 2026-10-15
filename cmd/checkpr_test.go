package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabiodrneles/go-release-manager/internal/prcheck"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// 005 AC-10: the title and description come from the Actions event, the commits from a file.
func TestCheckPREvent(t *testing.T) {
	inRepo(t, "commit:chore: init")
	ev := writeFile(t, "event.json", `{"pull_request":{"title":"feat: add flag","body":"BREAKING CHANGE: removes -x"}}`)
	commits := writeFile(t, "commits.json", `["feat: add flag"]`)
	out, err := execute(t, "check-pr", "--github-event", ev, "--commits-file", commits)
	if err == nil || !strings.Contains(out, "breaking-not-in-title") || !strings.Contains(out, "feat!: add flag") {
		t.Fatalf("expected failure with suggestion: %v\n%s", err, out)
	}
	ev = writeFile(t, "event.json", `{"pull_request":{"title":"feat!: add flag","body":null}}`)
	if out, err = execute(t, "check-pr", "--github-event", ev); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// 005 AC-11: the JSON result has the documented fields.
func TestCheckPRJSON(t *testing.T) {
	inRepo(t, "commit:chore: init")
	out, err := execute(t, "check-pr", "-o", "json", "--title", "feat: x", "--body", "BREAKING CHANGE: y")
	if err == nil {
		t.Fatal("expected exit 1")
	}
	var r prcheck.Report
	if jerr := json.NewDecoder(strings.NewReader(out)).Decode(&r); jerr != nil {
		t.Fatalf("%v\n%s", jerr, out)
	}
	if r.OK || !r.Breaking || r.SuggestedTitle != "feat!: x" || len(r.Findings) == 0 || r.Findings[0].Code != prcheck.CodeNotInTitle {
		t.Errorf("%+v", r)
	}
	if out, err = execute(t, "check-pr", "-o", "json", "--title", "fix: y"); err != nil || !strings.Contains(out, `"ok":true`) {
		t.Errorf("%v\n%s", err, out)
	}
}

// 005 AC-12: usage errors.
func TestCheckPRUsage(t *testing.T) {
	inRepo(t, "commit:chore: init")
	ev := writeFile(t, "event.json", `{"pull_request":{"title":"fix: a"}}`)
	bad := writeFile(t, "bad.json", `{"x":1}`)
	notList := writeFile(t, "c.json", `{"a":1}`)
	cases := map[string][]string{
		"no title":      {"check-pr"},
		"both sources":  {"check-pr", "--github-event", ev, "--title", "fix: a"},
		"not a PR":      {"check-pr", "--github-event", bad},
		"missing event": {"check-pr", "--github-event", "/nonexistent"},
		"bad commits":   {"check-pr", "--title", "fix: a", "--commits-file", notList},
		"bad output":    {"check-pr", "--title", "fix: a", "-o", "xml"},
		"body twice":    {"check-pr", "--title", "fix: a", "--body", "x", "--body-file", ev},
		"not a commit":  {"check-pr", "--title", "fix: a", "--merged", "nope"},
	}
	for name, args := range cases {
		if out, err := execute(t, args...); err == nil {
			t.Errorf("%s: expected an error\n%s", name, out)
		}
	}
}

// 005 AC-9: --merged compares the pull request with the commit that landed.
func TestCheckPRMerged(t *testing.T) {
	inRepo(t, "commit:feat: add flag (#7)")
	out, err := execute(t, "check-pr", "--title", "feat!: add flag", "--merged", "HEAD")
	if err == nil || !strings.Contains(out, "breaking-lost-in-merge") {
		t.Fatalf("expected the lost break: %v\n%s", err, out)
	}
	inRepo(t, "commit:feat!: add flag (#7)")
	if out, err = execute(t, "check-pr", "--title", "feat!: add flag", "--merged", "HEAD"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestCheckPRBodyFileAndAnnotations(t *testing.T) {
	inRepo(t, "commit:chore: init")
	t.Setenv("GITHUB_ACTIONS", "true")
	body := writeFile(t, "pr.md", "text\n\nBREAKING CHANGE: gone\n")
	out, err := execute(t, "check-pr", "--title", "feat: x", "--body-file", body)
	if err == nil || !strings.Contains(out, "::error title=breaking-not-in-title::") {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := escapeAnnotation("a%b\nc"); got != "a%25b%0Ac" {
		t.Errorf("escape: %q", got)
	}
	if plural(1, "error") != "1 error" || plural(2, "error") != "2 errors" {
		t.Error("plural")
	}
}
