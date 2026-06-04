package dashboard

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/fileutil"
	"gopkg.in/yaml.v3"
)

// classificationsMu serialises azure.yaml read/write access for the Logs
// section. The Connect LogsService classification + preferences adapters
// share this mutex so concurrent edits to classifications and other
// logs-scoped fields cannot interleave.
var classificationsMu sync.RWMutex

// loadAzureYaml loads and parses the azure.yaml file.
func loadAzureYaml(projectDir string) (*service.AzureYaml, error) {
	azureYaml, err := service.ParseAzureYaml(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	return azureYaml, nil
}

// saveAzureYaml saves the azure.yaml file atomically.
func saveAzureYaml(projectDir string, azureYaml *service.AzureYaml) error {
	azureYamlPath := filepath.Join(projectDir, "azure.yaml")

	data, err := yaml.Marshal(azureYaml)
	if err != nil {
		return fmt.Errorf("failed to marshal azure.yaml: %w", err)
	}

	if err := fileutil.AtomicWriteFile(azureYamlPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return nil
}
