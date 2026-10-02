package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// 003 AC-2: tags with a suffix (v1.2.0-beta.1) are published as pre-releases.
func TestGoreleaserMarksPrereleases(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Release struct {
			Prerelease string `yaml:"prerelease"`
		} `yaml:"release"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Release.Prerelease != "auto" {
		t.Errorf("release.prerelease = %q, quero \"auto\"", cfg.Release.Prerelease)
	}
}
