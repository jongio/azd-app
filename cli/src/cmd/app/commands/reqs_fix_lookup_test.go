package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// plantToolShim writes an executable named command onto a fresh directory and
// puts that directory on PATH for the duration of the test. On Windows the
// shim is a .cmd file, which is the shape that motivated the fix: npm, pnpm,
// az and func all ship as .cmd shims, and a lookup that appends .exe misses
// every one of them.
func plantToolShim(t *testing.T, command string) (dir, path string) {
	t.Helper()

	dir = t.TempDir()
	var body string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, command+".cmd")
		body = "@echo off\r\necho 99.0.0\r\n"
	} else {
		path = filepath.Join(dir, command)
		body = "#!/bin/sh\necho 99.0.0\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write shim: %v", err)
	}

	// Replace PATH entirely so a real installation of the tool cannot satisfy
	// the lookup and hide a regression.
	t.Setenv("PATH", dir)
	if runtime.GOOS == "windows" {
		// LookupTool consults PATHEXT to decide which extensions are runnable.
		// Pinning it keeps the test independent of the host's setting.
		t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
	}

	return dir, path
}

// TestCheckPrerequisiteResolvesShim runs in the default CI job, unlike the
// integration test that also covers this path. It pins the lookup that
// resolves .cmd shims on Windows: reverting to an .exe-only lookup makes this
// fail rather than staying green.
func TestCheckPrerequisiteResolvesShim(t *testing.T) {
	// pnpm is on the custom command allowlist and ships as a .cmd shim on
	// Windows, so it is the honest stand-in for the tools that broke.
	const command = "pnpm"

	_, want := plantToolShim(t, command)

	runner := newReqsFixRunner()
	got := runner.checkPrerequisite(Prerequisite{
		Name:       command,
		Command:    command,
		MinVersion: "1.0.0",
		Args:       []string{"--version"},
	})

	if !got.Found {
		t.Fatalf("shim at %s was not found; result: %+v", want, got)
	}
	if !strings.EqualFold(got.Path, want) {
		t.Errorf("resolved path = %q, want %q", got.Path, want)
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(got.Path), ".cmd") {
		t.Errorf("resolved path %q should be the .cmd shim", got.Path)
	}
}

// TestCheckPrerequisiteReportsMissingTool covers the other side of the same
// branch, so a lookup that returns a path for everything would also fail.
func TestCheckPrerequisiteReportsMissingTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	runner := newReqsFixRunner()
	got := runner.checkPrerequisite(Prerequisite{
		Name:       "pnpm",
		Command:    "pnpm",
		MinVersion: "1.0.0",
		Args:       []string{"--version"},
	})

	if got.Found {
		t.Fatalf("expected the tool to be missing, got %+v", got)
	}
}
