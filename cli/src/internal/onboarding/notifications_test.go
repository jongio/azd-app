package onboarding

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/config"
)

// TestNew tests creating a new onboarding instance
func TestNew(t *testing.T) {
	onboarding := New()
	if onboarding == nil {
		t.Fatal("New() returned nil")
	}

	if onboarding.reader == nil {
		t.Error("New() did not initialize reader")
	}
}

// TestShouldRun_ConfigDoesNotExist tests onboarding should run when config doesn't exist
func TestShouldRun_ConfigDoesNotExist(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			if strings.Contains(originalHome, "USERPROFILE") {
				os.Setenv("USERPROFILE", originalHome)
			} else {
				os.Setenv("HOME", originalHome)
			}
		}
	}()

	// Set temp home directory
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir) // For Windows

	onboarding := New()
	ctx := context.Background()

	shouldRun, err := onboarding.ShouldRun(ctx)
	if err != nil {
		t.Fatalf("ShouldRun() error = %v", err)
	}

	if !shouldRun {
		t.Error("ShouldRun() = false, want true when config doesn't exist")
	}
}

// TestShouldRun_ConfigExists tests onboarding should not run when config exists
func TestShouldRun_ConfigExists(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	err := os.MkdirAll(azdDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .azd directory: %v", err)
	}

	// Create notifications config file
	configPath := filepath.Join(azdDir, "notifications.json")
	err = os.WriteFile(configPath, []byte(`{"enabled":true}`), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			if strings.Contains(originalHome, "USERPROFILE") {
				os.Setenv("USERPROFILE", originalHome)
			} else {
				os.Setenv("HOME", originalHome)
			}
		}
	}()

	// Set temp home directory
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir) // For Windows

	onboarding := New()
	ctx := context.Background()

	shouldRun, err := onboarding.ShouldRun(ctx)
	if err != nil {
		t.Fatalf("ShouldRun() error = %v", err)
	}

	if shouldRun {
		t.Error("ShouldRun() = true, want false when config exists")
	}
}

// TestRun_EnableNotificationsAll tests running onboarding with all notifications enabled
func TestRun_EnableNotificationsAll(t *testing.T) {
	// Skip this test in short mode as it involves file I/O
	if testing.Short() {
		t.Skip("skipping onboarding test in short mode")
	}

	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	err := os.MkdirAll(azdDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .azd directory: %v", err)
	}

	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			if strings.Contains(originalHome, "USERPROFILE") {
				os.Setenv("USERPROFILE", originalHome)
			} else {
				os.Setenv("HOME", originalHome)
			}
		}
	}()

	// Set temp home directory
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir) // For Windows

	// Simulate user input: yes, choice 1 (all), no quiet hours
	input := "y\n1\nn\n"
	reader := bufio.NewReader(strings.NewReader(input))

	onboarding := &NotificationOnboarding{
		reader: reader,
	}

	ctx := context.Background()
	err = onboarding.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify config was saved
	prefs, err := config.LoadNotificationPreferences()
	if err != nil {
		t.Fatalf("failed to load saved preferences: %v", err)
	}

	if !prefs.OSNotifications {
		t.Error("OSNotifications should be enabled")
	}

	if prefs.SeverityFilter != "all" {
		t.Errorf("SeverityFilter = %v, want 'all'", prefs.SeverityFilter)
	}

	if len(prefs.QuietHours) != 0 {
		t.Errorf("QuietHours should be empty, got %v", prefs.QuietHours)
	}
}

// TestRun_EnableNotificationsCriticalOnly tests running onboarding with critical only
func TestRun_EnableNotificationsCriticalOnly(t *testing.T) {
	// Skip this test in short mode as it involves file I/O
	if testing.Short() {
		t.Skip("skipping onboarding test in short mode")
	}

	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	err := os.MkdirAll(azdDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .azd directory: %v", err)
	}

	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			if strings.Contains(originalHome, "USERPROFILE") {
				os.Setenv("USERPROFILE", originalHome)
			} else {
				os.Setenv("HOME", originalHome)
			}
		}
	}()

	// Set temp home directory
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir) // For Windows

	// Simulate user input: yes, choice 3 (critical), yes quiet hours
	input := "y\n3\ny\n"
	reader := bufio.NewReader(strings.NewReader(input))

	onboarding := &NotificationOnboarding{
		reader: reader,
	}

	ctx := context.Background()
	err = onboarding.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify config was saved
	prefs, err := config.LoadNotificationPreferences()
	if err != nil {
		t.Fatalf("failed to load saved preferences: %v", err)
	}

	if !prefs.OSNotifications {
		t.Error("OSNotifications should be enabled")
	}

	if prefs.SeverityFilter != "critical" {
		t.Errorf("SeverityFilter = %v, want 'critical'", prefs.SeverityFilter)
	}

	if len(prefs.QuietHours) != 1 {
		t.Fatalf("QuietHours should have 1 entry, got %d", len(prefs.QuietHours))
	}

	if prefs.QuietHours[0].Start != "22:00" || prefs.QuietHours[0].End != "08:00" {
		t.Errorf("QuietHours = %v, want Start='22:00', End='08:00'", prefs.QuietHours[0])
	}
}

// TestRun_DisableNotifications tests running onboarding with notifications disabled
func TestRun_DisableNotifications(t *testing.T) {
	// Skip this test in short mode as it involves file I/O
	if testing.Short() {
		t.Skip("skipping onboarding test in short mode")
	}

	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	err := os.MkdirAll(azdDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .azd directory: %v", err)
	}

	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			if strings.Contains(originalHome, "USERPROFILE") {
				os.Setenv("USERPROFILE", originalHome)
			} else {
				os.Setenv("HOME", originalHome)
			}
		}
	}()

	// Set temp home directory
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir) // For Windows

	// Simulate user input: no
	input := "n\n"
	reader := bufio.NewReader(strings.NewReader(input))

	onboarding := &NotificationOnboarding{
		reader: reader,
	}

	ctx := context.Background()
	err = onboarding.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify config was saved
	prefs, err := config.LoadNotificationPreferences()
	if err != nil {
		t.Fatalf("failed to load saved preferences: %v", err)
	}

	if prefs.OSNotifications {
		t.Error("OSNotifications should be disabled")
	}
}

// TestRun_WarningsOnly tests running onboarding with warnings and critical only
func TestRun_WarningsOnly(t *testing.T) {
	// Skip this test in short mode as it involves file I/O
	if testing.Short() {
		t.Skip("skipping onboarding test in short mode")
	}

	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	err := os.MkdirAll(azdDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .azd directory: %v", err)
	}

	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			if strings.Contains(originalHome, "USERPROFILE") {
				os.Setenv("USERPROFILE", originalHome)
			} else {
				os.Setenv("HOME", originalHome)
			}
		}
	}()

	// Set temp home directory
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir) // For Windows

	// Simulate user input: yes, choice 2 (warnings), no quiet hours
	input := "y\n2\nn\n"
	reader := bufio.NewReader(strings.NewReader(input))

	onboarding := &NotificationOnboarding{
		reader: reader,
	}

	ctx := context.Background()
	err = onboarding.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify config was saved
	prefs, err := config.LoadNotificationPreferences()
	if err != nil {
		t.Fatalf("failed to load saved preferences: %v", err)
	}

	if prefs.SeverityFilter != "warning" {
		t.Errorf("SeverityFilter = %v, want 'warning'", prefs.SeverityFilter)
	}
}
