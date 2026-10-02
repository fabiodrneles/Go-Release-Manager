package cmd

import (
	"fmt"
	"log"

	"go-release-manager/internal/config"
	"go-release-manager/internal/git"
	"go-release-manager/internal/semver"

	"github.com/fatih/color"
)

// Plan is the result of analyzing the repository. Its JSON form is the
// machine-readable output of `create --output json` (spec 002 FR-4).
type Plan struct {
	Previous  string `json:"previous"`
	Next      string `json:"next"`
	Increment string `json:"increment"`
	Commits   int    `json:"commits"`
	Channel   string `json:"channel,omitempty"`
	Created   bool   `json:"created"`

	increment semver.Increment
}

// Release reports whether the commits require a new version.
func (p Plan) Release() bool { return p.increment != semver.IncrementNone }

// plan computes the next version for the repository in the current directory.
func plan(channel string) (Plan, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return Plan{}, fmt.Errorf("erro ao carregar configuração %s: %w", config.FileName, err)
	}
	merged, err := git.MergedTags()
	if err != nil {
		return Plan{}, fmt.Errorf("erro ao listar as tags: %w", err)
	}
	allTags, err := git.AllTags()
	if err != nil {
		return Plan{}, fmt.Errorf("erro ao listar as tags: %w", err)
	}
	base := semver.LatestStable(merged)
	log.Printf(color.GreenString("Última versão estável encontrada: %s"), shown(base))

	commits, err := git.GetCommitsSince(base)
	if err != nil {
		return Plan{}, fmt.Errorf("erro ao obter commits: %w", err)
	}
	log.Printf("Analisando %d commits desde a tag %s...", len(commits), shown(base))
	if channel != "" {
		log.Printf(color.CyanString("Modo de pré-release ativado. Canal: %s"), channel)
	}

	next, inc, err := semver.DetermineNextVersion(cfg, base, commits, channel, allTags)
	if err != nil {
		return Plan{}, fmt.Errorf("erro ao determinar a próxima versão: %w", err)
	}
	p := Plan{Previous: base, Commits: len(commits), Channel: channel, Increment: inc.String(), increment: inc}
	if p.Release() {
		p.Next = next
	}
	return p, nil
}

func shown(tag string) string {
	if tag == "" {
		return "(nenhuma)"
	}
	return tag
}
