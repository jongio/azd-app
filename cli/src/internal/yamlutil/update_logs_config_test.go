package yamlutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateServiceLogsConfig_PreservesSchema(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	initialYaml := `# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json

name: test-project

services:
  api:
    host: containerapp
    language: js
    project: ./src/api
`

	// Write initial file
	if err := os.WriteFile(azureYamlPath, []byte(initialYaml), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Update the logs config
	tables := []string{"ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL"}
	if err := UpdateServiceLogsConfig(azureYamlPath, "api", tables, ""); err != nil {
		t.Fatalf("UpdateServiceLogsConfig failed: %v", err)
	}

	// Read the result
	result, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(result)

	// Verify schema is preserved
	if !strings.Contains(resultStr, "# yaml-language-server: $schema=") {
		t.Error("Schema comment was removed")
	}

	// Verify name is preserved
	if !strings.Contains(resultStr, "name: test-project") {
		t.Error("Project name was lost")
	}

	// Verify services section still exists
	if !strings.Contains(resultStr, "services:") {
		t.Error("Services section was lost")
	}

	// Verify api service still exists
	if !strings.Contains(resultStr, "api:") {
		t.Error("Service 'api' was lost")
	}

	// Verify logs config was added
	if !strings.Contains(resultStr, "logs:") {
		t.Error("Logs section was not added")
	}
	if !strings.Contains(resultStr, "analytics:") {
		t.Error("Analytics section was not added")
	}
	if !strings.Contains(resultStr, "tables:") {
		t.Error("Tables field was not added")
	}
	if !strings.Contains(resultStr, "ContainerAppConsoleLogs_CL") {
		t.Error("Table name was not added")
	}

	t.Logf("Result:\n%s", resultStr)
}

func TestUpdateServiceLogsConfig_UpdatesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	initialYaml := `# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json

name: test-project

services:
  api:
    host: containerapp
    language: js
    logs:
      analytics:
        tables:
          - OldTable1
          - OldTable2
`

	if err := os.WriteFile(azureYamlPath, []byte(initialYaml), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Update with new tables
	newTables := []string{"NewTable1", "NewTable2", "NewTable3"}
	if err := UpdateServiceLogsConfig(azureYamlPath, "api", newTables, ""); err != nil {
		t.Fatalf("UpdateServiceLogsConfig failed: %v", err)
	}

	result, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(result)

	// Verify schema is preserved
	if !strings.Contains(resultStr, "# yaml-language-server: $schema=") {
		t.Error("Schema comment was removed")
	}

	// Verify old tables are replaced
	if strings.Contains(resultStr, "OldTable1") {
		t.Error("Old table was not removed")
	}

	// Verify new tables are present
	if !strings.Contains(resultStr, "NewTable1") {
		t.Error("New table was not added")
	}
	if !strings.Contains(resultStr, "NewTable2") {
		t.Error("New table was not added")
	}
	if !strings.Contains(resultStr, "NewTable3") {
		t.Error("New table was not added")
	}

	t.Logf("Result:\n%s", resultStr)
}

func TestUpdateServiceLogsConfig_CustomQuery(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	initialYaml := `# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json

name: test-project

services:
  api:
    host: containerapp
    language: js
`

	if err := os.WriteFile(azureYamlPath, []byte(initialYaml), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Update with custom query
	customQuery := `FunctionAppLogs
| where _ResourceId contains "{serviceName}"
| project TimeGenerated, Message`

	if err := UpdateServiceLogsConfig(azureYamlPath, "api", nil, customQuery); err != nil {
		t.Fatalf("UpdateServiceLogsConfig failed: %v", err)
	}

	result, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(result)

	// Verify schema is preserved
	if !strings.Contains(resultStr, "# yaml-language-server: $schema=") {
		t.Error("Schema comment was removed")
	}

	// Verify query is present
	if !strings.Contains(resultStr, "query: |") {
		t.Error("Query field was not added")
	}
	if !strings.Contains(resultStr, "FunctionAppLogs") {
		t.Error("Query content was not added")
	}

	t.Logf("Result:\n%s", resultStr)
}

func TestUpdateServiceLogsConfig_PreservesOtherServices(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	initialYaml := `# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json

name: test-project

logs:
  filters:
    exclude: ["health"]

services:
  api:
    host: containerapp
    language: js
    ports: ["8080"]
  web:
    host: appservice
    language: python
    ports: ["3000"]
`

	if err := os.WriteFile(azureYamlPath, []byte(initialYaml), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Update only api service
	tables := []string{"ContainerAppLogs"}
	if err := UpdateServiceLogsConfig(azureYamlPath, "api", tables, ""); err != nil {
		t.Fatalf("UpdateServiceLogsConfig failed: %v", err)
	}

	result, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(result)

	// Verify schema is preserved
	if !strings.Contains(resultStr, "# yaml-language-server: $schema=") {
		t.Error("Schema comment was removed")
	}

	// Verify global logs section is preserved
	if !strings.Contains(resultStr, `filters:`) {
		t.Error("Global logs filters were removed")
	}

	// Verify web service is unchanged
	if !strings.Contains(resultStr, "web:") {
		t.Error("Web service was removed")
	}
	if !strings.Contains(resultStr, "appservice") {
		t.Error("Web service host was changed")
	}

	// Verify api service has logs config
	apiIdx := strings.Index(resultStr, "api:")
	webIdx := strings.Index(resultStr, "web:")

	// Find the service-level logs (not the global one)
	serviceLogs := strings.LastIndex(resultStr, "logs:")
	if serviceLogs < apiIdx || serviceLogs > webIdx {
		t.Error("Service-level logs config not in correct position")
	}

	t.Logf("Result:\n%s", resultStr)
}
