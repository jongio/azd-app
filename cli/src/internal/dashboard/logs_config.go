package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/fileutil"
)

// LogPattern represents a pattern for false positive/negative detection
type LogPattern struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Type        string `json:"type"`   // "positive" or "negative"
	Source      string `json:"source"` // "user" or "app"
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// ClassificationOverride represents a user-defined text classification
type ClassificationOverride struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Level     string `json:"level"` // "info", "warning", or "error"
	CreatedAt string `json:"createdAt"`
}

// PatternsResponse combines patterns and classification overrides
type PatternsResponse struct {
	Patterns  []LogPattern             `json:"patterns,omitempty"`
	Overrides []ClassificationOverride `json:"overrides,omitempty"`
}

// UserPreferences represents user preferences for the logs view
type UserPreferences struct {
	GridColumns int    `json:"gridColumns"`
	PaneHeight  int    `json:"paneHeight"`
	ViewMode    string `json:"viewMode"` // "grid" or "unified"
}

// Pattern/Override source types
const (
	SourceUser = "user"
	SourceApp  = "app"
)

var (
	patternsMu sync.RWMutex
	prefsMu    sync.RWMutex
)

// getUserConfigDir returns the user-level config directory (~/.azure/logs-dashboard/)
func getUserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(home, ".azure", "logs-dashboard")
	if err := fileutil.EnsureDir(configDir); err != nil {
		return "", err
	}
	return configDir, nil
}

// getAppConfigDir returns the app-level config directory (.azure/logs-dashboard/)
func getAppConfigDir(projectDir string) (string, error) {
	configDir := filepath.Join(projectDir, ".azure", "logs-dashboard")
	if err := fileutil.EnsureDir(configDir); err != nil {
		return "", err
	}
	return configDir, nil
}

// loadPatterns loads patterns from both user and app config directories
func loadPatterns(projectDir string) ([]LogPattern, error) {
	patternsMu.RLock()
	defer patternsMu.RUnlock()

	var allPatterns []LogPattern

	// Load user-level patterns
	userConfigDir, err := getUserConfigDir()
	if err != nil {
		log.Printf("Warning: Failed to get user config dir: %v", err)
	} else {
		userPatterns, err := loadPatternsFromDir(userConfigDir, "user")
		if err != nil {
			log.Printf("Warning: Failed to load user patterns: %v", err)
		} else {
			allPatterns = append(allPatterns, userPatterns...)
		}
	}

	// Load app-level patterns
	appConfigDir, err := getAppConfigDir(projectDir)
	if err != nil {
		log.Printf("Warning: Failed to get app config dir: %v", err)
	} else {
		appPatterns, err := loadPatternsFromDir(appConfigDir, "app")
		if err != nil {
			log.Printf("Warning: Failed to load app patterns: %v", err)
		} else {
			allPatterns = append(allPatterns, appPatterns...)
		}
	}

	return allPatterns, nil
}

// loadPatternsFromDir loads patterns from a specific directory
func loadPatternsFromDir(configDir string, source string) ([]LogPattern, error) {
	patternsFile := filepath.Join(configDir, "patterns.json")

	data, err := os.ReadFile(patternsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogPattern{}, nil // Return empty array if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read patterns file: %w", err)
	}

	var patterns []LogPattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, fmt.Errorf("failed to parse patterns JSON: %w", err)
	}

	// Set source for all patterns
	for i := range patterns {
		patterns[i].Source = source
	}

	return patterns, nil
}

// loadPatternsForSource loads patterns for a specific source (user or app).
// This is a helper to reduce code duplication.
func loadPatternsForSource(projectDir, source string) ([]LogPattern, error) {
	var configDir string
	var err error

	if source == SourceUser {
		configDir, err = getUserConfigDir()
	} else {
		configDir, err = getAppConfigDir(projectDir)
	}

	if err != nil {
		return nil, err
	}

	return loadPatternsFromDir(configDir, source)
}

// loadClassificationOverrides loads classification overrides from app config
func loadClassificationOverrides(projectDir string) ([]ClassificationOverride, error) {
	appConfigDir, err := getAppConfigDir(projectDir)
	if err != nil {
		return nil, err
	}

	overridesFile := filepath.Join(appConfigDir, "overrides.json")

	data, err := os.ReadFile(overridesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []ClassificationOverride{}, nil // Return empty array if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read overrides file: %w", err)
	}

	var overrides []ClassificationOverride
	if err := json.Unmarshal(data, &overrides); err != nil {
		return nil, fmt.Errorf("failed to parse overrides JSON: %w", err)
	}

	return overrides, nil
}

