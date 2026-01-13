package service_test

import (
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestAnalyticsConfigGlobal_GetPollingInterval tests polling interval parsing
func TestAnalyticsConfigGlobal_GetPollingInterval(t *testing.T) {
	tests := []struct {
		name     string
		config   *service.AnalyticsConfigGlobal
		expected time.Duration
	}{
		{
			name:     "Nil config returns default",
			config:   nil,
			expected: 10 * time.Second,
		},
		{
			name:     "Empty interval returns default",
			config:   &service.AnalyticsConfigGlobal{PollingInterval: ""},
			expected: 10 * time.Second,
		},
		{
			name:     "Valid 30s interval",
			config:   &service.AnalyticsConfigGlobal{PollingInterval: "30s"},
			expected: 30 * time.Second,
		},
		{
			name:     "Valid 1m interval",
			config:   &service.AnalyticsConfigGlobal{PollingInterval: "1m"},
			expected: 1 * time.Minute,
		},
		{
			name:     "Valid 5m interval",
			config:   &service.AnalyticsConfigGlobal{PollingInterval: "5m"},
			expected: 5 * time.Minute,
		},
		{
			name:     "Invalid interval returns default",
			config:   &service.AnalyticsConfigGlobal{PollingInterval: "invalid"},
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetPollingInterval()
			if result != tt.expected {
				t.Errorf("GetPollingInterval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAnalyticsConfigGlobal_GetDefaultTimespan tests timespan parsing
func TestAnalyticsConfigGlobal_GetDefaultTimespan(t *testing.T) {
	tests := []struct {
		name     string
		config   *service.AnalyticsConfigGlobal
		expected time.Duration
	}{
		{
			name:     "Nil config returns default",
			config:   nil,
			expected: 30 * time.Minute,
		},
		{
			name:     "Empty timespan returns default",
			config:   &service.AnalyticsConfigGlobal{DefaultTimespan: ""},
			expected: 30 * time.Minute,
		},
		{
			name:     "Valid 15m timespan",
			config:   &service.AnalyticsConfigGlobal{DefaultTimespan: "15m"},
			expected: 15 * time.Minute,
		},
		{
			name:     "Valid 1h timespan",
			config:   &service.AnalyticsConfigGlobal{DefaultTimespan: "1h"},
			expected: 1 * time.Hour,
		},
		{
			name:     "Valid 24h timespan",
			config:   &service.AnalyticsConfigGlobal{DefaultTimespan: "24h"},
			expected: 24 * time.Hour,
		},
		{
			name:     "Invalid timespan returns default",
			config:   &service.AnalyticsConfigGlobal{DefaultTimespan: "bad"},
			expected: 30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetDefaultTimespan()
			if result != tt.expected {
				t.Errorf("GetDefaultTimespan() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAnalyticsConfigService_HasOverride tests override detection
func TestAnalyticsConfigService_HasOverride(t *testing.T) {
	tests := []struct {
		name     string
		config   *service.AnalyticsConfigService
		expected bool
	}{
		{
			name:     "Nil config has no override",
			config:   nil,
			expected: false,
		},
		{
			name:     "Empty config has no override",
			config:   &service.AnalyticsConfigService{},
			expected: false,
		},
		{
			name: "Config with tables has override",
			config: &service.AnalyticsConfigService{
				Tables: []string{"ContainerAppConsoleLogs_CL"},
			},
			expected: true,
		},
		{
			name: "Config with query has override",
			config: &service.AnalyticsConfigService{
				Query: "ContainerAppConsoleLogs_CL | where ServiceName == '{serviceName}'",
			},
			expected: true,
		},
		{
			name: "Config with both tables and query has override",
			config: &service.AnalyticsConfigService{
				Tables: []string{"Table1"},
				Query:  "SELECT * FROM Table1",
			},
			expected: true,
		},
		{
			name: "Config with empty tables array has no override",
			config: &service.AnalyticsConfigService{
				Tables: []string{},
			},
			expected: false,
		},
		{
			name: "Config with empty query string has no override",
			config: &service.AnalyticsConfigService{
				Query: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasOverride()
			if result != tt.expected {
				t.Errorf("HasOverride() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAnalyticsConfig_EdgeCases tests edge cases for analytics config
func TestAnalyticsConfig_EdgeCases(t *testing.T) {
	t.Run("Multiple valid durations", func(t *testing.T) {
		config := &service.AnalyticsConfigGlobal{
			PollingInterval: "2m30s",
			DefaultTimespan: "1h30m",
		}

		pollingInterval := config.GetPollingInterval()
		expectedPolling := 2*time.Minute + 30*time.Second
		if pollingInterval != expectedPolling {
			t.Errorf("GetPollingInterval() = %v, want %v", pollingInterval, expectedPolling)
		}

		timespan := config.GetDefaultTimespan()
		expectedTimespan := 1*time.Hour + 30*time.Minute
		if timespan != expectedTimespan {
			t.Errorf("GetDefaultTimespan() = %v, want %v", timespan, expectedTimespan)
		}
	})

	t.Run("Config with all fields set", func(t *testing.T) {
		config := &service.AnalyticsConfigGlobal{
			Workspace:       "workspace-123",
			PollingInterval: "15s",
			DefaultTimespan: "2h",
			Realtime:        true,
		}

		if config.GetPollingInterval() != 15*time.Second {
			t.Error("Polling interval mismatch")
		}
		if config.GetDefaultTimespan() != 2*time.Hour {
			t.Error("Timespan mismatch")
		}
		if !config.Realtime {
			t.Error("Realtime should be true")
		}
		if config.Workspace != "workspace-123" {
			t.Error("Workspace mismatch")
		}
	})
}
