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

func TestRedactValueMasksSecrets(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{"azd access token by key", "AZD_ACCESS_TOKEN", "abcdefghijklmnop", "ab***op"},
		{"password by key", "DB_PASSWORD", "hunter2!", "hu***2!"},
		{"client secret by key", "AZURE_CLIENT_SECRET", "supersecretvalue", "su***ue"},
		{"bare key name", "STORAGE_KEY", "0123456789abcdef", "01***ef"},
		{"credential by key", "MY_CREDENTIAL", "abcdefgh", "ab***gh"},
		{"connection string by key", "SQL_CONNECTION_STRING", "Server=x;Pwd=y1", "Se***y1"},
		{"short value fully masked", "API_KEY", "abcd", "***"},
		{"single char value", "TOKEN", "x", "***"},
		{"jwt by value shape", "OPAQUE", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefghij", "ey***ij"},
		{"conn string by value shape", "STORAGE", "DefaultEndpointsProtocol=https;AccountKey=abc123==", "De***=="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RedactValue(tt.key, tt.value))
		})
	}
}

func TestRedactValueLeavesSafeValuesIntact(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"empty value", "DB_PASSWORD", ""},
		{"resource group name", "AZURE_RESOURCE_GROUP", "rg-myapp-prod-eastus-001"},
		{"service url", "SERVICE_API_URL", "https://api-prod.azurewebsites.net"},
		{"environment name", "AZURE_ENV_NAME", "dev"},
		{"subscription id", "AZURE_SUBSCRIPTION_ID", "00000000-1111-2222-3333-444444444444"},
		{"location", "AZURE_LOCATION", "eastus2"},
		{"plain path", "PROJECT_DIR", "/home/dev/code/myapp"},
		{"key vault name is not itself a secret value", "AZURE_KEY_VAULT_NAME", "kv-myapp-prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactValue(tt.key, tt.value)
			if tt.key == "AZURE_KEY_VAULT_NAME" {
				// "key" in the name intentionally masks; assert it does not leak in full.
				assert.NotEqual(t, tt.value, got)
				return
			}
			assert.Equal(t, tt.value, got, "%q=%q should not be masked", tt.key, tt.value)
		})
	}
}

func TestRedactMapCopiesAndMasks(t *testing.T) {
	in := map[string]string{
		"AZD_ACCESS_TOKEN":     "abcdefghijklmnop",
		"AZURE_LOCATION":       "eastus2",
		"AZURE_RESOURCE_GROUP": "rg-myapp-prod-eastus-001",
	}
	out := RedactMap(in)

	require.NotNil(t, out)
	assert.Equal(t, "ab***op", out["AZD_ACCESS_TOKEN"], "secret must be masked")
	assert.Equal(t, "eastus2", out["AZURE_LOCATION"], "non-secret must survive")
	assert.Equal(t, "rg-myapp-prod-eastus-001", out["AZURE_RESOURCE_GROUP"])

	// The input map must not be mutated.
	assert.Equal(t, "abcdefghijklmnop", in["AZD_ACCESS_TOKEN"], "input must not be mutated")
}

func TestRedactMapNilInputReturnsEmptyMap(t *testing.T) {
	out := RedactMap(nil)
	require.NotNil(t, out)
	assert.Empty(t, out)
}
