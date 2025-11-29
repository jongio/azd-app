package procutil

import (
	"os"
	"runtime"
	"testing"
)

func TestIsProcessRunningCurrentProcess(t *testing.T) {
	// Current process should always be running
	pid := os.Getpid()

	result := IsProcessRunning(pid)

	// On most systems, our own process should be detectable
	// On Windows, the result may vary due to Signal(0) limitations
	if runtime.GOOS != "windows" && !result {
		t.Errorf("IsProcessRunning(%d) = false for current process, expected true", pid)
	}
}

func TestIsProcessRunningInvalidPID(t *testing.T) {
	tests := []struct {
		name string
		pid  int
	}{
		{"zero pid", 0},
		{"negative pid", -1},
		{"very negative pid", -999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProcessRunning(tt.pid)
			if result {
				t.Errorf("IsProcessRunning(%d) = true, expected false for invalid PID", tt.pid)
			}
		})
	}
}

func TestIsProcessRunningNonExistentPID(t *testing.T) {
	// Use a very high PID that's unlikely to exist
	// 999999 is typically beyond the maximum PID on most systems
	pid := 999999

	result := IsProcessRunning(pid)

	// This should return false on most systems
	// Note: On Windows, this may incorrectly return true due to known limitations
	if runtime.GOOS != "windows" && result {
		t.Errorf("IsProcessRunning(%d) = true for non-existent PID, expected false", pid)
	}
}
