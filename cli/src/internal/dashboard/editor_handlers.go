package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-core/security"
	"gopkg.in/yaml.v3"
)

const (
	azureYamlFileName    = "azure.yaml"
	backupPrefix         = "azure.yaml.backup."
	maxBackups           = 10
	maxConfigFileSize    = 10 * 1024 * 1024 // 10MB
	maxRequestBodySize   = 10 * 1024 * 1024 // 10MB
	tempFilePattern      = "azure.yaml.tmp.*"
	timestampFormat      = "2006-01-02T150405Z" // ISO8601 compatible with filesystem
	errNoAzureYaml       = "azure.yaml not found"
	errBackupNotFound    = "Backup not found"
	errInvalidTimestamp  = "Invalid timestamp format"
)

// ConfigResponse represents the response for GET /api/editor/config
type ConfigResponse struct {
	Path         string    `json:"path"`
	Content      string    `json:"content"`
	LastModified time.Time `json:"lastModified"`
}

// SaveConfigRequest represents the request body for POST /api/editor/config
type SaveConfigRequest struct {
	Content string `json:"content"`
}

// SaveConfigResponse represents the response for POST /api/editor/config
type SaveConfigResponse struct {
	Success bool   `json:"success"`
	Backup  string `json:"backup"`
	Written bool   `json:"written"`
	Errors  []string `json:"errors,omitempty"`
}

// BackupInfo represents metadata about a backup file
type BackupInfo struct {
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
}

// BackupsListResponse represents the response for GET /api/editor/backups
type BackupsListResponse struct {
	Backups []BackupInfo `json:"backups"`
}

// BackupContentResponse represents the response for GET /api/editor/backups/:timestamp
type BackupContentResponse struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// RestoreBackupResponse represents the response for POST /api/editor/backups/:timestamp/restore
type RestoreBackupResponse struct {
	Success        bool   `json:"success"`
	RestoredFrom   string `json:"restoredFrom"`
	BackupCreated  string `json:"backupCreated"`
}

// handleGetConfig loads and returns the current azure.yaml content
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	azureYamlPath := filepath.Join(s.projectDir, azureYamlFileName)

	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		BadRequest(w, "Invalid path", err)
		return
	}

	// Check if file exists
	info, err := os.Stat(azureYamlPath)
	if os.IsNotExist(err) {
		NotFound(w, errNoAzureYaml)
		return
	}
	if err != nil {
		InternalError(w, "Failed to access azure.yaml", err)
		return
	}

	// Check file size
	if info.Size() > maxConfigFileSize {
		BadRequest(w, "Configuration file too large", fmt.Errorf("file size %d exceeds maximum %d", info.Size(), maxConfigFileSize))
		return
	}

	// Read file content
	// #nosec G304 -- Path validated by security.ValidatePath
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		InternalError(w, "Failed to read azure.yaml", err)
		return
	}

	response := ConfigResponse{
		Path:         azureYamlPath,
		Content:      string(content),
		LastModified: info.ModTime(),
	}

	WriteJSONSuccess(w, response)
}

// handleSaveConfig saves azure.yaml with automatic backup
func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var req SaveConfigRequest
	if !ReadJSONBody(w, r, &req, maxRequestBodySize) {
		return
	}

	// Validate YAML content
	var yamlContent interface{}
	if err := yaml.Unmarshal([]byte(req.Content), &yamlContent); err != nil {
		BadRequest(w, "Invalid YAML content", err)
		return
	}

	azureYamlPath := filepath.Join(s.projectDir, azureYamlFileName)

	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		BadRequest(w, "Invalid path", err)
		return
	}

	var backupPath string
	var errors []string

	// Create backup if file exists
	if _, err := os.Stat(azureYamlPath); err == nil {
		var err error
		backupPath, err = s.createBackup(azureYamlPath)
		if err != nil {
			log.Printf("Warning: Failed to create backup: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to create backup: %v", err))
		}
	}

	// Write content atomically using temp file
	if err := s.writeConfigAtomic(azureYamlPath, req.Content); err != nil {
		InternalError(w, "Failed to save configuration", err)
		return
	}

	// Cleanup old backups
	if err := s.cleanupOldBackups(s.projectDir); err != nil {
		log.Printf("Warning: Failed to cleanup old backups: %v", err)
		errors = append(errors, fmt.Sprintf("Failed to cleanup old backups: %v", err))
	}

	response := SaveConfigResponse{
		Success: true,
		Backup:  backupPath,
		Written: true,
		Errors:  errors,
	}

	WriteJSONSuccess(w, response)
}

// handleListBackups returns a list of all backup files
func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := s.listBackups(s.projectDir)
	if err != nil {
		InternalError(w, "Failed to list backups", err)
		return
	}

	response := BackupsListResponse{
		Backups: backups,
	}

	WriteJSONSuccess(w, response)
}

