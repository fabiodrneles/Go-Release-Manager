package auth

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
)

// GetToken busca um token de autenticação do GitHub.
// Prioriza a variável de ambiente GITHUB_TOKEN.
// Se não definida, tenta obter do GitHub CLI (gh auth token).
func GetToken() (string, error) {
	// 1. Prioridade 1: Variável de Ambiente (Padrão de CI)
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		log.Println("Token de autenticação encontrado via variável de ambiente GITHUB_TOKEN.")
		return token, nil
	}

	// 2. Prioridade 2: GitHub CLI (Uso Local)
	log.Println("GITHUB_TOKEN não definido. Tentando obter token do GitHub CLI (gh auth token)...")

	// Verifica se 'gh' está instalado no PATH
	path, err := exec.LookPath("gh")
	if err != nil {
		// 'gh' não está instalado.
		return "", fmt.Errorf("'gh' CLI não encontrado no PATH")
	}

	// 'gh' está instalado, tente obter o token
	cmd := exec.Command(path, "auth", "token")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// O comando falhou (provavelmente o usuário não está logado)
		errMsg := fmt.Sprintf("comando 'gh auth token' falhou: %s", stderr.String())
		log.Printf("%s", color.RedString(errMsg))
		return "", fmt.Errorf("%s", errMsg)
	}

	token = strings.TrimSpace(stdout.String())
	if token == "" {
		return "", fmt.Errorf("comando 'gh auth token' foi executado mas não retornou um token")
	}

	log.Println("Token de autenticação obtido com sucesso via GitHub CLI.")
	return token, nil
}
