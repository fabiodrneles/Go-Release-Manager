package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// inRepo runs the test inside a new repository built from steps: "commit:<msg>" or
// "tag:<name>".
func inRepo(t *testing.T, steps ...string) string {
	t.Helper()
	for _, k := range []string{"GIT_AUTHOR", "GIT_COMMITTER"} {
		t.Setenv(k+"_NAME", "t")
		t.Setenv(k+"_EMAIL", "t@t")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	gitRun(t, "init", "-q")
	for _, s := range steps {
		kind, arg, _ := strings.Cut(s, ":")
		if kind == "tag" {
			gitRun(t, "tag", arg)
		} else {
			gitRun(t, "commit", "-q", "--allow-empty", "-m", arg)
		}
	}
	return dir
}

func gitRun(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// execute runs the CLI in this process, with the flags back at their defaults.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	dryRun, preReleaseChannel, output, createRef, releaseAs = false, "", "text", "", ""
	nextChannel, nextRef = "", ""
	checkPRFlags.title, checkPRFlags.body, checkPRFlags.bodyFile, checkPRFlags.commitsFile = "", "", "", ""
	checkPRFlags.event, checkPRFlags.merged, checkPRFlags.output = "", "", "text"
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	err := rootCmd.Execute()
	return out.String(), err
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestPlan(t *testing.T) {
	inRepo(t, "commit:feat: one", "tag:v1.0.0", "commit:fix: two")
	p, err := plan(planOptions{})
	if err != nil || p.Next != "v1.0.1" || p.Previous != "v1.0.0" || p.Commits != 1 || !p.Release() {
		t.Fatalf("%+v %v", p, err)
	}
	if p, err = plan(planOptions{releaseAs: "v3.0.0"}); err != nil || !p.Forced || p.Next != "v3.0.0" {
		t.Errorf("forçada: %+v %v", p, err)
	}
	for o, want := range map[planOptions]string{
		{releaseAs: "v3.0.0", channel: "rc"}: "not both",
		{ref: "nope"}:                        "is not a commit",
		{releaseAs: "v0.1.0"}:                "must be higher",
	} {
		if _, err := plan(o); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: %v, quer %q", o, err, want)
		}
	}
	if err := os.WriteFile(".go-releaserc.yml", []byte("releaseRules: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := plan(planOptions{}); err == nil || !strings.Contains(err.Error(), "failed to load") {
		t.Errorf("configuração inválida: %v", err)
	}
}

func TestPlanBadBaseTag(t *testing.T) {
	inRepo(t, "commit:feat: one")
	if _, err := plan(planOptions{}); err != nil {
		t.Fatalf("sem tags: %v", err)
	}
}

func TestNext(t *testing.T) {
	inRepo(t, "commit:feat: one", "tag:v1.0.0", "commit:feat: two")
	if out, err := execute(t, "next"); err != nil || strings.TrimSpace(out) != "v1.1.0" {
		t.Errorf("%q %v", out, err)
	}
	if out, err := execute(t, "next", "-p", "rc"); err != nil || strings.TrimSpace(out) != "v1.1.0-rc.1" {
		t.Errorf("pré-release: %q %v", out, err)
	}
	if _, err := execute(t, "next", "--ref", "nope"); err == nil {
		t.Error("ref inexistente")
	}
	inRepo(t, "commit:feat: one", "tag:v1.0.0", "commit:docs: two")
	if out, err := execute(t, "next"); err != nil || out != "" {
		t.Errorf("sem release, nada: %q %v", out, err)
	}
}

func TestCreateDryRun(t *testing.T) {
	inRepo(t, "commit:feat: one", "tag:v1.0.0", "commit:feat: two")
	out, err := execute(t, "create", "-d")
	if err != nil || !strings.Contains(out, "DRY RUN") || !strings.Contains(out, "v1.1.0") || !strings.Contains(out, "Commits analyzed: 1") {
		t.Errorf("%q %v", out, err)
	}
	if out, _ = execute(t, "create", "-d", "-p", "beta"); !strings.Contains(out, "Pre-release channel: beta") {
		t.Errorf("canal: %q", out)
	}
	out, err = execute(t, "create", "-d", "-o", "json")
	var p Plan
	if err != nil || json.Unmarshal([]byte(out), &p) != nil || p.Next != "v1.1.0" || p.Created {
		t.Errorf("json: %q %v", out, err)
	}
	if out, err = execute(t, "create", "-d", "--release-as", "v2.0.0", "-o", "json"); err != nil || !strings.Contains(out, `"forced":true`) {
		t.Errorf("forçada: %q %v", out, err)
	}
	if _, err = execute(t, "create", "-o", "yaml"); err == nil || !strings.Contains(err.Error(), "--output") {
		t.Errorf("saída inválida: %v", err)
	}
	if _, err = execute(t, "create", "--ref", "nope"); err == nil {
		t.Error("ref inexistente")
	}
}

func TestCreateNothingToRelease(t *testing.T) {
	inRepo(t, "commit:feat: one", "tag:v1.0.0", "commit:docs: two")
	if out, err := execute(t, "create", "-d"); err != nil || strings.Contains(out, "DRY RUN") {
		t.Errorf("%q %v", out, err)
	}
	if out, err := execute(t, "create", "-o", "json"); err != nil || !strings.Contains(out, `"next":""`) {
		t.Errorf("json sem release: %q %v", out, err)
	}
}

func TestCreatePushesTag(t *testing.T) {
	inRepo(t, "commit:feat: one", "tag:v1.0.0", "commit:fix: two")
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitRun(t, "init", "-q", "--bare", remote)
	gitRun(t, "remote", "add", "origin", remote)

	// No token and no gh: nothing is pushed (002 FR-2). Only git stays on PATH.
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", bin)
	if _, err := execute(t, "create"); err == nil || !strings.Contains(err.Error(), "token") {
		t.Errorf("sem token, sem push: %v", err)
	}
	if tags := gitRun(t, "tag"); tags != "v1.0.0" {
		t.Errorf("sem token, nenhuma tag nova: %q", tags)
	}

	t.Setenv("GITHUB_TOKEN", "ghp_test")
	out, err := execute(t, "create", "-o", "json")
	if err != nil || !strings.Contains(out, `"created":true`) {
		t.Fatalf("%q %v", out, err)
	}
	if tags := gitRun(t, "--git-dir", remote, "tag"); tags != "v1.0.1" {
		t.Errorf("tag no remoto: %q", tags)
	}
	if _, err := execute(t, "create", "--release-as", "v1.0.1"); err == nil {
		t.Error("tag repetida")
	}
	gitRun(t, "commit", "-q", "--allow-empty", "-m", "fix: three")
	gitRun(t, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	if _, err := execute(t, "create"); err == nil || !strings.Contains(err.Error(), "failed to push") {
		t.Errorf("push falho: %v", err)
	}
}

func TestExecuteVersion(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	Execute("v1.2.3", "abc1234")
	if strings.TrimSpace(out.String()) != "v1.2.3 (abc1234)" {
		t.Errorf("%q", out.String())
	}
}
