package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinOpenURLPath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		extraPath string
		want      string
		wantErr   string
	}{
		{
			name:      "appends route to base path",
			base:      "http://localhost:3000/api",
			extraPath: "/health",
			want:      "http://localhost:3000/api/health",
		},
		{
			name:      "empty path returns base unchanged",
			base:      "http://localhost:3000/api",
			extraPath: "   ",
			want:      "http://localhost:3000/api",
		},
		{
			name:      "preserves trailing slash",
			base:      "http://localhost:3000",
			extraPath: "docs/",
			want:      "http://localhost:3000/docs/",
		},
		{
			name:      "preserves query string",
			base:      "http://localhost:3000/api?debug=1",
			extraPath: "health",
			want:      "http://localhost:3000/api/health?debug=1",
		},
		{
			name:      "preserves fragment",
			base:      "http://localhost:3000/api#top",
			extraPath: "health",
			want:      "http://localhost:3000/api/health#top",
		},
		{
			name:      "re-encodes path after join",
			base:      "http://localhost:3000/a%2Fb",
			extraPath: "health",
			want:      "http://localhost:3000/a/b/health",
		},
		{
			name:      "rejects url without scheme",
			base:      "localhost:3000",
			extraPath: "/health",
			wantErr:   "missing scheme or host",
		},
		{
			name:      "rejects unparsable url",
			base:      "http://[::1",
			extraPath: "/health",
			wantErr:   "invalid service URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := joinOpenURLPath(tt.base, tt.extraPath)
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

func TestOpenURLFromService(t *testing.T) {
	tests := []struct {
		name string
		svc  service.Service
		want string
	}{
		{
			name: "custom url wins over ports",
			svc: service.Service{
				Local: &service.LocalServiceConfig{CustomURL: "https://web.example.test"},
				Ports: []string{"8080:80"},
			},
			want: "https://web.example.test",
		},
		{
			name: "non-docker bare port is the host port",
			svc:  service.Service{Ports: []string{"8080"}},
			want: "http://localhost:8080",
		},
		{
			name: "docker bare port has no published host port",
			svc:  service.Service{Image: "nginx", Ports: []string{"80"}},
			want: "",
		},
		{
			name: "docker explicit mapping uses the host port",
			svc:  service.Service{Image: "nginx", Ports: []string{"3000:80"}},
			want: "http://localhost:3000",
		},
		{
			name: "bind ip mapping uses the host port",
			svc:  service.Service{Ports: []string{"127.0.0.1:3000:8080"}},
			want: "http://localhost:3000",
		},
		{
			name: "ipv6 bind mapping uses the host port",
			svc:  service.Service{Ports: []string{"[::1]:3000:8080"}},
			want: "http://localhost:3000",
		},
		{
			name: "tcp protocol suffix is stripped",
			svc:  service.Service{Ports: []string{"3000:8080/tcp"}},
			want: "http://localhost:3000",
		},
		{
			name: "udp mapping is skipped",
			svc:  service.Service{Ports: []string{"3000:8080/udp"}},
			want: "",
		},
		{
			name: "first publishable port wins",
			svc:  service.Service{Image: "nginx", Ports: []string{"80", "3000:8080"}},
			want: "http://localhost:3000",
		},
		{
			name: "malformed port spec is ignored",
			svc:  service.Service{Ports: []string{"not-a-port"}},
			want: "",
		},
		{
			name: "no ports yields no url",
			svc:  service.Service{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, openURLFromService(tt.svc))
		})
	}
}

func TestResolveOpenServiceURLFromCustomURL(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  web:
    host: local
    project: ./web
    local:
      customUrl: https://web.example.test
`)
	mkdirOpenService(t, dir, "web")

	got, err := resolveOpenServiceURL(dir, "web", "/health")
	require.NoError(t, err)
	assert.Equal(t, "https://web.example.test/health", got)
}

func TestResolveOpenServiceURLFromPorts(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  api:
    host: local
    project: ./api
    ports:
      - "8080:80"
`)
	mkdirOpenService(t, dir, "api")

	got, err := resolveOpenServiceURL(dir, "api", "docs")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/docs", got)
}

func TestResolveOpenServiceURLMissingURL(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  worker:
    host: local
    project: ./worker
`)
	mkdirOpenService(t, dir, "worker")

	_, err := resolveOpenServiceURL(dir, "worker", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no known URL")
}

func TestResolveOpenServiceURLNotFoundListsServices(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  api:
    host: local
    project: ./api
  web:
    host: local
    project: ./web
`)
	mkdirOpenService(t, dir, "api")
	mkdirOpenService(t, dir, "web")

	_, err := resolveOpenServiceURL(dir, "missing", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `service "missing" not found`)
	assert.Contains(t, err.Error(), "Available services: api, web")
}

func TestResolveOpenServiceURLSurfacesParseError(t *testing.T) {
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, "name: open-test\nservices: [this is not a map\n")

	_, err := resolveOpenServiceURL(dir, "web", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
	assert.NotContains(t, err.Error(), "No services are defined")
}

func TestRunOpenPrintWritesURLWithoutBrowser(t *testing.T) {
	newOpenCommandFixture(t)

	opened := false
	restoreOpenState(t)
	openURL = func(string) error {
		opened = true
		return nil
	}

	out, err := executeOpenCommand(t, "web", "--print", "--path", "/health")
	require.NoError(t, err)
	assert.Equal(t, "https://web.example.test/health\n", out)
	assert.False(t, opened, "browser should not be launched with --print")
}

func TestRunOpenLaunchesBrowser(t *testing.T) {
	newOpenCommandFixture(t)

	var openedURL string
	restoreOpenState(t)
	openURL = func(u string) error {
		openedURL = u
		return nil
	}

	out, err := executeOpenCommand(t, "web")
	require.NoError(t, err)
	assert.Equal(t, "https://web.example.test", openedURL)
	assert.Empty(t, out)
}

// newOpenCommandFixture creates a project with a single "web" service that has a
// custom URL and makes it the working directory for the test.
func newOpenCommandFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeOpenAzureYaml(t, dir, `
name: open-test
services:
  web:
    host: local
    project: ./web
    local:
      customUrl: https://web.example.test
`)
	mkdirOpenService(t, dir, "web")
	t.Chdir(dir)
}

// restoreOpenState resets the package-level flag and browser hooks that the open
// command binds to, so tests do not leak state into each other.
func restoreOpenState(t *testing.T) {
	t.Helper()
	originalOpenURL := openURL
	t.Cleanup(func() {
		openURL = originalOpenURL
		openPath = ""
		openPrint = false
	})
}

func executeOpenCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := NewOpenCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func writeOpenAzureYaml(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600))
}

func mkdirOpenService(t *testing.T, dir, name string) {
	t.Helper()
	require.NoError(t, os.Mkdir(filepath.Join(dir, name), 0o750))
}
