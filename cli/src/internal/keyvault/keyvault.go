// Package keyvault provides Azure Key Vault reference resolution for environment variables.
// It supports three reference formats:
//  1. @Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name[/version])
//  2. @Microsoft.KeyVault(VaultName=vault;SecretName=name[;SecretVersion=version])
//  3. akvs://<guid>/<vault>/<secret>[/<version>]
package keyvault

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// Regex patterns for Key Vault reference formats
var (
	// Format 1: @Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name[/version])
	kvRefSecretURIPattern = regexp.MustCompile(`^@Microsoft\.KeyVault\(SecretUri=(.+)\)$`)

	// Format 2: @Microsoft.KeyVault(VaultName=vault;SecretName=name[;SecretVersion=version])
	kvRefVaultNamePattern = regexp.MustCompile(`^@Microsoft\.KeyVault\(VaultName=([^;]+);SecretName=([^;)]+)(?:;SecretVersion=([^;)]+))?\)$`)

	// Format 3: akvs://<guid>/<vault>/<secret>[/<version>]
	kvRefAzdAkvsPattern = regexp.MustCompile(`^akvs://([^/]+)/([^/]+)/([^/]+)(?:/([^/]+))?$`)
)

// KeyVaultResolver resolves Azure Key Vault references to actual secret values.
type KeyVaultResolver struct {
	credential *azidentity.DefaultAzureCredential
	clients    map[string]*azsecrets.Client
	mu         sync.RWMutex
}

// KeyVaultResolutionWarning represents a warning that occurred during Key Vault resolution.
type KeyVaultResolutionWarning struct {
	Key string // The environment variable key that failed to resolve
	Err error  // The error that occurred
}

// ResolveEnvironmentOptions configures how environment variable resolution behaves.
type ResolveEnvironmentOptions struct {
	StopOnError bool // If true, stop resolution on first error; if false, continue with warnings
}

// NewKeyVaultResolver creates a new Key Vault resolver using DefaultAzureCredential.
func NewKeyVaultResolver() (*KeyVaultResolver, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DefaultAzureCredential: %w", err)
	}

	return &KeyVaultResolver{
		credential: cred,
		clients:    make(map[string]*azsecrets.Client),
	}, nil
}

// IsKeyVaultReference checks if a value is a Key Vault reference.
func IsKeyVaultReference(value string) bool {
	normalized := normalizeKeyVaultReferenceValue(value)

	// Check Format 1: SecretUri
	if kvRefSecretURIPattern.MatchString(normalized) {
		return true
	}

	// Check Format 2: VaultName
	if kvRefVaultNamePattern.MatchString(normalized) {
		return true
	}

	// Check Format 3: akvs
	if strings.HasPrefix(normalized, "akvs://") {
		return kvRefAzdAkvsPattern.MatchString(normalized)
	}

	return false
}

// ResolveReference resolves a single Key Vault reference to its actual value.
func (r *KeyVaultResolver) ResolveReference(ctx context.Context, reference string) (string, error) {
	reference = normalizeKeyVaultReferenceValue(reference)

	// Try SecretUri pattern first
	if matches := kvRefSecretURIPattern.FindStringSubmatch(reference); matches != nil {
		secretURI := strings.TrimSpace(matches[1])
		return r.resolveBySecretURI(ctx, secretURI)
	}

	// Try VaultName pattern
	if matches := kvRefVaultNamePattern.FindStringSubmatch(reference); matches != nil {
		vaultName := matches[1]
		secretName := matches[2]
		version := ""
		if len(matches) > 3 && matches[3] != "" {
			version = matches[3]
		}
		return r.resolveByVaultNameAndSecret(ctx, vaultName, secretName, version)
	}

	// Try azd akvs format
	if strings.HasPrefix(reference, "akvs://") {
		if !kvRefAzdAkvsPattern.MatchString(reference) {
			return "", fmt.Errorf("invalid akvs URI format")
		}

		guid, vaultName, secretName, version, err := parseAzdAkvsURI(reference)
		_ = guid // informational only
		if err != nil {
			return "", err
		}
		return r.resolveByVaultNameAndSecret(ctx, vaultName, secretName, version)
	}

	return "", fmt.Errorf("invalid Key Vault reference format")
}

