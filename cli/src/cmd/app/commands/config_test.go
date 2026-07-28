package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

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
			"web": {Language: "js", Ports: []string{"3000"}},
			"api": {Type: "process", Command: "worker"},
		},
	}

	configs, order, err := selectServiceConfigs(azureYaml, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
	if len(order) != 2 || order[0] != "api" || order[1] != "web" {
		t.Errorf("expected sorted order [api web], got %v", order)
	}
	if configs["api"].Type != "process" || configs["web"].Type != "http" {
		t.Errorf("unexpected resolved types: %+v", configs)
	}
}

func TestSelectServiceConfigsSingle(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {Ports: []string{"3000"}},
			"api": {Type: "process"},
		},
	}

	configs, order, err := selectServiceConfigs(azureYaml, []string{"api"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 || len(order) != 1 || order[0] != "api" {
		t.Fatalf("expected only api selected, got order %v configs %v", order, configs)
	}
	if _, ok := configs["web"]; ok {
		t.Error("expected web to be excluded")
	}
}

func TestSelectServiceConfigsUnknown(t *testing.T) {
	azureYaml := &service.AzureYaml{
		Services: map[string]service.Service{
			"web": {Ports: []string{"3000"}},
			"api": {Type: "process"},
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
		"api": buildServiceConfig(service.Service{
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
	api, ok := decoded["api"]
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
