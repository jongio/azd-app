package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortedServiceNames(t *testing.T) {
	services := map[string]service.Service{
		"web": {}, "api": {}, "worker": {},
	}
	assert.Equal(t, []string{"api", "web", "worker"}, sortedServiceNames(services))
	assert.Empty(t, sortedServiceNames(map[string]service.Service{}))
}

func TestDisplayPath(t *testing.T) {
	base := filepath.Join("home", "dev", "app")
	assert.Equal(t, ".env", displayPath(base, filepath.Join(base, ".env")))
}

func TestCollectSecretFindingsFromServices(t *testing.T) {
	services := map[string]service.Service{
		"api": {Environment: service.Environment{
			"DB_PASSWORD":  "hunter2!",
			"SERVICE_NAME": "orders-api",
			"VAULT_REF":    "@Microsoft.KeyVault(SecretUri=https://v.vault.azure.net/secrets/p/1)",
		}},
		"web": {Environment: service.Environment{
			"PUBLIC_URL": "https://localhost:3000",
		}},
	}

	// Use a directory with no .env so only service blocks are scanned.
	findings := collectSecretFindings(services, t.TempDir())

	require.Len(t, findings, 1)
	assert.Equal(t, "DB_PASSWORD", findings[0].Key)
	assert.Equal(t, "azure.yaml (service: api)", findings[0].Source)
}

func TestCollectSecretFindingsFromTrackedEnv(t *testing.T) {
	dir := initGitRepo(t)
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath,
		[]byte("DB_PASSWORD=hunter2!\nSERVICE_NAME=orders-api\n"), 0o600))
	runGit(t, dir, "add", ".env")

	findings := collectSecretFindings(nil, dir)

	require.Len(t, findings, 1)
	assert.Equal(t, "DB_PASSWORD", findings[0].Key)
	assert.Equal(t, ".env", findings[0].Source)
}

func TestCollectSecretFindingsSkipsUntrackedEnv(t *testing.T) {
	dir := initGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"),
		[]byte("DB_PASSWORD=hunter2!\n"), 0o600))
	// Do not add the file to git: an untracked .env must not be scanned.

	assert.Empty(t, collectSecretFindings(nil, dir))
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}
