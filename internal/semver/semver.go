// Package semver computes the next semantic version from Conventional Commits.
package semver

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"go-release-manager/internal/config"

	"github.com/Masterminds/semver/v3"
)

type Increment int

const (
	IncrementNone Increment = iota
	IncrementPatch
	IncrementMinor
	IncrementMajor
)

func (i Increment) String() string {
	return []string{"None", "Patch", "Minor", "Major"}[i]
}

var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!?): (.*)$`)

var footerRegex = regexp.MustCompile(`^([\w-]+): |^(BREAKING CHANGE): |^(BREAKING-CHANGE): `)

var paragraphSep = regexp.MustCompile("\n\n")

func parseCommit(rawCommit string) (header string, body string, footers string) {
	commit := strings.TrimSpace(rawCommit)
	parts := strings.SplitN(commit, "\n\n", 2)
	header = strings.Split(parts[0], "\n")[0]
	if len(parts) == 1 {
		return header, "", ""
	}
	paragraphs := paragraphSep.Split(parts[1], -1)
	footerStartIndex := len(paragraphs)
	for i := len(paragraphs) - 1; i >= 0; i-- {
		if !footerRegex.MatchString(paragraphs[i]) {
			break
		}
		footerStartIndex = i
	}
	body = strings.Join(paragraphs[:footerStartIndex], "\n\n")
	footers = strings.Join(paragraphs[footerStartIndex:], "\n\n")
	return header, body, footers
}

func stringToIncrement(releaseType string) Increment {
	switch strings.ToLower(releaseType) {
	case "major":
		return IncrementMajor
	case "minor":
		return IncrementMinor
	case "patch":
		return IncrementPatch
	default:
		return IncrementNone
	}
}

func mapConfigToIncrements(rules []config.ReleaseRule) map[string]Increment {
	ruleMap := make(map[string]Increment)
	for _, rule := range rules {
		ruleMap[rule.Type] = stringToIncrement(rule.Release)
	}
	return ruleMap
}

// parseTag parses a SemVer tag with an optional "v" prefix. Tags that are not
// SemVer (e.g. "latest") return nil (spec 001 FR-5).
func parseTag(tag string) *semver.Version {
	v, err := semver.StrictNewVersion(strings.TrimPrefix(tag, "v"))
	if err != nil {
		return nil
	}
	return v
}

// LatestStable returns the highest stable SemVer tag among tags, or "" when
// there is none (spec 001 FR-1).
func LatestStable(tags []string) string {
	var best *semver.Version
	bestTag := ""
	for _, t := range tags {
		v := parseTag(t)
		if v == nil || v.Prerelease() != "" {
			continue
		}
		if best == nil || v.GreaterThan(best) {
			best, bestTag = v, t
		}
	}
	return bestTag
}

// Analyze returns the highest increment required by commits.
func Analyze(cfg *config.Config, commits []string) Increment {
	highest := IncrementNone
	rules := mapConfigToIncrements(cfg.ReleaseRules)
	for _, commit := range commits {
		header, _, footers := parseCommit(commit)
		matches := commitRegex.FindStringSubmatch(header)
		if matches == nil {
			log.Printf("Commit não convencional, ignorando: [%.70s]", header)
			continue
		}
		inc := rules[matches[1]]
		if matches[3] == "!" || hasBreakingFooter(footers) {
			inc = IncrementMajor
		}
		if inc > highest {
			highest = inc
		}
	}
	return highest
}

func hasBreakingFooter(footers string) bool {
	for _, line := range strings.Split(footers, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// DetermineNextVersion computes the next version from the latest stable tag
// (base, "" when the repository has none), the commits since it, the
// pre-release channel ("" for a stable release) and every existing tag, used
// to continue a pre-release counter (spec 001 FR-1 to FR-3).
func DetermineNextVersion(cfg *config.Config, base string, commits []string, channel string, allTags []string) (string, Increment, error) {
	current := semver.New(0, 0, 0, "", "")
	if base != "" {
		current = parseTag(base)
		if current == nil {
			return "", IncrementNone, fmt.Errorf("a tag base '%s' não é SemVer", base)
		}
	}

	inc := Analyze(cfg, commits)
	log.Printf("Análise concluída. Maior incremento: %s", inc)
	if inc == IncrementNone {
		return "v" + current.String(), IncrementNone, nil
	}

	// SemVer §4: while in 0.x, an incompatible change bumps the minor.
	if inc == IncrementMajor && current.Major() == 0 {
		inc = IncrementMinor
	}
	var next semver.Version
	switch inc {
	case IncrementMajor:
		next = current.IncMajor()
	case IncrementMinor:
		next = current.IncMinor()
	default:
		next = current.IncPatch()
	}

	if channel == "" {
		return "v" + next.String(), inc, nil
	}
	n := 0
	prefix := channel + "."
	for _, t := range allTags {
		v := parseTag(t)
		if v == nil || v.Major() != next.Major() || v.Minor() != next.Minor() || v.Patch() != next.Patch() {
			continue
		}
		pre := v.Prerelease()
		if !strings.HasPrefix(pre, prefix) {
			continue
		}
		if k, err := strconv.Atoi(strings.TrimPrefix(pre, prefix)); err == nil && k > n {
			n = k
		}
	}
	return fmt.Sprintf("v%s-%s.%d", next.String(), channel, n+1), inc, nil
}
