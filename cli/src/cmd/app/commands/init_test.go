package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectServices_MultiServiceProject(t *testing.T) {
	// Use the testdata sample project
	rootDir := filepath.Join("testdata", "init-sample")
	absRoot, err := filepath.Abs(rootDir)
	require.NoError(t, err)

	services, err := detectServices(absRoot)
	require.NoError(t, err)

	// Build a map for easier assertion
	svcMap := make(map[string]DetectedService)
	for _, svc := range services {
		svcMap[svc.Name] = svc
	}

	// Verify Node.js Express API
	api, ok := svcMap["api"]
	require.True(t, ok, "should detect api service")
	assert.Equal(t, "ts", api.Language)
	assert.Equal(t, "Express", api.Framework)
	assert.Contains(t, api.Ports, "3000")
	assert.Equal(t, "http", api.Type)
	assert.Contains(t, api.Uses, "postgres")
	assert.Contains(t, api.Uses, "redis")
	assert.Contains(t, api.Uses, "servicebus")

	// Verify React+Vite frontend
	web, ok := svcMap["web"]
	require.True(t, ok, "should detect web service")
	assert.Equal(t, "ts", web.Language)
	assert.Equal(t, "Vite+React", web.Framework)
	assert.Contains(t, web.Ports, "5173")

	// Verify Python FastAPI worker
	worker, ok := svcMap["worker"]
	require.True(t, ok, "should detect worker service")
	assert.Equal(t, "python", worker.Language)
	assert.Equal(t, "FastAPI", worker.Framework)
	assert.Contains(t, worker.Ports, "8000")
	assert.Contains(t, worker.Uses, "mongodb")

	// Verify .NET gateway
	gateway, ok := svcMap["gateway"]
	require.True(t, ok, "should detect gateway service")
	assert.Equal(t, "dotnet", gateway.Language)
	assert.Contains(t, gateway.Ports, "5000")
	assert.Contains(t, gateway.Uses, "postgres")
	assert.Contains(t, gateway.Uses, "redis")

	// Verify Go notifications service
	notifications, ok := svcMap["notifications"]
	require.True(t, ok, "should detect notifications service")
	assert.Equal(t, "go", notifications.Language)
	assert.Equal(t, "http-server", notifications.Framework)
	assert.Contains(t, notifications.Ports, "8080")
	assert.Contains(t, notifications.Uses, "postgres")
	assert.Contains(t, notifications.Uses, "rabbitmq")

	// Verify Azure Functions jobs
	jobs, ok := svcMap["jobs"]
	require.True(t, ok, "should detect jobs service")
	assert.Contains(t, jobs.Ports, "7071")
	assert.Contains(t, jobs.Uses, "storage")
	assert.Contains(t, jobs.Uses, "cosmosdb")
}

func TestDetectServiceDependencies_NodePackageJson(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
		"dependencies": {
			"pg": "^8.0.0",
			"ioredis": "^5.0.0",
			"mongoose": "^7.0.0"
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	svc := DetectedService{
		Name:     "test",
		Language: "js",
		Project:  ".",
	}

	deps := detectServiceDependencies(svc, dir)
	assert.Contains(t, deps, "postgres")
	assert.Contains(t, deps, "redis")
	assert.Contains(t, deps, "mongodb")
}

func TestDetectServiceDependencies_PythonRequirements(t *testing.T) {
	dir := t.TempDir()
	reqs := "fastapi\nuvicorn\nasyncpg\naioredis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(reqs), 0o644))

	svc := DetectedService{
		Name:     "test",
		Language: "python",
		Project:  ".",
	}

	deps := detectServiceDependencies(svc, dir)
	assert.Contains(t, deps, "postgres")
	assert.Contains(t, deps, "redis")
}

func TestDetectServiceDependencies_GoMod(t *testing.T) {
	dir := t.TempDir()
	gomod := `module test

go 1.22

require (
	github.com/jackc/pgx/v5 v5.5.0
	github.com/segmentio/kafka-go v0.4.0
)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644))

	svc := DetectedService{
		Name:     "test",
		Language: "go",
		Project:  ".",
	}

	deps := detectServiceDependencies(svc, dir)
	assert.Contains(t, deps, "postgres")
	assert.Contains(t, deps, "kafka")
}

func TestDetectServiceDependencies_DotnetCsproj(t *testing.T) {
	dir := t.TempDir()
	csproj := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <ItemGroup>
    <PackageReference Include="Npgsql" Version="8.0.0" />
    <PackageReference Include="StackExchange.Redis" Version="2.7.0" />
    <PackageReference Include="MongoDB.Driver" Version="2.23.0" />
  </ItemGroup>
</Project>`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Test.csproj"), []byte(csproj), 0o644))

	svc := DetectedService{
		Name:     "test",
		Language: "dotnet",
		Project:  ".",
	}

	deps := detectServiceDependencies(svc, dir)
	assert.Contains(t, deps, "postgres")
	assert.Contains(t, deps, "redis")
	assert.Contains(t, deps, "mongodb")
}

func TestGenerateAzureYamlContent_IncludesUses(t *testing.T) {
	services := []DetectedService{
		{
			Name:     "api",
			Language: "ts",
			Project:  "api",
			Ports:    []string{"3000"},
			Command:  "npm run dev",
			Type:     "http",
			Uses:     []string{"postgres", "redis"},
		},
		{
			Name:     "web",
			Language: "ts",
			Project:  "web",
			Ports:    []string{"5173"},
			Command:  "npm run dev",
			Type:     "http",
		},
	}

	content := generateAzureYamlContent("/tmp/myapp", services)

	assert.Contains(t, content, "name: myapp")
	assert.Contains(t, content, "api:")
	assert.Contains(t, content, "uses:")
	assert.Contains(t, content, `- "postgres"`)
	assert.Contains(t, content, `- "redis"`)
	// web should NOT have uses
	lines := strings.Split(content, "\n")
	inWeb := false
	for _, line := range lines {
		if strings.Contains(line, "web:") {
			inWeb = true
		}
		if inWeb && strings.Contains(line, "uses:") {
			t.Fatal("web service should not have uses section")
		}
		if inWeb && strings.TrimSpace(line) == "" {
			break
		}
	}
}

func TestInferReqs(t *testing.T) {
	services := []DetectedService{
		{Language: "ts", PackageManager: "pnpm"},
		{Language: "python"},
		{Language: "go"},
		{Language: "dotnet"},
	}

	reqs := inferReqs(services)
	assert.Contains(t, reqs, "node")
	assert.Contains(t, reqs, "pnpm")
	assert.Contains(t, reqs, "python")
	assert.Contains(t, reqs, "go")
	assert.Contains(t, reqs, "dotnet")
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Project", "my-project"},
		{"api-service", "api-service"},
		{"My App (v2)", "my-app--v2"},
		{"", "app"},
		{"---", "app"},
		{"hello_world", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeName(tt.input))
		})
	}
}

