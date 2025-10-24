package semver

import (
	"fmt"
	"log"
	"regexp"
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
var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!?): (.*)$`)

// --- NOVO ---
// parseCommit disseca o raw commit (obtido com %B do git) em header, body e footers.
// Isso é crucial para analisar "BREAKING CHANGE:" apenas no footer.
func parseCommit(rawCommit string) (header string, body string, footers string) {
	commit := strings.TrimSpace(rawCommit)

	// Divide o header do resto (body + footers) pela primeira linha em branco
	parts := strings.SplitN(commit, "\n\n", 2)
	if len(parts) == 0 {
		return "", "", ""
	}

	// O header é apenas a primeira linha da primeira parte
	header = strings.Split(parts[0], "\n")[0]

	if len(parts) == 1 {
		// Sem body, sem footers
		return header, "", ""
	}

	bodyAndFooters := parts[1]

	// Procura pelo último parágrafo que seja um "footer".
	// Um footer é um bloco de texto separado por linha em branco
	// que começa com um token (ex: "Refs:", "BREAKING CHANGE:").
	paragraphs := regexp.MustCompile("\n\n").Split(bodyAndFooters, -1)
	footerStartIndex := len(paragraphs)

	// Itera de trás para frente nos parágrafos
	for i := len(paragraphs) - 1; i >= 0; i-- {
		p := paragraphs[i]
		// Regex simples para um token de footer
		isFooter, _ := regexp.MatchString(`^([\w-]+): |^(BREAKING CHANGE): |^(BREAKING-CHANGE): `, p)

		if isFooter {
			footerStartIndex = i
		} else {
			// Assim que encontramos um parágrafo que NÃO é um footer, paramos.
			// Tudo antes disso é "body".
			break
		}
	}

	body = strings.Join(paragraphs[:footerStartIndex], "\n\n")
	footers = strings.Join(paragraphs[footerStartIndex:], "\n\n")

	return header, body, footers
}

// --- ATUALIZADO ---
// DetermineNextVersion analisa os commits e retorna APENAS a próxima versão e o incremento.
// A lógica de Changelog foi removida para eliminar duplicidade com o GoReleaser.
func DetermineNextVersion(latestTag string, commits []string) (string, Increment) {
	// 1. Parse da última tag (ex: "v1.2.3")
	major, minor, patch := 0, 0, 0
	if latestTag != "" {
		cleanTag := strings.TrimPrefix(latestTag, "v")

		// Trata tags de pré-release (ex: v1.2.3-beta.1) pegando só a parte principal
		cleanTag = strings.Split(cleanTag, "-")[0]

		parts := strings.Split(cleanTag, ".")
		if len(parts) == 3 {
			major, _ = strconv.Atoi(parts[0])
			minor, _ = strconv.Atoi(parts[1])
			patch, _ = strconv.Atoi(parts[2])
		}
	}

	// 2. Determinar o nível de incremento
	highestIncrement := IncrementNone

	log.Printf("Iniciando análise de %d commits...", len(commits))

	for _, commit := range commits {
		cleanCommit := strings.TrimSpace(commit)
		if cleanCommit == "" {
			continue
		}

		// --- LÓGICA DE PARSE ATUALIZADA ---
		// Usa a nova função para dissecar o commit corretamente
		header, _, footers := parseCommit(cleanCommit)
		log.Printf("Analisando header: [%.70s]", header)

		// Analisa o header com regex
		matches := commitRegex.FindStringSubmatch(header)
		if matches == nil {
			log.Printf("Commit não convencional, ignorando: [%.70s]", header)
			continue
		}

		commitType := matches[1]
		isHeaderBreaking := matches[3] == "!"

		// --- LÓGICA DE BREAKING CHANGE ATUALIZADA ---
		// Verifica APENAS o bloco de footers por "BREAKING CHANGE:"
		isFooterBreaking := false
		if footers != "" {
			// Verifica se alguma linha nos footers começa com o token
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
		// --- FIM DA LÓGICA DE CHANGELOG (REMOVIDA) ---
	}

	log.Printf("Análise concluída. Maior incremento: %s", highestIncrement)

	// 3. Se nenhum incremento for encontrado, retorne a tag atual
	if highestIncrement == IncrementNone {
		// --- ATUALIZADO ---
		return latestTag, IncrementNone
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

	// --- ATUALIZADO ---
	return nextVersion, highestIncrement
}

// --- REMOVIDO ---
// A função buildChangelog foi removida.
// O GoReleaser é agora a única fonte da verdade para a geração do changelog.
