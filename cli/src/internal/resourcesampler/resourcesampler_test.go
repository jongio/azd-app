package resourcesampler

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSampleCurrentProcess verifies that sampling the running test process
// returns non-zero resident memory. A live Go process always has an RSS.
func TestSampleCurrentProcess(t *testing.T) {
	usage := Sample(os.Getpid())

	assert.Greater(t, usage.MemoryBytes, uint64(0), "current process should report resident memory")
	assert.GreaterOrEqual(t, usage.CPUPercent, 0.0, "CPU percent should never be negative")
}

// TestSampleInvalidPID verifies fail-soft behavior for pids that cannot map to
// a live process. Callers rely on a zero Usage instead of an error when a
// service exits between reading the registry and sampling it.
func TestSampleInvalidPID(t *testing.T) {
	tests := []struct {
		name string
		pid  int
	}{
		{name: "zero", pid: 0},
		{name: "negative", pid: -1},
		// 2147483646 is the largest int32 minus one; no real process uses it,
		// so NewProcess reports the process as gone.
		{name: "nonexistent", pid: 2147483646},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := Sample(tt.pid)
			assert.Equal(t, Usage{}, usage, "expected zero usage for pid %d", tt.pid)
		})
	}
}
