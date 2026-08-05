package commands

import (
	"errors"

	"bytes"
	"encoding/json"
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
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
	if cmd.Flags().Lookup("focus") == nil {
		t.Fatal("focus flag not found")
	}
	if cmd.Flags().Lookup("services-only") == nil {
		t.Fatal("services-only flag not found")
	}
}

func focusSampleGraph() graphResult {
	return graphResult{
		Project: "project",
		Nodes: []graphNode{
			{Name: "api", Type: "service", Level: 1},
			{Name: "db", Type: "resource", Level: 0},
			{Name: "web", Type: "service", Level: 2},
			{Name: "worker", Type: "service", Level: 1},
		},
		Edges: []graphEdge{
			{From: "api", To: "db"},
			{From: "web", To: "api"},
			{From: "worker", To: "db"},
		},
		Levels: [][]string{{"db"}, {"api", "worker"}, {"web"}},
	}
}

func nodeNameSet(nodes []graphNode) map[string]bool {
	set := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		set[n.Name] = true
	}
	return set
}

func TestFocusGraphResult(t *testing.T) {
	t.Run("focus keeps dependencies and dependents", func(t *testing.T) {
		got, err := focusGraphResult(focusSampleGraph(), "api")
		if err != nil {
			t.Fatalf("focusGraphResult failed: %v", err)
		}
		names := nodeNameSet(got.Nodes)
		for _, want := range []string{"api", "db", "web"} {
			if !names[want] {
				t.Fatalf("expected %q in focused nodes, got %#v", want, got.Nodes)
			}
		}
		if names["worker"] {
			t.Fatalf("worker should be excluded when focusing api, got %#v", got.Nodes)
		}
		// Only edges between kept nodes survive: api->db and web->api.
		if len(got.Edges) != 2 {
			t.Fatalf("edges = %d, want 2: %#v", len(got.Edges), got.Edges)
		}
		for _, e := range got.Edges {
			if e.From == "worker" || e.To == "worker" {
				t.Fatalf("worker edge should be dropped: %#v", e)
			}
		}
	})

	t.Run("focus on a leaf resource pulls in every dependent", func(t *testing.T) {
		got, err := focusGraphResult(focusSampleGraph(), "db")
		if err != nil {
			t.Fatalf("focusGraphResult failed: %v", err)
		}
		if len(got.Nodes) != 4 {
			t.Fatalf("nodes = %d, want 4 (db reaches every dependent): %#v", len(got.Nodes), got.Nodes)
		}
	})

	t.Run("empty levels are dropped", func(t *testing.T) {
		got, err := focusGraphResult(focusSampleGraph(), "worker")
		if err != nil {
			t.Fatalf("focusGraphResult failed: %v", err)
		}
		for _, level := range got.Levels {
			if len(level) == 0 {
				t.Fatalf("focused levels should not contain empty entries: %#v", got.Levels)
			}
		}
	})

	t.Run("unknown focus errors and lists available", func(t *testing.T) {
		_, err := focusGraphResult(focusSampleGraph(), "nope")
		if err == nil {
			t.Fatal("expected error for unknown focus")
		}
		var local *azdext.LocalError
		if !errors.As(err, &local) {
			t.Fatalf("expected *azdext.LocalError, got %T: %v", err, err)
		}
		if local.Code != ErrCodeServiceNotFound {
			t.Fatalf("Code = %q, want %q", local.Code, ErrCodeServiceNotFound)
		}
		if !strings.Contains(local.Suggestion, "api") {
			t.Fatalf("suggestion should list available nodes: %v", local.Suggestion)
		}
	})
}

func TestFilterGraphServicesOnly(t *testing.T) {
	result := filterGraphServicesOnly(focusSampleGraph())

	names := nodeNameSet(result.Nodes)
	for _, want := range []string{"api", "web", "worker"} {
		if !names[want] {
			t.Fatalf("expected service %q in graph, got %#v", want, result.Nodes)
		}
	}
	if names["db"] {
		t.Fatalf("resource db should be excluded, got %#v", result.Nodes)
	}
	if len(result.Edges) != 1 || result.Edges[0] != (graphEdge{From: "web", To: "api"}) {
		t.Fatalf("edges = %#v, want only web -> api", result.Edges)
	}
	for _, level := range result.Levels {
		for _, name := range level {
			if name == "db" {
				t.Fatalf("resource db should be excluded from levels: %#v", result.Levels)
			}
		}
	}
}

