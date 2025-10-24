package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort" // <-- NOVO PACOTE IMPORTADO
	"strings"

	"github.com/Masterminds/semver/v3" // <-- NOVO PACOTE IMPORTADO (precisará de 'go mod tidy')
)

// runCommand é uma função helper para executar comandos no shell
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("erro ao executar comando '%s %s': %s", name, strings.Join(args, " "), stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetLatestTag retorna a tag git mais recente
func GetLatestTag() (string, error) {
	tag, err := runCommand("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		if strings.Contains(err.Error(), "no tags found") || strings.Contains(err.Error(), "cannot describe") {
			return "v0.0.0", nil
		}
		return "", err
	}
	return tag, nil
}

// GetCommitsSince retorna uma lista de mensagens de commit desde uma tag específica
func GetCommitsSince(tag string) ([]string, error) {
	commitRange := fmt.Sprintf("%s..HEAD", tag)
	if tag == "v0.0.0" {
		commitRange = "HEAD"
	}

	// %B = Corpo inteiro do commit
	// %x00 = O "NUL byte", um delimitador seguro
	out, err := runCommand("git", "log", commitRange, "--pretty=format:%B%x00")

	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}

	// Agora dividimos pelo NUL byte
	return strings.Split(out, "\x00"), nil
}

// CreateTag cria uma nova tag git
func CreateTag(tag string) error {
	_, err := runCommand("git", "tag", tag)
	return err
}

// PushTag empurra uma tag para o repositório remoto (origin)
func PushTag(tag string) error {
	_, err := runCommand("git", "push", "origin", tag)
	return err
}

// GetCurrentRepo extrai o "dono/nome_repo" da URL remota
func GetCurrentRepo() (owner, repo string, err error) {
	remoteURL, err := runCommand("git", "config", "--get", "remote.origin.url")
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(remoteURL, "git@") {
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
		remoteURL = strings.Replace(remoteURL, "git@", "https://", 1)
	}
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	parts := strings.Split(remoteURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("URL remota inválida: %s", remoteURL)
	}
	owner = parts[len(parts)-2]
	repo = parts[len(parts)-1]
	return owner, repo, nil
}

// --- NOVO ---
// GetLatestPreReleaseTag encontra a tag de pre-release mais recente para uma
// versão estável e um canal específico.
// Ex: baseVersion = "v1.3.0", channel = "beta"
// Ele procura por "v1.3.0-beta.1", "v1.3.0-beta.2", etc., e retorna a mais alta.
func GetLatestPreReleaseTag(baseVersion string, channel string) (string, error) {
	pattern := fmt.Sprintf("%s-%s.*", baseVersion, channel)
	out, err := runCommand("git", "tag", "--list", pattern, "--sort=v:refname")
	if err != nil {
		return "", err // Erro ao executar o 'git tag'
	}

	if out == "" {
		return "", nil // Nenhuma tag encontrada, não é um erro
	}

	tags := strings.Split(out, "\n")
	if len(tags) == 0 {
		return "", nil // Nenhuma tag encontrada
	}

	// Para garantir a ordenação correta, usamos uma biblioteca de semver
	// (A ordenação do Git é boa, mas lexical. 'v1.0.0-beta.10' < 'v1.0.0-beta.2')
	// Vamos usar uma biblioteca de semver para garantir.
	vs := make([]*semver.Version, 0)
	for _, r := range tags {
		v, err := semver.NewVersion(r)
		if err == nil {
			vs = append(vs, v)
		}
	}

	if len(vs) == 0 {
		return "", nil // Nenhuma tag válida encontrada
	}

	// Ordena as versões
	sort.Sort(semver.Collection(vs))

	// Retorna a última (mais alta)
	return vs[len(vs)-1].String(), nil
}
