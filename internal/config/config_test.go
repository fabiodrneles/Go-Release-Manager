package config

import (
	"os"
	"strings"
	"testing"
)

func rule(c *Config, typ string) string {
	for _, r := range c.ReleaseRules {
		if r.Type == typ {
			return r.Release
		}
	}
	return ""
}

func TestDefault(t *testing.T) {
	c := Default()
	if rule(c, "feat") != "minor" || rule(c, "fix") != "patch" || rule(c, "docs") != "none" {
		t.Errorf("regras padrão: %+v", c.ReleaseRules)
	}
}

// 001 AC-4, AC-5 at the unit level.
func TestParse(t *testing.T) {
	c, err := Parse([]byte("releaseRules:\n  - type: docs\n    release: Patch\n  - type: deps\n    release: minor\n"))
	if err != nil {
		t.Fatal(err)
	}
	if rule(c, "docs") != "patch" || rule(c, "deps") != "minor" || rule(c, "feat") != "minor" {
		t.Errorf("mescla sobre o padrão: %+v", c.ReleaseRules)
	}
	for data, want := range map[string]string{
		"releaseRules: [":                                  "inválido",
		"releaseRules:\n  - release: patch\n":              "sem \"type\"",
		"releaseRules:\n  - type: feat\n    release: minr": "\"minr\"",
	} {
		if _, err := Parse([]byte(data)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, quer %q", data, err, want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	c, err := LoadConfig()
	if err != nil || rule(c, "feat") != "minor" {
		t.Fatalf("sem arquivo, o padrão: %v %+v", err, c)
	}
	if err := os.WriteFile(FileName, []byte("releaseRules:\n  - type: docs\n    release: patch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c, err = LoadConfig(); err != nil || rule(c, "docs") != "patch" {
		t.Fatalf("com arquivo: %v %+v", err, c)
	}
	_ = os.Remove(FileName)
	if err := os.Mkdir(FileName, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil {
		t.Error("arquivo ilegível deveria dar erro")
	}
}
