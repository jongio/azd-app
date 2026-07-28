//go:build mage

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Environment variables that let CI hand the gate a pull request's context.
// Both are optional; without them the gate runs structural rules only, which is
// the right default for a local run where there is no diff to reason about.
const (
	// envChangedFiles points at a file holding one repository-relative path per
	// line, typically written by `git diff --name-only`. A file is used rather
	// than a variable so a large pull request cannot blow the argument limit.
	envChangedFiles = "DOCS_GATE_CHANGED_FILES"
	// envSkipReason carries the recorded reason the change rules do not apply.
	envSkipReason = "DOCS_GATE_SKIP_REASON"
)

// docsGateDir holds the tool that compares the CLI surface against cli/docs.
// Mage compiles its own files outside the module, so it cannot import an
// internal package directly and drives the tool as a subprocess instead.
const docsGateDir = "src/cmd/docsgate"

// DocsGate fails if the shipped CLI surface is not documented.
//
// It reads the command tree from `azd app metadata`, the same hidden command azd
// itself uses, so the gate can never disagree with the binary users install.
func DocsGate() error {
	fmt.Println("📚 Running docs gate...")

	out, err := docsGateOutput()
	fmt.Print(string(out))
	return err
}

// quietDocsGate runs the gate with captured output for use inside preflight,
// where many steps run at once and interleaved reports are unreadable.
func quietDocsGate() error {
	out, err := docsGateOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimRight(string(out), "\r\n"))
	}
	return nil
}

// docsGateOutput runs the gate and returns its combined report.
func docsGateOutput() ([]byte, error) {
	metadata, err := cliMetadata()
	if err != nil {
		return nil, err
	}

	args := []string{"run", "./" + docsGateDir, "--repo-root", ".."}
	if changed := os.Getenv(envChangedFiles); changed != "" {
		args = append(args, "--changed-files", changed)
	}
	if reason := os.Getenv(envSkipReason); reason != "" {
		args = append(args, "--skip-reason", reason)
	}

	cmd := exec.Command("go", args...)
	cmd.Stdin = bytes.NewReader(metadata)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("docs gate failed: %w", err)
	}
	return out, nil
}

// cliMetadata runs the hidden metadata command and returns its JSON.
func cliMetadata() ([]byte, error) {
	// The CLI embeds the built dashboard, so it cannot compile from a clean
	// checkout. Say so plainly instead of leaving the author to decode a
	// go:embed error.
	dist := filepath.Join("src", "internal", "dashboard", "dist")
	if _, err := os.Stat(dist); os.IsNotExist(err) {
		return nil, fmt.Errorf("dashboard build output missing at %s; run 'mage dashboardBuild' first", dist)
	}

	cmd := exec.Command("go", "run", "./"+srcDir, "metadata")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to read CLI metadata: %w\n%s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
