package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var binary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "grm-bin")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "go-release-manager")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		panic(string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// repo creates a git repository with the given steps: "commit:<message>" or
// "tag:<name>".
func repo(t *testing.T, steps ...string) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	for _, s := range steps {
		kind, arg, _ := strings.Cut(s, ":")
		switch kind {
		case "commit":
			git("commit", "-q", "--allow-empty", "-m", arg)
		case "tag":
			git("tag", arg)
		default:
			t.Fatalf("passo inválido: %s", s)
		}
	}
	return dir
}

func run(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN=test-token", "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

var proposed = regexp.MustCompile(`A nova tag a ser criada seria: (\S+)`)

func nextVersion(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, code := run(t, dir, append([]string{"create", "-d"}, args...)...)
	if code != 0 {
		t.Fatalf("código %d\n%s", code, out)
	}
	m := proposed.FindStringSubmatch(out)
	if m == nil {
		return ""
	}
	return m[1]
}

func TestNextVersion(t *testing.T) {
	cases := []struct {
		name  string
		steps []string
		args  []string
		want  string
	}{
		// 001 AC-1
		{"estável depois de pré-release", []string{"commit:feat: a", "tag:v1.0.0", "commit:feat!: b", "tag:v2.0.0-beta.1", "commit:feat: c"}, nil, "v2.0.0"},
		// 001 AC-2
		{"pré-release continua o contador", []string{"commit:feat: a", "tag:v1.0.0", "commit:feat!: b", "tag:v2.0.0-beta.1", "commit:feat: c"}, []string{"-p", "beta"}, "v2.0.0-beta.2"},
		{"primeira pré-release do canal", []string{"commit:feat: a", "tag:v1.0.0", "commit:fix: b"}, []string{"-p", "rc"}, "v1.0.1-rc.1"},
		// 001 AC-3
		{"incompatível em 0.x sobe o minor", []string{"commit:feat: a", "tag:v0.3.0", "commit:feat!: b"}, nil, "v0.4.0"},
		{"incompatível pelo rodapé", []string{"commit:feat: a", "tag:v1.2.3", "commit:fix: b\n\nBREAKING CHANGE: x"}, nil, "v2.0.0"},
		// 001 AC-6
		{"tag não SemVer é ignorada", []string{"commit:feat: a", "tag:v1.0.0", "commit:fix: b", "tag:latest"}, nil, "v1.0.1"},
		// 001 AC-8
		{"repositório sem tags", []string{"commit:feat: a"}, nil, "v0.1.0"},
		{"fix", []string{"commit:feat: a", "tag:v1.0.0", "commit:fix: b"}, nil, "v1.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := repo(t, c.steps...)
			if got := nextVersion(t, dir, c.args...); got != c.want {
				t.Errorf("versão proposta = %q, quero %q", got, c.want)
			}
		})
	}
}

// 001 AC-4
func TestPartialConfigKeepsDefaults(t *testing.T) {
	dir := repo(t, "commit:feat: a", "tag:v1.0.0", "commit:feat: b")
	cfg := "releaseRules:\n  - type: docs\n    release: patch\n"
	if err := os.WriteFile(filepath.Join(dir, ".go-releaserc.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nextVersion(t, dir); got != "v1.1.0" {
		t.Errorf("versão proposta = %q, quero v1.1.0", got)
	}
}

// 001 AC-5
func TestInvalidReleaseValue(t *testing.T) {
	dir := repo(t, "commit:feat: a", "tag:v1.0.0", "commit:feat: b")
	cfg := "releaseRules:\n  - type: feat\n    release: minr\n"
	if err := os.WriteFile(filepath.Join(dir, ".go-releaserc.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, dir, "create", "-d")
	if code == 0 {
		t.Fatalf("esperava erro, saiu com 0\n%s", out)
	}
	if !strings.Contains(out, `"feat"`) || !strings.Contains(out, `"minr"`) {
		t.Errorf("a mensagem deve citar o tipo e o valor inválidos:\n%s", out)
	}
}

// 001 AC-7
func TestCommitCount(t *testing.T) {
	dir := repo(t, "commit:feat: a", "tag:v1.0.0", "commit:fix: b")
	out, _ := run(t, dir, "create", "-d")
	if !strings.Contains(out, "Commits analisados: 1\n") {
		t.Errorf("esperava 'Commits analisados: 1':\n%s", out)
	}
}

// 002 AC-1
func TestVersionFlag(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "grm")
	if out, err := exec.Command("go", "build", "-ldflags", "-X main.version=1.2.3", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "1.2.3\n" {
		t.Errorf("--version = %q, quero %q", out, "1.2.3\n")
	}
	if out, _ := run(t, t.TempDir(), "--version"); out != "dev\n" {
		t.Errorf("build local: --version = %q, quero %q", out, "dev\n")
	}
}

// 002 AC-2
func TestDryRunWithoutCredentials(t *testing.T) {
	dir := repo(t, "commit:feat: a", "tag:v1.0.0", "commit:fix: b")
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "create", "-d")
	cmd.Dir = dir
	// Only git on PATH (no gh) and no GITHUB_TOKEN.
	cmd.Env = []string{"PATH=" + filepath.Dir(gitPath), "HOME=" + t.TempDir(), "NO_COLOR=1"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create -d sem credenciais falhou: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "A nova tag a ser criada seria: v1.0.1") {
		t.Errorf("versão proposta ausente:\n%s", out)
	}
}
