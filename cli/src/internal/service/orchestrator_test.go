package service

import (
	"fmt"
	"testing"
	"time"
)

func TestOrchestrationResult(t *testing.T) {
	result := &OrchestrationResult{
		Processes: make(map[string]*ServiceProcess),
		Errors:    make(map[string]error),
		StartTime: time.Now(),
	}

	if result.Processes == nil {
		t.Error("Processes map should be initialized")
	}
	if result.Errors == nil {
		t.Error("Errors map should be initialized")
	}
}

func TestGetServiceURLs(t *testing.T) {
	processes := map[string]*ServiceProcess{
		"api": {
			Name:  "api",
			Port:  3000,
			URL:   "http://localhost:3000",
			Ready: true,
		},
		"web": {
			Name:  "web",
			Port:  3001,
			URL:   "http://localhost:3001",
			Ready: true,
		},
	}

	urls := GetServiceURLs(processes)

	if len(urls) != 2 {
		t.Errorf("GetServiceURLs returned %d URLs, want 2", len(urls))
	}

	expectedURLs := map[string]string{
		"api": "http://localhost:3000",
		"web": "http://localhost:3001",
	}

	for name, expected := range expectedURLs {
		if urls[name] != expected {
			t.Errorf("GetServiceURLs()[%q] = %q, want %q", name, urls[name], expected)
		}
	}
}

