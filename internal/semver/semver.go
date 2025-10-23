package semver

import (
	"fmt"
	"log"
	"regexp" // Importa o pacote de expressões regulares
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

// Regex para analisar o cabeçalho de um conventional commit.
// Ex: feat(parser)!: add new rule
// Grupo 1: tipo (feat)
// Grupo 2: escopo (parser) - opcional
// Grupo 3: '!' (breaking change) - opcional
// Grupo 4: mensagem (add new rule)
var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!?): (.*)$`)

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

	// 2. Determinar o nível de incremento e agrupar para o changelog
	highestIncrement := IncrementNone

	// Mapas para agrupar as entradas do changelog
	changelogGroups := map[string][]string{
		"breaking": {},
		"feat":     {},
		"fix":      {},
		"perf":     {},
		"refactor": {},
		"docs":     {},
		"test":     {},
		"build":    {},
		"ci":       {},
	}

	log.Printf("Iniciando análise de %d commits...", len(commits)) // Log de início

	for _, commit := range commits {
		cleanCommit := strings.TrimSpace(commit)
		if cleanCommit == "" {
			continue
		}

		// Log com a primeira linha do commit
		firstLine := strings.SplitN(cleanCommit, "\n", 2)[0]
		log.Printf("Analisando commit: [%.70s]", firstLine)

		// Verifica por BREAKING CHANGE no corpo
		isBodyBreaking := strings.Contains(cleanCommit, "\nBREAKING CHANGE:") || strings.Contains(cleanCommit, "\nBREAKING-CHANGE:")

		// Analisa a primeira linha com regex
		matches := commitRegex.FindStringSubmatch(firstLine)
		if matches == nil {
			log.Printf("Commit não convencional, ignorando: [%.70s]", firstLine)
			continue
		}

		commitType := matches[1]
		scope := matches[2] // Pode ser ""
		isHeaderBreaking := matches[3] == "!"
		message := matches[4]

		isBreaking := isBodyBreaking || isHeaderBreaking

		// Lógica de incremento
		if isBreaking {
			if highestIncrement < IncrementMajor {
				highestIncrement = IncrementMajor
			}
		} else if commitType == "feat" {
			if highestIncrement < IncrementMinor {
				highestIncrement = IncrementMinor
			}
		} else if commitType == "fix" {
			if highestIncrement < IncrementPatch {
				highestIncrement = IncrementPatch
			}
		}
		// Nota: refactor, perf, docs, etc., não causam incremento (None)

		// Adiciona ao changelog
		changelogEntry := message
		if scope != "" {
			changelogEntry = fmt.Sprintf("**%s**: %s", scope, message)
		}

		if isBreaking {
			changelogGroups["breaking"] = append(changelogGroups["breaking"], changelogEntry)
		} else if list, ok := changelogGroups[commitType]; ok {
			// Adiciona ao grupo correspondente (feat, fix, refactor, etc.)
			changelogGroups[commitType] = append(list, changelogEntry)
		}
	}

	log.Printf("Análise concluída. Maior incremento: %s", highestIncrement)

	// 3. Construir o changelog
	changelog := buildChangelog(changelogGroups)

	// 4. Se nenhum incremento for encontrado, retorne a tag atual
	// Mas com o novo changelog (que pode conter refactors, docs, etc.)
	if highestIncrement == IncrementNone {
		return latestTag, IncrementNone, changelog
	}

	// 5. Calcular a nova versão
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

	return nextVersion, highestIncrement, changelog
}

// buildChangelog constrói a string final do changelog a partir dos grupos
func buildChangelog(groups map[string][]string) string {
	var b strings.Builder
	b.WriteString("## Changelog\n")
	hasEntries := false

	// Helper para adicionar seções ao changelog
	appendGroup := func(title string, entries []string) {
		if len(entries) > 0 {
			b.WriteString(fmt.Sprintf("\n### %s\n\n", title))
			for _, entry := range entries {
				b.WriteString(fmt.Sprintf("- %s\n", entry))
			}
			hasEntries = true
		}
	}

	// A ordem aqui define a ordem no changelog
	appendGroup("💥 BREAKING CHANGES", groups["breaking"])
	appendGroup("✨ Features", groups["feat"])
	appendGroup("🐛 Bug Fixes", groups["fix"])
	appendGroup("⚡ Performance Improvements", groups["perf"])
	appendGroup("🔧 Code Refactoring", groups["refactor"])
	appendGroup("📚 Documentation", groups["docs"])
	appendGroup("🧪 Tests", groups["test"])
	appendGroup("🏗️ Build System", groups["build"])
	appendGroup("🤖 Continuous Integration", groups["ci"])

	if !hasEntries {
		b.WriteString("\nNenhuma mudança significativa para o changelog.\n")
	}

	return b.String()
}
