package authserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/gin-gonic/gin"
	gocache "github.com/patrickmn/go-cache"
)

// Handler manages HTTP request handling for the authentication server.
type Handler struct {
	config      *Config
	jwtManager  *JWTManager
	tokenCache  *gocache.Cache
	credential  azcore.TokenCredential
	rateLimiter *RateLimiter
	mu          sync.RWMutex
}

// NewHandler creates a new HTTP handler for the auth server.
func NewHandler(config *Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create Azure credential using azd's authentication
	// This uses the default Azure credential chain which includes azd auth login
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	return &Handler{
		config:      config,
		jwtManager:  NewJWTManager(config.SharedSecret),
		tokenCache:  gocache.New(config.CacheExpiry, 10*time.Minute),
		credential:  credential,
		rateLimiter: NewRateLimiter(config.RateLimitRequests),
	}, nil
}

// SetupRoutes configures the HTTP routes.
func (h *Handler) SetupRoutes(router *gin.Engine) {
	// Middleware for authentication
	router.Use(h.authMiddleware())
	
	// Token endpoint
	router.GET("/token", h.handleGetToken)
	
	// Health check endpoint (no auth required for this)
	router.GET("/health", h.handleHealth)
}

// authMiddleware validates the shared secret in the Authorization header.
func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health check
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Check rate limiting
		clientIP := c.ClientIP()
		if !h.rateLimiter.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Validate Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing Authorization header",
			})
			c.Abort()
			return
		}

		// Expected format: "Bearer <secret>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid Authorization header format",
			})
			c.Abort()
			return
		}

		// Constant-time comparison to prevent timing attacks
		if !secureCompare(parts[1], h.config.SharedSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// handleGetToken returns an Azure access token wrapped in a JWT.
func (h *Handler) handleGetToken(c *gin.Context) {
	// Get the requested scope (default to Azure Resource Manager)
	scope := c.Query("scope")
	if scope == "" {
		scope = "https://management.azure.com/.default"
	}

	// Validate scope format
	if !strings.HasSuffix(scope, "/.default") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "scope must end with /.default",
		})
		return
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("token:%s", scope)
	if cachedToken, found := h.tokenCache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cachedToken)
		return
	}

	// Request new token from Azure
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := h.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get Azure token: %v", err),
		})
		return
	}

	// Create JWT token wrapping the Azure token
	jwtToken, err := h.jwtManager.CreateToken(token.Token, scope, h.config.TokenExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create JWT token: %v", err),
		})
		return
	}

	// Calculate expiry duration
	expiresIn := int(h.config.TokenExpiry.Seconds())

	response := gin.H{
		"access_token": jwtToken,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
		"scope":        scope,
	}

	// Cache the response
	h.tokenCache.Set(cacheKey, response, h.config.TokenExpiry-time.Minute)

	c.JSON(http.StatusOK, response)
}

// handleHealth returns the server health status.
func (h *Handler) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// secureCompare performs a constant-time comparison of two strings.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	
	return result == 0
}

// RateLimiter implements simple rate limiting per client IP.
type RateLimiter struct {
	requests map[string]*clientLimit
	mu       sync.RWMutex
	limit    int
}

type clientLimit struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientLimit),
		limit:    requestsPerMinute,
	}
	
	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	
	return rl
}

// Allow checks if a request from the given IP should be allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	
	limit, exists := rl.requests[ip]
	if !exists || now.After(limit.resetTime) {
		rl.requests[ip] = &clientLimit{
			count:     1,
			resetTime: now.Add(time.Minute),
		}
		return true
	}

	if limit.count >= rl.limit {
		return false
	}

	limit.count++
	return true
}

// cleanup removes expired rate limit entries.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, limit := range rl.requests {
		if now.After(limit.resetTime) {
			delete(rl.requests, ip)
		}
	}
}