// handleGetBackup returns the content of a specific backup
func (s *Server) handleGetBackup(w http.ResponseWriter, r *http.Request) {
	// Extract timestamp from URL path
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/api/editor/backups/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		BadRequest(w, "Timestamp required", nil)
		return
	}

	timestampStr := parts[0]
	
	// Parse timestamp
	timestamp, err := time.Parse(timestampFormat, timestampStr)
	if err != nil {
		BadRequest(w, errInvalidTimestamp, err)
		return
	}

	backupPath := filepath.Join(s.projectDir, backupPrefix+timestampStr)

	// Validate path
	if err := security.ValidatePath(backupPath); err != nil {
		BadRequest(w, "Invalid path", err)
		return
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		NotFound(w, errBackupNotFound)
		return
	}

	// Read backup content
	// #nosec G304 -- Path validated by security.ValidatePath
	content, err := os.ReadFile(backupPath)
	if err != nil {
		InternalError(w, "Failed to read backup", err)
		return
	}

	response := BackupContentResponse{
		Content:   string(content),
		Timestamp: timestamp,
	}

	WriteJSONSuccess(w, response)
}

// handleRestoreBackup restores a backup file
func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	// Extract timestamp from URL path
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/api/editor/backups/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		BadRequest(w, "Timestamp required", nil)
		return
	}

	timestampStr := parts[0]
	
	// Verify this is a restore action
	if parts[1] != "restore" {
		BadRequest(w, "Invalid action", nil)
		return
	}

	backupPath := filepath.Join(s.projectDir, backupPrefix+timestampStr)
	azureYamlPath := filepath.Join(s.projectDir, azureYamlFileName)

	// Validate paths
	if err := security.ValidatePath(backupPath); err != nil {
		BadRequest(w, "Invalid backup path", err)
		return
	}
	if err := security.ValidatePath(azureYamlPath); err != nil {
		BadRequest(w, "Invalid path", err)
		return
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		NotFound(w, errBackupNotFound)
		return
	}

	// Read backup content
	// #nosec G304 -- Path validated by security.ValidatePath
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		InternalError(w, "Failed to read backup", err)
		return
	}

	// Create backup of current file before restore
	var currentBackupPath string
	if _, err := os.Stat(azureYamlPath); err == nil {
		currentBackupPath, err = s.createBackup(azureYamlPath)
		if err != nil {
			InternalError(w, "Failed to backup current file", err)
			return
		}
	}

	// Write backup content to azure.yaml atomically
	if err := s.writeConfigAtomic(azureYamlPath, string(backupContent)); err != nil {
		InternalError(w, "Failed to restore backup", err)
		return
	}

	response := RestoreBackupResponse{
		Success:       true,
		RestoredFrom:  backupPath,
		BackupCreated: currentBackupPath,
	}

	WriteJSONSuccess(w, response)
}

// handleDeleteBackup deletes a specific backup file
func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	// Extract timestamp from URL path
	path := r.URL.Path
	timestampStr := strings.TrimPrefix(path, "/api/editor/backups/")
	
	if timestampStr == "" {
		BadRequest(w, "Timestamp required", nil)
		return
	}

	// Validate timestamp format
	if _, err := time.Parse(timestampFormat, timestampStr); err != nil {
		BadRequest(w, errInvalidTimestamp, err)
		return
	}

	backupPath := filepath.Join(s.projectDir, backupPrefix+timestampStr)

	// Validate path
	if err := security.ValidatePath(backupPath); err != nil {
		BadRequest(w, "Invalid path", err)
		return
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		NotFound(w, errBackupNotFound)
		return
	}

	// Delete backup file
	if err := os.Remove(backupPath); err != nil {
		InternalError(w, "Failed to delete backup", err)
		return
	}

	WriteNoContent(w)
}

