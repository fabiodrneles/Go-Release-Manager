package prcheck

import (
	"strings"
	"testing"

	"github.com/fabiodrneles/go-release-manager/internal/config"
	"github.com/fabiodrneles/go-release-manager/internal/semver"
)

func codes(r Report) string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Severity[:1]+":"+f.Code)
	}
	return strings.Join(out, " ")
}

// 005 AC-1
func TestCleanTitlePasses(t *testing.T) {
	r := Check(config.Default(), Input{Title: "feat: add flag", Body: "Adds a flag.\n\nCloses #1", Commits: []string{"feat: add flag"}})
	if !r.OK || r.Breaking || len(r.Findings) != 0 || len(r.Sources) != 0 {
		t.Fatalf("%+v", r)
	}
}

// 005 AC-2
func TestTitleMustBeConventional(t *testing.T) {
	for _, title := range []string{"add flag", "wip: add flag", "Feat: add flag", "feat:add flag", ""} {
		r := Check(config.Default(), Input{Title: title})
		if r.OK || codes(r) != "e:title-not-conventional" {
			t.Errorf("%q: %s", title, codes(r))
		}
	}
	r := Check(config.Default(), Input{Title: "wip: x"})
	if !strings.Contains(r.Findings[0].Message, "not known") || !strings.Contains(r.Findings[0].Message, "chore, ci, docs") {
		t.Errorf("an unknown type lists the known ones: %s", r.Findings[0].Message)
	}
	cfg, err := config.Parse([]byte("releaseRules:\n  - type: wip\n    release: none\n"))
	if err != nil {
		t.Fatal(err)
	}
	if r := Check(cfg, Input{Title: "wip: x"}); !r.OK {
		t.Errorf("a type from .go-releaserc.yml is known: %s", codes(r))
	}
}

// 005 AC-3
func TestBreakingInDescriptionNeedsTheBangInTheTitle(t *testing.T) {
	r := Check(config.Default(), Input{Title: "feat: add flag", Body: "Renames the flags.\n\nBREAKING CHANGE: flags renamed"})
	if r.OK || codes(r) != "e:breaking-not-in-title" || r.SuggestedTitle != "feat!: add flag" || !r.Breaking {
		t.Fatalf("%s %+v", codes(r), r)
	}
	if got := strings.Join(r.Sources, ","); got != "PR description" {
		t.Errorf("sources %q", got)
	}
	if !strings.Contains(r.Findings[0].Message, `use "feat!: add flag"`) {
		t.Errorf("the message carries the fix: %s", r.Findings[0].Message)
	}
	scoped := Check(config.Default(), Input{Title: "fix(cli): crash", Body: "BREAKING-CHANGE: gone"})
	if scoped.SuggestedTitle != "fix(cli)!: crash" {
		t.Errorf("the scope is kept: %q", scoped.SuggestedTitle)
	}
}

// 005 AC-4
func TestBreakingInACommit(t *testing.T) {
	for name, commit := range map[string]string{
		"bang":   "fix!: x",
		"footer": "fix: x\n\nbody\n\nBREAKING CHANGE: gone",
	} {
		r := Check(config.Default(), Input{Title: "feat: add flag", Commits: []string{"feat: a", commit}})
		if r.OK || codes(r) != "e:breaking-not-in-title" || strings.Join(r.Sources, ",") != "commit 2" {
			t.Errorf("%s: %s %v", name, codes(r), r.Sources)
		}
	}
}

// 005 AC-5
func TestBangInTitlePasses(t *testing.T) {
	r := Check(config.Default(), Input{Title: "feat(cli)!: rename flags", Commits: []string{"feat(cli)!: rename flags", "test: more"}})
	if !r.OK || !r.Breaking || strings.Join(r.Sources, ",") != "title,commit 1" || len(r.Findings) != 0 {
		t.Fatalf("%+v", r)
	}
}

// 005 AC-6
func TestMisplacedFooterWarns(t *testing.T) {
	r := Check(config.Default(), Input{Title: "feat!: x", Body: "BREAKING CHANGE: flags renamed\n\nMore explanation after it."})
	if !r.OK || codes(r) != "w:footer-misplaced" {
		t.Fatalf("%s", codes(r))
	}
	c := Check(config.Default(), Input{Title: "feat!: x", Commits: []string{"feat: a", "feat: x\n\nBREAKING CHANGE: gone\n\ntrailing prose"}})
	if !c.OK || codes(c) != "w:footer-misplaced" || !strings.Contains(c.Findings[0].Message, "commit 2") {
		t.Fatalf("%s", codes(c))
	}
}

