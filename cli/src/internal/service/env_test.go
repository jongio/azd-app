package service

import (
	"os"
	"path/filepath"
	"testing"
)

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
			name: "service with env vars",
			service: Service{
				Env: []EnvVar{
					{Name: "API_KEY", Value: "test-key"},
					{Name: "DEBUG", Value: "true"},
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
			name: "service with secret",
			service: Service{
				Env: []EnvVar{
					{Name: "SECRET_KEY", Secret: "secret-value"},
				},
			},
			azureEnv:    map[string]string{},
			serviceURLs: map[string]string{},
			want: map[string]string{
				"SECRET_KEY": "secret-value",
			},
		},
		{
			name: "variable substitution",
			service: Service{
				Env: []EnvVar{
					{Name: "DATABASE_URL", Value: "${DB_HOST}:${DB_PORT}"},
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
			got, err := ResolveEnvironment(tt.service, tt.azureEnv, tt.dotEnvPath, tt.serviceURLs)
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
		Env: []EnvVar{
			{Name: "API_KEY", Secret: "secret123"},
			{Name: "PASSWORD", Secret: "pass456"},
			{Name: "DB_PASSWORD", Secret: "dbpass789"},
			{Name: "TOKEN", Secret: "token123"},
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
	}

	masked := MaskSecrets(service, env)

	// Variables marked as secrets in service should be masked
	secretKeys := []string{"API_KEY", "PASSWORD", "DB_PASSWORD", "TOKEN"}
	for _, key := range secretKeys {
		if masked[key] != "***" {
			t.Errorf("MaskSecrets()[%q] = %q, want ***", key, masked[key])
		}
	}

	// Variables not marked as secrets should not be masked
	publicKeys := []string{"PUBLIC_VAR", "NORMAL_VAR", "SECRET", "AUTH_TOKEN"}
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