// handleBackupsRouter routes backup-related requests
func (s *Server) handleBackupsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle /api/editor/backups (list)
	if path == "/api/editor/backups" {
		if r.Method == http.MethodGet {
			s.handleListBackups(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle /api/editor/backups/:timestamp
	if strings.HasPrefix(path, "/api/editor/backups/") {
		timestampPath := strings.TrimPrefix(path, "/api/editor/backups/")
		
		// Check for restore action
		if strings.HasSuffix(timestampPath, "/restore") {
			if r.Method == http.MethodPost {
				s.handleRestoreBackup(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle get or delete backup
		switch r.Method {
		case http.MethodGet:
			s.handleGetBackup(w, r)
		case http.MethodDelete:
			s.handleDeleteBackup(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

// createBackup creates a timestamped backup of azure.yaml
func (s *Server) createBackup(azureYamlPath string) (string, error) {
	// Read current file
	// #nosec G304 -- Path already validated by caller
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Generate timestamp
	timestamp := time.Now().UTC().Format(timestampFormat)
	backupPath := filepath.Join(filepath.Dir(azureYamlPath), backupPrefix+timestamp)

	// Validate backup path
	if err := security.ValidatePath(backupPath); err != nil {
		return "", fmt.Errorf("invalid backup path: %w", err)
	}

	// Get original file permissions
	info, err := os.Stat(azureYamlPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Write backup file with original permissions
	if err := os.WriteFile(backupPath, content, info.Mode()); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// writeConfigAtomic writes content to azure.yaml using atomic write pattern
func (s *Server) writeConfigAtomic(azureYamlPath, content string) error {
	// Validate path
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Get original file permissions (if file exists)
	var fileMode os.FileMode = 0600
	if info, err := os.Stat(azureYamlPath); err == nil {
		fileMode = info.Mode()
	}

	// Detect line endings from original content if file exists
	lineEnding := "\n"
	// #nosec G304 -- Path already validated
	if originalContent, err := os.ReadFile(azureYamlPath); err == nil {
		if strings.Contains(string(originalContent), "\r\n") {
			lineEnding = "\r\n"
		}
	}

	// Normalize line endings in new content
	normalizedContent := content
	if lineEnding == "\r\n" && !strings.Contains(content, "\r\n") {
		// Convert LF to CRLF
		normalizedContent = strings.ReplaceAll(content, "\n", "\r\n")
	} else if lineEnding == "\n" && strings.Contains(content, "\r\n") {
		// Convert CRLF to LF
		normalizedContent = strings.ReplaceAll(content, "\r\n", "\n")
	}

	// Create temp file in same directory
	dir := filepath.Dir(azureYamlPath)
	tempFile, err := os.CreateTemp(dir, tempFilePattern)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		// Clean up temp file if it still exists
		_ = os.Remove(tempPath)
	}()

	// Write content to temp file
	if _, err := tempFile.WriteString(normalizedContent); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Validate written content can be parsed
	// #nosec G304 -- Path is our own temp file
	writtenContent, err := os.ReadFile(tempPath)
	if err != nil {
		return fmt.Errorf("failed to read temp file for validation: %w", err)
	}

	var yamlValidation interface{}
	if err := yaml.Unmarshal(writtenContent, &yamlValidation); err != nil {
		return fmt.Errorf("written content is not valid YAML: %w", err)
	}

	// Set permissions on temp file
	if err := os.Chmod(tempPath, fileMode); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Atomic rename (this is the atomic operation)
	if err := os.Rename(tempPath, azureYamlPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// listBackups returns a list of all backup files sorted by timestamp (newest first)
func (s *Server) listBackups(projectDir string) ([]BackupInfo, error) {
	// Validate path
	if err := security.ValidatePath(projectDir); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Read directory
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var backups []BackupInfo

	// Find all backup files
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), backupPrefix) {
			continue
		}

		timestampStr := strings.TrimPrefix(entry.Name(), backupPrefix)
		timestamp, err := time.Parse(timestampFormat, timestampStr)
		if err != nil {
			// Skip files with invalid timestamp format
			log.Printf("Warning: Invalid backup timestamp format: %s", entry.Name())
			continue
		}

		info, err := entry.Info()
		if err != nil {
			log.Printf("Warning: Failed to get info for backup: %s", entry.Name())
			continue
		}

		backups = append(backups, BackupInfo{
			Timestamp: timestamp,
			Path:      filepath.Join(projectDir, entry.Name()),
			Size:      info.Size(),
		})
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// cleanupOldBackups removes backups beyond the maximum retention count
func (s *Server) cleanupOldBackups(projectDir string) error {
	backups, err := s.listBackups(projectDir)
	if err != nil {
		return err
	}

	// Delete backups beyond maxBackups
	for i := maxBackups; i < len(backups); i++ {
		if err := os.Remove(backups[i].Path); err != nil {
			log.Printf("Warning: Failed to remove old backup %s: %v", backups[i].Path, err)
			// Continue with other backups even if one fails
		}
	}

	return nil
}

// registerEditorRoutes adds editor API routes to the server mux
func (s *Server) registerEditorRoutes() {
	// Ensure endpointLimiter is initialized for middleware usage
	if s.endpointLimiter == nil {
		s.endpointLimiter = NewEndpointRateLimits()
	}

	s.mux.HandleFunc("/api/editor/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.handleGetConfig(w, r)
		case http.MethodPost:
			// Rate limit saves to avoid rapid writes
			RateLimitMiddleware(s.endpointLimiter, s.handleSaveConfig)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	s.mux.HandleFunc("/api/editor/backups", s.handleBackupsRouter)
	s.mux.HandleFunc("/api/editor/backups/", s.handleBackupsRouter)
	
	// Well-known services endpoints
	s.mux.HandleFunc("/api/editor/wellknown", s.handleWellKnownRouter)
	s.mux.HandleFunc("/api/editor/wellknown/", s.handleWellKnownRouter)

	// Schema and validation endpoints
	s.registerSchemaRoutes()
	s.registerValidationRoutes()
}