// 005 AC-7
func TestMarkerVariantsWarn(t *testing.T) {
	for _, line := range []string{"BREAKING CHANGES: x", "Breaking change: x", "breaking-change: x", "BREAKING CHANGE x", "> **Breaking changes** below"} {
		r := Check(config.Default(), Input{Title: "feat!: x", Body: "Text.\n\n" + line})
		if !r.OK || codes(r) != "w:breaking-typo" {
			t.Errorf("%q: %s", line, codes(r))
		}
	}
	// Prose that merely mentions the words is not a marker.
	if r := Check(config.Default(), Input{Title: "feat: x", Body: "This is not a breaking change for users."}); len(r.Findings) != 0 {
		t.Errorf("prose: %s", codes(r))
	}
}

// 005 AC-8
func TestSingleCommitSquashWarns(t *testing.T) {
	r := Check(config.Default(), Input{Title: "feat!: add flag", Body: "Why.\n\nBREAKING CHANGE: flags renamed", Commits: []string{"feat: add flag"}})
	if !r.OK || codes(r) != "w:single-commit-squash" {
		t.Fatalf("%s", codes(r))
	}
	two := Check(config.Default(), Input{Title: "feat!: add flag", Body: "BREAKING CHANGE: x", Commits: []string{"feat: a", "fix: b"}})
	if len(two.Findings) != 0 {
		t.Errorf("two commits: %s", codes(two))
	}
	carried := Check(config.Default(), Input{Title: "feat!: add flag", Commits: []string{"feat!: add flag"}})
	if len(carried.Findings) != 0 {
		t.Errorf("the commit carries it: %s", codes(carried))
	}
}

// 005 AC-9
func TestMergedCommitMustCarryTheBreakingChange(t *testing.T) {
	pr := Input{Title: "feat!: add flag", Body: "BREAKING CHANGE: x", Commits: []string{"feat: add flag"}}
	pr.Merged = "feat: add flag (#7)"
	r := Check(config.Default(), pr)
	if r.OK || codes(r) != "e:breaking-lost-in-merge" || !strings.Contains(r.Findings[0].Message, "title, PR description") {
		t.Fatalf("%s %s", codes(r), r.Findings)
	}
	for _, merged := range []string{"feat!: add flag (#7)", "feat: add flag (#7)\n\nWhy.\n\nBREAKING CHANGE: flags renamed"} {
		pr.Merged = merged
		if r := Check(config.Default(), pr); !r.OK || len(r.Findings) != 0 {
			t.Errorf("%q: %s", merged, codes(r))
		}
	}
	// No breaking change announced: nothing to lose, whatever the merged message.
	if r := Check(config.Default(), Input{Title: "feat: x", Merged: "feat: x (#8)"}); !r.OK {
		t.Errorf("%s", codes(r))
	}
	// A merged message that is not Conventional Commits does not count either.
	pr.Merged = "Merge pull request #7"
	if r := Check(config.Default(), pr); r.OK {
		t.Error("a non-conventional merged commit loses the breaking change")
	}
}

// 005 FR-7: the detection is the one the version calculation uses.
func TestIsBreakingAgreesWithAnalyze(t *testing.T) {
	cfg := config.Default()
	for _, c := range []string{
		"feat: x", "feat!: x", "fix: x\n\nBREAKING CHANGE: y", "fix: x\n\nBREAKING-CHANGE: y",
		"fix: x\n\nBREAKING CHANGE: y\n\nafter", "docs: x\n\nBREAKING CHANGE: y", "not conventional\n\nBREAKING CHANGE: y", "chore!: z",
	} {
		want := semver.Analyze(cfg, []string{c}) == semver.IncrementMajor
		if got := semver.IsBreaking(c); got != want {
			t.Errorf("%q: IsBreaking %v, Analyze major %v", c, got, want)
		}
	}
}

func TestTitleWithNewlineAndSpaces(t *testing.T) {
	r := Check(config.Default(), Input{Title: "  feat: x  \n\nignored second line"})
	if r.Title != "feat: x" || !r.OK {
		t.Fatalf("%+v", r)
	}
}
