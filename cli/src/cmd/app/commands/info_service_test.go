package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInfoCommandServiceFlag(t *testing.T) {
	cmd := NewInfoCommand()

	svc := cmd.Flags().Lookup("service")
	require.NotNil(t, svc, "--service flag should be defined")
	assert.Equal(t, "s", svc.Shorthand)
	assert.Equal(t, "", svc.DefValue)

	all := cmd.Flags().Lookup("all")
	require.NotNil(t, all, "--all flag should still be defined")
}
