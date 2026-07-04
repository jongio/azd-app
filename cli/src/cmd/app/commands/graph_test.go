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

func sampleGraphResult() graphResult {
	return graphResult{
		Project: "project",
		Nodes: []graphNode{
			{Name: "api", Type: "service", Level: 1, Language: "go", Host: "local"},
			{Name: "db", Type: "resource", Level: 0},
		},
		Edges:  []graphEdge{{From: "api", To: "db"}},
		Levels: [][]string{{"db"}, {"api"}},
	}
}

func TestRenderGraphMermaid(t *testing.T) {
	var buf bytes.Buffer
	renderGraphMermaid(&buf, sampleGraphResult())
	out := buf.String()

	if !strings.HasPrefix(out, "flowchart TD") {
		t.Fatalf("mermaid output should start with flowchart TD:\n%s", out)
	}
	// Service node uses square brackets, resource uses rounded shape.
	if !strings.Contains(out, "[\"api (service)\"]") {
		t.Fatalf("mermaid missing service node:\n%s", out)
	}
	if !strings.Contains(out, "([\"db (resource)\"])") {
		t.Fatalf("mermaid missing resource node:\n%s", out)
	}
	// Edge should reference the generated node ids, not raw names.
	if !strings.Contains(out, "-->") {
		t.Fatalf("mermaid missing edge:\n%s", out)
	}
}

func TestRenderGraphDOT(t *testing.T) {
	var buf bytes.Buffer
	renderGraphDOT(&buf, sampleGraphResult())
	out := buf.String()

	for _, want := range []string{
		"digraph services {",
		"rankdir=LR;",
		"\"api\" [label=\"api\\n(service)\", shape=box];",
		"\"db\" [label=\"db\\n(resource)\", shape=ellipse];",
		"\"api\" -> \"db\";",
		"}",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dot output missing %q:\n%s", want, out)
		}
	}
}

func TestEscapeMermaidLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"say \"hi\"", "say #quot;hi#quot;"},
		{"a[b]", "a#91;b#93;"},
		{"a{b}", "a#123;b#125;"},
		{"line\nbreak", "line break"},
	}
	for _, tt := range tests {
		if got := escapeMermaidLabel(tt.in); got != tt.want {
			t.Errorf("escapeMermaidLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeDOTString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"say \"hi\"", "say \\\"hi\\\""},
		{"back\\slash", "back\\\\slash"},
		{"line\nbreak", "line\\nbreak"},
	}
	for _, tt := range tests {
		if got := escapeDOTString(tt.in); got != tt.want {
			t.Errorf("escapeDOTString(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRunGraphInvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: "svg", writer: &buf})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGraphOutputFile(t *testing.T) {
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

	outFile := filepath.Join(dir, "graph.mmd")
	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: graphOutputMermaid, outputFile: outFile, writer: &buf})
	if err != nil {
		t.Fatalf("runGraph failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Wrote mermaid graph to") {
		t.Fatalf("stdout should confirm file write, got: %s", buf.String())
	}
	content, err := os.ReadFile(outFile) //nolint:gosec // path is a test temp file
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.HasPrefix(string(content), "flowchart TD") {
		t.Fatalf("output file should contain mermaid graph:\n%s", string(content))
	}
}
