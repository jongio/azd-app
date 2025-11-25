package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// NotificationPreferences represents user preferences for the notification system.
type NotificationPreferences struct {
	// OSNotifications controls whether OS-level notifications are enabled
	OSNotifications bool `json:"osNotifications"`

	// DashboardNotifications controls whether in-dashboard toast notifications are enabled
	DashboardNotifications bool `json:"dashboardNotifications"`

	// SeverityFilter controls which severity levels trigger notifications
	// Values: "critical", "warning", "info", "all"
	SeverityFilter string `json:"severityFilter"`

	// QuietHours defines time ranges when notifications should be suppressed
	QuietHours []QuietHourRange `json:"quietHours,omitempty"`

	// ServiceSettings contains per-service notification preferences
	ServiceSettings map[string]ServiceNotificationSettings `json:"serviceSettings,omitempty"`

	// RateLimitWindow defines the deduplication window (default: 5 minutes)
	// Format: "5m", "10s", etc. (parsed via time.ParseDuration)
	RateLimitWindow string `json:"rateLimitWindow,omitempty"`

	mu sync.RWMutex `json:"-"`
}

// QuietHourRange defines a time range when notifications are suppressed.
type QuietHourRange struct {
	Start string `json:"start"` // Format: "HH:MM" (24-hour)
	End   string `json:"end"`   // Format: "HH:MM" (24-hour)
}

// ServiceNotificationSettings contains notification preferences for a specific service.
type ServiceNotificationSettings struct {
	Enabled bool `json:"enabled"`
}

var (
	notificationPrefs     *NotificationPreferences
	notificationPrefsOnce sync.Once
	notificationPrefsMu   sync.RWMutex
)

// DefaultNotificationPreferences returns the default notification preferences.
func DefaultNotificationPreferences() *NotificationPreferences {
	return &NotificationPreferences{
		OSNotifications:        true,
		DashboardNotifications: true,
		SeverityFilter:         "critical", // Only critical by default for OS notifications
		QuietHours:             []QuietHourRange{},
		ServiceSettings:        make(map[string]ServiceNotificationSettings),
		RateLimitWindow:        "5m",
	}
}

// GetNotificationPreferencesPath returns the path to the notification preferences file.
// Returns ~/.azd/notifications.json (or OS-equivalent).
var GetNotificationPreferencesPath = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".azd")
	prefsPath := filepath.Join(configDir, "notifications.json")

	return prefsPath, nil
}

// LoadNotificationPreferences loads notification preferences from disk.
// Returns default preferences if the file doesn't exist.
func LoadNotificationPreferences() (*NotificationPreferences, error) {
	prefsPath, err := GetNotificationPreferencesPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return default preferences
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		return DefaultNotificationPreferences(), nil
	}

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read notification preferences: %w", err)
	}

	var prefs NotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("failed to parse notification preferences: %w", err)
	}

	// Validate and apply defaults
	if err := prefs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid notification preferences: %w", err)
	}

	prefs.ApplyDefaults()

	return &prefs, nil
}

// SaveNotificationPreferences saves notification preferences to disk.
func SaveNotificationPreferences(prefs *NotificationPreferences) error {
	if err := prefs.Validate(); err != nil {
		return fmt.Errorf("invalid notification preferences: %w", err)
	}

	prefsPath, err := GetNotificationPreferencesPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	prefsDir := filepath.Dir(prefsPath)
	if err := os.MkdirAll(prefsDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Acquire read lock for serialization
	prefs.mu.RLock()
	data, err := json.MarshalIndent(prefs, "", "  ")
	prefs.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to serialize notification preferences: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tempPath := prefsPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp preferences file: %w", err)
	}

	if err := os.Rename(tempPath, prefsPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to save notification preferences: %w", err)
	}

	return nil
}

// GetGlobalNotificationPreferences returns the global notification preferences instance.
// Loads from disk once and caches in memory.
func GetGlobalNotificationPreferences() *NotificationPreferences {
	notificationPrefsOnce.Do(func() {
		notificationPrefsMu.Lock()
		defer notificationPrefsMu.Unlock()

		prefs, err := LoadNotificationPreferences()
		if err != nil {
			// Log error but don't fail - return default preferences
			fmt.Fprintf(os.Stderr, "Warning: Failed to load notification preferences: %v\n", err)
			notificationPrefs = DefaultNotificationPreferences()
		} else {
			notificationPrefs = prefs
		}
	})

	return notificationPrefs
}

