package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	testrunner "github.com/jongio/azd-app/cli/src/internal/testing"
)

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitInitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Resolve symlinks so paths match gitRepoRoot's EvalSymlinks output (Windows
	// t.TempDir returns an 8.3 short path that git expands to the long form).
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	runTestGit(t, dir, "init")
	runTestGit(t, dir, "config", "user.email", "test@example.com")
	runTestGit(t, dir, "config", "user.name", "Test User")
	runTestGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func writeChangedTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setupChangedRepo(t *testing.T) (string, []testrunner.ServiceInfo) {
	t.Helper()
	repo := gitInitRepo(t)
	writeChangedTestFile(t, filepath.Join(repo, "api", "main.go"), "package main\n")
	writeChangedTestFile(t, filepath.Join(repo, "web", "index.js"), "console.log(1)\n")
	writeChangedTestFile(t, filepath.Join(repo, "README.md"), "# repo\n")
	runTestGit(t, repo, "add", ".")
	runTestGit(t, repo, "commit", "-m", "initial")

	services := []testrunner.ServiceInfo{
		{Name: "api", Dir: filepath.Join(repo, "api")},
		{Name: "web", Dir: filepath.Join(repo, "web")},
	}
	return repo, services
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("service names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("service names = %v, want %v", got, want)
		}
	}
}

func TestChangedServiceFilterNoChanges(t *testing.T) {
	repo, services := setupChangedRepo(t)
	t.Chdir(repo)

	affected, err := changedServiceFilter(services, nil, "HEAD")
	if err != nil {
		t.Fatalf("changedServiceFilter: %v", err)
	}
	if len(affected) != 0 {
		t.Fatalf("expected no affected services, got %v", affected)
	}
}

func TestChangedServiceFilterModifiedFile(t *testing.T) {
	repo, services := setupChangedRepo(t)
	t.Chdir(repo)

	writeChangedTestFile(t, filepath.Join(repo, "api", "main.go"), "package main\n// edit\n")

	affected, err := changedServiceFilter(services, nil, "HEAD")
	if err != nil {
		t.Fatalf("changedServiceFilter: %v", err)
	}
	assertNames(t, affected, []string{"api"})
}

func TestChangedServiceFilterUntrackedFile(t *testing.T) {
	repo, services := setupChangedRepo(t)
	t.Chdir(repo)

	// A brand new, unstaged file under web still counts as a change.
	writeChangedTestFile(t, filepath.Join(repo, "web", "extra.js"), "// new\n")

	affected, err := changedServiceFilter(services, nil, "HEAD")
	if err != nil {
		t.Fatalf("changedServiceFilter: %v", err)
	}
	assertNames(t, affected, []string{"web"})
}

func TestChangedServiceFilterIntersectsWithRequested(t *testing.T) {
	repo, services := setupChangedRepo(t)
	t.Chdir(repo)

	// Both services change, but only web is requested via --service.
	writeChangedTestFile(t, filepath.Join(repo, "api", "main.go"), "package main\n// edit\n")
	writeChangedTestFile(t, filepath.Join(repo, "web", "index.js"), "console.log(2)\n")

	affected, err := changedServiceFilter(services, []string{"web"}, "HEAD")
	if err != nil {
		t.Fatalf("changedServiceFilter: %v", err)
	}
	assertNames(t, affected, []string{"web"})
}

func TestChangedServiceFilterAgainstBaseRef(t *testing.T) {
	repo, services := setupChangedRepo(t)
	t.Chdir(repo)

	// Capture the initial commit, then commit a change to api.
	baseCmd := exec.Command("git", "rev-parse", "HEAD")
	baseCmd.Dir = repo
	baseOut, err := baseCmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	base := string(baseOut)
	base = base[:len(base)-1] // trim trailing newline

	writeChangedTestFile(t, filepath.Join(repo, "api", "main.go"), "package main\n// committed edit\n")
	runTestGit(t, repo, "add", ".")
	runTestGit(t, repo, "commit", "-m", "edit api")

	// Compared against the previous commit, only api changed.
	affected, err := changedServiceFilter(services, nil, base)
	if err != nil {
		t.Fatalf("changedServiceFilter: %v", err)
	}
	assertNames(t, affected, []string{"api"})

	// Compared against HEAD, nothing changed since we committed everything.
	affectedHead, err := changedServiceFilter(services, nil, "HEAD")
	if err != nil {
		t.Fatalf("changedServiceFilter HEAD: %v", err)
	}
	if len(affectedHead) != 0 {
		t.Fatalf("expected no changes vs HEAD, got %v", affectedHead)
	}
}

func TestChangedServiceFilterNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	_, err := changedServiceFilter([]testrunner.ServiceInfo{{Name: "api", Dir: dir}}, nil, "HEAD")
	if err == nil {
		t.Fatal("expected an error outside a git repository")
	}
}

func TestAnyPathUnder(t *testing.T) {
	dir := filepath.Join("repo", "api")
	inside := filepath.Join(dir, "main.go")
	sibling := filepath.Join("repo", "apiv2", "main.go")

	if !anyPathUnder([]string{inside}, dir) {
		t.Fatalf("expected %s to be under %s", inside, dir)
	}
	if !anyPathUnder([]string{dir}, dir) {
		t.Fatalf("expected the directory itself to match")
	}
	// A sibling directory that merely shares a name prefix must not match.
	if anyPathUnder([]string{sibling}, dir) {
		t.Fatalf("did not expect %s to be under %s", sibling, dir)
	}
}
