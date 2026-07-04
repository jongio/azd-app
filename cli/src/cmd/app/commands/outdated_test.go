package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-core/cliout"
)

// forceTextFormat resets the process-wide cliout output format to the default
// (text) for the duration of a test and restores the previous value afterwards.
//
// Sibling tests in this package set the global format to JSON and "reset" it
// with an invalid "text" value, which SetFormat treats as a silent no-op, so
// the format can leak as JSON into these tests. When that happens runOutdated
// sends its report to stdout via cliout.PrintJSON instead of the provided
// writer, leaving the test buffer empty. Resetting here makes these tests
// hermetic regardless of execution order.
func forceTextFormat(t *testing.T) {
	t.Helper()
	original := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(original)) })
	if err := cliout.SetFormat("default"); err != nil {
		t.Fatalf("reset cliout format: %v", err)
	}
}

func TestNewOutdatedCommand(t *testing.T) {
	cmd := NewOutdatedCommand()
	if cmd == nil {
		t.Fatal("NewOutdatedCommand returned nil")
	}
	if cmd.Use != "outdated" {
		t.Fatalf("Use = %q, want outdated", cmd.Use)
	}
	for _, name := range []string{"service", "format", "exit-code"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q not found", name)
		}
	}
}

func TestParseNpmOutdated(t *testing.T) {
	data := []byte(`{
		"left-pad": {"current": "1.0.0", "wanted": "1.3.0", "latest": "1.3.0"},
		"chalk": {"current": "4.0.0", "wanted": "4.1.2", "latest": "5.3.0"}
	}`)
	pkgs, err := parseNpmOutdated(data)
	if err != nil {
		t.Fatalf("parseNpmOutdated: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	// Sorted by name: chalk before left-pad.
	if pkgs[0].Name != "chalk" || pkgs[0].Current != "4.0.0" || pkgs[0].Latest != "5.3.0" {
		t.Errorf("chalk parsed wrong: %+v", pkgs[0])
	}
	if pkgs[1].Name != "left-pad" || pkgs[1].Wanted != "1.3.0" {
		t.Errorf("left-pad parsed wrong: %+v", pkgs[1])
	}
}

func TestParseNpmOutdatedEmpty(t *testing.T) {
	pkgs, err := parseNpmOutdated([]byte("  \n"))
	if err != nil {
		t.Fatalf("parseNpmOutdated empty: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected no packages, got %d", len(pkgs))
	}
	if _, err := parseNpmOutdated([]byte("not json")); err == nil {
		t.Error("expected error on invalid npm json")
	}
}

func TestParseYarnOutdated(t *testing.T) {
	data := []byte(`{"type":"info","data":"Color legend"}
{"type":"table","data":{"head":["Package","Current","Wanted","Latest","Package Type","URL"],"body":[["lodash","4.17.20","4.17.21","4.17.21","dependencies","x"],["react","17.0.1","17.0.2","18.2.0","dependencies","y"]]}}`)
	pkgs, err := parseYarnOutdated(data)
	if err != nil {
		t.Fatalf("parseYarnOutdated: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	if pkgs[0].Name != "lodash" || pkgs[0].Current != "4.17.20" || pkgs[0].Latest != "4.17.21" {
		t.Errorf("lodash parsed wrong: %+v", pkgs[0])
	}
	if pkgs[1].Name != "react" || pkgs[1].Latest != "18.2.0" {
		t.Errorf("react parsed wrong: %+v", pkgs[1])
	}
}

func TestParsePipOutdated(t *testing.T) {
	data := []byte(`[
		{"name": "requests", "version": "2.28.0", "latest_version": "2.31.0", "latest_filetype": "wheel"},
		{"name": "flask", "version": "2.0.0", "latest_version": "3.0.0", "latest_filetype": "wheel"}
	]`)
	pkgs, err := parsePipOutdated(data)
	if err != nil {
		t.Fatalf("parsePipOutdated: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	if pkgs[0].Name != "flask" || pkgs[0].Current != "2.0.0" || pkgs[0].Latest != "3.0.0" {
		t.Errorf("flask parsed wrong: %+v", pkgs[0])
	}
}

func TestParseDotnetOutdated(t *testing.T) {
	data := []byte(`{
		"projects": [{
			"frameworks": [{
				"topLevelPackages": [
					{"id": "Newtonsoft.Json", "requestedVersion": "12.0.0", "resolvedVersion": "12.0.0", "latestVersion": "13.0.3"},
					{"id": "Serilog", "requestedVersion": "2.10.0", "resolvedVersion": "2.10.0", "latestVersion": "3.1.1"}
				]
			}]
		}]
	}`)
	pkgs, err := parseDotnetOutdated(data)
	if err != nil {
		t.Fatalf("parseDotnetOutdated: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	if pkgs[0].Name != "Newtonsoft.Json" || pkgs[0].Latest != "13.0.3" || pkgs[0].Wanted != "12.0.0" {
		t.Errorf("Newtonsoft parsed wrong: %+v", pkgs[0])
	}
}

func TestParseGoOutdated(t *testing.T) {
	data := []byte(`{"Path":"example.com/mod","Main":true,"Version":"v0.0.0"}
{"Path":"github.com/pkg/errors","Version":"v0.9.0","Update":{"Path":"github.com/pkg/errors","Version":"v0.9.1"}}
{"Path":"github.com/stretchr/testify","Version":"v1.8.0"}
{"Path":"golang.org/x/sys","Version":"v0.1.0","Update":{"Path":"golang.org/x/sys","Version":"v0.15.0"}}`)
	pkgs, err := parseGoOutdated(data)
	if err != nil {
		t.Fatalf("parseGoOutdated: %v", err)
	}
	// Main module and the module without an Update are excluded.
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2: %+v", len(pkgs), pkgs)
	}
	if pkgs[0].Name != "github.com/pkg/errors" || pkgs[0].Latest != "v0.9.1" {
		t.Errorf("pkg/errors parsed wrong: %+v", pkgs[0])
	}
	if pkgs[1].Name != "golang.org/x/sys" || pkgs[1].Latest != "v0.15.0" {
		t.Errorf("x/sys parsed wrong: %+v", pkgs[1])
	}
}

func TestParseGoOutdatedInvalid(t *testing.T) {
	if _, err := parseGoOutdated([]byte("{ this is not valid")); err == nil {
		t.Error("expected error on invalid go json")
	}
}

func TestParseOutdatedDispatch(t *testing.T) {
	npm, err := parseOutdated(managerNpm, []byte(`{"a":{"current":"1.0.0","latest":"2.0.0"}}`))
	if err != nil || len(npm) != 1 || npm[0].Name != "a" {
		t.Errorf("npm dispatch failed: %+v err=%v", npm, err)
	}
	pip, err := parseOutdated(managerPip, []byte(`[{"name":"b","version":"1","latest_version":"2"}]`))
	if err != nil || len(pip) != 1 || pip[0].Name != "b" {
		t.Errorf("pip dispatch failed: %+v err=%v", pip, err)
	}
	if _, err := parseOutdated("brew", []byte("{}")); err == nil {
		t.Error("expected error for unsupported manager")
	}
}

func TestOutdatedArgs(t *testing.T) {
	cases := map[string][]string{
		managerNpm:    {"outdated", "--json"},
		managerPnpm:   {"outdated", "--json"},
		managerYarn:   {"outdated", "--json"},
		managerPip:    {"list", "--outdated", "--format=json"},
		managerDotnet: {"list", "package", "--outdated", "--format", "json"},
		managerGo:     {"list", "-u", "-m", "-json", "all"},
	}
	for mgr, want := range cases {
		got := outdatedArgs(mgr)
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("outdatedArgs(%q) = %v, want %v", mgr, got, want)
		}
	}
	if outdatedArgs("brew") != nil {
		t.Error("expected nil args for unsupported manager")
	}
}

func TestNormalizeOutdatedLanguage(t *testing.T) {
	cases := map[string]string{
		"js": "node", "JavaScript": "node", "ts": "node", "typescript": "node",
		"py": "python", "Python": "python",
		"csharp": "dotnet", "dotnet": "dotnet", ".NET": "dotnet",
		"go": "go", "golang": "go",
		"rust": "", "": "",
	}
	for in, want := range cases {
		if got := normalizeOutdatedLanguage(in); got != want {
			t.Errorf("normalizeOutdatedLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInferOutdatedLanguage(t *testing.T) {
	tests := []struct {
		marker string
		want   string
	}{
		{"package.json", "node"},
		{"go.mod", "go"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"app.csproj", "dotnet"},
	}
	for _, tt := range tests {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, tt.marker), "x")
		if got := inferOutdatedLanguage(dir); got != tt.want {
			t.Errorf("marker %q inferred %q, want %q", tt.marker, got, tt.want)
		}
	}
	if got := inferOutdatedLanguage(t.TempDir()); got != "" {
		t.Errorf("empty dir inferred %q, want empty", got)
	}
}

func TestDetectNodeManager(t *testing.T) {
	npmDir := t.TempDir()
	writeFile(t, filepath.Join(npmDir, "package.json"), "{}")
	if got := detectNodeManager(npmDir); got != managerNpm {
		t.Errorf("npm dir manager = %q, want npm", got)
	}

	pnpmDir := t.TempDir()
	writeFile(t, filepath.Join(pnpmDir, "pnpm-lock.yaml"), "")
	if got := detectNodeManager(pnpmDir); got != managerPnpm {
		t.Errorf("pnpm dir manager = %q, want pnpm", got)
	}

	yarnDir := t.TempDir()
	writeFile(t, filepath.Join(yarnDir, "yarn.lock"), "")
	if got := detectNodeManager(yarnDir); got != managerYarn {
		t.Errorf("yarn dir manager = %q, want yarn", got)
	}
}

func TestResolveManager(t *testing.T) {
	dir := t.TempDir()
	lang, mgr, ok := resolveManager(dir, "python")
	if !ok || lang != "Python" || mgr != managerPip {
		t.Errorf("python: got (%q,%q,%v)", lang, mgr, ok)
	}

	// Unsupported declared language is surfaced but marked unsupported.
	lang, mgr, ok = resolveManager(dir, "rust")
	if ok || mgr != "" || lang != "rust" {
		t.Errorf("rust: got (%q,%q,%v), want (rust,\"\",false)", lang, mgr, ok)
	}

	// Fallback to marker files when no language is declared.
	goDir := t.TempDir()
	writeFile(t, filepath.Join(goDir, "go.mod"), "module x")
	lang, mgr, ok = resolveManager(goDir, "")
	if !ok || lang != "Go" || mgr != managerGo {
		t.Errorf("go fallback: got (%q,%q,%v)", lang, mgr, ok)
	}
}

func TestRenderOutdatedText(t *testing.T) {
	result := outdatedResult{
		Services: []serviceOutdated{
			{Service: "web", Language: "Node", Manager: "npm", Packages: []outdatedPackage{
				{Name: "chalk", Current: "4.0.0", Wanted: "4.1.2", Latest: "5.3.0"},
			}},
			{Service: "api", Language: "Python", Manager: "pip"},
			{Service: "cli", Language: "Go", Skipped: true, SkipReason: "go not installed"},
		},
		TotalOutdated: 1,
	}
	var buf bytes.Buffer
	renderOutdatedText(&buf, result)
	out := buf.String()
	for _, want := range []string{"web (Node): 1 outdated", "chalk", "5.3.0", "api (Python): up to date", "cli (Go): skipped - go not installed", "Total: 1 outdated"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderOutdatedTextEmpty(t *testing.T) {
	var buf bytes.Buffer
	renderOutdatedText(&buf, outdatedResult{})
	if !strings.Contains(buf.String(), "No services") {
		t.Errorf("expected empty message, got %q", buf.String())
	}
}

// setupOutdatedProject writes an azure.yaml with a single Node service.
func setupOutdatedProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	azureYaml := "name: outdated-test\nservices:\n  web:\n    host: local\n    language: js\n    project: ./web\n"
	writeFile(t, filepath.Join(root, "azure.yaml"), azureYaml)
	writeFile(t, filepath.Join(root, "web", "package.json"), "{\"name\":\"web\"}")
	return root
}

func TestResolveOutdatedTargetsUnknownService(t *testing.T) {
	root := setupOutdatedProject(t)
	t.Chdir(root)
	if _, err := resolveOutdatedTargets(root, []string{"does-not-exist"}); err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestResolveOutdatedTargets(t *testing.T) {
	root := setupOutdatedProject(t)
	t.Chdir(root)
	targets, err := resolveOutdatedTargets(root, nil)
	if err != nil {
		t.Fatalf("resolveOutdatedTargets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("got %d targets, want 1", len(targets))
	}
	if targets[0].Service != "web" || targets[0].Manager != managerNpm || !targets[0].Supported {
		t.Errorf("unexpected target: %+v", targets[0])
	}
}

func TestRunOutdatedAggregation(t *testing.T) {
	root := setupOutdatedProject(t)
	t.Chdir(root)
	forceTextFormat(t)

	// Stub the environment: npm is "installed" and returns one outdated package.
	origLook, origRun := outdatedLookPath, outdatedRunner
	t.Cleanup(func() { outdatedLookPath, outdatedRunner = origLook, origRun })

	outdatedLookPath = func(string) (string, error) { return "npm", nil }
	outdatedRunner = func(dir, bin string, args []string) ([]byte, error) {
		return []byte(`{"chalk":{"current":"4.0.0","wanted":"5.0.0","latest":"5.3.0"}}`), nil
	}

	var buf bytes.Buffer
	err := runOutdated(&outdatedOptions{writer: &buf})
	if err != nil {
		t.Fatalf("runOutdated: %v", err)
	}
	if !strings.Contains(buf.String(), "chalk") {
		t.Errorf("expected chalk in output:\n%s", buf.String())
	}

	// With --exit-code, the same outdated package makes the run return an error.
	err = runOutdated(&outdatedOptions{writer: &bytes.Buffer{}, exitCode: true})
	if err == nil {
		t.Error("expected non-zero exit with --exit-code and outdated deps")
	}
}

func TestRunOutdatedSkipsMissingTool(t *testing.T) {
	root := setupOutdatedProject(t)
	t.Chdir(root)
	forceTextFormat(t)

	origLook, origRun := outdatedLookPath, outdatedRunner
	t.Cleanup(func() { outdatedLookPath, outdatedRunner = origLook, origRun })

	outdatedLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	called := false
	outdatedRunner = func(dir, bin string, args []string) ([]byte, error) {
		called = true
		return nil, nil
	}

	var buf bytes.Buffer
	if err := runOutdated(&outdatedOptions{writer: &buf}); err != nil {
		t.Fatalf("runOutdated should not fail when a tool is missing: %v", err)
	}
	if called {
		t.Error("runner should not be called when the tool is missing")
	}
	if !strings.Contains(buf.String(), "skipped") {
		t.Errorf("expected skip notice:\n%s", buf.String())
	}
}
