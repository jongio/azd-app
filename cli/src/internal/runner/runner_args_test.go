package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDotnetRunArgsSolution verifies a .sln is run from its own directory and
// is never passed via --project, which `dotnet run` rejects for solutions.
func TestDotnetRunArgsSolution(t *testing.T) {
	slnPath := filepath.Join(t.TempDir(), "MyApp.sln")

	args, dir, err := dotnetRunArgs(slnPath)
	if err != nil {
		t.Fatalf("dotnetRunArgs returned error: %v", err)
	}

	want := []string{"run"}
	if !equalArgs(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
	if got := filepath.Dir(slnPath); dir != got {
		t.Errorf("dir = %q, want %q (solutions run from their own directory)", dir, got)
	}
	for _, a := range args {
		if a == "--project" {
			t.Error("args contain --project; dotnet run rejects --project for .sln files")
		}
	}
}

// TestDotnetRunArgsProjectFile verifies project files are passed explicitly via
// --project so the command does not depend on the caller's directory layout.
func TestDotnetRunArgsProjectFile(t *testing.T) {
	for _, ext := range []string{".csproj", ".fsproj", ".vbproj"} {
		t.Run(ext, func(t *testing.T) {
			projPath := filepath.Join(t.TempDir(), "MyApp"+ext)

			args, dir, err := dotnetRunArgs(projPath)
			if err != nil {
				t.Fatalf("dotnetRunArgs returned error: %v", err)
			}

			want := []string{"run", "--project", projPath}
			if !equalArgs(args, want) {
				t.Errorf("args = %v, want %v", args, want)
			}

			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("os.Getwd: %v", err)
			}
			if dir != cwd {
				t.Errorf("dir = %q, want the working directory %q", dir, cwd)
			}
		})
	}
}

// TestDotnetRunArgsSolutionExtensionIsCaseSensitive pins current behaviour:
// filepath.Ext comparison is exact, so ".SLN" takes the project-file path.
// This documents the contract rather than asserting it is ideal.
func TestDotnetRunArgsSolutionExtensionIsCaseSensitive(t *testing.T) {
	projPath := filepath.Join(t.TempDir(), "MyApp.SLN")

	args, _, err := dotnetRunArgs(projPath)
	if err != nil {
		t.Fatalf("dotnetRunArgs returned error: %v", err)
	}
	if !containsArg(args, "--project") {
		t.Errorf("args = %v; want the project-file form for an uppercase .SLN extension", args)
	}
}

// TestPythonRunCommandManagedEnvironments verifies uv and poetry delegate to
// their own `run python` subcommand rather than resolving an interpreter path.
func TestPythonRunCommandManagedEnvironments(t *testing.T) {
	tests := []struct {
		packageManager string
		entryPoint     string
	}{
		{"uv", "main.py"},
		{"poetry", "src/app.py"},
	}

	for _, tt := range tests {
		t.Run(tt.packageManager, func(t *testing.T) {
			cmd, args, err := pythonRunCommand(tt.packageManager, t.TempDir(), tt.entryPoint)
			if err != nil {
				t.Fatalf("pythonRunCommand returned error: %v", err)
			}
			if cmd != tt.packageManager {
				t.Errorf("cmd = %q, want %q", cmd, tt.packageManager)
			}
			want := []string{"run", "python", tt.entryPoint}
			if !equalArgs(args, want) {
				t.Errorf("args = %v, want %v", args, want)
			}
		})
	}
}

// TestPythonRunCommandRejectsUnknownManager verifies an unrecognised package
// manager fails loudly instead of silently defaulting to a system interpreter.
func TestPythonRunCommandRejectsUnknownManager(t *testing.T) {
	cmd, args, err := pythonRunCommand("conda", t.TempDir(), "main.py")
	if err == nil {
		t.Fatal("pythonRunCommand accepted an unsupported package manager; want error")
	}
	if !strings.Contains(err.Error(), "unsupported package manager") {
		t.Errorf("error = %q, want it to mention an unsupported package manager", err.Error())
	}
	if cmd != "" || args != nil {
		t.Errorf("cmd = %q, args = %v; want zero values on error", cmd, args)
	}
}

