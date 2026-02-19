package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	corecache "github.com/jongio/azd-core/cache"
)

const (
	// CacheVersion tracks the cache schema version for invalidation on breaking changes
	CacheVersion = "1.0"
	// DefaultCacheTTL is the default cache time-to-live
	DefaultCacheTTL = 1 * time.Hour
	// cacheKey is the key used for the reqs cache entry
	cacheKey = "reqs_cache"
)

// ReqsCache represents cached reqs check results.
type ReqsCache struct {
	Version       string            `json:"version"` // Schema version for invalidation
	Timestamp     time.Time         `json:"timestamp"`
	AzureYamlHash string            `json:"azureYamlHash"`
	Results       []CachedReqResult `json:"results"`
	AllPassed     bool              `json:"allPassed"`
}

// CachedReqResult represents a cached req check result.
type CachedReqResult struct {
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	Required   string `json:"required"`
	Satisfied  bool   `json:"satisfied"`
	Running    bool   `json:"running,omitempty"`
	CheckedRun bool   `json:"checkedRunning,omitempty"`
	Message    string `json:"message,omitempty"`
}

// CacheManager handles reqs cache operations.
type CacheManager struct {
	manager *corecache.Manager
	enabled bool
	statsMu sync.Mutex // Protects stats
	stats   CacheStats
}

// CacheStats tracks cache hit/miss statistics.
type CacheStats struct {
	Hits   int `json:"hits"`
	Misses int `json:"misses"`
}

// CacheOptions configures the cache manager.
type CacheOptions struct {
	CacheDir string        // Custom cache directory (optional)
	TTL      time.Duration // Cache time-to-live (default: 1 hour)
	Enabled  bool          // Enable/disable caching
}

// NewCacheManager creates a new cache manager with default options.
func NewCacheManager() (*CacheManager, error) {
	return NewCacheManagerWithOptions(CacheOptions{
		Enabled: true,
		TTL:     DefaultCacheTTL,
	})
}

// NewCacheManagerWithOptions creates a new cache manager with custom options.
func NewCacheManagerWithOptions(opts CacheOptions) (*CacheManager, error) {
	// If caching is disabled, return a disabled manager
	if !opts.Enabled {
		return &CacheManager{enabled: false}, nil
	}

	var cacheDir string
	if opts.CacheDir != "" {
		// Use custom cache directory
		cacheDir = opts.CacheDir
	} else {
		// Find .azure directory in current or parent directories
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		azureDir := findAzureDir(cwd)
		if azureDir == "" {
			// Create .azure directory in current directory if not found
			azureDir = filepath.Join(cwd, ".azure")
			if err := os.MkdirAll(azureDir, 0750); err != nil {
				return nil, fmt.Errorf("failed to create .azure directory: %w", err)
			}
		}

		cacheDir = filepath.Join(azureDir, "cache")
	}

	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	ttl := opts.TTL
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}

	m := corecache.NewManager(corecache.Options{
		Dir:     cacheDir,
		TTL:     ttl,
		Version: CacheVersion,
	})

	return &CacheManager{
		manager: m,
		enabled: true,
	}, nil
}

// findAzureDir searches for .azure directory in current and parent directories.
// It stops at filesystem boundaries and does not search in user home directory.
func findAzureDir(startDir string) string {
	dir := startDir
	homeDir, _ := os.UserHomeDir()

	for homeDir == "" || dir != homeDir {

		azureDir := filepath.Join(dir, ".azure")
		if info, err := os.Stat(azureDir); err == nil && info.IsDir() {
			return azureDir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}

		dir = parent
	}
	return ""
}

// GetCachedResults retrieves cached reqs check results if valid.
func (cm *CacheManager) GetCachedResults(azureYamlPath string) (*ReqsCache, bool, error) {
	// If cache is disabled, return cache miss
	if !cm.enabled {
		return nil, false, nil
	}

	// Calculate hash of azure.yaml
	hash, err := corecache.HashFile(azureYamlPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to calculate azure.yaml hash: %w", err)
	}

	var cache ReqsCache
	ok, err := cm.manager.Get(cacheKey, &cache)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read cache: %w", err)
	}
	if !ok {
		cm.recordMiss()
		return nil, false, nil
	}

	// App-specific validation: check azure.yaml hash
	if cache.AzureYamlHash != hash {
		cm.recordMiss()
		return nil, false, nil // azure.yaml has changed - cache is invalid
	}

	cm.recordHit()
	return &cache, true, nil
}

// recordHit records a cache hit (helper to avoid duplicate lock code)
func (cm *CacheManager) recordHit() {
	cm.statsMu.Lock()
	cm.stats.Hits++
	cm.statsMu.Unlock()
}

// recordMiss records a cache miss (helper to avoid duplicate lock code)
func (cm *CacheManager) recordMiss() {
	cm.statsMu.Lock()
	cm.stats.Misses++
	cm.statsMu.Unlock()
}

// SaveResults saves reqs check results to cache.
// Only caches successful results (allPassed=true) to avoid blocking users with stale failures.
func (cm *CacheManager) SaveResults(azureYamlPath string, results []CachedReqResult, allPassed bool) error {
	// If cache is disabled, skip saving
	if !cm.enabled {
		return nil
	}

	// Only cache successful results - failed checks should always be re-run
	// so users aren't blocked by stale failures after installing missing tools
	if !allPassed {
		// Clear any existing cache on failure to ensure fresh check next time
		return cm.ClearCache()
	}

	// Calculate hash of azure.yaml
	hash, err := corecache.HashFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to calculate azure.yaml hash: %w", err)
	}

	cache := ReqsCache{
		Version:       CacheVersion,
		Timestamp:     time.Now(),
		AzureYamlHash: hash,
		Results:       results,
		AllPassed:     allPassed,
	}

	if err := cm.manager.Set(cacheKey, cache); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// ClearCache removes the reqs cache.
func (cm *CacheManager) ClearCache() error {
	// If cache is disabled, nothing to clear
	if !cm.enabled {
		return nil
	}

	err := cm.manager.Invalidate(cacheKey)
	// Reset stats when clearing cache
	cm.statsMu.Lock()
	cm.stats = CacheStats{}
	cm.statsMu.Unlock()
	return err
}

// GetStats returns cache hit/miss statistics.
func (cm *CacheManager) GetStats() CacheStats {
	cm.statsMu.Lock()
	defer cm.statsMu.Unlock()
	return cm.stats
}

// IsEnabled returns whether caching is enabled.
func (cm *CacheManager) IsEnabled() bool {
	return cm.enabled
}

// calculateFileHash calculates SHA256 hash of a file.
// Delegates to core cache.HashFile.
func calculateFileHash(filePath string) (string, error) {
	return corecache.HashFile(filePath)
}
