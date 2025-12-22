package azure

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

var (
	// Global client cache to reuse Log Analytics clients per workspace
	clientCache   = make(map[string]*LogAnalyticsClient)
	clientCacheMu sync.RWMutex
)

// GetOrCreateLogAnalyticsClient returns a cached Log Analytics client for the workspace ID,
// or creates a new one if not found. This enables HTTP connection pooling and reduces
// overhead from repeated client creation.
//
// Thread-safe using double-checked locking pattern.
func GetOrCreateLogAnalyticsClient(ctx context.Context, credential azcore.TokenCredential, workspaceID string) (*LogAnalyticsClient, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID cannot be empty")
	}

	// Fast path: read lock check
	clientCacheMu.RLock()
	if client, exists := clientCache[workspaceID]; exists {
		clientCacheMu.RUnlock()
		return client, nil
	}
	clientCacheMu.RUnlock()

	// Slow path: write lock and create
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have created it)
	if client, exists := clientCache[workspaceID]; exists {
		return client, nil
	}

	// Create new client
	client, err := NewLogAnalyticsClient(credential, workspaceID)
	if err != nil {
		return nil, err
	}

	// Cache for future use
	clientCache[workspaceID] = client
	return client, nil
}

// ClearClientCache removes all cached clients. Useful for testing or
// forcing credential refresh.
func ClearClientCache() {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()
	clientCache = make(map[string]*LogAnalyticsClient)
}

// GetCachedClientCount returns the number of cached clients. Useful for monitoring.
func GetCachedClientCount() int {
	clientCacheMu.RLock()
	defer clientCacheMu.RUnlock()
	return len(clientCache)
}