func TestDetectNodeFrameworkAndConfig_DevDependencies(t *testing.T) {
	dir := t.TempDir()
	// Real-world setup: vite in devDependencies, react in dependencies
	pkg := `{
		"scripts": { "dev": "vite" },
		"dependencies": { "react": "^18.0.0", "react-dom": "^18.0.0" },
		"devDependencies": { "vite": "^5.0.0", "@vitejs/plugin-react": "^4.0.0" }
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	framework, ports, cmd := detectNodeFrameworkAndConfig(dir, "npm")
	assert.Equal(t, "Vite+React", framework)
	assert.Contains(t, ports, "5173")
	assert.Equal(t, "npm run dev", cmd)
}

func TestDetectNodeFrameworkAndConfig_DevDepsOnly(t *testing.T) {
	dir := t.TempDir()
	// SvelteKit puts everything in devDependencies
	pkg := `{
		"scripts": { "dev": "vite dev" },
		"devDependencies": { "@sveltejs/kit": "^2.0.0", "svelte": "^4.0.0", "vite": "^5.0.0" }
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	framework, ports, _ := detectNodeFrameworkAndConfig(dir, "pnpm")
	assert.Equal(t, "SvelteKit", framework)
	assert.Contains(t, ports, "5173")
}

func TestEnrichAzureYaml_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "azure.yaml")

	// Write azure.yaml with comments
	original := `# My project config
name: myapp # inline comment

# Services section
services:
  api:
    language: ts
    project: ./api
    # API runs on port 3000
    ports:
      - "3000"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(original), 0o644))

	// Enrich with a new service
	services := []DetectedService{
		{
			Name:     "web",
			Language: "ts",
			Project:  "web",
			Ports:    []string{"5173"},
			Command:  "pnpm run dev",
		},
	}

	err := enrichAzureYaml(yamlPath, services)
	require.NoError(t, err)

	// Read back and verify comments are preserved
	data, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "# My project config")
	assert.Contains(t, content, "# Services section")
	assert.Contains(t, content, "# API runs on port 3000")
	// Verify new service was added
	assert.Contains(t, content, "web:")
	assert.Contains(t, content, "5173")
}

func TestEnrichAzureYaml_CompleteServiceNeedsNoChanges(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "azure.yaml")
	original := `name: myapp
services:
  api:
    language: ts
    project: ./api
    ports:
      - "3000"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(original), 0o644))

	services := []DetectedService{{
		Name:     "api",
		Language: "ts",
		Project:  "./api",
		Ports:    []string{"3000"},
	}}

	require.NoError(t, enrichAzureYaml(yamlPath, services))

	actual, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(actual))
}

func TestInitCommand_DryRunDoesNotWriteAzureYaml(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "package.json"),
		[]byte(`{"scripts":{"dev":"vite"},"devDependencies":{"vite":"^5.0.0"}}`),
		0o644,
	))

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--dry-run"})

	require.NoError(t, cmd.Execute())
	assert.NoFileExists(t, filepath.Join(dir, "azure.yaml"))
}

func TestDetectServiceDependencies_NoFalsePositives(t *testing.T) {
	dir := t.TempDir()
	// package.json with "pg" appearing in description/URLs but NOT as a dependency
	pkg := `{
		"name": "my-upgrading-app",
		"description": "Upgrading the homepage",
		"homepage": "https://example.com/page",
		"dependencies": {
			"express": "^4.0.0"
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	svc := DetectedService{
		Name:     "test",
		Language: "js",
		Project:  ".",
	}

	deps := detectServiceDependencies(svc, dir)
	assert.NotContains(t, deps, "postgres", "should not false-positive detect postgres from 'upgrading' or 'homepage'")
}
