// Package service provides runtime detection and service orchestration capabilities.
package service

import "time"

// AnalyticsConfigGlobal represents project-level Azure Log Analytics settings.
// These are global defaults that apply to all services.
type AnalyticsConfigGlobal struct {
	// Workspace is the Log Analytics workspace ID. If not specified, auto-detected from Azure environment.
	Workspace string `yaml:"workspace,omitempty" json:"workspace,omitempty"`
	// PollingInterval is the polling interval for Log Analytics queries (e.g., "10s", "30s", "1m").
	PollingInterval string `yaml:"pollingInterval,omitempty" json:"pollingInterval,omitempty"`
	// DefaultTimespan is the default time window for historical log queries (e.g., "15m", "1h", "24h").
	DefaultTimespan string `yaml:"defaultTimespan,omitempty" json:"defaultTimespan,omitempty"`
	// Realtime enables service-specific low-latency streaming when supported.
	Realtime bool `yaml:"realtime,omitempty" json:"realtime,omitempty"`
}

// GetPollingInterval parses and returns the polling interval duration.
// Returns 10s as default if not specified or invalid.
func (c *AnalyticsConfigGlobal) GetPollingInterval() time.Duration {
	if c == nil || c.PollingInterval == "" {
		return 10 * time.Second
	}
	d, err := time.ParseDuration(c.PollingInterval)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// GetDefaultTimespan parses and returns the default timespan duration.
// Returns 30m as default if not specified or invalid.
func (c *AnalyticsConfigGlobal) GetDefaultTimespan() time.Duration {
	if c == nil || c.DefaultTimespan == "" {
		return 30 * time.Minute
	}
	d, err := time.ParseDuration(c.DefaultTimespan)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

// AnalyticsConfigService represents service-level Azure Log Analytics settings.
// These override the default table selection or provide custom KQL queries.
type AnalyticsConfigService struct {
	// Tables is the list of Log Analytics tables to query. If not specified, uses defaults for resource type.
	Tables []string `yaml:"tables,omitempty" json:"tables,omitempty"`
	// Query is the custom KQL query. Use {serviceName} and {timespan} placeholders. Takes precedence over 'tables'.
	Query string `yaml:"query,omitempty" json:"query,omitempty"`
}

// HasOverride returns true if service has custom table or query configuration.
func (c *AnalyticsConfigService) HasOverride() bool {
	if c == nil {
		return false
	}
	return len(c.Tables) > 0 || c.Query != ""
}

// LogConfigMode represents the mode for log configuration.
type LogConfigMode string

const (
	// LogConfigModeTables indicates logs should be fetched from selected tables.
	LogConfigModeTables LogConfigMode = "tables"
	// LogConfigModeCustom indicates a custom KQL query is used.
	LogConfigModeCustom LogConfigMode = "custom"
)

// LogMode represents the source of logs.
type LogMode string

const (
	// LogModeLocal indicates logs from locally running services.
	LogModeLocal LogMode = "local"
	// LogModeAzure indicates logs from Azure-deployed services.
	LogModeAzure LogMode = "azure"
)

// AzureStatus represents the Azure log streaming status.
//
// Deprecated: This type is preserved for backward compatibility with old clients.
//
// New code should use /api/azure/logs/health endpoint instead.
type AzureStatus struct {
	Mode                 LogMode `json:"mode"`
	Connected            bool    `json:"connected"`
	Enabled              bool    `json:"enabled"`
	ResourceCount        int     `json:"resourceCount"`
	HasCredentials       bool    `json:"hasCredentials"`
	HasLogAnalytics      bool    `json:"hasLogAnalytics"`
	HasResourceDiscovery bool    `json:"hasResourceDiscovery"`
	ConnectionIssue      string  `json:"connectionIssue,omitempty"`
	ConnectionMessage    string  `json:"connectionMessage,omitempty"`
	LastError            string  `json:"lastError,omitempty"`
}
