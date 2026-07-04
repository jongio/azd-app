//go:build !windows

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetachSysProcAttr(t *testing.T) {
	attr := detachSysProcAttr()
	require.NotNil(t, attr)
	require.True(t, attr.Setsid)
}
