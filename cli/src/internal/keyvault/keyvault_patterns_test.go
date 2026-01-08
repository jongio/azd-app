package keyvault

import (
	"testing"
)

func TestSecretURIPattern(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantMatch bool
		wantURI   string
	}{
		{
			name:      "Valid SecretUri with version",
			value:     "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret/abc123)",
			wantMatch: true,
			wantURI:   "https://myvault.vault.azure.net/secrets/my-secret/abc123",
		},
		{
			name:      "Valid SecretUri without version",
			value:     "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret)",
			wantMatch: true,
			wantURI:   "https://myvault.vault.azure.net/secrets/my-secret",
		},
		{
			name:      "Invalid - missing closing paren",
			value:     "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret",
			wantMatch: false,
		},
		{
			name:      "Invalid - wrong parameter name",
			value:     "@Microsoft.KeyVault(Uri=https://myvault.vault.azure.net/secrets/my-secret)",
			wantMatch: false,
		},
		{
			name:      "Not a Key Vault reference",
			value:     "https://myvault.vault.azure.net/secrets/my-secret",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := kvRefSecretURIPattern.FindStringSubmatch(tt.value)
			gotMatch := matches != nil

			if gotMatch != tt.wantMatch {
				t.Errorf("kvRefSecretURIPattern.MatchString(%q) = %v, want %v", tt.value, gotMatch, tt.wantMatch)
			}

			if gotMatch && tt.wantMatch {
				if matches[1] != tt.wantURI {
					t.Errorf("kvRefSecretURIPattern captured URI = %q, want %q", matches[1], tt.wantURI)
				}
			}
		})
	}
}

func TestVaultNamePattern(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		wantMatch      bool
		wantVaultName  string
		wantSecretName string
		wantVersion    string
	}{
		{
			name:           "Valid with version",
			value:          "@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret;SecretVersion=abc123)",
			wantMatch:      true,
			wantVaultName:  "myvault",
			wantSecretName: "my-secret",
			wantVersion:    "abc123",
		},
		{
			name:           "Valid without version",
			value:          "@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret)",
			wantMatch:      true,
			wantVaultName:  "myvault",
			wantSecretName: "my-secret",
			wantVersion:    "",
		},
		{
			name:      "Invalid - missing SecretName",
			value:     "@Microsoft.KeyVault(VaultName=myvault)",
			wantMatch: false,
		},
		{
			name:      "Invalid - missing VaultName",
			value:     "@Microsoft.KeyVault(SecretName=my-secret)",
			wantMatch: false,
		},
		{
			name:      "Invalid - wrong separator",
			value:     "@Microsoft.KeyVault(VaultName=myvault,SecretName=my-secret)",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := kvRefVaultNamePattern.FindStringSubmatch(tt.value)
			gotMatch := matches != nil

			if gotMatch != tt.wantMatch {
				t.Errorf("kvRefVaultNamePattern.MatchString(%q) = %v, want %v", tt.value, gotMatch, tt.wantMatch)
			}

			if gotMatch && tt.wantMatch {
				if matches[1] != tt.wantVaultName {
					t.Errorf("kvRefVaultNamePattern captured VaultName = %q, want %q", matches[1], tt.wantVaultName)
				}
				if matches[2] != tt.wantSecretName {
					t.Errorf("kvRefVaultNamePattern captured SecretName = %q, want %q", matches[2], tt.wantSecretName)
				}
				expectedVersion := tt.wantVersion
				gotVersion := ""
				if len(matches) > 3 {
					gotVersion = matches[3]
				}
				if gotVersion != expectedVersion {
					t.Errorf("kvRefVaultNamePattern captured SecretVersion = %q, want %q", gotVersion, expectedVersion)
				}
			}
		})
	}
}

