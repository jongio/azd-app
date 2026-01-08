package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAzureYaml(t *testing.T) {
	t.Run("FindInCurrentDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create azure.yaml in the temp directory
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0644)
		require.NoError(t, err)

		// Find from the same directory
		found, err := FindAzureYaml(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, azureYamlPath, found)
	})

	t.Run("FindInParentDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create azure.yaml in parent directory
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0644)
		require.NoError(t, err)

		// Create a subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		// Find from subdirectory
		found, err := FindAzureYaml(subDir)
		require.NoError(t, err)
		assert.Equal(t, azureYamlPath, found)
	})

	t.Run("StopAtGitDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a .git directory
		gitDir := filepath.Join(tmpDir, ".git")
		err := os.Mkdir(gitDir, 0755)
		require.NoError(t, err)

		// Create a subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		// azure.yaml does not exist - search should stop at .git
		found, err := FindAzureYaml(subDir)
		require.NoError(t, err)
		assert.Empty(t, found)
	})

	t.Run("NotFound", func(t *testing.T) {
		tmpDir := t.TempDir()

		// No azure.yaml anywhere
		found, err := FindAzureYaml(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, found)
	})

	t.Run("RelativePath", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create azure.yaml
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0644)
		require.NoError(t, err)

		// Find using relative path
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		found, err := FindAzureYaml(".")
		require.NoError(t, err)
		// Compare cleaned absolute paths to be robust across environments
		// Use EvalSymlinks to resolve symlinks (e.g., /var -> /private/var on macOS)
		foundAbs, err := filepath.Abs(found)
		require.NoError(t, err)
		foundAbs, err = filepath.EvalSymlinks(foundAbs)
		require.NoError(t, err)

		expectedAbs, err := filepath.Abs(azureYamlPath)
		require.NoError(t, err)
		expectedAbs, err = filepath.EvalSymlinks(expectedAbs)
		require.NoError(t, err)

		assert.Equal(t, filepath.Clean(expectedAbs), filepath.Clean(foundAbs))
	})
}