// saveClassificationOverrides saves classification overrides to app config
func saveClassificationOverrides(projectDir string, overrides []ClassificationOverride) error {
	appConfigDir, err := getAppConfigDir(projectDir)
	if err != nil {
		return err
	}

	overridesFile := filepath.Join(appConfigDir, "overrides.json")
	return fileutil.AtomicWriteJSON(overridesFile, overrides)
}

// savePatterns saves patterns to the appropriate config directory
func savePatterns(projectDir string, patterns []LogPattern, source string) error {
	patternsMu.Lock()
	defer patternsMu.Unlock()

	var configDir string
	var err error

	if source == SourceUser {
		configDir, err = getUserConfigDir()
	} else {
		configDir, err = getAppConfigDir(projectDir)
	}

	if err != nil {
		return err
	}

	patternsFile := filepath.Join(configDir, "patterns.json")
	return fileutil.AtomicWriteJSON(patternsFile, patterns)
}

// loadPreferences loads user preferences
func loadPreferences() (UserPreferences, error) {
	prefsMu.RLock()
	defer prefsMu.RUnlock()

	userConfigDir, err := getUserConfigDir()
	if err != nil {
		return UserPreferences{}, err
	}

	prefsFile := filepath.Join(userConfigDir, "preferences.json")

	data, err := os.ReadFile(prefsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default preferences
			return UserPreferences{
				GridColumns: 3,
				PaneHeight:  400,
				ViewMode:    "grid",
			}, nil
		}
		return UserPreferences{}, fmt.Errorf("failed to read preferences file: %w", err)
	}

	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return UserPreferences{}, fmt.Errorf("failed to parse preferences JSON: %w", err)
	}

	return prefs, nil
}

// savePreferences saves user preferences
func savePreferences(prefs UserPreferences) error {
	prefsMu.Lock()
	defer prefsMu.Unlock()

	userConfigDir, err := getUserConfigDir()
	if err != nil {
		return err
	}

	prefsFile := filepath.Join(userConfigDir, "preferences.json")
	return fileutil.AtomicWriteJSON(prefsFile, prefs)
}

// HTTP Handlers

// handleGetPatterns returns all patterns (user + app) and classification overrides
func (s *Server) handleGetPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	patterns, err := loadPatterns(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load patterns", err)
		return
	}

	overrides, err := loadClassificationOverrides(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to load overrides: %v", err)
		overrides = []ClassificationOverride{}
	}

	response := PatternsResponse{
		Patterns:  patterns,
		Overrides: overrides,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write patterns JSON: %v", err)
	}
}

// handleCreatePattern creates a new pattern
func (s *Server) handleCreatePattern(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var pattern LogPattern
	if err := json.Unmarshal(body, &pattern); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate source
	if pattern.Source != SourceUser && pattern.Source != SourceApp {
		http.Error(w, "Invalid source (must be 'user' or 'app')", http.StatusBadRequest)
		return
	}

	// Load existing patterns for this source
	existingPatterns, _ := loadPatternsForSource(s.projectDir, pattern.Source)

	// Append new pattern
	existingPatterns = append(existingPatterns, pattern)

	// Save patterns
	if err := savePatterns(s.projectDir, existingPatterns, pattern.Source); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save pattern", err)
		return
	}

	if err := writeJSON(w, pattern); err != nil {
		log.Printf("Failed to write pattern JSON: %v", err)
	}
}

