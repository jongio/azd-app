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

func TestWaitForServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with already ready services
	processes := map[string]*ServiceProcess{
		"api": {
			Name:  "api",
			Ready: true,
		},
	}

	err := WaitForServices(processes)
	if err != nil {
		t.Errorf("WaitForServices() with ready services should not error, got %v", err)
	}
}
