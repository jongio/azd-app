package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// CacheVersion tracks the cache schema version for invalidation on breaking changes
	CacheVersion = "1.0"
	// DefaultCacheTTL is the default cache time-to-live
	DefaultCacheTTL = 1 * time.Hour
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
	cacheDir string
	ttl      time.Duration
	enabled  bool
	statsMu  sync.Mutex
	stats    CacheStats
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

	return &CacheManager{
		cacheDir: cacheDir,
		ttl:      ttl,
		enabled:  true,
	}, nil
}

// findAzureDir searches for .azure directory in current and parent directories.
// It stops at filesystem boundaries and does not search in user home directory.
func findAzureDir(startDir string) string {
	dir := startDir
	homeDir, _ := os.UserHomeDir()

	for {
		azureDir := filepath.Join(dir, ".azure")
		if info, err := os.Stat(azureDir); err == nil && info.IsDir() {
			return azureDir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}

		// Stop at home directory to avoid finding user's global .azure
		if homeDir != "" && parent == homeDir {
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
	hash, err := calculateFileHash(azureYamlPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to calculate azure.yaml hash: %w", err)
	}

	cacheFile := filepath.Join(cm.cacheDir, "reqs_cache.json")

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		cm.statsMu.Lock()
		cm.stats.Misses++
		cm.statsMu.Unlock()
		return nil, false, nil // No cache exists
	} else if err != nil {
		return nil, false, fmt.Errorf("failed to stat cache file: %w", err)
	}

	// Read cache file
	// #nosec G304 -- cacheFile comes from internal cache directory, not user input
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache ReqsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Check cache version - invalidate if schema changed
	if cache.Version != CacheVersion {
		cm.statsMu.Lock()
		cm.stats.Misses++
		cm.statsMu.Unlock()
		return nil, false, nil // Cache schema version mismatch
	}

	// Check if cache is still valid
	// Cache is valid if:
	// 1. Cache version matches (already checked above)
	// 2. azure.yaml hash matches (most important - if this fails, cache is definitely invalid)
	// 3. Cache is less than TTL age (use timestamp from cache JSON, not file ModTime)

	if cache.AzureYamlHash != hash {
		cm.statsMu.Lock()
		cm.stats.Misses++
		cm.statsMu.Unlock()
		return nil, false, nil // azure.yaml has changed - cache is invalid
	}

	if time.Since(cache.Timestamp) > cm.ttl {
		cm.statsMu.Lock()
		cm.stats.Misses++
		cm.statsMu.Unlock()
		return nil, false, nil // Cache is too old
	}

	cm.statsMu.Lock()
	cm.stats.Hits++
	cm.statsMu.Unlock()
	return &cache, true, nil
}

// SaveResults saves reqs check results to cache.
func (cm *CacheManager) SaveResults(azureYamlPath string, results []CachedReqResult, allPassed bool) error {
	// If cache is disabled, skip saving
	if !cm.enabled {
		return nil
	}

	// Calculate hash of azure.yaml
	hash, err := calculateFileHash(azureYamlPath)
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

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	cacheFile := filepath.Join(cm.cacheDir, "reqs_cache.json")
	// Write to temp file first, then rename for atomic write
	tempFile := cacheFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Atomic rename to prevent corruption during concurrent writes
	if err := os.Rename(tempFile, cacheFile); err != nil {
		_ = os.Remove(tempFile) // Clean up temp file on error (best effort)
		return fmt.Errorf("failed to save cache file: %w", err)
	}

	return nil
}

// ClearCache removes the reqs cache.
func (cm *CacheManager) ClearCache() error {
	// If cache is disabled, nothing to clear
	if !cm.enabled {
		return nil
	}

	cacheFile := filepath.Join(cm.cacheDir, "reqs_cache.json")
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	// Reset stats when clearing cache
	cm.statsMu.Lock()
	cm.stats = CacheStats{}
	cm.statsMu.Unlock()
	return nil
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
func calculateFileHash(filePath string) (string, error) {
	// #nosec G304 -- filePath is azureYamlPath which is validated by security.ValidatePath in caller
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
