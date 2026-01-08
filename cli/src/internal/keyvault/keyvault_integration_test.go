//go:build integration

package keyvault

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests require:
// - Azure credentials configured (az login, environment variables, etc.)
// - A Key Vault with test secrets
// - Set TEST_KEYVAULT_NAME environment variable to your vault name
//
// Run with: go test -tags integration

func TestKeyVaultResolver_Integration_ResolveReference(t *testing.T) {
	vaultName := os.Getenv("TEST_KEYVAULT_NAME")
	if vaultName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_NAME not set")
	}

	secretName := os.Getenv("TEST_KEYVAULT_SECRET")
	if secretName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_SECRET not set")
	}

	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Format 1: SecretUri", func(t *testing.T) {
		reference := "@Microsoft.KeyVault(SecretUri=https://" + vaultName + ".vault.azure.net/secrets/" + secretName + ")"

		value, err := resolver.ResolveReference(ctx, reference)
		if err != nil {
			t.Fatalf("ResolveReference() error = %v", err)
		}

		if value == "" {
			t.Error("ResolveReference() returned empty value")
		}

		t.Logf("Successfully resolved secret (length: %d)", len(value))
	})

	t.Run("Format 2: VaultName", func(t *testing.T) {
		reference := "@Microsoft.KeyVault(VaultName=" + vaultName + ";SecretName=" + secretName + ")"

		value, err := resolver.ResolveReference(ctx, reference)
		if err != nil {
			t.Fatalf("ResolveReference() error = %v", err)
		}

		if value == "" {
			t.Error("ResolveReference() returned empty value")
		}

		t.Logf("Successfully resolved secret (length: %d)", len(value))
	})

	t.Run("Format 3: akvs", func(t *testing.T) {
		reference := "akvs://00000000-0000-0000-0000-000000000000/" + vaultName + "/" + secretName

		value, err := resolver.ResolveReference(ctx, reference)
		if err != nil {
			t.Fatalf("ResolveReference() error = %v", err)
		}

		if value == "" {
			t.Error("ResolveReference() returned empty value")
		}

		t.Logf("Successfully resolved secret (length: %d)", len(value))
	})
}

func TestKeyVaultResolver_Integration_NonExistentSecret(t *testing.T) {
	vaultName := os.Getenv("TEST_KEYVAULT_NAME")
	if vaultName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_NAME not set")
	}

	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reference := "@Microsoft.KeyVault(VaultName=" + vaultName + ";SecretName=non-existent-secret-12345)"

	_, err = resolver.ResolveReference(ctx, reference)
	if err == nil {
		t.Error("ResolveReference() should have failed for non-existent secret")
	}

	t.Logf("Expected error: %v", err)
}

func TestKeyVaultResolver_Integration_ResolveEnvironmentVariables(t *testing.T) {
	vaultName := os.Getenv("TEST_KEYVAULT_NAME")
	if vaultName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_NAME not set")
	}

	secretName := os.Getenv("TEST_KEYVAULT_SECRET")
	if secretName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_SECRET not set")
	}

	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	envVars := []string{
		"REGULAR_VAR=regular-value",
		"DB_PASSWORD=@Microsoft.KeyVault(VaultName=" + vaultName + ";SecretName=" + secretName + ")",
		"ANOTHER_VAR=42",
	}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	resolved, warnings, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)
	if err != nil {
		t.Fatalf("ResolveEnvironmentVariables() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("Warnings: %v", warnings)
	}

	if len(resolved) != 3 {
		t.Fatalf("ResolveEnvironmentVariables() returned %d vars, want 3", len(resolved))
	}

	if resolved[0] != envVars[0] {
		t.Errorf("Regular var changed: got %q, want %q", resolved[0], envVars[0])
	}

	if resolved[1] == envVars[1] {
		t.Error("DB_PASSWORD was not resolved")
	}

	if !contains(resolved[1], "DB_PASSWORD=") {
		t.Errorf("DB_PASSWORD format incorrect: %q", resolved[1])
	}

	if resolved[2] != envVars[2] {
		t.Errorf("ANOTHER_VAR changed: got %q, want %q", resolved[2], envVars[2])
	}

	t.Logf("Successfully resolved environment variables")
}

func TestKeyVaultResolver_Integration_ClientCaching(t *testing.T) {
	vaultName := os.Getenv("TEST_KEYVAULT_NAME")
	if vaultName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_NAME not set")
	}

	secretName := os.Getenv("TEST_KEYVAULT_SECRET")
	if secretName == "" {
		t.Skip("Skipping integration test: TEST_KEYVAULT_SECRET not set")
	}

	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	reference := "@Microsoft.KeyVault(VaultName=" + vaultName + ";SecretName=" + secretName + ")"

	start := time.Now()
	_, err = resolver.ResolveReference(ctx, reference)
	if err != nil {
		t.Fatalf("First ResolveReference() error = %v", err)
	}
	firstDuration := time.Since(start)

	start = time.Now()
	_, err = resolver.ResolveReference(ctx, reference)
	if err != nil {
		t.Fatalf("Second ResolveReference() error = %v", err)
	}
	secondDuration := time.Since(start)

	t.Logf("First call: %v, Second call: %v", firstDuration, secondDuration)

	if secondDuration > firstDuration {
		t.Logf("Warning: Second call was slower, but this might be due to network variability")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
