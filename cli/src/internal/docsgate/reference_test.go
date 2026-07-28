package docsgate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReferenceSections(t *testing.T) {
	ref := ParseReference("## `azd app run`\n\n## `azd app support-bundle`\n\n### Subcommands\n\n| `list` | List things |\n")

	require.Contains(t, ref.Sections, "run")
	require.Contains(t, ref.Sections, "support-bundle",
		"hyphenated command headings must parse")
	assert.Equal(t, 1, ref.Sections["run"].Line)
	assert.True(t, ref.Sections["support-bundle"].Names["list"])
}

func TestParseReferenceIgnoresFencedCodeBlocks(t *testing.T) {
	// A shell comment inside a fence is indistinguishable from a heading. If the
	// parser treats it as one, the section closes and every later flag row is
	// attributed to nothing.
	content := "## `azd app health`\n\n" +
		"```bash\n" +
		"# Quick health check of all services\n" +
		"azd app health\n" +
		"```\n\n" +
		"### Flags\n\n" +
		"| `--summary-only` | | bool | | Print only the summary |\n"

	ref := ParseReference(content)

	require.Contains(t, ref.Sections, "health")
	assert.True(t, ref.Sections["health"].Flags["summary-only"],
		"a flag documented after a fenced code block still belongs to the section")
}

func TestParseReferenceFlagCells(t *testing.T) {
	tests := []struct {
		name    string
		row     string
		want    string
		wantErr bool
	}{
		{
			name: "first cell flag",
			row:  "| `--service` | `-s` | string | | Limit to a service |",
			want: "service",
		},
		{
			name: "hyphenated flag",
			row:  "| `--fail-on-degraded` | | bool | `false` | Fail on degraded |",
			want: "fail-on-degraded",
		},
		{
			name: "second cell flag behind a subcommand column",
			row:  "| `list` | `--unread` | bool | `false` | Show unread only |",
			want: "unread",
		},
		{
			name:    "flag named only in a description does not count",
			row:     "| `--context` | | int | `0` | Context lines (requires --level) |",
			want:    "context",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref := ParseReference("## `azd app x`\n\n### Flags\n\n" + tc.row + "\n")

			section := ref.Sections["x"]
			require.NotNil(t, section)
			assert.True(t, section.Flags[tc.want])
			if tc.wantErr {
				assert.False(t, section.Flags["level"],
					"a flag mentioned in a description column is not documented by it")
			}
		})
	}
}

func TestParseReferenceGlobalFlags(t *testing.T) {
	content := "## Global Information\n\n" +
		"### Global Flags\n\n" +
		"| `--no-prompt` | | bool | | Run without prompts |\n\n" +
		"### Something Else\n\n" +
		"| `--not-global` | | bool | | Not a global flag |\n"

	ref := ParseReference(content)

	assert.True(t, ref.GlobalFlags["no-prompt"])
	assert.False(t, ref.GlobalFlags["not-global"],
		"an H3 closes the Global Flags subsection")
}

func TestParseReferenceOverview(t *testing.T) {
	content := "## Commands Overview\n\n" +
		"| Command | Description | Detailed Spec |\n" +
		"|---------|-------------|---------------|\n" +
		"| `init` | Initialize | [→ Full Spec](commands/init.md) |\n" +
		"| `support-bundle` | Diagnostics | [→ Full Spec](commands/support-bundle.md) |\n" +
		"| `orphan` | No link |\n"

	ref := ParseReference(content)

	assert.Equal(t, "init", ref.Overview["init"])
	assert.Equal(t, "support-bundle", ref.Overview["support-bundle"])

	spec, ok := ref.Overview["orphan"]
	assert.True(t, ok, "a row without a link still records the command")
	assert.Empty(t, spec)
}

func TestParseReferenceHeadingClosesSection(t *testing.T) {
	content := "## `azd app run`\n\n" +
		"| `--detach` | | bool | | Detach |\n\n" +
		"## Exit Codes\n\n" +
		"| `--stray` | | bool | | Belongs to nothing |\n"

	ref := ParseReference(content)

	require.Contains(t, ref.Sections, "run")
	assert.True(t, ref.Sections["run"].Flags["detach"])
	assert.False(t, ref.Sections["run"].Flags["stray"],
		"an H2 closes the previous command section")
}

func TestTableCells(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "trimmed cells",
			line: "|  `--a`  |  b  |",
			want: []string{"`--a`", "b"},
		},
		{
			name: "separator row",
			line: "|------|-------|",
			want: []string{"------", "-------"},
		},
		{
			name: "not a table row",
			line: "plain text",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tableCells(tc.line))
		})
	}
}

func TestSlugify(t *testing.T) {
	assert.Equal(t, "notifications-list", slugify("notifications list"))
	assert.Equal(t, "run", slugify("run"))
	assert.Equal(t, "support-bundle", slugify("support-bundle"))
}