// ResolveEnvironmentVariables resolves Key Vault references in environment variables.
// Returns the resolved environment variables, any warnings, and an error if StopOnError is true.
func (r *KeyVaultResolver) ResolveEnvironmentVariables(ctx context.Context, envVars []string, options ResolveEnvironmentOptions) ([]string, []KeyVaultResolutionWarning, error) {
	resolved := make([]string, 0, len(envVars))
	var warnings []KeyVaultResolutionWarning

	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			// Not a valid env var, keep as-is
			resolved = append(resolved, envVar)
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if this is a Key Vault reference
		if !IsKeyVaultReference(value) {
			resolved = append(resolved, envVar)
			continue
		}

		// Attempt to resolve the reference
		secretValue, err := r.ResolveReference(ctx, value)
		if err != nil {
			warning := KeyVaultResolutionWarning{
				Key: key,
				Err: err,
			}
			warnings = append(warnings, warning)

			if options.StopOnError {
				return nil, warnings, fmt.Errorf("failed to resolve Key Vault reference for %s: %w", key, err)
			}

			// Keep original reference on error (graceful degradation)
			resolved = append(resolved, envVar)
			continue
		}

		// Successfully resolved
		resolved = append(resolved, fmt.Sprintf("%s=%s", key, secretValue))
	}

	return resolved, warnings, nil
}

// getClient gets or creates a cached client for the given vault URL.
// Uses double-check locking pattern for thread-safe client caching.
func (r *KeyVaultResolver) getClient(vaultURL string) (*azsecrets.Client, error) {
	// Check if client exists (read lock)
	r.mu.RLock()
	if client, ok := r.clients[vaultURL]; ok {
		r.mu.RUnlock()
		return client, nil
	}
	r.mu.RUnlock()

	// Create new client (write lock)
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := r.clients[vaultURL]; ok {
		return client, nil
	}

	client, err := azsecrets.NewClient(vaultURL, r.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client for %s: %w", vaultURL, err)
	}

	r.clients[vaultURL] = client
	return client, nil
}

// resolveBySecretURI resolves a secret using a full secret URI.
func (r *KeyVaultResolver) resolveBySecretURI(ctx context.Context, secretURI string) (string, error) {
	// Parse the URI to extract vault URL and secret name/version
	// Format: https://vault.vault.azure.net/secrets/name[/version]
	parts := strings.Split(secretURI, "/secrets/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid secret URI format: %s", secretURI)
	}

	vaultURL := parts[0]
	secretPath := parts[1]

	// Get or create client for this vault
	client, err := r.getClient(vaultURL)
	if err != nil {
		return "", err
	}

	// Split secret name and version (if present)
	secretParts := strings.Split(secretPath, "/")
	secretName := secretParts[0]
	version := ""
	if len(secretParts) > 1 {
		version = secretParts[1]
	}

	// Get the secret
	var resp azsecrets.GetSecretResponse
	if version != "" {
		resp, err = client.GetSecret(ctx, secretName, version, nil)
	} else {
		resp, err = client.GetSecret(ctx, secretName, "", nil)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get secret %s from %s: %w", secretName, vaultURL, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s has no value", secretName)
	}

	return *resp.Value, nil
}

// resolveByVaultNameAndSecret resolves a secret using vault name and secret name.
func (r *KeyVaultResolver) resolveByVaultNameAndSecret(ctx context.Context, vaultName, secretName, version string) (string, error) {
	// Construct vault URL from vault name
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net", vaultName)

	// Get or create client for this vault
	client, err := r.getClient(vaultURL)
	if err != nil {
		return "", err
	}

	// Get the secret
	var resp azsecrets.GetSecretResponse
	if version != "" {
		resp, err = client.GetSecret(ctx, secretName, version, nil)
	} else {
		resp, err = client.GetSecret(ctx, secretName, "", nil)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get secret %s from vault %s: %w", secretName, vaultName, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s has no value", secretName)
	}

	return *resp.Value, nil
}

// parseAzdAkvsURI parses the azd akvs format URI.
// Format: akvs://<guid>/<vault>/<secret>[/<version>]
func parseAzdAkvsURI(uri string) (guid, vaultName, secretName, version string, err error) {
	matches := kvRefAzdAkvsPattern.FindStringSubmatch(uri)
	if matches == nil {
		return "", "", "", "", fmt.Errorf("invalid akvs URI format: %s", uri)
	}

	guid = matches[1]
	vaultName = matches[2]
	secretName = matches[3]
	if len(matches) > 4 {
		version = matches[4]
	}

	return guid, vaultName, secretName, version, nil
}

// normalizeKeyVaultReferenceValue normalizes a Key Vault reference value.
// This handles cases where azd exports environment variables with wrapper quotes.
func normalizeKeyVaultReferenceValue(value string) string {
	normalized := strings.TrimSpace(value)
	if len(normalized) < 2 {
		return normalized
	}

	first := normalized[0]
	last := normalized[len(normalized)-1]

	// Strip wrapper quotes only when they wrap the entire value
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		normalized = strings.TrimSpace(normalized[1 : len(normalized)-1])
	}

	return normalized
}
