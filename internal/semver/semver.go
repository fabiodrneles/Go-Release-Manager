package semver

import (
	"fmt"
	"log"
	"regexp"
	"strconv" // <-- NOVO PACOTE IMPORTADO
	"strings"

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
	// (Esta função está 100% correta, permanece intacta)
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

func DetermineNextVersion(latestTag string, commits []string, preReleaseChannel string) (string, Increment, error) {

	// 1. Parse da última tag (Intacto)
	if latestTag == "v0.0.0" {
		latestTag = "0.0.0"
	}

	v, err := semver.NewVersion(strings.TrimPrefix(latestTag, "v"))
	if err != nil {
		return "", IncrementNone, fmt.Errorf("erro ao analisar a última tag '%s': %v", latestTag, err)
	}

	// 2. Determinar o nível de incremento (Intacto)
	highestIncrement := IncrementNone
	log.Printf("Iniciando análise de %d commits...", len(commits))

	// (Loop de análise de commits ... esta lógica está 100% correta e permanece intacta)
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

	// 5. LÓGICA DE PRÉ-RELEASE (Intacto)
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
		// Nenhuma pré-release encontrada. Esta é a primeira.
		nextVersionString = fmt.Sprintf("%s-%s.1", baseVersionStr, preReleaseChannel)
	} else {
		// --- AQUI ESTÁ A CORREÇÃO ---
		// Encontramos uma tag (ex: "v1.3.0-beta.2"). Vamos incrementá-la manualmente.

		// 1. Parse da tag de pré-release existente
		vPre, err := semver.NewVersion(strings.TrimPrefix(latestPreTagString, "v"))
		if err != nil {
			return "", highestIncrement, fmt.Errorf("erro ao analisar tag de pré-release '%s': %v", latestPreTagString, err)
		}

		// 2. Pega a string da pré-release (ex: "beta.2")
		prStr := vPre.Prerelease()

		// 3. Divide em partes (ex: ["beta", "2"])
		parts := strings.Split(prStr, ".")
		lastPart := parts[len(parts)-1]

		// 4. Tenta incrementar a parte numérica
		num, err := strconv.Atoi(lastPart)
		if err != nil {
			// Não é um número (ex: "beta" em vez de "beta.1"). Adiciona ".1"
			prStr = prStr + ".1"
		} else {
			// É um número. Incrementa.
			num++
			parts[len(parts)-1] = strconv.Itoa(num)
			prStr = strings.Join(parts, ".") // Reconstrói (ex: "beta.3")
		}

		// 5. Usa o método SetPrerelease (que existe) para definir a nova string
		// Note que SetPrerelease retorna uma 'Version' (valor), não um ponteiro.
		vNextPre, err := vPre.SetPrerelease(prStr)
		if err != nil {
			return "", highestIncrement, fmt.Errorf("erro ao definir pré-release '%s': %v", prStr, err)
		}

		nextVersionString = "v" + vNextPre.String() // Adiciona o 'v' de volta
		// --- FIM DA CORREÇÃO ---
	}

	return nextVersionString, highestIncrement, nil
}
