package provider

import (
	"context"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

type GitHubProvider struct {
	client   *github.Client
	owner    string
	repoName string
}

// NewGitHubProvider cria um novo cliente para a API do GitHub
func NewGitHubProvider(ctx context.Context, token, owner, repoName string) *GitHubProvider {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GitHubProvider{
		client:   client,
		owner:    owner,
		repoName: repoName,
	}
}

// CreateRelease implementa a interface Provider para o GitHub
func (g *GitHubProvider) CreateRelease(ctx context.Context, tag, changelog string) (string, error) {
	release := &github.RepositoryRelease{
		TagName: &tag,
		Name:    &tag,
		Body:    &changelog,
	}
	newRelease, _, err := g.client.Repositories.CreateRelease(ctx, g.owner, g.repoName, release)
	if err != nil {
		return "", err
	}
	return *newRelease.HTMLURL, nil
}
