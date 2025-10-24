package semver

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"go-release-manager/internal/config" // <-- NOVO PACOTE IMPORTADO
	"go-release-manager/internal/git"

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

func parseCommit(rawCommit string) (header string, body string, footers string) {
	// (Esta função permanece 100% intacta)
	commit := strings.TrimSpace(rawCommit)
	parts := strings.SplitN(commit, "\n\n", 2)
	if len(parts) == 0 {
		return "", "", ""
	}
	header = strings.Split(parts[0], "\n")[0]
	if len(parts) == 1 {
		return header, "", ""
	}
	bodyAndFooters := parts[1]
	paragraphs := regexp.MustCompile("\n\n").Split(bodyAndFooters, -1)
	footerStartIndex := len(paragraphs)
	for i := len(paragraphs) - 1; i >= 0; i-- {
		p := paragraphs[i]
		isFooter, _ := regexp.MatchString(`^([\w-]+): |^(BREAKING CHANGE): |^(BREAKING-CHANGE): `, p)
		if isFooter {
			footerStartIndex = i
		} else {
			break
		}
	}
	body = strings.Join(paragraphs[:footerStartIndex], "\n\n")
	footers = strings.Join(paragraphs[footerStartIndex:], "\n\n")
	return header, body, footers
}

// --- NOVA FUNÇÃO AUXILIAR ---
// Converte a string do YAML (ex: "patch") para o tipo Increment
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

// --- NOVA FUNÇÃO AUXILIAR ---
// Converte as regras do config em um mapa para consulta rápida
func mapConfigToIncrements(rules []config.ReleaseRule) map[string]Increment {
	ruleMap := make(map[string]Increment)
	for _, rule := range rules {
		ruleMap[rule.Type] = stringToIncrement(rule.Release)
	}
	return ruleMap
}

// --- ASSINATURA ATUALIZADA ---
// Agora recebe 'cfg *config.Config' como o primeiro parâmetro
func DetermineNextVersion(cfg *config.Config, latestTag string, commits []string, preReleaseChannel string) (string, Increment, error) {

	// 1. Parse da última tag (Intacto)
	if latestTag == "v0.0.0" {
		latestTag = "0.0.0"
	}
	v, err := semver.NewVersion(strings.TrimPrefix(latestTag, "v"))
	if err != nil {
		return "", IncrementNone, fmt.Errorf("erro ao analisar a última tag '%s': %v", latestTag, err)
	}

	// --- 2. LÓGICA DE INCREMENTO ATUALIZADA ---
	highestIncrement := IncrementNone
	// Converte as regras do .yml em um mapa de consulta
	releaseRules := mapConfigToIncrements(cfg.ReleaseRules)

	log.Printf("Iniciando análise de %d commits...", len(commits))
	for _, commit := range commits {
		cleanCommit := strings.TrimSpace(commit)
		if cleanCommit == "" {
			continue
		}
		header, _, footers := parseCommit(cleanCommit)
		log.Printf("Analisando header: [%.70s]", header)

		matches := commitRegex.FindStringSubmatch(header)
		if matches == nil {
			log.Printf("Commit não convencional, ignorando: [%.70s]", header)
			continue
		}

		commitType := matches[1]
		isHeaderBreaking := matches[3] == "!"

		// Lógica de Breaking Change (permanece intacta, 'breaking' sempre vence)
		isFooterBreaking := false
		if footers != "" {
			footerLines := strings.Split(footers, "\n")
			for _, line := range footerLines {
				trimmedLine := strings.TrimSpace(line)
				if strings.HasPrefix(trimmedLine, "BREAKING CHANGE:") || strings.HasPrefix(trimmedLine, "BREAKING-CHANGE:") {
					isFooterBreaking = true
					log.Println("Encontrado 'BREAKING CHANGE' no footer.")
					break
				}
			}
		}
		isBreaking := isHeaderBreaking || isFooterBreaking

		// --- LÓGICA DE INCREMENTO SUBSTITUÍDA ---
		// Em vez de 'if/else' para 'feat' e 'fix', usamos o mapa de regras
		if isBreaking {
			if highestIncrement < IncrementMajor {
				highestIncrement = IncrementMajor
			}
		} else {
			// Consulta o tipo de commit (ex: "docs") no mapa de regras
			inc, ok := releaseRules[commitType]
			if !ok {
				// Se o tipo não estiver no mapa (ex: "security"), não faz nada
				inc = IncrementNone
			}

			// Atualiza o incremento mais alto encontrado
			if inc > highestIncrement {
				highestIncrement = inc
			}
		}
		// --- FIM DA LÓGICA SUBSTITUÍDA ---
	}
	log.Printf("Análise concluída. Maior incremento: %s", highestIncrement)

	// 3. Se nenhum incremento for encontrado (Intacto)
	if highestIncrement == IncrementNone {
		return "v" + v.String(), IncrementNone, nil
	}

	// 4. Calcular a nova versão ESTÁVEL (Intacto)
	var nextStableVersion semver.Version
	switch highestIncrement {
	case IncrementMajor:
		if v.Major() == 0 {
			nextStableVersion = v.IncMajor()
		} else {
			nextStableVersion = v.IncMajor()
		}
	case IncrementMinor:
		nextStableVersion = v.IncMinor()
	case IncrementPatch:
		nextStableVersion = v.IncPatch()
	}

	// 5. LÓGICA DE PRÉ-RELEASE (Intacta, já funciona com a lógica acima)
	if preReleaseChannel == "" {
		return "v" + nextStableVersion.String(), highestIncrement, nil
	}

	baseVersionStr := "v" + nextStableVersion.String()
	latestPreTagString, err := git.GetLatestPreReleaseTag(baseVersionStr, preReleaseChannel)
	if err != nil {
		return "", highestIncrement, fmt.Errorf("erro ao buscar tags de pré-release: %v", err)
	}

	var nextVersionString string
	if latestPreTagString == "" {
		nextVersionString = fmt.Sprintf("%s-%s.1", baseVersionStr, preReleaseChannel)
	} else {
		vPre, err := semver.NewVersion(strings.TrimPrefix(latestPreTagString, "v"))
		if err != nil {
			return "", highestIncrement, fmt.Errorf("erro ao analisar tag de pré-release '%s': %v", latestPreTagString, err)
		}
		prStr := vPre.Prerelease()
		parts := strings.Split(prStr, ".")
		lastPart := parts[len(parts)-1]
		num, err := strconv.Atoi(lastPart)
		if err != nil {
			prStr = prStr + ".1"
		} else {
			num++
			parts[len(parts)-1] = strconv.Itoa(num)
			prStr = strings.Join(parts, ".")
		}
		vNextPre, err := vPre.SetPrerelease(prStr)
		if err != nil {
			return "", highestIncrement, fmt.Errorf("erro ao definir pré-release '%s': %v", prStr, err)
		}
		nextVersionString = "v" + vNextPre.String()
	}

	return nextVersionString, highestIncrement, nil
}
