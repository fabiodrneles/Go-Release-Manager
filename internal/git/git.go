package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// runCommand runs a command and returns its trimmed stdout.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("erro ao executar comando '%s %s': %s", name, strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func lines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// MergedTags returns the tags reachable from HEAD.
func MergedTags() ([]string, error) {
	out, err := runCommand("git", "tag", "--merged", "HEAD")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// AllTags returns every tag in the repository.
func AllTags() ([]string, error) {
	out, err := runCommand("git", "tag", "--list")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// GetCommitsSince returns the full message of each commit after tag, or of
// every commit when tag is empty.
func GetCommitsSince(tag string) ([]string, error) {
	commitRange := "HEAD"
	if tag != "" {
		commitRange = tag + "..HEAD"
	}
	// %B is the raw message; NUL separates the commits.
	out, err := runCommand("git", "log", commitRange, "--pretty=format:%B%x00")
	if err != nil {
		return nil, err
	}
	var commits []string
	for _, c := range strings.Split(out, "\x00") {
		if strings.TrimSpace(c) != "" {
			commits = append(commits, c)
		}
	}
	return commits, nil
}

// CreateTag creates a lightweight tag at HEAD.
func CreateTag(tag string) error {
	_, err := runCommand("git", "tag", tag)
	return err
}

// PushTag pushes a tag to origin.
func PushTag(tag string) error {
	_, err := runCommand("git", "push", "origin", tag)
	return err
}

// GetCurrentRepo extracts "owner" and "repo" from the origin URL.
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
	return parts[len(parts)-2], parts[len(parts)-1], nil
}
