package keyvault

import (
	"testing"
)

func TestIsKeyVaultReference(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "Format 1: SecretUri with version",
			value: "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret/abc123)",
			want:  true,
		},
		{
			name:  "Format 1: SecretUri without version",
			value: "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret)",
			want:  true,
		},
		{
			name:  "Format 2: VaultName with version",
			value: "@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret;SecretVersion=abc123)",
			want:  true,
		},
		{
			name:  "Format 2: VaultName without version",
			value: "@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret)",
			want:  true,
		},
		{
			name:  "Format 3: akvs with version",
			value: "akvs://12345678-1234-1234-1234-123456789abc/myvault/my-secret/abc123",
			want:  true,
		},
		{
			name:  "Format 3: akvs without version",
			value: "akvs://12345678-1234-1234-1234-123456789abc/myvault/my-secret",
			want:  true,
		},
		{
			name:  "Not a Key Vault reference",
			value: "just a regular value",
			want:  false,
		},
		{
			name:  "Empty string",
			value: "",
			want:  false,
		},
		{
			name:  "Invalid format - missing closing paren",
			value: "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret",
			want:  false,
		},
		{
			name:  "Invalid akvs format - missing parts",
			value: "akvs://guid/vault",
			want:  false,
		},
		{
			name:  "Format 1 with wrapper quotes",
			value: `"@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret)"`,
			want:  true,
		},
		{
			name:  "Format 1 with single quotes",
			value: `'@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret)'`,
			want:  true,
		},
		{
			name:  "Format 2 with leading/trailing whitespace",
			value: "  @Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret)  ",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsKeyVaultReference(tt.value)
			if got != tt.want {
				t.Errorf("IsKeyVaultReference(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeKeyVaultReferenceValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "No quotes",
			value: "@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)",
			want:  "@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)",
		},
		{
			name:  "Double quotes",
			value: `"@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)"`,
			want:  "@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)",
		},
		{
			name:  "Single quotes",
			value: `'@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)'`,
			want:  "@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)",
		},
		{
			name:  "Leading/trailing whitespace",
			value: "  @Microsoft.KeyVault(VaultName=vault;SecretName=secret)  ",
			want:  "@Microsoft.KeyVault(VaultName=vault;SecretName=secret)",
		},
		{
			name:  "Quotes with whitespace",
			value: `  "@Microsoft.KeyVault(VaultName=vault;SecretName=secret)"  `,
			want:  "@Microsoft.KeyVault(VaultName=vault;SecretName=secret)",
		},
		{
			name:  "Empty string",
			value: "",
			want:  "",
		},
		{
			name:  "Single character",
			value: "a",
			want:  "a",
		},
		{
			name:  "Mismatched quotes - not stripped",
			value: `"@Microsoft.KeyVault(VaultName=vault;SecretName=secret)'`,
			want:  `"@Microsoft.KeyVault(VaultName=vault;SecretName=secret)'`,
		},
		{
			name:  "Internal quotes - not stripped",
			value: `@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/"name")`,
			want:  `@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/"name")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeKeyVaultReferenceValue(tt.value)
			if got != tt.want {
				t.Errorf("normalizeKeyVaultReferenceValue(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseAzdAkvsURI(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		wantGUID       string
		wantVaultName  string
		wantSecretName string
		wantVersion    string
		wantErr        bool
	}{
		{
			name:           "Valid with version",
			uri:            "akvs://12345678-1234-1234-1234-123456789abc/myvault/my-secret/v1",
			wantGUID:       "12345678-1234-1234-1234-123456789abc",
			wantVaultName:  "myvault",
			wantSecretName: "my-secret",
			wantVersion:    "v1",
			wantErr:        false,
		},
		{
			name:           "Valid without version",
			uri:            "akvs://12345678-1234-1234-1234-123456789abc/myvault/my-secret",
			wantGUID:       "12345678-1234-1234-1234-123456789abc",
			wantVaultName:  "myvault",
			wantSecretName: "my-secret",
			wantVersion:    "",
			wantErr:        false,
		},
		{
			name:    "Invalid - missing secret name",
			uri:     "akvs://12345678-1234-1234-1234-123456789abc/myvault",
			wantErr: true,
		},
		{
			name:    "Invalid - empty uri",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "Invalid - wrong scheme",
			uri:     "https://12345678-1234-1234-1234-123456789abc/myvault/my-secret",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guid, vaultName, secretName, version, err := parseAzdAkvsURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAzdAkvsURI(%q) error = %v, wantErr %v", tt.uri, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if guid != tt.wantGUID {
				t.Errorf("parseAzdAkvsURI(%q) guid = %q, want %q", tt.uri, guid, tt.wantGUID)
			}
			if vaultName != tt.wantVaultName {
				t.Errorf("parseAzdAkvsURI(%q) vaultName = %q, want %q", tt.uri, vaultName, tt.wantVaultName)
			}
			if secretName != tt.wantSecretName {
				t.Errorf("parseAzdAkvsURI(%q) secretName = %q, want %q", tt.uri, secretName, tt.wantSecretName)
			}
			if version != tt.wantVersion {
				t.Errorf("parseAzdAkvsURI(%q) version = %q, want %q", tt.uri, version, tt.wantVersion)
			}
		})
	}
}

func TestKeyVaultResolver_New(t *testing.T) {
	// This test just verifies that we can create a resolver without panic.
	// Actual Azure authentication may fail without credentials, which is expected.
	resolver, err := NewKeyVaultResolver()

	// We don't assert on error here because it depends on the environment.
	// In CI/CD without Azure credentials, this will fail, which is OK.
	// The important part is that the function doesn't panic.

	if err == nil && resolver == nil {
		t.Error("NewKeyVaultResolver() returned nil resolver without error")
	}

	if resolver != nil {
		if resolver.clients == nil {
			t.Error("NewKeyVaultResolver() created resolver with nil clients map")
		}
	}
}

func TestKeyVaultResolutionWarning(t *testing.T) {
	// Test that the warning struct can be created and used
	warning := KeyVaultResolutionWarning{
		Key: "MY_SECRET",
		Err: nil,
	}

	if warning.Key != "MY_SECRET" {
		t.Errorf("KeyVaultResolutionWarning.Key = %q, want %q", warning.Key, "MY_SECRET")
	}
}

func TestResolveEnvironmentOptions(t *testing.T) {
	// Test that the options struct can be created and used
	opts := ResolveEnvironmentOptions{
		StopOnError: true,
	}

	if !opts.StopOnError {
		t.Error("ResolveEnvironmentOptions.StopOnError = false, want true")
	}

	opts2 := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	if opts2.StopOnError {
		t.Error("ResolveEnvironmentOptions.StopOnError = true, want false")
	}
}