func TestValidateOrchestration(t *testing.T) {
	tests := []struct {
		name    string
		result  *OrchestrationResult
		wantErr bool
	}{
		{
			name: "all services ready",
			result: &OrchestrationResult{
				Processes: map[string]*ServiceProcess{
					"api": {Ready: true},
					"web": {Ready: true},
				},
				Errors: map[string]error{},
			},
			wantErr: false,
		},
		{
			name: "some services not ready",
			result: &OrchestrationResult{
				Processes: map[string]*ServiceProcess{
					"api": {Ready: false},
					"web": {Ready: true},
				},
				Errors: map[string]error{},
			},
			wantErr: true,
		},
		{
			name: "service has generic error",
			result: &OrchestrationResult{
				Processes: map[string]*ServiceProcess{
					"api": {Ready: true},
				},
				Errors: map[string]error{
					"web": fmt.Errorf("failed to start"),
				},
			},
			wantErr: true,
		},
		{
			name: "empty result",
			result: &OrchestrationResult{
				Processes: map[string]*ServiceProcess{},
				Errors:    map[string]error{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrchestration(tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrchestration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStopAllServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with empty processes - should not panic
	processes := map[string]*ServiceProcess{}
	StopAllServices(processes)

	// Test with nil process - should not panic
	processes2 := map[string]*ServiceProcess{
		"api": nil,
	}
	StopAllServices(processes2)
}

// TestWaitForServices removed - WaitForServices function removed in favor of errgroup-based coordination
// Each service is now monitored in its own goroutine within monitorServicesUntilShutdown

func TestTopologicalSort_NoDependencies(t *testing.T) {
	// Services with no dependencies should all be in level 0
	services := map[string]Service{
		"web":   {Host: "containerapp"},
		"api":   {Host: "containerapp"},
		"redis": {Host: "containerapp"},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	levels := TopologicalSort(graph)

	if len(levels) != 1 {
		t.Errorf("Expected 1 level, got %d", len(levels))
	}

	if len(levels[0]) != 3 {
		t.Errorf("Expected 3 services in level 0, got %d", len(levels[0]))
	}
}

func TestTopologicalSort_LinearDependency(t *testing.T) {
	// Linear chain: frontend -> api -> db
	services := map[string]Service{
		"frontend": {Host: "containerapp", Uses: []string{"api"}},
		"api":      {Host: "containerapp", Uses: []string{"db"}},
		"db":       {Host: "containerapp"},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	levels := TopologicalSort(graph)

	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Level 0: db (no deps)
	if len(levels[0]) != 1 || levels[0][0] != "db" {
		t.Errorf("Expected level 0 to be [db], got %v", levels[0])
	}

	// Level 1: api (depends on db)
	if len(levels[1]) != 1 || levels[1][0] != "api" {
		t.Errorf("Expected level 1 to be [api], got %v", levels[1])
	}

	// Level 2: frontend (depends on api)
	if len(levels[2]) != 1 || levels[2][0] != "frontend" {
		t.Errorf("Expected level 2 to be [frontend], got %v", levels[2])
	}
}

func TestTopologicalSort_DiamondDependency(t *testing.T) {
	// Diamond pattern:
	//     frontend
	//     /      \
	//   api    worker
	//     \      /
	//       db
	services := map[string]Service{
		"frontend": {Host: "containerapp", Uses: []string{"api", "worker"}},
		"api":      {Host: "containerapp", Uses: []string{"db"}},
		"worker":   {Host: "containerapp", Uses: []string{"db"}},
		"db":       {Host: "containerapp"},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	levels := TopologicalSort(graph)

	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Level 0: db
	if len(levels[0]) != 1 || levels[0][0] != "db" {
		t.Errorf("Expected level 0 to be [db], got %v", levels[0])
	}

	// Level 1: api and worker (both depend only on db)
	if len(levels[1]) != 2 {
		t.Errorf("Expected level 1 to have 2 services, got %d", len(levels[1]))
	}

	// Level 2: frontend
	if len(levels[2]) != 1 || levels[2][0] != "frontend" {
		t.Errorf("Expected level 2 to be [frontend], got %v", levels[2])
	}
}

func TestTopologicalSort_ContainerDependencies(t *testing.T) {
	// Simulates containers-test azure.yaml pattern:
	// api depends on: azurite, cosmos, redis, postgres
	services := map[string]Service{
		"azurite":  {Image: "mcr.microsoft.com/azure-storage/azurite:latest"},
		"cosmos":   {Image: "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest"},
		"redis":    {Image: "redis:7-alpine"},
		"postgres": {Image: "postgres:16-alpine"},
		"api":      {Project: "./api", Language: "javascript", Uses: []string{"azurite", "cosmos", "redis", "postgres"}},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	levels := TopologicalSort(graph)

	if len(levels) != 2 {
		t.Errorf("Expected 2 levels, got %d", len(levels))
	}

	// Level 0: all containers (no deps) - 4 services
	if len(levels[0]) != 4 {
		t.Errorf("Expected level 0 to have 4 container services, got %d", len(levels[0]))
	}

	// Level 1: api (depends on all containers)
	if len(levels[1]) != 1 || levels[1][0] != "api" {
		t.Errorf("Expected level 1 to be [api], got %v", levels[1])
	}
}

func TestTopologicalSort_MixedDependencies(t *testing.T) {
	// Mix of services with different dependency depths
	services := map[string]Service{
		"standalone": {Host: "containerapp"}, // no deps
		"db":         {Host: "containerapp"}, // no deps
		"cache":      {Host: "containerapp"}, // no deps
		"api":        {Host: "containerapp", Uses: []string{"db", "cache"}},
		"worker":     {Host: "containerapp", Uses: []string{"db"}},
		"frontend":   {Host: "containerapp", Uses: []string{"api"}},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	levels := TopologicalSort(graph)

	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Level 0: standalone, db, cache (no deps) - 3 services
	if len(levels[0]) != 3 {
		t.Errorf("Expected level 0 to have 3 services, got %d: %v", len(levels[0]), levels[0])
	}

	// Level 1: api, worker (depend on level 0 services) - 2 services
	if len(levels[1]) != 2 {
		t.Errorf("Expected level 1 to have 2 services, got %d: %v", len(levels[1]), levels[1])
	}

	// Level 2: frontend (depends on api)
	if len(levels[2]) != 1 || levels[2][0] != "frontend" {
		t.Errorf("Expected level 2 to be [frontend], got %v", levels[2])
	}
}

func TestTopologicalSort_EmptyServices(t *testing.T) {
	// Empty services graph - edge case
	// BuildDependencyGraph may return an error for empty input
	services := map[string]Service{}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		// It's acceptable for BuildDependencyGraph to fail on empty input
		t.Logf("BuildDependencyGraph returned error for empty services: %v", err)
		return
	}

	levels := TopologicalSort(graph)

	if len(levels) != 0 {
		t.Errorf("Expected 0 levels for empty services, got %d", len(levels))
	}
}

func TestWaitForServiceHealthy_HealthCheckDisabled(t *testing.T) {
	// Service with health check disabled should return immediately
	svc := &Service{
		Healthcheck: &HealthcheckConfig{Disable: true},
	}

	process := &ServiceProcess{
		Name: "test-service",
		Runtime: ServiceRuntime{
			Name: "test-service",
			HealthCheck: HealthCheckConfig{
				Type: "http",
			},
		},
	}

	err := waitForServiceHealthy("test-service", process, svc, DefaultHealthWaitTimeout)
	if err != nil {
		t.Errorf("Expected no error for disabled health check, got: %v", err)
	}
}

func TestWaitForServiceHealthy_HealthCheckTypeNone(t *testing.T) {
	// Service with healthcheck type "none" should return immediately
	svc := &Service{
		Healthcheck: &HealthcheckConfig{Type: "none"},
	}

	process := &ServiceProcess{
		Name: "test-service",
		Runtime: ServiceRuntime{
			Name: "test-service",
			HealthCheck: HealthCheckConfig{
				Type: "none",
			},
		},
	}

	err := waitForServiceHealthy("test-service", process, svc, DefaultHealthWaitTimeout)
	if err != nil {
		t.Errorf("Expected no error for type=none health check, got: %v", err)
	}
}

func TestGetServiceDependencies(t *testing.T) {
	services := map[string]Service{
		"frontend": {Uses: []string{"api", "worker"}},
		"api":      {Uses: []string{"db"}},
		"worker":   {Uses: []string{"db"}},
		"db":       {},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test frontend dependencies
	frontendDeps := GetServiceDependencies("frontend", graph)
	if len(frontendDeps) != 2 {
		t.Errorf("Expected frontend to have 2 deps, got %d", len(frontendDeps))
	}

	// Test db has no dependencies
	dbDeps := GetServiceDependencies("db", graph)
	if len(dbDeps) != 0 {
		t.Errorf("Expected db to have 0 deps, got %d", len(dbDeps))
	}

	// Test nonexistent service
	noDeps := GetServiceDependencies("nonexistent", graph)
	if len(noDeps) != 0 {
		t.Errorf("Expected nonexistent service to have 0 deps, got %d", len(noDeps))
	}
}

func TestGetDependents(t *testing.T) {
	services := map[string]Service{
		"frontend": {Uses: []string{"api"}},
		"api":      {Uses: []string{"db"}},
		"worker":   {Uses: []string{"db"}},
		"db":       {},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test db dependents (api and worker depend on db)
	dbDependents := GetDependents("db", graph)
	if len(dbDependents) != 2 {
		t.Errorf("Expected db to have 2 dependents, got %d: %v", len(dbDependents), dbDependents)
	}

	// Test api dependents (frontend depends on api)
	apiDependents := GetDependents("api", graph)
	if len(apiDependents) != 1 || apiDependents[0] != "frontend" {
		t.Errorf("Expected api dependents to be [frontend], got %v", apiDependents)
	}

	// Test frontend has no dependents
	frontendDependents := GetDependents("frontend", graph)
	if len(frontendDependents) != 0 {
		t.Errorf("Expected frontend to have 0 dependents, got %d", len(frontendDependents))
	}
}

func TestFilterGraphByServices(t *testing.T) {
	services := map[string]Service{
		"frontend": {Uses: []string{"api"}},
		"api":      {Uses: []string{"db"}},
		"worker":   {Uses: []string{"db"}},
		"db":       {},
		"cache":    {},
	}

	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Filter to just frontend - should include frontend, api, and db (transitive deps)
	filtered, err := FilterGraphByServices(graph, []string{"frontend"})
	if err != nil {
		t.Fatalf("Failed to filter graph: %v", err)
	}

	if len(filtered.Nodes) != 3 {
		t.Errorf("Expected 3 nodes (frontend, api, db), got %d", len(filtered.Nodes))
	}

	// Verify cache and worker are not included
	if _, exists := filtered.Nodes["cache"]; exists {
		t.Error("cache should not be in filtered graph")
	}
	if _, exists := filtered.Nodes["worker"]; exists {
		t.Error("worker should not be in filtered graph")
	}
}
