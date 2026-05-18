package service

import (
	"sort"
	"strings"
	"testing"
)

func TestBuildDependencyGraph(t *testing.T) {
	tests := []struct {
		name      string
		services  map[string]Service
		resources map[string]Resource
		wantErr   bool
		errMsg    string
	}{
		{
			name: "linear dependencies",
			services: map[string]Service{
				"api":      {Uses: []string{"db"}},
				"frontend": {Uses: []string{"api"}},
			},
			resources: map[string]Resource{
				"db": {},
			},
			wantErr: false,
		},
		{
			name: "no dependencies",
			services: map[string]Service{
				"api": {},
				"web": {},
			},
			resources: map[string]Resource{},
			wantErr:   false,
		},
		{
			name: "resource dependencies",
			services: map[string]Service{
				"api": {Uses: []string{"cache", "db"}},
			},
			resources: map[string]Resource{
				"cache": {},
				"db":    {},
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			services: map[string]Service{
				"api": {Uses: []string{"nonexistent"}},
			},
			resources: map[string]Resource{},
			wantErr:   true,
			errMsg:    "does not exist",
		},
		{
			name: "cycle detection",
			services: map[string]Service{
				"a": {Uses: []string{"b"}},
				"b": {Uses: []string{"a"}},
			},
			resources: map[string]Resource{},
			wantErr:   true,
			errMsg:    "circular dependency",
		},
		{
			name: "self-reference cycle",
			services: map[string]Service{
				"a": {Uses: []string{"a"}},
			},
			resources: map[string]Resource{},
			wantErr:   true,
			errMsg:    "circular dependency",
		},
		{
			name: "diamond pattern (no cycle)",
			services: map[string]Service{
				"top":    {Uses: []string{"left", "right"}},
				"left":   {Uses: []string{"bottom"}},
				"right":  {Uses: []string{"bottom"}},
				"bottom": {},
			},
			resources: map[string]Resource{},
			wantErr:   false,
		},
		{
			name:      "empty graph",
			services:  map[string]Service{},
			resources: map[string]Resource{},
			wantErr:   true,
			errMsg:    "failed to calculate dependency levels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := BuildDependencyGraph(tt.services, tt.resources)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedNodes := len(tt.services) + len(tt.resources)
			if len(graph.Nodes) != expectedNodes {
				t.Errorf("got %d nodes, want %d", len(graph.Nodes), expectedNodes)
			}
		})
	}
}

func TestBuildDependencyGraph_Levels(t *testing.T) {
	services := map[string]Service{
		"frontend": {Uses: []string{"api"}},
		"api":      {Uses: []string{"db"}},
	}
	resources := map[string]Resource{
		"db": {},
	}

	graph, err := BuildDependencyGraph(services, resources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// db = level 0, api = level 1, frontend = level 2
	if graph.Nodes["db"].Level != 0 {
		t.Errorf("db level = %d, want 0", graph.Nodes["db"].Level)
	}
	if graph.Nodes["api"].Level != 1 {
		t.Errorf("api level = %d, want 1", graph.Nodes["api"].Level)
	}
	if graph.Nodes["frontend"].Level != 2 {
		t.Errorf("frontend level = %d, want 2", graph.Nodes["frontend"].Level)
	}
}

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name    string
		graph   *DependencyGraph
		wantErr bool
	}{
		{
			name: "no cycle",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{"a": {}, "b": {}},
				Edges: map[string][]string{"a": {"b"}, "b": {}},
			},
			wantErr: false,
		},
		{
			name: "has cycle",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{"a": {}, "b": {}},
				Edges: map[string][]string{"a": {"b"}, "b": {"a"}},
			},
			wantErr: true,
		},
		{
			name: "disconnected no cycle",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{"a": {}, "b": {}, "c": {}},
				Edges: map[string][]string{"a": {}, "b": {}, "c": {}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetectCycles(tt.graph)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectCycles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	// Build a graph with known levels
	services := map[string]Service{
		"frontend": {Uses: []string{"api"}},
		"api":      {Uses: []string{"db"}},
		"worker":   {},
	}
	resources := map[string]Resource{
		"db": {},
	}

	graph, err := BuildDependencyGraph(services, resources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	levels := TopologicalSort(graph)

	// Level 0: worker (db is a resource, excluded)
	// Level 1: api
	// Level 2: frontend
	if len(levels) != 3 {
		t.Fatalf("got %d levels, want 3", len(levels))
	}

	if !sliceContains(levels[0], "worker") {
		t.Errorf("level 0 should contain 'worker', got %v", levels[0])
	}
	if !sliceContains(levels[1], "api") {
		t.Errorf("level 1 should contain 'api', got %v", levels[1])
	}
	if !sliceContains(levels[2], "frontend") {
		t.Errorf("level 2 should contain 'frontend', got %v", levels[2])
	}
}

func TestTopologicalSort_ResourcesExcluded(t *testing.T) {
	services := map[string]Service{
		"api": {Uses: []string{"db"}},
	}
	resources := map[string]Resource{
		"db": {},
	}

	graph, err := BuildDependencyGraph(services, resources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	levels := TopologicalSort(graph)

	// Only services should appear in sorted output
	total := 0
	for _, level := range levels {
		for _, name := range level {
			total++
			if name == "db" {
				t.Error("resources should be excluded from TopologicalSort")
			}
		}
	}
	if total != 1 {
		t.Errorf("expected 1 service in output, got %d", total)
	}
}

func TestGraph_GetServiceDependencies(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{"api": {}, "db": {}},
		Edges: map[string][]string{"api": {"db"}, "db": {}},
	}

	deps := GetServiceDependencies("api", graph)
	if len(deps) != 1 || deps[0] != "db" {
		t.Errorf("got %v, want [db]", deps)
	}

	deps = GetServiceDependencies("nonexistent", graph)
	if len(deps) != 0 {
		t.Errorf("got %v, want empty", deps)
	}
}

func TestGraph_GetDependents(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{"api": {}, "frontend": {}, "worker": {}},
		Edges: map[string][]string{
			"api":      {},
			"frontend": {"api"},
			"worker":   {"api"},
		},
	}

	dependents := GetDependents("api", graph)
	sort.Strings(dependents)
	if len(dependents) != 2 || dependents[0] != "frontend" || dependents[1] != "worker" {
		t.Errorf("got %v, want [frontend worker]", dependents)
	}
}

func TestGraph_FilterGraphByServices(t *testing.T) {
	services := map[string]Service{
		"frontend": {Uses: []string{"api"}},
		"api":      {Uses: []string{"db"}},
		"worker":   {},
	}
	resources := map[string]Resource{
		"db": {},
	}

	graph, err := BuildDependencyGraph(services, resources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filtered, err := FilterGraphByServices(graph, []string{"frontend"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include frontend, api, and db (transitive dep)
	if len(filtered.Nodes) != 3 {
		t.Errorf("got %d nodes, want 3 (frontend+api+db)", len(filtered.Nodes))
	}
	if _, ok := filtered.Nodes["worker"]; ok {
		t.Error("worker should not be in filtered graph")
	}
}

func TestGraph_FilterGraphByServices_NotFound(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{"api": {}},
		Edges: map[string][]string{"api": {}},
	}

	_, err := FilterGraphByServices(graph, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}

// helpers
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
