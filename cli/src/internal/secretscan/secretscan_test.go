package secretscan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectFlagsLiteralSecrets(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"password literal", "DB_PASSWORD", "hunter2!"},
		{"client secret guid", "AZURE_CLIENT_SECRET", "literalClientSecretValue"},
		{"api key value", "STRIPE_API_KEY", "A1b2C3d4E5f6G7h8I9j0K1l2M3n4"},
		{"jwt token", "AUTH_TOKEN", "eyJhbGciOi.eyJzdWIiOi.SflKxwRJSM"},
		{"hex key by shape", "SIGNING_MATERIAL", "0123456789abcdef0123456789abcdef"},
		{"base64 token by shape", "OPAQUE_VALUE", "QWxhZGRpbjpvcGVuIHNlc2FtZTEyMzQ1"},
		{"connection string", "STORAGE", "DefaultEndpointsProtocol=https;AccountKey=abc123def456ghi=="},
		{"access key", "AWS_ACCESS_KEY", "literalAccessKeyValue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagged, detail := Inspect(tt.key, tt.value)
			assert.True(t, flagged, "expected %q=%q to be flagged", tt.key, tt.value)
			assert.NotEmpty(t, detail)
		})
	}
}

func TestInspectIgnoresSafeValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"empty value", "DB_PASSWORD", ""},
		{"whitespace only", "DB_PASSWORD", "   "},
		{"key vault reference", "DB_PASSWORD", "@Microsoft.KeyVault(SecretUri=https://v.vault.azure.net/secrets/p/1)"},
		{"env indirection braces", "DB_PASSWORD", "${DB_PASSWORD}"},
		{"env indirection dollar", "DB_PASSWORD", "$DB_PASSWORD"},
		{"windows env ref", "DB_PASSWORD", "%DB_PASSWORD%"},
		{"placeholder changeme", "DB_PASSWORD", "changeme"},
		{"placeholder angle", "API_KEY", "<your-api-key>"},
		{"placeholder braces", "API_KEY", "{password}"},
		{"repeated char", "API_KEY", "xxxxxxxx"},
		{"boolean", "FEATURE_TOKEN_ENABLED", "true"},
		{"number", "RETRY_COUNT", "12345"},
		{"key vault name not a secret", "KEY_VAULT_NAME", "my-prod-vault"},
		{"partition key not a secret", "PARTITION_KEY", "tenant-42"},
		{"ssh key path not a secret", "SSH_KEY_PATH", "/home/dev/.ssh/id_rsa"},
		{"secret dir path", "SECRETS_DIR", "/etc/app/secrets"},
		{"non secret plain var", "SERVICE_NAME", "orders-api"},
		{"secret url not literal", "TOKEN_ENDPOINT", "https://login.example.com/token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagged, _ := Inspect(tt.key, tt.value)
			assert.False(t, flagged, "expected %q=%q to be ignored", tt.key, tt.value)
		})
	}
}

func TestScanEnvReturnsSortedFindings(t *testing.T) {
	env := map[string]string{
		"DB_PASSWORD":  "hunter2!",
		"API_KEY":      "A1b2C3d4E5f6G7h8I9j0K1l2M3n4",
		"SERVICE_NAME": "orders-api",
		"KEY_VAULT":    "@Microsoft.KeyVault(SecretUri=https://v.vault.azure.net/secrets/p/1)",
	}

	findings := ScanEnv("azure.yaml (service: api)", env)

	require.Len(t, findings, 2)
	// Sorted by key: API_KEY before DB_PASSWORD.
	assert.Equal(t, "API_KEY", findings[0].Key)
	assert.Equal(t, "DB_PASSWORD", findings[1].Key)
	assert.Equal(t, "azure.yaml (service: api)", findings[0].Source)
}

func TestScanEnvEmpty(t *testing.T) {
	assert.Empty(t, ScanEnv("x", nil))
	assert.Empty(t, ScanEnv("x", map[string]string{}))
}
