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
	ErrTokenExpired      = errors.New("azure token has expired")
	ErrAuthNotConfigured = errors.New("azure authentication not configured. run 'azd auth login' to authenticate")
)

const wrapAuthFmt = "%w: %v"

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
// 2. Azure Developer CLI credentials (from 'azd auth login')
// 3. Azure CLI credentials (from 'az login')
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

	// Prefer Azure Developer CLI credential (azd auth login) when available, then fall back
	// to DefaultAzureCredential (Azure CLI, env vars, managed identity, etc.).
	azdCred, err := azidentity.NewAzureDeveloperCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf(wrapAuthFmt, ErrAuthNotConfigured, err)
	}

	defaultCred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf(wrapAuthFmt, ErrAuthNotConfigured, err)
	}

	chained, err := azidentity.NewChainedTokenCredential([]azcore.TokenCredential{azdCred, defaultCred}, nil)
	if err != nil {
		return nil, fmt.Errorf(wrapAuthFmt, ErrAuthNotConfigured, err)
	}

	return &CredentialChain{credential: chained, source: "azd+default-credential-chain"}, nil
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

// NewLogAnalyticsCredential creates credentials specifically for Log Analytics API.
// This skips the AZD_ACCESS_TOKEN because that token is typically scoped to ARM
// and won't work for Log Analytics API (api.loganalytics.io) which requires
// a different audience. Instead, it uses DefaultAzureCredential which can
// obtain tokens for any requested scope.
//
// AUTH SCOPE LIMITATION:
// The Log Analytics API requires scope 'https://api.loganalytics.io/.default'.
// However, most Azure SDK clients use 'https://management.azure.com/.default'.
// This creates a challenge:
// 1. The azlogs SDK client expects credentials that provide Log Analytics tokens
// 2. BUT the SDK's internal token request doesn't specify scope explicitly
// 3. DefaultAzureCredential defaults to the last requested scope
//
// WORKAROUND:
//   - We rely on Azure CLI (via DefaultAzureCredential) which can handle
//     multiple scopes by caching separate tokens per resource
//   - The azlogs.Client automatically requests the correct scope internally
//   - Users must be logged in via `az login` for this to work reliably
//
// FUTURE FIX:
// - Azure SDK for Go should expose scope configuration on azlogs.Client
// - Or provide a dedicated LogAnalyticsCredential type that handles scope internally
func NewLogAnalyticsCredential() (azcore.TokenCredential, error) {
	// Prefer Azure Developer CLI credential (azd auth login). This obtains tokens for
	// arbitrary scopes via `azd auth token --scope ...`.
	azdCred, err := azidentity.NewAzureDeveloperCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf(wrapAuthFmt, ErrAuthNotConfigured, err)
	}

	// Fall back to DefaultAzureCredential (Azure CLI, env vars, managed identity, etc.).
	defaultCred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf(wrapAuthFmt, ErrAuthNotConfigured, err)
	}

	cred, err := azidentity.NewChainedTokenCredential([]azcore.TokenCredential{azdCred, defaultCred}, nil)
	if err != nil {
		return nil, fmt.Errorf(wrapAuthFmt, ErrAuthNotConfigured, err)
	}

	return cred, nil
}

// ValidateCredentials tests that the credentials work by requesting a token.
// This can be used to provide early feedback to users about authentication issues.
func ValidateCredentials(ctx context.Context, cred azcore.TokenCredential) error {
	// Validate ARM scope (needed for discovery / workspace resolution).
	if _, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	}); err != nil {
		return fmt.Errorf("credential validation failed for Azure Resource Manager scope: %w", err)
	}

	// Validate Log Analytics scope (needed for api.loganalytics.io queries).
	if _, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://api.loganalytics.io/.default"},
	}); err != nil {
		return fmt.Errorf("credential validation failed for Log Analytics scope: %w", err)
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
