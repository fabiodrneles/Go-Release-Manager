package git

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// inRepo makes a repository in a temporary directory and moves the test into it.
func inRepo(t *testing.T) string {
	t.Helper()
	for _, k := range []string{"GIT_AUTHOR", "GIT_COMMITTER"} {
		t.Setenv(k+"_NAME", "t")
		t.Setenv(k+"_EMAIL", "t@t")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, "init", "-q")
	return dir
}

func mustGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func commit(t *testing.T, msg string) {
	t.Helper()
	mustGit(t, "commit", "-q", "--allow-empty", "-m", msg)
}

func TestTagsAndCommits(t *testing.T) {
	inRepo(t)
	if c, err := GetCommitsSince("", "HEAD"); err == nil {
		t.Errorf("sem commits, HEAD não existe: %v", c)
	}
	commit(t, "feat: one")
	mustGit(t, "tag", "v1.0.0")
	commit(t, "fix: two\n\nbody")
	commit(t, "docs: three")
	mustGit(t, "checkout", "-q", "-b", "side", "HEAD~2")
	commit(t, "chore: side")
	mustGit(t, "tag", "v9.0.0")
	mustGit(t, "checkout", "-q", "-")

	if tags, err := MergedTags("HEAD"); err != nil || strings.Join(tags, ",") != "v1.0.0" {
		t.Errorf("tags alcançáveis: %v %v", tags, err)
	}
	if tags, err := AllTags(); err != nil || len(tags) != 2 {
		t.Errorf("todas as tags: %v %v", tags, err)
	}
	c, err := GetCommitsSince("v1.0.0", "HEAD")
	if err != nil || len(c) != 2 || !strings.HasPrefix(c[0], "docs: three") || !strings.Contains(c[1], "body") {
		t.Errorf("commits desde a tag: %q %v", c, err)
	}
	if c, err := GetCommitsSince("", "HEAD"); err != nil || len(c) != 3 {
		t.Errorf("todos os commits: %q %v", c, err)
	}
	sha, err := ShortSHA("HEAD")
	if err != nil || sha == "" {
		t.Fatalf("sha: %q %v", sha, err)
	}
	if _, err := ShortSHA("nope"); err == nil || !strings.Contains(err.Error(), "rev-parse") {
		t.Errorf("ref inexistente: %v", err)
	}
	if err := CreateTag("v1.1.0", sha); err != nil {
		t.Fatal(err)
	}
	if err := CreateTag("v1.1.0", sha); err == nil {
		t.Error("tag repetida deveria falhar")
	}
	if _, err := MergedTags("nope"); err == nil {
		t.Error("ref inexistente em MergedTags")
	}
}

func TestPushTag(t *testing.T) {
	inRepo(t)
	if err := PushTag("v1.0.0"); err == nil {
		t.Error("sem origin, o push falha")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	mustGit(t, "init", "-q", "--bare", remote)
	mustGit(t, "remote", "add", "origin", remote)
	commit(t, "feat: one")
	mustGit(t, "tag", "v1.0.0")
	if err := PushTag("v1.0.0"); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("git", "--git-dir", remote, "tag").Output()
	if err != nil || strings.TrimSpace(string(out)) != "v1.0.0" {
		t.Errorf("a tag chega ao remoto: %q %v", out, err)
	}
}

func TestGetCurrentRepo(t *testing.T) {
	inRepo(t)
	if _, _, err := GetCurrentRepo(); err == nil {
		t.Error("sem origin deveria falhar")
	}
	for url, want := range map[string]string{
		"https://github.com/acme/tool.git": "acme/tool",
		"git@github.com:acme/tool.git":     "acme/tool",
		"https://github.com/acme/tool":     "acme/tool",
		"plain":                            "",
	} {
		mustGit(t, "config", "remote.origin.url", url)
		owner, repo, err := GetCurrentRepo()
		if want == "" {
			if err == nil {
				t.Errorf("%s: deveria falhar, deu %s/%s", url, owner, repo)
			}
			continue
		}
		if err != nil || owner+"/"+repo != want {
			t.Errorf("%s: %s/%s %v", url, owner, repo, err)
		}
	}
}
