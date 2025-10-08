package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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
		// Se não houver tags, o git retorna um erro. Tratamos como "sem tag"
		if strings.Contains(err.Error(), "no tags found") {
			return "v0.0.0", nil // Começamos do zero se não houver tags
		}
		return "", err
	}
	return tag, nil
}

// GetCommitsSince retorna uma lista de mensagens de commit desde uma tag específica
func GetCommitsSince(tag string) ([]string, error) {
	// Se a tag for v0.0.0 (inicial), pegamos todos os commits
	commitRange := fmt.Sprintf("%s..HEAD", tag)
	if tag == "v0.0.0" {
		commitRange = "HEAD"
	}
	out, err := runCommand("git", "log", commitRange, "--pretty=%s")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
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
	// Converte URLs SSH (git@github.com:user/repo.git) para um formato mais fácil de parsear
	if strings.HasPrefix(remoteURL, "git@") {
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
		remoteURL = strings.Replace(remoteURL, "git@", "https://", 1)
	}
	// Remove .git do final, se houver
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	parts := strings.Split(remoteURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("URL remota inválida: %s", remoteURL)
	}
	owner = parts[len(parts)-2]
	repo = parts[len(parts)-1]
	return owner, repo, nil
}