func TestAzdAkvsPattern(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		wantMatch      bool
		wantGUID       string
		wantVaultName  string
		wantSecretName string
		wantVersion    string
	}{
		{
			name:           "Valid with version",
			value:          "akvs://12345678-1234-1234-1234-123456789abc/myvault/my-secret/v1",
			wantMatch:      true,
			wantGUID:       "12345678-1234-1234-1234-123456789abc",
			wantVaultName:  "myvault",
			wantSecretName: "my-secret",
			wantVersion:    "v1",
		},
		{
			name:           "Valid without version",
			value:          "akvs://12345678-1234-1234-1234-123456789abc/myvault/my-secret",
			wantMatch:      true,
			wantGUID:       "12345678-1234-1234-1234-123456789abc",
			wantVaultName:  "myvault",
			wantSecretName: "my-secret",
			wantVersion:    "",
		},
		{
			name:      "Invalid - missing secret name",
			value:     "akvs://12345678-1234-1234-1234-123456789abc/myvault",
			wantMatch: false,
		},
		{
			name:      "Invalid - missing vault name",
			value:     "akvs://12345678-1234-1234-1234-123456789abc",
			wantMatch: false,
		},
		{
			name:      "Invalid - wrong scheme",
			value:     "https://12345678-1234-1234-1234-123456789abc/myvault/my-secret",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := kvRefAzdAkvsPattern.FindStringSubmatch(tt.value)
			gotMatch := matches != nil

			if gotMatch != tt.wantMatch {
				t.Errorf("kvRefAzdAkvsPattern.MatchString(%q) = %v, want %v", tt.value, gotMatch, tt.wantMatch)
			}

			if gotMatch && tt.wantMatch {
				if matches[1] != tt.wantGUID {
					t.Errorf("kvRefAzdAkvsPattern captured guid = %q, want %q", matches[1], tt.wantGUID)
				}
				if matches[2] != tt.wantVaultName {
					t.Errorf("kvRefAzdAkvsPattern captured vaultName = %q, want %q", matches[2], tt.wantVaultName)
				}
				if matches[3] != tt.wantSecretName {
					t.Errorf("kvRefAzdAkvsPattern captured secretName = %q, want %q", matches[3], tt.wantSecretName)
				}
				expectedVersion := tt.wantVersion
				gotVersion := ""
				if len(matches) > 4 {
					gotVersion = matches[4]
				}
				if gotVersion != expectedVersion {
					t.Errorf("kvRefAzdAkvsPattern captured version = %q, want %q", gotVersion, expectedVersion)
				}
			}
		})
	}
}

func TestAllPatterns_Comprehensive(t *testing.T) {
	// Test that each pattern only matches its own format and not the others
	testCases := []struct {
		value            string
		shouldMatchURI   bool
		shouldMatchVault bool
		shouldMatchAkvs  bool
	}{
		{
			value:            "@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)",
			shouldMatchURI:   true,
			shouldMatchVault: false,
			shouldMatchAkvs:  false,
		},
		{
			value:            "@Microsoft.KeyVault(VaultName=vault;SecretName=name)",
			shouldMatchURI:   false,
			shouldMatchVault: true,
			shouldMatchAkvs:  false,
		},
		{
			value:            "akvs://guid/vault/secret",
			shouldMatchURI:   false,
			shouldMatchVault: false,
			shouldMatchAkvs:  true,
		},
		{
			value:            "not a Key Vault reference",
			shouldMatchURI:   false,
			shouldMatchVault: false,
			shouldMatchAkvs:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			matchedURI := kvRefSecretURIPattern.MatchString(tc.value)
			if matchedURI != tc.shouldMatchURI {
				t.Errorf("SecretURI pattern match = %v, want %v for %q", matchedURI, tc.shouldMatchURI, tc.value)
			}

			matchedVault := kvRefVaultNamePattern.MatchString(tc.value)
			if matchedVault != tc.shouldMatchVault {
				t.Errorf("VaultName pattern match = %v, want %v for %q", matchedVault, tc.shouldMatchVault, tc.value)
			}

			matchedAkvs := kvRefAzdAkvsPattern.MatchString(tc.value)
			if matchedAkvs != tc.shouldMatchAkvs {
				t.Errorf("Akvs pattern match = %v, want %v for %q", matchedAkvs, tc.shouldMatchAkvs, tc.value)
			}
		})
	}
}
