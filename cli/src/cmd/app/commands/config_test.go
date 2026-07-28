package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configTestAPI = "api"
	configTestWeb = "web"
)

func TestNewConfigCommand(t *testing.T) {
	cmd := NewConfigCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "config [service]", cmd.Use)
	assert.Equal(t, "Show the effective configuration for each service", cmd.Short)
	require.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.ValidArgsFunction)
	assert.Nil(t, cmd.Flags().Lookup("output"))
}

func TestBuildServiceConfigInferredType(t *testing.T) {
	svc := service.Service{
		Language: "js",
		Project:  "./web",
		Command:  "npm run dev",
		Ports:    []string{"3000:8080"},
		Uses:     []string{"redis"},
	}

	cfg := buildServiceConfig(svc)

	if cfg.Type != "http" {
		t.Errorf("expected inferred type http, got %q", cfg.Type)
	}
	if cfg.TypeSource != "inferred" {
		t.Errorf("expected typeSource inferred, got %q", cfg.TypeSource)
	}
	if cfg.Language != "js" || cfg.Project != "./web" || cfg.Command != "npm run dev" {
		t.Errorf("unexpected passthrough fields: %+v", cfg)
	}
	if len(cfg.Ports) != 1 || cfg.Ports[0] != "3000:8080" {
		t.Errorf("expected ports [3000:8080], got %v", cfg.Ports)
	}
	if len(cfg.Uses) != 1 || cfg.Uses[0] != "redis" {
		t.Errorf("expected uses [redis], got %v", cfg.Uses)
	}
	if len(cfg.Blocks) != 0 {
		t.Errorf("expected no configured blocks, got %v", cfg.Blocks)
	}
}

func TestBuildServiceConfigExplicitType(t *testing.T) {
	svc := service.Service{Type: "process", Command: "worker"}

	cfg := buildServiceConfig(svc)

	if cfg.Type != "process" {
		t.Errorf("expected type process, got %q", cfg.Type)
	}
	if cfg.TypeSource != "explicit" {
		t.Errorf("expected typeSource explicit, got %q", cfg.TypeSource)
	}
}

func TestConfiguredBlocks(t *testing.T) {
	svc := service.Service{
		Docker:      &service.DockerConfig{},
		Healthcheck: &service.HealthcheckConfig{},
		Restart:     &service.RestartConfig{},
		Resources:   &service.ResourceThresholds{},
		Logs:        &service.ServiceLogsConfig{},
		Local:       &service.LocalServiceConfig{},
		Azure:       &service.AzureServiceConfig{},
	}

	blocks := configuredBlocks(svc)

	want := []string{"docker", "healthcheck", "restart", "resources", "logs", "local", "azure"}
	if len(blocks) != len(want) {
		t.Fatalf("expected %d blocks, got %v", len(want), blocks)
	}
	for i, b := range want {
		if blocks[i] != b {
			t.Errorf("block %d: expected %q, got %q", i, b, blocks[i])
		}
	}
}

func TestConfiguredBlocksEmpty(t *testing.T) {
	if blocks := configuredBlocks(service.Service{}); len(blocks) != 0 {
		t.Errorf("expected no blocks for empty service, got %v", blocks)
	}
}

func TestSelectServiceConfigsAll(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			configTestWeb: {Language: "js", Ports: []string{"3000"}},
			configTestAPI: {Type: "process", Command: "worker"},
		},
	}

	configs, order, err := selectServiceConfigs(azureYaml, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
	if len(order) != 2 || order[0] != configTestAPI || order[1] != configTestWeb {
		t.Errorf("expected sorted order [api web], got %v", order)
	}
	if configs[configTestAPI].Type != "process" || configs[configTestWeb].Type != "http" {
		t.Errorf("unexpected resolved types: %+v", configs)
	}
}

func TestSelectServiceConfigsSingle(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			configTestWeb: {Ports: []string{"3000"}},
			configTestAPI: {Type: "process"},
		},
	}

	configs, order, err := selectServiceConfigs(azureYaml, []string{configTestAPI})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 || len(order) != 1 || order[0] != configTestAPI {
		t.Fatalf("expected only api selected, got order %v configs %v", order, configs)
	}
	if _, ok := configs[configTestWeb]; ok {
		t.Error("expected web to be excluded")
	}
}

func TestSelectServiceConfigsUnknown(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			configTestWeb: {Ports: []string{"3000"}},
			configTestAPI: {Type: "process"},
		},
	}

	_, _, err := selectServiceConfigs(azureYaml, []string{"nope"})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not found") || !strings.Contains(msg, "api") || !strings.Contains(msg, "web") {
		t.Errorf("error should list available services, got %q", msg)
	}
}

