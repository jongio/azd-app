package service

import (
	"strings"
	"testing"
)

func TestNewFunctionsOutputParser(t *testing.T) {
	parser := NewFunctionsOutputParser(false)
	if parser == nil {
		t.Fatal("NewFunctionsOutputParser() returned nil")
	}
	if parser.endpoints == nil {
		t.Error("NewFunctionsOutputParser() did not initialize endpoints map")
	}
	if parser.verbose != false {
		t.Error("NewFunctionsOutputParser() verbose flag not set correctly")
	}

	parserVerbose := NewFunctionsOutputParser(true)
	if parserVerbose.verbose != true {
		t.Error("NewFunctionsOutputParser(true) verbose flag not set correctly")
	}
}

func TestParseLine_HTTPTrigger(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		expectEndpoint  bool
		expectedName    string
		expectedMethods []string
		expectedRoute   string
	}{
		{
			name:            "HTTP trigger with GET",
			line:            "  GetUser: [GET] http://localhost:7071/api/users/{id}",
			expectEndpoint:  true,
			expectedName:    "GetUser",
			expectedMethods: []string{"GET"},
			expectedRoute:   "api/users/{id}",
		},
		{
			name:            "HTTP trigger with multiple methods",
			line:            "  CreateUser: [POST, PUT] http://localhost:7071/api/users",
			expectEndpoint:  true,
			expectedName:    "CreateUser",
			expectedMethods: []string{"POST", "PUT"},
			expectedRoute:   "api/users",
		},
		{
			name:            "Simple HTTP pattern",
			line:            "  HelloWorld: http://localhost:7071/api/hello",
			expectEndpoint:  true,
			expectedName:    "HelloWorld",
			expectedMethods: []string{"GET", "POST"},
			expectedRoute:   "api/hello",
		},
		{
			name:           "Non-matching line",
			line:           "Starting host...",
			expectEndpoint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewFunctionsOutputParser(false)
			parser.ParseLine("test-service", tt.line)

			endpoints := parser.GetEndpoints("test-service")

			if tt.expectEndpoint {
				if len(endpoints) == 0 {
					t.Fatal("Expected endpoint to be parsed, but got none")
				}
				endpoint := endpoints[0]
				if endpoint.Name != tt.expectedName {
					t.Errorf("Endpoint.Name = %v, want %v", endpoint.Name, tt.expectedName)
				}
				if endpoint.TriggerType != "HTTP" {
					t.Errorf("Endpoint.TriggerType = %v, want HTTP", endpoint.TriggerType)
				}
				if len(endpoint.Methods) != len(tt.expectedMethods) {
					t.Errorf("Endpoint.Methods count = %v, want %v", len(endpoint.Methods), len(tt.expectedMethods))
				}
				for i, method := range tt.expectedMethods {
					if i >= len(endpoint.Methods) || endpoint.Methods[i] != method {
						t.Errorf("Endpoint.Methods[%d] = %v, want %v", i, endpoint.Methods[i], method)
					}
				}
				if endpoint.Route != tt.expectedRoute {
					t.Errorf("Endpoint.Route = %v, want %v", endpoint.Route, tt.expectedRoute)
				}
			} else {
				if len(endpoints) != 0 {
					t.Errorf("Expected no endpoint to be parsed, but got %d", len(endpoints))
				}
			}
		})
	}
}

func TestParseLine_NonHTTPTriggers(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		expectedTrigger string
		expectedName    string
	}{
		{
			name:            "Timer trigger",
			line:            "  ProcessOrders: [timerTrigger]",
			expectedTrigger: "Timer",
			expectedName:    "ProcessOrders",
		},
		{
			name:            "Queue trigger",
			line:            "  QueueProcessor: [queueTrigger]",
			expectedTrigger: "Queue",
			expectedName:    "QueueProcessor",
		},
		{
			name:            "ServiceBus trigger",
			line:            "  MessageHandler: [serviceBusTrigger]",
			expectedTrigger: "ServiceBus",
			expectedName:    "MessageHandler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewFunctionsOutputParser(false)
			parser.ParseLine("test-service", tt.line)

			endpoints := parser.GetEndpoints("test-service")
			if len(endpoints) == 0 {
				t.Fatal("Expected endpoint to be parsed, but got none")
			}

			endpoint := endpoints[0]
			if endpoint.Name != tt.expectedName {
				t.Errorf("Endpoint.Name = %v, want %v", endpoint.Name, tt.expectedName)
			}
			if endpoint.TriggerType != tt.expectedTrigger {
				t.Errorf("Endpoint.TriggerType = %v, want %v", endpoint.TriggerType, tt.expectedTrigger)
			}
		})
	}
}

func TestParseStream(t *testing.T) {
	output := `
Host Functions:

  GetUser: [GET] http://localhost:7071/api/users/{id}
  CreateUser: [POST] http://localhost:7071/api/users
  ProcessOrders: [timerTrigger]

For detailed output, run func with --verbose flag.
`

	parser := NewFunctionsOutputParser(false)
	reader := strings.NewReader(output)
	parser.ParseStream("test-service", reader)

	endpoints := parser.GetEndpoints("test-service")
	if len(endpoints) != 3 {
		t.Errorf("ParseStream() found %d endpoints, want 3", len(endpoints))
	}

	// Verify specific endpoints
	foundGet := false
	foundPost := false
	foundTimer := false

	for _, ep := range endpoints {
		switch ep.Name {
		case "GetUser":
			foundGet = true
			if ep.TriggerType != "HTTP" {
				t.Errorf("GetUser trigger type = %v, want HTTP", ep.TriggerType)
			}
		case "CreateUser":
			foundPost = true
			if ep.TriggerType != "HTTP" {
				t.Errorf("CreateUser trigger type = %v, want HTTP", ep.TriggerType)
			}
		case "ProcessOrders":
			foundTimer = true
			if ep.TriggerType != "Timer" {
				t.Errorf("ProcessOrders trigger type = %v, want Timer", ep.TriggerType)
			}
		}
	}

	if !foundGet || !foundPost || !foundTimer {
		t.Errorf("ParseStream() missing endpoints: foundGet=%v, foundPost=%v, foundTimer=%v", foundGet, foundPost, foundTimer)
	}
}

