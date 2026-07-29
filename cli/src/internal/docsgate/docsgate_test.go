package docsgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeRepo lays out the documentation tree the gate reads and returns its root.
func writeRepo(t *testing.T, reference string, detailDocs ...string) string {
	t.Helper()

	root := t.TempDir()
	docsDir := filepath.Join(root, filepath.FromSlash(commandsDir))
	require.NoError(t, os.MkdirAll(docsDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(referencePath)), []byte(reference), 0o600))

	for _, slug := range detailDocs {
		require.NoError(t, os.WriteFile(filepath.Join(docsDir, slug+".md"), []byte("# "+slug), 0o600))
	}
	return root
}

const runMetadata = `{"commands":[{"name":["run"],"short":"Run","flags":[{"name":"detach","type":"bool"}]}]}`

const runReference = "## Commands Overview\n\n| `run` | Run | [→ Full Spec](commands/run.md) |\n\n" +
	"## `azd app run`\n\n### Flags\n\n| `--detach` | | bool | | Detach |\n"

func TestRunPasses(t *testing.T) {
	root := writeRepo(t, runReference, "run")

	result, err := Run(Config{RepoRoot: root, Metadata: []byte(runMetadata)})

	require.NoError(t, err)
	assert.False(t, result.Failed())
	assert.Equal(t, 1, result.CommandCount)

	var sb strings.Builder
	result.Report(&sb)
	assert.Contains(t, sb.String(), "Docs gate passed")
}

func TestRunReportsStructuralGaps(t *testing.T) {
	root := writeRepo(t, "## Commands Overview\n\n| `other` | Other |\n")

	result, err := Run(Config{RepoRoot: root, Metadata: []byte(runMetadata)})

	require.NoError(t, err)
	assert.True(t, result.Failed())

	var sb strings.Builder
	result.Report(&sb)
	report := sb.String()
	assert.Contains(t, report, "Docs gate failed")
	assert.Contains(t, report, "Structural findings cannot be skipped")
	assert.NotContains(t, report, SkipMarker,
		"the escape hatch is not offered when it would not help")
}

func TestRunSkipReasonSuppressesChangeFindingsOnly(t *testing.T) {
	root := writeRepo(t, runReference, "run")
	changed := []string{"cli/src/cmd/app/commands/run.go"}

	withoutReason, err := Run(Config{RepoRoot: root, Metadata: []byte(runMetadata), ChangedFiles: changed})
	require.NoError(t, err)
	require.True(t, withoutReason.Failed())
	require.Len(t, withoutReason.Findings, 1)
	assert.Equal(t, RuleChangeUndocumented, withoutReason.Findings[0].Rule)

	var offered strings.Builder
	withoutReason.Report(&offered)
	assert.Contains(t, offered.String(), SkipMarker,
		"a run with only skippable findings points at the escape hatch")

	withReason, err := Run(Config{
		RepoRoot:     root,
		Metadata:     []byte(runMetadata),
		ChangedFiles: changed,
		SkipReason:   "internal refactor, no user-visible change",
	})
	require.NoError(t, err)
	assert.False(t, withReason.Failed())
	require.Len(t, withReason.Skipped, 1)

	var sb strings.Builder
	withReason.Report(&sb)
	assert.Contains(t, sb.String(), "internal refactor, no user-visible change")
}

func TestRunSkipReasonCannotClearStructuralFindings(t *testing.T) {
	root := writeRepo(t, "## Commands Overview\n\n| `other` | Other |\n")

	result, err := Run(Config{
		RepoRoot:     root,
		Metadata:     []byte(runMetadata),
		ChangedFiles: []string{"cli/src/cmd/app/commands/run.go"},
		SkipReason:   "trust me",
	})

	require.NoError(t, err)
	assert.True(t, result.Failed(), "the shipped command surface must be documented regardless")
}

func TestRunErrors(t *testing.T) {
	t.Run("invalid metadata", func(t *testing.T) {
		root := writeRepo(t, runReference, "run")
		_, err := Run(Config{RepoRoot: root, Metadata: []byte("{}")})
		require.ErrorIs(t, err, ErrEmptyCommandTree)
	})

	t.Run("missing reference", func(t *testing.T) {
		_, err := Run(Config{RepoRoot: t.TempDir(), Metadata: []byte(runMetadata)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), referencePath)
	})

	t.Run("missing commands directory", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "cli", "docs"), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(referencePath)), []byte(runReference), 0o600))

		_, err := Run(Config{RepoRoot: root, Metadata: []byte(runMetadata)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), commandsDir)
	})
}

func TestParseSkipReason(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "reason on its own line",
			body: "Fixes #1\n\nDocs-Not-Needed: internal refactor only\n",
			want: "internal refactor only",
		},
		{
			name: "leading whitespace is tolerated",
			body: "  Docs-Not-Needed:   spacing cleanup  ",
			want: "spacing cleanup",
		},
		{
			name: "bare marker records nothing",
			body: "Docs-Not-Needed:",
		},
		{
			name: "marker with only whitespace records nothing",
			body: "Docs-Not-Needed:    ",
		},
		{
			name: "no marker",
			body: "Just a normal pull request body",
		},
		{
			name: "first non-empty reason wins",
			body: "Docs-Not-Needed:\nDocs-Not-Needed: the real reason\n",
			want: "the real reason",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseSkipReason(tc.body))
		})
	}
}

func TestFindingString(t *testing.T) {
	withFlag := Finding{
		Rule:    RuleFlagUndocumented,
		Command: "run",
		Flag:    "--detach",
		File:    "cli/docs/cli-reference.md:370",
		Message: "add a Flags table row",
	}
	rendered := withFlag.String()
	assert.Contains(t, rendered, "[flag-undocumented] run --detach")
	assert.Contains(t, rendered, "cli/docs/cli-reference.md:370")
	assert.Contains(t, rendered, "add a Flags table row")

	withoutFlag := Finding{Rule: RuleCommandUndocumented, Command: "clean", File: referencePath, Message: "add a section"}
	assert.Contains(t, withoutFlag.String(), "[command-undocumented] clean\n")
}

func TestFindingSkippable(t *testing.T) {
	assert.True(t, Finding{Rule: RuleChangeUndocumented}.Skippable())

	for _, rule := range []string{
		RuleCommandUndocumented,
		RuleCommandMissingOverview,
		RuleCommandMissingDetailDoc,
		RuleSubcommandUndocumented,
		RuleFlagUndocumented,
		RuleDocOrphaned,
	} {
		assert.False(t, Finding{Rule: rule}.Skippable(), rule+" is structural")
	}
}

func TestReadDetailDocs(t *testing.T) {
	root := writeRepo(t, runReference, "run", "clean")
	dir := filepath.Join(root, filepath.FromSlash(commandsDir))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "nested"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0o600))

	docs, err := readDetailDocs(dir)

	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"run": true, "clean": true}, docs)
}
