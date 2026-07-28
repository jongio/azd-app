package docsgate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rulesFired counts findings by rule so a test can assert on what fired without
// depending on message wording.
func rulesFired(findings []Finding) map[string]int {
	out := map[string]int{}
	for _, f := range findings {
		out[f.Rule]++
	}
	return out
}

func metadataFor(t *testing.T, raw string) *Metadata {
	t.Helper()
	md, err := ParseMetadata([]byte(raw))
	require.NoError(t, err)
	return md
}

func TestCheckCoverageFullyDocumented(t *testing.T) {
	md := metadataFor(t, `{"commands":[
		{"name":["run"],"short":"Run","flags":[{"name":"detach","type":"bool"},{"name":"no-prompt","type":"bool"}]},
		{"name":["notifications"],"short":"Notify","subcommands":[
			{"name":["notifications","list"],"short":"List","flags":[{"name":"unread","type":"bool"}]}
		]}
	]}`)

	ref := ParseReference("## Commands Overview\n\n" +
		"| `run` | Run | [→ Full Spec](commands/run.md) |\n" +
		"| `notifications` | Notify | [→ Full Spec](commands/notifications.md) |\n\n" +
		"## Global Information\n\n### Global Flags\n\n" +
		"| `--no-prompt` | | bool | | Run without prompts |\n\n" +
		"## `azd app run`\n\n### Flags\n\n" +
		"| `--detach` | | bool | | Detach |\n\n" +
		"## `azd app notifications`\n\n### Subcommands\n\n" +
		"| `list` | List |\n\n### Flags\n\n" +
		"| `list` | `--unread` | bool | | Unread only |\n")

	findings := CheckCoverage(md, ref, map[string]bool{"run": true, "notifications": true})

	assert.Empty(t, findings)
}

func TestCheckCoverageMissingTopLevelDocs(t *testing.T) {
	md := metadataFor(t, `{"commands":[{"name":["clean"],"short":"Clean","flags":[{"name":"deps","type":"bool"}]}]}`)
	ref := ParseReference("## Commands Overview\n\n| `other` | Other |\n")

	findings := CheckCoverage(md, ref, map[string]bool{})

	fired := rulesFired(findings)
	assert.Equal(t, 1, fired[RuleCommandUndocumented])
	assert.Equal(t, 1, fired[RuleCommandMissingOverview])
	assert.Equal(t, 1, fired[RuleCommandMissingDetailDoc])
	assert.Zero(t, fired[RuleFlagUndocumented],
		"a brand new command reports the missing section, not every flag on it")
}

func TestCheckCoverageFlagFallsBackToGlobalFlags(t *testing.T) {
	md := metadataFor(t, `{"commands":[{"name":["run"],"short":"Run","flags":[{"name":"help","type":"bool"}]}]}`)

	documented := ParseReference("## Commands Overview\n\n| `run` | Run |\n\n" +
		"## Global Information\n\n### Global Flags\n\n| `--help` | `-h` | bool | | Show help |\n\n" +
		"## `azd app run`\n")
	assert.Empty(t, CheckCoverage(md, documented, map[string]bool{"run": true}))

	undocumented := ParseReference("## Commands Overview\n\n| `run` | Run |\n\n## `azd app run`\n")
	findings := CheckCoverage(md, undocumented, map[string]bool{"run": true})
	require.Len(t, findings, 1)
	assert.Equal(t, RuleFlagUndocumented, findings[0].Rule)
	assert.Equal(t, "--help", findings[0].Flag)
}

func TestCheckCoverageSubcommandFlagRollsUpToParent(t *testing.T) {
	md := metadataFor(t, `{"commands":[{"name":["notifications"],"short":"Notify","subcommands":[
		{"name":["notifications","clear"],"short":"Clear","flags":[{"name":"yes","type":"bool"}]}
	]}]}`)

	ref := ParseReference("## Commands Overview\n\n| `notifications` | Notify |\n\n" +
		"## `azd app notifications`\n\n### Subcommands\n\n| `clear` | Clear |\n\n" +
		"### Flags\n\n| `clear` | `--yes` | bool | | Skip the prompt |\n")

	findings := CheckCoverage(md, ref, map[string]bool{"notifications": true})

	assert.Empty(t, findings,
		"a subcommand's flag documented in the parent section counts as documented")
}

func TestCheckCoverageSubcommandNeedsNaming(t *testing.T) {
	md := metadataFor(t, `{"commands":[{"name":["notifications"],"short":"Notify","subcommands":[
		{"name":["notifications","stats"],"short":"Show stats"}
	]}]}`)

	ref := ParseReference("## Commands Overview\n\n| `notifications` | Notify |\n\n## `azd app notifications`\n")

	findings := CheckCoverage(md, ref, map[string]bool{"notifications": true})

	require.Len(t, findings, 1)
	assert.Equal(t, RuleSubcommandUndocumented, findings[0].Rule)
	assert.Equal(t, "notifications stats", findings[0].Command)
	assert.NotContains(t, rulesFired(findings), RuleCommandMissingDetailDoc,
		"a subcommand never needs its own detail doc")
}

func TestCheckCoverageSubcommandMayHaveOwnSection(t *testing.T) {
	md := metadataFor(t, `{"commands":[{"name":["notifications"],"short":"Notify","subcommands":[
		{"name":["notifications","stats"],"short":"Show stats"}
	]}]}`)

	ref := ParseReference("## Commands Overview\n\n| `notifications` | Notify |\n\n" +
		"## `azd app notifications`\n\n## `azd app notifications stats`\n")

	assert.Empty(t, CheckCoverage(md, ref, map[string]bool{"notifications": true}))
}

func TestCheckCoverageOrphans(t *testing.T) {
	md := metadataFor(t, `{"commands":[
		{"name":["run"],"short":"Run"},
		{"name":["mcp"],"short":"MCP","hidden":true}
	]}`)

	ref := ParseReference("## Commands Overview\n\n| `run` | Run |\n\n" +
		"## `azd app run`\n\n## `azd app mcp`\n\n## `azd app removed`\n")

	findings := CheckCoverage(md, ref, map[string]bool{"run": true, "mcp": true, "deleted": true})

	var orphans []string
	for _, f := range findings {
		if f.Rule == RuleDocOrphaned {
			orphans = append(orphans, f.Command)
		}
	}

	assert.ElementsMatch(t, []string{"removed", "deleted"}, orphans,
		"documenting a hidden command stays legal; documenting a deleted one does not")
}

func TestSortFindingsIsStable(t *testing.T) {
	findings := []Finding{
		{Rule: RuleFlagUndocumented, Command: "run", Flag: "--zebra"},
		{Rule: RuleCommandUndocumented, Command: "clean"},
		{Rule: RuleFlagUndocumented, Command: "run", Flag: "--alpha"},
		{Rule: RuleFlagUndocumented, Command: "add", Flag: "--dry-run"},
	}

	sortFindings(findings)

	assert.Equal(t, RuleCommandUndocumented, findings[0].Rule)
	assert.Equal(t, "add", findings[1].Command)
	assert.Equal(t, "--alpha", findings[2].Flag)
	assert.Equal(t, "--zebra", findings[3].Flag)
}
