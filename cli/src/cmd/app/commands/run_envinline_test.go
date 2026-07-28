package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeInlineEnv(t *testing.T) {
	tests := []struct {
		name    string
		initial map[string]string
		entries []string
		want    map[string]string
		wantErr string
	}{
		{
			name:    "single entry",
			initial: map[string]string{},
			entries: []string{"KEY=value"},
			want:    map[string]string{"KEY": "value"},
		},
		{
			name:    "multiple entries",
			initial: map[string]string{},
			entries: []string{"A=1", "B=2"},
			want:    map[string]string{"A": "1", "B": "2"},
		},
		{
			name:    "inline overrides existing",
			initial: map[string]string{"A": "fromfile"},
			entries: []string{"A=inline"},
			want:    map[string]string{"A": "inline"},
		},
		{
			name:    "empty value allowed",
			initial: map[string]string{},
			entries: []string{"EMPTY="},
			want:    map[string]string{"EMPTY": ""},
		},
		{
			name:    "value with equals signs preserved",
			initial: map[string]string{},
			entries: []string{"CONN=a=b=c"},
			want:    map[string]string{"CONN": "a=b=c"},
		},
		{
			name:    "key whitespace trimmed",
			initial: map[string]string{},
			entries: []string{" FOO=bar"},
			want:    map[string]string{"FOO": "bar"},
		},
		{
			name:    "value whitespace preserved",
			initial: map[string]string{},
			entries: []string{"FOO= bar "},
			want:    map[string]string{"FOO": " bar "},
		},
		{
			name:    "missing equals is an error",
			initial: map[string]string{},
			entries: []string{"NOEQUALS"},
			wantErr: `invalid --env value "NOEQUALS"`,
		},
		{
			name:    "blank trimmed key is an error",
			initial: map[string]string{},
			entries: []string{"  =value"},
			wantErr: `invalid --env value "  =value"`,
		},
		{
			name:    "empty key is an error",
			initial: map[string]string{},
			entries: []string{"=value"},
			wantErr: `invalid --env value "=value"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mergeInlineEnv(tt.initial, tt.entries)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.initial)
		})
	}
}

func TestLoadEnvironmentVariables_InlinePrecedence(t *testing.T) {
	origFile, origInline := runEnvFile, runEnvInline
	t.Cleanup(func() {
		runEnvFile, runEnvInline = origFile, origInline
	})

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("A=fromfile\nB=fromfile\n"), 0o600))

	runEnvFile = envPath
	runEnvInline = []string{"A=inline", "C=new"}

	got, err := loadEnvironmentVariables()
	require.NoError(t, err)
	assert.Equal(t, "inline", got["A"], "inline value should override the file value")
	assert.Equal(t, "fromfile", got["B"], "file-only value should remain")
	assert.Equal(t, "new", got["C"], "inline-only value should be added")
}

func TestLoadEnvironmentVariables_InlineOnly(t *testing.T) {
	origFile, origInline := runEnvFile, runEnvInline
	t.Cleanup(func() {
		runEnvFile, runEnvInline = origFile, origInline
	})

	runEnvFile = ""
	runEnvInline = []string{"ONLY=1"}

	got, err := loadEnvironmentVariables()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"ONLY": "1"}, got)
}

func TestLoadEnvironmentVariables_InlineError(t *testing.T) {
	origFile, origInline := runEnvFile, runEnvInline
	t.Cleanup(func() {
		runEnvFile, runEnvInline = origFile, origInline
	})

	runEnvFile = ""
	runEnvInline = []string{"bad"}

	_, err := loadEnvironmentVariables()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --env value")
}
