package docsgate

import (
	"fmt"
	"sort"
	"strings"
)

// RuleChangeUndocumented fires when a pull request touches a user-facing
// surface without touching the documentation for it.
const RuleChangeUndocumented = "change-undocumented"

// docPrefixes are the paths that count as documentation. A change under any of
// them satisfies every change rule, because a single edit often covers several
// surfaces at once and demanding a specific file would just be guessing.
//
// Root level entries match a whole file, so they are compared exactly rather
// than as prefixes.
var docPrefixes = []string{
	"cli/docs/",
	"web/src/",
	"web/public/",
	"docs/",
	"README.md",
	"CHANGELOG.md",
	"CONTRIBUTING.md",
}

// changeRule maps a user-facing source surface to the documentation that
// describes it.
//
// These rules are heuristics. They exist to catch behavior changes that add no
// flag and therefore slip past the structural rules, such as renaming an output
// column. Because a heuristic will sometimes be wrong, every change rule honors
// the recorded escape hatch.
type changeRule struct {
	// Name identifies the rule in output.
	Name string
	// Prefixes are the repository-relative path prefixes the rule watches.
	Prefixes []string
	// Suffix, when set, further narrows the watched files.
	Suffix string
	// Exclude drops files whose path contains any of these substrings.
	Exclude []string
	// Surface is the human readable name used in the failure message.
	Surface string
	// Hint tells the author where the documentation for this surface lives.
	Hint string
}

// changeRules is the full table. Keep it short. Every entry is a promise that
// touching those files really does change what a user sees.
var changeRules = []changeRule{
	{
		Name:     "cli-commands",
		Prefixes: []string{"cli/src/cmd/app/commands/"},
		Suffix:   ".go",
		Exclude:  []string{"_test.go"},
		Surface:  "CLI command behavior",
		Hint:     "update cli/docs/cli-reference.md or the matching cli/docs/commands/*.md",
	},
	{
		Name:     "mcp-tools",
		Prefixes: []string{"cli/src/internal/mcp/"},
		Suffix:   ".go",
		Exclude:  []string{"_test.go"},
		Surface:  "MCP tool surface",
		Hint:     "update web/src/pages/mcp/ or cli/docs/",
	},
	{
		Name:     "dashboard-ui",
		Prefixes: []string{"cli/dashboard/src/"},
		Surface:  "dashboard UI",
		Hint:     "update the dashboard docs under web/src/ or cli/docs/",
	},
}

// CheckChangedFiles applies the change rules to a pull request's file list.
//
// Passing a nil or empty list disables the rules, which is what happens on a
// local run where there is no diff to reason about.
func CheckChangedFiles(changed []string) []Finding {
	if len(changed) == 0 {
		return nil
	}
	if hasDocChange(changed) {
		return nil
	}

	var findings []Finding
	for _, rule := range changeRules {
		matched := rule.matches(changed)
		if len(matched) == 0 {
			continue
		}
		findings = append(findings, Finding{
			Rule:    RuleChangeUndocumented,
			Command: rule.Name,
			File:    matched[0],
			Message: fmt.Sprintf("this pull request changes %s (%s) but updates no documentation; %s",
				rule.Surface, summarize(matched), rule.Hint),
		})
	}

	sortFindings(findings)
	return findings
}

// matches returns the changed files this rule watches, sorted for stable output.
func (r changeRule) matches(changed []string) []string {
	var out []string
	for _, file := range changed {
		if !hasAnyPrefix(file, r.Prefixes) {
			continue
		}
		if r.Suffix != "" && !strings.HasSuffix(file, r.Suffix) {
			continue
		}
		if containsAny(file, r.Exclude) {
			continue
		}
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}

// hasDocChange reports whether the pull request touches documentation at all.
//
// One documentation edit clears every change rule. The gate's job is to make
// authors think about docs, not to police which file they chose.
func hasDocChange(changed []string) bool {
	for _, file := range changed {
		if isDocPath(file) {
			return true
		}
	}
	return false
}

// isDocPath reports whether a path is documentation.
//
// Entries ending in a slash match a directory tree. Everything else names a
// single file and is matched exactly, so a stray README.md.bak does not read as
// a documentation change.
func isDocPath(file string) bool {
	for _, p := range docPrefixes {
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(file, p) {
				return true
			}
			continue
		}
		if file == p {
			return true
		}
	}
	return false
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// summarize renders a short file list, naming at most three files so a large
// pull request does not produce an unreadable message.
func summarize(files []string) string {
	const maxNamed = 3
	if len(files) <= maxNamed {
		return strings.Join(files, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(files[:maxNamed], ", "), len(files)-maxNamed)
}
