package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandleGetConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test azure.yaml
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	testContent := `name: test-app
services:
  api:
    host: containerapp
    language: node
`
	if err := os.WriteFile(azureYamlPath, []byte(testContent), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create server
	server := &Server{
		projectDir: tempDir,
		mux:        http.NewServeMux(),
	}
	server.registerEditorRoutes()

	t.Run("successful load", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/config", nil)
		w := httptest.NewRecorder()

		server.handleGetConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ConfigResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Content != testContent {
			t.Errorf("Expected content %q, got %q", testContent, response.Content)
		}

		if response.Path != azureYamlPath {
			t.Errorf("Expected path %q, got %q", azureYamlPath, response.Path)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		emptyDir := t.TempDir()
		server := &Server{
			projectDir: emptyDir,
		}

		req := httptest.NewRequest(http.MethodGet, "/api/editor/config", nil)
		w := httptest.NewRecorder()

		server.handleGetConfig(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestHandleSaveConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create server
	server := &Server{
		projectDir: tempDir,
		mux:        http.NewServeMux(),
	}
	server.registerEditorRoutes()

	newContent := `name: updated-app
services:
  web:
    host: appservice
    language: python
`

	t.Run("successful save", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"content": %q}`, newContent)
		req := httptest.NewRequest(http.MethodPost, "/api/editor/config", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSaveConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response SaveConfigResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		if !response.Written {
			t.Error("Expected written to be true")
		}

		// Verify file was written
		azureYamlPath := filepath.Join(tempDir, "azure.yaml")
		content, err := os.ReadFile(azureYamlPath)
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		if string(content) != newContent {
			t.Errorf("Expected content %q, got %q", newContent, string(content))
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		invalidYaml := `this is: not: valid: yaml: : :`
		reqBody := fmt.Sprintf(`{"content": %q}`, invalidYaml)
		req := httptest.NewRequest(http.MethodPost, "/api/editor/config", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSaveConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("creates backup on save", func(t *testing.T) {
		// Create initial file
		azureYamlPath := filepath.Join(tempDir, "azure.yaml")
		initialContent := `name: initial
services:
  test:
    host: containerapp
`
		if err := os.WriteFile(azureYamlPath, []byte(initialContent), 0600); err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		// Wait a bit to ensure different timestamp
		time.Sleep(time.Second)

		// Save new content
		newContent := `name: modified
services:
  test:
    host: containerapp
    language: node
`
		reqBody := fmt.Sprintf(`{"content": %q}`, newContent)
		req := httptest.NewRequest(http.MethodPost, "/api/editor/config", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSaveConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response SaveConfigResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Backup == "" {
			t.Error("Expected backup path to be set")
		}

		// Verify backup file exists
		if _, err := os.Stat(response.Backup); os.IsNotExist(err) {
			t.Errorf("Backup file does not exist: %s", response.Backup)
		}

		// Verify backup content
		backupContent, err := os.ReadFile(response.Backup)
		if err != nil {
			t.Fatalf("Failed to read backup: %v", err)
		}

		if string(backupContent) != initialContent {
			t.Errorf("Expected backup content %q, got %q", initialContent, string(backupContent))
		}
	})
}

func TestListBackups(t *testing.T) {
	tempDir := t.TempDir()

	server := &Server{
		projectDir: tempDir,
	}

	// Create test backups
	backupTimes := []time.Time{
		time.Date(2026, 1, 11, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 11, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 11, 12, 0, 0, 0, time.UTC),
	}

	for _, backupTime := range backupTimes {
		timestamp := backupTime.Format(timestampFormat)
		backupPath := filepath.Join(tempDir, backupPrefix+timestamp)
		if err := os.WriteFile(backupPath, []byte("backup content"), 0600); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
	}

	t.Run("list backups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/backups", nil)
		w := httptest.NewRecorder()

		server.handleListBackups(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response BackupsListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Backups) != 3 {
			t.Errorf("Expected 3 backups, got %d", len(response.Backups))
		}

		// Verify sorted by timestamp (newest first)
		for i := 0; i < len(response.Backups)-1; i++ {
			if response.Backups[i].Timestamp.Before(response.Backups[i+1].Timestamp) {
				t.Error("Backups not sorted by timestamp (newest first)")
			}
		}
	})
}

func TestRestoreBackup(t *testing.T) {
	tempDir := t.TempDir()

	server := &Server{
		projectDir: tempDir,
	}

	// Create current azure.yaml
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	currentContent := "name: current"
	if err := os.WriteFile(azureYamlPath, []byte(currentContent), 0600); err != nil {
		t.Fatalf("Failed to create current file: %v", err)
	}

	// Create backup
	backupTime := time.Date(2026, 1, 11, 10, 0, 0, 0, time.UTC)
	timestamp := backupTime.Format(timestampFormat)
	backupPath := filepath.Join(tempDir, backupPrefix+timestamp)
	backupContent := "name: backup"
	if err := os.WriteFile(backupPath, []byte(backupContent), 0600); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	t.Run("successful restore", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/editor/backups/"+timestamp+"/restore", nil)
		w := httptest.NewRecorder()

		server.handleRestoreBackup(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response RestoreBackupResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		// Verify file was restored
		content, err := os.ReadFile(azureYamlPath)
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}

		if string(content) != backupContent {
			t.Errorf("Expected content %q, got %q", backupContent, string(content))
		}

		// Verify backup of current was created
		if response.BackupCreated == "" {
			t.Error("Expected backup of current file to be created")
		}
	})

	t.Run("backup not found", func(t *testing.T) {
		invalidTimestamp := "2026-01-01T000000Z"
		req := httptest.NewRequest(http.MethodPost, "/api/editor/backups/"+invalidTimestamp+"/restore", nil)
		w := httptest.NewRecorder()

		server.handleRestoreBackup(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestDeleteBackup(t *testing.T) {
	tempDir := t.TempDir()

	server := &Server{
		projectDir: tempDir,
	}

	// Create backup
	backupTime := time.Date(2026, 1, 11, 10, 0, 0, 0, time.UTC)
	timestamp := backupTime.Format(timestampFormat)
	backupPath := filepath.Join(tempDir, backupPrefix+timestamp)
	if err := os.WriteFile(backupPath, []byte("backup"), 0600); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	t.Run("successful delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/editor/backups/"+timestamp, nil)
		w := httptest.NewRecorder()

		server.handleDeleteBackup(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Verify file was deleted
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Error("Backup file should have been deleted")
		}
	})
}

func TestCleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()

	server := &Server{
		projectDir: tempDir,
	}

	// Create 15 backups (more than maxBackups)
	for i := 0; i < 15; i++ {
		backupTime := time.Date(2026, 1, 11, 10+i, 0, 0, 0, time.UTC)
		timestamp := backupTime.Format(timestampFormat)
		backupPath := filepath.Join(tempDir, backupPrefix+timestamp)
		if err := os.WriteFile(backupPath, []byte(fmt.Sprintf("backup %d", i)), 0600); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
	}

	// Run cleanup
	if err := server.cleanupOldBackups(tempDir); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify only maxBackups remain
	backups, err := server.listBackups(tempDir)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != maxBackups {
		t.Errorf("Expected %d backups after cleanup, got %d", maxBackups, len(backups))
	}

	// Verify newest backups were kept
	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Error("Oldest backups should have been deleted")
		}
	}
}

func TestAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")

	server := &Server{
		projectDir: tempDir,
	}

	t.Run("atomic write success", func(t *testing.T) {
		content := "name: test\nservices:\n  api:\n    host: containerapp\n"

		err := server.writeConfigAtomic(azureYamlPath, content)
		if err != nil {
			t.Fatalf("Atomic write failed: %v", err)
		}

		// Verify content
		written, err := os.ReadFile(azureYamlPath)
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		if string(written) != content {
			t.Errorf("Expected content %q, got %q", content, string(written))
		}
	})

	t.Run("preserves line endings", func(t *testing.T) {
		// Create file with CRLF
		crlfContent := "name: test\r\nservices:\r\n  api:\r\n    host: containerapp\r\n"
		if err := os.WriteFile(azureYamlPath, []byte(crlfContent), 0600); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Update with LF content
		lfContent := "name: updated\nservices:\n  api:\n    host: containerapp\n"
		err := server.writeConfigAtomic(azureYamlPath, lfContent)
		if err != nil {
			t.Fatalf("Atomic write failed: %v", err)
		}

		// Verify CRLF was preserved
		written, err := os.ReadFile(azureYamlPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(written), "\r\n") {
			t.Error("Expected CRLF line endings to be preserved")
		}
	})

	t.Run("validates YAML before commit", func(t *testing.T) {
		invalidYaml := "this is: not: valid: yaml: : :"

		err := server.writeConfigAtomic(azureYamlPath, invalidYaml)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}

		if !strings.Contains(err.Error(), "not valid YAML") {
			t.Errorf("Expected YAML validation error, got: %v", err)
		}
	})
}
