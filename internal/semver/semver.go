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
		// --- ESTA É A CORREÇÃO ---
		// Limpa espaços em branco antes e depois da mensagem
		cleanCommit := strings.TrimSpace(commit)
		// --- FIM DA CORREÇÃO ---

		// Agora, todas as verificações usam 'cleanCommit'
		if strings.Contains(cleanCommit, "BREAKING CHANGE") {
			highestIncrement = IncrementMajor
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

	var changelog string

	// Se nenhum incremento for detectado, retorne a versão antiga e changelog vazio
	if highestIncrement == IncrementNone {
		changelog = "## Changelog\n\nNenhuma mudança detectada."
		// NOTA: Retornar latestTag aqui estava errado se a versão inicial for v0.0.0
		// Se for o primeiro release, ele deve continuar e retornar nextVersion (ex: v0.0.1)
		// Vamos simplificar a lógica de retorno para o final.
		if latestTag == "v0.0.0" && highestIncrement == IncrementNone {
			// Não há commits relevantes no primeiro release
			return latestTag, highestIncrement, changelog
		} else if highestIncrement == IncrementNone {
			// Há commits, mas nenhum é relevante
			return latestTag, highestIncrement, changelog
		}
	}

	changelog = "## Changelog\n\n" + strings.Join(changelogEntries, "\n")

	return nextVersion, highestIncrement, changelog
}
