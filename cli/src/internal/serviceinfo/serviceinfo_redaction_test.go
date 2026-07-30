package serviceinfo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ServiceInfo.EnvironmentVars is serialized to the dashboard over Connect-RPC
// (rpc.serviceInfoToProto) and to MCP clients (getAppInfoForMCP). It used to
// carry a verbatim copy of the whole process environment, which handed
// AZD_ACCESS_TOKEN and any connection string to the browser and to whatever
// LLM was driving the MCP session.
func TestGetServiceInfoRedactsSecretEnvironmentValues(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "azure.yaml"),
		[]byte("name: test\nservices:\n  api:\n    host: containerapp\n    language: js\n    project: ./api\n"),
		0o600))

	t.Setenv("AZD_ACCESS_TOKEN", "tok_abcdefghijklmnopqrstuvwxyz")
	t.Setenv("SQL_CONNECTION_STRING", "Server=db;User=sa;Password=Sup3rS3cret!")
	t.Setenv("AZURE_CLIENT_SECRET", "clientsecretvaluehere")
	t.Setenv("AZURE_LOCATION", "eastus2")
	t.Setenv("AZURE_RESOURCE_GROUP", "rg-testapp-prod-eastus-001")

	services, err := GetServiceInfo(dir)
	require.NoError(t, err)
	require.NotEmpty(t, services)

	env := services[0].EnvironmentVars
	require.NotNil(t, env)

	for _, secret := range []string{
		"tok_abcdefghijklmnopqrstuvwxyz",
		"Server=db;User=sa;Password=Sup3rS3cret!",
		"clientsecretvaluehere",
	} {
		for k, v := range env {
			assert.NotEqual(t, secret, v, "%s must not expose its raw value", k)
		}
	}

	assert.Contains(t, env["AZD_ACCESS_TOKEN"], "***")
	assert.Contains(t, env["SQL_CONNECTION_STRING"], "***")
	assert.Contains(t, env["AZURE_CLIENT_SECRET"], "***")

	// Non-secret deployment metadata must survive: the dashboard env panel
	// exists to show these.
	assert.Equal(t, "eastus2", env["AZURE_LOCATION"])
	assert.Equal(t, "rg-testapp-prod-eastus-001", env["AZURE_RESOURCE_GROUP"])
}
