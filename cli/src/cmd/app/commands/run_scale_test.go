package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScaleFlags(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		want    map[string]int
		wantErr string
	}{
		{
			name:    "empty input",
			entries: nil,
			want:    map[string]int{},
		},
		{
			name:    "single valid entry with spaces",
			entries: []string{" worker = 3 "},
			want: map[string]int{
				"worker": 3,
			},
		},
		{
			name:    "multiple valid entries from repeated and comma-separated flags",
			entries: []string{"worker=3", "api=2", "jobs=1"},
			want: map[string]int{
				"worker": 3,
				"api":    2,
				"jobs":   1,
			},
		},
		{
			name:    "missing equals",
			entries: []string{"worker3"},
			wantErr: "expected name=count format",
		},
		{
			name:    "empty service name",
			entries: []string{"=3"},
			wantErr: "service name is required",
		},
		{
			name:    "invalid integer",
			entries: []string{"worker=abc"},
			wantErr: "invalid instance count",
		},
		{
			name:    "count less than one",
			entries: []string{"worker=0"},
			wantErr: "must be at least 1",
		},
		{
			name:    "duplicate service",
			entries: []string{"worker=2", "worker=3"},
			wantErr: "duplicate scale entry",
		},
		{
			name:    "empty entry",
			entries: []string{" "},
			wantErr: "empty scale entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScaleFlags(tt.entries)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunCommandScaleFlagParsing(t *testing.T) {
	runScale = nil
	cmd := NewRunCommand()

	err := cmd.ParseFlags([]string{"--scale", "worker=3,api=2", "--scale", "jobs=1"})
	require.NoError(t, err)
	assert.Equal(t, []string{"worker=3", "api=2", "jobs=1"}, runScale)
}
