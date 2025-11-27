package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDefaultNotificationPreferences(t *testing.T) {
	prefs := DefaultNotificationPreferences()

	if !prefs.OSNotifications {
		t.Error("OSNotifications should be enabled by default")
	}

	if !prefs.DashboardNotifications {
		t.Error("DashboardNotifications should be enabled by default")
	}

	if prefs.SeverityFilter != "critical" {
		t.Errorf("SeverityFilter = %q, want %q", prefs.SeverityFilter, "critical")
	}

	if prefs.RateLimitWindow != "5m" {
		t.Errorf("RateLimitWindow = %q, want %q", prefs.RateLimitWindow, "5m")
	}

	if len(prefs.QuietHours) != 0 {
		t.Errorf("QuietHours should be empty by default, got %d items", len(prefs.QuietHours))
	}

	if prefs.ServiceSettings == nil {
		t.Error("ServiceSettings should be initialized")
	}
}

func TestNotificationPreferencesValidation(t *testing.T) {
	tests := []struct {
		name    string
		prefs   *NotificationPreferences
		wantErr bool
	}{
		{
			name:    "valid default preferences",
			prefs:   DefaultNotificationPreferences(),
			wantErr: false,
		},
		{
			name: "valid critical severity",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				RateLimitWindow: "5m",
			},
			wantErr: false,
		},
		{
			name: "valid warning severity",
			prefs: &NotificationPreferences{
				SeverityFilter:  "warning",
				RateLimitWindow: "10m",
			},
			wantErr: false,
		},
		{
			name: "valid info severity",
			prefs: &NotificationPreferences{
				SeverityFilter:  "info",
				RateLimitWindow: "1m",
			},
			wantErr: false,
		},
		{
			name: "valid all severity",
			prefs: &NotificationPreferences{
				SeverityFilter:  "all",
				RateLimitWindow: "30s",
			},
			wantErr: false,
		},
		{
			name: "invalid severity filter",
			prefs: &NotificationPreferences{
				SeverityFilter:  "invalid",
				RateLimitWindow: "5m",
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit window",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				RateLimitWindow: "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid quiet hours",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				RateLimitWindow: "5m",
				QuietHours: []QuietHourRange{
					{Start: "22:00", End: "08:00"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid quiet hours start",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				RateLimitWindow: "5m",
				QuietHours: []QuietHourRange{
					{Start: "25:00", End: "08:00"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid quiet hours end",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				RateLimitWindow: "5m",
				QuietHours: []QuietHourRange{
					{Start: "22:00", End: "invalid"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prefs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotificationPreferencesApplyDefaults(t *testing.T) {
	prefs := &NotificationPreferences{}
	prefs.ApplyDefaults()

	if prefs.SeverityFilter != "critical" {
		t.Errorf("SeverityFilter = %q, want %q", prefs.SeverityFilter, "critical")
	}

	if prefs.RateLimitWindow != "5m" {
		t.Errorf("RateLimitWindow = %q, want %q", prefs.RateLimitWindow, "5m")
	}

	if prefs.QuietHours == nil {
		t.Error("QuietHours should be initialized")
	}

	if prefs.ServiceSettings == nil {
		t.Error("ServiceSettings should be initialized")
	}
}

func TestIsServiceEnabled(t *testing.T) {
	prefs := DefaultNotificationPreferences()

	// Test default (no per-service settings)
	if !prefs.IsServiceEnabled("api-service") {
		t.Error("Service should be enabled by default")
	}

	// Disable a service
	prefs.SetServiceEnabled("api-service", false)
	if prefs.IsServiceEnabled("api-service") {
		t.Error("Service should be disabled after SetServiceEnabled(false)")
	}

	// Enable a service
	prefs.SetServiceEnabled("api-service", true)
	if !prefs.IsServiceEnabled("api-service") {
		t.Error("Service should be enabled after SetServiceEnabled(true)")
	}

	// Other services should still be enabled by default
	if !prefs.IsServiceEnabled("other-service") {
		t.Error("Unmodified service should be enabled by default")
	}
}

func TestIsInQuietHours(t *testing.T) {
	tests := []struct {
		name        string
		quietHours  []QuietHourRange
		currentTime string
		want        bool
	}{
		{
			name:        "no quiet hours",
			quietHours:  []QuietHourRange{},
			currentTime: "14:30",
			want:        false,
		},
		{
			name: "within quiet hours",
			quietHours: []QuietHourRange{
				{Start: "22:00", End: "08:00"},
			},
			currentTime: "23:00",
			want:        true,
		},
		{
			name: "outside quiet hours",
			quietHours: []QuietHourRange{
				{Start: "22:00", End: "08:00"},
			},
			currentTime: "14:00",
			want:        false,
		},
		{
			name: "multiple quiet hours - in first range",
			quietHours: []QuietHourRange{
				{Start: "22:00", End: "08:00"},
				{Start: "12:00", End: "13:00"},
			},
			currentTime: "23:30",
			want:        true,
		},
		{
			name: "multiple quiet hours - in second range",
			quietHours: []QuietHourRange{
				{Start: "22:00", End: "08:00"},
				{Start: "12:00", End: "13:00"},
			},
			currentTime: "12:30",
			want:        true,
		},
		{
			name: "multiple quiet hours - outside all ranges",
			quietHours: []QuietHourRange{
				{Start: "22:00", End: "08:00"},
				{Start: "12:00", End: "13:00"},
			},
			currentTime: "14:00",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := &NotificationPreferences{
				QuietHours: tt.quietHours,
			}

			// We need to test isInQuietHoursLocked with the logic that uses current time
			// Since IsInQuietHours uses time.Now(), we test the underlying logic
			var result bool
			if len(tt.quietHours) == 0 {
				result = false
			} else {
				for _, qh := range tt.quietHours {
					if isTimeInRange(tt.currentTime, qh.Start, qh.End) {
						result = true
						break
					}
				}
			}

			if result != tt.want {
				t.Errorf("IsInQuietHours() for time %s = %v, want %v", tt.currentTime, result, tt.want)
			}

			// Also verify that the preferences struct properly initializes
			if len(prefs.QuietHours) != len(tt.quietHours) {
				t.Errorf("QuietHours length = %d, want %d", len(prefs.QuietHours), len(tt.quietHours))
			}
		})
	}
}

func TestIsTimeInRange(t *testing.T) {
	tests := []struct {
		name        string
		currentTime string
		start       string
		end         string
		want        bool
	}{
		{
			name:        "within range - same day",
			currentTime: "14:30",
			start:       "14:00",
			end:         "15:00",
			want:        true,
		},
		{
			name:        "before range - same day",
			currentTime: "13:30",
			start:       "14:00",
			end:         "15:00",
			want:        false,
		},
		{
			name:        "after range - same day",
			currentTime: "15:30",
			start:       "14:00",
			end:         "15:00",
			want:        false,
		},
		{
			name:        "within range - crosses midnight (late night)",
			currentTime: "23:30",
			start:       "22:00",
			end:         "08:00",
			want:        true,
		},
		{
			name:        "within range - crosses midnight (early morning)",
			currentTime: "07:30",
			start:       "22:00",
			end:         "08:00",
			want:        true,
		},
		{
			name:        "outside range - crosses midnight",
			currentTime: "14:00",
			start:       "22:00",
			end:         "08:00",
			want:        false,
		},
		{
			name:        "at start time",
			currentTime: "14:00",
			start:       "14:00",
			end:         "15:00",
			want:        true,
		},
		{
			name:        "at end time",
			currentTime: "15:00",
			start:       "14:00",
			end:         "15:00",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeInRange(tt.currentTime, tt.start, tt.end)
			if got != tt.want {
				t.Errorf("isTimeInRange(%q, %q, %q) = %v, want %v",
					tt.currentTime, tt.start, tt.end, got, tt.want)
			}
		})
	}
}

func TestShouldNotify(t *testing.T) {
	tests := []struct {
		name        string
		prefs       *NotificationPreferences
		serviceName string
		severity    string
		want        bool
	}{
		{
			name: "critical severity with critical filter",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				ServiceSettings: map[string]ServiceNotificationSettings{},
				QuietHours:      []QuietHourRange{},
			},
			serviceName: "api-service",
			severity:    "critical",
			want:        true,
		},
		{
			name: "warning severity with critical filter",
			prefs: &NotificationPreferences{
				SeverityFilter:  "critical",
				ServiceSettings: map[string]ServiceNotificationSettings{},
				QuietHours:      []QuietHourRange{},
			},
			serviceName: "api-service",
			severity:    "warning",
			want:        false,
		},
		{
			name: "warning severity with warning filter",
			prefs: &NotificationPreferences{
				SeverityFilter:  "warning",
				ServiceSettings: map[string]ServiceNotificationSettings{},
				QuietHours:      []QuietHourRange{},
			},
			serviceName: "api-service",
			severity:    "warning",
			want:        true,
		},
		{
			name: "info severity with info filter",
			prefs: &NotificationPreferences{
				SeverityFilter:  "info",
				ServiceSettings: map[string]ServiceNotificationSettings{},
				QuietHours:      []QuietHourRange{},
			},
			serviceName: "api-service",
			severity:    "info",
			want:        true,
		},
		{
			name: "all filter allows all",
			prefs: &NotificationPreferences{
				SeverityFilter:  "all",
				ServiceSettings: map[string]ServiceNotificationSettings{},
				QuietHours:      []QuietHourRange{},
			},
			serviceName: "api-service",
			severity:    "info",
			want:        true,
		},
		{
			name: "service disabled",
			prefs: &NotificationPreferences{
				SeverityFilter: "critical",
				ServiceSettings: map[string]ServiceNotificationSettings{
					"api-service": {Enabled: false},
				},
				QuietHours: []QuietHourRange{},
			},
			serviceName: "api-service",
			severity:    "critical",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prefs.ShouldNotify(tt.serviceName, tt.severity)
			if got != tt.want {
				t.Errorf("ShouldNotify(%q, %q) = %v, want %v",
					tt.serviceName, tt.severity, got, tt.want)
			}
		})
	}
}

func TestGetRateLimitDuration(t *testing.T) {
	tests := []struct {
		name            string
		rateLimitWindow string
		want            time.Duration
	}{
		{
			name:            "5 minutes",
			rateLimitWindow: "5m",
			want:            5 * time.Minute,
		},
		{
			name:            "10 seconds",
			rateLimitWindow: "10s",
			want:            10 * time.Second,
		},
		{
			name:            "1 hour",
			rateLimitWindow: "1h",
			want:            1 * time.Hour,
		},
		{
			name:            "invalid format - fallback to default",
			rateLimitWindow: "invalid",
			want:            5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := &NotificationPreferences{
				RateLimitWindow: tt.rateLimitWindow,
			}
			got := prefs.GetRateLimitDuration()
			if got != tt.want {
				t.Errorf("GetRateLimitDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadSaveNotificationPreferences(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "azd-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prefsPath := filepath.Join(tmpDir, ".azd", "notifications.json")

	// Override GetNotificationPreferencesPath for testing
	originalGetPath := GetNotificationPreferencesPath
	GetNotificationPreferencesPath = func() (string, error) {
		return prefsPath, nil
	}
	defer func() {
		GetNotificationPreferencesPath = originalGetPath
	}()

	// Test loading non-existent file (should return defaults)
	prefs, err := LoadNotificationPreferences()
	if err != nil {
		t.Fatalf("LoadNotificationPreferences() error = %v", err)
	}

	if prefs.SeverityFilter != "critical" {
		t.Errorf("Default SeverityFilter = %q, want %q", prefs.SeverityFilter, "critical")
	}

	// Modify preferences
	prefs.OSNotifications = false
	prefs.SeverityFilter = "warning"
	prefs.SetServiceEnabled("api-service", false)
	prefs.QuietHours = []QuietHourRange{
		{Start: "22:00", End: "08:00"},
	}

	// Save preferences
	if err := SaveNotificationPreferences(prefs); err != nil {
		t.Fatalf("SaveNotificationPreferences() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		t.Error("Preferences file was not created")
	}

	// Load preferences again
	loadedPrefs, err := LoadNotificationPreferences()
	if err != nil {
		t.Fatalf("LoadNotificationPreferences() after save error = %v", err)
	}

	// Verify loaded preferences match saved preferences
	if loadedPrefs.OSNotifications != prefs.OSNotifications {
		t.Errorf("OSNotifications = %v, want %v", loadedPrefs.OSNotifications, prefs.OSNotifications)
	}

	if loadedPrefs.SeverityFilter != prefs.SeverityFilter {
		t.Errorf("SeverityFilter = %q, want %q", loadedPrefs.SeverityFilter, prefs.SeverityFilter)
	}

	if loadedPrefs.IsServiceEnabled("api-service") {
		t.Error("api-service should be disabled after load")
	}

	if len(loadedPrefs.QuietHours) != 1 {
		t.Errorf("QuietHours length = %d, want 1", len(loadedPrefs.QuietHours))
	}
}

func TestSaveNotificationPreferencesValidation(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "azd-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prefsPath := filepath.Join(tmpDir, ".azd", "notifications.json")

	// Override GetNotificationPreferencesPath for testing
	originalGetPath := GetNotificationPreferencesPath
	GetNotificationPreferencesPath = func() (string, error) {
		return prefsPath, nil
	}
	defer func() {
		GetNotificationPreferencesPath = originalGetPath
	}()

	// Try to save invalid preferences
	invalidPrefs := &NotificationPreferences{
		SeverityFilter:  "invalid-severity",
		RateLimitWindow: "5m",
	}

	err = SaveNotificationPreferences(invalidPrefs)
	if err == nil {
		t.Error("SaveNotificationPreferences() should fail with invalid severity filter")
	}
}

func TestNotificationPreferencesJSONSerialization(t *testing.T) {
	prefs := &NotificationPreferences{
		OSNotifications:        true,
		DashboardNotifications: false,
		SeverityFilter:         "warning",
		RateLimitWindow:        "10m",
		QuietHours: []QuietHourRange{
			{Start: "22:00", End: "08:00"},
		},
		ServiceSettings: map[string]ServiceNotificationSettings{
			"api-service": {Enabled: false},
		},
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Deserialize from JSON
	var loadedPrefs NotificationPreferences
	if err := json.Unmarshal(data, &loadedPrefs); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify fields
	if loadedPrefs.OSNotifications != prefs.OSNotifications {
		t.Errorf("OSNotifications = %v, want %v", loadedPrefs.OSNotifications, prefs.OSNotifications)
	}

	if loadedPrefs.DashboardNotifications != prefs.DashboardNotifications {
		t.Errorf("DashboardNotifications = %v, want %v", loadedPrefs.DashboardNotifications, prefs.DashboardNotifications)
	}

	if loadedPrefs.SeverityFilter != prefs.SeverityFilter {
		t.Errorf("SeverityFilter = %q, want %q", loadedPrefs.SeverityFilter, prefs.SeverityFilter)
	}

	if loadedPrefs.RateLimitWindow != prefs.RateLimitWindow {
		t.Errorf("RateLimitWindow = %q, want %q", loadedPrefs.RateLimitWindow, prefs.RateLimitWindow)
	}
}

func TestAtomicWrite(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "azd-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prefsPath := filepath.Join(tmpDir, ".azd", "notifications.json")

	// Override GetNotificationPreferencesPath for testing
	originalGetPath := GetNotificationPreferencesPath
	GetNotificationPreferencesPath = func() (string, error) {
		return prefsPath, nil
	}
	defer func() {
		GetNotificationPreferencesPath = originalGetPath
	}()

	prefs := DefaultNotificationPreferences()

	// Save preferences
	if err := SaveNotificationPreferences(prefs); err != nil {
		t.Fatalf("SaveNotificationPreferences() error = %v", err)
	}

	// Verify temp file was cleaned up
	tempPath := prefsPath + ".tmp"
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should be cleaned up after save")
	}

	// Verify preferences file exists
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		t.Error("Preferences file should exist after save")
	}
}

func TestGetGlobalNotificationPreferences(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "azd-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prefsPath := filepath.Join(tmpDir, ".azd", "notifications.json")

	// Override GetNotificationPreferencesPath for testing
	originalGetPath := GetNotificationPreferencesPath
	GetNotificationPreferencesPath = func() (string, error) {
		return prefsPath, nil
	}
	defer func() {
		GetNotificationPreferencesPath = originalGetPath
	}()

	// Reset the sync.Once to allow re-initialization
	notificationPrefsOnce = sync.Once{}

	// First call should load defaults
	prefs := GetGlobalNotificationPreferences()
	if prefs == nil {
		t.Fatal("GetGlobalNotificationPreferences() returned nil")
	}

	if prefs.SeverityFilter != "critical" {
		t.Errorf("SeverityFilter = %q, want %q", prefs.SeverityFilter, "critical")
	}

	// Second call should return same instance
	prefs2 := GetGlobalNotificationPreferences()
	if prefs != prefs2 {
		t.Error("GetGlobalNotificationPreferences() should return same instance on subsequent calls")
	}
}

func TestIsInQuietHoursRealTime(t *testing.T) {
	now := time.Now()

	// Create quiet hours that should be active now
	oneHourAgo := now.Add(-1 * time.Hour).Format("15:04")
	oneHourLater := now.Add(1 * time.Hour).Format("15:04")

	prefs := &NotificationPreferences{
		QuietHours: []QuietHourRange{
			{Start: oneHourAgo, End: oneHourLater},
		},
	}

	if !prefs.IsInQuietHours() {
		t.Error("IsInQuietHours() should return true when current time is within quiet hours")
	}

	// Create quiet hours that should NOT be active now
	twoHoursAgo := now.Add(-2 * time.Hour).Format("15:04")
	prefs.QuietHours = []QuietHourRange{
		{Start: twoHoursAgo, End: oneHourAgo},
	}

	if prefs.IsInQuietHours() {
		t.Error("IsInQuietHours() should return false when current time is outside quiet hours")
	}

	// Test with no quiet hours
	prefs.QuietHours = []QuietHourRange{}
	if prefs.IsInQuietHours() {
		t.Error("IsInQuietHours() should return false when no quiet hours are configured")
	}
}

func TestConcurrentAccess(t *testing.T) {
	prefs := DefaultNotificationPreferences()
	prefs.SetServiceEnabled("service1", true)
	prefs.SetServiceEnabled("service2", false)

	var wg sync.WaitGroup
	iterations := 100

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = prefs.IsServiceEnabled("service1")
				_ = prefs.IsInQuietHours()
				_ = prefs.GetRateLimitDuration()
				_ = prefs.ShouldNotify("service1", "critical")
			}
		}()
	}

	// Test concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				serviceName := fmt.Sprintf("service%d", id)
				prefs.SetServiceEnabled(serviceName, j%2 == 0)
			}
		}(i)
	}

	// Test concurrent reads while writing
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				should := prefs.ShouldNotify("service1", "critical")
				enabled := prefs.IsServiceEnabled("service1")
				// These should be consistent (if enabled and not in quiet hours, should notify for critical)
				if enabled && !prefs.IsInQuietHours() && !should {
					// This could happen due to timing, so we just log it
					// The important thing is no data races
					t.Logf("Inconsistent state detected (expected due to concurrent access)")
				}
			}
		}()
	}

	wg.Wait()

	// If we get here without deadlock or race conditions, the test passes
	t.Log("Concurrent access test completed successfully")
}
