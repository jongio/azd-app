package service

import (
	"testing"
)

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a greater", 5, 3, 5},
		{"b greater", 2, 7, 7},
		{"equal", 4, 4, 4},
		{"negative numbers", -5, -3, -3},
		{"zero and positive", 0, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name     string
		graph    *DependencyGraph
		expected int // number of levels
	}{
		{
			name: "simple chain",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"a": {Name: "a", Level: 0, IsResource: false},
					"b": {Name: "b", Level: 1, IsResource: false},
					"c": {Name: "c", Level: 2, IsResource: false},
				},
				Edges: map[string][]string{
					"b": {"a"},
					"c": {"b"},
				},
			},
			expected: 3,
		},
		{
			name: "parallel services",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"a": {Name: "a", Level: 0, IsResource: false},
					"b": {Name: "b", Level: 0, IsResource: false},
				},
				Edges: map[string][]string{},
			},
			expected: 1,
		},
		{
			name: "skip resources",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"service": {Name: "service", Level: 0, IsResource: false},
					"db":      {Name: "db", Level: 0, IsResource: true},
				},
				Edges: map[string][]string{
					"service": {"db"},
				},
			},
			expected: 1, // Only service, resource is skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TopologicalSort(tt.graph)
			if len(result) != tt.expected {
				t.Errorf("TopologicalSort() returned %d levels, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestGetServiceDependencies(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"api": {Name: "api"},
			"db":  {Name: "db"},
		},
		Edges: map[string][]string{
			"api": {"db"},
			"db":  {},
		},
	}

	tests := []struct {
		name         string
		serviceName  string
		expectedDeps []string
	}{
		{
			name:         "service with dependencies",
			serviceName:  "api",
			expectedDeps: []string{"db"},
		},
		{
			name:         "service without dependencies",
			serviceName:  "db",
			expectedDeps: []string{},
		},
		{
			name:         "non-existent service",
			serviceName:  "nonexistent",
			expectedDeps: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetServiceDependencies(tt.serviceName, graph)
			if len(result) != len(tt.expectedDeps) {
				t.Errorf("GetServiceDependencies(%q) returned %d deps, want %d", tt.serviceName, len(result), len(tt.expectedDeps))
			}
		})
	}
}

func TestGetDependents(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"api": {Name: "api"},
			"web": {Name: "web"},
			"db":  {Name: "db"},
		},
		Edges: map[string][]string{
			"api": {"db"},
			"web": {"db"},
			"db":  {},
		},
	}

	tests := []struct {
		name              string
		serviceName       string
		expectedDependents int
	}{
		{
			name:               "service with multiple dependents",
			serviceName:        "db",
			expectedDependents: 2, // api and web
		},
		{
			name:               "service with no dependents",
			serviceName:        "api",
			expectedDependents: 0,
		},
		{
			name:               "non-existent service",
			serviceName:        "nonexistent",
			expectedDependents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDependents(tt.serviceName, graph)
			if len(result) != tt.expectedDependents {
				t.Errorf("GetDependents(%q) returned %d dependents, want %d", tt.serviceName, len(result), tt.expectedDependents)
			}
		})
	}
}

func TestFilterGraphByServices(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"api": {Name: "api", Dependencies: []string{"db"}},
			"web": {Name: "web", Dependencies: []string{"api"}},
			"db":  {Name: "db", Dependencies: []string{}},
		},
		Edges: map[string][]string{
			"api": {"db"},
			"web": {"api"},
			"db":  {},
		},
	}

	tests := []struct {
		name          string
		serviceNames  []string
		expectedNodes int
		wantError     bool
	}{
		{
			name:          "single service with deps",
			serviceNames:  []string{"api"},
			expectedNodes: 2, // api and db
			wantError:     false,
		},
		{
			name:          "service chain",
			serviceNames:  []string{"web"},
			expectedNodes: 3, // web, api, and db
			wantError:     false,
		},
		{
			name:          "non-existent service",
			serviceNames:  []string{"nonexistent"},
			expectedNodes: 0,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterGraphByServices(graph, tt.serviceNames)
			
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if len(result.Nodes) != tt.expectedNodes {
				t.Errorf("FilterGraphByServices() returned %d nodes, want %d", len(result.Nodes), tt.expectedNodes)
			}
		})
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	tests := []struct {
		name      string
		services  map[string]Service
		resources map[string]Resource
		wantError bool
	}{
		{
			name: "valid graph",
			services: map[string]Service{
				"api": {Language: "node", Uses: []string{"db"}},
			},
			resources: map[string]Resource{
				"db": {Type: "postgres"},
			},
			wantError: false,
		},
		{
			name: "missing dependency",
			services: map[string]Service{
				"api": {Uses: []string{"nonexistent"}},
			},
			resources: map[string]Resource{},
			wantError: true,
		},
		{
			name: "circular dependency",
			services: map[string]Service{
				"a": {Uses: []string{"b"}},
				"b": {Uses: []string{"a"}},
			},
			resources: map[string]Resource{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := BuildDependencyGraph(tt.services, tt.resources)
			
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if graph == nil {
				t.Error("expected non-nil graph")
			}
		})
	}
}

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name      string
		graph     *DependencyGraph
		wantError bool
	}{
		{
			name: "no cycles",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"a": {Name: "a"},
					"b": {Name: "b"},
				},
				Edges: map[string][]string{
					"a": {"b"},
					"b": {},
				},
			},
			wantError: false,
		},
		{
			name: "simple cycle",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"a": {Name: "a"},
					"b": {Name: "b"},
				},
				Edges: map[string][]string{
					"a": {"b"},
					"b": {"a"},
				},
			},
			wantError: true,
		},
		{
			name: "self-cycle",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"a": {Name: "a"},
				},
				Edges: map[string][]string{
					"a": {"a"},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetectCycles(tt.graph)
			
			if tt.wantError && err == nil {
				t.Error("expected error for cycle detection")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
