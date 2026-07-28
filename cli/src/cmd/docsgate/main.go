// Command docsgate fails the build when the shipped CLI surface is undocumented.
//
// It reads the command tree as JSON on stdin, produced by the hidden
// `azd app metadata` command, and compares it against cli/docs. Reading the
// binary's own metadata rather than parsing Go source means the gate cannot
// disagree with what users install.
//
// Usage:
//
//	azd app metadata | docsgate --repo-root ..
//	azd app metadata | docsgate --repo-root .. --changed-files changed.txt
//	azd app metadata | docsgate --repo-root .. --changed-files changed.txt --pr-body body.txt
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/docsgate"
)

func main() {
	code, err := run(os.Args[1:], os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

// run returns the process exit code separately from an error so a gate failure,
// which Report already explains on the output stream, does not also print a
// redundant line on stderr and interleave with the report.
//
// Taking the arguments and streams as parameters keeps the orchestration
// testable without a subprocess.
func run(args []string, stdin io.Reader, stdout io.Writer) (int, error) {
	fs := flag.NewFlagSet("docsgate", flag.ContinueOnError)
	fs.SetOutput(stdout)
	var (
		repoRoot     = fs.String("repo-root", ".", "path to the repository root")
		changedPath  = fs.String("changed-files", "", "path to a file listing changed paths, one per line; enables change rules")
		skipReason   = fs.String("skip-reason", "", "recorded reason the change rules do not apply")
		prBodyPath   = fs.String("pr-body", "", "path to a file holding the pull request body; the skip reason is read from it")
		metadataPath = fs.String("metadata", "", "path to CLI metadata JSON; defaults to stdin")
	)
	if err := fs.Parse(args); err != nil {
		return 1, err
	}

	metadata, err := readMetadata(*metadataPath, stdin)
	if err != nil {
		return 1, err
	}

	changed, err := readChangedFiles(*changedPath)
	if err != nil {
		return 1, err
	}

	reason, err := resolveSkipReason(*skipReason, *prBodyPath)
	if err != nil {
		return 1, err
	}

	result, err := docsgate.Run(docsgate.Config{
		RepoRoot:     *repoRoot,
		Metadata:     metadata,
		ChangedFiles: changed,
		SkipReason:   reason,
	})
	if err != nil {
		return 1, err
	}

	result.Report(stdout)
	if result.Failed() {
		return 1, nil
	}
	return 0, nil
}

// resolveSkipReason prefers an explicit reason and otherwise reads the marker
// out of a pull request body.
//
// The body is passed as a file so CI never has to interpolate author controlled
// text into a shell command, and parsing stays in one tested place rather than
// being reimplemented in workflow YAML.
func resolveSkipReason(explicit, bodyPath string) (string, error) {
	if explicit != "" || bodyPath == "" {
		return explicit, nil
	}

	body, err := os.ReadFile(bodyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read pull request body from %s: %w", bodyPath, err)
	}
	return docsgate.ParseSkipReason(string(body)), nil
}

// readMetadata loads the command tree JSON from a file or from the given stream.
func readMetadata(path string, stdin io.Reader) ([]byte, error) {
	if path == "" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read CLI metadata from stdin: %w", err)
		}
		return data, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read CLI metadata from %s: %w", path, err)
	}
	return data, nil
}

// readChangedFiles loads the pull request's file list.
//
// The list comes from a file rather than repeated flags so a large pull request
// cannot exceed the platform's argument limit.
func readChangedFiles(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read changed file list from %s: %w", path, err)
	}

	var files []string
	for _, line := range strings.Split(string(raw), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files, nil
}
