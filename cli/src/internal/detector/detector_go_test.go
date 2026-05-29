package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindGoProjects(t *testing.T) {
	// Create a temp directory structure
	root := t.TempDir()

	// Create a simple Go project
	goDir := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module github.com/test/api\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "main.go"), []byte("package main\n"), 0o644))

	// Create a nested Go project
	nestedDir := filepath.Join(root, "services", "worker")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "go.mod"), []byte("module github.com/test/worker\n\ngo 1.22\n"), 0o644))

	// Create a vendor dir with go.mod that should be skipped
	vendorDir := filepath.Join(root, "api", "vendor", "dep")
	require.NoError(t, os.MkdirAll(vendorDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "go.mod"), []byte("module dep\n"), 0o644))

	// Create a node_modules dir with go.mod that should be skipped
	nmDir := filepath.Join(root, "node_modules", "fake")
	require.NoError(t, os.MkdirAll(nmDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nmDir, "go.mod"), []byte("module fake\n"), 0o644))

	projects, err := FindGoProjects(root)
	require.NoError(t, err)

	assert.Len(t, projects, 2)

	// Check modules were extracted
	modules := make(map[string]string)
	for _, p := range projects {
		modules[p.Module] = p.Dir
	}

	assert.Contains(t, modules, "github.com/test/api")
	assert.Contains(t, modules, "github.com/test/worker")
}

func TestFindGoProjects_Empty(t *testing.T) {
	root := t.TempDir()

	projects, err := FindGoProjects(root)
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestExtractGoModule(t *testing.T) {
	root := t.TempDir()
	goModPath := filepath.Join(root, "go.mod")

	content := `module github.com/example/myapp

go 1.22

require (
	github.com/gin-gonic/gin v1.9.0
)
`
	require.NoError(t, os.WriteFile(goModPath, []byte(content), 0o644))

	module := extractGoModule(goModPath)
	assert.Equal(t, "github.com/example/myapp", module)
}
