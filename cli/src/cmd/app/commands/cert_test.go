package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertCommand(t *testing.T) {
	t.Parallel()

	cmd := NewCertCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "cert", cmd.Use)

	tests := []struct {
		name     string
		flagName string
		flagType string
	}{
		{
			name:     "host flag",
			flagName: "host",
			flagType: "stringSlice",
		},
		{
			name:     "force flag",
			flagName: "force",
			flagType: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.flagType, flag.Value.Type())
		})
	}
}
