package azure

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// mockTokenCredential implements azcore.TokenCredential for testing token cache.
type mockTokenCredential struct {
	token     string
	err       error
	callCount int
	mu        sync.Mutex
	expiresOn time.Time
}

func (m *mockTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return azcore.AccessToken{}, m.err
	}

	expiresOn := m.expiresOn
	if expiresOn.IsZero() {
		expiresOn = time.Now().Add(1 * time.Hour)
	}

	return azcore.AccessToken{
		Token:     m.token,
		ExpiresOn: expiresOn,
	}, nil
}

func (m *mockTokenCredential) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func TestTokenCache_GetSet(t *testing.T) {
	cache := &TokenCache{
		scope: "https://api.loganalytics.io/.default",
	}

	// Initial get should return empty
	token := cache.Get()
	if token != "" {
		t.Errorf("Expected empty token, got: %s", token)
	}

	// Set a token
	cache.Set("test-token-123")

	// Get should return the cached token
	token = cache.Get()
	if token != "test-token-123" {
		t.Errorf("Expected 'test-token-123', got: %s", token)
	}
}

func TestTokenCache_Expiry(t *testing.T) {
	cache := &TokenCache{
		scope: "https://api.loganalytics.io/.default",
	}

	// Set a token
	cache.Set("test-token")

	// Manually set expiry to past
	cache.mu.Lock()
	cache.expiresAt = time.Now().Add(-1 * time.Second)
	cache.mu.Unlock()

	// Get should return empty because token is expired
	token := cache.Get()
	if token != "" {
		t.Errorf("Expected empty token (expired), got: %s", token)
	}
}

func TestTokenCache_Clear(t *testing.T) {
	cache := &TokenCache{
		scope: "https://api.loganalytics.io/.default",
	}

	// Set a token
	cache.Set("test-token")

	// Verify it's cached
	if cache.Get() == "" {
		t.Error("Token should be cached")
	}

	// Clear the cache
	cache.Clear()

	// Get should return empty
	token := cache.Get()
	if token != "" {
		t.Errorf("Expected empty token after clear, got: %s", token)
	}
}

func TestTokenCache_ThreadSafety(t *testing.T) {
	cache := &TokenCache{
		scope: "https://api.loganalytics.io/.default",
	}

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent sets
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			cache.Set("token-" + string(rune('0'+index%10)))
		}(i)
	}

	// Concurrent gets
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cache.Get()
		}()
	}

	// Concurrent clears
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Clear()
		}()
	}

	wg.Wait()

	// No assertion needed - if there's a race condition, the race detector will catch it
}

func TestGetCachedToken_CacheHit(t *testing.T) {
	// Reset global cache
	logAnalyticsTokenCache.Clear()

	mock := &mockTokenCredential{
		token: "fresh-token",
	}

	ctx := context.Background()

	// First call - cache miss, should call credential
	token1, err := GetCachedToken(ctx, mock, "https://api.loganalytics.io/.default")
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}
	if token1 != "fresh-token" {
		t.Errorf("Expected 'fresh-token', got: %s", token1)
	}
	if mock.getCallCount() != 1 {
		t.Errorf("Expected 1 credential call, got: %d", mock.getCallCount())
	}

	// Second call - cache hit, should NOT call credential
	token2, err := GetCachedToken(ctx, mock, "https://api.loganalytics.io/.default")
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}
	if token2 != "fresh-token" {
		t.Errorf("Expected 'fresh-token', got: %s", token2)
	}
	if mock.getCallCount() != 1 {
		t.Errorf("Expected still 1 credential call (cache hit), got: %d", mock.getCallCount())
	}

	// Clean up
	logAnalyticsTokenCache.Clear()
}

func TestGetCachedToken_CacheMiss(t *testing.T) {
	// Reset global cache
	logAnalyticsTokenCache.Clear()

	mock := &mockTokenCredential{
		token: "fresh-token",
	}

	ctx := context.Background()

	// First call
	_, err := GetCachedToken(ctx, mock, "https://api.loganalytics.io/.default")
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}

	// Expire the cache
	logAnalyticsTokenCache.mu.Lock()
	logAnalyticsTokenCache.expiresAt = time.Now().Add(-1 * time.Second)
	logAnalyticsTokenCache.mu.Unlock()

	// Second call - cache expired, should call credential again
	token2, err := GetCachedToken(ctx, mock, "https://api.loganalytics.io/.default")
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}
	if token2 != "fresh-token" {
		t.Errorf("Expected 'fresh-token', got: %s", token2)
	}
	if mock.getCallCount() != 2 {
		t.Errorf("Expected 2 credential calls (cache miss), got: %d", mock.getCallCount())
	}

	// Clean up
	logAnalyticsTokenCache.Clear()
}

func TestGetCachedToken_Error(t *testing.T) {
	// Reset global cache
	logAnalyticsTokenCache.Clear()

	mock := &mockTokenCredential{
		err: errors.New("credential error"),
	}

	ctx := context.Background()

	// Should return error
	_, err := GetCachedToken(ctx, mock, "https://api.loganalytics.io/.default")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if mock.getCallCount() != 1 {
		t.Errorf("Expected 1 credential call, got: %d", mock.getCallCount())
	}

	// Clean up
	logAnalyticsTokenCache.Clear()
}

func TestClearTokenCacheOnError_AuthErrors(t *testing.T) {
	// Reset and populate cache
	logAnalyticsTokenCache.Clear()
	logAnalyticsTokenCache.Set("test-token")

	testCases := []struct {
		name        string
		err         error
		shouldClear bool
	}{
		{
			name:        "401 error",
			err:         errors.New("HTTP 401 Unauthorized"),
			shouldClear: true,
		},
		{
			name:        "403 error",
			err:         errors.New("HTTP 403 Forbidden"),
			shouldClear: true,
		},
		{
			name:        "AADSTS error",
			err:         errors.New("AADSTS50058: token expired"),
			shouldClear: true,
		},
		{
			name:        "AuthenticationFailed",
			err:         errors.New("AuthenticationFailed: invalid credentials"),
			shouldClear: true,
		},
		{
			name:        "Non-auth error",
			err:         errors.New("network timeout"),
			shouldClear: false,
		},
		{
			name:        "Nil error",
			err:         nil,
			shouldClear: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: ensure cache has a token
			logAnalyticsTokenCache.Set("test-token")

			// Act
			ClearTokenCacheOnError(tc.err)

			// Assert
			token := logAnalyticsTokenCache.Get()
			if tc.shouldClear {
				if token != "" {
					t.Errorf("Expected cache to be cleared, but got token: %s", token)
				}
			} else {
				if token == "" {
					t.Error("Expected cache to NOT be cleared, but it was")
				}
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substrs  []string
		expected bool
	}{
		{
			name:     "Single match",
			s:        "HTTP 401 Unauthorized",
			substrs:  []string{"401"},
			expected: true,
		},
		{
			name:     "Multiple substrs, first matches",
			s:        "AADSTS50058 error",
			substrs:  []string{"AADSTS", "401", "403"},
			expected: true,
		},
		{
			name:     "Multiple substrs, last matches",
			s:        "HTTP 403 Forbidden",
			substrs:  []string{"401", "AADSTS", "403"},
			expected: true,
		},
		{
			name:     "No match",
			s:        "network timeout",
			substrs:  []string{"401", "403", "AADSTS"},
			expected: false,
		},
		{
			name:     "Empty string",
			s:        "",
			substrs:  []string{"401"},
			expected: false,
		},
		{
			name:     "Empty substrs",
			s:        "test",
			substrs:  []string{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsAny(tc.s, tc.substrs...)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}