func TestSelectServiceConfigsNoServices(t *testing.T) {
	azureYaml := &service.AzureYaml{Services: map[string]service.Service{}}

	// No arg: empty selection, no error.
	configs, order, err := selectServiceConfigs(azureYaml, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 0 || len(order) != 0 {
		t.Errorf("expected empty selection, got order %v configs %v", order, configs)
	}

	// Named arg against an empty project reports the no-services message.
	_, _, err = selectServiceConfigs(azureYaml, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "No services are defined") {
		t.Errorf("expected no-services error, got %v", err)
	}
}

func TestServiceConfigJSONShape(t *testing.T) {
	configs := map[string]serviceConfig{
		configTestAPI: buildServiceConfig(service.Service{
			Type:        "http",
			Language:    "python",
			Ports:       []string{"8000"},
			Healthcheck: &service.HealthcheckConfig{},
		}),
	}

	data, err := json.Marshal(configs)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	api, ok := decoded[configTestAPI]
	if !ok {
		t.Fatal("expected object keyed by service name")
	}
	if api["type"] != "http" {
		t.Errorf("expected type http, got %v", api["type"])
	}
	if api["typeSource"] != "explicit" {
		t.Errorf("expected typeSource explicit, got %v", api["typeSource"])
	}
	blocks, ok := api["configured"].([]any)
	if !ok || len(blocks) != 1 || blocks[0] != "healthcheck" {
		t.Errorf("expected configured [healthcheck], got %v", api["configured"])
	}
	// Unset optional fields must be omitted.
	if _, ok := api["project"]; ok {
		t.Error("expected project to be omitted when empty")
	}
}

func TestPrintServiceConfig(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })
	require.NoError(t, cliout.SetFormat("default"))

	tests := []struct {
		name string
		cfg  serviceConfig
		want []string
	}{
		{
			name: configTestAPI,
			cfg: serviceConfig{
				Host:       "containerapp",
				Type:       "http",
				TypeSource: "explicit",
				Language:   "python",
				Project:    filepath.Join("project", configTestAPI),
				Command:    "python -m uvicorn main:app",
				Image:      "example.azurecr.io/api:latest",
				Ports:      []string{"8080:80", "8443:443"},
				Uses:       []string{"db", "cache"},
				Blocks:     []string{"docker", "logs"},
			},
			want: []string{
				configTestAPI,
				"host",
				"containerapp",
				"type",
				"http (explicit)",
				"language",
				"python",
				"project",
				filepath.Join("project", configTestAPI),
				"command",
				"python -m uvicorn main:app",
				"image",
				"example.azurecr.io/api:latest",
				"ports",
				"8080:80, 8443:443",
				"uses",
				"db, cache",
				"configured",
				"docker, logs",
			},
		},
		{
			name: "worker",
			cfg: serviceConfig{
				Type:       "process",
				TypeSource: "inferred",
			},
			want: []string{
				"worker",
				"type",
				"process (inferred)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureStdout(t, func() error {
				printServiceConfig(tt.name, tt.cfg)
				return nil
			})
			require.NoError(t, err)
			for _, want := range tt.want {
				assert.Contains(t, out, want)
			}
		})
	}
}

func TestRunConfigCommand(t *testing.T) {
	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })

	tests := []struct {
		name       string
		azureYaml  string
		args       []string
		format     string
		want       []string
		wantAbsent []string
		wantJSON   bool
		wantErr    string
	}{
		{
			name:      "all services as text",
			azureYaml: configCommandAzureYaml(),
			format:    "default",
			want: []string{
				"azd app config",
				configTestAPI,
				configTestWeb,
				"containerapp",
				"http (explicit)",
				"js",
				"npm run dev",
				"nginx:alpine",
				"3000:3000",
				"api",
				"docker, healthcheck, restart, resources, logs, local, azure",
			},
		},
		{
			name:      "single service as json",
			azureYaml: configCommandAzureYaml(),
			args:      []string{configTestAPI},
			format:    "json",
			wantJSON:  true,
		},
		{
			name:       "no services as text",
			azureYaml:  "name: empty\nservices: {}\n",
			format:     "default",
			want:       []string{"No services are defined in azure.yaml"},
			wantAbsent: []string{"Effective service configuration"},
		},
		{
			name:      "unknown service returns error",
			azureYaml: configCommandAzureYaml(),
			args:      []string{"missing"},
			format:    "default",
			wantErr:   `service "missing" not found`,
		},
		{
			name:    "missing azure yaml returns error",
			format:  "default",
			wantErr: "failed to load azure.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			// config reports the service project path as resolved by
			// ParseAzureYaml, which canonicalizes via filepath.EvalSymlinks.
			// t.TempDir() is not already canonical everywhere: on macOS /var
			// resolves to /private/var, and on Windows CI TEMP uses an 8.3 short
			// name that expands to its long form. Resolve tmpDir the same way so
			// the expected path matches config's output on every OS.
			if resolved, err := filepath.EvalSymlinks(tmpDir); err == nil {
				tmpDir = resolved
			}
			if tt.azureYaml != "" {
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(tt.azureYaml), 0o600))
			}
			t.Chdir(tmpDir)
			require.NoError(t, cliout.SetFormat(tt.format))

			out, runErr := captureStdout(t, func() error {
				return runConfig(nil, tt.args)
			})
			if tt.wantErr != "" {
				require.Error(t, runErr)
				assert.Contains(t, runErr.Error(), tt.wantErr)
				return
			}

			require.NoError(t, runErr)
			for _, want := range tt.want {
				assert.Contains(t, out, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, out, absent)
			}
			if tt.wantJSON {
				var parsed map[string]serviceConfig
				require.NoError(t, json.Unmarshal([]byte(out), &parsed))
				assert.Equal(t, map[string]serviceConfig{
					configTestAPI: {
						Host:       "containerapp",
						Type:       "http",
						TypeSource: "explicit",
						Language:   "go",
						Project:    filepath.Join(tmpDir, configTestAPI),
						Command:    "go run .",
						Ports:      []string{"8080:8080"},
						Uses:       []string{"db"},
						Blocks:     []string{"healthcheck"},
					},
				}, parsed)
			}
		})
	}
}

func configCommandAzureYaml() string {
	return `name: config-test
services:
  web:
    host: containerapp
    language: js
    project: ./web
    command: npm run dev
    image: nginx:alpine
    ports:
      - 3000:3000
    uses:
      - api
    docker:
      remoteBuild: true
    healthcheck:
      path: /healthz
    restart:
      policy: on-failure
    resources:
      cpuPercent: 80
    logs:
      enabled: true
    local:
      enabled: true
    azure:
      resourceName: web-app
  api:
    host: containerapp
    type: http
    language: go
    project: ./api
    command: go run .
    ports:
      - 8080:8080
    uses:
      - db
    healthcheck:
      path: /ready
`
}
