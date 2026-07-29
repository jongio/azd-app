package docsgate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleMetadata = `{
  "schemaVersion": "1.0",
  "id": "jongio.azd.app",
  "commands": [
    {
      "name": ["run"],
      "short": "Run the development environment",
      "flags": [
        {"name": "service", "shorthand": "s", "type": "string"},
        {"name": "trace-log-file", "type": "string", "hidden": true},
        {"name": "old-flag", "type": "bool", "deprecated": "use --new-flag"}
      ]
    },
    {
      "name": ["notifications"],
      "short": "Manage notifications",
      "flags": [{"name": "help", "type": "bool"}],
      "subcommands": [
        {
          "name": ["notifications", "list"],
          "short": "List notification history",
          "flags": [{"name": "unread", "type": "bool"}]
        }
      ]
    },
    {
      "name": ["listen"],
      "short": "Extension framework integration",
      "hidden": true,
      "subcommands": [
        {"name": ["listen", "inner"], "short": "Inner"}
      ]
    },
    {
      "name": ["completion"],
      "short": "Generate shell completion"
    }
  ]
}`

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr error
		assert  func(t *testing.T, md *Metadata)
	}{
		{
			name: "valid document",
			raw:  sampleMetadata,
			assert: func(t *testing.T, md *Metadata) {
				assert.Equal(t, "jongio.azd.app", md.ID)
				assert.Len(t, md.Commands, 4)
			},
		},
		{
			name: "leading noise is trimmed",
			raw:  "warning: something happened\n" + sampleMetadata,
			assert: func(t *testing.T, md *Metadata) {
				assert.Equal(t, "1.0", md.SchemaVersion)
			},
		},
		{
			name:    "empty command tree is rejected",
			raw:     `{"schemaVersion":"1.0","commands":[]}`,
			wantErr: ErrEmptyCommandTree,
		},
		{
			name: "invalid JSON is rejected",
			raw:  "not json at all",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			md, err := ParseMetadata([]byte(tc.raw))
			switch {
			case tc.wantErr != nil:
				require.ErrorIs(t, err, tc.wantErr)
			case tc.assert == nil:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				tc.assert(t, md)
			}
		})
	}
}

func TestCommandRefIdentity(t *testing.T) {
	cmd := CommandRef{Path: []string{"notifications", "list"}}

	assert.Equal(t, "notifications list", cmd.Name())
	assert.Equal(t, "notifications-list", cmd.Slug())
	assert.Equal(t, "notifications", cmd.Root())
	assert.Equal(t, "list", cmd.Leaf())
	assert.False(t, cmd.TopLevel())

	top := CommandRef{Path: []string{"run"}}
	assert.True(t, top.TopLevel())
	assert.Equal(t, "run", top.Root())
	assert.Equal(t, "run", top.Leaf())
}

func TestDocumentedFlagsExcludesHiddenAndDeprecated(t *testing.T) {
	cmd := CommandRef{Flags: []Flag{
		{Name: "zebra"},
		{Name: "secret", Hidden: true},
		{Name: "legacy", Deprecated: "use --modern"},
		{Name: "alpha"},
	}}

	flags := cmd.DocumentedFlags()

	require.Len(t, flags, 2)
	assert.Equal(t, "alpha", flags[0].Name, "flags should be sorted by name")
	assert.Equal(t, "zebra", flags[1].Name)
}

func TestDocumentableCommands(t *testing.T) {
	md, err := ParseMetadata([]byte(sampleMetadata))
	require.NoError(t, err)

	var names []string
	for _, c := range md.DocumentableCommands() {
		names = append(names, c.Name())
	}

	assert.Equal(t, []string{"notifications", "notifications list", "run"}, names,
		"hidden commands, their subcommands, and Cobra builtins are excluded")
}

func TestKnownCommandSlugsIncludesHiddenAndBuiltins(t *testing.T) {
	md, err := ParseMetadata([]byte(sampleMetadata))
	require.NoError(t, err)

	known := md.KnownCommandSlugs()

	assert.True(t, known["run"])
	assert.True(t, known["notifications-list"])
	assert.True(t, known["listen"], "hidden commands stay documentable without being required")
	assert.True(t, known["listen-inner"])
	assert.True(t, known["completion"])
	assert.False(t, known["nonexistent"])
}
