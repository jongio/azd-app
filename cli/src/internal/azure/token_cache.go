package azure

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// TokenCache provides thread-safe caching of Azure access tokens with expiry management.
// This reduces repeated calls to the Azure credential chain by caching tokens for a fixed duration.
type TokenCache struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	scope     string
}

// logAnalyticsTokenCache is the global cache instance for Log Analytics tokens.
var logAnalyticsTokenCache = &TokenCache{
	scope: "https://api.loganalytics.io/.default",
}

// Get returns the cached token if it's still valid, otherwise returns an empty string.
// This method is thread-safe and can be called from multiple goroutines.
func (tc *TokenCache) Get() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// Check if token exists and hasn't expired
	if tc.token == "" || time.Now().After(tc.expiresAt) {
		if os.Getenv("AZD_APP_DEBUG") == valTrue {
			if tc.token == "" {
				slog.Debug("Token cache miss", "reason", "no token", "scope", tc.scope)
			} else {
				slog.Debug("Token cache miss", "reason", "expired", "expiredAt", tc.expiresAt, "scope", tc.scope)
			}
		}
		return ""
	}

	if os.Getenv("AZD_APP_DEBUG") == valTrue {
		slog.Debug("Token cache hit", "expiresIn", time.Until(tc.expiresAt).Round(time.Second), "scope", tc.scope)
	}

	return tc.token
}

// Set stores a token in the cache with a 5-minute expiry.
// This method is thread-safe and can be called from multiple goroutines.
func (tc *TokenCache) Set(token string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.token = token
	// Set expiry to 5 minutes from now
	// Azure tokens are typically valid for 1 hour, so 5 minutes is a safe refresh interval
	tc.expiresAt = time.Now().Add(5 * time.Minute)

	if os.Getenv("AZD_APP_DEBUG") == valTrue {
		slog.Debug("Token cached", "expiresAt", tc.expiresAt, "scope", tc.scope)
	}
}

// Clear invalidates the cached token.
// This should be called when authentication errors occur to force a fresh token fetch.
// This method is thread-safe and can be called from multiple goroutines.
func (tc *TokenCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if os.Getenv("AZD_APP_DEBUG") == valTrue {
		if tc.token != "" {
			slog.Debug("Token cache cleared", "reason", "auth error", "scope", tc.scope)
		}
	}

	tc.token = ""
	tc.expiresAt = time.Time{}
}

// GetCachedToken attempts to get a cached token, or fetches a fresh one if cache miss.
// This is a convenience function that combines cache lookup with credential fetch.
func GetCachedToken(ctx context.Context, credential azcore.TokenCredential, scope string) (string, error) {
	// Try cache first
	token := logAnalyticsTokenCache.Get()
	if token != "" {
		return token, nil
	}

	// Cache miss - fetch fresh token
	accessToken, err := credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Store in cache
	logAnalyticsTokenCache.Set(accessToken.Token)

	return accessToken.Token, nil
}

// ClearTokenCacheOnError clears the token cache if the error indicates an auth issue.
// This ensures the next request will fetch a fresh token.
func ClearTokenCacheOnError(err error) {
	if err == nil {
		return
	}

	errStr := err.Error()
	// Check for common auth error indicators
	if containsAny(errStr, "401", "403", "AADSTS", "AuthenticationFailed", "Unauthorized", "Forbidden") {
		logAnalyticsTokenCache.Clear()
	}
}

// containsAny checks if a string contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
