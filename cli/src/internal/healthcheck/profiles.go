package healthcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// HealthProfile represents a health check configuration profile.
type HealthProfile struct {
	Name                   string        `yaml:"name"`
	Interval               time.Duration `yaml:"interval"`
	Timeout                time.Duration `yaml:"timeout"`
	Retries                int           `yaml:"retries"`
	CircuitBreaker         bool          `yaml:"circuitBreaker"`
	CircuitBreakerFailures int           `yaml:"circuitBreakerFailures"`
	CircuitBreakerTimeout  time.Duration `yaml:"circuitBreakerTimeout"`
	RateLimit              int           `yaml:"rateLimit"`
	Verbose                bool          `yaml:"verbose"`
	LogLevel               string        `yaml:"logLevel"`
	LogFormat              string        `yaml:"logFormat"`
	Metrics                bool          `yaml:"metrics"`
	MetricsPort            int           `yaml:"metricsPort"`
	CacheTTL               time.Duration `yaml:"cacheTTL"`
}

// HealthProfiles contains multiple named profiles.
type HealthProfiles struct {
	Profiles map[string]HealthProfile `yaml:"profiles"`
}

// LoadHealthProfiles loads health profiles from the project directory.
func LoadHealthProfiles(projectDir string) (*HealthProfiles, error) {
	profilePath := filepath.Join(projectDir, ".azd", "health-profiles.yaml")

	data, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultProfiles(), nil
		}
		return nil, fmt.Errorf("failed to read health profiles: %w", err)
	}

	var profiles HealthProfiles
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse health profiles: %w", err)
	}

	// Merge with defaults for any missing profiles
	defaults := getDefaultProfiles()
	for name, profile := range defaults.Profiles {
		if _, exists := profiles.Profiles[name]; !exists {
			profiles.Profiles[name] = profile
		}
	}

	return &profiles, nil
}

// getDefaultProfiles returns the default health check profiles.
func getDefaultProfiles() *HealthProfiles {
	return &HealthProfiles{
		Profiles: map[string]HealthProfile{
			"development": {
				Name:                   "development",
				Interval:               5 * time.Second,
				Timeout:                10 * time.Second,
				Retries:                1,
				CircuitBreaker:         false,
				CircuitBreakerFailures: 5,
				CircuitBreakerTimeout:  60 * time.Second,
				RateLimit:              0, // Unlimited
				Verbose:                true,
				LogLevel:               "debug",
				LogFormat:              "pretty",
				Metrics:                false,
				MetricsPort:            9090,
				CacheTTL:               0, // No caching in dev
			},
			"production": {
				Name:                   "production",
				Interval:               30 * time.Second,
				Timeout:                5 * time.Second,
				Retries:                3,
				CircuitBreaker:         true,
				CircuitBreakerFailures: 5,
				CircuitBreakerTimeout:  60 * time.Second,
				RateLimit:              10, // 10 checks/sec per service
				Verbose:                false,
				LogLevel:               "info",
				LogFormat:              "json",
				Metrics:                true,
				MetricsPort:            9090,
				CacheTTL:               5 * time.Second,
			},
			"ci": {
				Name:                   "ci",
				Interval:               10 * time.Second,
				Timeout:                30 * time.Second,
				Retries:                5,
				CircuitBreaker:         false,
				CircuitBreakerFailures: 10,
				CircuitBreakerTimeout:  30 * time.Second,
				RateLimit:              0, // Unlimited
				Verbose:                true,
				LogLevel:               "info",
				LogFormat:              "json",
				Metrics:                false,
				MetricsPort:            9090,
				CacheTTL:               0, // No caching in CI
			},
			"staging": {
				Name:                   "staging",
				Interval:               15 * time.Second,
				Timeout:                10 * time.Second,
				Retries:                3,
				CircuitBreaker:         true,
				CircuitBreakerFailures: 5,
				CircuitBreakerTimeout:  60 * time.Second,
				RateLimit:              20, // Higher limit for staging
				Verbose:                true,
				LogLevel:               "debug",
				LogFormat:              "json",
				Metrics:                true,
				MetricsPort:            9090,
				CacheTTL:               3 * time.Second,
			},
		},
	}
}

// GetProfile returns a profile by name, or an error if not found.
func (p *HealthProfiles) GetProfile(name string) (HealthProfile, error) {
	profile, exists := p.Profiles[name]
	if !exists {
		return HealthProfile{}, fmt.Errorf("profile '%s' not found. Available profiles: development, production, ci, staging", name)
	}
	return profile, nil
}

// SaveSampleProfiles saves a sample health-profiles.yaml file.
func SaveSampleProfiles(projectDir string) error {
	azdDir := filepath.Join(projectDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		return fmt.Errorf("failed to create .azd directory: %w", err)
	}

	profilePath := filepath.Join(azdDir, "health-profiles.yaml")

	// Check if file already exists
	if _, err := os.Stat(profilePath); err == nil {
		return fmt.Errorf("health-profiles.yaml already exists at %s", profilePath)
	}

	profiles := getDefaultProfiles()
	data, err := yaml.Marshal(profiles)
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	header := `# Health Check Profiles for azd app health
# 
# Profiles allow you to define different health check configurations
# for different environments (development, production, ci, staging)
#
# Usage: azd app health --profile production --stream
#
# Available settings:
#   interval:               Time between health checks in streaming mode
#   timeout:                Maximum time to wait for health check
#   retries:                Number of retries for failed checks
#   circuitBreaker:         Enable circuit breaker pattern
#   circuitBreakerFailures: Failures before circuit opens
#   circuitBreakerTimeout:  Time before circuit retry
#   rateLimit:              Max health checks per second (0 = unlimited)
#   verbose:                Show detailed output
#   logLevel:               Logging level (debug, info, warn, error)
#   logFormat:              Log format (json, pretty, text)
#   metrics:                Enable Prometheus metrics endpoint
#   metricsPort:            Prometheus metrics port
#   cacheTTL:               Cache health results for this duration (0 = no cache)

`

	content := header + string(data)
	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write profiles file: %w", err)
	}

	return nil
}
