// Package prcheck validates a pull request's title and its breaking-change
// announcements, so a squash merge never loses them (spec 005).
package prcheck

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/fabiodrneles/go-release-manager/internal/config"
	"github.com/fabiodrneles/go-release-manager/internal/semver"
)

// Severity of a finding: only errors make the check fail.
const (
	Error   = "error"
	Warning = "warning"
)

// Finding codes.
const (
	CodeTitle      = "title-not-conventional"
	CodeNotInTitle = "breaking-not-in-title"
	CodeLost       = "breaking-lost-in-merge"
	CodeMisplaced  = "footer-misplaced"
	CodeTypo       = "breaking-typo"
	CodeSingle     = "single-commit-squash"
)

// Input is what is known about a pull request.
type Input struct {
	Title   string
	Body    string   // the PR description
	Commits []string // the full message of each commit in the PR
	// Merged is the message of the commit that landed on the main branch; set
	// it to check the merge itself instead of the open PR (spec 005 FR-5).
	Merged string
}

// Finding is one problem or doubt found in the pull request.
type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// Report is the result of Check. Its JSON form is the machine-readable
// output of `check-pr --output json` (spec 005 FR-6).
type Report struct {
	OK             bool      `json:"ok"`
	Title          string    `json:"title"`
	Breaking       bool      `json:"breaking"`
	Sources        []string  `json:"sources"`
	SuggestedTitle string    `json:"suggested_title,omitempty"`
	Findings       []Finding `json:"findings"`
}

// Errors counts the findings that fail the check.
func (r Report) Errors() int { return r.count(Error) }

// Warnings counts the findings that only warn.
func (r Report) Warnings() int { return r.count(Warning) }

func (r Report) count(severity string) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == severity {
			n++
		}
	}
	return n
}

func (r *Report) add(severity, code, format string, a ...any) {
	r.Findings = append(r.Findings, Finding{Severity: severity, Code: code, Message: fmt.Sprintf(format, a...)})
}

var (
	// near matches a line that starts like a breaking-change marker, in any
	// case or plural; exact marker lines are told apart by exactMarker.
	near        = regexp.MustCompile(`(?i)^[\s>*_-]*breaking[ -]changes?\b`)
	exactMarker = regexp.MustCompile(`^BREAKING[ -]CHANGE:`)
)

// markerLines returns the lines of text that are exact breaking-change
// markers (the spelling the version calculation accepts).
func markerLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if exactMarker.MatchString(line) {
			out = append(out, line)
		}
	}
	return out
}

// nearMisses returns the lines that look like a marker but are not one.
func nearMisses(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if near.MatchString(line) && !exactMarker.MatchString(line) {
			out = append(out, line)
		}
	}
	return out
}

// footerCounts reports whether the marker lines in text would count as a
// footer if text were the body of a commit: the version calculation only
// reads the last paragraphs. The neutral header keeps a "!" in the real
// header from hiding a misplaced footer.
func footerCounts(text string) bool {
	return semver.IsBreaking("chore: x\n\n" + strings.TrimSpace(text))
}

// afterHeader returns the commit message without its first line.
func afterHeader(commit string) string {
	_, rest, _ := strings.Cut(strings.TrimSpace(commit), "\n")
	return strings.TrimSpace(rest)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}

// Check evaluates the pull request against the rules in cfg.
func Check(cfg *config.Config, in Input) Report {
	title := firstLine(in.Title)
	r := Report{Title: title, Sources: []string{}, Findings: []Finding{}}
	h, conventional := semver.ParseHeader(title)

	if conventional && h.Breaking {
		r.Sources = append(r.Sources, "title")
	}
	if len(markerLines(in.Body)) > 0 {
		r.Sources = append(r.Sources, "PR description")
	}
	for i, c := range in.Commits {
		if semver.IsBreaking(c) || len(markerLines(c)) > 0 {
			r.Sources = append(r.Sources, fmt.Sprintf("commit %d", i+1))
		}
	}
	r.Breaking = len(r.Sources) > 0

	if in.Merged != "" {
		if r.Breaking && !semver.IsBreaking(in.Merged) {
			r.add(Error, CodeLost, "the pull request announced a breaking change (%s), but the merged commit has no \"!\" and no BREAKING CHANGE: footer, so the next version will not be a Major", strings.Join(r.Sources, ", "))
		}
		r.OK = r.Errors() == 0
		return r
	}

	switch {
	case !conventional:
		r.add(Error, CodeTitle, "the title %q is not a Conventional Commit: use type(scope)!: description, for example \"feat: add a flag\"", title)
	case !knownType(cfg, h.Type):
		r.add(Error, CodeTitle, "the title type %q is not known: use one of %s", h.Type, strings.Join(types(cfg), ", "))
	case r.Breaking && !h.Breaking:
		r.SuggestedTitle = withBang(h)
		r.add(Error, CodeNotInTitle, "the pull request announces a breaking change (%s), but the title has no \"!\", and the title is what a squash merge keeps: use %q", strings.Join(r.Sources, ", "), r.SuggestedTitle)
	}

	warnFooters(&r, "the PR description", in.Body)
	for i, c := range in.Commits {
		warnFooters(&r, fmt.Sprintf("commit %d", i+1), afterHeader(c))
	}
	if len(in.Commits) == 1 && r.Breaking && !semver.IsBreaking(in.Commits[0]) {
		r.add(Warning, CodeSingle, "this pull request has a single commit and GitHub's default squash message uses the commit, not the title: the breaking change is lost unless the repository's squash default is \"pull request title and description\", or the commit carries the marker")
	}
	r.OK = r.Errors() == 0
	return r
}

// warnFooters adds the warnings about the markers found in text.
func warnFooters(r *Report, where, text string) {
	if len(markerLines(text)) > 0 && !footerCounts(text) {
		r.add(Warning, CodeMisplaced, "in %s, a BREAKING CHANGE: line is not in the last paragraph, so the version calculation ignores it: move it to the end", where)
	}
	for _, line := range nearMisses(text) {
		r.add(Warning, CodeTypo, "in %s, the line %q looks like a breaking-change marker but is not one: write exactly \"BREAKING CHANGE: description\"", where, truncate(line))
	}
}

func knownType(cfg *config.Config, t string) bool {
	for _, rule := range cfg.ReleaseRules {
		if rule.Type == t {
			return true
		}
	}
	return false
}

func types(cfg *config.Config) []string {
	out := make([]string, 0, len(cfg.ReleaseRules))
	for _, rule := range cfg.ReleaseRules {
		out = append(out, rule.Type)
	}
	sort.Strings(out)
	return out
}

func withBang(h semver.Header) string {
	scope := ""
	if h.Scope != "" {
		scope = "(" + h.Scope + ")"
	}
	return h.Type + scope + "!: " + h.Description
}

func truncate(s string) string {
	if r := []rune(s); len(r) > 50 {
		return string(r[:49]) + "…"
	}
	return s
}
