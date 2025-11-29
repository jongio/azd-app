package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/fileutil"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"gopkg.in/yaml.v3"
)

var classificationsMu sync.RWMutex

// ClassificationsResponse is the API response for GET /api/logs/classifications
type ClassificationsResponse struct {
	Classifications []service.LogClassification `json:"classifications"`
}

// handleGetClassifications returns all classifications from azure.yaml
func (s *Server) handleGetClassifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	classificationsMu.RLock()
	defer classificationsMu.RUnlock()

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		// If azure.yaml doesn't exist or can't be parsed, return empty list
		log.Printf("Warning: Failed to load azure.yaml: %v", err)
		response := ClassificationsResponse{
			Classifications: []service.LogClassification{},
		}
		if err := writeJSON(w, response); err != nil {
			log.Printf("Failed to write classifications JSON: %v", err)
		}
		return
	}

	classifications := []service.LogClassification{}
	if azureYaml.Logs != nil {
		classifications = azureYaml.Logs.GetClassifications()
	}

	response := ClassificationsResponse{
		Classifications: classifications,
	}

	if err := writeJSON(w, response); err != nil {
		log.Printf("Failed to write classifications JSON: %v", err)
	}
}

// handleCreateClassification adds a new classification to azure.yaml
func (s *Server) handleCreateClassification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var classification service.LogClassification
	if err := json.Unmarshal(body, &classification); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate
	if strings.TrimSpace(classification.Text) == "" {
		writeJSONError(w, http.StatusBadRequest, "Text is required", nil)
		return
	}
	if !service.ValidateClassificationLevel(classification.Level) {
		writeJSONError(w, http.StatusBadRequest, "Level must be 'info', 'warning', or 'error'", nil)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load azure.yaml", err)
		return
	}

	// Initialize logs section if needed
	if azureYaml.Logs == nil {
		azureYaml.Logs = &service.LogsConfig{}
	}

	// Check for existing classification with same text (case-insensitive) and update it
	for i, existing := range azureYaml.Logs.Classifications {
		if strings.EqualFold(existing.Text, classification.Text) {
			// Update existing classification's level
			azureYaml.Logs.Classifications[i].Level = classification.Level
			if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
				return
			}
			w.WriteHeader(http.StatusOK)
			if err := writeJSON(w, azureYaml.Logs.Classifications[i]); err != nil {
				log.Printf("Failed to write classification JSON: %v", err)
			}
			return
		}
	}

	// Add new classification
	azureYaml.Logs.Classifications = append(azureYaml.Logs.Classifications, classification)

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := writeJSON(w, classification); err != nil {
		log.Printf("Failed to write classification JSON: %v", err)
	}
}

// handleDeleteClassification removes a classification by index
func (s *Server) handleDeleteClassification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract index from URL path: /api/logs/classifications/{index}
	indexStr := strings.TrimPrefix(r.URL.Path, "/api/logs/classifications/")
	if indexStr == "" {
		http.Error(w, "Index required", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "Invalid index", http.StatusBadRequest)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load azure.yaml", err)
		return
	}

	if azureYaml.Logs == nil || len(azureYaml.Logs.Classifications) == 0 {
		http.Error(w, "No classifications found", http.StatusNotFound)
		return
	}

	if index < 0 || index >= len(azureYaml.Logs.Classifications) {
		http.Error(w, "Index out of range", http.StatusNotFound)
		return
	}

	// Remove classification at index
	azureYaml.Logs.Classifications = append(
		azureYaml.Logs.Classifications[:index],
		azureYaml.Logs.Classifications[index+1:]...,
	)

	// Clean up empty logs section
	if len(azureYaml.Logs.Classifications) == 0 && azureYaml.Logs.Filters == nil {
		azureYaml.Logs = nil
	}

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save azure.yaml", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleClassificationsRouter routes classification requests
func (s *Server) handleClassificationsRouter(w http.ResponseWriter, r *http.Request) {
	// Handle /api/logs/classifications
	if r.URL.Path == "/api/logs/classifications" {
		switch r.Method {
		case http.MethodGet:
			s.handleGetClassifications(w, r)
		case http.MethodPost:
			s.handleCreateClassification(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle /api/logs/classifications/{index}
	if strings.HasPrefix(r.URL.Path, "/api/logs/classifications/") {
		switch r.Method {
		case http.MethodDelete:
			s.handleDeleteClassification(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// loadAzureYaml loads and parses the azure.yaml file
func loadAzureYaml(projectDir string) (*service.AzureYaml, error) {
	azureYamlPath := filepath.Join(projectDir, "azure.yaml")

	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var azureYaml service.AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	return &azureYaml, nil
}

// saveAzureYaml saves the azure.yaml file atomically
func saveAzureYaml(projectDir string, azureYaml *service.AzureYaml) error {
	azureYamlPath := filepath.Join(projectDir, "azure.yaml")

	// Marshal to YAML with nice formatting
	data, err := yaml.Marshal(azureYaml)
	if err != nil {
		return fmt.Errorf("failed to marshal azure.yaml: %w", err)
	}

	// Write atomically
	if err := fileutil.AtomicWriteFile(azureYamlPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return nil
}
