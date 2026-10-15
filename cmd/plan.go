package cmd

import (
	"fmt"
	"log"

	"github.com/fabiodrneles/go-release-manager/internal/config"
	"github.com/fabiodrneles/go-release-manager/internal/git"
	"github.com/fabiodrneles/go-release-manager/internal/semver"

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
	Ref       string `json:"ref"`
	Forced    bool   `json:"forced"`
	Created   bool   `json:"created"`
	// Reasons are the commits that set the increment (only with --explain).
	Reasons []semver.Reason `json:"reasons,omitempty"`

	increment semver.Increment
}

// Release reports whether a new version will be created.
func (p Plan) Release() bool { return p.Forced || p.increment != semver.IncrementNone }

// planOptions selects the pre-release channel, the commit to analyze and tag
// (spec 004 FR-2) and an optional forced version (spec 004 FR-1).
type planOptions struct {
	channel   string
	ref       string
	releaseAs string
	explain   bool
}

// plan computes the next version for the repository in the current directory.
func plan(o planOptions) (Plan, error) {
	channel := o.channel
	ref := o.ref
	if ref == "" {
		ref = "HEAD"
	}
	if o.releaseAs != "" && channel != "" {
		return Plan{}, fmt.Errorf("use --release-as or --pre-release, not both")
	}
	sha, err := git.ShortSHA(ref)
	if err != nil {
		return Plan{}, fmt.Errorf("--ref %q is not a commit: %w", ref, err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return Plan{}, fmt.Errorf("failed to load %s: %w", config.FileName, err)
	}
	merged, err := git.MergedTags(ref)
	if err != nil {
		return Plan{}, fmt.Errorf("failed to list the tags: %w", err)
	}
	allTags, err := git.AllTags()
	if err != nil {
		return Plan{}, fmt.Errorf("failed to list the tags: %w", err)
	}
	base := semver.LatestStable(merged)
	log.Printf(color.GreenString("Latest stable version: %s"), shown(base))

	commits, err := git.GetCommitsSince(base, ref)
	if err != nil {
		return Plan{}, fmt.Errorf("failed to read the commits: %w", err)
	}
	log.Printf("Analyzing %d commit(s) since tag %s...", len(commits), shown(base))
	if channel != "" {
		log.Printf(color.CyanString("Pre-release channel: %s"), channel)
	}

	next, inc, err := semver.DetermineNextVersion(cfg, base, commits, channel, allTags)
	if err != nil {
		return Plan{}, fmt.Errorf("failed to compute the next version: %w", err)
	}
	p := Plan{Previous: base, Commits: len(commits), Channel: channel, Ref: sha, Increment: inc.String(), increment: inc}
	if o.explain {
		p.Reasons = semver.Explain(cfg, commits)
	}
	if o.releaseAs != "" {
		if err := semver.ValidateForced(o.releaseAs, base, allTags); err != nil {
			return Plan{}, err
		}
		p.Forced = true
		p.Next = o.releaseAs
		return p, nil
	}
	if p.Release() {
		p.Next = next
	}
	return p, nil
}

func shown(tag string) string {
	if tag == "" {
		return "(none)"
	}
	return tag
}
