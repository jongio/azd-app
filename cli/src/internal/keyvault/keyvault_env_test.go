package keyvault

import (
	"context"
	"testing"
)

func TestResolveEnvironmentVariables_NoReferences(t *testing.T) {
	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	envVars := []string{
		"FOO=bar",
		"BAZ=qux",
		"NUMBER=42",
	}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	ctx := context.Background()

	resolved, warnings, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)

	if err != nil {
		t.Errorf("ResolveEnvironmentVariables() with no references returned error: %v", err)
	}

	if len(warnings) > 0 {
		t.Errorf("ResolveEnvironmentVariables() with no references returned warnings: %v", warnings)
	}

	if len(resolved) != len(envVars) {
		t.Errorf("ResolveEnvironmentVariables() returned %d vars, want %d", len(resolved), len(envVars))
	}

	for i, want := range envVars {
		if resolved[i] != want {
			t.Errorf("ResolveEnvironmentVariables() var[%d] = %q, want %q", i, resolved[i], want)
		}
	}
}

func TestResolveEnvironmentVariables_MixedContent(t *testing.T) {
	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	envVars := []string{
		"REGULAR_VAR=some-value",
		"KV_SECRET=@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret)",
		"ANOTHER_VAR=42",
	}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	ctx := context.Background()

	resolved, warnings, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)

	if err != nil {
		t.Errorf("ResolveEnvironmentVariables() with StopOnError=false returned error: %v", err)
	}

	if len(resolved) != len(envVars) {
		t.Errorf("ResolveEnvironmentVariables() returned %d vars, want %d", len(resolved), len(envVars))
	}

	if resolved[0] != envVars[0] {
		t.Errorf("ResolveEnvironmentVariables() var[0] = %q, want %q", resolved[0], envVars[0])
	}

	if resolved[2] != envVars[2] {
		t.Errorf("ResolveEnvironmentVariables() var[2] = %q, want %q", resolved[2], envVars[2])
	}

	if len(warnings) > 0 {
		foundWarning := false
		for _, w := range warnings {
			if w.Key == "KV_SECRET" {
				foundWarning = true
				break
			}
		}
		if !foundWarning {
			t.Errorf("ResolveEnvironmentVariables() returned warnings but none for KV_SECRET: %v", warnings)
		}
	}
}

func TestResolveEnvironmentVariables_InvalidFormat(t *testing.T) {
	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	envVars := []string{
		"INVALID=",
		"=value",
		"NOEQUALS",
		"VALID=foo",
	}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	ctx := context.Background()

	resolved, _, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)

	if err != nil {
		t.Errorf("ResolveEnvironmentVariables() with invalid format returned error: %v", err)
	}

	if len(resolved) != len(envVars) {
		t.Errorf("ResolveEnvironmentVariables() returned %d vars, want %d", len(resolved), len(envVars))
	}

	for i, want := range envVars {
		if resolved[i] != want {
			t.Errorf("ResolveEnvironmentVariables() var[%d] = %q, want %q", i, resolved[i], want)
		}
	}
}

func TestResolveEnvironmentVariables_WithQuotes(t *testing.T) {
	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	envVars := []string{
		"DB_PASSWORD=\"@Microsoft.KeyVault(VaultName=myvault;SecretName=db-pass)\"",
		"API_KEY='@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/api-key)'",
	}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	ctx := context.Background()

	resolved, warnings, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)

	if err != nil {
		t.Errorf("ResolveEnvironmentVariables() with quoted refs returned error: %v", err)
	}

	if len(resolved) != len(envVars) {
		t.Errorf("ResolveEnvironmentVariables() returned %d vars, want %d", len(resolved), len(envVars))
	}

	_ = warnings
}

func TestResolveEnvironmentVariables_EmptyList(t *testing.T) {
	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	envVars := []string{}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	ctx := context.Background()

	resolved, warnings, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)

	if err != nil {
		t.Errorf("ResolveEnvironmentVariables() with empty list returned error: %v", err)
	}

	if len(warnings) > 0 {
		t.Errorf("ResolveEnvironmentVariables() with empty list returned warnings: %v", warnings)
	}

	if len(resolved) != 0 {
		t.Errorf("ResolveEnvironmentVariables() returned %d vars, want 0", len(resolved))
	}
}

func TestResolveEnvironmentVariables_AllFormats(t *testing.T) {
	resolver, err := NewKeyVaultResolver()
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	envVars := []string{
		"FORMAT1=@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/secret1)",
		"FORMAT2=@Microsoft.KeyVault(VaultName=myvault;SecretName=secret2)",
		"FORMAT3=akvs://12345678-1234-1234-1234-123456789abc/myvault/secret3",
		"REGULAR=not-a-secret",
	}

	opts := ResolveEnvironmentOptions{
		StopOnError: false,
	}

	ctx := context.Background()

	resolved, _, err := resolver.ResolveEnvironmentVariables(ctx, envVars, opts)

	if err != nil {
		t.Errorf("ResolveEnvironmentVariables() with all formats returned error: %v", err)
	}

	if len(resolved) != len(envVars) {
		t.Errorf("ResolveEnvironmentVariables() returned %d vars, want %d", len(resolved), len(envVars))
	}

	if resolved[3] != envVars[3] {
		t.Errorf("ResolveEnvironmentVariables() var[3] = %q, want %q", resolved[3], envVars[3])
	}
}
