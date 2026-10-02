package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the optional configuration file read from the repository root.
const FileName = ".go-releaserc.yml"

// Config is the content of .go-releaserc.yml.
type Config struct {
	ReleaseRules []ReleaseRule `yaml:"releaseRules"`
}

// ReleaseRule defines how a commit type affects the version.
type ReleaseRule struct {
	Type    string `yaml:"type"`
	Release string `yaml:"release"` // "major", "minor", "patch", "none"
}

var validReleases = map[string]bool{"major": true, "minor": true, "patch": true, "none": true}

// Default returns the built-in rules: feat → minor, fix → patch, other
// conventional types → none.
func Default() *Config {
	return &Config{
		ReleaseRules: []ReleaseRule{
			{Type: "feat", Release: "minor"},
			{Type: "fix", Release: "patch"},
			{Type: "docs", Release: "none"},
			{Type: "style", Release: "none"},
			{Type: "refactor", Release: "none"},
			{Type: "perf", Release: "none"},
			{Type: "test", Release: "none"},
			{Type: "chore", Release: "none"},
			{Type: "build", Release: "none"},
			{Type: "ci", Release: "none"},
		},
	}
}

// LoadConfig reads .go-releaserc.yml from the current directory. Without the
// file it returns the default rules.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(FileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Nenhum .go-releaserc.yml encontrado. Usando regras padrão (feat/fix).")
			return Default(), nil
		}
		return nil, err
	}
	log.Println("Arquivo .go-releaserc.yml encontrado. Carregando regras personalizadas.")
	return Parse(data)
}

// Parse merges the rules in data over the default rules: a type listed in the
// file overrides the default for that type only (spec 001 FR-4).
func Parse(data []byte) (*Config, error) {
	var file Config
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s inválido: %w", FileName, err)
	}
	cfg := Default()
	for _, rule := range file.ReleaseRules {
		release := strings.ToLower(strings.TrimSpace(rule.Release))
		if rule.Type == "" {
			return nil, fmt.Errorf("%s: regra sem \"type\"", FileName)
		}
		if !validReleases[release] {
			return nil, fmt.Errorf("%s: tipo %q com release %q inválido (use major, minor, patch ou none)", FileName, rule.Type, rule.Release)
		}
		cfg.set(ReleaseRule{Type: rule.Type, Release: release})
	}
	return cfg, nil
}

func (c *Config) set(rule ReleaseRule) {
	for i := range c.ReleaseRules {
		if c.ReleaseRules[i].Type == rule.Type {
			c.ReleaseRules[i] = rule
			return
		}
	}
	c.ReleaseRules = append(c.ReleaseRules, rule)
}
