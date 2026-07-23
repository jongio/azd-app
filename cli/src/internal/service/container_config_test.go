package service

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitVolumeSource(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantSource string
		wantRest   string
		wantFound  bool
	}{
		{"named", "pgdata:/var/lib/data", "pgdata", "/var/lib/data", true},
		{"relative bind", "./config.json:/app/config.json", "./config.json", "/app/config.json", true},
		{"absolute posix bind", "/host/data:/data", "/host/data", "/data", true},
		{"bind with mode", "/host:/data:ro", "/host", "/data:ro", true},
		{"windows drive backslash", `C:\data:/container`, `C:\data`, "/container", true},
		{"windows drive forwardslash", `C:/data:/container`, `C:/data`, "/container", true},
		{"anonymous", "/var/lib/data", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, rest, found := splitVolumeSource(tt.spec)
			assert.Equal(t, tt.wantFound, found)
			if found {
				assert.Equal(t, tt.wantSource, src)
				assert.Equal(t, tt.wantRest, rest)
			}
		})
	}
}

func TestIsNamedVolume(t *testing.T) {
	assert.True(t, isNamedVolume("pgdata"))
	assert.True(t, isNamedVolume("my-data.v2"))
	assert.False(t, isNamedVolume("./config"))
	assert.False(t, isNamedVolume("/abs/path"))
	assert.False(t, isNamedVolume(`C:\data`))
	assert.False(t, isNamedVolume(".hidden")) // leading dot -> not a named volume
}

func TestResolveVolumeSpec_PassThrough(t *testing.T) {
	// Named volumes and anonymous volumes are returned unchanged.
	for _, spec := range []string{"pgdata:/var/lib/postgresql/data", "/var/lib/data"} {
		got, err := resolveVolumeSpec(spec, filepath.FromSlash("/project"))
		require.NoError(t, err)
		assert.Equal(t, spec, got)
	}
}

func TestResolveVolumeSpec_RelativeBindResolves(t *testing.T) {
	projectDir := t.TempDir()
	got, err := resolveVolumeSpec("./config.json:/app/config.json", projectDir)
	require.NoError(t, err)
	want := filepath.Join(projectDir, "config.json") + ":/app/config.json"
	assert.Equal(t, want, got)
}

func TestResolveVolumeSpec_TraversalRejected(t *testing.T) {
	projectDir := t.TempDir()
	_, err := resolveVolumeSpec("../../etc/passwd:/etc/passwd", projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes")
}

func TestResolveVolumeSpec_AbsoluteBindPassThrough(t *testing.T) {
	// An absolute host path the user provided explicitly is kept (cleaned).
	abs := t.TempDir()
	got, err := resolveVolumeSpec(abs+":/data", abs)
	require.NoError(t, err)
	assert.Equal(t, filepath.Clean(abs)+":/data", got)
}

func TestResolveVolumeSpec_WindowsDriveBind(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows drive-path resolution is only exercised on windows")
	}
	got, err := resolveVolumeSpec(`C:\data:/container`, `C:\project`)
	require.NoError(t, err)
	assert.Equal(t, `C:\data:/container`, got)
}

func TestResolveVolumeSpec_Empty(t *testing.T) {
	_, err := resolveVolumeSpec("   ", "/project")
	assert.Error(t, err)
}

func TestDeriveNetworkName(t *testing.T) {
	dir := t.TempDir()
	n1 := DeriveNetworkName(dir)
	n2 := DeriveNetworkName(dir)

	assert.Equal(t, n1, n2, "must be deterministic for the same project dir")
	assert.True(t, strings.HasPrefix(n1, "azd-app-"), "got %q", n1)
	assert.NotEqual(t, n1, DeriveNetworkName(t.TempDir()), "different dirs -> different names")

	// Result must be a valid Docker network name.
	assert.Regexp(t, `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, n1)
}

func TestSanitizeNetworkComponent(t *testing.T) {
	assert.Equal(t, "my-project", sanitizeNetworkComponent("My Project"))
	assert.Equal(t, "web", sanitizeNetworkComponent("web"))
	assert.Equal(t, "a.b_c-d", sanitizeNetworkComponent("a.b_c-d"))
	assert.Equal(t, "app", sanitizeNetworkComponent("app!!!")) // trailing invalid trimmed
}

func TestParseCommandLine(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"azurite --blobHost 0.0.0.0", []string{"azurite", "--blobHost", "0.0.0.0"}},
		{`sh -c "echo hello world"`, []string{"sh", "-c", "echo hello world"}},
		{`echo 'single quoted'`, []string{"echo", "single quoted"}},
		{"  spaced   out  ", []string{"spaced", "out"}},
		{`hello "world`, []string{"hello", "world"}}, // unclosed quote: best-effort
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, parseCommandLine(tt.in))
		})
	}
}

func TestGetCommandArgs(t *testing.T) {
	// Array form is returned verbatim.
	s := Service{CommandArgs: []string{"postgres", "-c", "max_connections=200"}}
	assert.Equal(t, []string{"postgres", "-c", "max_connections=200"}, s.GetCommandArgs())

	// String form is tokenized.
	s2 := Service{Command: "azurite --blobHost 0.0.0.0"}
	assert.Equal(t, []string{"azurite", "--blobHost", "0.0.0.0"}, s2.GetCommandArgs())

	// Nothing configured -> nil.
	assert.Nil(t, (&Service{}).GetCommandArgs())
}
