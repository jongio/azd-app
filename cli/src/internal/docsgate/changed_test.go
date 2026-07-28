package docsgate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckChangedFiles(t *testing.T) {
	tests := []struct {
		name      string
		changed   []string
		wantRules []string
	}{
		{
			name:    "no changed files disables the rules",
			changed: nil,
		},
		{
			name:      "command change without docs fires",
			changed:   []string{"cli/src/cmd/app/commands/run.go"},
			wantRules: []string{"cli-commands"},
		},
		{
			name:    "command change with a reference edit is satisfied",
			changed: []string{"cli/src/cmd/app/commands/run.go", "cli/docs/cli-reference.md"},
		},
		{
			name:    "any doc edit satisfies every rule",
			changed: []string{"cli/src/cmd/app/commands/run.go", "cli/src/internal/mcp/tools.go", "README.md"},
		},
		{
			name:    "website edit satisfies the rules",
			changed: []string{"cli/dashboard/src/App.tsx", "web/src/pages/index.astro"},
		},
		{
			name:    "test-only command change is ignored",
			changed: []string{"cli/src/cmd/app/commands/run_test.go"},
		},
		{
			name:    "non-Go file under commands is ignored",
			changed: []string{"cli/src/cmd/app/commands/testdata/fixture.yaml"},
		},
		{
			name:      "mcp change fires its own rule",
			changed:   []string{"cli/src/internal/mcp/server.go"},
			wantRules: []string{"mcp-tools"},
		},
		{
			name:      "dashboard change fires regardless of extension",
			changed:   []string{"cli/dashboard/src/components/Card.tsx"},
			wantRules: []string{"dashboard-ui"},
		},
		{
			name: "several surfaces fire independently",
			changed: []string{
				"cli/src/cmd/app/commands/run.go",
				"cli/src/internal/mcp/server.go",
				"cli/dashboard/src/App.tsx",
			},
			wantRules: []string{"cli-commands", "dashboard-ui", "mcp-tools"},
		},
		{
			name:    "unrelated change fires nothing",
			changed: []string{"cli/src/internal/portmanager/manager.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := CheckChangedFiles(tc.changed)

			var got []string
			for _, f := range findings {
				assert.Equal(t, RuleChangeUndocumented, f.Rule)
				assert.True(t, f.Skippable(), "change findings honor the escape hatch")
				got = append(got, f.Command)
			}
			assert.ElementsMatch(t, tc.wantRules, got)
		})
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{name: "single", files: []string{"a.go"}, want: "a.go"},
		{name: "at the cap", files: []string{"a.go", "b.go", "c.go"}, want: "a.go, b.go, c.go"},
		{name: "over the cap", files: []string{"a.go", "b.go", "c.go", "d.go", "e.go"}, want: "a.go, b.go, c.go and 2 more"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, summarize(tc.files))
		})
	}
}

func TestChangeFindingNamesAFile(t *testing.T) {
	findings := CheckChangedFiles([]string{
		"cli/src/cmd/app/commands/zebra.go",
		"cli/src/cmd/app/commands/alpha.go",
	})

	require.Len(t, findings, 1)
	assert.Equal(t, "cli/src/cmd/app/commands/alpha.go", findings[0].File,
		"the named file is stable across runs")
	assert.Contains(t, findings[0].Message, "cli/docs/cli-reference.md")
}

func TestIsDocPath(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{name: "file under a doc tree", file: "cli/docs/cli-reference.md", want: true},
		{name: "website source", file: "web/src/pages/index.astro", want: true},
		{name: "root readme", file: "README.md", want: true},
		{name: "contributor guide", file: "CONTRIBUTING.md", want: true},
		{name: "changelog", file: "CHANGELOG.md", want: true},
		{name: "source file", file: "cli/src/cmd/app/commands/run.go", want: false},
		{name: "unrelated root file", file: "AGENTS.md", want: false},
		{name: "backup of a doc file is not a doc", file: "README.md.bak", want: false},
		{name: "directory entry matched as a prefix only", file: "cli/docsgate.go", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isDocPath(tc.file))
		})
	}
}

func TestContributorGuideSatisfiesChangeRules(t *testing.T) {
	// Documenting the gate itself has to clear the gate, or the first change
	// that explains the rules is blocked by them.
	findings := CheckChangedFiles([]string{
		"cli/src/cmd/app/commands/run.go",
		"CONTRIBUTING.md",
	})

	assert.Empty(t, findings)
}

func TestHasAnyPrefixAndContainsAny(t *testing.T) {
	assert.True(t, hasAnyPrefix("cli/docs/x.md", []string{"cli/docs/"}))
	assert.False(t, hasAnyPrefix("cli/src/x.go", []string{"cli/docs/"}))
	assert.True(t, containsAny("run_test.go", []string{"_test.go"}))
	assert.False(t, containsAny("run.go", []string{"_test.go"}))
}
