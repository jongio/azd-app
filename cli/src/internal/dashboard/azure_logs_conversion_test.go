package dashboard

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/azure"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestConvertAzureLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		azLevel  azure.LogLevel
		expected service.LogLevel
	}{
		{
			name:     "Info level",
			azLevel:  azure.LogLevelInfo,
			expected: service.LogLevelInfo,
		},
		{
			name:     "Warn level",
			azLevel:  azure.LogLevelWarn,
			expected: service.LogLevelWarn,
		},
		{
			name:     "Error level",
			azLevel:  azure.LogLevelError,
			expected: service.LogLevelError,
		},
		{
			name:     "Debug level",
			azLevel:  azure.LogLevelDebug,
			expected: service.LogLevelDebug,
		},
		{
			name:     "Unknown level defaults to Info",
			azLevel:  azure.LogLevel(999), // Invalid log level
			expected: service.LogLevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAzureLogLevel(tt.azLevel)
			if result != tt.expected {
				t.Errorf("convertAzureLogLevel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInvalidIntError_Error(t *testing.T) {
	err := &invalidIntError{}
	expected := "invalid integer"
	if err.Error() != expected {
		t.Errorf("invalidIntError.Error() = %v, want %v", err.Error(), expected)
	}
}
