package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/fileutil"
	"gopkg.in/yaml.v3"
)

var classificationsMu sync.RWMutex

// ClassificationsResponse is the API response for GET /api/logs/classifications
type ClassificationsResponse struct {
	Classifications []service.LogClassification `json:"classifications"`
}

// handleGetClassifications returns all classifications from azure.yaml
func (s *Server) handleGetClassifications(w http.ResponseWriter, _ *http.Request) {
	classificationsMu.RLock()
	defer classificationsMu.RUnlock()

	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		// If azure.yaml doesn't exist or can't be parsed, return empty list
		log.Printf("Warning: Failed to load azure.yaml: %v", err)
		response := ClassificationsResponse{
			Classifications: []service.LogClassification{},
		}
		WriteJSONSuccess(w, response)
		return
	}

	classifications := []service.LogClassification{}
	if azureYaml != nil && azureYaml.Logs != nil {
		classifications = azureYaml.Logs.GetClassifications()
	}

	response := ClassificationsResponse{
		Classifications: classifications,
	}

	WriteJSONSuccess(w, response)
}

// handleCreateClassification adds a new classification to azure.yaml
func (s *Server) handleCreateClassification(w http.ResponseWriter, r *http.Request) {
	var classification service.LogClassification
	if !ReadJSONBody(w, r, &classification, maxRequestBodySize) {
		return
	}

	// Validate
	if strings.TrimSpace(classification.Text) == "" {
		BadRequest(w, errTextRequired, nil)
		return
	}
	if !service.ValidateClassificationLevel(classification.Level) {
		BadRequest(w, "Level must be 'info', 'warning', or 'error'", nil)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		InternalError(w, "Failed to load azure.yaml", err)
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
				HandleSaveError(w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			WriteJSONSuccess(w, azureYaml.Logs.Classifications[i])
			return
		}
	}

	// Add new classification
	azureYaml.Logs.Classifications = append(azureYaml.Logs.Classifications, classification)

	// Save azure.yaml
	if err := saveAzureYaml(s.projectDir, azureYaml); err != nil {
		HandleSaveError(w, err)
		return
	}

	WriteJSONCreated(w, classification)
}

// handleDeleteClassification removes a classification by index
func (s *Server) handleDeleteClassification(w http.ResponseWriter, r *http.Request) {
	// Extract index from URL path: /api/logs/classifications/{index}
	indexStr := strings.TrimPrefix(r.URL.Path, "/api/logs/classifications/")
	if indexStr == "" {
		BadRequest(w, errIndexRequired, nil)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		BadRequest(w, errInvalidIndex, err)
		return
	}

	classificationsMu.Lock()
	defer classificationsMu.Unlock()

	// Load existing azure.yaml
	azureYaml, err := loadAzureYaml(s.projectDir)
	if err != nil {
		InternalError(w, "Failed to load azure.yaml", err)
		return
	}

	if azureYaml.Logs == nil || len(azureYaml.Logs.Classifications) == 0 {
		NotFound(w, errNoClassificationsFound)
		return
	}

	if index < 0 || index >= len(azureYaml.Logs.Classifications) {
		NotFound(w, errIndexOutOfRange)
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
		HandleSaveError(w, err)
		return
	}

	WriteNoContent(w)
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