// Validate validates the notification preferences.
func (p *NotificationPreferences) Validate() error {
	// Validate severity filter
	validSeverities := map[string]bool{
		"critical": true,
		"warning":  true,
		"info":     true,
		"all":      true,
	}
	if !validSeverities[p.SeverityFilter] {
		return fmt.Errorf("invalid severity filter: %s (must be critical, warning, info, or all)", p.SeverityFilter)
	}

	// Validate rate limit window format
	if p.RateLimitWindow != "" {
		if _, err := time.ParseDuration(p.RateLimitWindow); err != nil {
			return fmt.Errorf("invalid rate limit window format: %s (use format like '5m', '10s')", p.RateLimitWindow)
		}
	}

	// Validate quiet hours format
	for i, qh := range p.QuietHours {
		if !isValidTimeFormat(qh.Start) {
			return fmt.Errorf("invalid quiet hours start time at index %d: %s (use HH:MM format)", i, qh.Start)
		}
		if !isValidTimeFormat(qh.End) {
			return fmt.Errorf("invalid quiet hours end time at index %d: %s (use HH:MM format)", i, qh.End)
		}
	}

	return nil
}

// ApplyDefaults applies default values for missing fields.
func (p *NotificationPreferences) ApplyDefaults() {
	if p.SeverityFilter == "" {
		p.SeverityFilter = "critical"
	}
	if p.RateLimitWindow == "" {
		p.RateLimitWindow = "5m"
	}
	if p.QuietHours == nil {
		p.QuietHours = []QuietHourRange{}
	}
	if p.ServiceSettings == nil {
		p.ServiceSettings = make(map[string]ServiceNotificationSettings)
	}
}

// IsServiceEnabled checks if notifications are enabled for a specific service.
// Returns true if no per-service settings exist (default: enabled).
func (p *NotificationPreferences) IsServiceEnabled(serviceName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.isServiceEnabledLocked(serviceName)
}

// isServiceEnabledLocked is an internal helper that checks service status without locking.
// Must be called with p.mu held (read or write).
func (p *NotificationPreferences) isServiceEnabledLocked(serviceName string) bool {
	settings, exists := p.ServiceSettings[serviceName]
	if !exists {
		return true // Default: all services enabled
	}
	return settings.Enabled
}

// SetServiceEnabled sets whether notifications are enabled for a specific service.
// TODO: Add validation for serviceName format (e.g., non-empty, valid characters)
func (p *NotificationPreferences) SetServiceEnabled(serviceName string, enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ServiceSettings == nil {
		p.ServiceSettings = make(map[string]ServiceNotificationSettings)
	}
	p.ServiceSettings[serviceName] = ServiceNotificationSettings{Enabled: enabled}
}

// IsInQuietHours checks if the current time falls within any quiet hour range.
func (p *NotificationPreferences) IsInQuietHours() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.isInQuietHoursLocked()
}

// isInQuietHoursLocked is an internal helper that checks quiet hours without locking.
// Must be called with p.mu held (read or write).
func (p *NotificationPreferences) isInQuietHoursLocked() bool {
	if len(p.QuietHours) == 0 {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	for _, qh := range p.QuietHours {
		if isTimeInRange(currentTime, qh.Start, qh.End) {
			return true
		}
	}

	return false
}

// ShouldNotify determines if a notification should be sent based on preferences.
func (p *NotificationPreferences) ShouldNotify(serviceName string, severity string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if service is enabled (use unlocked helper to avoid double-locking)
	if !p.isServiceEnabledLocked(serviceName) {
		return false
	}

	// Check quiet hours (use unlocked helper to avoid double-locking)
	if p.isInQuietHoursLocked() {
		return false
	}

	// Check severity filter
	switch p.SeverityFilter {
	case "critical":
		return severity == "critical"
	case "warning":
		return severity == "critical" || severity == "warning"
	case "info":
		return severity == "critical" || severity == "warning" || severity == "info"
	case "all":
		return true
	default:
		return severity == "critical" // Default: critical only
	}
}

// GetRateLimitDuration parses and returns the rate limit window as a duration.
func (p *NotificationPreferences) GetRateLimitDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	duration, err := time.ParseDuration(p.RateLimitWindow)
	if err != nil {
		return 5 * time.Minute // Default: 5 minutes
	}
	return duration
}

// isValidTimeFormat checks if a time string is in HH:MM format (24-hour).
// Returns true if the string can be parsed as a valid time.
// TODO: Consider caching parsed time values to avoid repeated parsing in tight loops
func isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

// isTimeInRange checks if currentTime falls within the range [start, end].
// Handles ranges that cross midnight (e.g., 23:00 to 01:00).
// All times must be in HH:MM format (24-hour).
// Returns true if currentTime is >= start and < end.
func isTimeInRange(currentTime, start, end string) bool {
	current, _ := time.Parse("15:04", currentTime)
	startTime, _ := time.Parse("15:04", start)
	endTime, _ := time.Parse("15:04", end)

	// Range doesn't cross midnight
	if startTime.Before(endTime) {
		return !current.Before(startTime) && current.Before(endTime)
	}

	// Range crosses midnight (e.g., 23:00 to 01:00)
	return !current.Before(startTime) || current.Before(endTime)
}
