package docsgate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Repository-relative locations of the documentation the gate checks. They are
// written with forward slashes because they appear in messages and in the
// changed-file lists git produces, both of which use that separator regardless
// of host platform.
const (
	referencePath = "cli/docs/cli-reference.md"
	commandsDir   = "cli/docs/commands"
)

// Finding is one documentation gap.
type Finding struct {
	// Rule identifies which check produced the finding.
	Rule string
	// Command is the command the finding concerns, or the change rule name for
	// change findings.
	Command string
	// Flag is the offending flag, including its leading dashes, when the finding
	// concerns one.
	Flag string
	// File is where the author should go, with a line number when one is known.
	File string
	// Message says what to do about it.
	Message string
}

// String renders a finding as a single actionable line.
func (f Finding) String() string {
	subject := f.Command
	if f.Flag != "" {
		subject = fmt.Sprintf("%s %s", f.Command, f.Flag)
	}
	return fmt.Sprintf("[%s] %s\n    %s\n    %s", f.Rule, subject, f.File, f.Message)
}

// Skippable reports whether the recorded escape hatch applies to this finding.
//
// Structural findings are never skippable: the shipped CLI surface has to be
// documented, and no pull request has a good reason to leave it otherwise.
// Change findings are heuristic, so an author who knows the change is invisible
// to users can record why and move on.
func (f Finding) Skippable() bool {
	return f.Rule == RuleChangeUndocumented
}

// Config drives a single gate run.
type Config struct {
	// RepoRoot is the path to the repository root. Empty means the process
	// working directory.
	RepoRoot string
	// Metadata is the raw JSON emitted by `azd app metadata`.
	Metadata []byte
	// ChangedFiles holds the repository-relative paths a pull request touches.
	// Empty disables the change rules, which is the right behavior for a local
	// run with no diff to reason about.
	ChangedFiles []string
	// SkipReason, when set, suppresses change findings and is echoed into the
	// report so the decision stays on the record.
	SkipReason string
}

// Result is the outcome of a gate run.
type Result struct {
	// Findings holds every gap that survived the escape hatch.
	Findings []Finding
	// Skipped holds the change findings the escape hatch suppressed.
	Skipped []Finding
	// SkipReason is the reason the author recorded.
	SkipReason string
	// CommandCount is how many visible commands were checked, so a run that
	// silently checked nothing is obvious in the log.
	CommandCount int
}

// Failed reports whether the gate should fail the build.
func (r *Result) Failed() bool { return len(r.Findings) > 0 }

// Run executes the gate.
func Run(cfg Config) (*Result, error) {
	md, err := ParseMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}

	refBytes, err := os.ReadFile(filepath.Join(cfg.RepoRoot, filepath.FromSlash(referencePath)))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", referencePath, err)
	}
	ref := ParseReference(string(refBytes))

	detailDocs, err := readDetailDocs(filepath.Join(cfg.RepoRoot, filepath.FromSlash(commandsDir)))
	if err != nil {
		return nil, err
	}

	result := &Result{
		SkipReason:   cfg.SkipReason,
		CommandCount: len(md.DocumentableCommands()),
	}
	result.Findings = CheckCoverage(md, ref, detailDocs)

	for _, f := range CheckChangedFiles(cfg.ChangedFiles) {
		if cfg.SkipReason != "" && f.Skippable() {
			result.Skipped = append(result.Skipped, f)
			continue
		}
		result.Findings = append(result.Findings, f)
	}
	sortFindings(result.Findings)

	return result, nil
}

// Report writes a human readable summary of the run.
//
// Write errors are ignored because the destination is the build log: there is
// no recovery beyond the failure the caller is already reporting.
func (r *Result) Report(w io.Writer) {
	if len(r.Skipped) > 0 {
		_, _ = fmt.Fprintf(w, "Skipped %d change finding(s), reason: %s\n\n", len(r.Skipped), r.SkipReason)
		for _, f := range r.Skipped {
			_, _ = fmt.Fprintf(w, "  skipped: %s\n", f.Message)
		}
		_, _ = fmt.Fprintln(w)
	}

	if !r.Failed() {
		_, _ = fmt.Fprintf(w, "Docs gate passed. %d command(s) documented.\n", r.CommandCount)
		return
	}

	_, _ = fmt.Fprintf(w, "Docs gate failed with %d finding(s) across %d command(s).\n\n", len(r.Findings), r.CommandCount)
	for _, f := range r.Findings {
		_, _ = fmt.Fprintln(w, f.String())
		_, _ = fmt.Fprintln(w)
	}

	if r.hasOnlySkippable() {
		_, _ = fmt.Fprintf(w, "Every finding is a heuristic. If this change really is invisible to users, "+
			"add a line to the pull request body: %s <why>\n", SkipMarker)
		return
	}
	_, _ = fmt.Fprintln(w, "Structural findings cannot be skipped. Document the command surface and rerun.")
}

// hasOnlySkippable reports whether the escape hatch would clear the whole run.
func (r *Result) hasOnlySkippable() bool {
	for _, f := range r.Findings {
		if !f.Skippable() {
			return false
		}
	}
	return len(r.Findings) > 0
}

// SkipMarker is the prefix a pull request body uses to record why the change
// rules do not apply.
const SkipMarker = "Docs-Not-Needed:"

// ParseSkipReason extracts the recorded escape hatch from a pull request body.
//
// The reason is required to be non-empty: a bare marker records nothing, and a
// gate that accepts "because" teaches people to write "because".
func ParseSkipReason(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, SkipMarker) {
			continue
		}
		reason := strings.TrimSpace(strings.TrimPrefix(trimmed, SkipMarker))
		if reason != "" {
			return reason
		}
	}
	return ""
}

// readDetailDocs returns the set of command slugs that have a detail doc.
func readDetailDocs(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", commandsDir, err)
	}
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		out[strings.TrimSuffix(name, ".md")] = true
	}
	return out, nil
}
