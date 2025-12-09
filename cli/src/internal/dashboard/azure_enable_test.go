package dashboard

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// sampleAzureYaml is a minimal azure.yaml without logs.azure enabled.
const sampleAzureYaml = `name: azure-logs-test

services:
  containerapp-api:
    host: containerapp
    language: js
    project: ./src/containerapp-api
`

func TestHandleEnableAzureLogging(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "azure-enable-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// write sample azure.yaml
	yamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := ioutil.WriteFile(yamlPath, []byte(sampleAzureYaml), 0644); err != nil {
		t.Fatalf("failed to write azure.yaml: %v", err)
	}

	// Create server
	srv := &Server{projectDir: tmpDir}

	// Perform POST /api/azure/enable
	req := httptest.NewRequest(http.MethodPost, "/api/azure/enable", bytes.NewReader(nil))
	w := httptest.NewRecorder()

	srv.handleEnableAzureLogging(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Load file and verify logs.azure.enabled == true
	az, err := loadAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("failed to load azure.yaml after enable: %v", err)
	}
	if az.Logs == nil || az.Logs.Azure == nil || !az.Logs.Azure.Enabled {
		t.Fatalf("azure logging not enabled in azure.yaml: %+v", az.Logs)
	}

	// Verify Azure log buffer was initialized in memory (no restart needed)
	logMgr := service.GetLogManager(tmpDir)
	if logMgr.GetAzureLogBuffer() == nil {
		t.Fatal("Azure log buffer was not initialized - user would need to restart")
	}
}
