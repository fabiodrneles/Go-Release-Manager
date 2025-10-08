// internal/semver/semver.go
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

type Increment int

const (
	IncrementNone Increment = iota
	IncrementPatch
	IncrementMinor
	IncrementMajor
)

// String para facilitar a exibição do tipo de incremento
func (i Increment) String() string {
	return []string{"None", "Patch", "Minor", "Major"}[i]
}

// DetermineNextVersion analisa os commits e retorna a próxima versão e o changelog
func DetermineNextVersion(latestTag string, commits []string) (string, Increment, string) {
	// 1. Parse da última tag (ex: "v1.2.3")
	major, minor, patch := 0, 0, 0
	if latestTag != "" {
		cleanTag := strings.TrimPrefix(latestTag, "v")
		parts := strings.Split(cleanTag, ".")
		if len(parts) == 3 {
			major, _ = strconv.Atoi(parts[0])
			minor, _ = strconv.Atoi(parts[1])
			patch, _ = strconv.Atoi(parts[2])
		}
	}

	// 2. Determinar o nível de incremento baseado nos commits
	highestIncrement := IncrementNone
	var changelogEntries []string
	for _, commit := range commits {
		if strings.HasPrefix(commit, "feat:") {
			if highestIncrement < IncrementMinor {
				highestIncrement = IncrementMinor
			}
			changelogEntries = append(changelogEntries, fmt.Sprintf("- ✨ %s", commit))
		} else if strings.HasPrefix(commit, "fix:") {
			if highestIncrement < IncrementPatch {
				highestIncrement = IncrementPatch
			}
			changelogEntries = append(changelogEntries, fmt.Sprintf("- 🐛 %s", commit))
		} else if strings.Contains(commit, "BREAKING CHANGE") {
			highestIncrement = IncrementMajor
			changelogEntries = append(changelogEntries, fmt.Sprintf("- 💥 %s", commit))
		}
	}

	// 3. Calcular a nova versão
	switch highestIncrement {
	case IncrementMajor:
		major++
		minor = 0
		patch = 0
	case IncrementMinor:
		minor++
		patch = 0
	case IncrementPatch:
		patch++
	}

	nextVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	changelog := "## Changelog\n\n" + strings.Join(changelogEntries, "\n")

	return nextVersion, highestIncrement, changelog
}
