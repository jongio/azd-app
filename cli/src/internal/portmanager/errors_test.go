package portmanager

import (
	"strings"
	"testing"
)

func TestPortInUseError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *PortInUseError
		wantContain []string
	}{
		{
			name: "with process name",
			err: &PortInUseError{
				Port:        8080,
				PID:         1234,
				ProcessName: "node",
				ServiceName: "web-api",
			},
			wantContain: []string{"8080", "web-api", "node", "1234"},
		},
		{
			name: "without process name",
			err: &PortInUseError{
				Port:        3000,
				PID:         5678,
				ServiceName: "frontend",
			},
			wantContain: []string{"3000", "frontend", "5678"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, want := range tt.wantContain {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, should contain %q", errMsg, want)
				}
			}
		})
	}
}

func TestPortRangeExhaustedError_Error(t *testing.T) {
	err := &PortRangeExhaustedError{
		StartPort: 8000,
		EndPort:   8100,
	}

	errMsg := err.Error()
	wantContain := []string{"8000", "8100", "no available"}

	for _, want := range wantContain {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, should contain %q", errMsg, want)
		}
	}
}

func TestInvalidPortError_Error(t *testing.T) {
	err := &InvalidPortError{
		Port:   65536,
		Reason: "exceeds maximum port number",
	}

	errMsg := err.Error()
	wantContain := []string{"65536", "exceeds maximum"}

	for _, want := range wantContain {
		if !strings.Contains(errMsg, want) {
			t.Errorf("Error() = %q, should contain %q", errMsg, want)
		}
	}
}

func TestPortInUseError_Implementation(t *testing.T) {
	// Verify it implements error interface
	var _ error = &PortInUseError{}
}

func TestPortRangeExhaustedError_Implementation(t *testing.T) {
	// Verify it implements error interface
	var _ error = &PortRangeExhaustedError{}
}

func TestInvalidPortError_Implementation(t *testing.T) {
	// Verify it implements error interface
	var _ error = &InvalidPortError{}
}
