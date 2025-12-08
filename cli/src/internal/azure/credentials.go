// Package azure provides Azure cloud integration for log streaming and resource discovery.
package azure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// Common errors for Azure authentication.
var (
	ErrNoCredentials     = errors.New("no Azure credentials available")
	ErrTokenExpired      = errors.New("Azure token has expired")
	ErrAuthNotConfigured = errors.New("Azure authentication not configured. Run 'azd auth login' to authenticate")
)

// AzdTokenCredential implements azcore.TokenCredential using an azd access token.
// This credential is used when running within the azd extension context.
type AzdTokenCredential struct {
	token     string
	expiresOn time.Time
	mu        sync.RWMutex
}

// NewAzdTokenCredential creates a credential from an azd access token.
// The token is expected to come from the AZD_ACCESS_TOKEN environment variable.
func NewAzdTokenCredential(token string) (*AzdTokenCredential, error) {
	if token == "" {
		return nil, ErrNoCredentials
	}
	return &AzdTokenCredential{
		token: token,
		// Assume token is valid for 1 hour if we can't determine expiry
		// In practice, azd tokens typically last longer
		expiresOn: time.Now().Add(1 * time.Hour),
	}, nil
}

// GetToken returns the azd access token as an Azure SDK token.
func (c *AzdTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.token == "" {
		return azcore.AccessToken{}, ErrNoCredentials
	}

	if time.Now().After(c.expiresOn) {
		return azcore.AccessToken{}, ErrTokenExpired
	}

	return azcore.AccessToken{
		Token:     c.token,
		ExpiresOn: c.expiresOn,
	}, nil
}

// CredentialChain provides Azure credentials with multiple fallback options.
// It tries credentials in order: azd token, Azure CLI, environment variables, managed identity.
type CredentialChain struct {
	credential azcore.TokenCredential
	source     string
}

// Source returns the name of the credential source that was used.
func (c *CredentialChain) Source() string {
	return c.source
}

// GetToken returns an Azure access token using the configured credential chain.
func (c *CredentialChain) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if c.credential == nil {
		return azcore.AccessToken{}, ErrNoCredentials
	}
	return c.credential.GetToken(ctx, options)
}

// NewCredentialChain creates a new credential chain that tries multiple credential sources.
// Priority order:
// 1. AZD_ACCESS_TOKEN environment variable (from azd extension context)
// 2. Azure CLI credentials (from 'azd auth login' or 'az login')
// 3. Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
// 4. Managed Identity (when running in Azure)
func NewCredentialChain() (*CredentialChain, error) {
	// Try azd extension token first
	if token := os.Getenv("AZD_ACCESS_TOKEN"); token != "" {
		cred, err := NewAzdTokenCredential(token)
		if err == nil {
			return &CredentialChain{credential: cred, source: "azd-extension"}, nil
		}
	}

	// Try DefaultAzureCredential which includes CLI, env vars, and managed identity
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthNotConfigured, err)
	}

	return &CredentialChain{credential: cred, source: "default-credential-chain"}, nil
}

// NewAzureCredential creates the best available Azure credential.
// It wraps NewCredentialChain and provides a simpler interface.
func NewAzureCredential() (azcore.TokenCredential, error) {
	chain, err := NewCredentialChain()
	if err != nil {
		return nil, err
	}
	return chain, nil
}

// ValidateCredentials tests that the credentials work by requesting a token.
// This can be used to provide early feedback to users about authentication issues.
func ValidateCredentials(ctx context.Context, cred azcore.TokenCredential) error {
	// Request a token for Azure Resource Manager scope
	_, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}
	return nil
}

// GetCredentialSource returns a human-readable description of the credential source.
func GetCredentialSource(cred azcore.TokenCredential) string {
	if chain, ok := cred.(*CredentialChain); ok {
		return chain.Source()
	}
	return "unknown"
}
