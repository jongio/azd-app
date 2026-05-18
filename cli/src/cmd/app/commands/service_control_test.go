package commands

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/constants"
)

func TestParseServiceList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single service",
			input: "api",
			want:  []string{"api"},
		},
		{
			name:  "multiple services",
			input: "api,web,worker",
			want:  []string{"api", "web", "worker"},
		},
		{
			name:  "whitespace trimmed",
			input: " api , web , worker ",
			want:  []string{"api", "web", "worker"},
		},
		{
			name:  "trailing comma ignored",
			input: "api,web,",
			want:  []string{"api", "web"},
		},
		{
			name:    "invalid service name",
			input:   "../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseServiceList(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsRunning(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{constants.StatusRunning, true},
		{constants.StatusReady, true},
		{constants.StatusStopped, false},
		{constants.StatusNotRunning, false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isRunning(tt.status); got != tt.want {
				t.Errorf("isRunning(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsStopped(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{constants.StatusStopped, true},
		{constants.StatusNotRunning, true},
		{constants.StatusRunning, false},
		{constants.StatusReady, false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isStopped(tt.status); got != tt.want {
				t.Errorf("isStopped(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestNewErrorResult(t *testing.T) {
	result := newErrorResult("api", "something went wrong")

	if result.ServiceName != "api" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "api")
	}
	if result.Success {
		t.Error("Success should be false")
	}
	if result.Error != "something went wrong" {
		t.Errorf("Error = %q, want %q", result.Error, "something went wrong")
	}
	if result.Message != "something went wrong" {
		t.Errorf("Message = %q, want %q", result.Message, "something went wrong")
	}
}
