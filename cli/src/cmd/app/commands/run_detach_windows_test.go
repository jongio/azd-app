//go:build windows

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetachSysProcAttr(t *testing.T) {
	attr := detachSysProcAttr()
	require.NotNil(t, attr)

	const detachedProcess = 0x00000008
	const createNewProcessGroup = 0x00000200
	require.Equal(t, uint32(detachedProcess|createNewProcessGroup), attr.CreationFlags)
}