func TestAddEndpoint_NoDuplicates(t *testing.T) {
	parser := NewFunctionsOutputParser(false)

	// Add endpoint twice
	parser.ParseLine("test-service", "  GetUser: [GET] http://localhost:7071/api/users")
	parser.ParseLine("test-service", "  GetUser: [GET] http://localhost:7071/api/users")

	endpoints := parser.GetEndpoints("test-service")
	if len(endpoints) != 1 {
		t.Errorf("Expected 1 endpoint after adding duplicate, got %d", len(endpoints))
	}
}

func TestGetEndpoints_MultipleServices(t *testing.T) {
	parser := NewFunctionsOutputParser(false)

	parser.ParseLine("service1", "  Func1: [GET] http://localhost:7071/api/func1")
	parser.ParseLine("service2", "  Func2: [POST] http://localhost:7071/api/func2")

	endpoints1 := parser.GetEndpoints("service1")
	if len(endpoints1) != 1 || endpoints1[0].Name != "Func1" {
		t.Errorf("service1 endpoints incorrect")
	}

	endpoints2 := parser.GetEndpoints("service2")
	if len(endpoints2) != 1 || endpoints2[0].Name != "Func2" {
		t.Errorf("service2 endpoints incorrect")
	}
}

func TestHasEndpoints(t *testing.T) {
	parser := NewFunctionsOutputParser(false)

	if parser.HasEndpoints("test-service") {
		t.Error("HasEndpoints() = true for service with no endpoints")
	}

	parser.ParseLine("test-service", "  Func1: [GET] http://localhost:7071/api/func1")

	if !parser.HasEndpoints("test-service") {
		t.Error("HasEndpoints() = false for service with endpoints")
	}

	if parser.HasEndpoints("other-service") {
		t.Error("HasEndpoints() = true for non-existent service")
	}
}

func TestDisplayEndpoints(t *testing.T) {
	// This test verifies that DisplayEndpoints doesn't panic
	// and handles different endpoint types correctly
	parser := NewFunctionsOutputParser(false)

	// Add various endpoint types
	parser.ParseLine("test-service", "  GetUser: [GET] http://localhost:7071/api/users")
	parser.ParseLine("test-service", "  CreateUser: [POST, PUT] http://localhost:7071/api/users")
	parser.ParseLine("test-service", "  ProcessOrders: [timerTrigger]")
	parser.ParseLine("test-service", "  QueueProcessor: [queueTrigger]")
	parser.ParseLine("test-service", "  MessageHandler: [serviceBusTrigger]")

	// Should not panic
	parser.DisplayEndpoints("test-service", 7071)

	// Test with no endpoints
	parser.DisplayEndpoints("empty-service", 7071)
}

func TestParseHTTPMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single method",
			input:    "GET",
			expected: []string{"GET"},
		},
		{
			name:     "multiple methods",
			input:    "GET,POST,PUT",
			expected: []string{"GET", "POST", "PUT"},
		},
		{
			name:     "methods with spaces",
			input:    "GET, POST, PUT",
			expected: []string{"GET", "POST", "PUT"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHTTPMethods(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseHTTPMethods() returned %d methods, want %d", len(result), len(tt.expected))
				return
			}
			for i, method := range tt.expected {
				if result[i] != method {
					t.Errorf("parseHTTPMethods()[%d] = %v, want %v", i, result[i], method)
				}
			}
		})
	}
}

func TestParseStream_EmptyInput(t *testing.T) {
	parser := NewFunctionsOutputParser(false)
	reader := strings.NewReader("")
	parser.ParseStream("test-service", reader)

	endpoints := parser.GetEndpoints("test-service")
	if len(endpoints) != 0 {
		t.Errorf("ParseStream() with empty input should find 0 endpoints, got %d", len(endpoints))
	}
}

func TestConcurrency(t *testing.T) {
	// Test that the parser is safe for concurrent use
	parser := NewFunctionsOutputParser(false)

	done := make(chan bool)

	// Parse lines concurrently
	go func() {
		for i := 0; i < 100; i++ {
			parser.ParseLine("service1", "  Func1: [GET] http://localhost:7071/api/func1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			parser.ParseLine("service2", "  Func2: [POST] http://localhost:7071/api/func2")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = parser.GetEndpoints("service1")
			_ = parser.HasEndpoints("service2")
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// Verify endpoints were added correctly (should only have 1 of each due to duplicate filtering)
	endpoints1 := parser.GetEndpoints("service1")
	endpoints2 := parser.GetEndpoints("service2")

	if len(endpoints1) != 1 {
		t.Errorf("Expected 1 endpoint for service1, got %d", len(endpoints1))
	}
	if len(endpoints2) != 1 {
		t.Errorf("Expected 1 endpoint for service2, got %d", len(endpoints2))
	}
}
