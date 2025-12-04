package dashboard

import (
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
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

// prefsConfigKey is the key used to store preferences in azdconfig
const prefsConfigKey = "logs"

// getDefaultPreferences returns the default user preferences
func getDefaultPreferences() UserPreferences {
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
	}
}

// loadPreferencesWithClient loads user preferences using the provided config client.
func loadPreferencesWithClient(client azdconfig.ConfigClient) (UserPreferences, error) {
	prefsMu.RLock()
	defer prefsMu.RUnlock()

	data, err := client.GetPreferenceSection(prefsConfigKey)
	if err != nil {
		slog.Debug("failed to load preferences from config", "error", err)
		return getDefaultPreferences(), nil
	}

	if len(data) == 0 {
		return getDefaultPreferences(), nil
	}

	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		slog.Warn("failed to parse preferences, using defaults", "error", err)
		return getDefaultPreferences(), nil
	}

	return prefs, nil
}

// savePreferencesWithClient saves user preferences using the provided config client.
func savePreferencesWithClient(client azdconfig.ConfigClient, prefs UserPreferences) error {
	prefsMu.Lock()
	defer prefsMu.Unlock()

	data, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	return client.SetPreferenceSection(prefsConfigKey, data)
}

// HTTP Handlers

// handleGetPreferences returns user preferences
func (s *Server) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client := s.getOrCreateConfigClient()
	prefs, err := loadPreferencesWithClient(client)
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

	client := s.getOrCreateConfigClient()
	if err := savePreferencesWithClient(client, prefs); err != nil {
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
