package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ReqsCache represents cached reqs check results.
type ReqsCache struct {
	Timestamp     time.Time         `json:"timestamp"`
	AzureYamlHash string            `json:"azureYamlHash"`
	Results       []CachedReqResult `json:"results"`
	AllPassed     bool              `json:"allPassed"`
}

// CachedReqResult represents a cached req check result.
type CachedReqResult struct {
	ID         string `json:"id"`
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
}

// NewCacheManager creates a new cache manager.
func NewCacheManager() (*CacheManager, error) {
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

	cacheDir := filepath.Join(azureDir, "cache")
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CacheManager{cacheDir: cacheDir}, nil
}

// findAzureDir searches for .azure directory in current and parent directories.
func findAzureDir(startDir string) string {
	dir := startDir
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
		dir = parent
	}
	return ""
}

// GetCachedResults retrieves cached reqs check results if valid.
func (cm *CacheManager) GetCachedResults(azureYamlPath string) (*ReqsCache, bool, error) {
	// Calculate hash of azure.yaml
	hash, err := calculateFileHash(azureYamlPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to calculate azure.yaml hash: %w", err)
	}

	cacheFile := filepath.Join(cm.cacheDir, "reqs_cache.json")

	// Check if cache file exists
	cacheInfo, err := os.Stat(cacheFile)
	if os.IsNotExist(err) {
		return nil, false, nil // No cache exists
	}
	if err != nil {
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

	// Check if cache is still valid
	// Cache is valid if:
	// 1. azure.yaml hash matches
	// 2. Cache is less than 1 hour old (configurable)
	maxAge := time.Hour
	if cache.AzureYamlHash != hash {
		return nil, false, nil // azure.yaml has changed
	}
	if time.Since(cacheInfo.ModTime()) > maxAge {
		return nil, false, nil // Cache is too old
	}

	return &cache, true, nil
}

// SaveResults saves reqs check results to cache.
func (cm *CacheManager) SaveResults(azureYamlPath string, results []CachedReqResult, allPassed bool) error {
	// Calculate hash of azure.yaml
	hash, err := calculateFileHash(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to calculate azure.yaml hash: %w", err)
	}

	cache := ReqsCache{
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
	if err := os.WriteFile(cacheFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// ClearCache removes the reqs cache.
func (cm *CacheManager) ClearCache() error {
	cacheFile := filepath.Join(cm.cacheDir, "reqs_cache.json")
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
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
