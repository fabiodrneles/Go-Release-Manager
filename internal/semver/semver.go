package semver

import (
	"fmt"
	"log" // Mantenha o import de log
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

	log.Printf("Iniciando análise de %d commits...", len(commits)) // Log de início

	for _, commit := range commits {
		cleanCommit := strings.TrimSpace(commit)
		if cleanCommit == "" { // Ignora commits vazios (causados pelo split)
			continue
		}

		log.Printf("Analisando commit: [%.50s]", cleanCommit) // Log de cada commit

		// Agora, todas as verificações usam 'cleanCommit'
		if strings.Contains(cleanCommit, "BREAKING CHANGE") {
			if highestIncrement < IncrementMajor {
				highestIncrement = IncrementMajor
			}
			changelogEntries = append(changelogEntries, fmt.Sprintf("- 💥 %s", cleanCommit))

		} else if strings.HasPrefix(cleanCommit, "feat:") {
			if highestIncrement < IncrementMinor {
				highestIncrement = IncrementMinor
			}
			changelogEntries = append(changelogEntries, fmt.Sprintf("- ✨ %s", cleanCommit))

		} else if strings.HasPrefix(cleanCommit, "fix:") {
			if highestIncrement < IncrementPatch {
				highestIncrement = IncrementPatch
			}
			changelogEntries = append(changelogEntries, fmt.Sprintf("- BUG %s", cleanCommit))

		} else if strings.HasPrefix(cleanCommit, "refactor:") {
			if highestIncrement < IncrementPatch {
				highestIncrement = IncrementPatch
			}
			changelogEntries = append(changelogEntries, fmt.Sprintf("- 🔧 %s", cleanCommit))
		}
	}

	log.Printf("Análise concluída. Maior incremento: %s", highestIncrement)

	// 3. Se nenhum incremento for encontrado, retorne
	if highestIncrement == IncrementNone {
		return latestTag, IncrementNone, "## Changelog\n\nNenhuma mudança detectada."
	}

	// 4. Calcular a nova versão
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
