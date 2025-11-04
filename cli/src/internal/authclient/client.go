package authclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Client handles fetching tokens from the authentication server.
type Client struct {
	serverURL    string
	secret       string
	httpClient   *http.Client
	tokenCache   map[string]*CachedToken
	mu           sync.RWMutex
	maxRetries   int
	retryBackoff time.Duration
}

// CachedToken represents a cached token with its expiration time.
type CachedToken struct {
	AccessToken string
	ExpiresAt   time.Time
	Scope       string
}

// TokenResponse represents the response from the token endpoint.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// Config holds the client configuration.
type Config struct {
	ServerURL    string
	Secret       string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
}

// DefaultConfig returns a client configuration with sensible defaults.
func DefaultConfig(serverURL, secret string) *Config {
	return &Config{
		ServerURL:    serverURL,
		Secret:       secret,
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 1 * time.Second,
	}
}

// NewClient creates a new authentication client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.ServerURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}

	if config.Secret == "" {
		return nil, fmt.Errorf("secret is required")
	}

	// Validate server URL
	if _, err := url.Parse(config.ServerURL); err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	return &Client{
		serverURL:    config.ServerURL,
		secret:       config.Secret,
		httpClient:   &http.Client{Timeout: config.Timeout},
		tokenCache:   make(map[string]*CachedToken),
		maxRetries:   config.MaxRetries,
		retryBackoff: config.RetryBackoff,
	}, nil
}

// GetToken fetches an access token for the given scope.
// It uses local caching to minimize requests to the auth server.
func (c *Client) GetToken(scope string) (string, error) {
	if scope == "" {
		scope = "https://management.azure.com/.default"
	}

	// Check cache first
	if token := c.getCachedToken(scope); token != "" {
		return token, nil
	}

	// Fetch new token with retries
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := c.retryBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		token, err := c.fetchToken(scope)
		if err == nil {
			return token, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("failed to get token after %d retries: %w", c.maxRetries, lastErr)
}

// fetchToken fetches a new token from the auth server.
func (c *Client) fetchToken(scope string) (string, error) {
	// Build request URL
	tokenURL := fmt.Sprintf("%s/token?scope=%s", c.serverURL, url.QueryEscape(scope))

	// Create request
	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.secret))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the token
	c.cacheToken(scope, tokenResp.AccessToken, time.Duration(tokenResp.ExpiresIn)*time.Second)

	return tokenResp.AccessToken, nil
}

// getCachedToken retrieves a token from the cache if it's still valid.
func (c *Client) getCachedToken(scope string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.tokenCache[scope]
	if !exists {
		return ""
	}

	// Check if token is expired (with 1 minute buffer)
	if time.Now().Add(time.Minute).After(cached.ExpiresAt) {
		return ""
	}

	return cached.AccessToken
}

// cacheToken stores a token in the cache.
func (c *Client) cacheToken(scope, token string, expiresIn time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokenCache[scope] = &CachedToken{
		AccessToken: token,
		ExpiresAt:   time.Now().Add(expiresIn),
		Scope:       scope,
	}
}

// HealthCheck checks if the auth server is healthy.
func (c *Client) HealthCheck() error {
	healthURL := fmt.Sprintf("%s/health", c.serverURL)

	resp, err := c.httpClient.Get(healthURL)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unhealthy status: %d", resp.StatusCode)
	}

	return nil
}

// ClearCache clears the token cache.
func (c *Client) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokenCache = make(map[string]*CachedToken)
}
