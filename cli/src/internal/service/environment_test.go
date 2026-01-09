package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEnvironmentUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    Environment
		wantErr bool
	}{
		{
			name: "map format - Docker Compose standard",
			yaml: `
environment:
  NODE_ENV: production
  PORT: "3000"
  DEBUG: "true"
`,
			want: Environment{
				"NODE_ENV": "production",
				"PORT":     "3000",
				"DEBUG":    "true",
			},
			wantErr: false,
		},
		{
			name: "array of strings - Docker Compose format",
			yaml: `
environment:
  - NODE_ENV=production
  - PORT=3000
  - DEBUG=true
`,
			want: Environment{
				"NODE_ENV": "production",
				"PORT":     "3000",
				"DEBUG":    "true",
			},
			wantErr: false,
		},
		{
			name: "array of objects - legacy format",
			yaml: `
environment:
  - name: NODE_ENV
    value: production
  - name: PORT
    value: "3000"
  - name: DEBUG
    value: "true"
`,
			want: Environment{
				"NODE_ENV": "production",
				"PORT":     "3000",
				"DEBUG":    "true",
			},
			wantErr: false,
		},
		{
			name: "array of objects with secret",
			yaml: `
environment:
  - name: API_KEY
    secret: secret-value
  - name: PUBLIC_KEY
    value: public-value
`,
			want: Environment{
				"API_KEY":    "secret-value",
				"PUBLIC_KEY": "public-value",
			},
			wantErr: false,
		},
		{
			name: "array of objects - secret takes precedence over value",
			yaml: `
environment:
  - name: PASSWORD
    value: should-be-ignored
    secret: actual-secret
`,
			want: Environment{
				"PASSWORD": "actual-secret",
			},
			wantErr: false,
		},
		{
			name: "array of strings with equals in value",
			yaml: `
environment:
  - CONNECTION_STRING=Server=localhost;Password=abc=123
`,
			want: Environment{
				"CONNECTION_STRING": "Server=localhost;Password=abc=123",
			},
			wantErr: false,
		},
		{
			name: "array of strings - key without value",
			yaml: `
environment:
  - EMPTY_VAR
  - HAS_VALUE=something
`,
			want: Environment{
				"EMPTY_VAR": "",
				"HAS_VALUE": "something",
			},
			wantErr: false,
		},
		{
			name: "empty environment",
			yaml: `
environment:
`,
			want:    Environment{},
			wantErr: false,
		},
		{
			name: "map with special characters",
			yaml: `
environment:
  DATABASE_URL: "postgresql://user:pass@localhost:5432/db"
  API_ENDPOINT: "https://api.example.com/v1"
  SPECIAL_CHARS: "!@#$%^&*()"
`,
			want: Environment{
				"DATABASE_URL":  "postgresql://user:pass@localhost:5432/db",
				"API_ENDPOINT":  "https://api.example.com/v1",
				"SPECIAL_CHARS": "!@#$%^&*()",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Environment Environment `yaml:"environment"`
			}

			err := yaml.Unmarshal([]byte(tt.yaml), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(result.Environment) != len(tt.want) {
					t.Errorf("UnmarshalYAML() got %d vars, want %d", len(result.Environment), len(tt.want))
				}

				for key, wantValue := range tt.want {
					if gotValue, ok := result.Environment[key]; !ok {
						t.Errorf("missing key %q", key)
					} else if gotValue != wantValue {
						t.Errorf("key %q = %q, want %q", key, gotValue, wantValue)
					}
				}
			}
		})
	}
}

func TestServiceEnvironmentFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want map[string]string
	}{
		{
			name: "environment field - map format",
			yaml: `
name: test
services:
  api:
    host: localhost
    environment:
      PORT: "3000"
      DEBUG: "true"
`,
			want: map[string]string{
				"PORT":  "3000",
				"DEBUG": "true",
			},
		},
		{
			name: "environment with array format",
			yaml: `
name: test
services:
  api:
    host: localhost
    environment:
      - NODE_ENV=production
      - PORT=3000
`,
			want: map[string]string{
				"NODE_ENV": "production",
				"PORT":     "3000",
			},
		},
		{
			name: "environment with array of objects",
			yaml: `
name: test
services:
  api:
    host: localhost
    environment:
      - name: API_KEY
        value: test-key
      - name: SECRET
        secret: secret-value
`,
			want: map[string]string{
				"API_KEY": "test-key",
				"SECRET":  "secret-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var azureYaml AzureYaml
			err := yaml.Unmarshal([]byte(tt.yaml), &azureYaml)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			api, exists := azureYaml.Services["api"]
			if !exists {
				t.Fatal("service 'api' not found")
			}

			got := api.GetEnvironment()

			if len(got) != len(tt.want) {
				t.Errorf("GetEnvironment() returned %d vars, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				if gotValue, ok := got[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if gotValue != wantValue {
					t.Errorf("key %q = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestEnvironmentMixedFormats(t *testing.T) {
	yamlContent := `
name: fullstack-app
services:
  api:
    host: localhost
    language: python
    ports:
      - "5000"
    environment:
      FLASK_ENV: development
      FLASK_APP: app.py
      DATABASE_URL: postgresql://localhost:5432/db
  
  web:
    host: localhost
    language: node
    ports:
      - "3000"
    environment:
      - NODE_ENV=production
      - API_URL=http://localhost:5000
      - PORT=3000
  
  worker:
    host: localhost
    language: python
    environment:
      - name: QUEUE_URL
        value: redis://localhost:6379
      - name: API_SECRET
        secret: super-secret
`

	var azureYaml AzureYaml
	err := yaml.Unmarshal([]byte(yamlContent), &azureYaml)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Test api service (map format in environment)
	api := azureYaml.Services["api"]
	apiEnv := api.GetEnvironment()
	if apiEnv["FLASK_ENV"] != "development" {
		t.Errorf("api FLASK_ENV = %q, want %q", apiEnv["FLASK_ENV"], "development")
	}
	if apiEnv["DATABASE_URL"] != "postgresql://localhost:5432/db" {
		t.Errorf("api DATABASE_URL = %q, want %q", apiEnv["DATABASE_URL"], "postgresql://localhost:5432/db")
	}

	// Test web service (array of strings in environment)
	web := azureYaml.Services["web"]
	webEnv := web.GetEnvironment()
	if webEnv["NODE_ENV"] != "production" {
		t.Errorf("web NODE_ENV = %q, want %q", webEnv["NODE_ENV"], "production")
	}
	if webEnv["API_URL"] != "http://localhost:5000" {
		t.Errorf("web API_URL = %q, want %q", webEnv["API_URL"], "http://localhost:5000")
	}

	// Test worker service (array of objects in environment)
	worker := azureYaml.Services["worker"]
	workerEnv := worker.GetEnvironment()
	if workerEnv["QUEUE_URL"] != "redis://localhost:6379" {
		t.Errorf("worker QUEUE_URL = %q, want %q", workerEnv["QUEUE_URL"], "redis://localhost:6379")
	}
	if workerEnv["API_SECRET"] != "super-secret" {
		t.Errorf("worker API_SECRET = %q, want %q", workerEnv["API_SECRET"], "super-secret")
	}
}

func TestGetEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		service Service
		want    map[string]string
	}{
		{
			name:    "empty service",
			service: Service{},
			want:    map[string]string{},
		},
		{
			name: "environment field - simple",
			service: Service{
				Environment: Environment{
					"PORT":  "3000",
					"DEBUG": "true",
				},
			},
			want: map[string]string{
				"PORT":  "3000",
				"DEBUG": "true",
			},
		},
		{
			name: "environment field - multiple vars",
			service: Service{
				Environment: Environment{
					"PORT":      "8080",
					"DEBUG":     "false",
					"LOG_LEVEL": "warn",
				},
			},
			want: map[string]string{
				"PORT":      "8080",
				"DEBUG":     "false",
				"LOG_LEVEL": "warn",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.service.GetEnvironment()

			if len(got) != len(tt.want) {
				t.Errorf("GetEnvironment() returned %d vars, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				if gotValue, ok := got[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if gotValue != wantValue {
					t.Errorf("key %q = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestResolveEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		service     Service
		azureEnv    map[string]string
		dotEnvPath  string
		serviceURLs map[string]string
		want        map[string]string
	}{
		{
			name:        "empty service",
			service:     Service{},
			azureEnv:    map[string]string{"AZURE_VAR": "value1"},
			serviceURLs: map[string]string{"SERVICE_URL_API": "http://localhost:3000"},
			want: map[string]string{
				"AZURE_VAR":       "value1",
				"SERVICE_URL_API": "http://localhost:3000",
			},
		},
		{
			name: "service with environment field - map format",
			service: Service{
				Environment: Environment{
					"API_KEY": "test-key",
					"DEBUG":   "true",
				},
			},
			azureEnv:    map[string]string{},
			serviceURLs: map[string]string{},
			want: map[string]string{
				"API_KEY": "test-key",
				"DEBUG":   "true",
			},
		},
		{
			name: "service with environment field - multiple vars",
			service: Service{
				Environment: Environment{
					"DATABASE_URL": "postgresql://localhost:5432/db",
					"LOG_LEVEL":    "debug",
				},
			},
			azureEnv:    map[string]string{},
			serviceURLs: map[string]string{},
			want: map[string]string{
				"DATABASE_URL": "postgresql://localhost:5432/db",
				"LOG_LEVEL":    "debug",
			},
		},
		{
			name: "variable substitution",
			service: Service{
				Environment: Environment{
					"DATABASE_URL": "${DB_HOST}:${DB_PORT}",
				},
			},
			azureEnv: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
			serviceURLs: map[string]string{},
			want: map[string]string{
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DATABASE_URL": "localhost:5432",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := ResolveEnvironment(ctx, tt.service, tt.azureEnv, tt.dotEnvPath, tt.serviceURLs)
			if err != nil {
				t.Fatalf("ResolveEnvironment() error = %v", err)
			}

			// Check expected variables are present
			for key, expectedValue := range tt.want {
				if gotValue, ok := got[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if gotValue != expectedValue {
					t.Errorf("key %q = %q, want %q", key, gotValue, expectedValue)
				}
			}
		})
	}
}

func TestGenerateServiceURLs(t *testing.T) {
	tests := []struct {
		name      string
		processes map[string]*ServiceProcess
		want      map[string]string
	}{
		{
			name:      "empty processes",
			processes: map[string]*ServiceProcess{},
			want:      map[string]string{},
		},
		{
			name: "single service",
			processes: map[string]*ServiceProcess{
				"api": {
					Ready: true,
					URL:   "http://localhost:3000",
					Port:  3000,
				},
			},
			want: map[string]string{
				"SERVICE_URL_API":  "http://localhost:3000",
				"SERVICE_PORT_API": "3000",
				"SERVICE_HOST_API": "localhost",
			},
		},
		{
			name: "multiple services",
			processes: map[string]*ServiceProcess{
				"api": {
					Ready: true,
					URL:   "http://localhost:3000",
					Port:  3000,
				},
				"web": {
					Ready: true,
					URL:   "http://localhost:3001",
					Port:  3001,
				},
			},
			want: map[string]string{
				"SERVICE_URL_API":  "http://localhost:3000",
				"SERVICE_PORT_API": "3000",
				"SERVICE_HOST_API": "localhost",
				"SERVICE_URL_WEB":  "http://localhost:3001",
				"SERVICE_PORT_WEB": "3001",
				"SERVICE_HOST_WEB": "localhost",
			},
		},
		{
			name: "service with dashes in name",
			processes: map[string]*ServiceProcess{
				"api-server": {
					Ready: true,
					URL:   "http://localhost:3000",
					Port:  3000,
				},
			},
			want: map[string]string{
				"SERVICE_URL_API_SERVER":  "http://localhost:3000",
				"SERVICE_PORT_API_SERVER": "3000",
				"SERVICE_HOST_API_SERVER": "localhost",
			},
		},
		{
			name: "not ready service excluded",
			processes: map[string]*ServiceProcess{
				"api": {
					Ready: false,
					URL:   "http://localhost:3000",
					Port:  3000,
				},
			},
			want: map[string]string{},
		},
		{
			name: "nil process excluded",
			processes: map[string]*ServiceProcess{
				"api": nil,
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateServiceURLs(tt.processes)

			if len(got) != len(tt.want) {
				t.Errorf("GenerateServiceURLs() returned %d vars, want %d", len(got), len(tt.want))
			}

			for key, want := range tt.want {
				if got[key] != want {
					t.Errorf("GenerateServiceURLs()[%q] = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}

func TestLoadDotEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file I/O test in short mode")
	}

	tests := []struct {
		name    string
		content string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "simple vars",
			content: "API_KEY=test123\nDEBUG=true\n",
			want: map[string]string{
				"API_KEY": "test123",
				"DEBUG":   "true",
			},
			wantErr: false,
		},
		{
			name:    "vars with spaces",
			content: "NAME=John Doe\nEMAIL=test@example.com\n",
			want: map[string]string{
				"NAME":  "John Doe",
				"EMAIL": "test@example.com",
			},
			wantErr: false,
		},
		{
			name:    "vars with equals in value",
			content: "CONNECTION_STRING=server=localhost;password=abc=123\n",
			want: map[string]string{
				"CONNECTION_STRING": "server=localhost;password=abc=123",
			},
			wantErr: false,
		},
		{
			name:    "empty lines and comments",
			content: "# Comment\nAPI_KEY=test\n\nDEBUG=true\n",
			want: map[string]string{
				"API_KEY": "test",
				"DEBUG":   "true",
			},
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			want:    map[string]string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, ".env")
			err := os.WriteFile(envFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			got, err := LoadDotEnv(envFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDotEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(got) != len(tt.want) {
					t.Errorf("LoadDotEnv() returned %d vars, want %d", len(got), len(tt.want))
				}

				for key, want := range tt.want {
					if got[key] != want {
						t.Errorf("LoadDotEnv()[%q] = %q, want %q", key, got[key], want)
					}
				}
			}
		})
	}
}

func TestLoadDotEnvInvalidPath(t *testing.T) {
	_, err := LoadDotEnv("/nonexistent/path/to/.env")
	if err == nil {
		t.Error("LoadDotEnv() with invalid path should return error")
	}
}

func TestSubstituteEnvVars(t *testing.T) {
	env := map[string]string{
		"HOST":     "localhost",
		"PORT":     "3000",
		"PROTOCOL": "http",
		"NAME":     "myapp",
	}

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "simple substitution",
			value: "${HOST}",
			want:  "localhost",
		},
		{
			name:  "multiple substitutions",
			value: "${PROTOCOL}://${HOST}:${PORT}",
			want:  "http://localhost:3000",
		},
		{
			name:  "no substitution needed",
			value: "static-value",
			want:  "static-value",
		},
		{
			name:  "undefined variable",
			value: "${UNDEFINED}",
			want:  "",
		},
		{
			name:  "mixed static and dynamic",
			value: "app-${NAME}-service",
			want:  "app-myapp-service",
		},
		{
			name:  "empty string",
			value: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := substituteEnvVars(tt.value, env)
			if got != tt.want {
				t.Errorf("substituteEnvVars(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestMaskSecrets(t *testing.T) {
	service := Service{
		Environment: Environment{
			"API_KEY":     "secret123",
			"PASSWORD":    "pass456",
			"DB_PASSWORD": "dbpass789",
			"TOKEN":       "token123",
			"PUBLIC_VAR":  "public",
		},
	}

	env := map[string]string{
		"API_KEY":     "secret123",
		"PASSWORD":    "pass456",
		"DB_PASSWORD": "dbpass789",
		"PUBLIC_VAR":  "public",
		"TOKEN":       "token123",
		"SECRET":      "mysecret",
		"AUTH_TOKEN":  "authtoken",
		"NORMAL_VAR":  "normal",
		"PUBLIC_KEY":  "pubkey123", // Should NOT be masked (has PUBLIC)
	}

	masked := MaskSecrets(service, env)

	// Variables with secret-like patterns should be masked
	secretKeys := []string{"API_KEY", "PASSWORD", "DB_PASSWORD", "TOKEN", "SECRET", "AUTH_TOKEN"}
	for _, key := range secretKeys {
		if masked[key] != "***" {
			t.Errorf("MaskSecrets()[%q] = %q, want ***", key, masked[key])
		}
	}

	// Variables without secret patterns should not be masked
	publicKeys := []string{"PUBLIC_VAR", "NORMAL_VAR", "PUBLIC_KEY"}
	for _, key := range publicKeys {
		if masked[key] == "***" {
			t.Errorf("MaskSecrets()[%q] = %q, should not be masked", key, masked[key])
		}
	}
}

func TestLoadEnvFileIfExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file I/O test in short mode")
	}

	t.Run("file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := ".env"
		content := "TEST_VAR=test_value\n"
		err := os.WriteFile(filepath.Join(tmpDir, envFile), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		got, err := LoadEnvFileIfExists(tmpDir, envFile)
		if err != nil {
			t.Errorf("LoadEnvFileIfExists() error = %v", err)
		}
		if got["TEST_VAR"] != "test_value" {
			t.Errorf("LoadEnvFileIfExists()[TEST_VAR] = %q, want %q", got["TEST_VAR"], "test_value")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		got, err := LoadEnvFileIfExists("/nonexistent", ".env")
		if err != nil {
			t.Errorf("LoadEnvFileIfExists() with nonexistent file error = %v, should not error", err)
		}
		if len(got) != 0 {
			t.Errorf("LoadEnvFileIfExists() with nonexistent file should return empty map")
		}
	})
}

func TestInjectFunctionsWorkerRuntime(t *testing.T) {
	tests := []struct {
		name           string
		initialEnv     map[string]string
		runtime        *ServiceRuntime
		expectedKeys   []string
		expectedValues map[string]string
	}{
		{
			name:       "non-functions framework does nothing",
			initialEnv: map[string]string{"KEY": "value"},
			runtime: &ServiceRuntime{
				Framework: "Node.js",
			},
			expectedKeys: []string{"KEY"},
			expectedValues: map[string]string{
				"KEY": "value",
			},
		},
		{
			name:       "functions framework injects defaults",
			initialEnv: map[string]string{},
			runtime: &ServiceRuntime{
				Framework:  "Functions - Python",
				WorkingDir: "/tmp",
			},
			expectedKeys: []string{"AzureWebJobsStorage"},
			expectedValues: map[string]string{
				"AzureWebJobsStorage": "UseDevelopmentStorage=true",
			},
		},
		{
			name:       "logic apps injects node runtime",
			initialEnv: map[string]string{},
			runtime: &ServiceRuntime{
				Framework:  "Logic Apps Standard",
				WorkingDir: "/tmp",
			},
			expectedKeys: []string{"FUNCTIONS_WORKER_RUNTIME", "AzureWebJobsStorage"},
			expectedValues: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "node",
				"AzureWebJobsStorage":      "UseDevelopmentStorage=true",
			},
		},
		{
			name: "preserves existing values",
			initialEnv: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "python",
				"AzureWebJobsStorage":      "custom-storage",
			},
			runtime: &ServiceRuntime{
				Framework:  "Functions - Node.js",
				WorkingDir: "/tmp",
			},
			expectedValues: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "python",
				"AzureWebJobsStorage":      "custom-storage",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InjectFunctionsWorkerRuntime(tt.initialEnv, tt.runtime)

			for _, key := range tt.expectedKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("Expected key %q to exist in result", key)
				}
			}

			for key, expectedValue := range tt.expectedValues {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key %q to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %q: got %q, want %q", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestLoadLocalSettings(t *testing.T) {
	t.Run("non-existent file returns nil", func(t *testing.T) {
		result := loadLocalSettings("/nonexistent/path/local.settings.json")
		if result != nil {
			t.Errorf("Expected nil for non-existent file, got %v", result)
		}
	})

	t.Run("valid local.settings.json", func(t *testing.T) {
		tempDir := t.TempDir()
		settingsFile := filepath.Join(tempDir, "local.settings.json")

		content := `{
			"IsEncrypted": false,
			"Values": {
				"FUNCTIONS_WORKER_RUNTIME": "python",
				"AzureWebJobsStorage": "UseDevelopmentStorage=true",
				"CUSTOM_KEY": "custom_value"
			}
		}`

		err := os.WriteFile(settingsFile, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := loadLocalSettings(settingsFile)

		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		expectedValues := map[string]string{
			"FUNCTIONS_WORKER_RUNTIME": "python",
			"AzureWebJobsStorage":      "UseDevelopmentStorage=true",
			"CUSTOM_KEY":               "custom_value",
		}

		for key, expectedValue := range expectedValues {
			if actualValue, exists := result[key]; !exists {
				t.Errorf("Expected key %q to exist", key)
			} else if actualValue != expectedValue {
				t.Errorf("For key %q: got %q, want %q", key, actualValue, expectedValue)
			}
		}
	})

	t.Run("invalid JSON returns nil", func(t *testing.T) {
		tempDir := t.TempDir()
		settingsFile := filepath.Join(tempDir, "local.settings.json")

		err := os.WriteFile(settingsFile, []byte("invalid json"), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := loadLocalSettings(settingsFile)
		if result != nil {
			t.Errorf("Expected nil for invalid JSON, got %v", result)
		}
	})
}

func TestHasKeyVaultReferences(t *testing.T) {
	tests := []struct {
		name     string
		envVars  []string
		expected bool
	}{
		{
			name:     "no references",
			envVars:  []string{"API_KEY=secret123", "DEBUG=true"},
			expected: false,
		},
		{
			name:     "akvs format reference",
			envVars:  []string{"SECRET=akvs://my-vault/my-secret/default"},
			expected: true,
		},
		{
			name:     "@Microsoft.KeyVault SecretUri reference",
			envVars:  []string{"SECRET=@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/mysecret/abc123)"},
			expected: true,
		},
		{
			name:     "@Microsoft.KeyVault VaultName reference",
			envVars:  []string{"SECRET=@Microsoft.KeyVault(VaultName=myvault;SecretName=mysecret)"},
			expected: true,
		},
		{
			name:     "mixed with and without references",
			envVars:  []string{"API_KEY=@Microsoft.KeyVault(VaultName=vault;SecretName=key)", "DEBUG=true"},
			expected: true,
		},
		{
			name:     "empty list",
			envVars:  []string{},
			expected: false,
		},
		{
			name:     "multiple keyvault references",
			envVars:  []string{"SECRET1=akvs://vault/secret1/v1", "SECRET2=akvs://vault/secret2/v2"},
			expected: true,
		},
		{
			name:     "malformed env vars with reference",
			envVars:  []string{"NO_EQUALS", "KEY=akvs://vault/secret/v1"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasKeyVaultReferences(tt.envVars)
			if got != tt.expected {
				t.Errorf("hasKeyVaultReferences() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEnvMapToSliceAndBack(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "simple map",
			env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "empty map",
			env:  map[string]string{},
		},
		{
			name: "special characters in values",
			env: map[string]string{
				"CONNECTION": "server=localhost;user=admin;pass=p@ss=word",
				"URL":        "https://api.example.com?key=value&other=123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert map to slice
			slice := envMapToSlice(tt.env)

			// Convert back to map
			env := envSliceToMap(slice)

			// Verify all keys and values are preserved
			if len(env) != len(tt.env) {
				t.Errorf("envMapToSlice/envSliceToMap roundtrip: got %d vars, want %d", len(env), len(tt.env))
			}

			for key, want := range tt.env {
				if got, ok := env[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if got != want {
					t.Errorf("key %q = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestResolveEnvironmentWithKeyVaultReferences(t *testing.T) {
	tests := []struct {
		name         string
		service      Service
		azureEnv     map[string]string
		serviceURLs  map[string]string
		wantContains map[string]bool // Key -> should exist
		wantErr      bool
	}{
		{
			name: "no keyvault references",
			service: Service{
				Environment: Environment{
					"API_KEY": "plain-value",
					"DEBUG":   "true",
				},
			},
			azureEnv:    map[string]string{},
			serviceURLs: map[string]string{},
			wantContains: map[string]bool{
				"API_KEY": true,
				"DEBUG":   true,
			},
			wantErr: false,
		},
		{
			name: "resolves keyvault references gracefully",
			service: Service{
				Environment: Environment{
					"API_KEY": "@Microsoft.KeyVault(VaultName=myvault;SecretName=apikey)",
					"DEBUG":   "true",
				},
			},
			azureEnv:    map[string]string{},
			serviceURLs: map[string]string{},
			wantContains: map[string]bool{
				"DEBUG": true, // Non-KV vars should be present
			},
			wantErr: false, // Should not error but may have warnings
		},
		{
			name: "akvs format reference",
			service: Service{
				Environment: Environment{
					"SECRET": "akvs://vault-name/secret-name",
					"PUBLIC": "public-value",
				},
			},
			azureEnv:    map[string]string{},
			serviceURLs: map[string]string{},
			wantContains: map[string]bool{
				"PUBLIC": true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env, err := ResolveEnvironment(ctx, tt.service, tt.azureEnv, "", tt.serviceURLs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveEnvironment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for key, shouldExist := range tt.wantContains {
				_, exists := env[key]
				if shouldExist && !exists {
					t.Errorf("expected key %q to exist in result", key)
				}
			}
		})
	}
}

func TestEnvMapToSlice(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want []string
	}{
		{
			name: "single entry",
			env: map[string]string{
				"KEY": "value",
			},
			want: []string{"KEY=value"},
		},
		{
			name: "multiple entries",
			env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			want: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name: "empty map",
			env:  map[string]string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envMapToSlice(tt.env)

			// Convert both to maps for comparison since order is not guaranteed
			wantMap := envSliceToMap(tt.want)
			gotMap := envSliceToMap(got)

			if len(gotMap) != len(wantMap) {
				t.Errorf("envMapToSlice() returned %d vars, want %d", len(gotMap), len(wantMap))
			}

			for key, want := range wantMap {
				if gotMap[key] != want {
					t.Errorf("envMapToSlice()[%q] = %q, want %q", key, gotMap[key], want)
				}
			}
		})
	}
}

func TestEnvSliceToMap(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want map[string]string
	}{
		{
			name: "single entry",
			env:  []string{"KEY=value"},
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "multiple entries",
			env:  []string{"KEY1=value1", "KEY2=value2"},
			want: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name: "value with equals",
			env:  []string{"CONNECTION=server=localhost;pass=123"},
			want: map[string]string{"CONNECTION": "server=localhost;pass=123"},
		},
		{
			name: "empty slice",
			env:  []string{},
			want: map[string]string{},
		},
		{
			name: "malformed entries",
			env:  []string{"VALID=value", "INVALID", "KEY2=value2"},
			want: map[string]string{"VALID": "value", "KEY2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envSliceToMap(tt.env)

			if len(got) != len(tt.want) {
				t.Errorf("envSliceToMap() returned %d vars, want %d", len(got), len(tt.want))
			}

			for key, want := range tt.want {
				if got[key] != want {
					t.Errorf("envSliceToMap()[%q] = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}
