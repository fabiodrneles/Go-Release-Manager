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

// 003 AC-1: every pull request runs `make ci`.
func TestCIRunsMakeCIOnPullRequests(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	if _, ok := wf.On["pull_request"]; !ok {
		t.Error("ci.yml não roda em pull_request")
	}
	found := false
	for _, job := range wf.Jobs {
		for _, s := range job.Steps {
			if s.Run == "make ci" {
				found = true
			}
		}
	}
	if !found {
		t.Error("nenhum job do ci.yml roda `make ci`")
	}
}