func TestRunGraphFocus(t *testing.T) {
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
  web:
    host: local
    language: node
    project: ./web
    uses:
      - api
  lonely:
    host: local
    language: go
    project: ./lonely
resources:
  db:
    type: postgres
`)
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), azureYaml, 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}

	for _, sub := range []string{"api", "web", "lonely"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	t.Chdir(dir)

	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: graphOutputJSON, focus: "api", writer: &buf})
	if err != nil {
		t.Fatalf("runGraph failed: %v", err)
	}
	var got graphResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	names := nodeNameSet(got.Nodes)
	if !names["api"] || !names["db"] || !names["web"] {
		t.Fatalf("focused graph should contain api, db, web: %#v", got.Nodes)
	}
	if names["lonely"] {
		t.Fatalf("lonely service should be excluded when focusing api: %#v", got.Nodes)
	}
}

func TestRunGraphServicesOnly(t *testing.T) {
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
  web:
    host: local
    language: node
    project: ./web
    uses:
      - api
resources:
  db:
    type: postgres
`)
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), azureYaml, 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	for _, sub := range []string{"api", "web"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	t.Chdir(dir)

	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: graphOutputJSON, servicesOnly: true, writer: &buf})
	if err != nil {
		t.Fatalf("runGraph failed: %v", err)
	}
	var got graphResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	names := nodeNameSet(got.Nodes)
	if !names["api"] || !names["web"] {
		t.Fatalf("services-only graph should contain services: %#v", got.Nodes)
	}
	if names["db"] {
		t.Fatalf("services-only graph should exclude resources: %#v", got.Nodes)
	}
	if len(got.Edges) != 1 || got.Edges[0] != (graphEdge{From: "web", To: "api"}) {
		t.Fatalf("edges = %#v, want only web -> api", got.Edges)
	}
}

func TestRunGraphFocusUnknown(t *testing.T) {
	dir := t.TempDir()
	azureYaml := []byte(`
name: graph-test
services:
  api:
    host: local
    language: go
    project: ./api
`)
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), azureYaml, 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "api"), 0o750); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	t.Chdir(dir)

	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: graphOutputText, focus: "missing", writer: &buf})
	if err == nil {
		t.Fatal("expected error for unknown focus service")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
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

func TestRenderGraphPlantUML(t *testing.T) {
	var buf bytes.Buffer
	renderGraphPlantUML(&buf, sampleGraphResult())
	out := buf.String()

	if !strings.HasPrefix(out, "@startuml") {
		t.Fatalf("plantuml output should start with @startuml:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "@enduml") {
		t.Fatalf("plantuml output should end with @enduml:\n%s", out)
	}
	for _, want := range []string{
		"title project",
		// Service nodes use component, resources use database.
		"component \"api (service)\" as ",
		"database \"db (resource)\" as ",
		" --> ",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plantuml output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderGraphPlantUMLNoProject(t *testing.T) {
	result := sampleGraphResult()
	result.Project = ""
	var buf bytes.Buffer
	renderGraphPlantUML(&buf, result)
	if strings.Contains(buf.String(), "title") {
		t.Fatalf("plantuml output should omit title when project is empty:\n%s", buf.String())
	}
}

func TestRenderGraphMarkdown(t *testing.T) {
	var buf bytes.Buffer
	renderGraphMarkdown(&buf, sampleGraphResult())
	out := buf.String()

	for _, want := range []string{
		"# Dependency graph for `project`",
		"## Startup levels",
		"| Level | Nodes |",
		"| 0 | db |",
		"## Nodes",
		"| api | service | 1 | local | go | n/a |",
		"| db | resource | 0 | n/a | n/a | n/a |",
		"## Dependencies",
		"| api | db |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, out)
		}
	}
}

func TestRunGraphMarkdownOutputFile(t *testing.T) {
	dir := t.TempDir()
	azureYaml := []byte(`
name: graph-test
services:
  api:
    host: local
    language: go
    project: ./api
`)
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), azureYaml, 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "api"), 0o750); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	t.Chdir(dir)

	outputFile := filepath.Join(dir, "graph.md")
	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: graphOutputMarkdown, outputFile: outputFile, writer: &buf})
	if err != nil {
		t.Fatalf("runGraph failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Wrote markdown graph") {
		t.Fatalf("expected output-file confirmation, got:\n%s", buf.String())
	}
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(data), "# Dependency graph") {
		t.Fatalf("markdown output file missing heading:\n%s", string(data))
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

func TestRenderGraphD2(t *testing.T) {
	var buf bytes.Buffer
	renderGraphD2(&buf, sampleGraphResult())
	out := buf.String()

	if !strings.HasPrefix(out, "direction: right") {
		t.Fatalf("d2 output should start with direction: right:\n%s", out)
	}
	for _, want := range []string{
		"napi_0: \"api (service)\"",
		"ndb_1: \"db (resource)\" {",
		"shape: cylinder",
		"napi_0 -> ndb_1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("d2 output missing %q:\n%s", want, out)
		}
	}
	// Service nodes must not carry a shape block.
	if strings.Contains(out, "napi_0: \"api (service)\" {") {
		t.Fatalf("service node should not have a shape block:\n%s", out)
	}
}

func TestRenderGraphD2Dispatch(t *testing.T) {
	var buf bytes.Buffer
	if err := renderGraph(&buf, graphOutputD2, sampleGraphResult()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "napi_0 -> ndb_1") {
		t.Fatalf("renderGraph did not dispatch to d2:\n%s", buf.String())
	}
}

func TestEscapeD2Label(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"say \"hi\"", "say 'hi'"},
		{"line\nbreak", "line break"},
	}
	for _, tt := range tests {
		if got := escapeD2Label(tt.in); got != tt.want {
			t.Errorf("escapeD2Label(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapePlantUMLLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"say \"hi\"", "say 'hi'"},
		{"line\nbreak", "line\\nbreak"},
	}
	for _, tt := range tests {
		if got := escapePlantUMLLabel(tt.in); got != tt.want {
			t.Errorf("escapePlantUMLLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRunGraphInvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	err := runGraph(&graphOptions{output: "svg", writer: &buf})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid --output value") {
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
