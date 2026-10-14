package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGh puts a gh script that runs body first on PATH, alone.
func fakeGh(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if body != "" {
		if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("GITHUB_TOKEN", "")
}

func TestGetTokenFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_env")
	if tok, err := GetToken(); err != nil || tok != "ghp_env" {
		t.Errorf("%q %v", tok, err)
	}
}

func TestGetTokenFromGh(t *testing.T) {
	fakeGh(t, `echo "  ghp_cli  "`)
	if tok, err := GetToken(); err != nil || tok != "ghp_cli" {
		t.Errorf("%q %v", tok, err)
	}
}

func TestGetTokenErrors(t *testing.T) {
	for body, want := range map[string]string{
		"":                                 "não encontrado",
		`echo "not logged in" >&2; exit 1`: "not logged in",
		`exit 0`:                           "não retornou",
	} {
		fakeGh(t, body)
		if _, err := GetToken(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, quer %q", body, err, want)
		}
	}
}
