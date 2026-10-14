package semver

import (
	"strings"
	"testing"

	"github.com/fabiodrneles/go-release-manager/internal/config"
)

func TestIncrementString(t *testing.T) {
	for inc, want := range map[Increment]string{IncrementNone: "None", IncrementPatch: "Patch", IncrementMinor: "Minor", IncrementMajor: "Major"} {
		if got := inc.String(); got != want {
			t.Errorf("%d: %q, quer %q", inc, got, want)
		}
	}
}

func TestParseCommit(t *testing.T) {
	h, b, f := parseCommit("feat(cli): add next\n\nThe body.\n\nMore body.\n\nRefs: #1\n\nBREAKING CHANGE: flags renamed\n")
	if h != "feat(cli): add next" || b != "The body.\n\nMore body." || f != "Refs: #1\n\nBREAKING CHANGE: flags renamed" {
		t.Errorf("header %q, body %q, footers %q", h, b, f)
	}
	if h, b, f := parseCommit("fix: one line"); h != "fix: one line" || b != "" || f != "" {
		t.Errorf("só o cabeçalho: %q %q %q", h, b, f)
	}
}

func TestStringToIncrement(t *testing.T) {
	for s, want := range map[string]Increment{"MAJOR": IncrementMajor, "minor": IncrementMinor, "Patch": IncrementPatch, "none": IncrementNone, "x": IncrementNone} {
		if got := stringToIncrement(s); got != want {
			t.Errorf("%q: %v, quer %v", s, got, want)
		}
	}
}

func TestAnalyze(t *testing.T) {
	cfg := config.Default()
	cases := []struct {
		commits []string
		want    Increment
	}{
		{nil, IncrementNone},
		{[]string{"docs: readme", "not conventional"}, IncrementNone},
		{[]string{"fix: a", "docs: b"}, IncrementPatch},
		{[]string{"fix: a", "feat(x): b"}, IncrementMinor},
		{[]string{"feat!: a"}, IncrementMajor},
		{[]string{"fix: a\n\nbody\n\nBREAKING CHANGE: gone"}, IncrementMajor},
		{[]string{"fix: a\n\nBREAKING-CHANGE: gone"}, IncrementMajor},
		{[]string{"unknown: a"}, IncrementNone},
	}
	for _, c := range cases {
		if got := Analyze(cfg, c.commits); got != c.want {
			t.Errorf("%q: %v, quer %v", c.commits, got, c.want)
		}
	}
}

func TestLatestStable(t *testing.T) {
	if got := LatestStable([]string{"v1.0.0", "latest", "v2.0.0-beta.1", "v1.2.0", "1.1.0"}); got != "v1.2.0" {
		t.Errorf("maior estável: %q", got)
	}
	if got := LatestStable([]string{"latest", "v1.0.0-rc.1"}); got != "" {
		t.Errorf("sem estável: %q", got)
	}
}

// 001 AC-1, AC-2, AC-3, AC-8 at the unit level.
func TestDetermineNextVersion(t *testing.T) {
	cfg := config.Default()
	tags := []string{"v1.0.0", "v2.0.0-beta.1", "v2.0.0-beta.3", "v2.0.0-rc.1", "v2.0.0-beta.x"}
	cases := []struct {
		base    string
		commits []string
		channel string
		want    string
		inc     Increment
	}{
		{"v1.0.0", []string{"feat!: x"}, "", "v2.0.0", IncrementMajor},
		{"v1.0.0", []string{"feat!: x"}, "beta", "v2.0.0-beta.4", IncrementMajor},
		{"v1.0.0", []string{"feat!: x"}, "alpha", "v2.0.0-alpha.1", IncrementMajor},
		{"v0.3.0", []string{"feat!: x"}, "", "v0.4.0", IncrementMinor},
		{"", []string{"feat: x"}, "", "v0.1.0", IncrementMinor},
		{"v1.0.0", []string{"fix: x"}, "", "v1.0.1", IncrementPatch},
		{"v1.0.0", []string{"docs: x"}, "", "v1.0.0", IncrementNone},
	}
	for _, c := range cases {
		got, inc, err := DetermineNextVersion(cfg, c.base, c.commits, c.channel, tags)
		if err != nil || got != c.want || inc != c.inc {
			t.Errorf("%s %q %q: %s %v %v, quer %s %v", c.base, c.commits, c.channel, got, inc, err, c.want, c.inc)
		}
	}
	if _, _, err := DetermineNextVersion(cfg, "latest", nil, "", nil); err == nil || !strings.Contains(err.Error(), "latest") {
		t.Errorf("base que não é SemVer: %v", err)
	}
}

// 004 AC-1 at the unit level.
func TestValidateForced(t *testing.T) {
	tags := []string{"v1.0.0", "v1.1.0"}
	for v, want := range map[string]string{
		"v2.0.0": "",
		"2.0.0":  "não é uma versão",
		"vx":     "não é uma versão",
		"v1.1.0": "já existe",
		"v0.9.0": "precisa ser maior",
	} {
		err := ValidateForced(v, "v1.1.0", tags)
		if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Errorf("%s: %v, quer %q", v, err, want)
		}
	}
	if err := ValidateForced("v0.1.0", "", nil); err != nil {
		t.Errorf("sem base: %v", err)
	}
}
