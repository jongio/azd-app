package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	globalConfig     *Config
	globalConfigOnce sync.Once
	configMu         sync.RWMutex
)

// Config represents the user's azd configuration stored in ~/.azd/config.json.
type Config struct {
	App *AppConfig `json:"app,omitempty"`
}

// AppConfig represents app-level configuration.
type AppConfig struct {
	Dashboard *DashboardConfig `json:"dashboard,omitempty"`
}

// DashboardConfig represents dashboard-specific configuration.
type DashboardConfig struct {
	Browser string `json:"browser,omitempty"` // Browser target: default, system, none
}

// GetConfigPath returns the path to the azd config file.
// Returns ~/.azd/config.json (or OS-equivalent).
// This is a variable to allow test overrides.
var GetConfigPath = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".azd")
	configPath := filepath.Join(configDir, "config.json")

	return configPath, nil
}

// Load loads the configuration from ~/.azd/config.json.
// Returns an empty config if the file doesn't exist (not an error).
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save saves the configuration to ~/.azd/config.json.
func Save(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetGlobal returns the global config instance, loading it once.
func GetGlobal() *Config {
	globalConfigOnce.Do(func() {
		configMu.Lock()
		defer configMu.Unlock()

		config, err := Load()
		if err != nil {
			// Log error but don't fail - return empty config
			fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
			globalConfig = &Config{}
		} else {
			globalConfig = config
		}
	})

	return globalConfig
}

// Get retrieves a config value by key path.
// Supported keys: "app.dashboard.browser"
func Get(key string) (string, error) {
	config := GetGlobal()
	configMu.RLock()
	defer configMu.RUnlock()

	switch key {
	case "app.dashboard.browser":
		if config.App != nil && config.App.Dashboard != nil {
			return config.App.Dashboard.Browser, nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Set sets a config value by key path and saves to disk.
// Supported keys: "app.dashboard.browser"
func Set(key, value string) error {
	config := GetGlobal()
	configMu.Lock()
	defer configMu.Unlock()

	switch key {
	case "app.dashboard.browser":
		if config.App == nil {
			config.App = &AppConfig{}
		}
		if config.App.Dashboard == nil {
			config.App.Dashboard = &DashboardConfig{}
		}
		config.App.Dashboard.Browser = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(config)
}

// Unset removes a config value by key path and saves to disk.
// Supported keys: "app.dashboard.browser"
func Unset(key string) error {
	config := GetGlobal()
	configMu.Lock()
	defer configMu.Unlock()

	switch key {
	case "app.dashboard.browser":
		if config.App != nil && config.App.Dashboard != nil {
			config.App.Dashboard.Browser = ""
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(config)
}

// GetDashboardBrowser retrieves the dashboard browser preference.
// Returns empty string if not set.
func GetDashboardBrowser() string {
	value, _ := Get("app.dashboard.browser")
	return value
}

// SetDashboardBrowser sets the dashboard browser preference.
func SetDashboardBrowser(browser string) error {
	return Set("app.dashboard.browser", browser)
}

// UnsetDashboardBrowser removes the dashboard browser preference.
func UnsetDashboardBrowser() error {
	return Unset("app.dashboard.browser")
}
