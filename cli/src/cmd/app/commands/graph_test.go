package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestNewGraphCommand(t *testing.T) {
	cmd := NewGraphCommand()
	if cmd == nil {
		t.Fatal("NewGraphCommand returned nil")
	}
	if cmd.Use != "graph" {
		t.Fatalf("Use = %q, want graph", cmd.Use)
	}
	if cmd.Flags().Lookup("output") == nil {
		t.Fatal("output flag not found")
	}
}

func TestBuildGraphResult(t *testing.T) {
	graph, err := service.BuildDependencyGraph(
		map[string]service.Service{
			"api": {Host: "local", Language: "go", Project: "./api", Uses: []string{"db"}},
			"web": {Host: "local", Language: "node", Project: "./web", Uses: []string{"api"}},
		},
		map[string]service.Resource{"db": {Type: "postgres"}},
	)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	result := buildGraphResult("project", graph)
	if len(result.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(result.Nodes))
	}
	if len(result.Edges) != 2 {
		t.Fatalf("edges = %d, want 2", len(result.Edges))
	}
	if result.Edges[0] != (graphEdge{From: "api", To: "db"}) {
		t.Fatalf("first edge = %#v, want api -> db", result.Edges[0])
	}
	if len(result.Levels) != 2 || result.Levels[0][0] != "api" || result.Levels[1][0] != "web" {
		t.Fatalf("levels = %#v, want [[api] [web]]", result.Levels)
	}
}

func TestPrintGraphText(t *testing.T) {
	result := graphResult{
		Project: "project",
		Nodes: []graphNode{
			{Name: "api", Type: "service", Level: 1, Language: "go", Host: "local"},
			{Name: "db", Type: "resource", Level: 0},
		},
		Edges:  []graphEdge{{From: "api", To: "db"}},
		Levels: [][]string{{"api"}},
	}

	var buf bytes.Buffer
	printGraphText(&buf, result)
	out := buf.String()
	for _, want := range []string{"Dependency graph", "Level 0: api", "api -> db", "db (resource"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunGraphJSON(t *testing.T) {
	dir := t.TempDir()
	azureYaml := []byte(`
name: graph-test
services:
  api:
    host: local
    language: go
    project: ./api
    uses:
      - db
resources:
  db:
    type: postgres
`)
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), azureYaml, 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "api"), 0o750); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	t.Chdir(dir)

	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: graphOutputJSON, writer: &buf})
	if err != nil {
		t.Fatalf("runGraph failed: %v", err)
	}
	var got graphResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(got.Nodes) != 2 || len(got.Edges) != 1 {
		t.Fatalf("unexpected graph result: %#v", got)
	}
}
