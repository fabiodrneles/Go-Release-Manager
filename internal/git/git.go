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

	// --- ESTA É A MUDANÇA ---
	// Em vez de --pretty=%s, usamos --pretty=format:%B%x00
	// %B = Corpo inteiro do commit
	// %x00 = O "NUL byte", um delimitador seguro que não existe em mensagens de commit
	out, err := runCommand("git", "log", commitRange, "--pretty=format:%B%x00")
	// --- FIM DA MUDANÇA ---

	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}

	// Agora dividimos pelo NUL byte, e não por \n
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
