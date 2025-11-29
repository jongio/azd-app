package dashboard

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/fileutil"
)

// maxRequestBodySize is the maximum allowed size for HTTP request bodies (1MB).
// This prevents denial-of-service attacks via excessively large payloads.
const maxRequestBodySize = 1 << 20 // 1MB

// UIPreferences represents UI-related preferences
type UIPreferences struct {
	GridColumns      int      `json:"gridColumns"`
	ViewMode         string   `json:"viewMode"` // "grid" or "unified"
	SelectedServices []string `json:"selectedServices"`
}

// BehaviorPreferences represents behavior-related preferences
type BehaviorPreferences struct {
	AutoScroll      bool   `json:"autoScroll"`
	PauseOnScroll   bool   `json:"pauseOnScroll"`
	TimestampFormat string `json:"timestampFormat"`
}

// CopyPreferences represents copy-related preferences
type CopyPreferences struct {
	DefaultFormat    string `json:"defaultFormat"` // "plaintext", "json", "markdown", "csv"
	IncludeTimestamp bool   `json:"includeTimestamp"`
	IncludeService   bool   `json:"includeService"`
}

// UserPreferences represents user preferences for the logs view
type UserPreferences struct {
	Version  string              `json:"version"`
	UI       UIPreferences       `json:"ui"`
	Behavior BehaviorPreferences `json:"behavior"`
	Copy     CopyPreferences     `json:"copy"`
}

var prefsMu sync.RWMutex

// getUserConfigDir returns the user-level config directory (~/.azure/logs-dashboard/)
func getUserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".azure", "logs-dashboard")
	if err := fileutil.EnsureDir(configDir); err != nil {
		return "", err
	}
	return configDir, nil
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
				Version: "1.0",
				UI: UIPreferences{
					GridColumns:      2,
					ViewMode:         "grid",
					SelectedServices: []string{},
				},
				Behavior: BehaviorPreferences{
					AutoScroll:      true,
					PauseOnScroll:   true,
					TimestampFormat: "hh:mm:ss.sss",
				},
				Copy: CopyPreferences{
					DefaultFormat:    "plaintext",
					IncludeTimestamp: true,
					IncludeService:   true,
				},
			}, nil
		}
		return UserPreferences{}, err
	}

	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return UserPreferences{}, err
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

	// Limit request body size to prevent DoS attacks
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
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