// handleUpdatePattern updates an existing pattern
func (s *Server) handleUpdatePattern(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pattern ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/logs/patterns/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Pattern ID required", http.StatusBadRequest)
		return
	}
	patternID := pathParts[0]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var updates LogPattern
	if err := json.Unmarshal(body, &updates); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Load all patterns to find the one to update
	allPatterns, err := loadPatterns(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load patterns", err)
		return
	}

	// Find pattern and determine source
	var foundPattern *LogPattern
	for i := range allPatterns {
		if allPatterns[i].ID == patternID {
			foundPattern = &allPatterns[i]
			break
		}
	}

	if foundPattern == nil {
		http.Error(w, "Pattern not found", http.StatusNotFound)
		return
	}

	source := foundPattern.Source

	// Load patterns for this source
	sourcePatterns, _ := loadPatternsForSource(s.projectDir, source)

	// Update pattern
	found := false
	for i := range sourcePatterns {
		if sourcePatterns[i].ID == patternID {
			// Update fields
			if updates.Name != "" {
				sourcePatterns[i].Name = updates.Name
			}
			if updates.Pattern != "" {
				sourcePatterns[i].Pattern = updates.Pattern
			}
			if updates.Type != "" {
				sourcePatterns[i].Type = updates.Type
			}
			if updates.Description != "" {
				sourcePatterns[i].Description = updates.Description
			}
			sourcePatterns[i].Enabled = updates.Enabled
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Pattern not found in source", http.StatusNotFound)
		return
	}

	// Save patterns
	if err := savePatterns(s.projectDir, sourcePatterns, source); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save pattern", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeletePattern deletes a pattern
func (s *Server) handleDeletePattern(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pattern ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/logs/patterns/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Pattern ID required", http.StatusBadRequest)
		return
	}
	patternID := pathParts[0]

	// Load all patterns to find the one to delete
	allPatterns, err := loadPatterns(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load patterns", err)
		return
	}

	// Find pattern and determine source
	var foundPattern *LogPattern
	for i := range allPatterns {
		if allPatterns[i].ID == patternID {
			foundPattern = &allPatterns[i]
			break
		}
	}

	if foundPattern == nil {
		http.Error(w, "Pattern not found", http.StatusNotFound)
		return
	}

	source := foundPattern.Source

	// Load patterns for this source
	sourcePatterns, _ := loadPatternsForSource(s.projectDir, source)

	// Remove pattern
	updatedPatterns := make([]LogPattern, 0)
	for _, p := range sourcePatterns {
		if p.ID != patternID {
			updatedPatterns = append(updatedPatterns, p)
		}
	}

	// Save patterns
	if err := savePatterns(s.projectDir, updatedPatterns, source); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save patterns", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetPreferences returns user preferences
func (s *Server) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prefs, err := loadPreferences()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load preferences", err)
		return
	}

	if err := writeJSON(w, prefs); err != nil {
		log.Printf("Failed to write preferences JSON: %v", err)
	}
}

// handleSavePreferences saves user preferences
func (s *Server) handleSavePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var prefs UserPreferences
	if err := json.Unmarshal(body, &prefs); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := savePreferences(prefs); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save preferences", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handlePatternsRouter routes pattern requests based on method and path
func (s *Server) handlePatternsRouter(w http.ResponseWriter, r *http.Request) {
	// Handle /api/logs/patterns
	if r.URL.Path == "/api/logs/patterns" {
		switch r.Method {
		case http.MethodGet:
			s.handleGetPatterns(w, r)
		case http.MethodPost:
			s.handleCreatePattern(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle /api/logs/patterns/overrides
	if r.URL.Path == "/api/logs/patterns/overrides" {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateOverride(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle /api/logs/patterns/overrides/{id}
	if strings.HasPrefix(r.URL.Path, "/api/logs/patterns/overrides/") {
		switch r.Method {
		case http.MethodDelete:
			s.handleDeleteOverride(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle /api/logs/patterns/{id}
	if strings.HasPrefix(r.URL.Path, "/api/logs/patterns/") {
		switch r.Method {
		case http.MethodPatch:
			s.handleUpdatePattern(w, r)
		case http.MethodDelete:
			s.handleDeletePattern(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handlePreferencesRouter routes preference requests based on method
func (s *Server) handlePreferencesRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPreferences(w, r)
	case http.MethodPost:
		s.handleSavePreferences(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateOverride creates a new classification override
func (s *Server) handleCreateOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var override ClassificationOverride
	if err := json.Unmarshal(body, &override); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate level
	if override.Level != "info" && override.Level != "warning" && override.Level != "error" {
		http.Error(w, "Invalid level (must be 'info', 'warning', or 'error')", http.StatusBadRequest)
		return
	}

	// Load existing overrides
	existingOverrides, err := loadClassificationOverrides(s.projectDir)
	if err != nil {
		log.Printf("Warning: Failed to load overrides: %v", err)
		existingOverrides = []ClassificationOverride{}
	}

	// Append new override
	existingOverrides = append(existingOverrides, override)

	// Save overrides
	if err := saveClassificationOverrides(s.projectDir, existingOverrides); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save override", err)
		return
	}

	if err := writeJSON(w, override); err != nil {
		log.Printf("Failed to write override JSON: %v", err)
	}
}

// handleDeleteOverride deletes a classification override
func (s *Server) handleDeleteOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract override ID from URL path
	overrideID := strings.TrimPrefix(r.URL.Path, "/api/logs/patterns/overrides/")
	if overrideID == "" {
		http.Error(w, "Override ID required", http.StatusBadRequest)
		return
	}

	// Load existing overrides
	existingOverrides, err := loadClassificationOverrides(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load overrides", err)
		return
	}

	// Remove override
	updatedOverrides := make([]ClassificationOverride, 0)
	found := false
	for _, o := range existingOverrides {
		if o.ID != overrideID {
			updatedOverrides = append(updatedOverrides, o)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Override not found", http.StatusNotFound)
		return
	}

	// Save overrides
	if err := saveClassificationOverrides(s.projectDir, updatedOverrides); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save overrides", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