// TestPipInterpreterPrefersDotVenv verifies a project-local .venv wins over the
// system interpreter, so a project's pinned dependencies are actually used.
func TestPipInterpreterPrefersDotVenv(t *testing.T) {
	dir := t.TempDir()
	want := createFakeVenv(t, dir, ".venv")

	if got := pipInterpreter(dir); got != want {
		t.Errorf("pipInterpreter = %q, want %q", got, want)
	}
}

// TestPipInterpreterFallsBackToVenv verifies the legacy "venv" directory is
// used when ".venv" is absent.
func TestPipInterpreterFallsBackToVenv(t *testing.T) {
	dir := t.TempDir()
	want := createFakeVenv(t, dir, "venv")

	if got := pipInterpreter(dir); got != want {
		t.Errorf("pipInterpreter = %q, want %q", got, want)
	}
}

// TestPipInterpreterPrefersDotVenvOverVenv verifies precedence when both
// directories exist; ".venv" is the modern convention and must win.
func TestPipInterpreterPrefersDotVenvOverVenv(t *testing.T) {
	dir := t.TempDir()
	want := createFakeVenv(t, dir, ".venv")
	createFakeVenv(t, dir, "venv")

	if got := pipInterpreter(dir); got != want {
		t.Errorf("pipInterpreter = %q, want %q (.venv must take precedence)", got, want)
	}
}

// TestPipInterpreterFallsBackToSystemPython verifies a project with no
// virtualenv resolves to the bare "python" command found on PATH.
func TestPipInterpreterFallsBackToSystemPython(t *testing.T) {
	if got := pipInterpreter(t.TempDir()); got != "python" {
		t.Errorf("pipInterpreter = %q, want %q", got, "python")
	}
}

// TestPipInterpreterIgnoresVenvDirectoryWithoutInterpreter verifies a venv
// directory that exists but has no interpreter does not shadow the fallback.
func TestPipInterpreterIgnoresVenvDirectoryWithoutInterpreter(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".venv"), 0o750); err != nil {
		t.Fatalf("creating empty .venv: %v", err)
	}

	if got := pipInterpreter(dir); got != "python" {
		t.Errorf("pipInterpreter = %q, want %q; an empty .venv must not shadow the fallback", got, "python")
	}
}

// TestPythonRunCommandPipUsesResolvedInterpreter ties the pieces together: a
// pip project with a .venv must invoke that interpreter with the entry point
// as its only argument.
func TestPythonRunCommandPipUsesResolvedInterpreter(t *testing.T) {
	dir := t.TempDir()
	want := createFakeVenv(t, dir, ".venv")

	cmd, args, err := pythonRunCommand("pip", dir, "main.py")
	if err != nil {
		t.Fatalf("pythonRunCommand returned error: %v", err)
	}
	if cmd != want {
		t.Errorf("cmd = %q, want %q", cmd, want)
	}
	if !equalArgs(args, []string{"main.py"}) {
		t.Errorf("args = %v, want [main.py]", args)
	}
}

// TestVenvPythonPathIsPlatformSpecific verifies the interpreter layout matches
// the host platform: Windows venvs use Scripts/python.exe, others bin/python.
func TestVenvPythonPathIsPlatformSpecific(t *testing.T) {
	got := venvPythonPath(filepath.FromSlash("/proj"), ".venv")

	var want string
	if runtime.GOOS == "windows" {
		want = filepath.Join(filepath.FromSlash("/proj"), ".venv", "Scripts", "python.exe")
	} else {
		want = filepath.Join(filepath.FromSlash("/proj"), ".venv", "bin", "python")
	}

	if got != want {
		t.Errorf("venvPythonPath = %q, want %q", got, want)
	}
}

// createFakeVenv materialises the platform-correct interpreter inside venvDir
// and returns the path pipInterpreter is expected to select.
func createFakeVenv(t *testing.T, projectDir, venvDir string) string {
	t.Helper()
	interpreter := venvPythonPath(projectDir, venvDir)
	if err := os.MkdirAll(filepath.Dir(interpreter), 0o750); err != nil {
		t.Fatalf("creating venv dir: %v", err)
	}
	if err := os.WriteFile(interpreter, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("creating venv interpreter: %v", err)
	}
	return interpreter
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
