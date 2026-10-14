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

// GetToken returns a GitHub token: GITHUB_TOKEN when set, otherwise the one
// from the GitHub CLI (gh auth token).
func GetToken() (string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		log.Println("Using the token from the GITHUB_TOKEN environment variable.")
		return token, nil
	}

	log.Println("GITHUB_TOKEN is not set. Trying the GitHub CLI (gh auth token)...")

	path, err := exec.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("'gh' CLI not found on PATH")
	}

	cmd := exec.Command(path, "auth", "token")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Most likely not logged in.
		errMsg := fmt.Sprintf("'gh auth token' failed: %s", stderr.String())
		log.Printf("%s", color.RedString(errMsg))
		return "", fmt.Errorf("%s", errMsg)
	}

	token = strings.TrimSpace(stdout.String())
	if token == "" {
		return "", fmt.Errorf("'gh auth token' returned no token")
	}

	log.Println("Using the token from the GitHub CLI.")
	return token, nil
}
