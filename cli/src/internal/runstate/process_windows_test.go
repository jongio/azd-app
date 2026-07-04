//go:build windows

package runstate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPidAliveWindows(t *testing.T) {
	tests := []struct {
		name string
		pid  int
		want bool
	}{
		{name: "current process is alive", pid: os.Getpid(), want: true},
		{name: "zero pid is not alive", pid: 0, want: false},
		{name: "negative pid is not alive", pid: -1, want: false},
		{name: "pid above max is not alive", pid: 0x7FFFFFFF + 1, want: false},
		{name: "unused high pid is not alive", pid: 2_147_483_646, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pidAlive(tt.pid))
		})
	}
}
